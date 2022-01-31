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

var rxNVDCVEArchive = regexp.MustCompile(`nvdcve.*\.gz$`)

func TranslateCPEToCVE(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	logger kitlog.Logger,
	config config.FleetConfig,
	// TODO(mna): receive the enabled flag for vulnerability webhook, to indicate
	// if it should collect those new vulnerabilities during processing.
) error {
	err := SyncCVEData(vulnPath, config)
	if err != nil {
		return err
	}

	// TODO(mna): I assume those .gz NVD files get removed at some point, so we
	// don't unnecessarily process the same ones multiple times? Haven't seen
	// where that happens (e.g. doesn't seem to be in cronCleanups?)

	var files []string
	err = filepath.Walk(vulnPath, func(path string, info os.FileInfo, err error) error {
		if match := rxNVDCVEArchive.MatchString(path); !match {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	cpeList, err := ds.AllCPEs(ctx)
	if err != nil {
		return err
	}

	cpes := make([]*wfn.Attributes, 0, len(cpeList))
	for _, uri := range cpeList {
		attr, err := wfn.Parse(uri)
		if err != nil {
			return err
		}
		cpes = append(cpes, attr)
	}

	if len(cpes) == 0 {
		return nil
	}

	for _, file := range files {
		err := checkCVEs(ctx, ds, logger, cpes, file)
		if err != nil {
			return err
		}
	}

	// TODO(mna): return the collected CVE->CPEs map, as now returned by checkCVEs.
	return nil
}

func checkCVEs(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, cpes []*wfn.Attributes, files ...string) error {
	dict, err := cvefeed.LoadJSONDictionary(files...)
	if err != nil {
		return err
	}
	cache := cvefeed.NewCache(dict).SetRequireVersion(true).SetMaxSize(-1)
	// This index consumes too much RAM
	//cache.Idx = cvefeed.NewIndex(dict)

	cpeCh := make(chan *wfn.Attributes)

	var wg sync.WaitGroup

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
						}

						vuln := matches.CVE.(*feednvd.Vuln)
						fmt.Println(">>>>> ", vuln.ID(), vuln.Schema().PublishedDate)
						// Example output: >>>>>  CVE-2012-6369 2012-12-28T11:48Z

						// TODO(mna): if CVE is within 2 days of its published date, and
						// webhook is enabled, collect the CVE and its matching CPEs. How
						// to get the CVE's published date is still TBD at this time.
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

	// TODO(mna): if recent CVE->CPEs were collected, return the list (possibly
	// map of *CVE - a new struct holding the CVE ID, details link to nvd, and
	// published date - to a slice of CPEs).
	return nil
}
