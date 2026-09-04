package mysql

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// ListNanoEnrollmentIDsForAPNsSweep walks enabled nano_enrollments in primary
// key order, one bounded page per call. The inner page bounds the scan; the
// EXISTS marks eligibility without multiplying page rows (hosts.uuid is not
// unique — cloned VMs — so a join would corrupt the page length that pageFull
// is computed from; when duplicates disagree on MDM state, any enrolled one
// counts, which errs on the side of a spurious idempotent nudge). Both device
// and user channels are walked: user-channel rows carry device_id too, so
// they resolve to their host the same way.
func (ds *Datastore) ListNanoEnrollmentIDsForAPNsSweep(ctx context.Context, afterID string, batchSize int, silentFor time.Duration) ([]string, string, bool, error) {
	stmt := `
SELECT
    page.id,
    (page.last_seen_at < DATE_SUB(NOW(), INTERVAL ? SECOND)
     AND EXISTS (
         SELECT 1
         FROM hosts h
         JOIN host_mdm hmdm ON hmdm.host_id = h.id AND hmdm.enrolled = 1
         WHERE h.uuid = page.device_id
     )) AS eligible
FROM (
    SELECT ne.id, ne.device_id, ne.last_seen_at
    FROM nano_enrollments ne
    WHERE ne.enabled = 1 AND ne.id > ?
    ORDER BY ne.id
    LIMIT ?
) page
ORDER BY page.id`

	var rows []struct {
		ID       string `db:"id"`
		Eligible bool   `db:"eligible"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, int(silentFor.Seconds()), afterID, batchSize); err != nil {
		return nil, "", false, ctxerr.Wrap(ctx, err, "list nano enrollment ids for apns sweep")
	}

	var eligible []string
	for _, row := range rows {
		if row.Eligible {
			eligible = append(eligible, row.ID)
		}
	}
	var nextCursor string
	if len(rows) > 0 {
		nextCursor = rows[len(rows)-1].ID
	}
	return eligible, nextCursor, len(rows) >= batchSize, nil
}

// CountEnabledNanoEnrollments returns the number of nano_enrollments rows
// with enabled = 1.
func (ds *Datastore) CountEnabledNanoEnrollments(ctx context.Context) (int, error) {
	var count int
	err := sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(*) FROM nano_enrollments WHERE enabled = 1`)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count enabled nano enrollments")
	}
	return count, nil
}

// GetMDMAppleAPNsSweepState returns the APNs sweep cron's persisted pass
// state. The bare mysql.Datastore has no place to persist it, so this returns
// nil (fresh pass every tick). The mysqlredis wrapper overrides this to back
// it with Redis.
func (ds *Datastore) GetMDMAppleAPNsSweepState(_ context.Context) (*fleet.MDMAppleAPNsSweepState, error) {
	return nil, nil
}

// SetMDMAppleAPNsSweepState persists the APNs sweep cron's pass state. The
// bare mysql.Datastore is a no-op; the mysqlredis wrapper backs it with
// Redis.
func (ds *Datastore) SetMDMAppleAPNsSweepState(_ context.Context, _ *fleet.MDMAppleAPNsSweepState) error {
	return nil
}
