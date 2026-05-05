package execuser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadEnvFromProcFile(t *testing.T) {
	dir := t.TempDir()
	environPath := filepath.Join(dir, "environ")

	testCases := []struct {
		name     string
		environ  string
		envVar   string
		expected string
	}{
		{
			name:     "DISPLAY found",
			environ:  "HOME=/home/foo\x00DISPLAY=:0\x00LANG=en_US.UTF-8",
			envVar:   "DISPLAY",
			expected: ":0",
		},
		{
			name:     "DISPLAY :1",
			environ:  "HOME=/home/foo\x00DISPLAY=:1\x00LANG=en_US.UTF-8",
			envVar:   "DISPLAY",
			expected: ":1",
		},
		{
			name:     "DISPLAY not present",
			environ:  "HOME=/home/foo\x00LANG=en_US.UTF-8",
			envVar:   "DISPLAY",
			expected: "",
		},
		{
			name:     "empty environ",
			environ:  "",
			envVar:   "DISPLAY",
			expected: "",
		},
		{
			name:     "does not match prefix substring",
			environ:  "DISPLAY_NUM=5\x00DISPLAY=:2",
			envVar:   "DISPLAY",
			expected: ":2",
		},
		{
			name:     "other env var",
			environ:  "HOME=/home/foo\x00DISPLAY=:0\x00WAYLAND_DISPLAY=wayland-0",
			envVar:   "WAYLAND_DISPLAY",
			expected: "wayland-0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(environPath, []byte(tc.environ), 0o644))
			result, err := readEnvFromProcFile(environPath, tc.envVar)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestReadEnvFromProcFileMissing(t *testing.T) {
	_, err := readEnvFromProcFile("/nonexistent/path/environ", "DISPLAY")
	require.Error(t, err)
}
