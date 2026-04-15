// Command cpe-candidates reads a local NVD CVE feed and writes a JSON file
// listing the macOS-relevant (vendor, product) pairs referenced by CVEs that
// match the requested year and severity.
//
// The output feeds tools/nvd/accuracy/testdata-reconcile, which pairs each
// candidate with an osquery snapshot row to produce the final golden test
// cases consumed by tools/nvd/accuracy/cpe-accuracy.
//
// No network, no VM access. Reads files populated by:
//
//	fleetctl vulnerability-data-stream --dir /tmp/vulnsdb
//
// Usage:
//
//	go run ./tools/nvd/accuracy/cpe-candidates                                  # year=2026 severity=CRITICAL
//	go run ./tools/nvd/accuracy/cpe-candidates --year 2025 --min-cvss 7.0       # HIGH+
//	go run ./tools/nvd/accuracy/cpe-candidates --vulnsdb /custom/path --output /tmp/out.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed"
	feednvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

// -----------------------------------------------------------------------------
// Output schema (matches the format the reconcile tool will consume).
// -----------------------------------------------------------------------------

type candidatesFile struct {
	Metadata   candidatesMetadata `json:"metadata"`
	Candidates []candidate        `json:"candidates"`
}

type candidatesMetadata struct {
	GeneratedAt  string   `json:"generated_at"`
	Year         int      `json:"year"`
	Severity     string   `json:"severity"`
	MinCVSSScore float64  `json:"min_cvss_score"`
	SourceFiles  []string `json:"source_files"`
	CVECount     int      `json:"cve_count"`
}

type candidate struct {
	Vendor        string       `json:"vendor"`
	Product       string       `json:"product"`
	TargetSWHints []string     `json:"target_sw_hints"`
	RelatedCVEs   []relatedCVE `json:"related_cves"`
}

type relatedCVE struct {
	ID                  string  `json:"id"`
	CVSSScore           float64 `json:"cvss_score"`
	VersionEndExcluding string  `json:"version_end_excluding,omitempty"`
	VersionEndIncluding string  `json:"version_end_including,omitempty"`
	PublishedDate       string  `json:"published_date,omitempty"`
}

// -----------------------------------------------------------------------------
// NVD feed constants (duplicated here to avoid exporting internals).
// -----------------------------------------------------------------------------

// publishedDateFmt matches server/vulnerabilities/nvd/cve.go:152.
const publishedDateFmt = "2006-01-02T15:04Z"

// severityThreshold maps a severity name to the CVSS v3 base-score floor per
// https://nvd.nist.gov/vuln-metrics/cvss.
var severityThreshold = map[string]float64{
	"CRITICAL": 9.0,
	"HIGH":     7.0,
	"MEDIUM":   4.0,
	"LOW":      0.1,
}

// -----------------------------------------------------------------------------
// Entry point
// -----------------------------------------------------------------------------

func main() {
	vulnsdb := flag.String("vulnsdb", "/tmp/vulnsdb", "Directory with nvdcve-1.1-*.json.gz feed files")
	output := flag.String("output", "",
		"Output JSON file (default: server/vulnerabilities/nvd/testdata/accuracy/cpe_candidates_<year>_<severity>.json)")
	year := flag.Int("year", 2026, "Only include CVEs published in this year")
	severity := flag.String("severity", "CRITICAL",
		"Minimum CVSS v3 severity: CRITICAL (>=9.0), HIGH (>=7.0), MEDIUM (>=4.0), LOW (>=0.1)")
	minCVSS := flag.Float64("min-cvss", -1,
		"Override the CVSS v3 floor directly (takes precedence over --severity)")
	flag.Parse()

	sev := strings.ToUpper(*severity)
	threshold, ok := severityThreshold[sev]
	if !ok {
		fatalf("unknown --severity %q (want CRITICAL|HIGH|MEDIUM|LOW)", *severity)
	}
	if *minCVSS >= 0 {
		threshold = *minCVSS
	}

	feedPath := filepath.Join(*vulnsdb, fmt.Sprintf("nvdcve-1.1-%d.json.gz", *year))
	if _, err := os.Stat(feedPath); err != nil {
		fatalf("feed not found at %s: %v\nhint: run `fleetctl vulnerability-data-stream --dir %s` to populate it", feedPath, err, *vulnsdb)
	}

	outPath := *output
	if outPath == "" {
		outPath = defaultOutputPath(*year, sev)
	}

	candidates, cveCount, err := extractCandidates(feedPath, *year, threshold)
	if err != nil {
		fatalf("extract candidates: %v", err)
	}

	result := candidatesFile{
		Metadata: candidatesMetadata{
			GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
			Year:         *year,
			Severity:     sev,
			MinCVSSScore: threshold,
			SourceFiles:  []string{feedPath},
			CVECount:     cveCount,
		},
		Candidates: candidates,
	}

	if err := writeJSON(outPath, result); err != nil {
		fatalf("write output: %v", err)
	}

	fmt.Fprintf(os.Stderr,
		"wrote %d candidates (%d CVEs considered) to %s\n",
		len(candidates), cveCount, outPath)
}

// -----------------------------------------------------------------------------
// Core extraction
// -----------------------------------------------------------------------------

// extractCandidates loads a single NVD feed file and returns deduplicated
// vendor/product candidates for 2026-published CRITICAL (or higher) CVEs.
// The second return value is the number of CVEs that passed the year+severity
// filter (useful for sanity-checking).
func extractCandidates(feedPath string, year int, minCVSS float64) ([]candidate, int, error) {
	dict, err := cvefeed.LoadJSONDictionary(feedPath)
	if err != nil {
		return nil, 0, fmt.Errorf("load %s: %w", feedPath, err)
	}

	// Intermediate accumulator keyed by "vendor/product".
	type acc struct {
		Vendor        string
		Product       string
		TargetSWHints map[string]struct{}
		CVEs          []relatedCVE
	}
	byProduct := make(map[string]*acc)

	passedCount := 0
	for cveID := range dict {
		vuln, ok := dict[cveID].(*feednvd.Vuln)
		if !ok {
			continue
		}
		if vuln.CVSSv3BaseScore() < minCVSS {
			continue
		}
		item := vuln.Schema()
		if item == nil {
			continue
		}
		published, err := time.Parse(publishedDateFmt, item.PublishedDate)
		if err != nil || published.Year() != year {
			continue
		}
		passedCount++

		// Flatten the configuration tree (top-level nodes + children).
		var nodes []*schema.NVDCVEFeedJSON10DefNode
		if item.Configurations != nil {
			for _, n := range item.Configurations.Nodes {
				nodes = append(nodes, flattenNodes(n)...)
			}
		}

		// A single CVE can reference the same vendor/product multiple times with
		// different version ranges. Collapse to one relatedCVE per (CVE, product),
		// keeping the widest version-end hint we see.
		seenThisCVE := make(map[string]*relatedCVE)

		for _, n := range nodes {
			for _, m := range n.CPEMatch {
				if m == nil || !m.Vulnerable {
					continue
				}
				attrs, err := wfn.Parse(m.Cpe23Uri)
				if err != nil {
					continue
				}
				if !isMacOSRelevantApp(attrs) {
					continue
				}
				vendor := wfn.StripSlashes(attrs.Vendor)
				product := wfn.StripSlashes(attrs.Product)
				if vendor == "" || vendor == wfn.Any || product == "" || product == wfn.Any {
					continue
				}
				key := vendor + "/" + product
				a, found := byProduct[key]
				if !found {
					a = &acc{
						Vendor:        vendor,
						Product:       product,
						TargetSWHints: make(map[string]struct{}),
					}
					byProduct[key] = a
				}
				if ts := wfn.StripSlashes(attrs.TargetSW); ts != "" {
					a.TargetSWHints[ts] = struct{}{}
				}
				if existing, ok := seenThisCVE[key]; ok {
					// Widen the version-end hint if necessary (keep the one that isn't empty).
					if existing.VersionEndExcluding == "" {
						existing.VersionEndExcluding = m.VersionEndExcluding
					}
					if existing.VersionEndIncluding == "" {
						existing.VersionEndIncluding = m.VersionEndIncluding
					}
					continue
				}
				rel := relatedCVE{
					ID:                  cveID,
					CVSSScore:           vuln.CVSSv3BaseScore(),
					VersionEndExcluding: m.VersionEndExcluding,
					VersionEndIncluding: m.VersionEndIncluding,
					PublishedDate:       item.PublishedDate,
				}
				seenThisCVE[key] = &rel
				a.CVEs = append(a.CVEs, rel)
			}
		}
	}

	// Stable, deterministic output: sort by vendor/product; sort CVEs by ID.
	keys := make([]string, 0, len(byProduct))
	for k := range byProduct {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]candidate, 0, len(keys))
	for _, k := range keys {
		a := byProduct[k]
		if a == nil {
			continue
		}
		hints := make([]string, 0, len(a.TargetSWHints))
		for h := range a.TargetSWHints {
			hints = append(hints, h)
		}
		sort.Strings(hints)
		sort.Slice(a.CVEs, func(i, j int) bool { return a.CVEs[i].ID < a.CVEs[j].ID })
		out = append(out, candidate{
			Vendor:        a.Vendor,
			Product:       a.Product,
			TargetSWHints: hints,
			RelatedCVEs:   a.CVEs,
		})
	}
	return out, passedCount, nil
}

// -----------------------------------------------------------------------------
// Filtering helpers
// -----------------------------------------------------------------------------

// isMacOSRelevantApp returns true for application CPEs (part=a) whose target_sw
// is either unspecified, wildcard, or plausibly macOS-related. Windows-only and
// Linux-only CPEs are filtered out here; hardware (h) and operating system (o)
// CPEs are out of scope for this test suite.
func isMacOSRelevantApp(attrs *wfn.Attributes) bool {
	if attrs == nil {
		return false
	}
	if attrs.Part != "a" {
		return false
	}
	ts := wfn.StripSlashes(attrs.TargetSW)
	// Empty / wildcard / macOS-specific target_sw values are all acceptable
	// (wfn.Any is the empty/wildcard marker; wfn.NA is the negative "-").
	switch ts {
	case wfn.Any, wfn.NA, "macos", "mac_os_x", "apple_macos":
		return true
	// Cross-platform ecosystems Fleet captures on macOS too.
	case "chrome", "firefox", "safari",
		"python", "node.js", "nodejs",
		"visual_studio_code", "intellij":
		return true
	}
	return false
}

// -----------------------------------------------------------------------------
// Configuration tree walker
// -----------------------------------------------------------------------------

// flattenNodes returns n and every descendant node in a flat slice so the
// caller can iterate their CPE matches without recursion.
func flattenNodes(n *schema.NVDCVEFeedJSON10DefNode) []*schema.NVDCVEFeedJSON10DefNode {
	if n == nil {
		return nil
	}
	out := []*schema.NVDCVEFeedJSON10DefNode{n}
	for _, c := range n.Children {
		out = append(out, flattenNodes(c)...)
	}
	return out
}

// -----------------------------------------------------------------------------
// Output helpers
// -----------------------------------------------------------------------------

func defaultOutputPath(year int, severity string) string {
	return filepath.Join(
		"server", "vulnerabilities", "nvd", "testdata", "accuracy",
		fmt.Sprintf("cpe_candidates_%d_%s.json", year, strings.ToLower(severity)),
	)
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".cpe-candidates-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		// Remove stale tempfile on error path. os.Remove on a renamed file is a no-op.
		_ = os.Remove(tmpPath)
	}()
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		tmp.Close()
		return fmt.Errorf("encode: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", a...)
	os.Exit(1)
}
