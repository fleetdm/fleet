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
	"strings"
)

type baseStmt struct {
	q []string
}

func (s *baseStmt) add(parts ...string) *baseStmt {
	s.q = append(s.q, parts...)
	return s
}

func (s *baseStmt) join(parts []string) *baseStmt {
	s.q = append(s.q, strings.Join(parts, ", "))
	return s
}

func (s *baseStmt) group(g *baseStmt, as string) *baseStmt {
	s.add("(").add(g.q...).add(")")
	if as != "" {
		s.add("AS").add(as)
	}
	return s
}

// String returns the statement.
func (s *baseStmt) String() string {
	if len(s.q) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(s.q[0])
	for i := 1; i < len(s.q); i++ {
		if s.q[i-1] != "(" && s.q[i] != ")" && s.q[i] != "," {
			sb.WriteString(" ")
		}
		sb.WriteString(s.q[i])
	}
	return sb.String()
}

func genbindgroup(n int) string {
	if n == 0 {
		return ""
	}
	var s strings.Builder
	s.WriteString("(?")
	for i := 1; i < n; i++ {
		s.WriteString(", ?")
	}
	s.WriteString(")")
	return s.String()
}

// InsertStmt represents the INSERT statement.
type InsertStmt struct {
	baseStmt
	values []interface{}
}

// Insert creates and initializes a new INSERT statement.
func Insert() *InsertStmt {
	s := &InsertStmt{}
	s.add("INSERT")
	return s
}

// Replace creates and initializes a new REPLACE statement.
// Backed by InsertStmt to be used interchangeably.
func Replace() *InsertStmt {
	s := &InsertStmt{}
	s.add("REPLACE")
	return s
}

// Into adds the INTO part of the statement.
func (s *InsertStmt) Into(table string) *InsertStmt {
	s.add("INTO", table)
	return s
}

// Fields adds the (f1, fN) part of the insert part of the statement.
func (s *InsertStmt) Fields(fields ...string) *InsertStmt {
	s.add("(").join(fields).add(")")
	return s
}

// Values adds the (v1, vN) part of the insert part of the statement.
// Each record is added to the statement as a set of bindings (?, ?), and
// their values recorded. Use QueryArgs to get the values recorded.
func (s *InsertStmt) Values(records ...Record) *InsertStmt {
	if len(records) == 0 {
		return s
	}
	if len(s.values) == 0 {
		s.add("VALUES")
	} else {
		s.add(",")
	}
	g := make([]string, len(records))
	for i, r := range records {
		values := r.Values()
		if len(values) == 0 {
			continue
		}
		g[i] = genbindgroup(len(values))
		s.values = append(s.values, values...)
	}
	s.join(g)
	return s
}

// Select adds a Select to the statement.
func (s *InsertStmt) Select(stmt *SelectStmt) *InsertStmt {
	s.add(stmt.q...)
	s.values = append(s.values, stmt.values...)
	return s
}

// Literal adds literal string l to the statement.
func (s *InsertStmt) Literal(l string) *InsertStmt {
	s.add(l)
	return s
}

// QueryArgs returns the values corresponding to bindings (?, ?) from
// all calls to Values. e.g. db.Exec(stmt.String(), stmt.QueryArgs()...)
func (s *InsertStmt) QueryArgs() []interface{} {
	return s.values
}

// UpdateStmt represents the UPDATE statement.
type UpdateStmt struct {
	baseStmt
	values []interface{}
}

// Update creates and initializes a new UPDATE statement.
func Update(tables ...string) *UpdateStmt {
	s := &UpdateStmt{}
	s.add("UPDATE").join(tables)
	return s
}

// Set adds the SET part of the statement.
func (s *UpdateStmt) Set(al *AssignmentList) *UpdateStmt {
	s.add("SET").add(al.String())
	s.values = append(s.values, al.Values()...)
	return s
}

// Where adds the WHERE part of the statement.
func (s *UpdateStmt) Where(cond *QueryConditionSet) *UpdateStmt {
	s.add("WHERE").add(cond.String())
	s.values = append(s.values, cond.Values()...)
	return s
}

// QueryArgs returns the values corresponding to bindings (?, ?) from
// all calls to Set and Where. e.g. db.Exec(stmt.String(), stmt.QueryArgs()...)
func (s *UpdateStmt) QueryArgs() []interface{} {
	return s.values
}

// SelectStmt represents a SELECT statement.
type SelectStmt struct {
	baseStmt
	values []interface{}
}

// Select creates and initializes a new SELECT statement.
func Select(fields ...string) *SelectStmt {
	s := &SelectStmt{}
	s.add("SELECT").join(fields)
	return s
}

// Select adds another Select statement to the statement.
// e.g. Select("*").From().Select(...)
func (s *SelectStmt) Select(a *SelectStmt) *SelectStmt {
	s.add(a.baseStmt.q...)
	s.values = append(s.values, a.values...)
	return s
}

// SelectGroup adds another SelectStmt to the statement, as a group, with
// an optional alias.
func (s *SelectStmt) SelectGroup(as string, g *SelectStmt) *SelectStmt {
	s.group(&g.baseStmt, as)
	s.values = append(s.values, g.values...)
	return s
}

// From adds the FROM part of the statement.
func (s *SelectStmt) From(tables ...string) *SelectStmt {
	s.add("FROM")
	if len(tables) > 0 {
		s.join(tables)
	}
	return s
}

// Where adds the WHERE statement followed by conditions to the statement.
func (s *SelectStmt) Where(cond *QueryConditionSet) *SelectStmt {
	s.add("WHERE").add(cond.String())
	s.values = append(s.values, cond.Values()...)
	return s
}

// Literal adds literal string l to the statement.
// Useful for e.g. UNION, JOIN, LIMIT.
func (s *SelectStmt) Literal(l string) *SelectStmt {
	s.add(l)
	return s
}

// QueryArgs returns the values corresponding to bindings (?, ?) from
// all calls to Where. e.g. db.Query(stmt.String(), stmt.QueryArgs()...)
func (s *SelectStmt) QueryArgs() []interface{} {
	return s.values
}

// DeleteStmt represents a DELETE statement.
type DeleteStmt struct {
	baseStmt
	values []interface{}
}

// Delete creates and initializes a new DELETE statement.
func Delete() *DeleteStmt {
	s := &DeleteStmt{}
	s.add("DELETE")
	return s
}

// From adds the FROM part of the statement.
func (s *DeleteStmt) From(tables ...string) *DeleteStmt {
	s.add("FROM").join(tables)
	return s
}

// Where adds the WHERE statement followed by conditions to the statement.
func (s *DeleteStmt) Where(cond *QueryConditionSet) *DeleteStmt {
	s.add("WHERE").add(cond.String())
	s.values = append(s.values, cond.Values()...)
	return s
}

// QueryArgs returns the values corresponding to bindings (?, ?) from
// all calls to Where. e.g. db.Exec(stmt.String(), stmt.QueryArgs()...)
func (s *DeleteStmt) QueryArgs() []interface{} {
	return s.values
}

// Literal adds literal string l to the statement.
func (s *DeleteStmt) Literal(l string) *DeleteStmt {
	s.add(l)
	return s
}
