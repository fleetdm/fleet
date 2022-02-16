package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
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

// NewHost creates a new host on the datastore.
//
// Currently only used for testing.
func (ds *Datastore) NewHost(ctx context.Context, host *fleet.Host) (*fleet.Host, error) {
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
		team_id,
		distributed_interval,
		logger_tls_period,
		config_tls_refresh,
		refetch_requested
	)
	VALUES( ?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,? )
	`
	result, err := ds.writer.ExecContext(
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
		host.DistributedInterval,
		host.LoggerTLSPeriod,
		host.ConfigTLSRefresh,
		host.RefetchRequested,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new host")
	}
	id, _ := result.LastInsertId()
	host.ID = uint(id)

	_, err = ds.writer.ExecContext(ctx, `INSERT INTO host_seen_times (host_id, seen_time) VALUES (?,?)`, host.ID, host.SeenTime)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new host seen time")
	}

	return host, nil
}

func (ds *Datastore) SerialUpdateHost(ctx context.Context, host *fleet.Host) error {
	errCh := make(chan error, 1)
	defer close(errCh)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ds.writeCh <- itemToWrite{
		ctx:   ctx,
		errCh: errCh,
		item:  host,
	}:
		return <-errCh
	}
}

func (ds *Datastore) SaveHost(ctx context.Context, host *fleet.Host) error {
	if err := ds.UpdateHost(ctx, host); err != nil {
		return err
	}

	// Save host pack stats only if it is non-nil. Empty stats should be
	// represented by an empty slice.
	if host.PackStats != nil {
		if err := saveHostPackStatsDB(ctx, ds.writer, host.ID, host.PackStats); err != nil {
			return err
		}
	}

	ac, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "failed to get app config to see if we need to update host users and inventory")
	}

	if host.HostSoftware.Modified && ac.HostSettings.EnableSoftwareInventory && len(host.HostSoftware.Software) > 0 {
		if err := saveHostSoftwareDB(ctx, ds.writer, host); err != nil {
			return ctxerr.Wrap(ctx, err, "failed to save host software")
		}
	}

	if host.Modified {
		if host.Additional != nil {
			if err := saveHostAdditionalDB(ctx, ds.writer, host.ID, host.Additional); err != nil {
				return ctxerr.Wrap(ctx, err, "failed to save host additional")
			}
		}

		if ac.HostSettings.EnableHostUsers && len(host.Users) > 0 {
			if err := saveHostUsersDB(ctx, ds.writer, host.ID, host.Users); err != nil {
				return ctxerr.Wrap(ctx, err, "failed to save host users")
			}
		}
	}

	host.Modified = false
	return nil
}

func (ds *Datastore) SaveHostPackStats(ctx context.Context, hostID uint, stats []fleet.PackStats) error {
	return saveHostPackStatsDB(ctx, ds.writer, hostID, stats)
}

func saveHostPackStatsDB(ctx context.Context, db sqlx.ExecerContext, hostID uint, stats []fleet.PackStats) error {
	var args []interface{}
	queryCount := 0
	for _, pack := range stats {
		for _, query := range pack.QueryStats {
			queryCount++

			args = append(args,
				query.PackName,
				query.ScheduledQueryName,
				hostID,
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

func loadHostUsersDB(ctx context.Context, db sqlx.QueryerContext, hostID uint) ([]fleet.HostUser, error) {
	sql := `SELECT username, groupname, uid, user_type, shell FROM host_users WHERE host_id = ? and removed_at IS NULL`
	var users []fleet.HostUser
	if err := sqlx.SelectContext(ctx, db, &users, sql, hostID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load host users")
	}
	return users, nil
}

func (ds *Datastore) DeleteHost(ctx context.Context, hid uint) error {
	delHostRef := func(tx sqlx.ExtContext, table string) error {
		_, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE host_id=?`, table), hid)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "deleting %s for host %d", table, hid)
		}
		return nil
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM hosts WHERE id = ?`, hid)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "delete host")
		}

		hostRefs := []string{
			"host_seen_times",
			"host_software",
			"host_users",
			"host_emails",
			"host_additional",
			"scheduled_query_stats",
			"label_membership",
			"policy_membership",
			"host_mdm",
			"host_munki_info",
		}

		for _, table := range hostRefs {
			err := delHostRef(tx, table)
			if err != nil {
				return err
			}
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM pack_targets WHERE type=? AND target_id=?`, fleet.TargetHost, hid)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "deleting pack_targets for host %d", hid)
		}

		return nil
	})
}

func (ds *Datastore) Host(ctx context.Context, id uint, skipLoadingExtras bool) (*fleet.Host, error) {
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
	err := sqlx.GetContext(ctx, ds.reader, host, sqlStatement, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Host").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host by id")
	}

	packStats, err := loadHostPackStatsDB(ctx, ds.reader, host.ID, host.Platform)
	if err != nil {
		return nil, err
	}
	host.PackStats = packStats

	users, err := loadHostUsersDB(ctx, ds.reader, host.ID)
	if err != nil {
		return nil, err
	}
	host.Users = users

	return host, nil
}

func amountEnrolledHostsDB(ctx context.Context, db sqlx.QueryerContext) (int, error) {
	var amount int
	err := sqlx.GetContext(ctx, db, &amount, `SELECT count(*) FROM hosts`)
	if err != nil {
		return 0, err
	}
	return amount, nil
}

func (ds *Datastore) ListHosts(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
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

	sql, params = ds.applyHostFilters(opt, sql, filter, params)

	hosts := []*fleet.Host{}
	if err := sqlx.SelectContext(ctx, ds.reader, &hosts, sql, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list hosts")
	}

	return hosts, nil
}

func (ds *Datastore) applyHostFilters(opt fleet.HostListOptions, sql string, filter fleet.TeamFilter, params []interface{}) (string, []interface{}) {
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
    `, policyMembershipJoin, failingPoliciesJoin, ds.whereFilterHostsByTeams(filter, "h"), softwareFilter,
	)

	sql, params = filterHostsByStatus(sql, opt, params)
	sql, params = filterHostsByTeam(sql, opt, params)
	sql, params = filterHostsByPolicy(sql, opt, params)
	sql, params = hostSearchLike(sql, params, opt.MatchQuery, hostSearchColumns...)
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

func (ds *Datastore) CountHosts(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) (int, error) {
	sql := `SELECT count(*) `

	// ignore pagination in count
	opt.Page = 0
	opt.PerPage = 0

	var params []interface{}
	sql, params = ds.applyHostFilters(opt, sql, filter, params)

	var count int
	if err := sqlx.GetContext(ctx, ds.reader, &count, sql, params...); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count hosts")
	}

	return count, nil
}

func (ds *Datastore) CleanupIncomingHosts(ctx context.Context, now time.Time) error {
	sqlStatement := `
		DELETE FROM hosts
		WHERE hostname = '' AND osquery_version = ''
		AND created_at < (? - INTERVAL 5 MINUTE)
	`
	if _, err := ds.writer.ExecContext(ctx, sqlStatement, now); err != nil {
		return ctxerr.Wrap(ctx, err, "cleanup incoming hosts")
	}

	return nil
}

func (ds *Datastore) GenerateHostStatusStatistics(ctx context.Context, filter fleet.TeamFilter, now time.Time, platform *string) (*fleet.HostSummary, error) {
	// The logic in this function should remain synchronized with
	// host.Status and CountHostsInTargets - that is, the intervals associated
	// with each status must be the same.

	args := []interface{}{now, now, now, now, now}
	whereClause := ds.whereFilterHostsByTeams(filter, "h")
	if platform != nil {
		whereClause += " AND h.platform IN (?) "
		args = append(args, fleet.ExpandPlatform(*platform))
	}
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

	stmt, args, err := sqlx.In(sqlStatement, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating host statistics statement")
	}
	summary := fleet.HostSummary{TeamID: filter.TeamID}
	err = sqlx.GetContext(ctx, ds.reader, &summary, stmt, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, err, "generating host statistics")
	}

	// get the counts per platform, the `h` alias for hosts is required so that
	// reusing the whereClause is ok.
	args = []interface{}{}
	if platform != nil {
		args = append(args, fleet.ExpandPlatform(*platform))
	}
	sqlStatement = fmt.Sprintf(`
			SELECT
			  COUNT(*) total,
			  h.platform
			FROM hosts h
			WHERE %s
			GROUP BY h.platform
		`, whereClause)

	var platforms []*fleet.HostSummaryPlatform
	stmt, args, err = sqlx.In(sqlStatement, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating host platforms statement")
	}
	err = sqlx.SelectContext(ctx, ds.reader, &platforms, stmt, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating host platforms statistics")
	}
	summary.Platforms = platforms

	return &summary, nil
}

// EnrollHost enrolls a host
func (ds *Datastore) EnrollHost(ctx context.Context, osqueryHostID, nodeKey string, teamID *uint, cooldown time.Duration) (*fleet.Host, error) {
	if osqueryHostID == "" {
		return nil, ctxerr.New(ctx, "missing osquery host identifier")
	}

	var host fleet.Host
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
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

// GetContextTryStmt will attempt to run sqlx.GetContext on a cached statement if available, resorting to ds.reader.
func (ds *Datastore) GetContextTryStmt(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	var err error
	//nolint the statements are closed in Datastore.Close.
	if stmt := ds.loadOrPrepareStmt(ctx, query); stmt != nil {
		err = stmt.GetContext(ctx, dest, args...)
	} else {
		err = sqlx.GetContext(ctx, ds.reader, dest, query, args...)
	}
	return err
}

// LoadHostByNodeKey loads the whole host identified by the node key.
// If the node key is invalid it returns a NotFoundError.
func (ds *Datastore) LoadHostByNodeKey(ctx context.Context, nodeKey string) (*fleet.Host, error) {
	query := `SELECT * FROM hosts WHERE node_key = ?`

	var host fleet.Host
	switch err := ds.GetContextTryStmt(ctx, &host, query, nodeKey); {
	case err == nil:
		return &host, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, ctxerr.Wrap(ctx, notFound("Host"))
	default:
		return nil, ctxerr.Wrap(ctx, err, "find host")
	}
}

func (ds *Datastore) MarkHostsSeen(ctx context.Context, hostIDs []uint, t time.Time) error {
	if len(hostIDs) == 0 {
		return nil
	}

	// Sort by host id to prevent deadlocks:
	// https://percona.community/blog/2018/09/24/minimize-mysql-deadlocks-3-steps/
	// https://dev.mysql.com/doc/refman/5.7/en/innodb-deadlocks-handling.html
	sort.Slice(hostIDs, func(i, j int) bool { return hostIDs[i] < hostIDs[j] })

	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
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
func (ds *Datastore) SearchHosts(ctx context.Context, filter fleet.TeamFilter, query string, omit ...uint) ([]*fleet.Host, error) {
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
	sqlb.WriteString(ds.whereFilterHostsByTeams(filter, "h"))
	sqlb.WriteString(` ORDER BY h.id DESC LIMIT 10`)

	sql, args, err := sqlx.In(sqlb.String(), args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "searching default hosts")
	}
	sql = ds.reader.Rebind(sql)
	hosts := []*fleet.Host{}
	if err := sqlx.SelectContext(ctx, ds.reader, &hosts, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "searching hosts")
	}
	return hosts, nil
}

func (ds *Datastore) HostIDsByName(ctx context.Context, filter fleet.TeamFilter, hostnames []string) ([]uint, error) {
	if len(hostnames) == 0 {
		return []uint{}, nil
	}

	sqlStatement := fmt.Sprintf(`
			SELECT id FROM hosts
			WHERE hostname IN (?) AND %s
		`, ds.whereFilterHostsByTeams(filter, "hosts"),
	)

	sql, args, err := sqlx.In(sqlStatement, hostnames)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query to get host IDs")
	}

	var hostIDs []uint
	if err := sqlx.SelectContext(ctx, ds.reader, &hostIDs, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host IDs")
	}

	return hostIDs, nil
}

func (ds *Datastore) HostByIdentifier(ctx context.Context, identifier string) (*fleet.Host, error) {
	stmt := `
		SELECT * FROM hosts
		WHERE ? IN (hostname, osquery_host_id, node_key, uuid)
		LIMIT 1
	`
	host := &fleet.Host{}
	err := sqlx.GetContext(ctx, ds.reader, host, stmt, identifier)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Host").WithName(identifier))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host by identifier")
	}

	packStats, err := loadHostPackStatsDB(ctx, ds.reader, host.ID, host.Platform)
	if err != nil {
		return nil, err
	}
	host.PackStats = packStats

	return host, nil
}

func (ds *Datastore) AddHostsToTeam(ctx context.Context, teamID *uint, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
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

func (ds *Datastore) SaveHostAdditional(ctx context.Context, hostID uint, additional *json.RawMessage) error {
	return saveHostAdditionalDB(ctx, ds.writer, hostID, additional)
}

func saveHostAdditionalDB(ctx context.Context, exec sqlx.ExecerContext, hostID uint, additional *json.RawMessage) error {
	sql := `
		INSERT INTO host_additional (host_id, additional)
		VALUES (?, ?)
		ON DUPLICATE KEY UPDATE additional = VALUES(additional)
	`
	if _, err := exec.ExecContext(ctx, sql, hostID, additional); err != nil {
		return ctxerr.Wrap(ctx, err, "insert additional")
	}
	return nil
}

func (ds *Datastore) SaveHostUsers(ctx context.Context, hostID uint, users []fleet.HostUser) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return saveHostUsersDB(ctx, tx, hostID, users)
	})
}

func saveHostUsersDB(ctx context.Context, tx sqlx.ExtContext, hostID uint, users []fleet.HostUser) error {
	currentHostUsers, err := loadHostUsersDB(ctx, tx, hostID)
	if err != nil {
		return err
	}

	keyForUser := func(u *fleet.HostUser) string { return fmt.Sprintf("%d\x00%s", u.Uid, u.Username) }
	incomingUsers := make(map[string]bool)
	var insertArgs []interface{}
	for _, u := range users {
		insertArgs = append(insertArgs, hostID, u.Uid, u.Username, u.Type, u.GroupName, u.Shell)
		incomingUsers[keyForUser(&u)] = true
	}

	var removedArgs []interface{}
	for _, u := range currentHostUsers {
		if _, ok := incomingUsers[keyForUser(&u)]; !ok {
			removedArgs = append(removedArgs, u.Username)
		}
	}

	insertValues := strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?, ?),", len(users)), ",")
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
	if _, err := tx.ExecContext(ctx, removedSql, append([]interface{}{hostID}, removedArgs...)...); err != nil {
		return ctxerr.Wrap(ctx, err, "mark users as removed")
	}

	return nil
}

func (ds *Datastore) TotalAndUnseenHostsSince(ctx context.Context, daysCount int) (total int, unseen int, err error) {
	var counts struct {
		Total  int `db:"total"`
		Unseen int `db:"unseen"`
	}
	err = sqlx.GetContext(ctx, ds.reader, &counts,
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

func (ds *Datastore) DeleteHosts(ctx context.Context, ids []uint) error {
	_, err := ds.deleteEntities(ctx, hostsTable, ids)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting hosts")
	}

	query, args, err := sqlx.In(`DELETE FROM host_seen_times WHERE host_id in (?)`, ids)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "building delete host_seen_times query")
	}

	_, err = ds.writer.ExecContext(ctx, query, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting host seen times")
	}

	query, args, err = sqlx.In(`DELETE FROM host_emails WHERE host_id in (?)`, ids)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "building delete host_emails query")
	}

	_, err = ds.writer.ExecContext(ctx, query, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting host emails")
	}

	query, args, err = sqlx.In(`DELETE FROM pack_targets WHERE type=? AND target_id in (?)`, fleet.TargetHost, ids)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "building delete pack_targets query")
	}

	_, err = ds.writer.ExecContext(ctx, query, args...)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "deleting pack_targets for hosts")
	}

	return nil
}

func (ds *Datastore) ListPoliciesForHost(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
	if host.FleetPlatform() == "" {
		// We log to help troubleshooting in case this happens.
		level.Error(ds.logger).Log("err", fmt.Sprintf("host %d with empty platform", host.ID))
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
	if err := sqlx.SelectContext(ctx, ds.reader, &policies, query, host.ID, host.ID, host.FleetPlatform()); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host policies")
	}
	return policies, nil
}

func (ds *Datastore) CleanupExpiredHosts(ctx context.Context) error {
	ac, err := appConfigDB(ctx, ds.reader)
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

	rows, err := ds.writer.QueryContext(
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
		err = ds.DeleteHost(ctx, id)
		if err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return ctxerr.Wrap(ctx, err, "expired hosts, row err")
	}

	_, err = ds.writer.ExecContext(ctx, `DELETE FROM host_seen_times WHERE seen_time < DATE_SUB(NOW(), INTERVAL ? DAY)`, ac.HostExpirySettings.HostExpiryWindow)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting expired host seen times")
	}
	return nil
}

func (ds *Datastore) ListHostDeviceMapping(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
	stmt := `
    SELECT
      id,
      host_id,
      email,
      source
    FROM
      host_emails
    WHERE
      host_id = ?
    ORDER BY
      email, source`

	var mappings []*fleet.HostDeviceMapping
	err := sqlx.SelectContext(ctx, ds.reader, &mappings, stmt, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select host emails by host id")
	}
	return mappings, nil
}

func (ds *Datastore) ReplaceHostDeviceMapping(ctx context.Context, hid uint, mappings []*fleet.HostDeviceMapping) error {
	for _, m := range mappings {
		if hid != m.HostID {
			return ctxerr.Errorf(ctx, "host device mapping are not all for the provided host id %d, found %d", hid, m.HostID)
		}
	}

	// the following SQL statements assume a small number of emails reported
	// per host.
	const (
		selStmt = `
      SELECT
        id,
        email,
        source
      FROM
        host_emails
      WHERE
        host_id = ?`

		delStmt = `
      DELETE FROM
        host_emails
      WHERE
        id IN (?)`

		insStmt = `
      INSERT INTO
        host_emails (host_id, email, source)
      VALUES`

		insPart = ` (?, ?, ?),`
	)

	// index the mappings by email and source, to quickly check which ones
	// need to be deleted and inserted
	toIns := make(map[string]*fleet.HostDeviceMapping)
	for _, m := range mappings {
		toIns[m.Email+"\n"+m.Source] = m
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var prevMappings []*fleet.HostDeviceMapping
		if err := sqlx.SelectContext(ctx, tx, &prevMappings, selStmt, hid); err != nil {
			return ctxerr.Wrap(ctx, err, "select previous host emails")
		}

		var delIDs []uint
		for _, pm := range prevMappings {
			key := pm.Email + "\n" + pm.Source
			if _, ok := toIns[key]; ok {
				// already exists, no need to insert
				delete(toIns, key)
			} else {
				// does not exist anymore, must be deleted
				delIDs = append(delIDs, pm.ID)
			}
		}

		if len(delIDs) > 0 {
			stmt, args, err := sqlx.In(delStmt, delIDs)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "prepare delete statement")
			}
			if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "delete host emails")
			}
		}

		if len(toIns) > 0 {
			var args []interface{}
			for _, m := range toIns {
				args = append(args, hid, m.Email, m.Source)
			}
			stmt := insStmt + strings.TrimSuffix(strings.Repeat(insPart, len(toIns)), ",")
			if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "insert host emails")
			}
		}
		return nil
	})
}

func (ds *Datastore) updateOrInsert(ctx context.Context, updateQuery string, insertQuery string, args ...interface{}) error {
	res, err := ds.writer.ExecContext(ctx, updateQuery, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	if affected == 0 {
		_, err = ds.writer.ExecContext(ctx, insertQuery, args...)
	}
	return ctxerr.Wrap(ctx, err)
}

func (ds *Datastore) SetOrUpdateMunkiVersion(ctx context.Context, hostID uint, version string) error {
	if version == "" {
		// Only update deleted_at if there wasn't any deleted at for this host
		updateQuery := `UPDATE host_munki_info SET deleted_at=NOW() WHERE host_id=? AND deleted_at is NULL`
		_, err := ds.writer.ExecContext(ctx, updateQuery, hostID)
		if err != nil {
			return ctxerr.Wrap(ctx, err)
		}
		return nil
	}
	return ds.updateOrInsert(
		ctx,
		`UPDATE host_munki_info SET version=? WHERE host_id=?`,
		`INSERT INTO host_munki_info(version, host_id) VALUES (?,?)`,
		version, hostID,
	)
}

func (ds *Datastore) SetOrUpdateMDMData(ctx context.Context, hostID uint, enrolled bool, serverURL string, installedFromDep bool) error {
	return ds.updateOrInsert(
		ctx,
		`UPDATE host_mdm SET enrolled=?, server_url=?, installed_from_dep=? WHERE host_id=?`,
		`INSERT INTO host_mdm(enrolled, server_url, installed_from_dep, host_id) VALUES (?, ?, ?, ?)`,
		enrolled, serverURL, installedFromDep, hostID,
	)
}

func (ds *Datastore) GetMunkiVersion(ctx context.Context, hostID uint) (string, error) {
	var version string
	err := sqlx.GetContext(ctx, ds.reader, &version, `SELECT version FROM host_munki_info WHERE deleted_at is NULL AND host_id=?`, hostID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ctxerr.Wrap(ctx, notFound("MunkiInfo").WithID(hostID))
		}
		return "", ctxerr.Wrapf(ctx, err, "getting data from host_munki_info for host_id %d", hostID)
	}

	return version, nil
}

func (ds *Datastore) GetMDM(ctx context.Context, hostID uint) (bool, string, bool, error) {
	dest := struct {
		Enrolled         bool   `db:"enrolled"`
		ServerURL        string `db:"server_url"`
		InstalledFromDep bool   `db:"installed_from_dep"`
	}{}
	err := sqlx.GetContext(ctx, ds.reader, &dest, `SELECT enrolled, server_url, installed_from_dep FROM host_mdm WHERE host_id=?`, hostID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", false, ctxerr.Wrap(ctx, notFound("MDM").WithID(hostID))
		}
		return false, "", false, ctxerr.Wrapf(ctx, err, "getting data from host_mdm for host_id %d", hostID)
	}
	return dest.Enrolled, dest.ServerURL, dest.InstalledFromDep, nil
}

func (ds *Datastore) AggregatedMunkiVersion(ctx context.Context, teamID *uint) ([]fleet.AggregatedMunkiVersion, time.Time, error) {
	id := uint(0)

	if teamID != nil {
		id = *teamID
	}
	var versions []fleet.AggregatedMunkiVersion
	var versionsJson struct {
		JsonValue []byte    `db:"json_value"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	err := sqlx.GetContext(
		ctx, ds.reader, &versionsJson,
		`select json_value, updated_at from aggregated_stats where id=? and type='munki_versions'`,
		id,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// not having stats is not an error
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, ctxerr.Wrap(ctx, err, "selecting munki versions")
	}
	if err := json.Unmarshal(versionsJson.JsonValue, &versions); err != nil {
		return nil, time.Time{}, ctxerr.Wrap(ctx, err, "unmarshaling munki versions")
	}
	return versions, versionsJson.UpdatedAt, nil
}

func (ds *Datastore) AggregatedMDMStatus(ctx context.Context, teamID *uint) (fleet.AggregatedMDMStatus, time.Time, error) {
	id := uint(0)

	if teamID != nil {
		id = *teamID
	}

	var status fleet.AggregatedMDMStatus
	var statusJson struct {
		JsonValue []byte    `db:"json_value"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	err := sqlx.GetContext(
		ctx, ds.reader, &statusJson,
		`select json_value, updated_at from aggregated_stats where id=? and type='mdm_status'`,
		id,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// not having stats is not an error
			return fleet.AggregatedMDMStatus{}, time.Time{}, nil
		}
		return fleet.AggregatedMDMStatus{}, time.Time{}, ctxerr.Wrap(ctx, err, "selecting mdm status")
	}
	if err := json.Unmarshal(statusJson.JsonValue, &status); err != nil {
		return fleet.AggregatedMDMStatus{}, time.Time{}, ctxerr.Wrap(ctx, err, "unmarshaling mdm status")
	}
	return status, statusJson.UpdatedAt, nil
}

func (ds *Datastore) GenerateAggregatedMunkiAndMDM(ctx context.Context) error {
	var ids []uint
	if err := sqlx.SelectContext(ctx, ds.reader, &ids, `SELECT id FROM teams`); err != nil {
		return ctxerr.Wrap(ctx, err, "list teams")
	}

	for _, id := range ids {
		if err := ds.generateAggregatedMunkiVersion(ctx, &id); err != nil {
			return ctxerr.Wrap(ctx, err, "generating aggregated munki version")
		}
		if err := ds.generateAggregatedMDMStatus(ctx, &id); err != nil {
			return ctxerr.Wrap(ctx, err, "generating aggregated mdm status")
		}
	}

	if err := ds.generateAggregatedMunkiVersion(ctx, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "generating aggregated munki version")
	}
	if err := ds.generateAggregatedMDMStatus(ctx, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "generating aggregated mdm status")
	}
	return nil
}

func (ds *Datastore) generateAggregatedMunkiVersion(ctx context.Context, teamID *uint) error {
	id := uint(0)

	var versions []fleet.AggregatedMunkiVersion
	query := `SELECT count(*) as hosts_count, hm.version FROM host_munki_info hm`
	args := []interface{}{}
	if teamID != nil {
		args = append(args, *teamID)
		query += ` JOIN hosts h ON (h.id=hm.host_id) WHERE h.team_id=? AND `
		id = *teamID
	} else {
		query += `  WHERE `
	}
	query += ` hm.deleted_at is NULL GROUP BY hm.version`
	err := sqlx.SelectContext(ctx, ds.reader, &versions, query, args...)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting aggregated data from host_munki")
	}
	versionsJson, err := json.Marshal(versions)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling stats")
	}

	_, err = ds.writer.ExecContext(ctx,
		`INSERT INTO aggregated_stats(id, type, json_value) VALUES(?, ?, ?) ON DUPLICATE KEY UPDATE json_value=VALUES(json_value)`,
		id, "munki_versions", versionsJson,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "inserting stats for munki_versions id %d", id)
	}
	return nil
}

func (ds *Datastore) generateAggregatedMDMStatus(ctx context.Context, teamID *uint) error {
	id := uint(0)

	var status fleet.AggregatedMDMStatus
	query := `SELECT
				COUNT(DISTINCT host_id) as hosts_count,
				COALESCE(SUM(CASE WHEN NOT enrolled THEN 1 ELSE 0 END), 0) as unenrolled_hosts_count,
				COALESCE(SUM(CASE WHEN enrolled AND installed_from_dep THEN 1 ELSE 0 END), 0) as enrolled_automated_hosts_count,
				COALESCE(SUM(CASE WHEN enrolled AND NOT installed_from_dep THEN 1 ELSE 0 END), 0) as enrolled_manual_hosts_count
			 FROM host_mdm hm
       	`
	args := []interface{}{}
	if teamID != nil {
		args = append(args, *teamID)
		query += ` JOIN hosts h ON (h.id=hm.host_id) WHERE h.team_id=?`
		id = *teamID
	}
	err := sqlx.GetContext(ctx, ds.reader, &status, query, args...)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting aggregated data from host_mdm")
	}

	statusJson, err := json.Marshal(status)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling stats")
	}

	_, err = ds.writer.ExecContext(ctx,
		`INSERT INTO aggregated_stats(id, type, json_value) VALUES(?, ?, ?) ON DUPLICATE KEY UPDATE json_value=VALUES(json_value)`,
		id, "mdm_status", statusJson,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "inserting stats for mdm_status id %d", id)
	}
	return nil
}

// HostLite will load the primary data of the host with the given id.
// We define "primary data" as all host information except the
// details (like cpu, memory, gigs_disk_space_available, etc.).
//
// If the host doesn't exist, a NotFoundError is returned.
func (ds *Datastore) HostLite(ctx context.Context, id uint) (*fleet.Host, error) {
	query, args, err := dialect.From(goqu.I("hosts")).Select(
		"id",
		"created_at",
		"updated_at",
		"osquery_host_id",
		"node_key",
		"hostname",
		"uuid",
		"platform",
		"team_id",
		"distributed_interval",
		"logger_tls_period",
		"config_tls_refresh",
		"detail_updated_at",
		"label_updated_at",
		"last_enrolled_at",
		"policy_updated_at",
		"refetch_requested",
	).Where(goqu.I("id").Eq(id)).ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sql build")
	}
	var host fleet.Host
	if err := sqlx.GetContext(ctx, ds.reader, &host, query, args...); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Host").WithID(id))
		}
		return nil, ctxerr.Wrapf(ctx, err, "load host %d", id)
	}
	return &host, nil
}

// UpdateHostOsqueryIntervals updates the osquery intervals of a host.
func (ds *Datastore) UpdateHostOsqueryIntervals(ctx context.Context, id uint, intervals fleet.HostOsqueryIntervals) error {
	sqlStatement := `
		UPDATE hosts SET
			distributed_interval = ?,
			config_tls_refresh = ?,
			logger_tls_period = ?
		WHERE id = ?
	`
	_, err := ds.writer.ExecContext(ctx, sqlStatement,
		intervals.DistributedInterval,
		intervals.ConfigTLSRefresh,
		intervals.LoggerTLSPeriod,
		id,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "update host %d osquery intervals", id)
	}
	return nil
}

// UpdateHostRefetchRequested updates a host's refetch requested field.
func (ds *Datastore) UpdateHostRefetchRequested(ctx context.Context, id uint, value bool) error {
	sqlStatement := `UPDATE hosts SET refetch_requested = ? WHERE id = ?`
	_, err := ds.writer.ExecContext(ctx, sqlStatement, value, id)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "update host %d refetch_requested", id)
	}
	return nil
}

// UpdateHost updates a host.
//
// UpdateHost updates all columns of the `hosts` table.
// It only updates `hosts` table, other additional host information is ignored.
func (ds *Datastore) UpdateHost(ctx context.Context, host *fleet.Host) error {
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
	_, err := ds.writer.ExecContext(ctx, sqlStatement,
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
	return nil
}
