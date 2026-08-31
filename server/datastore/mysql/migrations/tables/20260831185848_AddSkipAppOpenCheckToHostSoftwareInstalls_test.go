package tables

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260831185848(t *testing.T) {
	db := applyUpToPrev(t)

	hostID := execNoErrLastID(t, db,
		`INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, 'darwin')`,
		"osquery-1", "node-key-1", "host1", "uuid-1",
	)
	execNoErr(t, db,
		`INSERT INTO host_software_installs (execution_id, host_id, installer_filename, version, software_title_name)
			VALUES (?, ?, '', '', '')`,
		"exec-1", hostID,
	)

	applyNext(t, db)

	// Installs queued before the migration keep the app-open check.
	var skipAppOpenCheck bool
	require.NoError(t, db.GetContext(context.Background(), &skipAppOpenCheck,
		`SELECT skip_app_open_check FROM host_software_installs WHERE execution_id = ?`, "exec-1"))
	assert.False(t, skipAppOpenCheck)

	execNoErr(t, db,
		`INSERT INTO host_software_installs (execution_id, host_id, installer_filename, version, software_title_name, skip_app_open_check)
			VALUES (?, ?, '', '', '', 1)`,
		"exec-2", hostID,
	)
	require.NoError(t, db.GetContext(context.Background(), &skipAppOpenCheck,
		`SELECT skip_app_open_check FROM host_software_installs WHERE execution_id = ?`, "exec-2"))
	assert.True(t, skipAppOpenCheck)
}
