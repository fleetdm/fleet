package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var teamSearchColumns = []string{"name"}

func (d *Datastore) NewTeam(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
	err := d.withRetryTxx(func(tx *sqlx.Tx) error {
		query := `
    INSERT INTO teams (
      name,
      agent_options,
      description
    ) VALUES ( ?, ?, ? )
    `
		result, err := tx.Exec(
			query,
			team.Name,
			team.AgentOptions,
			team.Description,
		)
		if err != nil {
			return errors.Wrap(err, "insert team")
		}

		id, _ := result.LastInsertId()
		team.ID = uint(id)

		return saveTeamSecretsDB(tx, team)
	})

	if err != nil {
		return nil, err
	}
	return team, nil
}

func (d *Datastore) Team(ctx context.Context, tid uint) (*fleet.Team, error) {
	return teamDB(d.reader, tid)
}

func teamDB(q sqlx.Queryer, tid uint) (*fleet.Team, error) {
	sql := `
		SELECT * FROM teams
			WHERE id = ?
	`
	team := &fleet.Team{}

	if err := sqlx.Get(q, team, sql, tid); err != nil {
		return nil, errors.Wrap(err, "select team")
	}

	if err := loadSecretsForTeamsDB(q, []*fleet.Team{team}); err != nil {
		return nil, errors.Wrap(err, "getting secrets for teams")
	}

	if err := loadUsersForTeamDB(q, team); err != nil {
		return nil, err
	}

	return team, nil
}

func saveTeamSecretsDB(exec sqlx.Execer, team *fleet.Team) error {
	if team.Secrets == nil {
		return nil
	}

	return applyEnrollSecretsDB(exec, &team.ID, team.Secrets)
}

func (d *Datastore) DeleteTeam(ctx context.Context, tid uint) error {
	if err := d.deleteEntity("teams", tid); err != nil {
		return errors.Wrapf(err, "delete team id %d", tid)
	}
	return nil
}

func (d *Datastore) TeamByName(ctx context.Context, name string) (*fleet.Team, error) {
	sql := `
		SELECT * FROM teams
			WHERE name = ?
	`
	team := &fleet.Team{}

	if err := d.reader.Get(team, sql, name); err != nil {
		return nil, errors.Wrap(err, "select team")
	}

	if err := loadSecretsForTeamsDB(d.reader, []*fleet.Team{team}); err != nil {
		return nil, errors.Wrap(err, "getting secrets for teams")
	}

	if err := loadUsersForTeamDB(d.reader, team); err != nil {
		return nil, err
	}

	return team, nil
}

func loadUsersForTeamDB(q sqlx.Queryer, team *fleet.Team) error {
	sql := `
		SELECT u.name, u.id, u.email, ut.role
		FROM user_teams ut JOIN users u ON (ut.user_id = u.id)
		WHERE ut.team_id = ?
	`
	rows := []fleet.TeamUser{}
	if err := sqlx.Select(q, &rows, sql, team.ID); err != nil {
		return errors.Wrap(err, "load users for team")
	}

	team.Users = rows
	return nil
}

func saveUsersForTeamDB(exec sqlx.Execer, team *fleet.Team) error {
	// Do a full user update by deleting existing users and then inserting all
	// the current users in a single transaction.
	// Delete before insert
	sql := `DELETE FROM user_teams WHERE team_id = ?`
	if _, err := exec.Exec(sql, team.ID); err != nil {
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
	if _, err := exec.Exec(sql, args...); err != nil {
		return errors.Wrap(err, "insert users")
	}

	return nil
}

func (d *Datastore) SaveTeam(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
	err := d.withRetryTxx(func(tx *sqlx.Tx) error {
		query := `
		UPDATE teams SET
			name = ?,
			agent_options = ?,
			description = ?
		WHERE id = ?
	`
		_, err := tx.Exec(query, team.Name, team.AgentOptions, team.Description, team.ID)
		if err != nil {
			return errors.Wrap(err, "saving team")
		}

		if err := saveUsersForTeamDB(tx, team); err != nil {
			return err
		}

		return updateTeamScheduleDB(tx, team)
	})

	if err != nil {
		return nil, err
	}
	return team, nil
}

func updateTeamScheduleDB(exec sqlx.Execer, team *fleet.Team) error {
	_, err := exec.Exec(
		`UPDATE packs SET name = ? WHERE pack_type = ?`, teamScheduleName(team), teamSchedulePackType(team),
	)
	return err
}

// ListTeams lists all teams with limit, sort and offset passed in with
// fleet.ListOptions
func (d *Datastore) ListTeams(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
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
	if err := d.reader.Select(&teams, query, params...); err != nil {
		return nil, errors.Wrap(err, "list teams")
	}
	if err := loadSecretsForTeamsDB(d.reader, teams); err != nil {
		return nil, errors.Wrap(err, "getting secrets for teams")
	}
	return teams, nil
}

func loadSecretsForTeamsDB(q sqlx.Queryer, teams []*fleet.Team) error {
	for _, team := range teams {
		secrets, err := getEnrollSecretsDB(q, ptr.Uint(team.ID))
		if err != nil {
			return err
		}
		team.Secrets = secrets
	}
	return nil
}

func (d *Datastore) SearchTeams(ctx context.Context, filter fleet.TeamFilter, matchQuery string, omit ...uint) ([]*fleet.Team, error) {
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
	if err := d.reader.Select(&teams, sql, params...); err != nil {
		return nil, errors.Wrap(err, "search teams")
	}
	if err := loadSecretsForTeamsDB(d.reader, teams); err != nil {
		return nil, errors.Wrap(err, "getting secrets for teams")
	}
	return teams, nil
}

func (d *Datastore) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*fleet.EnrollSecret, error) {
	sql := `
		SELECT * FROM enroll_secrets
		WHERE team_id = ?
	`
	var secrets []*fleet.EnrollSecret
	if err := d.reader.Select(&secrets, sql, teamID); err != nil {
		return nil, errors.Wrap(err, "get secrets")
	}
	return secrets, nil
}
