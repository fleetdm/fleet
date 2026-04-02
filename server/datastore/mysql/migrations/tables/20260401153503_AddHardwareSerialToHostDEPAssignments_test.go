package tables

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20260401153503_SomeAssignments(t *testing.T) {
	db := applyUpToPrev(t)

	// create a dozen hosts each for macOS, Windows and Linux
	macIDs, _, _, _ := insertHosts(t, db, 12, 12, 12)
	require.Len(t, macIDs, 12)

	// load the serials for the mac hosts
	type host struct {
		ID             uint   `db:"id"`
		HardwareSerial string `db:"hardware_serial"`
	}
	var hosts []host
	stmt, args, err := sqlx.In(`SELECT id, hardware_serial FROM hosts WHERE id IN (?)`, macIDs)
	require.NoError(t, err)

	err = db.Select(&hosts, stmt, args...)
	require.NoError(t, err)
	require.Len(t, hosts, 12)

	idToSerial := make(map[uint]string)
	for _, h := range hosts {
		idToSerial[h.ID] = h.HardwareSerial
	}

	// create DEP assignments for a few mac hosts
	for _, id := range macIDs[:3] {
		_, err := db.Exec(`INSERT INTO host_dep_assignments (host_id) VALUES (?)`, id)
		require.NoError(t, err)
	}
	// make macIDs[2] a deleted assignment
	_, err = db.Exec(`UPDATE host_dep_assignments SET deleted_at = NOW() WHERE host_id = ?`, macIDs[2]) //nolint:nilaway
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// load the assignments and verify that it has the expected hardware serials for non-deleted assignments
	var assignments []struct {
		HostID         uint       `db:"host_id"`
		HardwareSerial string     `db:"hardware_serial"`
		DeletedAt      *time.Time `db:"deleted_at"`
	}
	err = db.Select(&assignments, `SELECT host_id, hardware_serial, deleted_at FROM host_dep_assignments`)
	require.NoError(t, err)
	require.Len(t, assignments, 3)

	for _, a := range assignments {
		switch a.HostID {
		case macIDs[0], macIDs[1]:
			require.Nil(t, a.DeletedAt)
			require.Equal(t, idToSerial[a.HostID], a.HardwareSerial)
		case macIDs[2]:
			require.Empty(t, a.HardwareSerial)
			require.NotNil(t, a.DeletedAt)
		default:
			t.Fatalf("unexpected host_id %d in host_dep_assignments", a.HostID)
		}
	}
}

func TestUp_20260401153503_NoAssignment(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM host_dep_assignments`)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestUp_20260401153503_ManyAssignments(t *testing.T) {
	db := applyUpToPrev(t)

	// create a thousand macOS hosts and a few other
	macIDs, _, _, _ := insertHosts(t, db, 1000, 10, 10)
	require.Len(t, macIDs, 1000)

	// load the serials for the mac hosts
	type host struct {
		ID             uint   `db:"id"`
		HardwareSerial string `db:"hardware_serial"`
	}
	var hosts []host
	stmt, args, err := sqlx.In(`SELECT id, hardware_serial FROM hosts WHERE id IN (?)`, macIDs)
	require.NoError(t, err)

	err = db.Select(&hosts, stmt, args...)
	require.NoError(t, err)
	require.Len(t, hosts, len(macIDs))

	idToSerial := make(map[uint]string)
	for _, h := range hosts {
		idToSerial[h.ID] = h.HardwareSerial
	}

	// create DEP assignments for all mac hosts
	for _, id := range macIDs {
		_, err := db.Exec(`INSERT INTO host_dep_assignments (host_id) VALUES (?)`, id)
		require.NoError(t, err)
	}

	// Apply current migration.
	applyNext(t, db)

	// load the assignments and verify that it has the expected hardware serials for non-deleted assignments
	var assignments []struct {
		HostID         uint   `db:"host_id"`
		HardwareSerial string `db:"hardware_serial"`
	}
	err = db.Select(&assignments, `SELECT host_id, hardware_serial FROM host_dep_assignments`)
	require.NoError(t, err)
	require.Len(t, assignments, len(macIDs))

	for _, a := range assignments {
		require.Equal(t, idToSerial[a.HostID], a.HardwareSerial)
	}
}
