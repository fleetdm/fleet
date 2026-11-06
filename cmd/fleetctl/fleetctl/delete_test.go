package fleetctl

import (
	"context"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/assert"
)

func TestDeleteLabel(t *testing.T) {
	_, ds := testing_utils.RunServerWithMockedDS(t, &service.TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}})

	global := &fleet.LabelSpec{ID: 7, Name: "pending_updates"}
	onFleet3 := &fleet.LabelSpec{ID: 8, Name: "shared", TeamID: new(uint(3))}
	onFleet4 := &fleet.LabelSpec{ID: 9, Name: "shared", TeamID: new(uint(4))}
	ds.GetLabelSpecsFunc = func(ctx context.Context, filter fleet.TeamFilter) ([]*fleet.LabelSpec, error) {
		switch {
		case filter.TeamID == nil || *filter.TeamID == 0:
			return []*fleet.LabelSpec{global}, nil
		case *filter.TeamID == 3:
			return []*fleet.LabelSpec{onFleet3}, nil
		default:
			return []*fleet.LabelSpec{onFleet4}, nil
		}
	}
	ds.LabelFunc = testing_utils.LabelsByID(global, onFleet3, onFleet4)
	var deleted []string
	ds.DeleteLabelFunc = func(ctx context.Context, name string, filter fleet.TeamFilter) error {
		deleted = append(deleted, fmt.Sprintf("%s@%d", name, ptr.ValOrZero(filter.TeamID)))
		return nil
	}

	name := writeTmpYml(t, `---
apiVersion: v1
kind: label
spec:
  name: pending_updates
  query: select 1;
  platforms:
    - darwin
---
apiVersion: v1
kind: label
spec:
  name: shared
  fleet_id: 3
`)

	// Fleets can share label names, so each spec's label is found among its own fleet's labels and deleted by ID.
	assert.Empty(t, runAppForTest(t, []string{"delete", "-f", name}))
	assert.Equal(t, []string{"pending_updates@0", "shared@3"}, deleted)
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
