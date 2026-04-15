package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260415221415(t *testing.T) {
	db := applyUpToPrev(t)

	hostRes, err := db.Exec(`
		INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid)
		VALUES (?, ?, ?, ?)`,
		"test-host-pet-demo-1", "node-key-pet-demo-1", "pet-demo-host-1", "pet-demo-uuid-1",
	)
	require.NoError(t, err)
	hostID, err := hostRes.LastInsertId()
	require.NoError(t, err)

	applyNext(t, db)

	// INSERT with defaults.
	_, err = db.Exec(`INSERT INTO host_pet_demo_overrides (host_id) VALUES (?)`, hostID)
	require.NoError(t, err)

	var (
		seenTimeOverride     *string // NULL by default
		timeOffsetHours      int
		extraFailingPolicies uint
		extraCriticalVulns   uint
		extraHighVulns       uint
	)
	err = db.QueryRow(`
		SELECT seen_time_override, time_offset_hours,
		       extra_failing_policies, extra_critical_vulns, extra_high_vulns
		FROM host_pet_demo_overrides WHERE host_id = ?`, hostID,
	).Scan(&seenTimeOverride, &timeOffsetHours, &extraFailingPolicies, &extraCriticalVulns, &extraHighVulns)
	require.NoError(t, err)
	assert.Nil(t, seenTimeOverride)
	assert.Equal(t, 0, timeOffsetHours)
	assert.Equal(t, uint(0), extraFailingPolicies)
	assert.Equal(t, uint(0), extraCriticalVulns)
	assert.Equal(t, uint(0), extraHighVulns)

	// One row per host (PK).
	_, err = db.Exec(`INSERT INTO host_pet_demo_overrides (host_id) VALUES (?)`, hostID)
	require.Error(t, err)

	// Cascade delete with host.
	_, err = db.Exec(`DELETE FROM hosts WHERE id = ?`, hostID)
	require.NoError(t, err)

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM host_pet_demo_overrides WHERE host_id = ?`, hostID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
