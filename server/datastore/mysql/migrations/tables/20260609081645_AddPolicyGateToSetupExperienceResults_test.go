package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260609081645(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a policy and a setup-experience row before the migration so we can verify the new column and FK behavior.
	execNoErr(t, db, `INSERT INTO policies (name, query, description, checksum) VALUES ('p', 'SELECT 1', '', 'csum-policy-gate')`)
	var policyID uint
	require.NoError(t, db.Get(&policyID, `SELECT id FROM policies WHERE checksum = 'csum-policy-gate'`))

	// Apply current migration.
	applyNext(t, db)

	// A gated row carrying policy_id can be inserted.
	execNoErr(t, db,
		`INSERT INTO setup_experience_status_results (host_uuid, name, status, policy_id) VALUES (?, ?, 'pending', ?)`,
		"uuid-gated", "Gated app", policyID,
	)

	// Read it back.
	var gotPolicyID *uint
	require.NoError(t, db.Get(&gotPolicyID,
		`SELECT policy_id FROM setup_experience_status_results WHERE host_uuid = 'uuid-gated'`))
	require.NotNil(t, gotPolicyID)
	require.Equal(t, policyID, *gotPolicyID)

	// An un-gated row leaves policy_id NULL.
	execNoErr(t, db,
		`INSERT INTO setup_experience_status_results (host_uuid, name, status) VALUES (?, ?, 'pending')`,
		"uuid-ungated", "Un-gated app",
	)
	var ungatedPolicyID *uint
	require.NoError(t, db.Get(&ungatedPolicyID,
		`SELECT policy_id FROM setup_experience_status_results WHERE host_uuid = 'uuid-ungated'`))
	require.Nil(t, ungatedPolicyID)

	// Deleting the policy sets the gate back to NULL (ON DELETE SET NULL), it does not delete the setup-experience row.
	execNoErr(t, db, `DELETE FROM policies WHERE id = ?`, policyID)
	var afterDelete *uint
	require.NoError(t, db.Get(&afterDelete,
		`SELECT policy_id FROM setup_experience_status_results WHERE host_uuid = 'uuid-gated'`))
	require.Nil(t, afterDelete)
}
