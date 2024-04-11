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
	"strings"
)

// AssignmentList represents a list of assignments for the UPDATE command.
type AssignmentList struct {
	qs []string
	qv []interface{}
}

// Assign returns an empty AssignmentList.
func Assign() *AssignmentList {
	return &AssignmentList{}
}

// Literal adds literal string l to the list.
func (al *AssignmentList) Literal(l string) *AssignmentList {
	al.qs = append(al.qs, l)
	return al
}

// Equal adds k=v to the list.
func (al *AssignmentList) Equal(k string, v interface{}) *AssignmentList {
	al.qs = append(al.qs, fmt.Sprintf("%s=?", k))
	al.qv = append(al.qv, v)
	return al
}

// String returns the query.
func (al *AssignmentList) String() string {
	return strings.Join(al.qs, ", ")
}

// Values returns the values associated to each assignment.
func (al *AssignmentList) Values() []interface{} {
	return al.qv
}
