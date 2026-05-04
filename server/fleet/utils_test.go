package fleet

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteExpiredLicenseBanner(t *testing.T) {
	var buf strings.Builder
	WriteExpiredLicenseBanner(&buf)
	out := buf.String()

	require.Contains(t, out, "Your license for Fleet Premium is about to expire")
	require.Contains(t, out, "https://fleetdm.com/learn-more-about/downgrading")
}

func TestVersionToSemvarVersion(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{"1", "1.0.0"},
		{"1.0", "1.0.0"},
		{"0.0.4", "0.0.4"},
		{"1.0.0", "1.0.0"},
		{"12.0.9", "12.0.9"},
		{"1.0.0-alpha", "1.0.0-alpha"},
		{"1.1.2+meta", "1.1.2+meta"},
		{"13.3.1 (a)", "13.3.1"},
	}

	for _, tt := range tests {
		result, err := VersionToSemverVersion(tt.version)
		require.NoError(t, err)
		require.Equal(t, tt.want, result.String())
	}
}
