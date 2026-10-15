package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/govulndb"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Fetch failures are one of the cases under test; do not sit through the backoff.
	retryDelay = 0
	os.Exit(m.Run())
}

const (
	stdlibReport = `{
		"id": "GO-2024-2963",
		"aliases": ["CVE-2024-24791"],
		"affected": [{
			"package": {"name": "stdlib", "ecosystem": "Go"},
			"ranges": [{"type": "SEMVER", "events": [
				{"introduced": "0"}, {"fixed": "1.21.12"},
				{"introduced": "1.22.0-0"}, {"fixed": "1.22.5"}
			]}]
		}]
	}`

	moduleReport = `{
		"id": "GO-2024-1234",
		"aliases": ["CVE-2024-1234"],
		"affected": [{
			"package": {"name": "github.com/example/tool", "ecosystem": "Go"},
			"ranges": [{"type": "SEMVER", "events": [{"introduced": "0"}, {"fixed": "1.49.0"}]}]
		}]
	}`

	ghsaOnlyReport = `{
		"id": "GO-2024-4000",
		"aliases": ["GHSA-xxxx-yyyy-zzzz"],
		"affected": [{
			"package": {"name": "github.com/example/ghsa", "ecosystem": "Go"},
			"ranges": [{"type": "SEMVER", "events": [{"introduced": "0"}]}]
		}]
	}`

	misPairedReport = `{
		"id": "GO-2024-7777",
		"aliases": ["CVE-2024-7777"],
		"affected": [{
			"package": {"name": "github.com/example/broken", "ecosystem": "Go"},
			"ranges": [{"type": "SEMVER", "events": [{"introduced": "0"}, {"introduced": "2.0.0"}]}]
		}]
	}`

	nonSemverReport = `{
		"id": "GO-2024-8888",
		"aliases": ["CVE-2024-8888"],
		"affected": [{
			"package": {"name": "github.com/example/ecosystem", "ecosystem": "Go"},
			"ranges": [{"type": "ECOSYSTEM", "events": [{"introduced": "0"}, {"fixed": "1.0.0"}]}]
		}]
	}`
)

// database describes the vulndb.zip a test serves.
type database struct {
	// reports are OSV documents under ID/ that the module index also names.
	reports []string
	// unindexed are OSV documents under ID/ that the module index does not name.
	unindexed []string
	// indexOnly are report IDs the module index names but the archive does not carry, i.e. a
	// half-mirrored database.
	indexOnly []string
}

func healthy() database {
	return database{reports: []string{stdlibReport, moduleReport}}
}

// archive builds the vulndb.zip vuln.go.dev serves: index/db.json, index/modules.json and one
// document per report.
func archive(t *testing.T, db database) []byte {
	t.Helper()

	type indexVuln struct {
		ID string `json:"id"`
	}
	type indexEntry struct {
		Path  string      `json:"path"`
		Vulns []indexVuln `json:"vulns"`
	}

	var (
		buf   bytes.Buffer
		zw    = zip.NewWriter(&buf)
		index []indexEntry
	)

	add := func(name string, content []byte) {
		t.Helper()
		w, err := zw.Create(name)
		require.NoError(t, err, "creating %s in the archive", name)
		_, err = w.Write(content)
		require.NoError(t, err, "writing %s into the archive", name)
	}
	reportID := func(raw string) string {
		t.Helper()
		var doc struct {
			ID string `json:"id"`
		}
		require.NoError(t, json.Unmarshal([]byte(raw), &doc), "test report is not valid JSON")
		return doc.ID
	}
	// The publisher only reads report IDs out of the index; the paths are filler.
	indexed := func(id string) {
		index = append(index, indexEntry{Path: "example.com/" + id, Vulns: []indexVuln{{ID: id}}})
	}

	for _, raw := range db.reports {
		id := reportID(raw)
		add(reportDir+id+".json", []byte(raw))
		indexed(id)
	}
	for _, raw := range db.unindexed {
		add(reportDir+reportID(raw)+".json", []byte(raw))
	}
	for _, id := range db.indexOnly {
		indexed(id)
	}

	encoded, err := json.Marshal(index)
	require.NoError(t, err, "encoding the module index")
	add(indexPath, encoded)
	add(dbIndexPath, []byte(`{"modified":"2026-09-15T18:39:25Z"}`))

	require.NoError(t, zw.Close(), "closing the archive")

	return buf.Bytes()
}

// serveDatabase stands in for vuln.go.dev, serving the archive at the path the publisher asks
// for. A nil body makes every request fail, standing in for a bad upstream day.
func serveDatabase(t *testing.T, body []byte) string {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if body == nil {
			http.Error(w, "upstream is having a bad day", http.StatusInternalServerError)
			return
		}
		if r.URL.Path != dbArchive {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	return srv.URL
}

type result struct {
	exit      int
	outputs   map[string]string
	outputDir string
	logs      string
}

// artifacts lists the artifacts the run left in its output directory.
func (r result) artifacts(t *testing.T) []string {
	t.Helper()

	entries, err := filepath.Glob(filepath.Join(r.outputDir, govulndb.FilePrefix+"*"))
	require.NoError(t, err, "listing %s", r.outputDir)

	return entries
}

// mirror runs the command the way main does, without the process exiting. It runs as a first
// run unless withPrevious is passed.
func mirror(t *testing.T, url string, options ...func(*config)) result {
	t.Helper()

	cfg := config{
		baseURL:        url,
		outputDir:      t.TempDir(),
		firstRun:       true,
		maxDropPercent: defaultMaxDropPercent,
		now:            time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC),
	}
	for _, option := range options {
		option(&cfg)
	}

	var outputs, logs bytes.Buffer
	exit := execute(t.Context(), cfg, log.New(&logs, "", 0), &outputs)
	t.Logf("run log:\n%s", logs.String())

	parsed := make(map[string]string)
	for line := range strings.SplitSeq(strings.TrimSpace(outputs.String()), "\n") {
		if key, value, ok := strings.Cut(line, "="); ok {
			parsed[key] = value
		}
	}

	return result{exit: exit, outputs: parsed, outputDir: cfg.outputDir, logs: logs.String()}
}

func withPrevious(path string) func(*config) {
	return func(cfg *config) {
		cfg.previous = path
		cfg.firstRun = false
	}
}

func withOutputDir(dir string) func(*config) {
	return func(cfg *config) { cfg.outputDir = dir }
}

// assertSkipped is the behaviour every suspect-database case has to share: nothing published, an
// alert that says which check tripped, and a zero exit so a bad upstream day never pages anyone.
func assertSkipped(t *testing.T, got result, wantReason string) {
	t.Helper()

	require.Equal(t, 0, got.exit, "a suspect database must not fail the workflow")
	require.Empty(t, got.artifacts(t), "nothing may be published")
	require.Equal(t, "true", got.outputs["skipped"])
	require.Contains(t, got.outputs["reason"], wantReason)
}

// assertPublished is the healthy path: a zero exit, skipped=false and exactly one artifact.
func assertPublished(t *testing.T, got result) string {
	t.Helper()

	require.Equal(t, 0, got.exit)
	require.Equal(t, "false", got.outputs["skipped"])
	files := got.artifacts(t)
	require.Len(t, files, 1, "want exactly one artifact")

	return files[0]
}

func TestParseFlags(t *testing.T) {
	for _, tc := range []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "first run", args: []string{"--output", "out", "--first-run"}},
		{name: "with a previous artifact", args: []string{"--output", "out", "--previous", "prev.json.gz"}},
		{name: "missing output", args: []string{"--first-run"}, wantErr: "--output is required"},
		// A workflow that stopped passing --previous after a skipped day must not publish
		// blind; it has to say it is the first run in so many words.
		{name: "missing previous", args: []string{"--output", "out"}, wantErr: "--previous is required"},
		{name: "previous and first run", args: []string{"--output", "out", "--previous", "p", "--first-run"}, wantErr: "mutually exclusive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := parseFlags("govulndb-mirror", tc.args, io.Discard)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "out", cfg.outputDir)
			require.Equal(t, defaultBaseURL, cfg.baseURL)
			require.False(t, cfg.now.IsZero())
		})
	}

	t.Run("help", func(t *testing.T) {
		var usage bytes.Buffer
		_, err := parseFlags("govulndb-mirror", []string{"-h"}, &usage)
		require.ErrorIs(t, err, flag.ErrHelp)
		require.Contains(t, usage.String(), "-first-run")
	})
}

func TestExecutePublishesAHealthyDatabase(t *testing.T) {
	db := healthy()
	db.reports = append(db.reports, ghsaOnlyReport)

	got := mirror(t, serveDatabase(t, archive(t, db)))
	file := assertPublished(t, got)

	// Fleet takes the lexically last matching asset on the latest release, so the name carries
	// the snapshot date and nothing else.
	require.Equal(t, "govulndb-2026-09-09.json.gz", filepath.Base(file))
	require.Equal(t, file, got.outputs["artifact"])

	artifact := readArtifact(t, file)
	require.Equal(t, govulndb.SchemaVersion, artifact.SchemaVersion)
	require.True(t, artifact.Generated.Equal(time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)), "generated = %s", artifact.Generated)
	require.NotContains(t, artifact.Modules, "github.com/example/ghsa", "the GHSA-only report was published")
	require.Len(t, artifact.Modules, 2, "want stdlib and github.com/example/tool")
	require.Contains(t, artifact.Modules, govulndb.StdlibModule)
	require.Contains(t, artifact.Modules, "github.com/example/tool")
}

func TestExecuteAcceptsATrailingSlashOnTheURL(t *testing.T) {
	assertPublished(t, mirror(t, serveDatabase(t, archive(t, healthy()))+"/"))
}

// The index check guards against a truncated archive. An archive that carries more than the
// index names is the other way round and nothing to hold a publish over.
func TestExecuteKeepsReportsTheIndexDoesNotName(t *testing.T) {
	body := archive(t, database{reports: []string{stdlibReport}, unindexed: []string{moduleReport}})

	got := mirror(t, serveDatabase(t, body))
	artifact := readArtifact(t, assertPublished(t, got))

	require.Contains(t, artifact.Modules, "github.com/example/tool")
}

// A rerun on the same day replaces the artifact rather than leaving two behind, and the write
// leaves no temporary file in the directory the release step uploads.
func TestExecuteOverwritesASameDayArtifact(t *testing.T) {
	dir := t.TempDir()
	url := serveDatabase(t, archive(t, healthy()))

	assertPublished(t, mirror(t, url, withOutputDir(dir)))
	assertPublished(t, mirror(t, url, withOutputDir(dir)))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "want only the artifact in the output directory")
	require.Equal(t, "govulndb-2026-09-09.json.gz", entries[0].Name())
}

func TestExecuteSkipsWhenUpstreamFails(t *testing.T) {
	assertSkipped(t, mirror(t, serveDatabase(t, nil)), "failed after 3 attempts")
}

func TestExecuteSkipsAnIncompleteDownload(t *testing.T) {
	db := healthy()
	db.indexOnly = []string{"GO-2024-5555", "GO-2024-5556"}

	assertSkipped(t, mirror(t, serveDatabase(t, archive(t, db))), "incomplete download")
}

func TestExecuteSkipsAnEmptyArtifact(t *testing.T) {
	// Every report is dropped, leaving nothing to publish.
	body := archive(t, database{reports: []string{ghsaOnlyReport}})

	assertSkipped(t, mirror(t, serveDatabase(t, body)), "no modules at all")
}

func TestExecuteSkipsAnArtifactWithoutStdlib(t *testing.T) {
	body := archive(t, database{reports: []string{moduleReport}})

	assertSkipped(t, mirror(t, serveDatabase(t, body)), govulndb.StdlibModule)
}

func TestExecuteSkipsMisPairedEvents(t *testing.T) {
	body := archive(t, database{reports: []string{stdlibReport, misPairedReport}})

	assertSkipped(t, mirror(t, serveDatabase(t, body)), "GO-2024-7777")
}

func TestExecuteSkipsANonSemverRange(t *testing.T) {
	body := archive(t, database{reports: []string{stdlibReport, nonSemverReport}})

	// The reason is sanitized for the alert payload, so the quotes come back as apostrophes.
	assertSkipped(t, mirror(t, serveDatabase(t, body)), "range type 'ECOSYSTEM'")
}

func TestExecuteSkipsAShrunkenDatabase(t *testing.T) {
	// The last published artifact had far more modules than today's snapshot carries.
	previous := writePrevious(t, artifactOf(map[string]int{
		govulndb.StdlibModule:       1,
		"github.com/example/tool":   1,
		"github.com/example/other":  1,
		"github.com/example/third":  1,
		"github.com/example/fourth": 1,
	}))

	got := mirror(t, serveDatabase(t, archive(t, healthy())), withPrevious(previous))

	assertSkipped(t, got, "module count dropped")
	require.Contains(t, got.outputs["reason"], "5 to 2", "want the reason to report the counts")
}

func TestExecutePublishesWhenTheDatabaseGrew(t *testing.T) {
	previous := writePrevious(t, artifactOf(map[string]int{govulndb.StdlibModule: 1}))

	assertPublished(t, mirror(t, serveDatabase(t, archive(t, healthy())), withPrevious(previous)))
}

// Publishing without the drop check would be publishing blind, so an unreadable baseline skips
// and alerts rather than being shrugged off.
func TestExecuteSkipsAnUnreadablePreviousArtifact(t *testing.T) {
	corrupt := filepath.Join(t.TempDir(), "govulndb-2026-09-08.json.gz")
	require.NoError(t, os.WriteFile(corrupt, []byte("not gzip"), 0o644))

	got := mirror(t, serveDatabase(t, archive(t, healthy())), withPrevious(corrupt))

	assertSkipped(t, got, "previously published artifact")
}

// A local failure is not a bad upstream day: it fails the workflow, and still alerts.
func TestExecuteFailsOnAnUnwritableOutputDirectory(t *testing.T) {
	blocked := filepath.Join(t.TempDir(), "a-file")
	require.NoError(t, os.WriteFile(blocked, nil, 0o644))

	got := mirror(t, serveDatabase(t, archive(t, healthy())), withOutputDir(filepath.Join(blocked, "out")))

	require.Equal(t, 1, got.exit)
	require.Equal(t, "true", got.outputs["skipped"])
	require.NotEmpty(t, got.outputs["reason"])
}

func TestStepOutputs(t *testing.T) {
	t.Run("outside a workflow", func(t *testing.T) {
		t.Setenv("GITHUB_OUTPUT", "")

		outputs, closeOutputs, err := stepOutputs()
		require.NoError(t, err)

		require.Equal(t, io.Discard, outputs)
		require.NoError(t, closeOutputs())
	})

	t.Run("appends to GITHUB_OUTPUT", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "output")
		require.NoError(t, os.WriteFile(path, []byte("earlier=step\n"), 0o644))
		t.Setenv("GITHUB_OUTPUT", path)

		outputs, closeOutputs, err := stepOutputs()
		require.NoError(t, err)
		emit(outputs, "skipped", "false")
		require.NoError(t, closeOutputs())

		written, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "earlier=step\nskipped=false\n", string(written))
	})
}

func writePrevious(t *testing.T, artifact *govulndb.Artifact) string {
	t.Helper()

	path, err := writeArtifact(artifact, t.TempDir(), time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err, "writing the previous artifact")

	return path
}

func readArtifact(t *testing.T, path string) *govulndb.Artifact {
	t.Helper()

	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()

	gz, err := gzip.NewReader(f)
	require.NoError(t, err, "gunzipping %s", path)
	defer gz.Close()

	var artifact govulndb.Artifact
	require.NoError(t, json.NewDecoder(gz).Decode(&artifact), "decoding %s", path)
	_, err = io.ReadAll(gz)
	require.NoError(t, err, "reading the rest of %s", path)

	return &artifact
}

// The reason is interpolated into the alert's JSON payload and part of it is upstream text, so
// it has to come out as one quote-free, valid UTF-8 line.
func TestOneLine(t *testing.T) {
	require.Equal(t, "module 'github.com/x': ab dropped", oneLine("module \"github.com/x\":\n  a\\b dropped"))

	long := oneLine(strings.Repeat("x", maxReasonLen+50))
	require.Len(t, long, maxReasonLen+3)
	require.True(t, strings.HasSuffix(long, "..."))

	// Truncating in the middle of a multi-byte character would hand the alert invalid UTF-8.
	multibyte := oneLine(strings.Repeat("é", maxReasonLen))
	require.True(t, utf8.ValidString(multibyte), "truncated reason is not valid UTF-8: %q", multibyte)
	require.LessOrEqual(t, len(multibyte), maxReasonLen+3)
}
