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
	err := initAssets(ctx, ds)
	require.NoError(t, err)

	// Verify all assets were created
	expectedAssets := []fleet.MDMAssetName{
		fleet.MDMAssetConditionalAccessCACert,
		fleet.MDMAssetConditionalAccessCAKey,
		fleet.MDMAssetConditionalAccessIDPCert,
		fleet.MDMAssetConditionalAccessIDPKey,
	}
	savedAssets, err := ds.GetAllMDMConfigAssetsByName(ctx, expectedAssets, nil)
	require.NoError(t, err)
	require.Len(t, savedAssets, 4, "Should have created all four assets")

	// Extract assets by name
	caCertAsset, foundCACert := savedAssets[fleet.MDMAssetConditionalAccessCACert]
	caKeyAsset, foundCAKey := savedAssets[fleet.MDMAssetConditionalAccessCAKey]
	idpCertAsset, foundIdPCert := savedAssets[fleet.MDMAssetConditionalAccessIDPCert]
	idpKeyAsset, foundIdPKey := savedAssets[fleet.MDMAssetConditionalAccessIDPKey]

	require.True(t, foundCACert, "Should have CA cert")
	require.True(t, foundCAKey, "Should have CA key")
	require.True(t, foundIdPCert, "Should have IdP cert")
	require.True(t, foundIdPKey, "Should have IdP key")

	// Verify cert is valid PEM
	pemBlock, _ := pem.Decode(caCertAsset.Value)
	require.NotNil(t, pemBlock, "CA cert should be valid PEM")
	require.Equal(t, "CERTIFICATE", pemBlock.Type, "PEM block should be a certificate")

	// Parse the certificate
	cert, err := x509.ParseCertificate(pemBlock.Bytes)
	require.NoError(t, err, "Should be able to parse CA certificate")

	// Verify certificate attributes
	assert.Equal(t, "Fleet conditional access CA", cert.Subject.CommonName, "CA cert should have correct common name")
	assert.Contains(t, cert.Subject.Organization, "Local certificate authority", "CA cert should have correct organization")

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

	// Verify IdP cert is valid PEM
	idpPemBlock, _ := pem.Decode(idpCertAsset.Value)
	require.NotNil(t, idpPemBlock, "IdP cert should be valid PEM")
	require.Equal(t, "CERTIFICATE", idpPemBlock.Type, "PEM block should be a certificate")

	// Parse the IdP certificate
	idpCert, err := x509.ParseCertificate(idpPemBlock.Bytes)
	require.NoError(t, err, "Should be able to parse IdP certificate")

	// Verify IdP certificate attributes
	assert.Equal(t, "Fleet conditional access IdP", idpCert.Subject.CommonName, "IdP cert should have correct common name")
	assert.Contains(t, idpCert.Subject.Organization, "Local certificate authority", "IdP cert should have correct organization")

	// Save original values for idempotency check
	originalCACertValue := caCertAsset.Value
	originalCAKeyValue := caKeyAsset.Value
	originalIdPCertValue := idpCertAsset.Value
	originalIdPKeyValue := idpKeyAsset.Value

	// Second initialization - should not regenerate
	err = initAssets(ctx, ds)
	require.NoError(t, err)

	// Get all assets again
	secondAssets, err := ds.GetAllMDMConfigAssetsByName(ctx, expectedAssets, nil)
	require.NoError(t, err)
	require.Len(t, secondAssets, 4)

	// Verify all values are unchanged (idempotent)
	assert.Equal(t, originalCACertValue, secondAssets[fleet.MDMAssetConditionalAccessCACert].Value, "CA cert should not be regenerated")
	assert.Equal(t, originalCAKeyValue, secondAssets[fleet.MDMAssetConditionalAccessCAKey].Value, "CA key should not be regenerated")
	assert.Equal(t, originalIdPCertValue, secondAssets[fleet.MDMAssetConditionalAccessIDPCert].Value, "IdP cert should not be regenerated")
	assert.Equal(t, originalIdPKeyValue, secondAssets[fleet.MDMAssetConditionalAccessIDPKey].Value, "IdP key should not be regenerated")
}
