package mysql

import (
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
)

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
