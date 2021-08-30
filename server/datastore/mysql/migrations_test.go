package mysql

import (
	"bytes"
	"os/exec"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/tables"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationStatus(t *testing.T) {
	ds := createMySQLDSForMigrationTests(t, t.Name())
	defer ds.Close()

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

func TestMigrations(t *testing.T) {
	// Create the database (must use raw MySQL client to do this)
	ds := createMySQLDSForMigrationTests(t, t.Name())
	defer ds.Close()

	require.NoError(t, ds.MigrateTables())

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

func Test20210819131107_AddCascadeToHostSoftware(t *testing.T) {
	ds := createMySQLDSForMigrationTests(t, t.Name())
	defer ds.Close()

	for {
		version, err := tables.MigrationClient.GetDBVersion(ds.db.DB)
		require.NoError(t, err)

		// break right before the the constraint migration
		if version == 20210818182258 {
			break
		}
		require.NoError(t, tables.MigrationClient.UpByOne(ds.db.DB, ""))
	}

	host1 := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())

	soft1 := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
		},
	}
	host1.HostSoftware = soft1
	soft2 := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
			{Name: "foo", Version: "0.0.3", Source: "chrome_extensions"},
			{Name: "bar", Version: "0.0.3", Source: "deb_packages"},
		},
	}
	host2.HostSoftware = soft2
	host2.Modified = true

	require.NoError(t, ds.SaveHostSoftware(host1))
	require.NoError(t, ds.SaveHostSoftware(host2))

	require.NoError(t, ds.DeleteHost(host1.ID))

	require.NoError(t, tables.MigrationClient.UpByOne(ds.db.DB, ""))

	// Make sure we don't delete more than we need
	hostCheck, err := ds.Host(host2.ID)
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(hostCheck))
	require.Len(t, hostCheck.Software, 3)
}
