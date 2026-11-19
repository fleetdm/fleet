package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// globalOrTeamID converts a nullable team ID to the global_or_team_id value (0 = unassigned).
func globalOrTeamID(teamID *uint) uint {
	if teamID == nil {
		return 0
	}
	return *teamID
}

func (ds *AndroidDatastore) GetZeroTouchEnrollmentToken(ctx context.Context, teamID *uint) (*android.ZeroTouchToken, error) {
	stmt := `SELECT id, team_id, token_name, token_value, expires_at, created_at, updated_at, uses_fleet_policy
		FROM android_zero_touch_tokens WHERE global_or_team_id = ?`
	var token android.ZeroTouchToken
	err := sqlx.GetContext(ctx, ds.reader(ctx), &token, stmt, globalOrTeamID(teamID))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, common_mysql.NotFound("Android zero-touch enrollment token")
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting zero-touch enrollment token")
	}
	return &token, nil
}

func (ds *AndroidDatastore) CreateZeroTouchEnrollmentToken(ctx context.Context, token *android.ZeroTouchToken) (*android.ZeroTouchToken, error) {
	stmt := `INSERT INTO android_zero_touch_tokens (team_id, global_or_team_id, token_name, token_value, expires_at, uses_fleet_policy)
		VALUES (?, ?, ?, ?, ?, ?)`
	res, err := ds.Writer(ctx).ExecContext(ctx, stmt, token.TeamID, globalOrTeamID(token.TeamID), token.TokenName, token.TokenValue, token.ExpiresAt,
		token.UsesFleetPolicy)
	if IsDuplicate(err) {
		return ds.GetZeroTouchEnrollmentToken(ctxdb.RequirePrimary(ctx, true), token.TeamID)
	}
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating zero-touch enrollment token")
	}
	id, _ := res.LastInsertId()
	token.ID = uint(id) // nolint:gosec // dismiss G115
	return token, nil
}

func (ds *AndroidDatastore) ReplaceZeroTouchEnrollmentToken(ctx context.Context, oldTokenName string, token *android.ZeroTouchToken) (*android.ZeroTouchToken, error) {
	// Matching on the old name makes concurrent replacements safe: only the first one updates the row,
	// and every caller returns what's stored.
	stmt := `UPDATE android_zero_touch_tokens
		SET token_name = ?, token_value = ?, expires_at = ?, uses_fleet_policy = ?
		WHERE global_or_team_id = ? AND token_name = ?`
	_, err := ds.Writer(ctx).ExecContext(ctx, stmt, token.TokenName, token.TokenValue, token.ExpiresAt, token.UsesFleetPolicy,
		globalOrTeamID(token.TeamID), oldTokenName)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "replacing zero-touch enrollment token")
	}
	return ds.GetZeroTouchEnrollmentToken(ctxdb.RequirePrimary(ctx, true), token.TeamID)
}

func (ds *AndroidDatastore) DeleteZeroTouchEnrollmentTokens(ctx context.Context) error {
	_, err := ds.Writer(ctx).ExecContext(ctx, `DELETE FROM android_zero_touch_tokens`)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting all zero-touch enrollment tokens")
	}
	return nil
}
