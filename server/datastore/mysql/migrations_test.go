package mysql

import (
	"context"
	"os/exec"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/data"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/tables"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestV4732MigrationFix(t *testing.T) {
	ds := createMySQLDSForMigrationTests(t, t.Name())
	t.Cleanup(func() {
		ds.Close()
	})
	status, err := ds.MigrationStatus(context.Background())
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.EqualValues(t, fleet.NoMigrationsCompleted, status.StatusCode)

	recreate4732BadState(t, ds)

	status, err = ds.MigrationStatus(context.Background())
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.EqualValues(t, fleet.NeedsFleetv4732Fix, status.StatusCode)

	err = ds.FixFleetv4732Migrations(context.Background())
	require.NoError(t, err)

	err = ds.MigrateTables(context.Background())
	require.NoError(t, err)

	status, err = ds.MigrationStatus(context.Background())
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.EqualValues(t, fleet.AllMigrationsCompleted, status.StatusCode)

	_, err = ds.writer(context.Background()).Exec(`INSERT INTO `+tables.MigrationClient.TableName+` (version_id, is_applied) VALUES (?, 1)`, fleet4732BadMigrationID1)
	require.NoError(t, err)
	status, err = ds.MigrationStatus(context.Background())
	require.NoError(t, err)
	require.NotNil(t, status)
	assert.EqualValues(t, fleet.UnknownFleetv4732State, status.StatusCode)
}

// Apply the proper 4.73.2 migrations
func recreate4732GoodState(t *testing.T, ds *Datastore) {
	var version int64
	var err error

	const maxDataMigration = 20230525175650

	// Migrate up to 4.73.1
	for version < fleet4731GoodMigrationID {
		err = tables.MigrationClient.UpByOne(ds.writer(context.Background()).DB, "")
		require.NoError(t, err)
		version, err = tables.MigrationClient.GetDBVersion(ds.writer(context.Background()).DB)
		require.NoError(t, err)
	}
	require.Equal(t, int64(fleet4731GoodMigrationID), version)

	// Apply data migrations which were deprecated before 4.73.2 and should never change so no need for
	// upbyone, etc. but we'll verify below that we're at expected version
	err = data.MigrationClient.Up(ds.writer(context.Background()).DB, "")
	require.NoError(t, err)
	version, err = data.MigrationClient.GetDBVersion(ds.writer(context.Background()).DB)
	require.NoError(t, err)
	require.EqualValues(t, int64(maxDataMigration), version)

	// Apply the migrations from fleet v4.73.2
	err = tables.MigrationClient.UpByOne(ds.writer(context.Background()).DB, "")
	require.NoError(t, err)
	version, err = tables.MigrationClient.GetDBVersion(ds.writer(context.Background()).DB)
	require.NoError(t, err)
	require.EqualValues(t, fleet4732GoodMigrationID2, version)

	err = tables.MigrationClient.UpByOne(ds.writer(context.Background()).DB, "")
	require.NoError(t, err)
	version, err = tables.MigrationClient.GetDBVersion(ds.writer(context.Background()).DB)
	require.NoError(t, err)
	require.EqualValues(t, fleet4732GoodMigrationID1, version)
}

// Recreate the bad state that some customers ended up with after running fleet v4.73.2 migrations
func recreate4732BadState(t *testing.T, ds *Datastore) {
	recreate4732GoodState(t, ds)

	_, err := ds.writer(context.Background()).Exec(`UPDATE `+tables.MigrationClient.TableName+` SET version_id = ? WHERE version_id = ?`, fleet4732BadMigrationID1, fleet4732GoodMigrationID1)
	require.NoError(t, err)
	_, err = ds.writer(context.Background()).Exec(`UPDATE `+tables.MigrationClient.TableName+` SET version_id = ? WHERE version_id = ?`, fleet4732BadMigrationID2, fleet4732GoodMigrationID2)
	require.NoError(t, err)

	version, err := tables.MigrationClient.GetDBVersion(ds.writer(context.Background()).DB)
	require.NoError(t, err)
	require.EqualValues(t, fleet4732BadMigrationID1, version)
}

func TestMigrations(t *testing.T) {
	// Create the database (must use raw MySQL client to do this)
	ds := createMySQLDSForMigrationTests(t, t.Name())
	defer ds.Close()

	require.NoError(t, ds.MigrateTables(context.Background()))

	// Dump schema to dumpfile
	cmd := exec.Command( // nolint:gosec // Waive G204 since this is a test file
		"docker", "compose", "exec", "-T", "mysql_test",
		// Command run inside container
		"mysqldump", "-u"+testing_utils.TestUsername, "-p"+testing_utils.TestPassword, "TestMigrations", "--compact", "--skip-comments",
	)

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "mysqldump: %s", string(output))
}

func createMySQLDSForMigrationTests(t *testing.T, dbName string) *Datastore {
	// Create a datastore client in order to run migrations as usual
	config := config.MysqlConfig{
		Username: testing_utils.TestUsername,
		Password: testing_utils.TestPassword,
		Address:  testing_utils.TestAddress,
		Database: dbName,
	}
	ds, err := newDSWithConfig(t, dbName, config)
	require.NoError(t, err)
	return ds
}
