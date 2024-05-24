package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ActivityTypeTest struct {
	Name string `json:"name"`
}

func (a ActivityTypeTest) ActivityName() string {
	return "test_activity"
}

func (a ActivityTypeTest) Documentation() (activity string, details string, detailsExample string) {
	return "test_activity", "test_activity", "test_activity"
}

func TestListActivities(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	globalUsers := []*fleet.User{test.UserAdmin, test.UserMaintainer, test.UserObserver, test.UserObserverPlus}
	teamUsers := []*fleet.User{test.UserTeamAdminTeam1, test.UserTeamMaintainerTeam1, test.UserTeamObserverTeam1}

	ds.ListActivitiesFunc = func(ctx context.Context, opts fleet.ListActivitiesOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
		return []*fleet.Activity{
			{ID: 1},
			{ID: 2},
		}, nil, nil
	}

	// any global user can read activities
	for _, u := range globalUsers {
		activities, _, err := svc.ListActivities(test.UserContext(ctx, u), fleet.ListActivitiesOptions{})
		require.NoError(t, err)
		require.Len(t, activities, 2)
	}

	// team users cannot read activities
	for _, u := range teamUsers {
		_, _, err := svc.ListActivities(test.UserContext(ctx, u), fleet.ListActivitiesOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
	}

	// user with no roles cannot read activities
	_, _, err := svc.ListActivities(test.UserContext(ctx, test.UserNoRoles), fleet.ListActivitiesOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)

	// no user in context
	_, _, err = svc.ListActivities(ctx, fleet.ListActivitiesOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func Test_logRoleChangeActivities(t *testing.T) {
	tests := []struct {
		name             string
		oldRole          *string
		newRole          *string
		oldTeamRoles     map[uint]string
		newTeamRoles     map[uint]string
		expectActivities []string
	}{
		{
			name: "Empty",
		}, {
			name:             "AddGlobal",
			newRole:          ptr.String("role"),
			expectActivities: []string{"changed_user_global_role"},
		}, {
			name:             "NoChangeGlobal",
			oldRole:          ptr.String("role"),
			newRole:          ptr.String("role"),
			expectActivities: []string{},
		}, {
			name:             "ChangeGlobal",
			oldRole:          ptr.String("old"),
			newRole:          ptr.String("role"),
			expectActivities: []string{"changed_user_global_role"},
		}, {
			name:             "Delete",
			oldRole:          ptr.String("old"),
			newRole:          nil,
			expectActivities: []string{"deleted_user_global_role"},
		}, {
			name:    "SwitchGlobalToTeams",
			oldRole: ptr.String("old"),
			newTeamRoles: map[uint]string{
				1: "foo",
				2: "bar",
				3: "baz",
			},
			expectActivities: []string{"deleted_user_global_role", "changed_user_team_role", "changed_user_team_role", "changed_user_team_role"},
		}, {
			name: "DeleteModifyTeam",
			oldTeamRoles: map[uint]string{
				1: "foo",
				2: "bar",
				3: "baz",
			},
			newTeamRoles: map[uint]string{
				2: "newRole",
				3: "baz",
			},
			expectActivities: []string{"changed_user_team_role", "deleted_user_team_role"},
		},
	}
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	var activities []string
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		activities = append(activities, activity.ActivityName())
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			activities = activities[:0]
			oldTeams := make([]fleet.UserTeam, 0, len(tt.oldTeamRoles))
			for id, r := range tt.oldTeamRoles {
				oldTeams = append(oldTeams, fleet.UserTeam{
					Team: fleet.Team{ID: id},
					Role: r,
				})
			}
			newTeams := make([]fleet.UserTeam, 0, len(tt.newTeamRoles))
			for id, r := range tt.newTeamRoles {
				newTeams = append(newTeams, fleet.UserTeam{
					Team: fleet.Team{ID: id},
					Role: r,
				})
			}
			newUser := &fleet.User{
				GlobalRole: tt.newRole,
				Teams:      newTeams,
			}
			require.NoError(t, fleet.LogRoleChangeActivities(ctx, svc, &fleet.User{}, tt.oldRole, oldTeams, newUser))
			require.Equal(t, tt.expectActivities, activities)
		})
	}
}

func TestActivityWebhooks(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	var webhookBody = ActivityWebhookPayload{}
	webhookChannel := make(chan struct{}, 1)
	fail429 := false
	startMockServer := func(t *testing.T) string {
		// create a test http server
		srv := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					webhookBody = ActivityWebhookPayload{}
					if r.Method != "POST" {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return // don't send the channel signal
					}
					switch r.URL.Path {
					case "/ok":
						err := json.NewDecoder(r.Body).Decode(&webhookBody)
						if err != nil {
							t.Log(err)
							w.WriteHeader(http.StatusBadRequest)
						}
					case "/error":
						webhookBody.Type = "error" // to check for testing
						w.WriteHeader(http.StatusTeapot)
					case "/429":
						// Only the first request will fail
						fail429 = !fail429
						if fail429 {
							w.WriteHeader(http.StatusTooManyRequests)
							return // don't send the channel signal
						}
						err := json.NewDecoder(r.Body).Decode(&webhookBody)
						if err != nil {
							t.Log(err)
							w.WriteHeader(http.StatusBadRequest)
						}
					default:
						w.WriteHeader(http.StatusNotFound)
						return // don't send the channel signal
					}
					webhookChannel <- struct{}{}
				},
			),
		)
		t.Cleanup(srv.Close)
		return srv.URL
	}
	mockUrl := startMockServer(t)
	testUrl := mockUrl

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			WebhookSettings: fleet.WebhookSettings{
				ActivitiesWebhook: fleet.ActivitiesWebhookSettings{
					Enable:         true,
					DestinationURL: testUrl,
				},
			},
		}, nil
	}
	var activityUser *fleet.User
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		activityUser = user
		assert.NotEmpty(t, details)
		assert.True(t, createdAt.After(time.Now().Add(-10*time.Second)))
		assert.False(t, createdAt.After(time.Now()))
		return nil
	}

	tests := []struct {
		name    string
		user    *fleet.User
		url     string
		doError bool
	}{
		{
			name: "nil user",
			url:  mockUrl + "/ok",
			user: nil,
		},
		{
			name: "real user",
			url:  mockUrl + "/ok",
			user: &fleet.User{
				ID:    1,
				Name:  "testUser",
				Email: "testUser@example.com",
			},
		},
		{
			name:    "error",
			url:     mockUrl + "/error",
			doError: true,
		},
		{
			name: "429",
			url:  mockUrl + "/429",
			user: &fleet.User{
				ID:    2,
				Name:  "testUser2",
				Email: "testUser2@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				ds.NewActivityFuncInvoked = false
				testUrl = tt.url
				startTime := time.Now()
				activity := ActivityTypeTest{Name: tt.name}
				err := svc.NewActivity(ctx, tt.user, activity)
				require.NoError(t, err)
				select {
				case <-time.After(1 * time.Second):
					t.Error("timeout")
				case <-webhookChannel:
					if tt.doError {
						assert.Equal(t, "error", webhookBody.Type)
					} else {
						endTime := time.Now()
						assert.False(
							t, webhookBody.Timestamp.Before(startTime), "timestamp %s is before start time %s",
							webhookBody.Timestamp.String(), startTime.String(),
						)
						assert.False(t, webhookBody.Timestamp.After(endTime))
						if tt.user == nil {
							assert.Nil(t, webhookBody.ActorFullName)
							assert.Nil(t, webhookBody.ActorID)
							assert.Nil(t, webhookBody.ActorEmail)
						} else {
							require.NotNil(t, webhookBody.ActorFullName)
							assert.Equal(t, tt.user.Name, *webhookBody.ActorFullName)
							require.NotNil(t, webhookBody.ActorID)
							assert.Equal(t, tt.user.ID, *webhookBody.ActorID)
							require.NotNil(t, webhookBody.ActorEmail)
							assert.Equal(t, tt.user.Email, *webhookBody.ActorEmail)
						}
						assert.Equal(t, activity.ActivityName(), webhookBody.Type)
						var details map[string]string
						require.NoError(t, json.Unmarshal(*webhookBody.Details, &details))
						assert.Len(t, details, 1)
						assert.Equal(t, tt.name, details["name"])
					}
				}
				require.True(t, ds.NewActivityFuncInvoked)
				assert.Equal(t, tt.user, activityUser)
			},
		)
	}
}

func TestActivityWebhooksDisabled(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	startMockServer := func(t *testing.T) string {
		// create a test http server
		srv := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					t.Error("should not be called")
				},
			),
		)
		t.Cleanup(srv.Close)
		return srv.URL
	}
	mockUrl := startMockServer(t)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			WebhookSettings: fleet.WebhookSettings{
				ActivitiesWebhook: fleet.ActivitiesWebhookSettings{
					Enable:         false,
					DestinationURL: mockUrl,
				},
			},
		}, nil
	}
	var activityUser *fleet.User
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		activityUser = user
		assert.NotEmpty(t, details)
		assert.True(t, createdAt.After(time.Now().Add(-10*time.Second)))
		assert.False(t, createdAt.After(time.Now()))
		return nil
	}
	activity := ActivityTypeTest{Name: "no webhook"}
	user := &fleet.User{
		ID:    1,
		Name:  "testUser",
		Email: "testUser@example.com",
	}
	require.NoError(t, svc.NewActivity(ctx, user, activity))
	require.True(t, ds.NewActivityFuncInvoked)
	assert.Equal(t, user, activityUser)
}
