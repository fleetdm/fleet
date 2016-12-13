package mysql

import (
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

func (d *Datastore) NewScheduledQuery(sq *kolide.ScheduledQuery) (*kolide.ScheduledQuery, error) {
	sql := `
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
	result, err := d.db.Exec(sql, sq.PackID, sq.QueryID, sq.Snapshot, sq.Removed, sq.Interval, sq.Platform, sq.Version, sq.Shard)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	id, _ := result.LastInsertId()
	sq.ID = uint(id)
	return sq, nil
}

func (d *Datastore) SaveScheduledQuery(sq *kolide.ScheduledQuery) (*kolide.ScheduledQuery, error) {
	sql := `
		UPDATE scheduled_queries
			SET pack_id = ?, query_id = ?, ` + "`interval`" + ` = ?, snapshot = ?, removed = ?, platform = ?, version = ?, shard = ?
			WHERE id = ? AND NOT deleted
	`
	_, err := d.db.Exec(sql, sq.PackID, sq.QueryID, sq.Interval, sq.Snapshot, sq.Removed, sq.Platform, sq.Version, sq.Shard, sq.ID)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return sq, nil
}

func (d *Datastore) DeleteScheduledQuery(id uint) error {
	sql := `
		UPDATE scheduled_queries
			SET deleted_at = ?, deleted = ?
			WHERE id = ?
	`
	_, err := d.db.Exec(sql, d.clock.Now(), true, id)
	if err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

func (d *Datastore) ScheduledQuery(id uint) (*kolide.ScheduledQuery, error) {
	sql := `
		SELECT sq.*, q.query, q.name
		FROM scheduled_queries sq
		JOIN queries q
		ON sq.query_id = q.id
		WHERE sq.id = ?
		AND NOT sq.deleted
	`
	sq := &kolide.ScheduledQuery{}
	if err := d.db.Get(sq, sql, id); err != nil {
		return nil, errors.DatabaseError(err)
	}

	return sq, nil
}

func (d *Datastore) ListScheduledQueriesInPack(id uint, opts kolide.ListOptions) ([]*kolide.ScheduledQuery, error) {
	sql := `
		SELECT sq.*, q.query, q.name
		FROM scheduled_queries sq
		JOIN queries q
		ON sq.query_id = q.id
		WHERE sq.pack_id = ?
		AND NOT sq.deleted
	`
	sql = appendListOptionsToSQL(sql, opts)
	results := []*kolide.ScheduledQuery{}

	if err := d.db.Select(&results, sql, id); err != nil {
		return nil, errors.DatabaseError(err)
	}

	return results, nil
}
