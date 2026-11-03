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
