package tables

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20240126020643(t *testing.T) {
	db := applyUpToPrev(t)

	// create a couple users
	u1 := execNoErrLastID(t, db, `INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`, "u1", "u1@b.c", "1234", "salt")
	u2 := execNoErrLastID(t, db, `INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`, "u2", "u2@b.c", "1234", "salt")

	// create an activity
	act1 := execNoErrLastID(t, db, `INSERT INTO activities (user_id, user_name, user_email, activity_type) VALUES (?, ?, ?, ?)`, u1, "u1", "u1@b.c", "act1")

	// create a host execution request in the past
	minutesAgo := time.Now().UTC().Add(-5 * time.Minute).Truncate(time.Second)
	hsr1 := execNoErrLastID(t, db, `INSERT INTO host_script_results (host_id, execution_id, script_contents, output, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, 1, uuid.NewString(), "echo 'hello'", "", minutesAgo, minutesAgo)
	hsr2 := execNoErrLastID(t, db, `INSERT INTO host_script_results (host_id, execution_id, script_contents, output, created_at, updated_at, exit_code) VALUES (?, ?, ?, ?, ?, ?, ?)`, 1, uuid.NewString(), "echo 'hello'", "", minutesAgo, minutesAgo, 1)

	// Apply current migration.
	applyNext(t, db)

	// async request is set to `true` for existing results
	// existing host execution request's timestamp hasn't changed (despite
	// added column, and modified sync_request)
	type scriptResults struct {
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
		ExitCode  *int      `db:"exit_code"`
	}

	var sr scriptResults
	err := db.Get(&sr, `SELECT created_at, updated_at, exit_code FROM host_script_results WHERE id = ?`, hsr1)
	require.NoError(t, err)
	assert.Equal(t, minutesAgo, sr.CreatedAt)
	assert.Equal(t, minutesAgo, sr.UpdatedAt)
	assert.Equal(t, -1, *sr.ExitCode)

	sr = scriptResults{}
	err = db.Get(&sr, `SELECT created_at, updated_at, exit_code FROM host_script_results WHERE id = ?`, hsr2)
	require.NoError(t, err)
	assert.Equal(t, minutesAgo, sr.CreatedAt)
	assert.Equal(t, minutesAgo, sr.UpdatedAt)
	assert.Equal(t, 1, *sr.ExitCode)

	// create a new host execution request with user u1 and one with u2
	hsr3 := execNoErrLastID(t, db, `INSERT INTO host_script_results (host_id, execution_id, script_contents, output, user_id) VALUES (?, ?, ?, ?, ?)`, 1, uuid.NewString(), "echo 'hello'", "", u1)
	hsr4 := execNoErrLastID(t, db, `INSERT INTO host_script_results (host_id, execution_id, script_contents, output, user_id) VALUES (?, ?, ?, ?, ?)`, 1, uuid.NewString(), "echo 'hello'", "", u2)

	// create a host activity entry for act1
	execNoErr(t, db, `INSERT INTO host_activities (host_id, activity_id) VALUES (?, ?)`, 1, act1)

	// delete user u1
	execNoErr(t, db, `DELETE FROM users WHERE id = ?`, u1)

	var userID sql.NullInt64
	// hsr2 now has a NULL user id, but hsr3 still has user id u2
	err = db.Get(&userID, `SELECT user_id FROM host_script_results WHERE id = ?`, hsr3)
	require.NoError(t, err)
	assert.False(t, userID.Valid)
	err = db.Get(&userID, `SELECT user_id FROM host_script_results WHERE id = ?`, hsr4)
	require.NoError(t, err)
	assert.True(t, userID.Valid)
	assert.Equal(t, u2, userID.Int64)

	// host activity entry exists for host 1
	var actID sql.NullInt64
	err = db.Get(&actID, `SELECT activity_id FROM host_activities WHERE host_id = ?`, 1)
	require.NoError(t, err)
	assert.True(t, actID.Valid)
	assert.Equal(t, act1, actID.Int64)

	// delete activity act1
	execNoErr(t, db, `DELETE FROM activities WHERE id = ?`, act1)

	// host activity entry does not exist anymore
	err = db.Get(&actID, `SELECT activity_id FROM host_activities WHERE host_id = ?`, 1)
	require.Error(t, err)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}
