// Package mysql provides the MySQL datastore implementation for the activity bounded context.
package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// tracer is an OTEL tracer. It has no-op behavior when OTEL is not enabled.
var tracer = otel.Tracer("github.com/fleetdm/fleet/v4/server/activity/internal/mysql")

// Datastore is the MySQL implementation of the activity datastore.
type Datastore struct {
	primary *sqlx.DB
	replica *sqlx.DB
	logger  kitlog.Logger
}

// NewDatastore creates a new MySQL datastore for activities.
func NewDatastore(conns *platform_mysql.DBConnections, logger kitlog.Logger) *Datastore {
	return &Datastore{primary: conns.Primary, replica: conns.Replica, logger: logger}
}

func (ds *Datastore) reader(ctx context.Context) *sqlx.DB {
	return ds.replica
}

// Ensure Datastore implements types.Datastore
var _ types.Datastore = (*Datastore)(nil)

// ListActivities returns a slice of activities performed across the organization.
func (ds *Datastore) ListActivities(ctx context.Context, opt types.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	ctx, span := tracer.Start(ctx, "activity.mysql.ListActivities",
		trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	activitiesQ := `
		SELECT
			a.id,
			a.user_id,
			a.created_at,
			a.activity_type,
			a.user_name as name,
			a.streamed,
			a.user_email,
			a.fleet_initiated
		FROM activities a
		WHERE a.host_only = false`

	var args []any

	if opt.Streamed != nil {
		activitiesQ += " AND a.streamed = ?"
		args = append(args, *opt.Streamed)
	}

	if opt.ActivityType != "" {
		activitiesQ += " AND a.activity_type = ?"
		args = append(args, opt.ActivityType)
	}

	// MatchQuery: search by user_name/user_email in activity table, plus user IDs from users table search
	if opt.MatchQuery != "" {
		activitiesQ += " AND (a.user_name LIKE ? OR a.user_email LIKE ?"
		args = append(args, opt.MatchQuery+"%", opt.MatchQuery+"%")

		// Add user IDs from users table search (populated by service via ACL)
		if len(opt.MatchingUserIDs) > 0 {
			inQ, inArgs, err := sqlx.In(" OR a.user_id IN (?)", opt.MatchingUserIDs)
			if err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "bind user IDs for IN clause")
			}
			activitiesQ += inQ
			args = append(args, inArgs...)
		}

		activitiesQ += ")"
	}

	if opt.StartCreatedAt != "" {
		activitiesQ += " AND a.created_at >= ?"
		args = append(args, opt.StartCreatedAt)
	}

	if opt.EndCreatedAt != "" {
		activitiesQ += " AND a.created_at <= ?"
		args = append(args, opt.EndCreatedAt)
	} else if opt.StartCreatedAt != "" {
		// When filtering by start date, cap at now to ensure consistent results
		activitiesQ += " AND a.created_at <= ?"
		args = append(args, time.Now().UTC())
	}

	// Apply pagination using platform_mysql
	activitiesQ, args = platform_mysql.AppendListOptionsWithParams(activitiesQ, args, &opt)

	var activities []*api.Activity
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &activities, activitiesQ, args...)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select activities")
	}

	// Fetch details as a separate query (due to MySQL sort buffer issues with large JSON)
	if len(activities) > 0 {
		if err := ds.fetchActivityDetails(ctx, activities); err != nil {
			return nil, nil, err
		}
	}

	// Build pagination metadata
	var meta *api.PaginationMetadata
	if opt.IncludeMetadata {
		meta = &api.PaginationMetadata{
			HasPreviousResults: opt.Page > 0,
		}
		if uint(len(activities)) > opt.PerPage && opt.PerPage > 0 {
			meta.HasNextResults = true
			activities = activities[:len(activities)-1]
		}
	}

	return activities, meta, nil
}

// fetchActivityDetails fetches details for activities in a separate query
// to avoid MySQL sort buffer issues with large JSON entries.
func (ds *Datastore) fetchActivityDetails(ctx context.Context, activities []*api.Activity) error {
	ids := make([]uint, 0, len(activities))
	for _, a := range activities {
		ids = append(ids, a.ID)
	}

	detailsStmt, detailsArgs, err := sqlx.In("SELECT id, details FROM activities WHERE id IN (?)", ids)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bind activity IDs for details")
	}

	type activityDetails struct {
		ID      uint             `db:"id"`
		Details *json.RawMessage `db:"details"`
	}
	var details []activityDetails

	err = sqlx.SelectContext(ctx, ds.reader(ctx), &details, detailsStmt, detailsArgs...)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ctxerr.Wrap(ctx, err, "select activity details")
	}

	detailsLookup := make(map[uint]*json.RawMessage, len(details))
	for _, d := range details {
		detailsLookup[d.ID] = d.Details
	}

	for _, a := range activities {
		det, ok := detailsLookup[a.ID]
		if !ok {
			level.Warn(ds.logger).Log("msg", "Activity details not found", "activity_id", a.ID)
			continue
		}
		a.Details = det
	}

	return nil
}
