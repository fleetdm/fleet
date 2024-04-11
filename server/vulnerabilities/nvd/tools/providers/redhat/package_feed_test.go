// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redhat

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// package name -> list of CVEs
type summary map[string][]string

func feedSummary(feed packageFeed) summary {
	s := make(summary)
	for pkg, cves := range feed {
		var cveNames []string
		for _, cve := range cves {
			cveNames = append(cveNames, cve.Name)
		}
		s[pkg] = cveNames
	}
	return s

}

func testFeed(t *testing.T, cves ...string) *Feed {
	feedJSON := "{" + strings.Join(cves, ",") + "}"
	feed, err := loadFeed(strings.NewReader(feedJSON))
	if err != nil {
		t.Fatalf("failed to parse JSON: %v: %s", err, feedJSON)
	}
	return feed
}

const CVEMultiplePackages = `
"CVE-2020-7211": {
	"name": "CVE-2020-7211",
	"threat_severity": "Low",
	"public_date": "2019-12-30T00:00:00Z",
	"bugzilla": {
		"description": "CVE-2020-7211 QEMU: Slirp: potential directory traversal using relative paths via tftp server on Windows host",
		"id": "1792130",
		"url": "https://bugzilla.redhat.com/show_bug.cgi?id=1792130"
	},
	"CVSS3": {
		"cvss3_base_score": "3.8",
		"cvss3_scoring_vector": "CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:C/C:L/I:N/A:N",
		"status": "draft"
	},
	"cwe": "CWE-22",
	"details": [
		"tftp.c in libslirp 4.1.0, as used in QEMU 4.2.0, does not prevent ..\\ directory traversal on Windows.",
		"A potential directory traversal issue was found in the tftp server of the SLiRP user-mode networking implementation used by QEMU. It could occur on a Windows host, as it allows the use of both forward ('/') and backward slash('\\') tokens as separators in a file path. A user able to access the tftp server could use this flaw to access undue files by using relative paths."
	],
	"references": [
		"https://www.voidsecurity.in/2019/01/virtualbox-tftp-server-pxe-boot.html"
	],
	"acknowledgement": "Red Hat would like to thank Reno Robert for reporting this issue.",
	"package_state": [
		{
			"product_name": "Red Hat Enterprise Linux 7",
			"fix_state": "Not affected",
			"package_name": "qemu-kvm-ma",
			"cpe": "cpe:/o:redhat:enterprise_linux:7"
		},
		{
			"product_name": "Red Hat Enterprise Linux 7",
			"fix_state": "Not affected",
			"package_name": "qemu-kvm",
			"cpe": "cpe:/o:redhat:enterprise_linux:7"
		},
		{
			"product_name": "Red Hat Enterprise Linux 7",
			"fix_state": "Not affected",
			"package_name": "qemu-kvm-rhev",
			"cpe": "cpe:/o:redhat:enterprise_linux:7"
		},
		{
			"product_name": "Red Hat Enterprise Linux 7",
			"fix_state": "Not affected",
			"package_name": "slirp4netns",
			"cpe": "cpe:/o:redhat:enterprise_linux:7"
		}
	]
}
`

const CVENoPackageState = `
"CVE-2020-6377": {
	"name": "CVE-2020-6377",
	"threat_severity": "Important",
	"public_date": "2020-01-07T00:00:00Z",
	"bugzilla": {
		"description": "CVE-2020-6377 chromium-browser: Use after free in audio",
		"id": "1789441",
		"url": "https://bugzilla.redhat.com/show_bug.cgi?id=1789441"
	},
	"CVSS3": {
		"cvss3_base_score": "8.8",
		"cvss3_scoring_vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H",
		"status": "verified"
	},
	"cwe": "CWE-416",
	"details": [
		"Use after free in audio in Google Chrome prior to 79.0.3945.117 allowed a remote attacker to potentially exploit heap corruption via a crafted HTML page."
	],
	"references": [
		"https://chromereleases.googleblog.com/2020/01/stable-channel-update-for-desktop.html"
	],
	"upstream_fix": "chromium-browser 79.0.3945.117",
	"affected_release": [
		{
			"product_name": "Red Hat Enterprise Linux 6 Supplementary",
			"release_date": "2020-01-13T00:00:00Z",
			"advisory": "RHSA-2020:0084",
			"package": "chromium-browser-79.0.3945.117-1.el6_10",
			"cpe": "cpe:/a:redhat:rhel_extras:6"
		}
	],
	"package_state": null
}
`

const CVEAffectedReleaseAndPackageState = `
"CVE-2019-11745": {
	"name": "CVE-2019-11745",
	"threat_severity": "Important",
	"public_date": "2019-11-21T00:00:00Z",
	"bugzilla": {
		"description": "CVE-2019-11745 nss: Out-of-bounds write when passing an output buffer smaller than the block size to NSC_EncryptUpdate",
		"id": "1774831",
		"url": "https://bugzilla.redhat.com/show_bug.cgi?id=1774831"
	},
	"CVSS3": {
		"cvss3_base_score": "8.1",
		"cvss3_scoring_vector": "CVSS:3.0/AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H",
		"status": "verified"
	},
	"cwe": "CWE-787",
	"details": [
		"When encrypting with a block cipher, if a call to NSC_EncryptUpdate was made with data smaller than the block size, a small out of bounds write could occur. This could have caused heap corruption and a potentially exploitable crash. This vulnerability affects Thunderbird < 68.3, Firefox ESR < 68.3, and Firefox < 71.",
		"A heap-based buffer overflow was found in the NSC_EncryptUpdate() function in Mozilla nss. A remote attacker could trigger this flaw via SRTP encrypt or decrypt operations, to execute arbitrary code with the permissions of the user running the application (compiled with nss). While the attack complexity is high, the impact to confidentiality, integrity, and availability are high as well."
	],
	"references": [
		"https://developer.mozilla.org/en-US/docs/Mozilla/Projects/NSS/NSS_3.44.3_release_notes\nhttps://developer.mozilla.org/en-US/docs/Mozilla/Projects/NSS/NSS_3.47.1_release_notes"
	],
	"acknowledgement": "Red Hat would like to thank the Mozilla Project for reporting this issue.",
	"upstream_fix": "nss 3.44.3, nss 3.47.1",
	"affected_release": [
		{
			"product_name": "Red Hat Enterprise Linux 6",
			"release_date": "2019-12-10T00:00:00Z",
			"advisory": "RHSA-2019:4152",
			"package": "nss-softokn-3.44.0-6.el6_10",
			"cpe": "cpe:/o:redhat:enterprise_linux:6"
		},
		{
			"product_name": "Red Hat Enterprise Linux 7",
			"release_date": "2019-12-10T00:00:00Z",
			"advisory": "RHSA-2019:4190",
			"package": "nss-3.44.0-7.el7_7",
			"cpe": "cpe:/o:redhat:enterprise_linux:7"
		},
		{
			"product_name": "Red Hat Enterprise Linux 8",
			"release_date": "2019-12-09T00:00:00Z",
			"advisory": "RHSA-2019:4114",
			"package": "nss-3.44.0-9.el8_1",
			"cpe": "cpe:/a:redhat:enterprise_linux:8"
		}
	],
	"package_state": [
		{
			"product_name": "Red Hat Enterprise Linux 5",
			"fix_state": "Not affected",
			"package_name": "thunderbird",
			"cpe": "cpe:/o:redhat:enterprise_linux:5"
		},
		{
			"product_name": "Red Hat Enterprise Linux 5",
			"fix_state": "Not affected",
			"package_name": "firefox",
			"cpe": "cpe:/o:redhat:enterprise_linux:5"
		},
		{
			"product_name": "Red Hat Enterprise Linux 5",
			"fix_state": "Out of support scope",
			"package_name": "nss",
			"cpe": "cpe:/o:redhat:enterprise_linux:5"
		},
		{
			"product_name": "Red Hat Enterprise Linux 6",
			"fix_state": "Not affected",
			"package_name": "firefox",
			"cpe": "cpe:/o:redhat:enterprise_linux:6"
		},
		{
			"product_name": "Red Hat Enterprise Linux 6",
			"fix_state": "Not affected",
			"package_name": "thunderbird",
			"cpe": "cpe:/o:redhat:enterprise_linux:6"
		},
		{
			"product_name": "Red Hat Enterprise Linux 7",
			"fix_state": "Not affected",
			"package_name": "firefox",
			"cpe": "cpe:/o:redhat:enterprise_linux:7"
		},
		{
			"product_name": "Red Hat Enterprise Linux 7",
			"fix_state": "Not affected",
			"package_name": "thunderbird",
			"cpe": "cpe:/o:redhat:enterprise_linux:7"
		},
		{
			"product_name": "Red Hat Enterprise Linux 8",
			"fix_state": "Not affected",
			"package_name": "firefox",
			"cpe": "cpe:/o:redhat:enterprise_linux:8"
		},
		{
			"product_name": "Red Hat Enterprise Linux 8",
			"fix_state": "Not affected",
			"package_name": "thunderbird",
			"cpe": "cpe:/o:redhat:enterprise_linux:8"
		}
	]
}
`

const CVEAffectedReleaseAndPackageStateWithUppercaseLetters =`
"CVE-2018-16328": {
	"name": "CVE-2018-16328",
	"threat_severity": "Low",
	"public_date": "2018-07-23T00:00:00Z",
	"bugzilla": {
		"description": "CVE-2018-16328 ImageMagick: NULL pointer dereference in CheckEventLogging function in MagickCore/log.c",
		"id": "1624955",
		"url": "https://bugzilla.redhat.com/show_bug.cgi?id=1624955"
	},
	"CVSS3": {
		"cvss3_base_score": "4.3",
		"cvss3_scoring_vector": "CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:N/I:N/A:L",
		"status": "verified"
	},
	"cwe": "CWE-476",
	"details": [
		"In ImageMagick before 7.0.8-8, a NULL pointer dereference exists in the CheckEventLogging function in MagickCore/log.c."
	],
	"affected_release": [
		{
			"product_name": "Red Hat Enterprise Linux 7",
			"release_date": "2020-03-31T00:00:00Z",
			"advisory": "RHSA-2020:1180",
			"package": "autotrace-0:0.31.1-38.el7",
			"cpe": "cpe:/o:redhat:enterprise_linux:7"
		},
		{
			"product_name": "Red Hat Enterprise Linux 7",
			"release_date": "2020-03-31T00:00:00Z",
			"advisory": "RHSA-2020:1180",
			"package": "emacs-1:24.3-23.el7",
			"cpe": "cpe:/o:redhat:enterprise_linux:7"
		},
		{
			"product_name": "Red Hat Enterprise Linux 7",
			"release_date": "2020-03-31T00:00:00Z",
			"advisory": "RHSA-2020:1180",
			"package": "ImageMagick-0:6.9.10.68-3.el7",
			"cpe": "cpe:/o:redhat:enterprise_linux:7"
		},
		{
			"product_name": "Red Hat Enterprise Linux 7",
			"release_date": "2020-03-31T00:00:00Z",
			"advisory": "RHSA-2020:1180",
			"package": "inkscape-0:0.92.2-3.el7",
			"cpe": "cpe:/o:redhat:enterprise_linux:7"
		}
	],
	"package_state": [
		{
			"product_name": "Red Hat Enterprise Linux 5",
			"fix_state": "Will not fix",
			"package_name": "ImageMagick",
			"cpe": "cpe:/o:redhat:enterprise_linux:5"
		},
		{
			"product_name": "Red Hat Enterprise Linux 6",
			"fix_state": "Will not fix",
			"package_name": "ImageMagick",
			"cpe": "cpe:/o:redhat:enterprise_linux:6"
		},
		{
			"product_name": "Red Hat Enterprise Linux 8",
			"fix_state": "Not affected",
			"package_name": "ImageMagick",
			"cpe": "cpe:/o:redhat:enterprise_linux:8"
		}
	]
}
`

func TestPackageFeed(t *testing.T) {
	for i, test := range []struct {
		feed     *Feed
		expected summary
	}{
		{
			testFeed(t, CVEMultiplePackages),
			summary{
				"qemu-kvm-ma":   []string{"CVE-2020-7211"},
				"qemu-kvm":      []string{"CVE-2020-7211"},
				"qemu-kvm-rhev": []string{"CVE-2020-7211"},
				"slirp4netns":   []string{"CVE-2020-7211"},
			},
		},
		{
			testFeed(t, CVENoPackageState),
			summary{
				"chromium-browser": []string{"CVE-2020-6377"},
			},
		},
		{
			testFeed(t, CVEAffectedReleaseAndPackageState),
			summary{
				"nss-softokn": []string{"CVE-2019-11745"},
				"nss":         []string{"CVE-2019-11745"},
				"thunderbird": []string{"CVE-2019-11745"},
				"firefox":     []string{"CVE-2019-11745"},
			},
                },
                {
                        testFeed(t, CVEAffectedReleaseAndPackageStateWithUppercaseLetters),
                        summary{
                            "emacs": []string{"CVE-2018-16328"},
                            "autotrace": []string{"CVE-2018-16328"},
                            "inkscape": []string{"CVE-2018-16328"},
                            "imagemagick": []string{"CVE-2018-16328"},
                        },
		},
	} {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			pkgFeed := test.feed.packageFeed()
			assert.Equal(t, test.expected, feedSummary(pkgFeed))
		})
	}

}
