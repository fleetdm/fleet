package tablehelpers

import (
	"github.com/osquery/osquery-go/plugin/table"
)

func MockQueryContext(constraints map[string][]string) table.QueryContext {
	queryContext := table.QueryContext{
		Constraints: make(map[string]table.ConstraintList, len(constraints)),
	}

	for columnName, constraintExpressions := range constraints {
		tableConstraints := make([]table.Constraint, len(constraintExpressions))
		for i, c := range constraintExpressions {
			tableConstraints[i].Expression = c
		}
		queryContext.Constraints[columnName] = table.ConstraintList{Constraints: tableConstraints}
	}
	return queryContext
}
