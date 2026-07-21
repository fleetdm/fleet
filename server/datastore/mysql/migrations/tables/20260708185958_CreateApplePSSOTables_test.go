package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260708185958(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	const (
		hostUUID1 = "ABCDEFGH-0000-0000-0000-111111111111"
		hostUUID2 = "ABCDEFGH-0000-0000-0000-222222222222"
	)

	// Register two devices.
	execNoErr(t, db, `INSERT INTO mdm_apple_psso_devices (host_uuid) VALUES (?)`, hostUUID1)
	execNoErr(t, db, `INSERT INTO mdm_apple_psso_devices (host_uuid) VALUES (?)`, hostUUID2)

	var (
		gotCreatedAt time.Time
		gotUpdatedAt time.Time
	)
	err := db.QueryRow(`
		SELECT created_at, updated_at FROM mdm_apple_psso_devices WHERE host_uuid = ?
	`, hostUUID1).Scan(&gotCreatedAt, &gotUpdatedAt)
	require.NoError(t, err)
	assert.False(t, gotCreatedAt.IsZero())
	assert.False(t, gotUpdatedAt.IsZero())

	// Duplicate host_uuid is rejected by the PK.
	_, err = db.Exec(`INSERT INTO mdm_apple_psso_devices (host_uuid) VALUES (?)`, hostUUID1)
	require.Error(t, err)

	keyInsert := `INSERT INTO mdm_apple_psso_keys (kid, host_uuid, key_type, pem) VALUES (?, ?, ?, ?)`

	// One signing and one encryption key for host1.
	execNoErr(t, db, keyInsert, "kid-sign-host1", hostUUID1, "signing", "signing-pem-1")
	execNoErr(t, db, keyInsert, "kid-enc-host1", hostUUID1, "encryption", "encryption-pem-1")

	// Multiple keys of the same type per host are allowed (re-registration
	// keeps old keys working).
	execNoErr(t, db, keyInsert, "kid-sign-host1-v2", hostUUID1, "signing", "signing-pem-1-v2")
	execNoErr(t, db, keyInsert, "kid-enc-host1-v2", hostUUID1, "encryption", "encryption-pem-1-v2")

	// Duplicate kid is rejected by the PK.
	_, err = db.Exec(keyInsert, "kid-sign-host1", hostUUID2, "signing", "x")
	require.Error(t, err)

	// Invalid key_type is rejected by the ENUM.
	_, err = db.Exec(keyInsert, "kid-bogus", hostUUID1, "bogus", "x")
	require.Error(t, err)

	// Keys must reference a registered device.
	_, err = db.Exec(keyInsert, "kid-ghost", "no-such-device-uuid", "signing", "x")
	require.Error(t, err)

	// Key timestamps are populated.
	err = db.QueryRow(`
		SELECT created_at, updated_at FROM mdm_apple_psso_keys WHERE kid = ?
	`, "kid-sign-host1").Scan(&gotCreatedAt, &gotUpdatedAt)
	require.NoError(t, err)
	assert.False(t, gotCreatedAt.IsZero())
	assert.False(t, gotUpdatedAt.IsZero())

	// ON DELETE CASCADE: deleting a device wipes its keys.
	execNoErr(t, db, keyInsert, "kid-sign-host2", hostUUID2, "signing", "signing-pem-2")
	execNoErr(t, db, `DELETE FROM mdm_apple_psso_devices WHERE host_uuid = ?`, hostUUID2)

	var keysRemaining int
	err = db.QueryRow(`SELECT COUNT(*) FROM mdm_apple_psso_keys WHERE host_uuid = ?`, hostUUID2).Scan(&keysRemaining)
	require.NoError(t, err)
	assert.Equal(t, 0, keysRemaining)

	// host1's rows survive.
	var host1Keys int
	err = db.QueryRow(`SELECT COUNT(*) FROM mdm_apple_psso_keys WHERE host_uuid = ?`, hostUUID1).Scan(&host1Keys)
	require.NoError(t, err)
	assert.Equal(t, 4, host1Keys)
}
