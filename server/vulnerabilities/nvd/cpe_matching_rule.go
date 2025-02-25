package nvd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

// CPEMatchingRuleSpec allows you to match against a CPE. Version ranges are supported via SemVer constraints.
type CPEMatchingRuleSpec struct {
	Vendor   string // Software vendor.
	Product  string // Software product name.
	TargetSW string // Target software, this usually corresponds to the target OS.

	// Specifies a version constraint. See https://pkg.go.dev/github.com/Masterminds/semver@v1.5.0#hdr-Checking_Version_Constraints
	// for reference.
	SemVerConstraint string
}

func (rule CPEMatchingRuleSpec) getCPEMeta() *wfn.Attributes {
	return &wfn.Attributes{
		Vendor:   rule.Vendor,
		Product:  rule.Product,
		TargetSW: rule.TargetSW,
	}
}

// CPEMatchingRule allows you to express a matching rule based on some CPE properties, one or more
// CVEs and one or more SemVer constraint. This is used to 'fix' false positives resulting from bad
// data in the NVD dataset itself.
// For example: https://nvd.nist.gov/vuln/detail/CVE-2017-13797, one of the CPE entries specified is
// cpe:2.3:a:apple:icloud:*:*:*:*:*:*:*:* which will match with any iCloud installation, but the
// vulnerability in question only affects iCloud on Windows up to 7.0.x.
type CPEMatchingRule struct {
	// CPESpecs are the rules that will be used to match a CPE.
	CPESpecs []CPEMatchingRuleSpec
	// CVEs is the set of CVEs that this rule targets.
	CVEs map[string]struct{}
	// IgnoreAll will cause all CPEs to not match hence ignoring a CVE.
	IgnoreAll bool
	// IgnoreIf is a function that can determine if a CPE matching rule should be ignored or not.
	// If IgnoreIf is set, CPESpecs will not be evaluated.
	IgnoreIf func(cpeMeta *wfn.Attributes) bool
}

// CPEMatches returns true if the provided CPE matches the rule.
func (rule CPEMatchingRule) CPEMatches(cpeMeta *wfn.Attributes) bool {
	if cpeMeta == nil {
		return false
	}

	if rule.IgnoreAll {
		return false
	}

	if rule.IgnoreIf != nil {
		return !rule.IgnoreIf(cpeMeta)
	}

	ver, err := fleet.VersionToSemverVersion(wfn.StripSlashes(cpeMeta.Version))
	if err != nil {
		return false
	}

	for _, spec := range rule.CPESpecs {
		// The SemVer constraint is validated at instantiation time, so it should be ok to ignore the error.
		constraint, _ := semver.NewConstraint(spec.SemVerConstraint)
		if cpeMeta.MatchWithoutVersion(spec.getCPEMeta()) && constraint.Check(ver) {
			return true
		}
	}

	return false
}

// Validate validates the rule, returns an error if there's something wrong.
func (rule CPEMatchingRule) Validate() error {
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

	for _, spec := range rule.CPESpecs {
		// Validate CPE parts
		if err := validateCPEPart("Vendor", spec.Vendor); err != nil {
			return err
		}
		if err := validateCPEPart("Product", spec.Product); err != nil {
			return err
		}
		if err := validateCPEPart("TargetSW", spec.TargetSW); err != nil {
			return err
		}
		// Validate SemVerConstraint
		if _, err := semver.NewConstraint(spec.SemVerConstraint); err != nil {
			return err
		}
	}

	// Validate CVEs entries
	if len(rule.CVEs) == 0 {
		return errors.New("At least one CVE is required")
	}
	for cve := range rule.CVEs {
		if strings.TrimSpace(cve) == "" {
			return errors.New("CVE can't be empty")
		}
	}

	return nil
}
