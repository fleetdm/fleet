package homebrew

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	external_refs "github.com/fleetdm/fleet/v4/ee/maintained-apps/ingesters/homebrew/external_refs"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/pkg/patch_policy"
	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

func IngestApps(ctx context.Context, logger *slog.Logger, inputsPath, slugFilter string) ([]*maintained_apps.FMAManifestApp, error) {
	logger.InfoContext(ctx, "starting homebrew app data ingestion")
	// Read from our list of apps we should be ingesting
	files, err := os.ReadDir(inputsPath)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reading homebrew input data directory")
	}

	i := &brewIngester{
		baseURL:          baseBrewAPIURL,
		buildhubURL:      buildhubAPIURL,
		logger:           logger,
		client:           fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
		retryInterval:    2 * time.Second,
		retryMaxAttempts: 5,
	}

	var manifestApps []*maintained_apps.FMAManifestApp

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
			return nil, ctxerr.WrapWithData(ctx, err, "reading app input file", map[string]any{"fileName": f.Name()})
		}

		var input inputApp
		if err := json.Unmarshal(fileBytes, &input); err != nil {
			return nil, ctxerr.WrapWithData(ctx, err, "unmarshal app input file", map[string]any{"fileName": f.Name()})
		}

		if input.Token == "" {
			return nil, ctxerr.NewWithData(ctx, "missing token for app", map[string]any{"fileName": f.Name()})
		}

		if input.UniqueIdentifier == "" {
			return nil, ctxerr.NewWithData(ctx, "missing unique identifier for app", map[string]any{"fileName": f.Name()})
		}

		if input.Name == "" {
			return nil, ctxerr.NewWithData(ctx, "missing name for app", map[string]any{"fileName": f.Name()})
		}

		if slugFilter != "" && !strings.Contains(input.Slug, slugFilter) {
			continue
		}

		i.logger.InfoContext(ctx, "ingesting homebrew app", "name", input.Name)

		outApp, err := i.ingestOne(ctx, input)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "ingesting homebrew app")
		}

		manifestApps = append(manifestApps, outApp)

	}

	return manifestApps, nil
}

const (
	baseBrewAPIURL = "https://formulae.brew.sh/api/"
	// buildhubAPIURL is Mozilla's build metadata search API.
	buildhubAPIURL = "https://buildhub.moz.tools/api/search"
)

type brewIngester struct {
	baseURL     string
	buildhubURL string
	logger      *slog.Logger
	client      *http.Client

	// retryInterval and retryMaxAttempts control retries of transient brew API
	// failures (network errors and 5xx/429 responses). formulae.brew.sh is
	// served by GitHub Pages, which intermittently returns 503s; without
	// retries a single blip aborts the whole ingestion run. Defaults are set in
	// IngestApps; tests override them to keep runs fast.
	retryInterval    time.Duration
	retryMaxAttempts int
}

// transientErr wraps a brew API failure that is worth retrying (a network error
// or a 5xx/429 server response). Non-transient failures (404, other 4xx) are
// returned unwrapped so the retry loop gives up immediately.
type transientErr struct{ err error }

func (e *transientErr) Error() string { return e.err.Error() }
func (e *transientErr) Unwrap() error { return e.err }

func (i *brewIngester) ingestOne(ctx context.Context, input inputApp) (*maintained_apps.FMAManifestApp, error) {
	cask, err := i.fetchCask(ctx, input)
	if err != nil {
		return nil, err
	}

	out := &maintained_apps.FMAManifestApp{}

	// validate required fields
	if len(cask.Name) == 0 || cask.Name[0] == "" {
		return nil, ctxerr.Errorf(ctx, "missing name for cask %s", input.Token)
	}
	if cask.Token == "" {
		return nil, ctxerr.Errorf(ctx, "missing token for cask %s", input.Token)
	}
	if cask.Version == "" {
		return nil, ctxerr.Errorf(ctx, "missing version for cask %s", input.Token)
	}
	if cask.URL == "" {
		return nil, ctxerr.Errorf(ctx, "missing URL for cask %s", input.Token)
	}
	_, err = url.Parse(cask.URL)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "parse URL for cask %s", input.Token)
	}

	out.Name = input.Name
	out.Version = strings.Split(cask.Version, ",")[0]
	out.InstallerURL = cask.URL
	out.UniqueIdentifier = input.UniqueIdentifier
	out.SHA256 = cask.SHA256
	out.Queries = maintained_apps.FMAQueries{Exists: fmt.Sprintf("SELECT 1 FROM apps WHERE bundle_identifier = '%s';", out.UniqueIdentifier)}
	if input.Token == "swiftdialog" {
		// Orbit installs swiftDialog v2.5.6 for setup experience, MDM migration, or enrollment
		// profile renewal; don't treat orbit's copy as the installed app for install or patch status.
		// The patch policy generated below inherits this exclusion.
		out.Queries.Exists = fmt.Sprintf(
			"SELECT 1 FROM apps WHERE bundle_identifier = '%s' AND path != '/opt/orbit/bin/swiftDialog/macos/stable/Dialog.app';",
			out.UniqueIdentifier,
		)
	}
	out.Slug = input.Slug
	out.DefaultCategories = input.DefaultCategories

	var installScript, uninstallScript string

	switch input.InstallScriptPath {
	case "":
		installScript, err = installScriptForApp(input, &cask)
		if err != nil {
			return nil, ctxerr.WrapWithData(ctx, err, "generating install script for maintained app", map[string]any{"unique_identifier": input.UniqueIdentifier})
		}
	default:
		scriptBytes, err := os.ReadFile(input.InstallScriptPath)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "reading provided install script file")
		}

		installScript = string(scriptBytes)
	}

	switch input.UninstallScriptPath {
	case "":
		if len(input.PreUninstallScripts) != 0 {
			cask.PreUninstallScripts = input.PreUninstallScripts
		}

		if len(input.PostUninstallScripts) != 0 {
			cask.PostUninstallScripts = input.PostUninstallScripts
		}

		uninstallScript = uninstallScriptForApp(&cask)
	default:
		if len(input.PreUninstallScripts) != 0 {
			return nil, ctxerr.New(ctx, "cannot provide pre-uninstall scripts if uninstall script is provided")
		}

		if len(input.PostUninstallScripts) != 0 {
			return nil, ctxerr.New(ctx, "cannot provide post-uninstall scripts if uninstall script is provided")
		}

		scriptBytes, err := os.ReadFile(input.UninstallScriptPath)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "reading provided uninstall script file")
		}

		uninstallScript = string(scriptBytes)
	}

	out.InstallScript = installScript
	out.UninstallScript = uninstallScript

	out.UninstallScriptRef = maintained_apps.GetScriptRef(out.UninstallScript)
	out.InstallScriptRef = maintained_apps.GetScriptRef(out.InstallScript)
	out.Frozen = input.Frozen

	external_refs.EnrichManifest(out)

	// create patch policy
	out.Queries.Patched, err = patch_policy.GenerateQueryForManifest(patch_policy.PolicyData{
		Platform:    "darwin",
		Version:     out.Version,
		ExistsQuery: out.Queries.Exists,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating patch policy")
	}
	if input.Token == "docker-desktop" {
		// Docker's updater can leave Docker.app.back; do not treat it as the installed app for patch status.
		out.Queries.Patched = fmt.Sprintf(
			"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '%s' AND path NOT LIKE '%%.back' AND version_compare(bundle_short_version, '%s') < 0);",
			out.UniqueIdentifier, out.Version,
		)
	}
	if input.Token == "sonos" {
		// Sonos versions its cask by build number (matching CFBundleVersion, e.g.
		// "90.0.77070" after SonosVersionTransformer), while bundle_short_version is
		// the unrelated marketing version (e.g. "17.2.3"). Compare bundle_version so
		// patch status reflects the actual installed build.
		out.Queries.Patched = fmt.Sprintf(
			"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '%s' AND version_compare(bundle_version, '%s') < 0);",
			out.UniqueIdentifier, out.Version,
		)
	}
	if input.Token == "firefox@developer-edition" {
		// The bundle reports only the base version ("153.0") for cask version
		// "153.0b13", so compare CFBundleVersion (encodes the build date, resolved
		// via buildhub) to distinguish betas; fall back to a cycle-granular
		// base-version comparison if buildhub is unavailable.
		column := "bundle_version"
		patchVersion, err := i.firefoxDevEditionMacBundleVersion(ctx, out.Version)
		if err != nil {
			i.logger.WarnContext(ctx, "resolving Firefox Developer Edition bundle version failed; patch policy falls back to base-version comparison", "err", err.Error())
			column, patchVersion = "bundle_short_version", firefoxBetaBaseVersion(out.Version)
		}
		out.Queries.Patched = fmt.Sprintf(
			"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '%s' AND version_compare(%s, '%s') < 0);",
			out.UniqueIdentifier, column, patchVersion,
		)
	}
	if input.Token == "firefox@nightly" {
		// Nightly's CFBundleShortVersionString ("154.0a1") is constant all cycle;
		// derive CFBundleVersion from the cask version's build timestamp for
		// day-level patch status.
		bundleVersion, err := firefoxNightlyMacBundleVersion(cask.Version)
		if err != nil {
			i.logger.WarnContext(ctx, "deriving Firefox Nightly bundle version failed; patch policy falls back to short-version comparison", "err", err.Error())
		} else {
			out.Queries.Patched = fmt.Sprintf(
				"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '%s' AND version_compare(bundle_version, '%s') < 0);",
				out.UniqueIdentifier, bundleVersion,
			)
		}
	}

	out.Queries.Open, err = patch_policy.GenerateOpenQuery("darwin", out.Name)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating app open query")
	}

	return out, nil
}

var firefoxBetaVersionPattern = regexp.MustCompile(`^(\d+(?:\.\d+)*)b\d+$`)

// firefoxBetaBaseVersion strips the beta suffix from a Firefox pre-release
// version ("153.0b13" -> "153.0"); non-matching versions pass through unchanged.
func firefoxBetaBaseVersion(version string) string {
	if m := firefoxBetaVersionPattern.FindStringSubmatch(version); m != nil {
		return m[1]
	}
	return version
}

// firefoxMacBundleVersion computes a Firefox mac build's CFBundleVersion:
// "<major><yy>.<month>.<day>", unpadded ("153.0b13" + "20260715" -> "15326.7.15").
func firefoxMacBundleVersion(version, buildDate string) (string, error) {
	major, _, _ := strings.Cut(version, ".")
	if major == "" || strings.Trim(major, "0123456789") != "" {
		return "", fmt.Errorf("cannot parse major version from %q", version)
	}
	if len(buildDate) < 8 {
		return "", fmt.Errorf("invalid build date %q", buildDate)
	}
	date, err := time.Parse("20060102", buildDate[:8])
	if err != nil {
		return "", fmt.Errorf("invalid build date %q", buildDate)
	}
	yy := buildDate[2:4]
	return fmt.Sprintf("%s%s.%d.%d", major, yy, int(date.Month()), date.Day()), nil
}

// firefoxNightlyCaskVersionPattern extracts the build timestamp from a Firefox
// Nightly cask version ("154.0a1,2026-07-17-09-27-13").
var firefoxNightlyCaskVersionPattern = regexp.MustCompile(`^[^,]+,(\d{4})-(\d{2})-(\d{2})(?:-|$)`)

// firefoxNightlyMacBundleVersion derives CFBundleVersion from a Nightly cask
// version ("154.0a1,2026-07-17-09-27-13" -> "15426.7.17").
func firefoxNightlyMacBundleVersion(caskVersion string) (string, error) {
	m := firefoxNightlyCaskVersionPattern.FindStringSubmatch(caskVersion)
	if m == nil {
		return "", fmt.Errorf("cask version %q has no build timestamp", caskVersion)
	}
	return firefoxMacBundleVersion(caskVersion, m[1]+m[2]+m[3])
}

// firefoxDevEditionMacBundleVersion resolves a Developer Edition mac build's
// CFBundleVersion ("153.0b13" -> "15326.7.15") by looking up its build id in
// buildhub, where DevEd is indexed as product "firefox", channel "aurora".
func (i *brewIngester) firefoxDevEditionMacBundleVersion(ctx context.Context, version string) (string, error) {
	type term map[string]map[string]string
	reqBody, err := json.Marshal(map[string]any{
		"size": 1,
		"query": map[string]any{
			"bool": map[string]any{
				"must": []term{
					{"term": {"source.product": "firefox"}},
					{"term": {"target.channel": "aurora"}},
					{"term": {"target.platform": "mac"}},
					{"term": {"target.version": version}},
				},
			},
		},
		"sort": []term{{"build.id": {"order": "desc"}}},
	})
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "marshal buildhub query")
	}

	interval := i.retryInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	maxAttempts := i.retryMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	var body []byte
	attempt := 0
	err = retry.Do(func() error {
		attempt++

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, i.buildhubURL, bytes.NewReader(reqBody))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "create buildhub http request")
		}
		req.Header.Set("Content-Type", "application/json")

		res, err := i.client.Do(req)
		if err != nil {
			// Caller cancellation/deadline is not transient; stop retrying.
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			i.logger.WarnContext(ctx, "buildhub request failed, retrying", "attempt", attempt, "err", err.Error())
			return &transientErr{ctxerr.Wrap(ctx, err, "execute buildhub http request")}
		}
		defer res.Body.Close()

		body, err = io.ReadAll(res.Body)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			i.logger.WarnContext(ctx, "reading buildhub response failed, retrying", "attempt", attempt, "err", err.Error())
			return &transientErr{ctxerr.Wrap(ctx, err, "read buildhub response body")}
		}

		switch res.StatusCode {
		case http.StatusOK:
			return nil
		case http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			i.logger.WarnContext(ctx, "buildhub returned transient error, retrying", "attempt", attempt, "status", res.StatusCode)
			return &transientErr{ctxerr.Errorf(ctx, "buildhub returned status %d: %s", res.StatusCode, truncateBody(body))}
		default:
			return ctxerr.Errorf(ctx, "buildhub returned status %d: %s", res.StatusCode, truncateBody(body))
		}
	},
		retry.WithInterval(interval),
		retry.WithBackoffMultiplier(2),
		retry.WithMaxAttempts(maxAttempts),
		retry.WithErrorFilter(func(err error) retry.ErrorOutcome {
			if _, ok := errors.AsType[*transientErr](err); ok {
				return retry.ErrorOutcomeNormalRetry
			}
			return retry.ErrorOutcomeDoNotRetry
		}),
	)
	if err != nil {
		return "", err
	}

	var resp struct {
		Hits struct {
			Hits []struct {
				Source struct {
					Build struct {
						ID string `json:"id"`
					} `json:"build"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", ctxerr.Wrap(ctx, err, "unmarshal buildhub response")
	}
	if len(resp.Hits.Hits) == 0 {
		return "", ctxerr.Errorf(ctx, "no buildhub build found for version %s", version)
	}

	return firefoxMacBundleVersion(version, resp.Hits.Hits[0].Source.Build.ID)
}

// fetchCask resolves the brew cask JSON for the given input app from
// either a local file (cask_path) or the default brew API.
func (i *brewIngester) fetchCask(ctx context.Context, input inputApp) (brewCask, error) {
	var cask brewCask

	if input.CaskPath != "" {
		body, err := os.ReadFile(input.CaskPath)
		if err != nil {
			return cask, ctxerr.WrapWithData(ctx, err, "reading local cask JSON file", map[string]any{"cask_path": input.CaskPath})
		}
		if err := json.Unmarshal(body, &cask); err != nil {
			return cask, ctxerr.Wrapf(ctx, err, "unmarshal local cask JSON for %s", input.Token)
		}
		// Cross-check the cask file matches the configured input. This catches
		// subtle misconfiguration like pointing cask_path at the wrong JSON file.
		if cask.Token != input.Token {
			return cask, ctxerr.Errorf(ctx, "local cask JSON token %q does not match input token %q (cask_path: %s)", cask.Token, input.Token, input.CaskPath)
		}
		if len(cask.Name) == 0 {
			return cask, ctxerr.Errorf(ctx, "local cask JSON for %s has empty name (cask_path: %s)", input.Token, input.CaskPath)
		}
		return cask, nil
	}

	apiURL := fmt.Sprintf("%scask/%s.json", i.baseURL, input.Token)

	interval := i.retryInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	maxAttempts := i.retryMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	var body []byte
	attempt := 0
	err := retry.Do(func() error {
		attempt++

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "create http request")
		}

		res, err := i.client.Do(req)
		if err != nil {
			// Caller cancellation/deadline is not transient; stop retrying so
			// we don't sleep through the backoff after the run was canceled.
			// Checking ctx.Err() (rather than the returned error) avoids
			// misclassifying the client's own request timeout, which we do
			// want to retry.
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			// Network-level failures are transient; retry.
			i.logger.WarnContext(ctx, "brew API request failed, retrying", "token", input.Token, "attempt", attempt, "err", err.Error())
			return &transientErr{ctxerr.Wrap(ctx, err, "execute http request")}
		}
		defer res.Body.Close()

		body, err = io.ReadAll(res.Body)
		if err != nil {
			// Caller cancellation/deadline is not transient; stop retrying.
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			// A truncated/interrupted read is transient; retry.
			i.logger.WarnContext(ctx, "reading brew API response failed, retrying", "token", input.Token, "attempt", attempt, "err", err.Error())
			return &transientErr{ctxerr.Wrap(ctx, err, "read http response body")}
		}

		switch res.StatusCode {
		case http.StatusOK:
			return nil
		case http.StatusNotFound:
			return ctxerr.New(ctx, "app not found in brew API")
		case http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			// formulae.brew.sh (GitHub Pages) intermittently returns these;
			// retry before giving up.
			i.logger.WarnContext(ctx, "brew API returned transient error, retrying", "token", input.Token, "attempt", attempt, "status", res.StatusCode)
			return &transientErr{ctxerr.Errorf(ctx, "brew API returned status %d: %s", res.StatusCode, truncateBody(body))}
		default:
			return ctxerr.Errorf(ctx, "brew API returned status %d: %s", res.StatusCode, truncateBody(body))
		}
	},
		retry.WithInterval(interval),
		retry.WithBackoffMultiplier(2),
		retry.WithMaxAttempts(maxAttempts),
		retry.WithErrorFilter(func(err error) retry.ErrorOutcome {
			if _, ok := errors.AsType[*transientErr](err); ok {
				return retry.ErrorOutcomeNormalRetry
			}
			return retry.ErrorOutcomeDoNotRetry
		}),
	)
	if err != nil {
		return cask, err
	}

	if err := json.Unmarshal(body, &cask); err != nil {
		return cask, ctxerr.Wrapf(ctx, err, "unmarshal brew cask for %s", input.Token)
	}
	return cask, nil
}

// truncateBody limits an error-response body to a sane length for log/error output.
func truncateBody(body []byte) string {
	if len(body) > 512 {
		body = body[:512]
	}
	return string(body)
}

type inputApp struct {
	// Name is the user-friendly name of the app.
	Name string `json:"name"`
	// Token is the identifier in the source data for the app (e.g. homebrew token).
	Token string `json:"token"`
	// UniqueIdentifier is the app's unique identifier on its platform (e.g. bundle ID on macOS).
	UniqueIdentifier string `json:"unique_identifier"`
	// InstallerFormat is the installer format used for installing this app.
	InstallerFormat string `json:"installer_format"`
	// Slug is an identifier that combines the app's token and the target OS.
	Slug                 string   `json:"slug"`
	PreUninstallScripts  []string `json:"pre_uninstall_scripts"`
	PostUninstallScripts []string `json:"post_uninstall_scripts"`
	DefaultCategories    []string `json:"default_categories"`
	Frozen               bool     `json:"frozen"`
	InstallScriptPath    string   `json:"install_script_path"`
	UninstallScriptPath  string   `json:"uninstall_script_path"`
	PatchPolicyPath      string   `json:"patch_policy_path"`
	// CaskPath optionally points at a local file (relative to the repo
	// root) containing the cask JSON in the same schema as
	// https://formulae.brew.sh/api/cask/<token>.json. Used to commit cask
	// metadata for third-party taps directly into this repo (see
	// inputs/homebrew/custom-tap/). When empty, the ingester fetches from
	// formulae.brew.sh.
	CaskPath string `json:"cask_path"`
}

type brewCask struct {
	Token                string          `json:"token"`
	FullToken            string          `json:"full_token"`
	Tap                  string          `json:"tap"`
	Name                 []string        `json:"name"`
	Desc                 string          `json:"desc"`
	URL                  string          `json:"url"`
	Version              string          `json:"version"`
	SHA256               string          `json:"sha256"`
	Artifacts            []*brewArtifact `json:"artifacts"`
	PreUninstallScripts  []string        `json:"-"`
	PostUninstallScripts []string        `json:"-"`
}

// brew artifacts are objects that have one and only one of their fields set.
type brewArtifact struct {
	// App is an array that can contain strings or objects with a target field.
	// See grammarly-desktop cask.
	App []optjson.StringOr[*brewAppTarget] `json:"app"`
	// Pkg is a bit like Binary, it is an array with a string and an object as
	// first two elements. The object has a choices field with an array of
	// objects. See Microsoft Edge.
	Pkg []optjson.StringOr[*brewPkgChoices] `json:"pkg"`
	// Zap and Uninstall have the same format, they support the same stanzas.
	// It's just that in homebrew, Zaps are not processed by default (only when
	// --zap is provided on uninstall). For our uninstall scripts, we want to
	// process the zaps.
	Uninstall []*brewUninstall `json:"uninstall"`
	Zap       []*brewUninstall `json:"zap"`
	// Binary is an array with a string and an object as first two elements. See
	// the "docker" and "firefox" casks.
	Binary []optjson.StringOr[*brewBinaryTarget] `json:"binary"`
}

// The choiceChanges file is a property list containing an array of dictionaries. Each dictionary has the following three keys:
//
// Key                     Description
// choiceIdentifier        Identifier for the choice to be modified (string)
// choiceAttribute         One of the attribute names described below (string)
// attributeSetting        A setting that depends on the choiceAttribute, described below (number or string)
//
// The choiceAttribute and attributeSetting values are as follows:
//
// choiceAttribute    attributeSetting   Description
// selected           (number) 1         to select the choice, 0 to deselect it
// enabled            (number) 1         to enable the choice, 0 to disable it
// visible            (number) 1         to show the choice, 0 to hide it
// customLocation     (string)           path at which to install the choice (see below)
type brewPkgConfig struct {
	ChoiceIdentifier string `json:"choiceIdentifier" plist:"choiceIdentifier"`
	ChoiceAttribute  string `json:"choiceAttribute" plist:"choiceAttribute"`
	AttributeSetting int    `json:"attributeSetting" plist:"attributeSetting"`
}

type brewPkgChoices struct {
	Choices []brewPkgConfig `json:"choices"`
}

type brewBinaryTarget struct {
	Target string `json:"target"`
}

type brewAppTarget struct {
	Target string `json:"target"`
}

// unlike brewArtifact, a single brewUninstall can have many fields set.
// All fields can have one or multiple strings (string or []string).
type brewUninstall struct {
	LaunchCtl optjson.StringOr[[]string] `json:"launchctl"`
	Quit      optjson.StringOr[[]string] `json:"quit"`
	PkgUtil   optjson.StringOr[[]string] `json:"pkgutil"`
	// brew docs says string or hash, but our only case has a single string.
	Script optjson.StringOr[map[string]any] `json:"script"`
	// format: [0]=signal, [1]=process name (although the brew documentation says
	// it's an array of arrays, it's not like that in our single case that uses
	// it).
	Signal    optjson.StringOr[[]string] `json:"signal"`
	Delete    optjson.StringOr[[]string] `json:"delete"`
	RmDir     optjson.StringOr[[]string] `json:"rmdir"`
	Trash     optjson.StringOr[[]string] `json:"trash"`
	LoginItem optjson.StringOr[[]string] `json:"login_item"`
	Kext      optjson.StringOr[[]string] `json:"kext"`
}
