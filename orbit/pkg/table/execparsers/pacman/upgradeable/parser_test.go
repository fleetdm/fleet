package pacman_upgradeable

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed test-data/pacman_upgradeable.txt
var pacman_upgradeable []byte

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
			input: []byte("\ntest-Who\n.-->\t11 -> 25.52\n\t foop\n\n\nfoobar \t1.1.1 -> \t2.2.2\n\nboo \t\n"),
			expected: []map[string]string{
				{
					"package":         "foobar",
					"current_version": "1.1.1",
					"upgrade_version": "2.2.2",
				},
			},
		},
		{
			name:  "pacman_upgradeable",
			input: pacman_upgradeable,
			expected: []map[string]string{
				{
					"package":         "apr-util",
					"current_version": "1.6.1-9",
					"upgrade_version": "1.6.1-10",
				},
				{
					"package":         "archlinux-keyring",
					"current_version": "20220713-1",
					"upgrade_version": "20221123-1",
				},
				{
					"package":         "base",
					"current_version": "2-2",
					"upgrade_version": "3-1",
				},
				{
					"package":         "binutils",
					"current_version": "2.38-6",
					"upgrade_version": "2.39-4",
				},
				{
					"package":         "bzip2",
					"current_version": "1.0.8-4",
					"upgrade_version": "1.0.8-5",
				},
				{
					"package":         "ca-certificates",
					"current_version": "20210603-1",
					"upgrade_version": "20220905-1",
				},
				{
					"package":         "cheese",
					"current_version": "41.1-2",
					"upgrade_version": "43alpha+r8+g1de47dbc-1",
				},
				{
					"package":         "coreutils",
					"current_version": "9.1-1",
					"upgrade_version": "9.1-3",
				},
				{
					"package":         "gnome-shell",
					"current_version": "1:42.3.1-1",
					"upgrade_version": "1:43.2-1",
				},
				{
					"package":         "gnome-software",
					"current_version": "42.3-1",
					"upgrade_version": "43.2-1",
				},
				{
					"package":         "gnome-terminal",
					"current_version": "3.44.1-1",
					"upgrade_version": "3.46.7-1",
				},
				{
					"package":         "linux",
					"current_version": "5.18.14.arch1-1",
					"upgrade_version": "6.0.12.arch1-1",
				},
				{
					"package":         "osquery",
					"current_version": "5.3.0-2",
					"upgrade_version": "5.6.0-2",
				},
				{
					"package":         "pacman",
					"current_version": "6.0.1-5",
					"upgrade_version": "6.0.2-5",
				},
				{
					"package":         "x264",
					"current_version": "3:0.164.r3081.19856cc-2",
					"upgrade_version": "3:0.164.r3095.baee400-4",
				},
				{
					"package":         "xorgproto",
					"current_version": "2022.1-1",
					"upgrade_version": "2022.2-1",
				},
				{
					"package":         "xz",
					"current_version": "5.2.5-3",
					"upgrade_version": "5.2.9-1",
				},
				{
					"package":         "yelp",
					"current_version": "42.1-2",
					"upgrade_version": "42.2-1",
				},
				{
					"package":         "zbar",
					"current_version": "0.23.1-9",
					"upgrade_version": "0.23.90-1",
				},
				{
					"package":         "zlib",
					"current_version": "1:1.2.12-2",
					"upgrade_version": "1:1.2.13-2",
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
