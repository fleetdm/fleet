package vulnerabilities

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/facebookincubator/nvdtools/cvefeed"
	feednvd "github.com/facebookincubator/nvdtools/cvefeed/nvd"
	"github.com/facebookincubator/nvdtools/providers/nvd"
	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// DownloadNVDCVEFeed downloads the NVD CVE feed. Skips downloading if the cve feed has not changed since the last time.
func DownloadNVDCVEFeed(vulnPath string, cveFeedPrefixURL string) error {
	cve := nvd.SupportedCVE["cve-1.1.json.gz"]

	source := nvd.NewSourceConfig()
	if cveFeedPrefixURL != "" {
		parsed, err := url.Parse(cveFeedPrefixURL)
		if err != nil {
			return fmt.Errorf("parsing cve feed url prefix override: %w", err)
		}
		source.Host = parsed.Host
		source.CVEFeedPath = parsed.Path
		source.Scheme = parsed.Scheme
	}

	dfs := nvd.Sync{
		Feeds:    []nvd.Syncer{cve},
		Source:   source,
		LocalDir: vulnPath,
	}

	syncTimeout := 5 * time.Minute
	if os.Getenv("NETWORK_TEST") != "" {
		syncTimeout = 10 * time.Minute
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), syncTimeout)
	defer cancelFunc()

	if err := dfs.Do(ctx); err != nil {
		return fmt.Errorf("download nvd cve feed: %w", err)
	}

	return nil
}

const publishedDateFmt = "2006-01-02T15:04Z" // not quite RFC3339

var (
	rxNVDCVEArchive = regexp.MustCompile(`nvdcve.*\.gz$`)

	// this allows mocking the time package for tests, by default it is equivalent
	// to the time functions, e.g. theClock.Now() == time.Now().
	theClock clock.Clock = clock.C
)

func getNVDCVEFeedFiles(vulnPath string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(vulnPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if match := rxNVDCVEArchive.MatchString(path); !match {
			return nil
		}

		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

// TranslateCPEToCVE maps the CVEs found in NVD archive files in the
// vulnerabilities database folder to software CPEs in the fleet database.
// If collectRecentVulns is true, it also returns a mapping of recent CVEs
// to a list of CPEs affected by the CVE, otherwise that map is nil.
func TranslateCPEToCVE(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	logger kitlog.Logger,
	collectVulns bool,
	recentVulnerabilityMaxAge time.Duration,
) (map[string][]string, error) {
	files, err := getNVDCVEFeedFiles(vulnPath)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}

	// Skip CPEs from platforms supported by OVAL
	cpeList, err := ds.AllCPEs(ctx, oval.SupportedHostPlatforms)
	if err != nil {
		return nil, err
	}

	cpes := make([]*wfn.Attributes, 0, len(cpeList))
	for _, cpe := range cpeList {
		uri := cpe.CPE
		// Skip dummy CPEs
		if strings.HasPrefix(uri, "none") {
			continue
		}

		attr, err := wfn.Parse(uri)
		if err != nil {
			return nil, err
		}
		cpes = append(cpes, attr)
	}

	if len(cpes) == 0 {
		return nil, nil
	}

	var recentVulns map[string][]string
	if collectVulns {
		recentVulns = make(map[string][]string)
	}
	for _, file := range files {
		err := checkCVEs(ctx, ds, logger, cpes, file, recentVulns, recentVulnerabilityMaxAge)
		if err != nil {
			return nil, err
		}
	}

	return recentVulns, nil
}

func checkCVEs(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	cpes []*wfn.Attributes,
	file string,
	recentVulns map[string][]string,
	recentVulnMaxAge time.Duration,
) error {
	dict, err := cvefeed.LoadJSONDictionary(file)
	if err != nil {
		return err
	}
	cache := cvefeed.NewCache(dict).SetRequireVersion(true).SetMaxSize(-1)
	// This index consumes too much RAM
	// cache.Idx = cvefeed.NewIndex(dict)

	cpeCh := make(chan *wfn.Attributes)
	collectVulns := recentVulns != nil

	var wg sync.WaitGroup
	var mu sync.Mutex
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		goRoutineKey := i
		go func() {
			defer wg.Done()

			logKey := fmt.Sprintf("cpe-processing-%d", goRoutineKey)
			level.Debug(logger).Log(logKey, "start")

			for {
				select {
				case cpe, more := <-cpeCh:
					if !more {
						level.Debug(logger).Log(logKey, "done")
						return
					}
					cacheHits := cache.Get([]*wfn.Attributes{cpe})
					for _, matches := range cacheHits {
						ml := len(matches.CPEs)
						if ml == 0 {
							continue
						}

						cveID := matches.CVE.ID()
						matchingCPEs := make([]string, 0, ml)
						for _, attr := range matches.CPEs {
							if attr == nil {
								level.Error(logger).Log("matches nil CPE", cveID)
								continue
							}
							cpe := attr.BindToFmtString()
							if len(cpe) == 0 {
								continue
							}
							matchingCPEs = append(matchingCPEs, cpe)
						}

						newCount, err := ds.InsertCVEForCPE(ctx, cveID, matchingCPEs)
						if err != nil {
							level.Error(logger).Log("cpe processing", "error", "err", err)
							continue // do not report a recent vuln that failed to be inserted in the DB
						}

						// collect as recent vuln only if newCount > 0, otherwise we would send
						// webhook requests for the same vulnerability over and over again until
						// it is older than 2 days.
						if collectVulns && newCount > 0 {
							vuln, ok := matches.CVE.(*feednvd.Vuln)
							if !ok {
								level.Error(logger).Log("recent vuln", "unexpected type for Vuln interface", "cve", cveID,
									"type", fmt.Sprintf("%T", matches.CVE))
								continue
							}

							rawPubDate := vuln.Schema().PublishedDate
							if rawPubDate == "" {
								level.Error(logger).Log("recent vuln", "empty published date", "cve", cveID)
								continue
							}

							pubDate, err := time.Parse(publishedDateFmt, rawPubDate)
							if err != nil {
								level.Error(logger).Log("recent vuln", "unexpected published date format", "cve", cveID,
									"published_date", rawPubDate, "err", err)
								continue
							}

							// the second condition should only affect tests - to ignore pubDates in the future
							// when using a mocked current clock. When using the real clock, the published date
							// should always be in the past.
							if theClock.Since(pubDate) <= recentVulnMaxAge && theClock.Now().After(pubDate) {
								mu.Lock()
								recentVulns[cveID] = append(recentVulns[cveID], matchingCPEs...)
								mu.Unlock()
							}
						}
					}
				case <-ctx.Done():
					level.Debug(logger).Log(logKey, "quitting")
					return
				}
			}
		}()
	}

	level.Debug(logger).Log("pushing cpes", "start")
	for _, cpe := range cpes {
		cpeCh <- cpe
	}
	close(cpeCh)

	level.Debug(logger).Log("pushing cpes", "done")

	wg.Wait()
	return nil
}

// PostProcess performs additional processing over the results of
// the main vulnerability processing run (TranslateSoftwareToCPE+TranslateCPEToCVE).
func PostProcess(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	logger kitlog.Logger,
	config config.FleetConfig,
) error {
	dbPath := filepath.Join(vulnPath, "cpe.sqlite")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open cpe database: %w", err)
	}
	defer db.Close()

	if err := centosPostProcessing(ctx, ds, db, logger, config); err != nil {
		return err
	}
	return nil
}
