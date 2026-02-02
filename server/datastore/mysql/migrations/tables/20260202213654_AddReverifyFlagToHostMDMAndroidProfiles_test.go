package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260202213654(t *testing.T) {
	db := applyUpToPrev(t)
	detail := `bluetoothDisabled", and "passwordPolicies" settings couldn't apply to a host.
Reasons: PENDING, and USER_ACTION. Other settings are applied.`

	_, err := db.Exec(`INSERT INTO host_mdm_android_profiles (host_uuid, status, operation_type, detail) 
		VALUES ('hostid1', 'failed', 'install', ?)`, detail)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	var reverify bool
	err = db.QueryRow(`SELECT reverify FROM host_mdm_android_profiles WHERE host_uuid = 'hostid1'`).Scan(&reverify)
	require.False(t, reverify)
	require.NoError(t, err)

	_, err = db.Exec(`UPDATE host_mdm_android_profiles SET reverify = 1 WHERE host_uuid = 'hostid1'`)
	require.NoError(t, err)
}
