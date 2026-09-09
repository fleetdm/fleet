package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

func (ds *AndroidDatastore) GetZeroTouchEnrollmentToken(ctx context.Context, teamID *uint) (*android.ZeroTouchToken, error) {
	stmt := `SELECT id, team_id, token_name, token_value, enroll_secret, expires_at, created_at, updated_at
		FROM android_zero_touch_tokens WHERE team_id <=> ?`
	var token android.ZeroTouchToken
	err := sqlx.GetContext(ctx, ds.reader(ctx), &token, stmt, teamID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, common_mysql.NotFound("Android zero-touch enrollment token")
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting zero-touch enrollment token")
	}
	return &token, nil
}

func (ds *AndroidDatastore) CreateZeroTouchEnrollmentToken(ctx context.Context, token *android.ZeroTouchToken) (*android.ZeroTouchToken, error) {
	stmt := `INSERT INTO android_zero_touch_tokens (team_id, token_name, token_value, enroll_secret, expires_at)
		VALUES (?, ?, ?, ?, ?)`
	res, err := ds.Writer(ctx).ExecContext(ctx, stmt, token.TeamID, token.TokenName, token.TokenValue, token.EnrollSecret, token.ExpiresAt)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating zero-touch enrollment token")
	}
	id, _ := res.LastInsertId()
	token.ID = uint(id) // nolint:gosec // dismiss G115
	return token, nil
}

func (ds *AndroidDatastore) DeleteZeroTouchEnrollmentTokens(ctx context.Context) error {
	_, err := ds.Writer(ctx).ExecContext(ctx, `DELETE FROM android_zero_touch_tokens`)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting all zero-touch enrollment tokens")
	}
	return nil
}
