package mysql

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	//
	// Each row is inserted on its own because the ids of a multi-row insert are not guaranteed to be consecutive, and the host links need them.
	for start := 0; start < len(activities); start += newActivitiesChunkSize {
		end := min(start+newActivitiesChunkSize, len(activities))
		err := platform_mysql.WithRetryTxx(ctx, ds.primary, func(tx sqlx.ExtContext) error {
			var links []activityHostLink
			for i := start; i < end; i++ {
				actID, err := insertActivityRow(ctx, tx, user, activities[i], details[i], createdAt)
				if err != nil {
					return err
				}
				if ah, ok := activities[i].(types.ActivityHosts); ok {
					links = append(links, activityHostLink{activityID: actID, hostIDs: ah.HostIDs()})
				}
			}
			return insertActivityHosts(ctx, tx, links)
		}, ds.logger)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "new activities: %d of %d saved before failing chunk", start, len(activities))
		}
	}
	return nil
}

type activityHostLink struct {
	activityID int64
	hostIDs    []uint
}

func insertActivityRow(
	ctx context.Context, tx sqlx.ExtContext, user *api.User, activity api.ActivityDetails, details []byte, createdAt time.Time,
) (int64, error) {
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
