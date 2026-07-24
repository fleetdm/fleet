package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReportMarkdown(t *testing.T) {
	rep := &report{
		BaseRef: "origin/main",
		Mode:    "report-only",
		OS:      "linux",
		Apps: []*appVerification{
			{
				Slug:    "putty/windows",
				Version: "0.84.0.0",
				Hash:    checkResult{Status: statusPass, Detail: "recomputed hash matches manifest"},
				Signature: checkResult{
					Status: statusRecorded,
					Detail: "signed by [Simon Tatham] (no pin yet)",
				},
			},
			{
				Slug:      "7-zip/windows",
				Version:   "26.02",
				Hash:      checkResult{Status: statusPass},
				Signature: checkResult{Status: statusWarn, Detail: `installer has no Authenticode signature and no "unsigned" pin`},
				Warnings:  []string{"installer is unsigned"},
			},
			{
				Slug:      "evil/windows",
				Version:   "1.0",
				Hash:      checkResult{Status: statusFail, Detail: "manifest claims aaaa but downloaded bytes hash to bbbb"},
				Signature: checkResult{Status: statusSkipped},
				Failures:  []string{"SHA256 mismatch"},
			},
			{
				Slug:         "box-drive/darwin",
				Version:      "2.52.312",
				Hash:         checkResult{Status: statusPass},
				Signature:    checkResult{Status: statusPass, Detail: "signed by Developer ID Installer: Box, Inc. (M683GB7CPW) (matches pin)"},
				Notarization: checkResult{Status: statusPass, Detail: "accepted; source=Notarized Developer ID"},
			},
		},
	}

	md := rep.markdown()
	require.Contains(t, md, "**report-only**")
	require.Contains(t, md, "4 app(s) verified, **1 failure(s)**, 1 warning(s)")
	require.Contains(t, md, "| putty/windows | 0.84.0.0 |")
	require.Contains(t, md, "❌ FAIL")
	require.Contains(t, md, "⚠️ WARN")
	require.Contains(t, md, "✅ PASS")
	require.Contains(t, md, "Failure and warning details")
	require.Contains(t, md, "SHA256 mismatch")
	// Windows apps have no notarization column value.
	require.Contains(t, md, "| ✅ recomputed hash matches manifest | 📝 signed by [Simon Tatham] (no pin yet) | — | ✅ PASS |")
}

func TestReportMarkdownEmpty(t *testing.T) {
	rep := &report{BaseRef: "origin/main", Mode: "report-only", OS: "linux"}
	md := rep.markdown()
	require.Contains(t, md, "No changed installers to verify.")
}

func TestReportJSONRoundTrip(t *testing.T) {
	rep := &report{
		Mode: "enforce",
		OS:   "linux",
		Apps: []*appVerification{
			{
				Slug:          "putty/windows",
				Version:       "0.84.0.0",
				ClaimedSHA256: "aaaa",
				Hash:          checkResult{Status: statusPass},
				Signature:     checkResult{Status: statusPass},
			},
		},
	}
	data, err := rep.json()
	require.NoError(t, err)

	var parsed report
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Equal(t, "enforce", parsed.Mode)
	require.Len(t, parsed.Apps, 1)
	require.Equal(t, statusPass, parsed.Apps[0].Hash.Status)
	// Windows entries omit the notarization check entirely.
	require.Equal(t, checkResult{}, parsed.Apps[0].Notarization)
}

func TestReportFailuresAndWarnings(t *testing.T) {
	rep := &report{
		Apps: []*appVerification{
			{Slug: "a/windows", Failures: []string{"x"}},
			{Slug: "b/windows", Warnings: []string{"y"}},
			{Slug: "c/windows", Failures: []string{"x"}, Warnings: []string{"y"}},
			{Slug: "d/windows"},
		},
	}
	failures := rep.failures()
	require.Len(t, failures, 2)
	warnings := rep.warnings()
	require.Len(t, warnings, 1)
	require.Equal(t, "b/windows", warnings[0].Slug)
}
