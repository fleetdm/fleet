package condaccess

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitAssets(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer mysql.TruncateTables(t, ds, []string{"mdm_config_assets"}...)

	ctx := t.Context()

	// Initialize assets
	err := initAssets(ds)
	require.NoError(t, err)

	// Verify assets were created
	expectedAssets := []fleet.MDMAssetName{
		fleet.MDMAssetConditionalAccessCACert,
		fleet.MDMAssetConditionalAccessCAKey,
	}
	savedAssets, err := ds.GetAllMDMConfigAssetsByName(ctx, expectedAssets, nil)
	require.NoError(t, err)
	require.Len(t, savedAssets, 2, "Should have created both CA cert and key")

	// Verify we have both cert and key
	var caCertAsset, caKeyAsset fleet.MDMConfigAsset
	var foundCert, foundKey bool
	for _, asset := range savedAssets {
		if asset.Name == fleet.MDMAssetConditionalAccessCACert {
			caCertAsset = asset
			foundCert = true
		} else if asset.Name == fleet.MDMAssetConditionalAccessCAKey {
			caKeyAsset = asset
			foundKey = true
		}
	}
	require.True(t, foundCert, "Should have CA cert")
	require.True(t, foundKey, "Should have CA key")

	// Verify cert is valid PEM
	pemBlock, _ := pem.Decode(caCertAsset.Value)
	require.NotNil(t, pemBlock, "CA cert should be valid PEM")
	require.Equal(t, "CERTIFICATE", pemBlock.Type, "PEM block should be a certificate")

	// Parse the certificate
	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	require.NoError(t, err, "Should be able to parse CA certificate")

	// Verify certificate attributes
	assert.Equal(t, "Fleet Conditional Access CA", cert.Subject.CommonName, "CA cert should have correct common name")
	assert.Contains(t, cert.Subject.Organization, "Local Certificate Authority", "CA cert should have correct organization")

	// Verify certificate is valid for 10 years (with tolerance for leap years)
	// 10 years can be 3652 days (2 leap years) or 3653 days (3 leap years)
	expectedMinDuration := 10*365*24*time.Hour - 2*24*time.Hour // Allow for non-leap years
	expectedMaxDuration := 10*365*24*time.Hour + 3*24*time.Hour // Allow for leap years
	actualDuration := cert.NotAfter.Sub(cert.NotBefore)
	assert.True(t, actualDuration >= expectedMinDuration && actualDuration <= expectedMaxDuration,
		"CA cert should be valid for approximately 10 years (actual: %v, expected: %v-%v)",
		actualDuration, expectedMinDuration, expectedMaxDuration)

	// Verify cert is a CA
	assert.True(t, cert.IsCA, "Certificate should be marked as CA")
	assert.True(t, cert.BasicConstraintsValid, "Basic constraints should be valid")

	// Verify key usage
	assert.Equal(t, x509.KeyUsageCertSign|x509.KeyUsageCRLSign|x509.KeyUsageDigitalSignature, cert.KeyUsage,
		"CA cert should have correct key usage")

	// Verify the certificate uses RSA public key
	rsaPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	require.True(t, ok, "Certificate should use RSA public key")
	assert.Equal(t, 2048, rsaPubKey.N.BitLen(), "RSA key should be 2048 bits")

	// Verify key is valid PEM and is RSA
	keyPemBlock, _ := pem.Decode(caKeyAsset.Value)
	require.NotNil(t, keyPemBlock, "CA key should be valid PEM")
	require.Equal(t, "RSA PRIVATE KEY", keyPemBlock.Type, "PEM block should be an RSA private key")

	// Parse and verify the private key
	rsaPrivKey, err := x509.ParsePKCS1PrivateKey(keyPemBlock.Bytes)
	require.NoError(t, err, "Should be able to parse RSA private key")
	assert.Equal(t, 2048, rsaPrivKey.N.BitLen(), "RSA private key should be 2048 bits")

	// Save original values for idempotency check
	originalCertValue := caCertAsset.Value
	originalKeyValue := caKeyAsset.Value

	// Second initialization - should not regenerate
	err = initAssets(ds)
	require.NoError(t, err)

	// Get the assets again
	secondAssets, err := ds.GetAllMDMConfigAssetsByName(ctx, expectedAssets, nil)
	require.NoError(t, err)
	require.Len(t, secondAssets, 2)

	// Verify values are unchanged
	for _, asset := range secondAssets {
		if asset.Name == fleet.MDMAssetConditionalAccessCACert {
			assert.Equal(t, originalCertValue, asset.Value, "CA cert should not be regenerated")
		} else if asset.Name == fleet.MDMAssetConditionalAccessCAKey {
			assert.Equal(t, originalKeyValue, asset.Value, "CA key should not be regenerated")
		}
	}
}
