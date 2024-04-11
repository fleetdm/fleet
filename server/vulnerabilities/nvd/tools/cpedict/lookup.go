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

package cpedict

import (
	"fmt"

	"github.com/facebookincubator/nvdtools/wfn"
)

// MatchType represents the type of match in dictionary lookup
type MatchType int

// Possible values of MatchType
const (
	None MatchType = iota
	Subset
	Exact
	Superset
)

// String() implements Stringer interface for MatchType
func (mt MatchType) String() string {
	switch mt {
	case None:
		return "None"
	case Subset:
		return "Subset"
	case Exact:
		return "Exact"
	case Superset:
		return "Superset"
	default:
		return fmt.Sprintf("invalid MatchType %d", mt)
	}
}

// Search determinces how WFN (NamePattern) relates the given dictionary.
// Deprecated matching names are resolved to their replacements; since item can be deprecated by multiple
// names, which might contain wildcards and in general refer to the whole family of products, this resolve isn't
// performed during exact match.
// If exact is true and an exact match is found, the function will return the match and match type of Exact.
// If the needle is a superset of any of the dictionary names, the function  will return that set of names
// and match type of Superset. If the needle is a subset of one or more dictionary names, the function will
// return that set and the match type of Subset. Otherwise an empty slice and match type None are returned.
// TODO: optimise the performance -- now it's O(n) to O(n^2), it should be easy enough to make it O(log n)
//       or even O(1) (e.g. use map keyed with WFNs instead of slice)
func (dict CPEList) Search(needle NamePattern, exact bool) ([]CPEItem, MatchType) {
	if exact {
		result := make([]CPEItem, 0)
		for _, item := range dict.Items {
			cmp, _ := wfn.Compare((*wfn.Attributes)(&needle), (*wfn.Attributes)(&item.Name))
			if cmp.IsEqual() {
				return append(result, item), Exact
			}
		}
		return nil, None
	}
	superset := make([]CPEItem, 0)
	subset := make([]CPEItem, 0)
	for _, item := range dict.Items {
		cmp, _ := wfn.Compare((*wfn.Attributes)(&needle), (*wfn.Attributes)(&item.Name))
		if cmp.IsSuperset() {
			superset = append(superset, resolveDeprecation(dict, item)...)
		} else if cmp.IsSubset() {
			subset = append(subset, resolveDeprecation(dict, item)...)
		}
	}
	if len(superset) > 0 {
		return superset, Superset
	}
	if len(subset) > 0 {
		return subset, Subset
	}
	return nil, None
}

func resolveDeprecation(dict CPEList, item CPEItem) []CPEItem {
	if !item.Deprecated || item.CPE23.Deprecation == nil {
		return []CPEItem{item}
	}
	var names []wfn.Attributes
	for _, depBy := range item.CPE23.Deprecation.DeprecatedBy {
		names = append(names, wfn.Attributes(depBy.Name))
	}
	results := make([]CPEItem, 0)
	for _, i := range dict.Items {
		for _, name := range names {
			cmp, _ := wfn.Compare(&name, (*wfn.Attributes)(&i.Name))
			if cmp.IsEqual() {
				results = append(results, i)
			}
		}
	}
	return results
}
