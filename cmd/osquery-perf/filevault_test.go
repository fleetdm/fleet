package main

import (
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fileVaultProfile mirrors the layout of Fleet's FileVault profile: an escrow payload that
// points at a pkcs1 certificate payload by UUID.
func fileVaultProfile(certDER []byte) []byte {
	return fmt.Appendf(nil, `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadType</key>
			<string>com.apple.MCX.FileVault2</string>
			<key>Enable</key>
			<string>On</string>
		</dict>
		<dict>
			<key>EncryptCertPayloadUUID</key>
			<string>A326B71F-EB80-41A5-A8CD-A6F932544281</string>
			<key>PayloadType</key>
			<string>com.apple.security.FDERecoveryKeyEscrow</string>
		</dict>
		<dict>
			<key>PayloadContent</key>
			<data>%s</data>
			<key>PayloadType</key>
			<string>com.apple.security.pkcs1</string>
			<key>PayloadUUID</key>
			<string>A326B71F-EB80-41A5-A8CD-A6F932544281</string>
		</dict>
	</array>
	<key>PayloadIdentifier</key>
	<string>com.fleetdm.fleet.mdm.filevault</string>
	<key>PayloadType</key>
	<string>Configuration</string>
</dict>
</plist>`, base64.StdEncoding.EncodeToString(certDER))
}

func testEscrowCert(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	key, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "FileVault escrow test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(cryptorand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert, key
}

func TestEscrowFileVaultKey(t *testing.T) {
	cert, key := testEscrowCert(t)

	a := &agent{id: hostSeed{seed: 1, index: 1}}
	a.escrowFileVaultKey(installProfileCommandWithPayload(fileVaultProfile(cert.Raw)))
	escrowed := a.fileVaultKey.Load()
	require.NotNil(t, escrowed)

	// Fleet must be able to decrypt it with its own cert and key.
	prk, err := servermdm.DecryptBase64CMS(*escrowed, cert, key)
	require.NoError(t, err)
	assert.Len(t, strings.Split(string(prk), "-"), 6)

	// The host reports the key and an encrypted disk.
	_, rows, status, _, _ := a.processQuery("fleet_detail_query_mdm_disk_encryption_key_file_darwin", "", nil)
	require.NotNil(t, status)
	require.Equal(t, []map[string]string{{"encrypted": "1", "filevault_key": *escrowed}}, rows)
	assert.Equal(t, []map[string]string{{"1": "1"}}, a.diskEncryption())

	// Same host after a restart: same recovery key.
	b := &agent{id: hostSeed{seed: 1, index: 1}}
	b.escrowFileVaultKey(installProfileCommandWithPayload(fileVaultProfile(cert.Raw)))
	prkAgain, err := servermdm.DecryptBase64CMS(*b.fileVaultKey.Load(), cert, key)
	require.NoError(t, err)
	assert.Equal(t, prk, prkAgain)
}

func TestEscrowFileVaultKeyIgnoresOtherProfiles(t *testing.T) {
	a := &agent{}
	a.escrowFileVaultKey(installProfileCommand(t, "com.example.test", "Test Profile"))
	assert.Nil(t, a.fileVaultKey.Load())

	_, rows, _, _, _ := a.processQuery("fleet_detail_query_mdm_disk_encryption_key_file_darwin", "", nil)
	assert.Equal(t, []map[string]string{{"encrypted": "0", "filevault_key": ""}}, rows)

	handled, rows, status, _, _ := a.processQuery("fleet_detail_query_mdm_disk_encryption_key_file_lines_darwin", "", nil)
	assert.True(t, handled)
	assert.Nil(t, rows)
	assert.Nil(t, status)
}
