package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260908184941, Down_20260908184941)
}

// Up_20260908184941 indexes the command UUID columns the Apple MDM command
// cleanup checks for references before deleting a command. The install
// tables' existing verification index is on an expression and can't serve a
// lookup by UUID.
func Up_20260908184941(tx *sql.Tx) error {
	guards := []struct {
		table   string
		indexes []indexDef
	}{
		{"host_vpp_software_installs", []indexDef{
			{"idx_hvsi_verification_command_uuid", "verification_command_uuid"},
		}},
		{"host_in_house_software_installs", []indexDef{
			{"idx_hihsi_verification_command_uuid", "verification_command_uuid"},
		}},
		{"host_mdm_actions", []indexDef{
			{"idx_hma_lock_ref", "lock_ref"},
			{"idx_hma_wipe_ref", "wipe_ref"},
			{"idx_hma_unlock_ref", "unlock_ref"},
		}},
		{"host_managed_local_account_passwords", []indexDef{
			{"idx_hmlap_pending_command_uuid", "pending_command_uuid"},
		}},
		{"host_recovery_key_passwords", []indexDef{
			{"idx_rkp_pending_set_command_uuid", "pending_set_command_uuid"},
			{"idx_rkp_pending_verify_command_uuid", "pending_verify_command_uuid"},
			{"idx_rkp_set_command_uuid", "set_command_uuid"},
			{"idx_rkp_verify_command_uuid", "verify_command_uuid"},
		}},
	}
	for _, g := range guards {
		if err := addIndexesTx(tx, g.table, g.indexes...); err != nil {
			return err
		}
	}
	return nil
}

func Down_20260908184941(tx *sql.Tx) error {
	return nil
}
