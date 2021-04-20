package mysql

import (
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
)

var teamSearchColumns = []string{"name"}

func (d *Datastore) NewTeam(team *kolide.Team) (*kolide.Team, error) {
	query := `
	INSERT INTO teams (
		name,
		description
	) VALUES ( ?, ? )
	`
	result, err := d.db.Exec(
		query,
		team.Name,
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

	return team, nil
}

func (d *Datastore) SaveTeam(team *kolide.Team) (*kolide.Team, error) {
	query := `
		UPDATE teams SET
			name = ?,
			description = ?
		WHERE id = ?
	`
	_, err := d.db.Exec(query, team.Name, team.Description, team.ID)
	if err != nil {
		return nil, errors.Wrap(err, "saving team")
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
