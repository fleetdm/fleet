// Command cpe-accuracy runs Fleet's CPE generation pipeline against a set of
// golden test cases and reports accuracy metrics.
//
// It reads JSON "accuracy suites" from a testdata directory, runs each case's
// software input through nvd.TranslateSoftwareToCPE using the real CPE SQLite
// database and cpe_translations.json, and compares the generated CPE against
// the expected value.
//
// Preferred workflow (reuses the canonical Fleet downloader):
//
//	fleetctl vulnerability-data-stream --dir /tmp/vulnsdb
//	go run ./tools/nvd/cpe-accuracy --vuln-path /tmp/vulnsdb
//
// Other options:
//
//	# Let the tool download the CPE data itself into a temp dir (slow first run).
//	go run ./tools/nvd/cpe-accuracy
//
//	# Reuse the env var the existing Go tests use for caching.
//	NVD_TEST_VULNDB_DIR=/tmp/vulnsdb go run ./tools/nvd/cpe-accuracy
//
//	# Verbose / machine-readable output.
//	go run ./tools/nvd/cpe-accuracy --vuln-path /tmp/vulnsdb --verbose
//	go run ./tools/nvd/cpe-accuracy --vuln-path /tmp/vulnsdb --json
//
// Exit code is 0 when every case passes and 1 when any case fails. Pass-true-negative
// cases (expected empty, generated empty) are counted as passes.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
)

// -----------------------------------------------------------------------------
// Golden file schema
// -----------------------------------------------------------------------------

type accuracySuite struct {
	Metadata  accuracyMetadata `json:"metadata"`
	TestCases []accuracyCase   `json:"test_cases"`
}

type accuracyMetadata struct {
	GeneratedAt string `json:"generated_at"`
	Platform    string `json:"platform"`
	Year        int    `json:"year"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

type accuracyCase struct {
	ID          string        `json:"id"`
	Software    softwareInput `json:"software"`
	ExpectedCPE string        `json:"expected_cpe"` // empty = should NOT produce a CPE
	SourceCVE   string        `json:"source_cve,omitempty"`
	CVSSScore   float64       `json:"cvss_score,omitempty"`
	Tags        []string      `json:"tags,omitempty"`
	Notes       string        `json:"notes,omitempty"`
}

type softwareInput struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	Source           string `json:"source"`
	Vendor           string `json:"vendor,omitempty"`
	BundleIdentifier string `json:"bundle_identifier,omitempty"`
	ExtensionID      string `json:"extension_id,omitempty"`
}

// -----------------------------------------------------------------------------
// Result classification
// -----------------------------------------------------------------------------

type resultKind int

const (
	resultPass resultKind = iota
	resultPassTrueNegative
	resultFailMismatch
	resultFailMissing
	resultFailFalsePositive
)

func (k resultKind) Label() string {
	switch k {
	case resultPass:
		return "PASS"
	case resultPassTrueNegative:
		return "PASS (true negative)"
	case resultFailMismatch:
		return "FAIL (mismatch)"
	case resultFailMissing:
		return "FAIL (missing)"
	case resultFailFalsePositive:
		return "FAIL (false positive)"
	}
	return "UNKNOWN"
}

func (k resultKind) IsPass() bool {
	return k == resultPass || k == resultPassTrueNegative
}

type caseResult struct {
	File   string       `json:"file"`
	Case   accuracyCase `json:"case"`
	Actual string       `json:"actual_cpe"`
	Kind   resultKind   `json:"-"`
	KindID string       `json:"kind"`
}

// -----------------------------------------------------------------------------
// softwareIterator implements fleet.SoftwareIterator backed by a fixed slice.
// -----------------------------------------------------------------------------

type softwareIterator struct {
	software []fleet.Software
	i        int
}

func (s *softwareIterator) Next() bool                     { return s.i < len(s.software) }
func (s *softwareIterator) Err() error                     { return nil }
func (s *softwareIterator) Close() error                   { return nil }
func (s *softwareIterator) Value() (*fleet.Software, error) {
	ss := &s.software[s.i]
	s.i++
	return ss, nil
}

// -----------------------------------------------------------------------------
// Entry point
// -----------------------------------------------------------------------------

func main() {
	testdataDir := flag.String("testdata-dir",
		"server/vulnerabilities/nvd/testdata/accuracy",
		"Directory containing cves_*.json golden files")
	vulnPath := flag.String("vuln-path", "",
		"Directory containing cpe.sqlite + cpe_translations.json. Populate with 'fleetctl vulnerability-data-stream --dir <path>'. "+
			"If empty, falls back to $NVD_TEST_VULNDB_DIR, then downloads into a temp dir.")
	verbose := flag.Bool("verbose", false, "Print per-case results (not just failures)")
	jsonOut := flag.Bool("json", false, "Emit machine-readable JSON instead of a human report")
	flag.Parse()

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Load golden files.
	suites, err := loadSuites(*testdataDir)
	if err != nil {
		fatalf("load testdata: %v", err)
	}
	if totalCases(suites) == 0 {
		fmt.Fprintf(os.Stderr, "no test cases found in %s\n", *testdataDir)
		os.Exit(0)
	}

	// Resolve the vulnerabilities directory (cpe.sqlite + cpe_translations.json).
	resolvedVulnPath, cleanup, err := resolveVulnPath(*vulnPath)
	if err != nil {
		fatalf("resolve vuln path: %v", err)
	}
	defer cleanup()

	// Build software list and case index.
	software, caseIndex := buildSoftware(suites)

	// Mock datastore: capture what TranslateSoftwareToCPE upserts/deletes.
	ds, upserts, deletes := newCaptureStore(software)

	// Run.
	if err := nvd.TranslateSoftwareToCPE(ctx, ds, resolvedVulnPath, logger); err != nil {
		fatalf("translate software to CPE: %v", err)
	}

	// Classify results.
	results := classify(software, caseIndex, upserts, deletes)

	// Report.
	if *jsonOut {
		emitJSON(results)
	} else {
		emitHumanReport(results, suites, *verbose)
	}

	// Exit non-zero on any failure.
	for _, r := range results {
		if !r.Kind.IsPass() {
			os.Exit(1)
		}
	}
}

// -----------------------------------------------------------------------------
// Loading and pre-processing
// -----------------------------------------------------------------------------

func loadSuites(dir string) (map[string]*accuracySuite, error) {
	pattern := filepath.Join(dir, "cves_*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob: %w", err)
	}
	sort.Strings(matches)

	suites := make(map[string]*accuracySuite, len(matches))
	for _, path := range matches {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", path, err)
		}
		var s accuracySuite
		dec := json.NewDecoder(f)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&s); err != nil {
			f.Close()
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		f.Close()
		suites[path] = &s
	}
	return suites, nil
}

func totalCases(suites map[string]*accuracySuite) int {
	var n int
	for _, s := range suites {
		n += len(s.TestCases)
	}
	return n
}

// caseRef maps a synthetic software ID back to the suite/case it came from.
type caseRef struct {
	File string
	Case accuracyCase
}

// sentinelCPE forces TranslateSoftwareToCPE to emit a change for every case,
// including ones that generate an empty CPE (deletes). Without the sentinel,
// software with an empty GenerateCPE and an empty generated CPE is skipped.
const sentinelCPE = "cpe:2.3:a:__fleet_cpe_accuracy_sentinel__:*:*:*:*:*:*:*:*:*"

func buildSoftware(suites map[string]*accuracySuite) ([]fleet.Software, map[uint]caseRef) {
	// Deterministic order: sort suite keys, then iterate cases in file order.
	keys := make([]string, 0, len(suites))
	for k := range suites {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var software []fleet.Software
	index := make(map[uint]caseRef)
	var id uint
	for _, file := range keys {
		suite := suites[file]
		if suite == nil {
			continue
		}
		for _, c := range suite.TestCases {
			id++
			software = append(software, fleet.Software{
				ID:               id,
				Name:             c.Software.Name,
				Version:          c.Software.Version,
				Source:           c.Software.Source,
				Vendor:           c.Software.Vendor,
				BundleIdentifier: c.Software.BundleIdentifier,
				ExtensionID:      c.Software.ExtensionID,
				GenerateCPE:      sentinelCPE,
			})
			index[id] = caseRef{File: filepath.Base(file), Case: c}
		}
	}
	return software, index
}

// -----------------------------------------------------------------------------
// Mock datastore
// -----------------------------------------------------------------------------

// newCaptureStore returns a mock.Store that records the CPEs generated for
// each software ID. Upserted CPEs are stored in `upserts`; software that
// produced an empty CPE is stored in `deletes`.
func newCaptureStore(software []fleet.Software) (fleet.Datastore, map[uint]string, map[uint]struct{}) {
	upserts := make(map[uint]string)
	deletes := make(map[uint]struct{})

	ds := new(mock.Store)
	ds.AllSoftwareIteratorFunc = func(ctx context.Context, query fleet.SoftwareIterQueryOptions) (fleet.SoftwareIterator, error) {
		return &softwareIterator{software: software}, nil
	}
	ds.UpsertSoftwareCPEsFunc = func(ctx context.Context, cpes []fleet.SoftwareCPE) (int64, error) {
		for _, cpe := range cpes {
			upserts[cpe.SoftwareID] = cpe.CPE
		}
		return int64(len(cpes)), nil
	}
	ds.DeleteSoftwareCPEsFunc = func(ctx context.Context, cpes []fleet.SoftwareCPE) (int64, error) {
		for _, cpe := range cpes {
			deletes[cpe.SoftwareID] = struct{}{}
		}
		return int64(len(cpes)), nil
	}
	ds.ListSoftwareCPEsFunc = func(ctx context.Context) ([]fleet.SoftwareCPE, error) {
		return nil, nil
	}
	return ds, upserts, deletes
}

// -----------------------------------------------------------------------------
// Classification
// -----------------------------------------------------------------------------

func classify(
	software []fleet.Software,
	index map[uint]caseRef,
	upserts map[uint]string,
	deletes map[uint]struct{},
) []caseResult {
	results := make([]caseResult, 0, len(software))
	for _, sw := range software {
		ref := index[sw.ID]
		actual := upserts[sw.ID]
		_, wasDeleted := deletes[sw.ID]
		if wasDeleted {
			actual = ""
		}
		// Software without a version is skipped by TranslateSoftwareToCPE,
		// producing neither upsert nor delete. Treat as empty output.
		kind := classifyOne(ref.Case.ExpectedCPE, actual)
		results = append(results, caseResult{
			File:   ref.File,
			Case:   ref.Case,
			Actual: actual,
			Kind:   kind,
			KindID: resultKindID(kind),
		})
	}
	return results
}

func classifyOne(expected, actual string) resultKind {
	switch {
	case expected == "" && actual == "":
		return resultPassTrueNegative
	case expected == "" && actual != "":
		return resultFailFalsePositive
	case expected != "" && actual == "":
		return resultFailMissing
	case expected == actual:
		return resultPass
	default:
		return resultFailMismatch
	}
}

func resultKindID(k resultKind) string {
	switch k {
	case resultPass:
		return "pass"
	case resultPassTrueNegative:
		return "pass_true_negative"
	case resultFailMismatch:
		return "fail_mismatch"
	case resultFailMissing:
		return "fail_missing"
	case resultFailFalsePositive:
		return "fail_false_positive"
	}
	return "unknown"
}

// -----------------------------------------------------------------------------
// Reporting
// -----------------------------------------------------------------------------

func emitJSON(results []caseResult) {
	out := map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"results":      results,
		"summary":      summarize(results),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fatalf("encode json: %v", err)
	}
}

func emitHumanReport(results []caseResult, suites map[string]*accuracySuite, verbose bool) {
	fmt.Println("CPE Accuracy Report")
	fmt.Println("===================")
	fmt.Println()

	files := make([]string, 0, len(suites))
	for k := range suites {
		files = append(files, filepath.Base(k))
	}
	sort.Strings(files)
	fmt.Printf("Suites (%d): %v\n", len(files), files)
	fmt.Printf("Total cases: %d\n", len(results))
	fmt.Println()

	s := summarize(results)
	fmt.Println("Results:")
	printLine("  ✓ PASS                   ", s.Pass, len(results))
	printLine("  ✓ PASS (true negative)   ", s.PassTrueNegative, len(results))
	printLine("  ✗ FAIL (mismatch)        ", s.FailMismatch, len(results))
	printLine("  ✗ FAIL (missing)         ", s.FailMissing, len(results))
	printLine("  ✗ FAIL (false positive)  ", s.FailFalsePositive, len(results))
	fmt.Println()

	// Per-source breakdown.
	if len(s.BySource) > 0 {
		fmt.Println("By source:")
		srcs := make([]string, 0, len(s.BySource))
		for k := range s.BySource {
			srcs = append(srcs, k)
		}
		sort.Strings(srcs)
		for _, src := range srcs {
			bs := s.BySource[src]
			if bs == nil {
				continue
			}
			pct := 0.0
			if bs.Total > 0 {
				pct = 100.0 * float64(bs.Passed) / float64(bs.Total)
			}
			fmt.Printf("  %-26s: %d/%d passed (%.1f%%)\n", src, bs.Passed, bs.Total, pct)
		}
		fmt.Println()
	}

	// Individual case output.
	if verbose {
		fmt.Println("Cases:")
		for _, r := range results {
			printCase(r)
		}
		fmt.Println()
	}

	if s.PassTrueNegative+s.Pass < len(results) {
		fmt.Println("Failures:")
		for _, r := range results {
			if !r.Kind.IsPass() {
				printCase(r)
			}
		}
		fmt.Println()
	}

	if s.FailMismatch+s.FailMissing+s.FailFalsePositive == 0 {
		fmt.Println("All checks passed ✓")
	} else {
		fmt.Printf("%d failures\n", s.FailMismatch+s.FailMissing+s.FailFalsePositive)
	}
}

func printLine(label string, n, total int) {
	if total == 0 {
		fmt.Printf("%s %d\n", label, n)
		return
	}
	fmt.Printf("%s %d  (%.1f%%)\n", label, n, 100.0*float64(n)/float64(total))
}

func printCase(r caseResult) {
	fmt.Printf("  [%s] %s\n", r.Kind.Label(), r.Case.ID)
	fmt.Printf("    software: name=%q version=%q source=%q bundle=%q\n",
		r.Case.Software.Name, r.Case.Software.Version, r.Case.Software.Source, r.Case.Software.BundleIdentifier)
	fmt.Printf("    expected: %q\n", r.Case.ExpectedCPE)
	fmt.Printf("    actual:   %q\n", r.Actual)
	if r.Case.SourceCVE != "" {
		fmt.Printf("    cve:      %s (CVSS %.1f)\n", r.Case.SourceCVE, r.Case.CVSSScore)
	}
	fmt.Printf("    file:     %s\n", r.File)
}

type summary struct {
	Pass              int                       `json:"pass"`
	PassTrueNegative  int                       `json:"pass_true_negative"`
	FailMismatch      int                       `json:"fail_mismatch"`
	FailMissing       int                       `json:"fail_missing"`
	FailFalsePositive int                       `json:"fail_false_positive"`
	BySource          map[string]*sourceSummary `json:"by_source,omitempty"`
}

type sourceSummary struct {
	Total  int `json:"total"`
	Passed int `json:"passed"`
}

func summarize(results []caseResult) summary {
	s := summary{BySource: make(map[string]*sourceSummary)}
	for _, r := range results {
		switch r.Kind {
		case resultPass:
			s.Pass++
		case resultPassTrueNegative:
			s.PassTrueNegative++
		case resultFailMismatch:
			s.FailMismatch++
		case resultFailMissing:
			s.FailMissing++
		case resultFailFalsePositive:
			s.FailFalsePositive++
		}
		src := r.Case.Software.Source
		if _, ok := s.BySource[src]; !ok {
			s.BySource[src] = &sourceSummary{}
		}
		s.BySource[src].Total++
		if r.Kind.IsPass() {
			s.BySource[src].Passed++
		}
	}
	return s
}

// -----------------------------------------------------------------------------
// Vulnerability path resolution
// -----------------------------------------------------------------------------

// resolveVulnPath returns a directory containing cpe.sqlite and cpe_translations.json.
// Lookup order:
//  1. --vuln-path flag (used as-is; must contain both files)
//  2. $NVD_TEST_VULNDB_DIR (reuses the caching env var from the existing tests)
//  3. Download fresh copies into a temp directory (caller-managed cleanup)
//
// The returned cleanup function is a no-op unless we created a temp directory.
func resolveVulnPath(explicit string) (string, func(), error) {
	noop := func() {}

	switch {
	case explicit != "":
		if err := requireVulnFiles(explicit); err != nil {
			return "", noop, err
		}
		return explicit, noop, nil
	case os.Getenv("NVD_TEST_VULNDB_DIR") != "":
		p := os.Getenv("NVD_TEST_VULNDB_DIR")
		if err := requireVulnFiles(p); err != nil {
			return "", noop, err
		}
		return p, noop, nil
	}

	tempDir, err := os.MkdirTemp("", "cpe-accuracy-*")
	if err != nil {
		return "", noop, fmt.Errorf("mkdir temp: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }

	if err := nvd.DownloadCPEDBFromGithub(tempDir, ""); err != nil {
		cleanup()
		return "", noop, fmt.Errorf("download cpe db: %w", err)
	}
	if err := nvd.DownloadCPETranslationsFromGithub(tempDir, ""); err != nil {
		cleanup()
		return "", noop, fmt.Errorf("download cpe translations: %w", err)
	}
	return tempDir, cleanup, nil
}

func requireVulnFiles(dir string) error {
	required := []string{"cpe.sqlite", "cpe_translations.json"}
	var missing []string
	for _, name := range required {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return errors.New("missing files in vuln path: " + fmt.Sprint(missing))
	}
	return nil
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(2)
}
