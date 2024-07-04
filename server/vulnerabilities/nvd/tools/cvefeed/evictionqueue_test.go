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
	"testing"
	"time"
)

func TestEvictionQueue(t *testing.T) {
	var q evictionQueue
	cases := []string{"hello", "world", "quux", "baz", "foo"}
	for i, c := range cases {
		idx := q.push(c)
		if idx != i {
			t.Errorf("push() returned wrong index %d (%d was expected)", idx, i)
		}
		time.Sleep(1 * time.Millisecond)
	}

	// first, it should appear in order
	for i := range cases {
		if cases[i] != q.q[i].key {
			t.Errorf("unexpected queue order (before touch-ing):\nexpected %v\ngot      %v", cases, listKeys(q.q))
			break
		}
	}

	// touch it in reverse order
	for i := len(cases) - 1; i >= 0; i-- {
		q.touch(i)
		time.Sleep(1 * time.Millisecond)
	}

	// now baz and quux should be after foo and hello and world should be the last ones
	// but the exact order is non-deterministic
	for i, item := range q.q {
		switch i {
		case 0:
			if item.key != "foo" {
				t.Errorf("unexpected queue order (after touch-ing): %q at position %d", item.key, i)
			}
		case 1, 2:
			if item.key != "baz" && item.key != "quux" {
				t.Errorf("unexpected queue order (after touch-ing): %q at position %d", item.key, i)
			}
		case 3, 4:
			if item.key != "hello" && item.key != "world" {
				t.Errorf("unexpected queue order (after touch-ing): %q at position %d", item.key, i)
			}
		default:
			t.Fatal("unreacheable code reached o_O")
		}
	}

	// but when pop-ing the values from heap, it should come in order reverse to the one we started with
	for i := len(cases) - 1; i >= 0; i-- {
		item := q.pop()
		if item != cases[i] {
			t.Errorf("unexpected queue order (while pop-ing):\nexpected %v\ngot      %v", cases, listKeys(q.q))
			break
		}
	}
}

func listKeys(in []*evictionData) []string {
	out := make([]string, len(in))
	for i, item := range in {
		out[i] = item.key
	}
	return out
}
