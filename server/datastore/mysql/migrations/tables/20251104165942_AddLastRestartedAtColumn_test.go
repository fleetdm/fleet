package tables

import (
	"fmt"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20251104165942(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert test hosts
	host1ID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, uuid, platform, uptime, detail_updated_at) VALUES (?, ?, ?, ?, ?, ?)`, "host1", "key1", "uuid1", "darwin", 2388335000000000, "2025-11-04 23:07:56")
	host2ID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, uuid, platform, uptime, detail_updated_at) VALUES (?, ?, ?, ?, ?, ?)`, "host2", "key2", "uuid2", "darwin", 0, "2025-11-04 23:07:56")
	host3ID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, uuid, platform, uptime, detail_updated_at) VALUES (?, ?, ?, ?, ?, ?)`, "host3", "key3", "uuid3", "darwin", 2388335000000000, nil)

	// Apply current migration.
	applyNext(t, db)

	// Windows profile uuids were updated with the prefix
	var hosts []struct {
		HostID          string    `db:"id"`
		LastRestartedAt time.Time `db:"last_restarted_at"`
	}
	err := sqlx.Select(db, &hosts, `SELECT id, last_restarted_at FROM hosts ORDER BY id`)
	require.NoError(t, err)
	require.Len(t, hosts, 3)

	require.Equal(t, fmt.Sprint(host1ID), hosts[0].HostID)
	expectedRestartedAt1 := time.Date(2025, 11, 4, 23, 7, 56, 0, time.UTC).Add(-time.Duration(2388335000000000))
	require.Equal(t, expectedRestartedAt1, hosts[0].LastRestartedAt)

	require.Equal(t, fmt.Sprint(host2ID), hosts[1].HostID)
	expectedRestartedAt2 := time.Date(0o001, 1, 1, 0, 0, 0, 0, time.UTC)
	require.Equal(t, expectedRestartedAt2, hosts[1].LastRestartedAt)

	require.Equal(t, fmt.Sprint(host3ID), hosts[2].HostID)
	expectedRestartedAt3 := time.Date(0o001, 1, 1, 0, 0, 0, 0, time.UTC)
	require.Equal(t, expectedRestartedAt3, hosts[2].LastRestartedAt)
}
