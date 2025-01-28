package tables

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUp_20240314151747(t *testing.T) {
	db := applyUpToPrev(t)

	policy1 := execNoErrLastID(
		t, db, "INSERT INTO policies (name, query, description, checksum) VALUES (?,?,?,?)", "policy", "", "", "checksum",
	)

	// Apply current migration.
	applyNext(t, db)

	var policyCheck []struct {
		ID         int64 `db:"id"`
		CalEnabled bool  `db:"calendar_events_enabled"`
	}
	err := db.SelectContext(context.Background(), &policyCheck, `SELECT id, calendar_events_enabled FROM policies ORDER BY id`)
	require.NoError(t, err)
	require.Len(t, policyCheck, 1)
	assert.Equal(t, policy1, policyCheck[0].ID)
	assert.Equal(t, false, policyCheck[0].CalEnabled)

	policy2 := execNoErrLastID(
		t, db, "INSERT INTO policies (name, query, description, checksum, calendar_events_enabled) VALUES (?,?,?,?,?)", "policy2", "", "",
		"checksum2", 1,
	)

	policyCheck = nil
	err = db.SelectContext(context.Background(), &policyCheck, `SELECT id, calendar_events_enabled FROM policies WHERE id = ?`, policy2)
	require.NoError(t, err)
	require.Len(t, policyCheck, 1)
	assert.Equal(t, policy2, policyCheck[0].ID)
	assert.Equal(t, true, policyCheck[0].CalEnabled)

}
