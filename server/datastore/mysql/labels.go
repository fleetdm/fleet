package mysql

import (
	"database/sql"
	"strings"
	"time"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func (d *Datastore) ApplyLabelSpecs(specs []*kolide.LabelSpec) (err error) {
	err = d.withRetryTxx(func(tx *sqlx.Tx) error {
		sql := `
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type,
			label_membership_type
		) VALUES ( ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			description = VALUES(description),
			query = VALUES(query),
			platform = VALUES(platform),
			label_type = VALUES(label_type),
			label_membership_type = VALUES(label_membership_type)
	`
		stmt, err := tx.Prepare(sql)
		if err != nil {
			return errors.Wrap(err, "prepare ApplyLabelSpecs insert")
		}

		for _, s := range specs {
			if s.Name == "" {
				return errors.New("label name must not be empty")
			}
			_, err := stmt.Exec(s.Name, s.Description, s.Query, s.Platform, s.LabelType, s.LabelMembershipType)
			if err != nil {
				return errors.Wrap(err, "exec ApplyLabelSpecs insert")
			}

			if s.LabelType == kolide.LabelTypeBuiltIn ||
				s.LabelMembershipType != kolide.LabelMembershipTypeManual {
				// No need to update membership
				continue
			}

			var labelID uint
			sql = `
SELECT id from labels WHERE name = ?
`
			if err := tx.Get(&labelID, sql, s.Name); err != nil {
				return errors.Wrap(err, "get label ID")
			}

			sql = `
DELETE FROM label_membership WHERE label_id = ?
`
			_, err = tx.Exec(sql, labelID)
			if err != nil {
				return errors.Wrap(err, "clear membership for ID")
			}

			if len(s.Hosts) == 0 {
				continue
			}

			// Split hostnames into batches to avoid parameter limit in MySQL.
			for _, hostnames := range batchHostnames(s.Hosts) {
				// Use ignore because duplicate hostnames could appear in
				// different batches and would result in duplicate key errors.
				sql = `
INSERT IGNORE INTO label_membership (label_id, host_id) (SELECT ?, id FROM hosts where host_name IN (?))
`
				sql, args, err := sqlx.In(sql, labelID, hostnames)
				if err != nil {
					return errors.Wrap(err, "build membership IN statement")
				}
				_, err = tx.Exec(sql, args...)
				if err != nil {
					return errors.Wrap(err, "execute membership INSERT")
				}
			}
		}

		return nil
	})

	return errors.Wrap(err, "ApplyLabelSpecs transaction")
}

func batchHostnames(hostnames []string) [][]string {
	// Split hostnames into batches so that they can all be inserted without
	// overflowing the MySQL max number of parameters (somewhere around 65,000
	// but not well documented). Algorithm from
	// https://github.com/golang/go/wiki/SliceTricks#batching-with-minimal-allocation
	const batchSize = 50000 // Large, but well under the undocumented limit
	batches := make([][]string, 0, (len(hostnames)+batchSize-1)/batchSize)

	for batchSize < len(hostnames) {
		hostnames, batches = hostnames[batchSize:], append(batches, hostnames[0:batchSize:batchSize])
	}
	batches = append(batches, hostnames)
	return batches
}

func (d *Datastore) GetLabelSpecs() ([]*kolide.LabelSpec, error) {
	var specs []*kolide.LabelSpec
	// Get basic specs
	query := "SELECT name, description, query, platform, label_type, label_membership_type FROM labels"
	if err := d.db.Select(&specs, query); err != nil {
		return nil, errors.Wrap(err, "get labels")
	}

	for _, spec := range specs {
		if spec.LabelType != kolide.LabelTypeBuiltIn &&
			spec.LabelMembershipType == kolide.LabelMembershipTypeManual {
			if err := d.getLabelHostnames(spec); err != nil {
				return nil, err
			}
		}
	}

	return specs, nil
}

func (d *Datastore) GetLabelSpec(name string) (*kolide.LabelSpec, error) {
	var specs []*kolide.LabelSpec
	query := `
SELECT name, description, query, platform, label_type, label_membership_type
FROM labels
WHERE name = ?
`
	if err := d.db.Select(&specs, query, name); err != nil {
		return nil, errors.Wrap(err, "get label")
	}
	if len(specs) == 0 {
		return nil, notFound("Label").WithName(name)
	}
	if len(specs) > 1 {
		return nil, errors.Errorf("expected 1 label row, got %d", len(specs))
	}

	spec := specs[0]
	if spec.LabelType != kolide.LabelTypeBuiltIn &&
		spec.LabelMembershipType == kolide.LabelMembershipTypeManual {
		err := d.getLabelHostnames(spec)
		if err != nil {
			return nil, err
		}
	}

	return spec, nil
}

func (d *Datastore) getLabelHostnames(label *kolide.LabelSpec) error {
	sql := `
		SELECT host_name
		FROM hosts
		WHERE id IN
		(
			SELECT host_id
			FROM label_membership
			WHERE label_id = (SELECT id FROM labels WHERE name = ?)
		)
	`
	err := d.db.Select(&label.Hosts, sql, label.Name)
	if err != nil {
		return errors.Wrap(err, "get hostnames for label")
	}
	return nil
}

// NewLabel creates a new kolide.Label
func (d *Datastore) NewLabel(label *kolide.Label, opts ...kolide.OptionalArg) (*kolide.Label, error) {
	db := d.getTransaction(opts)
	query := `
	INSERT INTO labels (
		name,
		description,
		query,
		platform,
		label_type,
		label_membership_type
	) VALUES ( ?, ?, ?, ?, ?, ?)
	`
	result, err := db.Exec(
		query,
		label.Name,
		label.Description,
		label.Query,
		label.Platform,
		label.LabelType,
		label.LabelMembershipType,
	)
	if err != nil {
		return nil, errors.Wrap(err, "inserting label")
	}

	id, _ := result.LastInsertId()
	label.ID = uint(id)
	return label, nil

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

// DeleteLabel deletes a kolide.Label
func (d *Datastore) DeleteLabel(name string) error {
	return d.deleteEntityByName("labels", name)
}

// Label returns a kolide.Label identified by lid if one exists.
func (d *Datastore) Label(lid uint) (*kolide.Label, error) {
	sql := `
		SELECT * FROM labels
			WHERE id = ?
	`
	label := &kolide.Label{}

	if err := d.db.Get(label, sql, lid); err != nil {
		return nil, errors.Wrap(err, "selecting label")
	}

	return label, nil
}

// ListLabels returns all labels limited or sorted by kolide.ListOptions.
func (d *Datastore) ListLabels(opt kolide.ListOptions) ([]*kolide.Label, error) {
	query := `
		SELECT *, (SELECT COUNT(1) FROM label_membership WHERE label_id = id) AS host_count
		FROM labels
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
	var rows *sql.Rows
	var err error
	if host.LabelUpdateTime.Before(cutoff) {
		// Retrieve all labels (with matching platform) for this host
		sql := `
			SELECT id, query
			FROM labels
			WHERE platform = ? OR platform = ''
			AND label_membership_type = ?
`
		rows, err = d.db.Query(sql, host.Platform, kolide.LabelMembershipTypeDynamic)
	} else {
		// Retrieve all labels (with matching platform) iff there is a label
		// that has been created since this host last reported label query
		// executions
		sql := `
			SELECT id, query
			FROM labels
			WHERE ((SELECT max(created_at) FROM labels WHERE platform = ? OR platform = '') > ?)
			AND (platform = ? OR platform = '')
			AND label_membership_type = ?
`
		rows, err = d.db.Query(
			sql,
			host.Platform,
			host.LabelUpdateTime,
			host.Platform,
			kolide.LabelMembershipTypeDynamic,
		)
	}

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
	// Loop through results, collecting which labels we need to insert/update,
	// and which we need to delete
	vals := []interface{}{}
	bindvars := []string{}
	removes := []uint{}

	for labelID, matches := range results {
		if matches {
			// Add/update row
			bindvars = append(bindvars, "(?,?,?)")
			vals = append(vals, updated, labelID, host.ID)
		} else {
			// Delete row
			removes = append(removes, labelID)
		}
	}

	// Complete inserts if necessary
	if len(vals) > 0 {
		sql := `
			INSERT INTO label_membership (updated_at, label_id, host_id) VALUES
		`
		sql += strings.Join(bindvars, ",") +
			`
			ON DUPLICATE KEY UPDATE
			updated_at = VALUES(updated_at)
		`

		_, err := d.db.Exec(sql, vals...)
		if err != nil {
			return errors.Wrap(err, "insert label query executions")
		}
	}

	// Complete deletions if necessary
	if len(removes) > 0 {
		sql := `
			DELETE FROM label_membership WHERE host_id = ? AND label_id IN (?)
		`
		query, args, err := sqlx.In(sql, host.ID, removes)
		if err != nil {
			return errors.Wrap(err, "IN for DELETE FROM label_membership")
		}
		query = d.db.Rebind(query)
		_, err = d.db.Exec(query, args...)
		if err != nil {
			return errors.Wrap(err, "delete label query executions")
		}
	}

	return nil
}

// ListLabelsForHost returns a list of kolide.Label for a given host id.
func (d *Datastore) ListLabelsForHost(hid uint) ([]kolide.Label, error) {
	sqlStatement := `
		SELECT labels.* from labels JOIN label_membership lm
		WHERE lm.host_id = ?
		AND lm.label_id = labels.id
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
func (d *Datastore) ListHostsInLabel(lid uint, opt kolide.HostListOptions) ([]kolide.Host, error) {
	sql := `
		SELECT h.*
		FROM label_membership lm
		JOIN hosts h
		ON lm.host_id = h.id
		WHERE lm.label_id = ?
	`
	params := []interface{}{lid}

	sql, params = searchLike(sql, params, opt.MatchQuery, hostSearchColumns...)

	sql = appendListOptionsToSQL(sql, opt.ListOptions)
	hosts := []kolide.Host{}
	err := d.db.Select(&hosts, sql, params...)
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
		FROM label_membership lm
		JOIN hosts h
		ON lm.host_id = h.id
		WHERE lm.label_id IN (?)
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
	transformedQuery := transformQuery(query)

	sqlStatement := `
		SELECT *, (SELECT COUNT(1) FROM label_membership WHERE label_id = id) AS host_count
		FROM labels
		WHERE (
			MATCH(name) AGAINST(? IN BOOLEAN MODE)
		)
		AND id NOT IN (?)
		ORDER BY label_type DESC, id ASC
		LIMIT 10
	`

	sql, args, err := sqlx.In(sqlStatement, transformedQuery, omit)
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
		SELECT *, (SELECT COUNT(1) FROM label_membership WHERE label_id = id) AS host_count
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
	SELECT *, (SELECT COUNT(1) FROM label_membership WHERE label_id = id) AS host_count
	FROM labels
	WHERE id NOT IN (?)
	GROUP BY id
	ORDER BY label_type DESC, id ASC
	LIMIT 7
	`

	var in interface{}
	{
		// use -1 if there are no values to omit.
		// Avoids empty args error for `sqlx.In`
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
	transformedQuery := transformQuery(query)
	if !queryMinLength(transformedQuery) {
		return d.searchLabelsDefault(omit...)
	}
	if len(omit) > 0 {
		return d.searchLabelsWithOmits(query, omit...)
	}

	// Ordering first by label_type ensures that built-in labels come
	// first. We will probably need to make a custom ordering function here
	// if additional label types are added. Ordering next by ID ensures
	// that the order is always consistent.
	sqlStatement := `
		SELECT *, (SELECT COUNT(1) FROM label_membership WHERE label_id = id) AS host_count
		FROM labels
		WHERE (
			MATCH(name) AGAINST(? IN BOOLEAN MODE)
		)
		ORDER BY label_type DESC, id ASC
		LIMIT 10
	`
	matches := []kolide.Label{}
	err := d.db.Select(&matches, sqlStatement, transformedQuery)
	if err != nil {
		return nil, errors.Wrap(err, "selecting labels for search")
	}

	matches, err = d.addAllHostsLabelToList(matches, omit...)
	if err != nil {
		return nil, errors.Wrap(err, "adding all hosts label to matches")
	}

	return matches, nil
}

func (d *Datastore) LabelIDsByName(labels []string) ([]uint, error) {
	if len(labels) == 0 {
		return []uint{}, nil
	}

	sqlStatement := `
		SELECT id FROM labels
		WHERE name IN (?)
	`

	sql, args, err := sqlx.In(sqlStatement, labels)
	if err != nil {
		return nil, errors.Wrap(err, "building query to get label IDs")
	}

	var labelIDs []uint
	if err := d.db.Select(&labelIDs, sql, args...); err != nil {
		return nil, errors.Wrap(err, "get label IDs")
	}

	return labelIDs, nil

}
