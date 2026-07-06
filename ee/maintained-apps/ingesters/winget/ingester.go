package winget

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	external_refs "github.com/fleetdm/fleet/v4/ee/maintained-apps/ingesters/winget/external_refs"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/patch_policy"
	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	feednvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/google/go-github/v37/github"
	"gopkg.in/yaml.v2"
)

func IngestApps(ctx context.Context, logger *slog.Logger, inputsPath string, slugFilter string) ([]*maintained_apps.FMAManifestApp, error) {
	logger.InfoContext(ctx, "starting winget app data ingestion")
	// Read from our list of apps we should be ingesting
	files, err := os.ReadDir(inputsPath)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reading winget input data directory")
	}

	var manifestApps []*maintained_apps.FMAManifestApp

	githubHTTPClient := fleethttp.NewGithubClient()
	githubClient := github.NewClient(githubHTTPClient)
	opts := &github.RepositoryContentGetOptions{
		Ref: "master",
	}

	i := &wingetIngester{
		githubClient: githubClient,
		ghClientOpts: opts,
		logger:       logger,
		// Use the token-authenticated client (when NETWORK_TEST_GITHUB_TOKEN is set) for
		// raw.githubusercontent.com too: unauthenticated raw requests are rate-limited
		// per IP, which shared GitHub Actions runner IPs exhaust constantly.
		httpClient: githubHTTPClient,
		rawBaseURL: "https://raw.githubusercontent.com",
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		// Skip non-JSON files (e.g., .DS_Store on macOS)
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		fileBytes, err := os.ReadFile(path.Join(inputsPath, f.Name()))
		if err != nil {
			return nil, ctxerr.WrapWithData(ctx, err, "reading app input file", map[string]any{"file_name": f.Name()})
		}

		var input inputApp
		if err := json.Unmarshal(fileBytes, &input); err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "unmarshal app input file: %s", f.Name())
		}

		if input.Slug == "" {
			return nil, ctxerr.NewWithData(ctx, "missing slug for app", map[string]any{"file_name": f.Name()})
		}

		if input.UniqueIdentifier == "" {
			return nil, ctxerr.NewWithData(ctx, "missing unique identifier for app", map[string]any{"file_name": f.Name()})
		}

		if input.Name == "" {
			return nil, ctxerr.NewWithData(ctx, "missing name for app", map[string]any{"file_name": f.Name()})
		}

		if input.PackageIdentifier == "" {
			return nil, ctxerr.NewWithData(ctx, "missing package identifier for app", map[string]any{"file_name": f.Name()})
		}

		if slugFilter != "" && !strings.Contains(input.Slug, slugFilter) {
			continue
		}

		logger.InfoContext(ctx, "ingesting winget app", "name", input.Name)

		outApp, err := i.ingestOne(ctx, input)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "ingesting winget app")
		}

		manifestApps = append(manifestApps, outApp)
	}

	return manifestApps, nil
}

type wingetIngester struct {
	githubClient *github.Client
	ghClientOpts *github.RepositoryContentGetOptions
	logger       *slog.Logger
	// httpClient is used for fetching manifest files from rawBaseURL.
	httpClient *http.Client
	// rawBaseURL is the base URL for fetching raw manifest files (normally
	// https://raw.githubusercontent.com); overridable for tests.
	rawBaseURL string
	// consecutiveRawFailures counts back-to-back retry-exhausted raw fetch failures;
	// once it reaches rawFailureThreshold, subsequent manifest fetches skip raw and go
	// straight to the contents API instead of burning a full backoff cycle per file.
	consecutiveRawFailures int
}

// rawFailureThreshold is the number of consecutive raw fetch failures after which the
// ingester stops trying raw.githubusercontent.com for the rest of the run.
const rawFailureThreshold = 3

// errManifestNotFound indicates a manifest file does not exist at the requested path.
var errManifestNotFound = errors.New("winget manifest file not found")

// transientHTTPError is an HTTP response status that is worth retrying (e.g. 429 or 5xx).
type transientHTTPError struct {
	status int
	url    string
	body   string
}

func (e *transientHTTPError) Error() string {
	return fmt.Sprintf("GET %s: unexpected status %d: %s", e.url, e.status, e.body)
}

// fetchRetryInterval is the initial wait between fetch retries; a variable so tests can
// shrink it.
var fetchRetryInterval = 30 * time.Second

// fetchRetryOpts returns retry options for GitHub fetches. The winget-pkgs repo is one of
// the busiest on GitHub and its API responses intermittently fail with "429 gitmon refuses
// to schedule us" when GitHub's file servers are congested, so retry transient failures
// with exponential backoff (waits of 1x, 2x, and 4x the initial interval) instead of
// failing the whole ingestion run on the first 429.
func fetchRetryOpts() []retry.Option {
	return []retry.Option{
		retry.WithInterval(fetchRetryInterval),
		retry.WithBackoffMultiplier(2),
		retry.WithMaxAttempts(4),
		retry.WithErrorFilter(func(err error) retry.ErrorOutcome {
			// context cancellation is not transient: bail out instead of sleeping
			// through backoff waits (http.Client wraps ctx errors in *url.Error)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return retry.ErrorOutcomeDoNotRetry
			}
			if _, ok := errors.AsType[*transientHTTPError](err); ok {
				return retry.ErrorOutcomeNormalRetry
			}
			if _, ok := errors.AsType[*github.RateLimitError](err); ok {
				return retry.ErrorOutcomeNormalRetry
			}
			if _, ok := errors.AsType[*github.AbuseRateLimitError](err); ok {
				return retry.ErrorOutcomeNormalRetry
			}
			if ghErr, ok := errors.AsType[*github.ErrorResponse](err); ok && ghErr.Response != nil &&
				(ghErr.Response.StatusCode == http.StatusTooManyRequests || ghErr.Response.StatusCode >= 500) {
				return retry.ErrorOutcomeNormalRetry
			}
			// network-level errors (no HTTP response received) are also transient
			if _, ok := errors.AsType[*url.Error](err); ok {
				return retry.ErrorOutcomeNormalRetry
			}
			return retry.ErrorOutcomeDoNotRetry
		}),
	}
}

// getRepoDirContents lists a directory of the winget-pkgs repo via the GitHub contents
// API, retrying transient failures.
func (i *wingetIngester) getRepoDirContents(ctx context.Context, dirPath string) ([]*github.RepositoryContent, error) {
	var contents []*github.RepositoryContent
	err := retry.Do(func() error {
		_, repoContents, _, err := i.githubClient.Repositories.GetContents(ctx,
			"microsoft",
			"winget-pkgs",
			dirPath,
			i.ghClientOpts,
		)
		if err != nil {
			return err
		}
		contents = repoContents
		return nil
	}, fetchRetryOpts()...)
	return contents, err
}

// getRawManifestFile fetches a manifest file from raw.githubusercontent.com instead of the
// GitHub contents API: raw is CDN-backed and not subject to the gitmon scheduling limits
// that throttle API access to hot repos like microsoft/winget-pkgs. Returns
// errManifestNotFound if the file does not exist.
func (i *wingetIngester) getRawManifestFile(ctx context.Context, filePath string) ([]byte, error) {
	fileURL := fmt.Sprintf("%s/microsoft/winget-pkgs/master/%s", i.rawBaseURL, filePath)
	var body []byte
	err := retry.Do(func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
		if err != nil {
			return err
		}
		resp, err := i.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			body, err = io.ReadAll(resp.Body)
			return err
		case http.StatusNotFound:
			return errManifestNotFound
		default:
			// keep a snippet of the body so the reason (e.g. gitmon's message)
			// survives into logs
			snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			return &transientHTTPError{status: resp.StatusCode, url: fileURL, body: strings.TrimSpace(string(snippet))}
		}
	}, fetchRetryOpts()...)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// getAPIManifestFile fetches a manifest file via the GitHub contents API, retrying
// transient failures. Returns errManifestNotFound if the file does not exist.
func (i *wingetIngester) getAPIManifestFile(ctx context.Context, filePath string) ([]byte, error) {
	var body []byte
	err := retry.Do(func() error {
		fileContents, _, _, err := i.githubClient.Repositories.GetContents(ctx,
			"microsoft",
			"winget-pkgs",
			filePath,
			i.ghClientOpts,
		)
		if err != nil {
			if ghErr, ok := errors.AsType[*github.ErrorResponse](err); ok &&
				ghErr.Response != nil && ghErr.Response.StatusCode == http.StatusNotFound {
				return errManifestNotFound
			}
			return err
		}
		contents, err := fileContents.GetContent()
		if err != nil {
			return err
		}
		body = []byte(contents)
		return nil
	}, fetchRetryOpts()...)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// getManifestFile fetches a manifest file, preferring raw.githubusercontent.com and
// falling back to the contents API when raw is being throttled. The two endpoints are
// rate-limited independently (raw per IP, the API per token + gitmon's per-repo
// scheduling), so one being throttled doesn't imply the other is.
//
// A raw 404 is also confirmed via the contents API: raw is CDN-backed and can serve a
// stale or spurious 404, and callers treat a missing installer manifest as "try the next
// (older) version directory" — trusting a bad 404 would silently walk the ingester
// backwards to an older version.
func (i *wingetIngester) getManifestFile(ctx context.Context, filePath string) ([]byte, error) {
	if i.consecutiveRawFailures < rawFailureThreshold {
		contents, err := i.getRawManifestFile(ctx, filePath)
		switch {
		case err == nil:
			i.consecutiveRawFailures = 0
			return contents, nil
		case errors.Is(err, errManifestNotFound):
			// raw responded (not throttled), but only the contents API is
			// authoritative for non-existence
			i.consecutiveRawFailures = 0
		default:
			i.consecutiveRawFailures++
			i.logger.WarnContext(ctx, "raw manifest fetch failed, falling back to contents API",
				"path", filePath, "consecutive_failures", i.consecutiveRawFailures, "err", err)
		}
	}
	return i.getAPIManifestFile(ctx, filePath)
}

// wingetVersionManifestDirs keeps only subdirectory entries whose names look like winget
// package version folders (semver-style). The upstream repo may add other top-level
// folders (e.g. "Portable") that sort after numeric versions but are not manifest roots.
// It also skips single-segment "year" folders such as "2010"/"2013"/"2019" that some
// packages (e.g. Microsoft.Office) keep alongside their current multi-segment versions;
// those legacy folders would otherwise outrank a real version like "16.0.19929.20062"
// because a bare 2010 is numerically greater than 16.
func wingetVersionManifestDirs(contents []*github.RepositoryContent) []*github.RepositoryContent {
	var out []*github.RepositoryContent
	for _, c := range contents {
		if c.GetType() != "dir" {
			continue
		}
		name := c.GetName()
		if len(name) == 0 {
			continue
		}
		if name[0] < '0' || name[0] > '9' {
			continue
		}
		if !strings.Contains(name, ".") {
			continue
		}
		out = append(out, c)
	}
	return out
}

func (i *wingetIngester) ingestOne(ctx context.Context, input inputApp) (*maintained_apps.FMAManifestApp, error) {
	// this is the path within the winget GitHub repo where the manifests are located
	dirPath := path.Join(
		"manifests",
		strings.ToLower(input.PackageIdentifier[:1]),
		strings.ReplaceAll(input.PackageIdentifier, ".", "/"),
	)

	repoContents, err := i.getRepoDirContents(ctx, dirPath)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get data from winget repo")
	}

	versionDirs := wingetVersionManifestDirs(repoContents)
	if len(versionDirs) == 0 {
		return nil, ctxerr.NewWithData(ctx, "no version manifest directories found under package path", map[string]any{
			"path": dirPath,
		})
	}

	// sort the list of directories in descending order
	slices.SortFunc(versionDirs, func(a, b *github.RepositoryContent) int { return feednvd.SmartVerCmp(b.GetName(), a.GetName()) })

	// Try version directories in descending order. Some packages have nested
	// grouping directories (e.g. "2020/20.001.30002") that look like version
	// dirs but don't contain manifest files at the expected depth. Skip those
	// and fall through to the next candidate.
	var m installerManifest
	var l localeManifest
	var versionFound bool
	for _, versionDir := range versionDirs {
		vName := versionDir.GetName()
		if vName == "" {
			continue
		}

		installerManifestPath := path.Join(
			dirPath,
			vName,
			fmt.Sprintf("%s.installer.yaml", input.PackageIdentifier),
		)

		installerContents, err := i.getManifestFile(ctx, installerManifestPath)
		if err != nil {
			// Only a missing manifest means "wrong directory depth, try the next
			// candidate". Transient fetch errors must fail the run instead: silently
			// skipping to an older version dir would ingest a downgrade.
			if errors.Is(err, errManifestNotFound) {
				i.logger.DebugContext(ctx, "installer manifest not found, trying next version", "version", vName)
				continue
			}
			return nil, ctxerr.Wrap(ctx, err, "getting winget installer manifest file contents")
		}

		if err := yaml.Unmarshal(installerContents, &m); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "unmarshaling winget manifest")
		}

		localeManifestPath := path.Join(dirPath, vName, fmt.Sprintf("%s.locale.en-US.yaml", input.PackageIdentifier))
		localeContents, err := i.getManifestFile(ctx, localeManifestPath)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting winget manifest locale file contents")
		}

		if err := yaml.Unmarshal(localeContents, &l); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "unmarshaling winget locale manifest")
		}

		versionFound = true
		break
	}

	if !versionFound {
		return nil, ctxerr.NewWithData(ctx, "no valid version manifest found for app", map[string]any{
			"path": dirPath,
		})
	}

	var out maintained_apps.FMAManifestApp
	var selectedInstaller *installer
	var installScript, uninstallScript string
	productCode := m.ProductCode

	// if we have a provided install script, use that
	if input.InstallScriptPath != "" {
		scriptBytes, err := os.ReadFile(input.InstallScriptPath)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "reading provided install script file")
		}

		installScript = string(scriptBytes)
	}

	if input.UninstallScriptPath != "" {
		scriptBytes, err := os.ReadFile(input.UninstallScriptPath)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "reading provided uninstall script file")
		}

		uninstallScript = string(scriptBytes)
	}

	for _, installer := range m.Installers {
		i.logger.DebugContext(ctx, "checking installer", "arch", installer.Architecture, "type", installer.InstallerType, "locale", installer.InstallerLocale, "scope", installer.Scope)
		installerType := m.InstallerType
		if installerType == "" || isVendorType(installerType) {
			installerType = installer.InstallerType
		}

		if installerType == "" || isVendorType(installerType) {
			// try to get it from the URL
			installerType = strings.Trim(filepath.Ext(installer.InstallerURL), ".")
		}

		// Normalize wix (WiX Toolset) to msi since wix installers are MSI files
		if installerType == installerTypeWix {
			installerType = installerTypeMSI
		}

		// Normalize burn (WiX Burn bootstrapper) to exe since burn produces EXE bundles
		if installerType == installerTypeBurn {
			installerType = installerTypeExe
		}

		scope := m.Scope
		if scope == "" {
			scope = installer.Scope
			if scope == "" {
				switch installerType {
				case installerTypeMSI:
					scope = machineScope
				case installerTypeMSIX, "zip":
					// AppX/MSIX packages and zip files containing AppX are typically user-scoped
					scope = userScope
				}
			}
		}

		if !isFileType(installerType) && scope == machineScope {
			// assume we're an MSI
			installerType = installerTypeMSI
		}

		if input.InstallerLocale == "" {
			// We only care about the locale if one is specified
			installer.InstallerLocale = ""
		}

		// Check if this installer matches our criteria
		matches := installer.Architecture == input.InstallerArch &&
			scope == input.InstallerScope &&
			installer.InstallerLocale == input.InstallerLocale &&
			installerType == input.InstallerType

		if matches {
			// Prefer installers where the URL extension matches the desired installer type
			// This ensures we select the actual MSI installer over burn (EXE) installers
			urlExt := strings.Trim(filepath.Ext(installer.InstallerURL), ".")
			if urlExt == input.InstallerType {
				// Perfect match - URL extension matches desired type
				selectedInstaller = &installer
				break
			}
			// Keep as fallback candidate if we haven't found a perfect match yet
			if selectedInstaller == nil {
				selectedInstaller = &installer
			}
		}

	}

	if selectedInstaller == nil {
		return nil, ctxerr.New(ctx, "failed to find installer for app")
	}

	if input.InstallerType == installerTypeMSI && input.InstallerScope == machineScope {
		if installScript == "" {
			installScript = file.GetInstallScript(installerTypeMSI)
		}
	}

	var upgradeCode string
	if (input.InstallerType == installerTypeMSI || input.UninstallType == installerTypeMSI) && input.InstallerScope == machineScope {
		for _, fe := range m.AppsAndFeaturesEntries {
			if fe.UpgradeCode != "" {
				upgradeCode = fe.UpgradeCode
				break
			}
		}
		if upgradeCode == "" {
			for _, fe := range selectedInstaller.AppsAndFeaturesEntries {
				if fe.UpgradeCode != "" {
					upgradeCode = fe.UpgradeCode
					break
				}
			}
		}
		if uninstallScript == "" && upgradeCode != "" {
			var err error
			uninstallScript, err = buildUpgradeCodeBasedUninstallScript(upgradeCode)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "building upgrade code based uninstall script")
			}
		}

		if uninstallScript == "" {
			uninstallScript = file.GetUninstallScript(installerTypeMSI)
		}
	}

	if installScript == "" {
		return nil, ctxerr.New(ctx, "no install script found for app, aborting")
	}

	if uninstallScript == "" {
		return nil, ctxerr.New(ctx, "no uninstall script found for app, aborting")
	}

	if productCode == "" {
		productCode = selectedInstaller.ProductCode
	}
	if input.InstallerType == installerTypeMSIX && productCode == "" {
		productCode = selectedInstaller.PackageFamilyName
	}
	if input.InstallerType == installerTypeMSI && productCode != "" {
		productCode = strings.Split(productCode, ".")[0]
	}

	if upgradeCode != "" {
		out.UpgradeCode = upgradeCode
	}

	out.Name = input.Name
	out.Slug = input.Slug
	out.InstallerURL = selectedInstaller.InstallerURL
	out.UniqueIdentifier = input.UniqueIdentifier
	out.DefaultCategories = input.DefaultCategories
	out.SHA256 = "no_check"
	if !input.IgnoreHash {
		out.SHA256 = strings.ToLower(selectedInstaller.InstallerSha256) // maintain consistency with darwin outputs SHAs
	}
	out.Version = m.PackageVersion
	publisher := l.Publisher
	if input.ProgramPublisher != "" {
		publisher = input.ProgramPublisher
	}
	name := l.PackageName
	if input.UniqueIdentifier != "" {
		name = input.UniqueIdentifier
	}

	out.Queries = setUpExistsQuery(input.FuzzyMatchName, name, publisher)
	if input.ExistsQuery != "" {
		out.Queries.Exists = input.ExistsQuery
	}
	out.InstallScript = installScript
	processedUninstallScript, err := preProcessUninstallScript(uninstallScript, productCode)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "pre-processing uninstall script")
	}
	out.UninstallScript = processedUninstallScript
	out.InstallScriptRef = maintained_apps.GetScriptRef(out.InstallScript)
	out.UninstallScriptRef = maintained_apps.GetScriptRef(out.UninstallScript)
	out.Frozen = input.Frozen

	external_refs.EnrichManifest(&out)

	// The patch policy normally compares osquery's reported version against the
	// winget PackageVersion. Some installers report a registry DisplayVersion in a
	// different format (e.g. python.org reports "3.14.5150.0" for "3.14.5"), which
	// breaks version_compare ordering. When opted in, compare against the
	// DisplayVersion so the patch policy flags outdated installs correctly.
	patchVersion := out.Version
	if input.UseDisplayVersionForPatch {
		displayVersion := firstDisplayVersion(selectedInstaller.AppsAndFeaturesEntries)
		if displayVersion == "" {
			displayVersion = firstDisplayVersion(m.AppsAndFeaturesEntries)
		}
		if displayVersion == "" {
			return nil, ctxerr.New(ctx, "use_display_version_for_patch is set but no DisplayVersion found in winget manifest")
		}
		patchVersion = displayVersion
	}

	// create patch policy
	out.Queries.Patched, err = patch_policy.GenerateQueryForManifest(patch_policy.PolicyData{
		Platform:    "windows",
		Version:     patchVersion,
		ExistsQuery: out.Queries.Exists,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating patch policy")
	}

	return &out, nil
}

func escapeSQLParam(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// firstDisplayVersion returns the first non-empty DisplayVersion from a set of
// AppsAndFeaturesEntries, or "" if none is present.
func firstDisplayVersion(entries []appsAndFeaturesEntries) string {
	for _, fe := range entries {
		if v := strings.TrimSpace(fe.DisplayVersion); v != "" {
			return v
		}
	}
	return ""
}

func setUpExistsQuery(fuzzy fuzzyMatch, name string, publisher string) maintained_apps.FMAQueries {
	// TODO - consider UpgradeCode here?
	return maintained_apps.FMAQueries{
		Exists: fmt.Sprintf("SELECT 1 FROM programs WHERE %s AND publisher = '%s';",
			fuzzy.nameCondition(name), escapeSQLParam(publisher)),
	}
}

func buildUpgradeCodeBasedUninstallScript(upgradeCode string) (string, error) {
	if err := file.ValidatePackageIdentifiers(nil, upgradeCode); err != nil {
		return "", err
	}
	return file.UpgradeCodeRegex.ReplaceAllString(file.UninstallMsiWithUpgradeCodeScript, fmt.Sprintf("'%s'${suffix}", upgradeCode)), nil
}

func preProcessUninstallScript(uninstallScript, productCode string) (string, error) {
	if productCode == "" {
		return uninstallScript, nil
	}
	if err := file.ValidatePackageIdentifiers([]string{productCode}, ""); err != nil {
		return "", err
	}
	code := fmt.Sprintf("'%s'", productCode)
	return file.PackageIDRegex.ReplaceAllString(uninstallScript, fmt.Sprintf("%s${suffix}", code)), nil
}

// these are installer types that correspond to software vendors, not the actual installer type
// (like exe or msi).
var vendorTypes = map[string]struct{}{
	installerTypeWix:      {},
	installerTypeNullSoft: {},
	installerTypeInno:     {},
}

func isVendorType(installerType string) bool {
	_, ok := vendorTypes[installerType]
	return ok
}

var fileTypes = map[string]struct{}{
	installerTypeMSI:  {},
	installerTypeMSIX: {},
	installerTypeExe:  {},
	"zip":             {},
}

func isFileType(installerType string) bool {
	_, ok := fileTypes[installerType]
	return ok
}

// fuzzyMatch supports three JSON representations:
//   - false (or omitted): exact match on programs.name
//   - true: automatic LIKE pattern  "name LIKE '<unique_identifier> %'"
//   - "<pattern>": a custom LIKE pattern used verbatim, e.g. "Mozilla Firefox % ESR %"
type fuzzyMatch struct {
	Enabled bool   // true when the JSON value is the boolean `true`
	Custom  string // non-empty when the JSON value is a string pattern
}

func (f *fuzzyMatch) UnmarshalJSON(data []byte) error {
	// Try boolean first (handles true, false, and omitted-via-zero-value).
	var b bool
	if err := json.Unmarshal(data, &b); err == nil {
		f.Enabled = b
		f.Custom = ""
		return nil
	}
	// Try string.
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		f.Custom = s
		f.Enabled = s != ""
		return nil
	}
	return fmt.Errorf("fuzzy_match_name must be a boolean or a string, got %s", string(data))
}

func (f *fuzzyMatch) nameCondition(name string) string {
	if f.Custom != "" {
		return fmt.Sprintf("name LIKE '%s'", escapeSQLParam(f.Custom))
	}
	if f.Enabled {
		return fmt.Sprintf("name LIKE '%s %%'", escapeSQLParam(name))
	}
	return fmt.Sprintf("name = '%s'", escapeSQLParam(name))
}

type inputApp struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	// PackageIdentifier is the identifier used by winget. It's composed of a vendor part (e.g.
	// AgileBits) and an app part (e.g. 1Password), joined by a "."
	PackageIdentifier string `json:"package_identifier"`
	// The value matching programs.name for the primary app package in osquery
	UniqueIdentifier    string     `json:"unique_identifier"`
	InstallScriptPath   string     `json:"install_script_path"`
	UninstallScriptPath string     `json:"uninstall_script_path"`
	InstallerArch       string     `json:"installer_arch"`
	InstallerType       string     `json:"installer_type"`
	InstallerScope      string     `json:"installer_scope"`
	InstallerLocale     string     `json:"installer_locale"`
	ProgramPublisher    string     `json:"program_publisher"`
	UninstallType       string     `json:"uninstall_type"`
	FuzzyMatchName      fuzzyMatch `json:"fuzzy_match_name"`
	// ExistsQuery overrides the default programs-table exists query (e.g. portable zip installs).
	ExistsQuery string `json:"exists_query,omitempty"`
	// UseDisplayVersionForPatch makes the patch policy compare against the
	// installer's registry DisplayVersion (from AppsAndFeaturesEntries) instead of
	// the winget PackageVersion. Needed when the registry version format differs
	// from the marketing version in a way that breaks version_compare ordering
	// (e.g. python.org installers report "3.14.5150.0" for version "3.14.5").
	UseDisplayVersionForPatch bool `json:"use_display_version_for_patch"`
	// Whether to use "no_check" instead of the app's hash (e.g. for non-pinned download URLs)
	IgnoreHash        bool     `json:"ignore_hash"`
	DefaultCategories []string `json:"default_categories"`
	Frozen            bool     `json:"frozen"`
	PatchPolicyPath   string   `json:"patch_policy_path"`
}

type installerManifest struct {
	PackageIdentifier      string                   `yaml:"PackageIdentifier"`
	PackageVersion         string                   `yaml:"PackageVersion"`
	Installers             []installer              `yaml:"Installers"`
	InstallerType          string                   `yaml:"InstallerType"`
	AppsAndFeaturesEntries []appsAndFeaturesEntries `yaml:"AppsAndFeaturesEntries,omitempty"`
	ProductCode            string                   `yaml:"ProductCode"`
	Scope                  string                   `yaml:"Scope"`
}

type installer struct {
	Architecture string `yaml:"Architecture"`
	// InstallerType is the filetype of the installer. Either "exe" or "msi".
	InstallerType          string                   `yaml:"InstallerType"`
	Scope                  string                   `yaml:"Scope"`
	InstallerURL           string                   `yaml:"InstallerUrl"`
	InstallerSha256        string                   `yaml:"InstallerSha256"`
	InstallModes           []string                 `yaml:"InstallModes,omitempty"`
	InstallerSwitches      installerSwitches        `yaml:"InstallerSwitches,omitempty"`
	ProductCode            string                   `yaml:"ProductCode"`
	PackageFamilyName      string                   `yaml:"PackageFamilyName"`
	AppsAndFeaturesEntries []appsAndFeaturesEntries `yaml:"AppsAndFeaturesEntries,omitempty"`
	InstallerLocale        string                   `yaml:"InstallerLocale"`
}
type installerSwitches struct {
	Silent             string `yaml:"Silent"`
	SilentWithProgress string `yaml:"SilentWithProgress"`
}

type appsAndFeaturesEntries struct {
	Publisher      string `yaml:"Publisher"`
	ProductCode    string `yaml:"ProductCode"`
	UpgradeCode    string `yaml:"UpgradeCode"`
	DisplayVersion string `yaml:"DisplayVersion"`
}

type localeManifest struct {
	PackageIdentifier   string   `yaml:"PackageIdentifier"`
	PackageVersion      string   `yaml:"PackageVersion"`
	PackageLocale       string   `yaml:"PackageLocale"`
	Publisher           string   `yaml:"Publisher"`
	PublisherURL        string   `yaml:"PublisherUrl"`
	PublisherSupportURL string   `yaml:"PublisherSupportUrl"`
	PrivacyURL          string   `yaml:"PrivacyUrl"`
	Author              string   `yaml:"Author"`
	PackageName         string   `yaml:"PackageName"`
	PackageURL          string   `yaml:"PackageUrl"`
	License             string   `yaml:"License"`
	LicenseURL          string   `yaml:"LicenseUrl"`
	Copyright           string   `yaml:"Copyright"`
	CopyrightURL        string   `yaml:"CopyrightUrl"`
	ShortDescription    string   `yaml:"ShortDescription"`
	Description         string   `yaml:"Description"`
	Tags                []string `yaml:"Tags"`
	PurchaseURL         string   `yaml:"PurchaseUrl"`
	ManifestType        string   `yaml:"ManifestType"`
	ManifestVersion     string   `yaml:"ManifestVersion"`
}

const (
	machineScope          = "machine"
	userScope             = "user"
	installerTypeMSI      = "msi"
	installerTypeMSIX     = "msix"
	installerTypeExe      = "exe"
	installerTypeWix      = "wix"
	installerTypeNullSoft = "nullsoft"
	installerTypeInno     = "inno"
	installerTypeBurn     = "burn"
	arch64Bit             = "x64"
	arch32Bit             = "x86"
)
