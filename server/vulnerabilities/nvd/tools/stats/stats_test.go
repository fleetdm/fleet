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

package stats

import (
	"bytes"
	"fmt"
	"testing"
)

func TestCounters(t *testing.T) {
	s := New()
	s.IncrementCounter("c1")
	s.IncrementCounter("c2")
	s.IncrementCounterBy("c2", 2)
	s.IncrementCounterBy("c3", 3)

	for i, tc := range []struct {
		key   string
		value int64
	}{
		{"c1", 1},
		{"c2", 3},
		{"c3", 3},
	} {
		t.Run(fmt.Sprintf("case %2d", i+1), func(t *testing.T) {
			value := s.GetCounter(tc.key)
			if value != tc.value {
				t.Errorf("counter %s: expected %d - got %d", tc.key, tc.value, value)
			}
		})
	}

	s.Clear()

	for i, key := range []string{
		"c1",
		"c2",
		"c3",
	} {
		t.Run(fmt.Sprintf("case %2d", i+1), func(t *testing.T) {
			value := s.GetCounter(key)
			if value != 0 {
				t.Errorf("counter %s: expected 0 - got %d", key, value)
			}
		})
	}

}

func TestValues(t *testing.T) {
	s := New()
	s.AddToValue("v1", 1.2)
	s.AddToValue("v2", 2.1)

	for i, tc := range []struct {
		key   string
		value float64
	}{
		{"v1", 1.2},
		{"v2", 2.1},
	} {
		t.Run(fmt.Sprintf("case %2d", i+1), func(t *testing.T) {
			value := s.GetValue(tc.key)
			if value != tc.value {
				t.Errorf("value %s: expected %.1f - got %.1f", tc.key, tc.value, value)
			}
		})
	}

	s.Clear()

	for i, key := range []string{
		"v1",
		"v2",
	} {
		t.Run(fmt.Sprintf("case %2d", i+1), func(t *testing.T) {
			value := s.GetValue(key)
			if value != 0.0 {
				t.Errorf("value %s: expected 0 - got %.1f", key, value)
			}
		})
	}

}

func TestWrite(t *testing.T) {
	s := New()
	s.IncrementCounterBy("c", 1)
	s.AddToValue("v", 2.3)

	var buff bytes.Buffer
	if err := s.write(&buff); err != nil {
		t.Errorf("write csv error: %v", err)
	}
	got := buff.String()
	want := "c,1\nv,2.30\n"
	if got != want {
		t.Errorf("wrong csv format: got %q, want %q", got, want)
	}
}
