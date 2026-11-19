package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261117171906(t *testing.T) {
	db := applyUpToPrev(t)

	tables := []string{
		"network_interfaces",
		"osquery_options",
		"ca_config_assets",
		"legacy_host_filevault_profiles",
		"legacy_host_mdm_idp_accounts",
	}
	tableCount := func(table string) int {
		var n int
		require.NoError(t, db.Get(&n, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`, table))
		return n
	}
	for _, table := range tables {
		require.Equal(t, 1, tableCount(table), table)
	}

	hostID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES ('h1', 'k1', 'h1', 'u1', 'darwin')`)
	execNoErr(t, db, `INSERT INTO network_interfaces (host_id, ip_address, interface) VALUES (?, '10.0.0.1', 'en0')`, hostID)

	applyNext(t, db)

	for _, table := range tables {
		require.Zero(t, tableCount(table), table)
	}

	// Deleting a host no longer cascades into network_interfaces.
	execNoErr(t, db, `DELETE FROM hosts WHERE id = ?`, hostID)
}
