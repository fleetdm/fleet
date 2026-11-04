package fleetctl

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
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

func TestDeletePack(t *testing.T) {
	_, ds := testing_utils.RunServerWithMockedDS(t)

	var deletedPack string
	ds.DeletePackFunc = func(ctx context.Context, name string) error {
		deletedPack = name
		return nil
	}
	ds.PackByNameFunc = func(ctx context.Context, name string, opts ...fleet.OptionalArg) (*fleet.Pack, bool, error) {
		if name != "pack1" {
			return nil, false, nil
		}
		return &fleet.Pack{
			ID:          7,
			Name:        "pack1",
			Description: "some desc",
			Platform:    "darwin",
			Disabled:    false,
		}, true, nil
	}
	name := writeTmpYml(t, `---
apiVersion: v1
kind: pack
spec:
  description: some desc
  disabled: false
  id: 7
  name: pack1
  platform: darwin
  targets:
    labels: null
`)

	assert.Empty(t, runAppForTest(t, []string{"delete", "-f", name}))
	assert.True(t, ds.DeletePackFuncInvoked)
	assert.Equal(t, "pack1", deletedPack)
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

func TestDeleteReportWithNumericName(t *testing.T) {
	_, ds := testing_utils.RunServerWithMockedDS(t)

	var deletedReport string
	ds.DeleteQueryFunc = func(ctx context.Context, teamID *uint, name string) error {
		deletedReport = name
		return nil
	}
	ds.QueryByNameFunc = func(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
		return &fleet.Query{ID: 7, Name: name, Query: "select 1;"}, nil
	}
	name := writeTmpYml(t, `---
apiVersion: v1
kind: report
spec:
  name: "42"
  query: select 1;
`)

	assert.Empty(t, runAppForTest(t, []string{"delete", "-f", name}))
	assert.True(t, ds.DeleteQueryFuncInvoked)
	assert.Equal(t, "42", deletedReport)
	assert.False(t, ds.QueryFuncInvoked, "report name was routed as an ID")
}
