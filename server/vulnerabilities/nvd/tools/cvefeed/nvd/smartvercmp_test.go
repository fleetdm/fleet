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
	"testing"
)

func TestSmartVerCmp(t *testing.T) {
	cases := []struct {
		v1, v2 string
		ret    int
	}{
		{"5", "8", -1},
		{"15", "3", 1},
		{"4a", "4c", -1},
		{"1.0", "1.0", 0},
		{"1.0.1", "1.0", 1},
		{"1.0.14", "1.0.4", 1},
		{"95SE", "98SP1", -1},
		{"16.0.0", "3.2.7", 1},
		{"10.23", "10.21", 1},
		{"64.0", "3.6.24", 1},
		{"5-1.15.2", "5-1.16", -1},
		{"5-appl_1.16.1", "5-1.0.1", -1}, // this is wrong, but seems to be impossible to account for
		{"5-1.16", "5_1.0.6", 1},
		{"5-6", "5-16", -1},
		{"5-a1", "5a1", -1}, // meh, kind of makes sense
		{"5-a1", "5.a1", 0},
		{"1.4", "1.02", 1},
		{"5.0", "08.0", -1},
		{"10.0", "1.0", 1},
		{"2023.02.13", "2023.2.13", 0},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%q vs %q", c.v1, c.v2), func(t *testing.T) {
			if ret := SmartVerCmp(c.v1, c.v2); ret != c.ret {
				t.Fatalf("expected %d, got %d", c.ret, ret)
			}
		})
	}
}

func BenchmarkSmartVerCmp(b *testing.B) {
	cases := []struct {
		v1, v2 string
	}{
		{"1.0", "1.0"},
		{"1.0.1", "1.0"},
		{"1.0.14", "1.0.4"},
		{"95SE", "98SP1"},
		{"16.0.0", "3.2.7"},
		{"10.23", "10.21"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, c := range cases {
			SmartVerCmp(c.v1, c.v2)
		}
	}
}
