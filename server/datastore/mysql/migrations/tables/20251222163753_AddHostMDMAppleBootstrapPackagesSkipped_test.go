package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20251222163753(t *testing.T) {
	db := applyUpToPrev(t)

	db.Exec(`
          INSERT INTO nano_commands (command_uuid, request_type, command)
          VALUES ('cmd-1', 'foo', '<?xml')
	`)

	_, err := db.Exec(`INSERT INTO host_mdm_apple_bootstrap_packages (host_uuid, command_uuid) VALUES ('host-1', 'cmd-1')`)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	skipped := false
	err = db.QueryRow(`SELECT skipped FROM host_mdm_apple_bootstrap_packages WHERE host_uuid = 'host-1'`).Scan(&skipped)
	require.NoError(t, err)
	require.False(t, skipped)

	_, err = db.Exec(`INSERT INTO host_mdm_apple_bootstrap_packages (host_uuid, command_uuid, skipped) VALUES ('host-2', 'cmd-1', 1)`)
	require.Error(t, err)

	_, err = db.Exec(`INSERT INTO host_mdm_apple_bootstrap_packages (host_uuid, command_uuid, skipped) VALUES ('host-3', NULL, 0)`)
	require.Error(t, err)

	_, err = db.Exec(`INSERT INTO host_mdm_apple_bootstrap_packages (host_uuid, command_uuid, skipped) VALUES ('host-4', NULL, 1)`)
	require.NoError(t, err)

	_, err = db.Exec(`UPDATE host_mdm_apple_bootstrap_packages SET skipped=1 WHERE host_uuid='host-1'`)
	require.Error(t, err)
}
