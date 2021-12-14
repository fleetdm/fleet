package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/doug-martin/goqu/v9"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"
	"github.com/jmoiron/sqlx"
)

var hostSearchColumns = []string{"hostname", "uuid", "hardware_serial", "primary_ip"}

func (d *Datastore) NewHost(ctx context.Context, host *fleet.Host) (*fleet.Host, error) {
	sqlStatement := `
	INSERT INTO hosts (
		osquery_host_id,
		detail_updated_at,
		label_updated_at,
		policy_updated_at,
		node_key,
		hostname,
		uuid,
		platform,
		osquery_version,
		os_version,
		uptime,
		memory,
		team_id
	)
	VALUES( ?,?,?,?,?,?,?,?,?,?,?,?,? )
	`
	result, err := d.writer.ExecContext(
		ctx,
		sqlStatement,
		host.OsqueryHostID,
		host.DetailUpdatedAt,
		host.LabelUpdatedAt,
		host.PolicyUpdatedAt,
		host.NodeKey,
		host.Hostname,
		host.UUID,
		host.Platform,
		host.OsqueryVersion,
		host.OSVersion,
		host.Uptime,
		host.Memory,
		host.TeamID,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new host")
	}
	id, _ := result.LastInsertId()
	host.ID = uint(id)

	_, err = d.writer.ExecContext(ctx, `INSERT INTO host_seen_times (host_id, seen_time) VALUES (?,?)`, host.ID, host.SeenTime)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new host seen time")
	}

	return host, nil
}

func (d *Datastore) SerialSaveHost(ctx context.Context, host *fleet.Host) error {
	errCh := make(chan error, 1)
	defer close(errCh)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case d.writeCh <- itemToWrite{
		ctx:   ctx,
		errCh: errCh,
		item:  host,
	}:
		return <-errCh
	}
}

func (d *Datastore) SaveHost(ctx context.Context, host *fleet.Host) error {
	sqlStatement := `
		UPDATE hosts SET
			detail_updated_at = ?,
			label_updated_at = ?,
			policy_updated_at = ?,
			node_key = ?,
			hostname = ?,
			uuid = ?,
			platform = ?,
			osquery_version = ?,
			os_version = ?,
			uptime = ?,
			memory = ?,
			cpu_type = ?,
			cpu_subtype = ?,
			cpu_brand = ?,
			cpu_physical_cores = ?,
			hardware_vendor = ?,
			hardware_model = ?,
			hardware_version = ?,
			hardware_serial = ?,
			computer_name = ?,
			build = ?,
			platform_like = ?,
			code_name = ?,
			cpu_logical_cores = ?,
			distributed_interval = ?,
			config_tls_refresh = ?,
			logger_tls_period = ?,
			team_id = ?,
			primary_ip = ?,
			primary_mac = ?,
			refetch_requested = ?,
			gigs_disk_space_available = ?,
			percent_disk_space_available = ?
		WHERE id = ?
	`
	_, err := d.writer.ExecContext(ctx, sqlStatement,
		host.DetailUpdatedAt,
		host.LabelUpdatedAt,
		host.PolicyUpdatedAt,
		host.NodeKey,
		host.Hostname,
		host.UUID,
		host.Platform,
		host.OsqueryVersion,
		host.OSVersion,
		host.Uptime,
		host.Memory,
		host.CPUType,
		host.CPUSubtype,
		host.CPUBrand,
		host.CPUPhysicalCores,
		host.HardwareVendor,
		host.HardwareModel,
		host.HardwareVersion,
		host.HardwareSerial,
		host.ComputerName,
		host.Build,
		host.PlatformLike,
		host.CodeName,
		host.CPULogicalCores,
		host.DistributedInterval,
		host.ConfigTLSRefresh,
		host.LoggerTLSPeriod,
		host.TeamID,
		host.PrimaryIP,
		host.PrimaryMac,
		host.RefetchRequested,
		host.GigsDiskSpaceAvailable,
		host.PercentDiskSpaceAvailable,
		host.ID,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "save host with id %d", host.ID)
	}

	// Save host pack stats only if it is non-nil. Empty stats should be
	// represented by an empty slice.
	if host.PackStats != nil {
		if err := saveHostPackStatsDB(ctx, d.writer, host); err != nil {
			return err
		}
	}

	ac, err := d.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to get app config to see if we need to update host users and inventory")
	}

	if host.HostSoftware.Modified && ac.HostSettings.EnableSoftwareInventory && len(host.HostSoftware.Software) > 0 {
		if err := saveHostSoftwareDB(ctx, d.writer, host); err != nil {
			return ctxerr.Wrap(ctx, err, "failed to save host software")
		}
	}

	if host.Modified {
		if host.Additional != nil {
			if err := saveHostAdditionalDB(ctx, d.writer, host); err != nil {
				return ctxerr.Wrap(ctx, err, "failed to save host additional")
			}
		}

		if ac.HostSettings.EnableHostUsers && len(host.Users) > 0 {
			if err := saveHostUsersDB(ctx, d.writer, host); err != nil {
				return ctxerr.Wrap(ctx, err, "failed to save host users")
			}
		}
	}

	host.Modified = false
	return nil
}

func saveHostPackStatsDB(ctx context.Context, db sqlx.ExecerContext, host *fleet.Host) error {
	// Bulk insert software entries
	var args []interface{}
	queryCount := 0
	for _, pack := range host.PackStats {
		for _, query := range pack.QueryStats {
			queryCount++

			args = append(args,
				query.PackName,
				query.ScheduledQueryName,
				host.ID,
				query.AverageMemory,
				query.Denylisted,
				query.Executions,
				query.Interval,
				query.LastExecuted,
				query.OutputSize,
				query.SystemTime,
				query.UserTime,
				query.WallTime,
			)
		}
	}

	if queryCount == 0 {
		return nil
	}

	values := strings.TrimSuffix(strings.Repeat("((SELECT sq.id FROM scheduled_queries sq JOIN packs p ON (sq.pack_id = p.id) WHERE p.name = ? AND sq.name = ?),?,?,?,?,?,?,?,?,?,?),", queryCount), ",")
	sql := fmt.Sprintf(`
			INSERT IGNORE INTO scheduled_query_stats (
				scheduled_query_id,
				host_id,
				average_memory,
				denylisted,
				executions,
				schedule_interval,
				last_executed,
				output_size,
				system_time,
				user_time,
				wall_time
			)
			VALUES %s ON DUPLICATE KEY UPDATE
				scheduled_query_id = VALUES(scheduled_query_id),
				host_id = VALUES(host_id),
				average_memory = VALUES(average_memory),
				denylisted = VALUES(denylisted),
				executions = VALUES(executions),
				schedule_interval = VALUES(schedule_interval),
				last_executed = VALUES(last_executed),
				output_size = VALUES(output_size),
				system_time = VALUES(system_time),
				user_time = VALUES(user_time),
				wall_time = VALUES(wall_time)
		`, values)
	if _, err := db.ExecContext(ctx, sql, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert pack stats")
	}
	return nil
}

// MySQL is really particular about using zero values or old values for
// timestamps, so we set a default value that is plenty far in the past, but
// hopefully accepted by most MySQL configurations.
//
// NOTE: #3229 proposes a better fix that uses *time.Time for
// ScheduledQueryStats.LastExecuted.
var pastDate = "2000-01-01T00:00:00Z"

// loadhostPacksStatsDB will load all the pack stats for the given host. The scheduled
// queries that haven't run yet are returned with zero values.
func loadHostPackStatsDB(ctx context.Context, db sqlx.QueryerContext, hid uint, hostPlatform string) ([]fleet.PackStats, error) {
	packs, err := listPacksForHost(ctx, db, hid)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "list packs for host: %d", hid)
	}
	if len(packs) == 0 {
		return nil, nil
	}
	packIDs := make([]uint, len(packs))
	packTypes := make(map[uint]*string)
	for i := range packs {
		packIDs[i] = packs[i].ID
		packTypes[packs[i].ID] = packs[i].Type
	}
	ds := dialect.From(goqu.I("scheduled_queries").As("sq")).Select(
		goqu.I("sq.name").As("scheduled_query_name"),
		goqu.I("sq.id").As("scheduled_query_id"),
		goqu.I("sq.query_name").As("query_name"),
		goqu.I("q.description").As("description"),
		goqu.I("p.name").As("pack_name"),
		goqu.I("p.id").As("pack_id"),
		goqu.COALESCE(goqu.I("sqs.average_memory"), 0).As("average_memory"),
		goqu.COALESCE(goqu.I("sqs.denylisted"), false).As("denylisted"),
		goqu.COALESCE(goqu.I("sqs.executions"), 0).As("executions"),
		goqu.I("sq.interval").As("schedule_interval"),
		goqu.COALESCE(goqu.I("sqs.last_executed"), goqu.L("timestamp(?)", pastDate)).As("last_executed"),
		goqu.COALESCE(goqu.I("sqs.output_size"), 0).As("output_size"),
		goqu.COALESCE(goqu.I("sqs.system_time"), 0).As("system_time"),
		goqu.COALESCE(goqu.I("sqs.user_time"), 0).As("user_time"),
		goqu.COALESCE(goqu.I("sqs.wall_time"), 0).As("wall_time"),
	).Join(
		dialect.From("packs").As("p").Select(
			goqu.I("id"),
			goqu.I("name"),
		).Where(goqu.I("id").In(packIDs)),
		goqu.On(goqu.I("sq.pack_id").Eq(goqu.I("p.id"))),
	).Join(
		goqu.I("queries").As("q"),
		goqu.On(goqu.I("sq.query_name").Eq(goqu.I("q.name"))),
	).LeftJoin(
		dialect.From("scheduled_query_stats").As("sqs").Where(
			goqu.I("host_id").Eq(hid),
		),
		goqu.On(goqu.I("sqs.scheduled_query_id").Eq(goqu.I("sq.id"))),
	).Where(
		goqu.Or(
			// sq.platform empty or NULL means the scheduled query is set to
			// run on all hosts.
			goqu.I("sq.platform").Eq(""),
			goqu.I("sq.platform").IsNull(),
			// scheduled_queries.platform can be a comma-separated list of
			// platforms, e.g. "darwin,windows".
			goqu.L("FIND_IN_SET(?, sq.platform)", fleet.PlatformFromHost(hostPlatform)).Neq(0),
		),
	)
	sql, args, err := ds.ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sql build")
	}
	var stats []fleet.ScheduledQueryStats
	if err := sqlx.SelectContext(ctx, db, &stats, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load pack stats")
	}
	packStats := map[uint]fleet.PackStats{}
	for _, query := range stats {
		pack := packStats[query.PackID]
		pack.PackName = query.PackName
		pack.PackID = query.PackID
		pack.Type = getPackTypeFromDBField(packTypes[pack.PackID])
		pack.QueryStats = append(pack.QueryStats, query)
		packStats[pack.PackID] = pack
	}
	var ps []fleet.PackStats
	for _, pack := range packStats {
		ps = append(ps, pack)
	}
	return ps, nil
}

func getPackTypeFromDBField(t *string) string {
	if t == nil {
		return "pack"
	}
	return *t
}

func loadHostUsersDB(ctx context.Context, db sqlx.QueryerContext, host *fleet.Host) error {
	sql := `SELECT username, groupname, uid, user_type, shell FROM host_users WHERE host_id = ? and removed_at IS NULL`
	if err := sqlx.SelectContext(ctx, db, &host.Users, sql, host.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "load host users")
	}
	return nil
}

func (d *Datastore) DeleteHost(ctx context.Context, hid uint) error {
	err := d.deleteEntity(ctx, hostsTable, hid)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "deleting host with id %d", hid)
	}

	_, err = d.writer.ExecContext(ctx, `DELETE FROM host_seen_times WHERE host_id=?`, hid)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting host seen times")
	}

	_, err = d.writer.ExecContext(ctx, `DELETE FROM host_software WHERE host_id=?`, hid)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting host seen times")
	}

	return nil
}

func (d *Datastore) Host(ctx context.Context, id uint, skipLoadingExtras bool) (*fleet.Host, error) {
	policiesColumns := `,
		       coalesce(failing_policies.count, 0) as failing_policies_count,
		       coalesce(failing_policies.count, 0) as total_issues_count`
	policiesJoin := `
			JOIN (
		    	SELECT count(*) as count FROM policy_membership WHERE passes=0 AND host_id=?
			) failing_policies`
	args := []interface{}{id, id}
	if skipLoadingExtras {
		policiesColumns = ""
		policiesJoin = ""
		args = []interface{}{id}
	}
	sqlStatement := fmt.Sprintf(`
		SELECT
		       h.*,
		       COALESCE(hst.seen_time, h.created_at) AS seen_time,
		       t.name AS team_name,
		       (SELECT additional FROM host_additional WHERE host_id = h.id) AS additional
				%s
		FROM hosts h
			LEFT JOIN teams t ON (h.team_id = t.id)
			LEFT JOIN host_seen_times hst ON (h.id = hst.host_id)
			%s
		WHERE h.id = ?
		LIMIT 1`, policiesColumns, policiesJoin)
	host := &fleet.Host{}
	err := sqlx.GetContext(ctx, d.reader, host, sqlStatement, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host by id")
	}

	packStats, err := loadHostPackStatsDB(ctx, d.reader, host.ID, host.Platform)
	if err != nil {
		return nil, err
	}
	host.PackStats = packStats

	if err := loadHostUsersDB(ctx, d.reader, host); err != nil {
		return nil, err
	}

	return host, nil
}

func amountEnrolledHostsDB(db sqlx.Queryer) (int, error) {
	var amount int
	err := sqlx.Get(db, &amount, `SELECT count(*) FROM hosts`)
	if err != nil {
		return 0, err
	}
	return amount, nil
}

func (d *Datastore) ListHosts(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
	sql := `SELECT
		h.*,
		COALESCE(hst.seen_time, h.created_at) AS seen_time,
		t.name AS team_name
		`

	failingPoliciesSelect := `,
		coalesce(failing_policies.count, 0) as failing_policies_count,
		coalesce(failing_policies.count, 0) as total_issues_count
`
	if opt.DisableFailingPolicies {
		failingPoliciesSelect = ""
	}
	sql += failingPoliciesSelect

	var params []interface{}

	// Only include "additional" if filter provided.
	if len(opt.AdditionalFilters) == 1 && opt.AdditionalFilters[0] == "*" {
		// All info requested.
		sql += `
		, (SELECT additional FROM host_additional WHERE host_id = h.id) AS additional
		`
	} else if len(opt.AdditionalFilters) > 0 {
		// Filter specific columns.
		sql += `, (SELECT JSON_OBJECT(
			`
		for _, field := range opt.AdditionalFilters {
			sql += `?, JSON_EXTRACT(additional, ?), `
			params = append(params, field, fmt.Sprintf(`$."%s"`, field))
		}
		sql = sql[:len(sql)-2]
		sql += `
		    ) FROM host_additional WHERE host_id = h.id) AS additional
		    `
	}

	sql, params = d.applyHostFilters(opt, sql, filter, params)

	hosts := []*fleet.Host{}
	if err := sqlx.SelectContext(ctx, d.reader, &hosts, sql, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list hosts")
	}

	return hosts, nil
}

func (d *Datastore) applyHostFilters(opt fleet.HostListOptions, sql string, filter fleet.TeamFilter, params []interface{}) (string, []interface{}) {
	policyMembershipJoin := "JOIN policy_membership pm ON (h.id=pm.host_id)"
	if opt.PolicyIDFilter == nil {
		policyMembershipJoin = ""
	} else if opt.PolicyResponseFilter == nil {
		policyMembershipJoin = "LEFT " + policyMembershipJoin
	}

	softwareFilter := "TRUE"
	if opt.SoftwareIDFilter != nil {
		softwareFilter = "EXISTS (SELECT 1 FROM host_software hs WHERE hs.host_id=h.id AND hs.software_id=?)"
		params = append(params, opt.SoftwareIDFilter)
	}

	failingPoliciesJoin := `LEFT JOIN (
		    SELECT host_id, count(*) as count FROM policy_membership WHERE passes=0
		    GROUP BY host_id
		) as failing_policies ON (h.id=failing_policies.host_id)`
	if opt.DisableFailingPolicies {
		failingPoliciesJoin = ""
	}

	sql += fmt.Sprintf(`FROM hosts h
		LEFT JOIN host_seen_times hst ON (h.id=hst.host_id)
		LEFT JOIN teams t ON (h.team_id = t.id)
		%s
		%s
		WHERE TRUE AND %s AND %s
    `, policyMembershipJoin, failingPoliciesJoin, d.whereFilterHostsByTeams(filter, "h"), softwareFilter,
	)

	sql, params = filterHostsByStatus(sql, opt, params)
	sql, params = filterHostsByTeam(sql, opt, params)
	sql, params = filterHostsByPolicy(sql, opt, params)
	sql, params = searchLike(sql, params, opt.MatchQuery, hostSearchColumns...)
	sql, params = appendListOptionsWithCursorToSQL(sql, params, opt.ListOptions)

	return sql, params
}

func filterHostsByTeam(sql string, opt fleet.HostListOptions, params []interface{}) (string, []interface{}) {
	if opt.TeamFilter != nil {
		sql += ` AND h.team_id = ?`
		params = append(params, *opt.TeamFilter)
	}
	return sql, params
}

func filterHostsByPolicy(sql string, opt fleet.HostListOptions, params []interface{}) (string, []interface{}) {
	if opt.PolicyIDFilter != nil && opt.PolicyResponseFilter != nil {
		sql += ` AND pm.policy_id = ? AND pm.passes = ?`
		params = append(params, *opt.PolicyIDFilter, *opt.PolicyResponseFilter)
	} else if opt.PolicyIDFilter != nil && opt.PolicyResponseFilter == nil {
		sql += ` AND (pm.policy_id = ? OR pm.policy_id IS NULL) AND pm.passes IS NULL`
		params = append(params, *opt.PolicyIDFilter)
	}
	return sql, params
}

func filterHostsByStatus(sql string, opt fleet.HostListOptions, params []interface{}) (string, []interface{}) {
	switch opt.StatusFilter {
	case "new":
		sql += "AND DATE_ADD(h.created_at, INTERVAL 1 DAY) >= ?"
		params = append(params, time.Now())
	case "online":
		sql += fmt.Sprintf("AND DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL LEAST(h.distributed_interval, h.config_tls_refresh) + %d SECOND) > ?", fleet.OnlineIntervalBuffer)
		params = append(params, time.Now())
	case "offline":
		sql += fmt.Sprintf("AND DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL LEAST(h.distributed_interval, h.config_tls_refresh) + %d SECOND) <= ? AND DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL 30 DAY) >= ?", fleet.OnlineIntervalBuffer)
		params = append(params, time.Now(), time.Now())
	case "mia":
		sql += "AND DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL 30 DAY) <= ?"
		params = append(params, time.Now())
	}
	return sql, params
}

func (d *Datastore) CountHosts(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) (int, error) {
	sql := `SELECT count(*) `

	// ignore pagination in count
	opt.Page = 0
	opt.PerPage = 0

	var params []interface{}
	sql, params = d.applyHostFilters(opt, sql, filter, params)

	var count int
	if err := sqlx.GetContext(ctx, d.reader, &count, sql, params...); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count hosts")
	}

	return count, nil
}

func (d *Datastore) CleanupIncomingHosts(ctx context.Context, now time.Time) error {
	sqlStatement := `
		DELETE FROM hosts
		WHERE hostname = '' AND osquery_version = ''
		AND created_at < (? - INTERVAL 5 MINUTE)
	`
	if _, err := d.writer.ExecContext(ctx, sqlStatement, now); err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup incoming hosts")
	}

	return nil
}

func (d *Datastore) GenerateHostStatusStatistics(ctx context.Context, filter fleet.TeamFilter, now time.Time) (*fleet.HostSummary, error) {
	// The logic in this function should remain synchronized with
	// host.Status and CountHostsInTargets - that is, the intervals associated
	// with each status must be the same.

	whereClause := d.whereFilterHostsByTeams(filter, "h")
	sqlStatement := fmt.Sprintf(`
			SELECT
				COUNT(*) total,
				COALESCE(SUM(CASE WHEN DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL 30 DAY) <= ? THEN 1 ELSE 0 END), 0) mia,
				COALESCE(SUM(CASE WHEN DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL LEAST(distributed_interval, config_tls_refresh) + %d SECOND) <= ? AND DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL 30 DAY) >= ? THEN 1 ELSE 0 END), 0) offline,
				COALESCE(SUM(CASE WHEN DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL LEAST(distributed_interval, config_tls_refresh) + %d SECOND) > ? THEN 1 ELSE 0 END), 0) online,
				COALESCE(SUM(CASE WHEN DATE_ADD(created_at, INTERVAL 1 DAY) >= ? THEN 1 ELSE 0 END), 0) new
			FROM hosts h LEFT JOIN host_seen_times hst ON (h.id=hst.host_id) WHERE %s
			LIMIT 1;
		`, fleet.OnlineIntervalBuffer, fleet.OnlineIntervalBuffer, whereClause)

	summary := fleet.HostSummary{TeamID: filter.TeamID}
	err := sqlx.GetContext(ctx, d.reader, &summary, sqlStatement, now, now, now, now, now)
	if err != nil && err != sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, err, "generating host statistics")
	}

	// get the counts per platform, the `h` alias for hosts is required so that
	// reusing the whereClause is ok.
	sqlStatement = fmt.Sprintf(`
			SELECT
			  COUNT(*) total,
			  h.platform
			FROM hosts h
			WHERE %s
			GROUP BY h.platform
		`, whereClause)

	var platforms []*fleet.HostSummaryPlatform
	err = sqlx.SelectContext(ctx, d.reader, &platforms, sqlStatement)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating host platforms statistics")
	}
	summary.Platforms = platforms

	return &summary, nil
}

// EnrollHost enrolls a host
func (d *Datastore) EnrollHost(ctx context.Context, osqueryHostID, nodeKey string, teamID *uint, cooldown time.Duration) (*fleet.Host, error) {
	if osqueryHostID == "" {
		return nil, ctxerr.New(ctx, "missing osquery host identifier")
	}

	var host fleet.Host
	err := d.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		zeroTime := time.Unix(0, 0).Add(24 * time.Hour)

		var hostID int64
		err := sqlx.GetContext(ctx, tx, &host, `SELECT id, last_enrolled_at FROM hosts WHERE osquery_host_id = ?`, osqueryHostID)
		switch {
		case err != nil && !errors.Is(err, sql.ErrNoRows):
			return ctxerr.Wrap(ctx, err, "check existing")
		case errors.Is(err, sql.ErrNoRows):
			// Create new host record
			sqlInsert := `
				INSERT INTO hosts (
					detail_updated_at,
					label_updated_at,
					policy_updated_at,
					osquery_host_id,
					node_key,
					team_id
				) VALUES (?, ?, ?, ?, ?, ?)
			`
			result, err := tx.ExecContext(ctx, sqlInsert, zeroTime, zeroTime, zeroTime, osqueryHostID, nodeKey, teamID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "insert host")
			}
			hostID, _ = result.LastInsertId()
		default:
			// Prevent hosts from enrolling too often with the same identifier.
			// Prior to adding this we saw many hosts (probably VMs) with the
			// same identifier competing for enrollment and causing perf issues.
			if cooldown > 0 && time.Since(host.LastEnrolledAt) < cooldown {
				return backoff.Permanent(ctxerr.Errorf(ctx, "host identified by %s enrolling too often", osqueryHostID))
			}
			hostID = int64(host.ID)
			// Update existing host record
			sqlUpdate := `
				UPDATE hosts
				SET node_key = ?,
				team_id = ?,
				last_enrolled_at = NOW()
				WHERE osquery_host_id = ?
			`
			_, err := tx.ExecContext(ctx, sqlUpdate, nodeKey, teamID, osqueryHostID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "update host")
			}
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO host_seen_times (host_id, seen_time) VALUES (?,?)
			ON DUPLICATE KEY UPDATE seen_time = VALUES(seen_time)`,
			hostID, time.Now().UTC())
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new host seen time")
		}
		sqlSelect := `
			SELECT * FROM hosts WHERE id = ? LIMIT 1
		`
		err = sqlx.GetContext(ctx, tx, &host, sqlSelect, hostID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting the host to return")
		}
		_, err = tx.ExecContext(ctx, `INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, (SELECT id FROM labels WHERE name = 'All Hosts' AND label_type = 1))`, hostID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert new host into all hosts label")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func (d *Datastore) AuthenticateHost(ctx context.Context, nodeKey string) (*fleet.Host, error) {
	// Select everything besides `additional`
	sqlStatement := `SELECT * FROM hosts WHERE node_key = ? LIMIT 1`

	host := &fleet.Host{}
	if err := sqlx.GetContext(ctx, d.reader, host, sqlStatement, nodeKey); err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, ctxerr.Wrap(ctx, notFound("Host"))
		default:
			return nil, ctxerr.Wrap(ctx, err, "find host")
		}
	}

	return host, nil
}

func (d *Datastore) MarkHostsSeen(ctx context.Context, hostIDs []uint, t time.Time) error {
	if len(hostIDs) == 0 {
		return nil
	}

	// Sort by host id to prevent deadlocks:
	// https://percona.community/blog/2018/09/24/minimize-mysql-deadlocks-3-steps/
	// https://dev.mysql.com/doc/refman/5.7/en/innodb-deadlocks-handling.html
	sort.Slice(hostIDs, func(i, j int) bool { return hostIDs[i] < hostIDs[j] })

	if err := d.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var insertArgs []interface{}
		for _, hostID := range hostIDs {
			insertArgs = append(insertArgs, hostID, t)
		}
		insertValues := strings.TrimSuffix(strings.Repeat("(?, ?),", len(hostIDs)), ",")
		query := fmt.Sprintf(`
			INSERT INTO host_seen_times (host_id, seen_time) VALUES %s
			ON DUPLICATE KEY UPDATE seen_time = VALUES(seen_time)`,
			insertValues,
		)
		if _, err := tx.ExecContext(ctx, query, insertArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "exec update")
		}
		return nil
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "MarkHostsSeen transaction")
	}

	return nil
}

// SearchHosts performs a search on the hosts table using the following criteria:
//	- Use the provided team filter.
//	- Full-text search with the "query" argument (if query == "", then no fulltext matching is executed).
// 	Full-text search is used even if "query" is a short or stopword.
//	(what defines a short word is the "ft_min_word_len" VARIABLE, set to 4 by default in Fleet deployments).
//	- An optional list of IDs to omit from the search.
func (d *Datastore) SearchHosts(ctx context.Context, filter fleet.TeamFilter, query string, omit ...uint) ([]*fleet.Host, error) {
	var sqlb strings.Builder
	sqlb.WriteString(`SELECT
		h.*,
		COALESCE(hst.seen_time, h.created_at) AS seen_time
	FROM hosts h
	LEFT JOIN host_seen_times hst
	ON (h.id=hst.host_id) WHERE`)

	var args []interface{}
	if len(query) > 0 {
		sqlb.WriteString(` (
				MATCH (hostname, uuid) AGAINST (? IN BOOLEAN MODE)
				OR MATCH (primary_ip, primary_mac) AGAINST (? IN BOOLEAN MODE)
			) AND`)
		// Transform query argument and append the truncation operator "*" for MATCH.
		// From Oracle docs: "If a word is specified with the truncation operator, it is not
		// stripped from a boolean query, even if it is too short or a stopword."
		hostQuery := transformQueryWithSuffix(query, "*")
		// Needs quotes to avoid each "." marking a word boundary.
		// TODO(lucas): Currently matching the primary_mac doesn't work, see #1959.
		ipQuery := `"` + query + `"`
		args = append(args, hostQuery, ipQuery)
	}
	var in interface{}
	// use -1 if there are no values to omit.
	// Avoids empty args error for `sqlx.In`
	in = omit
	if len(omit) == 0 {
		in = -1
	}
	args = append(args, in)
	sqlb.WriteString(" id NOT IN (?) AND ")
	sqlb.WriteString(d.whereFilterHostsByTeams(filter, "h"))
	sqlb.WriteString(` ORDER BY COALESCE(hst.seen_time, h.created_at) DESC LIMIT 10`)

	sql, args, err := sqlx.In(sqlb.String(), args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "searching default hosts")
	}
	sql = d.reader.Rebind(sql)
	hosts := []*fleet.Host{}
	if err := sqlx.SelectContext(ctx, d.reader, &hosts, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "searching hosts")
	}
	return hosts, nil
}

func (d *Datastore) HostIDsByName(ctx context.Context, filter fleet.TeamFilter, hostnames []string) ([]uint, error) {
	if len(hostnames) == 0 {
		return []uint{}, nil
	}

	sqlStatement := fmt.Sprintf(`
			SELECT id FROM hosts
			WHERE hostname IN (?) AND %s
		`, d.whereFilterHostsByTeams(filter, "hosts"),
	)

	sql, args, err := sqlx.In(sqlStatement, hostnames)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query to get host IDs")
	}

	var hostIDs []uint
	if err := sqlx.SelectContext(ctx, d.reader, &hostIDs, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host IDs")
	}

	return hostIDs, nil
}

func (d *Datastore) HostByIdentifier(ctx context.Context, identifier string) (*fleet.Host, error) {
	sql := `
		SELECT * FROM hosts
		WHERE ? IN (hostname, osquery_host_id, node_key, uuid)
		LIMIT 1
	`
	host := &fleet.Host{}
	err := sqlx.GetContext(ctx, d.reader, host, sql, identifier)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host by identifier")
	}

	packStats, err := loadHostPackStatsDB(ctx, d.reader, host.ID, host.Platform)
	if err != nil {
		return nil, err
	}
	host.PackStats = packStats

	return host, nil
}

func (d *Datastore) AddHostsToTeam(ctx context.Context, teamID *uint, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	return d.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// hosts can only be in one team, so if there's a policy that has a team id and a result from one of our hosts
		// it can only be from the previous team they are being transferred from
		query, args, err := sqlx.In(`DELETE FROM policy_membership
					WHERE policy_id IN (SELECT id FROM policies WHERE team_id IS NOT NULL) AND host_id IN (?)`, hostIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "add host to team sqlx in")
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "exec AddHostsToTeam delete policy membership")
		}

		query, args, err = sqlx.In(`UPDATE hosts SET team_id = ? WHERE id IN (?)`, teamID, hostIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "sqlx.In AddHostsToTeam")
		}

		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "exec AddHostsToTeam")
		}

		return nil
	})
}

func saveHostAdditionalDB(ctx context.Context, exec sqlx.ExecerContext, host *fleet.Host) error {
	sql := `
		INSERT INTO host_additional (host_id, additional)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE additional = VALUES(additional)
	`
	if _, err := exec.ExecContext(ctx, sql, host.ID, host.Additional); err != nil {
		return ctxerr.Wrap(ctx, err, "insert additional")
	}

	return nil
}

func saveHostUsersDB(ctx context.Context, tx sqlx.ExtContext, host *fleet.Host) error {
	currentHost := &fleet.Host{ID: host.ID}
	if err := loadHostUsersDB(ctx, tx, currentHost); err != nil {
		return err
	}

	keyForUser := func(u *fleet.HostUser) string { return fmt.Sprintf("%d\x00%s", u.Uid, u.Username) }
	incomingUsers := make(map[string]bool)
	var insertArgs []interface{}
	for _, u := range host.Users {
		insertArgs = append(insertArgs, host.ID, u.Uid, u.Username, u.Type, u.GroupName, u.Shell)
		incomingUsers[keyForUser(&u)] = true
	}

	var removedArgs []interface{}
	for _, u := range currentHost.Users {
		if _, ok := incomingUsers[keyForUser(&u)]; !ok {
			removedArgs = append(removedArgs, u.Username)
		}
	}

	insertValues := strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?, ?),", len(host.Users)), ",")
	insertSql := fmt.Sprintf(
		`INSERT INTO host_users (host_id, uid, username, user_type, groupname, shell) 
				VALUES %s 
				ON DUPLICATE KEY UPDATE
				user_type = VALUES(user_type),
				groupname = VALUES(groupname),
				shell = VALUES(shell),
				removed_at=NULL`,
		insertValues,
	)
	if _, err := tx.ExecContext(ctx, insertSql, insertArgs...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert users")
	}

	if len(removedArgs) == 0 {
		return nil
	}
	removedValues := strings.TrimSuffix(strings.Repeat("?,", len(removedArgs)), ",")
	removedSql := fmt.Sprintf(
		`UPDATE host_users SET removed_at = CURRENT_TIMESTAMP WHERE host_id = ? and username IN (%s)`,
		removedValues,
	)
	if _, err := tx.ExecContext(ctx, removedSql, append([]interface{}{host.ID}, removedArgs...)...); err != nil {
		return ctxerr.Wrap(ctx, err, "mark users as removed")
	}

	return nil
}

func (d *Datastore) TotalAndUnseenHostsSince(ctx context.Context, daysCount int) (total int, unseen int, err error) {
	var counts struct {
		Total  int `db:"total"`
		Unseen int `db:"unseen"`
	}
	err = sqlx.GetContext(ctx, d.reader, &counts,
		`SELECT
			COUNT(*) as total,
			SUM(IF(DATEDIFF(CURRENT_DATE, COALESCE(hst.seen_time, h.created_at)) >= ?, 1, 0)) as unseen
		FROM hosts h
		LEFT JOIN host_seen_times hst
		ON h.id = hst.host_id`,
		daysCount,
	)
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "getting total and unseen host counts")
	}

	return counts.Total, counts.Unseen, nil
}

func (d *Datastore) DeleteHosts(ctx context.Context, ids []uint) error {
	_, err := d.deleteEntities(ctx, hostsTable, ids)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting hosts")
	}

	query, args, err := sqlx.In(`DELETE FROM host_seen_times WHERE host_id in (?)`, ids)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "building delete host_seen_times query")
	}

	_, err = d.writer.ExecContext(ctx, query, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting host seen times")
	}
	return nil
}

func (d *Datastore) ListPoliciesForHost(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
	if host.FleetPlatform() == "" {
		// We log to help troubleshooting in case this happens.
		level.Error(d.logger).Log("err", fmt.Sprintf("host %d with empty platform", host.ID))
	}
	query := `SELECT p.*,
		COALESCE(u.name, '<deleted>') AS author_name,
		COALESCE(u.email, '') AS author_email,
		CASE
			WHEN pm.passes = 1 THEN 'pass'
			WHEN pm.passes = 0 THEN 'fail'
			ELSE ''
		END AS response,
		coalesce(p.resolution, '') as resolution
	FROM policies p
	LEFT JOIN policy_membership pm ON (p.id=pm.policy_id AND host_id=?)
	LEFT JOIN users u ON p.author_id = u.id
	WHERE (p.team_id IS NULL OR p.team_id = (select team_id from hosts WHERE id = ?))
	AND (p.platforms IS NULL OR p.platforms = "" OR FIND_IN_SET(?, p.platforms) != 0)`

	var policies []*fleet.HostPolicy
	if err := sqlx.SelectContext(ctx, d.reader, &policies, query, host.ID, host.ID, host.FleetPlatform()); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host policies")
	}
	return policies, nil
}

func (d *Datastore) CleanupExpiredHosts(ctx context.Context) error {
	ac, err := appConfigDB(ctx, d.reader)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}
	if !ac.HostExpirySettings.HostExpiryEnabled {
		return nil
	}

	// Usual clean up queries used to be like this:
	// DELETE FROM hosts WHERE id in (SELECT host_id FROM host_seen_times WHERE seen_time < DATE_SUB(NOW(), INTERVAL ? DAY))
	// This means a full table scan for hosts, and for big deployments, that's not ideal
	// so instead, we get the ids one by one and delete things one by one
	// it might take longer, but it should lock only the row we need

	rows, err := d.writer.QueryContext(
		ctx,
		`SELECT h.id FROM hosts h
		LEFT JOIN host_seen_times hst
		ON h.id = hst.host_id
		WHERE COALESCE(hst.seen_time, h.created_at) < DATE_SUB(NOW(), INTERVAL ? DAY)`,
		ac.HostExpirySettings.HostExpiryWindow,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting expired host ids")
	}
	defer rows.Close()

	for rows.Next() {
		var id uint
		err := rows.Scan(&id)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "scanning expired host id")
		}
		_, err = d.writer.ExecContext(ctx, `DELETE FROM hosts WHERE id = ?`, id)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting expired hosts")
		}
		_, err = d.writer.ExecContext(ctx, `DELETE FROM host_software WHERE host_id = ?`, id)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting expired host software")
		}
	}
	if err := rows.Err(); err != nil {
		return ctxerr.Wrap(ctx, err, "expired hosts, row err")
	}

	_, err = d.writer.ExecContext(ctx, `DELETE FROM host_seen_times WHERE seen_time < DATE_SUB(NOW(), INTERVAL ? DAY)`, ac.HostExpirySettings.HostExpiryWindow)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting expired host seen times")
	}
	return nil
}
