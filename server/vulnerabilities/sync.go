package vulnerabilities

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed"
	feednvd "github.com/facebookincubator/nvdtools/cvefeed/nvd"
	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/multierr"
)

func Sync(vulnPath string, config config.FleetConfig, ds fleet.Datastore) error {
	if config.Vulnerabilities.DisableDataSync {
		return nil
	}

	client := fleethttp.NewClient()

	var syncErr error

	if err := DownloadCPEDatabase(vulnPath, client, WithCPEURL(config.Vulnerabilities.CPEDatabaseURL)); err != nil {
		syncErr = multierror.Append(syncErr, fmt.Errorf("sync CPE database: %w", err))
	}

	if err := DownloadNVDCVEFeed(vulnPath, ""); err != nil {
		syncErr = multierr.Append(syncErr, fmt.Errorf("sync NVD CVE feed: %w", err))
	}

	if err := DownloadEPSSFeed(vulnPath, client); err != nil {
		syncErr = multierr.Append(syncErr, fmt.Errorf("sync EPSS CVE feed: %w", err))
	}

	if err := DownloadCISAKnownExploitsFeed(vulnPath, client); err != nil {
		syncErr = multierr.Append(syncErr, fmt.Errorf("sync CISA known exploits feed: %w", err))
	}

	if err := LoadCVEScores(vulnPath, ds); err != nil {
		syncErr = multierr.Append(syncErr, err)
	}

	return syncErr
}

const epssFeedsURL = "https://epss.cyentia.com"
const epssFilename = "epss_scores-current.csv.gz"

func DownloadEPSSFeed(vulnPath string, client *http.Client) error {
	urlString := epssFeedsURL + "/" + epssFilename
	u, err := url.Parse(urlString)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	path := filepath.Join(vulnPath, strings.TrimSuffix(epssFilename, ".gz"))

	err = download.Download(client, u, path)
	if err != nil {
		return fmt.Errorf("download %s: %w", u, err)
	}

	return nil
}

// EPSSScore represents the EPSS score for a CVE.
type EPSSScore struct {
	CVE   string
	Score float64
}

func parseEPSSScoresFile(path string) ([]EPSSScore, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comment = '#'
	r.FieldsPerRecord = 3

	// skip the header
	r.Read()

	var epssScores []EPSSScore
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(rec) != 3 {
			continue
		}

		cve := rec[0]
		score, err := strconv.ParseFloat(rec[1], 64)
		if err != nil {
			return nil, fmt.Errorf("parse epss score: %w", err)
		}

		// ignore percentile

		epssScores = append(epssScores, EPSSScore{
			CVE:   cve,
			Score: score,
		})
	}

	return epssScores, nil
}

const cisaKnownExploitsURL = "https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json"
const cisaKnownExploitsFilename = "known_exploited_vulnerabilities.json"

// KnownExploitedVulnerabilitiesCatalog represents the CISA Catalog of Known Exploited Vulnerabilities.
type KnownExploitedVulnerabilitiesCatalog struct {
	Title           string                        `json:"title"`
	CatalogVersion  string                        `json:"catalogVersion"`
	DateReleased    time.Time                     `json:"dateReleased"`
	Count           int                           `json:"count"`
	Vulnerabilities []KnownExploitedVulnerability `json:"vulnerabilities"`
}

// KnownExplitedVulnerability represents a known exploit in the CISA catalog.
type KnownExploitedVulnerability struct {
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

func DownloadCISAKnownExploitsFeed(vulnPath string, client *http.Client) error {
	path := filepath.Join(vulnPath, cisaKnownExploitsFilename)

	u, err := url.Parse(cisaKnownExploitsURL)
	if err != nil {
		return err
	}

	err = download.Download(client, u, path)
	if err != nil {
		return fmt.Errorf("download cisa known exploits: %w", err)
	}

	return nil
}

func LoadCVEScores(vulnPath string, ds fleet.Datastore) error {
	// load cvss scores
	files, err := getNVDCVEFeedFiles(vulnPath)
	if err != nil {
		return fmt.Errorf("get nvd cve feeds: %w", err)
	}

	dict, err := cvefeed.LoadJSONDictionary(files...)
	if err != nil {
		return err
	}

	scoresMap := make(map[string]fleet.CVEScore)
	for cve := range dict {
		schema := dict[cve].(*feednvd.Vuln).Schema()
		if schema.Impact.BaseMetricV3 == nil {
			continue
		}
		baseScore := schema.Impact.BaseMetricV3.CVSSV3.BaseScore

		score := fleet.CVEScore{
			CVE:       cve,
			CVSSScore: &baseScore,
		}
		scoresMap[cve] = score
	}

	// load epss scores
	path := filepath.Join(vulnPath, strings.TrimSuffix(epssFilename, ".gz"))

	epssScores, err := parseEPSSScoresFile(path)
	if err != nil {
		return fmt.Errorf("parse epss scores: %w", err)
	}

	for _, epssScore := range epssScores {
		score, ok := scoresMap[epssScore.CVE]
		if !ok {
			score.CVE = epssScore.CVE
		}
		score.EPSSProbability = &epssScore.Score
		scoresMap[epssScore.CVE] = score
	}

	// load known exploits
	path = filepath.Join(vulnPath, cisaKnownExploitsFilename)
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var catalog KnownExploitedVulnerabilitiesCatalog
	if err := json.Unmarshal(b, &catalog); err != nil {
		return fmt.Errorf("unmarshal cisa known exploited vulnerabilities catalog: %w", err)
	}

	for _, vuln := range catalog.Vulnerabilities {
		score, ok := scoresMap[vuln.CVEID]
		if !ok {
			score.CVE = vuln.CVEID
		}
		score.CISAKnownExploit = ptr.Bool(true)
		scoresMap[vuln.CVEID] = score
	}

	// The catalog only contains "known" exploits, meaning all other CVEs should have known exploit set to false.
	for cve, score := range scoresMap {
		if score.CISAKnownExploit == nil {
			score.CISAKnownExploit = ptr.Bool(false)
		}
		scoresMap[cve] = score
	}

	if len(scoresMap) == 0 {
		return nil
	}

	// convert to slice
	var scores []fleet.CVEScore
	for _, score := range scoresMap {
		scores = append(scores, score)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	if err := ds.InsertCVEScores(ctx, scores); err != nil {
		return fmt.Errorf("insert cisa known exploits: %w", err)
	}

	return nil
}
