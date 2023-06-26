package nvd

import (
	"github.com/Masterminds/semver"
	"github.com/facebookincubator/nvdtools/wfn"
)

// CPEProcessingRuleAction what to do if the rules matches
type CPEProcessingRuleAction struct {
	Skip bool `json:"skip"`
}

// versionRange represents an inclusive version range
type versionRange struct {
	lower *semver.Version
	upper *semver.Version
}

// CPEProcessingRule specifies an action that needs to take place if both
// the CVE and the CPE attributes match
type CPEProcessingRule struct {
	Vendor      string                  `json:"vendor"`
	Product     string                  `json:"product"`
	TargetSW    string                  `json:"target_sw"`
	SemVerLower string                  `json:"sem_ver_lower"`
	SemVerUpper string                  `json:"sem_ver_upper"`
	CVE         string                  `json:"cve"`
	Action      CPEProcessingRuleAction `json:"action"`
}

type CPEProcessingRules []CPEProcessingRule

// getCPEAttrs maps the processing rule to its CPE attributes
func (rule CPEProcessingRule) getCPEAttrs() *wfn.Attributes {
	return &wfn.Attributes{
		Vendor:   rule.Vendor,
		Product:  rule.Product,
		TargetSW: rule.TargetSW,
	}
}

// getVerRange returns a inclusive version range based
func (rule CPEProcessingRule) getVerRange() (versionRange, error) {
	lower, err := semver.NewVersion(rule.SemVerLower)
	if err != nil {
		return versionRange{}, err
	}

	upper, err := semver.NewVersion(rule.SemVerUpper)
	if err != nil {
		return versionRange{}, err
	}

	return versionRange{
		lower: lower,
		upper: upper,
	}, nil
}

// Matches returns true if both the provided software and CVE match the rule.
func (rule CPEProcessingRule) Matches(software softwareCPEWithNVDMeta, cve string) bool {
	if cve != rule.CVE {
		return false
	}

	if software.meta == nil || !software.meta.MatchWithoutVersion(rule.getCPEAttrs()) {
		return false
	}

	// Check if versions match
	verRange, err := rule.getVerRange()
	if err != nil {
	}

	softwareVer, _ := semver.NewVersion(software.meta.Version)
	return (verRange.lower.GreaterThan(softwareVer) || verRange.lower.Equal(softwareVer)) &&
		(verRange.upper.LessThan(softwareVer) || verRange.lower.Equal(softwareVer))
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
