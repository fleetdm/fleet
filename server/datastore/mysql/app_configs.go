package mysql

import (
	"fmt"

	"github.com/kolide/fleet/server/kolide"
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

func (d *Datastore) isEventSchedulerEnabled() (bool, error) {
	rows, err := d.db.Query("SELECT @@event_scheduler")
	if err != nil {
		return false, err
	}
	if !rows.Next() {
		return false, errors.New("Error detecting MySQL event scheduler status.")
	}
	var value string
	if err := rows.Scan(&value); err != nil {
		return false, err
	}

	return value == "ON", nil
}

func (d *Datastore) ManageHostExpiryEvent(hostExpiryEnabled bool, hostExpiryWindow int) error {
	if !hostExpiryEnabled {
		_, err := d.db.Exec("DROP EVENT IF EXISTS host_expiry")
		return err
	}

	_, err := d.db.Exec(fmt.Sprintf("CREATE EVENT IF NOT EXISTS host_expiry ON SCHEDULE EVERY 1 HOUR ON COMPLETION PRESERVE DO DELETE FROM hosts WHERE seen_time < DATE_SUB(NOW(), INTERVAL %d DAY)", hostExpiryWindow))
	return err
}

func (d *Datastore) SaveAppConfig(info *kolide.AppConfig) error {
	eventSchedulerEnabled, err := d.isEventSchedulerEnabled()
	if err != nil {
		return err
	}

	if !eventSchedulerEnabled && info.HostExpiryEnabled {
		return errors.New("MySQL Event Scheduler must be enabled to configure Host Expiry.")
	}

	if err := d.ManageHostExpiryEvent(info.HostExpiryEnabled, info.HostExpiryWindow); err != nil {
		return err
	}

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
      smtp_enable_start_tls,
      entity_id,
      issuer_uri,
      idp_image_url,
      metadata,
      metadata_url,
      idp_name,
      enable_sso,
      fim_interval,
      fim_file_accesses,
      host_expiry_enabled,
      host_expiry_window
    )
    VALUES( 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? )
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
      smtp_enable_start_tls = VALUES(smtp_enable_start_tls),
      entity_id = VALUES(entity_id),
      issuer_uri = VALUES(issuer_uri),
      idp_image_url = VALUES(idp_image_url),
      metadata = VALUES(metadata),
      metadata_url = VALUES(metadata_url),
      idp_name = VALUES(idp_name),
      enable_sso = VALUES(enable_sso),
      fim_interval = VALUES(fim_interval),
      fim_file_accesses = VALUES(fim_file_accesses),
      host_expiry_enabled = VALUES(host_expiry_enabled),
      host_expiry_window = VALUES(host_expiry_window)
    `

	_, err = d.db.Exec(insertStatement,
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
		info.EntityID,
		info.IssuerURI,
		info.IDPImageURL,
		info.Metadata,
		info.MetadataURL,
		info.IDPName,
		info.EnableSSO,
		info.FIMInterval,
		info.FIMFileAccesses,
		info.HostExpiryEnabled,
		info.HostExpiryWindow,
	)

	return err
}
