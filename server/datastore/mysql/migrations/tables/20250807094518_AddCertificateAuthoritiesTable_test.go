package tables

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20250807094518(t *testing.T) {
	db := applyUpToPrev(t)
	var appConfigJSON fleet.AppConfig
	const (
		ndesPassword                = "ndes-password"
		digicertCA1Name             = "DigiCert_CA_1"
		digicertCA1Password         = "digicert-ca-1-password"
		digicertCA2Name             = "DigiCert_CA_2"
		digicertCA2Password         = "digicert-ca-2-password"
		customSCEPCA1Name           = "Custom_SCEP_Proxy_CA_1"
		customSCEPProxyCA1Challenge = "challenge1"
		customSCEPCA2Name           = "Custom_SCEP_Proxy_CA_2"
		customSCEPProxyCA2Challenge = "challenge2"
	)
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
	appConfigJSON.Integrations.NDESSCEPProxy.Value = ndesCA
	appConfigJSON.Integrations.DigiCert.Value = digicertCAs

	jsonBytes, err := json.Marshal(&appConfigJSON)
	if err != nil {
		t.Fatalf("failed to marshal appConfigJSON: %v", err)
	}

	insertNDESPasswordStmt := `INSERT INTO mdm_config_assets (name, value) VALUES (?, ?)`
	_, err = db.Exec(insertNDESPasswordStmt, fleet.MDMAssetNDESPassword, ndesPassword)
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
	stmt := `SELECT type, name, url, api_token, profile_id, certificate_common_name, certificate_user_principal_names, certificate_seat_id, admin_url, username, password
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
			expectedCA.APIToken = digicertCA1Password
			if ca.Name == customSCEPCA2Name {
				expectedCA = digicertCAs[1]
				expectedCA.APIToken = digicertCA2Password
			}
			require.Equal(t, expectedCA.URL, ca.URL, "DigiCert CA URL should match")
			require.Equal(t, expectedCA.APIToken, string(ca.APIToken), "DigiCert CA API token should match")
			require.Equal(t, expectedCA.ProfileID, ca.ProfileID, "DigiCert CA Profile ID should match")
			require.Equal(t, expectedCA.CertificateCommonName, *ca.CertificateCommonName, "DigiCert CA Certificate Common Name should match")
			require.Equal(t, expectedCA.CertificateUserPrincipalNames, ca.CertificateUserPrincipalNames, "DigiCert CA Certificate User Principal Names should match")
			require.Equal(t, expectedCA.CertificateSeatID, ca.CertificateSeatID, "DigiCert CA Certificate Seat ID should match")
		case "custom_scep_proxy":
			require.Contains(t, []string{customSCEPCA1Name, customSCEPCA2Name}, ca.Name, "unexpected Custom SCEP Proxy CA name")
			expectedCA := customSCEPProxyCAs[0]
			expectedCA.Challenge = customSCEPProxyCA1Challenge
			if ca.Name == customSCEPCA2Name {
				expectedCA = customSCEPProxyCAs[1]
				expectedCA.Challenge = customSCEPProxyCA2Challenge
			}
			require.Equal(t, expectedCA.URL, ca.URL, "Custom SCEP Proxy CA URL should match")
			require.Equal(t, expectedCA.Challenge, string(ca.Challenge), "Custom SCEP Proxy CA challenge should match")
			require.Nil(t, ca.CertificateUserPrincipalNames, "Custom SCEP Proxy CA should not have certificate user principal names")
		case "ndes_scep_proxy":
			require.Equal(t, "Default NDES SCEP Proxy", ca.Name)
			require.Equal(t, ndesCA.AdminURL, ca.AdminURL, "NDES SCEP Proxy CA admin URL should match")
			require.Equal(t, ndesCA.URL, ca.URL, "NDES SCEP Proxy CA URL should match")
			require.Equal(t, ndesCA.Username, ca.Username, "NDES SCEP Proxy CA Username should match")
			require.Equal(t, ndesPassword, ca.Password, "NDES SCEP Proxy CA password should match")
			require.Nil(t, ca.CertificateUserPrincipalNames, "NDES SCEP Proxy CA should not have certificate user principal names")
		default:
			t.Fatalf("unexpected certificate authority type: %s", ca.Type)
		}
	}
}
