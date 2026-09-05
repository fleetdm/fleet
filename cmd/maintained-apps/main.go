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

	if err := updateAppsListFile(ctx, app); err != nil {
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
	"Support":         {},
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

func updateAppsListFile(ctx context.Context, outApp *maintained_apps.FMAManifestApp) error {
	appListFilePath := path.Join(maintained_apps.OutputPath, "apps.json")
	inputJson, err := os.ReadFile(appListFilePath)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading output apps list file")
	}

	var outputAppsFile maintained_apps.FMAListFile
	if err := json.Unmarshal(inputJson, &outputAppsFile); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshaling output apps list file")
	}

	platform := outApp.Platform()
	if platform == "" {
		return ctxerr.New(ctx, fmt.Sprintf("invalid platform found for slug %s", outApp.Slug))
	}

	var changed bool
	idx := slices.IndexFunc(outputAppsFile.Apps, func(a maintained_apps.FMAListFileApp) bool { return a.Slug == outApp.Slug })
	switch idx {
	case -1:
		outputAppsFile.Apps = append(outputAppsFile.Apps, maintained_apps.FMAListFileApp{
			Name:             outApp.Name,
			Slug:             outApp.Slug,
			Platform:         platform,
			UniqueIdentifier: outApp.UniqueIdentifier,
		})
		changed = true

		// Keep existing order
		slices.SortFunc(outputAppsFile.Apps, func(a, b maintained_apps.FMAListFileApp) int { return strings.Compare(a.Slug, b.Slug) })
	default:
		// The app is already listed, but the input manifest is the source of truth for its
		// identity fields, so pick up any edits to them. apps.json is what the Fleet server syncs
		// into fleet_maintained_apps, so leaving a stale unique_identifier here ships the stale
		// value even though the per-app manifest was regenerated. Description has no manifest
		// counterpart, so it is left untouched.
		existing := &outputAppsFile.Apps[idx]
		if existing.Name != outApp.Name {
			existing.Name = outApp.Name
			changed = true
		}
		if existing.UniqueIdentifier != outApp.UniqueIdentifier {
			existing.UniqueIdentifier = outApp.UniqueIdentifier
			changed = true
		}
		if existing.Platform != platform {
			existing.Platform = platform
			changed = true
		}
	}

	if !changed {
		return nil
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(outputAppsFile); err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling updated output apps file")
	}
	updatedFile := buf.Bytes()

	if err := os.WriteFile(appListFilePath, updatedFile, 0o644); err != nil {
		return ctxerr.Wrap(ctx, err, "writing updated output apps file")
	}

	return nil
}
