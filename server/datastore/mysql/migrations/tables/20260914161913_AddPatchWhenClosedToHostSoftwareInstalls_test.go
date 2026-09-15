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

	// insertSkipShaped seeds a row shaped exactly like a real patch-when-closed
	// skip: empty pre_install_query_output + no exit codes, which makes the
	// generated status column evaluate to 'failed_install' (via the "empty
	// pre-install output" branch of the CASE).
	insertSkipShaped := func(execID string, policyID *int64) {
		execNoErrLastID(t, db,
			`INSERT INTO host_software_installs (execution_id, host_id, policy_id, pre_install_query_output) VALUES (?, 1, ?, '')`,
			execID, policyID,
		)
	}
	// insertSucceeded seeds a policy-driven install that succeeded (exit code 0).
	// It should NOT get the flag stamped, even though its policy currently has
	// patch_when_closed = 1 — the row is not a candidate for skip classification.
	insertSucceeded := func(execID string, policyID *int64) {
		execNoErrLastID(t, db,
			`INSERT INTO host_software_installs (execution_id, host_id, policy_id, install_script_exit_code) VALUES (?, 1, ?, 0)`,
			execID, policyID,
		)
	}

	// Seed more skip-shaped rows than the batch size to exercise multiple loop
	// iterations. Interleave with rows that must NOT be touched so a broken
	// filter (e.g. missing WHERE clause) would flip them too.
	for i := range 5 {
		insertSkipShaped(fmt.Sprintf("hsi-patch-%d", i), &patchPolicyID)
		insertSkipShaped(fmt.Sprintf("hsi-ordinary-%d", i), &ordinaryPolicyID)
	}
	insertSkipShaped("hsi-no-policy", nil)
	// Non-candidate rows on the patch-when-closed policy: successful install
	// (must not be stamped — it's not a failed_install).
	insertSucceeded("hsi-patch-installed", &patchPolicyID)

	applyNext(t, db)

	getFlag := func(db *sqlx.DB, execID string) int {
		var v int
		err := db.Get(&v, `SELECT patch_when_closed FROM host_software_installs WHERE execution_id = ?`, execID)
		require.NoError(t, err)
		return v
	}

	for i := range 5 {
		require.Equal(t, 1, getFlag(db, fmt.Sprintf("hsi-patch-%d", i)),
			"skip-shaped rows from a patch-when-closed policy should be backfilled to 1, even across batch boundaries")
		require.Equal(t, 0, getFlag(db, fmt.Sprintf("hsi-ordinary-%d", i)),
			"skip-shaped rows from an ordinary policy must stay 0")
	}
	require.Equal(t, 0, getFlag(db, "hsi-no-policy"), "row with no policy stays 0")
	require.Equal(t, 0, getFlag(db, "hsi-patch-installed"),
		"successful install on a patch-when-closed policy must not be stamped — it can't be a historical skip")

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

// TestUp_20260914161913_EnableThenUpgrade_DocumentedAmbiguity locks in the
// documented "enable-then-upgrade" limitation: if an admin toggled a policy's
// patch_when_closed on AFTER an ordinary pre-install-query failure but BEFORE
// this migration ran, the backfill can't tell that historical row apart from a
// real skip and will flag it as a skip. The migration comment spells out this
// trade-off; this test pins the behavior so a future change that switches to a
// durable-signal recovery path fails loudly here.
func TestUp_20260914161913_EnableThenUpgrade_DocumentedAmbiguity(t *testing.T) {
	db := applyUpToPrev(t)

	// Policy starts OFF (ordinary use), then flips ON before upgrade.
	patchPolicyID := execNoErrLastID(t, db,
		`INSERT INTO policies (name, query, description, checksum, patch_when_closed) VALUES ('enable-later-policy', 'SELECT 1', '', UNHEX(MD5('enable-later-policy')), 0)`,
	)
	// Ordinary pre-install-query failure recorded while patch_when_closed = 0.
	execNoErrLastID(t, db,
		`INSERT INTO host_software_installs (execution_id, host_id, policy_id, pre_install_query_output) VALUES ('hsi-ordinary-failure', 1, ?, '')`,
		patchPolicyID,
	)
	// Admin flips patch_when_closed on before the customer upgrades.
	_, err := db.Exec(`UPDATE policies SET patch_when_closed = 1 WHERE id = ?`, patchPolicyID)
	require.NoError(t, err)

	applyNext(t, db)

	var v int
	err = db.Get(&v, `SELECT patch_when_closed FROM host_software_installs WHERE execution_id = ?`, "hsi-ordinary-failure")
	require.NoError(t, err)
	// Documented false positive: the historical row was NOT a skip, but the
	// backfill can't distinguish from a real skip using only current policy
	// state. The migration comment calls this out explicitly.
	require.Equal(t, 1, v, "documented enable-then-upgrade limitation: ordinary failure is relabeled as skip")
}
