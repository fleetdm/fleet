package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260901174312(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a vitals row from before the radio identifiers existed, to make
	// sure the new columns are added to an already-populated table.
	_, err := db.Exec(`
		INSERT INTO host_mdm_android_device_vitals (host_uuid, manufacturer)
		VALUES (?, ?)`,
		"android-host-uuid", "Google",
	)
	require.NoError(t, err)

	applyNext(t, db)

	var (
		imei *string
		meid *string
	)
	err = db.QueryRow(`
		SELECT imei, meid FROM host_mdm_android_device_vitals WHERE host_uuid = ?`,
		"android-host-uuid").Scan(&imei, &meid)
	require.NoError(t, err)
	require.Nil(t, imei, "an existing row should get a NULL imei")
	require.Nil(t, meid, "an existing row should get a NULL meid")

	_, err = db.Exec(`
		UPDATE host_mdm_android_device_vitals SET imei = ?, meid = ? WHERE host_uuid = ?`,
		"A1000031212", "A00000292788E1", "android-host-uuid",
	)
	require.NoError(t, err)

	var (
		gotIMEI string
		gotMEID string
	)
	err = db.QueryRow(`
		SELECT imei, meid FROM host_mdm_android_device_vitals WHERE host_uuid = ?`,
		"android-host-uuid").Scan(&gotIMEI, &gotMEID)
	require.NoError(t, err)
	require.Equal(t, "A1000031212", gotIMEI)
	require.Equal(t, "A00000292788E1", gotMEID)

	// A device reports at most one of the two, so each must be independently
	// nullable.
	_, err = db.Exec(`
		INSERT INTO host_mdm_android_device_vitals (host_uuid, imei) VALUES (?, ?)`,
		"gsm-only-host-uuid", "A1000031213",
	)
	require.NoError(t, err, "a row with only an imei should be accepted")
}
