package nvd

import (
	"fmt"

	"github.com/facebookincubator/nvdtools/wfn"
)

type CPEProcessingRules []CPEProcessingRule

func GetCPEProcessingRules() (CPEProcessingRules, error) {
	// TODO: Move this to a metadata file?
	rules := CPEProcessingRules{
		CPEProcessingRule{
			Vendor:           "apple",
			Product:          "icloud",
			TargetSW:         "windows",
			SemVerConstraint: ">= 7.1.x",
			CVEs:             []string{"CVE-2017-13797"},
			Action:           CPEProcessingRuleAction{Skip: true},
		},
		CPEProcessingRule{
			Vendor:           "apple",
			Product:          "icloud",
			TargetSW:         "windows",
			SemVerConstraint: ">= 6.2.x",
			CVEs:             []string{"CVE-2017-2383"},
			Action:           CPEProcessingRuleAction{Skip: true},
		},
		CPEProcessingRule{
			Vendor:           "apple",
			Product:          "icloud",
			TargetSW:         "windows",
			SemVerConstraint: "> 6.1.1",
			CVEs:             []string{"CVE-2017-2366"},
			Action:           CPEProcessingRuleAction{Skip: true},
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
func (rules CPEProcessingRules) FindMatch(cpeMeta *wfn.Attributes, cve string) (*CPEProcessingRule, bool) {
	for _, rule := range rules {
		if rule.Matches(cpeMeta, cve) {
			return &rule, true
		}
	}
	return nil, false
}
