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
	"path"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"golang.org/x/sync/errgroup"
)

//go:embed apps.json
var appsJSON []byte

type maintainedApp struct {
	Identifier       string `json:"identifier"`
	BundleIdentifier string `json:"bundle_identifier"`
	InstallerFormat  string `json:"installer_format"`
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
		// TODO: delete the existing entry? do nothing and succeed? doing the latter for now.
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
	parsedURL, err := url.Parse(cask.URL)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "parse URL for cask %s", app.Identifier)
	}
	filename := path.Base(parsedURL.Path)

	installScript := installScriptForApp(app, &cask)
	uninstallScript := uninstallScriptForApp(app, &cask)

	err = i.ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
		Name:    cask.Name[0],
		Token:   cask.Token,
		Version: cask.Version,
		// for now, maintained apps are always macOS (darwin)
		Platform:         fleet.MacOSPlatform,
		InstallerURL:     cask.URL,
		Filename:         filename,
		SHA256:           cask.SHA256,
		BundleIdentifier: app.BundleIdentifier,
		InstallScript:    installScript,
		UninstallScript:  uninstallScript,
	})
	return ctxerr.Wrap(ctx, err, "upsert maintained app")
}

func installScriptForApp(app maintainedApp, cask *brewCask) string {
	// TODO: implement install script based on cask and app installer format
	return "install"
}

func uninstallScriptForApp(app maintainedApp, cask *brewCask) string {
	// TODO: implement uninstall script based on cask and app installer format
	return "uninstall"
}

type brewCask struct {
	Token     string          `json:"token"`
	FullToken string          `json:"full_token"`
	Tap       string          `json:"tap"`
	Name      []string        `json:"name"`
	Desc      string          `json:"desc"`
	URL       string          `json:"url"`
	Version   string          `json:"version"`
	SHA256    string          `json:"sha256"`
	Artifacts []*brewArtifact `json:"artifacts"`
}

// brew artifacts are objects that have one and only one of their fields set.
type brewArtifact struct {
	App []string `json:"app"`
	// Pkg is a bit like Binary, it is an array with a string and an object as
	// first two elements. The object has a choices field with an array of
	// objects. See Microsoft Edge.
	Pkg       []optjson.StringOr[*brewPkgChoices] `json:"pkg"`
	Uninstall []*brewUninstall                    `json:"uninstall"`
	Zap       []*brewZap                          `json:"zap"`
	// Binary is an array with a string and an object as first two elements. See
	// the "docker" and "firefox" casks.
	Binary []optjson.StringOr[*brewBinaryTarget] `json:"binary"`
}

type brewPkgChoices struct {
	// At the moment we don't care about the "choices" field on the pkg.
	Choices []any `json:"choices"`
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
	Script    optjson.StringOr[[]string] `json:"script"`
	// format: [0]=signal, [1]=process name
	Signal optjson.StringOr[[]string] `json:"signal"`
	Delete optjson.StringOr[[]string] `json:"delete"`
	RmDir  optjson.StringOr[[]string] `json:"rmdir"`
}

// same as brewUninstall, can be []string or string (see Microsoft Teams).
type brewZap struct {
	Trash optjson.StringOr[[]string] `json:"trash"`
	RmDir optjson.StringOr[[]string] `json:"rmdir"`
}
