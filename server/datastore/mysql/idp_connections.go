package mysql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// idpConnectionID is the SCIM directory for this request. An explicit context
// value wins. Otherwise the org default connection is used, which is where a
// Fleet API token and Google Workspace sync write.
func idpConnectionID(ctx context.Context, q sqlx.QueryerContext) (uint, error) {
	if id, ok := fleet.IDPConnectionFromContext(ctx); ok {
		return id, nil
	}
	return defaultIDPConnectionID(ctx, q)
}

func defaultIDPConnectionID(ctx context.Context, q sqlx.QueryerContext) (uint, error) {
	var id uint
	err := sqlx.GetContext(ctx, q, &id, `SELECT id FROM idp_connections WHERE is_default = 1 ORDER BY id LIMIT 1`)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ctxerr.New(ctx, "no default identity provider connection")
		}
		return 0, ctxerr.Wrap(ctx, err, "select default identity provider connection")
	}
	return id, nil
}

// idpConnectionIDForHost is the directory for the host's fleet. A fleet that
// names a connection uses that directory. Everyone else uses the org default.
func idpConnectionIDForHost(ctx context.Context, q sqlx.QueryerContext, hostID uint) (uint, error) {
	var teamID sql.NullInt64
	if err := sqlx.GetContext(ctx, q, &teamID, `SELECT team_id FROM hosts WHERE id = ?`, hostID); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "select host fleet for identity provider")
	}
	if teamID.Valid {
		var name sql.NullString
		err := sqlx.GetContext(ctx, q, &name, `
			SELECT NULLIF(JSON_UNQUOTE(JSON_EXTRACT(config, '$.mdm.identity_provider')), 'null')
			FROM teams WHERE id = ?`, teamID.Int64)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return 0, ctxerr.Wrap(ctx, err, "select fleet identity provider")
		}
		if name.Valid && name.String != "" && name.String != "null" {
			var id uint
			err = sqlx.GetContext(ctx, q, &id, `SELECT id FROM idp_connections WHERE name = ?`, name.String)
			if err == nil {
				return id, nil
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return 0, ctxerr.Wrap(ctx, err, "select identity provider connection by name")
			}
		}
	}
	return defaultIDPConnectionID(ctx, q)
}

// SyncIDPConnections keeps idp_connections aligned with the named SAML
// connections. The org default directory keeps the SCIM users that existed
// before connections were named.
func (ds *Datastore) SyncIDPConnections(ctx context.Context, providers []fleet.MDMIdentityProvider) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO idp_connections (name, is_default) VALUES (?, 1)`, fleet.DefaultIDPConnectionName); err != nil {
			return ctxerr.Wrap(ctx, err, "ensure default identity provider connection")
		}
		for _, provider := range providers {
			if provider.Name == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO idp_connections (name, is_default) VALUES (?, 0)`, provider.Name); err != nil {
				return ctxerr.Wrap(ctx, err, "insert identity provider connection")
			}
		}

		defaultName := fleet.DefaultIDPConnectionName
		for _, provider := range providers {
			if provider.Default && provider.Name != "" {
				defaultName = provider.Name
				break
			}
		}
		if _, err := tx.ExecContext(ctx, `UPDATE idp_connections SET is_default = 0`); err != nil {
			return ctxerr.Wrap(ctx, err, "clear identity provider defaults")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE idp_connections SET is_default = 1 WHERE name = ?`, defaultName); err != nil {
			return ctxerr.Wrap(ctx, err, "set default identity provider connection")
		}
		if defaultName == fleet.DefaultIDPConnectionName {
			return nil
		}

		var syntheticID, namedID uint
		if err := sqlx.GetContext(ctx, tx, &syntheticID, `SELECT id FROM idp_connections WHERE name = ?`, fleet.DefaultIDPConnectionName); err != nil {
			return ctxerr.Wrap(ctx, err, "select synthetic identity provider connection")
		}
		if err := sqlx.GetContext(ctx, tx, &namedID, `SELECT id FROM idp_connections WHERE name = ?`, defaultName); err != nil {
			return ctxerr.Wrap(ctx, err, "select named default identity provider connection")
		}
		if syntheticID == namedID {
			return nil
		}
		if _, err := tx.ExecContext(ctx, `UPDATE scim_users SET idp_connection_id = ? WHERE idp_connection_id = ?`, namedID, syntheticID); err != nil {
			return ctxerr.Wrap(ctx, err, "move scim users to the default identity provider")
		}
		if _, err := tx.ExecContext(ctx, `UPDATE scim_groups SET idp_connection_id = ? WHERE idp_connection_id = ?`, namedID, syntheticID); err != nil {
			return ctxerr.Wrap(ctx, err, "move scim groups to the default identity provider")
		}
		return nil
	})
}

// IDPConnectionBySCIMTokenHash returns the connection whose bearer token matches.
func (ds *Datastore) IDPConnectionBySCIMTokenHash(ctx context.Context, hash []byte) (uint, error) {
	var id uint
	err := sqlx.GetContext(ctx, ds.reader(ctx), &id, `SELECT id FROM idp_connections WHERE scim_token_hash = ?`, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, notFound("identity provider connection")
		}
		return 0, ctxerr.Wrap(ctx, err, "select identity provider connection by scim token")
	}
	return id, nil
}

// RotateIDPConnectionSCIMToken replaces the bearer token for one connection and
// returns the new secret. The secret is not stored.
func (ds *Datastore) RotateIDPConnectionSCIMToken(ctx context.Context, name string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", ctxerr.Wrap(ctx, err, "generate scim token")
	}
	token := "scim_" + base64.RawURLEncoding.EncodeToString(raw)
	result, err := ds.writer(ctx).ExecContext(ctx, `UPDATE idp_connections SET scim_token_hash = ? WHERE name = ?`, fleet.HashSCIMToken(token), name)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "store scim token")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "scim token rows affected")
	}
	if rows == 0 {
		return "", notFound("identity provider connection").WithName(name)
	}
	return token, nil
}

// EnsureIDPConnection makes sure a named connection row exists so a token can
// be issued before the next config sync.
func (ds *Datastore) EnsureIDPConnection(ctx context.Context, name string) error {
	if name == "" {
		name = fleet.DefaultIDPConnectionName
	}
	_, err := ds.writer(ctx).ExecContext(ctx, `INSERT IGNORE INTO idp_connections (name, is_default) VALUES (?, ?)`, name, name == fleet.DefaultIDPConnectionName)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "ensure identity provider connection")
	}
	return nil
}
