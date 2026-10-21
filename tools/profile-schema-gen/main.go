// Command profile-schema-gen generates pkg/mdm/profilevalidation/outputs/profile_schema.json from
// Apple's apple/device-management repo.
//
// Run it from the repository root:
//
//	go run ./tools/profile-schema-gen -update   # fetch the latest release-branch commit and regenerate
//	go run ./tools/profile-schema-gen           # regenerate from the already downloaded pinned commit, offline
//	go run ./tools/profile-schema-gen -fetch-pinned  # same, downloading the pinned commit if needed (CI)
package main

import (
	"context"
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

// commitMarker records which upstream commit a download directory holds; sha256Marker which zip.
const (
	commitMarker = ".commit"
	sha256Marker = ".sha256"
)

func main() {
	out := flag.String("out", "pkg/mdm/profilevalidation/outputs/profile_schema.json", "path of the generated schema; its version block is the pin for runs without -update")
	update := flag.Bool("update", false, "fetch the latest upstream commit and regenerate from it")
	fetchPinned := flag.Bool("fetch-pinned", false, "without -update, download the pinned commit if it isn't already downloaded (for CI)")
	overrides := flag.String("overrides", "pkg/mdm/profilevalidation/inputs/overrides.yml", `overrides file applied to the generated schema; skipped if missing, "" to skip`)
	verbose := flag.Bool("v", false, "log every file and decision (debug level)")
	flag.Parse()

	level := slog.LevelInfo
	if *verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	ctx := context.Background()
	if err := run(ctx, logger, *out, *overrides, *update, *fetchPinned); err != nil {
		// Override problems are reported together, one per line; log them as separate lines.
		for line := range strings.SplitSeq(err.Error(), "\n") {
			logger.ErrorContext(ctx, "generation failed", "err", line)
		}
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger, out, overrides string, update, fetchPinned bool) error {
	// Paths are relative to the repo root; fail clearly instead of writing somewhere unexpected.
	if _, err := os.Stat(workDir); err != nil {
		return errors.New("run from the repository root: " + workDir + " not found")
	}

	existing, err := readOutput(out)
	if err != nil {
		return err
	}
	var pinnedApple *profilevalidation.AppleVersion
	var pinnedWindows *profilevalidation.WindowsVersion
	if existing == nil {
		logger.InfoContext(ctx, "no existing output", "out", out)
	} else {
		v := existing.Version
		logger.InfoContext(ctx, "existing output", "out", out, "apple_release", v.Apple.Release,
			"apple_commit", v.Apple.Commit, "windows_release", v.Windows.Release, "windows_sha256", v.Windows.SHA256,
			"generated_at", v.GeneratedAt)
		if v.Apple.Commit != "" {
			pinnedApple = &v.Apple
		}
		if v.Windows.SHA256 != "" {
			pinnedWindows = &v.Windows
		}
	}

	client := fleethttp.NewClient(fleethttp.WithTimeout(2 * time.Minute))
	appleLogger := logger.With("platform", "apple")
	appleDir := filepath.Join(workDir, "apple")
	var schema profilevalidation.ProfileValidationSchema
	schema.Version.Apple, err = runApple(ctx, appleLogger, client, appleRunOptions{
		APIURL: githubAPIURL, TarballURL: codeloadURL, Dir: appleDir,
		Pinned: pinnedApple, Update: update, FetchPinned: fetchPinned,
	})
	if err != nil {
		return err
	}
	src, err := loadApple(ctx, appleLogger, appleDir)
	if err != nil {
		return err
	}
	if err := convertApple(ctx, appleLogger, src, &schema); err != nil {
		return err
	}
	windowsLogger := logger.With("platform", "windows")
	windowsDir := filepath.Join(workDir, "windows")
	schema.Version.Windows, err = runWindows(ctx, windowsLogger, client, windowsRunOptions{
		PageURL: ddfPageURL, Dir: windowsDir,
		Pinned: pinnedWindows, Update: update, FetchPinned: fetchPinned,
	})
	if err != nil {
		return err
	}
	if _, err := loadWindows(ctx, windowsLogger, windowsDir); err != nil {
		return err
	}
	schema.Windows = map[string]*profilevalidation.WindowsNode{}
	schema.WindowsXSDs = map[string]string{}
	if err := applyOverrides(ctx, logger.With("step", "overrides"), overrides, &schema); err != nil {
		return err
	}
	return writeOutput(ctx, logger, out, &schema, existing, time.Now())
}

type appleRunOptions struct {
	APIURL, TarballURL string
	Dir                string // where Apple's sources are downloaded
	Pinned             *profilevalidation.AppleVersion
	Update             bool // move to the latest release-branch commit
	FetchPinned        bool // without Update, download the pinned commit if dir doesn't hold it
}

// runApple returns the Apple version to generate and leaves its sources in o.Dir. Without
// o.Update it uses the pinned commit, and only touches the network to download it when
// o.FetchPinned is set (from codeload, which isn't API rate limited).
func runApple(ctx context.Context, logger *slog.Logger, client *http.Client, o appleRunOptions) (profilevalidation.AppleVersion, error) {
	if !o.Update {
		if o.Pinned == nil {
			return profilevalidation.AppleVersion{}, errors.New("no Apple commit pinned in the output; run with -update to fetch the latest")
		}
		pin := *o.Pinned
		downloaded, err := os.ReadFile(filepath.Join(o.Dir, commitMarker))
		got := strings.TrimSpace(string(downloaded))
		if err == nil && got == pin.Commit {
			logger.InfoContext(ctx, "using existing download for pinned commit", "dir", o.Dir, "release", pin.Release, "commit", pin.Commit)
			return pin, nil
		}
		if !o.FetchPinned {
			if err != nil {
				return profilevalidation.AppleVersion{}, fmt.Errorf("no Apple download in %s for pinned commit %s; run with -fetch-pinned to download it, or -update to move to the latest", o.Dir, pin.Commit)
			}
			return profilevalidation.AppleVersion{}, fmt.Errorf("%s holds Apple commit %s but the output pins %s; run with -fetch-pinned to download it, or -update to move to the latest", o.Dir, got, pin.Commit)
		}
		logger.InfoContext(ctx, "downloading pinned commit", "release", pin.Release, "commit", pin.Commit)
		return pin, downloadApple(ctx, logger, client, o.TarballURL, o.Dir, pin.Commit)
	}

	commit, err := resolveCommit(ctx, logger, client, o.APIURL, appleRepo, appleBranch)
	if err != nil {
		return profilevalidation.AppleVersion{}, err
	}
	// Every commit on Apple's release branch is titled with the release name, e.g. "Release-v27.0".
	version := profilevalidation.AppleVersion{Release: commit.Subject, Commit: commit.SHA}
	if version.Release == "" {
		return profilevalidation.AppleVersion{}, fmt.Errorf("commit %s on %s@%s has an empty message; can't name the release", commit.SHA, appleRepo, appleBranch)
	}
	switch {
	case o.Pinned == nil:
		logger.InfoContext(ctx, "no Apple pin; using latest", "release", version.Release, "commit", version.Commit)
	case o.Pinned.Commit == version.Commit:
		logger.InfoContext(ctx, "latest Apple commit matches pin; regenerating anyway", "release", version.Release, "commit", version.Commit)
	default:
		logger.InfoContext(ctx, "moving Apple pin to latest", "from_release", o.Pinned.Release, "from_commit", o.Pinned.Commit,
			"to_release", version.Release, "to_commit", version.Commit)
	}
	return version, downloadApple(ctx, logger, client, o.TarballURL, o.Dir, version.Commit)
}

// downloadApple replaces dir with Apple's sources at sha and marks which commit it holds.
func downloadApple(ctx context.Context, logger *slog.Logger, client *http.Client, tarballURL, dir, sha string) error {
	logger.DebugContext(ctx, "clearing previous download", "dir", dir)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := downloadAppleTarball(ctx, logger, client, tarballURL, sha, dir); err != nil {
		return err
	}
	// Written last, so an interrupted download never looks complete.
	return os.WriteFile(filepath.Join(dir, commitMarker), []byte(sha+"\n"), 0o644)
}

type windowsRunOptions struct {
	PageURL     string // the Learn page that links the current DDF drop
	Dir         string // where the DDF XML files are extracted
	Pinned      *profilevalidation.WindowsVersion
	Update      bool // move to the newest drop
	FetchPinned bool // without Update, download the pinned drop if dir doesn't hold it
}

// runWindows returns the DDF drop to generate and leaves its XML files in o.Dir. Like runApple,
// without o.Update it uses the pin and only downloads it when o.FetchPinned is set; a pinned zip
// whose sha256 changed fails.
func runWindows(ctx context.Context, logger *slog.Logger, client *http.Client, o windowsRunOptions) (profilevalidation.WindowsVersion, error) {
	if !o.Update {
		if o.Pinned == nil {
			return profilevalidation.WindowsVersion{}, errors.New("no DDF drop pinned in the output; run with -update to fetch the latest")
		}
		pin := *o.Pinned
		downloaded, err := os.ReadFile(filepath.Join(o.Dir, sha256Marker))
		got := strings.TrimSpace(string(downloaded))
		if err == nil && got == pin.SHA256 {
			logger.InfoContext(ctx, "using existing download for pinned drop", "dir", o.Dir, "release", pin.Release, "sha256", pin.SHA256)
			return pin, nil
		}
		if !o.FetchPinned {
			if err != nil {
				return profilevalidation.WindowsVersion{}, fmt.Errorf("no DDF download in %s for pinned drop %s; run with -fetch-pinned to download it, or -update to move to the latest", o.Dir, pin.Release)
			}
			return profilevalidation.WindowsVersion{}, fmt.Errorf("%s holds DDF zip %s but the output pins %s (%s); run with -fetch-pinned to download it, or -update to move to the latest", o.Dir, got, pin.SHA256, pin.Release)
		}
		logger.InfoContext(ctx, "downloading pinned drop", "release", pin.Release, "url", pin.URL)
		_, err = downloadWindows(ctx, logger, client, pin.URL, pin.SHA256, o.Dir)
		return pin, err
	}

	version, err := resolveDDF(ctx, logger, client, o.PageURL)
	if err != nil {
		return profilevalidation.WindowsVersion{}, err
	}
	if version.SHA256, err = downloadWindows(ctx, logger, client, version.URL, "", o.Dir); err != nil {
		return profilevalidation.WindowsVersion{}, err
	}
	switch {
	case o.Pinned == nil:
		logger.InfoContext(ctx, "no DDF pin; using latest", "release", version.Release, "sha256", version.SHA256)
	case *o.Pinned == version:
		logger.InfoContext(ctx, "latest DDF drop matches pin; regenerating anyway", "release", version.Release, "sha256", version.SHA256)
	case o.Pinned.URL == version.URL:
		// Same link, different bytes: Microsoft republished the drop in place.
		logger.WarnContext(ctx, "DDF zip changed at the pinned URL", "release", version.Release,
			"from_sha256", o.Pinned.SHA256, "to_sha256", version.SHA256)
	default:
		logger.InfoContext(ctx, "moving DDF pin to latest", "from_release", o.Pinned.Release, "to_release", version.Release,
			"to_sha256", version.SHA256)
	}
	return version, nil
}

// downloadWindows replaces dir with the DDF drop at url and marks which zip it holds.
func downloadWindows(ctx context.Context, logger *slog.Logger, client *http.Client, url, wantSHA256, dir string) (string, error) {
	logger.DebugContext(ctx, "clearing previous download", "dir", dir)
	if err := os.RemoveAll(dir); err != nil {
		return "", err
	}
	sum, err := downloadDDF(ctx, logger, client, url, wantSHA256, dir)
	if err != nil {
		return "", err
	}
	// Written last, so an interrupted download never looks complete.
	return sum, os.WriteFile(filepath.Join(dir, sha256Marker), []byte(sum+"\n"), 0o644)
}
