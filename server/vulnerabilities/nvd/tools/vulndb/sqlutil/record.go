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
	"fmt"
	"reflect"
	"unicode"
)

// Record represents a database record.
// See RecordType for mapping structs to Records.
type Record interface {
	Subset(...string) Record // Subset returns a subset of the record fields.
	Fields() []string        // Fields return the struct field name to be used as table column
	Values() []interface{}   // Values return the struct field value to be used in Scan
}

// Records implements the Record interface for a slice of Records.
type Records []Record

// NewRecords creates and initializes new Records from slice of any struct.
func NewRecords(T interface{}) Records {
	var r Records
	walkSlice(T, func(fv reflect.Value) {
		r = append(r, NewRecordType(fv.Interface()))
	})
	return r
}

// Subset returns Records with a subset of their fields.
//
// In order to use the Subset of Records (a []Record), one needs to type
// assert it: rows.Scan(NewRecords(&r).Subset("a", "b").(Records)...).
func (r Records) Subset(fields ...string) Record {
	s := make(Records, len(r))
	for i := 0; i < len(r); i++ {
		s[i] = r[i].Subset(fields...)
	}
	return s
}

// Fields returns a list of table column names from the first Record.
func (r Records) Fields() []string {
	if len(r) == 0 {
		return nil
	}
	return r[0].Fields()
}

// Values returns a list of table column values from all Records.
func (r Records) Values() []interface{} {
	if len(r) == 0 {
		return nil
	}
	v := make([]interface{}, 0, len(r)*len(r[0].Fields()))
	for _, record := range r {
		v = append(v, record.Values()...)
	}
	return v
}

// RecordType implements the Record interface for any struct with 'sql' tags as T.
type RecordType struct {
	T interface{}
}

// NewRecordType creates and initializes a new RecordType.
func NewRecordType(T interface{}) RecordType {
	return RecordType{T}
}

// Subset returns a subset of the struct fields from r.T.
func (r RecordType) Subset(fields ...string) Record {
	return newRecordSubset(r.Fields(), r.Values(), fields)
}

// Fields returns a list of table column names from struct fields.
// Uses the 'db' tag if available.
func (r RecordType) Fields() []string {
	var f []string
	const tag = "sql"
	walkStruct(r.T, func(ft reflect.StructField, fv reflect.Value) {
		n := ft.Tag.Get(tag)
		if n == "" {
			n = ft.Name
		}
		f = append(f, n)
	})
	return f
}

// Values returns a list of table column values from struct fields.
// These values are suitable for Row.Scan from Exec (e.g. INSERT, REPLACE) calls.
func (r RecordType) Values() []interface{} {
	var v []interface{}
	walkStruct(r.T, func(ft reflect.StructField, fv reflect.Value) {
		if fv.CanAddr() {
			v = append(v, fv.Addr().Interface())
		} else {
			v = append(v, fv.Interface())
		}
	})
	return v

}

type recordSubset struct {
	cols []string
	vals []interface{}
}

func newRecordSubset(cols []string, vals []interface{}, fields []string) *recordSubset {
	if len(cols) != len(vals) {
		panic("invalid cols/vals have different length")
	}
	inSubset := func(s string) bool {
		for _, f := range fields {
			if s == f {
				return true
			}
		}
		return false
	}
	rs := &recordSubset{}
	for i, col := range cols {
		if inSubset(col) {
			rs.cols = append(rs.cols, col)
			rs.vals = append(rs.vals, vals[i])
		}
	}
	return rs
}

func (rs *recordSubset) Subset(fields ...string) Record {
	return newRecordSubset(rs.cols, rs.vals, fields)
}

func (rs *recordSubset) Fields() []string      { return rs.cols }
func (rs *recordSubset) Values() []interface{} { return rs.vals }

// walkStruct walks the exported fields of a struct using reflection,
// and calls fn for each field.
func walkStruct(s interface{}, fn func(reflect.StructField, reflect.Value)) {
	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}
	if v.Kind() != reflect.Struct {
		panic(fmt.Sprintf("walkStruct: s is not struct: %T", s))
	}
	for i := 0; i < v.NumField(); i++ {
		ft := t.Field(i)
		if !exportedField(ft) {
			continue
		}
		fn(ft, v.Field(i))
	}
}

func exportedField(f reflect.StructField) bool {
	return !f.Anonymous && unicode.IsUpper(rune(f.Name[0]))
}

// walkSlice walks slice s using reflection, and calls fn for each element.
func walkSlice(s interface{}, fn func(reflect.Value)) {
	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)
	if t.Kind() != reflect.Slice {
		panic(fmt.Sprintf("walkSlice: s is not slice: %T", s))
	}
	for i := 0; i < v.Len(); i++ {
		fn(v.Index(i))
	}
}
