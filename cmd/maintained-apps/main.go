package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/ee/maintained-apps/ingesters/homebrew"
	"github.com/fleetdm/fleet/v4/ee/maintained-apps/ingesters/winget"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func main() {
	slugPtr := flag.String("slug", "", "app slug")
	debugPtr := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()
	ctx := context.Background()
	logger := kitlog.NewJSONLogger(os.Stderr)
	lvl := level.AllowInfo()
	if *debugPtr {
		lvl = level.AllowDebug()
	}
	logger = level.NewFilter(logger, lvl)
	logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)

	level.Info(logger).Log("msg", "starting maintained app ingestion")

	ingesters := map[string]maintained_apps.Ingester{
		"ee/maintained-apps/inputs/homebrew": homebrew.IngestApps,
		"ee/maintained-apps/inputs/winget":   winget.IngestApps,
	}

	for p, i := range ingesters {
		apps, err := i(ctx, logger, p, *slugPtr)
		if err != nil {
			level.Error(logger).Log("msg", "failed to ingest apps", "error", err)
		}

		for _, app := range apps {

			if app.IsEmpty() {
				level.Info(logger).Log("msg", "skipping manifest update due to empty output", "slug", app.Slug)
				continue
			}

			if err := processOutput(ctx, app); err != nil {
				level.Error(logger).Log("msg", "failed to process maintained app output", "err", err)
			}
		}
	}
}

func processOutput(ctx context.Context, app *maintained_apps.FMAManifestApp) error {
	if err := updateAppsListFile(ctx, app); err != nil {
		return ctxerr.Wrap(ctx, err, "updating apps list file")
	}
	app.UniqueIdentifier = "" // make sure we don't leak unique_identifier into individual app manifests

	outFile := maintained_apps.FMAManifestFile{
		Versions: []*maintained_apps.FMAManifestApp{app},
		Refs:     map[string]string{app.UninstallScriptRef: app.UninstallScript, app.InstallScriptRef: app.InstallScript},
	}

	outBytes, err := json.MarshalIndent(outFile, "", "  ")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling output app manifest")
	}

	// Overwrite the file, since right now we're only caring about 1 version (latest). If we
	// care about previous data, it will be in our Git history.
	outPath := path.Join(maintained_apps.OutputPath, app.SlugAppName())

	if err := os.MkdirAll(outPath, os.ModePerm); err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	if err := os.WriteFile(path.Join(maintained_apps.OutputPath, fmt.Sprintf("%s.json", app.Slug)), outBytes, 0o644); err != nil {
		return ctxerr.Wrap(ctx, err, "writing output json file")
	}

	return nil
}

func updateAppsListFile(ctx context.Context, outApp *maintained_apps.FMAManifestApp) error {
	appListFilePath := path.Join(maintained_apps.OutputPath, "apps.json")
	file, err := os.ReadFile(appListFilePath)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading output apps list file")
	}

	var outputAppsFile maintained_apps.FMAListFile
	if err := json.Unmarshal(file, &outputAppsFile); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshaling output apps list file")
	}

	var found bool
	for _, a := range outputAppsFile.Apps {
		if a.Slug == outApp.Slug {
			found = true
			break
		}
	}

	if !found {
		platform := outApp.Platform()
		if platform == "" {
			return ctxerr.New(ctx, fmt.Sprintf("invalid platform found for slug %s", outApp.Slug))
		}

		outputAppsFile.Apps = append(outputAppsFile.Apps, maintained_apps.FMAListFileApp{
			Name:             outApp.Name,
			Slug:             outApp.Slug,
			Platform:         platform,
			UniqueIdentifier: outApp.UniqueIdentifier,
		})

		// Keep existing order
		slices.SortFunc(outputAppsFile.Apps, func(a, b maintained_apps.FMAListFileApp) int { return strings.Compare(a.Slug, b.Slug) })

		updatedFile, err := json.MarshalIndent(outputAppsFile, "", "  ")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshaling updated output apps file")
		}

		if err := os.WriteFile(appListFilePath, updatedFile, 0o644); err != nil {
			return ctxerr.Wrap(ctx, err, "writing updated output apps file")
		}
	}

	return nil
}
