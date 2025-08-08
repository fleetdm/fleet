package tables

import (
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func md5ChecksumBytes(b []byte) string {
	rawChecksum := md5.Sum(b) //nolint:gosec
	return strings.ToUpper(hex.EncodeToString(rawChecksum[:]))
}

func TestUp_20250807094518(t *testing.T) {
	db := applyUpToPrev(t)
	var appConfigJSON fleet.AppConfig
	const (
		digicertCA1Name   = "DigiCert_CA_1"
		digicertCA2Name   = "DigiCert_CA_2"
		customSCEPCA1Name = "Custom_SCEP_Proxy_CA_1"
		customSCEPCA2Name = "Custom_SCEP_Proxy_CA_2"
	)
	// various strings encrypted with key my-secret-key-123456781234567890 just to verify the
	// migration works with opaque(to it) binary data
	customSCEPCA1EncryptedChallenge := []byte{0xe0, 0xd1, 0xba, 0x48, 0x20, 0x31, 0xff, 0x52, 0x11, 0x2c, 0x62, 0x0d, 0x8e, 0xc3, 0xb7, 0x88, 0x13, 0x5d, 0x37, 0x04, 0x10, 0xf4, 0xda, 0xa4, 0x8e, 0x18, 0xf7, 0x95, 0x8f, 0x5d, 0x1c, 0xc4, 0xb7, 0x45, 0xeb, 0xa9, 0xbf, 0x7d}
	customSCEPCA2EncryptedChallenge := []byte{0x8b, 0x92, 0x95, 0xbf, 0xda, 0xc1, 0x71, 0x11, 0x64, 0xdc, 0xd0, 0xa8, 0x1c, 0x91, 0x26, 0xec, 0x15, 0xd5, 0x21, 0xca, 0x5e, 0x62, 0x71, 0x01, 0x5e, 0xb1, 0xb1, 0xbf, 0x9f, 0xe6, 0x36, 0x79, 0xa2, 0xee, 0x32, 0x9d, 0x64, 0x7c}
	ndesEncryptedPassword := []byte{0xaa, 0x20, 0x6e, 0xb0, 0xb5, 0x10, 0x52, 0x6d, 0xe2, 0x78, 0x14, 0xbe, 0xc5, 0xe0, 0x8a, 0x04, 0xa6, 0xfd, 0x8b, 0x17, 0xd8, 0x15, 0x71, 0x3c, 0x72, 0xa2, 0x76, 0x5f, 0xba, 0x5d, 0x10, 0x41, 0x21, 0x56, 0xe4, 0x1d, 0xa0, 0x90, 0x0d, 0x9e, 0xe1}
	digicertCA1EncryptedPassword := []byte{0xd5, 0xf1, 0x50, 0x6f, 0x59, 0xb4, 0xfe, 0xa4, 0x3a, 0xc4, 0x24, 0xc8, 0xfa, 0xfd, 0x43, 0xc0, 0xec, 0x2d, 0x10, 0xb1, 0x2a, 0x1e, 0xa8, 0x1e, 0x62, 0x2f, 0x04, 0xeb, 0xb5, 0x55, 0xea, 0x92, 0xfe, 0xb2, 0x9b, 0x6b, 0xc0, 0x98, 0x70, 0x2c, 0x33, 0xf6, 0x01, 0x0f, 0x13, 0x06, 0xef, 0xee, 0x81, 0xb9}
	digicertCA2EncryptedPassword := []byte{0x24, 0x46, 0x38, 0xa5, 0x75, 0xe4, 0x34, 0x2a, 0x99, 0x5d, 0x52, 0xc9, 0xb1, 0x05, 0x05, 0xa1, 0xdf, 0x62, 0xe2, 0xf1, 0x01, 0x92, 0x0b, 0xcd, 0xd4, 0x49, 0x83, 0x2e, 0xff, 0xd6, 0x23, 0x5c, 0x75, 0x57, 0x57, 0x18, 0x42, 0x3c, 0x81, 0x78, 0xf2, 0x86, 0x59, 0x42, 0x11, 0xb5, 0x82, 0x23, 0x3a, 0x91}
	ndesCA := fleet.NDESSCEPProxyIntegration{
		URL:      "https://ndes.example.com",
		AdminURL: "https://admin.ndes.example.com",
		Username: "admin",
		Password: fleet.MaskedPassword,
	}

	digicertCAs := []fleet.DigiCertIntegration{{
		URL:                           "https://api.digicert.com",
		ProfileID:                     "profile-id-1",
		Name:                          digicertCA1Name,
		APIToken:                      fleet.MaskedPassword,
		CertificateCommonName:         "Common-Name1: $FLEET_VAR_HOST_HARDWARE_SERIAL",
		CertificateUserPrincipalNames: []string{"UPN1: $FLEET_VAR_HOST_HARDWARE_SERIAL"},
		CertificateSeatID:             "Seat-ID1: $FLEET_VAR_HOST_HARDWARE_SERIAL",
	}, {
		URL:                           "https://api.digicert.com",
		ProfileID:                     "profile-id-2",
		Name:                          digicertCA2Name,
		APIToken:                      fleet.MaskedPassword,
		CertificateCommonName:         "Common-Name2: $FLEET_VAR_HOST_HARDWARE_SERIAL",
		CertificateUserPrincipalNames: []string{"UPN2: $FLEET_VAR_HOST_HARDWARE_SERIAL"},
		CertificateSeatID:             "Seat-ID2: $FLEET_VAR_HOST_HARDWARE_SERIAL",
	}}
	customSCEPProxyCAs := []fleet.CustomSCEPProxyIntegration{{
		URL:       "https://custom-scep-1.example.com",
		Name:      customSCEPCA1Name,
		Challenge: fleet.MaskedPassword,
	}, {
		URL:       "https://custom-scep-2.example.com",
		Name:      customSCEPCA2Name,
		Challenge: fleet.MaskedPassword,
	}}
	appConfigJSON.Integrations.CustomSCEPProxy.Value = customSCEPProxyCAs
	appConfigJSON.Integrations.CustomSCEPProxy.Set = true
	appConfigJSON.Integrations.CustomSCEPProxy.Valid = true
	appConfigJSON.Integrations.NDESSCEPProxy.Value = ndesCA
	appConfigJSON.Integrations.NDESSCEPProxy.Set = true
	appConfigJSON.Integrations.NDESSCEPProxy.Valid = true
	appConfigJSON.Integrations.DigiCert.Value = digicertCAs
	appConfigJSON.Integrations.DigiCert.Set = true
	appConfigJSON.Integrations.DigiCert.Valid = true

	jsonBytes, err := json.Marshal(&appConfigJSON)
	if err != nil {
		t.Fatalf("failed to marshal appConfigJSON: %v", err)
	}

	insertNDESPasswordStmt := `INSERT INTO mdm_config_assets (name, value, md5_checksum) VALUES (?, ?, UNHEX(?))`
	_, err = db.Exec(insertNDESPasswordStmt, fleet.MDMAssetNDESPassword, ndesEncryptedPassword, md5ChecksumBytes(ndesEncryptedPassword))
	require.NoError(t, err, "failed to insert NDES SCEP Proxy password")

	insertCAAssetsStmt := `INSERT INTO ca_config_assets (name, value, type) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?), (?, ?, ?)`
	_, err = db.Exec(insertCAAssetsStmt,
		digicertCA1Name, digicertCA1EncryptedPassword, fleet.CAConfigDigiCert,
		digicertCA2Name, digicertCA2EncryptedPassword, fleet.CAConfigDigiCert,
		customSCEPCA1Name, customSCEPCA1EncryptedChallenge, fleet.CAConfigCustomSCEPProxy,
		customSCEPCA2Name, customSCEPCA2EncryptedChallenge, fleet.CAConfigCustomSCEPProxy,
	)
	require.NoError(t, err, "failed to insert ca_config_assets")

	_, err = db.Exec(
		`INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`,
		jsonBytes,
	)
	if err != nil {
		require.NoError(t, err, "failed to insert app_config_json")
	}
	// Apply current migration.
	applyNext(t, db)

	type dbCertificateAuthority struct {
		fleet.CertificateAuthority
		// Digicert
		APITokenRaw                      []byte `db:"api_token"`
		CertificateUserPrincipalNamesRaw []byte `db:"certificate_user_principal_names"`

		// NDES SCEP Proxy
		PasswordRaw []byte `db:"password"`

		// Custom SCEP Proxy
		ChallengeRaw []byte `db:"challenge"`

		// Hydrant
		ClientSecretRaw []byte `db:"client_secret"`
	}

	cas := []dbCertificateAuthority{}
	stmt := `SELECT type, name, url, api_token, profile_id, certificate_common_name, certificate_user_principal_names, certificate_seat_id, admin_url, username, password, challenge, client_id, client_secret, created_at, updated_at
FROM certificate_authorities`
	err = db.Select(&cas, stmt)

	casFound := []string{}
	require.NoError(t, err, "failed to select certificate authorities")
	require.Len(t, cas, 5, "expected 5 certificate authorities")

	for _, ca := range cas {
		if ca.CertificateUserPrincipalNamesRaw != nil {
			err = json.Unmarshal(ca.CertificateUserPrincipalNamesRaw, &ca.CertificateUserPrincipalNames)
			require.NoErrorf(t, err, "failed to unmarshal certificate user principal names for %s", ca.Name)
		}
		casFound = append(casFound, ca.Name)

		// No Hydrant CAs in this test so these should be nil
		assert.Nil(t, ca.ClientID)
		assert.Nil(t, ca.ClientSecret)
		switch ca.Type {
		case "digicert":
			assert.Contains(t, []string{digicertCA1Name, digicertCA2Name}, ca.Name, "unexpected DigiCert CA name")
			expectedCA := digicertCAs[0]
			expectedAPIToken := digicertCA1EncryptedPassword
			if ca.Name == digicertCA2Name {
				expectedCA = digicertCAs[1]
				expectedAPIToken = digicertCA2EncryptedPassword
			}
			assert.Equal(t, expectedCA.URL, ca.URL)
			assert.Equal(t, expectedAPIToken, ca.APITokenRaw)
			require.NotNil(t, ca.ProfileID)
			assert.Equal(t, expectedCA.ProfileID, *ca.ProfileID)
			require.NotNil(t, ca.CertificateCommonName)
			assert.Equal(t, expectedCA.CertificateCommonName, *ca.CertificateCommonName)
			assert.Equal(t, expectedCA.CertificateUserPrincipalNames, ca.CertificateUserPrincipalNames)
			require.NotNil(t, ca.CertificateSeatID)
			assert.Equal(t, expectedCA.CertificateSeatID, *ca.CertificateSeatID)
		case "custom_scep_proxy":
			require.Contains(t, []string{customSCEPCA1Name, customSCEPCA2Name}, ca.Name, "unexpected Custom SCEP Proxy CA name")
			expectedCA := customSCEPProxyCAs[0]
			expectedChallenge := customSCEPCA1EncryptedChallenge
			if ca.Name == customSCEPCA2Name {
				expectedCA = customSCEPProxyCAs[1]
				expectedChallenge = customSCEPCA2EncryptedChallenge
			}
			assert.Equal(t, expectedCA.URL, ca.URL)
			assert.Equal(t, expectedChallenge, ca.ChallengeRaw)
			assert.Nil(t, ca.CertificateUserPrincipalNames)
		case "ndes_scep_proxy":
			assert.Equal(t, "Default NDES SCEP Proxy", ca.Name)
			require.NotNil(t, ca.AdminURL)
			assert.Equal(t, ndesCA.AdminURL, *ca.AdminURL)
			assert.Equal(t, ndesCA.URL, ca.URL)
			require.NotNil(t, ca.Username)
			assert.Equal(t, ndesCA.Username, *ca.Username)
			assert.Equal(t, ndesEncryptedPassword, ca.PasswordRaw)
			assert.Nil(t, ca.CertificateUserPrincipalNames)
		default:
			require.Failf(t, "unexpected certificate authority type", "type: %s, name: %s", ca.Type, ca.Name)
		}
	}
	require.ElementsMatch(t, []string{digicertCA1Name, digicertCA2Name, customSCEPCA1Name, customSCEPCA2Name, "Default NDES SCEP Proxy"}, casFound)
}
