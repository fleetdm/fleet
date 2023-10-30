package dpkg

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed test-data/dpkg_info.txt
var dpkg_info []byte

func TestParse(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name     string
		input    []byte
		expected []map[string]string
	}{
		{
			name:     "empty input",
			expected: make([]map[string]string, 0),
		},
		{
			name:  "malformed input",
			input: []byte("\n\n\nPackage/ foo\nPriority; bar\n\nBlah: word\nSection: admin\nVersion: ubun2.5v\nPackage: test\n\n"),
			expected: []map[string]string{
				{
					"package": "test",
					"section": "admin",
					"version": "ubun2.5v",
				},
			},
		},
		{
			name:  "dpkg_info",
			input: dpkg_info,
			expected: []map[string]string{
				{
					"package":         "adduser",
					"priority":        "important",
					"section":         "admin",
					"version":         "3.118ubuntu5",
					"description":     "add and remove users and groups",
					"build-essential": "yes",
				},
				{
					"package":         "apt",
					"priority":        "important",
					"section":         "admin",
					"version":         "2.4.5",
					"description":     "commandline package manager",
					"build-essential": "yes",
				},
				{
					"package":     "apt-utils",
					"priority":    "important",
					"section":     "admin",
					"version":     "2.4.5",
					"description": "package management related utility programs",
				},
				{
					"package":     "base-files",
					"essential":   "yes",
					"priority":    "required",
					"section":     "admin",
					"version":     "12ubuntu4",
					"description": "Debian base system miscellaneous files",
				},
				{
					"package":     "base-passwd",
					"essential":   "yes",
					"priority":    "required",
					"section":     "admin",
					"version":     "3.5.52build1",
					"description": "Debian base system master password and group files",
				},
				{
					"package":     "bash",
					"essential":   "yes",
					"priority":    "required",
					"section":     "shells",
					"version":     "5.1-6ubuntu1",
					"description": "GNU Bourne Again SHell",
				},
				{
					"package":     "cron",
					"priority":    "standard",
					"section":     "admin",
					"version":     "3.0pl1-137ubuntu3",
					"description": "process scheduling daemon",
				},
				{
					"package":     "libkrb5-3",
					"priority":    "required",
					"section":     "libs",
					"version":     "1.19.2-2",
					"description": "MIT Kerberos runtime libraries",
				},
				{
					"package":     "liblocale-gettext-perl",
					"priority":    "important",
					"section":     "perl",
					"version":     "1.07-4build3",
					"description": "module using libc functions for internationalization in Perl",
				},
				{
					"package":     "sudo",
					"priority":    "important",
					"section":     "admin",
					"version":     "1.9.9-1ubuntu2",
					"description": "Provide limited super user privileges to specific users",
				},
				{
					"package":     "whiptail",
					"priority":    "important",
					"section":     "utils",
					"version":     "0.52.21-5ubuntu2",
					"description": "Displays user-friendly dialog boxes from shell scripts",
				},
				{
					"package":     "xdg-user-dirs",
					"priority":    "important",
					"section":     "utils",
					"version":     "0.17-2ubuntu4",
					"description": "tool to manage well known user directories",
				},
				{
					"package":     "xkb-data",
					"priority":    "important",
					"section":     "x11",
					"version":     "2.33-1",
					"description": "X Keyboard Extension (XKB) configuration data",
				},
				{
					"package":     "xxd",
					"priority":    "important",
					"section":     "editors",
					"version":     "2:8.2.3995-1ubuntu2",
					"description": "tool to make (or reverse) a hex dump",
				},
				{
					"package":     "zlib1g",
					"priority":    "required",
					"section":     "libs",
					"version":     "1:1.2.11.dfsg-2ubuntu9",
					"description": "compression library - runtime",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := New()
			result, err := p.Parse(bytes.NewReader(tt.input))
			require.NoError(t, err, "unexpected error parsing input")

			require.ElementsMatch(t, tt.expected, result)
		})
	}
}
