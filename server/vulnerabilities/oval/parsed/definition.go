package oval_parsed

type Criteria struct {
	Operator   OperatorType
	Criteriums []int
	Criterias  []*Criteria
}

type Definition struct {
	Criteria        *Criteria
	Vulnerabilities []string
}

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
