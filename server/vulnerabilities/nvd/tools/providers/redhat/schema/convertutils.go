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

package schema

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/facebookincubator/flog"
	nvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/rpm"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

const (
	timeLayout = "2006-01-02T15:04:05"
)

// cwe regex to match CWEs
var cweRegex = regexp.MustCompile("CWE-[0-9]+")

func convertTime(redhatTime string) (string, error) {
	t, err := time.Parse(time.RFC3339, redhatTime)
	if err != nil {
		t, err = time.Parse(timeLayout, redhatTime)
	}
	if err != nil { // should be parsable
		flog.Errorf("unable to parse time: %v", err)
		return redhatTime, err
	}
	return t.Format(nvd.TimeLayout), nil
}

func findCWEs(s string) []string {
	return cweRegex.FindAllString(s, -1)
}

// IsFixed returns true if the state string describe a CVE resolution meaning a
// packaged isn't vulnerable.
func IsFixed(fixState string) bool {
	// $ jq 'to_entries | .[].value.package_state' redhat.json | grep fix_state | sort -u
	// "fix_state": "Affected",
	// "fix_state": "Fix deferred",
	// "fix_state": "New",
	// "fix_state": "Not affected",
	// "fix_state": "Out of support scope",
	// "fix_state": "Under investigation",
	// "fix_state": "Will not fix",

	switch strings.TrimSpace(strings.ToLower(fixState)) {
	case "affected", "fix deferred", "new", "out of support scope", "will not fix", "under investigation":
		return false
	case "not affected":
		return true
	default:
		flog.Infof("unknown fix state: %q", fixState)
		return false
	}
}

func packageName2wfn(packageName string) (*wfn.Attributes, error) {
	product, err := wfn.WFNize(packageName)
	if err != nil {
		return nil, fmt.Errorf("can't wfnize package name %q: %v", packageName, err)
	}
	attrs := wfn.Attributes{
		Part:    "a",
		Product: product,
	}
	return &attrs, nil
}

func package2wfn(pkg string) (*wfn.Attributes, error) {
	attrs := wfn.NewAttributesWithAny()
	err := rpm.ToWFN(attrs, pkg+".src") // add release .src so it parses correctly
	return attrs, err
}
