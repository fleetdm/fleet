//go:build darwin
// +build darwin

package softwareupdate

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed test-data/beta-update-available-noscan.txt
var beta_update_available_noscan []byte

//go:embed test-data/beta-update-available-scan.txt
var beta_update_available_scan []byte

//go:embed test-data/error-scan.txt
var error_scan []byte

//go:embed test-data/multiple-updates-available-noscan.txt
var multiple_updates_available_noscan []byte

//go:embed test-data/no-update-available-noscan.txt
var no_update_available_noscan []byte

//go:embed test-data/no-update-available-scan.txt
var no_update_available_scan []byte

//go:embed test-data/update-available-noscan.txt
var update_available_noscan []byte

//go:embed test-data/update-available-scan.txt
var update_available_scan []byte

func TestParse(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name     string
		input    []byte
		expected []map[string]string
	}{
		{
			name:  "beta update available, --no-scan",
			input: beta_update_available_noscan,
			expected: []map[string]string{
				{
					"Label":       "macOS Ventura 13.3 Beta 3-22E5236f",
					"Title":       "macOS Ventura 13.3 Beta 3",
					"Version":     "13.3",
					"Size":        "3310848K",
					"Recommended": "YES",
					"Action":      "restart",
				},
			},
		},
		{
			name:  "beta update available",
			input: beta_update_available_scan,
			expected: []map[string]string{
				{
					"Label":       "macOS Ventura 13.3 Beta 3-22E5236f",
					"Title":       "macOS Ventura 13.3 Beta 3",
					"Version":     "13.3",
					"Size":        "3310848K",
					"Recommended": "YES",
					"Action":      "restart",
				},
			},
		},
		{
			name:     "error when scanning",
			input:    error_scan,
			expected: make([]map[string]string, 0),
		},
		{
			name:  "multiple updates available, --no-scan",
			input: multiple_updates_available_noscan,
			expected: []map[string]string{
				{
					"Label":       "Command Line Tools for Xcode-14.3",
					"Title":       "Command Line Tools for Xcode",
					"Version":     "14.3",
					"Size":        "711888KiB",
					"Recommended": "YES",
				},
				{
					"Label":       "macOS Ventura 13.4 Beta-22F5027f",
					"Title":       "macOS Ventura 13.4 Beta",
					"Version":     "13.4",
					"Size":        "11487824K",
					"Recommended": "YES",
					"Action":      "restart",
				},
			},
		},
		{
			name:  "no update available, --no-scan",
			input: no_update_available_noscan,
			expected: []map[string]string{
				{
					"UpToDate": "true",
				},
			},
		},
		{
			name:  "no update available",
			input: no_update_available_scan,
			expected: []map[string]string{
				{
					"UpToDate": "true",
				},
			},
		},
		{
			name:  "update available, --no-scan",
			input: update_available_noscan,
			expected: []map[string]string{
				{
					"Label":       "macOS Ventura 13.3.1-22E261",
					"Title":       "macOS Ventura 13.3.1",
					"Version":     "13.3.1",
					"Size":        "868648KiB",
					"Recommended": "YES",
					"Action":      "restart",
				},
			},
		},
		{
			name:  "update available",
			input: update_available_scan,
			expected: []map[string]string{
				{
					"Label":       "macOS Ventura 13.3.1-22E261",
					"Title":       "macOS Ventura 13.3.1",
					"Version":     "13.3.1",
					"Size":        "868648KiB",
					"Recommended": "YES",
					"Action":      "restart",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
		})

		p := New()
		result, err := p.Parse(bytes.NewReader(tt.input))
		require.NoError(t, err, "unexpected error parsing input")

		require.ElementsMatch(t, tt.expected, result)
	}
}
