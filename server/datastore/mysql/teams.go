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
      description,
      config
    ) VALUES (?, ?, ?)
    `
		result, err := tx.ExecContext(
			ctx,
			query,
			team.Name,
			team.Description,
			team.Config,
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
	return teamDB(ctx, ds.reader(ctx), tid)
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
	if err := loadHostCountForTeamDB(ctx, q, team); err != nil {
		return nil, err
	}
	if err := loadFeaturesForTeamDB(ctx, q, team); err != nil {
		return nil, err
	}

	return team, nil
}

func saveTeamSecretsDB(ctx context.Context, q sqlx.ExtContext, team *fleet.Team) error {
	if team.Secrets == nil {
		return nil
	}

	return applyEnrollSecretsDB(ctx, q, &team.ID, team.Secrets)
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

		_, err = tx.ExecContext(ctx, `DELETE FROM mdm_apple_configuration_profiles WHERE team_id=?`, tid)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "deleting mdm_apple_configuration_profiles for team %d", tid)
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM mdm_windows_configuration_profiles WHERE team_id=?`, tid)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "deleting mdm_windows_configuration_profiles for team %d", tid)
		}

		return nil
	})
}

func (ds *Datastore) TeamByName(ctx context.Context, name string) (*fleet.Team, error) {
	stmt := `
		SELECT * FROM teams
			WHERE name = ?
	`
	team := &fleet.Team{}

	if err := sqlx.GetContext(ctx, ds.reader(ctx), team, stmt, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Team").WithName(name))
		}
		return nil, ctxerr.Wrap(ctx, err, "select team")
	}

	if err := loadSecretsForTeamsDB(ctx, ds.reader(ctx), []*fleet.Team{team}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting secrets for teams")
	}

	if err := loadUsersForTeamDB(ctx, ds.reader(ctx), team); err != nil {
		return nil, err
	}
	if err := loadHostCountForTeamDB(ctx, ds.reader(ctx), team); err != nil {
		return nil, err
	}
	if err := loadFeaturesForTeamDB(ctx, ds.reader(ctx), team); err != nil {
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
	team.UserCount = len(rows)

	return nil
}

func loadHostCountForTeamDB(ctx context.Context, q sqlx.QueryerContext, team *fleet.Team) error {
	stmt := `
		SELECT COUNT(*) FROM hosts h
		WHERE h.team_id = ?
	`
	if err := q.QueryRowxContext(ctx, stmt, team.ID).Scan(&team.HostCount); err != nil {
		return ctxerr.Wrap(ctx, err, "load HostsCount for team")
	}

	return nil
}

func loadFeaturesForTeamDB(ctx context.Context, q sqlx.QueryerContext, team *fleet.Team) error {
	features, err := teamFeaturesDB(ctx, q, team.ID)
	if err != nil {
		return err
	}
	team.Config.Features = *features
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
UPDATE teams
SET
    name = ?,
    description = ?,
    config = ?
WHERE
    id = ?
`
		_, err := tx.ExecContext(ctx, query, team.Name, team.Description, team.Config, team.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "saving team")
		}

		if err := saveUsersForTeamDB(ctx, tx, team); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return team, nil
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
	query, params = appendListOptionsWithCursorToSQL(query, params, &opt)
	teams := []*fleet.Team{}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &teams, query, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list teams")
	}
	if err := loadSecretsForTeamsDB(ctx, ds.reader(ctx), teams); err != nil {
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
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &teamsSummary, "SELECT id, name, description FROM teams"); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "teams summary")
	}
	return teamsSummary, nil
}

func (ds *Datastore) TeamExists(ctx context.Context, teamID uint) (bool, error) {
	var exists bool
	err := ds.writer(ctx).GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM teams WHERE id = ?)", teamID)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "team exists")
	}
	return exists, nil
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
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &teams, sql, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "search teams")
	}
	if err := loadSecretsForTeamsDB(ctx, ds.reader(ctx), teams); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting secrets for teams")
	}
	return teams, nil
}

func (ds *Datastore) TeamEnrollSecrets(ctx context.Context, teamID uint) ([]*fleet.EnrollSecret, error) {
	sql := `
		SELECT secret, team_id, created_at FROM enroll_secrets
		WHERE team_id = ?
	`
	var secrets []*fleet.EnrollSecret
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &secrets, sql, teamID); err != nil {
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
	sql := `SELECT config->'$.agent_options' FROM teams WHERE id = ?`
	var agentOptions *json.RawMessage
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &agentOptions, sql, tid); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select team")
	}
	return agentOptions, nil
}

// TeamFeatures loads the features enabled for a team.
func (ds *Datastore) TeamFeatures(ctx context.Context, tid uint) (*fleet.Features, error) {
	return teamFeaturesDB(ctx, ds.reader(ctx), tid)
}

func teamFeaturesDB(ctx context.Context, q sqlx.QueryerContext, tid uint) (*fleet.Features, error) {
	sql := `SELECT config->'$.features' as features FROM teams WHERE id = ?`
	var raw *json.RawMessage
	if err := sqlx.GetContext(ctx, q, &raw, sql, tid); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get team config features")
	}

	var features fleet.Features
	features.ApplyDefaultsForNewInstalls()
	if raw != nil {
		if err := json.Unmarshal(*raw, &features); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "unmarshal team config features")
		}
	}
	return &features, nil
}

func (ds *Datastore) TeamMDMConfig(ctx context.Context, tid uint) (*fleet.TeamMDM, error) {
	sql := `SELECT config->'$.mdm' AS mdm FROM teams WHERE id = ?`
	var raw *json.RawMessage
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &raw, sql, tid); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select team MDM config")
	}
	var mdmConfig *fleet.TeamMDM
	if raw != nil {
		if err := json.Unmarshal(*raw, &mdmConfig); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "unmarshal team MDM config")
		}
	}
	return mdmConfig, nil
}

// DeleteIntegrationsFromTeams removes the deleted integrations from any team
// that uses it.
func (ds *Datastore) DeleteIntegrationsFromTeams(ctx context.Context, deletedIntgs fleet.Integrations) error {
	const (
		listTeams  = `SELECT id, config FROM teams WHERE config IS NOT NULL`
		updateTeam = `UPDATE teams SET config = ? WHERE id = ?`
	)

	rows, err := ds.writer(ctx).QueryxContext(ctx, listTeams)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "query teams")
	}
	defer rows.Close()

	for rows.Next() {
		var tm fleet.Team
		if err := rows.StructScan(&tm); err != nil {
			return ctxerr.Wrap(ctx, err, "scan team row in struct")
		}

		// ignore errors, it's ok for some integrations to not match with the
		// batch of deleted integrations, we're only interested in knowing if
		// some did match.
		if matches, _ := tm.Config.Integrations.MatchWithIntegrations(deletedIntgs); len(matches.Jira)+len(matches.Zendesk) > 0 {
			delJira, _ := fleet.IndexJiraIntegrations(matches.Jira)
			delZendesk, _ := fleet.IndexZendeskIntegrations(matches.Zendesk)

			var keepJira []*fleet.TeamJiraIntegration
			for _, tmIntg := range tm.Config.Integrations.Jira {
				if _, ok := delJira[tmIntg.UniqueKey()]; !ok {
					keepJira = append(keepJira, tmIntg)
				}
			}

			var keepZendesk []*fleet.TeamZendeskIntegration
			for _, tmIntg := range tm.Config.Integrations.Zendesk {
				if _, ok := delZendesk[tmIntg.UniqueKey()]; !ok {
					keepZendesk = append(keepZendesk, tmIntg)
				}
			}

			tm.Config.Integrations.Jira = keepJira
			tm.Config.Integrations.Zendesk = keepZendesk
			if _, err := ds.writer(ctx).ExecContext(ctx, updateTeam, tm.Config, tm.ID); err != nil {
				return ctxerr.Wrap(ctx, err, "update team config")
			}
		}
	}
	return rows.Err()
}
