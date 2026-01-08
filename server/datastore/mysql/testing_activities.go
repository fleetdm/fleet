package mysql

import (
	"context"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/activity"
	activityapi "github.com/fleetdm/fleet/v4/server/activity/api"
	activitybootstrap "github.com/fleetdm/fleet/v4/server/activity/bootstrap"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// DatastoreWithActivities wraps a Datastore and provides access to the activity
// bounded context's ListActivities. Use this in tests that need to verify activities
// were created correctly.
type DatastoreWithActivities struct {
	*Datastore
	activitySvc activityapi.ListActivitiesService
}

// CreateMySQLDSWithActivities creates a test datastore along with an activity service
// that shares the same database connection. Use this when tests need to call ListActivities.
func CreateMySQLDSWithActivities(t *testing.T) *DatastoreWithActivities {
	t.Helper()

	// Use unique test name option so each test gets its own database
	opts := &testing_utils.DatastoreTestOptions{
		UniqueTestName: t.Name(),
	}
	ds := CreateMySQLDSWithOptions(t, opts)

	// Use the same test name for the activity service connection
	cleanTestName := strings.ReplaceAll(t.Name(), "/", "_")
	cleanTestName = strings.ReplaceAll(cleanTestName, ".", "_")
	if len(cleanTestName) > 60 {
		cleanTestName = cleanTestName[len(cleanTestName)-60:]
	}

	commonCfg := testing_utils.MysqlTestConfig(cleanTestName)

	// Create a separate DB connection for the activity service
	activityDB, err := common_mysql.NewDB(commonCfg, &common_mysql.DBOptions{}, "")
	require.NoError(t, err)

	t.Cleanup(func() {
		activityDB.Close()
	})

	// Create the activity service
	activitySvc, _ := activitybootstrap.New(
		activityDB,
		activityDB,
		&testAuthorizer{},
		&testUserProvider{db: activityDB},
		log.NewNopLogger(),
	)

	return &DatastoreWithActivities{
		Datastore:   ds,
		activitySvc: activitySvc,
	}
}

// ListActivities returns activities using the activity bounded context.
// This replaces calls to ds.ListActivities in tests.
func (d *DatastoreWithActivities) ListActivities(ctx context.Context, opts fleet.ListActivitiesOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
	// Convert OrderDirection to string
	orderDir := "asc"
	if opts.OrderDirection == fleet.OrderDescending {
		orderDir = "desc"
	}

	// Apply default PerPage like legacy datastore
	perPage := opts.PerPage
	if perPage == 0 {
		perPage = fleet.DefaultPerPage
	}

	// Convert legacy options to API options
	apiOpts := activityapi.ListOptions{
		Page:           opts.Page,
		PerPage:        perPage,
		After:          opts.After,
		OrderKey:       opts.OrderKey,
		OrderDirection: orderDir,
		ActivityType:   opts.ActivityType,
		StartCreatedAt: opts.StartCreatedAt,
		EndCreatedAt:   opts.EndCreatedAt,
		MatchQuery:     opts.MatchQuery,
		Streamed:       opts.Streamed,
	}

	// Call activity service
	activities, meta, err := d.activitySvc.ListActivities(ctx, apiOpts)
	if err != nil {
		return nil, nil, err
	}

	// Convert API types to legacy types
	legacyActivities := make([]*fleet.Activity, len(activities))
	for i, a := range activities {
		legacyActivities[i] = &fleet.Activity{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: a.CreatedAt},
			ID:              a.ID,
			UUID:            a.UUID,
			ActorFullName:   a.ActorFullName,
			ActorID:         a.ActorID,
			ActorGravatar:   a.ActorGravatar,
			ActorEmail:      a.ActorEmail,
			ActorAPIOnly:    a.ActorAPIOnly,
			Type:            a.Type,
			Details:         a.Details,
			Streamed:        a.Streamed,
			FleetInitiated:  a.FleetInitiated,
		}
	}

	var legacyMeta *fleet.PaginationMetadata
	if meta != nil {
		legacyMeta = &fleet.PaginationMetadata{
			HasNextResults:     meta.HasNextResults,
			HasPreviousResults: meta.HasPreviousResults,
			TotalResults:       meta.TotalResults,
		}
	}

	return legacyActivities, legacyMeta, nil
}

// testAuthorizer is a simple authorizer that allows all operations for tests.
type testAuthorizer struct{}

func (a *testAuthorizer) Authorize(ctx context.Context, subject platform_authz.AuthzTyper, action string) error {
	return nil
}

// testUserProvider queries users from the database for activity enrichment.
type testUserProvider struct {
	db *sqlx.DB
}

func (p *testUserProvider) ListUsers(ctx context.Context, ids []uint) ([]*activity.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := sqlx.In(`
		SELECT id, name, email, gravatar_url AS gravatar, api_only AS apionly
		FROM users
		WHERE id IN (?)
	`, ids)
	if err != nil {
		return nil, err
	}

	var users []*activity.User
	if err := sqlx.SelectContext(ctx, p.db, &users, query, args...); err != nil {
		return nil, err
	}
	return users, nil
}

func (p *testUserProvider) SearchUsers(ctx context.Context, query string) ([]uint, error) {
	if query == "" {
		return nil, nil
	}

	var ids []uint
	err := sqlx.SelectContext(ctx, p.db, &ids, `
		SELECT id FROM users
		WHERE name LIKE ? OR email LIKE ?
	`, query+"%", query+"%")
	if err != nil {
		return nil, err
	}
	return ids, nil
}
