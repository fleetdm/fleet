package main

import (
	"context"
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/ee/maintained-apps/inputs/darwin"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type brewIngester struct {
	baseURL   string
	logger    kitlog.Logger
	client    *http.Client
	inputData embed.FS
}

type inputApp struct {
	Name             string `json:"name"`
	Identifier       string `json:"identifier"`
	UniqueIdentifier string `json:"unique_identifier"`
}

type Ingester interface {
	IngestApps(ctx context.Context) error
}

// TODO(JVE): better name?
type outputApp struct {
	Version string `json:"version"`
	Queries struct {
		Exists string `json:"exists"`
	} `json:"queries"`
	InstallerURL       string `json:"installer_url"`
	UniqueIdentifier   string `json:"unique_identifier"`
	InstallScriptRef   string `json:"install_script_ref"`
	UninstallScriptRef string `json:"uninstall_script_ref"`
	Sha256             string `json:"sha256"`
}

type outputFile struct {
	Versions []*outputApp      `json:"versions"`
	Refs     map[string]string `json:"refs"`
}

func (i *brewIngester) ingestOne(ctx context.Context, app inputApp) (*outputApp, error) {
	level.Debug(i.logger).Log("msg", "ingesting app", "name", app.Name)

	apiURL := fmt.Sprintf("%scask/%s.json", i.baseURL, app.Identifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create http request")
	}

	res, err := i.client.Do(req)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "execute http request")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read http response body")
	}

	switch res.StatusCode {
	case http.StatusOK:
		// success, go on
	case http.StatusNotFound:
		// nothing to do
		level.Warn(i.logger).Log("msg", "maintained app missing in brew API", "identifier", app.Identifier)
		return nil, nil
	default:
		if len(body) > 512 {
			body = body[:512]
		}
		return nil, ctxerr.Errorf(ctx, "brew API returned status %d: %s", res.StatusCode, string(body))
	}

	var cask brewCask
	if err := json.Unmarshal(body, &cask); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "unmarshal brew cask for %s", app.Identifier)
	}

	out := &outputApp{}

	if cask.URL == "" {
		return nil, ctxerr.Errorf(ctx, "missing URL for cask %s", app.Identifier)
	}
	_, err = url.Parse(cask.URL)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "parse URL for cask %s", app.Identifier)
	}

	out.Version = cask.Version
	out.InstallerURL = cask.URL
	out.UniqueIdentifier = app.UniqueIdentifier
	// TODO(JVE): add missing fields to output

	return out, nil
}

const baseBrewAPIURL = "https://formulae.brew.sh/api/"

func main() {
	ctx := context.Background()
	logger := kitlog.NewJSONLogger(os.Stderr)
	logger = level.NewFilter(logger, level.AllowDebug())
	logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)

	level.Info(logger).Log("msg", "starting maintained app ingestion")

	var ingesters []Ingester

	// init ingesters for different platforms
	brewIng := &brewIngester{
		baseURL:   baseBrewAPIURL,
		logger:    logger,
		client:    fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
		inputData: darwin.AppsJSON,
	}

	ingesters = append(ingesters, brewIng)

	for _, i := range ingesters {
		if err := i.IngestApps(ctx); err != nil {
			level.Error(logger).Log("msg", "failed to ingest apps", "error", err)
		}

		// TODO(JVE): open PR
	}
}

func (i *brewIngester) IngestApps(ctx context.Context) error {
	// Read from our list of apps we should be ingesting

	files, err := i.inputData.ReadDir(".")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading embedded data directory")
	}

	// TODO(JVE): probably can introduce some concurrency here
	for _, f := range files {
		fileBytes, err := i.inputData.ReadFile(f.Name())
		if err != nil {
			return ctxerr.Wrap(ctx, err, "reading app input file")
		}

		var input inputApp
		if err := json.Unmarshal(fileBytes, &input); err != nil {
			return ctxerr.Wrap(ctx, err, "unmarshal app input file")
		}

		if input.Identifier != "" {
			outApp, err := i.ingestOne(ctx, input)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "ingesting app")
			}

			outFile := outputFile{
				Versions: []*outputApp{outApp},
			}

			outBytes, err := json.MarshalIndent(outFile, "", "  ")
			if err != nil {
				return ctxerr.Wrap(ctx, err, "marshaling output app manifest")
			}

			fmt.Printf("outBytes: %v\n", string(outBytes))

			// Overwrite the file, since right now we're only caring about 1 version (latest). If we
			// care about previous data, it will be in our Git history.
			if err := os.WriteFile(fmt.Sprintf("ee/maintained-apps/outputs/darwin/%s.json", input.Identifier), outBytes, 0o644); err != nil {
				return ctxerr.Wrap(ctx, err, "writing output json file")
			}

			// TODO(JVE): update the output apps.json file

		}

	}

	return nil
}

// TODO(JVE): brew specific types. move this + ingester logic to a file in the
// ee/maintained-apps/inputs/darwin directory.

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
