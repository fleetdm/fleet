package nvd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	nvdsync "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/sync"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed"
	feednvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
	"github.com/go-kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/go-github/v37/github"
)

const (
	vulnRepo = "vulnerabilities"
)

// DownloadNVDCVEFeed downloads CVEs information from the NVD 2.0 API
// and supplements the data with CPE information from the Vulncheck API.
// This is used to download CVE information to vulnPath.
func GenerateCVEFeeds(vulnPath string, debug bool, logger log.Logger) error {
	cveSyncer, err := nvdsync.NewCVE(
		vulnPath,
		nvdsync.WithLogger(logger),
		nvdsync.WithDebug(debug),
	)
	if err != nil {
		return err
	}

	if err := cveSyncer.Do(context.Background()); err != nil {
		return fmt.Errorf("download nvd cve feed: %w", err)
	}

	if err := cveSyncer.DoVulnCheck(context.Background()); err != nil {
		return fmt.Errorf("download nvd cve feed: %w", err)
	}

	return nil
}

func DownloadCVEFeed(vulnPath, cveFeedPrefixURL string, debug bool, logger log.Logger) error {
	var err error

	if cveFeedPrefixURL == "" {
		cveFeedPrefixURL, err = GetGitHubCVEAssetPath()
		if err != nil {
			return fmt.Errorf("get cve asset path: %w", err)
		}
	}

	err = downloadNVDCVELegacy(vulnPath, cveFeedPrefixURL)
	if err != nil {
		return fmt.Errorf("download nvd cve feed: %w", err)
	}

	return nil
}

func GetGitHubCVEAssetPath() (string, error) {
	vulnOwner := os.Getenv("TEST_VULN_GITHUB_OWNER")
	if vulnOwner == "" {
		vulnOwner = owner
	}

	ghClient := github.NewClient(fleethttp.NewGithubClient())

	releases, _, err := ghClient.Repositories.ListReleases(
		context.Background(),
		vulnOwner,
		vulnRepo,
		&github.ListOptions{Page: 0, PerPage: 10},
	)
	if err != nil {
		return "", err
	}

	nvdregex := regexp.MustCompile(`cve-\d+`)
	var found string

	for _, release := range releases {
		// Skip draft releases
		if release.GetDraft() {
			continue
		}

		if nvdregex.MatchString(release.GetTagName()) {
			found = release.GetTagName()
			break
		}
	}

	if found == "" {
		return "", errors.New("no CVE feed found")
	}

	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/", vulnOwner, vulnRepo, found), nil
}

func downloadNVDCVELegacy(vulnPath string, cveFeedPrefixURL string) error {
	if cveFeedPrefixURL == "" {
		return errors.New("missing cve_feed_prefix_url")
	}

	source := nvd.NewSourceConfig()
	parsed, err := url.Parse(cveFeedPrefixURL)
	if err != nil {
		return fmt.Errorf("parsing cve feed url prefix override: %w", err)
	}
	source.Host = parsed.Host
	source.CVEFeedPath = parsed.Path
	source.Scheme = parsed.Scheme

	cve := nvd.SupportedCVE["cve-1.1.json.gz"]
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

var rxNVDCVEArchive = regexp.MustCompile(`nvdcve.*\.json.*$`)

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

// interface for items with NVD Meta Data
type itemWithNVDMeta interface {
	GetMeta() *wfn.Attributes
	GetID() uint
}

type softwareCPEWithNVDMeta struct {
	fleet.SoftwareCPE
	meta *wfn.Attributes
}

func (s softwareCPEWithNVDMeta) GetMeta() *wfn.Attributes {
	return s.meta
}

func (s softwareCPEWithNVDMeta) GetID() uint {
	return s.SoftwareID
}

type osCPEWithNVDMeta struct {
	fleet.OperatingSystem
	meta *wfn.Attributes
}

func (o osCPEWithNVDMeta) GetMeta() *wfn.Attributes {
	return o.meta
}

func (o osCPEWithNVDMeta) GetID() uint {
	return o.ID
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

	cpes, err := GetMacOSCPEs(ctx, ds)
	if err != nil {
		return nil, err
	}

	if len(parsed) == 0 && len(cpes) == 0 {
		return nil, nil
	}

	var interfaceParsed []itemWithNVDMeta
	for _, p := range parsed {
		interfaceParsed = append(interfaceParsed, p)
	}
	for _, c := range cpes {
		interfaceParsed = append(interfaceParsed, c)
	}

	knownNVDBugRules, err := GetKnownNVDBugRules()
	if err != nil {
		return nil, err
	}

	// we are using a map here to remove any duplicates - a vulnerability can be present in more than one
	// NVD feed file.
	softwareVulns := make(map[string]fleet.SoftwareVulnerability)
	osVulns := make(map[string]fleet.OSVulnerability)
	for _, file := range files {

		foundSoftwareVulns, foundOSVulns, err := checkCVEs(
			ctx,
			logger,
			interfaceParsed,
			file,
			knownNVDBugRules,
		)
		if err != nil {
			return nil, err
		}

		for _, e := range foundSoftwareVulns {
			softwareVulns[e.Key()] = e
		}
		for _, e := range foundOSVulns {
			osVulns[e.Key()] = e
		}
	}

	var newVulns []fleet.SoftwareVulnerability
	for _, vuln := range softwareVulns {
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

	for _, vuln := range osVulns {
		_, err := ds.InsertOSVulnerability(ctx, vuln, fleet.NVDSource)
		if err != nil {
			level.Error(logger).Log("cpe processing", "error", "err", err)
			continue
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
	if err = ds.DeleteOutOfDateOSVulnerabilities(ctx, fleet.NVDSource, 2*periodicity); err != nil {
		level.Error(logger).Log("msg", "error deleting out of date OS vulnerabilities", "err", err)
	}

	return newVulns, nil
}

// GetMacOSCPEs translates all found macOS Operating Systems to CPEs.
func GetMacOSCPEs(ctx context.Context, ds fleet.Datastore) ([]osCPEWithNVDMeta, error) {
	var cpes []osCPEWithNVDMeta

	oses, err := ds.ListOperatingSystemsForPlatform(ctx, "darwin")
	if err != nil {
		return cpes, ctxerr.Wrap(ctx, err, "list operating systems")
	}

	if len(oses) == 0 {
		return cpes, nil
	}

	// variants of macOS found in the NVD feed
	macosVariants := []string{"macos", "mac_os_x"}

	for _, os := range oses {
		for _, variant := range macosVariants {
			cpe := osCPEWithNVDMeta{
				OperatingSystem: os,
				meta: &wfn.Attributes{
					Part:      "o",
					Vendor:    "apple",
					Product:   variant,
					Version:   os.Version,
					Update:    wfn.Any,
					Edition:   wfn.Any,
					SWEdition: wfn.Any,
					TargetSW:  wfn.Any,
					TargetHW:  wfn.Any,
					Other:     wfn.Any,
					Language:  wfn.Any,
				},
			}
			cpes = append(cpes, cpe)
		}
	}

	return cpes, nil
}

func matchesExactTargetSW(softwareCPETargetSW string, targetSWs []string, configs []*wfn.Attributes) bool {
	for _, targetSW := range targetSWs {
		if softwareCPETargetSW == targetSW {
			for _, attr := range configs {
				if attr.TargetSW == targetSW {
					return true
				}
			}
		}
	}
	return false
}

func checkCVEs(
	ctx context.Context,
	logger kitlog.Logger,
	cpeItems []itemWithNVDMeta,
	jsonFile string,
	knownNVDBugRules CPEMatchingRules,
) ([]fleet.SoftwareVulnerability, []fleet.OSVulnerability, error) {
	dict, err := cvefeed.LoadJSONDictionary(jsonFile)
	if err != nil {
		return nil, nil, err
	}

	// Group dictionary by vendor using a map.
	// This is done to speed up the matching process (PR https://github.com/fleetdm/fleet/pull/17298).
	// A map uses a hash table to store the key-value pairs. By putting multiple vulnerabilities with the same vendor into a map,
	// we reduce the number of comparisons needed to find the vulnerabilities that match the CPEs. Specifically, we no longer need to
	// compare each CPE with each vulnerability, but only with the vulnerabilities that have the same vendor.
	// Further optimization can be done by also using a map for product name comparison.
	dictGrouped := make(map[string]cvefeed.Dictionary, len(dict))
	for key, vuln := range dict {
		attrsArray := vuln.Config()
		for _, attrs := range attrsArray {
			subDict, ok := dictGrouped[attrs.Vendor]
			if !ok {
				subDict = make(cvefeed.Dictionary, 1)
				dictGrouped[attrs.Vendor] = subDict
			}
			subDict[key] = vuln
		}
	}

	cacheGrouped := make(map[string]*cvefeed.Cache, len(dictGrouped))
	for vendor, subDict := range dictGrouped {
		cache := cvefeed.NewCache(subDict).SetRequireVersion(true).SetMaxSize(-1)
		cacheGrouped[vendor] = cache
	}

	CPEItemCh := make(chan itemWithNVDMeta)
	var foundSoftwareVulns []fleet.SoftwareVulnerability
	var foundOSVulns []fleet.OSVulnerability

	var wg sync.WaitGroup
	var softwareMu sync.Mutex
	var osMu sync.Mutex

	logger = log.With(logger, "json_file", jsonFile)

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		goRoutineKey := i
		go func() {
			defer wg.Done()

			logger := log.With(logger, "routine", goRoutineKey)
			level.Debug(logger).Log("msg", "start")

			for {
				select {
				case CPEItem, more := <-CPEItemCh:
					if !more {
						level.Debug(logger).Log("msg", "done")
						return
					}

					cache, ok := cacheGrouped[CPEItem.GetMeta().Vendor]
					if !ok {
						// No such vendor in the Vulnerability dictionary
						continue
					}

					cpeItemsWithAliases := expandCPEAliases(CPEItem.GetMeta())
					for _, cpeItem := range cpeItemsWithAliases {
						cacheHits := cache.Get([]*wfn.Attributes{cpeItem})
						for _, matches := range cacheHits {
							if len(matches.CPEs) == 0 {
								continue
							}

							if rule, ok := knownNVDBugRules.FindMatch(
								matches.CVE.ID(),
							); ok {
								if !rule.CPEMatches(cpeItem) {
									continue
								}
							}

							// For chrome/firefox extensions we only want to match vulnerabilities
							// that are reported explicitly for target_sw == "chrome" or target_sw = "firefox".
							//
							// Why? In many occasions the NVD dataset reports vulnerabilities in client applications
							// with target_sw == "*", meaning the client application is vulnerable on all operating systems.
							// Such rules we want to ignore here to prevent many false positives that do not apply to the
							// Chrome or Firefox environment.
							if cpeItem.TargetSW == "chrome" || cpeItem.TargetSW == "firefox" {
								if !matchesExactTargetSW(
									cpeItem.TargetSW,
									[]string{"chrome", "firefox"},
									matches.CVE.Config(),
								) {
									continue
								}
							}

							resolvedVersion, err := getMatchingVersionEndExcluding(ctx, matches.CVE.ID(), cpeItem, dict, logger)
							if err != nil {
								level.Debug(logger).Log("err", err)
							}

							if _, ok := CPEItem.(softwareCPEWithNVDMeta); ok {
								vuln := fleet.SoftwareVulnerability{
									SoftwareID:        CPEItem.GetID(),
									CVE:               matches.CVE.ID(),
									ResolvedInVersion: ptr.String(resolvedVersion),
								}

								softwareMu.Lock()
								foundSoftwareVulns = append(foundSoftwareVulns, vuln)
								softwareMu.Unlock()
							} else if _, ok := CPEItem.(osCPEWithNVDMeta); ok {

								vuln := fleet.OSVulnerability{
									OSID:              CPEItem.GetID(),
									CVE:               matches.CVE.ID(),
									ResolvedInVersion: ptr.String(resolvedVersion),
								}

								osMu.Lock()
								foundOSVulns = append(foundOSVulns, vuln)
								osMu.Unlock()
							}

						}
					}
				case <-ctx.Done():
					level.Debug(logger).Log("msg", "quitting")
					return
				}
			}
		}()
	}

	level.Debug(logger).Log("msg", "pushing cpes")

	for _, cpe := range cpeItems {
		CPEItemCh <- cpe
	}
	close(CPEItemCh)
	level.Debug(logger).Log("msg", "cpes pushed")
	wg.Wait()

	return foundSoftwareVulns, foundOSVulns, nil
}

// expandCPEAliases will generate new *wfn.Attributes from the given cpeItem.
// It returns a slice with the given cpeItem plus the generated *wfn.Attributes.
//
// We need this because entries in the CPE database are not consistent.
// E.g. some Visual Studio Code extensions are defined with target_sw=visual_studio_code
// and others are defined with target_sw=visual_studio.
// E.g. The python extension for Visual Studio Code is defined with
// product=python_extension,target_sw=visual_studio_code and with
// product=visual_studio_code,target_sw=python.
func expandCPEAliases(cpeItem *wfn.Attributes) []*wfn.Attributes {
	cpeItems := []*wfn.Attributes{cpeItem}

	// Some VSCode extensions are defined with target_sw=visual_studio_code
	// and others are defined with target_sw=visual_studio.
	for _, cpeItem := range cpeItems {
		if cpeItem.TargetSW == "visual_studio_code" {
			cpeItem2 := *cpeItem
			cpeItem2.TargetSW = "visual_studio"
			cpeItems = append(cpeItems, &cpeItem2)
		}
	}

	// The python extension is defined in two ways in the CPE database:
	// 	cpe:2.3:a:microsoft:python_extension:2024.2.1:*:*:*:*:visual_studio_code:*:*
	//	cpe:2.3:a:microsoft:visual_studio_code:2024.2.1:*:*:*:*:python:*:*
	for _, cpeItem := range cpeItems {
		if cpeItem.TargetSW == "visual_studio_code" &&
			cpeItem.Vendor == "microsoft" &&
			cpeItem.Product == "python_extension" {
			cpeItem2 := *cpeItem
			cpeItem2.Product = "visual_studio_code"
			cpeItem2.TargetSW = "python"
			cpeItems = append(cpeItems, &cpeItem2)
		}
	}

	for _, cpeItem := range cpeItems {
		if cpeItem.Vendor == "oracle" && cpeItem.Product == "virtualbox" {
			cpeItem2 := *cpeItem
			cpeItem2.Product = "vm_virtualbox"
			cpeItems = append(cpeItems, &cpeItem2)
		}
	}

	return cpeItems
}

// Returns the versionEndExcluding string for the given CVE and host software meta
// data, if it exists in the NVD feed.  This effectively gives us the version of the
// software it needs to upgrade to in order to address the CVE.
func getMatchingVersionEndExcluding(ctx context.Context, cve string, hostSoftwareMeta *wfn.Attributes, dict cvefeed.Dictionary, logger kitlog.Logger) (string, error) {
	vuln, ok := dict[cve].(*feednvd.Vuln)
	if !ok {
		return "", nil
	}

	// Schema() maps to the JSON schema of the NVD feed for a given CVE
	vulnSchema := vuln.Schema()
	if vulnSchema == nil {
		level.Error(logger).Log("msg", "error getting schema for CVE", "cve", cve)
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
	if len(cpeMatch) == 0 {
		return "", nil
	}

	// Check if the host software version matches any of the CPEMatch rules.
	// CPEMatch rules can include version strings for the following:
	// - versionStartIncluding
	// - versionStartExcluding
	// - versionEndExcluding
	// - versionEndIncluding - not used in this function as we don't want to assume the resolved version
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
		versionEnd, err := checkVersion(rule, hostSoftwareMeta.Version)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "checking version")
		}
		if versionEnd != "" {
			return versionEnd, nil
		}
	}

	return "", nil
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
func checkVersion(rule *schema.NVDCVEFeedJSON10DefCPEMatch, softwareVersionStr string) (string, error) {
	if rule.VersionStartIncluding == "" && rule.VersionStartExcluding == "" && rule.VersionEndExcluding == "" {
		return rule.VersionEndExcluding, nil
	}

	if rule.VersionStartIncluding == "" && rule.VersionStartExcluding == "" {
		// "softwareVersionStr < endExcluding",
		if feednvd.SmartVerCmp(softwareVersionStr, rule.VersionEndExcluding) == -1 {
			return rule.VersionEndExcluding, nil
		}
	}
	if rule.VersionStartIncluding != "" {
		// "softwareVersionStr >= startIncluding && softwareVersionStr < endExcluding"
		if (feednvd.SmartVerCmp(softwareVersionStr, rule.VersionStartIncluding) == 1 || feednvd.SmartVerCmp(softwareVersionStr, rule.VersionStartIncluding) == 0) &&
			feednvd.SmartVerCmp(softwareVersionStr, rule.VersionEndExcluding) == -1 {
			return rule.VersionEndExcluding, nil
		}
	}
	// "softwareVersionStr > startExcluding && softwareVersionStr < endExcluding"
	if feednvd.SmartVerCmp(softwareVersionStr, rule.VersionStartExcluding) == 1 && feednvd.SmartVerCmp(softwareVersionStr, rule.VersionEndExcluding) == -1 {
		return rule.VersionEndExcluding, nil
	}

	return "", nil
}
