package tables

import (
	"database/sql"
	"encoding/json"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210816141251, Down_20210816141251)
}

type __AppConfig struct {
	ID                         uint
	OrgName                    string           `db:"org_name"`
	OrgLogoURL                 string           `db:"org_logo_url"`
	ServerURL                  string           `db:"server_url"`
	SMTPConfigured             bool             `db:"smtp_configured"`
	SMTPSenderAddress          string           `db:"smtp_sender_address"`
	SMTPServer                 string           `db:"smtp_server"`
	SMTPPort                   uint             `db:"smtp_port"`
	SMTPAuthenticationType     int              `db:"smtp_authentication_type"`
	SMTPUserName               string           `db:"smtp_user_name"`
	SMTPPassword               string           `db:"smtp_password"`
	SMTPEnableTLS              bool             `db:"smtp_enable_ssl_tls"`
	SMTPAuthenticationMethod   int              `db:"smtp_authentication_method"`
	SMTPDomain                 string           `db:"smtp_domain"`
	SMTPVerifySSLCerts         bool             `db:"smtp_verify_ssl_certs"`
	SMTPEnableStartTLS         bool             `db:"smtp_enable_start_tls"`
	EntityID                   string           `db:"entity_id"`
	IssuerURI                  string           `db:"issuer_uri"`
	IDPImageURL                string           `db:"idp_image_url"`
	Metadata                   string           `db:"metadata"`
	MetadataURL                string           `db:"metadata_url"`
	IDPName                    string           `db:"idp_name"`
	EnableSSO                  bool             `db:"enable_sso"`
	EnableSSOIdPLogin          bool             `db:"enable_sso_idp_login"`
	FIMInterval                int              `db:"fim_interval"`
	FIMFileAccesses            string           `db:"fim_file_accesses"`
	HostExpiryEnabled          bool             `db:"host_expiry_enabled"`
	HostExpiryWindow           int              `db:"host_expiry_window"`
	LiveQueryDisabled          bool             `db:"live_query_disabled"`
	EnableAnalytics            bool             `db:"enable_analytics" json:"analytics"`
	AdditionalQueries          *json.RawMessage `db:"additional_queries"`
	AgentOptions               *json.RawMessage `db:"agent_options"`
	VulnerabilityDatabasesPath *string          `db:"vulnerability_databases_path"`
	EnableHostUsers            bool             `db:"enable_host_users" json:"enable_host_users"`
	EnableSoftwareInventory    bool             `db:"enable_software_inventory" json:"enable_software_inventory"`
}

func Up_20210816141251(tx *sql.Tx) error {
	sql := `
		CREATE TABLE IF NOT EXISTS kv_json (
			id int(10) unsigned NOT NULL AUTO_INCREMENT,
			json_key varchar(255) NOT NULL,
			json_value JSON NOT NULL,
			created_at timestamp DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY unique_kv_json_key(json_key) 
		)
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "create kv_json")
	}

	return nil
}

func Down_20210816141251(tx *sql.Tx) error {
	return nil
}
