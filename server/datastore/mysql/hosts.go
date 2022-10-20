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
	"unicode/utf8"

	"github.com/cenkalti/backoff/v4"
	"github.com/doug-martin/goqu/v9"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/jmoiron/sqlx"
)

var hostSearchColumns = []string{"hostname", "computer_name", "uuid", "hardware_serial", "primary_ip"}

// NewHost creates a new host on the datastore.
//
// Currently only used for testing.
func (ds *Datastore) NewHost(ctx context.Context, host *fleet.Host) (*fleet.Host, error) {
	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		sqlStatement := `
		INSERT INTO hosts (
			osquery_host_id,
			detail_updated_at,
			label_updated_at,
			policy_updated_at,
			node_key,
			hostname,
			computer_name,
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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		result, err := tx.ExecContext(
			ctx,
			sqlStatement,
			host.OsqueryHostID,
			host.DetailUpdatedAt,
			host.LabelUpdatedAt,
			host.PolicyUpdatedAt,
			host.NodeKey,
			host.Hostname,
			host.ComputerName,
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
			return ctxerr.Wrap(ctx, err, "new host")
		}
		id, _ := result.LastInsertId()
		host.ID = uint(id)

		_, err = ds.writer.ExecContext(ctx,
			`INSERT INTO host_seen_times (host_id, seen_time) VALUES (?,?)`,
			host.ID, host.SeenTime,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new host seen time")
		}
		_, err = ds.writer.ExecContext(ctx,
			`INSERT INTO host_display_names (host_id, display_name) VALUES (?,?)`,
			host.ID, host.DisplayName(),
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "host_display_names")
		}
		return nil
	})
	if err != nil {
		return nil, err
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

func (ds *Datastore) SaveHostPackStats(ctx context.Context, hostID uint, stats []fleet.PackStats) error {
	return saveHostPackStatsDB(ctx, ds.writer, hostID, stats)
}

func saveHostPackStatsDB(ctx context.Context, db sqlx.ExecerContext, hostID uint, stats []fleet.PackStats) error {
	// NOTE: this implementation must be kept in sync with the async/batch version
	// in AsyncBatchSaveHostsScheduledQueryStats (in scheduled_queries.go) - that is,
	// the behaviour per host must be the same.

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

// hostRefs are the tables referenced by hosts.
//
// Defined here for testing purposes.
var hostRefs = []string{
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
	"host_device_auth",
	"host_batteries",
	"host_operating_system",
	"host_munki_issues",
	"host_display_names",
	"windows_updates",
	"host_disks",
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

		for _, table := range hostRefs {
			err := delHostRef(tx, table)
			if err != nil {
				return err
			}
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM pack_targets WHERE type = ? AND target_id = ?`, fleet.TargetHost, hid)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "deleting pack_targets for host %d", hid)
		}

		return nil
	})
}

func (ds *Datastore) Host(ctx context.Context, id uint) (*fleet.Host, error) {
	sqlStatement := `
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
  h.team_id,
  h.policy_updated_at,
  h.public_ip,
  COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
  COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available,
  COALESCE(hst.seen_time, h.created_at) AS seen_time,
  t.name AS team_name,
  (
    SELECT
      additional
    FROM
      host_additional
    WHERE
      host_id = h.id
  ) AS additional,
  coalesce(failing_policies.count, 0) as failing_policies_count,
  coalesce(failing_policies.count, 0) as total_issues_count
FROM
  hosts h
  LEFT JOIN teams t ON (h.team_id = t.id)
  LEFT JOIN host_seen_times hst ON (h.id = hst.host_id)
  LEFT JOIN host_disks hd ON hd.host_id = h.id
  JOIN (
    SELECT
      count(*) as count
    FROM
      policy_membership
    WHERE
      passes = 0
      AND host_id = ?
  ) failing_policies
WHERE
  h.id = ?
LIMIT
  1
`
	args := []interface{}{id, id}

	var host fleet.Host
	err := sqlx.GetContext(ctx, ds.reader, &host, sqlStatement, args...)
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

	return &host, nil
}

func amountEnrolledHostsByOSDB(ctx context.Context, db sqlx.QueryerContext) (byOS map[string][]fleet.HostsCountByOSVersion, totalCount int, err error) {
	var hostsByOS []struct {
		Platform  string `db:"platform"`
		OSVersion string `db:"os_version"`
		NumHosts  int    `db:"num_hosts"`
	}

	const stmt = `
    SELECT platform, os_version, count(*) as num_hosts
    FROM hosts
    GROUP BY platform, os_version
  `
	if err := sqlx.SelectContext(ctx, db, &hostsByOS, stmt); err != nil {
		return nil, 0, err
	}

	byOS = make(map[string][]fleet.HostsCountByOSVersion)
	for _, h := range hostsByOS {
		totalCount += h.NumHosts
		byVersion := byOS[h.Platform]
		byVersion = append(byVersion, fleet.HostsCountByOSVersion{
			Version:     h.OSVersion,
			NumEnrolled: h.NumHosts,
		})
		byOS[h.Platform] = byVersion
	}
	return byOS, totalCount, nil
}

func (ds *Datastore) ListHosts(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
	sql := `SELECT
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
    h.team_id,
    h.policy_updated_at,
    h.public_ip,
	h.orbit_node_key,
    COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
    COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available,
    COALESCE(hst.seen_time, h.created_at) AS seen_time,
    t.name AS team_name
	`

	if opt.DeviceMapping {
		sql += `,
    COALESCE(dm.device_mapping, 'null') as device_mapping
		`
	}

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

	policyMembershipJoin := "JOIN policy_membership pm ON (h.id = pm.host_id)"
	if opt.PolicyIDFilter == nil {
		policyMembershipJoin = ""
	} else if opt.PolicyResponseFilter == nil {
		policyMembershipJoin = "LEFT " + policyMembershipJoin
	}

	softwareFilter := "TRUE"
	if opt.SoftwareIDFilter != nil {
		softwareFilter = "EXISTS (SELECT 1 FROM host_software hs WHERE hs.host_id = h.id AND hs.software_id = ?)"
		params = append(params, opt.SoftwareIDFilter)
	}

	failingPoliciesJoin := `LEFT JOIN (
		    SELECT host_id, count(*) as count FROM policy_membership WHERE passes = 0
		    GROUP BY host_id
		) as failing_policies ON (h.id=failing_policies.host_id)`
	if opt.DisableFailingPolicies {
		failingPoliciesJoin = ""
	}

	mdmJoin := ` JOIN host_mdm hmdm ON h.id = hmdm.host_id `
	if opt.MDMIDFilter == nil && opt.MDMEnrollmentStatusFilter == "" {
		mdmJoin = ""
	}

	operatingSystemJoin := ""
	if opt.OSIDFilter != nil || (opt.OSNameFilter != nil && opt.OSVersionFilter != nil) {
		operatingSystemJoin = `JOIN host_operating_system hos ON h.id = hos.host_id`
	}

	munkiFilter := "TRUE"
	munkiJoin := ""
	if opt.MunkiIssueIDFilter != nil {
		munkiJoin = ` JOIN host_munki_issues hmi ON h.id = hmi.host_id `
		munkiFilter = "hmi.munki_issue_id = ?"
		params = append(params, opt.MunkiIssueIDFilter)
	}

	displayNameJoin := ""
	if opt.ListOptions.OrderKey == "display_name" {
		displayNameJoin = ` JOIN host_display_names hdn ON h.id = hdn.host_id `
	}

	lowDiskSpaceFilter := "TRUE"
	if opt.LowDiskSpaceFilter != nil {
		lowDiskSpaceFilter = `hd.gigs_disk_space_available < ?`
		params = append(params, *opt.LowDiskSpaceFilter)
	}

	sql += fmt.Sprintf(`FROM hosts h
    LEFT JOIN host_seen_times hst ON (h.id = hst.host_id)
    LEFT JOIN teams t ON (h.team_id = t.id)
    LEFT JOIN host_disks hd ON hd.host_id = h.id
    %s
    %s
    %s
    %s
    %s
    %s
    %s
		WHERE TRUE AND %s AND %s AND %s AND %s
    `, deviceMappingJoin, policyMembershipJoin, failingPoliciesJoin, mdmJoin, operatingSystemJoin, munkiJoin, displayNameJoin, ds.whereFilterHostsByTeams(filter, "h"),
		softwareFilter, munkiFilter, lowDiskSpaceFilter,
	)

	now := ds.clock.Now()
	sql, params = filterHostsByStatus(now, sql, opt, params)
	sql, params = filterHostsByTeam(sql, opt, params)
	sql, params = filterHostsByPolicy(sql, opt, params)
	sql, params = filterHostsByMDM(sql, opt, params)
	sql, params = filterHostsByOS(sql, opt, params)
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

func filterHostsByMDM(sql string, opt fleet.HostListOptions, params []interface{}) (string, []interface{}) {
	if opt.MDMIDFilter != nil {
		sql += ` AND hmdm.mdm_id = ?`
		params = append(params, *opt.MDMIDFilter)
	}
	if opt.MDMEnrollmentStatusFilter != "" {
		switch opt.MDMEnrollmentStatusFilter {
		case fleet.MDMEnrollStatusAutomatic:
			sql += ` AND hmdm.enrolled = 1 AND hmdm.installed_from_dep = 1`
		case fleet.MDMEnrollStatusManual:
			sql += ` AND hmdm.enrolled = 1 AND hmdm.installed_from_dep = 0`
		case fleet.MDMEnrollStatusUnenrolled:
			sql += ` AND hmdm.enrolled = 0`
		}
	}
	return sql, params
}

func filterHostsByOS(sql string, opt fleet.HostListOptions, params []interface{}) (string, []interface{}) {
	if opt.OSIDFilter != nil {
		sql += ` AND hos.os_id = ?`
		params = append(params, *opt.OSIDFilter)
	} else if opt.OSNameFilter != nil && opt.OSVersionFilter != nil {
		sql += ` AND hos.os_id IN (SELECT id FROM operating_systems WHERE name = ? AND version = ?)`
		params = append(params, *opt.OSNameFilter, *opt.OSVersionFilter)
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

func filterHostsByStatus(now time.Time, sql string, opt fleet.HostListOptions, params []interface{}) (string, []interface{}) {
	switch opt.StatusFilter {
	case fleet.StatusNew:
		sql += "AND DATE_ADD(h.created_at, INTERVAL 1 DAY) >= ?"
		params = append(params, now)
	case fleet.StatusOnline:
		sql += fmt.Sprintf("AND DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL LEAST(h.distributed_interval, h.config_tls_refresh) + %d SECOND) > ?", fleet.OnlineIntervalBuffer)
		params = append(params, now)
	case fleet.StatusOffline:
		sql += fmt.Sprintf("AND DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL LEAST(h.distributed_interval, h.config_tls_refresh) + %d SECOND) <= ?", fleet.OnlineIntervalBuffer)
		params = append(params, now)
	case fleet.StatusMIA, fleet.StatusMissing:
		sql += "AND DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL 30 DAY) <= ?"
		params = append(params, now)
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

func (ds *Datastore) CleanupIncomingHosts(ctx context.Context, now time.Time) ([]uint, error) {
	var ids []uint
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		selectIDs := `
		SELECT
		  id
		FROM
		  hosts
		WHERE
		  hostname = '' AND
		  osquery_version = '' AND
		  created_at < (? - INTERVAL 5 MINUTE)`
		if err := ds.writer.SelectContext(ctx, &ids, selectIDs, now); err != nil {
			return ctxerr.Wrap(ctx, err, "load incoming hosts to cleanup")
		}

		cleanupHostDisplayName := fmt.Sprintf(
			`DELETE FROM host_display_names WHERE host_id IN (%s)`,
			selectIDs,
		)
		if _, err := ds.writer.ExecContext(ctx, cleanupHostDisplayName, now); err != nil {
			return ctxerr.Wrap(ctx, err, "cleanup host_display_names")
		}

		cleanupHosts := `
		DELETE FROM hosts
		WHERE hostname = '' AND osquery_version = ''
		AND created_at < (? - INTERVAL 5 MINUTE)
		`
		if _, err := ds.writer.ExecContext(ctx, cleanupHosts, now); err != nil {
			return ctxerr.Wrap(ctx, err, "cleanup incoming hosts")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (ds *Datastore) GenerateHostStatusStatistics(ctx context.Context, filter fleet.TeamFilter, now time.Time, platform *string, lowDiskSpace *int) (*fleet.HostSummary, error) {
	// The logic in this function should remain synchronized with
	// host.Status and CountHostsInTargets - that is, the intervals associated
	// with each status must be the same.

	args := []interface{}{now, now, now, now, now}
	hostDisksJoin := ``
	lowDiskSelect := `0 low_disk_space`
	if lowDiskSpace != nil {
		hostDisksJoin = `LEFT JOIN host_disks hd ON (h.id = hd.host_id)`
		lowDiskSelect = `COALESCE(SUM(CASE WHEN hd.gigs_disk_space_available <= ? THEN 1 ELSE 0 END), 0) low_disk_space`
		args = append(args, *lowDiskSpace)
	}

	whereClause := ds.whereFilterHostsByTeams(filter, "h")
	if platform != nil {
		whereClause += " AND h.platform IN (?) "
		args = append(args, fleet.ExpandPlatform(*platform))
	}

	sqlStatement := fmt.Sprintf(`
			SELECT
				COUNT(*) total,
				COALESCE(SUM(CASE WHEN DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL 30 DAY) <= ? THEN 1 ELSE 0 END), 0) mia,
				COALESCE(SUM(CASE WHEN DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL 30 DAY) <= ? THEN 1 ELSE 0 END), 0) missing_30_days_count,
				COALESCE(SUM(CASE WHEN DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL LEAST(distributed_interval, config_tls_refresh) + %d SECOND) <= ? THEN 1 ELSE 0 END), 0) offline,
				COALESCE(SUM(CASE WHEN DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL LEAST(distributed_interval, config_tls_refresh) + %d SECOND) > ? THEN 1 ELSE 0 END), 0) online,
				COALESCE(SUM(CASE WHEN DATE_ADD(h.created_at, INTERVAL 1 DAY) >= ? THEN 1 ELSE 0 END), 0) new,
				%s
			FROM hosts h
			LEFT JOIN host_seen_times hst ON (h.id = hst.host_id)
			%s
			WHERE %s
			LIMIT 1;
		`, fleet.OnlineIntervalBuffer, fleet.OnlineIntervalBuffer, lowDiskSelect, hostDisksJoin, whereClause)

	stmt, args, err := sqlx.In(sqlStatement, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating host statistics statement")
	}
	summary := fleet.HostSummary{TeamID: filter.TeamID}
	err = sqlx.GetContext(ctx, ds.reader, &summary, stmt, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, err, "generating host statistics")
	}
	if lowDiskSpace == nil {
		// don't return the low disk space count if it wasn't requested
		summary.LowDiskSpaceCount = nil
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

func (ds *Datastore) EnrollOrbit(ctx context.Context, hardwareUUID string, orbitNodeKey string, teamID *uint) (*fleet.Host, error) {
	if orbitNodeKey == "" {
		return nil, ctxerr.New(ctx, "orbit node key is empty")
	}

	if hardwareUUID == "" {
		return nil, ctxerr.New(ctx, "hardware uuid is empty")
	}

	var host fleet.Host
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		err := sqlx.GetContext(ctx, tx, &host, `SELECT id FROM hosts WHERE osquery_host_id = ?`, hardwareUUID)
		switch {
		case err == nil:
			sqlUpdate := `UPDATE hosts SET orbit_node_key = ? WHERE osquery_host_id = ? `
			_, err := tx.ExecContext(ctx, sqlUpdate, orbitNodeKey, hardwareUUID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "orbit enroll error updating host details")
			}
		case errors.Is(err, sql.ErrNoRows):
			zeroTime := time.Unix(0, 0).Add(24 * time.Hour)
			// Create new host record. We always create newly enrolled hosts with refetch_requested = true
			// so that the frontend automatically starts background checks to update the page whenever
			// the refetch is completed.
			// We are also initially setting node_key to be the same as orbit_node_key because node_key has a unique
			// constraint
			sqlInsert := `
				INSERT INTO hosts (
					last_enrolled_at,               
					detail_updated_at,
					label_updated_at,
					policy_updated_at,
					osquery_host_id,
					node_key,
					team_id,
					refetch_requested,
					orbit_node_key
				) VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?)
			`
			result, err := tx.ExecContext(ctx, sqlInsert, zeroTime, zeroTime, zeroTime, zeroTime, hardwareUUID, orbitNodeKey, teamID, orbitNodeKey)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "orbit enroll error inserting host details")
			}
			hostID, _ := result.LastInsertId()
			level.Info(ds.logger).Log("hostID", hostID)
			const sqlHostDisplayName = `
				INSERT INTO host_display_names (host_id, display_name) VALUES (?, '')
			`
			_, err = tx.ExecContext(ctx, sqlHostDisplayName, hostID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "insert host_display_names")
			}
		default:
			return ctxerr.Wrap(ctx, err, "orbit enroll error selecting host details")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &host, nil
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
		err := sqlx.GetContext(ctx, tx, &host, `SELECT id, last_enrolled_at, team_id FROM hosts WHERE osquery_host_id = ?`, osqueryHostID)
		switch {
		case err != nil && !errors.Is(err, sql.ErrNoRows):
			return ctxerr.Wrap(ctx, err, "check existing")
		case errors.Is(err, sql.ErrNoRows):
			// Create new host record. We always create newly enrolled hosts with refetch_requested = true
			// so that the frontend automatically starts background checks to update the page whenever
			// the refetch is completed.
			const sqlInsert = `
				INSERT INTO hosts (
					detail_updated_at,
					label_updated_at,
					policy_updated_at,
					osquery_host_id,
					node_key,
					team_id,
					refetch_requested
				) VALUES (?, ?, ?, ?, ?, ?, 1)
			`
			result, err := tx.ExecContext(ctx, sqlInsert, zeroTime, zeroTime, zeroTime, osqueryHostID, nodeKey, teamID)
			if err != nil {
				level.Info(ds.logger).Log("hostIDError", err.Error())
				return ctxerr.Wrap(ctx, err, "insert host")
			}
			hostID, _ = result.LastInsertId()
			level.Info(ds.logger).Log("hostID", hostID)
			const sqlHostDisplayName = `
				INSERT INTO host_display_names (host_id, display_name) VALUES (?, '')
			`
			_, err = tx.ExecContext(ctx, sqlHostDisplayName, hostID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "insert host_display_names")
			}
		default:
			// Prevent hosts from enrolling too often with the same identifier.
			// Prior to adding this we saw many hosts (probably VMs) with the
			// same identifier competing for enrollment and causing perf issues.
			if cooldown > 0 && time.Since(host.LastEnrolledAt) < cooldown {
				return backoff.Permanent(ctxerr.Errorf(ctx, "host identified by %s enrolling too often", osqueryHostID))
			}
			hostID = int64(host.ID)

			if err := deleteAllPolicyMemberships(ctx, tx, []uint{host.ID}); err != nil {
				return ctxerr.Wrap(ctx, err, "cleanup policy membership on re-enroll")
			}

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
			INSERT INTO host_seen_times (host_id, seen_time) VALUES (?, ?)
			ON DUPLICATE KEY UPDATE seen_time = VALUES(seen_time)`,
			hostID, time.Now().UTC())
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new host seen time")
		}
		sqlSelect := `
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
        h.team_id,
        h.policy_updated_at,
        h.public_ip,
		h.orbit_node_key,
        COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
        COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available
      FROM
        hosts h
      LEFT OUTER JOIN
        host_disks hd ON hd.host_id = h.id
      WHERE h.id = ?
      LIMIT 1
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

// getContextTryStmt will attempt to run sqlx.GetContext on a cached statement if available, resorting to ds.reader.
func (ds *Datastore) getContextTryStmt(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
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
	query := `
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
      h.team_id,
      h.policy_updated_at,
      h.public_ip,
      h.orbit_node_key,
      COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
      COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available
    FROM
      hosts h
    LEFT OUTER JOIN
      host_disks hd ON hd.host_id = h.id
    WHERE node_key = ?`

	var host fleet.Host
	switch err := ds.getContextTryStmt(ctx, &host, query, nodeKey); {
	case err == nil:
		return &host, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, ctxerr.Wrap(ctx, notFound("Host"))
	default:
		return nil, ctxerr.Wrap(ctx, err, "find host")
	}
}

// LoadHostByOrbitNodeKey loads the whole host identified by the node key.
// If the node key is invalid it returns a NotFoundError.
func (ds *Datastore) LoadHostByOrbitNodeKey(ctx context.Context, nodeKey string) (*fleet.Host, error) {
	query := `SELECT * FROM hosts WHERE orbit_node_key = ?`

	var host fleet.Host
	switch err := ds.getContextTryStmt(ctx, &host, query, nodeKey); {
	case err == nil:
		return &host, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, ctxerr.Wrap(ctx, notFound("Host"))
	default:
		return nil, ctxerr.Wrap(ctx, err, "find host")
	}
}

// LoadHostByDeviceAuthToken loads the whole host identified by the device auth token.
// If the token is invalid or expired it returns a NotFoundError.
func (ds *Datastore) LoadHostByDeviceAuthToken(ctx context.Context, authToken string, tokenTTL time.Duration) (*fleet.Host, error) {
	const query = `
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
      h.team_id,
      h.policy_updated_at,
      h.public_ip,
      COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
      COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available
    FROM
      host_device_auth hda
    INNER JOIN
      hosts h
    ON
      hda.host_id = h.id
    LEFT OUTER JOIN
      host_disks hd ON hd.host_id = hda.host_id
    WHERE hda.token = ? AND hda.updated_at >= DATE_SUB(NOW(), INTERVAL ? SECOND)`

	var host fleet.Host
	switch err := sqlx.GetContext(ctx, ds.reader, &host, query, authToken, tokenTTL.Seconds()); {
	case err == nil:
		return &host, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, ctxerr.Wrap(ctx, notFound("Host"))
	default:
		return nil, ctxerr.Wrap(ctx, err, "find host")
	}
}

// SetOrUpdateDeviceAuthToken inserts or updates the auth token for a host.
func (ds *Datastore) SetOrUpdateDeviceAuthToken(ctx context.Context, hostID uint, authToken string) error {
	// Note that by not specifying "updated_at = VALUES(updated_at)" in the UPDATE part
	// of the statement, it inherits the default behaviour which is that the updated_at
	// timestamp will NOT be changed if the new token is the same as the old token
	// (which is exactly what we want). The updated_at timestamp WILL be updated if the
	// new token is different.
	const stmt = `
		INSERT INTO
			host_device_auth ( host_id, token )
		VALUES
			(?, ?)
		ON DUPLICATE KEY UPDATE
			token = VALUES(token)
`
	_, err := ds.writer.ExecContext(ctx, stmt, hostID, authToken)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert host's device auth token")
	}
	return nil
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
//   - Use the provided team filter.
//   - Search hostname, uuid, hardware_serial, and primary_ip using LIKE (mimics ListHosts behavior)
//   - An optional list of IDs to omit from the search.
func (ds *Datastore) SearchHosts(ctx context.Context, filter fleet.TeamFilter, matchQuery string, omit ...uint) ([]*fleet.Host, error) {
	query := `SELECT
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
    h.team_id,
    h.policy_updated_at,
    h.public_ip,
	h.orbit_node_key,
    COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
    COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available,
    COALESCE(hst.seen_time, h.created_at) AS seen_time
  FROM hosts h
  LEFT JOIN host_seen_times hst ON (h.id = hst.host_id)
  LEFT JOIN host_disks hd ON hd.host_id = h.id
  WHERE TRUE`

	var args []interface{}
	if len(matchQuery) > 0 {
		query, args = hostSearchLike(query, args, matchQuery, hostSearchColumns...)
	}
	var in interface{}
	// use -1 if there are no values to omit.
	// Avoids empty args error for `sqlx.In`
	in = omit
	if len(omit) == 0 {
		in = -1
	}
	args = append(args, in)
	query += " AND id NOT IN (?) AND "
	query += ds.whereFilterHostsByTeams(filter, "h")
	query += ` ORDER BY h.id DESC LIMIT 10`

	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "searching default hosts")
	}
	query = ds.reader.Rebind(query)
	hosts := []*fleet.Host{}
	if err := sqlx.SelectContext(ctx, ds.reader, &hosts, query, args...); err != nil {
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
      h.team_id,
      h.policy_updated_at,
      h.public_ip,
	  h.orbit_node_key,
      COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
      COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available,
      COALESCE(hst.seen_time, h.created_at) AS seen_time
    FROM hosts h
    LEFT JOIN host_seen_times hst ON (h.id = hst.host_id)
    LEFT JOIN host_disks hd ON hd.host_id = h.id
    WHERE ? IN (h.hostname, h.osquery_host_id, h.node_key, h.uuid)
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
		if err := cleanupPolicyMembershipOnTeamChange(ctx, tx, hostIDs); err != nil {
			return ctxerr.Wrap(ctx, err, "AddHostsToTeam delete policy membership")
		}

		query, args, err := sqlx.In(`UPDATE hosts SET team_id = ? WHERE id IN (?)`, teamID, hostIDs)
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
				removed_at = NULL`,
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

	// convert daysCount to integer number of seconds for more precision in sql query
	unseenSeconds := daysCount * 24 * 60 * 60

	err = sqlx.GetContext(ctx, ds.reader, &counts,
		`SELECT
			COUNT(*) as total,
			SUM(IF(TIMESTAMPDIFF(SECOND, COALESCE(hst.seen_time, h.created_at), CURRENT_TIMESTAMP) >= ?, 1, 0)) as unseen
		FROM hosts h
		LEFT JOIN host_seen_times hst
		ON h.id = hst.host_id`,
		unseenSeconds,
	)

	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "getting total and unseen host counts")
	}

	return counts.Total, counts.Unseen, nil
}

func (ds *Datastore) DeleteHosts(ctx context.Context, ids []uint) error {
	for _, id := range ids {
		if err := ds.DeleteHost(ctx, id); err != nil {
			return ctxerr.Wrapf(ctx, err, "delete host %d", id)
		}
	}
	return nil
}

func (ds *Datastore) FailingPoliciesCount(ctx context.Context, host *fleet.Host) (uint, error) {
	if host.FleetPlatform() == "" {
		// We log to help troubleshooting in case this happens.
		level.Error(ds.logger).Log("err", fmt.Sprintf("host %d with empty platform", host.ID))
	}

	query := `
		SELECT SUM(1 - pm.passes) AS n_failed
		FROM policy_membership pm
		WHERE pm.host_id = ?
		GROUP BY host_id
	`

	var r uint
	if err := sqlx.GetContext(ctx, ds.reader, &r, query, host.ID); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, ctxerr.Wrap(ctx, err, "get failing policies count")
	}
	return r, nil
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
	AND (p.platforms IS NULL OR p.platforms = '' OR FIND_IN_SET(?, p.platforms) != 0)`

	var policies []*fleet.HostPolicy
	if err := sqlx.SelectContext(ctx, ds.reader, &policies, query, host.ID, host.ID, host.FleetPlatform()); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host policies")
	}
	return policies, nil
}

func (ds *Datastore) CleanupExpiredHosts(ctx context.Context) ([]uint, error) {
	ac, err := appConfigDB(ctx, ds.reader)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}
	if !ac.HostExpirySettings.HostExpiryEnabled {
		return nil, nil
	}

	// Usual clean up queries used to be like this:
	// DELETE FROM hosts WHERE id in (SELECT host_id FROM host_seen_times WHERE seen_time < DATE_SUB(NOW(), INTERVAL ? DAY))
	// This means a full table scan for hosts, and for big deployments, that's not ideal
	// so instead, we get the ids one by one and delete things one by one
	// it might take longer, but it should lock only the row we need

	var ids []uint
	err = ds.writer.SelectContext(
		ctx,
		&ids,
		`SELECT h.id FROM hosts h
		LEFT JOIN host_seen_times hst
		ON h.id = hst.host_id
		WHERE COALESCE(hst.seen_time, h.created_at) < DATE_SUB(NOW(), INTERVAL ? DAY)`,
		ac.HostExpirySettings.HostExpiryWindow,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting expired host ids")
	}

	for _, id := range ids {
		err = ds.DeleteHost(ctx, id)
		if err != nil {
			return nil, err
		}
	}

	_, err = ds.writer.ExecContext(ctx, `DELETE FROM host_seen_times WHERE seen_time < DATE_SUB(NOW(), INTERVAL ? DAY)`, ac.HostExpirySettings.HostExpiryWindow)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "deleting expired host seen times")
	}
	return ids, nil
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

func (ds *Datastore) ReplaceHostBatteries(ctx context.Context, hid uint, mappings []*fleet.HostBattery) error {
	const (
		replaceStmt = `
    INSERT INTO
      host_batteries (
        host_id,
        serial_number,
        cycle_count,
        health
      )
    VALUES
      %s
    ON DUPLICATE KEY UPDATE
      cycle_count = VALUES(cycle_count),
      health = VALUES(health),
      updated_at = CURRENT_TIMESTAMP
`
		valuesPart = `(?, ?, ?, ?),`

		deleteExceptStmt = `
    DELETE FROM
      host_batteries
    WHERE
      host_id = ? AND
      serial_number NOT IN (?)
`
		deleteAllStmt = `
    DELETE FROM
      host_batteries
    WHERE
      host_id = ?
`
	)

	replaceArgs := make([]interface{}, 0, len(mappings)*4)
	deleteNotIn := make([]string, 0, len(mappings))
	for _, hb := range mappings {
		deleteNotIn = append(deleteNotIn, hb.SerialNumber)
		replaceArgs = append(replaceArgs, hid, hb.SerialNumber, hb.CycleCount, hb.Health)
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// first, insert the new batteries or update the existing ones
		if len(replaceArgs) > 0 {
			if _, err := tx.ExecContext(ctx, fmt.Sprintf(replaceStmt, strings.TrimSuffix(strings.Repeat(valuesPart, len(mappings)), ",")), replaceArgs...); err != nil {
				return ctxerr.Wrap(ctx, err, "upsert host batteries")
			}
		}

		// then, delete the old ones
		if len(deleteNotIn) > 0 {
			delStmt, args, err := sqlx.In(deleteExceptStmt, hid, deleteNotIn)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "generating host batteries delete NOT IN statement")
			}
			if _, err := tx.ExecContext(ctx, delStmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "delete host batteries")
			}
		} else if _, err := tx.ExecContext(ctx, deleteAllStmt, hid); err != nil {
			return ctxerr.Wrap(ctx, err, "delete all host batteries")
		}
		return nil
	})
}

func (ds *Datastore) updateOrInsert(ctx context.Context, updateQuery string, insertQuery string, args ...interface{}) error {
	res, err := ds.writer.ExecContext(ctx, updateQuery, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update")
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "rows affected by update")
	}
	if affected == 0 {
		_, err = ds.writer.ExecContext(ctx, insertQuery, args...)
	}
	return ctxerr.Wrap(ctx, err, "insert")
}

func (ds *Datastore) SetOrUpdateMunkiInfo(ctx context.Context, hostID uint, version string, errors, warnings []string) error {
	// NOTE(mna): if we lose too many saves due to some errors, we could continue
	// processing even when part of the ingestion fails (e.g. process version if
	// issues fail or vice-versa, ignore issues that failed to be created, etc.).
	// Currently, taking a strict approach as a point that was mentioned in the
	// ticket was to care about data accuracy, so instead of allowing to save
	// only a subset of issues, we fail if we can't save the complete set.

	msgToID, err := ds.getOrInsertMunkiIssues(ctx, errors, warnings, fleet.DefaultMunkiIssuesBatchSize)
	if err != nil {
		return err
	}

	if err := ds.replaceHostMunkiIssues(ctx, hostID, msgToID); err != nil {
		return err
	}

	if version == "" {
		// Only update deleted_at if there wasn't any deleted at for this host
		updateQuery := `UPDATE host_munki_info SET deleted_at = NOW() WHERE host_id = ? AND deleted_at is NULL`
		_, err := ds.writer.ExecContext(ctx, updateQuery, hostID)
		if err != nil {
			return ctxerr.Wrap(ctx, err)
		}
		return nil
	}
	return ds.updateOrInsert(
		ctx,
		`UPDATE host_munki_info SET version = ? WHERE host_id = ?`,
		`INSERT INTO host_munki_info (version, host_id) VALUES (?, ?)`,
		version, hostID,
	)
}

func (ds *Datastore) replaceHostMunkiIssues(ctx context.Context, hostID uint, msgToID map[[2]string]uint) error {
	// This needs an efficient way to check if the batch of messages are the same
	// as existing ones, as this should be a common case (i.e. Munki does not run
	// *that* often, so it is likely that the host reports the same messages for
	// some time, or none at all). The approach is as follows:
	//
	// * Read a COUNT of new IDs (those to be saved) and a COUNT of old IDs
	//   (those to be removed) from the read replica.
	// * If COUNT(new) == len(newIDs) and COUNT(old) == 0, no write is required.
	// * If COUNT(old) > 0, delete those obsolete ids.
	// * If COUNT(new) < len(newIDs), insert those missing ids.
	//
	// In the best scenario, a single statement is executed on the read replica,
	// and in the worst case, 3 statements are executed, 2 on the primary.
	//
	// Of course this is racy, as the check is done in the replica and the write
	// is not transactional, but this is not an issue here as host-reported data
	// is eventually consistent in nature and that data is reported at regular
	// intervals.

	newIDs := make([]uint, 0, len(msgToID))
	for _, id := range msgToID {
		newIDs = append(newIDs, id)
	}

	const countStmt = `SELECT
    (SELECT COUNT(*) FROM host_munki_issues WHERE host_id = ? AND munki_issue_id IN (?)) as count_new,
    (SELECT COUNT(*) FROM host_munki_issues WHERE host_id = ? AND munki_issue_id NOT IN (?)) as count_old`

	var counts struct {
		CountNew int `db:"count_new"`
		CountOld int `db:"count_old"`
	}

	// required to get the old count if the new is empty
	idsForIN := newIDs
	if len(idsForIN) == 0 {
		// must have at least one for the IN/NOT IN to work, add an impossible one
		idsForIN = []uint{0}
	}
	stmt, args, err := sqlx.In(countStmt, hostID, idsForIN, hostID, idsForIN)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare read host_munki_issues counts arguments")
	}
	if err := sqlx.GetContext(ctx, ds.reader, &counts, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "read host_munki_issues counts")
	}
	if counts.CountNew == len(newIDs) && counts.CountOld == 0 {
		return nil
	}

	if counts.CountOld > 0 {
		// must delete those IDs
		const delStmt = `DELETE FROM host_munki_issues WHERE host_id = ? AND munki_issue_id NOT IN (?)`
		stmt, args, err := sqlx.In(delStmt, hostID, idsForIN)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "prepare delete host_munki_issues arguments")
		}
		if _, err := ds.writer.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete host_munki_issues")
		}
	}

	if counts.CountNew < len(newIDs) {
		// must insert missing IDs
		const (
			insStmt  = `INSERT INTO host_munki_issues (host_id, munki_issue_id) VALUES %s ON DUPLICATE KEY UPDATE host_id = host_id`
			stmtPart = `(?, ?),`
		)

		stmt := fmt.Sprintf(insStmt, strings.TrimSuffix(strings.Repeat(stmtPart, len(newIDs)), ","))
		args := make([]interface{}, 0, 2*len(newIDs))
		for _, id := range newIDs {
			args = append(args, hostID, id)
		}
		if _, err := ds.writer.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert host_munki_issues")
		}
	}

	return nil
}

const maxMunkiIssueNameLen = 255

func (ds *Datastore) getOrInsertMunkiIssues(ctx context.Context, errors, warnings []string, batchSize int) (msgToID map[[2]string]uint, err error) {
	for i, e := range errors {
		if n := utf8.RuneCountInString(e); n > maxMunkiIssueNameLen {
			runes := []rune(e)
			errors[i] = string(runes[:maxMunkiIssueNameLen])
		}
	}
	for i, w := range warnings {
		if n := utf8.RuneCountInString(w); n > maxMunkiIssueNameLen {
			runes := []rune(w)
			warnings[i] = string(runes[:maxMunkiIssueNameLen])
		}
	}

	// get list of unique messages+type to load ids and create if necessary
	msgToID = make(map[[2]string]uint, len(errors)+len(warnings))
	for _, e := range errors {
		msgToID[[2]string{e, "error"}] = 0
	}
	for _, w := range warnings {
		msgToID[[2]string{w, "warning"}] = 0
	}

	type munkiIssue struct {
		ID   uint   `db:"id"`
		Name string `db:"name"`
	}

	readIDs := func(q sqlx.QueryerContext, msgs [][2]string, typ string) error {
		const readStmt = `SELECT id, name FROM munki_issues WHERE issue_type = ? AND name IN (?)`

		names := make([]string, 0, len(msgs))
		for _, msg := range msgs {
			if msg[1] == typ {
				names = append(names, msg[0])
			}
		}

		for len(names) > 0 {
			batch := names
			if len(batch) > batchSize {
				batch = names[:batchSize]
			}
			names = names[len(batch):]

			var issues []*munkiIssue
			stmt, args, err := sqlx.In(readStmt, typ, batch)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "generate munki issues read batch statement")
			}
			if err := sqlx.SelectContext(ctx, q, &issues, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "read munki issues batch")
			}

			for _, issue := range issues {
				msgToID[[2]string{issue.Name, typ}] = issue.ID
			}
		}
		return nil
	}

	missingIDs := func() [][2]string {
		var missing [][2]string
		for msg, id := range msgToID {
			if id == 0 {
				missing = append(missing, msg)
			}
		}
		return missing
	}

	allMsgs := make([][2]string, 0, len(msgToID))
	for k := range msgToID {
		allMsgs = append(allMsgs, k)
	}

	// get the IDs for existing munki issues (from the read replica)
	if err := readIDs(ds.reader, allMsgs, "error"); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load error message IDs from reader")
	}
	if err := readIDs(ds.reader, allMsgs, "warning"); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load warning message IDs from reader")
	}

	// create any missing munki issues (using the primary)
	if missing := missingIDs(); len(missing) > 0 {
		const (
			// UPDATE issue_type = issue_type results in a no-op in mysql (https://stackoverflow.com/a/4596409/1094941)
			insStmt   = `INSERT INTO munki_issues (name, issue_type) VALUES %s ON DUPLICATE KEY UPDATE issue_type = issue_type`
			stmtParts = `(?, ?),`
		)

		msgsToReload := missing

		args := make([]interface{}, 0, batchSize*2)
		for len(missing) > 0 {
			batch := missing
			if len(batch) > batchSize {
				batch = missing[:batchSize]
			}
			missing = missing[len(batch):]

			args = args[:0]
			for _, msg := range batch {
				args = append(args, msg[0], msg[1])
			}
			stmt := fmt.Sprintf(insStmt, strings.TrimSuffix(strings.Repeat(stmtParts, len(batch)), ","))
			if _, err := ds.writer.ExecContext(ctx, stmt, args...); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "batch-insert munki issues")
			}
		}

		// load the IDs for the missing munki issues, from the primary as we just
		// inserted them
		if err := readIDs(ds.writer, msgsToReload, "error"); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "load error message IDs from writer")
		}
		if err := readIDs(ds.writer, msgsToReload, "warning"); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "load warning message IDs from writer")
		}
		if missing := missingIDs(); len(missing) > 0 {
			// some messages still have no IDs
			return nil, ctxerr.New(ctx, "found munki issues without id after batch-insert")
		}
	}

	return msgToID, nil
}

func (ds *Datastore) SetOrUpdateMDMData(ctx context.Context, hostID uint, enrolled bool, serverURL string, installedFromDep bool) error {
	mdmID, err := ds.getOrInsertMDMSolution(ctx, serverURL)
	if err != nil {
		return err
	}

	return ds.updateOrInsert(
		ctx,
		`UPDATE host_mdm SET enrolled = ?, server_url = ?, installed_from_dep = ?, mdm_id = ? WHERE host_id = ?`,
		`INSERT INTO host_mdm (enrolled, server_url, installed_from_dep, mdm_id, host_id) VALUES (?, ?, ?, ?, ?)`,
		enrolled, serverURL, installedFromDep, mdmID, hostID,
	)
}

// SetOrUpdateHostDisksSpace sets the available gigs and percentage of the
// disks for the specified host.
func (ds *Datastore) SetOrUpdateHostDisksSpace(ctx context.Context, hostID uint, gigsAvailable, percentAvailable float64) error {
	return ds.updateOrInsert(
		ctx,
		`UPDATE host_disks SET gigs_disk_space_available = ?, percent_disk_space_available = ? WHERE host_id = ?`,
		`INSERT INTO host_disks (gigs_disk_space_available, percent_disk_space_available, host_id) VALUES (?, ?, ?)`,
		gigsAvailable, percentAvailable, hostID,
	)
}

func (ds *Datastore) getOrInsertMDMSolution(ctx context.Context, serverURL string) (mdmID uint, err error) {
	mdmName := fleet.MDMNameFromServerURL(serverURL)

	readStmt := &parameterizedStmt{
		Statement: `SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`,
		Args:      []interface{}{mdmName, serverURL},
	}
	insStmt := &parameterizedStmt{
		Statement: `INSERT INTO mobile_device_management_solutions (name, server_url) VALUES (?, ?)`,
		Args:      []interface{}{mdmName, serverURL},
	}
	return ds.optimisticGetOrInsert(ctx, readStmt, insStmt)
}

func (ds *Datastore) GetHostMunkiVersion(ctx context.Context, hostID uint) (string, error) {
	var version string
	err := sqlx.GetContext(ctx, ds.reader, &version, `SELECT version FROM host_munki_info WHERE deleted_at is NULL AND host_id = ?`, hostID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ctxerr.Wrap(ctx, notFound("MunkiInfo").WithID(hostID))
		}
		return "", ctxerr.Wrapf(ctx, err, "getting data from host_munki_info for host_id %d", hostID)
	}

	return version, nil
}

func (ds *Datastore) GetHostMDM(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
	var hmdm fleet.HostMDM
	err := sqlx.GetContext(ctx, ds.reader, &hmdm, `
		SELECT
			hm.host_id, hm.enrolled, hm.server_url, hm.installed_from_dep, hm.mdm_id, COALESCE(mdms.name, ?) AS name
		FROM
			host_mdm hm
		LEFT OUTER JOIN
			mobile_device_management_solutions mdms
		ON hm.mdm_id = mdms.id
		WHERE hm.host_id = ?`, fleet.UnknownMDMName, hostID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDM").WithID(hostID))
		}
		return nil, ctxerr.Wrapf(ctx, err, "getting data from host_mdm for host_id %d", hostID)
	}
	return &hmdm, nil
}

func (ds *Datastore) GetMDMSolution(ctx context.Context, mdmID uint) (*fleet.MDMSolution, error) {
	var solution fleet.MDMSolution
	err := sqlx.GetContext(ctx, ds.reader, &solution, `
    SELECT
      id,
      name,
      server_url
    FROM
      mobile_device_management_solutions
    WHERE id = ?`, mdmID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MDMSolution").WithID(mdmID))
		}
		return nil, ctxerr.Wrapf(ctx, err, "select mobile_device_management_solutions for id %d", mdmID)
	}
	return &solution, nil
}

func (ds *Datastore) GetHostMunkiIssues(ctx context.Context, hostID uint) ([]*fleet.HostMunkiIssue, error) {
	var issues []*fleet.HostMunkiIssue
	err := sqlx.SelectContext(ctx, ds.reader, &issues, `
    SELECT
      hmi.munki_issue_id,
      mi.name,
      mi.issue_type,
      hmi.created_at
	  FROM
      host_munki_issues hmi
    INNER JOIN
      munki_issues mi
    ON
      hmi.munki_issue_id = mi.id
    WHERE host_id = ?
`, hostID)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "select host_munki_issues for host_id %d", hostID)
	}
	return issues, nil
}

func (ds *Datastore) GetMunkiIssue(ctx context.Context, munkiIssueID uint) (*fleet.MunkiIssue, error) {
	var issue fleet.MunkiIssue
	err := sqlx.GetContext(ctx, ds.reader, &issue, `
    SELECT
      id,
      name,
      issue_type
    FROM
      munki_issues
    WHERE id = ?`, munkiIssueID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("MunkiIssue").WithID(munkiIssueID))
		}
		return nil, ctxerr.Wrapf(ctx, err, "select munki_issues for id %d", munkiIssueID)
	}
	return &issue, nil
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
		`SELECT json_value, updated_at FROM aggregated_stats WHERE id = ? AND type = 'munki_versions'`,
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

func (ds *Datastore) AggregatedMunkiIssues(ctx context.Context, teamID *uint) ([]fleet.AggregatedMunkiIssue, time.Time, error) {
	id := uint(0)

	if teamID != nil {
		id = *teamID
	}

	var result []fleet.AggregatedMunkiIssue
	var resultJSON struct {
		JsonValue []byte    `db:"json_value"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	err := sqlx.GetContext(
		ctx, ds.reader, &resultJSON,
		`SELECT json_value, updated_at FROM aggregated_stats WHERE id = ? AND type = 'munki_issues'`,
		id,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// not having stats is not an error
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, ctxerr.Wrap(ctx, err, "selecting munki issues")
	}
	if err := json.Unmarshal(resultJSON.JsonValue, &result); err != nil {
		return nil, time.Time{}, ctxerr.Wrap(ctx, err, "unmarshaling munki issues")
	}
	return result, resultJSON.UpdatedAt, nil
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
		`select json_value, updated_at from aggregated_stats where id = ? and type = 'mdm_status'`,
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

func (ds *Datastore) AggregatedMDMSolutions(ctx context.Context, teamID *uint) ([]fleet.AggregatedMDMSolutions, time.Time, error) {
	id := uint(0)

	if teamID != nil {
		id = *teamID
	}

	var result []fleet.AggregatedMDMSolutions
	var resultJSON struct {
		JsonValue []byte    `db:"json_value"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	err := sqlx.GetContext(
		ctx, ds.reader, &resultJSON,
		`SELECT json_value, updated_at FROM aggregated_stats WHERE id = ? AND type = 'mdm_solutions'`,
		id,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// not having stats is not an error
			return nil, time.Time{}, nil
		}
		return nil, time.Time{}, ctxerr.Wrap(ctx, err, "selecting mdm solutions")
	}
	if err := json.Unmarshal(resultJSON.JsonValue, &result); err != nil {
		return nil, time.Time{}, ctxerr.Wrap(ctx, err, "unmarshaling mdm solutions")
	}
	return result, resultJSON.UpdatedAt, nil
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
		if err := ds.generateAggregatedMunkiIssues(ctx, &id); err != nil {
			return ctxerr.Wrap(ctx, err, "generating aggregated munki issues")
		}
		if err := ds.generateAggregatedMDMStatus(ctx, &id); err != nil {
			return ctxerr.Wrap(ctx, err, "generating aggregated mdm status")
		}
		if err := ds.generateAggregatedMDMSolutions(ctx, &id); err != nil {
			return ctxerr.Wrap(ctx, err, "generating aggregated mdm solutions")
		}
	}

	if err := ds.generateAggregatedMunkiVersion(ctx, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "generating aggregated munki version")
	}
	if err := ds.generateAggregatedMunkiIssues(ctx, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "generating aggregated munki issues")
	}
	if err := ds.generateAggregatedMDMStatus(ctx, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "generating aggregated mdm status")
	}
	if err := ds.generateAggregatedMDMSolutions(ctx, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "generating aggregated mdm solutions")
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
		query += ` JOIN hosts h ON (h.id = hm.host_id) WHERE h.team_id = ? AND `
		id = *teamID
	} else {
		query += `  WHERE `
	}
	query += ` hm.deleted_at IS NULL GROUP BY hm.version`
	err := sqlx.SelectContext(ctx, ds.reader, &versions, query, args...)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting aggregated data from host_munki")
	}
	versionsJson, err := json.Marshal(versions)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling stats")
	}

	_, err = ds.writer.ExecContext(ctx,
		`
INSERT INTO aggregated_stats (id, type, json_value)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
    json_value = VALUES(json_value),
    updated_at = CURRENT_TIMESTAMP
`,
		id, "munki_versions", versionsJson,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "inserting stats for munki_versions id %d", id)
	}
	return nil
}

func (ds *Datastore) generateAggregatedMunkiIssues(ctx context.Context, teamID *uint) error {
	id := uint(0)

	var issues []fleet.AggregatedMunkiIssue
	query := `
  SELECT
    COUNT(*) as hosts_count,
    hmi.munki_issue_id as id,
		mi.name,
		mi.issue_type
  FROM
    host_munki_issues hmi
  INNER JOIN
    munki_issues mi
  ON
    hmi.munki_issue_id = mi.id
`
	args := []interface{}{}
	if teamID != nil {
		args = append(args, *teamID)
		query += ` JOIN hosts h ON (h.id = hmi.host_id) WHERE h.team_id = ? `
		id = *teamID
	}
	query += `GROUP BY hmi.munki_issue_id, mi.name, mi.issue_type`

	err := sqlx.SelectContext(ctx, ds.reader, &issues, query, args...)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting aggregated data from host_munki_issues")
	}

	issuesJSON, err := json.Marshal(issues)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling stats")
	}

	_, err = ds.writer.ExecContext(ctx, `
INSERT INTO aggregated_stats (id, type, json_value)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
    json_value = VALUES(json_value),
    updated_at = CURRENT_TIMESTAMP
`, id, "munki_issues", issuesJSON)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "inserting stats for munki_issues id %d", id)
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
		query += ` JOIN hosts h ON (h.id = hm.host_id) WHERE h.team_id = ?`
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
		`
INSERT INTO aggregated_stats (id, type, json_value)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
    json_value = VALUES(json_value),
    updated_at = CURRENT_TIMESTAMP
`,
		id, "mdm_status", statusJson,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "inserting stats for mdm_status id %d", id)
	}
	return nil
}

func (ds *Datastore) generateAggregatedMDMSolutions(ctx context.Context, teamID *uint) error {
	id := uint(0)

	var results []fleet.AggregatedMDMSolutions
	query := `SELECT
				mdms.id,
				mdms.server_url,
				mdms.name,
				COUNT(DISTINCT hm.host_id) as hosts_count
			 FROM mobile_device_management_solutions mdms
			 INNER JOIN host_mdm hm
			 ON hm.mdm_id = mdms.id
`
	args := []interface{}{}
	if teamID != nil {
		args = append(args, *teamID)
		query += ` JOIN hosts h ON (h.id = hm.host_id) WHERE h.team_id = ?`
		id = *teamID
	}
	query += ` GROUP BY id, server_url, name`
	err := sqlx.SelectContext(ctx, ds.reader, &results, query, args...)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting aggregated data from host_mdm")
	}

	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling stats")
	}

	_, err = ds.writer.ExecContext(ctx,
		`
INSERT INTO aggregated_stats (id, type, json_value)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
    json_value = VALUES(json_value),
    updated_at = CURRENT_TIMESTAMP
`,
		id, "mdm_solutions", resultsJSON,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "inserting stats for mdm_solutions id %d", id)
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
			public_ip = ?,
			refetch_requested = ?,
			orbit_node_key = ?
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
		host.PublicIP,
		host.RefetchRequested,
		host.OrbitNodeKey,
		host.ID,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "save host with id %d", host.ID)
	}
	_, err = ds.writer.ExecContext(ctx, `
			UPDATE host_display_names
			SET display_name=?
			WHERE host_id=?`,
		host.DisplayName(),
		host.ID,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "update host_display_names for host id %d", host.ID)
	}
	return nil
}

// OSVersions gets the aggregated os version host counts. Records with the same name and version are combined into one count (e.g.,
// counts for the same macOS version on x86_64 and arm64 architectures are counted together.
// Results can be filtered using the following optional criteria: team id, platform, or name and
// version. Name cannot be used without version, and conversely, version cannot be used without name.
func (ds *Datastore) OSVersions(ctx context.Context, teamID *uint, platform *string, name *string, version *string) (*fleet.OSVersions, error) {
	if name != nil && version == nil {
		return nil, errors.New("invalid usage: cannot filter by name without version")
	}
	if name == nil && version != nil {
		return nil, errors.New("invalid usage: cannot filter by version without name")
	}

	query := `
SELECT
    json_value,
    updated_at
FROM aggregated_stats
WHERE
    id = ? AND
    type = 'os_versions'
`

	var row struct {
		JSONValue *json.RawMessage `db:"json_value"`
		UpdatedAt time.Time        `db:"updated_at"`
	}

	var args []interface{}
	if teamID == nil { // all hosts
		args = append(args, 0)
	} else {
		args = append(args, *teamID)
	}

	err := sqlx.GetContext(ctx, ds.reader, &row, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("OSVersions"))
		}
		return nil, err
	}

	res := &fleet.OSVersions{
		CountsUpdatedAt: row.UpdatedAt,
		OSVersions:      []fleet.OSVersion{},
	}

	var counts []fleet.OSVersion
	if row.JSONValue != nil {
		if err := json.Unmarshal(*row.JSONValue, &counts); err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
	}

	// filter counts by platform
	if platform != nil {
		var filtered []fleet.OSVersion
		for _, os := range counts {
			if *platform == os.Platform {
				filtered = append(filtered, os)
			}
		}
		counts = filtered
	}

	// aggregate counts by name and version
	byNameVers := make(map[string]fleet.OSVersion)
	for _, os := range counts {
		if name != nil &&
			version != nil &&
			*name != os.NameOnly &&
			*version != os.Version {
			continue
		}
		key := fmt.Sprintf("%s %s", os.NameOnly, os.Version)
		val, ok := byNameVers[key]
		if !ok {
			// omit os id
			byNameVers[key] = fleet.OSVersion{Name: os.Name, NameOnly: os.NameOnly, Version: os.Version, Platform: os.Platform, HostsCount: os.HostsCount}
		} else {
			newVal := val
			newVal.HostsCount += os.HostsCount
			byNameVers[key] = newVal
		}
	}

	for _, os := range byNameVers {
		res.OSVersions = append(res.OSVersions, os)
	}

	// Sort by os versions. We can't control the order when using json_arrayagg
	// See https://dev.mysql.com/doc/refman/5.7/en/aggregate-functions.html#function_json-arrayagg.
	sort.Slice(res.OSVersions, func(i, j int) bool { return res.OSVersions[i].Name < res.OSVersions[j].Name })

	return res, nil
}

// Aggregated stats for os versions are stored by team id with 0 representing the global case
// If existing team has no hosts, we explicity set the json value as an empty array
func (ds *Datastore) UpdateOSVersions(ctx context.Context) error {
	selectStmt := `
	SELECT
		COUNT(*) hosts_count,
		h.team_id,
		os.id,
		os.name,
		os.version,
		os.platform
	FROM hosts h
	JOIN host_operating_system hos ON h.id = hos.host_id
	JOIN operating_systems os ON hos.os_id = os.id
	GROUP BY team_id, os.id
	`

	var rows []struct {
		HostsCount int    `db:"hosts_count"`
		Name       string `db:"name"`
		Version    string `db:"version"`
		Platform   string `db:"platform"`
		ID         uint   `db:"id"`
		TeamID     *uint  `db:"team_id"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader, &rows, selectStmt); err != nil {
		return ctxerr.Wrap(ctx, err, "update os versions")
	}

	// each team has a slice of stats with team host counts per os version
	statsByTeamID := make(map[uint][]fleet.OSVersion)
	// stats are also aggregated globally per os version
	globalStats := make(map[uint]fleet.OSVersion)

	for _, r := range rows {
		os := fleet.OSVersion{
			HostsCount: r.HostsCount,
			Name:       fmt.Sprintf("%s %s", r.Name, r.Version),
			NameOnly:   r.Name,
			Version:    r.Version,
			Platform:   r.Platform,
			ID:         r.ID,
		}
		// increment global stats
		if _, ok := globalStats[os.ID]; !ok {
			globalStats[os.ID] = os
		} else {
			newStats := globalStats[os.ID]
			newStats.HostsCount += r.HostsCount
			globalStats[os.ID] = newStats
		}
		// push to team stats if applicable
		if r.TeamID != nil {
			statsByTeamID[*r.TeamID] = append(statsByTeamID[*r.TeamID], os)
		}
	}

	// if an existing team has no hosts assigned, we still want to store empty stats
	var teamIDs []uint
	if err := sqlx.SelectContext(ctx, ds.reader, &teamIDs, "SELECT id FROM teams"); err != nil {
		return ctxerr.Wrap(ctx, err, "update os versions")
	}
	for _, id := range teamIDs {
		if _, ok := statsByTeamID[id]; !ok {
			statsByTeamID[id] = []fleet.OSVersion{}
		}
	}

	// global stats are stored under id 0
	for _, os := range globalStats {
		statsByTeamID[0] = append(statsByTeamID[0], os)
	}

	// nothing to do so return early
	if len(statsByTeamID) < 1 {
		// log to help troubleshooting in case this happens
		level.Debug(ds.logger).Log("msg", "Cannot update aggregated stats for os versions: Check for records in operating_systems and host_perating_systems.")
		return nil
	}

	// assemble values as arguments for insert statement
	args := make([]interface{}, 0, len(statsByTeamID)*3)
	for id, stats := range statsByTeamID {
		jsonValue, err := json.Marshal(stats)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal os version stats")
		}
		args = append(args, id, "os_versions", jsonValue)
	}

	insertStmt := "INSERT INTO aggregated_stats (id, type, json_value) VALUES "
	insertStmt += strings.TrimSuffix(strings.Repeat("(?,?,?),", len(statsByTeamID)), ",")
	insertStmt += " ON DUPLICATE KEY UPDATE json_value = VALUES(json_value), updated_at = CURRENT_TIMESTAMP"

	if _, err := ds.writer.ExecContext(ctx, insertStmt, args...); err != nil {
		return ctxerr.Wrapf(ctx, err, "insert os versions into aggregated stats")
	}

	return nil
}

// EnrolledHostIDs returns the complete list of host IDs.
func (ds *Datastore) EnrolledHostIDs(ctx context.Context) ([]uint, error) {
	const stmt = `SELECT id FROM hosts`

	var ids []uint
	if err := sqlx.SelectContext(ctx, ds.reader, &ids, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get enrolled host IDs")
	}
	return ids, nil
}

// CountEnrolledHosts returns the current number of enrolled hosts.
func (ds *Datastore) CountEnrolledHosts(ctx context.Context) (int, error) {
	const stmt = `SELECT count(*) FROM hosts`

	var count int
	if err := sqlx.SelectContext(ctx, ds.reader, &count, stmt); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count enrolled host")
	}
	return count, nil
}

func (ds *Datastore) HostIDsByOSVersion(
	ctx context.Context,
	osVersion fleet.OSVersion,
	offset int,
	limit int,
) ([]uint, error) {
	var ids []uint

	stmt := dialect.From("hosts").
		Select("id").
		Where(
			goqu.C("platform").Eq(osVersion.Platform),
			goqu.C("os_version").Eq(osVersion.Name)).
		Order(goqu.I("id").Desc()).
		Offset(uint(offset)).
		Limit(uint(limit))

	sql, args, err := stmt.ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host IDs")
	}

	if err := sqlx.SelectContext(ctx, ds.reader, &ids, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host IDs")
	}

	return ids, nil
}

// ListHostBatteries returns battery information as reported by osquery for the identified host.
//
// Note: Because of a known osquery issue with M1 Macs, we are ignoring the stored `health` value
// in the db and replacing it at the service layer with custom a value determined by the cycle
// count. See https://github.com/fleetdm/fleet/pull/6782#discussion_r926103758.
// TODO: Update once the underlying osquery issue has been resolved.
func (ds *Datastore) ListHostBatteries(ctx context.Context, hid uint) ([]*fleet.HostBattery, error) {
	const stmt = `
    SELECT
      host_id,
      serial_number,
      cycle_count,
      health
    FROM
      host_batteries
    WHERE
      host_id = ?
`

	var batteries []*fleet.HostBattery
	if err := sqlx.SelectContext(ctx, ds.reader, &batteries, stmt, hid); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select host batteries")
	}
	return batteries, nil
}

// countHostNotResponding counts the hosts that haven't been submitting results for sent queries.
//
// Notes:
//   - We use `2 * interval`, because of the artificial jitter added to the intervals in Fleet.
//   - Default values for:
//   - host.DistributedInterval is usually 10s.
//   - svc.config.Osquery.DetailUpdateInterval is usually 1h.
//   - Count only includes hosts seen during the last 7 days.
func countHostsNotRespondingDB(ctx context.Context, db sqlx.QueryerContext, logger log.Logger, config config.FleetConfig) (int, error,
) {
	interval := config.Osquery.DetailUpdateInterval.Seconds()

	// The primary `WHERE` clause is intended to capture where Fleet hasn't received a distributed write
	// from the host during the interval since the host was last seen. Thus we assume the host
	// is having some issue in executing distributed queries or sending the results.
	// The subquery `WHERE` clause excludes from the count any hosts that were inactive during the
	// current seven-day statistics reporting period.
	sql := `
SELECT h.host_id FROM (
  SELECT hst.host_id, hst.seen_time, hosts.detail_updated_at, hosts.distributed_interval FROM hosts JOIN host_seen_times hst ON hosts.id = hst.host_id
  WHERE hst.seen_time >= DATE_SUB(NOW(), INTERVAL 7 DAY)
) h
WHERE
  TIME_TO_SEC(TIMEDIFF(h.seen_time, h.detail_updated_at)) >= (GREATEST(h.distributed_interval, ?) * 2)
`

	var ids []int
	if err := sqlx.SelectContext(ctx, db, &ids, sql, interval); err != nil {
		return len(ids), ctxerr.Wrap(ctx, err, "count hosts not responding")
	}
	if len(ids) > 0 {
		// We log to help troubleshooting in case this happens.
		level.Info(logger).Log("err", fmt.Sprintf("hosts detected that are not responding distributed queries %v", ids))
	}
	return len(ids), nil
}
