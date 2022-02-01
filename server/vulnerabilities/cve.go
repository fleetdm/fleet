package vulnerabilities

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/facebookincubator/nvdtools/cvefeed"
	feednvd "github.com/facebookincubator/nvdtools/cvefeed/nvd"
	"github.com/facebookincubator/nvdtools/providers/nvd"
	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func SyncCVEData(vulnPath string, config config.FleetConfig) error {
	if config.Vulnerabilities.DisableDataSync {
		return nil
	}

	cve := nvd.SupportedCVE["cve-1.1.json.gz"]

	source := nvd.NewSourceConfig()
	if config.Vulnerabilities.CVEFeedPrefixURL != "" {
		parsed, err := url.Parse(config.Vulnerabilities.CVEFeedPrefixURL)
		if err != nil {
			return fmt.Errorf("parsing cve feed url prefix override: %w", err)
		}
		source.Host = parsed.Host
		source.Scheme = parsed.Scheme
	}

	dfs := nvd.Sync{
		Feeds:    []nvd.Syncer{cve},
		Source:   source,
		LocalDir: vulnPath,
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancelFunc()

	return dfs.Do(ctx)
}

const publishedDateFmt = "2006-01-02T15:04Z" // not quite RFC3339

var (
	rxNVDCVEArchive = regexp.MustCompile(`nvdcve.*\.gz$`)

	// max age to be considered a recent vulnerability (relative to NVD's published date)
	// (a var to be able to change in tests)
	recentVulnMaxAge = 2 * 24 * time.Hour

	// this allows mocking the time package for tests, by default it is equivalent
	// to the time functions, e.g. theClock.Now() == time.Now().
	theClock clock.Clock = clock.C
)

// TranslateCPEToCVE maps the CVEs found in NVD archive files in the
// vulnerabilities database folder to software CPEs in the fleet database.
// If collectRecentVulns is true, it also returns a mapping of recent CVEs
// to a list of CPEs affected by the CVE, otherwise that map is nil.
func TranslateCPEToCVE(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	logger kitlog.Logger,
	config config.FleetConfig,
	collectRecentVulns bool,
) (map[string][]string, error) {
	err := SyncCVEData(vulnPath, config)
	if err != nil {
		return nil, err
	}

	var files []string
	err = filepath.Walk(vulnPath, func(path string, info os.FileInfo, err error) error {
		if match := rxNVDCVEArchive.MatchString(path); !match {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, nil
	}

	cpeList, err := ds.AllCPEs(ctx)
	if err != nil {
		return nil, err
	}

	cpes := make([]*wfn.Attributes, 0, len(cpeList))
	for _, uri := range cpeList {
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
	if collectRecentVulns {
		recentVulns = make(map[string][]string)
	}
	for _, file := range files {
		err := checkCVEs(ctx, ds, logger, cpes, file, recentVulns)
		if err != nil {
			return nil, err
		}
	}

	return recentVulns, nil
}

func checkCVEs(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger,
	cpes []*wfn.Attributes, file string, recentVulns map[string][]string) error {

	dict, err := cvefeed.LoadJSONDictionary(file)
	if err != nil {
		return err
	}
	cache := cvefeed.NewCache(dict).SetRequireVersion(true).SetMaxSize(-1)
	// This index consumes too much RAM
	//cache.Idx = cvefeed.NewIndex(dict)

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

						matchingCPEs := make([]string, 0, ml)
						for _, attr := range matches.CPEs {
							if attr == nil {
								level.Error(logger).Log("matches nil CPE", matches.CVE.ID())
								continue
							}
							cpe := attr.BindToFmtString()
							if len(cpe) == 0 {
								continue
							}
							matchingCPEs = append(matchingCPEs, cpe)
						}

						err = ds.InsertCVEForCPE(ctx, matches.CVE.ID(), matchingCPEs)
						if err != nil {
							level.Error(logger).Log("cpe processing", "error", "err", err)
							continue // do not report a recent vuln that failed to be inserted in the DB
						}

						if collectVulns {
							vuln, ok := matches.CVE.(*feednvd.Vuln)
							if !ok {
								level.Error(logger).Log("recent vuln", "unexpected type for Vuln interface", "type", fmt.Sprintf("%T", matches.CVE))
								continue
							}

							// TODO(mna): I think this will trigger the new vulnerability every time we
							// run this? That's what Lucas is referring to here, but I don't see any
							// decision on how to handle that: https://github.com/fleetdm/fleet/issues/3050#issuecomment-1018487787
							// I think that'd be the simplest approach, to change InsertCVEForCPE to return
							// information about whether it created a new row, and report as recent vuln only
							// if that's the case.
							if rawPubDate := vuln.Schema().PublishedDate; rawPubDate != "" {
								pubDate, err := time.Parse(publishedDateFmt, rawPubDate)
								if err != nil {
									level.Error(logger).Log("recent vuln", "unexpected published date format", "published_date", rawPubDate, "err", err)
									continue
								}
								// the second condition should only affect tests - to ignore pubDates in the future
								// when using a mocked current clock. When using the real clock, the published date
								// should always be in the past.
								if theClock.Since(pubDate) <= recentVulnMaxAge && theClock.Now().After(pubDate) {
									mu.Lock()
									recentVulns[matches.CVE.ID()] = matchingCPEs
									mu.Unlock()
								}
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
