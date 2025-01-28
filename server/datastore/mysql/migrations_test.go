package mysql

import (
	"context"
	"os/exec"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/tables"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(victor): Add test for feature migrations. Or simply combine feature migrations into existing MigrateTables

func TestMigrationStatus(t *testing.T) {
	ds := createMySQLDSForMigrationTests(t, t.Name())
	t.Cleanup(func() {
		ds.Close()
	})

	status, err := ds.MigrationStatus(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, fleet.NoMigrationsCompleted, status.StatusCode)
	assert.Empty(t, status.MissingTable)
	assert.Empty(t, status.MissingData)

	require.Nil(t, ds.MigrateTables(context.Background()))

	status, err = ds.MigrationStatus(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, fleet.SomeMigrationsCompleted, status.StatusCode)
	assert.NotEmpty(t, status.MissingData)

	require.Nil(t, ds.MigrateData(context.Background()))

	status, err = ds.MigrationStatus(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, fleet.AllMigrationsCompleted, status.StatusCode)
	assert.Empty(t, status.MissingTable)
	assert.Empty(t, status.MissingData)

	// Insert unknown migration.
	_, err = ds.writer(context.Background()).Exec(`INSERT INTO ` + tables.MigrationClient.TableName + ` (version_id, is_applied) VALUES (1638994765, 1)`)
	require.NoError(t, err)
	status, err = ds.MigrationStatus(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, fleet.UnknownMigrations, status.StatusCode)
	_, err = ds.writer(context.Background()).Exec(`DELETE FROM ` + tables.MigrationClient.TableName + ` WHERE version_id = 1638994765`)
	require.NoError(t, err)

	status, err = ds.MigrationStatus(context.Background())
	require.NoError(t, err)
	assert.EqualValues(t, fleet.AllMigrationsCompleted, status.StatusCode)
	assert.Empty(t, status.MissingTable)
	assert.Empty(t, status.MissingData)
}

func TestMigrations(t *testing.T) {
	// Create the database (must use raw MySQL client to do this)
	ds := createMySQLDSForMigrationTests(t, t.Name())
	defer ds.Close()

	require.NoError(t, ds.MigrateTables(context.Background()))

	// Dump schema to dumpfile
	cmd := exec.Command(
		"docker", "compose", "exec", "-T", "mysql_test",
		// Command run inside container
		"mysqldump", "-u"+testUsername, "-p"+testPassword, "TestMigrations", "--compact", "--skip-comments",
	)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "mysqldump: %s", string(output))
}

func createMySQLDSForMigrationTests(t *testing.T, dbName string) *Datastore {
	// Create a datastore client in order to run migrations as usual
	config := config.MysqlConfig{
		Username: testUsername,
		Password: testPassword,
		Address:  testAddress,
		Database: dbName,
	}
	ds, err := newDSWithConfig(t, dbName, config)
	require.NoError(t, err)
	return ds
}
