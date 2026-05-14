package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260512143542(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`
		INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid)
		VALUES (?, ?, ?, ?)`,
		"host-1-osquery-id", "host-1-node-key", "host-1", "host-1-uuid",
	)
	require.NoError(t, err)

	applyNext(t, db)

	// Default NULL on existing row.
	var debugUntil *time.Time
	err = db.QueryRow(`SELECT orbit_debug_until FROM hosts WHERE hostname = ?`, "host-1").Scan(&debugUntil)
	require.NoError(t, err)
	assert.Nil(t, debugUntil)

	// Set.
	future := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	_, err = db.Exec(`UPDATE hosts SET orbit_debug_until = ? WHERE hostname = ?`, future, "host-1")
	require.NoError(t, err)

	err = db.QueryRow(`SELECT orbit_debug_until FROM hosts WHERE hostname = ?`, "host-1").Scan(&debugUntil)
	require.NoError(t, err)
	require.NotNil(t, debugUntil)
	assert.True(t, debugUntil.Equal(future), "expected %s, got %s", future, debugUntil)

	// Clear.
	_, err = db.Exec(`UPDATE hosts SET orbit_debug_until = NULL WHERE hostname = ?`, "host-1")
	require.NoError(t, err)

	err = db.QueryRow(`SELECT orbit_debug_until FROM hosts WHERE hostname = ?`, "host-1").Scan(&debugUntil)
	require.NoError(t, err)
	assert.Nil(t, debugUntil)
}
