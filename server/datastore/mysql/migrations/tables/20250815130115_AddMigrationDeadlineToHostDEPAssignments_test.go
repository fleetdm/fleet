package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20250815130115(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`INSERT INTO host_dep_assignments (host_id) VALUES (1)`)
	require.NoError(t, err)
	// Apply current migration.
	applyNext(t, db)

	hda := struct {
		HostID                uint       `db:"host_id"`
		MDMMigrationDeadline  *time.Time `db:"mdm_migration_deadline"`
		MDMMigrationCompleted *time.Time `db:"mdm_migration_completed"`
	}{}
	err = db.QueryRow(`
		SELECT host_id, mdm_migration_deadline, mdm_migration_completed
		FROM host_dep_assignments
		WHERE host_id = ?
	`, 1).Scan(
		&hda.HostID,
		&hda.MDMMigrationDeadline,
		&hda.MDMMigrationCompleted,
	)
	require.NoError(t, err)
	require.Equal(t, uint(1), hda.HostID)
	require.Nil(t, hda.MDMMigrationDeadline)
	require.Nil(t, hda.MDMMigrationCompleted)

	_, err = db.Exec(`INSERT INTO host_dep_assignments (host_id) VALUES (2)`)
	require.NoError(t, err)
	err = db.QueryRow(`
		SELECT host_id, mdm_migration_deadline, mdm_migration_completed
		FROM host_dep_assignments
		WHERE host_id = ?
	`, 2).Scan(
		&hda.HostID,
		&hda.MDMMigrationDeadline,
		&hda.MDMMigrationCompleted,
	)
	require.NoError(t, err)
	require.Equal(t, uint(2), hda.HostID)
	require.Nil(t, hda.MDMMigrationDeadline)
	require.Nil(t, hda.MDMMigrationCompleted)

	deadline := time.Now().UTC().Truncate(time.Millisecond)
	completed := time.Now().Add(-1 * time.Hour).UTC().Truncate(time.Millisecond)
	_, err = db.Exec(`INSERT INTO host_dep_assignments (host_id, mdm_migration_deadline, mdm_migration_completed) VALUES (?, ?, ?)`, 3, deadline, completed)
	require.NoError(t, err)
	err = db.QueryRow(`
		SELECT host_id, mdm_migration_deadline, mdm_migration_completed
		FROM host_dep_assignments
		WHERE host_id = ?
	`, 3).Scan(
		&hda.HostID,
		&hda.MDMMigrationDeadline,
		&hda.MDMMigrationCompleted,
	)

	require.NoError(t, err)
	require.Equal(t, uint(3), hda.HostID)
	require.NotNil(t, hda.MDMMigrationDeadline)
	require.Equal(t, deadline, *hda.MDMMigrationDeadline)
	require.NotNil(t, hda.MDMMigrationCompleted)
	require.Equal(t, completed, *hda.MDMMigrationCompleted)
}
