package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/VividCortex/mysqlerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func (d *Datastore) NewAppConfig(ctx context.Context, info *fleet.AppConfig) (*fleet.AppConfig, error) {
	info.ApplyDefaultsForNewInstalls()

	if err := d.SaveAppConfig(ctx, info); err != nil {
		return nil, errors.Wrap(err, "new app config")
	}

	return info, nil
}

func (d *Datastore) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	info := &fleet.AppConfig{}
	var bytes []byte
	err := sqlx.GetContext(ctx, d.reader, &bytes, `SELECT json_value FROM app_config_json LIMIT 1`)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrap(err, "selecting app config")
	}
	if err == sql.ErrNoRows {
		return &fleet.AppConfig{}, nil
	}

	info.ApplyDefaults()

	err = json.Unmarshal(bytes, info)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshaling config")
	}
	return info, nil
}

func (d *Datastore) isEventSchedulerEnabled() (bool, error) {
	rows, err := d.writer.Query("SELECT @@event_scheduler")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	if !rows.Next() {
		err := errors.New("Error detecting MySQL event scheduler status.")
		if rerr := rows.Err(); rerr != nil {
			err = rerr
		}
		return false, err
	}
	var value string
	if err := rows.Scan(&value); err != nil {
		return false, err
	}

	return value == "ON", nil
}

func manageHostExpiryEventDB(ctx context.Context, tx sqlx.ExtContext, hostExpiryEnabled bool, hostExpiryWindow int) error {
	var err error
	hostExpiryConfig := struct {
		Window int `db:"host_expiry_window"`
	}{}
	if err = sqlx.GetContext(ctx, tx, &hostExpiryConfig, "SELECT host_expiry_window from app_configs LIMIT 1"); err != nil {
		return errors.Wrap(err, "get expiry window setting")
	}

	shouldUpdateWindow := hostExpiryEnabled && hostExpiryConfig.Window != hostExpiryWindow

	if !hostExpiryEnabled || shouldUpdateWindow {
		if _, err := tx.ExecContext(ctx, "DROP EVENT IF EXISTS host_expiry"); err != nil {
			if driverErr, ok := err.(*mysql.MySQLError); !ok || driverErr.Number != mysqlerr.ER_DBACCESS_DENIED_ERROR {
				return errors.Wrap(err, "drop existing host_expiry event")
			}
		}
	}

	if shouldUpdateWindow {
		sql := fmt.Sprintf("CREATE EVENT IF NOT EXISTS host_expiry ON SCHEDULE EVERY 1 HOUR ON COMPLETION PRESERVE DO DELETE FROM hosts WHERE seen_time < DATE_SUB(NOW(), INTERVAL %d DAY)", hostExpiryWindow)
		if _, err := tx.ExecContext(ctx, sql); err != nil {
			return errors.Wrap(err, "create new host_expiry event")
		}
	}
	return nil
}

func (d *Datastore) SaveAppConfig(ctx context.Context, info *fleet.AppConfig) error {
	eventSchedulerEnabled, err := d.isEventSchedulerEnabled()
	if err != nil {
		return err
	}

	expiryEnabled := info.HostExpirySettings.HostExpiryEnabled
	expiryWindow := info.HostExpirySettings.HostExpiryWindow

	if !eventSchedulerEnabled && expiryEnabled {
		return errors.New("MySQL event scheduler must be enabled to configure host expiry.")
	}

	configBytes, err := json.Marshal(info)
	if err != nil {
		return errors.Wrap(err, "marshaling config")
	}

	return d.withTx(ctx, func(tx sqlx.ExtContext) error {
		if err := manageHostExpiryEventDB(ctx, tx, expiryEnabled, expiryWindow); err != nil {
			return err
		}

		_, err := tx.ExecContext(ctx,
			`INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`,
			configBytes,
		)
		if err != nil {
			return err
		}

		if !info.SSOSettings.EnableSSO {
			_, err = tx.ExecContext(ctx, `UPDATE users SET sso_enabled=false`)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (d *Datastore) VerifyEnrollSecret(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
	var s fleet.EnrollSecret
	err := sqlx.GetContext(ctx, d.reader, &s, "SELECT team_id FROM enroll_secrets WHERE secret = ?", secret)
	if err != nil {
		return nil, errors.New("no matching secret found")
	}

	return &s, nil
}

func (d *Datastore) ApplyEnrollSecrets(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
	return d.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return applyEnrollSecretsDB(ctx, tx, teamID, secrets)
	})
}

func applyEnrollSecretsDB(ctx context.Context, exec sqlx.ExecerContext, teamID *uint, secrets []*fleet.EnrollSecret) error {
	if teamID != nil {
		sql := `DELETE FROM enroll_secrets WHERE team_id = ?`
		if _, err := exec.ExecContext(ctx, sql, teamID); err != nil {
			return errors.Wrap(err, "clear before insert")
		}
	} else {
		sql := `DELETE FROM enroll_secrets WHERE team_id IS NULL`
		if _, err := exec.ExecContext(ctx, sql); err != nil {
			return errors.Wrap(err, "clear before insert")
		}
	}

	for _, secret := range secrets {
		sql := `
				INSERT INTO enroll_secrets (secret, team_id)
				VALUES ( ?, ? )
			`
		if _, err := exec.ExecContext(ctx, sql, secret.Secret, teamID); err != nil {
			return errors.Wrap(err, "upsert secret")
		}
	}
	return nil
}

func (d *Datastore) GetEnrollSecrets(ctx context.Context, teamID *uint) ([]*fleet.EnrollSecret, error) {
	return getEnrollSecretsDB(ctx, d.reader, teamID)
}

func getEnrollSecretsDB(ctx context.Context, q sqlx.QueryerContext, teamID *uint) ([]*fleet.EnrollSecret, error) {
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
	if err := sqlx.SelectContext(ctx, q, &secrets, sql, args...); err != nil {
		return nil, errors.Wrap(err, "get secrets")
	}
	return secrets, nil
}
