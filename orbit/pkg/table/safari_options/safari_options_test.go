//go:build darwin
// +build darwin

package safari_options

import (
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var myConstraintsList table.ConstraintList
	var myConstraints []table.Constraint
	var tbl table.QueryContext
	tbl.Constraints = make(map[string]table.ConstraintList)

	myConstraint := table.Constraint{2, "some-user"}
	myConstraints = append(myConstraints, myConstraint)
	myConstraintsList = table.ConstraintList{"TEXT", myConstraints}
	tbl.Constraints["user_name"] = myConstraintsList

	_, err := Generate(ctx, tbl)
	require.Error(t, err)
}

func TestGetUserNameFromConstraints(t *testing.T) {
	var myConstraintsList table.ConstraintList
	var myConstraints []table.Constraint
	var tbl table.QueryContext
	tbl.Constraints = make(map[string]table.ConstraintList)

	myConstraint := table.Constraint{2, "some-user"}
	myConstraints = append(myConstraints, myConstraint)
	myConstraintsList = table.ConstraintList{"TEXT", myConstraints}
	tbl.Constraints["user_name"] = myConstraintsList

	user := getUserNameFromConstraints(tbl)
	require.Equal(t, "some-user", user)
}
