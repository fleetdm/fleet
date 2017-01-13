package mysql

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/pkg/errors"
)

func (d *Datastore) QueryByName(name string) (*kolide.Query, bool, error) {
	sqlStatement := `
		SELECT *
			FROM queries
			WHERE name = ? AND NOT deleted
	`
	var query kolide.Query
	err := d.db.Get(&query, sqlStatement, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, errors.Wrap(err, "selecting query by name")
	}
	return &query, true, nil
}

// NewQuery creates a Query
func (d *Datastore) NewQuery(query *kolide.Query) (*kolide.Query, error) {
	sql := `
		INSERT INTO queries (
			name,
			description,
			query,
			saved,
			author_id
		) VALUES ( ?, ?, ?, ?, ? )
	`
	result, err := d.db.Exec(sql, query.Name, query.Description, query.Query, query.Saved, query.AuthorID)
	if err != nil {
		return nil, errors.Wrap(err, "inserting new query")
	}

	id, _ := result.LastInsertId()
	query.ID = uint(id)
	query.Packs = []kolide.Pack{}
	return query, nil
}

// SaveQuery saves changes to a Query.
func (d *Datastore) SaveQuery(q *kolide.Query) error {
	sql := `
		UPDATE queries
			SET name = ?, description = ?, query = ?, author_id = ?, saved = ?
			WHERE id = ? AND NOT deleted
	`
	_, err := d.db.Exec(sql, q.Name, q.Description, q.Query, q.AuthorID, q.Saved, q.ID)
	if err != nil {
		return errors.Wrap(err, "updating query")
	}

	return nil
}

// DeleteQuery soft deletes Query identified by Query.ID
func (d *Datastore) DeleteQuery(qid uint) error {
	return d.deleteEntity("queries", qid)
}

// DeleteQueries (soft) deletes the existing query objects with the provided
// IDs. The number of deleted queries is returned along with any error.
func (d *Datastore) DeleteQueries(ids []uint) (uint, error) {
	sql := `
		UPDATE queries
			SET deleted_at = NOW(), deleted = true
			WHERE id IN (?)
	`
	query, args, err := sqlx.In(sql, ids)
	if err != nil {
		return 0, errors.Wrap(err, "building delete query query")
	}

	result, err := d.db.Exec(query, args...)
	if err != nil {
		return 0, errors.Wrap(err, "updating delete query")
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "fetching delete query rows effected")
	}

	return uint(deleted), nil
}

// Query returns a single Query identified by id, if such
// exists
func (d *Datastore) Query(id uint) (*kolide.Query, error) {
	sql := `
		SELECT q.*, IFNULL(u.name, "") AS author_name
		FROM queries q
		LEFT JOIN users u
			ON q.author_id = u.id
		WHERE q.id = ?
		AND NOT q.deleted
	`
	query := &kolide.Query{}
	if err := d.db.Get(query, sql, id); err != nil {
		return nil, errors.Wrap(err, "selecting query")
	}

	if err := d.loadPacksForQueries([]*kolide.Query{query}); err != nil {
		return nil, errors.Wrap(err, "loading packs for queries")
	}

	return query, nil
}

// ListQueries returns a list of queries with sort order and results limit
// determined by passed in kolide.ListOptions
func (d *Datastore) ListQueries(opt kolide.ListOptions) ([]*kolide.Query, error) {
	sql := `
		SELECT q.*, IFNULL(u.name, "") AS author_name
		FROM queries q
		LEFT JOIN users u
			ON q.author_id = u.id
		WHERE saved = true
		AND NOT q.deleted
	`
	sql = appendListOptionsToSQL(sql, opt)
	results := []*kolide.Query{}

	if err := d.db.Select(&results, sql); err != nil {
		return nil, errors.Wrap(err, "listing queries")
	}

	if err := d.loadPacksForQueries(results); err != nil {
		return nil, errors.Wrap(err, "loading packs for queries")
	}

	return results, nil

}

// loadPacksForQueries loads the packs associated with the provided queries
func (d *Datastore) loadPacksForQueries(queries []*kolide.Query) error {
	if len(queries) == 0 {
		return nil
	}

	sql := `
		SELECT p.*, sq.query_id AS query_id
		FROM packs p
		JOIN scheduled_queries sq
			ON p.id = sq.pack_id
		WHERE query_id IN (?)
	`

	// Used to map the results
	id_queries := map[uint]*kolide.Query{}
	// Used for the IN clause
	ids := []uint{}
	for _, q := range queries {
		q.Packs = make([]kolide.Pack, 0)
		ids = append(ids, q.ID)
		id_queries[q.ID] = q
	}

	query, args, err := sqlx.In(sql, ids)
	if err != nil {
		return errors.Wrap(err, "building query in load packs for queries")
	}

	rows := []struct {
		QueryID uint `db:"query_id"`
		kolide.Pack
	}{}

	err = d.db.Select(&rows, query, args...)
	if err != nil {
		return errors.Wrap(err, "selecting load packs for queries")
	}

	for _, row := range rows {
		q := id_queries[row.QueryID]
		q.Packs = append(q.Packs, row.Pack)
	}

	return nil
}
