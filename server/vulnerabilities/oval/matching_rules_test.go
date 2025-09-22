package oval

import (
	"fmt"
	"testing"

	// "github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/stretchr/testify/require"
)

// - All version variations (smaller, equal, larger, different looking)
// - All name variations (empty, lowercase/uppercase)
// - Bad rules (no name, no version, no cve's)
// - Benchmark (high num. of hosts)

func TestSoftwareMatchingRules(t *testing.T) {
	fmt.Println("Hello world")

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
		// Release: "70.fc42",
	}

	match := rules.MatchesAny(fleet.Software{Name: s1.Name, Version: s1.Version}, "CVE-2025-20012")
	require.True(t, match)
	match = rules.MatchesAny(fleet.Software{Name: s1.Name, Version: "2.2"}, "CVE-2025-20012")
	require.True(t, match)
	match = rules.MatchesAny(fleet.Software{Name: s1.Name, Version: "2.0"}, "CVE-2025-20012")
	require.False(t, match)
	match = rules.MatchesAny(fleet.Software{Name: s1.Name, Version: "20220207"}, "CVE-2024-23984")
	require.False(t, match)

	match = rules.MatchesAny(fleet.Software{Name: "  ", Version: s1.Version}, "CVE-2025-20012")
	require.False(t, match)
	match = rules.MatchesAny(fleet.Software{Name: s1.Name, Version: "    "}, "CVE-2025-20012")
	require.False(t, match)
	match = rules.MatchesAny(fleet.Software{Name: s1.Name, Version: s1.Version}, "CVE-1111-11111")
	require.False(t, match)
	match = rules.MatchesAny(fleet.Software{Name: s1.Name, Version: s1.Version}, "")
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

	// Test with ../rhel/software
	// so we need to make rules for software that is there...
	otherRules := SoftwareMatchingRules{
		{
			Name:            "rsyslog-udpspoof",
			VersionResolved: "8.2102.0",
			CVEs: map[string]struct{}{
				"CVE-2022-24903": {},
			},
		},
		{
			Name:            "java-11-openjdk-static-libs-slowdebug",
			VersionResolved: "11.0.15.0.10",
			CVEs: map[string]struct{}{
				"CVE-2022-21426": {},
			},
		},
		{
			Name:            "thunderbird",
			VersionResolved: "91.9.0",
			CVEs: map[string]struct{}{
				"CVE-2022-29917": {},
			},
		},
	}

	for _, r := range otherRules {
		err := r.Validate()
		require.NoError(t, err)
	}

	// rsyslog-udpspoof CVE-2022-24903          less than|0:8.2102.0-101.el9_0.1
	// java-11-openjdk-static-libs-slowdebug CVE-2022-21426 less than|1:11.0.15.0.10-1.el9_0
	// thunderbird CVE-2022-29917               less than|0:91.9.0-3.el9_0

	// Need some ubuntu programs
}
