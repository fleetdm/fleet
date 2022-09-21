package vulnerabilities

import (
	"context"
	"fmt"
	"io/fs"
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
	"github.com/fleetdm/fleet/v4/server/fleet"
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

var rxNVDCVEArchive = regexp.MustCompile(`nvdcve.*\.gz$`)

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

type softwareCPEWithNVDMeta struct {
	fleet.SoftwareCPE
	meta *wfn.Attributes
}

// TranslateCPEToCVE maps the CVEs found in NVD archive files in the
// vulnerabilities database folder to software CPEs in the fleet database.
// If collectVulns is true, returns a list of any new software vulnerabilities found.
func TranslateCPEToCVE(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	logger kitlog.Logger,
	collectVulns bool,
) ([]fleet.SoftwareVulnerability, error) {
	files, err := getNVDCVEFeedFiles(vulnPath)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}

	CPEs, err := ds.ListSoftwareCPEs(ctx)
	if err != nil {
		return nil, err
	}

	var parsed []softwareCPEWithNVDMeta
	for _, CPE := range CPEs {
		attr, err := wfn.Parse(CPE.CPE)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, softwareCPEWithNVDMeta{
			SoftwareCPE: CPE,
			meta:        attr,
		})
	}
	if len(parsed) == 0 {
		return nil, nil
	}

	var vulns []fleet.SoftwareVulnerability
	for _, file := range files {
		r, err := checkCVEs(ctx, ds, logger, parsed, file, collectVulns)
		if err != nil {
			return nil, err
		}

		vulns = append(vulns, r...)
	}

	return vulns, nil
}

func checkCVEs(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	softwareCPEs []softwareCPEWithNVDMeta,
	file string,
	collectVulns bool,
) ([]fleet.SoftwareVulnerability, error) {
	dict, err := cvefeed.LoadJSONDictionary(file)
	if err != nil {
		return nil, err
	}

	cache := cvefeed.NewCache(dict).SetRequireVersion(true).SetMaxSize(-1)
	// This index consumes too much RAM
	// cache.Idx = cvefeed.NewIndex(dict)

	softwareCPECh := make(chan softwareCPEWithNVDMeta)

	var results []fleet.SoftwareVulnerability
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
				case softwareCPE, more := <-softwareCPECh:
					if !more {
						level.Debug(logger).Log(logKey, "done")
						return
					}

					cacheHits := cache.Get([]*wfn.Attributes{softwareCPE.meta})
					for _, matches := range cacheHits {
						ml := len(matches.CPEs)
						if ml == 0 {
							continue
						}

						matchingVulns := make([]fleet.SoftwareVulnerability, 0, ml)
						cveID := matches.CVE.ID()
						for _, attr := range matches.CPEs {
							if attr == nil {
								level.Error(logger).Log("matches nil CPE", cveID)
								continue
							}
							matchingVulns = append(matchingVulns, fleet.SoftwareVulnerability{
								SoftwareID: softwareCPE.SoftwareID,
								CVE:        cveID,
							})
						}

						newCount, err := ds.InsertSoftwareVulnerabilities(ctx, matchingVulns, fleet.NVDSource)
						if err != nil {
							level.Error(logger).Log("cpe processing", "error", "err", err)
							continue // do not report a recent vuln that failed to be inserted in the DB
						}

						// collect vuln only if newCount > 0, otherwise we would send
						// webhook requests for the same vulnerability over and over again until
						// it is older than 2 days.
						if collectVulns && newCount > 0 {
							_, ok := matches.CVE.(*feednvd.Vuln)
							if !ok {
								level.Error(logger).Log(
									"recent vuln", "unexpected type for Vuln interface",
									"cve", cveID,
									"type", fmt.Sprintf("%T", matches.CVE))
								continue
							}

							mu.Lock()
							results = append(results, matchingVulns...)
							mu.Unlock()
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
	for _, cpe := range softwareCPEs {
		softwareCPECh <- cpe
	}
	close(softwareCPECh)

	level.Debug(logger).Log("pushing cpes", "done")

	wg.Wait()
	return results, nil
}
