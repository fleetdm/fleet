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
	"testing"
)

func TestConvertTime(t *testing.T) {
	for i, test := range []struct {
		input, expected string
	}{
		{"2006-01-02T15:04:05", "2006-01-02T15:04Z"},
		{"2006-01-02T15:04:05Z", "2006-01-02T15:04Z"},
	} {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			if got, _ := convertTime(test.input); got != test.expected {
				t.Fatalf("expected %s, got %s", test.expected, got)
			}
		})
	}
}
