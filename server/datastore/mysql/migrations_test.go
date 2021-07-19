package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationStatus(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	if ds.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}

	require.Nil(t, ds.Drop())

	status, err := ds.MigrationStatus()
	require.Nil(t, err)
	assert.EqualValues(t, fleet.NoMigrationsCompleted, status)

	require.Nil(t, ds.MigrateTables())

	status, err = ds.MigrationStatus()
	require.Nil(t, err)
	assert.EqualValues(t, fleet.SomeMigrationsCompleted, status)

	require.Nil(t, ds.MigrateData())

	status, err = ds.MigrationStatus()
	require.Nil(t, err)
	assert.EqualValues(t, fleet.AllMigrationsCompleted, status)
}
