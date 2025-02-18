package nvd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/license"

	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed"
	feednvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type SyncOptions struct {
	VulnPath           string
	CPEDBURL           string
	CPETranslationsURL string
	CVEFeedPrefixURL   string
	Debug              bool
}

// Sync downloads all the vulnerability data sources.
func Sync(opts SyncOptions, logger log.Logger) error {
	level.Debug(logger).Log("msg", "syncing CPE sqlite")
	start := time.Now()
	if err := DownloadCPEDBFromGithub(opts.VulnPath, opts.CPEDBURL); err != nil {
		return fmt.Errorf("sync CPE database: %w", err)
	}
	level.Debug(logger).Log("msg", "CPE sqlite synced", "duration", time.Since(start))

	level.Debug(logger).Log("msg", "downloading CPE translations", "url", opts.CPETranslationsURL)
	if err := DownloadCPETranslationsFromGithub(opts.VulnPath, opts.CPETranslationsURL); err != nil {
		return fmt.Errorf("sync CPE translations: %w", err)
	}

	level.Debug(logger).Log("msg", "syncing CVEs")
	start = time.Now()
	if err := DownloadCVEFeed(opts.VulnPath, opts.CVEFeedPrefixURL, opts.Debug, logger); err != nil {
		return fmt.Errorf("sync NVD CVE feed: %w", err)
	}
	level.Debug(logger).Log("msg", "CVEs synced", "duration", time.Since(start))

	if err := DownloadEPSSFeed(opts.VulnPath); err != nil {
		return fmt.Errorf("sync EPSS CVE feed: %w", err)
	}

	if err := DownloadCISAKnownExploitsFeed(opts.VulnPath); err != nil {
		return fmt.Errorf("sync CISA known exploits feed: %w", err)
	}

	return nil
}

const (
	epssFeedsURL = "https://epss.cyentia.com"
	epssFilename = "epss_scores-current.csv.gz"
)

// DownloadEPSSFeed downloads the EPSS scores feed.
func DownloadEPSSFeed(vulnPath string) error {
	urlString := epssFeedsURL + "/" + epssFilename
	u, err := url.Parse(urlString)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	path := filepath.Join(vulnPath, strings.TrimSuffix(epssFilename, ".gz"))

	client := fleethttp.NewClient()
	err = download.DownloadAndExtract(client, u, path)
	if err != nil {
		return fmt.Errorf("download %s: %w", u, err)
	}

	return nil
}

// epssScore represents the EPSS score for a CVE.
type epssScore struct {
	CVE   string
	Score float64
}

func parseEPSSScoresFile(path string) ([]epssScore, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comment = '#'
	r.FieldsPerRecord = 3

	// skip the header
	r.Read() //nolint:errcheck

	var epssScores []epssScore
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// each row should have 3 records: cve, epss, and percentile
		if len(rec) != 3 {
			continue
		}

		cve := rec[0]
		score, err := strconv.ParseFloat(rec[1], 64)
		if err != nil {
			return nil, fmt.Errorf("parse epss score: %w", err)
		}

		// ignore percentile

		epssScores = append(epssScores, epssScore{
			CVE:   cve,
			Score: score,
		})
	}

	return epssScores, nil
}

const (
	cisaKnownExploitsURL      = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"
	cisaKnownExploitsFilename = "known_exploited_vulnerabilities.json"
)

// knownExploitedVulnerabilitiesCatalog represents the CISA Catalog of Known Exploited Vulnerabilities.
type knownExploitedVulnerabilitiesCatalog struct {
	Title           string                        `json:"title"`
	CatalogVersion  string                        `json:"catalogVersion"`
	DateReleased    time.Time                     `json:"dateReleased"`
	Count           int                           `json:"count"`
	Vulnerabilities []knownExploitedVulnerability `json:"vulnerabilities"`
}

// knownExploitedVulnerability represents a known exploit in the CISA catalog.
type knownExploitedVulnerability struct {
	CVEID string `json:"cveID"`
	// remaining fields omitted
	// VendorProject     string `json:"vendorProject"`
	// Product           string `json:"product"`
	// VulnerabilityName string `json:"vulnerabilityName"`
	// DateAdded         time.time `json:"dateAdded"`
	// ShortDescription  string `json:"shortDescription"`
	// RequiredAction    string `json:"requiredAction"`
	// DueDate           time.time `json:"dueDate"`
}

// DownloadCISAKnownExploitsFeed downloads the CISA known exploited vulnerabilities feed.
func DownloadCISAKnownExploitsFeed(vulnPath string) error {
	path := filepath.Join(vulnPath, cisaKnownExploitsFilename)

	u, err := url.Parse(cisaKnownExploitsURL)
	if err != nil {
		return err
	}

	client := fleethttp.NewClient()
	err = download.Download(client, u, path)
	if err != nil {
		return fmt.Errorf("download cisa known exploits: %w", err)
	}

	return nil
}

// LoadCVEMeta loads the cvss scores, epss scores, and known exploits from the previously downloaded feeds and saves
// them to the database.
func LoadCVEMeta(ctx context.Context, logger log.Logger, vulnPath string, ds fleet.Datastore) error {
	if !license.IsPremium(ctx) {
		level.Info(logger).Log("msg", "skipping cve_meta parsing due to license check")
		return nil
	}
	// load cvss scores
	files, err := getNVDCVEFeedFiles(vulnPath)
	if err != nil {
		return fmt.Errorf("get nvd cve feeds: %w", err)
	}

	metaMap := make(map[string]fleet.CVEMeta)

	for _, file := range files {

		// Load json files one at a time. Attempting to load them all uses too much memory, > 1 GB.
		dict, err := cvefeed.LoadJSONDictionary(file)
		if err != nil {
			return err
		}

		for cve := range dict {
			vuln, ok := dict[cve].(*feednvd.Vuln)
			if !ok {
				level.Error(logger).Log("msg", "unexpected type for Vuln interface", "cve", cve, "type", fmt.Sprintf("%T", dict[cve]))
				continue
			}
			schema := vuln.Schema()

			meta := fleet.CVEMeta{CVE: cve}

			if len(schema.CVE.Description.DescriptionData) > 0 {
				meta.Description = schema.CVE.Description.DescriptionData[0].Value
			}

			if schema.Impact.BaseMetricV3 != nil {
				meta.CVSSScore = &schema.Impact.BaseMetricV3.CVSSV3.BaseScore
			}

			if published, err := time.Parse(publishedDateFmt, schema.PublishedDate); err != nil {
				level.Error(logger).Log("msg", "failed to parse published data", "cve", cve, "published_date", schema.PublishedDate, "err", err)
			} else {
				meta.Published = &published
			}

			metaMap[cve] = meta
		}
	}

	// load epss scores
	path := filepath.Join(vulnPath, strings.TrimSuffix(epssFilename, ".gz"))

	epssScores, err := parseEPSSScoresFile(path)
	if err != nil {
		return fmt.Errorf("parse epss scores: %w", err)
	}

	for _, epssScore := range epssScores {
		epssScore := epssScore // copy, don't take the address of loop variables
		score, ok := metaMap[epssScore.CVE]
		if !ok {
			score.CVE = epssScore.CVE
		}
		score.EPSSProbability = &epssScore.Score
		metaMap[epssScore.CVE] = score
	}

	// load known exploits
	path = filepath.Join(vulnPath, cisaKnownExploitsFilename)
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var catalog knownExploitedVulnerabilitiesCatalog
	if err := json.Unmarshal(b, &catalog); err != nil {
		return fmt.Errorf("unmarshal cisa known exploited vulnerabilities catalog: %w", err)
	}

	for _, vuln := range catalog.Vulnerabilities {
		score, ok := metaMap[vuln.CVEID]
		if !ok {
			score.CVE = vuln.CVEID
		}
		score.CISAKnownExploit = ptr.Bool(true)
		metaMap[vuln.CVEID] = score
	}

	// The catalog only contains "known" exploits, meaning all other CVEs should have known exploit set to false.
	for cve, meta := range metaMap {
		if meta.CISAKnownExploit == nil {
			meta.CISAKnownExploit = ptr.Bool(false)
		}
		metaMap[cve] = meta
	}

	if len(metaMap) == 0 {
		return nil
	}

	// convert to slice
	var meta []fleet.CVEMeta
	for _, score := range metaMap {
		meta = append(meta, score)
	}

	insertCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	if err := ds.InsertCVEMeta(insertCtx, meta); err != nil {
		return fmt.Errorf("insert cve meta: %w", err)
	}

	return nil
}
