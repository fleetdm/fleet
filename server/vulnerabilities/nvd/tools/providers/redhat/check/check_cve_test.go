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

package check

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/facebookincubator/nvdtools/providers/redhat/schema"
	"github.com/facebookincubator/nvdtools/rpm"
	"github.com/facebookincubator/nvdtools/wfn"
)

func TestCVEChecker(t *testing.T) {
	var cve schema.CVE
	if err := json.NewDecoder(strings.NewReader(cveStr)).Decode(&cve); err != nil {
		t.Fatal(err)
	}

	chk, err := CVEChecker(&cve)
	if err != nil {
		t.Fatal(err)
	}

	pkg, _ := rpm.Parse("firefox-68.1.0-1.el8_0.src")

	for i, tc := range []struct {
		distroVersion string
		expect        bool
	}{
		{"5", false},
		{"6", true},
		{"7", true},
		{"8", true},
		{"9", false},
	} {
		t.Run(fmt.Sprintf("case-%d", i+1), func(t *testing.T) {
			distro := wfn.Attributes{
				Part:    "o",
				Vendor:  "redhat",
				Product: "enterprise_linux",
				Version: tc.distroVersion,
			}
			if got := chk.Check(pkg, &distro, "CVE-2019-11735"); got != tc.expect {
				t.Fatalf("expecting %v for version %q, got %v", tc.expect, tc.distroVersion, got)
			}
			if chk.Check(pkg, &distro, "CVE-some-other") {
				t.Fatalf("shouldn't match when unknown cve is given")
			}
		})
	}
}

func TestCVECheckerUppercase(t *testing.T) {
	var cve schema.CVE
	cveStrWithUppercasePkg := strings.ReplaceAll(cveStr, "firefox", "Firefox")
	if err := json.NewDecoder(strings.NewReader(cveStrWithUppercasePkg)).Decode(&cve); err != nil {
		t.Fatal(err)
	}
	chk, err := CVEChecker(&cve)
	if err != nil {
		t.Fatal(err)
	}
	distro := wfn.Attributes{
		Part:    "o",
		Vendor:  "redhat",
		Product: "enterprise_linux",
		Version: "7",
	}
	pkg, _ := rpm.Parse("Firefox-68.1.0-1.el7_0.src")
	if got := chk.Check(pkg, &distro, "CVE-2019-11735"); !got {
		t.Fatalf("expecting true for version 7")
	}
}

var cveStr = `
  {
    "name": "CVE-2019-11735",
    "threat_severity": "Important",
    "public_date": "2019-09-03T00:00:00",
    "bugzilla": {
      "description": "\nCVE-2019-11735 Mozilla: Memory safety bugs fixed in Firefox 69 and Firefox ESR 68.1\n    ",
      "id": "1748661",
      "url": "https://bugzilla.redhat.com/show_bug.cgi?id=1748661"
    },
    "CVSS3": {
      "cvss3_base_score": "7.5",
      "cvss3_scoring_vector": "CVSS:3.0/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H",
      "status": "verified"
    },
    "cwe": "CWE-120",
    "details": [
      "\n** RESERVED ** This candidate has been reserved by an organization or individual that will use it when announcing a new security problem. When the candidate has been publicized, the details for this candidate will be provided.\n    "
    ],
    "references": [
      "\nhttps://www.mozilla.org/en-US/security/advisories/mfsa2019-26/#CVE-2019-11735\n    "
    ],
    "acknowledgement": "\nRed Hat would like to thank the Mozilla project for reporting this issue. Upstream acknowledges Mikhail Gavrilov, Tyson Smith, Marcia Knous, Tom Ritter, Philipp, and Bob Owens as the original reporters.\n    ",
    "upstream_fix": "firefox 68.1",
    "affected_release": [
      {
        "product_name": "Red Hat Enterprise Linux 8",
        "release_date": "2019-09-04T00:00:00",
        "advisory": "RHSA-2019:2663",
        "package": "firefox-68.1.0-1.el8_0",
        "cpe": "cpe:/a:redhat:enterprise_linux:8"
      }
    ],
    "package_state": [
      {
        "product_name": "Red Hat Enterprise Linux 5",
        "fix_state": "Out of support scope",
        "package_name": "firefox",
        "cpe": "cpe:/o:redhat:enterprise_linux:5"
      },
      {
        "product_name": "Red Hat Enterprise Linux 6",
        "fix_state": "Not affected",
        "package_name": "firefox",
        "cpe": "cpe:/o:redhat:enterprise_linux:6"
      },
      {
        "product_name": "Red Hat Enterprise Linux 7",
        "fix_state": "Not affected",
        "package_name": "firefox",
        "cpe": "cpe:/o:redhat:enterprise_linux:7"
      }
    ]
  }
`
