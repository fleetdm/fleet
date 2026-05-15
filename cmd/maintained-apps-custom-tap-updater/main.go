// maintained-apps-custom-tap-updater bumps the casks under
// ee/maintained-apps/inputs/homebrew/custom-tap/ when their upstream
// versions change. It is intended to run on Linux without Homebrew, so it
// updates only the fields that change on a version bump (version, sha256,
// url, the pkg artifact filenames, and the ruby_source_checksum). Any
// structural change to a cask (new uninstall directives, depends_on, etc.)
// still requires a human edit followed by regenerate.sh on macOS.
//
// Each vendor exposes version information differently, so the per-app
// checkers below all return the same {version, url} shape but get there
// through different routes — GitHub Releases for fleet-desktop and xcreds,
// best-effort page scraping for druva-insync and zoom-rooms (both vendors'
// upstream "latest" surfaces are unreliable and the cask DSLs explicitly
// skip livecheck).
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

const customTapRoot = "ee/maintained-apps/inputs/homebrew/custom-tap"

type upstream struct {
	// version is the canonical cask DSL form (e.g. "1.2.0" or "5.9,9148").
	version string
	// url is the resolved download URL that should be fetched to compute sha256.
	url string
}

type appConfig struct {
	cask string
	// check returns the latest upstream version, or (nil, nil) when no
	// reliable signal is available — callers treat that as "leave alone".
	check func(ctx context.Context, client *http.Client, logger *slog.Logger) (*upstream, error)
	// pkgFilenames renders the artifacts[].pkg list for the api JSON given
	// a version. Order must match the existing list in api/<cask>.json.
	pkgFilenames func(version string) []string
}

var apps = []appConfig{
	{
		cask:         "fleet-desktop",
		check:        checkFleetDesktop,
		pkgFilenames: func(v string) []string { return []string{fmt.Sprintf("fleet_desktop-v%s.pkg", v)} },
	},
	{
		cask:  "xcreds",
		check: checkXCreds,
		pkgFilenames: func(v string) []string {
			ver, build, _ := strings.Cut(v, ",")
			return []string{fmt.Sprintf("XCreds_Build-%s_Version-%s.pkg", build, ver)}
		},
	},
	{
		cask:         "druva-insync",
		check:        checkDruvaInSync,
		pkgFilenames: func(v string) []string { return []string{"Install inSync.pkg"} },
	},
	{
		cask:         "zoom-rooms",
		check:        checkZoomRooms,
		pkgFilenames: func(v string) []string { return []string{"ZoomRooms.pkg"} },
	},
}

// caskJSON mirrors `brew info --cask --json=v2` after regenerate.sh's
// strips. Field order matches brew's output so a regenerated file diffs
// cleanly against one produced on macOS. Fields we don't need to mutate
// are kept as json.RawMessage to round-trip their exact values (including
// the various null/false/[] defaults and the multi-line caveats string).
type caskJSON struct {
	Token                         string            `json:"token"`
	OldTokens                     json.RawMessage   `json:"old_tokens"`
	Name                          json.RawMessage   `json:"name"`
	Desc                          json.RawMessage   `json:"desc"`
	Homepage                      json.RawMessage   `json:"homepage"`
	URL                           string            `json:"url"`
	URLSpecs                      json.RawMessage   `json:"url_specs"`
	Version                       string            `json:"version"`
	Autobump                      json.RawMessage   `json:"autobump"`
	NoAutobumpMessage             json.RawMessage   `json:"no_autobump_message"`
	SkipLivecheck                 json.RawMessage   `json:"skip_livecheck"`
	BundleVersion                 json.RawMessage   `json:"bundle_version"`
	BundleShortVersion            json.RawMessage   `json:"bundle_short_version"`
	SHA256                        string            `json:"sha256"`
	Artifacts                     []json.RawMessage `json:"artifacts"`
	Caveats                       json.RawMessage   `json:"caveats"`
	CaveatsRosetta                json.RawMessage   `json:"caveats_rosetta"`
	DependsOn                     json.RawMessage   `json:"depends_on"`
	ConflictsWith                 json.RawMessage   `json:"conflicts_with"`
	Container                     json.RawMessage   `json:"container"`
	Rename                        json.RawMessage   `json:"rename"`
	AutoUpdates                   json.RawMessage   `json:"auto_updates"`
	Deprecated                    json.RawMessage   `json:"deprecated"`
	DeprecationDate               json.RawMessage   `json:"deprecation_date"`
	DeprecationReason             json.RawMessage   `json:"deprecation_reason"`
	DeprecationReplacementFormula json.RawMessage   `json:"deprecation_replacement_formula"`
	DeprecationReplacementCask    json.RawMessage   `json:"deprecation_replacement_cask"`
	DeprecateArgs                 json.RawMessage   `json:"deprecate_args"`
	Disabled                      json.RawMessage   `json:"disabled"`
	DisableDate                   json.RawMessage   `json:"disable_date"`
	DisableReason                 json.RawMessage   `json:"disable_reason"`
	DisableReplacementFormula     json.RawMessage   `json:"disable_replacement_formula"`
	DisableReplacementCask        json.RawMessage   `json:"disable_replacement_cask"`
	DisableArgs                   json.RawMessage   `json:"disable_args"`
	Languages                     json.RawMessage   `json:"languages"`
	RubySourcePath                string            `json:"ruby_source_path"`
	RubySourceChecksum            struct {
		SHA256 string `json:"sha256"`
	} `json:"ruby_source_checksum"`
}

func main() {
	var (
		debug bool
		only  string
	)
	flag.BoolVar(&debug, "debug", false, "verbose logging")
	flag.StringVar(&only, "only", "", "comma-separated tokens to process (default: all)")
	flag.Parse()

	ctx := context.Background()
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	cwd, err := os.Getwd()
	if err != nil {
		logger.ErrorContext(ctx, "getwd failed", "err", err)
		os.Exit(1)
	}
	if _, err := os.Stat(filepath.Join(cwd, customTapRoot)); err != nil {
		logger.ErrorContext(ctx, "expected to run from fleet repo root", "cwd", cwd, "err", err)
		os.Exit(1)
	}

	client := fleethttp.NewClient(fleethttp.WithTimeout(5 * time.Minute))

	filter := map[string]bool{}
	if only != "" {
		for t := range strings.SplitSeq(only, ",") {
			filter[strings.TrimSpace(t)] = true
		}
	}

	var failed []string
	for _, app := range apps {
		if len(filter) > 0 && !filter[app.cask] {
			continue
		}
		l := logger.With("app", app.cask)
		if err := updateApp(ctx, l, client, app); err != nil {
			l.ErrorContext(ctx, "update failed", "err", err)
			failed = append(failed, app.cask)
			continue
		}
	}
	if len(failed) > 0 {
		// Failures on individual apps shouldn't sink the workflow — any
		// apps that succeeded should still flow through to the PR.
		logger.WarnContext(ctx, "some apps failed; see prior errors", "apps", failed)
	}
}

func updateApp(ctx context.Context, logger *slog.Logger, client *http.Client, app appConfig) error {
	rbPath := filepath.Join(customTapRoot, "Casks", app.cask+".rb")
	jsonPath := filepath.Join(customTapRoot, "api", app.cask+".json")

	jsonBytes, err := os.ReadFile(jsonPath)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading api json")
	}
	var meta caskJSON
	if err := json.Unmarshal(jsonBytes, &meta); err != nil {
		return ctxerr.Wrap(ctx, err, "parsing api json")
	}

	up, err := app.check(ctx, client, logger)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upstream check")
	}
	if up == nil {
		logger.InfoContext(ctx, "no upstream version detected; leaving alone")
		return nil
	}
	if up.version == meta.Version {
		logger.InfoContext(ctx, "already up to date", "version", up.version)
		return nil
	}

	logger.InfoContext(ctx, "new version available", "from", meta.Version, "to", up.version, "url", up.url)

	// Stream the asset through the hasher rather than buffering — installer
	// packages can run to hundreds of MB.
	newSHA, err := downloadAndHash(ctx, client, up.url)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "downloading new asset")
	}

	rbBytes, err := os.ReadFile(rbPath)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading rb")
	}
	rbUpdated, err := updateRB(rbBytes, meta.Version, up.version, meta.SHA256, newSHA)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating rb")
	}

	rbSHASum := sha256.Sum256(rbUpdated)
	rbSHA := hex.EncodeToString(rbSHASum[:])

	meta.Version = up.version
	meta.URL = up.url
	meta.SHA256 = newSHA
	meta.RubySourceChecksum.SHA256 = rbSHA
	if err := updatePkgArtifacts(&meta, app.pkgFilenames(up.version)); err != nil {
		return ctxerr.Wrap(ctx, err, "updating pkg artifacts")
	}

	newJSON, err := encodeCask(&meta)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "encoding api json")
	}

	if err := os.WriteFile(rbPath, rbUpdated, 0o644); err != nil {
		return ctxerr.Wrap(ctx, err, "writing rb")
	}
	if err := os.WriteFile(jsonPath, newJSON, 0o644); err != nil {
		return ctxerr.Wrap(ctx, err, "writing api json")
	}
	logger.InfoContext(ctx, "wrote update", "version", up.version, "sha256", newSHA)
	return nil
}

// updateRB rewrites the version "..." and sha256 "..." lines in a Cask DSL
// source by anchoring on their previous values. We deliberately avoid
// parsing Ruby — the cask DSL files have exactly one top-level version
// and sha256 each.
func updateRB(content []byte, oldVersion, newVersion, oldSHA, newSHA string) ([]byte, error) {
	versionOld := fmt.Sprintf(`version "%s"`, oldVersion)
	versionNew := fmt.Sprintf(`version "%s"`, newVersion)
	shaOld := fmt.Sprintf(`sha256 "%s"`, oldSHA)
	shaNew := fmt.Sprintf(`sha256 "%s"`, newSHA)
	if !bytes.Contains(content, []byte(versionOld)) {
		return nil, fmt.Errorf("anchor %q not found in .rb", versionOld)
	}
	if !bytes.Contains(content, []byte(shaOld)) {
		return nil, fmt.Errorf("anchor %q not found in .rb", shaOld)
	}
	out := bytes.ReplaceAll(content, []byte(versionOld), []byte(versionNew))
	out = bytes.ReplaceAll(out, []byte(shaOld), []byte(shaNew))
	return out, nil
}

// updatePkgArtifacts locates the `{"pkg": [...]}` entry in artifacts and
// rewrites its list. The other artifact entries (uninstall, zap) are left
// untouched as raw JSON.
func updatePkgArtifacts(meta *caskJSON, pkgFilenames []string) error {
	for i, raw := range meta.Artifacts {
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(raw, &probe); err != nil {
			return fmt.Errorf("probing artifact %d: %w", i, err)
		}
		if _, ok := probe["pkg"]; !ok {
			continue
		}
		encoded, err := json.Marshal(map[string][]string{"pkg": pkgFilenames})
		if err != nil {
			return err
		}
		meta.Artifacts[i] = encoded
		return nil
	}
	return errors.New("no pkg entry in artifacts to update")
}

// encodeCask marshals the cask metadata back to JSON in the same style
// brew uses: 2-space indent, no HTML escaping, trailing newline. We
// marshal compactly first and then re-indent so the pre-formatted
// json.RawMessage fields are normalized into the indented layout.
// json.Encoder appends the trailing newline; json.Indent preserves it.
func encodeCask(meta *caskJSON) ([]byte, error) {
	var compact bytes.Buffer
	enc := json.NewEncoder(&compact)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(meta); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, compact.Bytes(), "", "  "); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func downloadAndHash(ctx context.Context, client *http.Client, downloadURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: status %d", downloadURL, resp.StatusCode)
	}
	h := sha256.New()
	if _, err := io.Copy(h, resp.Body); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// --- per-app upstream checkers ---

// fleet-desktop ships GitHub releases tagged "vX.Y.Z" with a single
// pkg asset named "fleet_desktop-vX.Y.Z.pkg".
func checkFleetDesktop(ctx context.Context, client *http.Client, _ *slog.Logger) (*upstream, error) {
	tag, err := githubLatestTag(ctx, client, "allenhouchins", "fleet-desktop")
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(tag, "v") {
		return nil, fmt.Errorf("unexpected tag shape %q", tag)
	}
	v := strings.TrimPrefix(tag, "v")
	return &upstream{
		version: v,
		url:     fmt.Sprintf("https://github.com/allenhouchins/fleet-desktop/releases/download/v%s/fleet_desktop-v%s.pkg", v, v),
	}, nil
}

// xcreds ships GitHub releases tagged "tag-X.Y(BUILD)" — the cask DSL
// stores both halves as a CSV ("X.Y,BUILD") so version.csv.first /
// version.csv.second resolve to the right things in the URL template.
func checkXCreds(ctx context.Context, client *http.Client, _ *slog.Logger) (*upstream, error) {
	tag, err := githubLatestTag(ctx, client, "twocanoes", "xcreds")
	if err != nil {
		return nil, err
	}
	m := regexp.MustCompile(`^tag-(.+)\((\d+)\)$`).FindStringSubmatch(tag)
	if m == nil {
		return nil, fmt.Errorf("unexpected tag shape %q", tag)
	}
	ver, build := m[1], m[2]
	return &upstream{
		version: fmt.Sprintf("%s,%s", ver, build),
		url:     fmt.Sprintf("https://github.com/twocanoes/xcreds/releases/download/tag-%s(%s)/XCreds_Build-%s_Version-%s.pkg", ver, build, build, ver),
	}, nil
}

// druva-insync: Druva doesn't expose a parseable version feed. Best
// effort here is to scrape their public release-notes page for the
// newest "inSync Mac Client X.Y.Z (Build NNNNNN)" header. If the page
// layout changes the regex fails and we return (nil, nil) so the
// workflow leaves the file alone instead of erroring.
func checkDruvaInSync(ctx context.Context, client *http.Client, logger *slog.Logger) (*upstream, error) {
	const docURL = "https://docs.druva.com/Knowledge_Base/inSync/Release_Notes/Release_Notes_inSync_Mac_Client"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, docURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.WarnContext(ctx, "druva docs unreachable", "err", err)
		return nil, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.WarnContext(ctx, "druva docs returned non-200", "status", resp.StatusCode)
		return nil, nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`inSync\s+Mac\s+Client\s+(\d+\.\d+(?:\.\d+)?)\s*\(\s*Build\s+(\d+)\s*\)`)
	m := re.FindSubmatch(body)
	if m == nil {
		logger.WarnContext(ctx, "no version pattern found on druva docs page")
		return nil, nil
	}
	ver, build := string(m[1]), string(m[2])
	return &upstream{
		version: fmt.Sprintf("%s,%s", ver, build),
		url:     fmt.Sprintf("https://downloads.druva.com/downloads/inSync/MAC/%s/inSync-%s-r%s.dmg", ver, ver, build),
	}, nil
}

// zoom-rooms: Zoom doesn't publish a parseable version feed for the
// Rooms client. Best effort: follow the "latest" URL's redirect chain
// to a versioned CDN URL and pluck the version out. Returns (nil, nil)
// if the redirect doesn't land where we expect.
func checkZoomRooms(ctx context.Context, client *http.Client, logger *slog.Logger) (*upstream, error) {
	const latestURL = "https://www.zoom.us/client/latest/ZoomRooms.pkg"
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, latestURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.WarnContext(ctx, "zoom latest unreachable", "err", err)
		return nil, nil
	}
	defer resp.Body.Close()
	finalURL := resp.Request.URL.String()
	m := regexp.MustCompile(`https://cdn\.zoom\.us/prod/([\d.]+)/ZoomRooms\.pkg`).FindStringSubmatch(finalURL)
	if m == nil {
		logger.WarnContext(ctx, "zoom redirect target did not match expected pattern", "final", finalURL)
		return nil, nil
	}
	return &upstream{version: m[1], url: finalURL}, nil
}

// githubLatestTag returns the tag_name of the latest non-prerelease
// release for the given repo. Uses GITHUB_TOKEN if set to dodge the
// unauthenticated rate limit.
func githubLatestTag(ctx context.Context, client *http.Client, owner, repo string) (string, error) {
	u := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: status %d", u, resp.StatusCode)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.TagName == "" {
		return "", errors.New("empty tag_name in latest release")
	}
	return body.TagName, nil
}
