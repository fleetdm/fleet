package oval_parsed

import "github.com/fleetdm/fleet/v4/server/fleet"

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
func (r Definition) Eval(OSTstResults map[int]bool, pkgTstResults map[int][]fleet.Software) bool {
	if r.Criteria == nil || (len(OSTstResults) == 0 && len(pkgTstResults) == 0) {
		return false
	}

	return evalCriteria(r.Criteria, OSTstResults, pkgTstResults)
}

func (r Definition) CollectTestIds() []int {
	if r.Criteria == nil {
		return nil
	}

	var results []int
	queue := []*Criteria{r.Criteria}

	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		results = append(results, next.Criteriums...)
		queue = append(queue, next.Criterias...)
	}

	return results
}

func evalCriteria(c *Criteria, OSTstResults map[int]bool, pkgTstResults map[int][]fleet.Software) bool {
	var vals []bool
	var result bool

	for _, co := range c.Criteriums {
		if _, ok := pkgTstResults[co]; ok {
			r := len(pkgTstResults[co]) > 0
			vals = append(vals, r)
		}

		if v, ok := OSTstResults[co]; ok {
			vals = append(vals, v)
		}
	}
	result = c.Operator.Eval(vals...)

	for _, ci := range c.Criterias {
		return c.Operator.Eval(result, evalCriteria(ci, OSTstResults, pkgTstResults))
	}

	return result
}
