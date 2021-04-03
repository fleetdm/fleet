package datastore

import (
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMigrationStatus(t *testing.T, ds kolide.Datastore) {
	if ds.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}

	require.Nil(t, ds.Drop())

	status, err := ds.MigrationStatus()
	require.Nil(t, err)
	assert.EqualValues(t, kolide.NoMigrationsCompleted, status)

	require.Nil(t, ds.MigrateTables())

	status, err = ds.MigrationStatus()
	require.Nil(t, err)
	assert.EqualValues(t, kolide.SomeMigrationsCompleted, status)

	require.Nil(t, ds.MigrateData())

	status, err = ds.MigrationStatus()
	require.Nil(t, err)
	assert.EqualValues(t, kolide.AllMigrationsCompleted, status)
}
