package tables

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20250904091745, Down_20250904091745)
}

// LegacyIntegrationsWithCertAuthorities represents the legacy integrations configuration when it included certificate authorities.
type LegacyIntegrationsWithCertAuthorities struct {
	Jira           []*fleet.JiraIntegration           `json:"jira"`
	Zendesk        []*fleet.ZendeskIntegration        `json:"zendesk"`
	GoogleCalendar []*fleet.GoogleCalendarIntegration `json:"google_calendar"`
	DigiCert       optjson.Slice[fleet.DigiCertCA]    `json:"digicert"`
	// NDESSCEPProxy settings. In JSON, not specifying this field means keep current setting, null means clear settings.
	NDESSCEPProxy   optjson.Any[fleet.NDESSCEPProxyCA]     `json:"ndes_scep_proxy"`
	CustomSCEPProxy optjson.Slice[fleet.CustomSCEPProxyCA] `json:"custom_scep_proxy"`
	// ConditionalAccessEnabled indicates whether conditional access is enabled/disabled for "No team".
	ConditionalAccessEnabled optjson.Bool `json:"conditional_access_enabled"`
}

// dbCertificateAuthority embeds fleet.CertificateAuthority and adds encrypted representation of sensitive
// fields for handling DB operations
type dbCertificateAuthority struct {
	fleet.CertificateAuthority
	// Digicert
	APITokenEncrypted                []byte `db:"api_token_encrypted"`
	CertificateUserPrincipalNamesRaw []byte `db:"certificate_user_principal_names"`

	// NDES SCEP Proxy
	PasswordEncrypted []byte `db:"password_encrypted"`

	// Custom SCEP Proxy
	ChallengeEncrypted []byte `db:"challenge_encrypted"`

	// Hydrant
	ClientSecretEncrypted []byte `db:"client_secret_encrypted"`
}

func Up_20250904091745(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	stmt := `
	CREATE TABLE IF NOT EXISTS certificate_authorities (
  id INT AUTO_INCREMENT PRIMARY KEY,
  type ENUM('digicert', 'ndes_scep_proxy', 'custom_scep_proxy', 'hydrant') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  
  -- Common fields
  name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,           -- Used by digicert and custom_scep_proxy
  url TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,                    -- Used by all types

  -- DigiCert fields
  api_token_encrypted BLOB, -- previously stored in ca_config_assets
  profile_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  certificate_common_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  certificate_user_principal_names JSON,       -- Array of UPNs
  certificate_seat_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,

  -- NDES fields
  admin_url TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  username VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  password_encrypted BLOB, -- previously stored in mdm_config_assets

  -- Custom SCEP
  challenge_encrypted BLOB, -- previously stored in ca_config_assets

  -- Hydrant fields
  client_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  client_secret_encrypted BLOB,

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	`

	// Create the table then iterate through app_config_json to populate it
	_, err := txx.Exec(stmt)
	if err != nil {
		return fmt.Errorf("failed to create certificate_authorities table: %w", err)
	}

	// Create unique indexes on name and type(i.e. can't have more than 1 CA of a given type with a
	// given name)
	_, err = txx.Exec(`CREATE UNIQUE INDEX idx_ca_type_name ON certificate_authorities (type, name)`)
	if err != nil {
		return fmt.Errorf("failed to create unique index on certificate_authorities: %w", err)
	}

	// Populate the table with existing data from app_config_json
	appConfigSelect := `SELECT json_value->>"$.integrations" FROM app_config_json LIMIT 1`
	var integrations LegacyIntegrationsWithCertAuthorities
	jsonBytes := []byte{}
	if err := txx.Get(&jsonBytes, appConfigSelect); err != nil {
		return fmt.Errorf("failed to get app_config_json: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, &integrations); err != nil {
		return fmt.Errorf("failed to unmarshal app_config_json: %w", err)
	}

	configAssets := []fleet.CAConfigAsset{}
	assetSelectStmt := `
SELECT
	   name, type, value
FROM
	  ca_config_assets
		`
	if err := txx.Select(&configAssets, assetSelectStmt); err != nil {
		return fmt.Errorf("failed to get ca_config_assets: %w", err)
	}

	getCAConfigAsset := func(name string, assetType fleet.CAConfigAssetType) *fleet.CAConfigAsset {
		for _, asset := range configAssets {
			if asset.Name == name && asset.Type == assetType {
				return &asset
			}
		}
		return nil
	}

	casToInsert := []dbCertificateAuthority{}

	if integrations.CustomSCEPProxy.Valid && len(integrations.CustomSCEPProxy.Value) > 0 {
		for _, customSCEPProxyCA := range integrations.CustomSCEPProxy.Value {
			customSCEPChallenge := getCAConfigAsset(customSCEPProxyCA.Name, fleet.CAConfigCustomSCEPProxy)
			if customSCEPChallenge == nil || len(customSCEPChallenge.Value) == 0 {
				return errors.New("Custom SCEP Proxy challenge not found in ca_config_assets")
			}
			casToInsert = append(casToInsert, dbCertificateAuthority{
				CertificateAuthority: fleet.CertificateAuthority{
					Type: string(fleet.CATypeCustomSCEPProxy),
					Name: &customSCEPProxyCA.Name,
					URL:  &customSCEPProxyCA.URL,
				},
				ChallengeEncrypted: customSCEPChallenge.Value,
			})
		}
	}
	if integrations.DigiCert.Valid && len(integrations.DigiCert.Value) > 0 {
		for _, digicertCA := range integrations.DigiCert.Value {
			digicertAPIToken := getCAConfigAsset(digicertCA.Name, fleet.CAConfigDigiCert)
			if digicertAPIToken == nil || len(digicertAPIToken.Value) == 0 {
				return errors.New("DigiCert API token not found in ca_config_assets")
			}
			casToInsert = append(casToInsert, dbCertificateAuthority{
				CertificateAuthority: fleet.CertificateAuthority{
					Type:                          string(fleet.CATypeDigiCert),
					Name:                          &digicertCA.Name,
					URL:                           &digicertCA.URL,
					ProfileID:                     &digicertCA.ProfileID,
					CertificateCommonName:         &digicertCA.CertificateCommonName,
					CertificateUserPrincipalNames: &digicertCA.CertificateUserPrincipalNames,
					CertificateSeatID:             &digicertCA.CertificateSeatID,
				},
				APITokenEncrypted: digicertAPIToken.Value,
			})
		}
	}

	if integrations.NDESSCEPProxy.Valid {
		ndesCAPassword := []byte{}
		err = txx.Get(&ndesCAPassword, `SELECT value FROM mdm_config_assets WHERE name = ?`, fleet.MDMAssetNDESPassword)
		if err != nil {
			return fmt.Errorf("failed to get NDES SCEP Proxy password: %w", err)
		}
		if len(ndesCAPassword) == 0 {
			return errors.New("NDES SCEP Proxy password not found in mdm_config_assets")
		}

		// Insert NDES SCEP Proxy data
		ndesCA := integrations.NDESSCEPProxy.Value
		dbNDESCA := dbCertificateAuthority{
			CertificateAuthority: fleet.CertificateAuthority{
				Type:     string(fleet.CATypeNDESSCEPProxy),
				Name:     ptr.String("NDES"),
				URL:      &ndesCA.URL,
				AdminURL: &ndesCA.AdminURL,
				Username: &ndesCA.Username,
			},
			PasswordEncrypted: ndesCAPassword,
		}
		casToInsert = append(casToInsert, dbNDESCA)
	}

	for _, ca := range casToInsert {
		insertStmt := `
INSERT INTO certificate_authorities (
	type,
	name,
	url,
	api_token_encrypted,
	profile_id,
	certificate_common_name,
	certificate_user_principal_names,
	certificate_seat_id,
	admin_url,
	username,
	password_encrypted,
	challenge_encrypted,
	client_id,
	client_secret_encrypted
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		var upns []byte
		if ca.CertificateUserPrincipalNames != nil {
			upns, err = json.Marshal(ca.CertificateUserPrincipalNames)
			if err != nil {
				return fmt.Errorf("failed to marshal certificate user principal names for %s: %w", *ca.Name, err)
			}
		}
		args := []interface{}{
			ca.Type,
			ca.Name,
			ca.URL,
			ca.APITokenEncrypted,
			ca.ProfileID,
			ca.CertificateCommonName,
			upns,
			ca.CertificateSeatID,
			ca.AdminURL,
			ca.Username,
			ca.PasswordEncrypted,
			ca.ChallengeEncrypted,
			ca.ClientID,
			ca.ClientSecretEncrypted,
		}
		_, err = txx.Exec(insertStmt, args...)
		if err != nil {
			return fmt.Errorf("failed to insert certificate authority %s: %w", *ca.Name, err)
		}
	}

	// Remove existing CAs from appconfig. We are specifically deleting them by path to avoid any
	// potential issues roundtripping the JSON value itself
	removeStmt := `UPDATE app_config_json SET json_value=JSON_REMOVE(json_value, '$.integrations.custom_scep_proxy', '$.integrations.ndes_scep_proxy', '$.integrations.digicert')`
	_, err = txx.Exec(removeStmt)
	if err != nil {
		return fmt.Errorf("failed to remove certificate authorities from app_config: %w", err)
	}

	return nil
}

func Down_20250904091745(tx *sql.Tx) error {
	return nil
}
