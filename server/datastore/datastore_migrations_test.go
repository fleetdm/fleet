package datastore

import (
	"testing"

	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/require"
)

func testMigrationStatus(t *testing.T, ds kolide.Datastore) {
	if ds.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}

	require.Nil(t, ds.Drop())
	require.NotNil(t, ds.MigrationStatus())

	require.Nil(t, ds.MigrateTables())
	require.NotNil(t, ds.MigrationStatus())

	// Should return nil with all migrations completed
	require.Nil(t, ds.MigrateData())
	require.Nil(t, ds.MigrationStatus())
}
