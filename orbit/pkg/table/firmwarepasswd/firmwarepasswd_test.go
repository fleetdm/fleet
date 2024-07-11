// based on github.com/kolide/launcher/pkg/osquery/tables
package firmwarepasswd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected map[string]string
	}{
		{
			input:    "check-no.txt",
			expected: map[string]string{"password_enabled": "0"},
		},
		{
			input:    "check-garbage.txt",
			expected: map[string]string{"password_enabled": "0"},
		},
		{
			input:    "check-yes.txt",
			expected: map[string]string{"password_enabled": "1"},
		},
		{
			input: "mode-command.txt",
			expected: map[string]string{
				"mode":                "command",
				"option_roms_allowed": "0",
			},
		},
		{
			input: "mode-none.txt",
			expected: map[string]string{
				"mode":                "none",
				"option_roms_allowed": "1",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		parser := New(zerolog.Nop()).parser

		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()

			inputBytes, err := os.ReadFile(filepath.Join("testdata", tt.input))
			require.NoError(t, err, "read file %s", tt.input)

			inputBuffer := bytes.NewBuffer(inputBytes)

			result := make(map[string]string)
			for _, row := range parser.Parse(inputBuffer) {
				for k, v := range row {
					result[k] = v
				}
			}

			require.EqualValues(t, tt.expected, result)
		})
	}
}
