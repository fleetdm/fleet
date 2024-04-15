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

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/facebookincubator/nvdtools/wfn"
)

// cpeMatch is a wrapper around the actual NVDCVEFeedJSON10DefCPEMatch
type cpeMatch struct {
	*wfn.Attributes
	vulnerable            bool
	versionEndExcluding   string
	versionEndIncluding   string
	versionStartExcluding string
	versionStartIncluding string
	hasVersionRanges      bool
}

// Matcher returns an object which knows how to match attributes
func cpeMatcher(ID string, nvdMatch *schema.NVDCVEFeedJSON10DefCPEMatch) (wfn.Matcher, error) {
	parse := func(uri string) (*wfn.Attributes, error) {
		if uri == "" {
			return nil, fmt.Errorf("%s: can't parse empty uri", ID)
		}
		return wfn.Parse(uri)
	}

	// parse
	match := cpeMatch{vulnerable: nvdMatch.Vulnerable}
	var err error
	if match.Attributes, err = parse(nvdMatch.Cpe23Uri); err != nil {
		if match.Attributes, err = parse(nvdMatch.Cpe22Uri); err != nil {
			return nil, fmt.Errorf("%s: unable to parse both cpe2.2 and cpe2.3", ID)
		}
	}

	match.versionEndExcluding = nvdMatch.VersionEndExcluding
	match.versionEndIncluding = nvdMatch.VersionEndIncluding
	match.versionStartExcluding = nvdMatch.VersionStartExcluding
	match.versionStartIncluding = nvdMatch.VersionStartIncluding

	if match.versionStartIncluding != "" || match.versionStartExcluding != "" ||
		match.versionEndIncluding != "" || match.versionEndExcluding != "" {
		match.hasVersionRanges = true
	}

	return &match, nil
}

// Match is part of the Matcher interface
func (cm *cpeMatch) Match(attrs []*wfn.Attributes, requireVersion bool) (matches []*wfn.Attributes) {
	for _, attr := range attrs {
		if cm.match(attr, requireVersion) {
			matches = append(matches, attr)
		}
	}
	return matches
}

// Match implements wfn.Matcher interface
func (cm *cpeMatch) match(attr *wfn.Attributes, requireVersion bool) bool {
	if cm == nil || cm.Attributes == nil {
		return false
	}

	if requireVersion {
		// if we require version, then we need either version ranges or version not to be *
		if !cm.hasVersionRanges && cm.Attributes.Version == wfn.Any {
			return false
		}
	}

	// here we have a version: either actual one or ranges

	// check whether everything except for version matches
	if !cm.Attributes.MatchWithoutVersion(attr) {
		return false
	}

	if cm.Attributes.Version == wfn.Any {
		if !cm.hasVersionRanges {
			// if version is any and doesn't have version ranges, then it matches any
			return !requireVersion
		} // otherwise we try to match it at the end of the function
	} else if cm.Attributes.MatchOnlyVersion(attr) {
		return true // version matched
	}

	// if it got to here, it means:
	//	- matched attr without version
	//  - didn't match version, or require version was set and version was *

	if attr.Version == wfn.Any {
		return true
	}

	if !cm.hasVersionRanges {
		return false
	}

	// if hasVersionRanges and attr version is NA, then return false
	if attr.Version == wfn.NA {
		return false
	}

	// match version to ranges
	ver := wfn.StripSlashes(attr.Version)

	matches := true

	if cm.versionStartIncluding != "" {
		matches = matches && smartVerCmp(ver, cm.versionStartIncluding) >= 0
	}
	if cm.versionStartExcluding != "" {
		matches = matches && smartVerCmp(ver, cm.versionStartExcluding) > 0
	}
	if cm.versionEndIncluding != "" {
		matches = matches && smartVerCmp(ver, cm.versionEndIncluding) <= 0
	}
	if cm.versionEndExcluding != "" {
		matches = matches && smartVerCmp(ver, cm.versionEndExcluding) < 0
	}

	return matches
}
