package pacman_group

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"
)

//go:embed test-data/pacman_group.txt
var pacman_group []byte

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
			input: []byte("\n\nGhdj%\n%@&-gh\t\nfoo   \t    bar\n\n"),
			expected: []map[string]string{
				{
					"group":   "foo",
					"package": "bar",
				},
			},
		},
		{
			name:  "pacman_group",
			input: pacman_group,
			expected: []map[string]string{
				{
					"group":   "base-devel",
					"package": "autoconf",
				},
				{
					"group":   "base-devel",
					"package": "binutils",
				},
				{
					"group":   "base-devel",
					"package": "pacman",
				},
				{
					"group":   "base-devel",
					"package": "sed",
				},
				{
					"group":   "base-devel",
					"package": "sudo",
				},
				{
					"group":   "base-devel",
					"package": "which",
				},
				{
					"group":   "gnome",
					"package": "cheese",
				},
				{
					"group":   "gnome",
					"package": "gedit",
				},
				{
					"group":   "gnome",
					"package": "gnome-shell",
				},
				{
					"group":   "gnome",
					"package": "gnome-software",
				},
				{
					"group":   "gnome",
					"package": "gnome-terminal",
				},
				{
					"group":   "gnome",
					"package": "sushi",
				},
				{
					"group":   "gnome",
					"package": "yelp",
				},
				{
					"group":   "pro-audio",
					"package": "fluidsynth",
				},
				{
					"group":   "default",
					"package": "launcher-kolide-k2",
				},
				{
					"group":   "xorg-drivers",
					"package": "xf86-input-libinput",
				},
				{
					"group":   "xorg",
					"package": "xf86-video-vesa",
				},
				{
					"group":   "xorg",
					"package": "xorg-server",
				},
				{
					"group":   "xorg-apps",
					"package": "xorg-xpr",
				},
				{
					"group":   "xorg-fonts",
					"package": "xorg-font-util",
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
