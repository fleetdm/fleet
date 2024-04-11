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
	"sort"
	"strconv"
	"strings"
)

// filedsToSkip is a custom type to be recognized by flag.Parse().
// It maps comma-separated numbers from command line option to a set of integers.
// It also provides methods for dropping configured indices from a slice.
type fieldsToSkip map[int]bool

// skipFields removes elements from  fields slice as per config
func (fs fieldsToSkip) skipFields(fields []string) []string {
	j := 0
	for i := 0; i < len(fields); i++ {
		if fs[i] {
			continue
		}
		fields[j] = fields[i]
		j++
	}
	return fields[:j]
}

// appendAt appends and element to a slice at position at after skipping configured fields.
// Negative pos skips the next element.
func (fs fieldsToSkip) appendAt(to []string, args ...interface{}) []string {
	to = fs.skipFields(to)
	fields := map[int]string{}
	keys := make([]int, 0, len(args)/2)
	pos := -1
	for _, arg := range args {
		switch arg := arg.(type) {
		case int:
			pos = arg
			if pos >= 0 {
				keys = append(keys, pos)
			}
		case string:
			if pos < 0 {
				break
			}
			fields[pos] = arg
			pos = -1
		default:
			panic(fmt.Sprintf("appendAt: unsupported type %T", arg))
		}
	}
	sort.Ints(keys)
	for _, at := range keys {
		if at > len(to) {
			at = len(to)
		}
		out := make([]string, 0, len(to)+1)
		out = append(out, to[:at]...)
		out = append(out, fields[at])
		out = append(out, to[at:]...)
		to = out
	}
	return to
}

// part of flag.Value interface implementation
func (fs fieldsToSkip) String() string {
	keys := make([]int, 0, len(fs))
	for k := range fs {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	fss := make([]string, len(keys))
	for i, v := range keys {
		fss[i] = fmt.Sprintf("%d", v+1)
	}
	return strings.Join(fss, ",")
}

// part of flag.Value interface implementation
func (fs *fieldsToSkip) Set(val string) error {
	if *fs == nil {
		*fs = fieldsToSkip{}
	}
	for _, v := range strings.Split(val, ",") {
		nn, err := atoii(v)
		if err != nil {
			return fmt.Errorf("bad fieldsToSkip value: %q: %v", v, err)
		}
		for _, n := range nn {
			if n < 1 {
				return fmt.Errorf("illegal field index %d", n)
			}
			(*fs)[n-1] = true
		}
	}
	return nil
}

// atoii parses ranges of positive integers from string r.
// E.g., it will return [1, 2, 3, 4] for "1-4", [3] for "3";
// Open ranges (e.g. "-3", "3-") are not allowed and are parsed as a single integer.
// Any character other than digit or '-' in the input will trigger an error.
func atoii(r string) ([]int, error) {
	var ret []int
	fromto := strings.Split(r, "-")
	if len(fromto) != 1 && len(fromto) != 2 {
		return nil, fmt.Errorf("illegal range: %q", r)
	}
	start, err := strconv.Atoi(fromto[0])
	if err != nil {
		return nil, err
	}
	ret = append(ret, start)
	if len(fromto) == 1 {
		return ret, nil
	}
	end, err := strconv.Atoi(fromto[1])
	if err != nil {
		return nil, err
	}
	for i := start + 1; i < end; i++ {
		ret = append(ret, i)
	}
	ret = append(ret, end)
	return ret, nil
}
