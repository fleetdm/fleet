package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
)

var teamSearchColumns = []string{"name"}

func (d *Datastore) NewTeam(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
	err := d.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		query := `
    INSERT INTO teams (
      name,
      agent_options,
      description
    ) VALUES ( ?, ?, ? )
    `
		result, err := tx.ExecContext(
			ctx,
			query,
			team.Name,
			team.AgentOptions,
			team.Description,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert team")
		}

		id, _ := result.LastInsertId()
		team.ID = uint(id)

		return saveTeamSecretsDB(ctx, tx, team)
	})
	if err != nil {
		return nil, err
	}
	return team, nil
}

func (d *Datastore) Team(ctx context.Context, tid uint) (*fleet.Team, error) {
	return teamDB(ctx, d.reader, tid)
}

func teamDB(ctx context.Context, q sqlx.QueryerContext, tid uint) (*fleet.Team, error) {
	sql := `
		SELECT * FROM teams
			WHERE id = ?
	`
	team := &fleet.Team{}

	if err := sqlx.GetContext(ctx, q, team, sql, tid); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select team")
	}

	if err := loadSecretsForTeamsDB(ctx, q, []*fleet.Team{team}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting secrets for teams")
	}

	if err := loadUsersForTeamDB(ctx, q, team); err != nil {
		return nil, err
	}

	return team, nil
}

func saveTeamSecretsDB(ctx context.Context, exec sqlx.ExecerContext, team *fleet.Team) error {
	if team.Secrets == nil {
		return nil
	}

	return applyEnrollSecretsDB(ctx, exec, &team.ID, team.Secrets)
}

func (d *Datastore) DeleteTeam(ctx context.Context, tid uint) error {
	if err := d.deleteEntity(ctx, teamsTable, tid); err != nil {
		return ctxerr.Wrapf(ctx, err, "delete team id %d", tid)
	}
	return nil
}

func (d *Datastore) TeamByName(ctx context.Context, name string) (*fleet.Team, error) {
	sql := `
		SELECT * FROM teams
			WHERE name = ?
	`
	team := &fleet.Team{}

	if err := sqlx.GetContext(ctx, d.reader, team, sql, name); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select team")
	}

	if err := loadSecretsForTeamsDB(ctx, d.reader, []*fleet.Team{team}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting secrets for teams")
	}

	if err := loadUsersForTeamDB(ctx, d.reader, team); err != nil {
		return nil, err
	}

	return team, nil
}

func loadUsersForTeamDB(ctx context.Context, q sqlx.QueryerContext, team *fleet.Team) error {
	sql := `
		SELECT u.name, u.id, u.email, ut.role
		FROM user_teams ut JOIN users u ON (ut.user_id = u.id)
		WHERE ut.team_id = ?
	`
	rows := []fleet.TeamUser{}
	if err := sqlx.SelectContext(ctx, q, &rows, sql, team.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "load users for team")
	}

	team.Users = rows
	return nil
}

func saveUsersForTeamDB(ctx context.Context, exec sqlx.ExecerContext, team *fleet.Team) error {
	// Do a full user update by deleting existing users and then inserting all
	// the current users in a single transaction.
	// Delete before insert
	sql := `DELETE FROM user_teams WHERE team_id = ?`
	if _, err := exec.ExecContext(ctx, sql, team.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "delete existing users")
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
	if _, err := exec.ExecContext(ctx, sql, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert users")
	}

	return nil
}

func (d *Datastore) SaveTeam(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
	err := d.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		query := `
		UPDATE teams SET
			name = ?,
			agent_options = ?,
			description = ?
		WHERE id = ?
	`
		_, err := tx.ExecContext(ctx, query, team.Name, team.AgentOptions, team.Description, team.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "saving team")
		}

		if err := saveUsersForTeamDB(ctx, tx, team); err != nil {
			return err
		}

		return updateTeamScheduleDB(ctx, tx, team)
	})
	if err != nil {
		return nil, err
	}
	return team, nil
}

func updateTeamScheduleDB(ctx context.Context, exec sqlx.ExecerContext, team *fleet.Team) error {
	_, err := exec.ExecContext(ctx,
		`UPDATE packs SET name = ? WHERE pack_type = ?`, teamScheduleName(team), teamSchedulePackType(team),
	)
	return ctxerr.Wrap(ctx, err, "update packs")
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
	if err := sqlx.SelectContext(ctx, d.reader, &teams, query, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list teams")
	}
	if err := loadSecretsForTeamsDB(ctx, d.reader, teams); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting secrets for teams")
	}
	return teams, nil
}

func loadSecretsForTeamsDB(ctx context.Context, q sqlx.QueryerContext, teams []*fleet.Team) error {
	for _, team := range teams {
		secrets, err := getEnrollSecretsDB(ctx, q, ptr.Uint(team.ID))
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
	teams := []*fleet.Team{}
	if err := sqlx.SelectContext(ctx, d.reader, &teams, sql, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "search teams")
	}
	if err := loadSecretsForTeamsDB(ctx, d.reader, teams); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting secrets for teams")
	}
	return teams, nil
}

func (d *Datastore) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*fleet.EnrollSecret, error) {
	sql := `
		SELECT * FROM enroll_secrets
		WHERE team_id = ?
	`
	var secrets []*fleet.EnrollSecret
	if err := sqlx.SelectContext(ctx, d.reader, &secrets, sql, teamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get secrets")
	}
	return secrets, nil
}

func amountTeamsDB(db sqlx.Queryer) (int, error) {
	var amount int
	err := sqlx.Get(db, &amount, `SELECT count(*) FROM teams`)
	if err != nil {
		return 0, err
	}
	return amount, nil
}
