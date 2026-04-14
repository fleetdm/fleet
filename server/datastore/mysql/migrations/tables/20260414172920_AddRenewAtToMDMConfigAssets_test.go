package tables

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5" //nolint:gosec
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// encrypt_20260414172920 is a test-local copy of the AES-GCM encryption
// routine used by the datastore to store mdm_config_assets. Duplicated here
// because the mysql package's encrypt function is not accessible from the
// migrations test package.
func encrypt_20260414172920(plainText []byte, privateKey string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(privateKey))
	if err != nil {
		return nil, fmt.Errorf("create new cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create new gcm: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	return aesGCM.Seal(nonce, nonce, plainText, nil), nil
}

// md5HexChecksum returns the uppercase hex-encoded MD5 hash of b.
func md5HexChecksum(b []byte) string {
	rawChecksum := md5.Sum(b) //nolint:gosec
	return strings.ToUpper(hex.EncodeToString(rawChecksum[:]))
}

// generateTestCertPEM creates a self-signed PEM certificate with the given
// NotAfter time, suitable for use as a fake APNs (or other) certificate in tests.
func generateTestCertPEM(t *testing.T, notAfter time.Time) []byte {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-apns-cert"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     notAfter,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
}

// setPrivateKeyEnv sets FLEET_SERVER_PRIVATE_KEY for the duration of the test
// and restores the original value when the test completes.
func setPrivateKeyEnv(t *testing.T, key string) {
	t.Helper()
	orig := os.Getenv("FLEET_SERVER_PRIVATE_KEY")
	require.NoError(t, os.Setenv("FLEET_SERVER_PRIVATE_KEY", key))
	t.Cleanup(func() {
		if orig == "" {
			os.Unsetenv("FLEET_SERVER_PRIVATE_KEY") //nolint:errcheck
		} else {
			os.Setenv("FLEET_SERVER_PRIVATE_KEY", orig) //nolint:errcheck
		}
	})
}

// testPrivateKey is a 32-byte key for AES-256 used in tests.
const testPrivateKey = "test-key-exactly-32-bytes-long!!"

func TestUp_20260414172920_BackfillAPNsCert(t *testing.T) {
	// Set the private key so the migration can decrypt existing assets.
	setPrivateKeyEnv(t, testPrivateKey)

	db := applyUpToPrev(t)

	// Generate a self-signed certificate with a known expiration date.
	certExpiry := time.Date(2032, 3, 15, 10, 30, 0, 0, time.UTC)
	certPEM := generateTestCertPEM(t, certExpiry)

	// Insert an encrypted APNs certificate asset (simulates a pre-migration state).
	encrypted, err := encrypt_20260414172920(certPEM, testPrivateKey)
	require.NoError(t, err)
	hexChecksum := md5HexChecksum(encrypted)
	apnsID := execNoErrLastID(t, db,
		`INSERT INTO mdm_config_assets (name, value, md5_checksum) VALUES (?, ?, UNHEX(?))`,
		"apns_cert", encrypted, hexChecksum,
	)
	require.NotZero(t, apnsID)

	// Also insert a non-certificate asset (a key) to verify it is NOT backfilled.
	keyData := []byte("this-is-a-private-key")
	encryptedKey, err := encrypt_20260414172920(keyData, testPrivateKey)
	require.NoError(t, err)
	keyChecksum := md5HexChecksum(encryptedKey)
	keyID := execNoErrLastID(t, db,
		`INSERT INTO mdm_config_assets (name, value, md5_checksum) VALUES (?, ?, UNHEX(?))`,
		"apns_key", encryptedKey, keyChecksum,
	)
	require.NotZero(t, keyID)

	// Insert a soft-deleted certificate asset to verify it is NOT backfilled.
	deletedCertPEM := generateTestCertPEM(t, time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC))
	encryptedDeleted, err := encrypt_20260414172920(deletedCertPEM, testPrivateKey)
	require.NoError(t, err)
	deletedChecksum := md5HexChecksum(encryptedDeleted)
	deletedID := execNoErrLastID(t, db,
		`INSERT INTO mdm_config_assets (name, value, md5_checksum, deletion_uuid) VALUES (?, ?, UNHEX(?), ?)`,
		"ca_cert", encryptedDeleted, deletedChecksum, "some-deletion-uuid",
	)
	require.NotZero(t, deletedID)

	// Apply the migration under test.
	applyNext(t, db)

	// Verify the APNs cert row was backfilled with the correct renew_at.
	var renewAt *time.Time
	err = db.Get(&renewAt, `SELECT renew_at FROM mdm_config_assets WHERE id = ?`, apnsID)
	require.NoError(t, err)
	require.NotNil(t, renewAt, "renew_at should be populated for the APNs certificate")
	require.True(t, certExpiry.Equal(*renewAt),
		"expected renew_at %v, got %v", certExpiry, *renewAt)

	// Verify the non-certificate asset was NOT backfilled.
	var keyRenewAt *time.Time
	err = db.Get(&keyRenewAt, `SELECT renew_at FROM mdm_config_assets WHERE id = ?`, keyID)
	require.NoError(t, err)
	require.Nil(t, keyRenewAt, "renew_at should remain NULL for non-certificate assets")

	// Verify the soft-deleted certificate was NOT backfilled.
	var deletedRenewAt *time.Time
	err = db.Get(&deletedRenewAt, `SELECT renew_at FROM mdm_config_assets WHERE id = ?`, deletedID)
	require.NoError(t, err)
	require.Nil(t, deletedRenewAt, "renew_at should remain NULL for soft-deleted certificate assets")
}

func TestUp_20260414172920_BackfillMultipleCerts(t *testing.T) {
	setPrivateKeyEnv(t, testPrivateKey)

	db := applyUpToPrev(t)

	// Insert several certificate assets with different names and expiration dates.
	type certAsset struct {
		name    string
		expiry  time.Time
		deleted bool
	}
	assets := []certAsset{
		{name: "apns_cert", expiry: time.Date(2032, 3, 15, 10, 30, 0, 0, time.UTC)},
		{name: "ca_cert", expiry: time.Date(2035, 12, 31, 23, 59, 59, 0, time.UTC)},
		{name: "abm_cert", expiry: time.Date(2028, 6, 1, 0, 0, 0, 0, time.UTC)},
	}

	type insertedAsset struct {
		id     int64
		expiry time.Time
	}
	inserted := make([]insertedAsset, 0, len(assets))

	for _, a := range assets {
		certPEM := generateTestCertPEM(t, a.expiry)
		encrypted, err := encrypt_20260414172920(certPEM, testPrivateKey)
		require.NoError(t, err)
		checksum := md5HexChecksum(encrypted)

		deletionUUID := ""
		id := execNoErrLastID(t, db,
			`INSERT INTO mdm_config_assets (name, value, md5_checksum, deletion_uuid) VALUES (?, ?, UNHEX(?), ?)`,
			a.name, encrypted, checksum, deletionUUID,
		)
		require.NotZero(t, id)
		inserted = append(inserted, insertedAsset{id: id, expiry: a.expiry})
	}

	// Apply migration.
	applyNext(t, db)

	// Verify each certificate got the correct renew_at.
	for i, ia := range inserted {
		var renewAt *time.Time
		err := db.Get(&renewAt, `SELECT renew_at FROM mdm_config_assets WHERE id = ?`, ia.id)
		require.NoError(t, err)
		require.NotNil(t, renewAt, "asset %d (%s): renew_at should be populated", i, assets[i].name)
		require.True(t, ia.expiry.Equal(*renewAt),
			"asset %d (%s): expected renew_at %v, got %v", i, assets[i].name, ia.expiry, *renewAt)
	}
}

func TestUp_20260414172920_NoAssets(t *testing.T) {
	setPrivateKeyEnv(t, testPrivateKey)

	db := applyUpToPrev(t)

	// No assets in the table — migration should succeed without error.
	applyNext(t, db)

	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM mdm_config_assets WHERE renew_at IS NOT NULL`)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestUp_20260414172920_NoPrivateKey(t *testing.T) {
	// Ensure the env var is unset so the migration skips the backfill.
	setPrivateKeyEnv(t, "")

	db := applyUpToPrev(t)

	// Insert a certificate asset. Without the private key the migration should
	// still succeed but skip the backfill (renew_at stays NULL).
	certPEM := generateTestCertPEM(t, time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC))
	encrypted, err := encrypt_20260414172920(certPEM, testPrivateKey)
	require.NoError(t, err)
	checksum := md5HexChecksum(encrypted)
	assetID := execNoErrLastID(t, db,
		`INSERT INTO mdm_config_assets (name, value, md5_checksum) VALUES (?, ?, UNHEX(?))`,
		"apns_cert", encrypted, checksum,
	)
	require.NotZero(t, assetID)

	// Apply migration — should not fail despite missing key.
	applyNext(t, db)

	// Verify the column exists but renew_at is NULL (backfill was skipped).
	var renewAt *time.Time
	err = db.Get(&renewAt, `SELECT renew_at FROM mdm_config_assets WHERE id = ?`, assetID)
	require.NoError(t, err)
	require.Nil(t, renewAt, "renew_at should remain NULL when private key is not available")
}

func TestUp_20260414172920_WrongPrivateKey(t *testing.T) {
	// Set a different key than the one used to encrypt. The migration should
	// gracefully skip rows it cannot decrypt rather than failing.
	setPrivateKeyEnv(t, "wrong-key-exactly-32-bytes-long!")

	db := applyUpToPrev(t)

	certPEM := generateTestCertPEM(t, time.Date(2032, 1, 1, 0, 0, 0, 0, time.UTC))
	encrypted, err := encrypt_20260414172920(certPEM, testPrivateKey)
	require.NoError(t, err)
	checksum := md5HexChecksum(encrypted)
	assetID := execNoErrLastID(t, db,
		`INSERT INTO mdm_config_assets (name, value, md5_checksum) VALUES (?, ?, UNHEX(?))`,
		"apns_cert", encrypted, checksum,
	)
	require.NotZero(t, assetID)

	// Migration should succeed even though decryption will fail.
	applyNext(t, db)

	// renew_at should remain NULL because decryption failed.
	var renewAt *time.Time
	err = db.Get(&renewAt, `SELECT renew_at FROM mdm_config_assets WHERE id = ?`, assetID)
	require.NoError(t, err)
	require.Nil(t, renewAt, "renew_at should remain NULL when decryption fails")
}
