package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261124143542(t *testing.T) {
	db := applyUpToPrev(t)

	newDEPHost := func(serial, model, platform string) int64 {
		hostID := execNoErrLastID(t, db,
			`INSERT INTO hosts (hardware_serial, hardware_model, platform) VALUES (?, ?, ?)`, serial, model, platform)
		execNoErr(t, db, `INSERT INTO host_dep_assignments (host_id, hardware_serial) VALUES (?, ?)`, hostID, serial)
		return hostID
	}
	pendingTV := newDEPHost("TV1", "APPLE TV 4K", "darwin")
	enrolledTV := newDEPHost("TV2", "AppleTV14,1", "darwin")
	mac := newDEPHost("MAC1", "MacBookPro18,3", "darwin")
	iPhoneNamedLikeTV := newDEPHost("PHONE1", "AppleTV14,1", "ios")

	applyNext(t, db)

	family := func(hostID int64) *string {
		var f *string
		require.NoError(t, db.Get(&f, `SELECT device_family FROM host_dep_assignments WHERE host_id = ?`, hostID))
		return f
	}
	require.Equal(t, new("AppleTV"), family(pendingTV))
	require.Equal(t, new("AppleTV"), family(enrolledTV))
	require.Nil(t, family(mac))
	require.Nil(t, family(iPhoneNamedLikeTV))

	for _, table := range []string{"mdm_apple_default_setup_assistants", "mdm_apple_setup_assistant_profiles"} {
		var n int
		require.NoError(t, db.Get(&n, `
			SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = ? AND column_name = 'tvos_profile_uuid' AND is_nullable = 'YES'`, table))
		require.Equal(t, 1, n, table)
	}
}
