package nvd

import (
	"github.com/facebookincubator/nvdtools/wfn"
)

// CPEProcessingRuleAction what to do if the rules match
type CPEProcessingRuleAction struct {
	Skip bool `json:"skip"`
}

// CPEProcessingRuleAttributes CPE attributes
type CPEProcessingRuleAttributes struct {
	Vendor   string `json:"vendor"`
	Product  string `json:"product"`
	Version  string `json:"version"`
	TargetSW string `json:"target_sw"`
}

// CPEProcessingRule specifies something we want to do post processing
type CPEProcessingRule struct {
	Attributes CPEProcessingRuleAttributes `json:"cpe_attributes"`
	CVE        string                      `json:"cve"`
	Action     CPEProcessingRuleAction     `json:"action"`
}

type CPEProcessingRules []CPEProcessingRule

// WfnAttrs Maps the processing rule CPE attributes to a wfn Attributes
func (rule CPEProcessingRule) CPEAttrs() *wfn.Attributes {
	return &wfn.Attributes{
		Vendor:   rule.Attributes.Vendor,
		Product:  rule.Attributes.Product,
		TargetSW: rule.Attributes.TargetSW,
	}
}

// Matches Returns true if both the CPE and CVE match the rule
func (rule CPEProcessingRule) Matches(software softwareCPEWithNVDMeta, cve string) bool {
	if cve != rule.CVE {
		return false
	}

	if software.meta == nil || !software.meta.MatchWithoutVersion(rule.CPEAttrs()) {
		return false
	}

	// Check if versions match
	return true
}

// FindMatch returns the first matching rule
func (rules CPEProcessingRules) FindMatch(software softwareCPEWithNVDMeta, cve string) (CPEProcessingRule, bool) {
	for _, rule := range rules {
		if rule.Matches(software, cve) {
			return rule, true
		}
	}
	return CPEProcessingRule{}, false
}
