package nvd

import (
	"fmt"
)

type CPEMatchingRules []CPEMatchingRule

// GetCPEMatchingRules returns a list of CPEMatchingRules used for
// ignoring false positives detected during the NVD vuln. detection process.
func GetCPEMatchingRules() (CPEMatchingRules, error) {
	rules := CPEMatchingRules{
		CPEMatchingRule{
			CPESpecs: []CPEMatchingRuleSpec{
				{
					Vendor:           "apple",
					Product:          "icloud",
					TargetSW:         "windows",
					SemVerConstraint: "< 7.1",
				},
			},

			CVEs: map[string]struct{}{
				"CVE-2017-13797": {},
			},
		},
		CPEMatchingRule{
			CPESpecs: []CPEMatchingRuleSpec{
				{
					Vendor:           "apple",
					Product:          "icloud",
					TargetSW:         "windows",
					SemVerConstraint: "<= 6.1.1",
				},
			},

			CVEs: map[string]struct{}{
				"CVE-2016-4613": {},
				"CVE-2017-2383": {},
			},
		},
		CPEMatchingRule{
			CPESpecs: []CPEMatchingRuleSpec{
				{
					Vendor:           "apple",
					Product:          "icloud",
					TargetSW:         "windows",
					SemVerConstraint: "<= 6.1.0",
				},
			},

			CVEs: map[string]struct{}{
				"CVE-2017-2366": {},
			},
		},
		CPEMatchingRule{
			CPESpecs: []CPEMatchingRuleSpec{
				{
					Vendor:           "apple",
					Product:          "icloud",
					TargetSW:         "windows",
					SemVerConstraint: "<= 6.0.0",
				},
			},

			CVEs: map[string]struct{}{
				"CVE-2016-4613": {},
				"CVE-2016-7583": {},
			},
		},
		CPEMatchingRule{
			CPESpecs: []CPEMatchingRuleSpec{
				{
					Vendor:           "apple",
					Product:          "icloud",
					TargetSW:         "windows",
					SemVerConstraint: "<= 6.0.1",
				},
			},

			CVEs: map[string]struct{}{
				"CVE-2016-4692": {},
				"CVE-2016-4743": {},
				"CVE-2016-7578": {},
				"CVE-2016-7586": {},
				"CVE-2016-7587": {},
				"CVE-2016-7589": {},
				"CVE-2016-7592": {},
				"CVE-2016-7598": {},
				"CVE-2016-7599": {},
				"CVE-2016-7610": {},
				"CVE-2016-7611": {},
				"CVE-2016-7614": {},
				"CVE-2016-7632": {},
				"CVE-2016-7635": {},
				"CVE-2016-7639": {},
				"CVE-2016-7640": {},
				"CVE-2016-7641": {},
				"CVE-2016-7642": {},
				"CVE-2016-7645": {},
				"CVE-2016-7646": {},
				"CVE-2016-7648": {},
				"CVE-2016-7649": {},
				"CVE-2016-7652": {},
				"CVE-2016-7654": {},
				"CVE-2016-7656": {},
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

// FindMatch returns the first matching rule
func (rules CPEMatchingRules) FindMatch(cve string) (*CPEMatchingRule, bool) {
	for _, rule := range rules {
		if _, ok := rule.CVEs[cve]; ok {
			return &rule, true
		}
	}
	return nil, false
}
