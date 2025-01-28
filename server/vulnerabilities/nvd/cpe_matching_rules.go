package nvd

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

type CPEMatchingRules []CPEMatchingRule

// GetKnownNVDBugRules returns a list of CPEMatchingRules used for
// ignoring false positives detected during the NVD vuln. detection process.
func GetKnownNVDBugRules() (CPEMatchingRules, error) {
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
		// The NVD dataset contains an invalid rule for CVE-2020-10146 that matches all versions of
		// Microsoft Teams.
		//
		//	"cve" : {
		//		"data_type" : "CVE",
		//		"data_format" : "MITRE",
		//		"data_version" : "4.0",
		//		"CVE_data_meta" : {
		// 			"ID" : "CVE-2020-10146",
		// 			"ASSIGNER" : "cert@cert.org"
		// 		},
		//	[...]
		//	"configurations" : {
		//		"CVE_data_version" : "4.0",
		//		"nodes" : [ {
		//			"operator" : "OR",
		//			"children" : [ ],
		//			"cpe_match" : [ {
		//				"vulnerable" : true,
		//				"cpe23Uri" : "cpe:2.3:a:microsoft:teams:*:*:*:*:*:*:*:*", <<<<<<
		//				"versionEndExcluding" : "2020-10-29",
		//				"cpe_name" : [ ]
		//			} ]
		//		} ]
		//	},
		//
		// Such CVE corresponds to a vulnerability on Microsoft's online service
		// that has been patched since October 2020.
		CPEMatchingRule{
			IgnoreAll: true,
			CVEs: map[string]struct{}{
				"CVE-2020-10146": {},
			},
		},
		// #9835 Python expat 2.1.0 CVE recommends rejecting the report, no CVSS score, broad CPE criteria
		CPEMatchingRule{
			IgnoreAll: true,
			CVEs: map[string]struct{}{
				"CVE-2013-0340": {},
			},
		},
		// CVE-2022-42919 only affects Python on Linux but the NVD dataset doesn't set target_sw=linux.
		// For instance, here's an invalid CPE sample from the NVD dataset from this vulnerability as of Oct 13th 2023:
		// `cpe:2.3:a:python:python:3.7.3:-:*:*:*:*:*:*`.
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2022-42919": {},
			},
			CPESpecs: []CPEMatchingRuleSpec{
				{
					Vendor:           "python",
					Product:          "python",
					TargetSW:         "linux",
					SemVerConstraint: ">= 3.9.0, < 3.9.16",
				},
				{
					Vendor:           "python",
					Product:          "python",
					TargetSW:         "linux",
					SemVerConstraint: ">= 3.10.0, < 3.10.9",
				},
				{
					Vendor:           "python",
					Product:          "python",
					TargetSW:         "linux",
					SemVerConstraint: ">= 3.8.3, <= 3.8.15",
				},
				{
					Vendor:           "python",
					Product:          "python",
					TargetSW:         "linux",
					SemVerConstraint: ">= 3.7.3, <= 3.7.15",
				},
			},
		},
		// These vulnerabilities in the MongoDB client incorrectly match
		// the VS Code extension.
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2012-6619": {},
				"CVE-2013-1892": {},
				"CVE-2013-2132": {},
				"CVE-2015-1609": {},
				"CVE-2016-6494": {},
			},
			IgnoreIf: func(cpeMeta *wfn.Attributes) bool {
				return cpeMeta.TargetSW == "visual_studio_code"
			},
		},
		// Issue #18733 incorrect CPEs that should be matching
		// visual studio code extensions
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2021-28967": {},
				"CVE-2020-1192":  {},
				"CVE-2020-1171":  {},
			},
			IgnoreIf: func(cpeMeta *wfn.Attributes) bool {
				return cpeMeta.Product == "visual_studio_code" && cpeMeta.TargetSW == wfn.Any
			},
		},
		// Old macos CPEs without version constraints that should be ignored
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2001-0102": {},
				"CVE-1999-0590": {},
				"CVE-1999-0524": {},
			},
			IgnoreAll: true,
		},
		// Windows OS vulnerabilities without version constraints that should be ignored
		// TODO(tim): This rule is too specific and should be generalized to ignore all
		// Windows OS vulnerabilities in NVD
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2010-3143":  {},
				"CVE-2011-5049":  {},
				"CVE-2012-2972":  {},
				"CVE-2018-0598":  {},
				"CVE-2010-3888":  {},
				"CVE-2010-3139":  {},
				"CVE-2021-36958": {},
				"CVE-2008-6194":  {},
				"CVE-2010-2157":  {},
				"CVE-2011-3389":  {},
				"CVE-2012-2971":  {},
				"CVE-2018-0599":  {},
				"CVE-2010-3889":  {},
				"CVE-2011-0638":  {},
			},
			IgnoreAll: true,
		},
		// CVE-2024-4030 only targets windows operating systems
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2024-4030": {},
			},
			IgnoreIf: func(cpeMeta *wfn.Attributes) bool {
				return cpeMeta.TargetSW != "windows"
			},
		},
		// these CVEs only target iOS, and we don't yet support iOS vuln scanning (and can't tell iOS/Mac CPEs apart yet)
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2024-10004": {},
				"CVE-2024-10327": {}, // also missing a CPE as of 2025-01-01
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

// FindMatch returns the first matching rule
func (rules CPEMatchingRules) FindMatch(cve string) (*CPEMatchingRule, bool) {
	for _, rule := range rules {
		if _, ok := rule.CVEs[cve]; ok {
			return &rule, true
		}
	}
	return nil, false
}
