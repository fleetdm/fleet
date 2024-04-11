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

package main

import (
	"fmt"
	"testing"
)

func TestFieldsToSkip(t *testing.T) {
	cases := []struct {
		in, out string
		fail    bool
	}{
		{"", "", true},
		{"foo", "", true},
		{"1", "1", false},
		{"1,2,3", "1,2,3", false},
		{"3-5", "3,4,5", false},
	}
	for n, c := range cases {
		c := c
		t.Run(fmt.Sprintf("%d", n+1), func(t *testing.T) {
			var f2s fieldsToSkip
			err := f2s.Set(c.in)
			if err != nil && !c.fail {
				t.Fatalf("unexpected error: %v", err)
			}
			out := f2s.String()
			if out != c.out {
				t.Fatalf("expected %q, got %q", c.out, out)
			}
		})
	}
}
