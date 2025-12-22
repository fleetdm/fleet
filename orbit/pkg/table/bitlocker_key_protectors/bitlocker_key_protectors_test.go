//go:build windows
// +build windows

package bitlocker_key_protectors

import (
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"

	"github.com/rs/zerolog"
)

func TestTable_parseOutput(t *testing.T) {
	testCases := []struct {
		name     string
		input    []byte
		expected []map[string]string
	}{
		{
			name: "as array",
			input: []byte(`
[
    {
        "MountPoint":  "C:",
        "KeyProtectorType":  3
    },
    {
        "MountPoint":  "C:",
        "KeyProtectorType":  1
    }
]`),
			expected: []map[string]string{
				{
					"drive_letter":       "C:",
					"key_protector_type": "3",
				},
				{
					"drive_letter":       "C:",
					"key_protector_type": "1",
				},
			},
		},
		{
			name: "as a single object",
			input: []byte(`
    {
        "MountPoint":  "C:",
        "KeyProtectorType":  3
    }
`),
			expected: []map[string]string{
				{
					"drive_letter":       "C:",
					"key_protector_type": "3",
				},
			},
		},
	}
	table := &Table{
		logger: zerolog.New(zerolog.NewTestWriter(t)),
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := table.parseOutput(testCase.input)
			require.NoError(t, err)
			if !reflect.DeepEqual(results, testCase.expected) {
				t.Errorf("parseOutput() = %v, want %v", results, testCase.expected)
			}
		})

	}
}

func TestTable_parseOutput_InvalidJSON(t *testing.T) {
	table := &Table{
		logger: zerolog.New(zerolog.NewTestWriter(t)),
	}

	_, err := table.parseOutput([]byte(`invalid json`))
	require.Error(t, err)
}

func TestTable_parseOutput_EmptyInput(t *testing.T) {
	table := &Table{
		logger: zerolog.New(zerolog.NewTestWriter(t)),
	}

	testCases := [][]byte{
		[]byte(""),
		[]byte(`[]`),
	}

	for _, testCase := range testCases {
		results, err := table.parseOutput(testCase)
		require.NoError(t, err)
		require.Empty(t, results)
	}
}
