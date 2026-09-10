package tables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var nanoCleanupGuardIndexes = []struct {
	table, index, column, seededValue string
}{
	{"host_vpp_software_installs", "idx_hvsi_verification_command_uuid", "verification_command_uuid", "vpp-verify-1"},
	{"host_in_house_software_installs", "idx_hihsi_verification_command_uuid", "verification_command_uuid", "ih-verify-1"},
	{"host_mdm_actions", "idx_hma_lock_ref", "lock_ref", "lock-1"},
	{"host_mdm_actions", "idx_hma_wipe_ref", "wipe_ref", "wipe-1"},
	{"host_mdm_actions", "idx_hma_unlock_ref", "unlock_ref", "unlock-1"},
	{"host_managed_local_account_passwords", "idx_hmlap_pending_command_uuid", "pending_command_uuid", "hmlap-pending-1"},
	{"host_recovery_key_passwords", "idx_rkp_pending_set_command_uuid", "pending_set_command_uuid", "rkp-pending-set-1"},
	{"host_recovery_key_passwords", "idx_rkp_pending_verify_command_uuid", "pending_verify_command_uuid", "rkp-pending-verify-1"},
	{"host_recovery_key_passwords", "idx_rkp_set_command_uuid", "set_command_uuid", "rkp-set-1"},
	{"host_recovery_key_passwords", "idx_rkp_verify_command_uuid", "verify_command_uuid", "rkp-verify-1"},
}

func TestUp_20260908184941(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform, name, latest_version) VALUES ('adam-1', 'darwin', 'App', '1.0')`)
	execNoErr(t, db, `
		INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid, verification_command_uuid)
		VALUES (1, 'adam-1', 'darwin', 'vpp-install-1', 'vpp-verify-1')`)
	appID := execNoErrLastID(t, db, `
		INSERT INTO in_house_apps (global_or_team_id, storage_id, platform, filename)
		VALUES (0, 'storage-1', 'ios', 'app.ipa')`)
	execNoErr(t, db, `
		INSERT INTO host_in_house_software_installs (host_id, in_house_app_id, platform, command_uuid, verification_command_uuid)
		VALUES (1, ?, 'ios', 'ih-install-1', 'ih-verify-1')`, appID)
	execNoErr(t, db, `INSERT INTO host_mdm_actions (host_id, lock_ref, wipe_ref, unlock_ref) VALUES (1, 'lock-1', 'wipe-1', 'unlock-1')`)
	execNoErr(t, db, `
		INSERT INTO host_managed_local_account_passwords (host_uuid, encrypted_password, command_uuid, status, pending_command_uuid)
		VALUES ('host-1', 'enc', 'hmlap-cmd-1', 'verified', 'hmlap-pending-1')`)
	execNoErr(t, db, `
		INSERT INTO host_recovery_key_passwords
			(host_uuid, encrypted_password, status, operation_type,
			 pending_set_command_uuid, pending_verify_command_uuid, set_command_uuid, verify_command_uuid)
		VALUES ('host-1', 'enc', 'verified', 'install', 'rkp-pending-set-1', 'rkp-pending-verify-1', 'rkp-set-1', 'rkp-verify-1')`)

	applyNext(t, db)

	for _, g := range nanoCleanupGuardIndexes {
		require.Equal(t, []string{g.column}, indexColumns(t, db, g.table, g.index), g.index)

		// Seeded rows survive the ALTER and the guard probe resolves by command UUID.
		var count int
		require.NoError(t, db.Get(&count, fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s = ?`, g.table, g.column), g.seededValue))
		require.Equal(t, 1, count, g.index)
	}
}

func TestUp_20260908184941_PartiallyApplied(t *testing.T) {
	db := applyUpToPrev(t)

	// A prior run that failed after some tables and part way through a
	// multi-index table leaves this shape behind.
	execNoErr(t, db, `ALTER TABLE host_vpp_software_installs ADD INDEX idx_hvsi_verification_command_uuid (verification_command_uuid)`)
	execNoErr(t, db, `ALTER TABLE host_mdm_actions ADD INDEX idx_hma_lock_ref (lock_ref)`)
	execNoErr(t, db, `ALTER TABLE host_recovery_key_passwords ADD INDEX idx_rkp_set_command_uuid (set_command_uuid)`)

	applyNext(t, db)

	for _, g := range nanoCleanupGuardIndexes {
		require.Equal(t, []string{g.column}, indexColumns(t, db, g.table, g.index), g.index)
	}
}
