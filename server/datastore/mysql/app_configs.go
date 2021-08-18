package mysql

import (
	"fmt"

	"github.com/VividCortex/mysqlerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func (d *Datastore) NewAppConfig(info *fleet.AppConfig) (*fleet.AppConfig, error) {
	if err := d.SaveAppConfig(info); err != nil {
		return nil, errors.Wrap(err, "new app config")
	}

	return info, nil
}

func (d *Datastore) AppConfig() (*fleet.AppConfig, error) {
	info := &fleet.AppConfig{}
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

func (d *Datastore) ManageHostExpiryEvent(tx *sqlx.Tx, hostExpiryEnabled bool, hostExpiryWindow int) error {
	var err error
	hostExpiryConfig := struct {
		Window int `db:"host_expiry_window"`
	}{}
	if err = tx.Get(&hostExpiryConfig, "SELECT host_expiry_window from app_configs LIMIT 1"); err != nil {
		return errors.Wrap(err, "get expiry window setting")
	}

	shouldUpdateWindow := hostExpiryEnabled && hostExpiryConfig.Window != hostExpiryWindow

	if !hostExpiryEnabled || shouldUpdateWindow {
		if _, err := tx.Exec("DROP EVENT IF EXISTS host_expiry"); err != nil {
			if driverErr, ok := err.(*mysql.MySQLError); !ok || driverErr.Number != mysqlerr.ER_DBACCESS_DENIED_ERROR {
				return errors.Wrap(err, "drop existing host_expiry event")
			}
		}
	}

	if shouldUpdateWindow {
		sql := fmt.Sprintf("CREATE EVENT IF NOT EXISTS host_expiry ON SCHEDULE EVERY 1 HOUR ON COMPLETION PRESERVE DO DELETE FROM hosts WHERE seen_time < DATE_SUB(NOW(), INTERVAL %d DAY)", hostExpiryWindow)
		if _, err := tx.Exec(sql); err != nil {
			return errors.Wrap(err, "create new host_expiry event")
		}
	}
	return nil
}

func (d *Datastore) SaveAppConfig(info *fleet.AppConfig) error {
	eventSchedulerEnabled, err := d.isEventSchedulerEnabled()
	if err != nil {
		return err
	}

	if !eventSchedulerEnabled && info.HostExpiryEnabled {
		return errors.New("MySQL Event Scheduler must be enabled to configure Host Expiry.")
	}

	return d.withTx(func(tx *sqlx.Tx) error {
		if err := d.ManageHostExpiryEvent(tx, info.HostExpiryEnabled, info.HostExpiryWindow); err != nil {
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
			enable_sso_idp_login,
			fim_interval,
			fim_file_accesses,
			host_expiry_enabled,
			host_expiry_window,
			live_query_disabled,
			additional_queries,
			agent_options,
			enable_analytics,
			vulnerability_databases_path,
			enable_host_users,
			enable_software_inventory
		)
		VALUES( 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? )
		ON DUPLICATE KEY UPDATE
			org_name = VALUES(org_name),
			org_logo_url = VALUES(org_logo_url),
			server_url = VALUES(server_url),
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
			additional_queries = VALUES(additional_queries),
			agent_options = VALUES(agent_options),
			enable_analytics = VALUES(enable_analytics),
			vulnerability_databases_path = VALUES(vulnerability_databases_path),
			enable_host_users = VALUES(enable_host_users),
			enable_software_inventory = VALUES(enable_software_inventory)
    `

		_, err = tx.Exec(insertStatement,
			info.OrgName,
			info.OrgLogoURL,
			info.ServerURL,
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
			info.AgentOptions,
			info.EnableAnalytics,
			info.VulnerabilityDatabasesPath,
			info.EnableHostUsers,
			info.EnableSoftwareInventory,
		)
		if err != nil {
			return err
		}

		if !info.EnableSSO {
			_, err = tx.Exec(`UPDATE users SET sso_enabled=false`)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (d *Datastore) VerifyEnrollSecret(secret string) (*fleet.EnrollSecret, error) {
	var s fleet.EnrollSecret
	err := d.db.Get(&s, "SELECT team_id FROM enroll_secrets WHERE secret = ?", secret)
	if err != nil {
		return nil, errors.New("no matching secret found")
	}

	return &s, nil
}

func (d *Datastore) ApplyEnrollSecrets(teamID *uint, secrets []*fleet.EnrollSecret) error {
	err := d.withRetryTxx(func(tx *sqlx.Tx) error {
		if teamID != nil {
			sql := `DELETE FROM enroll_secrets WHERE team_id = ?`
			if _, err := tx.Exec(sql, teamID); err != nil {
				return errors.Wrap(err, "clear before insert")
			}
		} else {
			sql := `DELETE FROM enroll_secrets WHERE team_id IS NULL`
			if _, err := tx.Exec(sql); err != nil {
				return errors.Wrap(err, "clear before insert")
			}
		}

		for _, secret := range secrets {
			sql := `
				INSERT INTO enroll_secrets (secret, team_id)
				VALUES ( ?, ? )
			`
			if _, err := tx.Exec(sql, secret.Secret, teamID); err != nil {
				return errors.Wrap(err, "upsert secret")
			}
		}
		return nil
	})

	return err
}

func (d *Datastore) GetEnrollSecrets(teamID *uint) ([]*fleet.EnrollSecret, error) {
	var args []interface{}
	sql := "SELECT * FROM enroll_secrets WHERE "
	// MySQL requires comparing NULL with IS. NULL = NULL evaluates to FALSE.
	if teamID == nil {
		sql += "team_id IS NULL"
	} else {
		sql += "team_id = ?"
		args = append(args, teamID)
	}
	var secrets []*fleet.EnrollSecret
	if err := d.db.Select(&secrets, sql, args...); err != nil {
		return nil, errors.Wrap(err, "get secrets")
	}
	return secrets, nil
}
