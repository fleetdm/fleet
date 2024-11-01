package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestWindowsUpdates(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"InsertWindowsUpdates", testInsertWindowsUpdates},
		{"ListWindowsUpdatesByHostID", testListWindowsUpdatesByHostID},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testListWindowsUpdatesByHostID(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	now := uint(time.Now().Unix()) //nolint:gosec // dismiss G115

	t.Run("with no stored updates", func(t *testing.T) {
		actual, err := ds.ListWindowsUpdatesByHostID(ctx, 1)
		require.NoError(t, err)
		require.Empty(t, actual)
	})

	t.Run("none matching", func(t *testing.T) {
		updates := []fleet.WindowsUpdate{
			{KBID: 1, DateEpoch: now},
			{KBID: 2, DateEpoch: now + 1},
		}

		err := ds.InsertWindowsUpdates(ctx, 1, updates)
		require.NoError(t, err)

		actual, err := ds.ListWindowsUpdatesByHostID(ctx, 2)
		require.NoError(t, err)
		require.Empty(t, actual)
	})

	t.Run("returns matching", func(t *testing.T) {
		expected := []fleet.WindowsUpdate{
			{KBID: 1, DateEpoch: now},
			{KBID: 2, DateEpoch: now + 1},
		}

		err := ds.InsertWindowsUpdates(ctx, 1, expected)
		require.NoError(t, err)

		actual, err := ds.ListWindowsUpdatesByHostID(ctx, 1)
		require.NoError(t, err)
		require.ElementsMatch(t, expected, actual)
	})
}

func testInsertWindowsUpdates(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	now := uint(time.Now().Unix()) //nolint:gosec // dismiss G115
	smt := `SELECT kb_id, date_epoch FROM windows_updates WHERE host_id = ?`

	t.Run("with no stored updates", func(t *testing.T) {
		hostID := 1

		updates := []fleet.WindowsUpdate{
			{KBID: 1, DateEpoch: now},
			{KBID: 2, DateEpoch: now + 1},
		}

		err := ds.InsertWindowsUpdates(ctx, 1, updates)
		require.NoError(t, err)

		var actual []fleet.WindowsUpdate
		err = sqlx.SelectContext(ctx, ds.reader(ctx), &actual, smt, hostID)
		require.NoError(t, err)

		require.ElementsMatch(t, updates, actual)
	})

	t.Run("with stored updates", func(t *testing.T) {
		hostID := 1
		updates := []fleet.WindowsUpdate{
			{KBID: 1, DateEpoch: now},
			{KBID: 2, DateEpoch: now + 1},
		}

		err := ds.InsertWindowsUpdates(ctx, 1, updates)
		require.NoError(t, err)

		updates = append(updates, fleet.WindowsUpdate{KBID: 3, DateEpoch: now + 2})
		err = ds.InsertWindowsUpdates(ctx, 1, updates)
		require.NoError(t, err)

		var actual []fleet.WindowsUpdate
		err = sqlx.SelectContext(ctx, ds.reader(ctx), &actual, smt, hostID)
		require.NoError(t, err)

		require.ElementsMatch(t, updates, actual)
	})
}
