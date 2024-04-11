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

import (
	"reflect"
	"testing"
)

type sampleRecord struct {
	K string `sql:"foo"`
	V int    `sql:"bar"`
	Z bool
}

func TestRecordType(t *testing.T) {
	s := sampleRecord{
		K: "hello",
		V: 42,
	}
	r := NewRecordType(&s).Subset("foo", "bar")
	f := r.Fields()
	v := r.Values()
	t.Run("fields", func(t *testing.T) {
		want := []string{"foo", "bar"}
		if !reflect.DeepEqual(f, want) {
			t.Fatalf("unexpected fields:\nwant: %v\nhave: %v\n", want, f)
		}
	})
	t.Run("values", func(t *testing.T) {
		want := []interface{}{"hello", 42}
		rk := v[0].(*string)
		rv := v[1].(*int)
		have := []interface{}{*rk, *rv}
		if !reflect.DeepEqual(have, want) {
			t.Fatalf("unexpected values:\nwant: %v\nhave: %v\n", want, have)
		}
		t.Run("scan", func(t *testing.T) {
			want := 13
			vv := v[1].(*int)
			*vv = want
			if s.V != want {
				t.Fatalf("unexpected value:\nwant: %d\nhave: %d\n", want, s.V)
			}
		})
	})
}

func TestRecords(t *testing.T) {
	rs := NewRecords([]sampleRecord{
		{
			K: "hello",
			V: 42,
		},
		{
			K: "world",
			V: 13,
			Z: true,
		},
	})
	t.Run("fields", func(t *testing.T) {
		want := []string{"foo", "bar", "Z"}
		have := rs.Fields()
		if !reflect.DeepEqual(want, have) {
			t.Fatalf("unexpected value:\nwant: %v\nhave: %v\n", want, have)
		}
	})
	t.Run("values", func(t *testing.T) {
		want := []interface{}{"hello", 42, false, "world", 13, true}
		have := rs.Values()
		if !reflect.DeepEqual(want, have) {
			t.Fatalf("unexpected value:\nwant: %v\nhave: %v\n", want, have)
		}
	})
}
