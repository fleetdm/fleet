package tables

import (
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestUp_20240709132642(t *testing.T) {
	db := applyUpToPrev(t)

	testUUID := strings.ToUpper(uuid.New().String())
	startTime := time.Now().UTC()
	endTime := time.Now().UTC().Add(30 * time.Minute)
	data := []byte("{\"foo\": \"bar\"}")
	const insertStmtUUID = `INSERT INTO calendar_events (email, start_time, end_time, event, uuid) VALUES (?, ?, ?, ?, ?)`
	eventID := execNoErrLastID(t, db, insertStmtUUID, "bob@example.com", startTime, endTime, data, testUUID)

	applyNext(t, db)
	// Check that uuid and uuid_bin are correct
	const selectUUIDStmt = `SELECT uuid, uuid_bin FROM calendar_events WHERE id = ?`
	type event struct {
		UUID    string `db:"uuid"`
		UUIDBin []byte `db:"uuid_bin"`
	}
	var e event
	err := db.Get(&e, selectUUIDStmt, eventID)
	require.NoError(t, err)
	assert.Equal(t, testUUID, e.UUID)
	uuidFromBytes, err := uuid.FromBytes(e.UUIDBin)
	require.NoError(t, err)
	assert.Equal(t, uuid.MustParse(testUUID), uuidFromBytes)

	// Try to use the same uuid again
	const insertStmtUUIDBin = `INSERT INTO calendar_events (email, start_time, end_time, event, uuid_bin) VALUES (?, ?, ?, ?, ?)`
	_, err = db.Exec(insertStmtUUIDBin, "alice@example.com", startTime, endTime, data, e.UUIDBin)
	assert.Error(t, err)

	// Insert a new event with a new UUID
	uuidBin := uuid.New()
	eventID = execNoErrLastID(t, db, insertStmtUUIDBin, "jane@example.com", startTime, endTime, data, uuidBin[:])
	err = db.Get(&e, selectUUIDStmt, eventID)
	require.NoError(t, err)
	assert.Equal(t, uuidBin[:], e.UUIDBin)
	assert.Equal(t, strings.ToUpper(uuidBin.String()), e.UUID)
}
