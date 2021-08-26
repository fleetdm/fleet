package mysql

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var userSearchColumns = []string{"name", "email"}

// NewUser creates a new user
func (d *Datastore) NewUser(user *fleet.User) (*fleet.User, error) {
	if err := fleet.ValidateRole(user.GlobalRole, user.Teams); err != nil {
		return nil, err
	}

	err := d.withTx(func(tx *sqlx.Tx) error {
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
		api_only,
		global_role
      ) VALUES (?,?,?,?,?,?,?,?,?,?)
      `
		result, err := tx.Exec(sqlStatement,
			user.Password,
			user.Salt,
			user.Name,
			user.Email,
			user.AdminForcedPasswordReset,
			user.GravatarURL,
			user.Position,
			user.SSOEnabled,
			user.APIOnly,
			user.GlobalRole)
		if err != nil {
			return errors.Wrap(err, "create new user")
		}

		id, _ := result.LastInsertId()
		user.ID = uint(id)

		if err := d.saveTeamsForUser(tx, user); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (d *Datastore) findUser(searchCol string, searchVal interface{}) (*fleet.User, error) {
	sqlStatement := fmt.Sprintf(
		"SELECT * FROM users "+
			"WHERE %s = ? LIMIT 1",
		searchCol,
	)

	user := &fleet.User{}

	err := d.db.Get(user, sqlStatement, searchVal)
	if err != nil && err == sql.ErrNoRows {
		return nil, notFound("User").
			WithMessage(fmt.Sprintf("with %s=%v", searchCol, searchVal))
	} else if err != nil {
		return nil, errors.Wrap(err, "find user")
	}

	if err := d.loadTeamsForUsers([]*fleet.User{user}); err != nil {
		return nil, errors.Wrap(err, "load teams")
	}

	return user, nil
}

// ListUsers lists all users with team ID, limit, sort and offset passed in with
// UserListOptions.
func (d *Datastore) ListUsers(opt fleet.UserListOptions) ([]*fleet.User, error) {
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
	sqlStatement = appendListOptionsToSQL(sqlStatement, opt.ListOptions)
	users := []*fleet.User{}

	if err := d.db.Select(&users, sqlStatement, params...); err != nil {
		return nil, errors.Wrap(err, "list users")
	}

	if err := d.loadTeamsForUsers(users); err != nil {
		return nil, errors.Wrap(err, "load teams")
	}

	return users, nil

}

func (d *Datastore) UserByEmail(email string) (*fleet.User, error) {
	return d.findUser("email", email)
}

func (d *Datastore) UserByID(id uint) (*fleet.User, error) {
	return d.findUser("id", id)
}

func (d *Datastore) SaveUser(user *fleet.User) error {
	return d.withTx(func(tx *sqlx.Tx) error {
		return d.saveUser(tx, user)
	})
}

func (d *Datastore) SaveUsers(users []*fleet.User) error {
	return d.withTx(func(tx *sqlx.Tx) error {
		for _, user := range users {
			err := d.saveUser(tx, user)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (d *Datastore) saveUser(tx *sqlx.Tx, user *fleet.User) error {
	if err := fleet.ValidateRole(user.GlobalRole, user.Teams); err != nil {
		return err
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
        api_only = ?,
		global_role = ?
      WHERE id = ?
      `
	result, err := d.db.Exec(sqlStatement,
		user.Password,
		user.Salt,
		user.Name,
		user.Email,
		user.AdminForcedPasswordReset,
		user.GravatarURL,
		user.Position,
		user.SSOEnabled,
		user.APIOnly,
		user.GlobalRole,
		user.ID)
	if err != nil {
		return errors.Wrap(err, "save user")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "rows affected save user")
	}
	if rows == 0 {
		return notFound("User").WithID(user.ID)
	}

	// REVIEW: Check if teams have been set?
	if err := d.saveTeamsForUser(tx, user); err != nil {
		return err
	}

	return nil
}

// loadTeamsForUsers will load the teams/roles for the provided users.
func (d *Datastore) loadTeamsForUsers(users []*fleet.User) error {
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
		return errors.Wrap(err, "sqlx.In loadTeamsForUsers")
	}

	var rows []struct {
		fleet.UserTeam
		UserID uint `db:"user_id"`
	}
	if err := d.db.Select(&rows, sql, args...); err != nil {
		return errors.Wrap(err, "get loadTeamsForUsers")
	}

	// Map each row to the appropriate user
	for _, r := range rows {
		user := idToUser[r.UserID]
		user.Teams = append(user.Teams, r.UserTeam)
	}

	return nil
}

func (d *Datastore) saveTeamsForUser(tx *sqlx.Tx, user *fleet.User) error {
	// Do a full teams update by deleting existing teams and then inserting all
	// the current teams in a single transaction.

	// Delete before insert
	sql := `DELETE FROM user_teams WHERE user_id = ?`
	if _, err := tx.Exec(sql, user.ID); err != nil {
		return errors.Wrap(err, "delete existing teams")
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
	if _, err := tx.Exec(sql, args...); err != nil {
		return errors.Wrap(err, "insert teams")
	}

	return nil
}

// DeleteUser deletes the associated user
func (d *Datastore) DeleteUser(id uint) error {
	return d.deleteEntity("users", id)
}
