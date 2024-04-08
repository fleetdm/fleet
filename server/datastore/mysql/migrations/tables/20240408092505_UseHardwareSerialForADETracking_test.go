package tables

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20240408092505(t *testing.T) {
	db := applyUpToPrev(t)

	threeDayAgo := time.Now().UTC().Add(-72 * time.Hour).Truncate(time.Second)

	// insert hosts
	execNoErr(t, db, `
INSERT INTO hosts
  (id, hardware_serial)
VALUES
  (1, 'foo'),
  (2, 'bar'),
  (3, 'zoo')`)

	// insert DEP assignments
	execNoErr(t, db, `
INSERT INTO host_dep_assignments
  (host_id, deleted_at)
VALUES
  -- matching host, not deleted entry
  (1, NULL),
  -- matching host, deleted entry
  (2, ?),
  -- orphaned entries
  (4, NULL),
  (5, ?)
  `, threeDayAgo, threeDayAgo)

	// insert nano DEP metadata
	execNoErr(t, db, `
INSERT INTO nano_dep_names
  (name, syncer_cursor, syncer_cursor_at)
VALUES
  ("fleet", "foo", NOW())
	`)

	applyNext(t, db)

	type assignment struct {
		HostHardwareSerial string     `db:"host_hardware_serial"`
		DeletedAt          *time.Time `db:"deleted_at"`
	}

	var assignments []assignment
	err := db.Select(
		&assignments,
		`SELECT host_hardware_serial, deleted_at FROM host_dep_assignments`,
	)
	require.NoError(t, err)
	require.Len(t, assignments, 2)
	require.ElementsMatch(t, []assignment{
		{HostHardwareSerial: "foo", DeletedAt: nil},
		{HostHardwareSerial: "bar", DeletedAt: &threeDayAgo},
	}, assignments)

	var cursor sql.NullString
	err = db.Get(&cursor, `SELECT syncer_cursor FROM nano_dep_names`)
	require.NoError(t, err)
	require.False(t, cursor.Valid)
	require.Empty(t, cursor.String)
}
