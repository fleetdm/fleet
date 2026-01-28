package oval

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/stretchr/testify/require"
)

func TestSoftwareMatchingRules(t *testing.T) {
	badRules := SoftwareMatchingRules{
		{
			Name:            "",
			VersionResolved: "",
			CVEs:            map[string]struct{}{},
		},
		{
			Name:            "  ",
			VersionResolved: "   ",
			CVEs: map[string]struct{}{
				"CVE-2024-42582": {},
			},
		},
		{
			Name:            "",
			VersionResolved: "1.0",
			CVEs: map[string]struct{}{
				"CVE-2024-42582": {},
			},
		},
	}

	for _, r := range badRules {
		err := r.Validate()
		require.Error(t, err)
	}

	rules, err := GetKnownOVALBugRules()
	require.NoError(t, err)

	for _, r := range rules {
		err := r.Validate()
		require.NoError(t, err)
	}

	s1 := softwareFixture{
		Name:    "microcode_ctl",
		Version: "2.1",
		Release: "70.fc42",
	}

	match := rules.MatchesAny(fleet.Software{Name: s1.Name, Version: s1.Version, Release: s1.Release}, "CVE-2025-20012")
	require.True(t, match)
	match = rules.MatchesAny(fleet.Software{Name: s1.Name, Version: "2.2", Release: s1.Release}, "CVE-2025-20012")
	require.True(t, match)
	match = rules.MatchesAny(fleet.Software{Name: s1.Name, Version: "2.0", Release: s1.Release}, "CVE-2025-20012")
	require.False(t, match)
	match = rules.MatchesAny(fleet.Software{Name: s1.Name, Version: "20250211", Release: "1.el9"}, "CVE-2024-23984")
	require.False(t, match)

	match = rules.MatchesAny(fleet.Software{Name: "  ", Version: s1.Version, Release: s1.Release}, "CVE-2025-20012")
	require.False(t, match)
	match = rules.MatchesAny(fleet.Software{Name: s1.Name, Version: "    ", Release: s1.Release}, "CVE-2025-20012")
	require.False(t, match)
	match = rules.MatchesAny(fleet.Software{Name: s1.Name, Version: s1.Version, Release: s1.Release}, "CVE-1111-11111")
	require.False(t, match)
	match = rules.MatchesAny(fleet.Software{Name: s1.Name, Version: s1.Version, Release: s1.Release}, "")
	require.False(t, match)

	rules = append(rules, SoftwareMatchingRule{
		Name:            "example",
		VersionResolved: "1.0",
		CVEs: map[string]struct{}{
			"CVE-1111-22222": {},
		},
		MatchIf: func(s fleet.Software) bool {
			return nvd.SmartVerCmp(s.Release, "53.1.fc37") >= 0
		},
	})

	match = rules.MatchesAny(fleet.Software{Name: "example", Version: "1.0", Release: "70.fc42"}, "CVE-1111-22222")
	require.True(t, match)
	match = rules.MatchesAny(fleet.Software{Name: "example", Version: "1.0", Release: "53.fc42"}, "CVE-1111-22222")
	require.False(t, match)
}
