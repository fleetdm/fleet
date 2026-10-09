package tables

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260901200001(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert rows with each lowercase status value before the migration.
	cmds := []struct {
		uuid   string
		status string
	}{
		{uuid.NewString(), "pending"},
		{uuid.NewString(), "acknowledged"},
		{uuid.NewString(), "error"},
	}
	for i, c := range cmds {
		_, err := db.Exec(`
			INSERT INTO mdm_android_commands (command_uuid, host_uuid, operation_name, command_type, status)
			VALUES (?, ?, ?, ?, ?)`,
			c.uuid, "host-uuid-status", "enterprises/E/devices/D/operations/status-"+c.uuid, "LOCK", c.status)
		require.NoError(t, err, "insert row %d with status %s", i, c.status)
	}

	// Record updated_at before migration to verify it is not reset.
	var beforeUpdatedAt time.Time
	err := db.QueryRow(`SELECT updated_at FROM mdm_android_commands WHERE command_uuid = ?`, cmds[0].uuid).Scan(&beforeUpdatedAt)
	require.NoError(t, err)

	// Apply migration.
	applyNext(t, db)

	// Verify each row's status is now Title case.
	wantStatuses := []string{"Pending", "Acknowledged", "Error"}
	for i, c := range cmds {
		var status string
		err := db.QueryRow(`SELECT status FROM mdm_android_commands WHERE command_uuid = ?`, c.uuid).Scan(&status)
		require.NoError(t, err)
		assert.Equal(t, wantStatuses[i], status, "row %d status should be Title case", i)
	}

	// Verify updated_at was NOT reset by the migration.
	var afterUpdatedAt time.Time
	err = db.QueryRow(`SELECT updated_at FROM mdm_android_commands WHERE command_uuid = ?`, cmds[0].uuid).Scan(&afterUpdatedAt)
	require.NoError(t, err)
	assert.Equal(t, beforeUpdatedAt, afterUpdatedAt, "updated_at should not change during migration")

	// Verify new inserts use the Title case default.
	newUUID := uuid.NewString()
	_, err = db.Exec(`
		INSERT INTO mdm_android_commands (command_uuid, host_uuid, operation_name, command_type)
		VALUES (?, ?, ?, ?)`,
		newUUID, "host-uuid-new", "enterprises/E/devices/D/operations/new-"+newUUID, "REBOOT")
	require.NoError(t, err)

	var newStatus string
	err = db.QueryRow(`SELECT status FROM mdm_android_commands WHERE command_uuid = ?`, newUUID).Scan(&newStatus)
	require.NoError(t, err)
	assert.Equal(t, "Pending", newStatus, "default status should be Title case Pending")
}
