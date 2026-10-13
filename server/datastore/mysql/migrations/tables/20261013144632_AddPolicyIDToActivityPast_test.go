package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261013144632(t *testing.T) {
	db := applyUpToPrev(t)

	policyActivityID := execNoErrLastID(t, db, `INSERT INTO activity_past (activity_type, details) VALUES ('ran_script', '{"policy_id": 7}')`)
	noPolicyActivityID := execNoErrLastID(t, db, `INSERT INTO activity_past (activity_type, details) VALUES ('ran_script', '{"team_id": 0}')`)

	applyNext(t, db)

	// read policy_id of activities created before the upgrade, it should come from details
	var policyID *uint
	require.NoError(t, db.Get(&policyID, `SELECT policy_id FROM activity_past WHERE id = ?`, policyActivityID))
	require.Equal(t, new(uint(7)), policyID)
	require.NoError(t, db.Get(&policyID, `SELECT policy_id FROM activity_past WHERE id = ?`, noPolicyActivityID))
	require.Nil(t, policyID)

	// insert an activity after the upgrade and look it up through the new index, it should be found by its policy
	newActivityID := execNoErrLastID(t, db, `INSERT INTO activity_past (activity_type, details) VALUES ('ran_script', '{"policy_id": 8}')`)
	var foundID int64
	require.NoError(t, db.Get(&foundID,
		`SELECT id FROM activity_past FORCE INDEX (idx_activity_past_policy_type_created) WHERE policy_id = 8 AND activity_type = 'ran_script'`))
	require.Equal(t, newActivityID, foundID)
}
