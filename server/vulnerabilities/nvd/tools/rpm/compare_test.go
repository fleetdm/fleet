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

package rpm

import (
	"fmt"
	"testing"
)

func TestVersionCompare(t *testing.T) {
	cases := []struct {
		v1, v2 string
		result int
	}{
		{"", "", 0},
		{"~1", "99z", -1},
		{"1", "2", -1},
		{"11", "2", 1},
		{"~1", "~2", -1},
		{"~2", "1", -1},
		{"a1", "b1", -1},
		{"0001", "1", 0},
		{"0001", "2", -1},
		{"1", "a", 1},
		{"a", "1", -1},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case-%d", i+1), func(t *testing.T) {
			if r := versionCompare(c.v1, c.v2); r != c.result {
				t.Errorf("compare(%q, %q) = %d, expecting %d", c.v1, c.v2, r, c.result)
			}
		})
	}
}

func BenchmarkVersionCompare(b *testing.B) {
	for i := 0; i < b.N; i++ {
		versionCompare("1a2b3c4d", "1a2b3c4de")
	}
}
