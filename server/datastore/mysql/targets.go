package mysql

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) CountHostsInTargets(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
	// The logic in this function should remain synchronized with
	// host.Status and GenerateHostStatusStatistics - that is, the intervals associated
	// with each status must be the same.

	if len(targets.HostIDs) == 0 && len(targets.LabelIDs) == 0 && len(targets.TeamIDs) == 0 {
		// No need to query if no targets selected
		return fleet.TargetMetrics{}, nil
	}

	sql := fmt.Sprintf(`
		SELECT
			COUNT(*) total,
			COALESCE(SUM(CASE WHEN DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL 30 DAY) <= ? THEN 1 ELSE 0 END), 0) mia,
			COALESCE(SUM(CASE WHEN DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL LEAST(distributed_interval, config_tls_refresh) + %d SECOND) <= ? AND DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL 30 DAY) >= ? THEN 1 ELSE 0 END), 0) offline,
			COALESCE(SUM(CASE WHEN DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL LEAST(distributed_interval, config_tls_refresh) + %d SECOND) > ? THEN 1 ELSE 0 END), 0) online,
			COALESCE(SUM(CASE WHEN DATE_ADD(created_at, INTERVAL 1 DAY) >= ? THEN 1 ELSE 0 END), 0) new
		FROM hosts h
		LEFT JOIN host_seen_times hst ON (h.id=hst.host_id)
		WHERE (id IN (?) OR (id IN (SELECT DISTINCT host_id FROM label_membership WHERE label_id IN (?))) OR team_id IN (?)) AND %s
`, fleet.OnlineIntervalBuffer, fleet.OnlineIntervalBuffer, ds.whereFilterHostsByTeams(filter, "h"))

	// Using -1 in the ID slices for the IN clause allows us to include the
	// IN clause even if we have no IDs to use. -1 will not match the
	// auto-increment IDs, and will also allow us to use the same query in
	// all situations (no need to remove the clause when there are no values)
	queryLabelIDs := []int{-1}
	for _, id := range targets.LabelIDs {
		queryLabelIDs = append(queryLabelIDs, int(id))
	}
	queryHostIDs := []int{-1}
	for _, id := range targets.HostIDs {
		queryHostIDs = append(queryHostIDs, int(id))
	}
	queryTeamIDs := []int{-1}
	for _, id := range targets.TeamIDs {
		queryTeamIDs = append(queryTeamIDs, int(id))
	}

	query, args, err := sqlx.In(sql, now, now, now, now, now, queryHostIDs, queryLabelIDs, queryTeamIDs)
	if err != nil {
		return fleet.TargetMetrics{}, ctxerr.Wrap(ctx, err, "sqlx.In CountHostsInTargets")
	}

	res := fleet.TargetMetrics{}
	err = sqlx.GetContext(ctx, ds.reader, &res, query, args...)
	if err != nil {
		return fleet.TargetMetrics{}, ctxerr.Wrap(ctx, err, "sqlx.Get CountHostsInTargets")
	}

	return res, nil
}

func (ds *Datastore) HostIDsInTargets(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets) ([]uint, error) {
	if len(targets.HostIDs) == 0 && len(targets.LabelIDs) == 0 && len(targets.TeamIDs) == 0 {
		// No need to query if no targets selected
		return []uint{}, nil
	}

	sql := fmt.Sprintf(`
			SELECT DISTINCT id
			FROM hosts
			WHERE (id IN (?) OR (id IN (SELECT host_id FROM label_membership WHERE label_id IN (?))) OR team_id IN (?)) AND %s
			ORDER BY id ASC
		`,
		ds.whereFilterHostsByTeams(filter, "hosts"),
	)

	// Using -1 in the ID slices for the IN clause allows us to include the
	// IN clause even if we have no IDs to use. -1 will not match the
	// auto-increment IDs, and will also allow us to use the same query in
	// all situations (no need to remove the clause when there are no values)
	queryLabelIDs := []int{-1}
	for _, id := range targets.LabelIDs {
		queryLabelIDs = append(queryLabelIDs, int(id))
	}
	queryHostIDs := []int{-1}
	for _, id := range targets.HostIDs {
		queryHostIDs = append(queryHostIDs, int(id))
	}
	queryTeamIDs := []int{-1}
	for _, id := range targets.TeamIDs {
		queryTeamIDs = append(queryTeamIDs, int(id))
	}

	query, args, err := sqlx.In(sql, queryHostIDs, queryLabelIDs, queryTeamIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sqlx.In HostIDsInTargets")
	}

	var res []uint
	err = sqlx.SelectContext(ctx, ds.reader, &res, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sqlx.Get HostIDsInTargets")
	}
	return res, nil
}
