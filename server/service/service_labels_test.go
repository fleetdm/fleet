package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLabel(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	svc := newTestService(ds, nil, nil)

	label := &fleet.Label{
		Name:  "foo",
		Query: "select * from foo;",
	}
	label, err := ds.NewLabel(label)
	assert.Nil(t, err)
	assert.NotZero(t, label.ID)

	labelVerify, err := svc.GetLabel(test.UserContext(test.UserAdmin), label.ID)
	assert.Nil(t, err)
	assert.Equal(t, label.ID, labelVerify.ID)
}

func TestGetLabels(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	require.NoError(t, ds.MigrateTables())
	require.NoError(t, ds.MigrateData())

	svc := newTestService(ds, nil, nil)

	labels, err := svc.ListLabels(test.UserContext(test.UserAdmin), fleet.ListOptions{Page: 0, PerPage: 1000})
	require.NoError(t, err)
	require.Len(t, labels, 7)
}
