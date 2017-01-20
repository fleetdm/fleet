package mysql

import (
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/pkg/errors"
)

func (d *Datastore) NewAppConfig(info *kolide.AppConfig) (*kolide.AppConfig, error) {
	if err := d.SaveAppConfig(info); err != nil {
		return nil, errors.Wrap(err, "new app config")
	}

	return info, nil

}

func (d *Datastore) AppConfig() (*kolide.AppConfig, error) {
	info := &kolide.AppConfig{}
	err := d.db.Get(info, "SELECT * FROM app_configs LIMIT 1")
	if err != nil {
		return nil, errors.Wrap(err, "selecting app config")
	}
	return info, nil
}

func (d *Datastore) SaveAppConfig(info *kolide.AppConfig) error {
	// Note that we hard code the ID column to 1, insuring that, if no rows
	// exist, a row will be created with INSERT, if a row does exist the key
	// will be violate uniqueness constraint and an UPDATE will occur
	insertStatement := `
		INSERT INTO app_configs (
			id,
			org_name,
			org_logo_url,
			kolide_server_url,
			osquery_enroll_secret,
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
			smtp_enable_start_tls
		)
		VALUES( 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? )
		ON DUPLICATE KEY UPDATE
			org_name = VALUES(org_name),
			org_logo_url = VALUES(org_logo_url),
			kolide_server_url = VALUES(kolide_server_url),
			osquery_enroll_secret = VALUES(osquery_enroll_secret),
			smtp_configured = VALUES(smtp_configured),
			smtp_sender_address = VALUES(smtp_sender_address),
			smtp_server = VALUES(smtp_server),
			smtp_port = VALUES(smtp_port),
			smtp_authentication_type = VALUES(smtp_authentication_type),
			smtp_enable_ssl_tls = VALUES(smtp_enable_ssl_tls),
			smtp_authentication_method = VALUES(smtp_authentication_method),
			smtp_domain = VALUES(smtp_domain),
			smtp_user_name = VALUES(smtp_user_name),
			smtp_password = VALUES(smtp_password),
			smtp_verify_ssl_certs = VALUES(smtp_verify_ssl_certs),
			smtp_enable_start_tls = VALUES(smtp_enable_start_tls)
	`

	_, err := d.db.Exec(insertStatement,
		info.OrgName,
		info.OrgLogoURL,
		info.KolideServerURL,
		info.EnrollSecret,
		info.SMTPConfigured,
		info.SMTPSenderAddress,
		info.SMTPServer,
		info.SMTPPort,
		info.SMTPAuthenticationType,
		info.SMTPEnableTLS,
		info.SMTPAuthenticationMethod,
		info.SMTPDomain,
		info.SMTPUserName,
		info.SMTPPassword,
		info.SMTPVerifySSLCerts,
		info.SMTPEnableStartTLS,
	)

	return err
}
