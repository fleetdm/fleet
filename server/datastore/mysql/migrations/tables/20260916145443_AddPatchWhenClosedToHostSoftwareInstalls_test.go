package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20260916145443(t *testing.T) {
	db := applyUpToPrev(t)

	// checksum is BINARY(16) NOT NULL, so use MD5(name) as a 16-byte value.
	insertPolicy := func(name string, patchWhenClosed int) int64 {
		return execNoErrLastID(t, db,
			`INSERT INTO policies (name, query, description, checksum, patch_when_closed) VALUES (?, 'SELECT 1', '', UNHEX(MD5(?)), ?)`,
			name, name, patchWhenClosed,
		)
	}
	patchPolicyID := insertPolicy("patch-policy", 1)
	ordinaryPolicyID := insertPolicy("ordinary-policy", 0)

	// pending_install shape: host_id set, no exit codes, empty pre-install-query
	// output NULL (not ''), canceled = 0, uninstall = 0. Generated column
	// execution_status evaluates to 'pending_install'.
	insertPending := func(execID string, policyID *int64) {
		execNoErrLastID(t, db,
			`INSERT INTO host_software_installs (execution_id, host_id, policy_id) VALUES (?, 1, ?)`,
			execID, policyID,
		)
	}
	// installed shape: install_script_exit_code = 0. Generated column resolves
	// to 'installed', so the backfill must not touch it even though its policy
	// currently has patch_when_closed = 1.
	insertInstalled := func(execID string, policyID *int64) {
		execNoErrLastID(t, db,
			`INSERT INTO host_software_installs (execution_id, host_id, policy_id, install_script_exit_code) VALUES (?, 1, ?, 0)`,
			execID, policyID,
		)
	}
	// failed_install shape: empty pre_install_query_output. Generated column
	// resolves to 'failed_install', so also skipped by the backfill.
	insertFailed := func(execID string, policyID *int64) {
		execNoErrLastID(t, db,
			`INSERT INTO host_software_installs (execution_id, host_id, policy_id, pre_install_query_output) VALUES (?, 1, ?, '')`,
			execID, policyID,
		)
	}

	insertPending("pending-patch", &patchPolicyID)
	insertPending("pending-ordinary", &ordinaryPolicyID)
	insertPending("pending-no-policy", nil)
	insertInstalled("installed-patch", &patchPolicyID)
	insertFailed("failed-patch", &patchPolicyID)

	applyNext(t, db)

	getFlag := func(db *sqlx.DB, execID string) int {
		var v int
		require.NoError(t, db.Get(&v,
			`SELECT patch_when_closed FROM host_software_installs WHERE execution_id = ?`, execID))
		return v
	}

	require.Equal(t, 1, getFlag(db, "pending-patch"),
		"in-flight row on a patch-when-closed policy must be stamped so fleetd and the classifier agree post-upgrade")
	require.Equal(t, 0, getFlag(db, "pending-ordinary"), "pending row on an ordinary policy stays 0")
	require.Equal(t, 0, getFlag(db, "pending-no-policy"), "pending row with no policy stays 0")
	require.Equal(t, 0, getFlag(db, "installed-patch"),
		"completed rows already have a classifier verdict; backfill must not rewrite them")
	require.Equal(t, 0, getFlag(db, "failed-patch"),
		"failed rows already have a classifier verdict; backfill must not rewrite them")
}
