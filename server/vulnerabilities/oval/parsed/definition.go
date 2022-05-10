package oval_parsed

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
func (r Definition) Eval(testResults map[int]bool) bool {
	if r.Criteria == nil || len(testResults) == 0 {
		return false
	}

	return evalCriteria(r.Criteria, testResults)
}

func evalCriteria(c *Criteria, testResults map[int]bool) bool {
	var vals []bool
	var result bool

	for _, co := range c.Criteriums {
		vals = append(vals, testResults[co])
	}
	result = c.Operator.Eval(vals...)

	for _, ci := range c.Criterias {
		return c.Operator.Eval(result, evalCriteria(ci, testResults))
	}

	return result
}
