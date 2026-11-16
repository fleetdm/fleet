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

func TestParseNonNilUUID(t *testing.T) {
	const canonical = "efdb13f9-44d6-4f99-a93f-08833fccef82"
	for _, tc := range []struct {
		name  string
		input string
		ok    bool
	}{
		{"canonical", canonical, true},
		{"upper case", "EFDB13F9-44D6-4F99-A93F-08833FCCEF82", true},
		{"braced", "{efdb13f9-44d6-4f99-a93f-08833fccef82}", true},
		{"all-zero UUID", "00000000-0000-0000-0000-000000000000", false},
		{"empty", "", false},
		{"not a UUID", "Default string", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, ok := ParseNonNilUUID(tc.input)
			require.Equal(t, tc.ok, ok)
			if tc.ok {
				require.Equal(t, canonical, parsed.String())
			}
		})
	}
}
