package simple_array

import (
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/kolide/kit/ulid"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	t.Parallel()

	nonce := ulid.New()
	testParser := New(nonce)

	var tests = []struct {
		name        string
		input       []byte
		expected    any
		expectedErr bool
	}{
		{
			name:     "empty",
			expected: []string{},
		},
		{
			name:     "simple",
			input:    readTestFile(t, path.Join("test-data", "simple.txt")),
			expected: []string{"123", "12345678901234", "12345678901235", "12345678901236", "12345678901237", "12345678901238", "12345678901239", "12345678901230"},
		},
		{
			name:     "complex",
			input:    readTestFile(t, path.Join("test-data", "complex.txt")),
			expected: []string{"one", "two", "three", "four", "five", "six", "seven", "eight"},
		},
		{
			name:        "malformed",
			input:       []byte("123, this is malformed"),
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := parse(bytes.NewReader(tt.input))
			if tt.expectedErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, actual)
		})

		t.Run(tt.name+" via struct", func(t *testing.T) {
			t.Parallel()

			actual, err := testParser.Parse(bytes.NewReader(tt.input))
			if tt.expectedErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, actual, 1)

			actualCasted, ok := actual.(map[string]any)
			require.True(t, ok)
			data, ok := actualCasted[nonce]
			require.True(t, ok)
			require.Equal(t, tt.expected, data)
		})
	}
}

func readTestFile(t *testing.T, filepath string) []byte {
	b, err := os.ReadFile(filepath)
	require.NoError(t, err)
	return b
}
