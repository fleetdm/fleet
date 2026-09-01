package nvd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	nvdsync "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/sync"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed"
	feednvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/providers/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
	"github.com/google/go-github/v37/github"
)

const (
	vulnRepo = "vulnerabilities"
)

// DownloadNVDCVEFeed downloads CVEs information from the NVD 2.0 API
// and supplements the data with CPE information from the Vulncheck API.
// This is used to download CVE information to vulnPath.
func GenerateCVEFeeds(vulnPath string, debug bool, logger *slog.Logger) error {
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

func DownloadCVEFeed(vulnPath, cveFeedPrefixURL string, debug bool, logger *slog.Logger) error {
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
	logger *slog.Logger,
	collectVulns bool,
	startTime time.Time,
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

	sink := newVulnSink(ds, logger, collectVulns)
	var feedErr error
	for _, file := range files {
		if err := checkCVEs(
			ctx,
			logger,
			interfaceParsed,
			file,
			knownNVDBugRules,
			sink,
		); err != nil {
			// Keep scanning the remaining files: earlier chunks are already
			// committed, so aborting would drop their collected results and
			// suppress their automations forever (the next run's inserts
			// would classify them as already known).
			logger.ErrorContext(ctx, "error checking cves", "file", file, "err", err)
			if feedErr == nil {
				feedErr = ctxerr.Wrap(ctx, err, "checking cves")
			}
		}
	}
	res := sink.flush(ctx)

	// Detect corrupted/empty CVE feeds. If we had CPE/OS inputs to match against but produced
	// zero results across every feed file, the feed is almost certainly empty or corrupted
	// (e.g., a failed/corrupted artifact from GitHub) — skip the deletes so we don't wipe
	// legitimate existing software_cve rows that will be re-matched on the next good sync.
	feedProducedNoData := res.totalSoftware == 0 && res.totalOS == 0

	// Delete any stale vulnerabilities. A vulnerability is stale iff the last time it was
	// updated was more than `2 * periodicity` ago. This assumes that the whole vulnerability
	// process completes in less than `periodicity` units of time.
	//
	// This is used to get rid of false positives once they are fixed and no longer detected as vulnerabilities.
	// Skip cleanup when the corresponding insert failed to avoid deleting data with nothing to replace it.
	// With part of the corpus unread (feedErr), rows not re-matched this run
	// can't be assumed stale, so skip the deletes entirely.
	if feedErr == nil && res.softwareErr == nil && !feedProducedNoData {
		if err = ds.DeleteOutOfDateVulnerabilities(ctx, fleet.NVDSource, startTime); err != nil {
			logger.ErrorContext(ctx, "error deleting out of date vulnerabilities", "err", err)
		}
	}
	if feedErr == nil && res.osErr == nil && !feedProducedNoData {
		if err = ds.DeleteOutOfDateOSVulnerabilities(ctx, fleet.NVDSource, startTime); err != nil {
			logger.ErrorContext(ctx, "error deleting out of date OS vulnerabilities", "err", err)
		}
	}
	if feedProducedNoData {
		logger.ErrorContext(ctx, "NVD scan produced no matches with non-empty input; skipping deletes to preserve existing software_cve rows (feed may be corrupted)",
			"software_cpes", len(parsed), "os_cpes", len(cpes), "feed_files", len(files))
	}

	return res.collected, feedErr
}

// matchFlushSize is the default for how many buffered matches trigger a
// datastore flush. A var so tests can lower it.
var matchFlushSize = 100_000

// vulnSink receives matched vulnerabilities from the checkCVEs workers and
// flushes them to the datastore in bounded chunks, so matches never accumulate
// in memory.
// Duplicates are deduped within each chunk and in the collected results;
// across chunks the inserts are upserts on the unique key. Insert errors are
// logged and sticky so callers can skip the stale-row deletes for the cycle.
type vulnSink struct {
	// mu guards every field below and is held for the datastore flush itself.
	// This is intentional backpressure so matcher workers block while a
	// chunk is being written instead of buffering ahead of a slow insert.
	mu        sync.Mutex
	ds        fleet.Datastore
	logger    *slog.Logger
	collect   bool
	flushSize int

	softwareBuf []fleet.SoftwareVulnerability
	osBuf       []fleet.OSVulnerability

	collectedSeen map[string]struct{}
	res           vulnSinkResult
}

type vulnSinkResult struct {
	// collected is only complete for a single successful process lifetime: if
	// a previous run was killed after a chunk was inserted but before it
	// returned, that chunk's pairs are already in the datastore and will be
	// classified as not-new on the next run, so their automations are never
	// retried. Same tradeoff as a mid-corpus feed error, just with no error to
	// react to — accepted as out of scope for this run-scoped sink.
	collected     []fleet.SoftwareVulnerability
	totalSoftware int
	totalOS       int
	softwareErr   error
	osErr         error
}

func newVulnSink(ds fleet.Datastore, logger *slog.Logger, collect bool) *vulnSink {
	s := &vulnSink{ds: ds, logger: logger, collect: collect, flushSize: matchFlushSize}
	if collect {
		s.collectedSeen = make(map[string]struct{})
	}
	return s
}

func (s *vulnSink) addSoftware(ctx context.Context, v fleet.SoftwareVulnerability) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.softwareBuf = append(s.softwareBuf, v)
	s.res.totalSoftware++
	if len(s.softwareBuf) >= s.flushSize {
		s.flushSoftwareLocked(ctx)
	}
}

func (s *vulnSink) addOS(ctx context.Context, v fleet.OSVulnerability) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.osBuf = append(s.osBuf, v)
	s.res.totalOS++
	if len(s.osBuf) >= s.flushSize {
		s.flushOSLocked(ctx)
	}
}

func (s *vulnSink) flush(ctx context.Context) vulnSinkResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushSoftwareLocked(ctx)
	s.flushOSLocked(ctx)
	return s.res
}

func (s *vulnSink) flushSoftwareLocked(ctx context.Context) {
	if len(s.softwareBuf) == 0 {
		return
	}
	// Dedupe in place: a pair can be matched by more than one rule.
	seen := make(map[string]struct{}, len(s.softwareBuf))
	batch := s.softwareBuf[:0]
	for _, v := range s.softwareBuf {
		if _, ok := seen[v.Key()]; ok {
			continue
		}
		seen[v.Key()] = struct{}{}
		batch = append(batch, v)
	}
	s.softwareBuf = s.softwareBuf[:0]

	newVulns, err := s.ds.InsertSoftwareVulnerabilities(ctx, batch, fleet.NVDSource)
	if err != nil {
		s.logger.ErrorContext(ctx, "cpe processing error", "err", err)
		s.res.softwareErr = err
		return
	}
	if s.collect {
		// The datastore's existence check reads from the replica, so a pair
		// straddling a flush boundary can be reported as new twice under
		// replication lag; collectedSeen keeps the collected results unique.
		for _, v := range newVulns {
			if _, ok := s.collectedSeen[v.Key()]; ok {
				continue
			}
			s.collectedSeen[v.Key()] = struct{}{}
			s.res.collected = append(s.res.collected, v)
		}
	}
}

func (s *vulnSink) flushOSLocked(ctx context.Context) {
	if len(s.osBuf) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(s.osBuf))
	batch := s.osBuf[:0]
	for _, v := range s.osBuf {
		if _, ok := seen[v.Key()]; ok {
			continue
		}
		seen[v.Key()] = struct{}{}
		batch = append(batch, v)
	}
	s.osBuf = s.osBuf[:0]

	if _, err := s.ds.InsertOSVulnerabilities(ctx, batch, fleet.NVDSource); err != nil {
		s.logger.ErrorContext(ctx, "cpe processing error", "err", err)
		s.res.osErr = err
	}
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
			versionParts := strings.Split(os.Version, ".")
			if len(versionParts) == 2 {
				// Vulncheck reports versions with all 3 parts, so pad with an extra 0 if we only
				// have 2 parts (15.3 -> 15.3.0)
				versionParts = append(versionParts, "0")
				os.Version = strings.Join(versionParts, ".")
			}
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
	logger *slog.Logger,
	cpeItems []itemWithNVDMeta,
	jsonFile string,
	knownNVDBugRules CPEMatchingRules,
	sink *vulnSink,
) error {
	dict, err := cvefeed.LoadJSONDictionary(jsonFile)
	if err != nil {
		return err
	}

	// Group dictionary by vendor using a map.
	// This is done to speed up the matching process (PR https://github.com/fleetdm/fleet/pull/17298).
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
		// Build a product index per vendor so that cache.Get() narrows the
		// dictionary to only CVEs matching the queried product name before
		// running version-range matching.
		idx := cvefeed.NewIndex(subDict)
		cache := cvefeed.NewCache(subDict).SetRequireVersion(true).SetMaxSize(-1)
		cache.Idx = idx
		cacheGrouped[vendor] = cache
	}

	CPEItemCh := make(chan itemWithNVDMeta)

	var wg sync.WaitGroup

	logger = logger.With("json_file", jsonFile)

	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		goRoutineKey := i
		go func() {
			defer wg.Done()

			logger := logger.With("routine", goRoutineKey)
			logger.DebugContext(ctx, "start")

			for {
				select {
				case CPEItem, more := <-CPEItemCh:
					if !more {
						logger.DebugContext(ctx, "done")
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
								logger.DebugContext(ctx, "version end excluding error", "err", err)
							}

							if _, ok := CPEItem.(softwareCPEWithNVDMeta); ok {
								sink.addSoftware(ctx, fleet.SoftwareVulnerability{
									SoftwareID:        CPEItem.GetID(),
									CVE:               matches.CVE.ID(),
									ResolvedInVersion: &resolvedVersion,
								})
							} else if _, ok := CPEItem.(osCPEWithNVDMeta); ok {
								sink.addOS(ctx, fleet.OSVulnerability{
									OSID:              CPEItem.GetID(),
									CVE:               matches.CVE.ID(),
									ResolvedInVersion: &resolvedVersion,
								})
							}

						}
					}
				case <-ctx.Done():
					logger.DebugContext(ctx, "quitting")
					return
				}
			}
		}()
	}

	logger.DebugContext(ctx, "pushing cpes")

	for _, cpe := range cpeItems {
		CPEItemCh <- cpe
	}
	close(CPEItemCh)
	logger.DebugContext(ctx, "cpes pushed")
	wg.Wait()

	return nil
}

var pythonVersionWithUpdate = regexp.MustCompile(`(alpha|beta|rc)(\d+)`)

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

	// 	cpe:2.3:a:microsoft:python_extension:2024.2.1:*:*:*:*:visual_studio_code:*:*
	//	cpe:2.3:a:microsoft:visual_studio_code:2024.2.1:*:*:*:*:python:*:*
	//	cpe:2.3:a:microsoft:python:2020.4.0:*:*:*:*:visual_studio_code:*:*
	for _, cpeItem := range cpeItems {
		if cpeItem.TargetSW == "visual_studio_code" &&
			cpeItem.Vendor == "microsoft" &&
			cpeItem.Product == "python_extension" {
			cpeItem2 := *cpeItem
			cpeItem2.Product = "visual_studio_code"
			cpeItem2.TargetSW = "python"
			cpeItems = append(cpeItems, &cpeItem2)

			cpeItem3 := *cpeItem
			cpeItem3.Product = "python"
			cpeItems = append(cpeItems, &cpeItem3)
		}
	}

	for _, cpeItem := range cpeItems {
		if cpeItem.Vendor == "oracle" && cpeItem.Product == "virtualbox" {
			cpeItem2 := *cpeItem
			cpeItem2.Product = "vm_virtualbox"
			cpeItems = append(cpeItems, &cpeItem2)
		}
	}

	// The NVD CPE dictionary contains an invalid CPE for Ipswitch WhatsUp with product="whatsup",
	// but CVE-2006-2354 references product="whatsup_professional".
	// See https://github.com/fleetdm/fleet/issues/32662.
	for _, cpeItem := range cpeItems {
		if cpeItem.Vendor == "ipswitch" && cpeItem.Product == "whatsup" {
			cpeItem2 := *cpeItem
			cpeItem2.Product = "whatsup_professional"
			cpeItems = append(cpeItems, &cpeItem2)
		}
	}

	// pgAdmin CVEs in NVD use target_sw=postgresql and product=pgadmin_4, but Fleet generates
	// CPEs with platform-based target_sw (macos, windows) and may use different product
	// names (pgadmin, pgadmin4). Add aliases with target_sw=postgresql and product name
	// variations to match NVD's criteria.
	// See https://github.com/fleetdm/fleet/issues/37957.
	for _, cpeItem := range cpeItems {
		if cpeItem.Vendor == "pgadmin" &&
			(cpeItem.Product == "pgadmin_4" || cpeItem.Product == "pgadmin" || cpeItem.Product == "pgadmin4") {
			// Add aliases with product name variations and target_sw=postgresql
			for _, productName := range []string{"pgadmin", "pgadmin_4", "pgadmin4"} {
				newItem := *cpeItem
				newItem.Product = productName
				newItem.TargetSW = "postgresql"
				cpeItems = append(cpeItems, &newItem)
			}
		}
	}

	// Python pre-release versions can have the pre-release part in the version field or in the
	// update field (the technically correct place). We generate the "correct" CPEs (with the
	// pre-release part in the update field), so we have to create an alias here with the
	// pre-release part in the version field to cover all the cases.
	// e.g. Python 3.14.0 alpha2 can be represented as both:
	// 1. cpe:2.3:a:python:python:3.14.0:alpha2:*:*:*:windows:*:*
	// 2. cpe:2.3:a:python:python:3.14.0a2:*:*:*:*:windows:*:*
	// We generate CPEs like 1, but in the feed (e.g. Vulncheck) it can also appear as 2.
	// See https://github.com/fleetdm/fleet/issues/25882.
	for _, cpeItem := range cpeItems {
		if cpeItem.Vendor == "python" &&
			cpeItem.Product == "python" &&
			cpeItem.Update != "" &&
			pythonVersionWithUpdate.MatchString(cpeItem.Update) {

			cpeItem2 := *cpeItem
			for _, submatches := range pythonVersionWithUpdate.FindAllStringSubmatchIndex(cpeItem2.Update, -1) {
				prefixBytes := []byte{}
				numberBytes := []byte{}
				prefixBytes = pythonVersionWithUpdate.ExpandString(prefixBytes, "${1}", cpeItem.Update, submatches)
				numberBytes = pythonVersionWithUpdate.ExpandString(numberBytes, "${2}", cpeItem.Update, submatches)
				var prefix string
				switch prefixBytes[0] {
				case 'a':
					prefix = string(prefixBytes[0])
				case 'b':
					prefix = string(prefixBytes[0])
				case 'r':
					prefix = string(prefixBytes)
				}

				cpeItem2.Version = fmt.Sprintf("%s%s%s", cpeItem.Version, prefix, string(numberBytes))
				cpeItem2.Update = ""
			}

			cpeItems = append(cpeItems, &cpeItem2)
		}
	}

	return cpeItems
}

// Returns the versionEndExcluding string for the given CVE and host software meta
// data, if it exists in the NVD feed.  This effectively gives us the version of the
// software it needs to upgrade to in order to address the CVE.
func getMatchingVersionEndExcluding(ctx context.Context, cve string, hostSoftwareMeta *wfn.Attributes, dict cvefeed.Dictionary, logger *slog.Logger) (string, error) {
	vuln, ok := dict[cve].(*feednvd.Vuln)
	if !ok {
		return "", nil
	}

	// Schema() maps to the JSON schema of the NVD feed for a given CVE
	vulnSchema := vuln.Schema()
	if vulnSchema == nil {
		logger.ErrorContext(ctx, "error getting schema for CVE", "cve", cve)
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

	// Back slashes are added to the version string during parsing; remove them to ensure that the version
	// comparison works correctly. See https://github.com/fleetdm/fleet/issues/25991.
	hostSoftwareVersion := wfn.StripSlashes(hostSoftwareMeta.Version)

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
		if attr.SWEdition != wfn.Any && attr.SWEdition != hostSoftwareMeta.SWEdition &&
			!(hostSoftwareMeta.SWEdition == wfn.Any && attr.SWEdition == wfn.NA) {
			continue
		}

		// versionEnd is the version string that the vulnerable host software version must be less than
		versionEnd, err := checkVersion(rule, hostSoftwareVersion)
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
