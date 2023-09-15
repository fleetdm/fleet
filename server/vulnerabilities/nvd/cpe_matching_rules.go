package nvd

import (
	"fmt"
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
