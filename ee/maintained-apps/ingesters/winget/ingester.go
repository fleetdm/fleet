package winget

import (
	"context"
	"encoding/json"
	"os"
	"path"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func IngestApps(ctx context.Context, logger kitlog.Logger, inputsPath string) ([]*maintained_apps.FMAManifestApp, error) {
	level.Info(logger).Log("msg", "starting winget app data ingestion")
	// Read from our list of apps we should be ingesting
	files, err := os.ReadDir(inputsPath)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reading winget input data directory")
	}

	var manifestApps []*maintained_apps.FMAManifestApp

	for _, f := range files {

		fileBytes, err := os.ReadFile(path.Join(inputsPath, f.Name()))
		if err != nil {
			return nil, ctxerr.WrapWithData(ctx, err, "reading app input file", map[string]any{"fileName": f.Name()})
		}

		var input inputApp
		if err := json.Unmarshal(fileBytes, &input); err != nil {
			return nil, ctxerr.WrapWithData(ctx, err, "unmarshal app input file", map[string]any{"fileName": f.Name()})
		}

		level.Info(logger).Log("msg", "ingesting winget app", "name", input.Name)

		// TODO: fully implement this ingester, right now it's just a stub/noop

		manifestApps = append(manifestApps, &maintained_apps.FMAManifestApp{Name: input.Name, Slug: input.Slug})

	}

	return manifestApps, nil
}

type inputApp struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}
