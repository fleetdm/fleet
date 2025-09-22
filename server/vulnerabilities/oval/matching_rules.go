package oval

import (
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
)

type SoftwareMatchingRule struct {
	Name       string
	VersionEnd string // Version where CVEs were resolved
	// TODO: OSVersion or FleetSoftware.Release
	// rpm packages specify a release but ubuntu dont
	// Maybe add release for rpm packages but make it nullable
	CVEs map[string]struct{} // Maybe just a slice?
	// TODO: IgnoreIf func(FleetSoftware ) bool
	MatchIf func(software fleet.Software) bool
}

type SoftwareMatchingRules []SoftwareMatchingRule

// GetKnownOVALBugRules returns a list of SoftwareMatchingRules used for
// ignoring false positives detected during the OVAL vuln. detection process.
func GetKnownOVALBugRules() (SoftwareMatchingRules, error) {
	rules := SoftwareMatchingRules{ // Would it be more efficient to use a map? It's a very small list of things
		{
			Name:       "microcode_ctl",
			VersionEnd: "2.1",
			CVEs: map[string]struct{}{
				"CVE-2022-21216": {}, // release: 53.1.fc37
				"CVE-2022-33196": {}, // release: 53.1.fc37
				"CVE-2022-41804": {}, // release: 55.1.fc38
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
func (rules SoftwareMatchingRules) MatchesAny(s fleet.Software, cve string) bool { //version string release string
	if strings.TrimSpace(s.Name) == "" {
		return false
	}
	if strings.TrimSpace(s.Version) == "" {
		// TODO: maybe log this
		return false
	}

	// MatchIf

	for _, r := range rules {
		if s.Name != r.Name {
			continue
		}
		// REMOVE COMMENT: if the version is >= VersionEnd
		// the CVEs have been fixed and it is false positive
		// so we want to return true
		// fmt.Println("Version compare: ", version, " , ", r.VersionEnd, " cve: ", cve)
		if r.MatchIf != nil && r.MatchIf(s) == false {
			continue
		}
		if nvd.SmartVerCmp(s.Version, r.VersionEnd) < 0 {
			continue // true positive
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
