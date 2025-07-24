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
	// Sample JSON output from Get-BitLockerVolume | ConvertTo-Json
	jsonOutput := `
[
    {
        "MountPoint":  "C:",
        "KeyProtectorType":  3
    },
    {
        "MountPoint":  "C:",
        "KeyProtectorType":  1
    }
]
`
	table := &Table{
		logger: zerolog.New(zerolog.NewTestWriter(t)),
	}

	expected := []map[string]string{
		{
			"drive_letter":       "C:",
			"key_protector_type": "3",
		},
		{
			"drive_letter":       "C:",
			"key_protector_type": "1",
		},
	}

	results, err := table.parseOutput([]byte(jsonOutput))

	require.NoError(t, err)
	if !reflect.DeepEqual(results, expected) {
		t.Errorf("parseOutput() = %v, want %v", results, expected)
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
