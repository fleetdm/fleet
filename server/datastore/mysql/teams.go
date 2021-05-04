package mysql

import (
	"strings"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var teamSearchColumns = []string{"name"}

func (d *Datastore) NewTeam(team *kolide.Team) (*kolide.Team, error) {
	query := `
	INSERT INTO teams (
		name,
		agent_options,
		description
	) VALUES ( ?, ?, ? )
	`
	result, err := d.db.Exec(
		query,
		team.Name,
		team.AgentOptions,
		team.Description,
	)
	if err != nil {
		return nil, errors.Wrap(err, "insert team")
	}

	id, _ := result.LastInsertId()
	team.ID = uint(id)
	return team, nil
}

func (d *Datastore) Team(tid uint) (*kolide.Team, error) {
	sql := `
		SELECT * FROM teams
			WHERE id = ?
	`
	team := &kolide.Team{}

	if err := d.db.Get(team, sql, tid); err != nil {
		return nil, errors.Wrap(err, "select team")
	}

	if err := d.loadUsersForTeam(team); err != nil {
		return nil, err
	}

	return team, nil
}

func (d *Datastore) DeleteTeam(tid uint) error {
	if err := d.deleteEntity("teams", tid); err != nil {
		return errors.Wrapf(err, "delete team id %d", tid)
	}
	return nil
}

func (d *Datastore) TeamByName(name string) (*kolide.Team, error) {
	sql := `
		SELECT * FROM teams
			WHERE name = ?
	`
	team := &kolide.Team{}

	if err := d.db.Get(team, sql, name); err != nil {
		return nil, errors.Wrap(err, "select team")
	}

	if err := d.loadUsersForTeam(team); err != nil {
		return nil, err
	}

	return team, nil
}

func (d *Datastore) loadUsersForTeam(team *kolide.Team) error {
	sql := `
		SELECT u.name, u.id, u.email, ut.role
		FROM user_teams ut JOIN users u ON (ut.user_id = u.id)
		WHERE ut.team_id = ?
	`
	rows := []kolide.TeamUser{}
	if err := d.db.Select(&rows, sql, team.ID); err != nil {
		return errors.Wrap(err, "load users for team")
	}

	team.Users = rows
	return nil
}

func (d *Datastore) saveUsersForTeam(team *kolide.Team) error {
	// Do a full user update by deleting existing users and then inserting all
	// the current users in a single transaction.
	if err := d.withRetryTxx(func(tx *sqlx.Tx) error {
		// Delete before insert
		sql := `DELETE FROM user_teams WHERE team_id = ?`
		if _, err := tx.Exec(sql, team.ID); err != nil {
			return errors.Wrap(err, "delete existing users")
		}

		if len(team.Users) == 0 {
			return nil
		}

		// Bulk insert
		const valueStr = "(?,?,?),"
		var args []interface{}
		for _, teamUser := range team.Users {
			args = append(args, teamUser.User.ID, team.ID, teamUser.Role)
		}
		sql = "INSERT INTO user_teams (user_id, team_id, role) VALUES " +
			strings.Repeat(valueStr, len(team.Users))
		sql = strings.TrimSuffix(sql, ",")
		if _, err := tx.Exec(sql, args...); err != nil {
			return errors.Wrap(err, "insert users")
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "save users for team")
	}

	return nil
}

func (d *Datastore) SaveTeam(team *kolide.Team) (*kolide.Team, error) {
	query := `
		UPDATE teams SET
			name = ?,
			agent_options = ?,
			description = ?
		WHERE id = ?
	`
	_, err := d.db.Exec(query, team.Name, team.AgentOptions, team.Description, team.ID)
	if err != nil {
		return nil, errors.Wrap(err, "saving team")
	}

	if err := d.saveUsersForTeam(team); err != nil {
		return nil, err
	}

	return team, nil
}

// ListTeams lists all teams with limit, sort and offset passed in with
// kolide.ListOptions
func (d *Datastore) ListTeams(opt kolide.ListOptions) ([]*kolide.Team, error) {
	query := `
		SELECT *,
			(SELECT count(*) FROM user_teams WHERE team_id = id) AS user_count,
			(SELECT count(*) FROM hosts WHERE team_id = id) AS host_count
		FROM teams
		WHERE TRUE
	`
	query, params := searchLike(query, nil, opt.MatchQuery, teamSearchColumns...)
	query = appendListOptionsToSQL(query, opt)
	teams := []*kolide.Team{}
	if err := d.db.Select(&teams, query, params...); err != nil {
		return nil, errors.Wrap(err, "list teams")
	}
	return teams, nil

}
