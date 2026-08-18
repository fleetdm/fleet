package tables

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260818162738(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a policy that pre-dates the migration, with timestamps far enough back that any
	// rewrite of the row would be obvious.
	policyID := execNoErrLastID(
		t, db,
		"INSERT INTO policies (name, query, description, checksum, created_at, updated_at) VALUES (?,?,?,?,?,?)",
		"policy1", "", "", "checksum1", "2020-01-01 00:00:00", "2020-01-02 03:04:05",
	)

	timestamps := func() (time.Time, time.Time) {
		var row struct {
			CreatedAt time.Time `db:"created_at"`
			UpdatedAt time.Time `db:"updated_at"`
		}
		require.NoError(t, db.GetContext(context.Background(), &row,
			`SELECT created_at, updated_at FROM policies WHERE id = ?`, policyID))
		return row.CreatedAt, row.UpdatedAt
	}
	createdBefore, updatedBefore := timestamps()

	applyNext(t, db)

	// Timestamps survive the migration. They matter because policies.updated_at is
	// ON UPDATE CURRENT_TIMESTAMP and the API serves it, so rewriting rows here would make every
	// policy look freshly edited.
	createdAfter, updatedAfter := timestamps()
	assert.Equal(t, createdBefore, createdAfter, "migration must not touch created_at")
	assert.Equal(t, updatedBefore, updatedAfter, "migration must not touch updated_at")

	// Existing rows get the default.
	var notifyBeforePatching bool
	require.NoError(t, db.GetContext(context.Background(), &notifyBeforePatching,
		`SELECT notify_before_patching FROM policies WHERE id = ?`, policyID))
	assert.False(t, notifyBeforePatching)

	// New rows can set the column.
	policy2 := execNoErrLastID(
		t, db, "INSERT INTO policies (name, query, description, checksum, notify_before_patching) VALUES (?,?,?,?,?)",
		"policy2", "", "", "checksum2", 1,
	)
	require.NoError(t, db.GetContext(context.Background(), &notifyBeforePatching,
		`SELECT notify_before_patching FROM policies WHERE id = ?`, policy2))
	assert.True(t, notifyBeforePatching)
}
