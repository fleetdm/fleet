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
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	hostSearchColumns             = []string{"hostname", "computer_name", "uuid", "hardware_serial", "primary_ip"}
	wildCardableHostSearchColumns = []string{"hostname", "computer_name"}
)

// Fixme: We should not make implementation details of the database schema part of the API.
var defaultHostColumnTableAliases = map[string]string{
	"created_at": "h.created_at",
	"updated_at": "h.updated_at",
}

func defaultHostColumnTableAlias(s string) string {
	if newCol, ok := defaultHostColumnTableAliases[s]; ok {
		return newCol
	}
	return s
}

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
			refetch_requested,
			hardware_serial,
			refetch_critical_queries_until
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			host.HardwareSerial,
			host.RefetchCriticalQueriesUntil,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new host")
		}
		id, _ := result.LastInsertId()
		host.ID = uint(id)

		_, err = tx.ExecContext(ctx,
			`INSERT INTO host_seen_times (host_id, seen_time) VALUES (?,?)`,
			host.ID, host.SeenTime,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new host seen time")
		}
		_, err = tx.ExecContext(ctx,
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

func (ds *Datastore) SaveHostPackStats(ctx context.Context, teamID *uint, hostID uint, stats []fleet.PackStats) error {
	return saveHostPackStatsDB(ctx, ds.writer(ctx), teamID, hostID, stats)
}

func saveHostPackStatsDB(ctx context.Context, db *sqlx.DB, teamID *uint, hostID uint, stats []fleet.PackStats) error {
	// NOTE: this implementation must be kept in sync with the async/batch version
	// in AsyncBatchSaveHostsScheduledQueryStats (in scheduled_queries.go) - that is,
	// the behaviour per host must be the same.

	var (
		userPacksArgs              []interface{}
		userPacksQueryCount        = 0
		scheduledQueriesArgs       []interface{}
		scheduledQueriesQueryCount = 0
	)

	for _, pack := range stats {
		if pack.PackName == "Global" || (teamID != nil && pack.PackName == fmt.Sprintf("team-%d", *teamID)) {
			for _, query := range pack.QueryStats {
				scheduledQueriesQueryCount++

				teamIDArg := uint(0)
				if pack.PackName != "Global" {
					teamIDArg = *teamID
				}
				scheduledQueriesArgs = append(scheduledQueriesArgs,
					teamIDArg,
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
		} else { // User 2017 packs
			for _, query := range pack.QueryStats {
				userPacksQueryCount++

				userPacksArgs = append(userPacksArgs,
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
	}

	if userPacksQueryCount == 0 && scheduledQueriesQueryCount == 0 {
		return nil
	}

	if scheduledQueriesQueryCount > 0 {
		// This query will import stats for queries (new format).
		values := strings.TrimSuffix(strings.Repeat("((SELECT q.id FROM queries q WHERE COALESCE(q.team_id, 0) = ? AND q.name = ?),?,?,?,?,?,?,?,?,?,?),", scheduledQueriesQueryCount), ",")
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
		if _, err := db.ExecContext(ctx, sql, scheduledQueriesArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert query schedule stats")
		}
	}

	if userPacksQueryCount > 0 {
		// This query will import stats for 2017 packs.
		// NOTE(lucas): If more than one scheduled query reference the same query then only one of the stats will be written.
		values := strings.TrimSuffix(strings.Repeat("((SELECT sq.query_id FROM scheduled_queries sq JOIN packs p ON (sq.pack_id = p.id) WHERE p.pack_type IS NULL AND p.name = ? AND sq.name = ?),?,?,?,?,?,?,?,?,?,?),", userPacksQueryCount), ",")
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
		if _, err := db.ExecContext(ctx, sql, userPacksArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert pack stats")
		}
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

// loadhostPacksStatsDB will load all the "2017 pack" stats for the given host. The scheduled
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
		goqu.On(goqu.I("sq.query_id").Eq(goqu.I("q.id"))),
	).LeftJoin(
		dialect.From("scheduled_query_stats").As("sqs").Where(
			goqu.I("host_id").Eq(hid),
		),
		goqu.On(goqu.I("sqs.scheduled_query_id").Eq(goqu.I("sq.query_id"))),
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

func loadHostScheduledQueryStatsDB(ctx context.Context, db sqlx.QueryerContext, hid uint, hostPlatform string, teamID *uint) ([]fleet.QueryStats, error) {
	var teamID_ uint
	if teamID != nil {
		teamID_ = *teamID
	}
	ds := dialect.From(goqu.I("queries").As("q")).Select(
		goqu.I("q.id"),
		goqu.I("q.name"),
		goqu.I("q.description"),
		goqu.I("q.team_id"),
		goqu.I("q.schedule_interval").As("schedule_interval"),
		goqu.COALESCE(goqu.I("sqs.average_memory"), 0).As("average_memory"),
		goqu.COALESCE(goqu.I("sqs.denylisted"), false).As("denylisted"),
		goqu.COALESCE(goqu.I("sqs.executions"), 0).As("executions"),
		goqu.COALESCE(goqu.I("sqs.last_executed"), goqu.L("timestamp(?)", pastDate)).As("last_executed"),
		goqu.COALESCE(goqu.I("sqs.output_size"), 0).As("output_size"),
		goqu.COALESCE(goqu.I("sqs.system_time"), 0).As("system_time"),
		goqu.COALESCE(goqu.I("sqs.user_time"), 0).As("user_time"),
		goqu.COALESCE(goqu.I("sqs.wall_time"), 0).As("wall_time"),
	).LeftJoin(
		dialect.From("scheduled_query_stats").As("sqs").Where(
			goqu.I("host_id").Eq(hid),
		),
		goqu.On(goqu.I("sqs.scheduled_query_id").Eq(goqu.I("q.id"))),
	).Where(
		goqu.And(
			goqu.Or(
				// sq.platform empty or NULL means the scheduled query is set to
				// run on all hosts.
				goqu.I("q.platform").Eq(""),
				goqu.I("q.platform").IsNull(),
				// scheduled_queries.platform can be a comma-separated list of
				// platforms, e.g. "darwin,windows".
				goqu.L("FIND_IN_SET(?, q.platform)", fleet.PlatformFromHost(hostPlatform)).Neq(0),
			),
			goqu.I("q.schedule_interval").Gt(0),
			goqu.I("q.automations_enabled").IsTrue(),
			goqu.Or(
				goqu.I("q.team_id").IsNull(),
				goqu.I("q.team_id").Eq(teamID_),
			),
		),
	)
	sql, args, err := ds.ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sql build")
	}
	var stats []fleet.QueryStats
	if err := sqlx.SelectContext(ctx, db, &stats, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load query stats")
	}
	return stats, nil
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
	"host_orbit_info",
	"host_munki_issues",
	"host_display_names",
	"windows_updates",
	"host_disks",
	"operating_system_vulnerabilities",
	"host_updates",
	"host_disk_encryption_keys",
	"host_software_installed_paths",
	"host_script_results",
}

// NOTE: The following tables are explicity excluded from hostRefs list and accordingly are not
// deleted from when a host is deleted in Fleet:
// - host_dep_assignments

// additionalHostRefsByUUID are host refs cannot be deleted using the host.id like the hostRefs
// above. They use the host.uuid instead. Additionally, the column name that refers to
// the host.uuid is not always named the same, so the map key is the table name
// and the map value is the column name to match to the host.uuid.
var additionalHostRefsByUUID = map[string]string{
	"host_mdm_apple_profiles":           "host_uuid",
	"host_mdm_apple_bootstrap_packages": "host_uuid",
}

func (ds *Datastore) DeleteHost(ctx context.Context, hid uint) error {
	delHostRef := func(tx sqlx.ExtContext, table string) error {
		_, err := tx.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE host_id=?`, table), hid)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "deleting %s for host %d", table, hid)
		}
		return nil
	}

	// load just the host uuid for the MDM tables that rely on this to be cleared.
	var hostUUID string
	if err := ds.writer(ctx).GetContext(ctx, &hostUUID, `SELECT uuid FROM hosts WHERE id = ?`, hid); err != nil {
		return ctxerr.Wrapf(ctx, err, "get uuid for host %d", hid)
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

		// no point trying the uuid-based tables if the host's uuid is missing
		if hostUUID != "" {
			for table, col := range additionalHostRefsByUUID {
				if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM `%s` WHERE `%s`=?", table, col), hostUUID); err != nil {
					return ctxerr.Wrapf(ctx, err, "deleting %s for host uuid %s", table, hostUUID)
				}
			}
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
  h.refetch_critical_queries_until,
  h.team_id,
  h.policy_updated_at,
  h.public_ip,
  COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
  COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available,
  hd.encrypted as disk_encryption_enabled,
  COALESCE(hst.seen_time, h.created_at) AS seen_time,
  t.name AS team_name,
  COALESCE(hu.software_updated_at, h.created_at) AS software_updated_at,
  (
    SELECT
      additional
    FROM
      host_additional
    WHERE
      host_id = h.id
  ) AS additional,
  COALESCE(failing_policies.count, 0) AS failing_policies_count,
  COALESCE(failing_policies.count, 0) AS total_issues_count
  ` + hostMDMSelect + `
FROM
  hosts h
  LEFT JOIN teams t ON (h.team_id = t.id)
  LEFT JOIN host_seen_times hst ON (h.id = hst.host_id)
  LEFT JOIN host_updates hu ON (h.id = hu.host_id)
  LEFT JOIN host_disks hd ON hd.host_id = h.id
  ` + hostMDMJoin + `
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
	err := sqlx.GetContext(ctx, ds.reader(ctx), &host, sqlStatement, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Host").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host by id")
	}
	if host.DiskEncryptionEnabled != nil && !(*host.DiskEncryptionEnabled) && fleet.IsLinux(host.Platform) {
		// omit disk encryption information for linux if it is not enabled, as we
		// cannot know for sure that it is not encrypted (See
		// https://github.com/fleetdm/fleet/issues/3906).
		host.DiskEncryptionEnabled = nil
	}

	packStats, err := loadHostPackStatsDB(ctx, ds.reader(ctx), host.ID, host.Platform)
	if err != nil {
		return nil, err
	}
	host.PackStats = packStats
	queriesStats, err := loadHostScheduledQueryStatsDB(ctx, ds.reader(ctx), host.ID, host.Platform, host.TeamID)
	if err != nil {
		return nil, err
	}
	var (
		globalQueriesStats   []fleet.QueryStats
		hostTeamQueriesStats []fleet.QueryStats
	)
	for _, queryStats := range queriesStats {
		if queryStats.TeamID == nil {
			globalQueriesStats = append(globalQueriesStats, queryStats)
		} else {
			hostTeamQueriesStats = append(hostTeamQueriesStats, queryStats)
		}
	}
	if len(globalQueriesStats) > 0 {
		host.PackStats = append(host.PackStats, fleet.PackStats{
			PackName:   "Global",
			Type:       "global",
			QueryStats: queryStatsToScheduledQueryStats(globalQueriesStats, "Global"),
		})
	}
	if host.TeamID != nil && len(hostTeamQueriesStats) > 0 {
		team, err := ds.Team(ctx, *host.TeamID)
		if err != nil {
			return nil, err
		}
		host.PackStats = append(host.PackStats, fleet.PackStats{
			PackName:   "Team: " + team.Name,
			Type:       fmt.Sprintf("team-%d", team.ID),
			QueryStats: queryStatsToScheduledQueryStats(hostTeamQueriesStats, "Team: "+team.Name),
		})
	}

	users, err := loadHostUsersDB(ctx, ds.reader(ctx), host.ID)
	if err != nil {
		return nil, err
	}
	host.Users = users

	return &host, nil
}

func queryStatsToScheduledQueryStats(queriesStats []fleet.QueryStats, packName string) []fleet.ScheduledQueryStats {
	scheduledQueriesStats := make([]fleet.ScheduledQueryStats, 0, len(queriesStats))
	for _, queryStats := range queriesStats {
		scheduledQueriesStats = append(scheduledQueriesStats, fleet.ScheduledQueryStats{
			ScheduledQueryName: queryStats.Name,
			ScheduledQueryID:   queryStats.ID,
			QueryName:          queryStats.Name,
			Description:        queryStats.Description,
			PackName:           packName,
			AverageMemory:      queryStats.AverageMemory,
			Denylisted:         queryStats.Denylisted,
			Executions:         queryStats.Executions,
			Interval:           queryStats.Interval,
			LastExecuted:       queryStats.LastExecuted,
			OutputSize:         queryStats.OutputSize,
			SystemTime:         queryStats.SystemTime,
			UserTime:           queryStats.UserTime,
			WallTime:           queryStats.WallTime,
		})
	}
	return scheduledQueriesStats
}

// hostMDMSelect is the SQL fragment used to construct the JSON object
// of MDM host data. It assumes that hostMDMJoin is included in the query.
const hostMDMSelect = `,
	JSON_OBJECT(
		'enrollment_status',
		CASE
			WHEN hmdm.is_server = 1 THEN NULL
			WHEN hmdm.enrolled = 1 AND hmdm.installed_from_dep = 0 THEN 'On (manual)'
			WHEN hmdm.enrolled = 1 AND hmdm.installed_from_dep = 1 THEN 'On (automatic)'
			WHEN hmdm.enrolled = 0 AND hmdm.installed_from_dep = 1 THEN 'Pending'
			WHEN hmdm.enrolled = 0 AND hmdm.installed_from_dep = 0 THEN 'Off'
			ELSE NULL
		END,
		'server_url',
		CASE
			WHEN hmdm.is_server = 1 THEN NULL
			ELSE hmdm.server_url
		END,
		'encryption_key_available',
		CASE
                       /* roberto: this is the only way I have found for MySQL to
                        * return true and false instead of 0 and 1 in the JSON, the
                        * unmarshaller was having problems converting int values to
                        * booleans.
                        */
			WHEN hdek.decryptable IS NULL OR hdek.decryptable = 0 THEN CAST(FALSE AS JSON)
			ELSE CAST(TRUE AS JSON)
		END,
		'raw_decryptable',
		CASE
			WHEN hdek.host_id IS NULL THEN -1
			ELSE hdek.decryptable
		END,
		'name', hmdm.name
	) mdm_host_data
	`

// hostMDMJoin is the SQL fragment used to join MDM-related tables to the hosts table. It is a
// dependency of the hostMDMSelect fragment.
const hostMDMJoin = `
  LEFT JOIN (
	SELECT
	  host_mdm.is_server,
	  host_mdm.enrolled,
	  host_mdm.installed_from_dep,
	  host_mdm.server_url,
	  host_mdm.mdm_id,
	  host_mdm.host_id,
	  name
	FROM
	  host_mdm
	  LEFT JOIN mobile_device_management_solutions ON host_mdm.mdm_id = mobile_device_management_solutions.id
  ) hmdm ON hmdm.host_id = h.id
  LEFT JOIN host_disk_encryption_keys hdek ON hdek.host_id = h.id
  `

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

// HostFailingPoliciesCountOptimPageSizeThreshold is the value of the page size that determines whether
// to run an optimized version of the hosts queries when pagination is used.
//
// If the page size is under this value then the queries will be optimized assuming a low number of hosts.
// If the page size is 0 or higher than this value then the queries will be optimized assuming a high number of hosts.
//
// IMPORTANT: The UI currently always uses PerPage=50 to list hosts. For better performance,
// HostFailingPoliciesCountOptimPageSizeThreshold should always be higher than what the UI uses.
//
// The optimization consists on calculating the failing policy count (which involves querying a large table, `policy_membership`)
// differently depending on the page size:
//   - When the page size is short (lower than or equal to this value) then hosts are queried and filtered first, and
//     then the failure policy count is calculated on such hosts only (with an IN clause).
//   - When the page size is large (higher than this value) or ALL hosts are being retrieved then the hosts are
//     filtered and their failing policy count are calculated on the same query (the IN clause performs worse
//     than a LEFT JOIN when the number of rows is high).
var HostFailingPoliciesCountOptimPageSizeThreshold = 100

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
    h.refetch_critical_queries_until,
    h.team_id,
    h.policy_updated_at,
    h.public_ip,
    h.orbit_node_key,
    COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
    COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available,
    COALESCE(hst.seen_time, h.created_at) AS seen_time,
    t.name AS team_name,
    COALESCE(hu.software_updated_at, h.created_at) AS software_updated_at
	`

	sql += hostMDMSelect

	if opt.DeviceMapping {
		sql += `,
    COALESCE(dm.device_mapping, 'null') as device_mapping
		`
	}

	// See definition of HostFailingPoliciesCountOptimPageSizeThreshold for more details.
	useHostPaginationOptim := opt.PerPage != 0 && opt.PerPage <= uint(HostFailingPoliciesCountOptimPageSizeThreshold)

	if !opt.DisableFailingPolicies && !useHostPaginationOptim {
		sql += `,
		COALESCE(failing_policies.count, 0) AS failing_policies_count,
		COALESCE(failing_policies.count, 0) AS total_issues_count
		`
	}

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

	leftJoinFailingPolicies := !useHostPaginationOptim
	sql, params = ds.applyHostFilters(opt, sql, filter, params, leftJoinFailingPolicies)

	hosts := []*fleet.Host{}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, sql, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list hosts")
	}

	if !opt.DisableFailingPolicies && useHostPaginationOptim {
		var err error
		hosts, err = ds.UpdatePolicyFailureCountsForHosts(ctx, hosts)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "update policy failure counts for hosts")
		}
	}

	return hosts, nil
}

// TODO(Sarah): Do we need to reconcile mutually exclusive filters?
func (ds *Datastore) applyHostFilters(opt fleet.HostListOptions, sql string, filter fleet.TeamFilter, params []interface{}, leftJoinFailingPolicies bool) (string, []interface{}) {
	opt.OrderKey = defaultHostColumnTableAlias(opt.OrderKey)

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

	failingPoliciesJoin := ""
	if !opt.DisableFailingPolicies && leftJoinFailingPolicies {
		failingPoliciesJoin = `LEFT JOIN (
		    SELECT host_id, count(*) as count FROM policy_membership WHERE passes = 0
		    GROUP BY host_id
		) as failing_policies ON (h.id=failing_policies.host_id)`
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
    LEFT JOIN host_updates hu ON (h.id = hu.host_id)
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
    `,

		// JOINs
		hostMDMJoin,
		deviceMappingJoin,
		policyMembershipJoin,
		failingPoliciesJoin,
		operatingSystemJoin,
		munkiJoin,
		displayNameJoin,

		// Conditions
		ds.whereFilterHostsByTeams(filter, "h"),
		softwareFilter,
		munkiFilter,
		lowDiskSpaceFilter,
	)

	now := ds.clock.Now()
	sql, params = filterHostsByStatus(now, sql, opt, params)
	sql, params = filterHostsByTeam(sql, opt, params)
	sql, params = filterHostsByPolicy(sql, opt, params)
	sql, params = filterHostsByMDM(sql, opt, params)
	sql, params = filterHostsByMacOSSettingsStatus(sql, opt, params)
	sql, params = filterHostsByMacOSDiskEncryptionStatus(sql, opt, params)
	sql, params = filterHostsByMDMBootstrapPackageStatus(sql, opt, params)
	sql, params = filterHostsByOS(sql, opt, params)
	sql, params, _ = hostSearchLike(sql, params, opt.MatchQuery, hostSearchColumns...)
	sql, params = appendListOptionsWithCursorToSQL(sql, params, &opt.ListOptions)

	return sql, params
}

func filterHostsByTeam(sql string, opt fleet.HostListOptions, params []interface{}) (string, []interface{}) {
	if opt.TeamFilter == nil {
		// default "all teams" option
		return sql, params
	}

	if *opt.TeamFilter == uint(0) {
		// "no team" option (where TeamFilter is explicitly zero) excludes hosts that are assigned to any team
		sql += ` AND h.team_id IS NULL`
		return sql, params
	}

	sql += ` AND h.team_id = ?`
	params = append(params, *opt.TeamFilter)

	return sql, params
}

func filterHostsByMDM(sql string, opt fleet.HostListOptions, params []interface{}) (string, []interface{}) {
	if opt.MDMIDFilter != nil {
		sql += ` AND hmdm.mdm_id = ?`
		params = append(params, *opt.MDMIDFilter)
	}
	if opt.MDMNameFilter != nil {
		sql += ` AND hmdm.name = ?`
		params = append(params, *opt.MDMNameFilter)
	}
	if opt.MDMEnrollmentStatusFilter != "" {
		// NOTE: ds.UpdateHostTablesOnMDMUnenroll sets installed_from_dep = 0 so DEP hosts are not counted as pending after unenrollment
		switch opt.MDMEnrollmentStatusFilter {
		case fleet.MDMEnrollStatusAutomatic:
			sql += ` AND hmdm.enrolled = 1 AND hmdm.installed_from_dep = 1`
		case fleet.MDMEnrollStatusManual:
			sql += ` AND hmdm.enrolled = 1 AND hmdm.installed_from_dep = 0`
		case fleet.MDMEnrollStatusEnrolled:
			sql += ` AND hmdm.enrolled = 1`
		case fleet.MDMEnrollStatusPending:
			sql += ` AND hmdm.enrolled = 0 AND hmdm.installed_from_dep = 1`
		case fleet.MDMEnrollStatusUnenrolled:
			sql += ` AND hmdm.enrolled = 0 AND hmdm.installed_from_dep = 0`
		}
	}
	if opt.MDMNameFilter != nil || opt.MDMIDFilter != nil || opt.MDMEnrollmentStatusFilter != "" {
		sql += ` AND NOT COALESCE(hmdm.is_server, false) AND h.platform IN('darwin', 'windows')`
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

func filterHostsByMacOSSettingsStatus(sql string, opt fleet.HostListOptions, params []interface{}) (string, []interface{}) {
	if !opt.MacOSSettingsFilter.IsValid() {
		return sql, params
	}

	newSQL := ""
	if opt.TeamFilter == nil {
		// macOS settings filter is not compatible with the "all teams" option so append the "no
		// team" filter here (note that filterHostsByTeam applies the "no team" filter if TeamFilter == 0)
		newSQL += ` AND h.team_id IS NULL`
	}

	var subquery string
	var subqueryParams []interface{}
	switch opt.MacOSSettingsFilter {
	case fleet.MacOSSettingsFailed:
		subquery, subqueryParams = subqueryHostsMacOSSettingsStatusFailing()
	case fleet.MacOSSettingsPending:
		subquery, subqueryParams = subqueryHostsMacOSSettingsStatusPending()
	case fleet.MacOSSettingsVerifying:
		subquery, subqueryParams = subqueryHostsMacOSSetttingsStatusVerifying()
	case fleet.MacOSSettingsVerified:
		subquery, subqueryParams = subqueryHostsMacOSSetttingsStatusVerified()
	}
	if subquery != "" {
		newSQL += fmt.Sprintf(` AND EXISTS (%s)`, subquery)
	}

	return sql + newSQL, append(params, subqueryParams...)
}

func filterHostsByMacOSDiskEncryptionStatus(sql string, opt fleet.HostListOptions, params []interface{}) (string, []interface{}) {
	if !opt.MacOSSettingsDiskEncryptionFilter.IsValid() {
		return sql, params
	}

	var subquery string
	var subqueryParams []interface{}
	switch opt.MacOSSettingsDiskEncryptionFilter {
	case fleet.DiskEncryptionVerified:
		subquery, subqueryParams = subqueryDiskEncryptionVerified()
	case fleet.DiskEncryptionVerifying:
		subquery, subqueryParams = subqueryDiskEncryptionVerifying()
	case fleet.DiskEncryptionActionRequired:
		subquery, subqueryParams = subqueryDiskEncryptionActionRequired()
	case fleet.DiskEncryptionEnforcing:
		subquery, subqueryParams = subqueryDiskEncryptionEnforcing()
	case fleet.DiskEncryptionFailed:
		subquery, subqueryParams = subqueryDiskEncryptionFailed()
	case fleet.DiskEncryptionRemovingEnforcement:
		subquery, subqueryParams = subqueryDiskEncryptionRemovingEnforcement()
	}

	return sql + fmt.Sprintf(` AND EXISTS (%s)`, subquery), append(params, subqueryParams...)
}

func filterHostsByMDMBootstrapPackageStatus(sql string, opt fleet.HostListOptions, params []interface{}) (string, []interface{}) {
	if opt.MDMBootstrapPackageFilter == nil || !opt.MDMBootstrapPackageFilter.IsValid() {
		return sql, params
	}

	subquery := `SELECT 1
	-- we need to JOIN on hosts again to account for 'pending' hosts that
	-- haven't been enrolled yet, and thus don't have an uuid nor a matching
	-- entry in nano_command_results.
        FROM
            hosts hh
        LEFT JOIN
            host_mdm_apple_bootstrap_packages hmabp ON hmabp.host_uuid = hh.uuid
        LEFT JOIN
            nano_command_results ncr ON ncr.command_uuid = hmabp.command_uuid
        WHERE
	      hh.id = h.id AND hmdm.installed_from_dep = 1`

	// NOTE: The approach below assumes that there is only one bootstrap package per host. If this
	// is not the case, then the query will need to be updated to use a GROUP BY and HAVING
	// clause to ensure that the correct status is returned.
	switch *opt.MDMBootstrapPackageFilter {
	case fleet.MDMBootstrapPackageFailed:
		subquery += ` AND ncr.status = 'Error'`
	case fleet.MDMBootstrapPackagePending:
		subquery += ` AND (ncr.status IS NULL OR (ncr.status != 'Acknowledged' AND ncr.status != 'Error'))`
	case fleet.MDMBootstrapPackageInstalled:
		subquery += ` AND ncr.status = 'Acknowledged'`
	}

	newSQL := ""
	if opt.TeamFilter == nil {
		// macOS setup filter is not compatible with the "all teams" option so append the "no
		// team" filter here (note that filterHostsByTeam applies the "no team" filter if TeamFilter == 0)
		newSQL += ` AND h.team_id IS NULL`
	}
	newSQL += fmt.Sprintf(` AND EXISTS (
        %s
    )
    `, subquery)

	return sql + newSQL, params
}

func (ds *Datastore) CountHosts(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) (int, error) {
	sql := `SELECT count(*) `

	// Ignore pagination in count.
	opt.Page = 0
	opt.PerPage = 0
	// We don't need the failing policy counts of each host for counting hosts.
	leftJoinFailingPolicies := false

	var params []interface{}
	sql, params = ds.applyHostFilters(opt, sql, filter, params, leftJoinFailingPolicies)

	var count int
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, sql, params...); err != nil {
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
		  hardware_serial = '' AND
		  created_at < (? - INTERVAL 5 MINUTE)`
		if err := sqlx.SelectContext(ctx, tx, &ids, selectIDs, now); err != nil {
			return ctxerr.Wrap(ctx, err, "load incoming hosts to cleanup")
		}

		cleanupHostDisplayName := fmt.Sprintf(
			`DELETE FROM host_display_names WHERE host_id IN (%s)`,
			selectIDs,
		)
		if _, err := tx.ExecContext(ctx, cleanupHostDisplayName, now); err != nil {
			return ctxerr.Wrap(ctx, err, "cleanup host_display_names")
		}

		cleanupHosts := `
		DELETE FROM hosts
		WHERE hostname = '' AND osquery_version = '' AND hardware_serial = ''
		AND created_at < (? - INTERVAL 5 MINUTE)
		`
		if _, err := tx.ExecContext(ctx, cleanupHosts, now); err != nil {
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
	err = sqlx.GetContext(ctx, ds.reader(ctx), &summary, stmt, args...)
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
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &platforms, stmt, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating host platforms statistics")
	}
	summary.Platforms = platforms

	return &summary, nil
}

// Attempts to find the matching host ID by osqueryID, host UUID or serial
// number. Any of those fields can be left empty if not available, and it will
// use the best match in this order:
// * if it matched on osquery_host_id (with osqueryID or uuid), use that host
// * otherwise if it matched on uuid, use that host
// * otherwise use the match on serial
//
// Note that in general, all options should result in a single match anyway.
// It's just that our DB schema doesn't enforce this (only osquery_host_id has
// a unique constraint). Also, due to things like VMs, the serial number is not
// guaranteed to match a single host. For that reason, we only attempt the
// serial number lookup if Fleet MDM is enabled on the server (as we must be
// able to match by serial in this scenario, since this is the only information
// we get when enrolling hosts via Apple DEP) AND if the matched host is on the
// macOS platform (darwin).
func matchHostDuringEnrollment(ctx context.Context, q sqlx.QueryerContext, isMDMEnabled bool, osqueryID, uuid, serial string) (uint, time.Time, error) {
	type hostMatch struct {
		ID             uint
		LastEnrolledAt time.Time `db:"last_enrolled_at"`
		Priority       int
	}

	var (
		query strings.Builder // note that writes to this cannot fail
		args  []interface{}
		rows  []hostMatch
	)

	if osqueryID != "" || uuid != "" {
		_, _ = query.WriteString(`(SELECT id, last_enrolled_at, 1 priority FROM hosts WHERE osquery_host_id = ?)`)
		if osqueryID == "" {
			// special-case, if there's no osquery identifier, use the uuid
			osqueryID = uuid
		}
		args = append(args, osqueryID)
	}

	// TODO(mna): for now do not match by UUID on the `uuid` field as it is not indexed.
	// See https://github.com/fleetdm/fleet/issues/9372 and
	// https://github.com/fleetdm/fleet/issues/9033#issuecomment-1411150758
	// (the latter shows that it might not be top priority to index this field, if we're
	// going to recommend using the host uuid as osquery identifier, as osquery_host_id
	// _is_ indexed and unique).
	// if uuid != "" {
	//	if query.Len() > 0 {
	//		_, _ = query.WriteString(" UNION ")
	//	}
	//	_, _ = query.WriteString(`(SELECT id, last_enrolled_at, 2 priority FROM hosts WHERE uuid = ? ORDER BY id LIMIT 1)`)
	//	args = append(args, uuid)
	// }

	if serial != "" && isMDMEnabled {
		if query.Len() > 0 {
			_, _ = query.WriteString(" UNION ")
		}
		_, _ = query.WriteString(`(SELECT id, last_enrolled_at, 3 priority FROM hosts WHERE hardware_serial = ? AND platform = ? ORDER BY id LIMIT 1)`)
		args = append(args, serial, "darwin")
	}

	if err := sqlx.SelectContext(ctx, q, &rows, query.String(), args...); err != nil {
		return 0, time.Time{}, ctxerr.Wrap(ctx, err, "match host during enrollment")
	}
	if len(rows) == 0 {
		return 0, time.Time{}, sql.ErrNoRows
	}
	sort.Slice(rows, func(i, j int) bool {
		l, r := rows[i], rows[j]
		return l.Priority < r.Priority
	})
	return rows[0].ID, rows[0].LastEnrolledAt, nil
}

func (ds *Datastore) EnrollOrbit(ctx context.Context, isMDMEnabled bool, hostInfo fleet.OrbitHostInfo, orbitNodeKey string, teamID *uint) (*fleet.Host, error) {
	if orbitNodeKey == "" {
		return nil, ctxerr.New(ctx, "orbit node key is empty")
	}
	if hostInfo.HardwareUUID == "" {
		return nil, ctxerr.New(ctx, "hardware uuid is empty")
	}
	// NOTE: allow an empty serial, currently it is empty for Windows.

	var host fleet.Host
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		hostID, _, err := matchHostDuringEnrollment(ctx, tx, isMDMEnabled, "", hostInfo.HardwareUUID, hostInfo.HardwareSerial)
		switch {
		case err == nil:
			sqlUpdate := `
      UPDATE
        hosts
      SET
        orbit_node_key = ?,
        uuid = COALESCE(NULLIF(uuid, ''), ?),
        osquery_host_id = COALESCE(NULLIF(osquery_host_id, ''), ?),
        hardware_serial = COALESCE(NULLIF(hardware_serial, ''), ?),
        team_id = ?
      WHERE id = ?`
			_, err := tx.ExecContext(ctx, sqlUpdate,
				orbitNodeKey,
				hostInfo.HardwareUUID,
				hostInfo.HardwareUUID,
				hostInfo.HardwareSerial,
				teamID,
				hostID,
			)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "orbit enroll error updating host details")
			}
			host.ID = hostID

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
					uuid,
					node_key,
					team_id,
					refetch_requested,
					orbit_node_key,
					hardware_serial,
					hostname,
					platform
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?)
			`
			result, err := tx.ExecContext(ctx, sqlInsert,
				zeroTime,
				zeroTime,
				zeroTime,
				zeroTime,
				hostInfo.HardwareUUID,
				hostInfo.HardwareUUID,
				orbitNodeKey,
				teamID,
				orbitNodeKey,
				hostInfo.HardwareSerial,
				hostInfo.Hostname,
				hostInfo.Platform,
			)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "orbit enroll error inserting host details")
			}
			hostID, _ := result.LastInsertId()
			const sqlHostDisplayName = `
				INSERT INTO host_display_names (host_id, display_name) VALUES (?, '')
			`
			_, err = tx.ExecContext(ctx, sqlHostDisplayName, hostID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "insert host_display_names")
			}
			host.ID = uint(hostID)

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
func (ds *Datastore) EnrollHost(ctx context.Context, isMDMEnabled bool, osqueryHostID, hardwareUUID, hardwareSerial, nodeKey string, teamID *uint, cooldown time.Duration) (*fleet.Host, error) {
	if osqueryHostID == "" {
		return nil, ctxerr.New(ctx, "missing osquery host identifier")
	}

	var host fleet.Host
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		zeroTime := time.Unix(0, 0).Add(24 * time.Hour)

		matchedID, lastEnrolledAt, err := matchHostDuringEnrollment(ctx, tx, isMDMEnabled, osqueryHostID, hardwareUUID, hardwareSerial)
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
					refetch_requested,
					uuid,
					hardware_serial
				) VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)
			`
			result, err := tx.ExecContext(ctx, sqlInsert, zeroTime, zeroTime, zeroTime, osqueryHostID, nodeKey, teamID, hardwareUUID, hardwareSerial)
			if err != nil {
				level.Info(ds.logger).Log("hostIDError", err.Error())
				return ctxerr.Wrap(ctx, err, "insert host")
			}
			hostID, _ := result.LastInsertId()
			const sqlHostDisplayName = `
				INSERT INTO host_display_names (host_id, display_name) VALUES (?, '')
			`
			_, err = tx.ExecContext(ctx, sqlHostDisplayName, hostID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "insert host_display_names")
			}
			matchedID = uint(hostID)

		default:
			// Prevent hosts from enrolling too often with the same identifier.
			// Prior to adding this we saw many hosts (probably VMs) with the
			// same identifier competing for enrollment and causing perf issues.
			if cooldown > 0 && time.Since(lastEnrolledAt) < cooldown {
				return backoff.Permanent(ctxerr.Errorf(ctx, "host identified by %s enrolling too often", osqueryHostID))
			}

			if err := deleteAllPolicyMemberships(ctx, tx, []uint{matchedID}); err != nil {
				return ctxerr.Wrap(ctx, err, "cleanup policy membership on re-enroll")
			}

			// Update existing host record
			sqlUpdate := `
				UPDATE hosts
				SET node_key = ?,
				team_id = ?,
				last_enrolled_at = NOW(),
				osquery_host_id = ?,
				uuid = COALESCE(NULLIF(uuid, ''), ?),
				hardware_serial = COALESCE(NULLIF(hardware_serial, ''), ?)
				WHERE id = ?
			`
			_, err := tx.ExecContext(ctx, sqlUpdate, nodeKey, teamID, osqueryHostID, hardwareUUID, hardwareSerial, matchedID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "update host")
			}
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO host_seen_times (host_id, seen_time) VALUES (?, ?)
			ON DUPLICATE KEY UPDATE seen_time = VALUES(seen_time)`,
			matchedID, time.Now().UTC())
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
        h.refetch_critical_queries_until,
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
		err = sqlx.GetContext(ctx, tx, &host, sqlSelect, matchedID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting the host to return")
		}
		_, err = tx.ExecContext(ctx, `INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, (SELECT id FROM labels WHERE name = 'All Hosts' AND label_type = 1))`, matchedID)
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
	// nolint the statements are closed in Datastore.Close.
	if stmt := ds.loadOrPrepareStmt(ctx, query); stmt != nil {
		err = stmt.GetContext(ctx, dest, args...)
	} else {
		err = sqlx.GetContext(ctx, ds.reader(ctx), dest, query, args...)
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
      h.refetch_critical_queries_until,
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

type hostWithMDMInfo struct {
	fleet.Host
	HostID           *uint   `db:"host_id"`
	Enrolled         *bool   `db:"enrolled"`
	ServerURL        *string `db:"server_url"`
	InstalledFromDep *bool   `db:"installed_from_dep"`
	IsServer         *bool   `db:"is_server"`
	MDMID            *uint   `db:"mdm_id"`
	Name             *string `db:"name"`
}

// LoadHostByOrbitNodeKey loads the whole host identified by the node key.
// If the node key is invalid it returns a NotFoundError.
func (ds *Datastore) LoadHostByOrbitNodeKey(ctx context.Context, nodeKey string) (*fleet.Host, error) {
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
      h.refetch_critical_queries_until,
      h.team_id,
      h.policy_updated_at,
      h.public_ip,
      h.orbit_node_key,
      hm.host_id,
      hm.enrolled,
      hm.server_url,
      hm.installed_from_dep,
      hm.mdm_id,
      COALESCE(hm.is_server, false) AS is_server,
      COALESCE(mdms.name, ?) AS name,
      COALESCE(hdek.reset_requested, false) AS disk_encryption_reset_requested,
      IF(hdep.host_id AND ISNULL(hdep.deleted_at), true, false) AS dep_assigned_to_fleet,
      t.name as team_name
    FROM
      hosts h
    LEFT OUTER JOIN
      host_mdm hm
    ON
      hm.host_id = h.id
    LEFT OUTER JOIN
      host_dep_assignments hdep
    ON
      hdep.host_id = h.id
    LEFT OUTER JOIN
      mobile_device_management_solutions mdms
    ON
      hm.mdm_id = mdms.id
    LEFT OUTER JOIN
      host_disk_encryption_keys hdek
    ON
      hdek.host_id = h.id
    LEFT OUTER JOIN
      teams t
    ON
      h.team_id = t.id
    WHERE
      h.orbit_node_key = ?`

	var hostWithMDM hostWithMDMInfo
	switch err := ds.getContextTryStmt(ctx, &hostWithMDM, query, fleet.UnknownMDMName, nodeKey); {
	case err == nil:
		host := hostWithMDM.Host
		// leave MDMInfo nil unless it has mdm information
		if hostWithMDM.HostID != nil {
			host.MDMInfo = &fleet.HostMDM{
				HostID:           *hostWithMDM.HostID,
				Enrolled:         *hostWithMDM.Enrolled,
				ServerURL:        *hostWithMDM.ServerURL,
				InstalledFromDep: *hostWithMDM.InstalledFromDep,
				IsServer:         *hostWithMDM.IsServer,
				MDMID:            hostWithMDM.MDMID,
				Name:             *hostWithMDM.Name,
			}
		}
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
      h.refetch_critical_queries_until,
      h.team_id,
      h.policy_updated_at,
      h.public_ip,
      COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
      COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available,
      hm.host_id,
      hm.enrolled,
      hm.server_url,
      hm.installed_from_dep,
      hm.mdm_id,
      COALESCE(hm.is_server, false) AS is_server,
      COALESCE(mdms.name, ?) AS name,
      IF(hdep.host_id AND ISNULL(hdep.deleted_at), true, false) AS dep_assigned_to_fleet
    FROM
      host_device_auth hda
    INNER JOIN
      hosts h
    ON
      hda.host_id = h.id
    LEFT OUTER JOIN
      host_disks hd ON hd.host_id = hda.host_id
    LEFT OUTER JOIN
      host_mdm hm  ON hm.host_id = h.id
    LEFT OUTER JOIN
      host_dep_assignments hdep ON hdep.host_id = h.id
    LEFT OUTER JOIN
      mobile_device_management_solutions mdms ON hm.mdm_id = mdms.id
    WHERE
      hda.token = ? AND
      hda.updated_at >= DATE_SUB(NOW(), INTERVAL ? SECOND)`

	var hostWithMDM hostWithMDMInfo
	switch err := ds.getContextTryStmt(ctx, &hostWithMDM, query, fleet.UnknownMDMName, authToken, tokenTTL.Seconds()); {
	case err == nil:
		host := hostWithMDM.Host
		// leave MDMInfo nil unless it has mdm information
		if hostWithMDM.HostID != nil {
			host.MDMInfo = &fleet.HostMDM{
				HostID:           *hostWithMDM.HostID,
				Enrolled:         *hostWithMDM.Enrolled,
				ServerURL:        *hostWithMDM.ServerURL,
				InstalledFromDep: *hostWithMDM.InstalledFromDep,
				IsServer:         *hostWithMDM.IsServer,
				MDMID:            hostWithMDM.MDMID,
				Name:             *hostWithMDM.Name,
			}
		}
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
	_, err := ds.writer(ctx).ExecContext(ctx, stmt, hostID, authToken)
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
    h.refetch_critical_queries_until,
    h.team_id,
    h.policy_updated_at,
    h.public_ip,
    h.orbit_node_key,
    COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
    COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available,
    COALESCE(hst.seen_time, h.created_at) AS seen_time,
	COALESCE(hu.software_updated_at, h.created_at) AS software_updated_at
	` + hostMDMSelect + `
  FROM hosts h
  LEFT JOIN host_seen_times hst ON (h.id = hst.host_id)
  LEFT JOIN host_updates hu ON (h.id = hu.host_id)
  LEFT JOIN host_disks hd ON hd.host_id = h.id
  ` + hostMDMJoin + `
  WHERE TRUE AND `

	matchingHostIDs := make([]int, 0)
	if len(matchQuery) > 0 {
		// first we'll find the hosts that match the search criteria, to keep thing simple, then we'll query again
		// to get all the additional data for hosts that match the search criteria by host_id
		matchingHosts := "SELECT id FROM hosts WHERE TRUE"
		var args []interface{}
		searchHostsQuery, args, matchesEmail := hostSearchLike(matchingHosts, args, matchQuery, hostSearchColumns...)
		// if matchQuery is "email like" then don't bother with the additional wildcard searching
		if !matchesEmail && len(matchQuery) > 2 && hasNonASCIIRegex(matchQuery) {
			union, wildCardArgs := hostSearchLikeAny(matchingHosts, args, matchQuery, wildCardableHostSearchColumns...)
			searchHostsQuery += " UNION " + union
			args = wildCardArgs
		}
		searchHostsQuery += " AND TRUE ORDER BY id DESC LIMIT 10"
		searchHostsQuery = ds.reader(ctx).Rebind(searchHostsQuery)
		err := sqlx.SelectContext(ctx, ds.reader(ctx), &matchingHostIDs, searchHostsQuery, args...)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "searching hosts")
		}
	}

	// we attempted to search for something that yielded no results, this should return empty set, no point in continuing
	if len(matchingHostIDs) == 0 && len(matchQuery) > 0 {
		return []*fleet.Host{}, nil
	}
	var args []interface{}
	if len(matchingHostIDs) > 0 {
		args = append(args, matchingHostIDs)
		query += " id IN (?) AND "
	}
	if len(omit) > 0 {
		args = append(args, omit)
		query += " id NOT IN (?) AND "
	}
	query += ds.whereFilterHostsByTeams(filter, "h")
	query += ` ORDER BY h.id DESC LIMIT 10`

	var err error
	if len(args) > 0 {
		query, args, err = sqlx.In(query, args...)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "searching default hosts")
		}
	}

	query = ds.reader(ctx).Rebind(query)
	var hosts []*fleet.Host
	if err = sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, query, args...); err != nil {
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
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostIDs, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host IDs")
	}

	return hostIDs, nil
}

func (ds *Datastore) ListHostsLiteByUUIDs(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
	if len(uuids) == 0 {
		return nil, nil
	}

	stmt := fmt.Sprintf(`
SELECT
	id,
	created_at,
	updated_at,
	osquery_host_id,
	node_key,
	hostname,
	uuid,
	hardware_serial,
	hardware_model,
	computer_name,
	platform,
	team_id,
	distributed_interval,
	logger_tls_period,
	config_tls_refresh,
	detail_updated_at,
	label_updated_at,
	last_enrolled_at,
	policy_updated_at,
	refetch_requested,
	refetch_critical_queries_until
FROM hosts
WHERE uuid IN (?) AND %s
		`, ds.whereFilterHostsByTeams(filter, "hosts"),
	)

	stmt, args, err := sqlx.In(stmt, uuids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query to select hosts by uuid")
	}

	var hosts []*fleet.Host
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select hosts by uuid")
	}

	return hosts, nil
}

func (ds *Datastore) ListHostsLiteByIDs(ctx context.Context, ids []uint) ([]*fleet.Host, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	stmt := `
SELECT
	id,
	created_at,
	updated_at,
	osquery_host_id,
	node_key,
	hostname,
	uuid,
	hardware_serial,
	hardware_model,
	computer_name,
	platform,
	team_id,
	distributed_interval,
	logger_tls_period,
	config_tls_refresh,
	detail_updated_at,
	label_updated_at,
	last_enrolled_at,
	policy_updated_at,
	refetch_requested,
	refetch_critical_queries_until
FROM hosts
WHERE id IN (?)`

	stmt, args, err := sqlx.In(stmt, ids)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query to select hosts by id")
	}

	var hosts []*fleet.Host
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &hosts, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select hosts by id")
	}

	return hosts, nil
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
      h.refetch_critical_queries_until,
      h.team_id,
      h.policy_updated_at,
      h.public_ip,
      h.orbit_node_key,
      COALESCE(hd.gigs_disk_space_available, 0) as gigs_disk_space_available,
      COALESCE(hd.percent_disk_space_available, 0) as percent_disk_space_available,
      COALESCE(hst.seen_time, h.created_at) AS seen_time,
	  COALESCE(hu.software_updated_at, h.created_at) AS software_updated_at
	  ` + hostMDMSelect + `
    FROM hosts h
    LEFT JOIN host_seen_times hst ON (h.id = hst.host_id)
	LEFT JOIN host_updates hu ON (h.id = hu.host_id)
    LEFT JOIN host_disks hd ON hd.host_id = h.id
	` + hostMDMJoin + `
    WHERE ? IN (h.hostname, h.osquery_host_id, h.node_key, h.uuid)
    LIMIT 1
	`
	host := &fleet.Host{}
	err := sqlx.GetContext(ctx, ds.reader(ctx), host, stmt, identifier)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Host").WithName(identifier))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host by identifier")
	}

	packStats, err := loadHostPackStatsDB(ctx, ds.reader(ctx), host.ID, host.Platform)
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

		if err := cleanupDiskEncryptionKeysOnTeamChangeDB(ctx, tx, hostIDs, teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "AddHostsToTeam cleanup disk encryption keys")
		}

		return nil
	})
}

func (ds *Datastore) SaveHostAdditional(ctx context.Context, hostID uint, additional *json.RawMessage) error {
	return saveHostAdditionalDB(ctx, ds.writer(ctx), hostID, additional)
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

	err = sqlx.GetContext(ctx, ds.reader(ctx), &counts,
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
		level.Error(ds.logger).Log("err", "unrecognized platform", "hostID", host.ID, "platform", host.Platform) //nolint:errcheck
	}

	query := `
		SELECT SUM(1 - pm.passes) AS n_failed
		FROM policy_membership pm
		WHERE pm.host_id = ? AND pm.passes IS NOT null
		GROUP BY host_id
	`

	var r uint
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &r, query, host.ID); err != nil {
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
		level.Error(ds.logger).Log("err", "unrecognized platform", "hostID", host.ID, "platform", host.Platform) //nolint:errcheck
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
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &policies, query, host.ID, host.ID, host.FleetPlatform()); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host policies")
	}
	return policies, nil
}

func (ds *Datastore) CleanupExpiredHosts(ctx context.Context) ([]uint, error) {
	ac, err := appConfigDB(ctx, ds.reader(ctx))
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
	err = ds.writer(ctx).SelectContext(
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

	_, err = ds.writer(ctx).ExecContext(ctx, `DELETE FROM host_seen_times WHERE seen_time < DATE_SUB(NOW(), INTERVAL ? DAY)`, ac.HostExpirySettings.HostExpiryWindow)
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
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &mappings, stmt, id)
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
	res, err := ds.writer(ctx).ExecContext(ctx, updateQuery, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update")
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "rows affected by update")
	}
	if affected == 0 {
		_, err = ds.writer(ctx).ExecContext(ctx, insertQuery, args...)
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
		_, err := ds.writer(ctx).ExecContext(ctx, updateQuery, hostID)
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
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &counts, stmt, args...); err != nil {
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
		if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
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
		if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
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
	if err := readIDs(ds.reader(ctx), allMsgs, "error"); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load error message IDs from reader")
	}
	if err := readIDs(ds.reader(ctx), allMsgs, "warning"); err != nil {
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
			if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "batch-insert munki issues")
			}
		}

		// load the IDs for the missing munki issues, from the primary as we just
		// inserted them
		if err := readIDs(ds.writer(ctx), msgsToReload, "error"); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "load error message IDs from writer")
		}
		if err := readIDs(ds.writer(ctx), msgsToReload, "warning"); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "load warning message IDs from writer")
		}
		if missing := missingIDs(); len(missing) > 0 {
			// some messages still have no IDs
			return nil, ctxerr.New(ctx, "found munki issues without id after batch-insert")
		}
	}

	return msgToID, nil
}

func (ds *Datastore) SetOrUpdateMDMData(
	ctx context.Context,
	hostID uint,
	isServer, enrolled bool,
	serverURL string,
	installedFromDep bool,
	name string,
) error {
	var mdmID *uint
	if serverURL != "" {
		id, err := ds.getOrInsertMDMSolution(ctx, serverURL, name)
		if err != nil {
			return err
		}
		mdmID = &id
	}

	return ds.updateOrInsert(
		ctx,
		`UPDATE host_mdm SET enrolled = ?, server_url = ?, installed_from_dep = ?, mdm_id = ?, is_server = ? WHERE host_id = ?`,
		`INSERT INTO host_mdm (enrolled, server_url, installed_from_dep, mdm_id, is_server, host_id) VALUES (?, ?, ?, ?, ?, ?)`,
		enrolled, serverURL, installedFromDep, mdmID, isServer, hostID,
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

// SetOrUpdateHostDisksEncryption sets the host's flag indicating if the disk
// encryption is enabled.
func (ds *Datastore) SetOrUpdateHostDisksEncryption(ctx context.Context, hostID uint, encrypted bool) error {
	return ds.updateOrInsert(
		ctx,
		`UPDATE host_disks SET encrypted = ? WHERE host_id = ?`,
		`INSERT INTO host_disks (encrypted, host_id) VALUES (?, ?)`,
		encrypted, hostID,
	)
}

func (ds *Datastore) SetOrUpdateHostDiskEncryptionKey(ctx context.Context, hostID uint, encryptedBase64Key string) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
           INSERT INTO host_disk_encryption_keys (host_id, base64_encrypted)
	   VALUES (?, ?)
	   ON DUPLICATE KEY UPDATE
   	     /* if the key has changed, NULLify this value so it can be calculated again */
             decryptable = IF(base64_encrypted = VALUES(base64_encrypted), decryptable, NULL),
   	     base64_encrypted = VALUES(base64_encrypted)
      `, hostID, encryptedBase64Key)
	return err
}

func (ds *Datastore) GetUnverifiedDiskEncryptionKeys(ctx context.Context) ([]fleet.HostDiskEncryptionKey, error) {
	var keys []fleet.HostDiskEncryptionKey
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &keys, `
          SELECT
            base64_encrypted,
            host_id,
            updated_at
          FROM
            host_disk_encryption_keys
          WHERE
            decryptable IS NULL
	`)
	return keys, err
}

func (ds *Datastore) SetHostsDiskEncryptionKeyStatus(
	ctx context.Context,
	hostIDs []uint,
	decryptable bool,
	threshold time.Time,
) error {
	if len(hostIDs) == 0 {
		return nil
	}

	query, args, err := sqlx.In(
		"UPDATE host_disk_encryption_keys SET decryptable = ? WHERE host_id IN (?) AND updated_at <= ?",
		decryptable, hostIDs, threshold,
	)
	if err != nil {
		return err
	}
	_, err = ds.writer(ctx).ExecContext(ctx, query, args...)
	return err
}

func (ds *Datastore) GetHostDiskEncryptionKey(ctx context.Context, hostID uint) (*fleet.HostDiskEncryptionKey, error) {
	var key fleet.HostDiskEncryptionKey
	err := sqlx.GetContext(ctx, ds.reader(ctx), &key, `
          SELECT
            host_id, base64_encrypted, decryptable, updated_at
          FROM
            host_disk_encryption_keys
          WHERE host_id = ?`, hostID)
	if err != nil {
		if err == sql.ErrNoRows {
			msg := fmt.Sprintf("for host %d", hostID)
			return nil, ctxerr.Wrap(ctx, notFound("HostDiskEncryptionKey").WithMessage(msg))
		}
		return nil, ctxerr.Wrapf(ctx, err, "getting data from host_disk_encryption_keys for host_id %d", hostID)
	}
	return &key, nil
}

func (ds *Datastore) SetOrUpdateHostOrbitInfo(ctx context.Context, hostID uint, version string) error {
	return ds.updateOrInsert(
		ctx,
		`UPDATE host_orbit_info SET version = ? WHERE host_id = ?`,
		`INSERT INTO host_orbit_info (version, host_id) VALUES (?, ?)`,
		version, hostID,
	)
}

func (ds *Datastore) getOrInsertMDMSolution(ctx context.Context, serverURL string, mdmName string) (mdmID uint, err error) {
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
	err := sqlx.GetContext(ctx, ds.reader(ctx), &version, `SELECT version FROM host_munki_info WHERE deleted_at is NULL AND host_id = ?`, hostID)
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
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hmdm, `
		SELECT
			hm.host_id, hm.enrolled, hm.server_url, hm.installed_from_dep, hm.mdm_id, COALESCE(hm.is_server, false) AS is_server, COALESCE(mdms.name, ?) AS name
		FROM
			host_mdm hm
		LEFT OUTER JOIN
			mobile_device_management_solutions mdms
		ON hm.mdm_id = mdms.id
		WHERE hm.host_id = ?`, fleet.UnknownMDMName, hostID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("HostMDMData").WithID(hostID))
		}
		return nil, ctxerr.Wrapf(ctx, err, "getting data from host_mdm for host_id %d", hostID)
	}
	return &hmdm, nil
}

func (ds *Datastore) GetHostMDMCheckinInfo(ctx context.Context, hostUUID string) (*fleet.HostMDMCheckinInfo, error) {
	var hmdm fleet.HostMDMCheckinInfo

	// use writer as it is used just after creation in some cases
	err := sqlx.GetContext(ctx, ds.writer(ctx), &hmdm, `
		SELECT
			h.hardware_serial,
			COALESCE(hm.installed_from_dep, false) as installed_from_dep,
			hd.display_name,
			COALESCE(h.team_id, 0) as team_id,
			hda.host_id IS NOT NULL AND hda.deleted_at IS NULL as dep_assigned_to_fleet
		FROM
			hosts h
		LEFT JOIN
			host_mdm hm
		ON h.id = hm.host_id
		LEFT JOIN
			host_display_names hd
		ON h.id = hd.host_id
		LEFT JOIN
			host_dep_assignments hda
		ON h.id = hda.host_id
		WHERE h.uuid = ? LIMIT 1`, hostUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Host").WithMessage(fmt.Sprintf("with UUID: %s", hostUUID)))
		}
		return nil, ctxerr.Wrapf(ctx, err, "host mdm checkin info for host UUID %s", hostUUID)
	}
	return &hmdm, nil
}

func (ds *Datastore) GetMDMSolution(ctx context.Context, mdmID uint) (*fleet.MDMSolution, error) {
	var solution fleet.MDMSolution
	err := sqlx.GetContext(ctx, ds.reader(ctx), &solution, `
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
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &issues, `
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
	err := sqlx.GetContext(ctx, ds.reader(ctx), &issue, `
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
	globalStats := true

	if teamID != nil {
		globalStats = false
		id = *teamID
	}
	var versions []fleet.AggregatedMunkiVersion
	var versionsJson struct {
		JsonValue []byte    `db:"json_value"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	err := sqlx.GetContext(
		ctx, ds.reader(ctx), &versionsJson,
		`SELECT json_value, updated_at FROM aggregated_stats WHERE id = ? AND global_stats = ? AND type = ?`,
		id, globalStats, aggregatedStatsTypeMunkiVersions,
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
	globalStats := true

	if teamID != nil {
		globalStats = false
		id = *teamID
	}

	var result []fleet.AggregatedMunkiIssue
	var resultJSON struct {
		JsonValue []byte    `db:"json_value"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	err := sqlx.GetContext(
		ctx, ds.reader(ctx), &resultJSON,
		`SELECT json_value, updated_at FROM aggregated_stats WHERE id = ? AND global_stats = ? AND type = ?`,
		id, globalStats, aggregatedStatsTypeMunkiIssues,
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

func (ds *Datastore) AggregatedMDMStatus(ctx context.Context, teamID *uint, platform string) (fleet.AggregatedMDMStatus, time.Time, error) {
	id := uint(0)
	globalStats := true

	if teamID != nil {
		globalStats = false
		id = *teamID
	}

	var status fleet.AggregatedMDMStatus
	var statusJson struct {
		JsonValue []byte    `db:"json_value"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	err := sqlx.GetContext(
		ctx, ds.reader(ctx), &statusJson,
		`select json_value, updated_at from aggregated_stats where id = ? and global_stats = ? and type = ?`,
		id, globalStats, platformKey(aggregatedStatsTypeMDMStatusPartial, platform),
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

func platformKey(key aggregatedStatsType, platform string) aggregatedStatsType {
	if platform == "" {
		return key
	}
	return key + "_" + aggregatedStatsType(platform)
}

func (ds *Datastore) AggregatedMDMSolutions(ctx context.Context, teamID *uint, platform string) ([]fleet.AggregatedMDMSolutions, time.Time, error) {
	id := uint(0)
	globalStats := true

	if teamID != nil {
		globalStats = false
		id = *teamID
	}

	var result []fleet.AggregatedMDMSolutions
	var resultJSON struct {
		JsonValue []byte    `db:"json_value"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	err := sqlx.GetContext(
		ctx, ds.reader(ctx), &resultJSON,
		`SELECT json_value, updated_at FROM aggregated_stats WHERE id = ? AND global_stats = ? AND type = ?`,
		id, globalStats, platformKey(aggregatedStatsTypeMDMSolutionsPartial, platform),
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
	var (
		platforms = []string{"", "darwin", "windows"}
		teamIDs   []uint
	)

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &teamIDs, `SELECT id FROM teams`); err != nil {
		return ctxerr.Wrap(ctx, err, "list teams")
	}

	// generate stats per team, append team id "0" to generate for "no team"
	teamIDs = append(teamIDs, 0)
	for _, teamID := range teamIDs {
		if err := ds.generateAggregatedMunkiVersion(ctx, &teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "generating aggregated munki version")
		}
		if err := ds.generateAggregatedMunkiIssues(ctx, &teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "generating aggregated munki issues")
		}
		for _, platform := range platforms {
			if err := ds.generateAggregatedMDMStatus(ctx, &teamID, platform); err != nil {
				return ctxerr.Wrap(ctx, err, "generating aggregated mdm status")
			}
			if err := ds.generateAggregatedMDMSolutions(ctx, &teamID, platform); err != nil {
				return ctxerr.Wrap(ctx, err, "generating aggregated mdm solutions")
			}
		}
	}

	// generate global stats, for "all teams"
	if err := ds.generateAggregatedMunkiVersion(ctx, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "generating aggregated munki version")
	}
	if err := ds.generateAggregatedMunkiIssues(ctx, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "generating aggregated munki issues")
	}
	for _, platform := range platforms {
		if err := ds.generateAggregatedMDMStatus(ctx, nil, platform); err != nil {
			return ctxerr.Wrap(ctx, err, "generating aggregated mdm status")
		}
		if err := ds.generateAggregatedMDMSolutions(ctx, nil, platform); err != nil {
			return ctxerr.Wrap(ctx, err, "generating aggregated mdm solutions")
		}
	}
	return nil
}

func (ds *Datastore) generateAggregatedMunkiVersion(ctx context.Context, teamID *uint) error {
	id := uint(0)
	globalStats := true

	var versions []fleet.AggregatedMunkiVersion
	query := `SELECT count(*) as hosts_count, hm.version FROM host_munki_info hm`
	args := []interface{}{}
	if teamID != nil {
		globalStats = false
		id = *teamID

		if *teamID > 0 {
			args = append(args, *teamID)
			query += ` JOIN hosts h ON (h.id = hm.host_id) WHERE h.team_id = ? AND `
		} else {
			query += ` JOIN hosts h ON (h.id = hm.host_id) WHERE h.team_id IS NULL AND `
		}
	} else {
		query += `  WHERE `
	}
	query += ` hm.deleted_at IS NULL GROUP BY hm.version`
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &versions, query, args...)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting aggregated data from host_munki")
	}
	versionsJson, err := json.Marshal(versions)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling stats")
	}

	_, err = ds.writer(ctx).ExecContext(ctx,
		`
INSERT INTO aggregated_stats (id, global_stats, type, json_value)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    json_value = VALUES(json_value),
    updated_at = CURRENT_TIMESTAMP
`,
		id, globalStats, aggregatedStatsTypeMunkiVersions, versionsJson,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "inserting stats for munki_versions id %d", id)
	}
	return nil
}

func (ds *Datastore) generateAggregatedMunkiIssues(ctx context.Context, teamID *uint) error {
	id := uint(0)
	globalStats := true

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
		globalStats = false
		id = *teamID

		if *teamID > 0 {
			args = append(args, *teamID)
			query += ` JOIN hosts h ON (h.id = hmi.host_id) WHERE h.team_id = ? `
		} else {
			query += ` JOIN hosts h ON (h.id = hmi.host_id) WHERE h.team_id IS NULL `
		}
	}
	query += `GROUP BY hmi.munki_issue_id, mi.name, mi.issue_type`

	err := sqlx.SelectContext(ctx, ds.reader(ctx), &issues, query, args...)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting aggregated data from host_munki_issues")
	}

	issuesJSON, err := json.Marshal(issues)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling stats")
	}

	_, err = ds.writer(ctx).ExecContext(ctx, `
INSERT INTO aggregated_stats (id, global_stats, type, json_value)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    json_value = VALUES(json_value),
    updated_at = CURRENT_TIMESTAMP
`, id, globalStats, aggregatedStatsTypeMunkiIssues, issuesJSON)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "inserting stats for munki_issues id %d", id)
	}
	return nil
}

func (ds *Datastore) generateAggregatedMDMStatus(ctx context.Context, teamID *uint, platform string) error {
	var (
		id          = uint(0)
		globalStats = true
		status      fleet.AggregatedMDMStatus
	)
	// NOTE: ds.UpdateHostTablesOnMDMUnenroll sets installed_from_dep = 0 so DEP hosts are not counted as pending after unenrollment
	query := `SELECT
				COUNT(DISTINCT host_id) as hosts_count,
				COALESCE(SUM(CASE WHEN NOT enrolled AND NOT installed_from_dep THEN 1 ELSE 0 END), 0) as unenrolled_hosts_count,
				COALESCE(SUM(CASE WHEN NOT enrolled AND installed_from_dep THEN 1 ELSE 0 END), 0) as pending_hosts_count,
				COALESCE(SUM(CASE WHEN enrolled AND installed_from_dep THEN 1 ELSE 0 END), 0) as enrolled_automated_hosts_count,
				COALESCE(SUM(CASE WHEN enrolled AND NOT installed_from_dep THEN 1 ELSE 0 END), 0) as enrolled_manual_hosts_count
			 FROM host_mdm hm
       	`
	args := []interface{}{}
	if teamID != nil || platform != "" {
		query += ` JOIN hosts h ON (h.id = hm.host_id) `
	}
	query += ` WHERE NOT COALESCE(hm.is_server, false) `
	if teamID != nil {
		globalStats = false
		id = *teamID

		if *teamID > 0 {
			args = append(args, *teamID)
			query += ` AND h.team_id = ? `
		} else {
			query += ` AND h.team_id IS NULL `
		}
	}
	if platform != "" {
		args = append(args, platform)
		query += " AND h.platform = ? "
	}
	err := sqlx.GetContext(ctx, ds.reader(ctx), &status, query, args...)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting aggregated data from host_mdm")
	}

	statusJson, err := json.Marshal(status)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling stats")
	}

	_, err = ds.writer(ctx).ExecContext(ctx,
		`
INSERT INTO aggregated_stats (id, global_stats, type, json_value)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    json_value = VALUES(json_value),
    updated_at = CURRENT_TIMESTAMP
`,
		id, globalStats, platformKey(aggregatedStatsTypeMDMStatusPartial, platform), statusJson,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "inserting stats for mdm_status id %d", id)
	}
	return nil
}

func (ds *Datastore) generateAggregatedMDMSolutions(ctx context.Context, teamID *uint, platform string) error {
	var (
		id          = uint(0)
		globalStats = true
		results     []fleet.AggregatedMDMSolutions
		whereAnd    = "WHERE"
	)
	query := `SELECT
				mdms.id,
				mdms.server_url,
				mdms.name,
				COUNT(DISTINCT hm.host_id) as hosts_count
			 FROM mobile_device_management_solutions mdms
			 INNER JOIN host_mdm hm
			 ON hm.mdm_id = mdms.id
			 AND NOT COALESCE(hm.is_server, false)
`
	args := []interface{}{}
	if teamID != nil || platform != "" {
		query += ` JOIN hosts h ON (h.id = hm.host_id) `
	}
	if teamID != nil {
		globalStats = false
		id = *teamID

		if *teamID > 0 {
			args = append(args, *teamID)
			query += ` WHERE h.team_id = ? `
		} else {
			query += ` WHERE h.team_id IS NULL `
		}
		whereAnd = "AND"
	}
	if platform != "" {
		args = append(args, platform)
		query += whereAnd + ` h.platform = ? `
	}
	query += ` GROUP BY id, server_url, name`
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, query, args...)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting aggregated data from host_mdm")
	}

	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling stats")
	}

	_, err = ds.writer(ctx).ExecContext(ctx,
		`
INSERT INTO aggregated_stats (id, global_stats, type, json_value)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    json_value = VALUES(json_value),
    updated_at = CURRENT_TIMESTAMP
`,
		id, globalStats, platformKey(aggregatedStatsTypeMDMSolutionsPartial, platform), resultsJSON,
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
		"hardware_serial",
		"hardware_model",
		"computer_name",
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
		"refetch_critical_queries_until",
	).Where(goqu.I("id").Eq(id)).ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sql build")
	}
	var host fleet.Host
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &host, query, args...); err != nil {
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
	_, err := ds.writer(ctx).ExecContext(ctx, sqlStatement,
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
	_, err := ds.writer(ctx).ExecContext(ctx, sqlStatement, value, id)
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
			orbit_node_key = ?,
			refetch_critical_queries_until = ?
		WHERE id = ?
	`
	_, err := ds.writer(ctx).ExecContext(ctx, sqlStatement,
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
		host.RefetchCriticalQueriesUntil,
		host.ID,
	)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "save host with id %d", host.ID)
	}
	_, err = ds.writer(ctx).ExecContext(ctx, `
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
    global_stats = ? AND
    type = ?
`

	var row struct {
		JSONValue *json.RawMessage `db:"json_value"`
		UpdatedAt time.Time        `db:"updated_at"`
	}

	id := uint(0)
	globalStats := true
	if teamID != nil {
		id = *teamID
		globalStats = false
	}

	err := sqlx.GetContext(ctx, ds.reader(ctx), &row, query, id, globalStats, aggregatedStatsTypeOSVersions)
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

// Aggregated stats for os versions are stored by team id with 0 representing
// no team or the all teams if global_stats is true.
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
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, selectStmt); err != nil {
		return ctxerr.Wrap(ctx, err, "update os versions")
	}

	// each team has a slice of stats with team host counts per os version, no
	// team are grouped under team id 0
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
		// push to team stats / no team
		if r.TeamID != nil {
			statsByTeamID[*r.TeamID] = append(statsByTeamID[*r.TeamID], os)
		} else {
			statsByTeamID[0] = append(statsByTeamID[0], os)
		}
	}

	// if an existing team has no hosts assigned, we still want to store empty stats
	var teamIDs []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &teamIDs, "SELECT id FROM teams"); err != nil {
		return ctxerr.Wrap(ctx, err, "update os versions")
	}
	for _, id := range teamIDs {
		if _, ok := statsByTeamID[id]; !ok {
			statsByTeamID[id] = []fleet.OSVersion{}
		}
	}
	// same for "no team"
	if _, ok := statsByTeamID[0]; !ok {
		statsByTeamID[0] = []fleet.OSVersion{}
	}

	// assemble values as arguments for insert statement
	args := make([]interface{}, 0, len(statsByTeamID)*4)
	for id, stats := range statsByTeamID {
		jsonValue, err := json.Marshal(stats)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal os version stats")
		}
		args = append(args, id, false, aggregatedStatsTypeOSVersions, jsonValue)
	}

	// add the global stats
	globalArray := make([]fleet.OSVersion, 0, len(globalStats))
	for _, os := range globalStats {
		globalArray = append(globalArray, os)
	}
	jsonValue, err := json.Marshal(globalArray)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshal global os version stats")
	}
	args = append(args, 0, true, aggregatedStatsTypeOSVersions, jsonValue)

	insertStmt := "INSERT INTO aggregated_stats (id, global_stats, type, json_value) VALUES "
	insertStmt += strings.TrimSuffix(strings.Repeat("(?,?,?,?),", len(statsByTeamID)+1), ",") // +1 due to global stats
	insertStmt += " ON DUPLICATE KEY UPDATE json_value = VALUES(json_value), updated_at = CURRENT_TIMESTAMP"

	if _, err := ds.writer(ctx).ExecContext(ctx, insertStmt, args...); err != nil {
		return ctxerr.Wrapf(ctx, err, "insert os versions into aggregated stats")
	}

	return nil
}

// EnrolledHostIDs returns the complete list of host IDs.
func (ds *Datastore) EnrolledHostIDs(ctx context.Context) ([]uint, error) {
	const stmt = `SELECT id FROM hosts`

	var ids []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &ids, stmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get enrolled host IDs")
	}
	return ids, nil
}

// CountEnrolledHosts returns the current number of enrolled hosts.
func (ds *Datastore) CountEnrolledHosts(ctx context.Context) (int, error) {
	const stmt = `SELECT count(*) FROM hosts`

	var count int
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &count, stmt); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count enrolled host")
	}
	return count, nil
}

func (ds *Datastore) HostIDsByOSID(
	ctx context.Context,
	osID uint,
	offset int,
	limit int,
) ([]uint, error) {
	var ids []uint

	stmt := dialect.From("host_operating_system").
		Select("host_id").
		Where(
			goqu.C("os_id").Eq(osID)).
		Order(goqu.I("host_id").Desc()).
		Offset(uint(offset)).
		Limit(uint(limit))

	sql, args, err := stmt.ToSQL()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host IDs")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &ids, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host IDs")
	}

	return ids, nil
}

// TODO Refactor this: We should be using the operating system type for this
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

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &ids, sql, args...); err != nil {
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
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &batteries, stmt, hid); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select host batteries")
	}
	return batteries, nil
}

func (ds *Datastore) SetDiskEncryptionResetStatus(ctx context.Context, hostID uint, status bool) error {
	const stmt = `
          INSERT INTO host_disk_encryption_keys (host_id, reset_requested, base64_encrypted)
            VALUES (?, ?, '')
          ON DUPLICATE KEY UPDATE
            reset_requested = VALUES(reset_requested)`

	_, err := ds.writer(ctx).ExecContext(ctx, stmt, hostID, status)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "upsert disk encryption reset status")
	}
	return nil
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

func amountHostsByOrbitVersionDB(ctx context.Context, db sqlx.QueryerContext) ([]fleet.HostsCountByOrbitVersion, error) {
	counts := make([]fleet.HostsCountByOrbitVersion, 0)

	const stmt = `
		SELECT version as orbit_version, count(*) as num_hosts
		FROM host_orbit_info
		GROUP BY version
  	`
	if err := sqlx.SelectContext(ctx, db, &counts, stmt); err != nil {
		return nil, err
	}

	return counts, nil
}

func amountHostsByOsqueryVersionDB(ctx context.Context, db sqlx.QueryerContext) ([]fleet.HostsCountByOsqueryVersion, error) {
	counts := make([]fleet.HostsCountByOsqueryVersion, 0)

	const stmt = `
		SELECT osquery_version, count(*) as num_hosts
		FROM hosts
		GROUP BY osquery_version
  	`
	if err := sqlx.SelectContext(ctx, db, &counts, stmt); err != nil {
		return nil, err
	}

	return counts, nil
}

func (ds *Datastore) GetMatchingHostSerials(ctx context.Context, serials []string) (map[string]*fleet.Host, error) {
	result := map[string]*fleet.Host{}
	if len(serials) == 0 {
		return result, nil
	}

	var args []interface{}
	for _, serial := range serials {
		args = append(args, serial)
	}
	stmt, args, err := sqlx.In("SELECT id, hardware_serial, team_id FROM hosts WHERE hardware_serial IN (?)", args)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building IN statement for matching hosts")
	}
	var matchingHosts []*fleet.Host
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &matchingHosts, stmt, args...); err != nil {
		return nil, err
	}

	for _, host := range matchingHosts {
		result[host.HardwareSerial] = host
	}

	return result, nil
}

func (ds *Datastore) NewHostScriptExecutionRequest(ctx context.Context, request *fleet.HostScriptRequestPayload) (*fleet.HostScriptResult, error) {
	const (
		insStmt = `INSERT INTO host_script_results (host_id, execution_id, script_contents, output) VALUES (?, ?, ?, '')`
		getStmt = `SELECT id, host_id, execution_id, script_contents, created_at FROM host_script_results WHERE id = ?`
	)

	execID := uuid.New().String()
	result, err := ds.writer(ctx).ExecContext(ctx, insStmt,
		request.HostID,
		execID,
		request.ScriptContents,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new host script execution request")
	}

	var script fleet.HostScriptResult
	id, _ := result.LastInsertId()
	if err := ds.writer(ctx).GetContext(ctx, &script, getStmt, id); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting the created host script result to return")
	}
	return &script, nil
}

func (ds *Datastore) SetHostScriptExecutionResult(ctx context.Context, result *fleet.HostScriptResultPayload) error {
	const updStmt = `
  UPDATE host_script_results SET
    output = ?,
    runtime = ?,
    exit_code = ?
  WHERE
    host_id = ? AND
    execution_id = ?`

	const maxOutputRuneLen = 10000
	output := result.Output
	if len(output) > utf8.UTFMax*maxOutputRuneLen {
		// truncate the bytes as we know the output is too long, no point
		// converting more bytes than needed to runes.
		output = output[len(output)-(utf8.UTFMax*maxOutputRuneLen):]
	}
	if utf8.RuneCountInString(output) > maxOutputRuneLen {
		outputRunes := []rune(output)
		output = string(outputRunes[len(outputRunes)-maxOutputRuneLen:])
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, updStmt,
		output,
		result.Runtime,
		result.ExitCode,
		result.HostID,
		result.ExecutionID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "update host script result")
	}
	return nil
}

func (ds *Datastore) ListPendingHostScriptExecutions(ctx context.Context, hostID uint, ignoreOlder time.Duration) ([]*fleet.HostScriptResult, error) {
	const listStmt = `
  SELECT
    id,
    host_id,
    execution_id,
    script_contents
  FROM
    host_script_results
  WHERE
    host_id = ? AND
    exit_code IS NULL AND
    created_at >= DATE_SUB(NOW(), INTERVAL ? SECOND)`

	var results []*fleet.HostScriptResult
	seconds := int(ignoreOlder.Seconds())
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, listStmt, hostID, seconds); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list pending host script results")
	}
	return results, nil
}

func (ds *Datastore) GetHostScriptExecutionResult(ctx context.Context, execID string) (*fleet.HostScriptResult, error) {
	const getStmt = `
  SELECT
    id,
    host_id,
    execution_id,
    script_contents,
    output,
    runtime,
    exit_code,
    created_at
  FROM
    host_script_results
  WHERE
    execution_id = ?`

	var result fleet.HostScriptResult
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &result, getStmt, execID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("HostScriptResult").WithName(execID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host script result")
	}
	return &result, nil
}
