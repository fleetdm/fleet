// Command profile-schema-gen generates pkg/mdm/profilevalidation/outputs/profile_schema.json from
// Apple's apple/device-management repo.
//
// Run it from the repository root:
//
//	go run ./tools/profile-schema-gen -update   # fetch the latest release-branch commit and regenerate
//	go run ./tools/profile-schema-gen           # regenerate from the already downloaded pinned commit, offline
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
)

// workDir holds downloaded upstream sources, ignored by git. Runs without -update read only from
// here, so they make no network calls.
const workDir = "tools/profile-schema-gen/downloads"

// commitMarker records which upstream commit a download directory holds.
const commitMarker = ".commit"

func main() {
	out := flag.String("out", "pkg/mdm/profilevalidation/outputs/profile_schema.json", "path of the generated schema; its version block is the pin for runs without -update")
	update := flag.Bool("update", false, "fetch the latest upstream commit and regenerate from it")
	verbose := flag.Bool("v", false, "log every file and decision (debug level)")
	flag.Parse()

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	ctx := context.Background()
	if err := run(ctx, logger, *out, *update); err != nil {
		logger.ErrorContext(ctx, "generation failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger, out string, update bool) error {
	// Paths are relative to the repo root; fail clearly instead of writing somewhere unexpected.
	if _, err := os.Stat(workDir); err != nil {
		return errors.New("run from the repository root: " + workDir + " not found")
	}

	existing, err := readOutput(out)
	if err != nil {
		return err
	}
	var pinnedApple *profilevalidation.AppleVersion
	if existing == nil {
		logger.InfoContext(ctx, "no existing output", "out", out)
	} else {
		v := existing.Version
		logger.InfoContext(ctx, "existing output", "out", out, "apple_release", v.Apple.Release,
			"apple_commit", v.Apple.Commit, "generated_at", v.GeneratedAt)
		if v.Apple.Commit != "" {
			pinnedApple = &v.Apple
		}
	}

	client := fleethttp.NewClient(fleethttp.WithTimeout(2 * time.Minute))
	appleLogger := logger.With("platform", "apple")
	appleDir := filepath.Join(workDir, "apple")
	var schema profilevalidation.ProfileValidationSchema
	if schema.Version.Apple, err = runApple(ctx, appleLogger, client, githubAPIURL, codeloadURL, appleDir, pinnedApple, update); err != nil {
		return err
	}
	src, err := loadApple(ctx, appleLogger, appleDir)
	if err != nil {
		return err
	}
	if err := convertApple(ctx, appleLogger, src, &schema); err != nil {
		return err
	}
	if err := runWindows(ctx, logger.With("platform", "windows"), &schema); err != nil {
		return err
	}
	return writeOutput(ctx, logger, out, &schema, existing, time.Now())
}

// runApple returns the Apple version to generate and leaves its sources in dir. Without update it
// uses the pinned commit's existing download and makes no network calls.
func runApple(ctx context.Context, logger *slog.Logger, client *http.Client, apiURL, tarballURL, dir string, pinned *profilevalidation.AppleVersion, update bool) (profilevalidation.AppleVersion, error) {
	if !update {
		if pinned == nil {
			return profilevalidation.AppleVersion{}, errors.New("no Apple commit pinned in the output; run with -update to fetch the latest")
		}
		downloaded, err := os.ReadFile(filepath.Join(dir, commitMarker))
		if err != nil {
			return profilevalidation.AppleVersion{}, fmt.Errorf("no Apple download in %s for pinned commit %s; run with -update to fetch the latest", dir, pinned.Commit)
		}
		if got := strings.TrimSpace(string(downloaded)); got != pinned.Commit {
			return profilevalidation.AppleVersion{}, fmt.Errorf("%s holds Apple commit %s but the output pins %s; run with -update to fetch the latest", dir, got, pinned.Commit)
		}
		logger.InfoContext(ctx, "using existing download for pinned commit", "dir", dir, "release", pinned.Release, "commit", pinned.Commit)
		return *pinned, nil
	}

	commit, err := resolveCommit(ctx, logger, client, apiURL, appleRepo, appleBranch)
	if err != nil {
		return profilevalidation.AppleVersion{}, err
	}
	// Every commit on Apple's release branch is titled with the release name, e.g. "Release-v27.0".
	version := profilevalidation.AppleVersion{Release: commit.Subject, Commit: commit.SHA}
	if version.Release == "" {
		return profilevalidation.AppleVersion{}, fmt.Errorf("commit %s on %s@%s has an empty message; can't name the release", commit.SHA, appleRepo, appleBranch)
	}
	switch {
	case pinned == nil:
		logger.InfoContext(ctx, "no Apple pin; using latest", "release", version.Release, "commit", version.Commit)
	case pinned.Commit == version.Commit:
		logger.InfoContext(ctx, "latest Apple commit matches pin; regenerating anyway", "release", version.Release, "commit", version.Commit)
	default:
		logger.InfoContext(ctx, "moving Apple pin to latest", "from_release", pinned.Release, "from_commit", pinned.Commit,
			"to_release", version.Release, "to_commit", version.Commit)
	}

	logger.DebugContext(ctx, "clearing previous download", "dir", dir)
	if err := os.RemoveAll(dir); err != nil {
		return profilevalidation.AppleVersion{}, err
	}
	if err := downloadAppleTarball(ctx, logger, client, tarballURL, version.Commit, dir); err != nil {
		return profilevalidation.AppleVersion{}, err
	}
	// Written last, so an interrupted download never looks complete.
	if err := os.WriteFile(filepath.Join(dir, commitMarker), []byte(version.Commit+"\n"), 0o644); err != nil {
		return profilevalidation.AppleVersion{}, err
	}
	return version, nil
}

func runWindows(ctx context.Context, logger *slog.Logger, schema *profilevalidation.ProfileValidationSchema) error {
	logger.InfoContext(ctx, "skipping: Windows sections are not generated yet; writing them empty")
	schema.Windows = json.RawMessage("{}")
	schema.WindowsXSDs = json.RawMessage("{}")
	return nil
}
