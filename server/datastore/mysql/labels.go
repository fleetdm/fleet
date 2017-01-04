package mysql

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kolide/kolide-ose/server/errors"
	"github.com/kolide/kolide-ose/server/kolide"
)

// NewLabel creates a new kolide.Label
func (d *Datastore) NewLabel(label *kolide.Label) (*kolide.Label, error) {

	sql := `
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type
		) VALUES ( ?, ?, ?, ?, ?)
	`
	result, err := d.db.Exec(sql, label.Name, label.Description, label.Query, label.Platform, label.LabelType)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	id, _ := result.LastInsertId()
	label.ID = uint(id)
	return label, nil

}

// DeleteLabel soft deletes a kolide.Label
func (d *Datastore) DeleteLabel(lid uint) error {
	return d.deleteEntity("labels", lid)
}

// Label returns a kolide.Label identified by  lid if one exists
func (d *Datastore) Label(lid uint) (*kolide.Label, error) {
	sql := `
		SELECT * FROM labels
			WHERE id = ? AND NOT deleted
	`
	label := &kolide.Label{}

	if err := d.db.Get(label, sql, lid); err != nil {
		return nil, errors.DatabaseError(err)
	}

	return label, nil
}

// ListLabels returns all labels limited or sorted  by kolide.ListOptions
func (d *Datastore) ListLabels(opt kolide.ListOptions) ([]*kolide.Label, error) {
	sql := `
		SELECT * FROM labels WHERE NOT deleted
	`
	sql = appendListOptionsToSQL(sql, opt)
	labels := []*kolide.Label{}

	if err := d.db.Select(&labels, sql); err != nil {
		return nil, errors.DatabaseError(err)
	}

	return labels, nil
}

func (d *Datastore) LabelQueriesForHost(host *kolide.Host, cutoff time.Time) (map[string]string, error) {
	sqlStatment := `
			SELECT l.id, l.query
			FROM labels l
			WHERE (l.platform = ? OR l.platform = '')
			AND NOT l.deleted
			AND l.id NOT IN /* subtract the set of executions that are recent enough */
			(
			  SELECT l.id
			  FROM labels l
			  JOIN label_query_executions lqe
			  ON lqe.label_id = l.id
			  WHERE lqe.host_id = ? AND lqe.updated_at > ?
			)
	`
	rows, err := d.db.Query(sqlStatment, host.Platform, host.ID, cutoff)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.DatabaseError(err)
	}

	defer rows.Close()
	results := map[string]string{}

	for rows.Next() {
		var id, query string

		if err = rows.Scan(&id, &query); err != nil {
			return nil, errors.DatabaseError(err)
		}

		results[id] = query
	}

	return results, nil

}

func (d *Datastore) RecordLabelQueryExecutions(host *kolide.Host, results map[string]bool, updated time.Time) error {
	sqlStatement := `
	INSERT INTO label_query_executions (updated_at, matches, label_id, host_id) VALUES

	`
	vals := []interface{}{}
	bindvars := ""

	for labelID, result := range results {
		if bindvars != "" {
			bindvars += ","
		}
		bindvars += "(?,?,?,?)"
		vals = append(vals, updated, result, labelID, host.ID)
	}

	sqlStatement += bindvars
	sqlStatement += `
		ON DUPLICATE KEY UPDATE
		updated_at = VALUES(updated_at),
		matches = VALUES(matches)
	`

	_, err := d.db.Exec(sqlStatement, vals...)
	if err != nil {
		return errors.DatabaseError(err)
	}

	return nil
}

// ListLabelsForHost returns a list of kolide.Label for a given host id.
func (d *Datastore) ListLabelsForHost(hid uint) ([]kolide.Label, error) {
	sqlStatement := `
		SELECT labels.* from labels, label_query_executions lqe
		WHERE lqe.host_id = ?
		AND lqe.label_id = labels.id
		AND lqe.matches
		AND NOT labels.deleted
	`

	labels := []kolide.Label{}
	err := d.db.Select(&labels, sqlStatement, hid)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}
	return labels, nil

}

// ListHostsInLabel returns a list of kolide.Host that are associated
// with kolide.Label referened by Label ID
func (d *Datastore) ListHostsInLabel(lid uint) ([]kolide.Host, error) {
	sqlStatement := `
		SELECT h.*
		FROM label_query_executions lqe
		JOIN hosts h
		ON lqe.host_id = h.id
		WHERE lqe.label_id = ?
		AND lqe.matches = 1
		AND NOT h.deleted
	`
	hosts := []kolide.Host{}
	err := d.db.Select(&hosts, sqlStatement, lid)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}
	return hosts, nil
}

func (d *Datastore) ListUniqueHostsInLabels(labels []uint) ([]kolide.Host, error) {
	if len(labels) == 0 {
		return []kolide.Host{}, nil
	}

	sqlStatement := `
		SELECT h.*
		FROM label_query_executions lqe
		JOIN hosts h
		ON lqe.host_id = h.id
		WHERE lqe.label_id IN (?)
		AND lqe.matches = 1
		AND NOT h.deleted
		GROUP BY h.id;
	`
	query, args, err := sqlx.In(sqlStatement, labels)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	query = d.db.Rebind(query)
	hosts := []kolide.Host{}
	err = d.db.Select(&hosts, query, args...)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return hosts, nil

}

func (d *Datastore) searchLabelsWithOmits(query string, omit ...uint) ([]kolide.Label, error) {
	if len(query) > 0 {
		query += "*"
	}
	sqlStatement := `
		SELECT *
		FROM labels
		WHERE (
			(
				MATCH(name) AGAINST(? IN BOOLEAN MODE)
				AND NOT deleted
			)
			OR (
				label_type=?
				AND name = 'All Hosts'
			)
		)
		AND id NOT IN (?)
		ORDER BY id ASC
		LIMIT 10
	`

	sql, args, err := sqlx.In(sqlStatement, query, kolide.LabelTypeBuiltIn, omit)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	sql = d.db.Rebind(sql)

	matches := []kolide.Label{}
	err = d.db.Select(&matches, sql, args...)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return matches, nil
}

// SearchLabels performs wildcard searches on kolide.Label name
func (d *Datastore) SearchLabels(query string, omit ...uint) ([]kolide.Label, error) {
	if len(omit) > 0 {
		return d.searchLabelsWithOmits(query, omit...)
	}

	if len(query) > 0 {
		query += "*"
	}

	sqlStatement := `
		SELECT *
		FROM labels
		WHERE (
			(
				MATCH(name) AGAINST(? IN BOOLEAN MODE)
				AND NOT deleted
			)
			OR (
				label_type=?
				AND name = 'All Hosts'
			)
		)
		ORDER BY id ASC
		LIMIT 10
	`
	matches := []kolide.Label{}
	err := d.db.Select(&matches, sqlStatement, query, kolide.LabelTypeBuiltIn)
	if err != nil {
		return nil, errors.DatabaseError(err)
	}

	return matches, nil
}
