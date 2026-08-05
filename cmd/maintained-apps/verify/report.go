package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// report is the full verification report, written as JSON (for machine
// consumption, e.g. reverting hard-failed apps from an ingest PR) and as
// markdown (for the PR body / job summary the human reviewer reads).
type report struct {
	// BaseRef is the git ref the outputs were diffed against; empty for
	// full-catalog (--all) runs.
	BaseRef string `json:"base_ref,omitempty"`
	// Mode is "report-only" or "enforce".
	Mode string             `json:"mode"`
	OS   string             `json:"os"`
	Apps []*appVerification `json:"apps"`
}

func (r *report) json() ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r *report) failures() []*appVerification {
	var failed []*appVerification
	for _, av := range r.Apps {
		if len(av.Failures) > 0 {
			failed = append(failed, av)
		}
	}
	return failed
}

func (r *report) warnings() []*appVerification {
	var warned []*appVerification
	for _, av := range r.Apps {
		if len(av.Failures) == 0 && len(av.Warnings) > 0 {
			warned = append(warned, av)
		}
	}
	return warned
}

func (r *report) markdown() string {
	var b strings.Builder

	b.WriteString("### FMA installer verification report\n\n")

	failures := r.failures()
	warnings := r.warnings()
	scope := fmt.Sprintf("changed vs. `%s`", r.BaseRef)
	if r.BaseRef == "" {
		scope = "full catalog"
	}
	fmt.Fprintf(&b, "Mode: **%s** · Scope: %s · %d app(s) verified, **%d failure(s)**, %d warning(s)\n\n",
		r.Mode, scope, len(r.Apps), len(failures), len(warnings))

	if len(r.Apps) == 0 {
		b.WriteString("No changed installers to verify.\n")
		return b.String()
	}

	b.WriteString("| App | Version | Hash | Signature | Notarization | Result |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for _, av := range r.Apps {
		result := "✅ PASS"
		switch {
		case len(av.Failures) > 0:
			result = "❌ FAIL"
		case len(av.Warnings) > 0:
			result = "⚠️ WARN"
		}
		notarization := "—"
		if av.Notarization.Status != "" {
			notarization = renderCheck(av.Notarization)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n",
			av.Slug,
			av.Version,
			renderCheck(av.Hash),
			renderCheck(av.Signature),
			notarization,
			result,
		)
	}
	b.WriteString("\n")

	if len(failures) > 0 || len(warnings) > 0 {
		b.WriteString("<details><summary>Failure and warning details</summary>\n\n")
		for _, av := range r.Apps {
			if len(av.Failures) == 0 && len(av.Warnings) == 0 {
				continue
			}
			fmt.Fprintf(&b, "**%s** (%s)\n", av.Slug, av.Version)
			for _, f := range av.Failures {
				fmt.Fprintf(&b, "- ❌ %s\n", f)
			}
			for _, w := range av.Warnings {
				fmt.Fprintf(&b, "- ⚠️ %s\n", w)
			}
			b.WriteString("\n")
		}
		b.WriteString("</details>\n")
	}

	return b.String()
}

func renderCheck(c checkResult) string {
	icon := map[checkStatus]string{
		statusPass:     "✅",
		statusFail:     "❌",
		statusWarn:     "⚠️",
		statusRecorded: "📝",
		statusSkipped:  "⏭️",
		statusError:    "❌",
	}[c.Status]
	if icon == "" {
		icon = "—"
	}
	detail := c.Detail
	// Keep the table readable; full detail lives in the JSON report.
	if runes := []rune(detail); len(runes) > 90 {
		detail = string(runes[:87]) + "…"
	}
	detail = strings.ReplaceAll(detail, "|", "\\|")
	detail = strings.ReplaceAll(detail, "\n", " ")
	if detail == "" {
		return icon + " " + string(c.Status)
	}
	return fmt.Sprintf("%s %s", icon, detail)
}
