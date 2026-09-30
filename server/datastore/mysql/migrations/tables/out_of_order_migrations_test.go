package tables

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/goose"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const (
	oooFloor = int64(20260101000000)
	oooV1    = int64(20260101000001)
	oooV2    = int64(20260101000002)
	oooV3    = int64(20260101000003)
	oooVOld  = int64(20200101000001)
)

// newOutOfOrderTestClient builds a goose client with synthetic migrations,
// each creating a marker table named after its version.
func newOutOfOrderTestClient(minOutOfOrderVersion int64, versions ...int64) *goose.Client {
	c := goose.New("migration_status_out_of_order_test", goose.MySqlDialect{})
	c.MinOutOfOrderVersion = minOutOfOrderVersion
	for _, v := range versions {
		version := v
		c.Migrations = append(c.Migrations, &goose.Migration{
			Version:  version,
			Next:     -1,
			Previous: -1,
			Source:   fmt.Sprintf("%d_TestOutOfOrder.go", version),
			UpFn: func(tx *sql.Tx) error {
				_, err := tx.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS out_of_order_%d (id INT)", version))
				return err
			},
			DownFn: func(tx *sql.Tx) error { return nil },
		})
	}
	return c
}

func outOfOrderMarkerExists(t *testing.T, db *sqlx.DB, version int64) bool {
	t.Helper()
	var count int
	err := db.Get(&count, `
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema = DATABASE() AND table_name = ?`,
		fmt.Sprintf("out_of_order_%d", version))
	require.NoError(t, err)
	return count > 0
}

func TestUpOutOfOrder(t *testing.T) {
	t.Run("patched", func(t *testing.T) {
		db := newDBConnForTests(t)

		// Apply v1 and v3 only — the state of a database before a patch
		// release adds v2 with a timestamp older than the already-applied v3.
		require.NoError(t, newOutOfOrderTestClient(oooFloor, oooV1, oooV3).Up(db.DB, ""))
		require.True(t, outOfOrderMarkerExists(t, db, oooV1))
		require.True(t, outOfOrderMarkerExists(t, db, oooV3))

		// With out-of-order disabled (zero floor), v2 stays unapplied.
		require.NoError(t, newOutOfOrderTestClient(0, oooV1, oooV2, oooV3).Up(db.DB, ""))
		require.False(t, outOfOrderMarkerExists(t, db, oooV2))

		// With the floor set, v2 is applied even though v3 already was.
		require.NoError(t, newOutOfOrderTestClient(oooFloor, oooV1, oooV2, oooV3).Up(db.DB, ""))
		require.True(t, outOfOrderMarkerExists(t, db, oooV2))

		// The out-of-order application was recorded like any other migration.
		var count int
		require.NoError(t, db.Get(&count,
			"SELECT COUNT(*) FROM migration_status_out_of_order_test WHERE version_id = ? AND is_applied", oooV2))
		require.Equal(t, 1, count)

		// A missing migration below the floor is left alone.
		require.NoError(t, newOutOfOrderTestClient(oooFloor, oooVOld, oooV1, oooV2, oooV3).Up(db.DB, ""))
		require.False(t, outOfOrderMarkerExists(t, db, oooVOld))

		// Up is idempotent: nothing is re-applied.
		require.NoError(t, newOutOfOrderTestClient(oooFloor, oooV1, oooV2, oooV3).Up(db.DB, ""))
		require.NoError(t, db.Get(&count,
			"SELECT COUNT(*) FROM migration_status_out_of_order_test WHERE version_id > 0 AND is_applied"))
		require.Equal(t, 3, count)
	})

	t.Run("fresh", func(t *testing.T) {
		db := newDBConnForTests(t)

		// Nothing is applied yet, so even versions below the floor run — the
		// floor only constrains out-of-order application.
		require.NoError(t, newOutOfOrderTestClient(oooFloor, oooVOld, oooV1, oooV2).Up(db.DB, ""))
		require.True(t, outOfOrderMarkerExists(t, db, oooVOld))
		require.True(t, outOfOrderMarkerExists(t, db, oooV1))
		require.True(t, outOfOrderMarkerExists(t, db, oooV2))
	})
}
