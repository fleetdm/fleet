package mysql

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (d *Datastore) CountHostsInTargets(hostIDs []uint, labelIDs []uint, now time.Time) (kolide.TargetMetrics, error) {
	// The logic in this function should remain synchronized with
	// host.Status and GenerateHostStatusStatistics

	if len(hostIDs) == 0 && len(labelIDs) == 0 {
		// No need to query if no targets selected
		return kolide.TargetMetrics{}, nil
	}

	sql := fmt.Sprintf(`
		SELECT
			COUNT(*) total,
			COALESCE(SUM(CASE WHEN DATE_ADD(seen_time, INTERVAL 30 DAY) <= ? THEN 1 ELSE 0 END), 0) mia,
			COALESCE(SUM(CASE WHEN DATE_ADD(seen_time, INTERVAL LEAST(distributed_interval, config_tls_refresh) + %d SECOND) <= ? AND DATE_ADD(seen_time, INTERVAL 30 DAY) >= ? THEN 1 ELSE 0 END), 0) offline,
			COALESCE(SUM(CASE WHEN DATE_ADD(seen_time, INTERVAL LEAST(distributed_interval, config_tls_refresh) + %d SECOND) > ? THEN 1 ELSE 0 END), 0) online,
			COALESCE(SUM(CASE WHEN DATE_ADD(created_at, INTERVAL 1 DAY) >= ? THEN 1 ELSE 0 END), 0) new
		FROM hosts h
		WHERE (id IN (?) OR (id IN (SELECT DISTINCT host_id FROM label_query_executions WHERE label_id IN (?) AND matches = 1)))
		AND NOT deleted
`, kolide.OnlineIntervalBuffer, kolide.OnlineIntervalBuffer)

	// Using -1 in the ID slices for the IN clause allows us to include the
	// IN clause even if we have no IDs to use. -1 will not match the
	// auto-increment IDs, and will also allow us to use the same query in
	// all situations (no need to remove the clause when there are no values)
	queryLabelIDs := []int{-1}
	for _, id := range labelIDs {
		queryLabelIDs = append(queryLabelIDs, int(id))
	}
	queryHostIDs := []int{-1}
	for _, id := range hostIDs {
		queryHostIDs = append(queryHostIDs, int(id))
	}

	query, args, err := sqlx.In(sql, now, now, now, now, now, queryHostIDs, queryLabelIDs)
	if err != nil {
		return kolide.TargetMetrics{}, errors.Wrap(err, "sqlx.In CountHostsInTargets")
	}

	res := kolide.TargetMetrics{}
	err = d.db.Get(&res, query, args...)
	if err != nil {
		return kolide.TargetMetrics{}, errors.Wrap(err, "sqlx.Get CountHostsInTargets")
	}

	return res, nil
}
