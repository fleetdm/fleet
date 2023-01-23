package dsregcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name        string
		input       []byte
		expected    any
		expectedErr bool
	}{
		{
			name:     "empty input",
			expected: map[string]any{},
		},
		{
			name:     "not configured",
			input:    readTestFile(t, path.Join("test-data", "not_configured.txt")),
			expected: jsonUnmarshal(t, readTestFile(t, path.Join("test-data", "not_configured.expected.json"))),
		},

		//
		// Error Cases
		//
		{
			name:        "lines before header",
			input:       readTestFile(t, path.Join("test-data", "error_lines_before_header.txt")),
			expectedErr: true,
		},
		{
			name:        "header missing line 3",
			input:       readTestFile(t, path.Join("test-data", "error_header_missing_line3.txt")),
			expectedErr: true,
		},
		{
			name:        "no section title",
			input:       readTestFile(t, path.Join("test-data", "error_no_section_title.txt")),
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual, err := parseDsreg(bytes.NewReader(tt.input))
			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, actual)
				return
			}

			require.NoError(t, err)

			// To compare the values, we marshal to JSON and compare the JSON. We do this to avoid issues around the
			// typing on `any`
			require.Equal(t, jsonMarshal(t, tt.expected), jsonMarshal(t, actual))
		})
	}
}

func jsonMarshal(t *testing.T, v any) []byte {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func readTestFile(t *testing.T, filepath string) []byte {
	b, err := os.ReadFile(filepath)
	require.NoError(t, err)
	return b
}

func jsonUnmarshal(t *testing.T, data []byte) any {
	var v any
	err := json.Unmarshal(data, &v)
	require.NoError(t, err)
	return v
}
