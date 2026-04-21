package connectivity

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"
)

// featureTitle maps a Feature to the heading shown in human output.
var featureTitle = map[Feature]string{
	FeatureOsquery:    "osquery",
	FeatureDesktop:    "Fleet Desktop",
	FeatureFleetctl:   "fleetctl",
	FeatureMDMMacOS:   "MDM (macOS)",
	FeatureMDMWindows: "MDM (Windows)",
	FeatureMDMIOS:     "MDM (iOS / iPadOS)",
	FeatureMDMAndroid: "MDM (Android)",
	FeatureSCEPProxy:  "SCEP proxy",
}

// RenderHuman writes a grouped report of results to w.
func RenderHuman(w io.Writer, baseURL string, results []Result) error {
	if _, err := fmt.Fprintf(w, "Fleet connectivity check: %s\n\n", baseURL); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "Legend: ✅ reachable  ⚠️ responded but did not look like Fleet  ❌ unreachable"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	byFeature := make(map[Feature][]Result, len(AllFeatures()))
	for _, r := range results {
		byFeature[r.Check.Feature] = append(byFeature[r.Check.Feature], r)
	}

	for _, f := range AllFeatures() {
		group, ok := byFeature[f]
		if !ok {
			continue
		}
		if _, err := fmt.Fprintf(w, "%s\n", featureTitle[f]); err != nil {
			return err
		}
		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		for _, r := range group {
			if _, err := fmt.Fprintf(tw, "  %s\t%s\t%s\t%s\n",
				statusMarker(r),
				r.Check.Method,
				r.Check.Path,
				detail(r),
			); err != nil {
				return err
			}
		}
		if err := tw.Flush(); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	s := Summarize(results)
	_, err := fmt.Fprintf(w,
		"Summary: %d reachable, %d not-fleet, %d forbidden, %d route-not-found, %d blocked (of %d checked)\n",
		s.Reachable, s.NotFleet, s.Forbidden, s.NotFound, s.Blocked, s.Total,
	)
	return err
}

// RenderJSON writes a stable, machine-readable report to w.
func RenderJSON(w io.Writer, baseURL string, results []Result) error {
	type outResult struct {
		Feature     Feature `json:"feature"`
		Method      string  `json:"method"`
		Path        string  `json:"path"`
		Description string  `json:"description"`
		Status      Status  `json:"status"`
		HTTPStatus  int     `json:"http_status,omitempty"`
		LatencyMS   int64   `json:"latency_ms"`
		Error       string  `json:"error,omitempty"`
	}
	type report struct {
		FleetURL  string      `json:"fleet_url"`
		CheckedAt time.Time   `json:"checked_at"`
		Results   []outResult `json:"results"`
		Summary   Summary     `json:"summary"`
	}

	out := report{
		FleetURL:  baseURL,
		CheckedAt: time.Now().UTC(),
		Results:   make([]outResult, len(results)),
		Summary:   Summarize(results),
	}
	for i, r := range results {
		out.Results[i] = outResult{
			Feature:     r.Check.Feature,
			Method:      r.Check.Method,
			Path:        r.Check.Path,
			Description: r.Check.Description,
			Status:      r.Status,
			HTTPStatus:  r.HTTPStatus,
			LatencyMS:   r.Latency.Milliseconds(),
			Error:       r.Error,
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// ListCatalogue writes the set of endpoints that would be probed, without
// performing any network I/O. It supports the --list flag on the CLI.
func ListCatalogue(w io.Writer, checks []Check) error {
	sorted := make([]Check, len(checks))
	copy(sorted, checks)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Feature != sorted[j].Feature {
			return featureOrder(sorted[i].Feature) < featureOrder(sorted[j].Feature)
		}
		return sorted[i].Path < sorted[j].Path
	})
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, c := range sorted {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			c.Feature, c.Method, c.Path, c.Description,
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func statusMarker(r Result) string {
	switch r.Status {
	case StatusReachable:
		return "✅"
	case StatusNotFleet:
		return "⚠️ "
	case StatusForbidden, StatusNotFound, StatusBlocked:
		return "❌"
	default:
		return "? "
	}
}

func detail(r Result) string {
	switch r.Status {
	case StatusBlocked:
		if r.Error != "" {
			return fmt.Sprintf("blocked: %s", r.Error)
		}
		return "blocked"
	case StatusForbidden:
		return fmt.Sprintf("HTTP %d (likely blocked by reverse proxy or WAF)", r.HTTPStatus)
	case StatusNotFleet:
		if r.Error != "" {
			return fmt.Sprintf("HTTP %d (%s)", r.HTTPStatus, r.Error)
		}
		return fmt.Sprintf("HTTP %d (response does not look like Fleet)", r.HTTPStatus)
	default:
		return fmt.Sprintf("HTTP %d (%s)", r.HTTPStatus, truncateLatency(r.Latency))
	}
}

func truncateLatency(d time.Duration) string {
	if d >= time.Second {
		return d.Round(10 * time.Millisecond).String()
	}
	return d.Round(time.Millisecond).String()
}

func featureOrder(f Feature) int {
	for i, known := range AllFeatures() {
		if known == f {
			return i
		}
	}
	return len(AllFeatures())
}

// featureNamesList returns a human-readable comma list of the known features.
// Used in CLI flag help so --features' accepted values stay in sync with the
// catalogue.
func featureNamesList() string {
	names := make([]string, 0, len(AllFeatures()))
	for _, f := range AllFeatures() {
		names = append(names, string(f))
	}
	return strings.Join(names, ", ")
}

// FeatureNamesList is the exported form of featureNamesList for CLI use.
func FeatureNamesList() string { return featureNamesList() }
