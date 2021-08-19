package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGlobalPolicy(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	q, err := ds.NewQuery(&fleet.Query{
		Name:        "query1",
		Description: "query1 desc",
		Query:       "select 1;",
		Saved:       true,
	})
	require.NoError(t, err)
	p, err := ds.NewGlobalPolicy(q.ID)
	require.NoError(t, err)

	assert.Equal(t, "query1", p.QueryName)

	q2, err := ds.NewQuery(&fleet.Query{
		Name:        "query2",
		Description: "query2 desc",
		Query:       "select 42;",
		Saved:       true,
	})
	require.NoError(t, err)
	_, err = ds.NewGlobalPolicy(q.ID)
	require.NoError(t, err)

	policies, err := ds.ListGlobalPolicies()
	require.NoError(t, err)
	require.Len(t, policies, 2)
	assert.Equal(t, q.ID, policies[0].ID)
	assert.Equal(t, q2.ID, policies[1].ID)

	_, err = ds.DeleteGlobalPolicies([]uint{policies[0].ID, policies[1].ID})
	require.NoError(t, err)

	policies, err = ds.ListGlobalPolicies()
	require.NoError(t, err)
	require.Len(t, policies, 0)
}
