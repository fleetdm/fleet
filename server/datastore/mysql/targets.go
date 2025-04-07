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

	queryTargetLogicCondition, queryTargetArgs := targetSQLCondAndArgs(targets)

	// As of Fleet 4.15, mia hosts are also included in the total for offline hosts
	sql := fmt.Sprintf(`
		SELECT
			COUNT(*) total,
			COALESCE(SUM(CASE WHEN DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL 30 DAY) <= ? THEN 1 ELSE 0 END), 0) mia,
			COALESCE(SUM(CASE WHEN DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL LEAST(distributed_interval, config_tls_refresh) + %d SECOND) <= ? THEN 1 ELSE 0 END), 0) offline,
			COALESCE(SUM(CASE WHEN DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL LEAST(distributed_interval, config_tls_refresh) + %d SECOND) > ? THEN 1 ELSE 0 END), 0) online,
			COALESCE(SUM(CASE WHEN DATE_ADD(created_at, INTERVAL 1 DAY) >= ? THEN 1 ELSE 0 END), 0) new
		FROM hosts h
		LEFT JOIN host_seen_times hst ON (h.id=hst.host_id)
		WHERE %s AND %s`,
		fleet.OnlineIntervalBuffer, fleet.OnlineIntervalBuffer,
		queryTargetLogicCondition, ds.whereFilterHostsByTeams(filter, "h"),
	)

	query, args, err := sqlx.In(sql, append([]interface{}{now, now, now, now}, queryTargetArgs...)...)
	if err != nil {
		return fleet.TargetMetrics{}, ctxerr.Wrap(ctx, err, "sqlx.In CountHostsInTargets")
	}

	res := fleet.TargetMetrics{}
	err = sqlx.GetContext(ctx, ds.reader(ctx), &res, query, args...)
	if err != nil {
		return fleet.TargetMetrics{}, ctxerr.Wrap(ctx, err, "sqlx.Get CountHostsInTargets")
	}

	return res, nil
}

// targetSQLCondAndArgs returns the SQL condition and the arguments for matching whether
// a host ID is a target of a live query.
func targetSQLCondAndArgs(targets fleet.HostTargets) (sql string, args []interface{}) {
	const queryTargetLogicCondition = `(
	/* The host was selected explicitly. */
	id IN (? /* queryHostIDs */)
	OR
	(
		/* 'All hosts' builtin label was selected. */
		id IN (SELECT DISTINCT host_id FROM label_membership WHERE label_id = (SELECT id from labels WHERE name = 'All Hosts') AND label_id IN (? /* queryLabelIDs */))
	)
	OR
	(
		/* A team filter OR a label filter was specified. */
		(? /* labelsSpecified */ OR ? /* teamsSpecified */ )
		AND
		/* A non-builtin label (aka platform) filter was not specified OR if it was specified then the host must be
		 * a member of one of the specified non-builtin labels. */
		(
			SELECT NOT EXISTS (SELECT id FROM labels WHERE label_type <> 1 AND id IN (? /* queryLabelIDs */))
			OR
			(id IN (SELECT DISTINCT host_id FROM label_membership lm JOIN labels l ON lm.label_id = l.id WHERE l.label_type <> 1 AND lm.label_id IN (? /* queryLabelIDs */)))
		)
		AND
		/* A builtin label filter was not specified OR if it was specified then the host must be
		 * a member of one of the specified builtin labels. */
		(
			SELECT NOT EXISTS (SELECT id FROM labels WHERE label_type = 1 AND id IN (? /* queryLabelIDs */))
			OR
			(id IN (SELECT DISTINCT host_id FROM label_membership lm JOIN labels l ON lm.label_id = l.id WHERE l.label_type = 1 AND lm.label_id IN (? /* queryLabelIDs */)))
		)
		AND
		/* A team filter was not specified OR if it was specified then the host must be a
		 * member of one of the teams. */
		(? /* !teamsSpecified */ OR team_id IN (? /* queryTeamIDs */) %s)
	)
)`

	// Using -1 in the ID slices for the IN clause allows us to include the
	// IN clause even if we have no IDs to use. -1 will not match the
	// auto-increment IDs, and will also allow us to use the same query in
	// all situations (no need to remove the clause when there are no values)
	queryLabelIDs := []int{-1}
	for _, id := range targets.LabelIDs {
		queryLabelIDs = append(queryLabelIDs, int(id)) //nolint:gosec // dismiss G115
	}
	queryHostIDs := []int{-1}
	for _, id := range targets.HostIDs {
		queryHostIDs = append(queryHostIDs, int(id)) //nolint:gosec // dismiss G115
	}
	queryTeamIDs := []int{-1}
	extraTeamIDCondition := ""
	for _, id := range targets.TeamIDs {
		if id == 0 {
			extraTeamIDCondition = "OR team_id IS NULL"
			continue
		}
		queryTeamIDs = append(queryTeamIDs, int(id)) //nolint:gosec // dismiss G115
	}

	labelsSpecified := len(queryLabelIDs) > 1
	teamsSpecified := len(queryTeamIDs) > 1 || extraTeamIDCondition != ""

	return fmt.Sprintf(queryTargetLogicCondition, extraTeamIDCondition), []interface{}{
		queryHostIDs,
		queryLabelIDs,
		labelsSpecified, teamsSpecified,
		queryLabelIDs, queryLabelIDs,
		queryLabelIDs, queryLabelIDs,
		!teamsSpecified, queryTeamIDs,
	}
}

func (ds *Datastore) HostIDsInTargets(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets) ([]uint, error) {
	if len(targets.HostIDs) == 0 && len(targets.LabelIDs) == 0 && len(targets.TeamIDs) == 0 {
		// No need to query if no targets selected
		return []uint{}, nil
	}

	queryTargetLogicCondition, queryTargetArgs := targetSQLCondAndArgs(targets)

	sql := fmt.Sprintf(`
			SELECT DISTINCT id
			FROM hosts
			WHERE %s AND %s
			ORDER BY id ASC
		`,
		queryTargetLogicCondition,
		ds.whereFilterHostsByTeams(filter, "hosts"),
	)

	query, args, err := sqlx.In(sql, queryTargetArgs...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sqlx.In HostIDsInTargets")
	}

	var res []uint
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &res, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sqlx.Get HostIDsInTargets")
	}
	return res, nil
}
