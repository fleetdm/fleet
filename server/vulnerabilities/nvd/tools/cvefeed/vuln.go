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
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/wfn"
)

// Vuln is a vulnerability interface
type Vuln interface {
	// vulnerability should also be able to match attributes
	wfn.Matcher
	// ID returns the vulnerability ID
	ID() string
	// CVEs returns all CVEs it includes/references
	CVEs() []string
	// CWEs returns all CWEs for this vulnerability
	CWEs() []string
	// CVSSv2BaseScore returns CVSS v2 base score
	CVSSv2BaseScore() float64
	// CVSSv2BaseScore returns CVSS v2 vector
	CVSSv2Vector() string
	// CVSSv2BaseScore returns CVSS v3 base score
	CVSSv3BaseScore() float64
	// CVSSv2BaseScore returns CVSS v3 vector
	CVSSv3Vector() string
}

// MergeVuln combines two Vulns:
// resulted Vuln inherits all mutually exclusive methods (e.g. ID()) from Vuln x;
// functions returning CVEs and CWEs return distinct(union(x,y))
// the returned vuln matches attributes if x matches AND y doesn't
func OverrideVuln(v, override Vuln) Vuln {
	return &overriden{
		Vuln:    v,
		matcher: &andMatcher{v, wfn.DontMatch(override)},
	}
}

type overriden struct {
	Vuln
	matcher wfn.Matcher
}

// Match is a part of the wfn.Matcher interface
func (v *overriden) Match(attrs []*wfn.Attributes, requireVersion bool) []*wfn.Attributes {
	return v.matcher.Match(attrs, requireVersion)
}

// Attrs is a part of the wfn.Matcher interface
func (v *overriden) Config() []*wfn.Attributes {
	return v.matcher.Config()
}

// matches are the ones matched by both
type andMatcher struct {
	m1, m2 wfn.Matcher
}

// Match is a part of the wfn.Matcher interface
func (m *andMatcher) Match(attrs []*wfn.Attributes, requireVersion bool) []*wfn.Attributes {
	return m.m2.Match(m.m1.Match(attrs, requireVersion), requireVersion)
}

// Attrs is a part of the wfn.Matcher interface
func (m *andMatcher) Config() []*wfn.Attributes {
	return append(m.m1.Config(), m.m2.Config()...)
}
