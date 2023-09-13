package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) NewAppConfig(ctx context.Context, info *fleet.AppConfig) (*fleet.AppConfig, error) {
	info.ApplyDefaultsForNewInstalls()

	if err := ds.SaveAppConfig(ctx, info); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new app config")
	}

	return info, nil
}

func (ds *Datastore) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	return appConfigDB(ctx, ds.reader(ctx))
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

func (ds *Datastore) SaveAppConfig(ctx context.Context, info *fleet.AppConfig) error {
	configBytes, err := json.Marshal(info)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling config")
	}

	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`,
			configBytes,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert app_config_json")
		}

		if info.SSOSettings != nil && !info.SSOSettings.EnableSSO {
			_, err = tx.ExecContext(ctx, `UPDATE users SET sso_enabled=false`)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "update users sso")
			}
		}

		return nil
	})
}

func (ds *Datastore) VerifyEnrollSecret(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
	var s fleet.EnrollSecret
	err := sqlx.GetContext(ctx, ds.reader(ctx), &s, "SELECT team_id FROM enroll_secrets WHERE secret = ?", secret)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("EnrollSecret"), "no matching secret found")
		}
		return nil, ctxerr.Wrap(ctx, err, "verify enroll secret")
	}

	return &s, nil
}

func (ds *Datastore) ApplyEnrollSecrets(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return applyEnrollSecretsDB(ctx, tx, teamID, secrets)
	})
}

func applyEnrollSecretsDB(ctx context.Context, q sqlx.ExtContext, teamID *uint, secrets []*fleet.EnrollSecret) error {
	// NOTE: this is called from within a transaction (either from
	// ApplyEnrollSecrets or saveTeamSecretsDB). We don't do a simple DELETE then
	// INSERT as we need to keep the existing created_at timestamps of
	// already-existing secrets. We also can't do a DELETE unused ones and then
	// UPSERT new ones, because we need to fail the INSERT if the secret already
	// exists for a different team or globally (i.e. the `secret` column is
	// unique across all values of team_id, NULL or not). An "ON DUPLICATE KEY
	// UPDATE" clause would silence such errors.
	//
	// For this reason, we first read the existing secrets to have their
	// created_at timestamps, then we delete and re-insert them, failing the call
	// if the insert failed (due to a secret existing at a different team/global
	// level).

	var args []interface{}
	teamWhere := "team_id IS NULL"
	if teamID != nil {
		teamWhere = "team_id = ?"
		args = append(args, *teamID)
	}

	// first, load the existing secrets and their created_at timestamp
	const loadStmt = `SELECT secret, created_at FROM enroll_secrets WHERE `
	var existingSecrets []*fleet.EnrollSecret
	if err := sqlx.SelectContext(ctx, q, &existingSecrets, loadStmt+teamWhere, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "load existing secrets")
	}
	secretsCreatedAt := make(map[string]*time.Time, len(existingSecrets))
	for _, es := range existingSecrets {
		es := es
		secretsCreatedAt[es.Secret] = &es.CreatedAt
	}

	// next, remove all existing secrets for that team or global
	const delStmt = `DELETE FROM enroll_secrets WHERE `
	if _, err := q.ExecContext(ctx, delStmt+teamWhere, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "clear before insert")
	}

	newSecrets := make([]string, len(secrets))
	for i, s := range secrets {
		newSecrets[i] = s.Secret
	}

	// finally, insert the new secrets, using the existing created_at timestamp
	// if available.
	const insStmt = `INSERT INTO enroll_secrets (secret, team_id, created_at) VALUES %s`
	if len(newSecrets) > 0 {
		var args []interface{}
		defaultCreatedAt := time.Now()
		sql := fmt.Sprintf(insStmt, strings.TrimSuffix(strings.Repeat(`(?,?,?),`, len(newSecrets)), ","))

		for _, s := range secrets {
			secretCreatedAt := defaultCreatedAt
			if ts := secretsCreatedAt[s.Secret]; ts != nil {
				secretCreatedAt = *ts
			}
			args = append(args, s.Secret, teamID, secretCreatedAt)
		}
		if _, err := q.ExecContext(ctx, sql, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert secrets")
		}
	}
	return nil
}

func (ds *Datastore) GetEnrollSecrets(ctx context.Context, teamID *uint) ([]*fleet.EnrollSecret, error) {
	return getEnrollSecretsDB(ctx, ds.reader(ctx), teamID)
}

func getEnrollSecretsDB(ctx context.Context, q sqlx.QueryerContext, teamID *uint) ([]*fleet.EnrollSecret, error) {
	var args []interface{}
	sql := "SELECT secret, team_id, created_at FROM enroll_secrets WHERE "
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

func (ds *Datastore) AggregateEnrollSecretPerTeam(ctx context.Context) ([]*fleet.EnrollSecret, error) {
	query := `
          SELECT
             COALESCE((
             SELECT
                es.secret
             FROM
                enroll_secrets es
             WHERE
                es.team_id = t.id
             ORDER BY
                es.created_at DESC LIMIT 1), '') as secret,
                t.id as team_id
             FROM
                teams t
             UNION
          (
             SELECT
                COALESCE(secret, '') as secret, team_id
             FROM
                enroll_secrets
             WHERE
                team_id IS NULL
             ORDER BY
                created_at DESC LIMIT 1)
	`
	var secrets []*fleet.EnrollSecret
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &secrets, query); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get secrets")
	}
	return secrets, nil
}
