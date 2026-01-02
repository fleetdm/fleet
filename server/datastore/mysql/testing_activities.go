package mysql

import (
	"context"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/activity"
	activityapi "github.com/fleetdm/fleet/v4/server/activity/api"
	activitybootstrap "github.com/fleetdm/fleet/v4/server/activity/bootstrap"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	"github.com/go-kit/log"
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
		&testUserProvider{},
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
	// Convert legacy options to API options
	apiOpts := activityapi.ListOptions{
		Page:           opts.Page,
		PerPage:        opts.PerPage,
		OrderKey:       opts.OrderKey,
		OrderDirection: string(opts.OrderDirection),
		ActivityType:   opts.ActivityType,
		StartCreatedAt: opts.StartCreatedAt,
		EndCreatedAt:   opts.EndCreatedAt,
		MatchQuery:     opts.MatchQuery,
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

// testUserProvider is a simple user provider that returns empty results for tests.
type testUserProvider struct{}

func (p *testUserProvider) ListUsers(ctx context.Context, ids []uint) ([]*activity.User, error) {
	return nil, nil
}

func (p *testUserProvider) SearchUsers(ctx context.Context, query string) ([]uint, error) {
	return nil, nil
}
