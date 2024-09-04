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

package cvefeed

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd/schema"
)

var cpeParts = map[string]string{
	"a": "application",
	"h": "hardware",
	"o": "operating system",
}

type stack struct {
	items  []string
	rwLock sync.RWMutex
}

func (s *stack) push(item string) {
	s.rwLock.Lock()
	defer s.rwLock.Unlock()
	if s.items == nil {
		s.items = []string{}
	}
	s.items = append(s.items, item)
}

func (s *stack) pop() (string, bool) {
	if s.isEmpty() {
		return "", false
	}
	s.rwLock.Lock()
	defer s.rwLock.Unlock()
	item := s.items[len(s.items)-1]
	s.items = s.items[0 : len(s.items)-1]
	return item, true
}

func (s *stack) isEmpty() bool {
	s.rwLock.Lock()
	defer s.rwLock.Unlock()
	return len(s.items) == 0
}

// Stats contains the stats information of a NVD JSON feed
type Stats struct {
	totalCVEs         int64
	totalRules        int64
	totalRulesWithAND int64
	operatorANDs      map[string]int64
}

// NewStats creates a new Stats object
func NewStats() *Stats {
	s := Stats{}
	s.Reset()
	return &s
}

// Reset clears out a Stats object
func (s *Stats) Reset() {
	s.totalCVEs = 0
	s.totalRules = 0
	s.totalRulesWithAND = 0
	s.operatorANDs = make(map[string]int64)
}

// ReportOperatorAND prints the stats of operator AND
func (s *Stats) ReportOperatorAND() {
	if s.totalRulesWithAND <= 0 {
		fmt.Println("No rules found with AND operator.")
		return
	}
	keys := make([]string, 0, len(s.operatorANDs))
	for key := range s.operatorANDs {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return s.operatorANDs[keys[i]] > s.operatorANDs[keys[j]]
	})
	fmt.Printf("Total rules with AND operator: %0.2f%%\n", percentage(s.totalRulesWithAND, s.totalRules))
	for _, key := range keys {
		fmt.Printf("%05.2f%%: %s\n", percentage(s.operatorANDs[key], s.totalRulesWithAND), key)
	}
}

// Gather feeds a Stats object by gathering stats from a NVD JSON feed dictionary
func (s *Stats) Gather(dict Dictionary) {
	for key := range dict {
		s.totalCVEs++
		schema := dict[key].(*nvd.Vuln).Schema()
		for _, node := range schema.Configurations.Nodes {
			s.totalRules++
			rule := flattenRule(node, &stack{})
			if strings.Contains(rule, "AND") {
				s.totalRulesWithAND++
				s.operatorANDs[rule]++
			}
		}
	}
}

func flattenRule(node *schema.NVDCVEFeedJSON10DefNode, operators *stack) string {
	cpePart := ""
	operators.push(node.Operator)
	switch {
	case len(node.Children) > 0:
		outputs := []string{}
		for _, c := range node.Children {
			outputs = append(outputs, flattenRule(c, operators))
		}
		operator, _ := operators.pop()
		return fmt.Sprintf("(%s)", strings.Join(outputs, fmt.Sprintf(" %s ", operator)))
	case len(node.CPEMatch) > 0:
		for _, cpeMatch := range node.CPEMatch {
			cpeItems := strings.Split(cpeMatch.Cpe23Uri, ":")
			if len(cpeItems) > 2 {
				part := cpeItems[2]
				if _, ok := cpeParts[part]; ok && !strings.Contains(cpePart, part) {
					cpePart += part
				}
			}
		}
		operator, _ := operators.pop()
		if len(cpePart) > 1 {
			return fmt.Sprintf("(%s)", strings.Join(strings.Split(cpePart, ""), fmt.Sprintf(" %s ", operator)))
		}
	}
	return cpePart
}

func percentage(partial, total int64) (delta float64) {
	delta = (float64(partial) / float64(total)) * 100
	return
}
