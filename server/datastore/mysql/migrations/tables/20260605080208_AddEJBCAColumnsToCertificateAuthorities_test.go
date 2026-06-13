package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260605080208(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	// The new columns should be readable, and the ENUM should accept 'ejbca'.
	_, err := db.Exec(`
INSERT INTO certificate_authorities (
	type, name, url,
	client_cert_pem, client_key_encrypted, trust_ca_bundle_pem,
	ejbca_ca_name, ejbca_certificate_profile, ejbca_end_entity_profile,
	ejbca_username_template
) VALUES (
	'ejbca', 'TestEJBCA', 'https://ejbca.example.com:8443',
	?, ?, ?,
	'WifiIssuingCA', 'WifiClientProfile', 'WifiUsers',
	'$FLEET_VAR_HOST_HARDWARE_SERIAL'
)`,
		[]byte("-----BEGIN CERTIFICATE-----\nfake\n-----END CERTIFICATE-----"),
		[]byte("encrypted-key-bytes"),
		[]byte("-----BEGIN CERTIFICATE-----\nca\n-----END CERTIFICATE-----"),
	)
	require.NoError(t, err, "insert with EJBCA type and new columns should succeed")

	var (
		caName  string
		profile string
	)
	err = db.QueryRow(`SELECT ejbca_ca_name, ejbca_certificate_profile FROM certificate_authorities WHERE name = 'TestEJBCA'`).Scan(&caName, &profile)
	require.NoError(t, err)
	require.Equal(t, "WifiIssuingCA", caName)
	require.Equal(t, "WifiClientProfile", profile)
}
