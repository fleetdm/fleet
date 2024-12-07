package maintainedapps

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"golang.org/x/sync/errgroup"
)

//go:embed apps.json
var appsJSON []byte

type maintainedApp struct {
	Identifier           string   `json:"identifier"`
	BundleIdentifier     string   `json:"bundle_identifier"`
	InstallerFormat      string   `json:"installer_format"`
	PreUninstallScripts  []string `json:"pre_uninstall_scripts"`
	PostUninstallScripts []string `json:"post_uninstall_scripts"`
}

const baseBrewAPIURL = "https://formulae.brew.sh/api/"

// Refresh fetches the latest information about maintained apps from the brew
// API and updates the Fleet database with the new information.
func Refresh(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger) error {
	var apps []maintainedApp
	if err := json.Unmarshal(appsJSON, &apps); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal embedded apps.json")
	}

	// allow mocking of the brew API for tests
	baseURL := baseBrewAPIURL
	if v := os.Getenv("FLEET_DEV_BREW_API_URL"); v != "" {
		baseURL = v
	}

	i := ingester{
		baseURL: baseURL,
		ds:      ds,
		logger:  logger,
	}
	return i.ingest(ctx, apps)
}

// ExtensionForBundleIdentifier returns an extension for the given FMA
// identifier. If one can't be found it returns an empty string.
//
// This function is used because we can't always extract the extension based on
// the installer URL.
func ExtensionForBundleIdentifier(identifier string) (string, error) {
	var apps []maintainedApp
	if err := json.Unmarshal(appsJSON, &apps); err != nil {
		return "", fmt.Errorf("unmarshal embedded apps.json: %w", err)
	}

	for _, app := range apps {
		if app.BundleIdentifier == identifier {
			formats := strings.Split(app.InstallerFormat, ":")
			if len(formats) > 0 {
				return formats[0], nil
			}
		}
	}

	return "", nil
}

type ingester struct {
	baseURL string
	ds      fleet.Datastore
	logger  kitlog.Logger
}

func (i ingester) ingest(ctx context.Context, apps []maintainedApp) error {
	var g errgroup.Group

	if !strings.HasSuffix(i.baseURL, "/") {
		i.baseURL += "/"
	}

	client := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))

	// run at most 3 concurrent requests to avoid overwhelming the brew API
	g.SetLimit(3)
	for _, app := range apps {
		app := app // capture loop variable, not required in Go 1.23+
		g.Go(func() error {
			return i.ingestOne(ctx, app, client)
		})
	}
	return ctxerr.Wrap(ctx, g.Wait(), "ingest apps")
}

func (i ingester) ingestOne(ctx context.Context, app maintainedApp, client *http.Client) error {
	apiURL := fmt.Sprintf("%scask/%s.json", i.baseURL, app.Identifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create http request")
	}

	res, err := client.Do(req)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "execute http request")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "read http response body")
	}

	switch res.StatusCode {
	case http.StatusOK:
		// success, go on
	case http.StatusNotFound:
		// nothing to do, either it currently exists in the DB and it keeps on
		// existing, or it doesn't and keeps on being missing.
		level.Warn(i.logger).Log("msg", "maintained app missing in brew API", "identifier", app.Identifier)
		return nil
	default:
		if len(body) > 512 {
			body = body[:512]
		}
		return ctxerr.Errorf(ctx, "brew API returned status %d: %s", res.StatusCode, string(body))
	}

	var cask brewCask
	if err := json.Unmarshal(body, &cask); err != nil {
		return ctxerr.Wrapf(ctx, err, "unmarshal brew cask for %s", app.Identifier)
	}

	// validate required fields
	if len(cask.Name) == 0 || cask.Name[0] == "" {
		return ctxerr.Errorf(ctx, "missing name for cask %s", app.Identifier)
	}
	if cask.Token == "" {
		return ctxerr.Errorf(ctx, "missing token for cask %s", app.Identifier)
	}
	if cask.Version == "" {
		return ctxerr.Errorf(ctx, "missing version for cask %s", app.Identifier)
	}
	if cask.URL == "" {
		return ctxerr.Errorf(ctx, "missing URL for cask %s", app.Identifier)
	}
	_, err = url.Parse(cask.URL)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "parse URL for cask %s", app.Identifier)
	}

	installScript, err := installScriptForApp(app, &cask)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "create install script for cask %s", app.Identifier)
	}

	cask.PreUninstallScripts = app.PreUninstallScripts
	cask.PostUninstallScripts = app.PostUninstallScripts
	uninstallScript := uninstallScriptForApp(&cask)

	_, err = i.ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:    cask.Name[0],
		Token:   cask.Token,
		Version: cask.Version,
		// for now, maintained apps are always macOS (darwin)
		Platform:         fleet.MacOSPlatform,
		InstallerURL:     cask.URL,
		SHA256:           cask.SHA256,
		BundleIdentifier: app.BundleIdentifier,
		InstallScript:    installScript,
		UninstallScript:  uninstallScript,
	})
	return ctxerr.Wrap(ctx, err, "upsert maintained app")
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
	App []string `json:"app"`
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
