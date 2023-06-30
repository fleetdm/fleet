package file

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidMacOSName(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		output bool
	}{
		{"valid", "filename.txt", true},
		{"spaces", "file name with spaces.txt", true},
		{"dashes", "file-name-with-dashes.txt", true},
		{"underscores", "file_underscored.txt", true},
		{"non-ASCII characters", "中文文件名.txt", true},

		{"colon", "file:name.txt", false},
		{"backslash", "file\\name.txt", false},
		{"asterisk", "file*name.txt", false},
		{"question mark", "file?name.txt", false},
		{"double quote", "file\"name.txt", false},
		{"less than", "file<name.txt", false},
		{"greater than", "file>name.txt", false},
		{"pipe", "file|name.txt", false},
		{"null character", "file\x00name.txt", false},
		{"empty", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.output, IsValidMacOSName(tc.input))
		})
	}
}
