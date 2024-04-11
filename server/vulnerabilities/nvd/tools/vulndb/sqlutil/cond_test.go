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

package sqlutil

import "testing"

func TestQueryConditionSet(t *testing.T) {
	c := Cond().Group(
		Cond().Equal("foo", "bar").And().Equal("x", "y"),
	).Or().In("z", []string{"a", "b"})

	want := "(foo=? AND x=?) OR z IN (?, ?)"
	have := c.String()
	if want != have {
		t.Fatalf("unexpected query condition:\nwant: %q\nhave: %q\n", want, have)
	}
}
