package mysql

import (
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var teamSearchColumns = []string{"name"}

func (d *Datastore) NewTeam(team *fleet.Team) (*fleet.Team, error) {
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

	if err := d.saveTeamSecrets(team); err != nil {
		return nil, err
	}

	return team, nil
}

func (d *Datastore) Team(tid uint) (*fleet.Team, error) {
	sql := `
		SELECT * FROM teams
			WHERE id = ?
	`
	team := &fleet.Team{}

	if err := d.db.Get(team, sql, tid); err != nil {
		return nil, errors.Wrap(err, "select team")
	}

	if err := d.loadSecretsForTeams([]*fleet.Team{team}); err != nil {
		return nil, errors.Wrap(err, "getting secrets for teams")
	}

	if err := d.loadUsersForTeam(team); err != nil {
		return nil, err
	}

	return team, nil
}

func (d *Datastore) saveTeamSecrets(team *fleet.Team) error {
	if team.Secrets == nil {
		return nil
	}

	return d.ApplyEnrollSecrets(&team.ID, team.Secrets)
}

func (d *Datastore) DeleteTeam(tid uint) error {
	if err := d.deleteEntity("teams", tid); err != nil {
		return errors.Wrapf(err, "delete team id %d", tid)
	}
	return nil
}

func (d *Datastore) TeamByName(name string) (*fleet.Team, error) {
	sql := `
		SELECT * FROM teams
			WHERE name = ?
	`
	team := &fleet.Team{}

	if err := d.db.Get(team, sql, name); err != nil {
		return nil, errors.Wrap(err, "select team")
	}

	if err := d.loadSecretsForTeams([]*fleet.Team{team}); err != nil {
		return nil, errors.Wrap(err, "getting secrets for teams")
	}

	if err := d.loadUsersForTeam(team); err != nil {
		return nil, err
	}

	return team, nil
}

func (d *Datastore) loadUsersForTeam(team *fleet.Team) error {
	sql := `
		SELECT u.name, u.id, u.email, ut.role
		FROM user_teams ut JOIN users u ON (ut.user_id = u.id)
		WHERE ut.team_id = ?
	`
	rows := []fleet.TeamUser{}
	if err := d.db.Select(&rows, sql, team.ID); err != nil {
		return errors.Wrap(err, "load users for team")
	}

	team.Users = rows
	return nil
}

func (d *Datastore) saveUsersForTeam(team *fleet.Team) error {
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

func (d *Datastore) SaveTeam(team *fleet.Team) (*fleet.Team, error) {
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
// fleet.ListOptions
func (d *Datastore) ListTeams(filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
	query := fmt.Sprintf(`
			SELECT *,
				(SELECT count(*) FROM user_teams WHERE team_id = t.id) AS user_count,
				(SELECT count(*) FROM hosts WHERE team_id = t.id) AS host_count
			FROM teams t
			WHERE %s
		`,
		d.whereFilterTeams(filter, "t"),
	)
	query, params := searchLike(query, nil, opt.MatchQuery, teamSearchColumns...)
	query = appendListOptionsToSQL(query, opt)
	teams := []*fleet.Team{}
	if err := d.db.Select(&teams, query, params...); err != nil {
		return nil, errors.Wrap(err, "list teams")
	}
	if err := d.loadSecretsForTeams(teams); err != nil {
		return nil, errors.Wrap(err, "getting secrets for teams")
	}
	return teams, nil
}

func (d *Datastore) loadSecretsForTeams(teams []*fleet.Team) error {
	for _, team := range teams {
		secrets, err := d.GetEnrollSecrets(ptr.Uint(team.ID))
		if err != nil {
			return err
		}
		team.Secrets = secrets
	}
	return nil
}

func (d *Datastore) SearchTeams(filter fleet.TeamFilter, matchQuery string, omit ...uint) ([]*fleet.Team, error) {
	sql := fmt.Sprintf(`
			SELECT *,
				(SELECT count(*) FROM user_teams WHERE team_id = t.id) AS user_count,
				(SELECT count(*) FROM hosts WHERE team_id = t.id) AS host_count
			FROM teams t
			WHERE %s AND %s
		`,
		d.whereOmitIDs("t.id", omit),
		d.whereFilterTeams(filter, "t"),
	)
	sql, params := searchLike(sql, nil, matchQuery, teamSearchColumns...)
	sql += "\nLIMIT 5"
	teams := []*fleet.Team{}
	if err := d.db.Select(&teams, sql, params...); err != nil {
		return nil, errors.Wrap(err, "search teams")
	}
	if err := d.loadSecretsForTeams(teams); err != nil {
		return nil, errors.Wrap(err, "getting secrets for teams")
	}
	return teams, nil
}

func (d *Datastore) TeamEnrollSecrets(teamID uint) ([]*fleet.EnrollSecret, error) {
	sql := `
		SELECT * FROM enroll_secrets
		WHERE team_id = ?
	`
	var secrets []*fleet.EnrollSecret
	if err := d.db.Select(&secrets, sql, teamID); err != nil {
		return nil, errors.Wrap(err, "get secrets")
	}
	return secrets, nil
}
