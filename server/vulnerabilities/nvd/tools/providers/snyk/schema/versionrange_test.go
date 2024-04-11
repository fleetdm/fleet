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
	"reflect"
	"testing"
)

func TestLooksLikeParenRange(t *testing.T) {
	cases := []struct {
		in  string
		out bool
	}{
		{"", false},
		{"<3.0.0-beta1 >1.12.3 || <1.12.0 >=1.4.0", false},
		{"[3.0.0,3.6.12), [3.7,3.9.14), [3.10.0,3.19.0)", true},
	}
	for _, c := range cases {
		if yes := looksLikeParenRange(c.in); yes != c.out {
			t.Errorf("range %q: expected: %t, got: %t", c.in, c.out, yes)
		}
	}
}

func TestParseParenRanges(t *testing.T) {
	cases := []struct {
		in    string
		out   []versionRange
		isErr bool
	}{
		{"", nil, true},
		{"[,2.8.10)", []versionRange{{maxVerExcl: "2.8.10"}}, false},
		{"[1.9.4,)", []versionRange{{minVerIncl: "1.9.4"}}, false},
		{" [3.0.0-rc.1]", []versionRange{{minVerIncl: "3.0.0-rc.1", maxVerIncl: "3.0.0-rc.1"}}, false},
		{"(1.9.4,2.8.10] ", []versionRange{{minVerExcl: "1.9.4", maxVerIncl: "2.8.10"}}, false},
		{
			in: "[,1.1.0-CR0-3), [1.1.0-CR1,1.1.0-CR3_1), [1.1.0-CR4, 1.3.7-CR1_2), [1.4.0,1.4.2-CR4_1), [1.5.0,1.5.4-CR6_2), [1.5.4-CR7,1.5.4-CR7_1], [1.5.5,1.5.7_9)",
			out: []versionRange{
				{maxVerExcl: "1.1.0-CR0-3"},
				{minVerIncl: "1.1.0-CR1", maxVerExcl: "1.1.0-CR3_1"},
				{minVerIncl: "1.1.0-CR4", maxVerExcl: "1.3.7-CR1_2"},
				{minVerIncl: "1.4.0", maxVerExcl: "1.4.2-CR4_1"},
				{minVerIncl: "1.5.0", maxVerExcl: "1.5.4-CR6_2"},
				{minVerIncl: "1.5.4-CR7", maxVerIncl: "1.5.4-CR7_1"},
				{minVerIncl: "1.5.5", maxVerExcl: "1.5.7_9"},
			},
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("parseParenRanges(%q)", c.in), func(t *testing.T) {
			vr, err := parseParenRanges(c.in)
			if err != nil && !c.isErr {
				t.Fatalf("unexpected error %v", err)
			}
			if !reflect.DeepEqual(vr, c.out) {
				t.Fatalf("\nexpected: %+v\ngot: %+v", c.out, vr)
			}
		})
	}
}

func TestParseCmpRanges(t *testing.T) {
	cases := []struct {
		in  string
		out []versionRange
	}{
		{"", nil},
		{"<3.0.1", []versionRange{{maxVerExcl: "3.0.1"}}},
		{"<=3.0.1", []versionRange{{maxVerIncl: "3.0.1"}}},
		{">3.0.0", []versionRange{{minVerExcl: "3.0.0"}}},
		{">=3.0.0", []versionRange{{minVerIncl: "3.0.0"}}},
		{">=3.0.0  <3.0.1", []versionRange{{minVerIncl: "3.0.0", maxVerExcl: "3.0.1"}}},
		{">3.0.0  <=3.0.1", []versionRange{{minVerExcl: "3.0.0", maxVerIncl: "3.0.1"}}},
		{"=3.0.0-rc.1", []versionRange{{minVerIncl: "3.0.0-rc.1", maxVerIncl: "3.0.0-rc.1"}}},
		{
			in: "< 1.12.4 || >= 2.0.0 <2.0.2",
			out: []versionRange{
				{maxVerExcl: "1.12.4"},
				{minVerIncl: "2.0.0", maxVerExcl: "2.0.2"},
			},
		},
	}
	for _, c := range cases {
		vr, _ := parseCmpRanges(c.in)
		if !reflect.DeepEqual(vr, c.out) {
			t.Errorf("range %q:\nexpected: %+v\ngot: %+v", c.in, c.out, vr)
		}
	}
}
