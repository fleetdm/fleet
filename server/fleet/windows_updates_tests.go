package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseKBId(t *testing.T) {
	testCases := []struct {
		input    string
		expected uint
		errors   bool
	}{
		{
			input:    "2022-04 Update for Windows 10 Version 21H2 for x64-based Systems based on (KB2267602) based on KB2267601 (KB5005463)",
			expected: 5005463,
			errors:   false,
		},
		{
			input:    "Security Intelligence Update for Microsoft Defender Antivirus - KB2267602 (Version 1.371.1239.0)",
			expected: 2267602,
			errors:   false,
		},
		{
			input:    "2022-04 Update for Windows 10 Version 21H2 for x64-based Systems (KB5005463)",
			expected: 5005463,
			errors:   false,
		},
		{
			input:    "2022-04 Update for Windows 10 Version 21H2 for x64-based Systems (KB-5005463)",
			expected: 0,
			errors:   true,
		},
		{
			input:    "2022-04 Update for Windows 10 Version 21H2 for x64-based Systems (KB0)",
			expected: 0,
			errors:   true,
		},
		{
			input:    "Some random string",
			expected: 0,
			errors:   true,
		},
	}

	for _, tCase := range testCases {
		actual, err := parseKBID(tCase.input)
		require.Equal(t, tCase.expected, actual)

		if !tCase.errors {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}
