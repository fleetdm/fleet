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

type wingetIngester struct {
	logger kitlog.Logger
}

func NewWingetIngester(logger kitlog.Logger) maintained_apps.Ingester {
	return &wingetIngester{
		logger: logger,
	}
}

func (i *wingetIngester) IngestApps(ctx context.Context, inputsPath string) ([]*maintained_apps.FMAManifestApp, error) {
	level.Info(i.logger).Log("msg", "starting winget app data ingestion")
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

		level.Info(i.logger).Log("msg", "ingesting winget app", "name", input.Name)

		// TODO: fully implement this ingester, right now it's just a stub/noop

		manifestApps = append(manifestApps, &maintained_apps.FMAManifestApp{Name: input.Name, Slug: input.Slug})

	}

	return manifestApps, nil
}

func (i *wingetIngester) IngestApp(ctx context.Context, inputPath string) (*maintained_apps.FMAManifestApp, error) {
	// TODO: implement
	return nil, nil
}

type inputApp struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}
