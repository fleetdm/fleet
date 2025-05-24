package tables

import (
	"errors"
	"os"
	"testing"

	"github.com/fleetdm/fleet/v4/server/goose"
	"github.com/stretchr/testify/require"
)

func TestCollation(t *testing.T) {
	require.NoError(t, os.Setenv("FLEET_TEST_DISABLE_COLLATION_UPDATES", "true"))
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("FLEET_TEST_DISABLE_COLLATION_UPDATES"))
	})

	db := newDBConnForTests(t)
	for {
		current, err := MigrationClient.GetDBVersion(db.DB)
		require.NoError(t, err)
		_, err = MigrationClient.Migrations.Next(current)
		if errors.Is(err, goose.ErrNoNextVersion) {
			break
		}
		require.NoError(t, err)
		applyNext(t, db)
	}

	checkCollation(t, db)
}
