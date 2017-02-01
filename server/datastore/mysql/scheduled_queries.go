package mysql

import (
	"database/sql"

	"github.com/kolide/kolide/server/kolide"
	"github.com/pkg/errors"
)

func (d *Datastore) NewScheduledQuery(sq *kolide.ScheduledQuery) (*kolide.ScheduledQuery, error) {
	query := `
	    INSERT INTO scheduled_queries (
			pack_id,
			query_id,
			snapshot,
			removed,
			` + "`interval`" + `,
			platform,
			version,
			shard
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`
	result, err := d.db.Exec(query, sq.PackID, sq.QueryID, sq.Snapshot, sq.Removed, sq.Interval, sq.Platform, sq.Version, sq.Shard)
	if err != nil {
		return nil, errors.Wrap(err, "inserting scheduled query")
	}

	id, _ := result.LastInsertId()
	sq.ID = uint(id)

	query = `SELECT query, name FROM queries WHERE id = ? LIMIT 1`
	metadata := []struct {
		Query string
		Name  string
	}{}

	err = d.db.Select(&metadata, query, sq.QueryID)
	if err != nil && err == sql.ErrNoRows {
		return nil, notFound("Query").WithID(sq.QueryID)
	} else if err != nil {
		return nil, errors.Wrap(err, "select query by ID")
	}

	if len(metadata) != 1 {
		return nil, errors.Wrap(err, "wrong number of results returned from database")
	}

	sq.Query = metadata[0].Query
	sq.Name = metadata[0].Name

	return sq, nil
}

func (d *Datastore) SaveScheduledQuery(sq *kolide.ScheduledQuery) (*kolide.ScheduledQuery, error) {
	query := `
		UPDATE scheduled_queries
			SET pack_id = ?, query_id = ?, ` + "`interval`" + ` = ?, snapshot = ?, removed = ?, platform = ?, version = ?, shard = ?
			WHERE id = ? AND NOT deleted
	`
	_, err := d.db.Exec(query, sq.PackID, sq.QueryID, sq.Interval, sq.Snapshot, sq.Removed, sq.Platform, sq.Version, sq.Shard, sq.ID)
	if err != nil {
		return nil, errors.Wrap(err, "saving a scheduled query")
	}

	return sq, nil
}

func (d *Datastore) DeleteScheduledQuery(id uint) error {
	return d.deleteEntity("scheduled_queries", id)
}

func (d *Datastore) ScheduledQuery(id uint) (*kolide.ScheduledQuery, error) {
	query := `
		SELECT sq.*, q.query, q.name
		FROM scheduled_queries sq
		JOIN queries q
		ON sq.query_id = q.id
		WHERE sq.id = ?
		AND NOT sq.deleted
	`
	sq := &kolide.ScheduledQuery{}
	if err := d.db.Get(sq, query, id); err != nil {
		return nil, errors.Wrap(err, "selecting a scheduled query")
	}

	return sq, nil
}

func (d *Datastore) ListScheduledQueriesInPack(id uint, opts kolide.ListOptions) ([]*kolide.ScheduledQuery, error) {
	query := `
		SELECT sq.*, q.query, q.name
		FROM scheduled_queries sq
		JOIN queries q
		ON sq.query_id = q.id
		WHERE sq.pack_id = ?
		AND NOT sq.deleted
	`
	query = appendListOptionsToSQL(query, opts)
	results := []*kolide.ScheduledQuery{}

	if err := d.db.Select(&results, query, id); err != nil {
		return nil, errors.Wrap(err, "listing scheduled queries")
	}

	return results, nil
}
