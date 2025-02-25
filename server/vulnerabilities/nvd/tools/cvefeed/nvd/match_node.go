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

package nvd

import (
	"fmt"
	"strings"

	"github.com/facebookincubator/flog"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

// Matcher returns an object which knows how to match attributes
func nodeMatcher(id string, node *schema.NVDCVEFeedJSON10DefNode) (wfn.Matcher, error) {
	if node == nil {
		return nil, fmt.Errorf("%s: node is nil", id)
	}

	var ms []wfn.Matcher
	for _, match := range node.CPEMatch {
		if match != nil {
			if m, err := cpeMatcher(id, match); err == nil {
				ms = append(ms, m)
			}
		}
	}
	for _, child := range node.Children {
		if child != nil {
			if m, err := nodeMatcher(id, child); err == nil {
				ms = append(ms, m)
			}
		}
	}

	if len(ms) == 0 {
		return nil, fmt.Errorf("%s: empty configuration for node", id)
	}

	var m wfn.Matcher

	switch strings.ToUpper(node.Operator) {
	default:
		flog.Warningf("%s: unknown operator, defaulting to OR: got %q", id, node.Operator)
		fallthrough
	case "OR":
		m = wfn.MatchAny(ms...)
	case "AND":
		m = wfn.MatchAll(ms...)
	}

	if node.Negate {
		m = wfn.DontMatch(m)
	}

	return m, nil
}
