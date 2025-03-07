package darwin

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
)

//go:embed *.json
var appsJSON embed.FS

const baseBrewAPIURL = "https://formulae.brew.sh/api/"

type brewIngester struct {
	baseURL string
	logger  kitlog.Logger
	client  *http.Client
}

func (i *brewIngester) IngestOne(ctx context.Context, app maintained_apps.InputApp) (*maintained_apps.FMAManifestApp, map[string]string, error) {
	level.Debug(i.logger).Log("msg", "ingesting app", "name", app.Name)

	apiURL := fmt.Sprintf("%scask/%s.json", i.baseURL, app.SourceIdentifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "create http request")
	}

	res, err := i.client.Do(req)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "execute http request")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "read http response body")
	}

	switch res.StatusCode {
	case http.StatusOK:
		// success, go on
	case http.StatusNotFound:
		// nothing to do
		level.Warn(i.logger).Log("msg", "maintained app missing in brew API", "identifier", app.SourceIdentifier)
		return nil, nil, nil
	default:
		if len(body) > 512 {
			body = body[:512]
		}
		return nil, nil, ctxerr.Errorf(ctx, "brew API returned status %d: %s", res.StatusCode, string(body))
	}

	var cask brewCask
	if err := json.Unmarshal(body, &cask); err != nil {
		return nil, nil, ctxerr.Wrapf(ctx, err, "unmarshal brew cask for %s", app.SourceIdentifier)
	}

	out := &maintained_apps.FMAManifestApp{}

	if cask.URL == "" {
		return nil, nil, ctxerr.Errorf(ctx, "missing URL for cask %s", app.SourceIdentifier)
	}
	_, err = url.Parse(cask.URL)
	if err != nil {
		return nil, nil, ctxerr.Wrapf(ctx, err, "parse URL for cask %s", app.SourceIdentifier)
	}

	// TODO(JVE): we should no-op if there was no change, so we need to read in the existing file
	// and diff it here

	out.Version = cask.Version
	out.InstallerURL = cask.URL
	out.UniqueIdentifier = app.UniqueIdentifier
	out.Sha256 = cask.SHA256
	out.Queries = map[string]string{maintained_apps.ExistsKey: fmt.Sprintf("SELECT 1 FROM apps WHERE bundle_identifier = '%s';", out.UniqueIdentifier)}
	out.Description = cask.Desc

	// Script generation
	scriptRefs := make(map[string]string)
	uninstallScript := uninstallScriptForApp(&cask)
	uninstallRef := uuid.NewString()
	out.UninstallScriptRef = uninstallRef
	installScript, err := installScriptForApp(app, &cask)
	if err != nil {
		return nil, nil, ctxerr.WrapWithData(ctx, err, "generating install script for maintained app", map[string]any{"unique_identifier": app.UniqueIdentifier})
	}
	installRef := uuid.NewString()
	scriptRefs[installRef] = installScript
	out.InstallScriptRef = installRef

	scriptRefs[uninstallRef] = uninstallScript

	return out, scriptRefs, nil
}

func (i *darwinIngester) IngestApps(ctx context.Context) error {
	// Read from our list of apps we should be ingesting
	files, err := appsJSON.ReadDir(".")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading embedded data directory")
	}

	for _, f := range files {
		fileBytes, err := appsJSON.ReadFile(f.Name())
		if err != nil {
			return ctxerr.Wrap(ctx, err, "reading app input file")
		}

		var input maintained_apps.InputApp
		if err := json.Unmarshal(fileBytes, &input); err != nil {
			return ctxerr.Wrap(ctx, err, "unmarshal app input file")
		}

		if input.SourceIdentifier == "" {
			return ctxerr.New(ctx, "missing source identifier for app")
		}

		if input.UniqueIdentifier == "" {
			return ctxerr.New(ctx, "missing unique identifier for app")
		}

		if input.Name == "" {
			return ctxerr.Wrap(ctx, err, "missing name for app")
		}

		if input.Source == "" {
			return ctxerr.New(ctx, "missing source for app")
		}

		ingester, ok := i.sourceIngesters[input.Source]
		if !ok {
			return ctxerr.New(ctx, "invalid source for app")
		}

		outApp, scripts, err := ingester.IngestOne(ctx, input)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "ingesting app")
		}

		outFile := maintained_apps.FMAManifestFile{
			Versions: []*maintained_apps.FMAManifestApp{outApp},
			Refs:     scripts,
		}

		outBytes, err := json.MarshalIndent(outFile, "", "  ")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshaling output app manifest")
		}

		// Overwrite the file, since right now we're only caring about 1 version (latest). If we
		// care about previous data, it will be in our Git history.
		if err := os.WriteFile(fmt.Sprintf("ee/maintained-apps/outputs/darwin/%s.json", input.SourceIdentifier), outBytes, 0o644); err != nil {
			return ctxerr.Wrap(ctx, err, "writing output json file")
		}

		if err := updateAppsListFile(ctx, input, outApp); err != nil {
			return ctxerr.Wrap(ctx, err, "updating apps list file")
		}

	}

	return nil
}

func updateAppsListFile(ctx context.Context, input maintained_apps.InputApp, outApp *maintained_apps.FMAManifestApp) error {
	file, err := os.ReadFile("ee/maintained-apps/outputs/apps.json")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading output apps list file")
	}

	var outputAppsFile maintained_apps.FMAListFile
	if err := json.Unmarshal(file, &outputAppsFile); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshaling output apps list file")
	}

	var found bool
	for _, a := range outputAppsFile.Apps {
		if a.Slug == fmt.Sprintf("%s/%s", input.SourceIdentifier, fleet.MacOSPlatform) {
			found = true
			break
		}
	}

	if !found {
		outputAppsFile.Apps = append(outputAppsFile.Apps, maintained_apps.FMAListFileApp{
			Name:             input.Name,
			Slug:             fmt.Sprintf("%s/%s", input.SourceIdentifier, fleet.MacOSPlatform),
			Platform:         string(fleet.MacOSPlatform),
			UniqueIdentifier: outApp.UniqueIdentifier,
			Description:      outApp.Description,
		})

		// Keep existing order
		slices.SortFunc(outputAppsFile.Apps, func(a, b maintained_apps.FMAListFileApp) int { return strings.Compare(a.Slug, b.Slug) })

		updatedFile, err := json.Marshal(outputAppsFile)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshaling updated output apps file")
		}

		if err := os.WriteFile("ee/maintained-apps/outputs/apps.json", updatedFile, 0o644); err != nil {
			return ctxerr.Wrap(ctx, err, "writing updated output apps file")
		}
	}

	return nil
}

type darwinIngester struct {
	sourceIngesters map[string]maintained_apps.SourceIngester
}

func NewDarwinIngester(logger kitlog.Logger) maintained_apps.Ingester {
	baseURL := baseBrewAPIURL
	// Use FLEET_DEV_BREW_API_URL for automated or manual testing purposes
	if v := os.Getenv("FLEET_DEV_BREW_API_URL"); v != "" {
		baseURL = v
	}

	return &darwinIngester{
		sourceIngesters: map[string]maintained_apps.SourceIngester{
			maintained_apps.SourceHomebrew: &brewIngester{
				baseURL: baseURL,
				logger:  logger,
				client:  fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
			},
		},
	}
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
