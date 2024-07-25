package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20240618142419(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	// Create new activities
	u1 := execNoErrLastID(t, db, `INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`, "u1", "u1@b.c", "1234", "salt")
	act1 := execNoErrLastID(
		t, db, `INSERT INTO activities (user_id, user_name, user_email, activity_type) VALUES (?, ?, ?, ?)`, u1, "u1", "u1@b.c", "act1",
	)
	act2 := execNoErrLastID(
		t, db, `INSERT INTO activities (user_id, user_name, user_email, activity_type) VALUES (?, ?, ?, ?)`, u1, "u1", "u1@b.c", "act1",
	)
	act3 := execNoErrLastID(
		t, db, `INSERT INTO activities (user_id, user_name, user_email, activity_type) VALUES (?, ?, ?, ?)`, u1, "u1", "u1@b.c", "act1",
	)

	selectStmt := `SELECT created_at from activities WHERE id = ?`
	var act1CreatedAt, act2CreatedAt, act3CreatedAt time.Time
	require.NoError(t, db.Get(&act1CreatedAt, selectStmt, act1))
	require.NoError(t, db.Get(&act2CreatedAt, selectStmt, act2))
	require.NoError(t, db.Get(&act3CreatedAt, selectStmt, act3))

	assert.NotZero(t, act1CreatedAt)
	assert.True(t, act1CreatedAt.Before(act2CreatedAt))
	assert.True(t, act2CreatedAt.Before(act3CreatedAt))
}
