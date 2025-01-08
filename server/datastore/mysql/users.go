package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

var userSearchColumns = []string{"name", "email"}

// NewUser creates a new user
func (ds *Datastore) NewUser(ctx context.Context, user *fleet.User) (*fleet.User, error) {
	if err := fleet.ValidateRole(user.GlobalRole, user.Teams); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validate role")
	}

	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		sqlStatement := `
      INSERT INTO users (
      	password,
      	salt,
      	name,
      	email,
      	admin_forced_password_reset,
      	gravatar_url,
      	position,
        sso_enabled,
	    mfa_enabled,
		api_only,
		global_role
      ) VALUES (?,?,?,?,?,?,?,?,?,?,?)
      `
		result, err := tx.ExecContext(ctx, sqlStatement,
			user.Password,
			user.Salt,
			user.Name,
			user.Email,
			user.AdminForcedPasswordReset,
			user.GravatarURL,
			user.Position,
			user.SSOEnabled,
			user.MFAEnabled,
			user.APIOnly,
			user.GlobalRole)

		// set timestamp as close as possible to insert query to be as accurate as possible without needing to SELECT
		user.CreatedAt = time.Now().UTC().Truncate(time.Second) // truncating because DB is at second resolution
		user.UpdatedAt = user.CreatedAt

		if err != nil {
			return ctxerr.Wrap(ctx, err, "create new user")
		}

		id, _ := result.LastInsertId()
		user.ID = uint(id) //nolint:gosec // dismiss G115

		if err := saveTeamsForUserDB(ctx, tx, user); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (ds *Datastore) findUser(ctx context.Context, searchCol string, searchVal interface{}) (*fleet.User, error) {
	sqlStatement := fmt.Sprintf(
		// everything except `settings`
		"SELECT id, created_at, updated_at, password, salt, name, email, admin_forced_password_reset, gravatar_url, position, sso_enabled, global_role, api_only, mfa_enabled FROM users "+
			"WHERE %s = ? LIMIT 1",
		searchCol,
	)

	user := &fleet.User{}

	err := sqlx.GetContext(ctx, ds.reader(ctx), user, sqlStatement, searchVal)
	if err != nil && err == sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, notFound("User").
			WithMessage(fmt.Sprintf("with %s=%v", searchCol, searchVal)))
	} else if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "find user")
	}

	if err := ds.loadTeamsForUsers(ctx, []*fleet.User{user}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load teams")
	}

	// When SSO is enabled, we can ignore forced password resets
	// However, we want to leave the db untouched, to cover cases where SSO is toggled
	if user.SSOEnabled {
		user.AdminForcedPasswordReset = false
	}

	return user, nil
}

// ListUsers lists all users with team ID, limit, sort and offset passed in with
// UserListOptions.
func (ds *Datastore) ListUsers(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
	sqlStatement := `
		SELECT * FROM users
		WHERE TRUE
	`
	var params []interface{}
	if opt.TeamID != 0 {
		sqlStatement += " AND id IN (SELECT user_id FROM user_teams WHERE team_id = ?)"
		params = append(params, opt.TeamID)
	}

	sqlStatement, params = searchLike(sqlStatement, params, opt.MatchQuery, userSearchColumns...)
	sqlStatement, params = appendListOptionsWithCursorToSQL(sqlStatement, params, &opt.ListOptions)
	users := []*fleet.User{}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &users, sqlStatement, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list users")
	}

	if err := ds.loadTeamsForUsers(ctx, users); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load teams")
	}

	return users, nil
}

func (ds *Datastore) UserByEmail(ctx context.Context, email string) (*fleet.User, error) {
	return ds.findUser(ctx, "email", email)
}

func (ds *Datastore) UserByID(ctx context.Context, id uint) (*fleet.User, error) {
	return ds.findUser(ctx, "id", id)
}

func (ds *Datastore) SaveUser(ctx context.Context, user *fleet.User) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		return saveUserDB(ctx, tx, user)
	})
}

func (ds *Datastore) SaveUsers(ctx context.Context, users []*fleet.User) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		for _, user := range users {
			err := saveUserDB(ctx, tx, user)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func saveUserDB(ctx context.Context, tx sqlx.ExtContext, user *fleet.User) error {
	if err := fleet.ValidateRole(user.GlobalRole, user.Teams); err != nil {
		return ctxerr.Wrap(ctx, err, "validate role")
	}
	sqlStatement := `
      UPDATE users SET
      	password = ?,
      	salt = ?,
      	name = ?,
      	email = ?,
      	admin_forced_password_reset = ?,
      	gravatar_url = ?,
      	position = ?,
        sso_enabled = ?,
        mfa_enabled = ?,
        api_only = ?,
				settings = ?,
		global_role = ?
      WHERE id = ?
      `
	settingsBytes, err := json.Marshal(user.Settings)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling user settings")
	}
	result, err := tx.ExecContext(ctx, sqlStatement,
		user.Password,
		user.Salt,
		user.Name,
		user.Email,
		user.AdminForcedPasswordReset,
		user.GravatarURL,
		user.Position,
		user.SSOEnabled,
		user.MFAEnabled,
		user.APIOnly,
		settingsBytes,
		user.GlobalRole,
		user.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "save user")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "rows affected save user")
	}
	if rows == 0 {
		return ctxerr.Wrap(ctx, notFound("User").WithID(user.ID))
	}

	// REVIEW: Check if teams have been set?
	if err := saveTeamsForUserDB(ctx, tx, user); err != nil {
		return err
	}

	return nil
}

// loadTeamsForUsers will load the teams/roles for the provided users.
func (ds *Datastore) loadTeamsForUsers(ctx context.Context, users []*fleet.User) error {
	userIDs := make([]uint, 0, len(users)+1)
	// Make sure the slice is never empty for IN by filling a nonexistent ID
	userIDs = append(userIDs, 0)
	idToUser := make(map[uint]*fleet.User, len(users))
	for _, u := range users {
		// Initialize empty slice so we get an array in JSON responses instead
		// of null if it is empty
		u.Teams = []fleet.UserTeam{}
		// Track IDs for queries and matching
		userIDs = append(userIDs, u.ID)
		idToUser[u.ID] = u
	}

	sql := `
		SELECT ut.team_id AS id, ut.user_id, ut.role, t.name
		FROM user_teams ut INNER JOIN teams t ON ut.team_id = t.id
		WHERE ut.user_id IN (?)
		ORDER BY user_id, team_id
	`
	sql, args, err := sqlx.In(sql, userIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sqlx.In loadTeamsForUsers")
	}

	var rows []struct {
		fleet.UserTeam
		UserID uint `db:"user_id"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, sql, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "get loadTeamsForUsers")
	}

	// Map each row to the appropriate user
	for _, r := range rows {
		user := idToUser[r.UserID]
		user.Teams = append(user.Teams, r.UserTeam)
	}

	return nil
}

func saveTeamsForUserDB(ctx context.Context, tx sqlx.ExtContext, user *fleet.User) error {
	// Do a full teams update by deleting existing teams and then inserting all
	// the current teams in a single transaction.

	// Delete before insert
	sql := `DELETE FROM user_teams WHERE user_id = ?`
	if _, err := tx.ExecContext(ctx, sql, user.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete existing teams")
	}

	if len(user.Teams) == 0 {
		return nil
	}

	// Bulk insert
	const valueStr = "(?,?,?),"
	var args []interface{}
	for _, userTeam := range user.Teams {
		args = append(args, user.ID, userTeam.Team.ID, userTeam.Role)
	}
	sql = "INSERT INTO user_teams (user_id, team_id, role) VALUES " +
		strings.Repeat(valueStr, len(user.Teams))
	sql = strings.TrimSuffix(sql, ",")
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert teams")
	}

	return nil
}

// DeleteUser deletes the associated user
func (ds *Datastore) DeleteUser(ctx context.Context, id uint) error {
	return ds.deleteEntity(ctx, usersTable, id)
}

func tableRowsCount(ctx context.Context, db sqlx.QueryerContext, tableName string) (int, error) {
	var count int
	err := sqlx.GetContext(ctx, db, &count, fmt.Sprintf(`SELECT count(*) FROM %s`, tableName))
	if err != nil {
		return 0, err
	}
	return count, nil
}

func amountActiveUsersSinceDB(ctx context.Context, db sqlx.QueryerContext, since time.Time) (int, error) {
	var amount int
	err := sqlx.GetContext(ctx, db, &amount, `
    SELECT count(*)
    FROM users u
    WHERE EXISTS (
      SELECT 1
      FROM sessions ssn
      WHERE ssn.user_id = u.id AND
      ssn.accessed_at >= ?
    )`, since)
	if err != nil {
		return 0, err
	}
	return amount, nil
}

func (ds *Datastore) UserSettings(ctx context.Context, userID uint) (*fleet.UserSettings, error) {
	settings := &fleet.UserSettings{}
	var bytes []byte
	stmt := `
		SELECT settings FROM users WHERE id = ?
	`
	err := sqlx.GetContext(ctx, ds.reader(ctx), &bytes, stmt, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("UserSettings").WithID(userID))
		}
		return nil, ctxerr.Wrap(ctx, err, "selecting user settings")
	}

	if len(bytes) == 0 {
		return settings, nil
	}

	err = json.Unmarshal(bytes, settings)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshaling user settings")
	}
	return settings, nil
}
