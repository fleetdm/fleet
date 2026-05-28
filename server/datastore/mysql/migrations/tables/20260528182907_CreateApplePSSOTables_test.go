package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260528182907(t *testing.T) {
	db := applyUpToPrev(t)

	hostInsert := `INSERT INTO hosts (hardware_serial, osquery_host_id, node_key, uuid, platform) VALUES (?, ?, ?, ?, ?)`
	hostID1 := execNoErrLastID(t, db, hostInsert, "serial-1", "osq-1", "node-key-1", "uuid-1", "darwin")
	hostID2 := execNoErrLastID(t, db, hostInsert, "serial-2", "osq-2", "node-key-2", "uuid-2", "darwin")

	applyNext(t, db)

	// Insert a device row with explicit values.
	_, err := db.Exec(`
		INSERT INTO mdm_apple_psso_devices
			(host_id, device_uuid, signing_key_pem, encryption_key_pem, key_exchange_key)
		VALUES (?, ?, ?, ?, ?)
	`, hostID1, "ABCDEFGH-0000-0000-0000-111111111111", "signing-pem-1", "encryption-pem-1", []byte("0123456789abcdef0123456789abcdef"))
	require.NoError(t, err)

	var (
		gotUUID        string
		gotSigningPEM  string
		gotEncryptPEM  string
		gotKEK         []byte
		gotCreatedAt   time.Time
		gotUpdatedAt   time.Time
	)
	err = db.QueryRow(`
		SELECT device_uuid, signing_key_pem, encryption_key_pem, key_exchange_key, created_at, updated_at
		FROM mdm_apple_psso_devices WHERE host_id = ?
	`, hostID1).Scan(&gotUUID, &gotSigningPEM, &gotEncryptPEM, &gotKEK, &gotCreatedAt, &gotUpdatedAt)
	require.NoError(t, err)
	assert.Equal(t, "ABCDEFGH-0000-0000-0000-111111111111", gotUUID)
	assert.Equal(t, "signing-pem-1", gotSigningPEM)
	assert.Equal(t, "encryption-pem-1", gotEncryptPEM)
	assert.Equal(t, []byte("0123456789abcdef0123456789abcdef"), gotKEK)
	assert.False(t, gotCreatedAt.IsZero())
	assert.False(t, gotUpdatedAt.IsZero())

	// Duplicate host_id is rejected by the PK.
	_, err = db.Exec(`
		INSERT INTO mdm_apple_psso_devices
			(host_id, device_uuid, signing_key_pem, encryption_key_pem, key_exchange_key)
		VALUES (?, ?, ?, ?, ?)
	`, hostID1, "different-uuid", "x", "y", []byte("kek"))
	require.Error(t, err)

	// Duplicate device_uuid across hosts is rejected by the unique index.
	_, err = db.Exec(`
		INSERT INTO mdm_apple_psso_devices
			(host_id, device_uuid, signing_key_pem, encryption_key_pem, key_exchange_key)
		VALUES (?, ?, ?, ?, ?)
	`, hostID2, "ABCDEFGH-0000-0000-0000-111111111111", "x", "y", []byte("kek"))
	require.Error(t, err)

	// FK to hosts is enforced.
	_, err = db.Exec(`
		INSERT INTO mdm_apple_psso_devices
			(host_id, device_uuid, signing_key_pem, encryption_key_pem, key_exchange_key)
		VALUES (?, ?, ?, ?, ?)
	`, 999999, "ghost-uuid", "x", "y", []byte("kek"))
	require.Error(t, err)

	// Second host can register independently.
	_, err = db.Exec(`
		INSERT INTO mdm_apple_psso_devices
			(host_id, device_uuid, signing_key_pem, encryption_key_pem, key_exchange_key)
		VALUES (?, ?, ?, ?, ?)
	`, hostID2, "ABCDEFGH-0000-0000-0000-222222222222", "signing-pem-2", "encryption-pem-2", []byte("ffeeddccbbaa99887766554433221100"))
	require.NoError(t, err)

	// Insert key_id rows for host1: one signing, one encryption.
	_, err = db.Exec(`
		INSERT INTO mdm_apple_psso_key_ids (kid, host_id, key_type, pem)
		VALUES (?, ?, ?, ?)
	`, "kid-sign-host1", hostID1, "signing", "signing-pem-1")
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO mdm_apple_psso_key_ids (kid, host_id, key_type, pem)
		VALUES (?, ?, ?, ?)
	`, "kid-enc-host1", hostID1, "encryption", "encryption-pem-1")
	require.NoError(t, err)

	// Duplicate kid is rejected by PK.
	_, err = db.Exec(`
		INSERT INTO mdm_apple_psso_key_ids (kid, host_id, key_type, pem)
		VALUES (?, ?, ?, ?)
	`, "kid-sign-host1", hostID2, "signing", "x")
	require.Error(t, err)

	// Duplicate (host_id, key_type) is rejected by unique index — a host has at most one signing and one encryption key.
	_, err = db.Exec(`
		INSERT INTO mdm_apple_psso_key_ids (kid, host_id, key_type, pem)
		VALUES (?, ?, ?, ?)
	`, "kid-sign-host1-v2", hostID1, "signing", "x")
	require.Error(t, err)

	// Invalid key_type is rejected by ENUM.
	_, err = db.Exec(`
		INSERT INTO mdm_apple_psso_key_ids (kid, host_id, key_type, pem)
		VALUES (?, ?, ?, ?)
	`, "kid-bogus", hostID1, "bogus", "x")
	require.Error(t, err)

	// FK to hosts is enforced.
	_, err = db.Exec(`
		INSERT INTO mdm_apple_psso_key_ids (kid, host_id, key_type, pem)
		VALUES (?, ?, ?, ?)
	`, "kid-ghost", 999999, "signing", "x")
	require.Error(t, err)

	// ON DELETE CASCADE: deleting host2 wipes its psso rows.
	_, err = db.Exec(`DELETE FROM hosts WHERE id = ?`, hostID2)
	require.NoError(t, err)

	var devicesRemaining int
	err = db.QueryRow(`SELECT COUNT(*) FROM mdm_apple_psso_devices WHERE host_id = ?`, hostID2).Scan(&devicesRemaining)
	require.NoError(t, err)
	assert.Equal(t, 0, devicesRemaining)

	// host1's rows survive.
	var host1Devices int
	err = db.QueryRow(`SELECT COUNT(*) FROM mdm_apple_psso_devices WHERE host_id = ?`, hostID1).Scan(&host1Devices)
	require.NoError(t, err)
	assert.Equal(t, 1, host1Devices)

	var host1Keys int
	err = db.QueryRow(`SELECT COUNT(*) FROM mdm_apple_psso_key_ids WHERE host_id = ?`, hostID1).Scan(&host1Keys)
	require.NoError(t, err)
	assert.Equal(t, 2, host1Keys)
}
