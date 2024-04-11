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

func TestSelect(t *testing.T) {
	selectCases := []struct {
		Want string
		Have *SelectStmt
	}{
		{
			Want: "SELECT k, v FROM foobar WHERE k=? AND v=? LIMIT 1",
			Have: Select("k", "v").From("foobar").Where(
				Cond().Equal("k", "hello").And().Equal("v", "world"),
			).Literal("LIMIT 1"),
		},
		{
			Want: "SELECT t.k, t.v FROM (SELECT * FROM x UNION ALL SELECT * FROM y) AS t LEFT JOIN z ON t.k = z.k WHERE k IN (?, ?)",
			Have: Select("t.k", "t.v").From().SelectGroup("t",
				Select("*").
					From("x").
					Literal("UNION ALL").
					Select(Select("*").From("y")),
			).Literal("LEFT JOIN z ON t.k = z.k").Where(
				Cond().In("k", []string{"foo", "bar"}),
			),
		},
	}

	for _, tc := range selectCases {
		have := tc.Have.String()
		if have != tc.Want {
			t.Fatalf("unexpected statement:\nhave: %s\nwant: %s\n", have, tc.Want)
		}
	}
}

func TestInsert(t *testing.T) {
	r1 := NewRecordType(sampleRecord{K: "hello"}).Subset("foo")
	r2 := NewRecordType(sampleRecord{K: "hello", V: 42}).Subset("foo", "bar")
	insertCases := []struct {
		Want   string
		Have   *InsertStmt
		Values []interface{}
	}{
		{
			Want:   "INSERT INTO table (foo) VALUES (?)",
			Have:   Insert().Into("table").Fields(r1.Fields()...).Values(r1),
			Values: []interface{}{"hello"},
		},
		{
			Want:   "INSERT INTO table (foo) VALUES (?), (?)",
			Have:   Insert().Into("table").Fields(r1.Fields()...).Values(r1, r1),
			Values: []interface{}{"hello", "hello"},
		},
		{
			Want:   "INSERT INTO table (foo, bar) VALUES (?, ?), (?, ?)",
			Have:   Insert().Into("table").Fields(r2.Fields()...).Values(r2, r2),
			Values: []interface{}{"hello", 42, "hello", 42},
		},
		{
			Want:   "INSERT INTO table (foo, bar) VALUES (?, ?), (?, ?)",
			Have:   Insert().Into("table").Fields(r2.Fields()...).Values(r2).Values(r2),
			Values: []interface{}{"hello", 42, "hello", 42},
		},
	}

	for _, tc := range insertCases {
		t.Run("statement", func(t *testing.T) {
			have := tc.Have.String()
			if have != tc.Want {
				t.Fatalf("unexpected statement:\nhave: %s\nwant: %s\n", have, tc.Want)
			}
		})
		t.Run("values", func(t *testing.T) {
			have := tc.Have.QueryArgs()
			if !reflect.DeepEqual(have, tc.Values) {
				t.Fatalf("unexpected statement:\nhave: %v\nwant: %v\n", have, tc.Values)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	stmt := Update("table").Set(
		Assign().Equal("v", "world"),
	).Where(
		Cond().Equal("k", "hello"),
	)

	wantstr := "UPDATE table SET v=? WHERE k=?"
	havestr := stmt.String()
	if havestr != wantstr {
		t.Fatalf("unexpected statement:\nhave: %s\nwant: %s\n", havestr, wantstr)
	}
	wantval := []interface{}{"world", "hello"}
	haveval := stmt.QueryArgs()
	if !reflect.DeepEqual(haveval, wantval) {
		t.Fatalf("unexpected statement:\nhave: %v\nwant: %v\n", haveval, wantval)
	}
}

func TestDelete(t *testing.T) {
	stmt := Delete().From("table").Where(
		Cond().Equal("k", "hello").And().Equal("v", "world"),
	)

	wantstr := "DELETE FROM table WHERE k=? AND v=?"
	havestr := stmt.String()
	if havestr != wantstr {
		t.Fatalf("unexpected statement:\nhave: %s\nwant: %s\n", havestr, wantstr)
	}
	wantval := []interface{}{"hello", "world"}
	haveval := stmt.QueryArgs()
	if !reflect.DeepEqual(haveval, wantval) {
		t.Fatalf("unexpected statement:\nhave: %v\nwant: %v\n", haveval, wantval)
	}
}
