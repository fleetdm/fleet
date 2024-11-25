// Package nvdsync provides a CVE syncer that uses the NVD 2.0 API to download JSON formatted CVE information
// and stores it in the legacy NVD 1.1 format. The reason we decided to store in the legacy format is because
// the github.com/facebookincubator/nvdtools doesn't yet support parsing the new API 2.0 JSON format.
package nvdsync

import (
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/pandatix/nvdapi/common"
	"github.com/pandatix/nvdapi/v2"
)

// CVE syncs CVE information from the NVD database (nvd.nist.gov) using its API 2.0
// to the directory specified in the dbDir field in the form of JSON files.
// It stores the CVE information using the legacy feed format.
// The reason we decided to store in the legacy format is because
// the github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools doesn't yet support parsing
// the new API 2.0 JSON format.
type CVE struct {
	client           *http.Client
	dbDir            string
	logger           log.Logger
	debug            bool
	WaitTimeForRetry time.Duration
	MaxTryAttempts   int
}

var (
	// timeBetweenRequests is the recommended time to wait between NVD API requests.
	timeBetweenRequests = 6 * time.Second
	// maxRetryAttempts is the maximum number of request to retry in case of API failure.
	maxRetryAttempts = 10
	// waitTimeForRetry is the time to wait between retries.
	waitTimeForRetry = 30 * time.Second
	// vulnCheckStartDate is the earliest date to start processing the vulncheck data.
	vulnCheckStartDate = time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC)
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
		client:           fleethttp.NewClient(),
		dbDir:            dbDir,
		logger:           log.NewNopLogger(),
		MaxTryAttempts:   maxRetryAttempts,
		WaitTimeForRetry: waitTimeForRetry,
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
	lastModStartDate, err := s.sync(ctx, nil)
	if err != nil {
		return err
	}

	// Write the lastModStartDate to be used in the next sync.
	if err := s.writeLastModStartDateFile(lastModStartDate); err != nil {
		return err
	}

	return nil
}

// removeLegacyFeeds removes all the legacy feed files downloaded by previous versions of Fleet.
func (s *CVE) removeLegacyFeeds() error {
	// Using * to remove new unfinished syncs (uncompressed)
	jsonGzs, err := filepath.Glob(filepath.Join(s.dbDir, "nvdcve-1.1-*.json*"))
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
	lastModStartDate, err = s.sync(ctx, &lastModStartDate)
	if err != nil {
		return err
	}

	// Update the lastModStartDate for the next synchronization.
	if err := s.writeLastModStartDateFile(lastModStartDate); err != nil {
		return err
	}

	return nil
}

func (s *CVE) updateYearFile(year int, cves []nvdapi.CVEItem) error {
	// The NVD legacy feed files start at year 2002.
	// This is assumed by the github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools package.
	if year < 2002 {
		year = 2002
	}

	// Read the CVE file for the year.
	readStart := time.Now()
	storedCVEFeed, err := readCVEsLegacyFormat(s.dbDir, year)
	if err != nil {
		return err
	}
	level.Debug(s.logger).Log("msg", "read cves", "year", year, "duration", time.Since(readStart))

	// Convert new API 2.0 format to legacy feed format and create map of new CVE information.
	newLegacyCVEs := make(map[string]*schema.NVDCVEFeedJSON10DefCVEItem)
	for _, cve := range cves {
		if cve.CVE.VulnStatus != nil && *cve.CVE.VulnStatus == "Rejected" {
			continue
		}
		legacyCVE := convertAPI20CVEToLegacy(cve.CVE, s.logger)
		newLegacyCVEs[legacyCVE.CVE.CVEDataMeta.ID] = legacyCVE
	}

	// Update existing CVEs with the latest updates (e.g. NVD updated a CVSS metric on an existing CVE).
	//
	// This loop iterates the existing slice and, if there's an update for the item, it will
	// update the item in place. The next for loop takes care of adding the newly reported CVEs.
	updateStart := time.Now()
	for i, storedCVE := range storedCVEFeed.CVEItems {
		if newLegacyCVE, ok := newLegacyCVEs[storedCVE.CVE.CVEDataMeta.ID]; ok {
			storedCVEFeed.CVEItems[i] = newLegacyCVE
			delete(newLegacyCVEs, storedCVE.CVE.CVEDataMeta.ID)
		}
	}
	level.Debug(s.logger).Log("msg", "updated cves", "year", year, "duration", time.Since(updateStart))

	// Add any new CVEs (e.g. a new vulnerability has been found since last time so a new CVE number was reported).
	//
	// Any leftover items from the previous loop in newLegacyCVEs are new CVEs.
	for _, cve := range newLegacyCVEs {
		storedCVEFeed.CVEItems = append(storedCVEFeed.CVEItems, cve)
	}
	storedCVEFeed.CVEDataNumberOfCVEs = strconv.FormatInt(int64(len(storedCVEFeed.CVEItems)), 10)

	// Store the file for the year.
	storeStart := time.Now()
	if err := storeCVEsInLegacyFormat(s.dbDir, year, storedCVEFeed); err != nil {
		return err
	}
	level.Debug(s.logger).Log("msg", "stored cves", "year", year, "duration", time.Since(storeStart))

	return nil
}

func (s *CVE) updateVulnCheckYearFile(year int, cves []VulnCheckCVE, modCount, addCount *int) error {
	// The NVD legacy feed files start at year 2002.
	// This is assumed by the facebookincubator/nvdtools package.
	if year < 2002 {
		year = 2002
	}

	storedCVEFeed, err := readCVEsLegacyFormat(s.dbDir, year)
	if err != nil {
		return err
	}

	// Convert new API 2.0 format to legacy feed format and create map of new CVE information.
	newLegacyCVEs := make(map[string]*schema.NVDCVEFeedJSON10DefCVEItem)
	for _, cve := range cves {
		if cve.CVE.VulnStatus != nil && *cve.CVE.VulnStatus == "Rejected" {
			continue
		}
		legacyCVE := convertAPI20CVEToLegacy(cve.CVE, s.logger)
		updateWithVulnCheckConfigurations(legacyCVE, cve.VcConfigurations)
		newLegacyCVEs[legacyCVE.CVE.CVEDataMeta.ID] = legacyCVE
	}

	// Update existing CVEs with the latest updates (e.g. NVD updated a CVSS metric on an existing CVE).
	//
	// This loop iterates the existing slice and, if there's an update for the item, it will
	// update the item in place. The next for loop takes care of adding the newly reported CVEs.
	updateStart := time.Now()
	counter := 0
	for i, storedCVE := range storedCVEFeed.CVEItems {
		if newLegacyCVE, ok := newLegacyCVEs[storedCVE.CVE.CVEDataMeta.ID]; ok {
			// Don't overwrite the configurations if they are already set.
			if storedCVE.Configurations != nil && len(storedCVE.Configurations.Nodes) > 0 {
				delete(newLegacyCVEs, storedCVE.CVE.CVEDataMeta.ID)
				continue
			}

			if len(newLegacyCVE.Configurations.Nodes) > 0 {
				storedCVEFeed.CVEItems[i].Configurations = newLegacyCVE.Configurations
				counter++
			}

			delete(newLegacyCVEs, storedCVE.CVE.CVEDataMeta.ID)
		}
	}
	*modCount += counter
	level.Debug(s.logger).Log("msg", "updating vulncheck cves", "year", year, "count", counter, "duration", time.Since(updateStart))

	// Add any new CVEs (e.g. a new vulnerability has been found since last time so a new CVE number was reported).
	//
	// Any leftover items from the previous loop in newLegacyCVEs are new CVEs.
	level.Debug(s.logger).Log("msg", "adding new vulncheck cves", "year", year, "count", len(newLegacyCVEs))
	*addCount += len(newLegacyCVEs)
	for _, cve := range newLegacyCVEs {
		storedCVEFeed.CVEItems = append(storedCVEFeed.CVEItems, cve)
	}
	storedCVEFeed.CVEDataNumberOfCVEs = strconv.FormatInt(int64(len(storedCVEFeed.CVEItems)), 10)

	// Store the file for the year.
	if err := storeCVEsInLegacyFormat(s.dbDir, year, storedCVEFeed); err != nil {
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

// httpClient wraps an http.Client to allow for debug and setting a request context.
type httpClient struct {
	*http.Client
	ctx context.Context

	debug bool
}

// Do implements common.HTTPClient.
func (c *httpClient) Do(request *http.Request) (*http.Response, error) {
	start := time.Now()
	if c.debug {
		fmt.Fprintf(os.Stderr, "%s, request: %+v\n", time.Now(), request)
	}

	response, err := c.Client.Do(request.WithContext(c.ctx))
	if err != nil {
		return nil, err
	}

	if c.debug {
		fmt.Fprintf(os.Stderr, "%s (%s) response: %+v\n", time.Now(), time.Since(start), response)
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

// sync performs requests to the NVD https://services.nvd.nist.gov/rest/json/cves/2.0 service to get CVE information
// and updates the files in the local directory.
// It returns the lastModStartDate to use on a subsequent sync call.
//
// If lastModStartDate is nil, it performs the initial (full) synchronization of ALL CVEs.
// If lastModStartDate is set, then it fetches updates since the last sync call.
//
// Reference: https://nvd.nist.gov/developers/api-workflows.
func (s *CVE) sync(ctx context.Context, lastModStartDate *string) (newLastModStartDate string, err error) {
	var (
		startIdx                = int64(0)
		totalResults            = 1
		cvesByYear              = make(map[int][]nvdapi.CVEItem)
		retryAttempts           = 0
		lastModEndDate          *string
		now                     = time.Now().UTC().Format("2006-01-02T15:04:05.000")
		vulnerabilitiesReceived = 0
	)
	if lastModStartDate != nil {
		lastModEndDate = ptr.String(now)
	}

	// Environment variable NETWORK_TEST_NVD_CVE_START_IDX is set only in tests
	// (to reduce test duration time).
	if v := os.Getenv("NETWORK_TEST_NVD_CVE_START_IDX"); v != "" {
		startIdx, err = strconv.ParseInt(v, 10, 32)
		if err != nil {
			return "", err
		}
		totalResults = int(startIdx) + 1
	}

	for startIndex := int(startIdx); startIndex < totalResults; {
		startRequestTime := time.Now()
		cveResponse, err := nvdapi.GetCVEs(s.getHTTPClient(ctx, s.debug), nvdapi.GetCVEsParams{
			StartIndex:       ptr.Int(startIndex),
			LastModStartDate: lastModStartDate,
			LastModEndDate:   lastModEndDate,
		})
		if err != nil {
			if retryAttempts > maxRetryAttempts {
				return "", err
			}
			s.logger.Log("msg", "NVD request returned error", "err", err, "retry-in", waitTimeForRetry)
			retryAttempts++
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(waitTimeForRetry):
				continue
			}
		}
		requestDuration := time.Since(startRequestTime)
		retryAttempts = 0
		totalResults = cveResponse.TotalResults
		startIndex += cveResponse.ResultsPerPage
		newLastModStartDate = cveResponse.Timestamp

		// Environment variable NETWORK_TEST_NVD_CVE_END_IDX is set only in tests
		// (to reduce test duration time).
		if v := os.Getenv("NETWORK_TEST_NVD_CVE_END_IDX"); v != "" {
			endIdx, err := strconv.ParseInt(v, 10, 32)
			if err != nil {
				return "", err
			}
			totalResults = int(endIdx)
		}

		for _, vuln := range cveResponse.Vulnerabilities {
			year, err := strconv.Atoi((*vuln.CVE.ID)[4:8])
			if err != nil {
				return "", err
			}
			vulnerabilitiesReceived++
			cvesByYear[year] = append(cvesByYear[year], vuln)
		}

		// Dump vulnerabilities to the year files to reduce memory footprint.
		// Keeping all vulnerabilities in memory consumed around 11 GB of RAM.
		var updateDuration time.Duration
		if vulnerabilitiesReceived > 10_000 {
			var (
				yearWithMostVulns int
				maxVulnsInYear    int
			)
			for year, cvesInYear := range cvesByYear {
				if len(cvesInYear) > maxVulnsInYear {
					yearWithMostVulns = year
					maxVulnsInYear = len(cvesInYear)
				}
			}
			start := time.Now()
			if err := s.updateYearFile(yearWithMostVulns, cvesByYear[yearWithMostVulns]); err != nil {
				return "", err
			}
			updateDuration = time.Since(start)
			level.Debug(s.logger).Log("msg", "updated file", "year", yearWithMostVulns, "duration", updateDuration, "vulns", maxVulnsInYear)

			vulnerabilitiesReceived -= maxVulnsInYear
			delete(cvesByYear, yearWithMostVulns)
		}

		if startIndex < totalResults {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(timeBetweenRequests - requestDuration - updateDuration):
			}
		}
	}

	for year, cvesInYear := range cvesByYear {
		start := time.Now()
		if err := s.updateYearFile(year, cvesInYear); err != nil {
			return "", err
		}
		level.Debug(s.logger).Log("msg", "updated file", "year", year, "duration", time.Since(start), "vulns", len(cvesInYear))
	}

	return newLastModStartDate, nil
}

func (s *CVE) DoVulnCheck(ctx context.Context) error {
	vulnCheckArchive := "vulncheck.zip"
	baseURL := "https://api.vulncheck.com/v3/backup/nist-nvd2"

	downloadURL, err := s.fetchVulnCheckDownloadURL(ctx, baseURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "error fetching download URL")
	}

	err = s.downloadVulnCheckArchive(ctx, downloadURL, vulnCheckArchive)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "error downloading archive")
	}

	err = s.processVulnCheckFile(vulnCheckArchive)
	if err != nil {
		return fmt.Errorf("error processing VulnCheck file: %w", err)
	}

	return nil
}

// fetchVulnCheckDownloadURL fetches the download URL for the VulnCheck archive
// from the VulnCheck API.
func (s *CVE) fetchVulnCheckDownloadURL(ctx context.Context, baseURL string) (string, error) {
	apiKey := os.Getenv("VULNCHECK_API_KEY")
	if apiKey == "" {
		return "", ctxerr.New(ctx, "VULNCHECK_API_KEY environment variable not set")
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "error parsing URL")
	}

	var resp *http.Response
	for attempt := 0; attempt <= s.MaxTryAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "error creating request")
		}

		req.Header.Add("Authorization", "Bearer "+apiKey)

		resp, err = s.client.Do(req)
		if err != nil {
			if resp != nil {
				resp.Body.Close()
			}
			s.logger.Log("msg", "VulnCheck API request failed", "attempt", attempt, "error", err)
			if attempt == s.MaxTryAttempts {
				return "", ctxerr.Wrap(ctx, err, "max retry attempts reached")
			}
			time.Sleep(s.WaitTimeForRetry)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			break
		}

		resp.Body.Close() // Close the body if we are going to retry or fail
		s.logger.Log("msg", "VulnCheck API request failed", "attempt", attempt, "status", resp.StatusCode, "retry-in", s.WaitTimeForRetry)
		if attempt == s.MaxTryAttempts {
			return "", ctxerr.New(ctx, "max retry attempts reached")
		}
		time.Sleep(s.WaitTimeForRetry)
	}

	if resp == nil || resp.Body == nil {
		return "", ctxerr.New(ctx, "no response or response body is nil")
	}

	defer resp.Body.Close()

	var vcResponse VulnCheckBackupResponse
	if err := json.NewDecoder(resp.Body).Decode(&vcResponse); err != nil {
		return "", ctxerr.Wrap(ctx, err, "error decoding response")
	}

	var downloadURL string
	if len(vcResponse.Data) > 0 {
		downloadURL = vcResponse.Data[0].URL
	}

	if downloadURL == "" {
		return "", ctxerr.New(ctx, "no download URL found")
	}

	return downloadURL, nil
}

// downloadVulnCheckArchive downloads the VulnCheck archive from the provided URL
// and saves it to the configured DB directory
func (s *CVE) downloadVulnCheckArchive(ctx context.Context, downloadURL, outFile string) error {
	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "error creating request")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	out, err := os.Create(filepath.Join(s.dbDir, outFile))
	if err != nil {
		return err
	}

	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (s *CVE) processVulnCheckFile(fileName string) error {
	sanitizedPath, err := sanitizeArchivePath(s.dbDir, fileName)
	if err != nil {
		return fmt.Errorf("error sanitizing archive path: %w", err)
	}

	zipReader, err := zip.OpenReader(sanitizedPath)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	sort.Slice(zipReader.File, func(i, j int) bool {
		return zipReader.File[i].Name > zipReader.File[j].Name
	})

	// files are in reverse chronological order by modification date
	// so we can stop processing files once we find one that is older
	// than the configured vulnCheckStartDate
	var addCount int
	var modCount int
	for _, file := range zipReader.File {
		cvesByYear := make(map[int][]VulnCheckCVE)
		var stopProcessing bool

		gzFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("error opening file %s: %w", file.Name, err)
		}

		gReader, err := gzip.NewReader(gzFile)
		if err != nil {
			return fmt.Errorf("error creating gzip reader for file %s: %w", file.Name, err)
		}

		var data VulnCheckBackupDataFile
		if err := json.NewDecoder(gReader).Decode(&data); err != nil {
			return fmt.Errorf("error decoding JSON from file %s: %w", file.Name, err)
		}

		for _, cve := range data.Vulnerabilities {
			if cve.Item.CVE.LastModified == nil {
				continue
			}
			lastMod, err := time.Parse("2006-01-02T15:04:05.999", *cve.Item.CVE.LastModified)
			if err != nil {
				return fmt.Errorf("error parsing last modified date for %s: %w", *cve.Item.ID, err)
			}

			// Stop processing files if the last modified date is older than the vulncheck start
			// date in order to avoid processing unnecessary files.
			if lastMod.Before(vulnCheckStartDate) {
				stopProcessing = true
				continue
			}

			year, err := strconv.Atoi((*cve.Item.CVE.ID)[4:8])
			if err != nil {
				return err
			}

			cvesByYear[year] = append(cvesByYear[year], cve.Item)
		}

		level.Debug(s.logger).Log("msg", "read vulncheck file", "file", file.Name)

		for year, cvesInYear := range cvesByYear {
			if err := s.updateVulnCheckYearFile(year, cvesInYear, &modCount, &addCount); err != nil {
				return err
			}
		}

		if stopProcessing {
			break
		}

		gReader.Close()
		gzFile.Close()
	}

	level.Debug(s.logger).Log("total updated", modCount, "total added", addCount)

	return nil
}

// sanitizeArchivePath sanitizes the archive file pathing from "G305: Zip Slip vulnerability"
func sanitizeArchivePath(d, t string) (string, error) {
	v := filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	return "", fmt.Errorf("%s: %s", "content filepath is tainted", t)
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

// storeCVEsInLegacyFormat stores the CVEs in legacy feed format.
func storeCVEsInLegacyFormat(dbDir string, year int, cveFeed *schema.NVDCVEFeedJSON10) error {
	sort.Slice(cveFeed.CVEItems, func(i, j int) bool {
		return cveFeed.CVEItems[i].CVE.CVEDataMeta.ID < cveFeed.CVEItems[j].CVE.CVEDataMeta.ID
	})

	path := filepath.Join(dbDir, fmt.Sprintf("nvdcve-1.1-%d.json", year))
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	jsonEncoder := json.NewEncoder(file)
	jsonEncoder.SetIndent("", "  ")
	if err := jsonEncoder.Encode(cveFeed); err != nil {
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}
	return nil
}

// readCVEsLegacyFormat loads the CVEs stored in the legacy feed format.
func readCVEsLegacyFormat(dbDir string, year int) (*schema.NVDCVEFeedJSON10, error) {
	path := filepath.Join(dbDir, fmt.Sprintf("nvdcve-1.1-%d.json", year))

	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &schema.NVDCVEFeedJSON10{
				CVEDataFormat:    "MITRE",
				CVEDataTimestamp: time.Now().Format("2006-01-02T15:04:05Z"),
				CVEDataType:      "CVE",
				CVEDataVersion:   "4.0",
			}, nil
		}
		return nil, err
	}
	defer file.Close()

	var cveFeed schema.NVDCVEFeedJSON10
	if err := json.NewDecoder(file).Decode(&cveFeed); err != nil {
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
func convertAPI20CVEToLegacy(cve nvdapi.CVE, logger log.Logger) *schema.NVDCVEFeedJSON10DefCVEItem {
	logger = log.With(logger, "cve", cve.ID)

	descriptions := make([]*schema.CVEJSON40LangString, 0, len(cve.Descriptions))
	for _, description := range cve.Descriptions {
		// Keep only English descriptions to match the legacy format.
		var lang string
		switch description.Lang {
		case "en":
			lang = description.Lang
		case "en-US": // This occurred starting with Microsoft CVE-2024-38200.
			lang = "en"
		// non-English descriptions with known language tags are ignored.
		case "es": // This occurred in a number of 2004 CVEs
			continue
		// non-English descriptions with unknown language tags are ignored and warned.
		default:
			level.Warn(logger).Log("msg", "Unknown CVE description language tag", "lang", description.Lang)
			continue
		}
		descriptions = append(descriptions, &schema.CVEJSON40LangString{
			Lang:  lang,
			Value: description.Value,
		})
	}

	if len(descriptions) == 0 {
		// Populate a blank description to prevent Fleet cron job from crashing: https://github.com/fleetdm/fleet/issues/21239
		descriptions = append(descriptions, &schema.CVEJSON40LangString{
			Lang:  "en",
			Value: "",
		})
	}

	problemtypeData := make([]*schema.CVEJSON40ProblemtypeProblemtypeData, 0, len(cve.Weaknesses))
	if len(cve.Weaknesses) == 0 {
		problemtypeData = append(problemtypeData, &schema.CVEJSON40ProblemtypeProblemtypeData{
			Description: []*schema.CVEJSON40LangString{},
		})
	}
	for _, weakness := range cve.Weaknesses {
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

	referenceData := make([]*schema.CVEJSON40Reference, 0, len(cve.References))
	for _, reference := range cve.References {
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
	for _, configuration := range cve.Configurations {
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
					Negate:   derefPtr(node.Negate),
					Operator: string(node.Operator),
				})
			}
		}
	}

	var baseMetricV2 *schema.NVDCVEFeedJSON10DefImpactBaseMetricV2
	for _, cvssMetricV2 := range cve.Metrics.CVSSMetricV2 {
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
	for _, cvssMetricV30 := range cve.Metrics.CVSSMetricV30 {
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
	for _, cvssMetricV31 := range cve.Metrics.CVSSMetricV31 {
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

	lastModified, err := convertAPI20TimeToLegacy(cve.LastModified)
	if err != nil {
		logger.Log("msg", "failed to parse lastModified time", "err", err)
	}
	publishedDate, err := convertAPI20TimeToLegacy(cve.Published)
	if err != nil {
		logger.Log("msg", "failed to parse published time", "err", err)
	}

	return &schema.NVDCVEFeedJSON10DefCVEItem{
		CVE: &schema.CVEJSON40{
			Affects: nil, // Doesn't seem used.
			CVEDataMeta: &schema.CVEJSON40CVEDataMeta{
				ID:       *cve.ID,
				ASSIGNER: derefPtr(cve.SourceIdentifier),
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

func updateWithVulnCheckConfigurations(cve *schema.NVDCVEFeedJSON10DefCVEItem, vcConfigurations []nvdapi.Config) {
	nodes := []*schema.NVDCVEFeedJSON10DefNode{} // Legacy entries define an empty list if there are no nodes.
	for _, configuration := range vcConfigurations {
		if configuration.Operator != nil {
			children := make([]*schema.NVDCVEFeedJSON10DefNode, 0, len(configuration.Nodes))
			for _, node := range configuration.Nodes {
				if node.Operator == "" {
					node.Operator = nvdapi.OperatorOr // Default to OR operator if not set
				}
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
				if node.Operator == "" {
					node.Operator = nvdapi.OperatorOr // Default to OR operator if not set
				}
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
					Negate:   derefPtr(node.Negate),
					Operator: string(node.Operator),
				})
			}
		}
	}

	cve.Configurations = &schema.NVDCVEFeedJSON10DefConfigurations{
		CVEDataVersion: "4.0", // All entries seem to have this version string.
		Nodes:          nodes,
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
