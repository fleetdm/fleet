package tables

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260818162738(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a policy that pre-dates the migration.
	policyID := execNoErrLastID(
		t, db, "INSERT INTO policies (name, query, description, checksum) VALUES (?,?,?,?)",
		"policy1", "", "", "checksum1",
	)

	applyNext(t, db)

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
