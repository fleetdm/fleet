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

package cvefeed

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

func TestMatchOverrides(t *testing.T) {
	cases := [][]*wfn.Attributes{
		{},
		{
			{Part: "o", Vendor: "linux", Product: "linux_kernel", Version: "2\\.6\\.1"},
			{Part: "a", Vendor: "djvulibre_project", Product: "djvulibre", Version: "3\\.5\\.11"},
		},
		{
			{Part: "o", Vendor: "microsoft", Product: "windows_xp", Update: "sp3"},
			{Part: "a", Vendor: "microsoft", Product: "ie", Version: "5\\.0", Update: wfn.NA},
		},
		{
			{Part: "o", Vendor: "microsoft", Product: "windows_xp", Update: "sp3"},
			{Part: "a", Vendor: "microsoft", Product: "ie", Version: "5\\.0", Update: "patched"},
		},
	}
	dict, err := LoadFeed(func(_ string) ([]Vuln, error) {
		return ParseJSON(bytes.NewBufferString(testJSONdict))
	}, "")
	if err != nil {
		t.Fatalf("could not load test JSON feed: %v", err)
	}
	original, _ := LoadFeed(func(_ string) ([]Vuln, error) {
		return ParseJSON(bytes.NewBufferString(testJSONdict))
	}, "")
	overrides, err := LoadFeed(func(_ string) ([]Vuln, error) {
		return ParseJSON(bytes.NewBufferString(testJSONoverride))
	}, "")
	if err != nil {
		t.Fatalf("could not load test overrides: %v", err)
	}
	dict.Override(overrides)

	for n, c := range cases {
		c := c
		t.Run(fmt.Sprintf("%d", n), func(t *testing.T) {
			var matchOriginal, matchOverride, matchDict bool
			if m := dict["TESTVE-2018-0002"].Match(c, false); len(m) > 0 {
				matchDict = true
			}
			if m := original["TESTVE-2018-0002"].Match(c, false); len(m) > 0 {
				matchOriginal = true
			}
			if m := overrides["TESTVE-2018-0002"].Match(c, false); len(m) > 0 {
				matchOverride = true
			}
			if matchOriginal && matchDict && matchOverride {
				t.Fatal("case was not overriden")
			} else if matchDict && !matchOriginal {
				t.Fatal("unexpected match")
			}
		})
	}
}

var testJSONoverride = `{
"CVE_data_type" : "CVE",
"CVE_data_format" : "MITRE",
"CVE_data_version" : "4.0",
"CVE_data_numberOfCVEs" : "7083",
"CVE_data_timestamp" : "2018-07-31T07:00Z",
"CVE_Items" : [
  {
    "cve" : {
      "data_type" : "CVE",
      "data_format" : "MITRE",
      "data_version" : "4.0",
      "CVE_data_meta" : {
        "ID" : "TESTVE-2018-0002",
        "ASSIGNER" : "cve@mitre.org"
      }
    },
    "configurations" : {
      "CVE_data_version" : "4.0",
      "nodes" : [
        {
          "operator" : "OR",
          "cpe_match" : [ {
            "vulnerable" : true,
              "cpe23Uri" : "cpe:2.3:a:microsoft:ie:*:patched:*:*:*:*:*:*"
          } ]
        }
      ]
    }
  }
]
}`
