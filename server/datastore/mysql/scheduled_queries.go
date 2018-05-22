package mysql

import (
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (d *Datastore) ListScheduledQueriesInPack(id uint, opts kolide.ListOptions) ([]*kolide.ScheduledQuery, error) {
	query := `
		SELECT
			sq.id,
			sq.pack_id,
			COALESCE(sq.name, q.name) AS name,
			sq.query_name,
			COALESCE(sq.description, '') AS description,
			sq.interval,
			sq.snapshot,
			sq.removed,
			COALESCE(sq.platform, '') AS platform,
			sq.version,
			sq.shard,
			q.query,
			q.id AS query_id
		FROM scheduled_queries sq
		JOIN queries q
		ON sq.query_name = q.name
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
