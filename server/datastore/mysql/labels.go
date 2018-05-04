package mysql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (d *Datastore) ApplyLabelSpecs(specs []*kolide.LabelSpec) (err error) {
	tx, err := d.db.Beginx()
	if err != nil {
		return errors.Wrap(err, "begin ApplyLabelSpecs transaction")
	}

	defer func() {
		if err != nil {
			rbErr := tx.Rollback()
			// It seems possible that there might be a case in
			// which the error we are dealing with here was thrown
			// by the call to tx.Commit(), and the docs suggest
			// this call would then result in sql.ErrTxDone.
			if rbErr != nil && rbErr != sql.ErrTxDone {
				panic(fmt.Sprintf("got err '%s' rolling back after err '%s'", rbErr, err))
			}
		}
	}()

	sql := `
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type
		) VALUES ( ?, ?, ?, ?, ? )
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			description = VALUES(description),
			query = VALUES(query),
			platform = VALUES(platform),
			label_type = VALUES(label_type),
			deleted = false
	`
	stmt, err := tx.Prepare(sql)
	if err != nil {
		return errors.Wrap(err, "prepare ApplyLabelSpecs insert")
	}

	for _, s := range specs {
		_, err := stmt.Exec(s.Name, s.Description, s.Query, s.Platform, s.LabelType)
		if err != nil {
			return errors.Wrap(err, "exec ApplyLabelSpecs insert")
		}
	}

	err = tx.Commit()
	return errors.Wrap(err, "commit ApplyLabelSpecs transaction")
}

func (d *Datastore) GetLabelSpecs() ([]*kolide.LabelSpec, error) {
	var specs []*kolide.LabelSpec
	// Get basic specs
	query := "SELECT name, description, query, platform, label_type FROM labels"
	if err := d.db.Select(&specs, query); err != nil {
		return nil, errors.Wrap(err, "get labels")
	}

	return specs, nil
}

// DeleteLabel deletes a kolide.Label
func (d *Datastore) DeleteLabel(name string) error {
	return d.deleteEntityByName("labels", name)
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
			MATCH(name) AGAINST(? IN BOOLEAN MODE)
			AND NOT deleted
		)
		AND id NOT IN (?)
		ORDER BY label_type DESC, id ASC
		LIMIT 10
	`

	sql, args, err := sqlx.In(sqlStatement, query, omit)
	if err != nil {
		return nil, errors.Wrap(err, "building query for labels with omits")
	}

	sql = d.db.Rebind(sql)

	matches := []kolide.Label{}
	err = d.db.Select(&matches, sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "selecting labels with omits")
	}

	matches, err = d.addAllHostsLabelToList(matches, omit...)
	if err != nil {
		return nil, errors.Wrap(err, "adding all hosts label to matches")
	}

	return matches, nil
}

// When we search labels, we always want to make sure that the All Hosts label
// is included in the results set. Sometimes it already is and we don't need to
// add it, sometimes it's not so we explicitly add it.
func (d *Datastore) addAllHostsLabelToList(labels []kolide.Label, omit ...uint) ([]kolide.Label, error) {
	sqlStatement := `
		SELECT *
		FROM labels
		WHERE
		  label_type=?
			AND name = 'All Hosts'
		LIMIT 1
	`

	var allHosts kolide.Label
	err := d.db.Get(&allHosts, sqlStatement, kolide.LabelTypeBuiltIn)
	if err != nil {
		return nil, errors.Wrap(err, "getting all hosts label")
	}

	for _, omission := range omit {
		if omission == allHosts.ID {
			return labels, nil
		}
	}

	for _, label := range labels {
		if label.ID == allHosts.ID {
			return labels, nil
		}
	}

	return append(labels, allHosts), nil
}

func (d *Datastore) searchLabelsDefault(omit ...uint) ([]kolide.Label, error) {
	sqlStatement := `
	SELECT *
	FROM labels
	WHERE NOT deleted
	AND id NOT IN (?)
	ORDER BY label_type DESC, id ASC
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

	labels, err = d.addAllHostsLabelToList(labels, omit...)
	if err != nil {
		return nil, errors.Wrap(err, "getting all host label")
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

	// Ordering first by label_type ensures that built-in labels come
	// first. We will probably need to make a custom ordering function here
	// if additional label types are added. Ordering next by ID ensures
	// that the order is always consistent.
	sqlStatement := `
		SELECT *
		FROM labels
		WHERE (
			MATCH(name) AGAINST(? IN BOOLEAN MODE)
			AND NOT deleted
		)
		ORDER BY label_type DESC, id ASC
		LIMIT 10
	`
	matches := []kolide.Label{}
	err := d.db.Select(&matches, sqlStatement, query)
	if err != nil {
		return nil, errors.Wrap(err, "selecting labels for search")
	}

	matches, err = d.addAllHostsLabelToList(matches, omit...)
	if err != nil {
		return nil, errors.Wrap(err, "adding all hosts label to matches")
	}

	return matches, nil
}
