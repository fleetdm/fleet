// Package perf holds the catalog of osquery-perf host templates the load
// generator accepts. Ported from src-tauri/src/perf.rs.
package perf

// Template is one OS template the osquery-perf agent can simulate.
type Template struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Version string `json:"version"`
	// Mobile templates enroll via MDM only (no enroll secret). The UI uses
	// this to drop the enroll-secret requirement when only mobile templates
	// are selected.
	Mobile bool `json:"mobile"`
	// Apple platforms (macOS, iOS, iPadOS) need a SCEP challenge for MDM
	// enrollment; Windows does not. The UI requires the SCEP field only
	// when an Apple template is in the run.
	Apple bool `json:"apple"`
}

// Templates returns the hardcoded set of templates the osquery-perf agent
// accepts. Mirrors validTemplateNames in cmd/osquery-perf/agent.go — the
// agent only embeds these, so listing anything else just produces an
// "Invalid template name" error at launch.
//
// Order: desktop first, then mobile, alphabetical within each group.
func Templates() []Template {
	return []Template{
		{ID: "macos_13.6.2", Label: "macOS", Version: "13 Ventura", Mobile: false, Apple: true},
		{ID: "macos_14.1.2", Label: "macOS", Version: "14 Sonoma", Mobile: false, Apple: true},
		{ID: "windows_11", Label: "Windows", Version: "11", Mobile: false, Apple: false},
		{ID: "windows_11_22H2_2861", Label: "Windows", Version: "11 22H2 (build 2861)", Mobile: false, Apple: false},
		{ID: "windows_11_22H2_3007", Label: "Windows", Version: "11 22H2 (build 3007)", Mobile: false, Apple: false},
		{ID: "ubuntu_22.04", Label: "Ubuntu", Version: "22.04 LTS", Mobile: false, Apple: false},
		{ID: "rhel_8", Label: "RHEL", Version: "8", Mobile: false, Apple: false},
		{ID: "rhel_9", Label: "RHEL", Version: "9", Mobile: false, Apple: false},
		{ID: "rhel_10", Label: "RHEL", Version: "10", Mobile: false, Apple: false},
		{ID: "iphone_14.6", Label: "iOS", Version: "14.6", Mobile: true, Apple: true},
		{ID: "iphone_17", Label: "iOS", Version: "17", Mobile: true, Apple: true},
		{ID: "ipad_13.18", Label: "iPadOS", Version: "13.18", Mobile: true, Apple: true},
	}
}
