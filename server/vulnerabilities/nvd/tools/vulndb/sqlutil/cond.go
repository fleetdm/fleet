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
	"strings"
)

// QueryConditionSet represents a set of database query conditions.
type QueryConditionSet struct {
	qs []string
	qv []interface{}
}

// Cond returns an empty QueryConditioSet.
func Cond() *QueryConditionSet {
	return &QueryConditionSet{}
}

// Equal adds k=v to the set.
func (c *QueryConditionSet) Equal(k string, v interface{}) *QueryConditionSet {
	c.qs = append(c.qs, fmt.Sprintf("%s=?", k))
	c.qv = append(c.qv, v)
	return c
}

// In adds k IN (v) to the condition set.
// v must be a slice of any type or a *SelectStmt.
func (c *QueryConditionSet) In(k string, v interface{}) *QueryConditionSet {
	n := 0
	walkSlice(v, func(fv reflect.Value) {
		n++
		c.qv = append(c.qv, fv.Interface())
	})
	c.qs = append(c.qs, fmt.Sprintf("%s IN %s", k, genbindgroup(n)))
	return c
}

// InSelect adds k IN (select) to the condition set.
func (c *QueryConditionSet) InSelect(k string, v *SelectStmt) *QueryConditionSet {
	c.qs = append(c.qs, fmt.Sprintf("%s IN (%s)", k, v.String()))
	c.qv = append(c.qv, v.values...)
	return c
}

// And adds AND to the set.
func (c *QueryConditionSet) And() *QueryConditionSet {
	c.qs = append(c.qs, "AND")
	return c
}

// Or adds OR to the set.
func (c *QueryConditionSet) Or() *QueryConditionSet {
	c.qs = append(c.qs, "OR")
	return c
}

// Not adds NOT to the set.
func (c *QueryConditionSet) Not() *QueryConditionSet {
	c.qs = append(c.qs, "NOT")
	return c
}

// IsNull adds v IS NULL to the query.
func (c *QueryConditionSet) IsNull(v string) *QueryConditionSet {
	c.qs = append(c.qs, fmt.Sprintf("%s IS NULL", v))
	return c
}

// Group adds (cond) to the set.
func (c *QueryConditionSet) Group(cond *QueryConditionSet) *QueryConditionSet {
	c.qs = append(c.qs, fmt.Sprintf("(%s)", cond.String()))
	c.qv = append(c.qv, cond.qv...)
	return c
}

// Literal adds literal string l to the condition.
func (c *QueryConditionSet) Literal(l string) *QueryConditionSet {
	c.qs = append(c.qs, l)
	return c
}

// String returns the query.
func (c *QueryConditionSet) String() string {
	return strings.Join(c.qs, " ")
}

// Values returns the values associated to each condition.
func (c *QueryConditionSet) Values() []interface{} {
	return c.qv
}
