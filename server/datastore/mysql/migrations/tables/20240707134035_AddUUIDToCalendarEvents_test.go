package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20240707134035(t *testing.T) {
	db := applyUpToPrev(t)

	startTime := time.Now().UTC()
	endTime := time.Now().UTC().Add(30 * time.Minute)
	data := []byte("{\"foo\": \"bar\"}")
	const insertStmt = `INSERT INTO calendar_events (email, start_time, end_time, event) VALUES (?, ?, ?, ?)`
	event1ID := uint(execNoErrLastID(t, db, insertStmt, "foo@example.com", startTime, endTime, data)) //nolint:gosec // dismiss G115
	event2ID := uint(execNoErrLastID(t, db, insertStmt, "bar@example.com", startTime, endTime, data)) //nolint:gosec // dismiss G115

	// Apply current migration.
	applyNext(t, db)

	// check that UUID is not NULL
	const selectUUIDStmt = `SELECT uuid FROM calendar_events WHERE id = ?`
	var uuid1, uuid2 string
	err := db.Get(&uuid1, selectUUIDStmt, event1ID)
	require.NoError(t, err)
	assert.NotEmpty(t, uuid1)
	err = db.Get(&uuid2, selectUUIDStmt, event2ID)
	require.NoError(t, err)
	assert.NotEmpty(t, uuid2)
	assert.NotEqual(t, uuid1, uuid2)

	const testUUID = "test-uuid"
	const insertStmtUUID = `INSERT INTO calendar_events (email, start_time, end_time, event, uuid) VALUES (?, ?, ?, ?, ?)`
	_ = execNoErrLastID(t, db, insertStmtUUID, "bob@example.com", startTime, endTime, data, testUUID)
	// Try to use the same uuid again
	_, err = db.Exec(insertStmt, "alice@example.com", startTime, endTime, data, testUUID)
	assert.Error(t, err)

}
