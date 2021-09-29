package mysql

import (
	"bytes"
	"context"
	"os/exec"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationStatus(t *testing.T) {
	ds := createMySQLDSForMigrationTests(t, t.Name())
	defer ds.Close()

	status, err := ds.MigrationStatus(context.Background())
	require.Nil(t, err)
	assert.EqualValues(t, fleet.NoMigrationsCompleted, status)

	require.Nil(t, ds.MigrateTables(context.Background()))

	status, err = ds.MigrationStatus(context.Background())
	require.Nil(t, err)
	assert.EqualValues(t, fleet.SomeMigrationsCompleted, status)

	require.Nil(t, ds.MigrateData(context.Background()))

	status, err = ds.MigrationStatus(context.Background())
	require.Nil(t, err)
	assert.EqualValues(t, fleet.AllMigrationsCompleted, status)
}

func TestMigrations(t *testing.T) {
	// Create the database (must use raw MySQL client to do this)
	ds := createMySQLDSForMigrationTests(t, t.Name())
	defer ds.Close()

	require.NoError(t, ds.MigrateTables(context.Background()))

	// Dump schema to dumpfile
	cmd := exec.Command(
		"docker-compose", "exec", "-T", "mysql_test",
		// Command run inside container
		"mysqldump", "-u"+testUsername, "-p"+testPassword, "TestMigrations", "--compact", "--skip-comments",
	)
	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	require.NoError(t, cmd.Run())

	require.NotEmpty(t, stdoutBuf.String())
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
