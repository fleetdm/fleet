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
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver"
	"github.com/facebookincubator/nvdtools/cvefeed"
	feednvd "github.com/facebookincubator/nvdtools/cvefeed/nvd"
	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/facebookincubator/nvdtools/providers/nvd"
	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
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

						resolvedVersion, err := getMatchingVersionEndExcluding(ctx, matches.CVE.ID(), softwareCPE.meta, dict, file)
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

	return foundVulns, nil
}

// Returns the versionEndExcluding string for the given CVE and host software meta
// data, if it exists in the NVD feed.  This effectively gives us the version of the
// software it needs to upgrade to in order to address the CVE.
func getMatchingVersionEndExcluding(ctx context.Context, cve string, hostSoftwareMeta *wfn.Attributes, dict cvefeed.Dictionary, nvdfile string) (string, error) {
	vuln, ok := dict[cve].(*feednvd.Vuln)
	if !ok {
		return "", nil
	}

	// Schema() maps to the JSON schema of the NVD feed for a given CVE
	vulnSchema := vuln.Schema()
	if vulnSchema == nil {
		return "", nil
	}

	config := vulnSchema.Configurations
	if config == nil {
		return "", nil
	}

	nodes := config.Nodes
	if len(nodes) == 0 {
		return "", nil
	}

	cpeMatch := findCPEMatch(nodes)
	if cpeMatch == nil || len(cpeMatch) == 0 {
		return "", nil
	}

	// convert the host software version to semver for later comparison
	formattedVersion := preprocessVersion(wfn.StripSlashes(hostSoftwareMeta.Version))
	softwareVersion, err := semver.NewVersion(formattedVersion)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "parsing software version", hostSoftwareMeta.Product, hostSoftwareMeta.Version)
	}

	// Check if the host software version matches any of the CPEMatch rules.
	// CPEMatch rules can include version strings for the following:
	// - versionStartIncluding
	// - versionStartExcluding
	// - versionEndExcluding
	// - versionEndIncluding - not used in this function
	for _, rule := range cpeMatch {
		if rule.VersionEndExcluding == "" {
			continue
		}

		// convert the NVD cpe23URi to wfn.Attributes for later comparison
		attr, err := wfn.Parse(rule.Cpe23Uri)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "parsing cpe23Uri")
		}

		// ensure the product and vendor match
		if attr.Product != hostSoftwareMeta.Product || attr.Vendor != hostSoftwareMeta.Vendor {
			continue
		}

		// versionEnd is the version string that the vulnerable host software version must be less than
		versionEnd, err := checkVersion(ctx, rule, softwareVersion)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "checking version")
		}
		if versionEnd != "" {
			return versionEnd, nil
		}
	}

	return "", fmt.Errorf("getting resolved in version: no matching rule found for CVE: %s", cve)
}

// CPEMatch can be nested in Children nodes. Recursively search the nodes for a CPEMatch
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

// checkVersion checks if the host software version matches the CPEMatch rule
func checkVersion(ctx context.Context, rule *schema.NVDCVEFeedJSON10DefCPEMatch, softwareVersion *semver.Version) (string, error) {
	for _, condition := range []struct {
		startIncluding string
		startExcluding string
	}{
		{rule.VersionStartIncluding, ""},
		{"", rule.VersionStartExcluding},
	} {
		constraintStr := buildConstraintString(condition.startIncluding, condition.startExcluding, rule.VersionEndExcluding)
		if constraintStr == "" {
			return rule.VersionEndExcluding, nil
		}

		constraint, err := semver.NewConstraint(constraintStr)
		if err != nil {
			return "", ctxerr.Wrap(nil, err, "parsing constraint:", constraintStr)
		}

		if constraint.Check(softwareVersion) {
			return rule.VersionEndExcluding, nil
		}
	}

	return "", nil
}

// buildConstraintString builds a semver constraint string from the startIncluding,
// startExcluding, and endExcluding strings
func buildConstraintString(startIncluding, startExcluding, endExcluding string) string {
	if startIncluding == "" && startExcluding == "" {
		return ""
	}
	if startIncluding != "" {
		return fmt.Sprintf(">= %s, < %s", startIncluding, endExcluding)
	}
	return fmt.Sprintf("> %s, < %s", startExcluding, endExcluding)
}

// Products using 4 part versioning scheme (ie. docker desktop)
// need to be converted to 3 part versioning scheme (2.3.0.2 -> 2.3.0+3) for use with
// the semver library.
func preprocessVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + "." + parts[2] + "+" + parts[3]
	}
	return version
}
