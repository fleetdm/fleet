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
	"unicode"
)

type versionRange struct {
	minVerIncl string
	minVerExcl string
	maxVerIncl string
	maxVerExcl string
}

func parseVersionRange(rangeStr string) ([]versionRange, error) {
	if looksLikeParenRange(rangeStr) {
		return parseParenRanges(rangeStr)
	}
	return parseCmpRanges(rangeStr)
}

// looksLikeParenRange tries to detect the format version range(s) are provided.
// The interval can be specified either in mathematical notation, where square brackets
// and parentheses denote open and closed intervals respectively and any number can be omitted if there is
// no corresponding limit; e.g. [0, 5), [,3], (8,); any of comma separated list such intervals can be applied;
// or with help of comparison operators (<, >, <=, >=); edge cases separated by space, separate inervals are
// combined with logical or || operator.
// It returns true if the range looks like math notation, false otherwise
func looksLikeParenRange(s string) bool {
	if s == "" {
		return false
	}
	s = strings.TrimSpace(s)
	return (s[0] == '[' || s[0] == '(') && (s[len(s)-1] == ']' || s[len(s)-1] == ')')
}

// parseParenRanges parses a sequence of intervals in mathematical notation into a slice of versionRange structs.
// Single boundary in open-ended interval means equality check.
func parseParenRanges(s string) (vr []versionRange, err error) {
	for len(s) > 0 {
		var r versionRange
		left := strings.IndexAny(s, "([")
		right := strings.IndexAny(s, ")]")
		if left == -1 || right == -1 || right < left {
			return nil, fmt.Errorf("invalid range %q", s)
		}
		boundaries := strings.Split(s[left+1:right], ",")
		if len(boundaries) == 1 {
			r.minVerIncl = strings.TrimSpace(boundaries[0])
			r.maxVerIncl = r.minVerIncl
			vr = append(vr, r)
			s = s[right+1:]
			continue
		}
		if len(boundaries) != 2 {
			return nil, fmt.Errorf("invalid range %q", s)
		}
		if s[left] == '(' {
			r.minVerExcl = strings.TrimSpace(boundaries[0])
		} else {
			r.minVerIncl = strings.TrimSpace(boundaries[0])
		}
		if s[right] == ')' {
			r.maxVerExcl = strings.TrimSpace(boundaries[1])
		} else {
			r.maxVerIncl = strings.TrimSpace(boundaries[1])
		}
		vr = append(vr, r)
		// skip trailing spaces
		suffix := s[right+1:]
		for _, r := range suffix {
			if !unicode.IsSpace(r) {
				break
			}
			right++
		}
		s = s[right+1:]
	}
	return vr, nil
}

// parseCmpRange parses a sequence of intervals described as comparison operators (<=, <, =, >, >=);
// ranges can be combined with || operators for boolean OR logic.
func parseCmpRanges(s string) (vr []versionRange, err error) {
	re := regexp.MustCompile(`([<>](:?=)?)\s*(\S+)`)
	ss := strings.Split(s, "||")
	for _, rs := range ss {
		var r versionRange
		if rs == "" {
			continue
		}
		rs = strings.TrimSpace(rs)
		// first, check if it is just an equality check
		if rs[0] == '=' {
			r.maxVerIncl = strings.TrimSpace(rs[1:])
			r.minVerIncl = r.maxVerIncl
			vr = append(vr, r)
			continue
		}
		// then process the range
		matches := re.FindAllString(strings.TrimSpace(rs), -1)
		for _, match := range matches {
			switch match[0] {
			case '<':
				if match[1] == '=' {
					r.maxVerIncl = strings.TrimSpace(match[2:])
				} else {
					r.maxVerExcl = strings.TrimSpace(match[1:])
				}
			case '>':
				if match[1] == '=' {
					r.minVerIncl = strings.TrimSpace(match[2:])
				} else {
					r.minVerExcl = strings.TrimSpace(match[1:])
				}
			}
		}
		vr = append(vr, r)
	}
	return vr, nil
}
