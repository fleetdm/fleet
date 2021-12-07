package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/goose"
	"github.com/stretchr/testify/require"
)

func TestUp_20211116184030(t *testing.T) {
	db, err := mysql.NewDBConnForTests(t.Name())
	require.NoError(t, err)

	v, err := goose.NumericComponent(t.Name())
	require.NoError(t, err)

	for {
		current, err := MigrationClient.GetDBVersion(db.DB)
		require.NoError(t, err)

		next, err := MigrationClient.Migrations.Next(current)
		require.NoError(t, err)
		if next.Version == v {
			break
		}
	}

	// test here
	// users
	// queries
	// policies

	db.Exec(`INSERT INTO users(password,salt,name,email,)`)

	require.NoError(t, MigrationClient.UpByOne(db.DB, ""))

	// checks here
}
