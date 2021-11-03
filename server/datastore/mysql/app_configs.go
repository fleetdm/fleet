package mysql

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (d *Datastore) NewAppConfig(ctx context.Context, info *fleet.AppConfig) (*fleet.AppConfig, error) {
	info.ApplyDefaultsForNewInstalls()

	if err := d.SaveAppConfig(ctx, info); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new app config")
	}

	return info, nil
}

func (d *Datastore) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	return appConfigDB(ctx, d.reader)
}

func appConfigDB(ctx context.Context, q sqlx.QueryerContext) (*fleet.AppConfig, error) {
	info := &fleet.AppConfig{}
	var bytes []byte
	err := sqlx.GetContext(ctx, q, &bytes, `SELECT json_value FROM app_config_json LIMIT 1`)
	if err != nil && err != sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, err, "selecting app config")
	}
	if err == sql.ErrNoRows {
		return &fleet.AppConfig{}, nil
	}

	info.ApplyDefaults()

	err = json.Unmarshal(bytes, info)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshaling config")
	}
	return info, nil
}

func (d *Datastore) SaveAppConfig(ctx context.Context, info *fleet.AppConfig) error {
	configBytes, err := json.Marshal(info)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling config")
	}

	return d.withTx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`,
			configBytes,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert app_config_json")
		}

		if !info.SSOSettings.EnableSSO {
			_, err = tx.ExecContext(ctx, `UPDATE users SET sso_enabled=false`)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "update users sso")
			}
		}

		return nil
	})
}

func (d *Datastore) VerifyEnrollSecret(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
	var s fleet.EnrollSecret
	err := sqlx.GetContext(ctx, d.reader, &s, "SELECT team_id FROM enroll_secrets WHERE secret = ?", secret)
	if err != nil {
		return nil, ctxerr.New(ctx, "no matching secret found")
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
			return ctxerr.Wrap(ctx, err, "clear before insert")
		}
	} else {
		sql := `DELETE FROM enroll_secrets WHERE team_id IS NULL`
		if _, err := exec.ExecContext(ctx, sql); err != nil {
			return ctxerr.Wrap(ctx, err, "clear before insert")
		}
	}

	for _, secret := range secrets {
		sql := `
				INSERT INTO enroll_secrets (secret, team_id)
				VALUES ( ?, ? )
			`
		if _, err := exec.ExecContext(ctx, sql, secret.Secret, teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert secret")
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
		return nil, ctxerr.Wrap(ctx, err, "get secrets")
	}
	return secrets, nil
}
