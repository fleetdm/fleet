package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOperatingSystemIsWindows(t *testing.T) {
	testCases := []struct {
		platform  string
		isWindows bool
	}{
		{platform: "chrome"},
		{platform: "darwin"},
		{platform: "windows", isWindows: true},
	}

	for _, tc := range testCases {
		sut := OperatingSystem{Platform: tc.platform}
		require.Equal(t, tc.isWindows, sut.IsWindows())
	}
}

func TestOperatingSystemRequiresNudge(t *testing.T) {
	testCases := []struct {
		platform      string
		version       string
		requiresNudge bool
		parseError    bool
	}{
		{platform: "chrome"},
		{platform: "chrome", version: "12.1"},
		{platform: "chrome", version: "15"},
		{platform: "darwin", parseError: true},
		{platform: "darwin", version: "12.0.9", requiresNudge: true},
		{platform: "darwin", version: "11", requiresNudge: true},
		{platform: "darwin", version: "13.3.1 (a)", requiresNudge: true},
		{platform: "darwin", version: "13.4.1 (c)", requiresNudge: true},
		{platform: "darwin", version: "14.0"},
		{platform: "darwin", version: "14.3.2"},
		{platform: "darwin", version: "15.0.1"},
		{platform: "darwin", version: "15.0.1 (a)"},
		{platform: "windows"},
		{platform: "windows", version: "12.2"},
		{platform: "windows", version: "15.4"},
	}

	for _, tc := range testCases {
		os := OperatingSystem{Platform: tc.platform, Version: tc.version}
		req, err := os.RequiresNudge()
		if tc.parseError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, tc.requiresNudge, req)
	}
}
