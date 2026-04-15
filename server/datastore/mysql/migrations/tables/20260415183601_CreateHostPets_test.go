package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260415183601(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert a host so the FK is satisfied.
	hostRes, err := db.Exec(`
		INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid)
		VALUES (?, ?, ?, ?)`,
		"test-host-pet-1", "node-key-pet-1", "pet-host-1", "pet-uuid-1",
	)
	require.NoError(t, err)
	hostID, err := hostRes.LastInsertId()
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// INSERT with defaults.
	_, err = db.Exec(`
		INSERT INTO host_pets (host_id, name) VALUES (?, ?)`,
		hostID, "Whiskers",
	)
	require.NoError(t, err)

	// Verify defaults.
	var (
		name        string
		species     string
		health      uint8
		happiness   uint8
		hunger      uint8
		cleanliness uint8
		lastInter   time.Time
	)
	err = db.QueryRow(`
		SELECT name, species, health, happiness, hunger, cleanliness, last_interacted_at
		FROM host_pets WHERE host_id = ?`, hostID,
	).Scan(&name, &species, &health, &happiness, &hunger, &cleanliness, &lastInter)
	require.NoError(t, err)
	assert.Equal(t, "Whiskers", name)
	assert.Equal(t, "cat", species)
	assert.Equal(t, uint8(80), health)
	assert.Equal(t, uint8(80), happiness)
	assert.Equal(t, uint8(20), hunger)
	assert.Equal(t, uint8(80), cleanliness)
	assert.False(t, lastInter.IsZero())

	// One pet per host (unique constraint).
	_, err = db.Exec(`
		INSERT INTO host_pets (host_id, name) VALUES (?, ?)`,
		hostID, "Imposter",
	)
	require.Error(t, err)

	// Cascade delete when host is removed.
	_, err = db.Exec(`DELETE FROM hosts WHERE id = ?`, hostID)
	require.NoError(t, err)

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM host_pets WHERE host_id = ?`, hostID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
