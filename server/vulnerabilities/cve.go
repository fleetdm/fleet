package vulnerabilities

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/facebookincubator/nvdtools/cvefeed"
	"github.com/facebookincubator/nvdtools/providers/nvd"
	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func syncCVEData() error {
	cve := nvd.SupportedCVE["cve-1.1.json.gz"]

	source := nvd.NewSourceConfig()
	localdir, err := os.Getwd()
	if err != nil {
		return err
	}

	dfs := nvd.Sync{
		Feeds:    []nvd.Syncer{cve},
		Source:   source,
		LocalDir: localdir,
	}

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Minute)

	return dfs.Do(ctx)
}

func TranslateCPEToCVE(ctx context.Context, ds fleet.Datastore, vulnPath string, logger kitlog.Logger) error {
	err := syncCVEData()
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

	var files []string
	err = filepath.Walk(vulnPath, func(path string, info os.FileInfo, err error) error {
		if match, err := regexp.MatchString("^nvdcve-1.1-.*.gz$", path); !match || err != nil {
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

	counter := new(uint64)
	total := uint64(len(cpeList))

	cancelCtx, cancelFunc := context.WithCancel(ctx)

	for i := 0; i < 4; i++ {
		go func() {
			level.Debug(logger).Log("cpe processing", "start")

			accumulated := 0
			var args []interface{}

			for {
				select {
				case cpe := <-cpeCh:
					for _, matches := range cache.Get([]*wfn.Attributes{cpe}) {
						ml := len(matches.CPEs)
						if ml == 0 {
							continue
						}
						matchingCPEs := make([]string, ml)
						for _, attr := range matches.CPEs {
							if attr == nil {
								level.Error(logger).Log("matches nil CPE", matches.CVE.ID())
								continue
							}
							matchingCPEs = append(matchingCPEs, attr.BindToFmtString())
						}
						//cveMatches.Store(matches.CVE.ID(), matchingCPEs)
					}

					if atomic.CompareAndSwapUint64(counter, total, total) {
						cancelFunc()
						level.Debug(logger).Log("cpe processing", "done")
						return
					}

					atomic.AddUint64(counter, uint64(1))
				case <-ctx.Done():
					level.Debug(logger).Log("cpe processing", "quitting")
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
