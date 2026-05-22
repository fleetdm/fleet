package tables

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260522195236(t *testing.T) {
	db := applyUpToPrev(t)

	policy1 := execNoErrLastID(
		t, db, "INSERT INTO policies (name, query, description, checksum) VALUES (?,?,?,?)",
		"policy1", "", "", "checksum1",
	)

	applyNext(t, db)

	var policyCheck []struct {
		ID                           int64 `db:"id"`
		ContinuousAutomationsEnabled bool  `db:"continuous_automations_enabled"`
	}
	err := db.SelectContext(context.Background(), &policyCheck, `SELECT id, continuous_automations_enabled FROM policies WHERE id = ?`, policy1)
	require.NoError(t, err)
	require.Len(t, policyCheck, 1)
	assert.Equal(t, policy1, policyCheck[0].ID)
	assert.False(t, policyCheck[0].ContinuousAutomationsEnabled)

	policy2 := execNoErrLastID(
		t, db, "INSERT INTO policies (name, query, description, checksum, continuous_automations_enabled) VALUES (?,?,?,?,?)",
		"policy2", "", "", "checksum2", 1,
	)

	policyCheck = nil
	err = db.SelectContext(context.Background(), &policyCheck, `SELECT id, continuous_automations_enabled FROM policies WHERE id = ?`, policy2)
	require.NoError(t, err)
	require.Len(t, policyCheck, 1)
	assert.Equal(t, policy2, policyCheck[0].ID)
	assert.True(t, policyCheck[0].ContinuousAutomationsEnabled)
}
