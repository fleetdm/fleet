package nvd

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
	"github.com/stretchr/testify/require"
)

func TestCPEMatchVersionRangeTrailingZeros(t *testing.T) {
	// NVD writes Firefox ranges with three version parts, while Firefox reports two.
	matcher, err := cpeMatcher("CVE-2026-100756", &schema.NVDCVEFeedJSON10DefCPEMatch{
		Cpe23Uri:              "cpe:2.3:a:mozilla:firefox:*:*:*:*:*:*:*:*",
		VersionStartIncluding: "154.0.0",
		VersionEndExcluding:   "157.0.0",
		Vulnerable:            true,
	})
	require.NoError(t, err)

	cases := []struct {
		version    string
		vulnerable bool
	}{
		{"153.0.4", false},
		{"154.0", true},
		{"154.0.0", true},
		{"156.0.1", true},
		{"157.0", false},
		{"157.0.0", false},
		{"157.0.0.0", false},
		{"157.0.1", false},
	}
	for _, c := range cases {
		t.Run(c.version, func(t *testing.T) {
			attr, err := wfn.Parse("cpe:2.3:a:mozilla:firefox:" + c.version + ":*:*:*:*:windows:*:*")
			require.NoError(t, err)
			matches := matcher.Match([]*wfn.Attributes{attr}, false)
			require.Equal(t, c.vulnerable, len(matches) == 1)
		})
	}
}
