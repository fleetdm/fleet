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

			CVEs: map[string]bool{
				"CVE-2017-13797": true,
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

			CVEs: map[string]bool{
				"CVE-2016-4613": true,
				"CVE-2017-2383": true,
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

			CVEs: map[string]bool{
				"CVE-2017-2366": true,
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

			CVEs: map[string]bool{
				"CVE-2016-4613": true,
				"CVE-2016-7583": true,
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

			CVEs: map[string]bool{
				"CVE-2016-4692": true,
				"CVE-2016-4743": true,
				"CVE-2016-7578": true,
				"CVE-2016-7586": true,
				"CVE-2016-7587": true,
				"CVE-2016-7589": true,
				"CVE-2016-7592": true,
				"CVE-2016-7598": true,
				"CVE-2016-7599": true,
				"CVE-2016-7610": true,
				"CVE-2016-7611": true,
				"CVE-2016-7614": true,
				"CVE-2016-7632": true,
				"CVE-2016-7635": true,
				"CVE-2016-7639": true,
				"CVE-2016-7640": true,
				"CVE-2016-7641": true,
				"CVE-2016-7642": true,
				"CVE-2016-7645": true,
				"CVE-2016-7646": true,
				"CVE-2016-7648": true,
				"CVE-2016-7649": true,
				"CVE-2016-7652": true,
				"CVE-2016-7654": true,
				"CVE-2016-7656": true,
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
		if rule.CVEs[cve] {
			return &rule, true
		}
	}
	return nil, false
}
