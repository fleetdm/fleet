package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20261029154707(t *testing.T) {
	db := applyUpToPrev(t)

	policyID := execNoErrLastID(
		t, db,
		"INSERT INTO policies (name, query, description, checksum) VALUES (?,?,?,?)",
		"policy1", "", "", "checksum1",
	)

	applyNext(t, db)

	var patchDeadline bool
	require.NoError(t, db.GetContext(t.Context(), &patchDeadline,
		`SELECT patch_deadline FROM policies WHERE id = ?`, policyID))
	assert.False(t, patchDeadline)
}
