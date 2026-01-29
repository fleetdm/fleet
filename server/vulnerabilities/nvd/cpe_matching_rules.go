package nvd

import (
	"fmt"
	"strconv"
	"strings"

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
		// When we're inventorying the Steam launcher for Dota, version recorded is 1.0,
		// which shows a bunch of false positive CVEs. See #34323.
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2020-7949": {},
				"CVE-2020-7950": {},
				"CVE-2020-7951": {},
				"CVE-2020-7952": {},
				"CVE-2020-9005": {},
			},
			IgnoreIf: func(cpeMeta *wfn.Attributes) bool {
				return cpeMeta.Vendor == "valvesoftware" && cpeMeta.Product == "dota_2" &&
					cpeMeta.TargetSW == "macos" && (cpeMeta.Version == "1\\.0" || cpeMeta.Version == "1\\.0\\.0")
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
		// CVE-2023-48795 in NVD incorrectly mentions PowerShell as vulnerable when the issue is actually with OpenSSH,
		// which is packaged separately. It also includes a bogus resolved-in version number. See #26073.
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2023-48795": {},
			},
			IgnoreIf: func(cpeMeta *wfn.Attributes) bool {
				return cpeMeta.Vendor == "microsoft" && cpeMeta.Product == "powershell"
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
		// CVE-2024-4030 and CVE-2024-6286 only target windows operating systems
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2024-4030": {},
				"CVE-2024-6286": {},
			},
			IgnoreIf: func(cpeMeta *wfn.Attributes) bool {
				return cpeMeta.TargetSW != "windows"
			},
		},
		// CVE-2024-12254 only targets Mac/Linux operating systems
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2024-12254": {},
			},
			IgnoreIf: func(cpeMeta *wfn.Attributes) bool {
				return cpeMeta.TargetSW == "windows"
			},
		},
		// CVE-2024-7006 only targets Linux operating systems (libtiff vulnerability)
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2024-7006": {},
			},
			IgnoreIf: func(cpeMeta *wfn.Attributes) bool {
				return cpeMeta.TargetSW != "linux"
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
		// Gitk and Git GUI CVEs should not match the base git package
		// These CVEs affect gitk/git-gui which is git-gui on Homebrew
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2025-27613": {}, // Gitk file creation/truncation via OS command injection
				"CVE-2025-27614": {}, // Gitk arbitrary command execution
				"CVE-2025-46835": {}, // Git GUI arbitrary file overwrite
			},
			IgnoreIf: func(cpeMeta *wfn.Attributes) bool {
				return cpeMeta.Vendor == "git" && cpeMeta.Product == "git"
			},
		},
		// CVE-2023-28205 WebKit vulnerability
		// Apple released fixes via:
		// - Safari 16.4.1 standalone update for Big Sur/Monterey (HT213722)
		// - macOS Ventura 13.3.1 system update (HT213721)
		//
		// - Safari 16.0-16.4.0 are vulnerable
		// - Safari < 16.0 not vulnerable
		// - macOS Ventura < 13.3.1 is vulnerable
		// - macOS < 13.0 ignore for macOS matches, no system-level fix, rely on Safari version matching
		CPEMatchingRule{
			CVEs: map[string]struct{}{
				"CVE-2023-28205": {},
			},
			IgnoreIf: func(cpeMeta *wfn.Attributes) bool {
				// For Safari CPE matches, only match versions 16.0-16.4.0
				if cpeMeta.Vendor == "apple" && cpeMeta.Product == "safari" {
					version := wfn.StripSlashes(cpeMeta.Version)
					parts := strings.Split(version, ".")

					if len(parts) > 0 {
						if majorVer, err := strconv.Atoi(parts[0]); err == nil {
							if majorVer < 16 {
								return true
							}
							if majorVer > 16 {
								return true
							}
						}
					}
				}

				// For macOS CPE matches, only match Ventura < 13.3.1
				if cpeMeta.Vendor == "apple" && cpeMeta.Product == "macos" {
					version := wfn.StripSlashes(cpeMeta.Version)
					parts := strings.Split(version, ".")

					if len(parts) > 0 {
						majorVer, err := strconv.Atoi(parts[0])
						if err != nil {
							return false
						}

						// Ignore non-Ventura
						if majorVer != 13 {
							return true
						}

						// For Ventura, check if >= 13.3.1
						if len(parts) >= 3 {
							minorVer, _ := strconv.Atoi(parts[1])
							patchVer, _ := strconv.Atoi(parts[2])
							if minorVer > 3 || (minorVer == 3 && patchVer >= 1) {
								return true
							}
						}
					}
				}

				return false
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
