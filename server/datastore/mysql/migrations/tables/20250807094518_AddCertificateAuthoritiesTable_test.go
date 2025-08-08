package tables

import (
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
		digicertCA1Name             = "DigiCert_CA_1"
		digicertCA2Name             = "DigiCert_CA_2"
		customSCEPCA1Name           = "Custom_SCEP_Proxy_CA_1"
		customSCEPProxyCA1Challenge = "challenge1"
		customSCEPCA2Name           = "Custom_SCEP_Proxy_CA_2"
		customSCEPProxyCA2Challenge = "challenge2"
	)
	ndesPassword := []byte("ndes-password")
	digicertCA1Password := []byte("digicert-ca-1-password")
	digicertCA2Password := []byte("digicert-ca-2-password")
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
	_, err = db.Exec(insertNDESPasswordStmt, fleet.MDMAssetNDESPassword, ndesPassword, md5ChecksumBytes([]byte(ndesPassword)))
	require.NoError(t, err, "failed to insert NDES SCEP Proxy password")

	insertCAAssetsStmt := `INSERT INTO ca_config_assets (name, value, type) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?), (?, ?, ?)`
	_, err = db.Exec(insertCAAssetsStmt,
		digicertCA1Name, digicertCA1Password, fleet.CAConfigDigiCert,
		digicertCA2Name, digicertCA2Password, fleet.CAConfigDigiCert,
		customSCEPCA1Name, customSCEPProxyCA1Challenge, fleet.CAConfigCustomSCEPProxy,
		customSCEPCA2Name, customSCEPProxyCA2Challenge, fleet.CAConfigCustomSCEPProxy,
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

	type fleetCertificateAuthority struct {
		ID   int64  `db:"id"`
		Type string `db:"type"` // TODO

		// common
		Name string `db:"name"`
		URL  string `db:"url"`

		// Digicert
		APIToken                         []byte   `db:"api_token"`
		ProfileID                        *string  `db:"profile_id"`
		CertificateCommonName            *string  `db:"certificate_common_name"`
		CertificateUserPrincipalNames    []string `db:"-"`                                // TODO
		CertificateUserPrincipalNamesRaw []byte   `db:"certificate_user_principal_names"` // JSON array
		CertificateSeatID                *string  `db:"certificate_seat_id"`

		// NDES SCEP Proxy
		AdminURL *string `db:"admin_url"`
		Username *string `db:"username"`
		Password []byte  `db:"password"`

		// Custom SCEP Proxy
		Challenge []byte `db:"challenge"`

		// Hydrant
		ClientID     *string `db:"client_id"`
		ClientSecret []byte  `db:"client_secret"`

		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	cas := []fleetCertificateAuthority{}
	stmt := `SELECT type, name, url, api_token, profile_id, certificate_common_name, certificate_user_principal_names, certificate_seat_id, admin_url, username, password, challenge, client_id, client_secret, created_at, updated_at
FROM certificate_authorities`
	err = db.Select(&cas, stmt)

	casFound := []string{}
	require.NoError(t, err, "failed to select certificate authorities")
	require.Len(t, cas, 5, "expected 5 certificate authorities")

	for _, ca := range cas {
		if ca.CertificateUserPrincipalNamesRaw != nil {
			err = json.Unmarshal(ca.CertificateUserPrincipalNamesRaw, &ca.CertificateUserPrincipalNames)
			require.NoError(t, err, "failed to unmarshal certificate user principal names for %s", ca.Name)
		}
		casFound = append(casFound, ca.Name)
		switch ca.Type {
		case "digicert":
			require.Contains(t, []string{digicertCA1Name, digicertCA2Name}, ca.Name, "unexpected DigiCert CA name")
			expectedCA := digicertCAs[0]
			expectedCA.APIToken = string(digicertCA1Password)
			if ca.Name == digicertCA2Name {
				expectedCA = digicertCAs[1]
				expectedCA.APIToken = string(digicertCA2Password)
			}
			require.Equal(t, expectedCA.URL, ca.URL)
			require.Equal(t, expectedCA.APIToken, string(ca.APIToken))
			require.NotNil(t, ca.ProfileID)
			require.Equal(t, expectedCA.ProfileID, *ca.ProfileID)
			require.NotNil(t, ca.CertificateCommonName)
			require.Equal(t, expectedCA.CertificateCommonName, *ca.CertificateCommonName)
			require.Equal(t, expectedCA.CertificateUserPrincipalNames, ca.CertificateUserPrincipalNames)
			require.NotNil(t, ca.CertificateSeatID)
			require.Equal(t, expectedCA.CertificateSeatID, *ca.CertificateSeatID)
		case "custom_scep_proxy":
			require.Contains(t, []string{customSCEPCA1Name, customSCEPCA2Name}, ca.Name, "unexpected Custom SCEP Proxy CA name")
			expectedCA := customSCEPProxyCAs[0]
			expectedCA.Challenge = customSCEPProxyCA1Challenge
			if ca.Name == customSCEPCA2Name {
				expectedCA = customSCEPProxyCAs[1]
				expectedCA.Challenge = customSCEPProxyCA2Challenge
			}
			require.Equal(t, expectedCA.URL, ca.URL)
			require.Equal(t, expectedCA.Challenge, string(ca.Challenge))
			require.Nil(t, ca.CertificateUserPrincipalNames)
		case "ndes_scep_proxy":
			require.Equal(t, "Default NDES SCEP Proxy", ca.Name)
			require.NotNil(t, ca.AdminURL)
			require.Equal(t, ndesCA.AdminURL, *ca.AdminURL)
			require.Equal(t, ndesCA.URL, ca.URL)
			require.NotNil(t, ca.Username)
			require.Equal(t, ndesCA.Username, *ca.Username)
			require.NotNil(t, ca.Password)
			require.Equal(t, ndesPassword, ca.Password)
			require.Nil(t, ca.CertificateUserPrincipalNames)
		default:
			t.Fatalf("unexpected certificate authority type: %s", ca.Type)
		}
	}
}
