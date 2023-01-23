package falconctl

import (
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOptions(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name        string
		input       []byte
		expected    any
		expectedErr bool
	}{
		{
			name:     "empty",
			expected: map[string]any{},
		},
		{
			name:     "--cid",
			input:    []byte(`cid="REDACTED"`),
			expected: map[string]any{"cid": "REDACTED"},
		},
		{
			name:     "--aid",
			input:    []byte(`aid="REDACTED"`),
			expected: map[string]any{"aid": "REDACTED"},
		},
		{
			name:     "--apd",
			input:    []byte(`apd is not set,`),
			expected: map[string]any{"apd": "is not set"},
		},
		{

			name:     "--aph",
			input:    []byte(`aph is not set,`),
			expected: map[string]any{"aph": "is not set"},
		},
		{
			name:     "--app",
			input:    []byte(`app is not set,`),
			expected: map[string]any{"app": "is not set"},
		},
		{
			name:     "--rfm-state",
			input:    []byte(`rfm-state=false,`),
			expected: map[string]any{"rfm-state": "false"},
		},
		{
			name:     "--rfm-reason",
			input:    []byte(`rfm-reason=None, code=0x0,`),
			expected: map[string]any{"rfm-reason": "None", "rfm-reason-code": "0x0"},
		},
		{
			name:     "--trace",
			input:    []byte(`trace is not set,`),
			expected: map[string]any{"trace": "is not set"},
		},
		{
			name:     "--feature",
			input:    []byte(`feature= (hex bitmask: 0),`),
			expected: map[string]any{"feature": "(hex bitmask: 0)"},
		},
		{
			name:     "--metadata-query",
			input:    []byte(`metadata-query=enable (unset default),`),
			expected: map[string]any{"metadata-query": "enable"},
		},
		{
			name:     "--version",
			input:    []byte(`version = 6.45.14203.0,`),
			expected: map[string]any{"version": "6.45.14203.0"},
		},
		{
			name:     "--billing",
			input:    []byte(`billing is not set,`),
			expected: map[string]any{"billing": "is not set"},
		},

		// Tags are quite tricky to parse\
		{
			name:     "--tags",
			input:    []byte(`tags=kolide-test-1,kolide-test-2,`),
			expected: map[string]any{"tags": []string{"kolide-test-1", "kolide-test-2"}},
		},
		{
			name:  "--rfm-state --rfm-reason --aph --tags",
			input: []byte("aph is not set, rfm-state=false, rfm-reason=None, code=0x0, tags=kolide-test-1,kolide-test-2."),
			expected: map[string]any{
				"aph":             "is not set",
				"rfm-reason":      "None",
				"rfm-reason-code": "0x0",
				"rfm-state":       "false",
				"tags":            []string{"kolide-test-1", "kolide-test-2"},
			},
		},
		{
			name:  "-rfm-state --rfm-reason --aph --tags --version",
			input: []byte("aph is not set, rfm-state=false, rfm-reason=None, code=0x0, version = 6.45.14203.0\ntags=kolide-test-1,kolide-test-2,"),
			expected: map[string]any{
				"aph":             "is not set",
				"rfm-reason":      "None",
				"rfm-reason-code": "0x0",
				"rfm-state":       "false",
				"version":         "6.45.14203.0",
				"tags":            []string{"kolide-test-1", "kolide-test-2"},
			},
		},

		// something with a bunch of things
		{

			name:  "normal",
			input: readTestFile(t, path.Join("test-data", "options.txt")),
			expected: map[string]any{
				"aid":            "is not set",
				"aph":            "is not set",
				"app":            "is not set",
				"cid":            "ac917ab****************************",
				"feature":        "is not set",
				"metadata-query": "enable",
				"rfm-reason":     "is not set",
				"rfm-state":      "is not set",
				"version":        "6.38.13501.0"},
		},
		{
			name:     "--rfm-state --rfm-reason --aph",
			input:    []byte("aph is not set, rfm-state=false, rfm-reason=None, code=0x0.\n"),
			expected: map[string]any{"aph": "is not set", "rfm-reason": "None", "rfm-reason-code": "0x0", "rfm-state": "false"},
		},
		{
			name:        "cid not set",
			input:       readTestFile(t, path.Join("test-data", "cid-error.txt")),
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := parseOptions(bytes.NewReader(tt.input))
			if tt.expectedErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})
	}

}

func readTestFile(t *testing.T, filepath string) []byte {
	b, err := os.ReadFile(filepath)
	require.NoError(t, err)
	return b
}
