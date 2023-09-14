package nvd

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

	"github.com/Masterminds/semver"
	"github.com/facebookincubator/nvdtools/cvefeed"
	feednvd "github.com/facebookincubator/nvdtools/cvefeed/nvd"
	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
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
	periodicity time.Duration,
) ([]fleet.SoftwareVulnerability, error) {
	files, err := getNVDCVEFeedFiles(vulnPath)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}

	// get all the software CPEs from the database
	CPEs, err := ds.ListSoftwareCPEs(ctx)
	if err != nil {
		return nil, err
	}

	// hydrate the CPEs with the meta data
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

	knownNVDBugRules, err := GetKnownNVDBugRules()
	if err != nil {
		return nil, err
	}

	// we are using a map here to remove any duplicates - a vulnerability can be present in more than one
	// NVD feed file.
	vulns := make(map[string]fleet.SoftwareVulnerability)
	for _, file := range files {

		foundVulns, err := checkCVEs(
			ctx,
			ds,
			logger,
			parsed,
			file,
			collectVulns,
			knownNVDBugRules,
		)
		if err != nil {
			return nil, err
		}

		for _, e := range foundVulns {
			vulns[e.Key()] = e
		}
	}

	var newVulns []fleet.SoftwareVulnerability
	for _, vuln := range vulns {
		ok, err := ds.InsertSoftwareVulnerability(ctx, vuln, fleet.NVDSource)
		if err != nil {
			level.Error(logger).Log("cpe processing", "error", "err", err)
			continue
		}

		// collect vuln only if inserted, otherwise we would send
		// webhook requests for the same vulnerability over and over again until
		// it is older than 2 days.
		if collectVulns && ok {
			newVulns = append(newVulns, vuln)
		}
	}

	// Delete any stale vulnerabilities. A vulnerability is stale iff the last time it was
	// updated was more than `2 * periodicity` ago. This assumes that the whole vulnerability
	// process completes in less than `periodicity` units of time.
	//
	// This is used to get rid of false positives once they are fixed and no longer detected as vulnerabilities.
	if err = ds.DeleteOutOfDateVulnerabilities(ctx, fleet.NVDSource, 2*periodicity); err != nil {
		level.Error(logger).Log("msg", "error deleting out of date vulnerabilities", "err", err)
	}

	return newVulns, nil
}

func checkCVEs(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	softwareCPEs []softwareCPEWithNVDMeta,
	file string,
	collectVulns bool,
	knownNVDBugRules CPEMatchingRules,
) ([]fleet.SoftwareVulnerability, error) {
	dict, err := cvefeed.LoadJSONDictionary(file)
	if err != nil {
		return nil, err
	}

	cache := cvefeed.NewCache(dict).SetRequireVersion(true).SetMaxSize(-1)
	// This index consumes too much RAM
	// cache.Idx = cvefeed.NewIndex(dict)

	softwareCPECh := make(chan softwareCPEWithNVDMeta)
	var foundVulns []fleet.SoftwareVulnerability

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
						if len(matches.CPEs) == 0 {
							continue
						}

						if rule, ok := knownNVDBugRules.FindMatch(
							matches.CVE.ID(),
						); ok {
							if !rule.CPEMatches(softwareCPE.meta) {
								continue
							}
						}

						resolvedVersion, err := getMatchingVersionEndExcluding(matches.CVE.ID(), softwareCPE.meta, dict)
						if err != nil {
							level.Error(logger).Log(logKey, "error", "err", err)
						}

						vuln := fleet.SoftwareVulnerability{
							SoftwareID:        softwareCPE.SoftwareID,
							CVE:               matches.CVE.ID(),
							ResolvedInVersion: resolvedVersion,
						}

						mu.Lock()
						foundVulns = append(foundVulns, vuln)
						mu.Unlock()

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

	fmt.Println("checkCVE foundVulns", foundVulns)
	return foundVulns, nil
}

func getMatchingVersionEndExcluding(cve string, hostSoftwareMeta *wfn.Attributes, dict cvefeed.Dictionary) (string, error) {
	vuln, ok := dict[cve].(*feednvd.Vuln)
	if !ok {
		fmt.Println("msg", "Vuln interface", "cve", cve, "type", fmt.Sprintf("%T", dict[cve])) // TODO remove
		return "", nil
	}

	vulnSchema := vuln.Schema()
	if vulnSchema == nil {
		fmt.Println("msg", "vulnSchema is nil", "cve", cve) // TODO remove
		return "", nil
	}

	config := vulnSchema.Configurations
	if config == nil {
		fmt.Println("msg", "Configurations is nil", "cve", cve) // TODO remove
		return "", nil
	}

	nodes := config.Nodes
	if len(nodes) == 0 {
		fmt.Println("msg", "Nodes is nil", "cve", cve) // TODO remove
		return "", nil
	}

	cpeMatch := findCPEMatch(nodes)
	if cpeMatch == nil || len(cpeMatch) == 0 {
		fmt.Println("msg", "CPEMatch is nil or empty", "cve", cve) // TODO remove
		return "", nil
	}

	softwareVersion, err := semver.NewVersion(wfn.StripSlashes(hostSoftwareMeta.Version))
	if err != nil {
		fmt.Println("msg", "error parsing host software version", "cve", cve, "version", hostSoftwareMeta.Version) // TODO remove
		return "", err
	}

	for _, rule := range cpeMatch {
		if rule.VersionEndExcluding == "" {
			fmt.Println("msg", "VersionEndExcluding is empty", "cve", cve)
			continue
		}

		// convert the NVD cpe23URi to wfn.Attributes
		attr, err := wfn.Parse(rule.Cpe23Uri)
		if err != nil {
			fmt.Println("msg", "error parsing cpe23Uri", "cve", cve, "cpe23Uri", rule.Cpe23Uri)
			return "", err
		}

		// ensure the product and vendor match
		if attr.Product != hostSoftwareMeta.Product || attr.Vendor != hostSoftwareMeta.Vendor {
			fmt.Println("msg", "product or vendor do not match", "cve", cve, "product", attr.Product, "vendor", attr.Vendor)
			continue
		}

		if rule.VersionStartIncluding == "" && rule.VersionStartExcluding == "" {
			fmt.Println("msg", "VersionStartIncluding and VersionStartExcluding are empty, returning VersionEndExcluding", "cve", cve)
			return rule.VersionEndExcluding, nil
		}

		if rule.VersionStartIncluding != "" {
			constraint, err := semver.NewConstraint(">= " + rule.VersionStartIncluding + " < " + rule.VersionEndExcluding)
			if err != nil {
				fmt.Println("msg", "error parsing constraint", "cve", cve, "constraint", ">= "+rule.VersionStartIncluding+" < "+rule.VersionEndExcluding)
				return "", err
			}

			if constraint.Check(softwareVersion) {
				fmt.Println("msg", "constraint check passed", "cve", cve, "constraint", ">= "+rule.VersionStartIncluding+" < "+rule.VersionEndExcluding)
				return rule.VersionEndExcluding, nil
			} else {
				fmt.Println("msg", "constraint check failed", "cve", cve, "constraint", ">= "+rule.VersionStartIncluding+" < "+rule.VersionEndExcluding)
			}
		}

		if rule.VersionStartExcluding != "" {
			constraint, err := semver.NewConstraint("> " + rule.VersionStartExcluding + " < " + rule.VersionEndExcluding)
			if err != nil {
				fmt.Println("msg", "error parsing constraint", "cve", cve, "constraint", "> "+rule.VersionStartExcluding+" < "+rule.VersionEndExcluding)
				return "", err
			}

			if constraint.Check(softwareVersion) {
				fmt.Println("msg", "constraint check passed", "cve", cve, "constraint", "> "+rule.VersionStartExcluding+" < "+rule.VersionEndExcluding)
				return rule.VersionEndExcluding, nil
			} else {
				fmt.Println("msg", "constraint check failed", "cve", cve, "constraint", "> "+rule.VersionStartExcluding+" < "+rule.VersionEndExcluding)
			}
		}
	}

	fmt.Println("msg", "no matching rule found", "cve", cve)

	return "", nil
}

func findCPEMatch(nodes []*schema.NVDCVEFeedJSON10DefNode) []*schema.NVDCVEFeedJSON10DefCPEMatch {
	for _, node := range nodes {
		if len(node.CPEMatch) > 0 {
			return node.CPEMatch
		}

		if len(node.Children) > 0 {
			match := findCPEMatch(node.Children)
			if match != nil {
				return match
			}
		}
	}
	return nil
}
