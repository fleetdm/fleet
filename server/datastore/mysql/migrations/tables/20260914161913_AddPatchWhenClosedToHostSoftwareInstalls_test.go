package tables

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20260914161913(t *testing.T) {
	// Shrink the batch size so the backfill must loop across multiple iterations
	// under test; catches a broken keyset advance (e.g. lastID not updated).
	origBatch := backfillPatchWhenClosedBatchSize
	backfillPatchWhenClosedBatchSize = 2
	t.Cleanup(func() { backfillPatchWhenClosedBatchSize = origBatch })

	db := applyUpToPrev(t)

	// Minimal policy row — checksum is BINARY(16) NOT NULL, use the MD5 of the
	// name as a 16-byte value.
	insertPolicy := func(name string, patchWhenClosed int) int64 {
		return execNoErrLastID(t, db,
			`INSERT INTO policies (name, query, description, checksum, patch_when_closed) VALUES (?, 'SELECT 1', '', UNHEX(MD5(?)), ?)`,
			name, name, patchWhenClosed,
		)
	}
	patchPolicyID := insertPolicy("patch-policy", 1)
	ordinaryPolicyID := insertPolicy("ordinary-policy", 0)

	insertInstall := func(execID string, policyID *int64) {
		execNoErrLastID(t, db,
			`INSERT INTO host_software_installs (execution_id, host_id, policy_id) VALUES (?, 1, ?)`,
			execID, policyID,
		)
	}
	// Seed more patch-policy rows than the batch size to exercise multiple loop
	// iterations. Interleave with rows that must NOT be touched so a broken
	// filter (e.g. missing WHERE clause) would flip them too.
	for i := 0; i < 5; i++ {
		insertInstall(fmt.Sprintf("hsi-patch-%d", i), &patchPolicyID)
		insertInstall(fmt.Sprintf("hsi-ordinary-%d", i), &ordinaryPolicyID)
	}
	insertInstall("hsi-no-policy", nil)

	applyNext(t, db)

	getFlag := func(db *sqlx.DB, execID string) int {
		var v int
		err := db.Get(&v, `SELECT patch_when_closed FROM host_software_installs WHERE execution_id = ?`, execID)
		require.NoError(t, err)
		return v
	}

	for i := 0; i < 5; i++ {
		require.Equal(t, 1, getFlag(db, fmt.Sprintf("hsi-patch-%d", i)),
			"all patch-policy rows should be backfilled to 1, even across batch boundaries")
		require.Equal(t, 0, getFlag(db, fmt.Sprintf("hsi-ordinary-%d", i)),
			"rows from an ordinary policy must stay 0")
	}
	require.Equal(t, 0, getFlag(db, "hsi-no-policy"), "row with no policy stays 0")

	// Toggling the source policy off after the backfill must NOT re-classify the
	// historical row — that's the whole reason we snapshot rather than live-read.
	_, err := db.Exec(`UPDATE policies SET patch_when_closed = 0 WHERE id = ?`, patchPolicyID)
	require.NoError(t, err)
	require.Equal(t, 1, getFlag(db, "hsi-patch-0"), "toggling policy off must not undo the snapshot")

	// Same for policy deletion (ON DELETE SET NULL zeros out policy_id, leaves flag intact).
	_, err = db.Exec(`DELETE FROM policies WHERE id = ?`, patchPolicyID)
	require.NoError(t, err)
	require.Equal(t, 1, getFlag(db, "hsi-patch-0"), "deleting the policy must not undo the snapshot")
}
