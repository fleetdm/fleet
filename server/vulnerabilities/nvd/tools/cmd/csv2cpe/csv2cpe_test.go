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
	"bytes"
	"flag"
	"fmt"
	"reflect"
	"testing"
)

func TestIntSet(t *testing.T) {
	t.Run("ReverseSortedSet", func(t *testing.T) {
		want := []int{3, 2, 1}
		have := NewIntSet(1, 1, 2, 2, 3).ReverseSortedSet()
		if !reflect.DeepEqual(want, have) {
			t.Fatalf("unexpected slice:\nwant: %+v\nhave: %+v\n", want, have)
		}
	})

	t.Run("FromString", func(t *testing.T) {
		s, err := NewIntSetFromString("1-3", "7", "9")
		if err != nil {
			t.Fatal(err)
		}
		want := []int{9, 7, 3, 2, 1}
		have := s.ReverseSortedSet()
		if !reflect.DeepEqual(want, have) {
			t.Fatalf("unexpected slice:\nwant: %+v\nhave: %+v\n", want, have)
		}
	})
}

func TestRemoveColumns(t *testing.T) {
	cases := []struct {
		want []string
		have []string
	}{
		{
			want: []string{"foo", "bar"},
			have: RemoveColumns([]string{"foo", "", "bar"}, NewIntSet(2)),
		},
		{
			want: []string{"foo"},
			have: RemoveColumns([]string{"foo", "", "bar"}, NewIntSet(2, 3)),
		},
		{
			want: []string{"foo", "bar"},
			have: RemoveColumns([]string{"", "foo", "bar"}, NewIntSet(0, 1)),
		},
		{
			want: []string{"bar"},
			have: RemoveColumns([]string{"", "foo", "bar"}, NewIntSet(1, 2)),
		},
	}

	for _, tc := range cases {
		if !reflect.DeepEqual(tc.want, tc.have) {
			t.Fatalf("unexpected slice:\nwant: %+v\nhave: %+v\n", tc.want, tc.have)
		}
	}
}

func TestProcessor(t *testing.T) {
	cases := []struct {
		flags []string
		skips IntSet
		na    bool
		in    string
		out   string
	}{
		{
			[]string{"-cpe_product=1", "-cpe_version=2"},
			NewIntSet(1, 2, 3),
			true,
			"Foo\t1.0...\tdelet\ta\nbar\t2.0\tdelet\tb",
			"a,cpe:/-:-:foo:1.0:-:-:-\nb,cpe:/-:-:bar:2.0:-:-:-\n",
		},
		{
			[]string{"-cpe_part=1", "-cpe_product=2", "-cpe_product=4"},
			NewIntSet(1, 2, 3),
			true,
			"a\tb\tc\n",
			"cpe:/a:-:-:-:-:-:-\n",
		},
		{
			[]string{"-cpe_part=1", "-cpe_product=2", "-cpe_version=3"},
			NewIntSet(1, 2, 3),
			true,
			"a\tbash\t4.4\n",
			"cpe:/a:-:bash:4.4:-:-:-\n",
		},
		{
			[]string{"-cpe_part=1", "-cpe_product=2", "-cpe_version=3"},
			NewIntSet(1, 2, 3),
			false,
			"a\tbash\t4.4\n",
			"cpe:/a::bash:4.4\n",
		},
	}

	for n, c := range cases {
		t.Run(fmt.Sprintf("case_%d", n), func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)

			acm := &AttributeColumnMap{}
			acm.AddFlags(fs)

			err := fs.Parse(c.flags)
			if err != nil {
				t.Fatal(err)
			}

			var stdin, stdout bytes.Buffer

			p := &Processor{
				InputComma:        rune('\t'),
				OutputComma:       rune(','),
				CPEToLower:        true,
				CPEOutputColumn:   2,
				EraseInputColumns: c.skips,
				DefaultNA:         c.na,
			}

			stdin.Write([]byte(c.in))

			err = p.Process(acm, &stdin, &stdout)
			if err != nil {
				t.Fatal(err)
			}

			if out := stdout.String(); out != c.out {
				t.Fatalf("unexpected output:\nwant: %q\nhave: %q\n", c.out, out)
			}
		})
	}

}
