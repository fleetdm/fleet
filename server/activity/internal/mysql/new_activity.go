package mysql

import (
	"context"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// NewActivity stores an activity record in the database.
// The webhook context key must be set in the context before calling this method.
func (ds *Datastore) NewActivity(
	ctx context.Context, user *api.User, activity api.ActivityDetails, details []byte, createdAt time.Time,
) error {
	ctx, span := tracer.Start(ctx, "activity.mysql.NewActivity")
	defer span.End()

	// Sanity check to ensure we processed activity webhook before storing the activity
	processed, _ := ctx.Value(types.ActivityWebhookContextKey).(bool)
	if !processed {
		return ctxerr.New(
			ctx, "activity webhook not processed. Please use svc.NewActivity instead of ds.NewActivity. This is a Fleet server bug.",
		)
	}

	return platform_mysql.WithRetryTxx(ctx, ds.primary, func(tx sqlx.ExtContext) error {
		actID, err := insertActivityRow(ctx, tx, user, activity, details, createdAt)
		if err != nil {
			return err
		}

		// Insert into activity_host_past table if the activity is associated with hosts.
		if ah, ok := activity.(types.ActivityHosts); ok {
			return insertActivityHosts(ctx, tx, []activityHostLink{{activityID: actID, hostIDs: ah.HostIDs()}})
		}
		return nil
	}, ds.logger)
}

// newActivitiesChunkSize bounds the activities written per transaction by NewActivities.
const newActivitiesChunkSize = 1000

// NewActivities stores one activity record per item of activities.
// The webhook context key must be set in the context before calling this method.
func (ds *Datastore) NewActivities(
	ctx context.Context, user *api.User, activities []api.ActivityDetails, details [][]byte, createdAt time.Time,
) error {
	ctx, span := tracer.Start(ctx, "activity.mysql.NewActivities")
	defer span.End()

	processed, _ := ctx.Value(types.ActivityWebhookContextKey).(bool)
	if !processed {
		return ctxerr.New(
			ctx, "activity webhook not processed. Please use svc.NewActivities instead of ds.NewActivities. This is a Fleet server bug.",
		)
	}
	if len(activities) != len(details) {
		return ctxerr.New(ctx, "activities and details must have the same length")
	}

	// The batch is not atomic: each chunk commits on its own, and the webhooks have already been sent by the service. A failure
	// reports how many activities were saved before it.
	for start := 0; start < len(activities); start += newActivitiesChunkSize {
		end := min(start+newActivitiesChunkSize, len(activities))
		err := platform_mysql.WithRetryTxx(ctx, ds.primary, func(tx sqlx.ExtContext) error {
			return insertActivitiesChunk(ctx, tx, user, activities[start:end], details[start:end], createdAt)
		}, ds.logger)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "new activities: %d of %d saved before failing chunk", start, len(activities))
		}
	}
	return nil
}

// insertActivitiesChunk inserts the activities with one multi-row statement and links them to their hosts.
//
// Each row carries a unique batch_row_uuid and the ids are read back by it. The read is a primary key range scan starting after
// the max id seen before the insert, which is below every id this transaction allocates and keeps the unindexed batch_row_uuid
// column out of a table scan.
func insertActivitiesChunk(
	ctx context.Context, tx sqlx.ExtContext, user *api.User, activities []api.ActivityDetails, details [][]byte, createdAt time.Time,
) error {
	var maxIDBefore int64
	if err := sqlx.GetContext(ctx, tx, &maxIDBefore, `SELECT COALESCE(MAX(id), 0) FROM activity_past`); err != nil {
		return ctxerr.Wrap(ctx, err, "get max activity id")
	}

	var cols []string
	rowUUIDs := make([]uuid.UUID, len(activities))
	var args []any
	for i, activity := range activities {
		rowCols, rowArgs := activityRow(user, activity, details[i], createdAt)
		cols = rowCols // the columns only depend on the user, which is shared by the batch
		rowUUIDs[i] = uuid.NewV7()
		args = append(args, rowArgs...)
		args = append(args, rowUUIDs[i][:])
	}
	cols = append(cols, "batch_row_uuid")
	rowPlaceholders := "(" + strings.Repeat("?,", len(cols)-1) + "?)"
	stmt := fmt.Sprintf(`INSERT INTO activity_past (%s) VALUES %s`,
		strings.Join(cols, ","), strings.TrimSuffix(strings.Repeat(rowPlaceholders+",", len(activities)), ","))
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "new activities")
	}

	var inserted []struct {
		ID   int64  `db:"id"`
		UUID []byte `db:"batch_row_uuid"`
	}
	if err := sqlx.SelectContext(ctx, tx, &inserted,
		`SELECT id, batch_row_uuid FROM activity_past WHERE id > ? AND batch_row_uuid IS NOT NULL`, maxIDBefore); err != nil {
		return ctxerr.Wrap(ctx, err, "read back new activity ids")
	}
	idsByUUID := make(map[uuid.UUID]int64, len(inserted))
	for _, row := range inserted {
		if len(row.UUID) != len(uuid.UUID{}) {
			return ctxerr.Errorf(ctx, "unexpected batch_row_uuid length %d", len(row.UUID))
		}
		idsByUUID[uuid.UUID(row.UUID)] = row.ID
	}

	var links []activityHostLink
	for i, activity := range activities {
		ah, ok := activity.(types.ActivityHosts)
		if !ok {
			continue
		}
		actID, found := idsByUUID[rowUUIDs[i]]
		if !found {
			return ctxerr.New(ctx, "new activity id not found after insert")
		}
		links = append(links, activityHostLink{activityID: actID, hostIDs: ah.HostIDs()})
	}
	return insertActivityHosts(ctx, tx, links)
}

type activityHostLink struct {
	activityID int64
	hostIDs    []uint
}

// activityRow returns the activity_past columns and values to insert for an activity.
func activityRow(user *api.User, activity api.ActivityDetails, details []byte, createdAt time.Time) ([]string, []any) {
	var userID *uint
	var userName *string
	var userEmail *string
	var fleetInitiated bool
	var hostOnly bool

	if user != nil {
		// To support creating activities with users that were deleted. This can happen
		// for automatically installed software which uses the author of the upload as the author of
		// the installation.
		if user.ID != 0 && !user.Deleted {
			userID = &user.ID
		}
		userName = &user.Name
		userEmail = &user.Email
	}

	if automatableActivity, ok := activity.(types.AutomatableActivity); ok && automatableActivity.WasFromAutomation() {
		automationAuthor := types.ActivityAutomationAuthor
		userName = &automationAuthor
		fleetInitiated = true
	}

	if hostOnlyActivity, ok := activity.(types.ActivityHostOnly); ok && hostOnlyActivity.HostOnly() {
		hostOnly = true
	}

	cols := []string{"fleet_initiated", "user_id", "user_name", "activity_type", "details", "created_at", "host_only"}
	args := []any{
		fleetInitiated,
		userID,
		userName,
		activity.ActivityName(),
		details,
		createdAt,
		hostOnly,
	}
	// For system/automated activities (user == nil), user_email defaults to empty (not null).
	if userEmail != nil {
		args = append(args, userEmail)
		cols = append(cols, "user_email")
	}
	return cols, args
}

func insertActivityRow(
	ctx context.Context, tx sqlx.ExtContext, user *api.User, activity api.ActivityDetails, details []byte, createdAt time.Time,
) (int64, error) {
	cols, args := activityRow(user, activity, details, createdAt)
	const insertActStmt = `INSERT INTO activity_past (%s) VALUES (%s)`
	sqlStmt := fmt.Sprintf(insertActStmt, strings.Join(cols, ","), strings.Repeat("?,", len(cols)-1)+"?")
	res, err := tx.ExecContext(ctx, sqlStmt, args...)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "new activity")
	}
	actID, _ := res.LastInsertId()
	return actID, nil
}

func insertActivityHosts(ctx context.Context, tx sqlx.ExtContext, links []activityHostLink) error {
	const insertActHostStmt = `INSERT INTO activity_host_past (host_id, activity_id) VALUES `

	var sb strings.Builder
	for _, link := range links {
		for _, hid := range link.hostIDs {
			sb.WriteString(fmt.Sprintf("(%d, %d),", hid, link.activityID))
		}
	}
	if sb.Len() == 0 {
		return nil
	}
	stmt := insertActHostStmt + strings.TrimSuffix(sb.String(), ",")
	if _, err := tx.ExecContext(ctx, stmt); err != nil {
		return ctxerr.Wrap(ctx, err, "insert host activity")
	}
	return nil
}
