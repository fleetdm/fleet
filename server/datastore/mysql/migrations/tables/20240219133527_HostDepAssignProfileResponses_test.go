package tables

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20240219133527(t *testing.T) {
	db := applyUpToPrev(t)
	insertStmt := "INSERT INTO host_dep_assignments (host_id) VALUES (?);"
	execNoErr(t, db, insertStmt, 1337)

	// Apply current migration.
	applyNext(t, db)

	// profile_uuid and assign_profile_response are now present and NULL
	type hda struct {
		HostID                uint       `db:"host_id"`
		AddedAt               time.Time  `db:"added_at"`
		DeletedAt             *time.Time `db:"deleted_at"`
		ProfileUUID           *string    `db:"profile_uuid"`
		AssignProfileResponse *string    `db:"assign_profile_response"`
	}
	var dest hda
	err := sqlx.Get(db, &dest, `SELECT host_id, added_at, deleted_at, profile_uuid, assign_profile_response FROM host_dep_assignments WHERE host_id = ?`, 1337)
	require.NoError(t, err)
	require.Equal(t, uint(1337), dest.HostID)
	require.NotZero(t, dest.AddedAt)
	require.Nil(t, dest.DeletedAt)
	require.Nil(t, dest.ProfileUUID)
	require.Nil(t, dest.AssignProfileResponse)

	// set profile_uuid and assign_profile_response to non-NULL values
	execNoErr(t, db, `UPDATE host_dep_assignments SET profile_uuid = 'foo', assign_profile_response = 'bar' WHERE host_id = ?`, 1337)

	dest = hda{}
	err = sqlx.Get(db, &dest, `SELECT host_id, added_at, deleted_at, profile_uuid, assign_profile_response FROM host_dep_assignments WHERE host_id = ?`, 1337)
	require.NoError(t, err)
	require.Equal(t, uint(1337), dest.HostID)
	require.NotZero(t, dest.AddedAt)
	require.Nil(t, dest.DeletedAt)
	require.Equal(t, "foo", *dest.ProfileUUID)
	require.Equal(t, "bar", *dest.AssignProfileResponse)
}
