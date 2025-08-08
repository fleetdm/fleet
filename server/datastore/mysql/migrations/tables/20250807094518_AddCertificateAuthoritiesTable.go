package tables

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20250807094518, Down_20250807094518)
}

// dbCertificateAuthority embeds fleet.CertificateAuthority and adds raw representation of encrypted
// fields for handling DB operations
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

func Up_20250807094518(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	stmt := `
	CREATE TABLE certificate_authorities (
  id INT AUTO_INCREMENT PRIMARY KEY,
  type ENUM('digicert', 'ndes_scep_proxy', 'custom_scep_proxy', 'hydrant') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  
  -- Common fields
  name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,           -- Used by digicert and custom_scep_proxy
  url TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,                    -- Used by all types

  -- DigiCert fields
  api_token BLOB, -- previously stored in ca_config_assets
  profile_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  certificate_common_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  certificate_user_principal_names JSON,       -- Array of UPNs
  certificate_seat_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,

  -- NDES fields
  admin_url TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  username VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  password BLOB, -- previously stored in mdm_config_assets

  -- Custom SCEP
  challenge BLOB, -- previously stored in ca_config_assets

  -- Hydrant fields
  client_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
  client_secret BLOB,

  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
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
	// if appConfigJSON.integrations.ndes_scep_proxy ...
	// if appConfigJSON.integrations.custom_scep_proxy
	// if appConfigJSON.integrations.hydrant ...
	// if appConfigJSON.integrations.digicert ...

	appConfigSelect := `SELECT json_value FROM app_config_json LIMIT 1`
	var appConfigJSON fleet.AppConfig
	jsonBytes := []byte{}
	if err := txx.Get(&jsonBytes, appConfigSelect); err != nil {
		return fmt.Errorf("failed to get app_config_json: %w", err)
	}
	if err := json.Unmarshal(jsonBytes, &appConfigJSON); err != nil {
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

	if appConfigJSON.Integrations.CustomSCEPProxy.Valid && len(appConfigJSON.Integrations.CustomSCEPProxy.Value) > 0 {
		for _, customSCEPProxyCA := range appConfigJSON.Integrations.CustomSCEPProxy.Value {
			customSCEPChallenge := getCAConfigAsset(customSCEPProxyCA.Name, fleet.CAConfigCustomSCEPProxy)
			if customSCEPChallenge == nil || len(customSCEPChallenge.Value) == 0 {
				return errors.New("Custom SCEP Proxy challenge not found in mdm_config_assets")
			}
			casToInsert = append(casToInsert, dbCertificateAuthority{
				CertificateAuthority: fleet.CertificateAuthority{
					Type: string(fleet.CATypeCustomSCEPProxy),
					Name: customSCEPProxyCA.Name,
					URL:  customSCEPProxyCA.URL,
				},
				ChallengeRaw: customSCEPChallenge.Value,
			})
		}
	}
	if appConfigJSON.Integrations.DigiCert.Valid && len(appConfigJSON.Integrations.DigiCert.Value) > 0 {
		for _, digicertCA := range appConfigJSON.Integrations.DigiCert.Value {
			digicertAPIToken := getCAConfigAsset(digicertCA.Name, fleet.CAConfigDigiCert)
			if digicertAPIToken == nil || len(digicertAPIToken.Value) == 0 {
				return errors.New("DigiCert API token not found in ca_config_assets")
			}
			casToInsert = append(casToInsert, dbCertificateAuthority{
				CertificateAuthority: fleet.CertificateAuthority{
					Type:                          string(fleet.CATypeDigiCert),
					Name:                          digicertCA.Name,
					URL:                           digicertCA.URL,
					ProfileID:                     &digicertCA.ProfileID,
					CertificateCommonName:         &digicertCA.CertificateCommonName,
					CertificateUserPrincipalNames: digicertCA.CertificateUserPrincipalNames,
					CertificateSeatID:             &digicertCA.CertificateSeatID,
				},
				APITokenRaw: digicertAPIToken.Value,
			})
		}
	}

	if appConfigJSON.Integrations.NDESSCEPProxy.Valid {
		ndesCAPassword := []byte{}
		err = txx.Get(&ndesCAPassword, `SELECT value FROM mdm_config_assets WHERE name = ?`, fleet.MDMAssetNDESPassword)
		if err != nil {
			return fmt.Errorf("failed to get NDES SCEP Proxy password: %w", err)
		}
		if len(ndesCAPassword) == 0 {
			return errors.New("NDES SCEP Proxy password not found in mdm_config_assets")
		}

		// Insert NDES SCEP Proxy data
		ndesCA := appConfigJSON.Integrations.NDESSCEPProxy.Value
		dbNDESCA := dbCertificateAuthority{
			CertificateAuthority: fleet.CertificateAuthority{
				Type:     string(fleet.CATypeNDESSCEPProxy),
				Name:     "DEFAULT_NDES_CA",
				URL:      ndesCA.URL,
				AdminURL: &ndesCA.AdminURL,
				Username: &ndesCA.Username,
			},
			PasswordRaw: ndesCAPassword,
		}
		casToInsert = append(casToInsert, dbNDESCA)
	}

	for _, ca := range casToInsert {
		insertStmt := `
INSERT INTO certificate_authorities (
	type,
	name,
	url,
	api_token,
	profile_id,
	certificate_common_name,
	certificate_user_principal_names,
	certificate_seat_id,
	admin_url,
	username,
	password,
	challenge,
	client_id,
	client_secret
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		var upns []byte
		if ca.CertificateUserPrincipalNames != nil {
			upns, err = json.Marshal(ca.CertificateUserPrincipalNames)
			if err != nil {
				return fmt.Errorf("failed to marshal certificate user principal names for %s: %w", ca.Name, err)
			}
		}
		args := []interface{}{
			ca.Type,
			ca.Name,
			ca.URL,
			ca.APITokenRaw,
			ca.ProfileID,
			ca.CertificateCommonName,
			upns,
			ca.CertificateSeatID,
			ca.AdminURL,
			ca.Username,
			ca.PasswordRaw,
			ca.ChallengeRaw,
			ca.ClientID,
			ca.ClientSecret,
		}
		_, err = txx.Exec(insertStmt, args...)
		if err != nil {
			return fmt.Errorf("failed to insert certificate authority %s: %w", ca.Name, err)
		}
	}

	return nil
}

func Down_20250807094518(tx *sql.Tx) error {
	return nil
}
