package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261124154656(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// Verify android_ca_certificates table exists and allows insertion
	res, err := db.Exec(`
		INSERT INTO android_ca_certificates (
			team_id, name, certificate_pem, sha256_fingerprint, subject, issuer
		) VALUES (
			0, 'Test Root CA', '-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----',
			'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
			'CN=Test Root CA', 'CN=Test Root CA'
		)
	`)
	require.NoError(t, err)
	caID, err := res.LastInsertId()
	require.NoError(t, err)
	require.Greater(t, caID, int64(0))

	// Verify android_ca_certificate_labels table exists and allows insertion
	_, err = db.Exec(`
		INSERT INTO android_ca_certificate_labels (
			ca_certificate_id, label_name, exclude, require_all
		) VALUES (
			?, 'All Android Devices', 0, 0
		)
	`, caID)
	require.NoError(t, err)

	// Verify unique constraint on (team_id, name)
	_, err = db.Exec(`
		INSERT INTO android_ca_certificates (
			team_id, name, certificate_pem, sha256_fingerprint, subject, issuer
		) VALUES (
			0, 'Test Root CA', 'pem', 'hash', 'sub', 'iss'
		)
	`)
	require.Error(t, err)
}
