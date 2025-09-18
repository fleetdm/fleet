package oval

import (
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
)

type SoftwareMatchingRule struct {
	Name       string
	VersionEnd string // Version where CVEs were resolved
	// TODO: OSVersion or FleetSoftware.Release
	// rpm packages specify a release but ubuntu dont
	// Maybe add release for rpm packages but make it nullable
	CVEs      map[string]struct{} // Maybe just a slice?
	IgnoreAll bool
	// TODO: IgnoreIf func(FleetSoftware ) bool
}

type SoftwareMatchingRules []SoftwareMatchingRule

func GetKnownOVALBugRules() (SoftwareMatchingRules, error) {
	rules := SoftwareMatchingRules{ // Would it be more efficient to use a map? It's a very small list of things
		{
			Name:       "microcode_ctl",
			VersionEnd: "2.1",
			CVEs: map[string]struct{}{
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
			},
			IgnoreAll: true,
		},
		{
			Name:       "shim-x64",
			VersionEnd: "15.8",
			CVEs: map[string]struct{}{
				"CVE-2023-40546": {},
				"CVE-2023-40547": {},
				"CVE-2023-40548": {},
				"CVE-2023-40549": {},
				"CVE-2023-40550": {},
				"CVE-2023-40551": {},
			},
			IgnoreAll: true,
		},
	}

	for i, rule := range rules {
		if err := rule.Validate(); err != nil {
			return nil, fmt.Errorf("invalid rule %d: %w", i, err)
		}
	}

	return rules, nil
}

// Returns true if the software and cve have a matching ignore rule
func (rules SoftwareMatchingRules) MatchesAny(name string, version string, cve string) bool { //version string release string
	if strings.TrimSpace(name) == "" {
		return false
	}
	if strings.TrimSpace(version) == "" {
		// TODO: maybe log this
		return false
	}

	for _, r := range rules {
		if name != r.Name {
			continue
		}
		// REMOVE COMMENT: if the version is >= VersionEnd
		// the CVEs have been fixed and it is false positive
		// so we want to return true
		fmt.Println("Version compare: ", version, " , ", r.VersionEnd)
		if nvd.SmartVerCmp(version, r.VersionEnd) > 0 {
			continue
		}
		if _, found := r.CVEs[cve]; found {
			return true
		}
	}
	return false
}

// Use in testing not runtime
func (rule SoftwareMatchingRule) Validate() error {
	if strings.TrimSpace(rule.Name) == "" {
		return fmt.Errorf("Name can't be empty")
	}
	if strings.TrimSpace(rule.VersionEnd) == "" {
		return fmt.Errorf("Version can't be empty")
	}
	for cve := range rule.CVEs {
		if strings.TrimSpace(cve) == "" {
			return fmt.Errorf("CVE can't be empty")
		}
	}
	return nil
}
