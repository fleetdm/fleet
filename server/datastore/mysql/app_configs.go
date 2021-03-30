package mysql

import (
	"fmt"

	"github.com/VividCortex/mysqlerr"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
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
	var err error
	hostExpiryConfig := struct {
		Window int `db:"host_expiry_window"`
	}{}
	if err = d.db.Get(&hostExpiryConfig, "SELECT host_expiry_window from app_configs LIMIT 1"); err != nil {
		return errors.Wrap(err, "get expiry window setting")
	}

	shouldUpdateWindow := hostExpiryEnabled && hostExpiryConfig.Window != hostExpiryWindow

	if !hostExpiryEnabled || shouldUpdateWindow {
		if _, err := d.db.Exec("DROP EVENT IF EXISTS host_expiry"); err != nil {
			if driverErr, ok := err.(*mysql.MySQLError); !ok || driverErr.Number != mysqlerr.ER_DBACCESS_DENIED_ERROR {
				return errors.Wrap(err, "drop existing host_expiry event")
			}
		}
	}

	if shouldUpdateWindow {
		sql := fmt.Sprintf("CREATE EVENT IF NOT EXISTS host_expiry ON SCHEDULE EVERY 1 HOUR ON COMPLETION PRESERVE DO DELETE FROM hosts WHERE seen_time < DATE_SUB(NOW(), INTERVAL %d DAY)", hostExpiryWindow)
		if _, err := d.db.Exec(sql); err != nil {
			return errors.Wrap(err, "create new host_expiry event")
		}
	}
	return nil
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
      enable_sso_idp_login,
      fim_interval,
      fim_file_accesses,
      host_expiry_enabled,
      host_expiry_window,
      live_query_disabled,
      additional_queries
    )
    VALUES( 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? )
    ON DUPLICATE KEY UPDATE
      org_name = VALUES(org_name),
      org_logo_url = VALUES(org_logo_url),
      kolide_server_url = VALUES(kolide_server_url),
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
      enable_sso_idp_login = VALUES(enable_sso_idp_login),
      fim_interval = VALUES(fim_interval),
      fim_file_accesses = VALUES(fim_file_accesses),
      host_expiry_enabled = VALUES(host_expiry_enabled),
      host_expiry_window = VALUES(host_expiry_window),
      live_query_disabled = VALUES(live_query_disabled),
      additional_queries = VALUES(additional_queries)
    `

	_, err = d.db.Exec(insertStatement,
		info.OrgName,
		info.OrgLogoURL,
		info.KolideServerURL,
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
		info.EnableSSOIdPLogin,
		info.FIMInterval,
		info.FIMFileAccesses,
		info.HostExpiryEnabled,
		info.HostExpiryWindow,
		info.LiveQueryDisabled,
		info.AdditionalQueries,
	)

	return err
}

func (d *Datastore) VerifyEnrollSecret(secret string) (string, error) {
	var s kolide.EnrollSecret
	err := d.db.Get(&s, "SELECT name, active FROM enroll_secrets WHERE secret = ?", secret)
	if err != nil {
		return "", errors.New("no matching secret found")
	}
	if !s.Active {
		return "", errors.New("secret is inactive")
	}

	return s.Name, nil
}

func (d *Datastore) ApplyEnrollSecretSpec(spec *kolide.EnrollSecretSpec) error {
	err := d.withRetryTxx(func(tx *sqlx.Tx) error {
		for _, secret := range spec.Secrets {
			sql := `
				INSERT INTO enroll_secrets (name, secret, active)
				VALUES (?, ?, ?)
				ON DUPLICATE KEY UPDATE
					secret = VALUES(secret),
					active = VALUES(active)
			`
			if _, err := tx.Exec(sql, secret.Name, secret.Secret, secret.Active); err != nil {
				return errors.Wrap(err, "upsert secret")
			}
		}
		return nil
	})

	return err
}

func (d *Datastore) GetEnrollSecretSpec() (*kolide.EnrollSecretSpec, error) {
	var spec kolide.EnrollSecretSpec
	sql := `SELECT * FROM enroll_secrets`
	if err := d.db.Select(&spec.Secrets, sql); err != nil {
		return nil, errors.Wrap(err, "get secrets")
	}
	return &spec, nil
}
