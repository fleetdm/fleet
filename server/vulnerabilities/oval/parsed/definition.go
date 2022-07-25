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
func (d Definition) Eval(OSTstResults map[int]bool, pkgTstResults map[int][]fleet.Software) bool {
	if d.Criteria == nil || (len(OSTstResults) == 0 && len(pkgTstResults) == 0) {
		return false
	}

	rEval, err := evalCriteria(d.Criteria, OSTstResults, pkgTstResults)
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

func evalCriteria(c *Criteria, OSTstResults map[int]bool, pkgTstResults map[int][]fleet.Software) (bool, error) {
	var vals []bool
	var result bool

	for _, co := range c.Criteriums {
		pkgTstR, pkgOk := pkgTstResults[co]
		if pkgOk {
			vals = append(vals, len(pkgTstR) > 0)
		}

		OSTstR, OSTstOk := OSTstResults[co]
		if OSTstOk {
			vals = append(vals, OSTstR)
		}

		if !pkgOk && !OSTstOk {
			return false, fmt.Errorf("test not found: %d", co)
		}
	}

	result = c.Operator.Eval(vals...)

	for _, ci := range c.Criterias {
		rEval, err := evalCriteria(ci, OSTstResults, pkgTstResults)
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
