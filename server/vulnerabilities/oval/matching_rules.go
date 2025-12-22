package oval

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
)

type SoftwareMatchingRule struct {
	Name            string
	VersionResolved string
	CVEs            map[string]struct{}
	MatchIf         func(software fleet.Software) bool
}

type SoftwareMatchingRules []SoftwareMatchingRule

// GetKnownOVALBugRules returns a list of SoftwareMatchingRules used for
// ignoring false positives detected during the OVAL vuln. detection process.
func GetKnownOVALBugRules() (SoftwareMatchingRules, error) {
	rules := SoftwareMatchingRules{
		// OVAL source only lists date versions of microcode_ctl
		// while the fedora package uses semantic version. Causing
		// it to match 2.1 < 20250211
		{
			Name:            "microcode_ctl",
			VersionResolved: "2.1",
			CVEs: map[string]struct{}{
				"CVE-2022-21216": {},
				"CVE-2022-33196": {},
			},
			MatchIf: func(s fleet.Software) bool {
				return nvd.SmartVerCmp(s.Release, "53.1.fc37") >= 0
			},
		},
		{
			Name:            "microcode_ctl",
			VersionResolved: "2.1",
			CVEs: map[string]struct{}{
				"CVE-2022-41804": {},
			},
			MatchIf: func(s fleet.Software) bool {
				return nvd.SmartVerCmp(s.Release, "55.1.fc38") >= 0
			},
		},
		{
			Name:            "microcode_ctl",
			VersionResolved: "2.1",
			CVEs: map[string]struct{}{
				"CVE-2023-22655": {},
				"CVE-2023-28746": {},
				"CVE-2023-34440": {},
				"CVE-2023-38575": {},
				"CVE-2023-39368": {},
				"CVE-2023-43490": {},
				"CVE-2023-43758": {},
				"CVE-2023-45733": {},
				"CVE-2023-46103": {},
				"CVE-2024-24582": {},
				"CVE-2024-28047": {},
				"CVE-2024-28127": {},
				"CVE-2024-28956": {},
				"CVE-2024-29214": {},
				"CVE-2024-31157": {},
				"CVE-2024-39279": {},
				"CVE-2024-43420": {},
				"CVE-2024-45332": {},
				"CVE-2025-20012": {},
				"CVE-2025-20623": {},
				"CVE-2025-24495": {},
			},
			MatchIf: func(s fleet.Software) bool {
				return nvd.SmartVerCmp(s.Release, "70.fc42") >= 0
			},
		},
		{
			Name:            "shim-x64",
			VersionResolved: "15.8",
			CVEs: map[string]struct{}{
				"CVE-2023-40546": {},
				"CVE-2023-40547": {},
				"CVE-2023-40548": {},
				"CVE-2023-40549": {},
				"CVE-2023-40550": {},
				"CVE-2023-40551": {},
			},
		},
	}

	for i, rule := range rules {
		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("invalid rule %d: %w", i, err)
		}
	}

	return rules, nil
}

// Returns true if the software, cve pair have a matching ignore rule
func (rules SoftwareMatchingRules) MatchesAny(s fleet.Software, cve string) bool {
	if strings.TrimSpace(s.Name) == "" {
		return false
	}
	if strings.TrimSpace(s.Version) == "" {
		return false
	}

	for _, r := range rules {
		if s.Name != r.Name {
			continue
		}
		if r.MatchIf != nil && !r.MatchIf(s) {
			continue
		}
		if nvd.SmartVerCmp(s.Version, r.VersionResolved) < 0 {
			continue // true positive
		}
		if _, found := r.CVEs[cve]; found {
			return true
		}
	}
	return false
}

func (rule SoftwareMatchingRule) Validate() error {
	if strings.TrimSpace(rule.Name) == "" {
		return errors.New("Name can't be empty")
	}
	if strings.TrimSpace(rule.VersionResolved) == "" {
		return errors.New("Version can't be empty")
	}
	for cve := range rule.CVEs {
		if strings.TrimSpace(cve) == "" {
			return errors.New("CVE can't be empty")
		}
	}
	return nil
}
