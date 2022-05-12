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
func (r Definition) Eval(testResults map[int][]uint) bool {
	if r.Criteria == nil || len(testResults) == 0 {
		return false
	}

	return evalCriteria(r.Criteria, testResults)
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

func evalCriteria(c *Criteria, testResults map[int][]uint) bool {
	var vals []bool
	var result bool

	for _, co := range c.Criteriums {
		r := len(testResults[co]) > 0
		vals = append(vals, r)
	}
	result = c.Operator.Eval(vals...)

	for _, ci := range c.Criterias {
		return c.Operator.Eval(result, evalCriteria(ci, testResults))
	}

	return result
}
