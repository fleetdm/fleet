package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261124143542(t *testing.T) {
	db := applyUpToPrev(t)

	hostID := execNoErrLastID(t, db,
		`INSERT INTO hosts (hardware_serial, hardware_model, platform) VALUES ('TV1', 'Apple TV 4K (3rd generation)', 'darwin')`)
	execNoErr(t, db, `INSERT INTO host_dep_assignments (host_id, hardware_serial) VALUES (?, 'TV1')`, hostID)

	applyNext(t, db)

	var family *string
	require.NoError(t, db.Get(&family, `SELECT device_family FROM host_dep_assignments WHERE host_id = ?`, hostID))
	require.Nil(t, family)

	for _, table := range []string{"mdm_apple_default_setup_assistants", "mdm_apple_setup_assistant_profiles"} {
		var n int
		require.NoError(t, db.Get(&n, `
			SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = ? AND column_name = 'tvos_profile_uuid' AND is_nullable = 'YES'`, table))
		require.Equal(t, 1, n, table)
	}
}
