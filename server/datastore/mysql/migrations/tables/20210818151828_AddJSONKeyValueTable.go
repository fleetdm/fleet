package tables

import (
	"database/sql"
	"encoding/json"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210818151828, Down_20210818151828)
}

type testAppConfig struct {
	OrgInfo struct {
		OrgName    string `json:"org_name"`
		OrgLogoURL string `json:"org_logo_url"`
	} `json:"org_info"`
	SMTPSettings struct {
		SMTPEnabled              bool   `json:"enable_smtp"`
		SMTPConfigured           bool   `json:"configured"`
		SMTPSenderAddress        string `json:"sender_address"`
		SMTPServer               string `json:"server"`
		SMTPPort                 uint   `json:"port"`
		SMTPAuthenticationType   string `json:"authentication_type"`
		SMTPUserName             string `json:"user_name"`
		SMTPPassword             string `json:"password"`
		SMTPEnableTLS            bool   `json:"enable_ssl_tls"`
		SMTPAuthenticationMethod string `json:"authentication_method"`
		SMTPDomain               string `json:"domain"`
		SMTPVerifySSLCerts       bool   `json:"verify_ssl_certs"`
		SMTPEnableStartTLS       bool   `json:"enable_start_tls"`
	} `json:"smtp_settings"`
	SSOSettings struct {
		EntityID              string `json:"entity_id"`
		IssuerURI             string `json:"issuer_uri"`
		IDPImageURL           string `json:"idp_image_url"`
		Metadata              string `json:"metadata"`
		MetadataURL           string `json:"metadata_url"`
		IDPName               string `json:"idp_name"`
		EnableSSO             bool   `json:"enable_sso"`
		EnableSSOIdPLogin     bool   `json:"enable_sso_idp_login"`
		EnableJITProvisioning bool   `json:"enable_jit_provisioning"`
	} `json:"sso_settings"`
	HostExpirySettings struct {
		HostExpiryEnabled bool `json:"host_expiry_enabled"`
		HostExpiryWindow  int  `json:"host_expiry_window"`
	} `json:"host_expiry_settings"`
	ServerSettings struct {
		ServerURL         string `json:"server_url"`
		LiveQueryDisabled bool   `json:"live_query_disabled"`
		EnableAnalytics   bool   `json:"enable_analytics"`
	} `json:"server_settings"`
	HostSettings struct {
		EnableHostUsers         bool             `json:"enable_host_users"`
		EnableSoftwareInventory bool             `json:"enable_software_inventory"`
		AdditionalQueries       *json.RawMessage `json:"additional_queries,omitempty"`
	} `json:"host_settings"`
	VulnerabilitySettings struct {
		DatabasesPath string `json:"databases_path"`
	} `json:"vulnerability_settings"`
	AgentOptions *json.RawMessage `json:"agent_options"`
}

func Up_20210818151828(tx *sql.Tx) error {
	sql := `
		CREATE TABLE IF NOT EXISTS app_config_json (
			id int(10) unsigned NOT NULL UNIQUE default 1,
			json_value JSON NOT NULL,
			created_at timestamp DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id)
		)
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "create app_config_json")
	}
	row := tx.QueryRow(
		`SELECT
       			org_name,
				org_logo_url,
				server_url,
				smtp_configured,
				smtp_sender_address,
				smtp_server,
				smtp_port,
				smtp_authentication_type,
				smtp_enable_ssl_tls,
				smtp_authentication_method,
				smtp_domain,
				smtp_user_name,
				smtp_password,
				smtp_verify_ssl_certs,
				smtp_enable_start_tls,
				entity_id,
				issuer_uri,
				idp_image_url,
				metadata,
				metadata_url,
				idp_name,
				enable_sso,
				host_expiry_enabled,
				host_expiry_window,
				live_query_disabled,
				additional_queries,
				enable_sso_idp_login,
				agent_options,
				enable_analytics,
				vulnerability_databases_path,
				enable_host_users,
				enable_software_inventory
				FROM app_configs LIMIT 1`,
	)
	config := &testAppConfig{}
	// Apply default settings
	config.HostSettings.EnableHostUsers = true
	var vulnPath *string
	err := row.Scan(
		&config.OrgInfo.OrgName,
		&config.OrgInfo.OrgLogoURL,
		&config.ServerSettings.ServerURL,
		&config.SMTPSettings.SMTPConfigured,
		&config.SMTPSettings.SMTPSenderAddress,
		&config.SMTPSettings.SMTPServer,
		&config.SMTPSettings.SMTPPort,
		&config.SMTPSettings.SMTPAuthenticationType,
		&config.SMTPSettings.SMTPEnableTLS,
		&config.SMTPSettings.SMTPAuthenticationMethod,
		&config.SMTPSettings.SMTPDomain,
		&config.SMTPSettings.SMTPUserName,
		&config.SMTPSettings.SMTPPassword,
		&config.SMTPSettings.SMTPVerifySSLCerts,
		&config.SMTPSettings.SMTPEnableStartTLS,
		&config.SSOSettings.EntityID,
		&config.SSOSettings.IssuerURI,
		&config.SSOSettings.IDPImageURL,
		&config.SSOSettings.Metadata,
		&config.SSOSettings.MetadataURL,
		&config.SSOSettings.IDPName,
		&config.SSOSettings.EnableSSO,
		&config.HostExpirySettings.HostExpiryEnabled,
		&config.HostExpirySettings.HostExpiryWindow,
		&config.ServerSettings.LiveQueryDisabled,
		&config.HostSettings.AdditionalQueries,
		&config.SSOSettings.EnableSSOIdPLogin,
		&config.AgentOptions,
		&config.ServerSettings.EnableAnalytics,
		&vulnPath,
		&config.HostSettings.EnableHostUsers,
		&config.HostSettings.EnableSoftwareInventory,
	)
	if err != nil {
		return errors.Wrap(err, "scanning config row")
	}
	if vulnPath != nil {
		config.VulnerabilitySettings.DatabasesPath = *vulnPath
	}

	configBytes, err := json.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "marshaling config")
	}
	//nolint
	_, err = tx.Exec(
		`INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`,
		configBytes,
	)
	return nil
}

func Down_20210818151828(tx *sql.Tx) error {
	return nil
}
