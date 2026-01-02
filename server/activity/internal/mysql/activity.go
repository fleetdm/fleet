// Package mysql provides the MySQL datastore implementation for the activity bounded context.
package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/jmoiron/sqlx"
)

// Datastore is the MySQL implementation of the activity datastore.
type Datastore struct {
	primary *sqlx.DB
	replica *sqlx.DB
}

// NewDatastore creates a new MySQL datastore for activities.
func NewDatastore(primary, replica *sqlx.DB) *Datastore {
	return &Datastore{primary: primary, replica: replica}
}

func (ds *Datastore) reader(ctx context.Context) *sqlx.DB {
	return ds.replica
}

// Ensure Datastore implements types.Datastore
var _ types.Datastore = (*Datastore)(nil)

// ListActivities returns a slice of activities performed across the organization.
func (ds *Datastore) ListActivities(ctx context.Context, opt types.ListOptions) ([]*api.Activity, *api.PaginationMetadata, error) {
	activities := []*api.Activity{}

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
	// This matches the legacy behavior in server/datastore/mysql/activities.go ListActivities
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

	if opt.StartCreatedAt != "" || opt.EndCreatedAt != "" {
		start := opt.StartCreatedAt
		end := opt.EndCreatedAt
		switch {
		case start == "" && end != "":
			activitiesQ += " AND a.created_at <= ?"
			args = append(args, end)
		case start != "" && end == "":
			activitiesQ += " AND a.created_at >= ? AND a.created_at <= ?"
			args = append(args, start, time.Now().UTC())
		case start != "" && end != "":
			activitiesQ += " AND a.created_at >= ? AND a.created_at <= ?"
			args = append(args, start, end)
		}
	}

	// Apply pagination using common_mysql
	activitiesQ, args = common_mysql.AppendListOptionsWithParams(activitiesQ, args, &listOptsAdapter{opt: opt})

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
	if err != nil && err != sql.ErrNoRows {
		return ctxerr.Wrap(ctx, err, "select activity details")
	}

	detailsLookup := make(map[uint]*json.RawMessage, len(details))
	for _, d := range details {
		detailsLookup[d.ID] = d.Details
	}

	for _, a := range activities {
		if det, ok := detailsLookup[a.ID]; ok {
			a.Details = det
		}
	}

	return nil
}

// listOptsAdapter adapts types.ListOptions to common_mysql.ListOptions interface.
type listOptsAdapter struct {
	opt types.ListOptions
}

func (a *listOptsAdapter) GetPage() uint                { return a.opt.Page }
func (a *listOptsAdapter) GetPerPage() uint             { return a.opt.PerPage }
func (a *listOptsAdapter) GetOrderKey() string          { return a.opt.OrderKey }
func (a *listOptsAdapter) IsDescending() bool           { return a.opt.OrderDirection == "desc" }
func (a *listOptsAdapter) GetCursorValue() string       { return "" } // Not used for activities list
func (a *listOptsAdapter) WantsPaginationInfo() bool    { return a.opt.IncludeMetadata }
func (a *listOptsAdapter) GetSecondaryOrderKey() string { return "" }
func (a *listOptsAdapter) IsSecondaryDescending() bool  { return false }
