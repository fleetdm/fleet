package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ApplyLabelSpecs(ctx context.Context, specs []*fleet.LabelSpec) (err error) {
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
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

		prepTx, ok := tx.(sqlx.PreparerContext)
		if !ok {
			return ctxerr.New(ctx, "tx in ApplyLabelSpecs is not a sqlx.PreparerContext")
		}
		stmt, err := prepTx.PrepareContext(ctx, sql)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "prepare ApplyLabelSpecs insert")
		}
		defer stmt.Close()

		for _, s := range specs {
			if s.Name == "" {
				return ctxerr.New(ctx, "label name must not be empty")
			}
			_, err := stmt.ExecContext(ctx, s.Name, s.Description, s.Query, s.Platform, s.LabelType, s.LabelMembershipType)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "exec ApplyLabelSpecs insert")
			}

			if s.LabelType == fleet.LabelTypeBuiltIn ||
				s.LabelMembershipType != fleet.LabelMembershipTypeManual {
				// No need to update membership
				continue
			}

			var labelID uint
			sql = `
SELECT id from labels WHERE name = ?
`
			if err := sqlx.GetContext(ctx, tx, &labelID, sql, s.Name); err != nil {
				return ctxerr.Wrap(ctx, err, "get label ID")
			}

			sql = `
DELETE FROM label_membership WHERE label_id = ?
`
			_, err = tx.ExecContext(ctx, sql, labelID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "clear membership for ID")
			}

			if len(s.Hosts) == 0 {
				continue
			}

			// Split hostnames into batches to avoid parameter limit in MySQL.
			for _, hostnames := range batchHostnames(s.Hosts) {
				// Use ignore because duplicate hostnames could appear in
				// different batches and would result in duplicate key errors.
				sql = `
INSERT IGNORE INTO label_membership (label_id, host_id) (SELECT ?, id FROM hosts where hostname IN (?))
`
				sql, args, err := sqlx.In(sql, labelID, hostnames)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "build membership IN statement")
				}
				_, err = tx.ExecContext(ctx, sql, args...)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "execute membership INSERT")
				}
			}
		}

		return nil
	})

	return ctxerr.Wrap(ctx, err, "ApplyLabelSpecs transaction")
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

func (ds *Datastore) GetLabelSpecs(ctx context.Context) ([]*fleet.LabelSpec, error) {
	var specs []*fleet.LabelSpec
	// Get basic specs
	query := "SELECT id, name, description, query, platform, label_type, label_membership_type FROM labels"
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &specs, query); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get labels")
	}

	for _, spec := range specs {
		if spec.LabelType != fleet.LabelTypeBuiltIn &&
			spec.LabelMembershipType == fleet.LabelMembershipTypeManual {
			if err := ds.getLabelHostnames(ctx, spec); err != nil {
				return nil, err
			}
		}
	}

	return specs, nil
}

func (ds *Datastore) GetLabelSpec(ctx context.Context, name string) (*fleet.LabelSpec, error) {
	var specs []*fleet.LabelSpec
	query := `
SELECT id, name, description, query, platform, label_type, label_membership_type
FROM labels
WHERE name = ?
`
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &specs, query, name); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get label")
	}
	if len(specs) == 0 {
		return nil, ctxerr.Wrap(ctx, notFound("Label").WithName(name))
	}
	if len(specs) > 1 {
		return nil, ctxerr.Errorf(ctx, "expected 1 label row, got %d", len(specs))
	}

	spec := specs[0]
	if spec.LabelType != fleet.LabelTypeBuiltIn &&
		spec.LabelMembershipType == fleet.LabelMembershipTypeManual {
		err := ds.getLabelHostnames(ctx, spec)
		if err != nil {
			return nil, err
		}
	}

	return spec, nil
}

func (ds *Datastore) getLabelHostnames(ctx context.Context, label *fleet.LabelSpec) error {
	sql := `
		SELECT hostname
		FROM hosts
		WHERE id IN
		(
			SELECT host_id
			FROM label_membership
			WHERE label_id = (SELECT id FROM labels WHERE name = ?)
		)
	`
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &label.Hosts, sql, label.Name)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get hostnames for label")
	}
	return nil
}

// NewLabel creates a new fleet.Label
func (ds *Datastore) NewLabel(ctx context.Context, label *fleet.Label, opts ...fleet.OptionalArg) (*fleet.Label, error) {
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
	result, err := ds.writer(ctx).ExecContext(
		ctx,
		query,
		label.Name,
		label.Description,
		label.Query,
		label.Platform,
		label.LabelType,
		label.LabelMembershipType,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting label")
	}

	id, _ := result.LastInsertId()
	label.ID = uint(id)
	return label, nil
}

func (ds *Datastore) SaveLabel(ctx context.Context, label *fleet.Label) (*fleet.Label, error) {
	query := `UPDATE labels SET name = ?, description = ? WHERE id = ?`
	_, err := ds.writer(ctx).ExecContext(ctx, query, label.Name, label.Description, label.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "saving label")
	}
	return labelDB(ctx, label.ID, ds.writer(ctx))
}

// DeleteLabel deletes a fleet.Label
func (ds *Datastore) DeleteLabel(ctx context.Context, name string) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var labelID uint
		err := sqlx.GetContext(ctx, tx, &labelID, `select id FROM labels WHERE name = ?`, name)
		if err != nil {
			if err == sql.ErrNoRows {
				return ctxerr.Wrap(ctx, notFound("Label").WithName(name))
			}
			return ctxerr.Wrapf(ctx, err, "getting label id to delete")
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM labels WHERE id = ?`, labelID)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "delete label")
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM label_membership WHERE label_id = ?`, labelID)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "delete label_membership")
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM pack_targets WHERE type=? AND target_id=?`, fleet.TargetLabel, labelID)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "deleting pack_targets for label %d", labelID)
		}

		return nil
	})
}

// Label returns a fleet.Label identified by lid if one exists.
func (ds *Datastore) Label(ctx context.Context, lid uint) (*fleet.Label, error) {
	return labelDB(ctx, lid, ds.reader(ctx))
}

func labelDB(ctx context.Context, lid uint, q sqlx.QueryerContext) (*fleet.Label, error) {
	stmt := `
		SELECT
		       l.*,
		       (SELECT COUNT(1) FROM label_membership lm JOIN hosts h ON (lm.host_id = h.id) WHERE label_id = l.id) AS host_count
		FROM labels l
		WHERE id = ?
	`
	label := &fleet.Label{}

	if err := sqlx.GetContext(ctx, q, label, stmt, lid); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Label").WithID(lid))
		}
		return nil, ctxerr.Wrap(ctx, err, "selecting label")
	}

	return label, nil
}

// ListLabels returns all labels limited or sorted by fleet.ListOptions.
func (ds *Datastore) ListLabels(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Label, error) {
	query := fmt.Sprintf(`
			SELECT *,
				(SELECT COUNT(1) FROM label_membership lm JOIN hosts h ON (lm.host_id = h.id) WHERE label_id = l.id AND %s) AS host_count
			FROM labels l
		`, ds.whereFilterHostsByTeams(filter, "h"),
	)

	query = appendListOptionsToSQL(query, &opt)
	labels := []*fleet.Label{}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, query); err != nil {
		// it's ok if no labels exist
		if err == sql.ErrNoRows {
			return labels, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "selecting labels")
	}

	return labels, nil
}

func platformForHost(host *fleet.Host) string {
	if host.Platform != "rhel" {
		return host.Platform
	}
	if strings.Contains(strings.ToLower(host.OSVersion), "centos") {
		return "centos"
	}
	return host.Platform
}

func (ds *Datastore) LabelQueriesForHost(ctx context.Context, host *fleet.Host) (map[string]string, error) {
	var rows *sql.Rows
	var err error
	platform := platformForHost(host)
	query := `SELECT id, query FROM labels WHERE platform = ? OR platform = '' AND label_membership_type = ?`
	rows, err = ds.reader(ctx).QueryContext(ctx, query, platform, fleet.LabelMembershipTypeDynamic)

	if err != nil && err != sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, err, "selecting label queries for host")
	}

	defer rows.Close()
	results := map[string]string{}

	for rows.Next() {
		var id, query string

		if err = rows.Scan(&id, &query); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "scanning label queries for host")
		}

		results[id] = query
	}
	if err := rows.Err(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "iterating over returned rows")
	}

	return results, nil
}

func (ds *Datastore) RecordLabelQueryExecutions(ctx context.Context, host *fleet.Host, results map[uint]*bool, updated time.Time, deferredSaveHost bool) error {
	// Sort the results to have generated SQL queries ordered to minimize
	// deadlocks. See https://github.com/fleetdm/fleet/issues/1146.
	orderedIDs := make([]uint, 0, len(results))
	for labelID := range results {
		orderedIDs = append(orderedIDs, labelID)
	}
	sort.Slice(orderedIDs, func(i, j int) bool { return orderedIDs[i] < orderedIDs[j] })

	// Loop through results, collecting which labels we need to insert/update,
	// and which we need to delete
	vals := []interface{}{}
	bindvars := []string{}
	removes := []uint{}
	for _, labelID := range orderedIDs {
		matches := results[labelID]
		if matches != nil && *matches {
			// Add/update row
			bindvars = append(bindvars, "(?,?,?)")
			vals = append(vals, updated, labelID, host.ID)
		} else {
			// Delete row
			removes = append(removes, labelID)
		}
	}

	// NOTE: the insert/delete of label membership that follows must be kept in
	// sync with the async implementations in
	// AsyncBatch{Insert,Delete}LabelMembership, and the update of the
	// label_updated_at timestamp in sync with the
	// AsyncBatchUpdateLabelTimestamp method (that is, their processing must be
	// semantically equivalent, even though here it processes a single host and
	// in async mode it processes a batch of hosts).

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Complete inserts if necessary
		if len(vals) > 0 {
			sql := `INSERT INTO label_membership (updated_at, label_id, host_id) VALUES `
			sql += strings.Join(bindvars, ",") + ` ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at)`

			_, err := tx.ExecContext(ctx, sql, vals...)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "insert label query executions (%v)", vals)
			}
		}

		// Complete deletions if necessary
		if len(removes) > 0 {
			sql := `DELETE FROM label_membership WHERE host_id = ? AND label_id IN (?)`
			query, args, err := sqlx.In(sql, host.ID, removes)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "IN for DELETE FROM label_membership")
			}
			query = tx.Rebind(query)
			_, err = tx.ExecContext(ctx, query, args...)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "delete label query executions")
			}
		}

		// if we are deferring host updates, we return at this point and do the change outside of the tx
		if deferredSaveHost {
			return nil
		}

		_, err := tx.ExecContext(ctx, `UPDATE hosts SET label_updated_at = ? WHERE id=?`, host.LabelUpdatedAt, host.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "updating hosts label updated at")
		}

		return nil
	})
	if err != nil {
		return err
	}

	if deferredSaveHost {
		errCh := make(chan error, 1)
		defer close(errCh)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ds.writeCh <- itemToWrite{
			ctx:   ctx,
			errCh: errCh,
			item: hostXUpdatedAt{
				hostID:    host.ID,
				updatedAt: updated,
				what:      "label_updated_at",
			},
		}:
			return <-errCh
		}
	}
	return nil
}

// ListLabelsForHost returns a list of fleet.Label for a given host id.
func (ds *Datastore) ListLabelsForHost(ctx context.Context, hid uint) ([]*fleet.Label, error) {
	sqlStatement := `
		SELECT labels.* from labels JOIN label_membership lm
		WHERE lm.host_id = ?
		AND lm.label_id = labels.id
	`

	labels := []*fleet.Label{}
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, sqlStatement, hid)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting host labels")
	}

	return labels, nil
}

// ListHostsInLabel returns a list of fleet.Host that are associated
// with fleet.Label referenced by Label ID
func (ds *Datastore) ListHostsInLabel(ctx context.Context, filter fleet.TeamFilter, lid uint, opt fleet.HostListOptions) ([]*fleet.Host, error) {
	queryFmt := `
    SELECT
      h.id,
      h.osquery_host_id,
      h.created_at,
      h.updated_at,
      h.detail_updated_at,
      h.node_key,
      h.hostname,
      h.uuid,
      h.platform,
      h.osquery_version,
      h.os_version,
      h.build,
      h.platform_like,
      h.code_name,
      h.uptime,
      h.memory,
      h.cpu_type,
      h.cpu_subtype,
      h.cpu_brand,
      h.cpu_physical_cores,
      h.cpu_logical_cores,
      h.hardware_vendor,
      h.hardware_model,
      h.hardware_version,
      h.hardware_serial,
      h.computer_name,
      h.primary_ip_id,
      h.distributed_interval,
      h.logger_tls_period,
      h.config_tls_refresh,
      h.primary_ip,
      h.primary_mac,
      h.label_updated_at,
      h.last_enrolled_at,
      h.refetch_requested,
      h.refetch_critical_queries_until,
      h.team_id,
      h.policy_updated_at,
      h.public_ip,
      COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
      COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available,
      COALESCE(hst.seen_time, h.created_at) as seen_time,
      COALESCE(hu.software_updated_at, h.created_at) AS software_updated_at,
      (SELECT name FROM teams t WHERE t.id = h.team_id) AS team_name
      %s
      %s
			%s
    FROM label_membership lm
    JOIN hosts h ON (lm.host_id = h.id)
    LEFT JOIN host_seen_times hst ON (h.id=hst.host_id)
    LEFT JOIN host_updates hu ON (h.id = hu.host_id)
    LEFT JOIN host_disks hd ON (h.id=hd.host_id)
    %s
    %s
		%s
`
	failingPoliciesSelect := `,
		COALESCE(failing_policies.count, 0) AS failing_policies_count,
		COALESCE(failing_policies.count, 0) AS total_issues_count
	`
	failingPoliciesJoin := `LEFT JOIN (
		SELECT host_id, count(*) as count FROM policy_membership WHERE passes = 0
		GROUP BY host_id
	) as failing_policies ON (h.id=failing_policies.host_id)`

	if opt.DisableFailingPolicies {
		failingPoliciesSelect = ""
		failingPoliciesJoin = ""
	}

	deviceMappingJoin := `LEFT JOIN (
	SELECT
		host_id,
		CONCAT('[', GROUP_CONCAT(JSON_OBJECT('email', email, 'source', source)), ']') AS device_mapping
	FROM
		host_emails
	GROUP BY
		host_id) dm ON dm.host_id = h.id`
	if !opt.DeviceMapping {
		deviceMappingJoin = ""
	}

	var deviceMappingSelect string
	if opt.DeviceMapping {
		deviceMappingSelect = `,
	COALESCE(dm.device_mapping, 'null') as device_mapping`
	}

	query := fmt.Sprintf(queryFmt, hostMDMSelect, failingPoliciesSelect, deviceMappingSelect, hostMDMJoin, failingPoliciesJoin, deviceMappingJoin)

	query, params := ds.applyHostLabelFilters(filter, lid, query, opt)

	hosts := []*fleet.Host{}
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, query, params...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting label query executions")
	}
	return hosts, nil
}

// NOTE: the hosts table must be aliased to `h` in the query passed to this function.
func (ds *Datastore) applyHostLabelFilters(filter fleet.TeamFilter, lid uint, query string, opt fleet.HostListOptions) (string, []interface{}) {
	params := []interface{}{lid}

	if opt.ListOptions.OrderKey == "display_name" {
		query += ` JOIN host_display_names hdn ON h.id = hdn.host_id `
	}

	query += fmt.Sprintf(` WHERE lm.label_id = ? AND %s `, ds.whereFilterHostsByTeams(filter, "h"))
	if opt.LowDiskSpaceFilter != nil {
		query += ` AND hd.gigs_disk_space_available < ? `
		params = append(params, *opt.LowDiskSpaceFilter)
	}

	query, params = filterHostsByStatus(ds.clock.Now(), query, opt, params)
	query, params = filterHostsByTeam(query, opt, params)
	query, params = filterHostsByMDM(query, opt, params)
	query, params = filterHostsByMacOSSettingsStatus(query, opt, params)
	query, params = filterHostsByMacOSDiskEncryptionStatus(query, opt, params)
	query, params = filterHostsByMDMBootstrapPackageStatus(query, opt, params)
	query, params = searchLike(query, params, opt.MatchQuery, hostSearchColumns...)

	query, params = appendListOptionsWithCursorToSQL(query, params, &opt.ListOptions)
	return query, params
}

func (ds *Datastore) CountHostsInLabel(ctx context.Context, filter fleet.TeamFilter, lid uint, opt fleet.HostListOptions) (int, error) {
	query := `SELECT count(*) FROM label_membership lm
    JOIN hosts h ON (lm.host_id = h.id)
	LEFT JOIN host_seen_times hst ON (h.id=hst.host_id)
 	`

	query += hostMDMJoin

	if opt.LowDiskSpaceFilter != nil {
		query += ` LEFT JOIN host_disks hd ON (h.id=hd.host_id) `
	}

	query, params := ds.applyHostLabelFilters(filter, lid, query, opt)

	var count int
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, query, params...); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count hosts")
	}

	return count, nil
}

func (ds *Datastore) ListUniqueHostsInLabels(ctx context.Context, filter fleet.TeamFilter, labels []uint) ([]*fleet.Host, error) {
	if len(labels) == 0 {
		return []*fleet.Host{}, nil
	}

	sqlStatement := fmt.Sprintf(`
      SELECT DISTINCT
        h.id,
        h.osquery_host_id,
        h.created_at,
        h.updated_at,
        h.detail_updated_at,
        h.node_key,
        h.hostname,
        h.uuid,
        h.platform,
        h.osquery_version,
        h.os_version,
        h.build,
        h.platform_like,
        h.code_name,
        h.uptime,
        h.memory,
        h.cpu_type,
        h.cpu_subtype,
        h.cpu_brand,
        h.cpu_physical_cores,
        h.cpu_logical_cores,
        h.hardware_vendor,
        h.hardware_model,
        h.hardware_version,
        h.hardware_serial,
        h.computer_name,
        h.primary_ip_id,
        h.distributed_interval,
        h.logger_tls_period,
        h.config_tls_refresh,
        h.primary_ip,
        h.primary_mac,
        h.label_updated_at,
        h.last_enrolled_at,
        h.refetch_requested,
        h.refetch_critical_queries_until,
        h.team_id,
        h.policy_updated_at,
        h.public_ip,
        COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
        COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available,
        (SELECT name FROM teams t WHERE t.id = h.team_id) AS team_name
      FROM label_membership lm
      JOIN hosts h ON lm.host_id = h.id
      LEFT JOIN host_disks hd ON hd.host_id = h.id
      WHERE lm.label_id IN (?) AND %s
`, ds.whereFilterHostsByTeams(filter, "h"),
	)

	query, args, err := sqlx.In(sqlStatement, labels)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query listing unique hosts in labels")
	}

	query = ds.reader(ctx).Rebind(query)
	hosts := []*fleet.Host{}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing unique hosts in labels")
	}

	return hosts, nil
}

func (ds *Datastore) searchLabelsWithOmits(ctx context.Context, filter fleet.TeamFilter, query string, omit ...uint) ([]*fleet.Label, error) {
	transformedQuery := transformQuery(query)

	sqlStatement := fmt.Sprintf(`
			SELECT *,
				(SELECT COUNT(1)
					FROM label_membership lm JOIN hosts h ON (lm.host_id = h.id)
					WHERE label_id = l.id AND %s
				) AS host_count
			FROM labels l
			WHERE (
				MATCH(name) AGAINST(? IN BOOLEAN MODE)
			)
			AND id NOT IN (?)
			ORDER BY label_type DESC, id ASC
		`, ds.whereFilterHostsByTeams(filter, "h"),
	)

	sql, args, err := sqlx.In(sqlStatement, transformedQuery, omit)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query for labels with omits")
	}

	sql = ds.reader(ctx).Rebind(sql)

	matches := []*fleet.Label{}
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &matches, sql, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting labels with omits")
	}

	matches, err = ds.addAllHostsLabelToList(ctx, filter, matches, omit...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "adding all hosts label to matches")
	}

	return matches, nil
}

// When we search labels, we always want to make sure that the All Hosts label
// is included in the results set. Sometimes it already is and we don't need to
// add it, sometimes it's not so we explicitly add it.
func (ds *Datastore) addAllHostsLabelToList(ctx context.Context, filter fleet.TeamFilter, labels []*fleet.Label, omit ...uint) ([]*fleet.Label, error) {
	sql := fmt.Sprintf(`
			SELECT *,
				(SELECT COUNT(1)
					FROM label_membership lm JOIN hosts h ON (lm.host_id = h.id)
					WHERE label_id = l.id AND %s
				) AS host_count
			FROM labels l
			WHERE
			  label_type=?
				AND name = 'All Hosts'
			LIMIT 1
		`, ds.whereFilterHostsByTeams(filter, "h"),
	)

	var allHosts fleet.Label
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &allHosts, sql, fleet.LabelTypeBuiltIn); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get all hosts label")
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

	return append(labels, &allHosts), nil
}

func (ds *Datastore) searchLabelsDefault(ctx context.Context, filter fleet.TeamFilter, omit ...uint) ([]*fleet.Label, error) {
	sql := fmt.Sprintf(`
			SELECT *,
				(SELECT COUNT(1)
					FROM label_membership lm JOIN hosts h ON (lm.host_id = h.id)
					WHERE label_id = l.id AND %s
				) AS host_count
			FROM labels l
			WHERE id NOT IN (?)
			GROUP BY id
			ORDER BY label_type DESC, id ASC
		`, ds.whereFilterHostsByTeams(filter, "h"),
	)

	var in interface{}
	{
		// use -1 if there are no values to omit.
		// Avoids empty args error for `sqlx.In`
		in = omit
		if len(omit) == 0 {
			in = -1
		}
	}

	var labels []*fleet.Label
	sql, args, err := sqlx.In(sql, in)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "searching default labels")
	}
	sql = ds.reader(ctx).Rebind(sql)
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "searching default labels rebound")
	}

	labels, err = ds.addAllHostsLabelToList(ctx, filter, labels, omit...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting all host label")
	}

	return labels, nil
}

// SearchLabels performs wildcard searches on fleet.Label name
func (ds *Datastore) SearchLabels(ctx context.Context, filter fleet.TeamFilter, query string, omit ...uint) ([]*fleet.Label, error) {
	transformedQuery := transformQuery(query)
	if !queryMinLength(transformedQuery) {
		return ds.searchLabelsDefault(ctx, filter, omit...)
	}
	if len(omit) > 0 {
		return ds.searchLabelsWithOmits(ctx, filter, query, omit...)
	}

	// Ordering first by label_type ensures that built-in labels come
	// first. We will probably need to make a custom ordering function here
	// if additional label types are added. Ordering next by ID ensures
	// that the order is always consistent.
	sql := fmt.Sprintf(`
			SELECT *,
				(SELECT COUNT(1)
						FROM label_membership lm JOIN hosts h ON (lm.host_id = h.id)
						WHERE label_id = l.id AND %s
					) AS host_count
				FROM labels l
			WHERE (
				MATCH(name) AGAINST(? IN BOOLEAN MODE)
			)
			ORDER BY label_type DESC, id ASC
		`, ds.whereFilterHostsByTeams(filter, "h"),
	)

	matches := []*fleet.Label{}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &matches, sql, transformedQuery); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting labels for search")
	}

	matches, err := ds.addAllHostsLabelToList(ctx, filter, matches, omit...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "adding all hosts label to matches")
	}

	return matches, nil
}

func (ds *Datastore) LabelIDsByName(ctx context.Context, labels []string) ([]uint, error) {
	if len(labels) == 0 {
		return []uint{}, nil
	}

	sqlStatement := `
		SELECT id FROM labels
		WHERE name IN (?)
	`

	sql, args, err := sqlx.In(sqlStatement, labels)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query to get label IDs")
	}

	var labelIDs []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labelIDs, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get label IDs")
	}

	return labelIDs, nil
}

// AsyncBatchInsertLabelMembership inserts into the label_membership table the
// batch of label_id + host_id tuples represented by the [2]uint array.
func (ds *Datastore) AsyncBatchInsertLabelMembership(ctx context.Context, batch [][2]uint) error {
	// NOTE: this is tested via the server/service/async package tests.

	sql := `INSERT INTO label_membership (label_id, host_id) VALUES `
	sql += strings.Repeat(`(?, ?),`, len(batch))
	sql = strings.TrimSuffix(sql, ",")
	sql += ` ON DUPLICATE KEY UPDATE updated_at = VALUES(updated_at)`

	vals := make([]interface{}, 0, len(batch)*2)
	for _, tup := range batch {
		vals = append(vals, tup[0], tup[1])
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, sql, vals...)
		return ctxerr.Wrap(ctx, err, "insert into label_membership")
	})
}

// AsyncBatchDeleteLabelMembership deletes from the label_membership table the
// batch of label_id + host_id tuples represented by the [2]uint array.
func (ds *Datastore) AsyncBatchDeleteLabelMembership(ctx context.Context, batch [][2]uint) error {
	// NOTE: this is tested via the server/service/async package tests.

	rest := strings.Repeat(`UNION ALL SELECT ?, ? `, len(batch)-1)
	sql := fmt.Sprintf(`
    DELETE
      lm
    FROM
      label_membership lm
    JOIN
      (SELECT ? label_id, ? host_id %s) del_list
    ON
      lm.label_id = del_list.label_id AND
      lm.host_id = del_list.host_id`, rest)

	vals := make([]interface{}, 0, len(batch)*2)
	for _, tup := range batch {
		vals = append(vals, tup[0], tup[1])
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, sql, vals...)
		return ctxerr.Wrap(ctx, err, "delete from label_membership")
	})
}

// AsyncBatchUpdateLabelTimestamp updates the hosts' label_updated_at timestamp
// for the batch of host ids provided.
func (ds *Datastore) AsyncBatchUpdateLabelTimestamp(ctx context.Context, ids []uint, ts time.Time) error {
	// NOTE: this is tested via the server/service/async package tests.
	sql := `
      UPDATE
        hosts
      SET
        label_updated_at = ?
      WHERE
        id IN (?)`
	query, args, err := sqlx.In(sql, ts, ids)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building query to update hosts.label_updated_at")
	}
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return ctxerr.Wrap(ctx, err, "update hosts.label_updated_at")
	})
}

func amountLabelsDB(ctx context.Context, db sqlx.QueryerContext) (int, error) {
	var amount int
	err := sqlx.GetContext(ctx, db, &amount, `SELECT count(*) FROM labels`)
	if err != nil {
		return 0, err
	}
	return amount, nil
}

func (ds *Datastore) LabelsSummary(ctx context.Context) ([]*fleet.LabelSummary, error) {
	labelsSummary := []*fleet.LabelSummary{}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labelsSummary, "SELECT id, name, description, label_type FROM labels"); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "labels summary")
	}
	return labelsSummary, nil
}

// HostMemberOfAllLabels returns whether the given host is a member of all the provided labels.
// If the labels do not exist, then the host is considered not a member of the provided labels.
// A host will always be a member of an empty label set, so this method returns (true, nil)
// if labelNames is empty.
func (ds *Datastore) HostMemberOfAllLabels(ctx context.Context, hostID uint, labelNames []string) (bool, error) {
	if len(labelNames) == 0 {
		return true, nil
	}

	sqlStatement := `
		SELECT COUNT(*) = ? FROM labels l
		LEFT JOIN (SELECT label_id FROM label_membership WHERE host_id = ?) lm
		ON l.id = lm.label_id
		WHERE l.name IN (?) AND lm.label_id IS NOT NULL;
	`
	sql, args, err := sqlx.In(sqlStatement, len(labelNames), hostID, labelNames)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "building query to get label IDs")
	}

	var ok bool
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &ok, sql, args...); err != nil {
		return false, ctxerr.Wrap(ctx, err, "get label IDs")
	}

	return ok, nil
}
