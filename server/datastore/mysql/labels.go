package mysql

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kolide/kolide/server/kolide"
	"github.com/pkg/errors"
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
		return nil, errors.Wrap(err, "inserting label")
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
		return nil, errors.Wrap(err, "selecting label")
	}

	return label, nil
}

// ListLabels returns all labels limited or sorted  by kolide.ListOptions
func (d *Datastore) ListLabels(opt kolide.ListOptions) ([]*kolide.Label, error) {
	query := `
		SELECT * FROM labels WHERE NOT deleted
	`
	query = appendListOptionsToSQL(query, opt)
	labels := []*kolide.Label{}

	if err := d.db.Select(&labels, query); err != nil {
		// it's ok if no labels exist
		if err == sql.ErrNoRows {
			return labels, nil
		}
		return nil, errors.Wrap(err, "selecting labels")
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
		return nil, errors.Wrap(err, "selecting label queries for host")
	}

	defer rows.Close()
	results := map[string]string{}

	for rows.Next() {
		var id, query string

		if err = rows.Scan(&id, &query); err != nil {
			return nil, errors.Wrap(err, "scanning label queries for host")
		}

		results[id] = query
	}

	return results, nil

}

func (d *Datastore) RecordLabelQueryExecutions(host *kolide.Host, results map[uint]bool, updated time.Time) error {
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
		return errors.Wrap(err, "inserting label query execution")
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
		return nil, errors.Wrap(err, "selecting host labels")
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
		return nil, errors.Wrap(err, "selecting label query executions")
	}
	return hosts, nil
}

func (d *Datastore) ListUniqueHostsInLabels(labels []uint) ([]kolide.Host, error) {
	if len(labels) == 0 {
		return []kolide.Host{}, nil
	}

	sqlStatement := `
		SELECT DISTINCT h.*
		FROM label_query_executions lqe
		JOIN hosts h
		ON lqe.host_id = h.id
		WHERE lqe.label_id IN (?)
		AND lqe.matches = 1
		AND NOT h.deleted
	`
	query, args, err := sqlx.In(sqlStatement, labels)
	if err != nil {
		return nil, errors.Wrap(err, "building query listing unique hosts in labels")
	}

	query = d.db.Rebind(query)
	hosts := []kolide.Host{}
	err = d.db.Select(&hosts, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "listing unique hosts in labels")
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
		return nil, errors.Wrap(err, "building query for labels with omits")
	}

	sql = d.db.Rebind(sql)

	matches := []kolide.Label{}
	err = d.db.Select(&matches, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "selecting labels with omits")
	}

	return matches, nil
}

func (d *Datastore) searchLabelsDefault(omit ...uint) ([]kolide.Label, error) {
	sqlStatement := `
	SELECT *
	FROM labels
	WHERE NOT deleted
	AND id NOT IN (?)
	LIMIT 5
	`

	var in interface{}
	{
		// use -1 if there are no values to omit.
		//Avoids empty args error for `sqlx.In`
		in = omit
		if len(omit) == 0 {
			in = -1
		}
	}

	var labels []kolide.Label
	sql, args, err := sqlx.In(sqlStatement, in)
	if err != nil {
		return nil, errors.Wrap(err, "searching default labels")
	}
	sql = d.db.Rebind(sql)
	err = d.db.Select(&labels, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "searching default labels rebound")
	}
	return labels, nil
}

// SearchLabels performs wildcard searches on kolide.Label name
func (d *Datastore) SearchLabels(query string, omit ...uint) ([]kolide.Label, error) {
	if query == "" {
		return d.searchLabelsDefault(omit...)
	}
	if len(omit) > 0 {
		return d.searchLabelsWithOmits(query, omit...)
	}

	query += "*"

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
		return nil, errors.Wrap(err, "selecting labels for search")
	}
	return matches, nil
}

func (d *Datastore) SaveLabel(label *kolide.Label) (*kolide.Label, error) {
	query := `
		UPDATE labels SET
			name = ?,
			description = ?
		WHERE id = ?
	`
	_, err := d.db.Exec(query, label.Name, label.Description, label.ID)
	if err != nil {
		return nil, errors.Wrap(err, "saving label")
	}
	return label, nil
}
