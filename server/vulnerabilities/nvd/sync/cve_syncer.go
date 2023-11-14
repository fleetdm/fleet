package nvdsync

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/pandatix/nvdapi/common"
	"github.com/pandatix/nvdapi/v2"
)

// CVE syncs CVE information from the NVD database (nvd.nist.gov) using its API 2.0.
// It stores the CVE information using the legacy feed format.
type CVE struct {
	client *http.Client
	dbDir  string
	logger log.Logger
	debug  bool
}

var (
	// timeBetweenRequests is the recommended time to wait between NVD API requests.
	timeBetweenRequests = 6 * time.Second
	// maxRetryAttempts is the maximum number of request to retry in case of API failure.
	maxRetryAttempts = 10
	// waitTimeForRetry is the time to wait between retries.
	waitTimeForRetry = 30 * time.Second
)

// CVEOption allows configuring a CVE syncer.
type CVEOption func(*CVE)

// WithLogger sets the logger for a CVE syncer.
//
// Default value is log.NewNopLogger().
func WithLogger(logger log.Logger) CVEOption {
	return func(s *CVE) {
		s.logger = logger
	}
}

// WithDebug sets the debug mode for a CVE syncer.
//
// Default value is false.
func WithDebug(debug bool) CVEOption {
	return func(s *CVE) {
		s.debug = debug
	}
}

// NewCVE creates and returns a CVE syncer.
// The provided dbDir is the local directory to use to store/update
// CVE information from NVD.
func NewCVE(dbDir string, opts ...CVEOption) (*CVE, error) {
	if dbDir == "" {
		return nil, errors.New("directory not set")
	}
	s := CVE{
		client: fleethttp.NewClient(),
		dbDir:  dbDir,
		logger: log.NewNopLogger(),
	}
	for _, fn := range opts {
		fn(&s)
	}
	return &s, nil
}

func (s *CVE) lastModStartDateFilePath() string {
	return filepath.Join(s.dbDir, "last_mod_start_date.txt")
}

// Do runs the synchronization from the NVD service to the local DB directory.
func (s *CVE) Do(ctx context.Context) error {
	ok, err := fileExists(s.lastModStartDateFilePath())
	if err != nil {
		return err
	}
	if !ok {
		level.Debug(s.logger).Log("msg", "initial NVD CVE sync")
		return s.initSync(ctx)
	}
	level.Debug(s.logger).Log("msg", "NVD CVE update")
	return s.update(ctx)
}

// initSync performs the initial synchronization (full download) of all CVEs.
func (s *CVE) initSync(ctx context.Context) error {
	// Remove any legacy feeds from previous versions of Fleet.
	if err := s.removeLegacyFeeds(); err != nil {
		return err
	}

	// Perform the initial download of all CVE information.
	cvesInYears, lastModStartDate, err := s.sync(ctx, nil)
	if err != nil {
		return err
	}

	// Store all CVEs using the legacy feed format grouped by year.
	for _, cveInYear := range cvesInYears {
		if err := storeAPI20CVEsInLegacyFormat(s.dbDir, cveInYear.year, cveInYear.cves, s.logger); err != nil {
			return err
		}
	}

	// Write the lastModStartDate to be used in the next sync.
	if err := s.writeLastModStartDateFile(lastModStartDate); err != nil {
		return err
	}

	return nil
}

// removeLegacyFeeds removes all the legacy feed files downloaded by previous versions of Fleet.
func (s *CVE) removeLegacyFeeds() error {
	jsonGzs, err := filepath.Glob(filepath.Join(s.dbDir, "nvdcve-1.1-*.json.gz"))
	if err != nil {
		return err
	}
	metas, err := filepath.Glob(filepath.Join(s.dbDir, "nvdcve-1.1-*.meta"))
	if err != nil {
		return err
	}
	for _, path := range append(jsonGzs, metas...) {
		level.Debug(s.logger).Log("msg", "removing legacy feed file", "path", path)
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

// update downloads all the new CVE updates since the last synchronization.
func (s *CVE) update(ctx context.Context) error {
	// Load the lastModStartDate from the previous synchronization.
	lastModStartDate_, err := os.ReadFile(s.lastModStartDateFilePath())
	if err != nil {
		return err
	}
	lastModStartDate := string(lastModStartDate_)

	// Get the new CVE updates since the previous synchronization.
	newCVEsInYears, lastModStartDate, err := s.sync(ctx, &lastModStartDate)
	if err != nil {
		return err
	}

	for _, yearCVEs := range newCVEsInYears {
		// Read the CVE file for the year.
		storedCVEFeed, err := readCVEsLegacyFormat(s.dbDir, yearCVEs.year)
		if err != nil {
			return err
		}

		// Convert new API 2.0 format to legacy feed format and create map of new CVE information.
		newLegacyCVEs := make(map[string]*schema.NVDCVEFeedJSON10DefCVEItem)
		for _, cve := range yearCVEs.cves {
			legacyCVE := convertAPI20CVEToLegacy(cve, s.logger)
			newLegacyCVEs[legacyCVE.CVE.CVEDataMeta.ID] = legacyCVE
		}

		// Update existing CVEs with the latest updates (e.g. NVD updated a CVSS metric on an existing CVE).
		//
		// This loop iterates the existing slice and, if there's an update for the item, it will
		// update the item in place. The next for loop takes care of adding the newly reported CVEs.
		for i, storedCVE := range storedCVEFeed.CVEItems {
			if newLegacyCVE, ok := newLegacyCVEs[storedCVE.CVE.CVEDataMeta.ID]; ok {
				storedCVEFeed.CVEItems[i] = newLegacyCVE
				delete(newLegacyCVEs, storedCVE.CVE.CVEDataMeta.ID)
			}
		}

		// Add any new CVEs (e.g. a new vulnerability has been found since last time so a new CVE number was reported).
		//
		// Any leftover items from the previous loop in newLegacyCVEs are new CVEs.
		for _, cve := range newLegacyCVEs {
			storedCVEFeed.CVEItems = append(storedCVEFeed.CVEItems, cve)
		}

		// Store the file for the year.
		if err := storeCVEsInLegacyFormat(s.dbDir, yearCVEs.year, storedCVEFeed); err != nil {
			return err
		}
	}

	// Update the lastModStartDate for the next synchronization.
	if err := s.writeLastModStartDateFile(lastModStartDate); err != nil {
		return err
	}

	return nil
}

// writeLastModStartDateFile writes the lastModStartDate to a file in the local DB directory.
func (s *CVE) writeLastModStartDateFile(lastModStartDate string) error {
	if err := os.WriteFile(
		s.lastModStartDateFilePath(),
		[]byte(lastModStartDate),
		constant.DefaultWorldReadableFileMode,
	); err != nil {
		return err
	}
	return nil
}

// cvesInYear contains a list of CVEs for a specific year.
type cvesInYear struct {
	year int
	cves []nvdapi.CVEItem
}

// httpClient wraps an http.Client to allow for debug and setting a request context.
type httpClient struct {
	*http.Client
	ctx context.Context

	debug bool
}

// Do implements common.HTTPClient.
func (c *httpClient) Do(request *http.Request) (*http.Response, error) {
	if c.debug {
		fmt.Fprintf(os.Stderr, "%+v\n", request)
	}

	response, err := c.Client.Do(request.WithContext(c.ctx))
	if err != nil {
		return nil, err
	}

	if c.debug {
		bodyBytes, _ := io.ReadAll(response.Body)
		response.Body.Close()
		response.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		fmt.Fprintf(os.Stderr, "%+v %s\n%s\n", response, err, bodyBytes)
	}

	return response, err
}

// getHTTPClient returns common.HTTPClient to be used by nvdapi methods.
func (s *CVE) getHTTPClient(ctx context.Context, debug bool) common.HTTPClient {
	return &httpClient{
		Client: s.client,
		ctx:    ctx,

		debug: debug,
	}
}

// sync performs requests to the NVD https://services.nvd.nist.gov/rest/json/cves/2.0 service to get CVE information.
// It returns the fetched CVEs and the lastModStartDate to use on a subsequent sync call.
//
// If lastModStartDate is nil, it performs the initial (full) synchronization of ALL CVEs.
// If lastModStartDate is set, then it fetches updates since the last sync call.
//
// Reference: https://nvd.nist.gov/developers/api-workflows.
func (s *CVE) sync(ctx context.Context, lastModStartDate *string) (cves []cvesInYear, newLastModStartDate string, err error) {
	var (
		totalResults   = 1
		cvesByYear     = make(map[int][]nvdapi.CVEItem)
		retryAttempts  = 0
		lastModEndDate *string
		now            = time.Now().UTC().Format("2006-01-02T15:04:05.000")
	)
	if lastModStartDate != nil {
		lastModEndDate = ptr.String(now)
	}
	for startIndex := 0; startIndex < totalResults; {
		cveResponse, err := nvdapi.GetCVEs(s.getHTTPClient(ctx, s.debug), nvdapi.GetCVEsParams{
			StartIndex:       ptr.Int(startIndex),
			LastModStartDate: lastModStartDate,
			LastModEndDate:   lastModEndDate,
		})
		if err != nil {
			if retryAttempts > maxRetryAttempts {
				return nil, "", err
			}
			s.logger.Log("msg", "NVD request returned error", "err", err, "retry-in", waitTimeForRetry)
			retryAttempts++
			select {
			case <-ctx.Done():
				return nil, "", ctx.Err()
			case <-time.After(waitTimeForRetry):
				continue
			}
		}
		retryAttempts = 0
		totalResults = cveResponse.TotalResults
		startIndex += cveResponse.ResultsPerPage
		newLastModStartDate = cveResponse.Timestamp

		for _, vuln := range cveResponse.Vulnerabilities {
			year, err := strconv.Atoi((*vuln.CVE.ID)[4:8])
			if err != nil {
				return nil, "", err
			}
			cvesByYear[year] = append(cvesByYear[year], vuln)
		}

		if startIndex < totalResults {
			select {
			case <-ctx.Done():
				return nil, "", ctx.Err()
			case <-time.After(timeBetweenRequests):
			}
		}
	}
	cves = cvesByYearSlice(cvesByYear)

	return cves, newLastModStartDate, nil
}

// cvesByYearSlice returns a slice of CVEs per year sorted by year.
func cvesByYearSlice(cvesByYear map[int][]nvdapi.CVEItem) []cvesInYear {
	cves := make([]cvesInYear, 0, len(cvesByYear))
	for year, yearCVEs := range cvesByYear {
		cves = append(cves, cvesInYear{
			year: year,
			cves: yearCVEs,
		})
	}
	sort.Slice(cves, func(i, j int) bool {
		return cves[i].year < cves[j].year
	})
	return cves
}

// fileExists returns whether a file at path exists.
func fileExists(path string) (bool, error) {
	switch _, err := os.Stat(path); {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}

// storeAPI20CVEsInLegacyFormat stores the provided CVEs in API 2.0 format in the legacy feed format (gzipped JSON by year).
func storeAPI20CVEsInLegacyFormat(dbDir string, year int, cves []nvdapi.CVEItem, logger log.Logger) error {
	sort.Slice(cves, func(i, j int) bool {
		return *cves[i].CVE.ID < *cves[j].CVE.ID
	})
	cveFeed := schema.NVDCVEFeedJSON10{
		CVEDataFormat:       "MITRE",
		CVEDataNumberOfCVEs: strconv.FormatInt(int64(len(cves)), 10),
		CVEDataTimestamp:    time.Now().Format("2006-01-02T15:04:05Z"),
		CVEDataType:         "CVE",
		CVEDataVersion:      "4.0",
		CVEItems:            make([]*schema.NVDCVEFeedJSON10DefCVEItem, 0, len(cves)),
	}
	for _, cve := range cves {
		cveFeed.CVEItems = append(cveFeed.CVEItems, convertAPI20CVEToLegacy(cve, logger))
	}

	if err := storeCVEsInLegacyFormat(dbDir, year, &cveFeed); err != nil {
		return err
	}
	return nil
}

// storeCVEsInLegacyFormat stores the CVEs in legacy feed format (gzipped JSON).
func storeCVEsInLegacyFormat(dbDir string, year int, cveFeed *schema.NVDCVEFeedJSON10) error {
	path := filepath.Join(dbDir, fmt.Sprintf("nvdcve-1.1-%d.json.gz", year))
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	w := gzip.NewWriter(file)
	defer w.Close()

	jsonEncoder := json.NewEncoder(w)
	jsonEncoder.SetIndent("", "  ")
	if err := jsonEncoder.Encode(cveFeed); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return nil
}

// readCVEsLegacyFormat loads the CVEs stored in the legacy feed format.
func readCVEsLegacyFormat(dbDir string, year int) (*schema.NVDCVEFeedJSON10, error) {
	path := filepath.Join(dbDir, fmt.Sprintf("nvdcve-1.1-%d.json.gz", year))

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	r, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var cveFeed schema.NVDCVEFeedJSON10
	if err := json.NewDecoder(r).Decode(&cveFeed); err != nil {
		return nil, err
	}

	if err := r.Close(); err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return &cveFeed, nil
}

func derefPtr[T any](p *T) T {
	if p != nil {
		return *p
	}
	var t T
	return t
}

// convertAPI20CVEToLegacy performs the conversion of a CVE in API 2.0 format to the legacy feed format.
func convertAPI20CVEToLegacy(cve nvdapi.CVEItem, logger log.Logger) *schema.NVDCVEFeedJSON10DefCVEItem {
	logger = log.With(logger, "cve", cve.CVE.ID)

	descriptions := make([]*schema.CVEJSON40LangString, 0, len(cve.CVE.Descriptions))
	for _, description := range cve.CVE.Descriptions {
		// Keep only english descriptions to match the legacy.
		if description.Lang != "en" {
			continue
		}
		descriptions = append(descriptions, &schema.CVEJSON40LangString{
			Lang:  description.Lang,
			Value: description.Value,
		})
	}

	problemtypeData := make([]*schema.CVEJSON40ProblemtypeProblemtypeData, 0, len(cve.CVE.Weaknesses))
	if len(cve.CVE.Weaknesses) == 0 {
		problemtypeData = append(problemtypeData, &schema.CVEJSON40ProblemtypeProblemtypeData{
			Description: []*schema.CVEJSON40LangString{},
		})
	}
	for _, weakness := range cve.CVE.Weaknesses {
		if weakness.Type != "Primary" {
			continue
		}
		descriptions := make([]*schema.CVEJSON40LangString, 0, len(weakness.Description))
		for _, description := range weakness.Description {
			descriptions = append(descriptions, &schema.CVEJSON40LangString{
				Lang:  description.Lang,
				Value: description.Value,
			})
		}
		problemtypeData = append(problemtypeData, &schema.CVEJSON40ProblemtypeProblemtypeData{
			Description: descriptions,
		})
	}

	referenceData := make([]*schema.CVEJSON40Reference, 0, len(cve.CVE.References))
	for _, reference := range cve.CVE.References {
		tags := []string{} // Entries that have no tag set an empty list.
		if len(reference.Tags) != 0 {
			tags = reference.Tags
		}
		referenceData = append(referenceData, &schema.CVEJSON40Reference{
			Name:      reference.URL, // Most entries have name set to the URL, and there's no name field on API 2.0.
			Refsource: "",            // Not available on API 2.0.
			Tags:      tags,
			URL:       reference.URL,
		})
	}

	nodes := []*schema.NVDCVEFeedJSON10DefNode{} // Legacy entries define an empty list if there are no nodes.
	for _, configuration := range cve.CVE.Configurations {
		if configuration.Operator != nil {
			children := make([]*schema.NVDCVEFeedJSON10DefNode, 0, len(configuration.Nodes))
			for _, node := range configuration.Nodes {
				cpeMatches := make([]*schema.NVDCVEFeedJSON10DefCPEMatch, 0, len(node.CPEMatch))
				for _, cpeMatch := range node.CPEMatch {
					cpeMatches = append(cpeMatches, &schema.NVDCVEFeedJSON10DefCPEMatch{
						CPEName:               []*schema.NVDCVEFeedJSON10DefCPEName{}, // All entries have this field with an empty array.
						Cpe23Uri:              cpeMatch.Criteria,                      // All entries are in CPE 2.3 format.
						VersionEndExcluding:   derefPtr(cpeMatch.VersionEndExcluding),
						VersionEndIncluding:   derefPtr(cpeMatch.VersionEndIncluding),
						VersionStartExcluding: derefPtr(cpeMatch.VersionStartExcluding),
						VersionStartIncluding: derefPtr(cpeMatch.VersionStartIncluding),
						Vulnerable:            cpeMatch.Vulnerable,
					})
				}
				children = append(children, &schema.NVDCVEFeedJSON10DefNode{
					CPEMatch: cpeMatches,
					Children: []*schema.NVDCVEFeedJSON10DefNode{},
					Negate:   derefPtr(node.Negate),
					Operator: string(node.Operator),
				})
			}
			nodes = append(nodes, &schema.NVDCVEFeedJSON10DefNode{
				CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{},
				Children: children,
				Negate:   derefPtr(configuration.Negate),
				Operator: string(*configuration.Operator),
			})
		} else {
			for _, node := range configuration.Nodes {
				cpeMatches := make([]*schema.NVDCVEFeedJSON10DefCPEMatch, 0, len(node.CPEMatch))
				for _, cpeMatch := range node.CPEMatch {
					cpeMatches = append(cpeMatches, &schema.NVDCVEFeedJSON10DefCPEMatch{
						CPEName:               []*schema.NVDCVEFeedJSON10DefCPEName{}, // All entries have this field with an empty array.
						Cpe23Uri:              cpeMatch.Criteria,                      // All entries are in CPE 2.3 format.
						VersionEndExcluding:   derefPtr(cpeMatch.VersionEndExcluding),
						VersionEndIncluding:   derefPtr(cpeMatch.VersionEndIncluding),
						VersionStartExcluding: derefPtr(cpeMatch.VersionStartExcluding),
						VersionStartIncluding: derefPtr(cpeMatch.VersionStartIncluding),
						Vulnerable:            cpeMatch.Vulnerable,
					})
				}
				nodes = append(nodes, &schema.NVDCVEFeedJSON10DefNode{
					CPEMatch: cpeMatches,
					Children: []*schema.NVDCVEFeedJSON10DefNode{},
					Negate:   *node.Negate,
					Operator: string(node.Operator),
				})
			}
		}
	}

	var baseMetricV2 *schema.NVDCVEFeedJSON10DefImpactBaseMetricV2
	for _, cvssMetricV2 := range cve.CVE.Metrics.CVSSMetricV2 {
		if cvssMetricV2.Type != "Primary" {
			continue
		}
		baseMetricV2 = &schema.NVDCVEFeedJSON10DefImpactBaseMetricV2{
			AcInsufInfo: *cvssMetricV2.ACInsufInfo,
			CVSSV2: &schema.CVSSV20{
				AccessComplexity:           derefPtr(cvssMetricV2.CVSSData.AccessComplexity),
				AccessVector:               derefPtr(cvssMetricV2.CVSSData.AccessVector),
				Authentication:             derefPtr(cvssMetricV2.CVSSData.Authentication),
				AvailabilityImpact:         derefPtr(cvssMetricV2.CVSSData.AvailabilityImpact),
				AvailabilityRequirement:    derefPtr(cvssMetricV2.CVSSData.AvailabilityRequirement),
				BaseScore:                  cvssMetricV2.CVSSData.BaseScore,
				CollateralDamagePotential:  derefPtr(cvssMetricV2.CVSSData.CollateralDamagePotential),
				ConfidentialityImpact:      derefPtr(cvssMetricV2.CVSSData.ConfidentialityImpact),
				ConfidentialityRequirement: derefPtr(cvssMetricV2.CVSSData.ConfidentialityRequirement),
				EnvironmentalScore:         derefPtr(cvssMetricV2.CVSSData.EnvironmentalScore),
				Exploitability:             derefPtr(cvssMetricV2.CVSSData.Exploitability),
				IntegrityImpact:            derefPtr(cvssMetricV2.CVSSData.IntegrityImpact),
				IntegrityRequirement:       derefPtr(cvssMetricV2.CVSSData.IntegrityRequirement),
				RemediationLevel:           derefPtr(cvssMetricV2.CVSSData.RemediationLevel),
				ReportConfidence:           derefPtr(cvssMetricV2.CVSSData.ReportConfidence),
				TargetDistribution:         derefPtr(cvssMetricV2.CVSSData.TargetDistribution),
				TemporalScore:              derefPtr(cvssMetricV2.CVSSData.TemporalScore),
				VectorString:               cvssMetricV2.CVSSData.VectorString,
				Version:                    cvssMetricV2.CVSSData.Version,
			},
			ExploitabilityScore:     derefPtr((*float64)(cvssMetricV2.ExploitabilityScore)),
			ImpactScore:             derefPtr((*float64)(cvssMetricV2.ImpactScore)),
			ObtainAllPrivilege:      derefPtr(cvssMetricV2.ObtainAllPrivilege),
			ObtainOtherPrivilege:    derefPtr(cvssMetricV2.ObtainOtherPrivilege),
			ObtainUserPrivilege:     derefPtr(cvssMetricV2.ObtainUserPrivilege),
			Severity:                derefPtr(cvssMetricV2.BaseSeverity),
			UserInteractionRequired: derefPtr(cvssMetricV2.UserInteractionRequired),
		}
	}

	var baseMetricV3 *schema.NVDCVEFeedJSON10DefImpactBaseMetricV3
	for _, cvssMetricV30 := range cve.CVE.Metrics.CVSSMetricV30 {
		if cvssMetricV30.Type != "Primary" {
			continue
		}
		baseMetricV3 = &schema.NVDCVEFeedJSON10DefImpactBaseMetricV3{
			CVSSV3: &schema.CVSSV30{
				AttackComplexity:              derefPtr(cvssMetricV30.CVSSData.AttackComplexity),
				AttackVector:                  derefPtr(cvssMetricV30.CVSSData.AttackVector),
				AvailabilityImpact:            derefPtr(cvssMetricV30.CVSSData.AvailabilityImpact),
				AvailabilityRequirement:       derefPtr(cvssMetricV30.CVSSData.AvailabilityRequirement),
				BaseScore:                     cvssMetricV30.CVSSData.BaseScore,
				BaseSeverity:                  cvssMetricV30.CVSSData.BaseSeverity,
				ConfidentialityImpact:         derefPtr(cvssMetricV30.CVSSData.ConfidentialityImpact),
				ConfidentialityRequirement:    derefPtr(cvssMetricV30.CVSSData.ConfidentialityRequirement),
				EnvironmentalScore:            derefPtr(cvssMetricV30.CVSSData.EnvironmentalScore),
				EnvironmentalSeverity:         derefPtr(cvssMetricV30.CVSSData.EnvironmentalSeverity),
				ExploitCodeMaturity:           derefPtr(cvssMetricV30.CVSSData.ExploitCodeMaturity),
				IntegrityImpact:               derefPtr(cvssMetricV30.CVSSData.IntegrityImpact),
				IntegrityRequirement:          derefPtr(cvssMetricV30.CVSSData.IntegrityRequirement),
				ModifiedAttackComplexity:      derefPtr(cvssMetricV30.CVSSData.ModifiedAttackComplexity),
				ModifiedAttackVector:          derefPtr(cvssMetricV30.CVSSData.ModifiedAttackVector),
				ModifiedAvailabilityImpact:    derefPtr(cvssMetricV30.CVSSData.ModifiedAvailabilityImpact),
				ModifiedConfidentialityImpact: derefPtr(cvssMetricV30.CVSSData.ModifiedConfidentialityImpact),
				ModifiedIntegrityImpact:       derefPtr(cvssMetricV30.CVSSData.ModifiedIntegrityImpact),
				ModifiedPrivilegesRequired:    derefPtr(cvssMetricV30.CVSSData.ModifiedPrivilegesRequired),
				ModifiedScope:                 derefPtr(cvssMetricV30.CVSSData.ModifiedScope),
				ModifiedUserInteraction:       derefPtr(cvssMetricV30.CVSSData.ModifiedUserInteraction),
				PrivilegesRequired:            derefPtr(cvssMetricV30.CVSSData.PrivilegesRequired),
				RemediationLevel:              derefPtr(cvssMetricV30.CVSSData.RemediationLevel),
				ReportConfidence:              derefPtr(cvssMetricV30.CVSSData.ReportConfidence),
				Scope:                         derefPtr(cvssMetricV30.CVSSData.Scope),
				TemporalScore:                 derefPtr(cvssMetricV30.CVSSData.TemporalScore),
				TemporalSeverity:              derefPtr(cvssMetricV30.CVSSData.TemporalSeverity),
				UserInteraction:               derefPtr(cvssMetricV30.CVSSData.UserInteraction),
				VectorString:                  cvssMetricV30.CVSSData.VectorString,
				Version:                       cvssMetricV30.CVSSData.Version,
			},
			ExploitabilityScore: derefPtr((*float64)(cvssMetricV30.ExploitabilityScore)),
			ImpactScore:         derefPtr((*float64)(cvssMetricV30.ImpactScore)),
		}
	}
	// Use CVSSMetricV31 if available (override CVSSMetricV30)
	for _, cvssMetricV31 := range cve.CVE.Metrics.CVSSMetricV31 {
		if cvssMetricV31.Type != "Primary" {
			continue
		}
		baseMetricV3 = &schema.NVDCVEFeedJSON10DefImpactBaseMetricV3{
			CVSSV3: &schema.CVSSV30{
				AttackComplexity:              derefPtr(cvssMetricV31.CVSSData.AttackComplexity),
				AttackVector:                  derefPtr(cvssMetricV31.CVSSData.AttackVector),
				AvailabilityImpact:            derefPtr(cvssMetricV31.CVSSData.AvailabilityImpact),
				AvailabilityRequirement:       derefPtr(cvssMetricV31.CVSSData.AvailabilityRequirement),
				BaseScore:                     cvssMetricV31.CVSSData.BaseScore,
				BaseSeverity:                  cvssMetricV31.CVSSData.BaseSeverity,
				ConfidentialityImpact:         derefPtr(cvssMetricV31.CVSSData.ConfidentialityImpact),
				ConfidentialityRequirement:    derefPtr(cvssMetricV31.CVSSData.ConfidentialityRequirement),
				EnvironmentalScore:            derefPtr(cvssMetricV31.CVSSData.EnvironmentalScore),
				EnvironmentalSeverity:         derefPtr(cvssMetricV31.CVSSData.EnvironmentalSeverity),
				ExploitCodeMaturity:           derefPtr(cvssMetricV31.CVSSData.ExploitCodeMaturity),
				IntegrityImpact:               derefPtr(cvssMetricV31.CVSSData.IntegrityImpact),
				IntegrityRequirement:          derefPtr(cvssMetricV31.CVSSData.IntegrityRequirement),
				ModifiedAttackComplexity:      derefPtr(cvssMetricV31.CVSSData.ModifiedAttackComplexity),
				ModifiedAttackVector:          derefPtr(cvssMetricV31.CVSSData.ModifiedAttackVector),
				ModifiedAvailabilityImpact:    derefPtr(cvssMetricV31.CVSSData.ModifiedAvailabilityImpact),
				ModifiedConfidentialityImpact: derefPtr(cvssMetricV31.CVSSData.ModifiedConfidentialityImpact),
				ModifiedIntegrityImpact:       derefPtr(cvssMetricV31.CVSSData.ModifiedIntegrityImpact),
				ModifiedPrivilegesRequired:    derefPtr(cvssMetricV31.CVSSData.ModifiedPrivilegesRequired),
				ModifiedScope:                 derefPtr(cvssMetricV31.CVSSData.ModifiedScope),
				ModifiedUserInteraction:       derefPtr(cvssMetricV31.CVSSData.ModifiedUserInteraction),
				PrivilegesRequired:            derefPtr(cvssMetricV31.CVSSData.PrivilegesRequired),
				RemediationLevel:              derefPtr(cvssMetricV31.CVSSData.RemediationLevel),
				ReportConfidence:              derefPtr(cvssMetricV31.CVSSData.ReportConfidence),
				Scope:                         derefPtr(cvssMetricV31.CVSSData.Scope),
				TemporalScore:                 derefPtr(cvssMetricV31.CVSSData.TemporalScore),
				TemporalSeverity:              derefPtr(cvssMetricV31.CVSSData.TemporalSeverity),
				UserInteraction:               derefPtr(cvssMetricV31.CVSSData.UserInteraction),
				VectorString:                  cvssMetricV31.CVSSData.VectorString,
				Version:                       cvssMetricV31.CVSSData.Version,
			},
			ExploitabilityScore: derefPtr((*float64)(cvssMetricV31.ExploitabilityScore)),
			ImpactScore:         derefPtr((*float64)(cvssMetricV31.ImpactScore)),
		}
	}

	lastModified, err := convertAPI20TimeToLegacy(cve.CVE.LastModified)
	if err != nil {
		logger.Log("msg", "failed to parse lastModified time", "err", err)
	}
	publishedDate, err := convertAPI20TimeToLegacy(cve.CVE.Published)
	if err != nil {
		logger.Log("msg", "failed to parse published time", "err", err)
	}

	return &schema.NVDCVEFeedJSON10DefCVEItem{
		CVE: &schema.CVEJSON40{
			Affects: nil, // Doesn't seem used.
			CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
				ID:       *cve.CVE.ID,
				ASSIGNER: derefPtr(cve.CVE.SourceIdentifier),
				STATE:    "", // Doesn't seem used.
			},
			DataFormat:  "MITRE", // All entries seem to have this format string.
			DataType:    "CVE",   // All entries seem to have this type string.
			DataVersion: "4.0",   // All entries seem to have this version string.
			Description: &schema.CVEJSON40Description{
				DescriptionData: descriptions,
			},
			Problemtype: &schema.CVEJSON40Problemtype{
				ProblemtypeData: problemtypeData,
			},
			References: &schema.CVEJSON40References{
				ReferenceData: referenceData,
			},
		},
		Configurations: &schema.NVDCVEFeedJSON10DefConfigurations{
			CVEDataVersion: "4.0", // All entries seem to have this version string.
			Nodes:          nodes,
		},
		Impact: &schema.NVDCVEFeedJSON10DefImpact{
			BaseMetricV2: baseMetricV2,
			BaseMetricV3: baseMetricV3,
		},
		LastModifiedDate: lastModified,
		PublishedDate:    publishedDate,
	}
}

// convertAPI20TimeToLegacy converts the timestamps from API 2.0 format to the expected legacy feed time format.
func convertAPI20TimeToLegacy(t *string) (string, error) {
	const (
		api20TimeFormat  = "2006-01-02T15:04:05"
		legacyTimeFormat = "2006-01-02T15:04Z"
	)
	var ts string
	if t != nil {
		tt, err := time.Parse(api20TimeFormat, *t)
		if err != nil {
			return "", err
		}
		ts = tt.Format(legacyTimeFormat)
	}
	return ts, nil
}
