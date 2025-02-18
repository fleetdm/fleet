package oval_parsed

import (
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Criteria is used to express an arbitrary logic tree.
// Each node in the tree references a particular test.
type Criteria struct {
	Operator   OperatorType
	Criteriums []int
	Criterias  []*Criteria
}

// Definition is a container of one or more criteria and one or more vulnerabilities.
// If the logic tree expressed by the Criterias evaluates to true, then we say that
// a host is susceptible to `Vulnerabilities`.
type Definition struct {
	Criteria        *Criteria
	Vulnerabilities []string
}

// Eval evaluates the given definition using the provided test results.
// Tests results can come from two sources:
// - OSTstResults: Test results from making assertions against the installed OS Version
// - pkTstResults: Tests results from making assertions against the installed software packages.
func (d Definition) Eval(osTstResults map[int]bool, pkgTstResults map[int][]fleet.Software) bool {
	if d.Criteria == nil || (len(osTstResults) == 0 && len(pkgTstResults) == 0) {
		return false
	}

	rEval, err := evalCriteria(d.Criteria, osTstResults, pkgTstResults)
	if err != nil {
		return false
	}
	return rEval
}

func (d Definition) CollectTestIds() []int {
	if d.Criteria == nil {
		return nil
	}

	var results []int
	queue := []*Criteria{d.Criteria}

	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		results = append(results, next.Criteriums...)
		queue = append(queue, next.Criterias...)
	}

	return results
}

func evalCriteria(c *Criteria, osTstResults map[int]bool, pkgTstResults map[int][]fleet.Software) (bool, error) {
	var vals []bool
	var result bool

	for _, co := range c.Criteriums {
		pkgTstR, pkgOk := pkgTstResults[co]
		if pkgOk {
			vals = append(vals, len(pkgTstR) > 0)
		}

		OSTstR, OSTstOk := osTstResults[co]
		if OSTstOk {
			vals = append(vals, OSTstR)
		}

		if !pkgOk && !OSTstOk {
			return false, fmt.Errorf("test not found: %d", co)
		}
	}

	result = c.Operator.Eval(vals...)

	for _, ci := range c.Criterias {
		rEval, err := evalCriteria(ci, osTstResults, pkgTstResults)
		if err != nil {
			return false, err
		}
		result = c.Operator.Eval(result, rEval)
	}

	return result, nil
}

// CveVulnerabilities Returns only CVE vulnerabilities, excluding any 'advisory'
// entries. 'Advisory' entries are excluded because we only want to report entries for which we
// might have a NVD link.
func (d Definition) CveVulnerabilities() []string {
	var r []string
	for _, v := range d.Vulnerabilities {
		if strings.HasPrefix(strings.ToLower(v), "cve") {
			r = append(r, v)
		}
	}
	return r
}

// intersect returns the intersection of two slices of uints.
func intersect(a, b []uint) []uint {
	m := make(map[uint]bool)
	for _, v := range a {
		m[v] = true
	}

	var r []uint
	for _, v := range b {
		if m[v] {
			r = append(r, v)
		}
	}
	return r
}

// unionAll returns the union of two slices of uints without duplicates.
func unionAll(a, b []uint) []uint {
	m := make(map[uint]bool)
	var result []uint

	for _, v := range a {
		if !m[v] {
			m[v] = true
			result = append(result, v)
		}
	}

	for _, v := range b {
		if !m[v] {
			m[v] = true
			result = append(result, v)
		}
	}

	return result
}

// findMatchingSoftware returns the software IDs that match the given OVAL criteria.
func findMatchingSoftware(c Criteria, uTests map[int][]uint) []uint {
	switch c.Operator {
	case And:
		return findAndMatch(c, uTests)
	case Or:
		return findOrMatch(c, uTests)
	}
	return nil
}

// findAndMatch finds the software that matches all the criteria using the AND operator
func findAndMatch(c Criteria, uTests map[int][]uint) []uint {
	if c.Criteriums != nil {
		return intersectSoftware(c.Criteriums, uTests)
	}

	matchingSoftware := make([]uint, 0)
	for _, subCriteria := range c.Criterias {
		subMatchingSoftware := findMatchingSoftware(*subCriteria, uTests)
		if len(matchingSoftware) == 0 {
			matchingSoftware = subMatchingSoftware
		} else {
			matchingSoftware = intersect(matchingSoftware, subMatchingSoftware)
		}
	}
	return matchingSoftware
}

// intersectSoftware returns the intersection of the software IDs for the given criteria.
func intersectSoftware(criteriums []int, uTests map[int][]uint) []uint {
	if len(criteriums) == 0 {
		return nil
	}

	softwareSets := make([][]uint, 0, len(criteriums))
	for _, c := range criteriums {
		softwareSets = append(softwareSets, uTests[c])
	}

	intersected := softwareSets[0]
	for _, s := range softwareSets[1:] {
		intersected = intersect(intersected, s)
	}

	return intersected
}

// findOrMatch finds the software that matches any of the criteria using the OR operator
func findOrMatch(c Criteria, uTests map[int][]uint) []uint {
	matchingSoftware := make([]uint, 0)
	for _, subCriteria := range c.Criterias {
		subMatchingSoftware := findMatchingSoftware(*subCriteria, uTests)
		matchingSoftware = unionAll(matchingSoftware, subMatchingSoftware)
	}
	return matchingSoftware
}
