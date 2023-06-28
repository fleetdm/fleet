package nvd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/facebookincubator/nvdtools/wfn"
)

// CPEProcessingRuleAction what to do if the rules matches
type CPEProcessingRuleAction struct {
	Skip bool `json:"skip"`
}

// CPEProcessingRule specifies an action that will be executed if the rule matches a CPE and a CVE.
type CPEProcessingRule struct {
	Vendor   string `json:"vendor"`    // Software vendor.
	Product  string `json:"product"`   // Software product name.
	TargetSW string `json:"target_sw"` // Target software, this usually corresponds to the target OS.

	// Specifies a version constraint. See https://pkg.go.dev/github.com/Masterminds/semver@v1.5.0#hdr-Checking_Version_Constraints
	// for reference.
	SemVerConstraint string `json:"sem_ver_constraint"`

	// Set of CVEs that this rule targets
	CVEs []string `json:"cves"`

	// What to do if the rule matches
	Action CPEProcessingRuleAction `json:"action"`
}

// getCPEMeta maps the processing rule to its CPE attributes
func (rule CPEProcessingRule) getCPEMeta() *wfn.Attributes {
	return &wfn.Attributes{
		Vendor:   rule.Vendor,
		Product:  rule.Product,
		TargetSW: rule.TargetSW,
	}
}

// containsCVE returns whether the rule contains the cve
func (rule CPEProcessingRule) containsCVE(cve string) bool {
	for _, rCVE := range rule.CVEs {
		if rCVE == cve {
			return true
		}
	}
	return false
}

// Matches returns true if both the provided CPE and CVE match the rule.
func (rule CPEProcessingRule) Matches(cpeMeta *wfn.Attributes, cve string) bool {
	if cpeMeta == nil || !cpeMeta.MatchWithoutVersion(rule.getCPEMeta()) {
		return false
	}

	if ok := rule.containsCVE(cve); !ok {
		return false
	}

	// The SemVer constraint is validated at instantiation time, so it should be ok to ignore the error.
	constraint, _ := semver.NewConstraint(rule.SemVerConstraint)

	ver, err := semver.NewVersion(wfn.StripSlashes(cpeMeta.Version))
	if err != nil {
		return false
	}
	return constraint.Check(ver)
}

// Validate validates the rule, returns an error if there's something wrong.
func (rule CPEProcessingRule) Validate() error {
	validateCPEPart := func(errPrefix, val string) error {
		switch strings.TrimSpace(val) {
		case "":
			return fmt.Errorf("%s can't be empty", errPrefix)
		case "*":
			return fmt.Errorf("%s can't be 'ANY'", errPrefix)
		case "-":
			return fmt.Errorf("%s can't be 'NA'", errPrefix)
		default:
			return nil
		}
	}

	// Validate CPE parts
	if err := validateCPEPart("Vendor", rule.Vendor); err != nil {
		return err
	}
	if err := validateCPEPart("Product", rule.Product); err != nil {
		return err
	}
	if err := validateCPEPart("TargetSW", rule.TargetSW); err != nil {
		return err
	}

	// Validate SemVerConstraint
	if _, err := semver.NewConstraint(rule.SemVerConstraint); err != nil {
		return err
	}

	// Validate CVEs have no dups
	if len(rule.CVEs) == 0 {
		return errors.New("At least one CVE is required")
	}
	cveMap := make(map[string]bool, len(rule.CVEs))
	for _, cve := range rule.CVEs {
		if strings.TrimSpace(cve) == "" {
			return fmt.Errorf("CVE can't be empty")
		}
		if isDup := cveMap[cve]; isDup {
			return fmt.Errorf("duplicated CVE '%s'", cve)
		}
		cveMap[cve] = true
	}

	return nil
}
