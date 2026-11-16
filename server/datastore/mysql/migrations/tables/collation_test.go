package tables

import (
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/goose"
	"github.com/stretchr/testify/require"
)

func TestCollation(t *testing.T) {
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
