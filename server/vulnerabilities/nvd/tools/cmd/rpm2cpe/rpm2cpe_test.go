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
	"reflect"
	"strings"
	"testing"
)

func TestSkipFields(t *testing.T) {
	cases := []struct {
		f2s fieldsToSkip
		in  []string
		out []string
	}{
		{
			f2s: fieldsToSkip(map[int]struct{}{}),
			in:  []string{"0", "1", "2", "3", "4", "5"},
			out: []string{"0", "1", "2", "3", "4", "5"},
		},
		{
			f2s: fieldsToSkip(map[int]struct{}{1: {}, 3: {}}),
			in:  []string{"0", "1", "2", "3", "4", "5"},
			out: []string{"0", "2", "4", "5"},
		},
		{
			f2s: fieldsToSkip(map[int]struct{}{0: {}, 1: {}, 2: {}}),
			in:  []string{"0", "1", "2"},
			out: []string{},
		},
	}
	for n, c := range cases {
		in := make([]string, len(c.in))
		copy(in, c.in)
		in = c.f2s.skipFields(in)
		if !reflect.DeepEqual(in, c.out) {
			t.Errorf("case #%d: expected %v, got %v", n, c.out, in)
		}
	}
}

func TestProcessRecord(t *testing.T) {
	cases := []struct {
		in, out   string
		defaultNA bool
		fail      bool
	}{
		{"", "", false, true},
		{"0,name-1.0-1.noarch.rpm", "name-1.0-1.noarch.rpm;cpe:/a::name:1.0:1:~-~-~-~~-:-", true, false},
		{
			in:        "0,name-1.0-1.i386.rpm,2,3,4,5,6",
			out:       "name-1.0-1.i386.rpm;2;4;5;cpe:/a::name:1.0:1:~-~-~-~i386~-:-;6",
			defaultNA: true,
		},
	}
	cfg := config{
		rpmField:    2,
		cpeField:    5,
		inFieldSep:  ",",
		outFieldSep: ";",
		skip:        fieldsToSkip(map[int]struct{}{0: {}, 3: {}}),
	}
	for _, c := range cases {
		cfg.defaultNA = c.defaultNA
		record, err := processRecord(strings.Split(c.in, cfg.inFieldSep), cfg)
		if err != nil {
			if !c.fail {
				t.Errorf("line %q was expected to succeed, but failed: %v", c.in, err)
			}
			continue
		}
		if c.fail {
			t.Errorf("line %q was expected to fail, but succeeded", c.in)
			continue
		}
		out := strings.Join(record, cfg.outFieldSep)
		if c.out != out {
			t.Errorf("line %q:\nhave: %q\nwant: %q", c.in, c.out, out)
		}
	}
}
