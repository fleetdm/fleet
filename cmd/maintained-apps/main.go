package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path"
	"slices"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/ee/maintained-apps/ingesters/homebrew"
	"github.com/fleetdm/fleet/v4/ee/maintained-apps/ingesters/winget"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

func main() {
	slugPtr := flag.String("slug", "", "app slug")
	debugPtr := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()
	ctx := context.Background()
	logLevel := slog.LevelInfo
	if *debugPtr {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	logger.InfoContext(ctx, "starting maintained app ingestion")

	ingesters := map[string]maintained_apps.Ingester{
		"ee/maintained-apps/inputs/homebrew": homebrew.IngestApps,
		"ee/maintained-apps/inputs/winget":   winget.IngestApps,
	}

	for inputDir, ingest := range ingesters {
		apps, err := ingest(ctx, logger, inputDir, *slugPtr)
		if err != nil {
			panic(err)
		}

		for _, app := range apps {

			if app.IsEmpty() {
				logger.InfoContext(ctx, "skipping manifest update due to empty output", "slug", app.Slug)
				continue
			}

			if err := processOutput(ctx, app); err != nil {
				logger.ErrorContext(ctx, "failed to process maintained app output", "err", err)
			}
		}
	}
}

func appsListJSONPath() string {
	return path.Join(maintained_apps.OutputPath, "apps.json")
}

func processOutput(ctx context.Context, app *maintained_apps.FMAManifestApp) error {
	// validate categories before writing any files
	if err := validateCategories(ctx, app); err != nil {
		// Make the validation failure very obvious on stderr.
		fmt.Fprintf(
			os.Stderr,
			"maintained-apps: fatal error processing %s: %v\n",
			app.Slug,
			err,
		)
		// Wrap so callers still see a proper error.
		return ctxerr.Wrap(ctx, err, "validating categories")
	}

	if err := updateAppsListFileAt(ctx, appsListJSONPath(), app); err != nil {
		return ctxerr.Wrap(ctx, err, "updating apps list file")
	}
	app.UniqueIdentifier = "" // make sure we don't leak unique_identifier into individual app manifests

	outFile := maintained_apps.FMAManifestFile{
		Versions: []*maintained_apps.FMAManifestApp{app},
		Refs:     map[string]string{app.UninstallScriptRef: app.UninstallScript, app.InstallScriptRef: app.InstallScript},
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(outFile); err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling output app manifest")
	}
	outBytes := buf.Bytes()

	outDir := path.Join(maintained_apps.OutputPath, app.SlugAppName())

	if err := os.MkdirAll(outDir, os.ModePerm); err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	outFilePath := path.Join(maintained_apps.OutputPath, fmt.Sprintf("%s.json", app.Slug))
	outFileExists, err := file.Exists(outFilePath)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking if output json file exists")
	}

	// Overwrite the file unless frozen, since right now we're only caring about 1 version (latest). If we
	// care about previous data, it will be in our Git history.
	if !app.Frozen || !outFileExists {
		if err := os.WriteFile(outFilePath, outBytes, 0o644); err != nil {
			return ctxerr.Wrap(ctx, err, "writing output json file")
		}
	}

	return nil
}

// Match types in frontend/interfaces/software.ts
var allowedCategories = map[string]struct{}{
	"Browsers":        {},
	"Communication":   {},
	"Developer tools": {},
	"Productivity":    {},
	"Security":        {},
	"Utilities":       {},
}

func allowedCategoriesString() string {
	cats := make([]string, 0, len(allowedCategories))
	for c := range allowedCategories {
		cats = append(cats, c)
	}
	slices.Sort(cats)
	return strings.Join(cats, ", ")
}

// validateCategories ensures every category on the app is one of the supported values.
func validateCategories(ctx context.Context, app *maintained_apps.FMAManifestApp) error {
	for _, c := range app.DefaultCategories {
		if _, ok := allowedCategories[c]; !ok {
			return ctxerr.New(ctx, fmt.Sprintf(
				"invalid category %q for slug %s (allowed: %s)",
				c, app.Slug, allowedCategoriesString(),
			))
		}
	}
	return nil
}

func listFileAppFromManifest(outApp *maintained_apps.FMAManifestApp) (maintained_apps.FMAListFileApp, error) {
	platform := outApp.Platform()
	if platform == "" {
		return maintained_apps.FMAListFileApp{}, fmt.Errorf("invalid platform found for slug %s", outApp.Slug)
	}
	return maintained_apps.FMAListFileApp{
		Name:             outApp.Name,
		Slug:             outApp.Slug,
		Platform:         platform,
		UniqueIdentifier: outApp.UniqueIdentifier,
	}, nil
}

func writeAppsListJSON(appListFilePath string, outputAppsFile *maintained_apps.FMAListFile) error {
	slices.SortFunc(outputAppsFile.Apps, func(a, b maintained_apps.FMAListFileApp) int {
		return strings.Compare(a.Slug, b.Slug)
	})
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(outputAppsFile); err != nil {
		return fmt.Errorf("marshaling apps list: %w", err)
	}
	if err := os.WriteFile(appListFilePath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing apps list: %w", err)
	}
	return nil
}

func updateAppsListFileAt(ctx context.Context, appListFilePath string, outApp *maintained_apps.FMAManifestApp) error {
	newRow, err := listFileAppFromManifest(outApp)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building apps list row")
	}

	inputJson, err := os.ReadFile(appListFilePath)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading output apps list file")
	}

	var outputAppsFile maintained_apps.FMAListFile
	if err := json.Unmarshal(inputJson, &outputAppsFile); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshaling output apps list file")
	}

	for i, a := range outputAppsFile.Apps {
		if a.Slug != outApp.Slug {
			continue
		}
		updated := newRow
		updated.Description = a.Description
		if updated == a {
			return nil
		}
		outputAppsFile.Apps[i] = updated
		if err := writeAppsListJSON(appListFilePath, &outputAppsFile); err != nil {
			return ctxerr.Wrap(ctx, err, "writing updated output apps file")
		}
		return nil
	}

	outputAppsFile.Apps = append(outputAppsFile.Apps, newRow)
	if err := writeAppsListJSON(appListFilePath, &outputAppsFile); err != nil {
		return ctxerr.Wrap(ctx, err, "writing updated output apps file")
	}
	return nil
}
