// Package mysql implements the MySQL datastore for the activity bounded context.
// This package is internal to the activity bounded context and should not be
// imported by other bounded contexts.
package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

var columnCharsRegexp = regexp.MustCompile(`[^a-zA-Z0-9_.\-]`)

// Datastore implements the activity bounded context's datastore interface.
type Datastore struct {
	primary *sqlx.DB
	replica *sqlx.DB
}

// NewDatastore creates a new MySQL store for activities.
// It accepts the same database connections used by the main datastore,
// allowing the activity bounded context to share connections.
func NewDatastore(primary *sqlx.DB, replica *sqlx.DB) *Datastore {
	return &Datastore{
		primary: primary,
		replica: replica,
	}
}

// Ping verifies database connectivity by querying the activities table.
func (s *Datastore) Ping(ctx context.Context) error {
	var result int
	return s.replica.QueryRowxContext(ctx, "SELECT 1 FROM activities LIMIT 1").Scan(&result)
}

// ListActivities returns a paginated list of activities for the organization.
func (s *Datastore) ListActivities(ctx context.Context, opt activity.ListActivitiesOptions) ([]*activity.Activity, *activity.PaginationMetadata, error) {
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
		activitiesQ += " AND (a.user_name LIKE ? OR a.user_email LIKE ?"
		args = append(args, opt.ListOptions.MatchQuery+"%", opt.ListOptions.MatchQuery+"%")

		// Also search the users table to get the most up to date information
		users, err := s.ListUsers(ctx, activity.UserListOptions{
			ListOptions: activity.ListOptions{
				MatchQuery: opt.ListOptions.MatchQuery,
			},
		})
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
			inQ = s.replica.Rebind(inQ)
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

	activitiesQ, args = appendListOptionsWithCursorToSQL(activitiesQ, args, &opt.ListOptions)

	if err := sqlx.SelectContext(ctx, s.replica, &activities, activitiesQ, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select activities")
	}

	if len(activities) > 0 {
		// Fetch details as a separate query due to sort buffer issue with large JSON
		IDs := make([]uint, 0, len(activities))
		for _, a := range activities {
			IDs = append(IDs, a.ID)
		}
		detailsStmt, detailsArgs, err := sqlx.In("SELECT id, details FROM activities WHERE id IN (?)", IDs)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "error binding activity IDs")
		}
		type activityDetails struct {
			ID      uint             `db:"id"`
			Details *json.RawMessage `db:"details"`
		}
		var details []activityDetails
		if err := sqlx.SelectContext(ctx, s.replica, &details, detailsStmt, detailsArgs...); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "select activities details")
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
	}

	// Fetch users as a stand-alone query for performance
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
			return nil, nil, ctxerr.Wrap(ctx, err, "error binding user IDs")
		}

		var usersR []struct {
			ID          uint   `db:"id"`
			Name        string `db:"name"`
			GravatarURL string `db:"gravatar_url"`
			Email       string `db:"email"`
			APIOnly     bool   `db:"api_only"`
		}

		if err := sqlx.SelectContext(ctx, s.replica, &usersR, usersQ, usersArgs...); err != nil && err != sql.ErrNoRows {
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
		metaData = &activity.PaginationMetadata{HasPreviousResults: opt.Page > 0}
		if len(activities) > int(opt.ListOptions.PerPage) { //nolint:gosec // dismiss G115
			metaData.HasNextResults = true
			activities = activities[:len(activities)-1]
		}
	}

	return activities, metaData, nil
}

// ListUsers returns a list of users matching the given options.
func (s *Datastore) ListUsers(ctx context.Context, opt activity.UserListOptions) ([]*activity.User, error) {
	query := `SELECT id, name, email, gravatar_url, api_only FROM users WHERE TRUE`
	var args []any

	if opt.ListOptions.MatchQuery != "" {
		query += " AND (name LIKE ? OR email LIKE ?)"
		args = append(args, opt.ListOptions.MatchQuery+"%", opt.ListOptions.MatchQuery+"%")
	}

	query += " ORDER BY name"

	if opt.ListOptions.PerPage > 0 {
		query += fmt.Sprintf(" LIMIT %d", opt.ListOptions.PerPage)
	}

	var users []*activity.User
	if err := sqlx.SelectContext(ctx, s.replica, &users, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list users")
	}
	return users, nil
}

// Helper functions for SQL query building

func sanitizeColumn(col string) string {
	col = columnCharsRegexp.ReplaceAllString(col, "")
	oldParts := strings.Split(col, ".")
	parts := oldParts[:0]
	for _, p := range oldParts {
		if len(p) != 0 {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	col = "`" + strings.Join(parts, "`.`") + "`"
	return col
}

func appendListOptionsWithCursorToSQL(sql string, params []any, opts *activity.ListOptions) (string, []any) {
	orderKey := sanitizeColumn(opts.OrderKey)

	if opts.After != "" && orderKey != "" {
		afterSQL := " WHERE "
		if strings.Contains(strings.ToLower(sql), "where") {
			afterSQL = " AND "
		}
		if strings.HasSuffix(orderKey, "id") {
			i, _ := strconv.Atoi(opts.After)
			params = append(params, i)
		} else {
			params = append(params, opts.After)
		}
		direction := ">"
		if opts.OrderDirection == activity.OrderDescending {
			direction = "<"
		}
		sql = fmt.Sprintf("%s %s %s %s ?", sql, afterSQL, orderKey, direction)
		opts.Page = 0
	}

	if orderKey != "" {
		direction := "ASC"
		if opts.OrderDirection == activity.OrderDescending {
			direction = "DESC"
		}
		sql = fmt.Sprintf("%s ORDER BY %s %s", sql, orderKey, direction)
	}

	perPage := opts.PerPage
	if opts.IncludeMetadata {
		perPage++
	}

	sql = appendLimitOffsetToSQL(sql, perPage, opts.Page)

	return sql, params
}

func appendLimitOffsetToSQL(sql string, perPage, page uint) string {
	if perPage > 0 {
		sql = fmt.Sprintf("%s LIMIT %d", sql, perPage)
	}
	offset := perPage * page
	if offset > 0 {
		sql = fmt.Sprintf("%s OFFSET %d", sql, offset)
	}
	return sql
}
