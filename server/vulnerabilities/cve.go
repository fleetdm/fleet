package vulnerabilities

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed"
	"github.com/facebookincubator/nvdtools/providers/nvd"
	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func syncCVEData(vulnPath string) error {
	cve := nvd.SupportedCVE["cve-1.1.json.gz"]

	source := nvd.NewSourceConfig()

	dfs := nvd.Sync{
		Feeds:    []nvd.Syncer{cve},
		Source:   source,
		LocalDir: vulnPath,
	}

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)

	return dfs.Do(ctx)
}

func TranslateCPEToCVE(ctx context.Context, ds fleet.Datastore, vulnPath string, logger kitlog.Logger) error {
	err := syncCVEData(vulnPath)
	if err != nil {
		return err
	}

	cpeList, err := ds.AllCPEs()
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

	var files []string
	err = filepath.Walk(vulnPath, func(path string, info os.FileInfo, err error) error {
		if match, err := regexp.MatchString("nvdcve.*\\.gz$", path); !match || err != nil {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return err
	}

	dict, err := cvefeed.LoadJSONDictionary(files...)
	if err != nil {
		return err
	}
	cache := cvefeed.NewCache(dict).SetRequireVersion(true).SetMaxSize(0)
	cache.Idx = cvefeed.NewIndex(dict)

	cpeCh := make(chan *wfn.Attributes)

	counter := 0
	counterLock := &sync.Mutex{}
	total := len(cpeList)

	cancelCtx, cancelFunc := context.WithCancel(ctx)

	for i := 0; i < runtime.NumCPU(); i++ {
		goRoutineKey := i
		go func() {
			logKey := fmt.Sprintf("cpe-processing-%d", goRoutineKey)
			level.Debug(logger).Log(logKey, "start")

			for {
				select {
				case cpe := <-cpeCh:
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
						err = ds.InsertCVEForCPE(matches.CVE.ID(), matchingCPEs)
						if err != nil {
							level.Error(logger).Log("cpe processing", "error", "err", err)
						}
					}

					doneProcessingCPEs := false
					counterLock.Lock()
					counter++
					if counter >= total {
						doneProcessingCPEs = true
					}
					counterLock.Unlock()

					if doneProcessingCPEs {
						cancelFunc()
						level.Debug(logger).Log(logKey, "done")
						return
					}
				case <-ctx.Done():
					level.Debug(logger).Log(logKey, "quitting")
					return
				case <-cancelCtx.Done():
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
	level.Debug(logger).Log("pushing cpes", "done")

	<-cancelCtx.Done()

	return nil
}
