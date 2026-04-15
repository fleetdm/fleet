package main

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsMacOSRelevantApp(t *testing.T) {
	cases := []struct {
		name string
		cpe  string
		want bool
	}{
		{"app with no target_sw", "cpe:2.3:a:mozilla:firefox:125.0:*:*:*:*:*:*:*", true},
		{"app with macos target_sw", "cpe:2.3:a:mozilla:firefox:125.0:*:*:*:*:macos:*:*", true},
		{"app with mac_os_x target_sw", "cpe:2.3:a:apple:icloud:1.0:*:*:*:*:mac_os_x:*:*", true},
		{"chrome extension", "cpe:2.3:a:1password:1password:2.3:*:*:*:*:chrome:*:*", true},
		{"firefox addon", "cpe:2.3:a:bitwarden:bitwarden:1.55:*:*:*:*:firefox:*:*", true},
		{"python package", "cpe:2.3:a:pypa:pip:22.0:*:*:*:*:python:*:*", true},
		{"npm package", "cpe:2.3:a:openjsf:express:4.0:*:*:*:*:node.js:*:*", true},
		{"vscode extension", "cpe:2.3:a:microsoft:python_extension:2024.0:*:*:*:*:visual_studio_code:*:*", true},
		{"windows-specific app is filtered", "cpe:2.3:a:foo:bar:1.0:*:*:*:*:windows:*:*", false},
		{"linux-specific app is filtered", "cpe:2.3:a:foo:bar:1.0:*:*:*:*:linux:*:*", false},
		{"android-specific app is filtered", "cpe:2.3:a:foo:bar:1.0:*:*:*:*:android:*:*", false},
		{"operating system CPE is filtered (part=o)", "cpe:2.3:o:apple:macos:15.1:*:*:*:*:*:*:*", false},
		{"hardware CPE is filtered (part=h)", "cpe:2.3:h:some:router:1.0:*:*:*:*:*:*:*", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			attrs, err := wfn.Parse(tc.cpe)
			require.NoError(t, err)
			assert.Equal(t, tc.want, isMacOSRelevantApp(attrs))
		})
	}
}

func TestIsMacOSRelevantAppNilSafe(t *testing.T) {
	assert.False(t, isMacOSRelevantApp(nil))
}

func TestFlattenNodesNil(t *testing.T) {
	assert.Nil(t, flattenNodes(nil))
}

func TestFlattenNodesWalksChildren(t *testing.T) {
	root := &schema.NVDCVEFeedJSON10DefNode{
		Operator: "AND",
		CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
			{Cpe23Uri: "cpe:2.3:a:root:match:*:*:*:*:*:*:*:*", Vulnerable: true},
		},
		Children: []*schema.NVDCVEFeedJSON10DefNode{
			{
				Operator: "OR",
				CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
					{Cpe23Uri: "cpe:2.3:a:child:one:*:*:*:*:*:*:*:*"},
					{Cpe23Uri: "cpe:2.3:a:child:two:*:*:*:*:*:*:*:*"},
				},
			},
			{
				Operator: "OR",
				Children: []*schema.NVDCVEFeedJSON10DefNode{
					{
						CPEMatch: []*schema.NVDCVEFeedJSON10DefCPEMatch{
							{Cpe23Uri: "cpe:2.3:a:grandchild:deep:*:*:*:*:*:*:*:*"},
						},
					},
				},
			},
		},
	}
	flat := flattenNodes(root)

	// 1 root + 2 children + 1 grandchild = 4 nodes
	require.Len(t, flat, 4)
	assert.Equal(t, "AND", flat[0].Operator)
}

func TestSeverityThresholds(t *testing.T) {
	// Sanity-check that our threshold constants match NIST conventions.
	assert.InDelta(t, 9.0, severityThreshold["CRITICAL"], 0.01)
	assert.InDelta(t, 7.0, severityThreshold["HIGH"], 0.01)
	assert.InDelta(t, 4.0, severityThreshold["MEDIUM"], 0.01)
	assert.InDelta(t, 0.1, severityThreshold["LOW"], 0.01)
}
