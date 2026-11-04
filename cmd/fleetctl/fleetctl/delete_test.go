package fleetctl

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteLabel(t *testing.T) {
	_, ds := testing_utils.RunServerWithMockedDS(t)

	var deletedLabel string
	ds.DeleteLabelFunc = func(ctx context.Context, name string, filter fleet.TeamFilter) error {
		deletedLabel = name
		return nil
	}
	ds.LabelByNameFunc = func(ctx context.Context, name string, filter fleet.TeamFilter) (*fleet.Label, error) {
		return &fleet.Label{Name: name}, nil
	}

	name := writeTmpYml(t, `---
apiVersion: v1
kind: label
spec:
  name: pending_updates
  query: select 1;
  platforms:
    - darwin
`)

	assert.Empty(t, runAppForTest(t, []string{"delete", "-f", name}))
	assert.True(t, ds.DeleteLabelFuncInvoked)
	assert.Equal(t, "pending_updates", deletedLabel)
}

func TestDeletePackRemoved(t *testing.T) {
	_, ds := testing_utils.RunServerWithMockedDS(t)

	name := writeTmpYml(t, `---
apiVersion: v1
kind: report
spec:
  name: query1
  query: SELECT 1
---
apiVersion: v1
kind: pack
spec:
  name: pack1
`)

	_, err := runAppNoChecks([]string{"delete", "-f", name})
	require.ErrorContains(t, err, `kind "pack" is no longer supported`)
	require.False(t, ds.DeleteQueryFuncInvoked)
}

func TestDeleteQuery(t *testing.T) {
	_, ds := testing_utils.RunServerWithMockedDS(t)

	var deletedQuery string
	ds.DeleteQueryFunc = func(ctx context.Context, teamID *uint, name string) error {
		deletedQuery = name
		return nil
	}
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		if name != "query1" {
			return nil, nil
		}
		return &fleet.Query{
			ID:             33,
			Name:           "query1",
			Description:    "some desc",
			Query:          "select 1;",
			Saved:          false,
			ObserverCanRun: false,
		}, nil
	}
	name := writeTmpYml(t, `---
apiVersion: v1
kind: query
spec:
  description: some desc
  name: query1
  query: select 1;
`)

	assert.Empty(t, runAppForTest(t, []string{"delete", "-f", name}))
	assert.True(t, ds.DeleteQueryFuncInvoked)
	assert.Equal(t, "query1", deletedQuery)
}
