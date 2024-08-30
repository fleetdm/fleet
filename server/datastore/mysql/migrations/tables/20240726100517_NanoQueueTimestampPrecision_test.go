package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20240726100517(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	// Create new commands
	execNoErr(
		t, db, `INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, ?, ?)`, "a", "a", "<?xmla",
	)
	execNoErr(
		t, db, `INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, ?, ?)`, "b", "b", "<?xmlb",
	)
	execNoErr(
		t, db, `INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, ?, ?)`, "c", "c", "<?xmlc",
	)

	selectStmt := `SELECT created_at from nano_commands WHERE command_uuid = ? AND created_at = updated_at`
	var item1CreatedAt, item2CreatedAt, item3CreatedAt time.Time
	require.NoError(t, db.Get(&item1CreatedAt, selectStmt, "a"))
	require.NoError(t, db.Get(&item2CreatedAt, selectStmt, "b"))
	require.NoError(t, db.Get(&item3CreatedAt, selectStmt, "c"))
	assert.NotZero(t, item1CreatedAt)
	assert.True(t, item1CreatedAt.Before(item2CreatedAt), "item1CreatedAt: %v, item2CreatedAt: %v", item1CreatedAt, item2CreatedAt)
	assert.True(t, item2CreatedAt.Before(item3CreatedAt), "item2CreatedAt: %v, item3CreatedAt: %v", item2CreatedAt, item3CreatedAt)

}
