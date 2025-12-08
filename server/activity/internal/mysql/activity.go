// Package mysql implements the MySQL datastore for the activity bounded context.
// This package is internal to the activity bounded context and should not be
// imported by other bounded contexts.
package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

var (
	automationActivityAuthor = "Fleet"
	deleteIDsBatchSize       = 1000
)

// Datastore implements the activity datastore interface using MySQL.
type Datastore struct {
	primary *sqlx.DB
	replica *sqlx.DB
	logger  log.Logger
}

// NewDatastore creates a new MySQL store for activities.
// It accepts the same database connections used by the main datastore,
// allowing the activity bounded context to share connections.
func NewDatastore(primary *sqlx.DB, replica *sqlx.DB, logger log.Logger) *Datastore {
	return &Datastore{
		primary: primary,
		replica: replica,
		logger:  logger,
	}
}

func (ds *Datastore) writer(ctx context.Context) *sqlx.DB {
	return ds.primary
}

func (ds *Datastore) reader(ctx context.Context) *sqlx.DB {
	return ds.replica
}

// Ping verifies database connectivity by querying the activities table.
func (ds *Datastore) Ping(ctx context.Context) error {
	var result int
	return ds.replica.QueryRowxContext(ctx, "SELECT 1 FROM activities LIMIT 1").Scan(&result)
}

// NewActivity stores an activity item that the user performed.
// Note: This method does NOT handle unified queue activation (activateNextUpcomingActivity).
// That functionality remains in the main datastore as it's part of the unified queue (out of scope for this pilot).
func (ds *Datastore) NewActivity(
	ctx context.Context, actor *activity.Actor, details activity.Details, detailsJSON []byte, createdAt time.Time,
) error {
	// Sanity check to ensure we processed activity webhook before storing the activity
	processed, _ := ctx.Value(activity.WebhookContextKey).(bool)
	if !processed {
		return ctxerr.New(
			ctx, "activity webhook not processed. Please use svc.NewActivity instead of ds.NewActivity. This is a Fleet server bug.",
		)
	}

	var userID *uint
	var userName *string
	var userEmail *string
	var fleetInitiated bool
	var hostOnly bool
	if actor != nil {
		// To support creating activities with users that were deleted. This can happen
		// for automatically installed software which uses the author of the upload as the author of
		// the installation.
		if actor.ID != 0 && !actor.Deleted {
			userID = &actor.ID
		}
		userName = &actor.Name
		userEmail = &actor.Email
	}
	if automatableDetails, ok := details.(activity.AutomatableDetails); ok && automatableDetails.WasFromAutomation() {
		userName = &automationActivityAuthor
		fleetInitiated = true
	}

	if hostOnlyDetails, ok := details.(activity.HostOnlyDetails); ok && hostOnlyDetails.HostOnly() {
		hostOnly = true
	}

	cols := []string{"fleet_initiated", "user_id", "user_name", "activity_type", "details", "created_at", "host_only"}
	args := []any{
		fleetInitiated,
		userID,
		userName,
		details.ActivityName(),
		detailsJSON,
		createdAt,
		hostOnly,
	}
	if userEmail != nil {
		args = append(args, userEmail)
		cols = append(cols, "user_email")
	}

	return common_mysql.WithRetryTxx(ctx, ds.primary, func(tx sqlx.ExtContext) error {
		const insertActStmt = `INSERT INTO activities (%s) VALUES (%s)`
		sqlStmt := fmt.Sprintf(insertActStmt, strings.Join(cols, ","), strings.Repeat("?,", len(cols)-1)+"?")
		res, err := tx.ExecContext(ctx, sqlStmt, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new activity")
		}

		// this supposes a reasonable amount of hosts per activity, to revisit if we
		// get in the 10K+.
		if ah, ok := details.(activity.DetailsWithHosts); ok {
			const insertActHostStmt = `INSERT INTO host_activities (host_id, activity_id) VALUES `

			var sb strings.Builder
			if hostIDs := ah.HostIDs(); len(hostIDs) > 0 {
				sb.WriteString(insertActHostStmt)
				actID, _ := res.LastInsertId()
				for _, hid := range hostIDs {
					sb.WriteString(fmt.Sprintf("(%d, %d),", hid, actID))
				}

				stmt := strings.TrimSuffix(sb.String(), ",")
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return ctxerr.Wrap(ctx, err, "insert host activity")
				}
			}
		}
		return nil
	}, ds.logger)
}

// userForActivitySearch is a minimal user struct for searching.
type userForActivitySearch struct {
	ID    uint   `db:"id"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

// ListActivities returns a slice of activities performed across the organization.
func (ds *Datastore) ListActivities(ctx context.Context, opt activity.ListActivitiesOptions) ([]*activity.Activity, *activity.PaginationMetadata, error) {
	activities := []*activity.Activity{}
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
	opt.ListOptions.IncludeMetadata = !(opt.ListOptions.UsesCursorPagination())

	// Searching activities currently only supports searching by user name or email.
	if opt.ListOptions.MatchQuery != "" {
		activitiesQ += " AND (a.user_name LIKE ? OR a.user_email LIKE ?" // Final ')' will be added at the bottom of this IF
		args = append(args, opt.ListOptions.MatchQuery+"%", opt.ListOptions.MatchQuery+"%")

		// Also search the users table here to get the most up to date information
		users, err := ds.listUsersForActivitySearch(ctx, opt.ListOptions.MatchQuery)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "list users for activity search")
		}

		if len(users) != 0 {
			userIDs := make([]uint, 0, len(users))
			for _, u := range users {
				userIDs = append(userIDs, u.ID)
			}

			inQ, inArgs, err := sqlx.In("a.user_id IN (?)", userIDs)
			if err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "bind user IDs for IN clause")
			}
			inQ = ds.reader(ctx).Rebind(inQ)
			activitiesQ += " OR " + inQ
			args = append(args, inArgs...)
		}

		activitiesQ += ")"
	}

	if opt.ActivityType != "" {
		activitiesQ += " AND a.activity_type = ?"
		args = append(args, opt.ActivityType)
	}

	if opt.StartCreatedAt != "" || opt.EndCreatedAt != "" {
		start := opt.StartCreatedAt
		end := opt.EndCreatedAt
		switch {
		case start == "" && end != "":
			// Only EndCreatedAt is set, so filter up to end
			activitiesQ += " AND a.created_at <= ?"
			args = append(args, end)
		case start != "" && end == "":
			// Only StartCreatedAt is set, so filter from start to now
			activitiesQ += " AND a.created_at >= ? AND a.created_at <= ?"
			args = append(args, start, time.Now().UTC())
		case start != "" && end != "":
			// Both are set
			activitiesQ += " AND a.created_at >= ? AND a.created_at <= ?"
			args = append(args, start, end)
		}
	}

	// Convert activity.ListOptions to common_mysql.ListOptions
	commonOpts := toCommonListOptions(&opt.ListOptions)
	activitiesQ, args = common_mysql.AppendListOptionsWithCursorToSQL(activitiesQ, args, commonOpts)
	// Copy back any modifications (like IncludeMetadata changes)
	opt.ListOptions.IncludeMetadata = commonOpts.IncludeMetadata
	opt.ListOptions.Page = commonOpts.Page
	opt.ListOptions.PerPage = commonOpts.PerPage

	err := sqlx.SelectContext(ctx, ds.reader(ctx), &activities, activitiesQ, args...)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select activities")
	}

	if len(activities) > 0 {
		// Fetch details as a separate query due to sort buffer issue triggered by large JSON details entries.
		IDs := make([]uint, 0, len(activities))
		for _, a := range activities {
			IDs = append(IDs, a.ID)
		}
		detailsStmt, detailsArgs, err := sqlx.In("SELECT id, details FROM activities WHERE id IN (?)", IDs)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "Error binding activity IDs")
		}
		type activityDetails struct {
			ID      uint             `db:"id"`
			Details *json.RawMessage `db:"details"`
		}
		var details []activityDetails
		err = sqlx.SelectContext(ctx, ds.reader(ctx), &details, detailsStmt, detailsArgs...)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "select activities details")
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
	}

	// Fetch users as a stand-alone query (because of performance reasons)
	lookup := make(map[uint][]int)
	for idx, a := range activities {
		if a.ActorID != nil {
			lookup[*a.ActorID] = append(lookup[*a.ActorID], idx)
		}
	}

	if len(lookup) != 0 {
		usersQ := `
			SELECT u.id, u.name, u.gravatar_url, u.email, u.api_only
			FROM users u
			WHERE id IN (?)
		`
		userIDs := make([]uint, 0, len(lookup))
		for k := range lookup {
			userIDs = append(userIDs, k)
		}

		usersQ, usersArgs, err := sqlx.In(usersQ, userIDs)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "Error binding usersIDs")
		}

		var usersR []struct {
			ID          uint   `db:"id"`
			Name        string `db:"name"`
			GravatarURL string `db:"gravatar_url"`
			Email       string `db:"email"`
			APIOnly     bool   `db:"api_only"`
		}

		err = sqlx.SelectContext(ctx, ds.reader(ctx), &usersR, usersQ, usersArgs...)
		if err != nil && err != sql.ErrNoRows {
			return nil, nil, ctxerr.Wrap(ctx, err, "selecting users")
		}

		for _, r := range usersR {
			entries, ok := lookup[r.ID]
			if !ok {
				continue
			}

			email := r.Email
			gravatar := r.GravatarURL
			name := r.Name
			apiOnly := r.APIOnly

			for _, idx := range entries {
				activities[idx].ActorEmail = &email
				activities[idx].ActorGravatar = &gravatar
				activities[idx].ActorFullName = &name
				activities[idx].ActorAPIOnly = &apiOnly
			}
		}
	}

	var metaData *activity.PaginationMetadata
	if opt.ListOptions.IncludeMetadata {
		metaData = &activity.PaginationMetadata{HasPreviousResults: opt.ListOptions.Page > 0}
		if len(activities) > int(opt.ListOptions.PerPage) { //nolint:gosec // dismiss G115
			metaData.HasNextResults = true
			activities = activities[:len(activities)-1]
		}
	}

	return activities, metaData, nil
}

// listUsersForActivitySearch is a helper to search users for the activity search feature.
func (ds *Datastore) listUsersForActivitySearch(ctx context.Context, matchQuery string) ([]userForActivitySearch, error) {
	usersQ := `
		SELECT id, name, email
		FROM users
		WHERE name LIKE ? OR email LIKE ?
	`
	var users []userForActivitySearch
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &users, usersQ, matchQuery+"%", matchQuery+"%")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list users for activity search")
	}
	return users, nil
}

// ListHostPastActivities returns past activities for a specific host.
func (ds *Datastore) ListHostPastActivities(ctx context.Context, hostID uint, opt activity.ListOptions) ([]*activity.Activity, *activity.PaginationMetadata, error) {
	const listStmt = `
	SELECT
		ha.activity_id as id,
		a.user_email as user_email,
		a.user_name as name,
		a.activity_type as activity_type,
		a.details as details,
		u.gravatar_url as gravatar_url,
		a.created_at as created_at,
		u.id as user_id,
		u.api_only as api_only,
		a.fleet_initiated as fleet_initiated
	FROM
		host_activities ha
		JOIN activities a
			ON ha.activity_id = a.id
		LEFT OUTER JOIN
			users u ON u.id = a.user_id
	WHERE
		ha.host_id = ?
	`

	args := []any{hostID}
	commonOpts := toCommonListOptions(&opt)
	stmt, args := common_mysql.AppendListOptionsWithCursorToSQL(listStmt, args, commonOpts)
	// Copy back any modifications
	opt.IncludeMetadata = commonOpts.IncludeMetadata
	opt.Page = commonOpts.Page
	opt.PerPage = commonOpts.PerPage

	var activities []*activity.Activity
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &activities, stmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select host past activities")
	}

	var metaData *activity.PaginationMetadata
	if opt.IncludeMetadata {
		metaData = &activity.PaginationMetadata{HasPreviousResults: opt.Page > 0}
		if len(activities) > int(opt.PerPage) { //nolint:gosec // dismiss G115
			metaData.HasNextResults = true
			activities = activities[:len(activities)-1]
		}
	}

	return activities, metaData, nil
}

// MarkActivitiesAsStreamed marks the specified activities as streamed.
func (ds *Datastore) MarkActivitiesAsStreamed(ctx context.Context, activityIDs []uint) error {
	stmt := `UPDATE activities SET streamed = true WHERE id IN (?);`
	query, args, err := sqlx.In(stmt, activityIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sqlx.In mark activities as streamed")
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "exec mark activities as streamed")
	}
	return nil
}

// CleanupActivitiesAndAssociatedData removes old activities and related data.
func (ds *Datastore) CleanupActivitiesAndAssociatedData(ctx context.Context, maxCount int, expiredWindowDays int) error {
	const selectActivitiesQuery = `
		SELECT a.id FROM activities a
		LEFT JOIN host_activities ha ON (a.id=ha.activity_id)
		WHERE ha.activity_id IS NULL AND a.created_at < DATE_SUB(NOW(), INTERVAL ? DAY)
		ORDER BY a.id ASC
		LIMIT ?;`
	var activityIDs []uint
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &activityIDs, selectActivitiesQuery, expiredWindowDays, maxCount); err != nil {
		return ctxerr.Wrap(ctx, err, "select activities for deletion")
	}
	if len(activityIDs) > 0 {
		deleteActivitiesQuery, args, err := sqlx.In(`DELETE FROM activities WHERE id IN (?);`, activityIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build activities IN query")
		}
		if _, err := ds.writer(ctx).ExecContext(ctx, deleteActivitiesQuery, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete expired activities")
		}
	}

	// Cleanup unsaved queries
	const selectUnsavedQueryIDs = `
		SELECT id
		FROM queries
		WHERE NOT saved
		AND created_at < DATE_SUB(NOW(), INTERVAL ? DAY)
		ORDER BY id`
	var allUnsavedQueryIDs []uint

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &allUnsavedQueryIDs, selectUnsavedQueryIDs, expiredWindowDays); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting expired unsaved query IDs")
	}

	unsavedQueryIter := slices.Chunk(allUnsavedQueryIDs, deleteIDsBatchSize)

	for unsavedQueryIDs := range unsavedQueryIter {
		const deleteStmt = `DELETE FROM queries WHERE id IN (?)`
		deleteQuery, args, err := sqlx.In(deleteStmt, unsavedQueryIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating query to delete unsaved queries")
		}
		if _, err := ds.writer(ctx).ExecContext(ctx, deleteQuery, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting expired unsaved queries")
		}
	}

	// Cleanup orphaned distributed campaigns that reference non-existing queries.
	const selectCampaignIDs = `
		SELECT id
		FROM distributed_query_campaigns dqc
		WHERE NOT EXISTS (
			SELECT 1
			FROM queries q
			WHERE q.id = dqc.query_id
		)
		ORDER BY id`
	var allCampaignIDs []uint

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &allCampaignIDs, selectCampaignIDs); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting expired distributed query campaigns")
	}

	campaignIter := slices.Chunk(allCampaignIDs, deleteIDsBatchSize)

	for campaignIDs := range campaignIter {
		const deleteStmt = `DELETE FROM distributed_query_campaigns WHERE id IN (?)`
		deleteQuery, args, err := sqlx.In(deleteStmt, campaignIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating delete expired distributed query campaigns stmt")
		}
		if _, err := ds.writer(ctx).ExecContext(ctx, deleteQuery, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting expired distributed query campaigns")
		}
	}

	// Cleanup orphaned distributed campaign targets that reference non-existing distributed campaigns.
	const selectCampaignTargets = `
		SELECT id
		FROM distributed_query_campaign_targets dqct
		WHERE NOT EXISTS (
			SELECT 1
			FROM distributed_query_campaigns dqc
			WHERE dqc.id = dqct.distributed_query_campaign_id
		)
		ORDER BY id`
	var allCampaignTargetIDs []uint

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &allCampaignTargetIDs, selectCampaignTargets); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting expired distributed query campaign targets")
	}

	campaignTargetIter := slices.Chunk(allCampaignTargetIDs, deleteIDsBatchSize)

	for campaignTargetIDs := range campaignTargetIter {
		const deleteStmt = `DELETE FROM distributed_query_campaign_targets WHERE id IN (?)`
		deleteQuery, args, err := sqlx.In(deleteStmt, campaignTargetIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating query to delete expired query campaign targets")
		}
		if _, err := ds.writer(ctx).ExecContext(ctx, deleteQuery, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting expired distributed query campaign targets")
		}
	}

	return nil
}

// toCommonListOptions converts activity.ListOptions to common_mysql.ListOptions.
func toCommonListOptions(opt *activity.ListOptions) *common_mysql.ListOptions {
	return &common_mysql.ListOptions{
		Page:                        opt.Page,
		PerPage:                     opt.PerPage,
		OrderKey:                    opt.OrderKey,
		OrderDirection:              common_mysql.OrderDirection(opt.OrderDirection),
		MatchQuery:                  opt.MatchQuery,
		After:                       opt.After,
		IncludeMetadata:             opt.IncludeMetadata,
		TestSecondaryOrderKey:       opt.TestSecondaryOrderKey,
		TestSecondaryOrderDirection: common_mysql.OrderDirection(opt.TestSecondaryOrderDirection),
	}
}
