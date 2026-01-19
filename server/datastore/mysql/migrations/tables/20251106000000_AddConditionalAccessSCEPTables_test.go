package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20251106000000(t *testing.T) {
	db := applyUpToPrev(t)

	// Create a host to reference (conditional_access_scep_certificates requires a host_id)
	hostID := insertHost(t, db, nil)

	// Apply current migration
	applyNext(t, db)

	// Valid certificate PEM that starts with "-----BEGIN CERTIFICATE-----"
	validCertPEM := `-----BEGIN CERTIFICATE-----
MIIDjzCCAnegAwIBAgIBATANBgkqhkiG9w0BAQsFADBpMQkwBwYDVQQGEwAxJDAi
BgNVBAoTG0xvY2FsIGNlcnRpZmljYXRlIGF1dGhvcml0eTEQMA4GA1UECxMHU0NF
UCBDQTEkMCIGA1UEAxMbRmxlZXQgY29uZGl0aW9uYWwgYWNjZXNzIENBMB4XDTI1
MTEwNjE2MjEyNloXDTM1MTEwNjE2MjEyNlowaTEJMAcGA1UEBhMAMSQwIgYDVQQK
ExtMb2NhbCBjZXJ0aWZpY2F0ZSBhdXRob3JpdHkxEDAOBgNVBAsTB1NDRVAgQ0Ex
JDAiBgNVBAMTG0ZsZWV0IGNvbmRpdGlvbmFsIGFjY2VzcyBDQTCCASIwDQYJKoZI
hvcNAQEBBQADggEPADCCAQoCggEBALZIk1qcmD1r9Plj2SC+FZgfXNUIIGJmnLXD
oGflLLkBpTjfm48NH0gOQwbRLfudi/Kdo2kx2d7cvV2Seu1Dgx4+Suh87Zj277Xp
280qSFTxbo+2W+rpTRoACf774+cw/fribH/j+k58hBPFHCIvx/iUBWXqjLxvx+b+
borRH6jWKevVCeh2x6KsRO1UM5ll3pJa3StAMPSdtldgI8iTt18vfc8+53AslTw+
7ri9SbE26zxh0XhUUuR2uzfSiptbKmwNc7CsrS3juCmi8CAayQHQ8NjIyXv5d3zT
uoR0Agk4Wes29Z0WRCJ9gskxaB6pM6idyccp39bB0qqjhIEo6MECAwEAAaNCMEAw
DgYDVR0PAQH/BAQDAgGGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFAErVNWV
jX16em5Jw9IFi02Q9y5GMA0GCSqGSIb3DQEBCwUAA4IBAQBNRbVDL+0p8V29hvRx
+Ea0E87DRONM0ym4DEH2fQV23FyQzXyxlYbLdN32ssHsQNU+eHrjjWfjxcy6b3H/
64fNLFvS4ThfJymJB8gvj+b180MmX+YUOhUsLLPTOA4gCdZagDS80ngmcjoh2E4J
sO1WlnLrMmXCwtU+VZXxfVU2oXkSoy+wpzuixNbxi6WsH6PObRZ2FKcZSqQyRp01
fU7N5JakKVGW43vKWYK4oB9EFc2pO/yuZYz/BXaMtW3AUpCJd+YZjWEkfqzKj11+
kLWyc3155w2EmkO2J21v/53o5gZWgjeyPY4edtOaoWWz2eHkn3k2QQZ76V1nzfWb
CT1g
-----END CERTIFICATE-----`

	// Insert a serial number first (required by foreign key)
	serialID := execNoErrLastID(t, db, `INSERT INTO conditional_access_scep_serials (created_at) VALUES (?)`, time.Now())

	// Test 1: Valid certificate PEM should be accepted
	execNoErr(t, db, `
		INSERT INTO conditional_access_scep_certificates
		(serial, host_id, name, not_valid_before, not_valid_after, certificate_pem, revoked)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, serialID, hostID, "Test Device", time.Now(), time.Now().Add(365*24*time.Hour), validCertPEM, false)

	// Verify the certificate was inserted
	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM conditional_access_scep_certificates WHERE serial = ?`, serialID)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Test 2: Invalid certificate PEM (not starting with "-----BEGIN CERTIFICATE-----") should fail
	invalidCertPEM := `INVALID CERTIFICATE DATA
This does not start with the required prefix`

	// Insert another serial number
	invalidSerialID := execNoErrLastID(t, db, `INSERT INTO conditional_access_scep_serials (created_at) VALUES (?)`, time.Now())

	// This should fail due to CHECK constraint
	_, err = db.Exec(`
		INSERT INTO conditional_access_scep_certificates
		(serial, host_id, name, not_valid_before, not_valid_after, certificate_pem, revoked)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, invalidSerialID, hostID, "Invalid Device", time.Now(), time.Now().Add(365*24*time.Hour), invalidCertPEM, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "Check constraint 'conditional_access_scep_certificates_chk_1' is violated")

	// Test 3: Certificate PEM starting with wrong prefix should also fail
	// nolint:gosec,G101
	wrongPrefixCertPEM := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAtN...
-----END RSA PRIVATE KEY-----`

	// Insert another serial number
	wrongPrefixSerialID := execNoErrLastID(t, db, `INSERT INTO conditional_access_scep_serials (created_at) VALUES (?)`, time.Now())

	_, err = db.Exec(`
		INSERT INTO conditional_access_scep_certificates
		(serial, host_id, name, not_valid_before, not_valid_after, certificate_pem, revoked)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, wrongPrefixSerialID, hostID, "Wrong Prefix Device", time.Now(), time.Now().Add(365*24*time.Hour), wrongPrefixCertPEM, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "Check constraint 'conditional_access_scep_certificates_chk_1' is violated")
}
