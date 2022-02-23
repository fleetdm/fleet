package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
)

var teamSearchColumns = []string{"name"}

func (ds *Datastore) NewTeam(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
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

func (ds *Datastore) Team(ctx context.Context, tid uint) (*fleet.Team, error) {
	return teamDB(ctx, ds.reader, tid)
}

func teamDB(ctx context.Context, q sqlx.QueryerContext, tid uint) (*fleet.Team, error) {
	stmt := `
		SELECT * FROM teams
			WHERE id = ?
	`
	team := &fleet.Team{}

	if err := sqlx.GetContext(ctx, q, team, stmt, tid); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Team").WithID(tid))
		}
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

func (ds *Datastore) DeleteTeam(ctx context.Context, tid uint) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM teams WHERE id = ?`, tid)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "delete team %d", tid)
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM pack_targets WHERE type=? AND target_id=?`, fleet.TargetTeam, tid)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "deleting pack_targets for team %d", tid)
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM packs WHERE pack_type=?`, teamSchedulePackTypeByID(tid))
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "deleting team global packs for team %d", tid)
		}

		return nil
	})
}

func (ds *Datastore) TeamByName(ctx context.Context, name string) (*fleet.Team, error) {
	sql := `
		SELECT * FROM teams
			WHERE name = ?
	`
	team := &fleet.Team{}

	if err := sqlx.GetContext(ctx, ds.reader, team, sql, name); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select team")
	}

	if err := loadSecretsForTeamsDB(ctx, ds.reader, []*fleet.Team{team}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting secrets for teams")
	}

	if err := loadUsersForTeamDB(ctx, ds.reader, team); err != nil {
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

func (ds *Datastore) SaveTeam(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
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
func (ds *Datastore) ListTeams(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
	query := fmt.Sprintf(`
			SELECT *,
				(SELECT count(*) FROM user_teams WHERE team_id = t.id) AS user_count,
				(SELECT count(*) FROM hosts WHERE team_id = t.id) AS host_count
			FROM teams t
			WHERE %s
		`,
		ds.whereFilterTeams(filter, "t"),
	)
	query, params := searchLike(query, nil, opt.MatchQuery, teamSearchColumns...)
	query = appendListOptionsToSQL(query, opt)
	teams := []*fleet.Team{}
	if err := sqlx.SelectContext(ctx, ds.reader, &teams, query, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list teams")
	}
	if err := loadSecretsForTeamsDB(ctx, ds.reader, teams); err != nil {
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

func (ds *Datastore) TeamsSummary(ctx context.Context) ([]*fleet.TeamSummary, error) {
	teamsSummary := []*fleet.TeamSummary{}
	if err := sqlx.SelectContext(ctx, ds.reader, &teamsSummary, "SELECT id, name, description FROM teams"); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "teams summary")
	}
	return teamsSummary, nil
}

func (ds *Datastore) SearchTeams(ctx context.Context, filter fleet.TeamFilter, matchQuery string, omit ...uint) ([]*fleet.Team, error) {
	sql := fmt.Sprintf(`
			SELECT *,
				(SELECT count(*) FROM user_teams WHERE team_id = t.id) AS user_count,
				(SELECT count(*) FROM hosts WHERE team_id = t.id) AS host_count
			FROM teams t
			WHERE %s AND %s
		`,
		ds.whereOmitIDs("t.id", omit),
		ds.whereFilterTeams(filter, "t"),
	)
	sql, params := searchLike(sql, nil, matchQuery, teamSearchColumns...)
	teams := []*fleet.Team{}
	if err := sqlx.SelectContext(ctx, ds.reader, &teams, sql, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "search teams")
	}
	if err := loadSecretsForTeamsDB(ctx, ds.reader, teams); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting secrets for teams")
	}
	return teams, nil
}

func (ds *Datastore) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*fleet.EnrollSecret, error) {
	sql := `
		SELECT * FROM enroll_secrets
		WHERE team_id = ?
	`
	var secrets []*fleet.EnrollSecret
	if err := sqlx.SelectContext(ctx, ds.reader, &secrets, sql, teamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get secrets")
	}
	return secrets, nil
}

func amountTeamsDB(ctx context.Context, db sqlx.QueryerContext) (int, error) {
	var amount int
	err := sqlx.GetContext(ctx, db, &amount, `SELECT count(*) FROM teams`)
	if err != nil {
		return 0, err
	}
	return amount, nil
}

// TeamAgentOptions loads the agents options of a team.
func (ds *Datastore) TeamAgentOptions(ctx context.Context, tid uint) (*json.RawMessage, error) {
	sql := `SELECT agent_options FROM teams WHERE id = ?`
	var agentOptions *json.RawMessage
	if err := sqlx.GetContext(ctx, ds.reader, &agentOptions, sql, tid); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select team")
	}
	return agentOptions, nil
}
