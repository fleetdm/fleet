package dataflatten

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	iniTestFilePath = path.Join("testdata", "secdata.ini")
	iniTestFileLen  = 87
)

func TestIniToBool(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		in       string
		expected bool
		isBool   bool
	}{
		{
			in: "hello world",
		},
		{
			in:       "Yes",
			expected: true,
			isBool:   true,
		},
		{
			in:       "No",
			expected: false,
			isBool:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()

			asBool, ok := iniToBool(tt.in)
			if tt.isBool {
				require.True(t, ok)
				require.Equal(t, tt.expected, asBool)
			} else {
				require.False(t, ok)
			}
		})
	}

}

func TestIniFile(t *testing.T) {
	t.Parallel()

	rows, err := IniFile(iniTestFilePath)
	require.NoError(t, err)
	require.Len(t, rows, iniTestFileLen)
}

func TestIni(t *testing.T) {
	t.Parallel()

	fileBytes, err := os.ReadFile(iniTestFilePath)
	require.NoError(t, err)

	rows, err := Ini(fileBytes)
	require.NoError(t, err)
	require.Len(t, rows, iniTestFileLen)
}

func TestIniSecedit(t *testing.T) {
	t.Parallel()

	rows, err := IniFile(path.Join("testdata", "secdata.ini"))
	require.NoError(t, err)

	var tests = []struct {
		name     string
		expected Row
	}{
		{
			name:     "converted boolean",
			expected: Row{Path: []string{"Unicode", "Unicode"}, Value: "true"},
		},
		{
			name:     "string value",
			expected: Row{Path: []string{"System Access", "NewAdministratorName"}, Value: "Administrator"},
		},
		{
			// We're not casting this to false
			name:     "number value",
			expected: Row{Path: []string{"Event Audit", "AuditDSAccess"}, Value: "0"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Contains(t, rows, tt.expected)
		})
	}
}
