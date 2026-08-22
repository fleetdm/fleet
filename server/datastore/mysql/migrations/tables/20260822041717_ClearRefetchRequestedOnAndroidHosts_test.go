package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260822041717(t *testing.T) {
	db := applyUpToPrev(t)

	insertHost := func(name, platform string, refetchRequested bool) {
		execNoErr(t, db, `
			INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform, refetch_requested)
			VALUES (?, ?, ?, ?, ?, ?)`,
			name, name, name, name, platform, refetchRequested)
	}

	insertHost("android-flagged", "android", true)
	insertHost("android-clear", "android", false)
	insertHost("darwin-flagged", "darwin", true)
	insertHost("ios-flagged", "ios", true)

	type hostRow struct {
		Hostname         string `db:"hostname"`
		RefetchRequested bool   `db:"refetch_requested"`
		UpdatedAt        string `db:"updated_at"`
	}
	snapshot := func() map[string]hostRow {
		var rows []hostRow
		require.NoError(t, db.Select(&rows, `SELECT hostname, refetch_requested, updated_at FROM hosts`))
		byName := make(map[string]hostRow, len(rows))
		for _, row := range rows {
			byName[row.Hostname] = row
		}
		return byName
	}
	before := snapshot()

	// Apply current migration.
	applyNext(t, db)

	after := snapshot()

	// The stranded flag on an Android host is what this migration exists to clear.
	require.False(t, after["android-flagged"].RefetchRequested)
	require.False(t, after["android-clear"].RefetchRequested)

	// Other platforms do clear the flag on their own, so a pending refetch there is
	// real work that has to survive.
	require.True(t, after["darwin-flagged"].RefetchRequested)
	require.True(t, after["ios-flagged"].RefetchRequested)

	// updated_at is ON UPDATE CURRENT_TIMESTAMP and the migration assigns it to
	// itself, so clearing the flag doesn't make the host look freshly updated.
	require.Equal(t, before["android-flagged"].UpdatedAt, after["android-flagged"].UpdatedAt)
}
