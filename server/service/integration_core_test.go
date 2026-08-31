package service

import (
	"bytes"
	"context"
	"crypto/sha1" // nolint: gosec
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	logtestutils "github.com/fleetdm/fleet/v4/server/platform/logging/testutils"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type integrationTestSuite struct {
	suite.Suite

	withServer
}

func (s *integrationTestSuite) SetupSuite() {
	s.withServer.lq = live_query_mock.New(s.T())
	s.withServer.SetupSuite("integrationTestSuite")
}

func (s *integrationTestSuite) TearDownTest() {
	s.withServer.commonTearDownTest(s.T())
}

func TestIntegrations(t *testing.T) {
	testingSuite := new(integrationTestSuite)
	testingSuite.withServer.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type slowReader struct{}

func (s *slowReader) Read(p []byte) (n int, err error) {
	time.Sleep(3 * time.Second)
	return 0, nil
}

func (s *integrationTestSuite) TestSlowOsqueryHost() {
	t := s.T()
	_, server := RunServerForTestsWithDS(
		t,
		s.ds,
		&TestServerOpts{
			SkipCreateTestUsers: true,
			//nolint:gosec // G112: server is just run for testing this explicit config.
			HTTPServerConfig: &http.Server{ReadTimeout: 2 * time.Second},
			EnableCachedDS:   true,
		},
	)
	defer func() {
		server.Close()
	}()

	req, err := http.NewRequest("POST", server.URL+"/api/v1/osquery/distributed/write", &slowReader{})
	require.NoError(t, err)

	client := fleethttp.NewClient()

	resp, err := client.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusRequestTimeout, resp.StatusCode)
}

// TestMDMAnyMiddlewareAccess performs an end-to-end check through the HTTP
// handler to confirm the new middleware respects each platform toggle.
func (s *integrationTestSuite) TestMDMAnyMiddlewareAccess() {
	t := s.T()
	ctx := context.Background()
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)

	ensureAppleMDMAssets := func() {
		assets := []fleet.MDMConfigAsset{
			{Name: fleet.MDMAssetCACert, Value: []byte("test-ca-cert")},
			{Name: fleet.MDMAssetCAKey, Value: []byte("test-ca-key")},
			{Name: fleet.MDMAssetAPNSCert, Value: []byte("test-apns-cert")},
			{Name: fleet.MDMAssetAPNSKey, Value: []byte("test-apns-key")},
		}
		if err := s.ds.InsertMDMConfigAssets(ctx, assets, nil); err != nil && !mysql.IsDuplicate(err) {
			require.NoError(t, err)
		}
	}
	ensureAppleMDMAssets()

	origMDM := appCfg.MDM
	defer func(orig fleet.MDM) {
		appCfg.MDM = orig
		require.NoError(t, s.ds.SaveAppConfig(ctx, appCfg))
		require.NoError(t, s.ds.SetAndroidEnabledAndConfigured(ctx, orig.AndroidEnabledAndConfigured))
	}(origMDM)

	const endpoint = "/api/latest/fleet/configuration_profiles"

	requestProfiles := func() *http.Response {
		req, err := http.NewRequest("GET", s.server.URL+endpoint, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+s.token)

		resp, err := fleethttp.NewClient().Do(req)
		require.NoError(t, err)
		return resp
	}

	setConfig := func(apple, windows, android bool) {
		appCfg.MDM.EnabledAndConfigured = apple
		appCfg.MDM.WindowsEnabledAndConfigured = windows
		require.NoError(t, s.ds.SaveAppConfig(ctx, appCfg))

		require.NoError(t, s.ds.SetAndroidEnabledAndConfigured(ctx, android))

		appCfg, err = s.ds.AppConfig(ctx)
		require.NoError(t, err)
	}

	setConfig(false, false, false)
	res := s.Do("GET", endpoint, nil, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.ErrMDMNotConfigured.Error())
	require.NoError(t, res.Body.Close())

	assertNotMDMNotConfigured := func() {
		resp := requestProfiles()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.NotEqual(t, http.StatusBadRequest, resp.StatusCode)
		require.NotContains(t, string(body), fleet.ErrMDMNotConfigured.Error())
	}

	setConfig(true, false, false)
	assertNotMDMNotConfigured()

	setConfig(false, true, false)
	assertNotMDMNotConfigured()

	setConfig(false, false, true)
	assertNotMDMNotConfigured()
}

func (s *integrationTestSuite) TestDistributedReadWithChangedQueries() {
	t := s.T()

	spec := []byte(`
  features:
    enable_software_inventory: true
    enable_host_users: true
    detail_query_overrides:
      users: null
      software_macos: "SELECT * FROM foo;"
      unknown_query: "SELECT * FROM bar;"
`)
	s.applyConfig(spec)

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	s.lq.On("QueriesForHost", host.ID).Return(map[string]string{fmt.Sprintf("%d", host.ID): "SELECT 1 FROM osquery;"}, nil)

	// Ensure we can read distributed queries for the host.
	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)

	// Get distributed queries for the host.
	req := getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.Equal(t, "SELECT * FROM foo;", dqResp.Queries["fleet_detail_query_software_macos"])

	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)

	spec = []byte(`
  features:
    enable_software_inventory: true
    enable_host_users: true
    detail_query_overrides:
`)
	s.applyConfig(spec)

	// Get distributed queries for the host.
	req = getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.Contains(t, dqResp.Queries["fleet_detail_query_software_macos"], "FROM apps")
	require.Contains(t, dqResp.Queries["fleet_detail_query_users"], "FROM users")
}

func (s *integrationTestSuite) TestDoubleUserCreationErrors() {
	t := s.T()

	params := fleet.UserPayload{
		Name:       new("user1"),
		Email:      new("email@asd.com"),
		Password:   &test.GoodPassword,
		GlobalRole: new(fleet.RoleObserver),
	}

	s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusOK)
	respSecond := s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusConflict)

	assertBodyContains(t, respSecond, `Error 1062`)
}

func (s *integrationTestSuite) TestUserWithoutRoleErrors() {
	t := s.T()

	params := fleet.UserPayload{
		Name:     new("user1"),
		Email:    new("email@asd.com"),
		Password: new(test.GoodPassword),
	}

	resp := s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusUnprocessableEntity)
	assertErrorCodeAndMessage(t, resp, fleet.ErrNoRoleNeeded, "either global role or fleet role needs to be defined")
}

func (s *integrationTestSuite) TestUserEmailValidation() {
	params := fleet.UserPayload{
		Name:       new("user_invalid_email"),
		Email:      new("invalid"),
		Password:   &test.GoodPassword,
		GlobalRole: new(fleet.RoleObserver),
	}

	s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusUnprocessableEntity)

	params.Email = new("user_valid_mail@example.com")
	s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusOK)
}

func (s *integrationTestSuite) TestUserPasswordLengthValidation() {
	params := fleet.UserPayload{
		Name:  new("user_invalid_email"),
		Email: new("test@example.com"),
		// This is 73 characters long
		Password:   new("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaX@1"),
		GlobalRole: new(fleet.RoleObserver),
	}

	resp := s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusUnprocessableEntity)
	assertBodyContains(s.T(), resp, "Could not create user. Password is over the 48 characters limit. If the password is under 48 characters, please check the auth_salt_key_size in your Fleet server config.")
}

func (s *integrationTestSuite) TestUserWithWrongRoleErrors() {
	t := s.T()

	params := fleet.UserPayload{
		Name:       new("user1"),
		Email:      new("email@asd.com"),
		Password:   new(test.GoodPassword),
		GlobalRole: new("wrongrole"),
	}
	resp := s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusUnprocessableEntity)
	assertErrorCodeAndMessage(t, resp, fleet.ErrNoRoleNeeded, "invalid global role: wrongrole")
}

func (s *integrationTestSuite) TestUserCreationWrongTeamErrors() {
	t := s.T()

	teams := []fleet.UserTeam{
		{
			Team: fleet.Team{
				ID: 9999, // non-existent team
			},
			Role: fleet.RoleObserver,
		},
	}

	params := fleet.UserPayload{
		Name:     new("user2"),
		Email:    new("email2@asd.com"),
		Password: new(test.GoodPassword),
		Teams:    &teams,
	}
	resp := s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusUnprocessableEntity)
	assertBodyContains(t, resp, `fleet with id 9999 does not exist`)
}

func (s *integrationTestSuite) TestCreateUserAPIEndpointsRejected() {
	t := s.T()

	// api_endpoints cannot be specified directly on this endpoint.
	var resp createUserResponse
	s.DoJSON("POST", "/api/latest/fleet/users/admin", fleet.UserPayload{
		Name:         new("user1"),
		Email:        new("apireject@example.com"),
		Password:     &test.GoodPassword,
		GlobalRole:   new(fleet.RoleObserver),
		APIEndpoints: &[]fleet.APIEndpointRef{{Method: "GET", Path: "/api/v1/fleet/config"}},
	}, http.StatusUnprocessableEntity, &resp)

	var apiOnlyResp createUserResponse
	s.DoJSON("POST", "/api/latest/fleet/users/admin", fleet.UserPayload{
		Name:       new("api-only-legacy"),
		Email:      new("api-only-legacy@example.com"),
		Password:   &test.GoodPassword,
		GlobalRole: new(fleet.RoleObserver),
		APIOnly:    new(true),
	}, http.StatusOK, &apiOnlyResp)
	require.True(t, apiOnlyResp.User.APIOnly)
	require.Empty(t, apiOnlyResp.User.APIEndpoints) // nil/empty = full access
}

func (s *integrationTestSuite) TestModifyUserAPIOnlyRejected() {
	t := s.T()

	// Create a regular user to use as target.
	var createResp createUserResponse
	s.DoJSON("POST", "/api/latest/fleet/users/admin", fleet.UserPayload{
		Name:       new("regular-api-protect"),
		Email:      new("regular-api-protect@example.com"),
		Password:   &test.GoodPassword,
		GlobalRole: new(fleet.RoleObserver),
	}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)
	regularID := createResp.User.ID

	// Create an API-only user to use as target.
	var createAPIResp struct {
		User struct {
			ID uint `json:"id"`
		} `json:"user"`
	}
	s.DoJSON("POST", "/api/latest/fleet/users/api_only", map[string]any{
		"name":        "api-only-protect",
		"global_role": "observer",
	}, http.StatusOK, &createAPIResp)
	require.NotZero(t, createAPIResp.User.ID)
	apiOnlyID := createAPIResp.User.ID

	cases := []struct {
		name   string
		userID uint
		body   fleet.UserPayload
	}{
		{"regular user api_only false", regularID, fleet.UserPayload{APIOnly: new(false)}},
		{"regular user api_only true", regularID, fleet.UserPayload{APIOnly: new(true)}},
		{"api-only user api_only false", apiOnlyID, fleet.UserPayload{APIOnly: new(false)}},
		{"api-only user api_only true", apiOnlyID, fleet.UserPayload{APIOnly: new(true)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var modResp modifyUserResponse
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", tc.userID), tc.body, http.StatusUnprocessableEntity, &modResp)
		})
	}
}

func (s *integrationTestSuite) TestCreateAPIOnlyUser() {
	t := s.T()

	type createAPIOnlyUserResponse struct {
		User struct {
			ID         uint    `json:"id"`
			Name       string  `json:"name"`
			Email      string  `json:"email"`
			APIOnly    bool    `json:"api_only"`
			GlobalRole *string `json:"global_role"`
		} `json:"user"`
		Token string `json:"token"`
		Err   string `json:"error,omitempty"`
	}

	cases := []struct {
		name       string
		body       map[string]any
		wantStatus int
		verify     func(t *testing.T, resp createAPIOnlyUserResponse)
	}{
		{
			name:       "missing name",
			body:       map[string]any{"global_role": "observer"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "neither global_role nor fleets",
			body:       map[string]any{"name": "Jane Doe"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "fleets without premium",
			body: map[string]any{
				"name":   "Jane Doe",
				"fleets": []map[string]any{{"id": 9999, "role": "observer"}},
			},
			wantStatus: http.StatusPaymentRequired,
		},
		{
			name: "api_endpoints without premium",
			body: map[string]any{
				"name":        "Jane Doe",
				"global_role": "observer",
				"api_endpoints": []map[string]any{
					{"method": "GET", "path": "/api/v1/fleet/hosts/:id"},
				},
			},
			wantStatus: http.StatusPaymentRequired,
		},
		{
			name: "both global_role and fleets without premium",
			body: map[string]any{
				"name":        "Jane Doe",
				"global_role": "observer",
				"fleets":      []map[string]any{{"id": 9999, "role": "observer"}},
			},
			wantStatus: http.StatusPaymentRequired,
		},
		{
			name: "successful creation with global_role only",
			body: map[string]any{
				"name":        "Jane Doe",
				"global_role": "observer",
			},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, resp createAPIOnlyUserResponse) {
				require.NotEmpty(t, resp.Token, "token must be set")
				require.NotZero(t, resp.User.ID, "user ID must be set")
				require.Equal(t, "Jane Doe", resp.User.Name)
				require.NotEmpty(t, resp.User.Email)
				require.True(t, resp.User.APIOnly, "user must be api_only")
				require.NotNil(t, resp.User.GlobalRole)
				require.Equal(t, "observer", *resp.User.GlobalRole)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var resp createAPIOnlyUserResponse
			s.DoJSON("POST", "/api/latest/fleet/users/api_only", tc.body, tc.wantStatus, &resp)
			if tc.verify != nil {
				tc.verify(t, resp)
			}
		})
	}
}

func (s *integrationTestSuite) TestModifyAPIOnlyUser() {
	t := s.T()

	var createResp struct {
		User struct {
			ID uint `json:"id"`
		} `json:"user"`
		Token string `json:"token"`
		Err   string `json:"error,omitempty"`
	}
	s.DoJSON("POST", "/api/latest/fleet/users/api_only", map[string]any{
		"name":        "API User",
		"global_role": "observer",
	}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)
	require.NotEmpty(t, createResp.Token)
	apiUserID := createResp.User.ID
	apiUserToken := createResp.Token

	s.DoRawNoAuth("PATCH", fmt.Sprintf("/api/latest/fleet/users/api_only/%d", apiUserID), []byte(`{}`), http.StatusUnauthorized)

	s.Do("PATCH", "/api/latest/fleet/users/api_only/999999", map[string]any{
		"name": "New Name",
	}, http.StatusNotFound)

	// Targeting a non-API-only user must be rejected.
	admin := s.users["admin1@example.com"]
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/users/api_only/%d", admin.ID), map[string]any{
		"name": "New Name",
	}, http.StatusUnprocessableEntity)

	var createRegularResp createUserResponse
	s.DoJSON("POST", "/api/latest/fleet/users/admin", fleet.UserPayload{
		Name:       new("regular-modify-api-only"),
		Email:      new("regular-modify-api-only@example.com"),
		Password:   &test.GoodPassword,
		GlobalRole: new(fleet.RoleObserver),
	}, http.StatusOK, &createRegularResp)
	require.NotZero(t, createRegularResp.User.ID)
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/users/api_only/%d", createRegularResp.User.ID), map[string]any{
		"name": "New Name",
	}, http.StatusUnprocessableEntity)

	// An API-only user cannot modify itself: the service layer rejects the
	// self-modify attempt with 422.
	//
	// This is to protect against privilege escalation vulnerability.
	s.token = apiUserToken
	defer func() { s.token = s.getTestAdminToken() }()
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/users/api_only/%d", apiUserID), map[string]any{
		"name": "Self Update",
	}, http.StatusUnprocessableEntity)
	s.token = s.getTestAdminToken()

	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/users/api_only/%d", apiUserID), map[string]any{
		"api_endpoints": []map[string]any{
			{"method": "GET", "path": "/api/v1/fleet/config"},
		},
	}, http.StatusPaymentRequired)
}

func (s *integrationTestSuite) TestQueryCreationLogsActivity() {
	t := s.T()

	admin1 := s.users["admin1@example.com"]
	admin1.GravatarURL = "http://iii.com"
	err := s.ds.SaveUser(context.Background(), &admin1)
	require.NoError(t, err)

	params := fleet.QueryPayload{
		Name:  new("user1"),
		Query: new("select * from time;"),
	}
	var createQueryResp fleet.CreateQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/queries", &params, http.StatusOK, &createQueryResp)
	defer s.cleanupQuery(createQueryResp.Query.ID)
	assert.False(t, createQueryResp.Query.CreatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), createQueryResp.Query.CreatedAt, time.Minute)
	assert.False(t, createQueryResp.Query.UpdatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), createQueryResp.Query.UpdatedAt, time.Minute)

	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)

	assert.GreaterOrEqual(t, len(activities.Activities), 1)
	found := false
	for _, activity := range activities.Activities {
		if activity.Type == "created_saved_query" {
			found = true
			assert.Equal(t, "Test Name admin1@example.com", *activity.ActorFullName)
			require.NotNil(t, activity.ActorGravatar)
			assert.Equal(t, "http://iii.com", *activity.ActorGravatar)
		}
	}
	require.True(t, found)
}

func (s *integrationTestSuite) TestQueryLabelsIncludeAnyRequiresPremium() {
	// POST /api/v1/fleet/queries with labels_include_any should fail with 402 on free tier
	var createResp fleet.CreateQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.QueryPayload{
		Name:             new("test-labels-query"),
		Query:            new("SELECT 1"),
		LabelsIncludeAny: []string{"some-label"},
	}, http.StatusPaymentRequired, &createResp)

	// Create a query without labels_include_any to use for the PATCH test
	var createOKResp fleet.CreateQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.QueryPayload{
		Name:  new("test-labels-query-for-patch"),
		Query: new("SELECT 1"),
	}, http.StatusOK, &createOKResp)
	defer s.cleanupQuery(createOKResp.Query.ID)

	// PATCH with labels_include_any should also fail with 402 on free tier
	var modifyResp fleet.ModifyQueryResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", createOKResp.Query.ID), fleet.QueryPayload{
		LabelsIncludeAny: []string{"some-label"},
	}, http.StatusPaymentRequired, &modifyResp)

	// POST /api/latest/fleet/spec/queries with labels_include_any should fail with 402 on free tier
	var applyResp fleet.ApplyQuerySpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: "test-labels-spec-query", Query: "SELECT 1", LabelsIncludeAny: []string{"some-label"}},
		},
	}, http.StatusPaymentRequired, &applyResp)
}

func (s *integrationTestSuite) TestCreatingAPIOnlyUserReturnsAPIToken() {
	t := s.T()

	defer func() {
		s.token = s.getTestAdminToken()
	}()

	var createResp createUserResponse
	params := fleet.UserPayload{
		Name:       new("someadmin"),
		Email:      new("someadmin@example.com"),
		Password:   new(test.GoodPassword),
		GlobalRole: new(fleet.RoleAdmin),
		APIOnly:    new(false),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.User.ID)
	assert.Nil(t, createResp.Token)

	params = fleet.UserPayload{
		Name:       new("apionly"),
		Email:      new("apionly@example.com"),
		Password:   new(test.GoodPassword),
		GlobalRole: new(fleet.RoleObserver),
		APIOnly:    new(true),
		// AdminForcedPasswordReset is set to false when creating api-only users via `fleetctl user create --api-only`.
		AdminForcedPasswordReset: new(false),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.User.ID)
	assert.NotNil(t, createResp.Token)

	s.token = *createResp.Token
	var chr countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", countHostsRequest{}, http.StatusOK, &chr)
	assert.Equal(t, 0, chr.Count)
}

func (s *integrationTestSuite) TestActivityUserEmailPersistsAfterDeletion() {
	t := s.T()

	// create a new user
	var createResp createUserResponse
	userRawPwd := test.GoodPassword
	params := fleet.UserPayload{
		Name:       new("Gonna B Deleted"),
		Email:      new("goingto@delete.com"),
		Password:   new(userRawPwd),
		GlobalRole: new(fleet.RoleObserver),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.User.ID)
	assert.True(t, createResp.User.AdminForcedPasswordReset)
	u := *createResp.User

	var loginResp fleet.LoginResponse
	s.DoJSON("POST", "/api/latest/fleet/login", params, http.StatusOK, &loginResp)
	require.Equal(t, loginResp.User.ID, u.ID)

	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)

	assert.GreaterOrEqual(t, len(activities.Activities), 1)
	found := false
	for _, activity := range activities.Activities {
		if activity.Type == "user_logged_in" && *activity.ActorFullName == u.Name {
			found = true
			assert.Equal(t, u.Email, *activity.ActorEmail)
		}
	}
	require.True(t, found)

	err := s.ds.DeleteUser(context.Background(), u.ID)
	require.NoError(t, err)

	activities = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)

	assert.GreaterOrEqual(t, len(activities.Activities), 1)
	found = false
	for _, activity := range activities.Activities {
		if activity.Type == "user_logged_in" && *activity.ActorFullName == u.Name {
			found = true
			assert.Equal(t, u.Email, *activity.ActorEmail)
		}
	}
	require.True(t, found)

	// ensure that on exit, the admin token is used
	s.token = s.getTestAdminToken()
}

func (s *integrationTestSuite) TestPremiumOnlyRoles() {
	t := s.T()

	for _, role := range []string{fleet.RoleTechnician, fleet.RoleGitOps, fleet.RoleObserverPlus} {
		t.Run(role, func(t *testing.T) {
			t.Run("login", func(t *testing.T) {
				user := &fleet.User{
					Name:       role,
					Email:      fmt.Sprintf("%s@example.com", role),
					GlobalRole: new(role),
				}
				err := user.SetPassword(test.GoodPassword, 10, 10)
				require.NoError(t, err)
				_, err = s.ds.NewUser(t.Context(), user)
				require.NoError(t, err)

				var loginResp fleet.LoginResponse
				s.DoJSON("POST", "/api/latest/fleet/login", fleet.LoginRequest{
					Email:    fmt.Sprintf("%s@example.com", role),
					Password: test.GoodPassword,
				}, http.StatusPaymentRequired, &loginResp)
			})

			t.Run("create", func(t *testing.T) {
				var createResp createUserResponse
				params := fleet.UserPayload{
					Name:       new(role),
					Email:      new(fmt.Sprintf("%s@example.com", role)),
					Password:   new(test.GoodPassword),
					GlobalRole: new(role),
				}
				s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusPaymentRequired, &createResp)
			})
		})
	}
}

func (s *integrationTestSuite) TestPolicyDeletionLogsActivity() {
	t := s.T()

	admin1 := s.users["admin1@example.com"]
	admin1.GravatarURL = "http://iii.com"
	err := s.ds.SaveUser(context.Background(), &admin1)
	require.NoError(t, err)

	testPolicies := []fleet.PolicyPayload{{
		Name:  "policy1",
		Query: "select * from time;",
	}, {
		Name:  "policy2",
		Query: "select * from osquery_info;",
	}}

	var policyIDs []uint
	for _, policy := range testPolicies {
		var resp fleet.GlobalPolicyResponse
		s.DoJSON("POST", "/api/latest/fleet/policies", policy, http.StatusOK, &resp)
		policyIDs = append(policyIDs, resp.Policy.PolicyData.ID)
	}

	// critical is premium only.
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.PolicyPayload{
		Name:     "policy3",
		Query:    "select * from time;",
		Critical: true,
	}, http.StatusBadRequest, new(struct{}))

	prevActivities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &prevActivities)
	require.GreaterOrEqual(t, len(prevActivities.Activities), 2)

	var deletePoliciesResp fleet.DeleteGlobalPoliciesResponse
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", fleet.DeleteGlobalPoliciesRequest{IDs: policyIDs}, http.StatusOK, &deletePoliciesResp)
	require.Equal(t, len(policyIDs), len(deletePoliciesResp.Deleted))

	newActivities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &newActivities)
	require.Equal(t, len(newActivities.Activities), (len(prevActivities.Activities) + 2))

	var prevDeletes []*fleet.Activity
	for _, a := range prevActivities.Activities {
		if a.Type == "deleted_policy" {
			prevDeletes = append(prevDeletes, a)
		}
	}
	var newDeletes []*fleet.Activity
	for _, a := range newActivities.Activities {
		if a.Type == "deleted_policy" {
			newDeletes = append(newDeletes, a)
		}
	}
	require.Equal(t, len(newDeletes), (len(prevDeletes) + 2))

	type policyDetails struct {
		PolicyID   uint   `json:"policy_id"`
		PolicyName string `json:"policy_name"`
	}
	for _, id := range policyIDs {
		found := false
		for _, d := range newDeletes {
			var details policyDetails
			err := json.Unmarshal([]byte(*d.Details), &details)
			require.NoError(t, err)
			require.NotNil(t, details.PolicyID)
			if id == details.PolicyID {
				found = true
			}

		}
		require.True(t, found)
	}
	for _, p := range testPolicies {
		found := false
		for _, d := range newDeletes {
			var details policyDetails
			err := json.Unmarshal([]byte(*d.Details), &details)
			require.NoError(t, err)
			require.NotNil(t, details.PolicyName)
			if p.Name == details.PolicyName {
				found = true
			}

		}
		require.True(t, found)
	}
}

func (s *integrationTestSuite) TestAppConfigAdditionalQueriesCanBeRemoved() {
	t := s.T()

	spec := []byte(`
  host_expiry_settings:
    host_expiry_enabled: true
    host_expiry_window: 0
  features:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
`)
	s.applyConfig(spec)

	spec = []byte(`
  features:
    enable_host_users: true
    additional_queries: null
`)
	s.applyConfig(spec)

	config := s.getConfig()
	assert.Nil(t, config.Features.AdditionalQueries)
	assert.True(t, config.HostExpirySettings.HostExpiryEnabled)
}

func (s *integrationTestSuite) TestAppConfigDetailQueriesOverrides() {
	t := s.T()

	spec := []byte(`
  features:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
    detail_query_overrides:
      users: null
      software_linux: "select * from blah;"
`)
	s.applyConfig(spec)

	config := s.getConfig()
	require.NotNil(t, config.Features.DetailQueryOverrides)
	require.Nil(t, config.Features.DetailQueryOverrides["users"])
	require.NotNil(t, config.Features.DetailQueryOverrides["software_linux"])
	require.Equal(t, "select * from blah;", *config.Features.DetailQueryOverrides["software_linux"])
}

func (s *integrationTestSuite) TestAppConfigDefaultValues() {
	config := s.getConfig()
	s.Run("Update interval", func() {
		require.Equal(s.T(), 1*time.Hour, config.UpdateInterval.OSQueryDetail)
	})

	s.Run("has logging", func() {
		require.NotNil(s.T(), config.Logging)
	})
}

func (s *integrationTestSuite) TestAppConfigDeprecatedFields() {
	t := s.T()

	spec := []byte(`
  host_settings:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
    enable_software_inventory: true
`)
	s.applyConfig(spec)
	config := s.getConfig()
	require.NotNil(t, config.Features.AdditionalQueries)
	require.True(t, config.Features.EnableHostUsers)
	require.True(t, config.Features.EnableSoftwareInventory)

	spec = []byte(`
  host_settings:
    additional_queries: null
    enable_host_users: false
    enable_software_inventory: false
`)
	s.applyConfig(spec)
	config = s.getConfig()
	require.Nil(t, config.Features.AdditionalQueries)
	require.False(t, config.Features.EnableHostUsers)
	require.False(t, config.Features.EnableSoftwareInventory)

	// Test raw API interactions
	appConfigSpec := map[string]map[string]bool{
		"host_settings":   {"enable_software_inventory": true},
		"server_settings": {"enable_analytics": false},
	}
	s.Do("PATCH", "/api/latest/fleet/config", appConfigSpec, http.StatusOK)
	config = s.getConfig()
	require.True(t, config.Features.EnableSoftwareInventory)

	// Skip our serialization mechanism, to make sure an old config stored in the DB is still valid
	var previousRawConfig string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		err := sqlx.GetContext(context.Background(), q, &previousRawConfig, "SELECT json_value FROM app_config_json")
		if err != nil {
			return err
		}
		insertAppConfigQuery := `INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`
		_, err = q.ExecContext(context.Background(), insertAppConfigQuery, `
    {
      "host_settings": {
        "enable_host_users": false,
        "enable_software_inventory": true,
        "additional_queries": { "foo": "bar" }
      }
    }`)
		return err
	})

	var resp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &resp)
	require.False(t, resp.Features.EnableHostUsers)
	require.True(t, resp.Features.EnableSoftwareInventory)
	require.NotNil(t, resp.Features.AdditionalQueries)

	// restore the previous appconfig so that other tests are not impacted
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		insertAppConfigQuery := `INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`
		_, err := q.ExecContext(context.Background(), insertAppConfigQuery, previousRawConfig)
		return err
	})
}

func (s *integrationTestSuite) TestAppConfigHistoricalData() {
	t := s.T()
	ctx := context.Background()

	// Ensure a known starting state — earlier tests in this suite may have
	// PATCHed the AppConfig (the suite shares state), so an earlier no-op
	// SaveAppConfig with a zero-value Features could have stamped
	// historical_data={false,false} into the stored JSON.
	s.Do("PATCH", "/api/latest/fleet/config",
		map[string]any{"features": map[string]any{"historical_data": map[string]any{"uptime": true, "vulnerabilities": true}}},
		http.StatusOK)
	cfg := s.getConfig()
	require.True(t, cfg.Features.HistoricalData.Uptime)
	require.True(t, cfg.Features.HistoricalData.Vulnerabilities)

	// PATCH only the vulnerabilities sub-key — uptime SHALL remain true.
	// Snapshot the most recent activity ID (any type) as a watermark so we can
	// confirm a new disabled_historical_dataset row is actually emitted.
	preDisableWatermark := s.lastActivityMatches("", "", 0)
	s.Do("PATCH", "/api/latest/fleet/config",
		map[string]any{"features": map[string]any{"historical_data": map[string]any{"vulnerabilities": false}}},
		http.StatusOK)
	cfg = s.getConfig()
	require.True(t, cfg.Features.HistoricalData.Uptime, "uptime preserved when omitted from PATCH")
	require.False(t, cfg.Features.HistoricalData.Vulnerabilities)

	// A new disabled_historical_dataset activity for vulnerabilities, no fleet scope.
	require.Greater(t, s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(),
		`{"dataset":"vulnerabilities","fleet_id":null,"fleet_name":null}`,
		0,
	), preDisableWatermark, "new disable activity emitted for PATCH")

	// PATCH the same value back — no new activity should be emitted.
	priorActivityID := s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(), "", 0,
	)
	s.Do("PATCH", "/api/latest/fleet/config",
		map[string]any{"features": map[string]any{"historical_data": map[string]any{"vulnerabilities": false}}},
		http.StatusOK)
	require.Equal(t, priorActivityID, s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(), "", 0,
	), "no new activity for no-op PATCH")

	// Flip both in one PATCH — re-enable vulnerabilities, disable uptime → 2 activities.
	// Use the most recent activity ID (any type) as a watermark; the new
	// enabled/disabled activities for this PATCH must have IDs greater than it.
	preFlipWatermark := s.lastActivityMatches("", "", 0)
	s.Do("PATCH", "/api/latest/fleet/config",
		map[string]any{"features": map[string]any{"historical_data": map[string]any{"uptime": false, "vulnerabilities": true}}},
		http.StatusOK)
	cfg = s.getConfig()
	require.False(t, cfg.Features.HistoricalData.Uptime)
	require.True(t, cfg.Features.HistoricalData.Vulnerabilities)
	require.Greater(t, s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEnabledHistoricalDataset{}.ActivityName(),
		`{"dataset":"vulnerabilities","fleet_id":null,"fleet_name":null}`,
		0,
	), preFlipWatermark, "new enable activity emitted for vulnerabilities re-enable")
	require.Greater(t, s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledHistoricalDataset{}.ActivityName(),
		`{"dataset":"uptime","fleet_id":null,"fleet_name":null}`,
		0,
	), preFlipWatermark, "new disable activity emitted for uptime")

	// Existing rows whose stored JSON omits historical_data SHALL read back
	// with both sub-keys true. Simulate a pre-change deployment by writing a
	// row whose features block lacks the key, then verify that AppConfig
	// reads back with defaults applied.
	var previousRawConfig string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		if err := sqlx.GetContext(ctx, q, &previousRawConfig, "SELECT json_value FROM app_config_json"); err != nil {
			return err
		}
		preChangeJSON := []byte(`{"features": {"enable_host_users": true, "enable_software_inventory": false, "additional_queries": null}}`)
		_, err := q.ExecContext(ctx,
			`INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`,
			preChangeJSON)
		return err
	})
	t.Cleanup(func() {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx,
				`INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`,
				previousRawConfig)
			return err
		})
	})

	loadedCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	require.True(t, loadedCfg.Features.HistoricalData.Uptime, "pre-change row reads back as default true")
	require.True(t, loadedCfg.Features.HistoricalData.Vulnerabilities, "pre-change row reads back as default true")
}

func (s *integrationTestSuite) TestUserRolesSpec() {
	t := s.T()

	_, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	email := t.Name() + "@asd.com"
	u := &fleet.User{
		Password:    []byte("asd"),
		Name:        t.Name(),
		Email:       email,
		GravatarURL: "http://asd.com",
		GlobalRole:  new(fleet.RoleObserver),
	}
	user, err := s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)
	assert.Len(t, user.Teams, 0)

	spec := []byte(fmt.Sprintf(`
  roles:
    %s:
      global_role: null
      teams:
      - role: maintainer
        team: team1
`,
		email))

	var userRoleSpec applyUserRoleSpecsRequest
	err = yaml.Unmarshal(spec, &userRoleSpec.Spec)
	require.NoError(t, err)

	s.Do("POST", "/api/latest/fleet/users/roles/spec", &userRoleSpec, http.StatusOK)

	user, err = s.ds.UserByEmail(context.Background(), email)
	require.NoError(t, err)
	require.Len(t, user.Teams, 1)
	assert.Equal(t, fleet.RoleMaintainer, user.Teams[0].Role)

	spec = []byte(fmt.Sprintf(`
  roles:
    %s:
      global_role: null
      teams:
      - role: maintainer
        team: non-existent
`,
		email))
	userRoleSpec = applyUserRoleSpecsRequest{}
	err = yaml.Unmarshal(spec, &userRoleSpec.Spec)
	require.NoError(t, err)
	s.Do("POST", "/api/latest/fleet/users/roles/spec", &userRoleSpec, http.StatusBadRequest)
}

func (s *integrationTestSuite) TestGlobalSchedule() {
	t := s.T()

	// list the existing global schedules (none yet)
	gs := fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 0)

	// create a query that can be scheduled
	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery1",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Saved:          true,
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	// schedule that query
	gsParams := fleet.ScheduledQueryPayload{QueryID: new(qr.ID), Interval: new(uint(42))}
	r := globalScheduleQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/schedule", gsParams, http.StatusOK, &r)

	// list the scheduled queries, get the one just created
	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)
	assert.Equal(t, uint(42), gs.GlobalSchedule[0].Interval)
	assert.Contains(t, gs.GlobalSchedule[0].Name, "Copy of TestQuery1 (")
	id := gs.GlobalSchedule[0].ID

	// list page 2, should be empty
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs, "page", "2", "per_page", "4")
	require.Len(t, gs.GlobalSchedule, 0)

	// update the scheduled query
	gs = fleet.GlobalSchedulePayload{}
	gsParams = fleet.ScheduledQueryPayload{Interval: new(uint(55))}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/schedule/%d", id), gsParams, http.StatusOK, &gs)

	// update a non-existing schedule
	gsParams = fleet.ScheduledQueryPayload{Interval: new(uint(66))}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/schedule/%d", id+1), gsParams, http.StatusNotFound, &gs)

	// read back that updated scheduled query
	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)
	assert.Equal(t, id, gs.GlobalSchedule[0].ID)
	assert.Equal(t, uint(55), gs.GlobalSchedule[0].Interval)

	// delete the scheduled query
	r = globalScheduleQueryResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/schedule/%d", id), nil, http.StatusOK, &r)

	// delete a non-existing schedule
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/schedule/%d", id+1), nil, http.StatusNotFound, &r)

	// list the scheduled queries, back to none
	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 0)
}

func (s *integrationTestSuite) TestTranslator() {
	t := s.T()

	payload := translatorResponse{}
	params := translatorRequest{List: []fleet.TranslatePayload{
		{
			Type:    fleet.TranslatorTypeUserEmail,
			Payload: fleet.StringIdentifierToIDPayload{Identifier: "admin1@example.com"},
		},
	}}
	s.DoJSON("POST", "/api/latest/fleet/translate", &params, http.StatusOK, &payload)
	require.Len(t, payload.List, 1)

	assert.Equal(t, s.users[payload.List[0].Payload.Identifier].ID, payload.List[0].Payload.ID)

	// empty body
	s.DoJSON("POST", "/api/latest/fleet/translate", &translatorRequest{}, http.StatusBadRequest, &payload)

	s.DoJSON("POST", "/api/latest/fleet/translate", &translatorRequest{List: []fleet.TranslatePayload{{Type: "notavalidtype", Payload: fleet.StringIdentifierToIDPayload{}}}}, http.StatusBadRequest, &payload)
}

func (s *integrationTestSuite) TestVulnerableSoftware() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OSVersion:       "Mac OS X 10.14.6",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions", ExtensionID: "abc", ExtensionFor: "edge"},
		{Name: "bar", Version: "0.0.3", Source: "apps", ExtensionID: "xyz", ExtensionFor: "chrome"},
		{Name: "baz", Version: "0.0.4", Source: "apps"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))

	soft1 := host.Software[0]
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	cpes := []fleet.SoftwareCPE{{SoftwareID: soft1.ID, CPE: "somecpe"}}
	_, err = s.ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	// Reload software so that 'GeneratedCPEID is set.
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))
	soft1 = host.Software[0]
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	inserted, err := s.ds.InsertSoftwareVulnerability(
		context.Background(), fleet.SoftwareVulnerability{
			SoftwareID: soft1.ID,
			CVE:        "cve-123-123-132",
		}, fleet.NVDSource,
	)
	require.NoError(t, err)
	require.True(t, inserted)

	var hostResponse getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResponse)

	assertSoftware := func(t *testing.T, software []fleet.HostSoftwareEntry, contains *fleet.Software) {
		t.Helper()
		var found bool
		for _, s := range software {
			if s.Name == contains.Name {
				found = true
				assert.Equal(t, s.Name, contains.Name)
				assert.Equal(t, s.Version, contains.Version)
				assert.Equal(t, s.Source, contains.Source)
				assert.Equal(t, s.ExtensionID, contains.ExtensionID)
				assert.Equal(t, s.ExtensionFor, contains.ExtensionFor)
				assert.Equal(t, s.GenerateCPE, contains.GenerateCPE)
				assert.Len(t, contains.Vulnerabilities, len(s.Vulnerabilities))
				for i, vuln := range s.Vulnerabilities {
					assert.Equal(t, vuln.CVE, contains.Vulnerabilities[i].CVE)
					assert.Equal(t, vuln.DetailsLink, contains.Vulnerabilities[i].DetailsLink)
				}
			}
		}
		if !found {
			t.Fatalf("software not found")
		}
	}

	expectedSoft2 := &fleet.Software{
		Name:         "bar",
		Version:      "0.0.3",
		Source:       "apps",
		ExtensionID:  "xyz",
		ExtensionFor: "chrome",
		GenerateCPE:  "somecpe",
		Vulnerabilities: fleet.Vulnerabilities{
			{
				CVE:         "cve-123-123-132",
				DetailsLink: "https://nvd.nist.gov/vuln/detail/cve-123-123-132",
			},
		},
	}

	expectedSoft1 := &fleet.Software{
		Name:            "foo",
		Version:         "0.0.1",
		Source:          "chrome_extensions",
		ExtensionID:     "abc",
		ExtensionFor:    "edge",
		GenerateCPE:     "",
		Vulnerabilities: nil,
	}

	assertSoftware(t, hostResponse.Host.Software, expectedSoft1)
	assertSoftware(t, hostResponse.Host.Software, expectedSoft2)

	// no software host counts have been calculated yet, so this returns nothing
	var lsResp listSoftwareResponse
	resp := s.Do("GET", "/api/latest/fleet/software", nil, http.StatusOK, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(bodyBytes), `"counts_updated_at": null`)
	require.NoError(t, json.Unmarshal(bodyBytes, &lsResp))
	require.Len(t, lsResp.Software, 0)
	assert.Nil(t, lsResp.CountsUpdatedAt)

	var versionsResp listSoftwareVersionsResponse
	resp = s.Do("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	bodyBytes, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(bodyBytes), `"counts_updated_at": null`)
	require.NoError(t, json.Unmarshal(bodyBytes, &versionsResp))
	require.Len(t, versionsResp.Software, 0)
	require.Equal(t, 0, versionsResp.Count)
	assert.Nil(t, versionsResp.CountsUpdatedAt)

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))

	countReq := countSoftwareRequest{}
	countResp := countSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/count", countReq, http.StatusOK, &countResp)
	assert.Equal(t, 3, countResp.Count)

	// the software/count endpoint is different, it doesn't care about hosts counts
	countResp = countSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/count", countReq, http.StatusOK, &countResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	assert.Equal(t, 1, countResp.Count)

	// now the list software endpoint returns the software
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	require.Len(t, lsResp.Software, 1)
	assert.Equal(t, soft1.ID, lsResp.Software[0].ID)
	assert.Equal(t, soft1.ExtensionID, lsResp.Software[0].ExtensionID)
	assert.Equal(t, soft1.ExtensionFor, lsResp.Software[0].ExtensionFor)
	// Browser field should be populated for browser extensions
	if soft1.Source == "chrome_extensions" || soft1.Source == "firefox_addons" || soft1.Source == "ie_extensions" || soft1.Source == "safari_extensions" {
		assert.Equal(t, soft1.ExtensionFor, lsResp.Software[0].Browser)
	} else {
		assert.Equal(t, "", lsResp.Software[0].Browser)
	}
	assert.Len(t, lsResp.Software[0].Vulnerabilities, 1)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	require.Len(t, versionsResp.Software, 1)
	require.Equal(t, 1, versionsResp.Count)
	assert.Equal(t, soft1.ID, versionsResp.Software[0].ID)
	assert.Equal(t, soft1.ExtensionID, versionsResp.Software[0].ExtensionID)
	assert.Equal(t, soft1.ExtensionFor, versionsResp.Software[0].ExtensionFor)
	// Browser field should be populated for browser extensions
	if soft1.Source == "chrome_extensions" || soft1.Source == "firefox_addons" || soft1.Source == "ie_extensions" || soft1.Source == "safari_extensions" {
		assert.Equal(t, soft1.ExtensionFor, versionsResp.Software[0].Browser)
	} else {
		assert.Equal(t, "", versionsResp.Software[0].Browser)
	}
	assert.Len(t, versionsResp.Software[0].Vulnerabilities, 1)
	require.NotNil(t, versionsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *versionsResp.CountsUpdatedAt, time.Second)

	// the count endpoint still returns 1
	countResp = countSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/count", countReq, http.StatusOK, &countResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	assert.Equal(t, 1, countResp.Count)

	// an unsupported order_key returns 422
	s.Do("GET", "/api/latest/fleet/software", nil, http.StatusUnprocessableEntity, "order_key", "vendor_old")
	s.Do("GET", "/api/latest/fleet/software/versions", nil, http.StatusUnprocessableEntity, "order_key", "vendor_old")

	// default sort, not only vulnerable
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp)
	require.True(t, len(lsResp.Software) >= len(software))
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp)
	require.True(t, len(versionsResp.Software) >= len(software))
	require.True(t, versionsResp.Count >= len(software))
	require.NotNil(t, versionsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *versionsResp.CountsUpdatedAt, time.Second)

	// request with a per_page limit (see #4058)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "page", "0", "per_page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, lsResp.Software, 2)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "page", "0", "per_page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, versionsResp.Software, 2)
	require.True(t, versionsResp.Count >= 2)
	require.NotNil(t, versionsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *versionsResp.CountsUpdatedAt, time.Second)

	// request next page, with per_page limit
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, lsResp.Software, 1)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "per_page", "2", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, versionsResp.Software, 1)
	require.True(t, versionsResp.Count >= 2)
	require.NotNil(t, versionsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *versionsResp.CountsUpdatedAt, time.Second)

	// request one past the last page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, lsResp.Software, 0)
	require.Nil(t, lsResp.CountsUpdatedAt)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "per_page", "2", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, versionsResp.Software, 0)
	require.True(t, versionsResp.Count >= 2)
	require.Nil(t, versionsResp.CountsUpdatedAt) // CONFIRM: legacy counts updated at is calculated by the server based on the software entries in the paginated response so how should we handle now?

	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusBadRequest, &lsResp, "per_page", "2", "page", "-10")
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusBadRequest, &lsResp, "per_page", "-2", "page", "2")
	s.DoJSON("GET", "/api/latest/fleet/software/count", nil, http.StatusBadRequest, &lsResp, "per_page", "-2", "page", "2")
}

func (s *integrationTestSuite) TestGlobalPolicies() {
	t := s.T()

	// create 3 hosts
	for i := 0; i < 3; i++ {
		_, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   new(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         new(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
		})
		require.NoError(t, err)
	}

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery3",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	// create a global policy
	gpParams := fleet.GlobalPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some global resolution",
	}
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, qr.Name, gpResp.Policy.Name)
	assert.Equal(t, qr.Query, gpResp.Policy.Query)
	assert.Equal(t, qr.Description, gpResp.Policy.Description)
	require.NotNil(t, gpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution", *gpResp.Policy.Resolution)

	// list global policies
	policiesResponse := fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, qr.Name, policiesResponse.Policies[0].Name)
	assert.Equal(t, qr.Query, policiesResponse.Policies[0].Query)
	assert.Equal(t, qr.Description, policiesResponse.Policies[0].Description)

	// invalid order_key returns 422
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusUnprocessableEntity, &fleet.ListGlobalPoliciesResponse{}, "order_key", "invalid")

	// Get an unexistent policy
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", 9999), nil, http.StatusNotFound)

	singlePolicyResponse := fleet.GetPolicyByIDResponse{}
	singlePolicyURL := fmt.Sprintf("/api/latest/fleet/policies/%d", policiesResponse.Policies[0].ID)
	s.DoJSON("GET", singlePolicyURL, nil, http.StatusOK, &singlePolicyResponse)
	assert.Equal(t, qr.Name, singlePolicyResponse.Policy.Name)
	assert.Equal(t, qr.Query, singlePolicyResponse.Policy.Query)
	assert.Equal(t, qr.Description, singlePolicyResponse.Policy.Description)

	listHostsURL := fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d", policiesResponse.Policies[0].ID)
	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 3)

	h1 := listHostsResp.Hosts[0]
	h2 := listHostsResp.Hosts[1]

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: new(true)}, time.Now(), false, nil)))
	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false, nil)))

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	// count global policies
	cGPRes := fleet.CountGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &cGPRes)
	assert.Equal(t, 1, cGPRes.Count)

	// count global policies with matching search query
	cGPRes = fleet.CountGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &cGPRes, "query", "estQue")
	assert.Equal(t, 1, cGPRes.Count)

	// count global policies with matching search query containing leading/trailing whitespace
	cGPRes = fleet.CountGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &cGPRes, "query", " estQue    ")
	assert.Equal(t, 1, cGPRes.Count)

	// count global policies with non-matching search query
	cGPRes = fleet.CountGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &cGPRes, "query", "Query4")
	assert.Equal(t, 0, cGPRes.Count)

	// delete the policy
	deletePolicyParams := fleet.DeleteGlobalPoliciesRequest{IDs: []uint{policiesResponse.Policies[0].ID}}
	deletePolicyResp := fleet.DeleteGlobalPoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)

	policiesResponse = fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 0)
}

func (s *integrationTestSuite) createHosts(t *testing.T, platforms ...string) []*fleet.Host {
	var hosts []*fleet.Host
	if len(platforms) == 0 {
		platforms = []string{"debian", "rhel", "linux"}
	}
	for i, platform := range platforms {
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   new(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         new(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
			Platform:        platform,
		})
		require.NoError(t, err)
		hosts = append(hosts, host)
	}
	return hosts
}

func (s *integrationTestSuite) TestPacks() {
	t := s.T()

	var packResp getPackResponse
	// get non-existing pack
	s.Do("GET", "/api/latest/fleet/packs/999", nil, http.StatusNotFound)

	// create some packs
	packs := make([]fleet.Pack, 3)
	for i := range packs {
		req := &createPackRequest{
			PackPayload: fleet.PackPayload{
				Name: new(fmt.Sprintf("%s_%d", strings.ReplaceAll(t.Name(), "/", "_"), i)),
			},
		}

		var createResp createPackResponse
		s.DoJSON("POST", "/api/latest/fleet/packs", req, http.StatusOK, &createResp)
		packs[i] = createResp.Pack.Pack
	}

	// get existing pack
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d", packs[0].ID), nil, http.StatusOK, &packResp)
	require.Equal(t, packs[0].ID, packResp.Pack.ID)

	// list packs
	var listResp listPacksResponse
	s.DoJSON("GET", "/api/latest/fleet/packs", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 2)
	assert.Equal(t, packs[0].ID, listResp.Packs[0].ID)
	assert.Equal(t, packs[1].ID, listResp.Packs[1].ID)

	// get page 1
	s.DoJSON("GET", "/api/latest/fleet/packs", nil, http.StatusOK, &listResp, "page", "1", "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 1)
	assert.Equal(t, packs[2].ID, listResp.Packs[0].ID)

	// get page 2, empty
	s.DoJSON("GET", "/api/latest/fleet/packs", nil, http.StatusOK, &listResp, "page", "2", "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 0)

	var delResp deletePackResponse
	// delete non-existing pack by name
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/%s", "zzz"), nil, http.StatusNotFound, &delResp)

	// delete existing pack by name
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/%s", url.PathEscape(packs[0].Name)), nil, http.StatusOK, &delResp)

	// delete non-existing pack by id
	var delIDResp deletePackByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/id/%d", packs[2].ID+1), nil, http.StatusNotFound, &delIDResp)

	// delete existing pack by id
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/id/%d", packs[1].ID), nil, http.StatusOK, &delIDResp)

	var modResp modifyPackResponse
	// modify non-existing pack
	req := &fleet.PackPayload{Name: new("updated_" + packs[2].Name)}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/%d", packs[2].ID+1), req, http.StatusNotFound, &modResp)

	// modify existing pack
	req = &fleet.PackPayload{Name: new("updated_" + packs[2].Name)}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/%d", packs[2].ID), req, http.StatusOK, &modResp)
	require.Equal(t, packs[2].ID, modResp.Pack.ID)
	require.Contains(t, modResp.Pack.Name, "updated_")

	// list packs, only packs[2] remains
	s.DoJSON("GET", "/api/latest/fleet/packs", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 1)
	assert.Equal(t, packs[2].ID, listResp.Packs[0].ID)
}

func (s *integrationTestSuite) TestInvites() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name() + "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	// list invites, none yet
	var listResp listInvitesResponse
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Invites, 0)

	// create valid invite
	createInviteReq := createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      new("some email"),
		Name:       new("some name"),
		GlobalRole: null.StringFrom(fleet.RoleAdmin),
	}}
	createInviteResp := createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
	require.NotNil(t, createInviteResp.Invite)
	require.NotZero(t, createInviteResp.Invite.ID)
	validInvite := *createInviteResp.Invite

	// create user from valid invite - the token was not returned via the
	// response's json, must get it from the db
	inv, err := s.ds.Invite(context.Background(), validInvite.ID)
	require.NoError(t, err)
	validInviteToken := inv.Token

	// verify the token with valid invite
	var verifyInvResp verifyInviteResponse
	s.DoJSON("GET", "/api/latest/fleet/invites/"+validInviteToken, nil, http.StatusOK, &verifyInvResp)
	require.Equal(t, validInvite.ID, verifyInvResp.Invite.ID)

	// verify the token with an invalid invite
	s.DoJSON("GET", "/api/latest/fleet/invites/invalid", nil, http.StatusNotFound, &verifyInvResp)

	// create invite without an email
	createInviteReq = createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      nil,
		Name:       new("some other name"),
		GlobalRole: null.StringFrom(fleet.RoleObserver),
	}}
	createInviteResp = createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusUnprocessableEntity, &createInviteResp)

	// create invite for an existing user
	existingEmail := "admin1@example.com"
	createInviteReq = createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      new(existingEmail),
		Name:       new("some other name"),
		GlobalRole: null.StringFrom(fleet.RoleObserver),
	}}
	createInviteResp = createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusUnprocessableEntity, &createInviteResp)

	// create invite for an existing user with email ALL CAPS
	createInviteReq = createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      new(strings.ToUpper(existingEmail)),
		Name:       new("some other name"),
		GlobalRole: null.StringFrom(fleet.RoleObserver),
	}}
	createInviteResp = createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusUnprocessableEntity, &createInviteResp)

	// list invites, we have one now
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Invites, 1)
	require.Equal(t, validInvite.ID, listResp.Invites[0].ID)

	// invalid order_key returns 422
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusUnprocessableEntity, &listResp, "order_key", "invalid")

	// list invites filtered by search query with leading/trailing whitespace
	// matches name
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp, "query", " some name                     ")
	require.Len(t, listResp.Invites, 1)
	require.Equal(t, validInvite.ID, listResp.Invites[0].ID)

	// list invites filtered by search query with leading/trailing whitespace
	// matches email
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp, "query", " some email                     ")
	require.Len(t, listResp.Invites, 1)
	require.Equal(t, validInvite.ID, listResp.Invites[0].ID)

	// list invites filtered by search query with leading/trailing whitespace
	// matches nothing
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp, "query", " no match                     ")
	require.Len(t, listResp.Invites, 0)

	// list invites, next page is empty
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp, "page", "1", "per_page", "2")
	require.Len(t, listResp.Invites, 0)

	// update a non-existing invite
	updateInviteReq := updateInviteRequest{InvitePayload: fleet.InvitePayload{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: team.ID}, Role: fleet.RoleObserver},
		},
		MFAEnabled: new(true),
	}}
	updateInviteResp := updateInviteResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID+1), updateInviteReq, http.StatusNotFound, &updateInviteResp)

	// update the valid invite created earlier, make it an observer of a team
	updateInviteResp = updateInviteResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), updateInviteReq, http.StatusPaymentRequired, &updateInviteResp)
	updateInviteReq.MFAEnabled = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), updateInviteReq, http.StatusOK, &updateInviteResp)

	// update the valid invite: set an email that already exists for a user
	updateInviteReq = updateInviteRequest{
		InvitePayload: fleet.InvitePayload{
			Email: new(s.users["admin1@example.com"].Email),
			Teams: []fleet.UserTeam{
				{Team: fleet.Team{ID: team.ID}, Role: fleet.RoleObserver},
			},
		},
	}
	updateInviteResp = updateInviteResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), updateInviteReq, http.StatusConflict, &updateInviteResp)

	// update the valid invite: set an email that already exists for another invite
	createInviteReq = createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      new("some@other.email"),
		Name:       new("some name"),
		GlobalRole: null.StringFrom(fleet.RoleAdmin),
	}}
	createInviteResp = createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
	updateInviteReq = updateInviteRequest{
		InvitePayload: fleet.InvitePayload{
			Email: createInviteReq.Email,
			Teams: []fleet.UserTeam{
				{Team: fleet.Team{ID: team.ID}, Role: fleet.RoleObserver},
			},
		},
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), updateInviteReq, http.StatusConflict, &updateInviteResp)

	// update the valid invite to an email that is ok
	updateInviteReq = updateInviteRequest{
		InvitePayload: fleet.InvitePayload{
			Email: new("something@nonexistent.yet123"),
			Teams: []fleet.UserTeam{
				{Team: fleet.Team{ID: team.ID}, Role: fleet.RoleObserver},
			},
		},
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), updateInviteReq, http.StatusOK, &updateInviteResp)

	verify, err := s.ds.Invite(context.Background(), validInvite.ID)
	require.NoError(t, err)
	require.Equal(t, "", verify.GlobalRole.String)
	require.Len(t, verify.Teams, 1)
	assert.Equal(t, team.ID, verify.Teams[0].ID)

	// Try to create an user with an email different that the one associated with the invite
	var createFromInviteResp createUserResponse
	userPayload := fleet.UserPayload{
		Name:        new("Full Name"),
		Password:    new(test.GoodPassword),
		Email:       new("a@b.c"),
		InviteToken: new(validInviteToken),
	}
	s.DoJSON("POST", "/api/latest/fleet/users", userPayload, http.StatusUnprocessableEntity, &createFromInviteResp)

	// Adjust email and try again, this should be OK
	userPayload.Email = new(verify.Email)
	s.DoJSON("POST", "/api/latest/fleet/users", userPayload, http.StatusOK, &createFromInviteResp)

	// Check that user is associated with unique invite ID
	user, err := s.ds.UserByEmail(context.Background(), verify.Email)
	require.NoError(t, err)
	require.Equal(t, inv.ID, *user.InviteID)

	// keep the invite token from the other valid invite (before deleting it)
	inv, err = s.ds.Invite(context.Background(), createInviteResp.Invite.ID)
	require.NoError(t, err)
	deletedInviteToken := inv.Token

	// delete an existing invite
	var delResp deleteInviteResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/invites/%d", createInviteResp.Invite.ID), nil, http.StatusOK, &delResp)

	// list invites, is now empty
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Invites, 0)

	// delete a now non-existing invite
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), nil, http.StatusNotFound, &delResp)

	// create user from never used but deleted invite
	s.DoJSON("POST", "/api/latest/fleet/users", fleet.UserPayload{
		Name:        new("Full Name"),
		Password:    new(test.GoodPassword),
		Email:       new(inv.Email),
		InviteToken: new(deletedInviteToken),
	}, http.StatusNotFound, &createFromInviteResp)
}

func (s *integrationTestSuite) TestCrossOriginJSONSecurity() {
	t := s.T()

	// valid request with no Origin or Referer headers
	createInviteReq := createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      new("some email"),
		Name:       new("some name"),
		GlobalRole: null.StringFrom(fleet.RoleAdmin),
	}}
	createInviteResp := createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
	require.NotNil(t, createInviteResp.Invite)
	require.NotZero(t, createInviteResp.Invite.ID)

	createInviteReq.Email = new("other@email.com")
	createInviteReq.Name = new("other name")
	req, err := json.Marshal(createInviteReq)
	require.NoError(t, err)

	// cross origin request with Origin header and no Content-Type
	resp := s.DoRawWithHeaders("POST", "/api/latest/fleet/invites", req, http.StatusUnsupportedMediaType, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", s.withServer.token),
		"Origin":        "example.com",
	})
	resp.Body.Close()

	// cross origin request with Referer header and no Content-Type
	resp = s.DoRawWithHeaders("POST", "/api/latest/fleet/invites", req, http.StatusUnsupportedMediaType, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", s.withServer.token),
		"Referer":       "example.com",
	})
	resp.Body.Close()

	// cross origin request with valid Content-Type
	resp = s.DoRawWithHeaders("POST", "/api/latest/fleet/invites", req, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", s.withServer.token),
		"Origin":        "example.com",
		"Referer":       "example.com",
		"Content-Type":  "application/json",
	})
	resp.Body.Close()
}

func (s *integrationTestSuite) TestCreateUserFromInviteErrors() {
	t := s.T()

	// create a valid invite
	createInviteReq := createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      new("a@b.c"),
		Name:       new("A"),
		GlobalRole: null.StringFrom(fleet.RoleObserver),
	}}
	createInviteResp := createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)

	// make sure to delete it on exit
	defer func() {
		var delResp deleteInviteResponse
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/invites/%d", createInviteResp.Invite.ID), nil, http.StatusOK, &delResp)
	}()

	// the token is not returned via the response's json, must get it from the db
	invite, err := s.ds.Invite(context.Background(), createInviteResp.Invite.ID)
	require.NoError(t, err)

	cases := []struct {
		desc string
		pld  fleet.UserPayload
		want int
	}{
		{
			"empty name",
			fleet.UserPayload{
				Name:        new(""),
				Password:    &test.GoodPassword,
				Email:       new("a@b.c"),
				InviteToken: new(invite.Token),
			},
			http.StatusUnprocessableEntity,
		},
		{
			"empty email",
			fleet.UserPayload{
				Name:        new("Name"),
				Password:    &test.GoodPassword,
				Email:       new(""),
				InviteToken: new(invite.Token),
			},
			http.StatusUnprocessableEntity,
		},
		{
			"empty password",
			fleet.UserPayload{
				Name:        new("Name"),
				Password:    new(""),
				Email:       new("a@b.c"),
				InviteToken: new(invite.Token),
			},
			http.StatusUnprocessableEntity,
		},
		{
			"empty token",
			fleet.UserPayload{
				Name:        new("Name"),
				Password:    &test.GoodPassword,
				Email:       new("a@b.c"),
				InviteToken: new(""),
			},
			http.StatusUnprocessableEntity,
		},
		{
			"invalid token",
			fleet.UserPayload{
				Name:        new("Name"),
				Password:    &test.GoodPassword,
				Email:       new("a@b.c"),
				InviteToken: new("invalid"),
			},
			http.StatusNotFound,
		},
		{
			"invalid password",
			fleet.UserPayload{
				Name:        new("Name"),
				Password:    new("password"), // no number or symbol
				Email:       new("a@b.c"),
				InviteToken: new(invite.Token),
			},
			http.StatusUnprocessableEntity,
		},
		{
			"api_endpoints not accepted",
			fleet.UserPayload{
				Name:         new("Name"),
				Password:     &test.GoodPassword,
				Email:        new("a@b.c"),
				InviteToken:  new(invite.Token),
				APIEndpoints: &[]fleet.APIEndpointRef{{Method: "GET", Path: "/api/v1/fleet/config"}},
			},
			http.StatusUnprocessableEntity,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			var resp createUserResponse
			s.DoJSON("POST", "/api/latest/fleet/users", c.pld, c.want, &resp)
		})
	}
}

func (s *integrationTestSuite) TestCreateUserFromSSOInvite() {
	t := s.T()
	ctx := context.Background()

	createInvite := func(email string, ssoEnabled bool) *fleet.Invite {
		createInviteReq := createInviteRequest{InvitePayload: fleet.InvitePayload{
			Email:      new(email),
			Name:       new("SSO Invitee"),
			GlobalRole: null.StringFrom(fleet.RoleObserver),
			SSOEnabled: new(ssoEnabled),
		}}
		createInviteResp := createInviteResponse{}
		s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
		t.Cleanup(func() {
			// Ignore the error: an accepted invite is consumed (deleted) by the
			// acceptance flow, so it may no longer exist at cleanup time.
			_ = s.ds.DeleteInvite(ctx, createInviteResp.Invite.ID)
		})
		// the token is not returned via the response's json, must get it from the db
		invite, err := s.ds.Invite(ctx, createInviteResp.Invite.ID)
		require.NoError(t, err)
		return invite
	}

	// An SSO-only invite must not be acceptable through the password flow.
	t.Run("sso invite rejects password payload", func(t *testing.T) {
		email := "sso-password-attack@b.c"
		invite := createInvite(email, true)

		var resp createUserResponse
		s.DoJSON("POST", "/api/latest/fleet/users", fleet.UserPayload{
			Name:        new("Attacker"),
			Email:       new(email),
			Password:    &test.GoodPassword,
			InviteToken: new(invite.Token),
		}, http.StatusUnprocessableEntity, &resp)

		// no user should have been created
		_, err := s.ds.UserByEmail(ctx, email)
		require.True(t, fleet.IsNotFound(err), "expected no user to be created, got err: %v", err)
	})

	// An empty password field on an SSO invite must also be rejected: the
	// presence of the field at all is not allowed.
	t.Run("sso invite rejects empty password field", func(t *testing.T) {
		email := "sso-empty-password@b.c"
		invite := createInvite(email, true)

		var resp createUserResponse
		s.DoJSON("POST", "/api/latest/fleet/users", fleet.UserPayload{
			Name:        new("Attacker"),
			Email:       new(email),
			Password:    new(""),
			SSOInvite:   new(true),
			InviteToken: new(invite.Token),
		}, http.StatusUnprocessableEntity, &resp)

		_, err := s.ds.UserByEmail(ctx, email)
		require.True(t, fleet.IsNotFound(err), "expected no user to be created, got err: %v", err)
	})

	// The legitimate SSO acceptance flow must create an SSO-enabled user.
	t.Run("sso invite accepted via sso flow", func(t *testing.T) {
		email := "sso-legit@b.c"
		invite := createInvite(email, true)

		var resp createUserResponse
		s.DoJSON("POST", "/api/latest/fleet/users", fleet.UserPayload{
			Name:        new("SSO User"),
			Email:       new(email),
			SSOInvite:   new(true),
			InviteToken: new(invite.Token),
		}, http.StatusOK, &resp)
		require.NotNil(t, resp.User)
		require.True(t, resp.User.SSOEnabled)
		t.Cleanup(func() { require.NoError(t, s.ds.DeleteUser(ctx, resp.User.ID)) })
	})

	// A non-SSO invite accepted with SSO flags but no password must be rejected
	// (and must not panic on a nil password).
	t.Run("password invite rejects sso payload without password", func(t *testing.T) {
		email := "password-as-sso@b.c"
		invite := createInvite(email, false)

		var resp createUserResponse
		s.DoJSON("POST", "/api/latest/fleet/users", fleet.UserPayload{
			Name:        new("No Password"),
			Email:       new(email),
			SSOInvite:   new(true),
			InviteToken: new(invite.Token),
		}, http.StatusUnprocessableEntity, &resp)

		_, err := s.ds.UserByEmail(ctx, email)
		require.True(t, fleet.IsNotFound(err), "expected no user to be created, got err: %v", err)
	})

	// A non-SSO invite accepted with SSOInvite falsely set must still enforce
	// password complexity: setting the SSO flag must not let a weak password
	// slip past validation.
	t.Run("password invite rejects sso payload with weak password", func(t *testing.T) {
		email := "password-as-sso-weak@b.c"
		invite := createInvite(email, false)

		var resp createUserResponse
		s.DoJSON("POST", "/api/latest/fleet/users", fleet.UserPayload{
			Name:        new("Weak Password"),
			Email:       new(email),
			Password:    new("weak"), // too short, no number or symbol
			SSOInvite:   new(true),
			InviteToken: new(invite.Token),
		}, http.StatusUnprocessableEntity, &resp)

		_, err := s.ds.UserByEmail(ctx, email)
		require.True(t, fleet.IsNotFound(err), "expected no user to be created, got err: %v", err)
	})
}

func (s *integrationTestSuite) TestGlobalPoliciesProprietary() {
	t := s.T()

	for i := 0; i < 3; i++ {
		_, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   new(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         new(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
			Platform:        "darwin",
		})
		require.NoError(t, err)
	}

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery321",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	// Cannot set both QueryID and Query.
	gpParams0 := fleet.GlobalPolicyRequest{
		QueryID: &qr.ID,
		Query:   "select * from osquery;",
	}
	gpResp0 := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams0, http.StatusBadRequest, &gpResp0)
	require.Nil(t, gpResp0.Policy)

	gpParams := fleet.GlobalPolicyRequest{
		Name:        "TestQuery3",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some global resolution",
		Platform:    "darwin",
	}
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NotEmpty(t, gpResp.Policy.ID)
	assert.Equal(t, "TestQuery3", gpResp.Policy.Name)
	assert.Equal(t, "select * from osquery;", gpResp.Policy.Query)
	assert.Equal(t, "Some description", gpResp.Policy.Description)
	require.NotNil(t, gpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution", *gpResp.Policy.Resolution)
	assert.NotNil(t, gpResp.Policy.AuthorID)
	assert.Equal(t, "Test Name admin1@example.com", gpResp.Policy.AuthorName)
	assert.Equal(t, "admin1@example.com", gpResp.Policy.AuthorEmail)
	assert.Equal(t, "darwin", gpResp.Policy.Platform)

	response := s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), []byte(`{
		"name": "TestQuery4",
		"query": "select * from osquery_info;",
		"description": "Some description updated",
		"resolution": "some global resolution updated"
	}`), http.StatusOK)
	var mgpResp fleet.ModifyGlobalPolicyResponse
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	err = json.Unmarshal(responseBody, &mgpResp)
	require.NoError(t, err)

	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, "TestQuery4", mgpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", mgpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mgpResp.Policy.Description)
	require.NotNil(t, mgpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *mgpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mgpResp.Policy.Platform)
	assert.Equal(t, uint(0), mgpResp.Policy.FailingHostCount)
	assert.Equal(t, uint(0), mgpResp.Policy.PassingHostCount)

	ggpResp := fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), fleet.GetPolicyByIDRequest{}, http.StatusOK, &ggpResp)
	require.NotNil(t, ggpResp.Policy)
	assert.Equal(t, "TestQuery4", ggpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", ggpResp.Policy.Query)
	assert.Equal(t, "Some description updated", ggpResp.Policy.Description)
	require.NotNil(t, ggpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *ggpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mgpResp.Policy.Platform)
	assert.Equal(t, uint(0), mgpResp.Policy.FailingHostCount)
	assert.Equal(t, uint(0), mgpResp.Policy.PassingHostCount)

	policiesResponse := fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "TestQuery4", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery_info;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some global resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].FailingHostCount)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].PassingHostCount)

	listHostsURL := fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d", policiesResponse.Policies[0].ID)
	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 3)
	h1 := listHostsResp.Hosts[0]
	h2 := listHostsResp.Hosts[1]

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=failing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: new(true)}, time.Now(), false, nil)))
	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false, nil)))

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	response = s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), []byte(`{
		"query": "select * from users;"
	}`), http.StatusOK)
	responseBody, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	err = json.Unmarshal(responseBody, &mgpResp)
	require.NoError(t, err)

	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, "TestQuery4", mgpResp.Policy.Name)
	assert.Equal(t, "select * from users;", mgpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mgpResp.Policy.Description)
	require.NotNil(t, mgpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *mgpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mgpResp.Policy.Platform)
	assert.Equal(t, uint(0), mgpResp.Policy.FailingHostCount)
	assert.Equal(t, uint(0), mgpResp.Policy.PassingHostCount)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=failing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	policiesResponse = fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "TestQuery4", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from users;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some global resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].FailingHostCount)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].PassingHostCount)

	// Record query executions
	require.NoError(
		t, errOnly(s.ds.RecordPolicyQueryExecutions(
			context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: new(true)}, time.Now(), false, nil,
		)),
	)
	require.NoError(
		t, errOnly(s.ds.RecordPolicyQueryExecutions(
			context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false, nil,
		)),
	)
	// Update policy stats
	require.NoError(t, s.ds.UpdateHostPolicyCounts(context.Background()))

	// Fetch policy to make sure stats are updated
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].FailingHostCount)
	assert.Equal(t, uint(1), policiesResponse.Policies[0].PassingHostCount)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	// Modify the platform for the policy, which should clear the policy stats
	response = s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), []byte(`{
		"platform": "linux"
	}`), http.StatusOK)
	responseBody, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	err = json.Unmarshal(responseBody, &mgpResp)
	require.NoError(t, err)

	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, "TestQuery4", mgpResp.Policy.Name)
	assert.Equal(t, "select * from users;", mgpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mgpResp.Policy.Description)
	require.NotNil(t, mgpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *mgpResp.Policy.Resolution)
	assert.Equal(t, "linux", mgpResp.Policy.Platform)
	assert.Equal(t, uint(0), mgpResp.Policy.FailingHostCount)
	assert.Equal(t, uint(0), mgpResp.Policy.PassingHostCount)

	// Fetch policy to make sure stats are updated
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].FailingHostCount)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].PassingHostCount)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=failing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	deletePolicyParams := fleet.DeleteGlobalPoliciesRequest{IDs: []uint{policiesResponse.Policies[0].ID}}
	deletePolicyResp := fleet.DeleteGlobalPoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)

	policiesResponse = fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 0)
}

func (s *integrationTestSuite) TestTeamPoliciesProprietary() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1-policies",
		Description: "desc team1",
	})
	require.NoError(t, err)
	hosts := make([]uint, 2)
	for i := 0; i < 2; i++ {
		h, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   new(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         new(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
			Platform:        "darwin",
		})
		require.NoError(t, err)
		hosts[i] = h.ID
	}
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team1.ID, hosts))
	require.NoError(t, err)

	tpName := "TestPolicy3"
	tpParams := fleet.TeamPolicyRequest{
		Name:        tpName,
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
		Platform:    "darwin",
	}
	tpResp := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	require.NotEmpty(t, tpResp.Policy.ID)
	assert.Equal(t, tpName, tpResp.Policy.Name)
	assert.Equal(t, "select * from osquery;", tpResp.Policy.Query)
	assert.Equal(t, "Some description", tpResp.Policy.Description)
	require.NotNil(t, tpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution", *tpResp.Policy.Resolution)
	assert.NotNil(t, tpResp.Policy.AuthorID)
	assert.Equal(t, "Test Name admin1@example.com", tpResp.Policy.AuthorName)
	assert.Equal(t, "admin1@example.com", tpResp.Policy.AuthorEmail)

	tpNameNew := "TestPolicy4"

	response := s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, tpResp.Policy.ID), []byte(fmt.Sprintf(`{
		"name": "%s",
		"query": "select * from osquery_info;",
		"description": "Some description updated",
		"resolution": "some team resolution updated"
	}`, tpNameNew)), http.StatusOK)
	var mtpResp fleet.ModifyGlobalPolicyResponse
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	err = json.Unmarshal(responseBody, &mtpResp)
	require.NoError(t, err)

	require.NotNil(t, mtpResp.Policy)
	assert.Equal(t, tpNameNew, mtpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", mtpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mtpResp.Policy.Description)
	require.NotNil(t, mtpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution updated", *mtpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mtpResp.Policy.Platform)

	gtpResp := fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, tpResp.Policy.ID), fleet.GetPolicyByIDRequest{}, http.StatusOK, &gtpResp)
	require.NotNil(t, gtpResp.Policy)
	assert.Equal(t, tpNameNew, gtpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", gtpResp.Policy.Query)
	assert.Equal(t, "Some description updated", gtpResp.Policy.Description)
	require.NotNil(t, gtpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution updated", *gtpResp.Policy.Resolution)
	assert.Equal(t, "darwin", gtpResp.Policy.Platform)

	policiesResponse := fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, tpNameNew, policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery_info;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some team resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)
	require.Len(t, policiesResponse.InheritedPolicies, 0)

	// test team policy count endpoint
	tpCountResp := fleet.CountTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tpCountResp)
	assert.Equal(t, 1, tpCountResp.Count)
	assert.Equal(t, 0, tpCountResp.InheritedPolicyCount)

	tpCountResp = fleet.CountTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tpCountResp, "query", tpNameNew)
	assert.Equal(t, 1, tpCountResp.Count)
	assert.Equal(t, 0, tpCountResp.InheritedPolicyCount)

	tpCountResp = fleet.CountTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tpCountResp, "query", " "+tpNameNew+" ")
	assert.Equal(t, 1, tpCountResp.Count)
	assert.Equal(t, 0, tpCountResp.InheritedPolicyCount)

	tpCountResp = fleet.CountTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tpCountResp, "query", " nomatch")
	assert.Equal(t, 0, tpCountResp.Count)
	assert.Equal(t, 0, tpCountResp.InheritedPolicyCount)

	listHostsURL := fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d", policiesResponse.Policies[0].ID)
	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 2)
	h1 := listHostsResp.Hosts[0]
	h2 := listHostsResp.Hosts[1]

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?team_id=%d&policy_id=%d&policy_response=passing", team1.ID, policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: new(true)}, time.Now(), false, nil)))
	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false, nil)))

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?team_id=%d&policy_id=%d&policy_response=passing", team1.ID, policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	deletePolicyParams := fleet.DeleteTeamPoliciesRequest{IDs: []uint{policiesResponse.Policies[0].ID}}
	deletePolicyResp := fleet.DeleteTeamPoliciesResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", team1.ID), deletePolicyParams, http.StatusOK, &deletePolicyResp)

	policiesResponse = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 0)
}

func (s *integrationTestSuite) TestTeamPoliciesProprietaryInvalid() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1-policies-2",
		Description: "desc team1",
	})
	require.NoError(t, err)

	tpParams := fleet.TeamPolicyRequest{
		Name:        "TestQuery3-Team",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
	}
	tpResp := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	teamPolicyID := tpResp.Policy.ID

	gpParams := fleet.GlobalPolicyRequest{
		Name:        "TestQuery3-Global",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some global resolution",
	}
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NotEmpty(t, gpResp.Policy.ID)
	globalPolicyID := gpResp.Policy.ID

	for _, tc := range []struct {
		tname      string
		testUpdate bool
		queryID    *uint
		name       string
		query      string
		platforms  string
	}{
		{
			tname:      "set both QueryID and Query",
			testUpdate: false,
			queryID:    new(uint(1)),
			name:       "Some name",
			query:      "select * from osquery;",
		},
		{
			tname:      "empty query",
			testUpdate: true,
			name:       "Some name",
			query:      "",
		},
		{
			tname:      "empty name",
			testUpdate: true,
			name:       "",
			query:      "select 1;",
		},
		{
			tname:      "empty with space",
			testUpdate: true,
			name:       " ", // #3704
			query:      "select 1;",
		},
		{
			tname:      "Invalid query",
			testUpdate: true,
			name:       "Invalid query",
			query:      "",
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			tpReq := fleet.TeamPolicyRequest{
				QueryID:  tc.queryID,
				Name:     tc.name,
				Query:    tc.query,
				Platform: tc.platforms,
			}
			tpResp := fleet.TeamPolicyResponse{}
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpReq, http.StatusBadRequest, &tpResp)
			require.Nil(t, tpResp.Policy)

			testUpdate := tc.queryID == nil

			if testUpdate {
				tpReq := fleet.ModifyTeamPolicyRequest{
					ModifyPolicyPayload: fleet.ModifyPolicyPayload{
						Name:  new(tc.name),
						Query: new(tc.query),
					},
				}
				tpResp := fleet.ModifyTeamPolicyResponse{}
				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, teamPolicyID), tpReq, http.StatusBadRequest, &tpResp)
				require.Nil(t, tpResp.Policy)
			}

			gpReq := fleet.GlobalPolicyRequest{
				QueryID:  tc.queryID,
				Name:     tc.name,
				Query:    tc.query,
				Platform: tc.platforms,
			}
			gpResp := fleet.GlobalPolicyResponse{}
			s.DoJSON("POST", "/api/latest/fleet/policies", gpReq, http.StatusBadRequest, &gpResp)
			require.Nil(t, tpResp.Policy)

			if testUpdate {
				gpReq := fleet.ModifyGlobalPolicyRequest{
					ModifyPolicyPayload: fleet.ModifyPolicyPayload{
						Name:  new(tc.name),
						Query: new(tc.query),
					},
				}
				gpResp := fleet.ModifyGlobalPolicyResponse{}
				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", globalPolicyID), gpReq, http.StatusBadRequest, &gpResp)
				require.Nil(t, tpResp.Policy)
			}
		})
	}
}

func (s *integrationTestSuite) TestHostDetailsPolicies() {
	t := s.T()

	hosts := s.createHosts(t)
	host1 := hosts[0]
	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "HostDetailsPolicies-Team",
		Description: "desc team1",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team1.ID, []uint{host1.ID}))
	require.NoError(t, err)

	gpParams := fleet.GlobalPolicyRequest{
		Name:        "HostDetailsPolicies",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some global resolution",
	}
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NotEmpty(t, gpResp.Policy.ID)

	tpParams := fleet.TeamPolicyRequest{
		Name:        "HostDetailsPolicies-Team",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
	}
	tpResp := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	require.NotEmpty(t, tpResp.Policy.ID)

	_, err = s.ds.RecordPolicyQueryExecutions(
		context.Background(),
		host1,
		map[uint]*bool{gpResp.Policy.ID: new(true)},
		time.Now(),
		false,
		nil,
	)
	require.NoError(t, err)

	resp := s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host1.ID), nil, http.StatusOK)
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var r struct {
		Host *fleet.HostDetailResponse `json:"host"`
		Err  error                     `json:"error,omitempty"`
	}
	err = json.Unmarshal(b, &r)
	require.NoError(t, err)
	require.Nil(t, r.Err)
	hd := r.Host.HostDetail
	policies := *hd.Policies
	require.Len(t, policies, 2)
	// Policies that did not run are listed before passing policies
	// TODO(JVE): verify that this passes once JK merges his code
	require.True(t, reflect.DeepEqual(tpResp.Policy.PolicyData, policies[0].PolicyData))
	require.Equal(t, policies[0].Response, "") // policy didn't "run"

	require.True(t, reflect.DeepEqual(gpResp.Policy.PolicyData, policies[1].PolicyData))
	require.Equal(t, policies[1].Response, "pass")

	// Try to create a global policy with an existing name.
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusConflict, &gpResp)
	// Try to create a team policy with an existing name.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusConflict, &tpResp)
}

func (s *integrationTestSuite) TestListActivities() {
	t := s.T()

	ctx := context.Background()
	u := s.users["admin1@example.com"]

	prevActivities := s.listActivities()

	activitySvc := mysqltest.NewTestActivityService(t, s.ds)
	apiUser := &activity_api.User{ID: u.ID, Name: u.Name, Email: u.Email}
	err := activitySvc.NewActivity(ctx, apiUser, fleet.ActivityTypeAppliedSpecPack{})
	require.NoError(t, err)

	err = activitySvc.NewActivity(ctx, apiUser, fleet.ActivityTypeDeletedPack{})
	require.NoError(t, err)

	err = activitySvc.NewActivity(ctx, apiUser, fleet.ActivityTypeEditedPack{})
	require.NoError(t, err)

	lenPage := len(prevActivities) + 2

	var listResp listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(lenPage), "order_key", "id")
	require.Len(t, listResp.Activities, lenPage)
	require.NotNil(t, listResp.Meta)
	assert.Equal(t, fleet.ActivityTypeAppliedSpecPack{}.ActivityName(), listResp.Activities[lenPage-2].Type)
	assert.Equal(t, fleet.ActivityTypeDeletedPack{}.ActivityName(), listResp.Activities[lenPage-1].Type)

	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(lenPage), "order_key", "id", "page", "1")
	require.Len(t, listResp.Activities, 1)
	assert.Equal(t, fleet.ActivityTypeEditedPack{}.ActivityName(), listResp.Activities[0].Type)

	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listResp, "per_page", "1", "order_key", "id", "order_direction", "desc")
	require.Len(t, listResp.Activities, 1)
	assert.Equal(t, fleet.ActivityTypeEditedPack{}.ActivityName(), listResp.Activities[0].Type)

	listResp = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listResp, "per_page", "1", "order_key", "id", "after", "0")
	require.Len(t, listResp.Activities, 1)
	require.Nil(t, listResp.Meta)
}

func (s *integrationTestSuite) TestListGetCarves() {
	t := s.T()

	ctx := context.Background()

	hosts := s.createHosts(t)
	c1, err := s.ds.NewCarve(ctx, &fleet.CarveMetadata{
		CreatedAt: time.Now(),
		HostId:    hosts[0].ID,
		Name:      t.Name() + "_1",
		SessionId: "ssn1",
	})
	require.NoError(t, err)
	c2, err := s.ds.NewCarve(ctx, &fleet.CarveMetadata{
		CreatedAt: time.Now(),
		HostId:    hosts[1].ID,
		Name:      t.Name() + "_2",
		SessionId: "ssn2",
	})
	require.NoError(t, err)
	c3, err := s.ds.NewCarve(ctx, &fleet.CarveMetadata{
		CreatedAt: time.Now(),
		HostId:    hosts[2].ID,
		Name:      t.Name() + "_3",
		SessionId: "ssn3",
	})
	require.NoError(t, err)

	// set c1 max block
	c1.MaxBlock = 3
	require.NoError(t, s.ds.UpdateCarve(ctx, c1))
	// make c2 expired, set max block
	c2.Expired = true
	c2.MaxBlock = 3
	require.NoError(t, s.ds.UpdateCarve(ctx, c2))

	var listResp fleet.ListCarvesResponse
	s.DoJSON("GET", "/api/latest/fleet/carves", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "id")
	require.Len(t, listResp.Carves, 2)
	assert.Equal(t, c1.ID, listResp.Carves[0].ID)
	assert.Equal(t, c3.ID, listResp.Carves[1].ID)

	// with 'after' param
	s.DoJSON(
		"GET", "/api/latest/fleet/carves", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "id", "after",
		strconv.FormatInt(c1.ID, 10),
	)
	require.Len(t, listResp.Carves, 1)
	assert.Equal(t, c3.ID, listResp.Carves[0].ID)

	// include expired
	s.DoJSON("GET", "/api/latest/fleet/carves", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "id", "expired", "1")
	require.Len(t, listResp.Carves, 2)
	assert.Equal(t, c1.ID, listResp.Carves[0].ID)
	assert.Equal(t, c2.ID, listResp.Carves[1].ID)

	// empty page
	s.DoJSON("GET", "/api/latest/fleet/carves", nil, http.StatusOK, &listResp, "page", "3", "per_page", "2", "order_key", "id", "expired", "1")
	require.Len(t, listResp.Carves, 0)

	// get specific carve
	var getResp fleet.GetCarveResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d", c2.ID), nil, http.StatusOK, &getResp)
	require.Equal(t, c2.ID, getResp.Carve.ID)
	require.True(t, getResp.Carve.Expired)

	// get non-existing carve
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d", c3.ID+1), nil, http.StatusNotFound, &getResp)

	// get expired carve block
	var blkResp fleet.GetCarveBlockResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d/block/%d", c2.ID, 1), nil, http.StatusInternalServerError, &blkResp)

	// get valid carve block, but block not inserted yet
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d/block/%d", c1.ID, 1), nil, http.StatusNotFound, &blkResp)

	require.NoError(t, s.ds.NewBlock(ctx, c1, 1, []byte("block1")))
	require.NoError(t, s.ds.NewBlock(ctx, c1, 2, []byte("block2")))
	require.NoError(t, s.ds.NewBlock(ctx, c1, 3, []byte("block3")))

	// get valid carve block
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d/block/%d", c1.ID, 1), nil, http.StatusOK, &blkResp)
	require.Equal(t, "block1", string(blkResp.Data))
}

func (s *integrationTestSuite) TestScheduledQueries() {
	t := s.T()

	// create a pack
	var createPackResp createPackResponse
	reqPack := &createPackRequest{
		PackPayload: fleet.PackPayload{
			Name: new(strings.ReplaceAll(t.Name(), "/", "_")),
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/packs", reqPack, http.StatusOK, &createPackResp)
	pack := createPackResp.Pack.Pack

	// try a non existent query
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/queries/%d", 9999), nil, http.StatusNotFound)

	// list queries
	var listQryResp fleet.ListQueriesResponse
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp)
	assert.Len(t, listQryResp.Queries, 0)
	assert.Equal(t, 0, listQryResp.Count)
	assert.Equal(t, 0, listQryResp.InheritedQueryCount)

	// create a query
	sql := "select * from time;"
	var createQueryResp fleet.CreateQueryResponse
	reqQuery := &fleet.QueryPayload{
		Name:  new(strings.ReplaceAll(t.Name(), "/", "_")),
		Query: new(sql),
	}
	s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	query := createQueryResp.Query
	assert.Equal(t, query.Query, sql)
	assert.Equal(t, createQueryResp.Report.Query, sql)

	// listing returns that query
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp)
	require.Len(t, listQryResp.Queries, 1)
	assert.Equal(t, query.Name, listQryResp.Queries[0].Name)
	assert.Equal(t, 1, listQryResp.Count)
	assert.Equal(t, 0, listQryResp.InheritedQueryCount)

	// listing with matching name returns that query
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", query.Name)
	require.Len(t, listQryResp.Queries, 1)
	assert.Equal(t, query.Name, listQryResp.Queries[0].Name)

	// listing with matching name plus whitespace returns that query
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", "  "+query.Name+" ")
	require.Len(t, listQryResp.Queries, 1)
	assert.Equal(t, query.Name, listQryResp.Queries[0].Name)

	// listing with non-matching name returns nothing
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", "  nomatch")
	require.Len(t, listQryResp.Queries, 0)

	// Return that query by name
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries?query=%s", query.Name), nil, http.StatusOK, &listQryResp)
	require.Len(t, listQryResp.Queries, 1)
	assert.Equal(t, query.Name, listQryResp.Queries[0].Name)

	// next page returns nothing, count and meta are correct
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "per_page", "2", "page", "1")
	require.Len(t, listQryResp.Queries, 0)
	require.Equal(t, 1, listQryResp.Count)
	require.Equal(t, 0, listQryResp.InheritedQueryCount)
	require.True(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)

	// getting that query works
	var getQryResp fleet.GetQueryResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d", query.ID), nil, http.StatusOK, &getQryResp)
	assert.Equal(t, query.ID, getQryResp.Query.ID)
	assert.Equal(t, query.ID, getQryResp.Report.ID)
	assert.Equal(t, sql, getQryResp.Query.Query)
	assert.Equal(t, sql, getQryResp.Report.Query)

	// list scheduled queries in pack, none yet
	var getInPackResp fleet.GetScheduledQueriesInPackResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID), nil, http.StatusOK, &getInPackResp)
	assert.Len(t, getInPackResp.Scheduled, 0)

	// list scheduled queries in non-existing pack
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID+1), nil, http.StatusOK, &getInPackResp)
	assert.Len(t, getInPackResp.Scheduled, 0)

	// create scheduled query
	var createResp fleet.ScheduleQueryResponse
	reqSQ := &fleet.ScheduleQueryRequest{
		PackID:   pack.ID,
		QueryID:  query.ID,
		Interval: 1,
	}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", reqSQ, http.StatusOK, &createResp)
	sq1 := createResp.Scheduled.ScheduledQuery
	assert.NotZero(t, sq1.ID)
	assert.Equal(t, uint(1), sq1.Interval)

	// create scheduled query with invalid pack
	reqSQ = &fleet.ScheduleQueryRequest{
		PackID:   pack.ID + 1,
		QueryID:  query.ID,
		Interval: 2,
	}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", reqSQ, http.StatusUnprocessableEntity, &createResp)

	// create scheduled query with invalid query
	reqSQ = &fleet.ScheduleQueryRequest{
		PackID:   pack.ID,
		QueryID:  query.ID + 1,
		Interval: 3,
	}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", reqSQ, http.StatusNotFound, &createResp)

	// list scheduled queries in pack
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID), nil, http.StatusOK, &getInPackResp)
	require.Len(t, getInPackResp.Scheduled, 1)
	assert.Equal(t, sq1.ID, getInPackResp.Scheduled[0].ID)

	// list scheduled queries in pack, next page
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID), nil, http.StatusOK, &getInPackResp, "page", "1", "per_page", "2")
	require.Len(t, getInPackResp.Scheduled, 0)

	// get non-existing scheduled query
	var getResp fleet.GetScheduledQueryResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/schedule/%d", sq1.ID+1), nil, http.StatusNotFound, &getResp)

	// get existing scheduled query
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/schedule/%d", sq1.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, sq1.ID, getResp.Scheduled.ID)
	assert.Equal(t, sq1.Interval, getResp.Scheduled.Interval)

	// modify scheduled query
	var modResp fleet.ModifyScheduledQueryResponse
	reqMod := fleet.ScheduledQueryPayload{
		Interval: new(uint(4)),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sq1.ID), reqMod, http.StatusOK, &modResp)
	assert.Equal(t, sq1.ID, modResp.Scheduled.ID)
	assert.Equal(t, uint(4), modResp.Scheduled.Interval)

	// modify non-existing scheduled query
	reqMod = fleet.ScheduledQueryPayload{
		Interval: new(uint(5)),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sq1.ID+1), reqMod, http.StatusNotFound, &modResp)

	// delete non-existing scheduled query
	var delResp fleet.DeleteScheduledQueryResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sq1.ID+1), nil, http.StatusNotFound, &delResp)

	// delete existing scheduled query
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sq1.ID), nil, http.StatusOK, &delResp)

	// get the now-deleted scheduled query
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/schedule/%d", sq1.ID), nil, http.StatusNotFound, &getResp)

	// modify the query
	var modQryResp fleet.ModifyQueryResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", query.ID), fleet.QueryPayload{Description: new("updated")}, http.StatusOK, &modQryResp)
	assert.Equal(t, "updated", modQryResp.Query.Description)
	assert.Equal(t, sql, modQryResp.Query.Query)
	assert.Equal(t, sql, modQryResp.Report.Query)

	// TODO(jahziel): check that the query results were deleted

	// TODO(jahziel): check that the query results were deleted after setting `discard_data`

	// modify a non-existing query
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", query.ID+1), fleet.QueryPayload{Description: new("updated")}, http.StatusNotFound, &modQryResp)

	// delete the query by name
	var delByNameResp fleet.DeleteQueryResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/%s", query.Name), nil, http.StatusOK, &delByNameResp)

	// delete unknown query by name (i.e. the same, now deleted)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/%s", query.Name), nil, http.StatusNotFound, &delByNameResp)

	// create another query
	reqQuery = &fleet.QueryPayload{
		Name:  new(strings.ReplaceAll(t.Name(), "/", "_") + "_2"),
		Query: new("select 2"),
	}
	s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	query2 := createQueryResp.Query

	// delete it by id
	var delByIDResp fleet.DeleteQueryByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", query2.ID), nil, http.StatusOK, &delByIDResp)

	// delete unknown query by id (same id just deleted)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", query2.ID), nil, http.StatusNotFound, &delByIDResp)

	// create another query
	reqQuery = &fleet.QueryPayload{
		Name:  new(strings.ReplaceAll(t.Name(), "/", "_") + "_3"),
		Query: new("select 3"),
	}
	s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	query3 := createQueryResp.Query

	// batch-delete by id, 3 ids, only one exists
	var delBatchResp fleet.DeleteQueriesResponse
	s.DoJSON("POST", "/api/latest/fleet/queries/delete", map[string]interface{}{
		"ids": []uint{query.ID, query2.ID, query3.ID},
	}, http.StatusOK, &delBatchResp)
	assert.Equal(t, uint(1), delBatchResp.Deleted)

	// batch-delete by id, none exist
	delBatchResp.Deleted = 0
	s.DoJSON("POST", "/api/latest/fleet/queries/delete", map[string]interface{}{
		"ids": []uint{query.ID, query2.ID, query3.ID},
	}, http.StatusNotFound, &delBatchResp)
	assert.Equal(t, uint(0), delBatchResp.Deleted)
}

func (s *integrationTestSuite) TestScheduledQueriesInPackOrderKey() {
	t := s.T()

	// create a pack
	var createPackResp createPackResponse
	s.DoJSON("POST", "/api/latest/fleet/packs", &createPackRequest{
		PackPayload: fleet.PackPayload{
			Name: new(strings.ReplaceAll(t.Name(), "/", "_")),
		},
	}, http.StatusOK, &createPackResp)
	pack := createPackResp.Pack.Pack

	// create a query
	var createQueryResp fleet.CreateQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/queries", &fleet.QueryPayload{
		Name:  new(strings.ReplaceAll(t.Name(), "/", "_")),
		Query: new("select 1"),
	}, http.StatusOK, &createQueryResp)
	query := createQueryResp.Query

	// schedule the query in the pack so the listing has at least one row
	var createSchedResp fleet.ScheduleQueryResponse
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", &fleet.ScheduleQueryRequest{
		PackID:   pack.ID,
		QueryID:  query.ID,
		Interval: 60,
	}, http.StatusOK, &createSchedResp)

	// every key in scheduledQueriesAllowedOrderKeys must work end-to-end with cursor pagination.
	allowedOrderKeys := []string{
		"id",
		"pack_id",
		"name",
		"query_name",
		"description",
		"interval",
		"snapshot",
		"removed",
		"platform",
		"version",
		"shard",
		"denylist",
		"query",
		"query_id",
		"user_time_p50",
		"user_time_p95",
		"system_time_p50",
		"system_time_p95",
		"total_executions",
	}
	for _, orderKey := range allowedOrderKeys {
		t.Run(orderKey, func(t *testing.T) {
			var getInPackResp fleet.GetScheduledQueriesInPackResponse
			s.DoJSON(
				"GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID),
				nil, http.StatusOK, &getInPackResp,
				"order_key", orderKey,
				"after", "0",
			)
		})
	}
}

func (s *integrationTestSuite) TestScheduledQueriesInPackInvalidOrderKey() {
	t := s.T()

	// create a pack so the endpoint has a real id to operate on
	var createPackResp createPackResponse
	s.DoJSON("POST", "/api/latest/fleet/packs", &createPackRequest{
		PackPayload: fleet.PackPayload{
			Name: new(strings.ReplaceAll(t.Name(), "/", "_")),
		},
	}, http.StatusOK, &createPackResp)
	pack := createPackResp.Pack.Pack

	var getInPackResp fleet.GetScheduledQueriesInPackResponse
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID),
		nil, http.StatusUnprocessableEntity, &getInPackResp,
		"order_key", "not_a_real_column",
		"after", "0",
	)
}

func (s *integrationTestSuite) TestQueriesPaginationAndPlatformFilter() {
	t := s.T()

	// make a few queries
	testQueries := []*fleet.Query{
		{Name: "PPTestQuery1", Query: "select 1", Platform: "darwin"},
		{Name: "PPTestQuery2", Query: "select 2", Platform: "linux"},
		{Name: "PPTestQuery3", Query: "select 3", Platform: "windows"},
		{Name: "PPTestQuery4", Query: "select 4", Platform: "darwin,windows,linux"},
		{Name: "PPTestQuery5", Query: "select 5"},
		{Name: "PPTestQuery6", Query: "select 6"},
		{Name: "PPTestQuery7", Query: "select 7"},
		{Name: "PPTestQuery8", Query: "select 8"},
		{Name: "PPTestQuery9", Query: "select 9"},
		{Name: "PPTestQuery10", Query: "select 10"},
	}
	var createQueryResp fleet.CreateQueryResponse
	for _, q := range testQueries {
		reqQuery := &fleet.QueryPayload{
			Name:     &q.Name,
			Query:    &q.Query,
			Platform: &q.Platform,
		}
		s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
		require.Equal(t, createQueryResp.Query.Name, q.Name)
		require.Equal(t, createQueryResp.Query.Platform, q.Platform)
	}

	var listQryResp fleet.ListQueriesResponse
	queryNameToMatch := "TestQuery"

	// Test pagination, no filter

	// middle of the pages
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", queryNameToMatch, "per_page", "2", "page", "1")
	require.Len(t, listQryResp.Queries, 2)
	require.Equal(t, listQryResp.Count, 10)
	require.True(t, listQryResp.Meta.HasPreviousResults)
	require.True(t, listQryResp.Meta.HasNextResults)

	// first and only page
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", queryNameToMatch, "per_page", "10", "page", "0")
	require.Len(t, listQryResp.Queries, 10)
	require.Equal(t, listQryResp.Count, 10)
	require.False(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)

	// first of a few pages
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", queryNameToMatch, "per_page", "2", "page", "0")
	require.Len(t, listQryResp.Queries, 2)
	require.Equal(t, listQryResp.Count, 10)
	require.False(t, listQryResp.Meta.HasPreviousResults)
	require.True(t, listQryResp.Meta.HasNextResults)

	// last page
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", queryNameToMatch, "per_page", "5", "page", "1")
	require.Len(t, listQryResp.Queries, 5)
	require.Equal(t, listQryResp.Count, 10)
	require.True(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)

	// after last page
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", queryNameToMatch, "per_page", "2", "page", "5")
	require.Len(t, listQryResp.Queries, 0)
	require.Equal(t, listQryResp.Count, 10)
	require.True(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)

	// invalid order_key returns 422
	listQryResp = fleet.ListQueriesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusUnprocessableEntity, &listQryResp, "order_key", "invalid")

	// test platform filtering

	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "platform", "macos")
	require.Len(t, listQryResp.Queries, 8)
	require.Equal(t, listQryResp.Count, 8)
	require.False(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)
	require.Equal(t, "darwin", listQryResp.Queries[0].Platform)
	require.Equal(t, "darwin,windows,linux", listQryResp.Queries[1].Platform)

	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "platform", "windows")
	require.Len(t, listQryResp.Queries, 8)
	require.Equal(t, listQryResp.Count, 8)
	require.False(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)
	require.Equal(t, "windows", listQryResp.Queries[0].Platform)
	require.Equal(t, "darwin,windows,linux", listQryResp.Queries[1].Platform)

	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "platform", "linux")
	require.Len(t, listQryResp.Queries, 8)
	require.Equal(t, listQryResp.Count, 8)
	require.False(t, listQryResp.Meta.HasPreviousResults)
	require.False(t, listQryResp.Meta.HasNextResults)
	require.Equal(t, "linux", listQryResp.Queries[0].Platform)
	require.Equal(t, "darwin,windows,linux", listQryResp.Queries[1].Platform)

	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "platform", "linux", "per_page", "1", "page", "0")
	require.Len(t, listQryResp.Queries, 1)
	require.Equal(t, listQryResp.Count, 8)
	require.False(t, listQryResp.Meta.HasPreviousResults)
	require.True(t, listQryResp.Meta.HasNextResults)
	require.Equal(t, "linux", listQryResp.Queries[0].Platform)

	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "platform", "linux", "per_page", "1", "page", "1")
	require.Len(t, listQryResp.Queries, 1)
	require.Equal(t, listQryResp.Count, 8)
	require.True(t, listQryResp.Meta.HasPreviousResults)
	require.True(t, listQryResp.Meta.HasNextResults)
	require.Equal(t, "darwin,windows,linux", listQryResp.Queries[0].Platform)

	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusBadRequest, &listQryResp, "platform", "lucas", "per_page", "1", "page", "1")

	// delete them by name
	var delByNameResp fleet.DeleteQueryResponse
	// for _, qId := range testQueryIds {
	for _, q := range testQueries {
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/%s", q.Name), nil, http.StatusOK, &delByNameResp)
	}
}

type macadminsDataResponse struct {
	Macadmins *struct {
		Munki       *fleet.HostMunkiInfo    `json:"munki"`
		MunkiIssues []*fleet.HostMunkiIssue `json:"munki_issues"`
		MDM         *struct {
			EnrollmentStatus string  `json:"enrollment_status"`
			ServerURL        string  `json:"server_url"`
			Name             *string `json:"name"`
			ID               *uint   `json:"id"`
		} `json:"mobile_device_management"`
	} `json:"macadmins"`
}

func (s *integrationTestSuite) TestLabels() {
	t := s.T()

	// create some hosts to use for manual labels
	hosts := s.createHosts(t, "debian", "linux", "fedora", "darwin", "darwin", "darwin", "darwin")
	manualHosts := hosts[:3]
	lbl2Hosts := hosts[3:6]

	t.Run("Manual and Dynamic Labels", func(t *testing.T) {
		// list labels, has the built-in ones
		builtinsMap := fleet.ReservedLabelNames()
		var listResp fleet.ListLabelsResponse
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp)
		assert.True(t, len(listResp.Labels) > 0)
		var builtinLbl fleet.Label
		for _, lbl := range listResp.Labels {
			_, ok := builtinsMap[lbl.Name]
			assert.True(t, ok)
			assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
			builtinLbl = lbl.Label
		}
		builtInsCount := len(listResp.Labels)
		require.Equal(t, builtInsCount, len(builtinsMap))

		// labels summary has the built-in ones
		var summaryResp fleet.GetLabelsSummaryResponse
		s.DoJSON("GET", "/api/latest/fleet/labels/summary", nil, http.StatusOK, &summaryResp)
		assert.Len(t, summaryResp.Labels, builtInsCount)
		for _, lbl := range summaryResp.Labels {
			_, ok := builtinsMap[lbl.Name]
			assert.True(t, ok)
			assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
		}

		// create a label without name, an error
		var createResp fleet.CreateLabelResponse
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Query: "select 1"}, http.StatusUnprocessableEntity, &createResp)

		// create a label with both a query and hosts, error
		res := s.Do("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: t.Name(), Query: "select 1", Hosts: []string{manualHosts[0].UUID}}, http.StatusUnprocessableEntity)
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, `Only one of "criteria", "query" or "hosts/host_ids" can be included in the request.`)

		// create invalid label, conflicts with builtin name (case-insensitive)
		for n := range builtinsMap {
			s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: n, Query: "select 1"}, http.StatusUnprocessableEntity, &createResp)
			s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: strings.ToLower(n), Query: "select 1"}, http.StatusUnprocessableEntity, &createResp)
			s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: strings.ToUpper(n), Query: "select 1"}, http.StatusUnprocessableEntity, &createResp)
		}

		// try to create a label with an invalid platform
		s.DoJSON(
			"POST",
			"/api/latest/fleet/labels",
			&fleet.LabelPayload{
				Name:     "amazing label",
				Query:    "select 1",
				Platform: "bados",
			},
			http.StatusUnprocessableEntity,
			&createResp,
		)

		// create a label with the generic "linux" platform (matches all Linux distros)
		s.DoJSON(
			"POST",
			"/api/latest/fleet/labels",
			&fleet.LabelPayload{
				Name:     "linux label",
				Query:    "select 1",
				Platform: "linux",
			},
			http.StatusOK,
			&createResp,
		)
		assert.NotZero(t, createResp.Label.ID)
		assert.Equal(t, "linux", createResp.Label.Platform)
		linuxLbl := createResp.Label.Label

		// create a valid dynamic label
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: t.Name(), Query: "select 1"}, http.StatusOK, &createResp)
		assert.NotZero(t, createResp.Label.ID)
		assert.Equal(t, t.Name(), createResp.Label.Name)
		assert.Empty(t, createResp.Label.HostIDs)
		assert.False(t, createResp.Label.CreatedAt.IsZero())
		assert.WithinDuration(t, time.Now(), createResp.Label.CreatedAt, time.Minute)
		assert.False(t, createResp.Label.UpdatedAt.IsZero())
		assert.WithinDuration(t, time.Now(), createResp.Label.UpdatedAt, time.Minute)
		lbl1 := createResp.Label.Label

		// try to create a manual label with the same name
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: lbl1.Name, Hosts: []string{manualHosts[0].UUID}}, http.StatusConflict, &createResp)
		// try to create a dynamic label with the same name
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: lbl1.Name, Query: "select 2"}, http.StatusConflict, &createResp)

		// get the label
		var getResp fleet.GetLabelResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), nil, http.StatusOK, &getResp)
		assert.Equal(t, lbl1.ID, getResp.Label.ID)
		assert.Empty(t, getResp.Label.HostIDs)

		// get a non-existing label
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID+1), nil, http.StatusNotFound, &getResp)

		// create a valid manual label
		createResp = fleet.CreateLabelResponse{}
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: t.Name() + "manual", Hosts: []string{manualHosts[0].UUID, manualHosts[1].Hostname, *manualHosts[2].NodeKey}}, http.StatusOK, &createResp)
		assert.NotZero(t, createResp.Label.ID)
		assert.Equal(t, t.Name()+"manual", createResp.Label.Name)
		assert.ElementsMatch(t, []uint{manualHosts[0].ID, manualHosts[1].ID, manualHosts[2].ID}, createResp.Label.HostIDs)
		manualLbl1 := createResp.Label.Label

		// get the label
		getResp = fleet.GetLabelResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl1.ID), nil, http.StatusOK, &getResp)
		assert.Equal(t, manualLbl1.ID, getResp.Label.ID)
		assert.Equal(t, fleet.LabelTypeRegular, getResp.Label.LabelType)
		assert.Equal(t, fleet.LabelMembershipTypeManual, getResp.Label.LabelMembershipType)
		assert.ElementsMatch(t, []uint{manualHosts[0].ID, manualHosts[1].ID, manualHosts[2].ID}, getResp.Label.HostIDs)
		assert.EqualValues(t, 3, getResp.Label.HostCount)

		// create a valid empty manual label
		createResp = fleet.CreateLabelResponse{}
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: strings.ReplaceAll(t.Name(), "/", "_") + "manual2"}, http.StatusOK, &createResp)
		assert.NotZero(t, createResp.Label.ID)
		assert.Equal(t, strings.ReplaceAll(t.Name(), "/", "_")+"manual2", createResp.Label.Name)
		assert.Empty(t, createResp.Label.HostIDs)
		manualLbl2 := createResp.Label.Label

		// try to create a manual label with the same name
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: manualLbl2.Name, Hosts: []string{manualHosts[0].UUID}}, http.StatusConflict, &createResp)
		// try to create a dynamic label with the same name
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: manualLbl2.Name, Query: "select 2"}, http.StatusConflict, &createResp)

		// get the label
		getResp = fleet.GetLabelResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl2.ID), nil, http.StatusOK, &getResp)
		assert.Equal(t, manualLbl2.ID, getResp.Label.ID)
		assert.Equal(t, fleet.LabelTypeRegular, getResp.Label.LabelType)
		assert.Equal(t, fleet.LabelMembershipTypeManual, getResp.Label.LabelMembershipType)
		assert.Empty(t, getResp.Label.HostIDs)
		assert.EqualValues(t, 0, getResp.Label.HostCount)

		// get a non-existing label
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", 9999), nil, http.StatusNotFound, &getResp)

		// modify dynamic label lbl1
		var modResp fleet.ModifyLabelResponse
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), &fleet.ModifyLabelPayload{Name: new(t.Name() + "zzz")}, http.StatusOK, &modResp)
		assert.Equal(t, lbl1.ID, modResp.Label.ID)
		assert.Empty(t, modResp.Label.HostIDs)
		assert.NotEqual(t, lbl1.Name, modResp.Label.Name)

		// attempt to modify a label to a reserved name
		for n := range builtinsMap {
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), &fleet.ModifyLabelPayload{Name: new(n)}, http.StatusUnprocessableEntity, &modResp)
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), &fleet.ModifyLabelPayload{Name: new(strings.ToLower(n))}, http.StatusUnprocessableEntity, &modResp)
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), &fleet.ModifyLabelPayload{Name: new(strings.ToUpper(n))}, http.StatusUnprocessableEntity, &modResp)
		}

		// modify a non-existing label
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", 9999), &fleet.ModifyLabelPayload{Name: new("zzz")}, http.StatusNotFound, &modResp)
		// modify a built-in label
		res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", builtinLbl.ID), &fleet.ModifyLabelPayload{Name: new("zzz")}, http.StatusUnprocessableEntity)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "cannot modify built-in label")

		// modify manual label 1 without modifying its hosts
		modResp = fleet.ModifyLabelResponse{}
		newName := "modified_manual_label1"
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl1.ID), &fleet.ModifyLabelPayload{Name: &newName}, http.StatusOK,
			&modResp)
		assert.Equal(t, manualLbl1.ID, modResp.Label.ID)
		assert.Equal(t, fleet.LabelTypeRegular, modResp.Label.LabelType)
		assert.Equal(t, fleet.LabelMembershipTypeManual, modResp.Label.LabelMembershipType)
		assert.ElementsMatch(t, []uint{manualHosts[0].ID, manualHosts[1].ID, manualHosts[2].ID}, modResp.Label.HostIDs)
		assert.EqualValues(t, 3, modResp.Label.HostCount)
		assert.Equal(t, newName, modResp.Label.Name)

		// add a host with the same name as another host to manual label 2, confirm only one host is added
		sameName, err := s.ds.NewHost(context.Background(), &fleet.Host{
			HardwareSerial: "ABCDE",
			Hostname:       manualHosts[0].Hostname,
			Platform:       "darwin",
		})
		require.NoError(t, err)

		modResp = fleet.ModifyLabelResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl2.ID),
			&fleet.ModifyLabelPayload{Hosts: []string{sameName.HardwareSerial}}, http.StatusOK, &modResp)
		assert.Len(t, modResp.Label.HostIDs, 1)
		assert.NotEqual(t, manualHosts[0].ID, modResp.Label.HostIDs[0])
		assert.Equal(t, manualLbl2.ID, modResp.Label.ID)
		assert.Equal(t, fleet.LabelTypeRegular, modResp.Label.LabelType)
		assert.Equal(t, fleet.LabelMembershipTypeManual, modResp.Label.LabelMembershipType)
		assert.ElementsMatch(t, []uint{sameName.ID}, modResp.Label.HostIDs)
		assert.EqualValues(t, 1, modResp.Label.HostCount)

		// modify manual label 2 adding some hosts
		modResp = fleet.ModifyLabelResponse{}
		newName = "modified_manual_label2"
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl2.ID),
			&fleet.ModifyLabelPayload{Name: &newName, Hosts: []string{manualHosts[0].UUID}}, http.StatusOK, &modResp)
		assert.Equal(t, manualLbl2.ID, modResp.Label.ID)
		assert.Equal(t, fleet.LabelTypeRegular, modResp.Label.LabelType)
		assert.Equal(t, fleet.LabelMembershipTypeManual, modResp.Label.LabelMembershipType)
		assert.ElementsMatch(t, []uint{manualHosts[0].ID}, modResp.Label.HostIDs)
		assert.EqualValues(t, 1, modResp.Label.HostCount)
		assert.Equal(t, newName, modResp.Label.Name)
		manualLbl2.Name = newName

		// modify manual label 2 adding some hosts by ID
		modResp = fleet.ModifyLabelResponse{}
		newName = "modified_manual_label2"
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl2.ID),
			&fleet.ModifyLabelPayload{Name: &newName, HostIDs: []uint{manualHosts[1].ID, manualHosts[2].ID}}, http.StatusOK, &modResp)
		assert.Equal(t, manualLbl2.ID, modResp.Label.ID)
		assert.Equal(t, fleet.LabelTypeRegular, modResp.Label.LabelType)
		assert.Equal(t, fleet.LabelMembershipTypeManual, modResp.Label.LabelMembershipType)
		assert.ElementsMatch(t, []uint{manualHosts[1].ID, manualHosts[2].ID}, modResp.Label.HostIDs)
		assert.EqualValues(t, 2, modResp.Label.HostCount)
		assert.Equal(t, newName, modResp.Label.Name)
		manualLbl2.Name = newName

		// modify manual label 2 clearing its hosts
		modResp = fleet.ModifyLabelResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl2.ID), &fleet.ModifyLabelPayload{Hosts: []string{}, Description: new("desc")}, http.StatusOK, &modResp)
		assert.Equal(t, manualLbl2.ID, modResp.Label.ID)
		assert.Equal(t, "desc", modResp.Label.Description)
		assert.Empty(t, modResp.Label.HostIDs)
		assert.EqualValues(t, 0, modResp.Label.HostCount)

		// list labels
		dynamicLabels := []fleet.Label{lbl1, linuxLbl}
		manualLabels := []fleet.Label{manualLbl1, manualLbl2}
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(100))
		assert.Len(t, listResp.Labels, builtInsCount+len(dynamicLabels)+len(manualLabels))

		// labels summary
		s.DoJSON("GET", "/api/latest/fleet/labels/summary", nil, http.StatusOK, &summaryResp)
		assert.Len(t, summaryResp.Labels, builtInsCount+len(dynamicLabels)+len(manualLabels))

		// next page is empty
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "per_page", "100", "page", "1")
		assert.Len(t, listResp.Labels, 0)

		// list labels with invalid query params
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusBadRequest, &listResp, "per_page", strconv.Itoa(builtInsCount+1), "order_key", "id", "after", "1")
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusBadRequest, &listResp, "per_page", strconv.Itoa(builtInsCount+1), "query", "no match query for this endpoint")

		// create another dynamic label
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: strings.ReplaceAll(t.Name(), "/", "_"), Query: "select 1"}, http.StatusOK, &createResp)
		assert.NotZero(t, createResp.Label.ID)
		lbl2 := createResp.Label.Label
		dynamicLabels = append(dynamicLabels, lbl2)
		require.Len(t, dynamicLabels, 3) // to make linter happy (dynamicLabels is not used past this point)

		// add lbl2 hosts to that label
		for _, h := range lbl2Hosts {
			err := s.ds.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{lbl2.ID: new(true)}, time.Now(), false)
			require.NoError(t, err)
		}

		// list hosts in dynamic label lbl2
		var listHostsResp listHostsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp)
		assert.Len(t, listHostsResp.Hosts, len(lbl2Hosts))

		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "order_key", "id", "after", fmt.Sprintf("%d", lbl2Hosts[0].ID))
		assert.Len(t, listHostsResp.Hosts, 2)
		assert.Equal(t, lbl2Hosts[1].ID, listHostsResp.Hosts[0].ID)
		assert.Equal(t, lbl2Hosts[2].ID, listHostsResp.Hosts[1].ID)

		// a dynamic label's membership cannot be replaced, and an empty list is a
		// replacement too (it would clear all of its members)
		for _, payload := range []fleet.ModifyLabelPayload{
			{HostIDs: []uint{}},
			{HostIDs: []uint{lbl2Hosts[0].ID}},
			{Hosts: []string{}},
			{Hosts: []string{lbl2Hosts[0].UUID}},
		} {
			res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl2.ID), &payload, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, `"hosts" or "host_ids" can only be provided for a manual label`)
		}

		listHostsResp = listHostsResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp)
		assert.Len(t, listHostsResp.Hosts, len(lbl2Hosts))

		// list hosts in manual label 1
		listHostsResp = listHostsResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", manualLbl1.ID), nil, http.StatusOK, &listHostsResp, "order_key", "id")
		assert.Len(t, listHostsResp.Hosts, manualLbl1.HostCount)
		assert.Equal(t, manualHosts[0].ID, listHostsResp.Hosts[0].ID)
		assert.Equal(t, manualHosts[1].ID, listHostsResp.Hosts[1].ID)
		assert.Equal(t, manualHosts[2].ID, listHostsResp.Hosts[2].ID)

		// list hosts in manual label 2
		listHostsResp = listHostsResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", manualLbl2.ID), nil, http.StatusOK, &listHostsResp, "order_key", "id")
		assert.Len(t, listHostsResp.Hosts, 0)

		// list hosts in dynamic label 2 searching by display_name
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "order_key", "display_name", "order_direction", "desc")
		assert.Len(t, listHostsResp.Hosts, len(lbl2Hosts))
		// first in the list is the last one, as the names are ordered with the index
		// of creation, and vice-versa
		assert.Equal(t, lbl2Hosts[len(lbl2Hosts)-1].ID, listHostsResp.Hosts[0].ID)
		assert.Equal(t, lbl2Hosts[0].ID, listHostsResp.Hosts[len(lbl2Hosts)-1].ID)

		mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(
				context.Background(),
				`INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
				lbl2Hosts[0].ID, "a@b.c", "src1",
			)

			return err
		})

		// list hosts in label searching by email address
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "query", "a@b.c")
		assert.Len(t, listHostsResp.Hosts, 1)
		assert.Equal(t, lbl2Hosts[0].ID, listHostsResp.Hosts[0].ID)

		// list hosts in label searching by email address with leading/trailing whitespace
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "query", "    a@b.c   ")
		assert.Len(t, listHostsResp.Hosts, 1)
		assert.Equal(t, lbl2Hosts[0].ID, listHostsResp.Hosts[0].ID)

		// count hosts in label order by display_name
		var countResp countHostsResponse
		s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "label_id", fmt.Sprint(lbl2.ID), "order_key", "display_name", "order_direction", "desc")
		assert.Equal(t, len(lbl2Hosts), countResp.Count)

		// lists hosts in label without hosts
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl1.ID), nil, http.StatusOK, &listHostsResp)
		assert.Len(t, listHostsResp.Hosts, 0)

		// count hosts in label
		s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "label_id", fmt.Sprint(lbl1.ID))
		assert.Equal(t, 0, countResp.Count)

		// lists hosts in invalid label
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID+1), nil, http.StatusOK, &listHostsResp)
		assert.Len(t, listHostsResp.Hosts, 0)

		// set MDM information on a host
		require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), lbl2Hosts[0].ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "", false))
		var mdmID uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &mdmID,
				`SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`, fleet.WellKnownMDMSimpleMDM, "https://simplemdm.com")
		})
		// generate aggregated stats
		require.NoError(t, s.ds.GenerateAggregatedMunkiAndMDM(context.Background()))

		// list host in label by mdm_id
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "mdm_id", fmt.Sprint(mdmID))
		require.Len(t, listHostsResp.Hosts, 1)
		assert.Nil(t, listHostsResp.Software)
		assert.Nil(t, listHostsResp.MunkiIssue)
		require.NotNil(t, listHostsResp.MDMSolution)
		assert.Equal(t, mdmID, listHostsResp.MDMSolution.ID)
		assert.Equal(t, fleet.WellKnownMDMSimpleMDM, listHostsResp.MDMSolution.Name)
		assert.Equal(t, "https://simplemdm.com", listHostsResp.MDMSolution.ServerURL)

		// invalid order_key returns 422 (sensitive columns must not be sortable to prevent
		// information disclosure via binary search extraction).
		for _, key := range []string{
			"node_key", "h.node_key",
			"orbit_node_key", "h.orbit_node_key",
			"invalid_column", "h.invalid_column",
			// computer_name is a valid field, but must not contain the table alias
			// (previous version of the endpoint allowed setting aliases here).
			"h.computer_name",
		} {
			res := s.Do("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusUnprocessableEntity, "order_key", key)
			errMsg := extractServerErrorText(res.Body)
			assert.Contains(t, errMsg, "invalid order_key")
			assert.Contains(t, errMsg, key)
		}

		// all allowed order_key values must be accepted by the endpoint and return the
		// hosts in lbl2.
		allowedOrderKeys := []string{
			"id", "osquery_host_id", "created_at", "updated_at", "detail_updated_at",
			"hostname", "uuid", "platform", "osquery_version", "os_version", "build",
			"platform_like", "code_name", "uptime", "memory", "cpu_type", "cpu_subtype",
			"cpu_brand", "cpu_physical_cores", "cpu_logical_cores",
			"hardware_vendor", "hardware_model", "hardware_version", "hardware_serial",
			"computer_name", "primary_ip_id", "distributed_interval", "logger_tls_period",
			"config_tls_refresh", "primary_ip", "primary_mac", "label_updated_at",
			"last_enrolled_at", "refetch_requested", "refetch_critical_queries_until",
			"team_id", "policy_updated_at", "public_ip",
			"gigs_disk_space_available", "percent_disk_space_available",
			"gigs_total_disk_space", "seen_time", "software_updated_at",
			"last_restarted_at", "timezone", "team_name",
			"failing_policies_count", "critical_vulnerabilities_count",
			"total_issues_count",
			"issues", // supported as alias for "total_issues_count"
			"device_mapping",
			"display_name",
		}
		for _, key := range allowedOrderKeys {
			listHostsResp = listHostsResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "order_key", key, "device_mapping", "true")
			assert.Len(t, listHostsResp.Hosts, len(lbl2Hosts), "order_key=%s", key)
		}

		// every allowed order_key must also accept the `after` cursor without
		// erroring — guards against SELECT-list aliases leaking into the WHERE
		// clause (MySQL disallows aliases in WHERE).
		for _, key := range allowedOrderKeys {
			listHostsResp = listHostsResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp,
				"order_key", key, "order_direction", "asc", "after", "0", "device_mapping", "true")
		}

		// issue-related order_keys are rejected when disable_issues=true.
		for _, key := range []string{"issues", "failing_policies_count", "critical_vulnerabilities_count", "total_issues_count"} {
			res := s.Do("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusBadRequest,
				"order_key", key, "disable_issues", "true")
			errMsg := extractServerErrorText(res.Body)
			assert.Contains(t, errMsg, "Invalid order_key")
			assert.Contains(t, errMsg, key)
		}

		// device_mapping order_key is rejected when device_mapping is not enabled.
		res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusBadRequest,
			"order_key", "device_mapping")
		errMsg = extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, "Invalid order_key")
		assert.Contains(t, errMsg, "device_mapping")

		// device_mapping=false is also rejected.
		res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusBadRequest,
			"order_key", "device_mapping", "device_mapping", "false")
		errMsg = extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, "Invalid order_key")
		assert.Contains(t, errMsg, "device_mapping")

		// delete a label by id
		var delIDResp fleet.DeleteLabelByIDResponse
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", linuxLbl.ID), nil, http.StatusOK, &delIDResp)
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", lbl1.ID), nil, http.StatusOK, &delIDResp)

		// delete a non-existing label by id
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", lbl2.ID+1), nil, http.StatusNotFound, &delIDResp)

		// delete a label by name
		var delResp fleet.DeleteLabelResponse
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(lbl2.Name)), nil, http.StatusOK, &delResp)

		// delete a non-existing label by name
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(lbl2.Name)), nil, http.StatusNotFound, &delResp)

		// delete a built-in label by name (case-insensitive)
		for n := range builtinsMap {
			for _, variant := range []string{n, strings.ToLower(n), strings.ToUpper(n)} {
				res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(variant)), nil, http.StatusUnprocessableEntity)
				errMsg = extractServerErrorText(res.Body)
				require.Contains(t, errMsg, "cannot delete built-in label")
			}
		}
		listResp = fleet.ListLabelsResponse{}
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp)
		var remainingBuiltIns int
		for _, lbl := range listResp.Labels {
			if _, ok := builtinsMap[lbl.Name]; ok {
				remainingBuiltIns++
			}
		}
		require.Equal(t, len(builtinsMap), remainingBuiltIns)

		// delete a manual label by id
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", manualLbl1.ID), nil, http.StatusOK, &delIDResp)

		// delete a manual label by name
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(manualLbl2.Name)), nil, http.StatusOK, &delResp)

		// list labels, only the built-ins remain
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(builtInsCount+1))
		assert.Len(t, listResp.Labels, builtInsCount)
		idsByName := make(map[string]uint, len(listResp.Labels))
		for _, lbl := range listResp.Labels {
			_, ok := builtinsMap[lbl.Name]
			assert.True(t, ok)
			assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
			idsByName[lbl.Name] = lbl.ID
		}

		// labels summary, only the built-ins remains
		s.DoJSON("GET", "/api/latest/fleet/labels/summary", nil, http.StatusOK, &summaryResp)
		assert.Len(t, summaryResp.Labels, builtInsCount)
		for _, lbl := range summaryResp.Labels {
			_, ok := builtinsMap[lbl.Name]
			assert.True(t, ok)
			assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
			assert.Equal(t, idsByName[lbl.Name], lbl.ID)
		}

		// host summary matches built-ins count
		var hostSummaryResp getHostSummaryResponse
		s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &hostSummaryResp)
		assert.Len(t, hostSummaryResp.BuiltinLabels, builtInsCount)
		for _, lbl := range hostSummaryResp.BuiltinLabels {
			_, ok := builtinsMap[lbl.Name]
			assert.True(t, ok)
			assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
			assert.Equal(t, idsByName[lbl.Name], lbl.ID)
		}

		require.Len(t, idsByName, len(builtinsMap))
		for name := range builtinsMap {
			id, ok := idsByName[name]
			require.True(t, ok)

			// attempt to delete by name
			s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(name)), nil, http.StatusUnprocessableEntity, &delResp)

			// attempt to delete by id
			s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", id), nil, http.StatusUnprocessableEntity, &delIDResp)
		}

		// A modify that changes membership but fails metadata validation (a
		// duplicate name) must roll back as a unit: the membership change must not
		// be committed on its own, and no edited_label activity must be recorded
		// for the failed request.
		createResp = fleet.CreateLabelResponse{}
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: "atomic_conflict_label"}, http.StatusOK, &createResp)
		conflictName := createResp.Label.Name

		createResp = fleet.CreateLabelResponse{}
		s.DoJSON("POST", "/api/latest/fleet/labels",
			&fleet.LabelPayload{Name: "atomic_target_label", HostIDs: []uint{manualHosts[0].ID, manualHosts[1].ID}}, http.StatusOK, &createResp)
		atomicLbl := createResp.Label.Label
		require.ElementsMatch(t, []uint{manualHosts[0].ID, manualHosts[1].ID}, createResp.Label.HostIDs)

		// watermark: id of the most recent activity before the failed request
		lastActID := s.lastActivityMatches("", "", 0)

		// rename to the conflicting name while also changing membership: the
		// duplicate name must fail the whole request with 409
		modResp = fleet.ModifyLabelResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", atomicLbl.ID),
			&fleet.ModifyLabelPayload{Name: &conflictName, HostIDs: []uint{manualHosts[2].ID}}, http.StatusConflict, &modResp)

		// name and membership must be unchanged (no partial commit)
		getResp = fleet.GetLabelResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", atomicLbl.ID), nil, http.StatusOK, &getResp)
		assert.Equal(t, atomicLbl.Name, getResp.Label.Name)
		assert.ElementsMatch(t, []uint{manualHosts[0].ID, manualHosts[1].ID}, getResp.Label.HostIDs)

		// no new activity must have been recorded for the failed request
		assert.Equal(t, lastActID, s.lastActivityMatches("", "", 0))

		// a valid rename that also changes membership must commit both and record
		// the edited_label activity
		modResp = fleet.ModifyLabelResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", atomicLbl.ID),
			&fleet.ModifyLabelPayload{Name: new("atomic_target_renamed"), HostIDs: []uint{manualHosts[2].ID}}, http.StatusOK, &modResp)
		assert.Equal(t, "atomic_target_renamed", modResp.Label.Name)
		assert.ElementsMatch(t, []uint{manualHosts[2].ID}, modResp.Label.HostIDs)
		require.Greater(t, s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedLabel{}.ActivityName(), "", 0), lastActID)
	})

	t.Run("IdP Labels", func(t *testing.T) {
		// Add some SCIM users
		mysqltest.ExecAdhocSQL(
			t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO scim_users (id, user_name, department) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?), (?, ?, ?), (?, ?, ?)",
					1,
					"no_groups_no_department",
					nil,
					2,
					"one_group_good_department",
					"department_good",
					3,
					"all_the_groups_good_department",
					"department_good",
					4,
					"wrong_groups_wrong_department",
					"department_other",
					5,
					"no_groups_with_department",
					"department_other_2",
				)
				return err
			},
		)
		// Add some SCIM groups
		mysqltest.ExecAdhocSQL(
			t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO scim_groups (id, display_name) VALUES (?, ?), (?, ?), (?, ?)",
					1,
					"group_good",
					2,
					"group_bad",
					3,
					"group_great",
				)
				return err
			},
		)
		// Add some SCIM group memberships
		mysqltest.ExecAdhocSQL(
			t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO scim_user_group (scim_user_id, group_id) VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)",
					2, 1, // "one_group" -> "group_good"
					3, 1, // "all_the_groups" -> "group_good"
					3, 2, // "all_the_groups" -> "group_bad"
					3, 3, // "all_the_groups" -> "group_great"
					4, 2, // "wrong_groups" -> "group_bad"
				)
				return err
			},
		)
		// Add some host->scim user mappings
		mysqltest.ExecAdhocSQL(
			t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO host_scim_user (host_id, scim_user_id) VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)",
					hosts[0].ID, 1,
					hosts[1].ID, 2,
					hosts[2].ID, 3,
					hosts[3].ID, 2,
					hosts[4].ID, 4,
					hosts[6].ID, 5,
				)
				return err
			},
		)

		t.Run("IdP Group Label", func(t *testing.T) {
			// Create a label for an IdP group
			criteria := &fleet.HostVitalCriteria{
				Vital: new("end_user_idp_group"),
				Value: new("group_good"),
			}

			labelParams := fleet.CreateLabelRequest{
				LabelPayload: fleet.LabelPayload{
					Name:     "Test IdP Group Label",
					Criteria: criteria,
				},
			}

			labelResp := fleet.CreateLabelResponse{}
			s.DoJSON("POST", "/api/latest/fleet/labels", labelParams, http.StatusOK, &labelResp)
			require.NotNil(t, labelResp.Label)

			filter := fleet.TeamFilter{User: test.UserAdmin}
			label, _, err := s.ds.Label(context.Background(), labelResp.Label.Label.ID, filter)
			require.NoError(t, err)

			// Verify that the query and values are correct.
			// Test parsing the criteria
			query, queryValues, err := label.CalculateHostVitalsQuery()
			require.NoError(t, err)
			queryValuesJson, err := json.Marshal(queryValues)
			require.NoError(t, err)

			// Compare whitespace-normalized SQL: the IdP join fragment is a multi-line
			// raw string whose indentation is irrelevant to the query's meaning.
			assert.Equal(t,
				"SELECT %s FROM %s JOIN host_scim_user ON (hosts.id = host_scim_user.host_id) JOIN scim_users ON (host_scim_user.scim_user_id = scim_users.id) LEFT JOIN ( WITH RECURSIVE scim_user_group_expanded AS ( SELECT scim_user_id, group_id FROM scim_user_group WHERE scim_user_id IN (SELECT scim_user_id FROM host_scim_user) UNION SELECT e.scim_user_id, gg.parent_group_id AS group_id FROM scim_user_group_expanded e JOIN scim_group_group gg ON gg.child_group_id = e.group_id ) SELECT scim_user_id, group_id FROM scim_user_group_expanded ) scim_user_group ON (host_scim_user.scim_user_id = scim_user_group.scim_user_id) LEFT JOIN scim_groups ON (scim_user_group.group_id = scim_groups.id) WHERE scim_groups.display_name = ? GROUP BY hosts.id",
				strings.Join(strings.Fields(query), " "))
			assert.Equal(t, `["group_good"]`, string(queryValuesJson))

			// Update label membership.
			_, _, err = s.ds.UpdateLabelMembershipByHostCriteria(context.Background(), label)
			require.NoError(t, err)

			// Verify that the label has the correct hosts.
			// Check that the label has the correct hosts
			//
			// host 1 shouldn't be returned because its scim user has no groups
			// host 2 should be returned because its scim user has the "group_good" group
			// host 3 should be returned because it has a scim user with the "group_good" group
			// host 4 should be returned because it has a scim user with the "group_good" group
			// host 5 shouldn't be returned because its scim user only has the "group_bad" group
			// host 6 shouldn't be returned because it has no scim user
			//
			hostsInLabel, err := s.ds.ListHostsInLabel(context.Background(), filter, label.ID, fleet.HostListOptions{})
			require.NoError(t, err)
			require.Len(t, hostsInLabel, 3)
			require.ElementsMatch(t, []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID}, []uint{hostsInLabel[0].ID, hostsInLabel[1].ID, hostsInLabel[2].ID})

			// Check that the label has the correct host count
			label, _, err = s.ds.Label(context.Background(), labelResp.Label.Label.ID, filter)
			require.NoError(t, err)
			assert.Equal(t, 3, label.HostCount)
		})

		t.Run("IdP Department Label", func(t *testing.T) {
			// Create a label for an IdP department
			criteria := &fleet.HostVitalCriteria{
				Vital: new("end_user_idp_department"),
				Value: new("department_good"),
			}

			labelParams := fleet.CreateLabelRequest{
				LabelPayload: fleet.LabelPayload{
					Name:     "Test IdP Department Label",
					Criteria: criteria,
				},
			}

			labelResp := fleet.CreateLabelResponse{}
			s.DoJSON("POST", "/api/latest/fleet/labels", labelParams, http.StatusOK, &labelResp)
			require.NotNil(t, labelResp.Label)

			filter := fleet.TeamFilter{User: test.UserAdmin}
			label, _, err := s.ds.Label(context.Background(), labelResp.Label.Label.ID, filter)
			require.NoError(t, err)

			// Verify that the query and values are correct.
			// Test parsing the criteria
			query, queryValues, err := label.CalculateHostVitalsQuery()
			require.NoError(t, err)
			queryValuesJson, err := json.Marshal(queryValues)
			require.NoError(t, err)

			// Compare whitespace-normalized SQL (see the IdP Group Label subtest above).
			assert.Equal(t,
				"SELECT %s FROM %s JOIN host_scim_user ON (hosts.id = host_scim_user.host_id) JOIN scim_users ON (host_scim_user.scim_user_id = scim_users.id) LEFT JOIN ( WITH RECURSIVE scim_user_group_expanded AS ( SELECT scim_user_id, group_id FROM scim_user_group WHERE scim_user_id IN (SELECT scim_user_id FROM host_scim_user) UNION SELECT e.scim_user_id, gg.parent_group_id AS group_id FROM scim_user_group_expanded e JOIN scim_group_group gg ON gg.child_group_id = e.group_id ) SELECT scim_user_id, group_id FROM scim_user_group_expanded ) scim_user_group ON (host_scim_user.scim_user_id = scim_user_group.scim_user_id) LEFT JOIN scim_groups ON (scim_user_group.group_id = scim_groups.id) WHERE scim_users.department = ? GROUP BY hosts.id",
				strings.Join(strings.Fields(query), " "))
			assert.Equal(t, `["department_good"]`, string(queryValuesJson))

			// Update label membership.
			_, _, err = s.ds.UpdateLabelMembershipByHostCriteria(context.Background(), label)
			require.NoError(t, err)

			// Verify that the label has the correct hosts.
			hostsInLabel, err := s.ds.ListHostsInLabel(context.Background(), filter, label.ID, fleet.HostListOptions{})
			require.NoError(t, err)
			require.Len(t, hostsInLabel, 3)
			require.ElementsMatch(t, []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID}, []uint{hostsInLabel[0].ID, hostsInLabel[1].ID, hostsInLabel[2].ID})

			// Check that the label has the correct host count
			label, _, err = s.ds.Label(context.Background(), labelResp.Label.Label.ID, filter)
			require.NoError(t, err)
			assert.Equal(t, 3, label.HostCount)

			t.Run("No Groups", func(t *testing.T) {
				// Create a label for IdP department to test users with department but no groups
				criteria := &fleet.HostVitalCriteria{
					Vital: new("end_user_idp_department"),
					Value: new("department_other_2"),
				}
				labelParams := fleet.CreateLabelRequest{
					LabelPayload: fleet.LabelPayload{
						Name:     "Test IdP Department Label 2",
						Criteria: criteria,
					},
				}

				labelResp := fleet.CreateLabelResponse{}
				s.DoJSON("POST", "/api/latest/fleet/labels", labelParams, http.StatusOK, &labelResp)
				require.NotNil(t, labelResp.Label)

				label, _, err = s.ds.Label(context.Background(), labelResp.Label.Label.ID, filter)
				require.NoError(t, err)

				// Update label membership.
				_, _, err = s.ds.UpdateLabelMembershipByHostCriteria(context.Background(), label)
				require.NoError(t, err)

				// Verify that the label has the correct hosts.
				hostsInLabel, err := s.ds.ListHostsInLabel(context.Background(), filter, label.ID, fleet.HostListOptions{})
				require.NoError(t, err)
				require.Len(t, hostsInLabel, 1)
				// host 7 is mapped to user 5 which matches the department but has no groups.
				require.ElementsMatch(t, []uint{hosts[6].ID}, []uint{hostsInLabel[0].ID})

				// Check that the label has the correct host count
				label, _, err = s.ds.Label(context.Background(), label.ID, filter)
				require.NoError(t, err)
				assert.Equal(t, 1, label.HostCount)
			})
		})
	})

	t.Run("Sort by order_key", func(t *testing.T) {
		ctx := context.Background()

		// Create three teams so each host has a distinct team_id and team_name.
		// Names sort A < B < C and team IDs are assigned in creation order.
		teamPrefix := strings.ReplaceAll(t.Name(), "/", "_")
		teamA, err := s.ds.NewTeam(ctx, &fleet.Team{Name: teamPrefix + "-team-A"})
		require.NoError(t, err)
		teamB, err := s.ds.NewTeam(ctx, &fleet.Team{Name: teamPrefix + "-team-B"})
		require.NoError(t, err)
		teamC, err := s.ds.NewTeam(ctx, &fleet.Team{Name: teamPrefix + "-team-C"})
		require.NoError(t, err)
		teamIDs := []*uint{&teamA.ID, &teamB.ID, &teamC.ID}

		// Each sortHost[i] is set up with field values that sort in the same direction
		// as the index: sortHost[0] is "smallest", sortHost[2] is "largest", for every
		// orderable field below. This means ASC order is always [h0, h1, h2] and DESC
		// order is always [h2, h1, h0], which keeps the per-field test cases compact.
		base := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
		platforms := []string{"darwin", "linux", "windows"}
		sortHosts := make([]*fleet.Host, 3)
		for i := range 3 {
			h, err := s.ds.NewHost(ctx, &fleet.Host{
				OsqueryHostID:               new(fmt.Sprintf("sort-osq-%d", i)),
				NodeKey:                     new(fmt.Sprintf("sort-nk-%d", i)),
				UUID:                        fmt.Sprintf("aaaaaaaa-0000-0000-0000-00000000000%d", i+1),
				Hostname:                    fmt.Sprintf("sort-host-%d", i),
				ComputerName:                fmt.Sprintf("sort-comp-%d", i),
				HardwareSerial:              fmt.Sprintf("sort-ser-%d", i),
				Platform:                    platforms[i],
				PlatformLike:                fmt.Sprintf("sort-plike-%d", i),
				OsqueryVersion:              fmt.Sprintf("%d.0.0", i+1),
				OSVersion:                   fmt.Sprintf("OS-%d.0", i+1),
				Uptime:                      time.Duration(i+1) * time.Hour,
				Memory:                      int64(i+1) * 1024,
				DistributedInterval:         uint(i+1) * 10,
				LoggerTLSPeriod:             uint(i+1) * 100,
				ConfigTLSRefresh:            uint(i+1) * 5,
				DetailUpdatedAt:             base.Add(time.Duration(i+1) * time.Hour),
				LabelUpdatedAt:              base.Add(time.Duration(i+1) * 2 * time.Hour),
				PolicyUpdatedAt:             base.Add(time.Duration(i+1) * 3 * time.Hour),
				RefetchCriticalQueriesUntil: new(base.Add(time.Duration(i+1) * 4 * time.Hour)),
				RefetchRequested:            i == 2, // only the "largest" host has refetch_requested = true
				TeamID:                      teamIDs[i],
			})
			require.NoError(t, err)
			sortHosts[i] = h
		}

		// Set columns not handled by NewHost via direct UPDATEs.
		for i, h := range sortHosts {
			mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					ctx, `
					UPDATE hosts SET
						created_at = ?,
						updated_at = ?,
						last_enrolled_at = ?,
						last_restarted_at = ?,
						primary_ip_id = ?,
						primary_ip = ?,
						primary_mac = ?,
						public_ip = ?,
						timezone = ?,
						build = ?,
						code_name = ?,
						cpu_type = ?,
						cpu_subtype = ?,
						cpu_brand = ?,
						cpu_physical_cores = ?,
						cpu_logical_cores = ?,
						hardware_vendor = ?,
						hardware_model = ?,
						hardware_version = ?
					WHERE id = ?`,
					base.Add(time.Duration(i+1)*5*time.Hour),
					base.Add(time.Duration(i+1)*6*time.Hour),
					base.Add(time.Duration(i+1)*7*time.Hour),
					base.Add(time.Duration(i+1)*8*time.Hour),
					uint(i+1)*100, //nolint:gosec // ignore G115
					fmt.Sprintf("10.0.0.%d", i+1),
					fmt.Sprintf("aa:bb:cc:00:00:0%d", i+1),
					fmt.Sprintf("8.0.0.%d", i+1),
					fmt.Sprintf("UTC-%d", i),
					fmt.Sprintf("sort-build-%d", i),
					fmt.Sprintf("sort-code-%d", i),
					fmt.Sprintf("sort-ct-%d", i),
					fmt.Sprintf("sort-cs-%d", i),
					fmt.Sprintf("sort-cb-%d", i),
					i+1,
					(i+1)*2,
					fmt.Sprintf("sort-hv-%d", i),
					fmt.Sprintf("sort-hm-%d", i),
					fmt.Sprintf("sort-hver-%d", i),
					h.ID,
				)
				return err
			})
		}

		// Populate joined tables (host_disks, host_seen_times, host_updates,
		// host_issues, host_emails) with values that also sort by host index.
		for i, h := range sortHosts {
			mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(ctx, `
					INSERT INTO host_disks (host_id, gigs_disk_space_available, percent_disk_space_available, gigs_total_disk_space)
					VALUES (?, ?, ?, ?)`,
					h.ID, float64((i+1)*100), float64((i+1)*10), float64((i+1)*1000))
				return err
			})
			mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(ctx,
					`UPDATE host_seen_times SET seen_time = ? WHERE host_id = ?`,
					base.Add(time.Duration(i+1)*9*time.Hour), h.ID)
				return err
			})
			mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(ctx,
					`INSERT INTO host_updates (host_id, software_updated_at) VALUES (?, ?)`,
					h.ID, base.Add(time.Duration(i+1)*10*time.Hour))
				return err
			})
			mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(ctx, `
					INSERT INTO host_issues (host_id, failing_policies_count, critical_vulnerabilities_count, total_issues_count)
					VALUES (?, ?, ?, ?)`,
					h.ID, uint(i+1), uint(i+1)*2, uint(i+1)*3) //nolint:gosec // ignore G115
				return err
			})
			mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(ctx,
					`INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
					h.ID, fmt.Sprintf("sort-%d@example.com", i), "src")
				return err
			})
		}

		// Create a dynamic label and add the three hosts to it.
		var createResp fleet.CreateLabelResponse
		s.DoJSON("POST", "/api/latest/fleet/labels",
			&fleet.LabelPayload{Name: teamPrefix + "-sort", Query: "select 1"}, http.StatusOK, &createResp)
		sortLabel := createResp.Label.Label
		for _, h := range sortHosts {
			require.NoError(t, s.ds.RecordLabelQueryExecutions(ctx, h,
				map[uint]*bool{sortLabel.ID: new(true)}, time.Now(), false))
		}

		ascIDs := []uint{sortHosts[0].ID, sortHosts[1].ID, sortHosts[2].ID}
		descIDs := []uint{sortHosts[2].ID, sortHosts[1].ID, sortHosts[0].ID}
		labelHostsURL := fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", sortLabel.ID)

		// orderKeys lists every field for which sortHosts is set up to produce a
		// strict, deterministic ASC ordering of [h0, h1, h2]. Each field gets its
		// own subtest verifying both ASC and DESC orderings.
		orderKeys := []string{
			"id", "osquery_host_id", "created_at", "updated_at", "detail_updated_at",
			"hostname", "uuid", "platform", "osquery_version", "os_version", "build",
			"platform_like", "code_name", "uptime", "memory", "cpu_type", "cpu_subtype",
			"cpu_brand", "cpu_physical_cores", "cpu_logical_cores",
			"hardware_vendor", "hardware_model", "hardware_version", "hardware_serial",
			"computer_name", "primary_ip_id", "distributed_interval", "logger_tls_period",
			"config_tls_refresh", "primary_ip", "primary_mac", "label_updated_at",
			"last_enrolled_at", "refetch_critical_queries_until", "team_id",
			"policy_updated_at", "public_ip",
			"gigs_disk_space_available", "percent_disk_space_available",
			"gigs_total_disk_space", "seen_time", "software_updated_at",
			"last_restarted_at", "timezone", "team_name",
			"failing_policies_count", "critical_vulnerabilities_count",
			"total_issues_count", "issues",
			"device_mapping",
			"display_name",
		}
		for _, key := range orderKeys {
			t.Run(key, func(t *testing.T) {
				params := []string{"order_key", key, "order_direction", "asc"}
				if key == "device_mapping" {
					params = append(params, "device_mapping", "true")
				}

				var resp listHostsResponse
				s.DoJSON("GET", labelHostsURL, nil, http.StatusOK, &resp, params...)
				require.Len(t, resp.Hosts, 3)
				gotAsc := []uint{resp.Hosts[0].ID, resp.Hosts[1].ID, resp.Hosts[2].ID}
				assert.Equal(t, ascIDs, gotAsc, "asc order mismatch")

				params[3] = "desc"
				resp = listHostsResponse{}
				s.DoJSON("GET", labelHostsURL, nil, http.StatusOK, &resp, params...)
				require.Len(t, resp.Hosts, 3)
				gotDesc := []uint{resp.Hosts[0].ID, resp.Hosts[1].ID, resp.Hosts[2].ID}
				assert.Equal(t, descIDs, gotDesc, "desc order mismatch")
			})
		}

		// refetch_requested is a bool, so only h2 (true) has a unique value. ASC must
		// place h2 last; DESC must place h2 first. The order between h0 and h1 (both
		// false) is not deterministic, so we only assert the position of h2.
		t.Run("refetch_requested", func(t *testing.T) {
			var resp listHostsResponse
			s.DoJSON("GET", labelHostsURL, nil, http.StatusOK, &resp,
				"order_key", "refetch_requested", "order_direction", "asc")
			require.Len(t, resp.Hosts, 3)
			assert.Equal(t, sortHosts[2].ID, resp.Hosts[2].ID, "asc: h2 should be last")

			resp = listHostsResponse{}
			s.DoJSON("GET", labelHostsURL, nil, http.StatusOK, &resp,
				"order_key", "refetch_requested", "order_direction", "desc")
			require.Len(t, resp.Hosts, 3)
			assert.Equal(t, sortHosts[2].ID, resp.Hosts[0].ID, "desc: h2 should be first")
		})

		// Cursor pagination (`after`) injects the order_key into the WHERE
		// clause; SELECT-list aliases like team_name would error there. Verify
		// that paging through team_name with a cursor returns the expected
		// hosts and does not error.
		t.Run("team_name with after cursor", func(t *testing.T) {
			var resp listHostsResponse
			s.DoJSON("GET", labelHostsURL, nil, http.StatusOK, &resp,
				"order_key", "team_name", "order_direction", "asc",
				"after", teamA.Name, "per_page", "10")
			require.Len(t, resp.Hosts, 2)
			assert.Equal(t, sortHosts[1].ID, resp.Hosts[0].ID)
			assert.Equal(t, sortHosts[2].ID, resp.Hosts[1].ID)
		})
	})
}

// Sanity test to make sure fleet/labels/<all>/hosts and fleet/hosts return the same thing.
func (s *integrationTestSuite) TestListHostsByLabel() {
	t := s.T()

	lblIDs, err := s.ds.LabelIDsByName(context.Background(), []string{"All Hosts"}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, lblIDs, 1)
	labelID := lblIDs["All Hosts"]

	hosts := s.createHosts(t, "darwin")
	host := hosts[0]

	// Update label
	mysqltest.ExecAdhocSQL(
		t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(
				context.Background(),
				"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, (SELECT id FROM labels WHERE name = 'All Hosts' AND label_type = 1))",
				host.ID,
			)
			return err
		},
	)

	// set disk space information
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), host.ID, 10.0, 2.0, 500.0, nil)) // low disk

	// Update host fields
	host.Uptime = 30 * time.Second
	host.RefetchRequested = true
	host.OSVersion = "macOS 14.2"
	host.Build = "abc"
	host.PlatformLike = "darwin"
	host.CodeName = "sky"
	host.Memory = 1000
	host.CPUType = "arm64"
	host.CPUSubtype = "ARM64e"
	host.CPUBrand = "Apple M2 Pro"
	host.CPUPhysicalCores = 12
	host.CPULogicalCores = 14
	host.HardwareVendor = "Apple Inc."
	host.HardwareModel = "Mac14,10"
	host.HardwareVersion = "23"
	host.HardwareSerial = "ABC123"
	host.ComputerName = "MBP"
	host.PublicIP = "1.1.1.1"
	host.PrimaryIP = "10.10.10.10"
	host.PrimaryMac = "11:22:33"
	host.DistributedInterval = 10
	host.ConfigTLSRefresh = 9
	host.OsqueryVersion = "5.10"
	err = s.ds.UpdateHost(context.Background(), host)
	require.NoError(t, err)

	// Add team
	team, err := s.ds.NewTeam(
		context.Background(), &fleet.Team{
			Name: uuid.New().String(),
		},
	)
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

	// Add pack
	_, err = s.ds.NewPack(
		context.Background(), &fleet.Pack{
			Name: t.Name(),
			Hosts: []fleet.Target{
				{
					Type:     fleet.TargetHost,
					TargetID: hosts[0].ID,
				},
			},
		},
	)
	require.NoError(t, err)

	// Add policy
	qr, err := s.ds.NewQuery(
		context.Background(), &fleet.Query{
			Name:           t.Name(),
			Description:    "Some description",
			Query:          "select * from osquery;",
			ObserverCanRun: true,
			Logging:        fleet.LoggingSnapshot,
		},
	)
	require.NoError(t, err)

	gpParams := fleet.GlobalPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some global resolution",
	}
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NoError(
		t,
		errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{gpResp.Policy.ID: new(false)}, time.Now(), false, nil)),
	)

	// Add MDM info
	require.NoError(
		t,
		s.ds.SetOrUpdateMDMData(
			context.Background(), host.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "", false,
		),
	)

	// Add device mapping
	require.NoError(
		t, s.ds.ReplaceHostDeviceMapping(
			context.Background(), host.ID, []*fleet.HostDeviceMapping{
				{HostID: hosts[0].ID, Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
				{HostID: hosts[0].ID, Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
			}, fleet.DeviceMappingGoogleChromeProfiles,
		),
	)

	// Now do the actual API calls that we will compare.
	var hostsResp, labelsResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hostsResp, "device_mapping", "true")
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", labelID), nil, http.StatusOK, &labelsResp, "device_mapping", "true")

	// Converting to formatted JSON for easier diffs
	hostsJson, _ := json.MarshalIndent(hostsResp, "", "  ")
	labelsJson, _ := json.MarshalIndent(labelsResp, "", "  ")
	assert.Equal(t, string(hostsJson), string(labelsJson))

	// Do request with include_device_status, since it's an additional feature
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hostsResp, "device_mapping", "true", "include_device_status", "true")
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", labelID), nil, http.StatusOK, &labelsResp, "device_mapping", "true", "include_device_status", "true")

	// Converting to formatted JSON for easier diffs
	hostsJson, _ = json.MarshalIndent(hostsResp, "", "  ")
	labelsJson, _ = json.MarshalIndent(labelsResp, "", "  ")
	assert.Equal(t, string(hostsJson), string(labelsJson))
}

func (s *integrationTestSuite) TestLabelSpecs() {
	t := s.T()

	// list label specs, only those of the built-ins
	var listResp fleet.GetLabelSpecsResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/labels", nil, http.StatusOK, &listResp)
	assert.True(t, len(listResp.Specs) > 0)
	for _, spec := range listResp.Specs {
		assert.Equal(t, fleet.LabelTypeBuiltIn, spec.LabelType)
	}
	builtInsCount := len(listResp.Specs)

	name := strings.ReplaceAll(t.Name(), "/", "_")
	// apply an invalid label spec - dynamic membership with host specified
	var applyResp fleet.ApplyLabelSpecsResponse
	s.DoJSON(
		"POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{
			Specs: []*fleet.LabelSpec{
				{
					Name:                name,
					Query:               "select 1",
					Platform:            "linux",
					LabelMembershipType: fleet.LabelMembershipTypeDynamic,
					Hosts:               []string{"abc"},
				},
			},
		}, http.StatusUnprocessableEntity, &applyResp,
	)

	// apply an invalid label spec - invalid platform
	s.DoJSON(
		"POST",
		"/api/latest/fleet/spec/labels",
		fleet.ApplyLabelSpecsRequest{
			Specs: []*fleet.LabelSpec{
				{
					Name:                name,
					Query:               "select 1",
					Platform:            "bados",
					LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				},
			},
		},
		http.StatusUnprocessableEntity,
		&applyResp,
	)

	// apply a valid label spec - generic "linux" platform
	s.DoJSON(
		"POST",
		"/api/latest/fleet/spec/labels",
		fleet.ApplyLabelSpecsRequest{
			Specs: []*fleet.LabelSpec{
				{
					Name:                name + "_linux",
					Query:               "select 1",
					Platform:            "linux",
					LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				},
			},
		},
		http.StatusOK,
		&applyResp,
	)

	// apply a valid label spec - manual membership without hosts specified (preserves existing membership)
	s.DoJSON(
		"POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{
			Specs: []*fleet.LabelSpec{
				{
					Name:                name,
					LabelMembershipType: fleet.LabelMembershipTypeManual,
				},
			},
		}, http.StatusOK, &applyResp,
	)

	// apply an invalid label spec - builtin label type
	s.DoJSON("POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{
		Specs: []*fleet.LabelSpec{
			{
				Name:                name,
				Query:               "select 1",
				Platform:            "linux",
				LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				LabelType:           fleet.LabelTypeBuiltIn,
			},
		},
	}, http.StatusUnprocessableEntity, &applyResp)

	// apply an invalid label spec - builtin label name
	for n := range fleet.ReservedLabelNames() {
		s.DoJSON("POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{
			Specs: []*fleet.LabelSpec{
				{
					Name:                n,
					Query:               "select 1",
					Platform:            "linux",
					LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				},
			},
		}, http.StatusUnprocessableEntity, &applyResp)
	}

	// apply a valid label spec
	s.DoJSON("POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{
		Specs: []*fleet.LabelSpec{
			{
				Name:                name,
				Query:               "select 1",
				Platform:            "centos",
				LabelMembershipType: fleet.LabelMembershipTypeDynamic,
			},
		},
	}, http.StatusOK, &applyResp)

	// list label specs, has the newly created ones
	s.DoJSON("GET", "/api/latest/fleet/spec/labels", nil, http.StatusOK, &listResp)
	assert.Len(t, listResp.Specs, builtInsCount+2)

	// get a specific label spec
	var getResp fleet.GetLabelSpecResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/spec/labels/%s", url.PathEscape(name)), nil, http.StatusOK, &getResp)
	assert.Equal(t, name, getResp.Spec.Name)
	assert.NotEqual(t, 0, getResp.Spec.ID)

	// the generic "linux" platform round-trips through the spec endpoints
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/spec/labels/%s", url.PathEscape(name+"_linux")), nil, http.StatusOK, &getResp)
	assert.Equal(t, "linux", getResp.Spec.Platform)

	// get a non-existing label spec
	s.DoJSON("GET", "/api/latest/fleet/spec/labels/zzz", nil, http.StatusNotFound, &getResp)
}

func (s *integrationTestSuite) TestUsers() {
	// ensure that on exit, the admin token is used
	defer func() { s.token = s.getTestAdminToken() }()

	t := s.T()

	// existing users:
	// {ID: 1, Name: "Test Name admin1@example.com", Email: "admin1@example.com", ...}
	// {ID: 2, Name: "Test Name user1@example.com", Email: "user1@example.com", ...}
	// {ID: 3, Name: "Test Name user2@example.com", Email: "user2@example.com", ...}

	// list existing users
	var listResp listUsersResponse
	s.DoJSON("GET", "/api/latest/fleet/users", nil, http.StatusOK, &listResp)
	assert.Len(t, listResp.Users, len(s.users))

	// invalid order_key returns 422
	listResp = listUsersResponse{}
	s.DoJSON("GET", "/api/latest/fleet/users", nil, http.StatusUnprocessableEntity, &listResp, "order_key", "invalid")

	// with non-matching query
	s.DoJSON("GET", "/api/latest/fleet/users", nil, http.StatusOK, &listResp, "query", "noone")
	assert.Len(t, listResp.Users, 0)

	// with matching query containing leading/trailing whitespaces
	s.DoJSON("GET", "/api/latest/fleet/users", nil, http.StatusOK, &listResp, "query", " user 	")
	assert.Len(t, listResp.Users, 2)
	assert.Equal(t, uint(2), listResp.Users[0].ID)
	assert.Equal(t, uint(3), listResp.Users[1].ID)

	// test available teams returned by `/me` endpoint for existing user
	var getMeResp getUserResponse
	ssn := createSession(t, 1, s.ds)
	resp := s.DoRawWithHeaders("GET", "/api/latest/fleet/me", []byte(""), http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ssn.Key),
	})
	err := json.NewDecoder(resp.Body).Decode(&getMeResp)
	require.NoError(t, err)
	assert.Equal(t, uint(1), getMeResp.User.ID)
	assert.NotNil(t, getMeResp.User.GlobalRole)
	assert.Len(t, getMeResp.User.Teams, 0)
	assert.Len(t, getMeResp.AvailableTeams, 0)

	// test user settings from 2 endpoints

	// get session user with ui settings, which should be empty, two endpoints
	var getResp getUserResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", 1), nil, http.StatusOK, &getResp, "include_ui_settings", "true")
	assert.Equal(t, uint(1), getResp.User.ID)
	assert.Empty(t, getResp.User.Settings)

	resp = s.DoRawWithHeaders("GET", "/api/latest/fleet/me", []byte(""), http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ssn.Key),
	}, "include_ui_settings", "true")
	err = json.NewDecoder(resp.Body).Decode(&getMeResp)
	require.NoError(t, err)
	// session user id 1
	assert.Equal(t, uint(1), getMeResp.User.ID)
	assert.NotNil(t, getMeResp.User.GlobalRole)
	// settings should only be present in dedicated settings field, not in user object
	assert.Nil(t, getMeResp.User.Settings)
	assert.Empty(t, getMeResp.Settings)

	// modify session user - add ui setting
	var modResp modifyUserResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", 1), json.RawMessage(`{
		"settings": {
			"hidden_host_columns": ["osquery_version"]}
	}`), http.StatusOK, &modResp)

	// get session user with ui settings, should now be present, two endpoints
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", 1), nil, http.StatusOK, &getResp, "include_ui_settings", "true")
	assert.Equal(t, uint(1), getResp.User.ID)
	assert.Nil(t, getResp.User.Settings)
	assert.Equal(t, getResp.Settings, &fleet.UserSettings{HiddenHostColumns: []string{"osquery_version"}})

	resp = s.DoRawWithHeaders("GET", "/api/latest/fleet/me", []byte(""), http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ssn.Key),
	}, "include_ui_settings", "true")
	err = json.NewDecoder(resp.Body).Decode(&getMeResp)
	require.NoError(t, err)
	assert.Equal(t, uint(1), getMeResp.User.ID)
	assert.NotNil(t, getMeResp.User.GlobalRole)
	assert.Nil(t, getMeResp.User.Settings)
	assert.Equal(t, getResp.Settings, &fleet.UserSettings{HiddenHostColumns: []string{"osquery_version"}})

	// modify user ui settings, check they are returned modified
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", 1), json.RawMessage(`{
		"settings": {
			"hidden_host_columns": ["hostname", "osquery_version"]}
	}`), http.StatusOK, &modResp)

	// get session user with ui settings, should now be modified, two endpoints
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", 1), nil, http.StatusOK, &getResp, "include_ui_settings", "true")
	assert.Equal(t, uint(1), getResp.User.ID)
	assert.Nil(t, getResp.User.Settings)
	assert.Equal(t, getResp.Settings, &fleet.UserSettings{HiddenHostColumns: []string{"hostname", "osquery_version"}})

	resp = s.DoRawWithHeaders("GET", "/api/latest/fleet/me", []byte(""), http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ssn.Key),
	}, "include_ui_settings", "true")
	err = json.NewDecoder(resp.Body).Decode(&getMeResp)
	require.NoError(t, err)
	assert.Equal(t, uint(1), getMeResp.User.ID)
	assert.NotNil(t, getMeResp.User.GlobalRole)
	assert.Nil(t, getMeResp.User.Settings)
	assert.Equal(t, getMeResp.Settings, &fleet.UserSettings{HiddenHostColumns: []string{"hostname", "osquery_version"}})

	// modify user ui settings, empty array, check they are returned correctly
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", 1), json.RawMessage(`{
		"settings": {
			"hidden_host_columns": []}
	}`), http.StatusOK, &modResp)

	// get session user with ui settings, should now be modified, two endpoints
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", 1), nil, http.StatusOK, &getResp, "include_ui_settings", "true")
	assert.Equal(t, uint(1), getResp.User.ID)
	assert.Nil(t, getResp.User.Settings)
	assert.Equal(t, getResp.Settings, &fleet.UserSettings{HiddenHostColumns: []string{}})

	resp = s.DoRawWithHeaders("GET", "/api/latest/fleet/me", []byte(""), http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ssn.Key),
	}, "include_ui_settings", "true")
	err = json.NewDecoder(resp.Body).Decode(&getMeResp)
	require.NoError(t, err)
	assert.Equal(t, uint(1), getMeResp.User.ID)
	assert.NotNil(t, getMeResp.User.GlobalRole)
	assert.Nil(t, getMeResp.User.Settings)
	assert.Equal(t, getMeResp.Settings, &fleet.UserSettings{HiddenHostColumns: []string{}})

	// create a new user
	var createResp createUserResponse
	userRawPwd := test.GoodPassword
	params := fleet.UserPayload{
		Name:       new("extra"),
		Email:      new("extra@asd.com"),
		Password:   new(userRawPwd),
		GlobalRole: new(fleet.RoleObserver),
		MFAEnabled: new(true),
	}
	// mailer isn't set up, which fails prior to silently ignoring MFA
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusBadRequest, &createResp)
	params.MFAEnabled = nil
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.User.ID)
	assert.True(t, createResp.User.AdminForcedPasswordReset)
	u := *createResp.User

	var loginResp fleet.LoginResponse

	// try MFA
	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `UPDATE users SET mfa_enabled = TRUE WHERE id = ?`, u.ID)
		return err
	})
	s.DoJSONWithoutAuth("POST", "/api/latest/fleet/sessions", sessionCreateRequest{Token: "foo"}, http.StatusUnauthorized, &loginResp)

	loginErrMessage := func(rawBody []byte) string {
		var body struct {
			Message string `json:"message"`
			Errors  []struct {
				Name   string `json:"name"`
				Reason string `json:"reason"`
			} `json:"errors"`
		}
		require.NoError(t, json.Unmarshal(rawBody, &body))
		return fmt.Sprintf("%s %+v", body.Message, body.Errors)
	}

	mfaUnsupportedResp := s.DoRawNoAuth("POST", "/api/latest/fleet/login",
		jsonMustMarshal(t, fleet.LoginRequest{Email: "extra@asd.com", Password: userRawPwd}),
		http.StatusUnauthorized)
	mfaUnsupportedBody, err := io.ReadAll(mfaUnsupportedResp.Body)
	require.NoError(t, err)
	mfaUnsupportedResp.Body.Close()

	wrongPwdResp := s.DoRawNoAuth("POST", "/api/latest/fleet/login",
		jsonMustMarshal(t, fleet.LoginRequest{Email: "extra@asd.com", Password: "wrong-" + userRawPwd}),
		http.StatusUnauthorized)
	wrongPwdBody, err := io.ReadAll(wrongPwdResp.Body)
	require.NoError(t, err)
	wrongPwdResp.Body.Close()

	nonexistentResp := s.DoRawNoAuth("POST", "/api/latest/fleet/login",
		jsonMustMarshal(t, fleet.LoginRequest{Email: "does-not-exist@asd.com", Password: userRawPwd}),
		http.StatusUnauthorized)
	nonexistentBody, err := io.ReadAll(nonexistentResp.Body)
	require.NoError(t, err)
	nonexistentResp.Body.Close()

	require.Equal(t, loginErrMessage(wrongPwdBody), loginErrMessage(mfaUnsupportedBody))
	require.Equal(t, loginErrMessage(wrongPwdBody), loginErrMessage(nonexistentBody))
	// MFA supported; send email
	s.DoJSONWithoutAuth("POST", "/api/latest/fleet/login",
		fleet.LoginRequest{Email: "extra@asd.com", Password: userRawPwd, SupportsEmailVerification: true}, http.StatusAccepted, &loginResp)
	var mfaToken string
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), tx, &mfaToken, `SELECT token FROM verification_tokens WHERE user_id = ? LIMIT 1`, createResp.User.ID)
	})
	// create session from MFA token
	s.DoJSONWithoutAuth("POST", "/api/latest/fleet/sessions", sessionCreateRequest{Token: mfaToken}, http.StatusOK, &loginResp)
	// can't use the same MFA token twice
	s.DoJSONWithoutAuth("POST", "/api/latest/fleet/sessions", sessionCreateRequest{Token: mfaToken}, http.StatusUnauthorized, &loginResp)

	// send another email, which we'll expire the token for
	s.DoJSONWithoutAuth("POST", "/api/latest/fleet/login",
		fleet.LoginRequest{Email: "extra@asd.com", Password: userRawPwd, SupportsEmailVerification: true}, http.StatusAccepted, &loginResp)
	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(
			context.Background(),
			`UPDATE verification_tokens SET created_at = NOW() - INTERVAL ? SECOND - INTERVAL 0.5 SECOND WHERE user_id = ?`,
			(time.Minute * 15).Seconds(),
			u.ID,
		)
		if err != nil {
			return err
		}

		return sqlx.GetContext(context.Background(), db, &mfaToken, `SELECT token FROM verification_tokens WHERE user_id = ? LIMIT 1`, createResp.User.ID)
	})
	s.DoJSONWithoutAuth("POST", "/api/latest/fleet/sessions", sessionCreateRequest{Token: mfaToken}, http.StatusUnauthorized, &loginResp)

	// turn off MFA
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{MFAEnabled: new(false)}, http.StatusOK, &modResp)
	require.False(t, modResp.User.MFAEnabled)

	// can't turn MFA back on
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{MFAEnabled: new(true)}, http.StatusPaymentRequired, &modResp)

	// login as that user and check that teams info is empty
	s.DoJSON("POST", "/api/latest/fleet/login", params, http.StatusOK, &loginResp)
	require.Equal(t, loginResp.User.ID, u.ID)
	assert.Len(t, loginResp.User.Teams, 0)
	assert.Len(t, loginResp.AvailableTeams, 0)

	// get that user from `/users` endpoint and check that teams info is empty
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, u.ID, getResp.User.ID)
	assert.Len(t, getResp.User.Teams, 0)
	assert.Len(t, getResp.AvailableTeams, 0)

	// get non-existing user
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID+1), nil, http.StatusNotFound, &getResp)

	// modify that user - simple name change
	params = fleet.UserPayload{
		Name: new("extraz"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), params, http.StatusOK, &modResp)
	assert.Equal(t, u.ID, modResp.User.ID)
	assert.Equal(t, u.Name+"z", modResp.User.Name)

	// modify that user - set an existing email
	params = fleet.UserPayload{
		Email: &getMeResp.User.Email,
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), params, http.StatusConflict, &modResp)

	// modify that user - set an email that has an invite for it
	createInviteReq := createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      new("colliding@email.com"),
		Name:       new("some name"),
		GlobalRole: null.StringFrom(fleet.RoleAdmin),
		MFAEnabled: new(true),
	}}
	createInviteResp := createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusPaymentRequired, &createInviteResp)
	createInviteReq.MFAEnabled = nil
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
	params = fleet.UserPayload{
		Email: new("colliding@email.com"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), params, http.StatusConflict, &modResp)

	// modify that user - set a non existent email
	params = fleet.UserPayload{
		Email: new("someemail@qowieuowh.com"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), params, http.StatusOK, &modResp)

	// modify user - email change, password does not match
	params = fleet.UserPayload{
		Email:    new("extra2@asd.com"),
		Password: new("wrongpass"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), params, http.StatusForbidden, &modResp)

	// modify user - email change, password ok
	params = fleet.UserPayload{
		Email:    new("extra2@asd.com"),
		Password: new(test.GoodPassword),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), params, http.StatusOK, &modResp)
	assert.Equal(t, u.ID, modResp.User.ID)
	assert.NotEqual(t, u.ID, modResp.User.Email)

	// modify invalid user
	params = fleet.UserPayload{
		Name: new("nosuchuser"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID+1), params, http.StatusNotFound, &modResp)

	var perfPwdResetResp performRequiredPasswordResetResponse
	newRawPwd := test.GoodPassword2
	// Try a required password change without authentication
	s.DoJSON(
		"POST", "/api/latest/fleet/perform_required_password_reset", performRequiredPasswordResetRequest{
			Password: newRawPwd,
			ID:       u.ID,
		}, http.StatusForbidden, &perfPwdResetResp,
	)

	// perform a required password change as the user themselves
	s.token = s.getTestToken(u.Email, userRawPwd)
	s.DoJSON("POST", "/api/latest/fleet/perform_required_password_reset", performRequiredPasswordResetRequest{
		Password: newRawPwd,
		ID:       u.ID,
	}, http.StatusOK, &perfPwdResetResp)
	assert.False(t, perfPwdResetResp.User.AdminForcedPasswordReset)
	oldUserRawPwd := userRawPwd
	userRawPwd = newRawPwd

	// perform a required password change again, this time it fails as there is no request pending
	perfPwdResetResp = performRequiredPasswordResetResponse{}
	newRawPwd = "new_password2!"
	s.DoJSON("POST", "/api/latest/fleet/perform_required_password_reset", performRequiredPasswordResetRequest{
		Password: newRawPwd,
		ID:       u.ID,
	}, http.StatusForbidden, &perfPwdResetResp)
	s.token = s.getTestAdminToken()

	// login as that user to verify that the new password is active (userRawPwd was updated to the new pwd)
	loginResp = fleet.LoginResponse{}
	s.DoJSON("POST", "/api/latest/fleet/login", fleet.LoginRequest{Email: u.Email, Password: userRawPwd}, http.StatusOK, &loginResp)
	require.Equal(t, loginResp.User.ID, u.ID)

	// logout for that user
	s.token = loginResp.Token
	var logoutResp fleet.LogoutResponse
	s.DoJSON("POST", "/api/latest/fleet/logout", nil, http.StatusOK, &logoutResp)

	// logout again, even though not logged in
	s.DoJSON("POST", "/api/latest/fleet/logout", nil, http.StatusUnauthorized, &logoutResp)

	s.token = s.getTestAdminToken()

	// login as that user with previous pwd fails
	loginResp = fleet.LoginResponse{}
	s.DoJSON("POST", "/api/latest/fleet/login", fleet.LoginRequest{Email: u.Email, Password: oldUserRawPwd}, http.StatusUnauthorized, &loginResp)

	// require a password reset
	var reqResetResp requirePasswordResetResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/users/%d/require_password_reset", u.ID), map[string]bool{"require": true}, http.StatusOK, &reqResetResp)
	assert.Equal(t, u.ID, reqResetResp.User.ID)
	assert.True(t, reqResetResp.User.AdminForcedPasswordReset)

	// require a password reset to invalid user
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/users/%d/require_password_reset", u.ID+1), map[string]bool{"require": true}, http.StatusNotFound, &reqResetResp)

	// delete user
	var delResp deleteUserResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), nil, http.StatusOK, &delResp)

	// delete invalid user
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), nil, http.StatusNotFound, &delResp)
}

func (s *integrationTestSuite) TestGlobalPoliciesAutomationConfig() {
	t := s.T()

	gpParams := fleet.GlobalPolicyRequest{
		Name:  "policy1",
		Query: "select 41;",
	}
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)

	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(fmt.Sprintf(`{
		"webhook_settings": {
    		"failing_policies_webhook": {
     	 		"enable_failing_policies_webhook": true,
     	 		"destination_url": "http://some/url",
     			"policy_ids": [%d],
				"host_batch_size": 1000
    		},
    		"interval": "1h"
  		}
	}`, gpResp.Policy.ID)), http.StatusOK)

	config := s.getConfig()
	require.True(t, config.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	require.Equal(t, []uint{gpResp.Policy.ID}, config.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
	require.Equal(t, 1*time.Hour, config.WebhookSettings.Interval.Duration)
	require.Equal(t, 1000, config.WebhookSettings.FailingPoliciesWebhook.HostBatchSize)

	deletePolicyParams := fleet.DeleteGlobalPoliciesRequest{IDs: []uint{gpResp.Policy.ID}}
	deletePolicyResp := fleet.DeleteGlobalPoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)

	config = s.getConfig()
	require.Empty(t, config.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
}

func (s *integrationTestSuite) TestActivitiesWebhookConfig() {
	t := s.T()

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "http://some/url"
    		}
  		}
	}`,
		), http.StatusOK,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEnabledActivityAutomations{}.ActivityName(),
		`{"webhook_url": "http://some/url"}`,
		0,
	)

	appConfig := s.getConfig()
	require.True(t, appConfig.WebhookSettings.ActivitiesWebhook.Enable)
	require.Equal(t, "http://some/url", appConfig.WebhookSettings.ActivitiesWebhook.DestinationURL)

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "http://some/other/url"
    		}
  		}
	}`,
		), http.StatusOK,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedActivityAutomations{}.ActivityName(),
		`{"webhook_url": "http://some/other/url"}`,
		0,
	)

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "invalid-url"
    		}
  		}
	}`,
		), http.StatusUnprocessableEntity,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedActivityAutomations{}.ActivityName(),
		`{"webhook_url": "http://some/other/url"}`,
		0,
	)

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": false
    		}
  		}
	}`,
		), http.StatusOK,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledActivityAutomations{}.ActivityName(),
		``,
		0,
	)

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "foo.baz"
    		}
  		}
	}`,
		), http.StatusUnprocessableEntity,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEnabledActivityAutomations{}.ActivityName(),
		`{"webhook_url": "http://some/url"}`,
		0,
	)
}

func (s *integrationTestSuite) TestHostStatusWebhookConfig() {
	t := s.T()

	// enable with valid config
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": true,
     	 		"destination_url": "http://some/url",
				  "host_percentage": 2,
					"days_count": 1
    		},
    		"interval": "1h"
  		}
	}`), http.StatusOK)

	config := s.getConfig()
	require.True(t, config.WebhookSettings.HostStatusWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.HostStatusWebhook.DestinationURL)
	require.Equal(t, 2.0, config.WebhookSettings.HostStatusWebhook.HostPercentage)
	require.Equal(t, 1, config.WebhookSettings.HostStatusWebhook.DaysCount)

	// update without a destination url
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": true,
     	 		"destination_url": "",
				  "host_percentage": 2,
					"days_count": 1
    		},
    		"interval": "1h"
  		}
	}`), http.StatusUnprocessableEntity)

	// update without a negative days count
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": true,
					"destination_url": "http://other/url",
				  "host_percentage": 2,
					"days_count": -123
    		},
    		"interval": "1h"
  		}
	}`), http.StatusUnprocessableEntity)

	// update with 0%
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": true,
					"destination_url": "http://other/url",
				  "host_percentage": 0,
					"days_count": 12
    		},
    		"interval": "1h"
  		}
	}`), http.StatusUnprocessableEntity)

	// config left unmodified since last successful call
	config = s.getConfig()
	require.True(t, config.WebhookSettings.HostStatusWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.HostStatusWebhook.DestinationURL)
	require.Equal(t, 2.0, config.WebhookSettings.HostStatusWebhook.HostPercentage)
	require.Equal(t, 1, config.WebhookSettings.HostStatusWebhook.DaysCount)

	// disabling ignores the invalid parameters
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": false,
     	 		"destination_url": "",
				  "host_percentage": 0
    		},
    		"interval": "1h"
  		}
	}`), http.StatusOK)

	config = s.getConfig()
	require.False(t, config.WebhookSettings.HostStatusWebhook.Enable)
}

func (s *integrationTestSuite) TestVulnerabilitiesWebhookConfig() {
	t := s.T()

	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"integrations": {"jira": [], "zendesk": []},
		"webhook_settings": {
    		"vulnerabilities_webhook": {
     	 		"enable_vulnerabilities_webhook": true,
     	 		"destination_url": "http://some/url",
     	 		"host_batch_size": 1234
    		},
    		"interval": "1h"
  		}
	}`), http.StatusOK)

	config := s.getConfig()
	require.True(t, config.WebhookSettings.VulnerabilitiesWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.VulnerabilitiesWebhook.DestinationURL)
	require.Equal(t, 1234, config.WebhookSettings.VulnerabilitiesWebhook.HostBatchSize)
	require.Equal(t, 1*time.Hour, config.WebhookSettings.Interval.Duration)
}

func (s *integrationTestSuite) TestExternalIntegrationsConfig() {
	t := s.T()

	// create a test http server to act as the Jira and Zendesk server
	srvURL := startExternalServiceWebServer(t)

	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusOK)

	config := s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, srvURL, config.Integrations.Jira[0].URL)
	require.Equal(t, "ok", config.Integrations.Jira[0].Username)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Jira[0].APIToken)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// add a second, disabled Jira integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 2)

	// first integration
	require.Equal(t, srvURL, config.Integrations.Jira[0].URL)
	require.Equal(t, "ok", config.Integrations.Jira[0].Username)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Jira[0].APIToken)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// second integration
	require.Equal(t, srvURL, config.Integrations.Jira[1].URL)
	require.Equal(t, "ok", config.Integrations.Jira[1].Username)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Jira[1].APIToken)
	require.Equal(t, "qux2", config.Integrations.Jira[1].ProjectKey)
	require.False(t, config.Integrations.Jira[1].EnableSoftwareVulnerabilities)

	// make an unrelated appconfig change, should not remove the integrations
	var appCfgResp appConfigResponse
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"org_info": {
			"org_name": "test-integrations"
		}
	}`), http.StatusOK, &appCfgResp)
	require.Equal(t, "test-integrations", appCfgResp.OrgInfo.OrgName)
	require.Len(t, appCfgResp.Integrations.Jira, 2)

	// delete first Jira integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux2", config.Integrations.Jira[0].ProjectKey)

	// replace Jira integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "ok",
					"project_key": "qux",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.False(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// try adding Jira integration without sending API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "ok",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusBadRequest)

	// try adding Jira integration with masked API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "ok",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": %q,
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusBadRequest)

	// edit Jira integration without sending API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// edit Jira integration with masked API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": %q,
					"project_key": "qux",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.False(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// edit Jira integration sending explicit "" as API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// unknown fields fails as bad request
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"UNKNOWN_FIELD": "foo"
			}]
		}
	}`, srvURL)), http.StatusBadRequest)

	// unknown project key fails as bad request
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": %q,
					"project_key": "qux3",
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusBadRequest)

	// cannot have two integrations enabled at the same time
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "bar2",
					"project_key": "qux2",
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// cannot have two jira integrations with the same project key
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "bar2",
					"project_key": "qux",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// even disabled integrations are tested for Jira connection and credentials,
	// so this fails because the 2nd one uses the "fail" username.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "fail",
					"api_token": "bar2",
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusBadRequest)

	// cannot enable webhook with a jira integration already enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`), http.StatusUnprocessableEntity)

	// disable jira, now we can enable webhook
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
		"jira": [{
			"url": %q,
			"username": "ok",
			"api_token": "bar",
			"project_key": "qux",
			"enable_software_vulnerabilities": false
		}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusOK)

	// cannot enable jira with webhook already enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// disable webhook, enable jira with wrong credentials
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "fail",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusBadRequest)

	// update jira config to correct credentials (need to disable webhook too as
	// last request failed)
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusOK)

	// if no jira nor zendesk integrations are provided, does not remove integrations
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"integrations": {}
	}`), http.StatusOK, &appCfgResp)
	require.Len(t, appCfgResp.Integrations.Jira, 1)

	// if explicitly-empty arrays are provided, remove all integrations
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"integrations": {
			"jira": [],
			"zendesk": []
		}
	}`), http.StatusOK, &appCfgResp)
	require.Len(t, appCfgResp.Integrations.Jira, 0)

	// set environmental varible to use Zendesk test client
	t.Setenv("TEST_ZENDESK_CLIENT", "true")
	// create zendesk integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, srvURL, config.Integrations.Zendesk[0].URL)
	require.Equal(t, "ok@example.com", config.Integrations.Zendesk[0].Email)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Zendesk[0].APIToken)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// add a second, disabled Zendesk integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "test123@example.com",
					"api_token": "ok",
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 2)

	// first integration
	require.Equal(t, srvURL, config.Integrations.Zendesk[0].URL)
	require.Equal(t, "ok@example.com", config.Integrations.Zendesk[0].Email)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Zendesk[0].APIToken)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// second integration
	require.Equal(t, srvURL, config.Integrations.Zendesk[1].URL)
	require.Equal(t, "test123@example.com", config.Integrations.Zendesk[1].Email)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Zendesk[1].APIToken)
	require.Equal(t, int64(123), config.Integrations.Zendesk[1].GroupID)
	require.False(t, config.Integrations.Zendesk[1].EnableSoftwareVulnerabilities)

	// make an unrelated appconfig change, should not remove the integrations
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"org_info": {
			"org_name": "test-integrations-zendesk"
		}
	}`), http.StatusOK, &appCfgResp)
	require.Equal(t, "test-integrations-zendesk", appCfgResp.OrgInfo.OrgName)
	require.Len(t, appCfgResp.Integrations.Zendesk, 2)

	// delete first Zendesk integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "test123@example.com",
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(123), config.Integrations.Zendesk[0].GroupID)

	// replace Zendesk integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.False(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// try adding Zendesk integration without sending API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "test123@example.com",
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusBadRequest)

	// try adding Zendesk integration with masked API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "test123@example.com",
					"api_token": %q,
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusBadRequest)

	// edit Zendesk integration without sending API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// edit Zendesk integration with masked API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": %q,
					"group_id": 122,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.False(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// edit Zendesk integration with explicit "" API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// unknown fields fails as bad request
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"UNKNOWN_FIELD": "foo"
			}]
		}
	}`, srvURL)), http.StatusBadRequest)

	// unknown group id fails as bad request
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 999,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusBadRequest)

	// cannot have two zendesk integrations enabled at the same time
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "not.ok@example.com",
					"api_token": "ok",
					"group_id": 123,
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// cannot have two zendesk integrations with the same group id
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "not.ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// even disabled integrations are tested for Zendesk connection and credentials,
	// so this fails because the 2nd one uses the "fail" token.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "not.ok@example.com",
					"api_token": "fail",
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusBadRequest)

	// cannot enable webhook with a zendesk integration already enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`), http.StatusUnprocessableEntity)

	// disable zendesk, now we can enable webhook
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": false
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusOK)

	// cannot enable zendesk with webhook already enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// disable webhook, enable zendesk with wrong credentials
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "not.ok@example.com",
				"api_token": "fail",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusBadRequest)

	// update zendesk config to correct credentials (need to disable webhook too as
	// last request failed)
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusOK)

	// can have jira enabled and zendesk disabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}],
			"zendesk": [{
				"url": %[1]q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": false
			}]
		}
	}`, srvURL)), http.StatusOK)
	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.False(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// can have jira disabled and zendesk enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": false
			}],
			"zendesk": [{
				"url": %[1]q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusOK)
	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.False(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// cannot have both jira enabled and zendesk enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}],
			"zendesk": [{
				"url": %[1]q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// if no jira nor zendesk integrations are provided, does not remove integrations
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"integrations": {}
	}`), http.StatusOK, &appCfgResp)
	require.Len(t, appCfgResp.Integrations.Jira, 1)
	require.Len(t, appCfgResp.Integrations.Zendesk, 1)

	// remove all integrations on exit, so that other tests can enable the
	// webhook as needed
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"integrations": {
		"jira": [],
		"zendesk": []
		}
	}`), http.StatusOK)
	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 0)

	// enable webhooks
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "http://some/url"
    			},
	    		"failing_policies_webhook": {
	     	 		"enable_failing_policies_webhook": true,
     		 		"destination_url": "http://some/url",
				"host_batch_size": 1000
	    		},
	    		"host_status_webhook": {
	     	 		"enable_host_status_webhook": true,
	     	 		"destination_url": "http://some/url",
					  "host_percentage": 2,
						"days_count": 1
	    		}
		}
	}`), http.StatusOK)
	config = s.getConfig()
	require.True(t, config.WebhookSettings.ActivitiesWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.ActivitiesWebhook.DestinationURL)
	require.True(t, config.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	require.True(t, config.WebhookSettings.HostStatusWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.HostStatusWebhook.DestinationURL)
}

func (s *integrationTestSuite) TestGoogleCalendarIntegrations() {
	t := s.T()
	email := "service-account@example.com"
	privateKey := "-----BEGIN PRIVATE KEY-----\nXXXXX\n-----END"
	domain := "example.com"
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": %q
			}]
		}
	}`, email, privateKey, domain,
		)), http.StatusOK,
	)

	appConfig := s.getConfig()
	require.Len(t, appConfig.Integrations.GoogleCalendar, 1)
	assert.True(t, appConfig.Integrations.GoogleCalendar[0].ApiKey.IsMasked())
	assert.Equal(t, domain, appConfig.Integrations.GoogleCalendar[0].Domain)

	// Add 2nd config -- not allowed at this time
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": %q
			},
			{
				"api_key_json": {
					"client_email": "bozo@example.com",
					"private_key": "abc"
				},
				"domain": "example.com"
			}]
		}
	}`, email, privateKey, domain,
		)), http.StatusUnprocessableEntity,
	)

	// Make an unrelated config change, should not remove the integrations
	var appCfgResp appConfigResponse
	s.DoJSON(
		"PATCH", "/api/v1/fleet/config", json.RawMessage(
			`{
		"org_info": {
			"org_name": "test-google-calendar-integrations"
		}
	}`,
		), http.StatusOK, &appCfgResp,
	)
	require.Equal(t, "test-google-calendar-integrations", appCfgResp.OrgInfo.OrgName)
	require.Len(t, appCfgResp.Integrations.GoogleCalendar, 1)

	// Update calendar config
	domain = "new.com"
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": %q
			}]
		}
	}`, email, privateKey, domain,
		)), http.StatusOK,
	)
	appConfig = s.getConfig()
	require.Len(t, appConfig.Integrations.GoogleCalendar, 1)
	assert.True(t, appConfig.Integrations.GoogleCalendar[0].ApiKey.IsMasked())
	assert.Equal(t, domain, appConfig.Integrations.GoogleCalendar[0].Domain)

	// Clearing other integrations does not clear Google Calendar integration
	appCfgResp = appConfigResponse{}
	s.DoJSON(
		"PATCH", "/api/v1/fleet/config", json.RawMessage(
			`{
		"integrations": {
			"jira": [],
			"zendesk": []
		}
	}`,
		), http.StatusOK, &appCfgResp,
	)
	require.Len(t, appCfgResp.Integrations.GoogleCalendar, 1)

	// Clearing Google Calendar integration
	appCfgResp = appConfigResponse{}
	s.DoJSON(
		"PATCH", "/api/v1/fleet/config", json.RawMessage(
			`{
		"integrations": {
			"google_calendar": []
		}
	}`,
		), http.StatusOK, &appCfgResp,
	)
	assert.Empty(t, appCfgResp.Integrations.GoogleCalendar)

	// Try adding Google Calendar integration without sending private key -- not allowed
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q
				},
				"domain": %q
			}]
		}
	}`, email, domain,
		)), http.StatusUnprocessableEntity,
	)

	// Empty email -- not allowed
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": " ",
					"private_key": %q
				},
				"domain": %q
			}]
		}
	}`, privateKey, domain,
		)), http.StatusUnprocessableEntity,
	)

	// Empty domain -- not allowed
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": ""
			}]
		}
	}`, email, privateKey,
		)), http.StatusUnprocessableEntity,
	)

	// Unknown fields fails as bad request
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": %q,
				"foo": "bar"
			}]
		}
	}`, email, privateKey, domain,
		)), http.StatusBadRequest,
	)

	// Null api_key_json -- fails validation
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": null,
				"domain": %q
			}]
		}
	}`, domain,
		)), http.StatusUnprocessableEntity,
	)
}

func (s *integrationTestSuite) TestQueriesBadRequests() {
	t := s.T()

	reqQuery := &fleet.QueryPayload{
		Name:  new("existing query"),
		Query: new("select 42;"),
	}
	createQueryResp := fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	require.NotNil(t, createQueryResp.Query)
	existingQueryID := createQueryResp.Query.ID
	defer s.cleanupQuery(existingQueryID)

	for _, tc := range []struct {
		tname    string
		name     string
		query    string
		platform string
		logging  string
	}{
		{
			tname: "empty name",
			name:  " ", // #3704
			query: "select 42;",
		},
		{
			tname: "empty query",
			name:  "Some name",
			query: "",
		},
		{
			tname: "Invalid query",
			name:  "Invalid query",
			query: "",
		},
		{
			tname:    "unsupported platform",
			name:     "bad query",
			query:    "select 1",
			platform: "oops",
		},
		{
			tname:    "unsupported platform",
			name:     "bad query",
			query:    "select 1",
			platform: "charles,darwin",
		},
		{
			tname:    "missing platform comma delimeter",
			name:     "bad query",
			query:    "select 1",
			platform: "linuxdarwin",
		},
		{
			tname:    "missing platform comma delimeter",
			name:     "bad query",
			query:    "select 1",
			platform: "windows darwin",
		},
		{
			tname:   "invalid logging value",
			name:    "bad query",
			query:   "select 1",
			logging: "foobar",
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			reqQuery := &fleet.QueryPayload{
				Name:     new(tc.name),
				Query:    new(tc.query),
				Platform: new(tc.platform),
				Logging:  new(tc.logging),
			}
			createQueryResp := fleet.CreateQueryResponse{}
			s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusBadRequest, &createQueryResp)
			require.Nil(t, createQueryResp.Query)

			payload := fleet.QueryPayload{
				Name:     new(tc.name),
				Query:    new(tc.query),
				Platform: new(tc.platform),
				Logging:  new(tc.logging),
			}
			mResp := fleet.ModifyQueryResponse{}
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", existingQueryID), &payload, http.StatusBadRequest, &mResp)
			require.Nil(t, mResp.Query)
			// TODO – add checks for specific errors
		})
	}

	for _, tc := range []struct {
		tname string
		body  string
	}{
		{
			tname: "null name",
			body:  `{"name": null, "query": "SELECT 1;"}`,
		},
		{
			tname: "null query",
			body:  `{"name": "Test", "query": null}`,
		},
		{
			tname: "null name and query",
			body:  `{"name": null, "query": null}`,
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			resp := s.DoRaw("POST", "/api/latest/fleet/queries", []byte(tc.body), http.StatusBadRequest)
			resp.Body.Close()
		})
	}

	for _, tc := range []struct {
		tname string
		body  string
	}{
		{
			tname: "spec null name",
			body:  `{"specs": [{"name": null, "query": "SELECT 1;"}]}`,
		},
		{
			tname: "spec null query",
			body:  `{"specs": [{"name": "Test", "query": null}]}`,
		},
		{
			tname: "spec null name and query",
			body:  `{"specs": [{"name": null, "query": null}]}`,
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			resp := s.DoRaw("POST", "/api/latest/fleet/spec/queries", []byte(tc.body), http.StatusBadRequest)
			resp.Body.Close()
		})
	}
}

func (s *integrationTestSuite) TestPacksBadRequests() {
	t := s.T()

	reqPacks := &fleet.PackPayload{
		Name: new("existing pack"),
	}
	createPackResp := createPackResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs", reqPacks, http.StatusOK, &createPackResp)
	existingPackID := createPackResp.Pack.ID

	for _, tc := range []struct {
		tname string
		name  string
	}{
		{
			tname: "empty name",
			name:  " ", // #3704
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			reqQuery := &fleet.PackPayload{
				Name: new(tc.name),
			}
			createPackResp := createPackResponse{}
			s.DoJSON("POST", "/api/latest/fleet/packs", reqQuery, http.StatusBadRequest, &createPackResp)

			payload := fleet.PackPayload{
				Name: new(tc.name),
			}
			mResp := modifyPackResponse{}
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/%d", existingPackID), &payload, http.StatusBadRequest, &mResp)
		})
	}

	t.Run("null name on create", func(t *testing.T) {
		res := s.Do("POST", "/api/latest/fleet/packs", json.RawMessage(`{"name": null}`), http.StatusBadRequest)
		assertBodyContains(t, res, "pack name cannot be empty")
	})
}

func (s *integrationTestSuite) TestPremiumEndpointsWithoutLicense() {
	t := s.T()

	// list teams, none
	var listResp listTeamsResponse
	s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusPaymentRequired, &listResp)
	assert.Len(t, listResp.Teams, 0)

	// get team
	var getResp getTeamResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/123", nil, http.StatusPaymentRequired, &getResp)
	assert.Nil(t, getResp.Team)

	// create team
	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// modify team
	s.DoJSON("PATCH", "/api/latest/fleet/teams/123", fleet.TeamPayload{}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// delete team
	var delResp deleteTeamResponse
	s.DoJSON("DELETE", "/api/latest/fleet/teams/123", nil, http.StatusPaymentRequired, &delResp)

	// apply team specs
	var specResp applyTeamSpecsResponse
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "newteam", Secrets: &[]fleet.EnrollSecret{{Secret: "ABC"}}}}}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusPaymentRequired, &specResp)

	// modify team agent options
	s.DoJSON("POST", "/api/latest/fleet/teams/123/agent_options", nil, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// list team users
	var usersResp listUsersResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/123/users", nil, http.StatusPaymentRequired, &usersResp, "page", "1")
	assert.Len(t, usersResp.Users, 0)

	// add team users
	s.DoJSON("PATCH", "/api/latest/fleet/teams/123/users", modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: 1}}}}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// delete team users
	s.DoJSON("DELETE", "/api/latest/fleet/teams/123/users", modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: 1}}}}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// get team enroll secrets
	var secResp teamEnrollSecretsResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/123/secrets", nil, http.StatusPaymentRequired, &secResp)
	assert.Len(t, secResp.Secrets, 0)

	// modify team enroll secrets
	s.DoJSON("PATCH", "/api/latest/fleet/teams/123/secrets", modifyTeamEnrollSecretsRequest{Secrets: []fleet.EnrollSecret{{Secret: "DEF"}}}, http.StatusPaymentRequired, &secResp)
	assert.Len(t, secResp.Secrets, 0)

	// get apple BM configuration
	var appleBMResp getAppleBMResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple_bm", nil, http.StatusPaymentRequired, &appleBMResp)
	assert.Nil(t, appleBMResp.AppleBM)

	// batch-apply an empty set of MDM profiles succeeds even though MDM is not
	// enabled, because it wouldn't change anything (and it needs to support the
	// case where `fleetctl get config`'s output is used as input to `fleetctl
	// apply`).
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch", nil, http.StatusNoContent)

	// batch-apply a non-empty set of MDM profiles fails
	res := s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch",
		map[string]interface{}{"profiles": [][]byte{[]byte(`xyz`)}}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.ErrMDMNotConfigured.Error())

	// update MDM disk encryption
	_ = s.Do("POST", "/api/latest/fleet/disk_encryption", fleet.MDMAppleSettingsPayload{}, http.StatusPaymentRequired)

	// update MDM host name template
	_ = s.Do("POST", "/api/latest/fleet/host_name_template", updateHostNameTemplateRequest{}, http.StatusPaymentRequired)

	// Turn on MDM.
	ctx := t.Context()
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	origEnabledAndConfigured := appCfg.MDM.EnabledAndConfigured
	appCfg.MDM.EnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	defer func() {
		appCfg.MDM.EnabledAndConfigured = origEnabledAndConfigured
		err = s.ds.SaveAppConfig(ctx, appCfg)
		require.NoError(t, err)
	}()

	// device migrate mdm endpoint returns an error if not premium
	createHostAndDeviceToken(t, s.ds, "some-token")
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "some-token"), nil, http.StatusPaymentRequired)

	// uploading a DDM declaration with a Fleet variable returns a license error
	// (single profile upload endpoint)
	ddmWithFleetVar := []byte(`{
		"Type": "com.apple.configuration.management.test",
		"Identifier": "com.example.fleetvar-test",
		"Payload": {"Value": "$FLEET_VAR_HOST_HARDWARE_SERIAL"}
	}`)
	body, headers := generateNewProfileMultipartRequest(t, "fleetvar-test.json", ddmWithFleetVar, s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusPaymentRequired, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Requires Fleet Premium license")

	// uploading a DDM declaration with a Fleet variable returns a license error
	// (batch profiles endpoint)
	res = s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: ddmWithFleetVar},
	}}, http.StatusPaymentRequired)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Requires Fleet Premium license")

	// software titles
	// a normal request works fine
	var resp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp)
	// TODO: there's a race condition that makes this number change from
	// 0-3, commenting for now since it's not really relevant for this
	// test (we only care about the response status)
	// require.NotEmpty(t, 0, resp.Count)
	// require.Nil(t, resp.SoftwareTitles)

	// a request with a team_id parameter returns a license error
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{}, http.StatusPaymentRequired, &resp,
		"team_id", "1",
	)

	// a request with a premium vulnerability filter returns a license error
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{fleet.SoftwareTitleListOptions{VulnerableOnly: true, MinimumCVSS: 7.5}}, http.StatusPaymentRequired, &resp,
	)
	verResp := listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, MinimumCVSS: 7.5}}, http.StatusPaymentRequired, &verResp,
	)
	countResp := countSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/count",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, MinimumCVSS: 7.5}}, http.StatusPaymentRequired, &countResp,
	)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{fleet.SoftwareTitleListOptions{VulnerableOnly: true, MaximumCVSS: 7.5}}, http.StatusPaymentRequired, &resp,
	)
	verResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, MaximumCVSS: 7.5}}, http.StatusPaymentRequired, &verResp,
	)
	countResp = countSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/count",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, MaximumCVSS: 7.5}}, http.StatusPaymentRequired, &countResp,
	)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{fleet.SoftwareTitleListOptions{VulnerableOnly: true, KnownExploit: true}}, http.StatusPaymentRequired, &resp,
	)
	verResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, KnownExploit: true}}, http.StatusPaymentRequired, &verResp,
	)
	countResp = countSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/count",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, KnownExploit: true}}, http.StatusPaymentRequired, &countResp,
	)

	// lock/unlock/wipe a host. Wipe is Premium-only only for non-Android platforms (Android COBO wipe is available on Fleet Free).
	wipeHost := s.createHosts(t, "darwin")[0]
	s.Do("POST", "/api/v1/fleet/hosts/123/lock", nil, http.StatusPaymentRequired)
	s.Do("POST", "/api/v1/fleet/hosts/123/unlock", nil, http.StatusPaymentRequired)
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/hosts/%d/wipe", wipeHost.ID), nil, http.StatusPaymentRequired)

	// try to update the enable_release_device_manually setting, requires premium.
	s.Do("PATCH", "/api/v1/fleet/setup_experience", fleet.MDMAppleSetupPayload{EnableReleaseDeviceManually: new(true)}, http.StatusPaymentRequired)

	res = s.Do("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_setup": { "enable_release_device_manually": true } }
	}`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "missing or invalid license")

	res = s.Do("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "apple_require_hardware_attestation": true }
	}`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "missing or invalid license")

	res = s.Do("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "windows_migration_enabled": true }
	}`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "missing or invalid license")

	// list API endpoints requires premium license
	var listAPIEndpointsResp listAPIEndpointsResponse
	s.DoJSON("GET", "/api/latest/fleet/rest_api", nil, http.StatusPaymentRequired, &listAPIEndpointsResp)
}

func (s *integrationTestSuite) TestScriptsEndpointsWithoutLicense() {
	t := s.T()

	// this is just checking that the endpoints do not fail with "no license", the actual tests
	// for scripts endpoints are in the enterprise integrations tests.

	// run a script
	var runResp fleet.RunScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: 1, ScriptContents: "echo foo"}, http.StatusNotFound, &runResp)

	// run a script sync
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: 1, ScriptContents: "echo foo"}, http.StatusNotFound, &runResp)

	// get script result
	var scriptResultResp fleet.GetScriptResultResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/test-id", nil, http.StatusNotFound, &scriptResultResp)

	// create a saved script
	body, headers := generateNewScriptMultipartRequest(t,
		"myscript.sh", []byte(`echo "hello"`), s.token, nil)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)

	// run a saved script by name without team id (should fail host not found)
	res := s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.RunScriptSyncRequest{ScriptName: "myscript.sh"}, http.StatusNotFound)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Host was not found in the datastore")

	// run a saved script by name with team id (should fail with license error)
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.RunScriptSyncRequest{ScriptName: "myscript.sh", TeamID: 1}, http.StatusPaymentRequired)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Requires Fleet Premium license")

	// scripts containing Fleet variables require a premium license
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: 1, ScriptContents: "echo $FLEET_VAR_HOST_UUID"}, http.StatusPaymentRequired)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Requires Fleet Premium license")

	body, headers = generateNewScriptMultipartRequest(t,
		"varscript.sh", []byte("echo $FLEET_VAR_HOST_UUID"), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusPaymentRequired, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Requires Fleet Premium license")

	res = s.Do("POST", "/api/v1/fleet/scripts/batch", fleet.BatchSetScriptsRequest{Scripts: []fleet.ScriptPayload{
		{Name: "vars.sh", ScriptContents: []byte("echo $FLEET_VAR_HOST_UUID")},
	}}, http.StatusPaymentRequired)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Requires Fleet Premium license")

	// delete a saved script
	var delScriptResp fleet.DeleteScriptResponse
	s.DoJSON("DELETE", "/api/latest/fleet/scripts/123", nil, http.StatusNotFound, &delScriptResp)

	// list saved scripts
	var listScriptsResp fleet.ListScriptsResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts", nil, http.StatusOK, &listScriptsResp, "per_page", "10")

	// get a saved script
	var getScriptResp fleet.GetScriptResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts/123", nil, http.StatusNotFound, &getScriptResp)

	// get host script details
	var getHostScriptDetailsResp fleet.GetHostScriptDetailsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/123/scripts", nil, http.StatusNotFound, &getHostScriptDetailsResp)

	// batch set scripts
	s.Do("POST", "/api/v1/fleet/scripts/batch", fleet.BatchSetScriptsRequest{Scripts: nil}, http.StatusOK)
}

// TestGlobalPoliciesBrowsing tests that team users can browse (read) global policies (see #3722).
func (s *integrationTestSuite) TestGlobalPoliciesBrowsing() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team_for_global_policies_browsing",
		Description: "desc team1",
	})
	require.NoError(t, err)

	gpParams0 := fleet.GlobalPolicyRequest{
		Name:  "global policy",
		Query: "select * from osquery;",
	}
	gpResp0 := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams0, http.StatusOK, &gpResp0)
	require.NotNil(t, gpResp0.Policy)

	email := "team.observer@example.com"
	teamObserver := &fleet.User{
		Name:       "team observer user",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team,
				Role: fleet.RoleObserver,
			},
		},
	}
	password := test.GoodPassword
	require.NoError(t, teamObserver.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), teamObserver)
	require.NoError(t, err)

	oldToken := s.token
	s.token = s.getTestToken(email, password)
	t.Cleanup(func() {
		s.token = oldToken
	})

	policiesResponse := fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "global policy", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery;", policiesResponse.Policies[0].Query)
}

// TestGetPolicyByIDCrossTeamAccess verifies that GET /api/latest/fleet/policies/{id}
// returns a team policy to a user that has access to the team it belongs to and
// rejects the request with 403 when the requesting user has no role on that team.
// This guards the security fix for the "Cross-Team Policy Data Exposure" disclosure.
func (s *integrationTestSuite) TestGetPolicyByIDCrossTeamAccess() {
	t := s.T()
	ctx := context.Background()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_team2"})
	require.NoError(t, err)

	policy, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:  t.Name() + "_team1_policy",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)

	password := test.GoodPassword

	team1Observer := &fleet.User{
		Name:  "team1 observer",
		Email: t.Name() + "_team1_observer@example.com",
		Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleObserver}},
	}
	require.NoError(t, team1Observer.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(ctx, team1Observer)
	require.NoError(t, err)

	team2Observer := &fleet.User{
		Name:  "team2 observer",
		Email: t.Name() + "_team2_observer@example.com",
		Teams: []fleet.UserTeam{{Team: *team2, Role: fleet.RoleObserver}},
	}
	require.NoError(t, team2Observer.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(ctx, team2Observer)
	require.NoError(t, err)

	policyURL := fmt.Sprintf("/api/latest/fleet/policies/%d", policy.ID)

	// A user with access to the policy's team can read it.
	s.setTokenForTest(t, team1Observer.Email, password)
	var okResp fleet.GetPolicyByIDResponse
	s.DoJSON("GET", policyURL, fleet.GetPolicyByIDRequest{}, http.StatusOK, &okResp)
	require.NotNil(t, okResp.Policy)
	assert.Equal(t, policy.ID, okResp.Policy.ID)
	require.NotNil(t, okResp.Policy.TeamID)
	assert.Equal(t, team1.ID, *okResp.Policy.TeamID)

	// A user with no role on the policy's team is forbidden from reading it.
	s.setTokenForTest(t, team2Observer.Email, password)
	s.Do("GET", policyURL, fleet.GetPolicyByIDRequest{}, http.StatusForbidden)
}

func (s *integrationTestSuite) TestTeamPoliciesTeamNotExists() {
	t := s.T()

	teamPoliciesResponse := fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", 9999999), nil, http.StatusNotFound, &teamPoliciesResponse)
	require.Len(t, teamPoliciesResponse.Policies, 0)

	deleteTeamPoliciesResp := fleet.DeleteTeamPoliciesResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", 9999999), fleet.DeleteTeamPoliciesRequest{IDs: []uint{1, 1000}}, http.StatusNotFound, &deleteTeamPoliciesResp)
}

func (s *integrationTestSuite) TestSessionInfo() {
	t := s.T()

	ssn := createSession(t, 1, s.ds)

	var meResp getUserResponse
	resp := s.DoRawWithHeaders("GET", "/api/latest/fleet/me", nil, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ssn.Key),
	})
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&meResp))
	assert.Equal(t, uint(1), meResp.User.ID)

	// get info about session
	var getResp getInfoAboutSessionResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, ssn.ID, getResp.SessionID)
	assert.Equal(t, uint(1), getResp.UserID)

	// get info about session
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID+1), nil, http.StatusNotFound, &getResp)

	// delete session
	var delResp deleteSessionResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID), nil, http.StatusOK, &delResp)

	// delete session - non-existing
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID), nil, http.StatusNotFound, &delResp)
}

// Free tier must not expose or accept fleet_desktop.sso_enabled, and must not
// let a value stored under a premium license (before a downgrade) block
// unrelated config changes.
func (s *integrationTestSuite) TestFleetDesktopSSOFreeTier() {
	t := s.T()
	ctx := t.Context()

	// PATCH attempting to enable it is rejected with the license error
	res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"fleet_desktop":{"sso_enabled":true}}`), http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "missing or invalid license")

	// seed a value as if it had been set while premium, then downgraded
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.FleetDesktop.SSOEnabled = true
	require.NoError(t, s.ds.SaveAppConfig(ctx, appCfg))

	// GET /config rebuilds FleetDesktopSettings field by field and premium-gates
	// the values, so the stored flag must read back as false here
	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.False(t, acResp.FleetDesktop.SSOEnabled)

	// an unrelated PATCH still succeeds and resets the stored value rather than
	// failing on a premium-only setting left over from the downgrade
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"host_expiry_settings":{"host_expiry_enabled":false}}`), http.StatusOK, &acResp)
	require.False(t, acResp.FleetDesktop.SSOEnabled)

	appCfg, err = s.ds.AppConfig(ctx)
	require.NoError(t, err)
	require.False(t, appCfg.FleetDesktop.SSOEnabled)
}

func (s *integrationTestSuite) TestAppConfig() {
	t := s.T()
	ctx := context.Background()

	// get the app config
	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Equal(t, "free", acResp.License.Tier)
	assert.Equal(t, "FleetTest", acResp.OrgInfo.OrgName) // set in SetupSuite
	assert.False(t, acResp.MDM.AppleBMTermsExpired)
	assert.False(t, acResp.ActivityExpirySettings.ActivityExpiryEnabled)
	assert.Zero(t, acResp.ActivityExpirySettings.ActivityExpiryWindow)
	assert.False(t, acResp.ServerSettings.AIFeaturesDisabled)
	assert.False(t, acResp.GitOpsConfig.GitopsModeEnabled)
	assert.Zero(t, acResp.GitOpsConfig.RepositoryURL)
	expectedMaxPackageSize := config.TestConfig().Server.MaxInstallerSizeBytes
	assert.Equal(t, expectedMaxPackageSize, acResp.MaxSoftwarePackageSize)

	// set the apple BM terms expired flag, and the enabled and configured flags,
	// we'll check again at the end of this test to make sure they weren't
	// modified by any PATCH request (it cannot be set via this endpoint).
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.MDM.AppleBMTermsExpired = true
	appCfg.MDM.AppleBMEnabledAndConfigured = true
	appCfg.MDM.EnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)

	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.AppleBMTermsExpired)
	assert.True(t, acResp.MDM.AppleBMEnabledAndConfigured)
	assert.True(t, acResp.MDM.EnabledAndConfigured)

	// no server settings set for the URL, so not possible to test the
	// certificate endpoint
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "org_info": {
        "org_name": "test"
    }
  }`), http.StatusOK, &acResp)
	assert.Equal(t, "test", acResp.OrgInfo.OrgName)
	assert.True(t, acResp.MDM.AppleBMTermsExpired)

	// the global agent options were not modified by the last call, so the
	// corresponding activity should not have been created.
	var listActivities listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listActivities, "order_key", "id", "order_direction", "desc")
	if len(listActivities.Activities) > 1 {
		// if there is an activity, make sure it is not edited_agent_options
		require.NotEqual(t, fleet.ActivityTypeEditedAgentOptions{}.ActivityName(), listActivities.Activities[0].Type)
	}

	// and it did not update the appconfig
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"logger_plugin": "tls"`) // default agent options has this setting

	// Invalid activity expiry window.
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "activity_expiry_settings": {
        "activity_expiry_enabled": true,
        "activity_expiry_window": -1
    }
  }`), http.StatusUnprocessableEntity, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.False(t, acResp.ActivityExpirySettings.ActivityExpiryEnabled)
	require.Zero(t, acResp.ActivityExpirySettings.ActivityExpiryWindow)

	// Valid activity expiry window.
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "activity_expiry_settings": {
        "activity_expiry_enabled": true,
        "activity_expiry_window": 42
    }
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.ActivityExpirySettings.ActivityExpiryEnabled)
	require.Equal(t, 42, acResp.ActivityExpirySettings.ActivityExpiryWindow)

	// preserve_host_activities_on_reenrollment round-trip.
	initialPreserve := acResp.ActivityExpirySettings.PreserveHostActivitiesOnReenrollment
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "activity_expiry_settings": {
        "preserve_host_activities_on_reenrollment": true
    }
  }`), http.StatusOK, &acResp)
	require.True(t, acResp.ActivityExpirySettings.PreserveHostActivitiesOnReenrollment)
	require.True(t, acResp.ActivityExpirySettings.ActivityExpiryEnabled)
	require.Equal(t, 42, acResp.ActivityExpirySettings.ActivityExpiryWindow)

	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "activity_expiry_settings": {
        "preserve_host_activities_on_reenrollment": false
    }
  }`), http.StatusOK, &acResp)
	require.False(t, acResp.ActivityExpirySettings.PreserveHostActivitiesOnReenrollment)

	// Restore initial value to keep subsequent tests order-independent.
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
    "activity_expiry_settings": {
        "preserve_host_activities_on_reenrollment": %t
    }
  }`, initialPreserve)), http.StatusOK, &acResp)
	require.Equal(t, initialPreserve, acResp.ActivityExpirySettings.PreserveHostActivitiesOnReenrollment)

	// Disable AI features.
	acResp = appConfigResponse{}
	s.DoJSON(
		"PATCH", "/api/latest/fleet/config", json.RawMessage(
			`{
    "server_settings": {
        "ai_features_disabled": true
    }
  }`,
		), http.StatusOK, &acResp,
	)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.ServerSettings.AIFeaturesDisabled)

	// test a change that does clear the agent options (the field is provided but empty).
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": {}
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Equal(t, string(*acResp.AgentOptions), "{}")
	assert.True(t, acResp.MDM.AppleBMTermsExpired)

	// test a change that does modify the agent options.
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"views": {"foo": "bar"}} }
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listActivities, "order_key", "id", "order_direction", "desc")
	require.True(t, len(listActivities.Activities) > 1)
	require.Equal(t, fleet.ActivityTypeEditedAgentOptions{}.ActivityName(), listActivities.Activities[0].Type)
	require.NotNil(t, listActivities.Activities[0].Details)
	assert.JSONEq(t, `{"global": true, "fleet_id": null, "fleet_name": null, "team_id": null, "team_name": null}`, string(*listActivities.Activities[0].Details))

	// try to set invalid agent options
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"nope": true} }
  }`), http.StatusBadRequest, &acResp)
	// did not update the appconfig
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotContains(t, string(*acResp.AgentOptions), `"nope"`)

	// try to set an invalid agent options logger_tls_endpoint (must start with "/")
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"options": {"logger_tls_endpoint": "not-a-rooted-path"}} }
  }`), http.StatusBadRequest, &acResp)
	// did not update the appconfig
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotContains(t, string(*acResp.AgentOptions), `"not-a-rooted-path"`)

	// try to set a valid agent options logger_tls_endpoint
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"options": {"logger_tls_endpoint": "/rooted-path"}} }
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"/rooted-path"`)

	// force-set invalid agent options
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"nope": true} }
  }`), http.StatusOK, &acResp, "force", "true")
	require.Contains(t, string(*acResp.AgentOptions), `"nope"`)

	// dry-run valid agent options
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"views":{"yep": "ok"}} }
  }`), http.StatusOK, &acResp, "dry_run", "true")
	require.NotContains(t, string(*acResp.AgentOptions), `"yep"`)
	require.Contains(t, string(*acResp.AgentOptions), `"nope"`)

	// dry-run invalid agent options
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"invalid": true} }
  }`), http.StatusBadRequest, &acResp, "dry_run", "true")
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotContains(t, string(*acResp.AgentOptions), `"invalid"`)

	// set valid agent options command-line flag
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "command_line_flags": {"enable_tables":"table1"}}
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"enable_tables": "table1"`)

	// set invalid agent options command-line flag
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "command_line_flags": {"no_such_flag":true}}
  }`), http.StatusBadRequest, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"enable_tables": "table1"`)
	require.NotContains(t, string(*acResp.AgentOptions), `"no_such_flag"`)

	// set invalid value for a valid agent options command-line flag
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "command_line_flags": {"enable_tables":true}}
  }`), http.StatusBadRequest, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"enable_tables": "table1"`)

	// force-set invalid value for a valid agent options command-line flag
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "command_line_flags": {"enable_tables":true}}
  }`), http.StatusOK, &acResp, "force", "true")
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotContains(t, string(*acResp.AgentOptions), `"enable_tables": "table1"`)
	require.Contains(t, string(*acResp.AgentOptions), `"enable_tables": true`)

	// dry-run valid appconfig that uses legacy settings (returns error)
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"host_settings": { "additional_queries": {"foo": "bar"} }
  }`), http.StatusBadRequest, &acResp, "dry_run", "true")
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Nil(t, acResp.Features.AdditionalQueries)

	// without dry-run, the valid appconfig that uses legacy settings is accepted
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"host_settings": { "additional_queries": {"foo": "bar"} }
  }`), http.StatusOK, &acResp, "dry_run", "false")
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp.Features.AdditionalQueries)
	require.Contains(t, string(*acResp.Features.AdditionalQueries), `"foo": "bar"`)

	var verResp versionResponse
	s.DoJSON("GET", "/api/latest/fleet/version", nil, http.StatusOK, &verResp)
	assert.NotEmpty(t, verResp.Branch)

	// get enroll secrets, none yet
	var specResp getEnrollSecretSpecResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/enroll_secret", nil, http.StatusOK, &specResp)
	assert.Empty(t, specResp.Spec.Secrets)

	seenActivitiesIDs := map[uint]struct{}{}
	activityName := fleet.ActivityTypeEditedEnrollSecrets{}.ActivityName()

	// apply spec, one secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: "XYZ"}},
		},
	}, http.StatusOK, &applyResp)

	// adding a new secret should create a new activity entry
	seenActivitiesIDs[s.lastActivityMatches(activityName, "", 0)] = struct{}{}
	require.Len(t, seenActivitiesIDs, 1)

	// applying the same secret again shouldn't create a new activity since we are only interested in mutations
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: "XYZ"}},
		},
	}, http.StatusOK, &applyResp)

	seenActivitiesIDs[s.lastActivityMatches(activityName, "", 0)] = struct{}{}
	require.Len(t, seenActivitiesIDs, 1)

	// apply spec, too many secrets
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: createEnrollSecrets(t, fleet.MaxEnrollSecretsCount+1),
		},
	}, http.StatusUnprocessableEntity, &applyResp)

	// apply spec, empty and whitespace-only secrets are rejected
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: ""}}},
	}, http.StatusUnprocessableEntity, &applyResp)
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: "   "}}},
	}, http.StatusUnprocessableEntity, &applyResp)

	// error conditions should create new activities
	seenActivitiesIDs[s.lastActivityMatches(activityName, "", 0)] = struct{}{}
	require.Len(t, seenActivitiesIDs, 1)

	// get enroll secrets, one
	s.DoJSON("GET", "/api/latest/fleet/spec/enroll_secret", nil, http.StatusOK, &specResp)
	require.Len(t, specResp.Spec.Secrets, 1)
	assert.Equal(t, "XYZ", specResp.Spec.Secrets[0].Secret)

	// remove secret just to prevent affecting other tests
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{},
	}, http.StatusOK, &applyResp)

	// removing the secret should create a new activity entry
	seenActivitiesIDs[s.lastActivityMatches(activityName, "", 0)] = struct{}{}
	require.Len(t, seenActivitiesIDs, 2)

	s.DoJSON("GET", "/api/latest/fleet/spec/enroll_secret", nil, http.StatusOK, &specResp)
	require.Len(t, specResp.Spec.Secrets, 0)

	// try to update the apple bm terms flag via PATCH /config
	// request is ok but modified value is ignored
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "apple_bm_terms_expired": false }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.AppleBMTermsExpired)

	// try to update the mdm configured flags via PATCH /config
	// request is ok but modified value is ignored
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	  "mdm": { "enabled_and_configured": false, "apple_bm_enabled_and_configured": false }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnabledAndConfigured)
	assert.True(t, acResp.MDM.AppleBMEnabledAndConfigured)

	// set the macos disk encryption field, fails due to license
	res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")

	// legacy config
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": { "enable_disk_encryption": true } }
  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")

	// try to set the apple bm default team, which is premium only
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "apple_bm_default_team": "xyz" }
  }`), http.StatusUnprocessableEntity, &acResp)

	// try to set Okta conditional access settings, which is premium only
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"conditional_access": {
			"okta_idp_id": "https://www.okta.com/saml2/service-provider/test",
			"okta_assertion_consumer_service_url": "https://dev-test.okta.com/sso/saml2/test",
			"okta_audience_uri": "https://www.okta.com/saml2/service-provider/test",
			"okta_certificate": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"
		}
  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")

	// try to set the windows updates, which is premium only
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_updates": {"deadline_days": 1, "grace_period_days": 0} }
  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")

	// try to enable Windows MDM, impossible without the WSTEP certs
	// (only set in mdm integrations tests)
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_enabled_and_configured": true }
  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "Please configure Fleet with a certificate and key pair first.")

	// verify that the Apple BM terms expired flag was never modified
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.AppleBMTermsExpired)

	// set the apple BM terms back to false
	appCfg, err = s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.MDM.AppleBMTermsExpired = false
	appCfg.MDM.AppleBMEnabledAndConfigured = false
	appCfg.MDM.EnabledAndConfigured = false
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)

	// set the macos custom settings fields, fails due to MDM not configured
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": { "macos_settings": { "custom_settings": ["foo", "bar"] } }
	  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "Couldn't update apple_settings because MDM features aren't turned on in Fleet.")

	// test setting the default app config we use for new installs (this check
	// ensures that the default config passes the validation)
	var defAppCfg fleet.AppConfig
	defAppCfg.ApplyDefaultsForNewInstalls()
	// must set org name and server settings
	defAppCfg.OrgInfo.OrgName = acResp.OrgInfo.OrgName
	defAppCfg.ServerSettings.ServerURL = acResp.ServerSettings.ServerURL
	s.DoRaw("PATCH", "/api/latest/fleet/config", jsonMustMarshal(t, defAppCfg), http.StatusOK)

	// turn on GitOps mode, premium only
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"gitops": { "gitops_mode_enabled": true, "repository_url": "" }
	  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")
}

func (s *integrationTestSuite) TestQuerySpecs() {
	t := s.T()

	s.lq.On("SetQueryResultsCount", mock.Anything, mock.Anything).Return(nil)

	// list specs, none yet
	var getSpecsResp fleet.GetQuerySpecsResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/queries", nil, http.StatusOK, &getSpecsResp)
	assert.Len(t, getSpecsResp.Specs, 0)

	// get unknown one
	var getSpecResp fleet.GetQuerySpecResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/queries/nonesuch", nil, http.StatusNotFound, &getSpecResp)

	// create some queries via apply specs
	q1 := strings.ReplaceAll(t.Name(), "/", "_")
	q2 := q1 + "_2"
	var applyResp fleet.ApplyQuerySpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q1, Query: "SELECT 1"},
			{Name: q2, Query: "SELECT 2"},
		},
	}, http.StatusOK, &applyResp)

	// get the queries back
	var listQryResp fleet.ListQueriesResponse
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "order_key", "name")
	require.Len(t, listQryResp.Queries, 2)
	assert.Equal(t, q1, listQryResp.Queries[0].Name)
	assert.Equal(t, q2, listQryResp.Queries[1].Name)
	q1ID, q2ID := listQryResp.Queries[0].ID, listQryResp.Queries[1].ID

	// list specs
	s.DoJSON("GET", "/api/latest/fleet/spec/queries", nil, http.StatusOK, &getSpecsResp)
	require.Len(t, getSpecsResp.Specs, 2)
	names := []string{getSpecsResp.Specs[0].Name, getSpecsResp.Specs[1].Name}
	assert.ElementsMatch(t, []string{q1, q2}, names)

	// get specific spec
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/spec/queries/%s", q1), nil, http.StatusOK, &getSpecResp)
	assert.Equal(t, getSpecResp.Spec.Query, "SELECT 1")

	// apply specs again - create q3 and update q2
	q3 := q1 + "_3"
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q2, Query: "SELECT -2"},
			{Name: q3, Query: "SELECT 3"},
		},
	}, http.StatusOK, &applyResp)

	// try to create a query with invalid platform, fail
	q4 := q1 + "_4"
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q4, Query: "SELECT 4", Platform: "not valid"},
		},
	}, http.StatusBadRequest, &applyResp)

	// try to edit a query with invalid platform, fail
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q3, Query: "SELECT 3", Platform: "charles darwin"},
		},
	}, http.StatusBadRequest, &applyResp)

	// list specs - has 3, not 4 (one was an update)
	s.DoJSON("GET", "/api/latest/fleet/spec/queries", nil, http.StatusOK, &getSpecsResp)
	require.Len(t, getSpecsResp.Specs, 3)
	names = []string{getSpecsResp.Specs[0].Name, getSpecsResp.Specs[1].Name, getSpecsResp.Specs[2].Name}
	assert.ElementsMatch(t, []string{q1, q2, q3}, names)

	// get the queries back again
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "order_key", "name")
	require.Len(t, listQryResp.Queries, 3)
	assert.Equal(t, q1ID, listQryResp.Queries[0].ID)
	assert.Equal(t, q2ID, listQryResp.Queries[1].ID)
	assert.Equal(t, "SELECT -2", listQryResp.Queries[1].Query)
	q3ID := listQryResp.Queries[2].ID

	// delete all queries created
	var delBatchResp fleet.DeleteQueriesResponse
	s.DoJSON("POST", "/api/latest/fleet/queries/delete", map[string]interface{}{
		"ids": []uint{q1ID, q2ID, q3ID},
	}, http.StatusOK, &delBatchResp)
	assert.Equal(t, uint(3), delBatchResp.Deleted)
}

func (s *integrationTestSuite) TestListSoftwareAndSoftwareDetails() {
	t := s.T()

	// create a few hosts specific to this test
	hosts := make([]*fleet.Host, 20)
	for i := range hosts {
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         new(t.Name() + strconv.Itoa(i)),
			OsqueryHostID:   new(t.Name() + strconv.Itoa(i)),
			UUID:            t.Name() + strconv.Itoa(i),
			Hostname:        t.Name() + "foo" + strconv.Itoa(i) + ".local",
			PrimaryIP:       "192.168.1." + strconv.Itoa(i),
			PrimaryMac:      fmt.Sprintf("30-65-EC-6F-C4-%02d", i),
		})
		require.NoError(t, err)
		require.NotNil(t, host)
		hosts[i] = host
	}

	// create a bunch of software
	sws := make([]fleet.Software, 20)
	for i := range sws {
		sw := fleet.Software{Name: fmt.Sprintf("sw%02d", i), Version: fmt.Sprintf("0.0.%02d", i), Source: "apps"}
		if i%2 == 0 {
			sw.Source = "chrome_extensions"
			sw.ExtensionFor = "chrome"
		}
		sws[i] = sw
	}

	sortByNameAlphanumeric := func(sw []fleet.Software, a, b int) bool {
		aNum, _ := strconv.Atoi(strings.TrimPrefix(sw[a].Name, "sw"))
		bNum, _ := strconv.Atoi(strings.TrimPrefix(sw[b].Name, "sw"))
		return aNum < bNum
	}
	sortEntryByNameAlphanumeric := func(sw []fleet.HostSoftwareEntry, a, b int) bool {
		aNum, _ := strconv.Atoi(strings.TrimPrefix(sw[a].Name, "sw"))
		bNum, _ := strconv.Atoi(strings.TrimPrefix(sw[b].Name, "sw"))
		return aNum < bNum
	}

	// mark them as installed on the hosts, with host at index 0 having all 20,
	// at index 1 having 19, index 2 = 18, etc. until index 19 = 1. So software
	// sws[0] is only used by 1 host, while sws[19] is used by all.
	for i, h := range hosts {
		_, err := s.ds.UpdateHostSoftware(context.Background(), h.ID, sws[i:])
		require.NoError(t, err)
		require.NoError(t, s.ds.LoadHostSoftware(context.Background(), h, false))

		if i == 0 {
			// this host has all software, refresh the list so we have the software.ID filled
			sws = make([]fleet.Software, 0, len(h.Software))
			for _, s := range h.Software {
				sws = append(sws, s.Software)
			}
			// Sort software by Name (alphanumeric)
			sort.Slice(
				sws, func(a, b int) bool {
					return sortByNameAlphanumeric(sws, a, b)
				},
			)
		}
	}

	var cpes []fleet.SoftwareCPE
	for i, sw := range sws {
		cpes = append(cpes, fleet.SoftwareCPE{SoftwareID: sw.ID, CPE: "somecpe" + strconv.Itoa(i)})
	}

	_, err := s.ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	// Reload software to load GeneratedCPEID
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), hosts[0], false))

	// add CVEs for the first 10 software, which are the least used (lower hosts_count)
	// Sort software by Name (alphanumeric)
	sort.Slice(
		hosts[0].Software, func(a, b int) bool {
			return sortEntryByNameAlphanumeric(hosts[0].Software, a, b)
		},
	)
	testCvePrefix := "cve-123-123"
	for i, sw := range hosts[0].Software[:10] {
		inserted, err := s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
			SoftwareID: sw.ID,
			CVE:        fmt.Sprintf(testCvePrefix+"-%03d", i),
		}, fleet.NVDSource)
		require.NoError(t, err)
		require.True(t, inserted)
	}
	expectedVulnVersionsCount := 10

	// create a team and make the last 3 hosts part of it (meaning 3 that use
	// sws[19], 2 for sws[18], and 1 for sws[17])
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: t.Name(),
	})
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&tm.ID, []uint{hosts[19].ID, hosts[18].ID, hosts[17].ID})))
	expectedTeamVersionsCount := 3

	assertSoftwareDetails := func(expectedSoftware []fleet.Software, team string) {
		t.Helper()
		// this is just a basic sanity check of the software details endpoints and doesn't test all of the
		// fields that may be present in the response (e.g., vulnerabilities)
		for _, sw := range expectedSoftware {
			var detailsResp getSoftwareResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", sw.ID), nil, http.StatusOK, &detailsResp, "team_id", team)
			assert.Equal(t, sw.ID, detailsResp.Software.ID)
			assert.Equal(t, sw.Name, detailsResp.Software.Name)
			assert.Equal(t, sw.Version, detailsResp.Software.Version)
			assert.Equal(t, sw.Source, detailsResp.Software.Source)
			assert.Equal(t, sw.ExtensionFor, detailsResp.Software.ExtensionFor)
			// Browser field should be populated for browser extensions
			if sw.Source == "chrome_extensions" || sw.Source == "firefox_addons" || sw.Source == "ie_extensions" || sw.Source == "safari_extensions" {
				assert.Equal(t, sw.ExtensionFor, detailsResp.Software.Browser)
			} else {
				assert.Equal(t, "", detailsResp.Software.Browser)
			}

			detailsResp = getSoftwareResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", sw.ID), nil, http.StatusOK, &detailsResp, "team_id", team)
			assert.Equal(t, sw.ID, detailsResp.Software.ID)
			assert.Equal(t, sw.Name, detailsResp.Software.Name)
			assert.Equal(t, sw.Version, detailsResp.Software.Version)
			assert.Equal(t, sw.Source, detailsResp.Software.Source)
			assert.Equal(t, sw.ExtensionFor, detailsResp.Software.ExtensionFor)
			// Browser field should be populated for browser extensions
			if sw.Source == "chrome_extensions" || sw.Source == "firefox_addons" || sw.Source == "ie_extensions" || sw.Source == "safari_extensions" {
				assert.Equal(t, sw.ExtensionFor, detailsResp.Software.Browser)
			} else {
				assert.Equal(t, "", detailsResp.Software.Browser)
			}
			if len(sw.Vulnerabilities) > 0 {
				assert.Len(t, detailsResp.Software.Vulnerabilities, len(sw.Vulnerabilities))
				assert.Greater(t, detailsResp.Software.Vulnerabilities[0].CreatedAt, time.Now().Add(-time.Hour)) // asserting a non-zero time
			}
		}
	}

	assertResp := func(resp listSoftwareResponse, want []fleet.Software, ts time.Time, team string, counts ...int) {
		t.Helper()
		require.Len(t, resp.Software, len(want))
		for i := range resp.Software {
			wantID, gotID := want[i].ID, resp.Software[i].ID
			assert.Equal(t, wantID, gotID, "want.Name: %s got.Name: %s", want[i].Name, resp.Software[i].Name)
			wantName, gotName := want[i].Name, resp.Software[i].Name
			assert.Equal(t, wantName, gotName)
			wantVersion, gotVersion := want[i].Version, resp.Software[i].Version
			assert.Equal(t, wantVersion, gotVersion)
			wantSource, gotSource := want[i].Source, resp.Software[i].Source
			assert.Equal(t, wantSource, gotSource)
			wantExtensionFor, gotExtensionFor := want[i].ExtensionFor, resp.Software[i].ExtensionFor
			assert.Equal(t, wantExtensionFor, gotExtensionFor)
			// Browser field should be populated for browser extensions
			if want[i].Source == "chrome_extensions" || want[i].Source == "firefox_addons" || want[i].Source == "ie_extensions" || want[i].Source == "safari_extensions" {
				assert.Equal(t, want[i].ExtensionFor, resp.Software[i].Browser)
			} else {
				assert.Equal(t, "", resp.Software[i].Browser)
			}
			wantCount, gotCount := counts[i], resp.Software[i].HostsCount
			assert.Equal(t, wantCount, gotCount)
		}
		if ts.IsZero() {
			assert.Nil(t, resp.CountsUpdatedAt)
		} else {
			require.NotNil(t, resp.CountsUpdatedAt)
			assert.WithinDuration(t, ts, *resp.CountsUpdatedAt, time.Second)
		}
		assertSoftwareDetails(resp.Software, team)
	}

	assertVersionsResp := func(
		resp listSoftwareVersionsResponse, want []fleet.Software, ts time.Time, team string, swCount int, hostCounts ...int,
	) {
		require.Equal(t, swCount, resp.Count)
		require.Len(t, resp.Software, len(want))
		for i := range resp.Software {
			wantID, gotID := want[i].ID, resp.Software[i].ID
			assert.Equal(t, wantID, gotID)
			wantCount, gotCount := hostCounts[i], resp.Software[i].HostsCount
			assert.Equal(t, wantCount, gotCount)
			wantName, gotName := want[i].Name, resp.Software[i].Name
			assert.Equal(t, wantName, gotName)
			wantVersion, gotVersion := want[i].Version, resp.Software[i].Version
			assert.Equal(t, wantVersion, gotVersion)
			wantSource, gotSource := want[i].Source, resp.Software[i].Source
			assert.Equal(t, wantSource, gotSource)
			wantExtensionFor, gotExtensionFor := want[i].ExtensionFor, resp.Software[i].ExtensionFor
			assert.Equal(t, wantExtensionFor, gotExtensionFor)
			// Browser field should be populated for browser extensions
			if want[i].Source == "chrome_extensions" || want[i].Source == "firefox_addons" || want[i].Source == "ie_extensions" || want[i].Source == "safari_extensions" {
				assert.Equal(t, want[i].ExtensionFor, resp.Software[i].Browser)
			} else {
				assert.Equal(t, "", resp.Software[i].Browser)
			}
		}
		if ts.IsZero() {
			assert.Nil(t, resp.CountsUpdatedAt)
		} else {
			require.NotNil(t, resp.CountsUpdatedAt)
			assert.WithinDuration(t, ts, *resp.CountsUpdatedAt, time.Second)
		}
		assertSoftwareDetails(resp.Software, team)
	}

	// no software host counts have been calculated yet, so this returns nothing
	var lsResp listSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{}, "")
	var versResp listSoftwareVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, nil, time.Time{}, "", 0)

	// same with a team filter
	teamStr := fmt.Sprintf("%d", tm.ID)
	lsResp = listSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "order_key", "hosts_count", "order_direction", "desc", "team_id",
		teamStr,
	)
	assertResp(lsResp, nil, time.Time{}, teamStr)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "order_key", "hosts_count", "order_direction", "desc",
		"team_id", teamStr,
	)
	assertVersionsResp(versResp, nil, time.Time{}, teamStr, 0)

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))

	// now the list software endpoint returns the software, get the first page without vulns
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[19], sws[18], sws[17], sws[16], sws[15]}, hostsCountTs, "", 20, 19, 18, 17, 16)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(
		versResp, []fleet.Software{sws[19], sws[18], sws[17], sws[16], sws[15]}, hostsCountTs, "", len(sws), 20, 19, 18, 17, 16,
	)
	require.False(t, versResp.Meta.HasPreviousResults)
	require.True(t, versResp.Meta.HasNextResults)

	// second page (page=1)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[14], sws[13], sws[12], sws[11], sws[10]}, hostsCountTs, "", 15, 14, 13, 12, 11)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(
		versResp, []fleet.Software{sws[14], sws[13], sws[12], sws[11], sws[10]}, hostsCountTs, "", len(sws), 15, 14, 13, 12, 11,
	)
	require.True(t, versResp.Meta.HasPreviousResults)
	require.True(t, versResp.Meta.HasNextResults)

	// third page (page=2)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, "", 10, 9, 8, 7, 6)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, "", len(sws), 10, 9, 8, 7, 6)
	require.True(t, versResp.Meta.HasPreviousResults)
	require.True(t, versResp.Meta.HasNextResults)

	// last page (page=3)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "3", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, "", 5, 4, 3, 2, 1)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "3", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, "", len(sws), 5, 4, 3, 2, 1)
	require.True(t, versResp.Meta.HasPreviousResults)
	require.False(t, versResp.Meta.HasNextResults)

	// past the end
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "4", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{}, "")
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "4", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, nil, time.Time{}, "", len(sws))

	// no explicit sort order, defaults to hosts_count DESC
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "0")
	assertResp(lsResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, "", 20, 19)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "2", "page", "0")
	assertVersionsResp(versResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, "", len(sws), 20, 19)

	// hosts_count ascending
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "3", "page", "0", "order_key", "hosts_count", "order_direction", "asc")
	assertResp(lsResp, []fleet.Software{sws[0], sws[1], sws[2]}, hostsCountTs, "", 1, 2, 3)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "3", "page", "0", "order_key", "hosts_count", "order_direction", "asc")
	assertVersionsResp(versResp, []fleet.Software{sws[0], sws[1], sws[2]}, hostsCountTs, "", len(sws), 1, 2, 3)

	// vulnerable software only
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, "", 10, 9, 8, 7, 6)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "vulnerable", "true", "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(
		versResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, "", expectedVulnVersionsCount, 10, 9, 8, 7, 6,
	)

	// vulnerable software only, next page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, "", 5, 4, 3, 2, 1)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "vulnerable", "true", "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(
		versResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, "", expectedVulnVersionsCount, 5, 4, 3, 2, 1,
	)

	// vulnerable software only, past last page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{}, "")
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "vulnerable", "true", "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, nil, time.Time{}, "", expectedVulnVersionsCount)

	// /software/versions  filtered by name, version, cve (`/software` is deprecated)
	versionsResp := listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", sws[0].Name)
	assertVersionsResp(versionsResp, []fleet.Software{sws[0]}, hostsCountTs, "", 1, 1)
	// with whitespace
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", " "+sws[0].Name+"\n")
	assertVersionsResp(versionsResp, []fleet.Software{sws[0]}, hostsCountTs, "", 1, 1)

	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", sws[0].Version)
	assertVersionsResp(versionsResp, []fleet.Software{sws[0]}, hostsCountTs, "", 1, 1)
	// with whitespace
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", "\n"+sws[0].Version+"  ")
	assertVersionsResp(versionsResp, []fleet.Software{sws[0]}, hostsCountTs, "", 1, 1)

	// All 10 CVEs added to the first 10 software have the same cvePrefix, so should return all
	// 10 vulnerable software versions
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", testCvePrefix)
	require.Len(t, versionsResp.Software, 10)
	require.Equal(t, 10, versionsResp.Count)
	// TODO(jacob) use `assertVersionsResp`
	// assertVersionsResp(versionsResp, sws[:10], hostsCountTs, "", 10, 1)
	// with whitespace
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", "  "+testCvePrefix+"\n")
	require.Len(t, versionsResp.Software, 10)
	require.Equal(t, 10, versionsResp.Count)
	// TODO(jacob) use `assertVersionsResp`
	// assertVersionsResp(versionsResp, sws[:10], hostsCountTs, "", 10, 1)

	// filter by the team, 2 by page
	lsResp = listSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "0", "order_key", "hosts_count",
		"order_direction", "desc", "team_id", teamStr,
	)
	assertResp(lsResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, teamStr, 3, 2)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "2", "page", "0", "order_key",
		"hosts_count", "order_direction", "desc", "team_id", teamStr,
	)
	assertVersionsResp(versResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, teamStr, expectedTeamVersionsCount, 3, 2)

	// filter by the team, 2 by page, next page
	lsResp = listSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "1", "order_key", "hosts_count",
		"order_direction", "desc", "team_id", teamStr,
	)
	assertResp(lsResp, []fleet.Software{sws[17]}, hostsCountTs, teamStr, 1)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "2", "page", "1", "order_key",
		"hosts_count", "order_direction", "desc", "team_id", teamStr,
	)
	assertVersionsResp(versResp, []fleet.Software{sws[17]}, hostsCountTs, teamStr, expectedTeamVersionsCount, 1)

	// filter by no team, 2 by page
	lsResp = listSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "0", "order_key", "name",
		"order_direction", "desc", "team_id", "0",
	)
	fmt.Printf("lsResp: %+v\n", lsResp)
	assertResp(lsResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, "", 17, 17)

	// Invalid software team -- admin gets a 404, team users get a 403
	detailsResp := getSoftwareResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", versResp.Software[0].ID), nil, http.StatusNotFound, &detailsResp,
		"team_id", "999999",
	)

	// a request with without_vulnerability_details set to false does not return extra details
	respVersions := listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"without_vulnerability_details", "false",
	)
	for _, s := range respVersions.Software {
		for _, cve := range s.Vulnerabilities {
			require.Nil(t, cve.CVSSScore)
			require.Nil(t, cve.EPSSProbability)
			require.Nil(t, cve.CISAKnownExploit)
			require.Nil(t, cve.CVEPublished)
			require.Nil(t, cve.Description)
			require.Nil(t, cve.ResolvedInVersion)
		}
	}
}

func (s *integrationTestSuite) TestChangeUserEmail() {
	t := s.T()

	// create a new test user
	user := &fleet.User{
		Name:       t.Name(),
		Email:      "testchangeemail@example.com",
		GlobalRole: new(fleet.RoleObserver),
	}
	userRawPwd := "foobarbaz1234!"
	err := user.SetPassword(userRawPwd, 10, 10)
	require.Nil(t, err)
	user, err = s.ds.NewUser(context.Background(), user)
	require.Nil(t, err)

	// try to change email with an invalid token
	var changeResp changeEmailResponse
	s.DoJSON("GET", "/api/latest/fleet/email/change/invalidtoken", nil, http.StatusNotFound, &changeResp)

	// create a valid token for the test user
	err = s.ds.PendingEmailChange(context.Background(), user.ID, "testchangeemail2@example.com", "validtoken")
	require.Nil(t, err)

	// try to change email with a valid token, but request made from different user
	changeResp = changeEmailResponse{}
	s.DoJSON("GET", "/api/latest/fleet/email/change/validtoken", nil, http.StatusNotFound, &changeResp)

	// switch to the test user and make the change email request
	s.token = s.getTestToken(user.Email, userRawPwd)
	defer func() { s.token = s.getTestAdminToken() }()

	changeResp = changeEmailResponse{}
	s.DoJSON("GET", "/api/latest/fleet/email/change/validtoken", nil, http.StatusOK, &changeResp)
	require.Equal(t, "testchangeemail2@example.com", changeResp.NewEmail)

	// using the token consumes it, so making another request with the same token fails
	changeResp = changeEmailResponse{}
	s.DoJSON("GET", "/api/latest/fleet/email/change/validtoken", nil, http.StatusNotFound, &changeResp)
}

func (s *integrationTestSuite) TestSearchTargets() {
	t := s.T()
	hosts := s.createHosts(t)

	var builtinNames []string
	for name := range fleet.ReservedLabelNames() {
		builtinNames = append(builtinNames, name)
	}
	lblMap, err := s.ds.LabelIDsByName(context.Background(), builtinNames, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, lblMap, len(builtinNames))

	// no search criteria
	var searchResp fleet.SearchTargetsResponse
	s.DoJSON("POST", "/api/latest/fleet/targets", fleet.SearchTargetsRequest{}, http.StatusOK, &searchResp)
	require.Equal(t, uint(0), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, len(hosts)) // the HostTargets.HostIDs are actually host IDs to *omit* from the search
	require.Len(t, searchResp.Targets.Labels, len(lblMap))
	require.Len(t, searchResp.Targets.Teams, 0)

	var lblIDs []uint
	for _, labelID := range lblMap {
		lblIDs = append(lblIDs, labelID)
	}

	searchResp = fleet.SearchTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets", fleet.SearchTargetsRequest{Selected: fleet.HostTargets{LabelIDs: lblIDs}}, http.StatusOK, &searchResp)
	require.Equal(t, uint(0), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, len(hosts)) // no omitted host id
	require.Len(t, searchResp.Targets.Labels, 0)         // All built-in labels have been omitted (pre-selected)
	require.Len(t, searchResp.Targets.Teams, 0)

	searchResp = fleet.SearchTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets", fleet.SearchTargetsRequest{Selected: fleet.HostTargets{HostIDs: []uint{hosts[1].ID}}}, http.StatusOK, &searchResp)
	require.Equal(t, uint(1), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, len(hosts)-1) // one omitted host id
	require.Len(t, searchResp.Targets.Labels, len(lblMap)) // labels have not been omitted
	require.Len(t, searchResp.Targets.Teams, 0)

	searchResp = fleet.SearchTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets", fleet.SearchTargetsRequest{MatchQuery: "foo.local1"}, http.StatusOK, &searchResp)
	require.Equal(t, uint(0), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, 1)
	require.Len(t, searchResp.Targets.Labels, 1) // with a match query, only matching label names and "All Hosts" can be returned (here, only all hosts)
	require.Len(t, searchResp.Targets.Teams, 0)
	require.Contains(t, searchResp.Targets.Hosts[0].Hostname, "foo.local1")
}

func (s *integrationTestSuite) TestCountTargets() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "TestTeam"})
	require.NoError(t, err)
	require.Equal(t, "TestTeam", team.Name)

	hosts := s.createHosts(t)

	lblMap, err := s.ds.LabelIDsByName(context.Background(), []string{"All Hosts"}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, lblMap, 1)

	for i := range hosts {
		err = s.ds.RecordLabelQueryExecutions(context.Background(), hosts[i], map[uint]*bool{lblMap["All Hosts"]: new(true)}, time.Now(), false)
		require.NoError(t, err)
	}

	var hostIDs []uint
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.ID)
	}

	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(new(team.ID), []uint{hostIDs[0]})) // nolint:nilaway // createHosts always returns at least one host
	require.NoError(t, err)

	var countResp fleet.CountTargetsResponse
	// sleep to reduce flake in last seen time so that online/offline counts can be tested
	time.Sleep(1 * time.Second)

	// none selected
	s.DoJSON("POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{}, http.StatusOK, &countResp)
	require.Equal(t, uint(0), countResp.TargetsCount)
	require.Equal(t, uint(0), countResp.TargetsOnline)
	require.Equal(t, uint(0), countResp.TargetsOffline)

	var lblIDs []uint
	for _, labelID := range lblMap {
		lblIDs = append(lblIDs, labelID)
	}
	// all hosts label selected
	countResp = fleet.CountTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{Selected: fleet.HostTargets{LabelIDs: lblIDs}}, http.StatusOK, &countResp)
	require.Equal(t, uint(3), countResp.TargetsCount)
	require.Equal(t, uint(1), countResp.TargetsOnline)
	require.Equal(t, uint(2), countResp.TargetsOffline)

	// team selected
	countResp = fleet.CountTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{Selected: fleet.HostTargets{TeamIDs: []uint{team.ID}}}, http.StatusOK, &countResp)
	require.Equal(t, uint(1), countResp.TargetsCount)
	require.Equal(t, uint(1), countResp.TargetsOnline)
	require.Equal(t, uint(0), countResp.TargetsOffline)

	// 'No team' selected
	countResp = fleet.CountTargetsResponse{}
	s.DoJSON(
		"POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{Selected: fleet.HostTargets{TeamIDs: []uint{0}}},
		http.StatusOK, &countResp,
	)
	assert.Equal(t, uint(2), countResp.TargetsCount)
	assert.Equal(t, uint(0), countResp.TargetsOnline)
	assert.Equal(t, uint(2), countResp.TargetsOffline)

	// host id selected
	countResp = fleet.CountTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{Selected: fleet.HostTargets{HostIDs: []uint{hosts[1].ID}}}, http.StatusOK, &countResp)
	require.Equal(t, uint(1), countResp.TargetsCount)
	require.Equal(t, uint(0), countResp.TargetsOnline)
	require.Equal(t, uint(1), countResp.TargetsOffline)
}

func (s *integrationTestSuite) TestStatus() {
	var statusResp statusResponse
	s.DoJSON("GET", "/api/latest/fleet/status/result_store", nil, http.StatusOK, &statusResp)
	s.DoJSON("GET", "/api/latest/fleet/status/live_query", nil, http.StatusOK, &statusResp)
}

func (s *integrationTestSuite) TestOsqueryConfig() {
	t := s.T()

	hosts := s.createHosts(t)
	req := getClientConfigRequest{NodeKey: *hosts[0].NodeKey}
	var resp getClientConfigResponse
	s.DoJSON("POST", "/api/osquery/config", req, http.StatusOK, &resp)

	// test with invalid node key
	var errRes map[string]interface{}
	req.NodeKey += "zzzz"
	s.DoJSON("POST", "/api/osquery/config", req, http.StatusUnauthorized, &errRes)
	assert.Contains(t, errRes["error"], "invalid node key")
}

func (s *integrationTestSuite) TestOsqueryConfigETag() {
	t := s.T()

	hosts := s.createHosts(t)
	nodeKey := *hosts[0].NodeKey

	// Request bodies for the three etag states of the body-carried protocol:
	// field absent (legacy), field empty (opted in, no validator yet), and
	// field carrying an echoed validator.
	legacyRequestBody, err := json.Marshal(map[string]string{"node_key": nodeKey})
	require.NoError(t, err)
	etagRequestBody := func(etag string) []byte {
		b, err := json.Marshal(map[string]string{"node_key": nodeKey, "etag": etag})
		require.NoError(t, err)
		return b
	}
	readAll := func(resp *http.Response) []byte {
		t.Cleanup(func() { resp.Body.Close() })
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return b
	}
	etagOf := func(body []byte) string {
		var decoded map[string]any
		require.NoError(t, json.Unmarshal(body, &decoded))
		etag, _ := decoded["etag"].(string)
		return etag
	}

	// 1. A legacy agent (no etag field) gets the config with no "etag" key —
	// byte-for-byte the pre-feature response.
	legacyBody := readAll(s.DoRaw("POST", "/api/osquery/config", legacyRequestBody, http.StatusOK))
	require.NotEmpty(t, legacyBody)
	require.NotContains(t, string(legacyBody), `"etag"`)

	// 2. An opted-in agent with no validator (empty etag) gets the full
	// config with the "etag" key added; the validator covers the etag-less
	// representation, i.e. the legacy body.
	optedInBody := readAll(s.DoRaw("POST", "/api/osquery/config", etagRequestBody(""), http.StatusOK))
	expectedETag := clientConfigETag(legacyBody)
	require.Equal(t, expectedETag, etagOf(optedInBody))
	require.NotEqual(t, "ok", etagOf(optedInBody))

	// Stripping the etag key yields the same configuration. This is a
	// semantic (unmarshaled) comparison, not a byte comparison: pack content
	// is embedded as struct-marshaled json.RawMessage whose field order a
	// round-trip through map[string]any cannot preserve, so re-marshaling
	// here would alphabetize nested keys and spuriously mismatch whenever
	// earlier suite tests left scheduled reports behind. The byte-level
	// guarantee — the validator covers the exact legacy bytes — is already
	// pinned by the expectedETag assertion above.
	var legacyConfig, optedInConfig map[string]any
	require.NoError(t, json.Unmarshal(legacyBody, &legacyConfig))
	require.NoError(t, json.Unmarshal(optedInBody, &optedInConfig))
	delete(optedInConfig, "etag")
	require.Equal(t, legacyConfig, optedInConfig)

	// 3. Echoing the validator gets the constant unchanged body, HTTP 200.
	unchangedBody := readAll(s.DoRaw("POST", "/api/osquery/config", etagRequestBody(expectedETag), http.StatusOK))
	require.JSONEq(t, configUnchangedBody, string(unchangedBody))

	// 4. A stale etag gets the full config again, current validator included.
	staleBody := readAll(s.DoRaw("POST", "/api/osquery/config", etagRequestBody("wrong-etag"), http.StatusOK))
	require.Equal(t, expectedETag, etagOf(staleBody))

	// 5. An empty etag is never answered "unchanged", even with the record
	// warm from the previous requests.
	emptyAgainBody := readAll(s.DoRaw("POST", "/api/osquery/config", etagRequestBody(""), http.StatusOK))
	require.Equal(t, expectedETag, etagOf(emptyAgainBody))

	// 6. Invalid node key fails auth regardless of a matching etag.
	invalidBody, _ := json.Marshal(map[string]string{"node_key": "invalid-key", "etag": expectedETag})
	resp := s.DoRaw("POST", "/api/osquery/config", invalidBody, http.StatusUnauthorized)
	readAll(resp)

	// 7. The /api/v1/osquery/config alias speaks the same protocol.
	aliasBody := readAll(s.DoRaw("POST", "/api/v1/osquery/config", etagRequestBody(expectedETag), http.StatusOK))
	require.JSONEq(t, configUnchangedBody, string(aliasBody))

	// 8. A config change produces a new validator: the old etag downloads
	// the full new config, and the new etag is then answered "unchanged".
	//
	// This suite shares one server, so agent options may already hold anything
	// a previously executed test left behind. Derive the new value from the
	// current one so the PATCH cannot be a no-op, and restore the previous
	// options afterwards so this mutation does not leak into later tests.
	ctx := context.Background()
	appCfgBefore, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	require.NotNil(t, appCfgBefore.AgentOptions)
	prevAgentOptions := append(json.RawMessage(nil), *appCfgBefore.AgentOptions...)
	t.Cleanup(func() {
		// Restored through the datastore rather than the API: the stored blob is
		// not necessarily accepted by the write validator (the suite's defaults
		// carry command-line flags that PATCH rejects inside config.options), and
		// the point here is to put back exactly what was there.
		appCfg, err := s.ds.AppConfig(ctx)
		require.NoError(t, err)
		appCfg.AgentOptions = &prevAgentOptions
		require.NoError(t, s.ds.SaveAppConfig(ctx, appCfg))
	})

	// logger_tls_period is echoed into the rendered config, so bumping it past
	// whatever is there now guarantees the body — and therefore the validator —
	// changes.
	var currentOptions struct {
		Config struct {
			Options struct {
				LoggerTLSPeriod int `json:"logger_tls_period"`
			} `json:"options"`
		} `json:"config"`
	}
	require.NoError(t, json.Unmarshal(prevAgentOptions, &currentOptions))
	newPeriod := currentOptions.Config.Options.LoggerTLSPeriod + 7
	s.DoRaw("PATCH", "/api/latest/fleet/config",
		[]byte(fmt.Sprintf(`{"agent_options":{"config":{"options":{"logger_tls_period":%d}}}}`, newPeriod)),
		http.StatusOK)

	changedBody := readAll(s.DoRaw("POST", "/api/osquery/config", etagRequestBody(expectedETag), http.StatusOK))
	require.Contains(t, string(changedBody), fmt.Sprintf(`"logger_tls_period": %d`, newPeriod),
		"the PATCH must actually change the rendered config, or the validator assertion below is vacuous")
	newETag := etagOf(changedBody)
	require.NotEmpty(t, newETag)
	require.NotEqual(t, "ok", newETag)
	require.NotEqual(t, expectedETag, newETag, "validator should have changed")

	unchangedAfterBody := readAll(s.DoRaw("POST", "/api/osquery/config", etagRequestBody(newETag), http.StatusOK))
	require.JSONEq(t, configUnchangedBody, string(unchangedAfterBody))

	// 9. Legacy agents see the new config with no "etag" key.
	newLegacyBody := readAll(s.DoRaw("POST", "/api/osquery/config", legacyRequestBody, http.StatusOK))
	require.NotContains(t, string(newLegacyBody), `"etag"`)
	require.Equal(t, newETag, clientConfigETag(newLegacyBody))
}

func (s *integrationTestSuite) TestEnrollOsquery() {
	t := s.T()

	// set the enroll secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// invalid enroll secret fails
	j, err := json.Marshal(&contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   "nosuchsecret",
		HostIdentifier: "abcd",
	})
	require.NoError(t, err)
	s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusUnauthorized)

	// valid enroll secret succeeds
	j, err = json.Marshal(&contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   t.Name(),
		HostIdentifier: t.Name(),
	})
	require.NoError(t, err)

	var resp contract.EnrollOsqueryAgentResponse
	hres := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusOK)
	defer hres.Body.Close()
	require.NoError(t, json.NewDecoder(hres.Body).Decode(&resp))
	require.NotEmpty(t, resp.NodeKey)

	// A team may retain an empty enroll secret created before the create/update
	// validation existed. Simulate that by writing an empty secret directly via
	// the datastore, bypassing the service-layer validation.
	ctx := context.Background()
	emptyTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "empty"})
	require.NoError(t, err)
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, &emptyTeam.ID, []*fleet.EnrollSecret{{Secret: "", TeamID: &emptyTeam.ID}}))

	// Enrolling with an empty or whitespace-only secret must be rejected as
	// node_invalid, even though an empty secret exists in storage.
	for _, badSecret := range []string{"", "   "} {
		j, err = json.Marshal(&contract.EnrollOsqueryAgentRequest{
			EnrollSecret:   badSecret,
			HostIdentifier: t.Name() + "empty-host",
		})
		require.NoError(t, err)
		badRes := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusUnauthorized)
		var body map[string]any
		require.NoError(t, json.NewDecoder(badRes.Body).Decode(&body))
		badRes.Body.Close()
		require.Equal(t, true, body["node_invalid"])
	}
}

func (s *integrationTestSuite) TestReenrollHostCleansPolicies() {
	t := s.T()
	ctx := context.Background()
	host := s.createHosts(t)[0]

	// set the enroll secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Empty(t, getHostResp.Host.Policies)

	// create a policy and make the host fail it
	pol, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: t.Name(), Query: "SELECT 1", Platform: host.FleetPlatform()})
	require.NoError(t, err)
	_, err = s.ds.RecordPolicyQueryExecutions(ctx, &fleet.Host{ID: host.ID}, map[uint]*bool{pol.ID: new(false)}, time.Now(), false, nil)
	require.NoError(t, err)

	// refetch the host details
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Len(t, *getHostResp.Host.Policies, 1)

	// re-enroll the host, but using a different platform
	j, err := json.Marshal(&contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   t.Name(),
		HostIdentifier: *host.OsqueryHostID,
		HostDetails:    map[string](map[string]string){"os_version": map[string]string{"platform": "windows"}},
	})
	require.NoError(t, err)

	// prevent the enroll cooldown from being applied
	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(
			context.Background(),
			"UPDATE hosts SET last_enrolled_at = DATE_SUB(NOW(), INTERVAL '1' HOUR) WHERE id = ?",
			host.ID,
		)
		return err
	})
	var resp contract.EnrollOsqueryAgentResponse
	hres := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusOK)
	defer hres.Body.Close()
	require.NoError(t, json.NewDecoder(hres.Body).Decode(&resp))
	require.NotEmpty(t, resp.NodeKey)

	// refetch the host details
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)

	// policies should be gone
	require.Empty(t, getHostResp.Host.Policies)
}

func (s *integrationTestSuite) TestCarve() {
	t := s.T()
	hosts := s.createHosts(t)

	// begin a carve with an invalid node key
	var errRes map[string]interface{}
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *hosts[0].NodeKey + "zzz",
		BlockCount: 1,
		BlockSize:  1,
		CarveSize:  1,
		CarveId:    "c1",
	}, http.StatusUnauthorized, &errRes)
	assert.Contains(t, errRes["error"], "invalid node key")

	// invalid carve size
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  3,
		CarveSize:  0,
		CarveId:    "c1",
	}, http.StatusInternalServerError, &errRes) // TODO: should be 4xx, see #4406
	assert.Contains(t, errRes["error"], "carve_size must be greater")

	// invalid block size too big
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  maxBlockSize + 1,
		CarveSize:  maxCarveSize,
		CarveId:    "c1",
	}, http.StatusInternalServerError, &errRes) // TODO: should be 4xx, see #4406
	assert.Contains(t, errRes["error"], "block_size exceeds max")

	// invalid carve size too big
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  maxBlockSize,
		CarveSize:  maxCarveSize + 1,
		CarveId:    "c1",
	}, http.StatusInternalServerError, &errRes) // TODO: should be 4xx, see #4406
	assert.Contains(t, errRes["error"], "carve_size exceeds max")

	// invalid carve size, does not match blocks
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  3,
		CarveSize:  1,
		CarveId:    "c1",
	}, http.StatusInternalServerError, &errRes) // TODO: should be 4xx, see #4406
	assert.Contains(t, errRes["error"], "carve_size does not match")

	// valid carve begin
	var beginResp fleet.CarveBeginResponse
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  3,
		CarveSize:  8,
		CarveId:    "c1",
		RequestId:  "r1",
	}, http.StatusOK, &beginResp)
	require.NotEmpty(t, beginResp.SessionId)
	sid := beginResp.SessionId

	// sending a block with invalid session id
	var blockResp fleet.CarveBlockResponse
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   1,
		SessionId: sid + "zz",
		RequestId: "??",
		Data:      []byte("p1."),
	}, http.StatusUnauthorized, &blockResp)

	// sending a block with valid session id but invalid request id
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   1,
		SessionId: sid,
		RequestId: "??",
		Data:      []byte("p1."),
	}, http.StatusUnauthorized, &blockResp)

	checkCarveError := func(id uint, err string) {
		var getResp fleet.GetCarveResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d", id), nil, http.StatusOK, &getResp)
		require.Equal(t, err, *getResp.Carve.Error)
	}

	// sending a block with unexpected block id (expects 0, got 1)
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   1,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p1."),
	}, http.StatusBadRequest, &blockResp)
	checkCarveError(1, "block_id does not match expected block (0): 1")

	// sending a block with valid payload, block 0
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   0,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p1."),
	}, http.StatusOK, &blockResp)
	require.True(t, blockResp.Success)

	// sending next block
	blockResp = fleet.CarveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   1,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p2."),
	}, http.StatusOK, &blockResp)
	require.True(t, blockResp.Success)

	// sending already-sent block again
	blockResp = fleet.CarveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   1,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p2."),
	}, http.StatusBadRequest, &blockResp)
	checkCarveError(1, "block_id does not match expected block (2): 1")

	// sending final block with too many bytes
	blockResp = fleet.CarveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   2,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p3extra"),
	}, http.StatusBadRequest, &blockResp)
	checkCarveError(1, "exceeded declared block size 3: 7")

	// sending actual final block
	blockResp = fleet.CarveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   2,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p3"),
	}, http.StatusOK, &blockResp)
	require.True(t, blockResp.Success)

	// sending unexpected block
	blockResp = fleet.CarveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   3,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p4."),
	}, http.StatusBadRequest, &blockResp)
	checkCarveError(1, "block_id exceeds expected max (2): 3")
}

func (s *integrationTestSuite) TestCarveUnauthenticated() {
	t := s.T()

	verifyAuthError := func(t *testing.T, res *http.Response) {
		var errs validationErrResp
		err := json.NewDecoder(res.Body).Decode(&errs)
		require.NoError(t, err)
		res.Body.Close()
		assert.Equal(t, "Authentication failed", errs.Message)
		require.Len(t, errs.Errors, 1)
		assert.Equal(t, "Authentication failed", errs.Errors[0].Reason)
	}

	// Sending invalid format for data on purpose on purpose to check that the error is a HTTP 401 error
	// vs a decoding/parsing error (this way we check it never gets to parse "data").
	for _, tc := range []struct {
		testName       string
		rawJSONRequest string
	}{
		{
			testName:       "empty-json",
			rawJSONRequest: `{}`,
		},
		{
			testName: "with-spaces", // osquery does not send spaces in the JSON
			rawJSONRequest: `{
				"block_id":   1,
				"request_id": "invalid",
				"data":      9999999999
			}`,
		},
		{
			testName:       "without-session-id",
			rawJSONRequest: `{"block_id":1,"request_id":"invalid","data":9999999999}`,
		},
		{
			testName:       "invalid-session-id-format",
			rawJSONRequest: `{"block_id":1,"session_id":2,"request_id": "invalid","data":9999999999}`,
		},
		{
			testName:       "invalid-session-id",
			rawJSONRequest: `{"block_id":1,"session_id":"invalid","request_id":"invalid","data":9999999999}`,
		},
		{
			testName:       "invalid-JSON",
			rawJSONRequest: `{"block_ASDASDASDASDASDASDASDASDASDASDASDASDASD":1}`,
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			res := s.DoRaw("POST", "/api/osquery/carve/block", []byte(tc.rawJSONRequest), http.StatusUnauthorized)
			verifyAuthError(t, res)
		})
	}
}

func (s *integrationTestSuite) TestLogLoginAttempts() {
	t := s.T()

	// create a new user
	var createResp createUserResponse
	params := fleet.UserPayload{
		Name:       new("foobar"),
		Email:      new("foobar@example.com"),
		Password:   new(test.GoodPassword),
		GlobalRole: new(fleet.RoleObserver),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)
	u := *createResp.User

	// Register current number of activities.
	activitiesResp := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	oldActivitiesCount := len(activitiesResp.Activities)

	// Login with invalid passwordm, should fail.
	res := s.DoRawNoAuth(
		"POST", "/api/latest/fleet/login",
		jsonMustMarshal(t, fleet.LoginRequest{Email: u.Email, Password: test.GoodPassword2}),
		http.StatusUnauthorized,
	)
	res.Body.Close()

	// A new activity item for the failed login attempt is created.
	activitiesResp = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	require.Len(t, activitiesResp.Activities, oldActivitiesCount+1)
	sort.Slice(activitiesResp.Activities, func(i, j int) bool {
		return activitiesResp.Activities[i].ID < activitiesResp.Activities[j].ID
	})
	activity := activitiesResp.Activities[len(activitiesResp.Activities)-1]
	require.Equal(t, activity.Type, fleet.ActivityTypeUserFailedLogin{}.ActivityName())
	require.NotNil(t, activity.Details)
	actDetails := fleet.ActivityTypeUserFailedLogin{}
	err := json.Unmarshal(*activity.Details, &actDetails)
	require.NoError(t, err)
	require.Equal(t, actDetails.Email, "foobar@example.com")

	// login with good password, should succeed
	res = s.DoRawNoAuth(
		"POST", "/api/latest/fleet/login",
		jsonMustMarshal(t, fleet.LoginRequest{
			Email:    u.Email,
			Password: test.GoodPassword,
		}), http.StatusOK,
	)
	res.Body.Close()

	// A new activity item for the successful login is created.
	activitiesResp = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	require.Len(t, activitiesResp.Activities, oldActivitiesCount+2)
	sort.Slice(activitiesResp.Activities, func(i, j int) bool {
		return activitiesResp.Activities[i].ID < activitiesResp.Activities[j].ID
	})
	activity = activitiesResp.Activities[len(activitiesResp.Activities)-1]
	require.Equal(t, activity.Type, fleet.ActivityTypeUserLoggedIn{}.ActivityName())
	require.NotNil(t, activity.Details)
	err = json.Unmarshal(*activity.Details, &fleet.ActivityTypeUserLoggedIn{})
	require.NoError(t, err)
}

func (s *integrationTestSuite) TestChangePassword() {
	t := s.T()

	endpoint := "/api/latest/fleet/change_password"
	// also the default password for the default logged in admin user
	startPwd := test.GoodPassword

	testCases := []struct {
		oldPw          string
		newPw          string
		expectedStatus int
	}{
		// valid changes – 12-48 characters, with at least 1 number (e.g. 0 - 9) and 1 symbol (e.g. &*#).
		{startPwd, "password123$", http.StatusOK},
		{"password123$", "Password$321", http.StatusOK},

		// invalid changes
		// empty old
		{"", "PassworD$321", http.StatusUnprocessableEntity},
		// empty new
		{"password123$", "", http.StatusUnprocessableEntity},
		// too short
		{"password123$", "Password$21", http.StatusUnprocessableEntity},
		// too long
		{"password123$", "Password$321Password$321Password$321Password$321Password$321", http.StatusUnprocessableEntity},
		// no numbers
		{"password123$", "Password$!@#", http.StatusUnprocessableEntity},
		// no symbols
		{"password123$", "Password4321", http.StatusUnprocessableEntity},
		// new pw is same as old
		{"password123$", "password123$", http.StatusUnprocessableEntity},
		// wrong old pw
		{"passgord123$", "Password$321", http.StatusUnprocessableEntity},

		// change back to original password for cleanup
		{"Password$321", startPwd, http.StatusOK},
	}

	runTestCases := func(name, userEmail string) {
		currentPwd := startPwd
		for _, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				var changePwResp changePasswordResponse
				s.DoJSON("POST", endpoint, changePasswordRequest{OldPassword: tc.oldPw, NewPassword: tc.newPw}, tc.expectedStatus, &changePwResp)
				// After a successful password change, the session is invalidated, so we need to re-authenticate
				if tc.expectedStatus == http.StatusOK {
					currentPwd = tc.newPw
					s.token = s.getTestToken(userEmail, currentPwd)
				}
			})
		}
	}

	adminEmail := "admin1@example.com"
	// Clear the cached admin token so it will be refreshed after password changes
	s.cachedAdminToken = ""
	runTestCases("test change passwords as admin", adminEmail)

	// create a new user
	testUserEmail := "changepwd@example.com"
	var createResp createUserResponse
	params := fleet.UserPayload{
		Name:                     new("Test Change Password"),
		Email:                    new(testUserEmail),
		Password:                 new(startPwd),
		GlobalRole:               new(fleet.RoleObserver),
		AdminForcedPasswordReset: new(false),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)

	// Clear the cached admin token again and schedule cleanup to restore it
	s.cachedAdminToken = ""
	t.Cleanup(func() { s.token = s.getTestAdminToken() })

	// login and run the change password tests as the user
	s.token = s.getTestToken(testUserEmail, startPwd)
	runTestCases("test change passwords as user", testUserEmail)
}

func (s *integrationTestSuite) TestPasswordReset() {
	t := s.T()

	// Clear any previous usage of forgot_password in the test suite to start from scatch.
	clearKeys := func() {
		clearRedisKey(t, s.redisPool, "ratelimit::forgot_password")
	}
	clearKeys()
	s.T().Cleanup(func() {
		clearKeys()
	})

	// create a new user
	var createResp createUserResponse
	userRawPwd := test.GoodPassword
	params := fleet.UserPayload{
		Name:       new("forgotpwd"),
		Email:      new("forgotpwd@example.com"),
		Password:   new(userRawPwd),
		GlobalRole: new(fleet.RoleObserver),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)
	u := *createResp.User

	// Request password reset when SMTP/SES is not configured
	res := s.DoRawNoAuth("POST", "/api/latest/fleet/forgot_password", jsonMustMarshal(t, forgotPasswordRequest{Email: "invalid@asd.com"}), http.StatusInternalServerError)
	res.Body.Close()

	// Configure SMTP
	var configResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage("{\"smtp_settings\":{\"enable_smtp\":true,\"sender_address\":\"user@example.com\",\"server\":\"127.0.0.1\",\"port\":1025,\"authentication_type\":\"authtype_none\"}}"), http.StatusOK, &configResp)

	// request forgot password, invalid email
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/forgot_password", jsonMustMarshal(t, forgotPasswordRequest{Email: "invalid@asd.com"}), http.StatusAccepted)
	res.Body.Close()

	// TODO: tested manually (adds too much time to the test), works but hitting the rate
	// limit returns 500 instead of 429, see #4406. We get the authz check missing error instead.
	// // trigger the rate limit with a batch of requests in a short burst
	// for i := 0; i < 20; i++ {
	//	s.DoJSON("POST", "/api/latest/fleet/forgot_password", forgotPasswordRequest{Email: "invalid@asd.com"}, http.StatusAccepted, &forgotResp)
	// }

	// request forgot password, valid email
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/forgot_password", jsonMustMarshal(t, forgotPasswordRequest{Email: u.Email}), http.StatusAccepted)
	res.Body.Close()

	var token string
	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), db, &token, "SELECT token FROM password_reset_requests WHERE user_id = ?", u.ID)
	})

	// proceed with reset password
	userNewPwd := test.GoodPassword2
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/reset_password", jsonMustMarshal(t, resetPasswordRequest{PasswordResetToken: token, NewPassword: userNewPwd}), http.StatusOK)
	res.Body.Close()

	// attempt it again with already-used token
	userUnusedPwd := "unusedpassw0rd!"
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/reset_password", jsonMustMarshal(t, resetPasswordRequest{PasswordResetToken: token, NewPassword: userUnusedPwd}), http.StatusUnauthorized)
	res.Body.Close()

	// login with the old password, should not succeed
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/login", jsonMustMarshal(t, fleet.LoginRequest{Email: u.Email, Password: userRawPwd}),
		http.StatusUnauthorized)
	res.Body.Close()

	// login with the new password, should succeed
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/login", jsonMustMarshal(t, fleet.LoginRequest{Email: u.Email, Password: userNewPwd}),
		http.StatusOK)
	res.Body.Close()
}

func (s *integrationTestSuite) TestModifyUser() {
	t := s.T()

	// create a new user
	var createResp createUserResponse
	userRawPwd := test.GoodPassword
	params := fleet.UserPayload{
		Name:                     new("moduser"),
		Email:                    new("moduser@example.com"),
		Password:                 new(userRawPwd),
		GlobalRole:               new(fleet.RoleObserver),
		AdminForcedPasswordReset: new(false),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)
	u := *createResp.User

	s.token = s.getTestToken(u.Email, userRawPwd)
	require.NotEmpty(t, s.token)
	defer func() { s.token = s.getTestAdminToken() }()

	// as the user: modify email without providing current password
	var modResp modifyUserResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		Email: new("moduser2@example.com"),
	}, http.StatusUnprocessableEntity, &modResp)

	// as the user: modify email with invalid password
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		Email:    new("moduser2@example.com"),
		Password: new("nosuchpwd"),
	}, http.StatusForbidden, &modResp)

	// as the user: modify email with current password
	newEmail := "moduser2@example.com"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		Email:    new(newEmail),
		Password: new(userRawPwd),
	}, http.StatusOK, &modResp)
	require.Equal(t, u.ID, modResp.User.ID)
	require.Equal(t, u.Email, modResp.User.Email) // new email is pending confirmation, not changed immediately

	// as the user: set new password without providing current one
	newRawPwd := test.GoodPassword2
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		NewPassword: new(newRawPwd),
	}, http.StatusUnprocessableEntity, &modResp)

	// as the user: set new password with an invalid current password
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		NewPassword: new(newRawPwd),
		Password:    new("nosuchpwd"),
	}, http.StatusForbidden, &modResp)

	// as the user: set new password and change name, with a valid current password
	modResp = modifyUserResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		NewPassword: new(newRawPwd),
		Password:    new(userRawPwd),
		Name:        new("moduser2"),
	}, http.StatusOK, &modResp)
	require.Equal(t, u.ID, modResp.User.ID)
	require.Equal(t, "moduser2", modResp.User.Name)

	s.token = s.getTestToken(testUsers["user2"].Email, testUsers["user2"].PlaintextPassword)

	// as a different user: set new password with different user's old password (ensure
	// any other user that is not admin cannot change another user's password)
	newRawPwd = userRawPwd + "3"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		NewPassword: new(newRawPwd),
		Password:    new(testUsers["user2"].PlaintextPassword),
	}, http.StatusForbidden, &modResp)

	s.token = s.getTestAdminToken()

	// as an admin, set a new email, name and password without a current password
	newRawPwd = userRawPwd + "4"
	modResp = modifyUserResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		SSOEnabled:  new(false),
		NewPassword: new(newRawPwd),
		Email:       new("moduser3@example.com"),
		Name:        new("moduser3"),
	}, http.StatusOK, &modResp)
	require.Equal(t, u.ID, modResp.User.ID)

	// as an admin, set new password that doesn't meet requirements
	invalidUserPwd := "abc"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		NewPassword: new(invalidUserPwd),
	}, http.StatusUnprocessableEntity, &modResp)

	// login as the user, with the last password successfully set (to confirm it is the current one)
	var loginResp fleet.LoginResponse
	resp := s.DoRawNoAuth("POST", "/api/latest/fleet/login", jsonMustMarshal(t, fleet.LoginRequest{
		Email:    u.Email, // all email changes made are still pending, never confirmed
		Password: newRawPwd,
	}), http.StatusOK)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&loginResp))
	resp.Body.Close()
	require.Equal(t, u.ID, loginResp.User.ID)

	// as an admin, api_endpoints must be rejected on this endpoint
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		APIEndpoints: &[]fleet.APIEndpointRef{{Method: "GET", Path: "/api/v1/fleet/config"}},
	}, http.StatusUnprocessableEntity, &modResp)

	// as an admin, create a new user with SSO authentication enabled
	params = fleet.UserPayload{
		Name:                     new("moduser1"),
		Email:                    new("moduser1@example.com"),
		SSOInvite:                new(true),
		GlobalRole:               new(fleet.RoleObserver),
		AdminForcedPasswordReset: new(false),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)
	u = *createResp.User

	// as an admin, try to disable sso for that user without providing a password
	res := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		SSOEnabled: new(false),
	}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "a new password must be provided when disabling SSO")

	// as an admin, try to disable sso for that user while providing a password
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		SSOEnabled:  new(false),
		NewPassword: new("Password123#"),
	}, http.StatusOK)
}

func (s *integrationTestSuite) TestSSODisabled() {
	t := s.T()

	s.DoRawNoAuth("POST", "/api/v1/fleet/sso", nil, http.StatusBadRequest)

	// callback without SAML response
	s.DoRawNoAuth("POST", "/api/v1/fleet/sso/callback", nil, http.StatusBadRequest)

	// callback with invalid SAML response
	s.DoRawNoAuth("POST", "/api/v1/fleet/sso/callback?SAMLResponse=zz", nil, http.StatusBadRequest)

	// callback with valid SAML response
	validResponse := `<?xml version="1.0" encoding="UTF-8"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" Destination="https://localhost:8080/api/v1/fleet/sso/callback" ID="_52f2515c5319f2adf3f072d9d9f2b6881493305396746" InResponseTo="4982b430-73e1-4ad2-885a-4a775a91f820" IssueInstant="2017-04-27T15:03:16.747Z" Version="2.0">
</samlp:Response>`
	samlResponse := base64.StdEncoding.EncodeToString([]byte(validResponse))
	res := s.DoRawNoAuth("POST", "/api/v1/fleet/sso/callback?SAMLResponse="+url.QueryEscape(samlResponse), nil, http.StatusOK)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "/login?status=org_disabled") // html contains a script that redirects to this path
}

var hostIOSVitalsJSONKeys = []string{
	"udid", "model_number", "modem_firmware_version", "supplemental_build_version",
	"supplemental_os_version_extra", "bluetooth_mac", "wifi_mac", "eas_device_identifier",
	"itunes_store_account_hash", "push_token", "battery_level", "cellular_technology",
	"app_analytics_enabled", "awaiting_configuration", "data_roaming_enabled",
	"diagnostic_submission_enabled", "is_cloud_backup_enabled", "is_device_locator_service_enabled",
	"is_do_not_disturb_in_effect", "is_mdm_lost_mode_enabled", "is_network_tethered",
	"itunes_store_account_is_active", "personal_hotspot_enabled", "last_cloud_backup_date",
	"accessibility_settings", "organization_info", "mdm_options", "device_properties_attestation",
	"service_subscriptions",
}

func (s *integrationTestSuite) getHostJSON(path string) map[string]any {
	t := s.T()
	res := s.DoRaw("GET", path, nil, http.StatusOK)
	defer res.Body.Close()

	var raw struct {
		Host map[string]any `json:"host"`
	}
	require.NoError(t, json.NewDecoder(res.Body).Decode(&raw))
	return raw.Host
}

func (s *integrationTestSuite) TestListVulnerabilities() {
	t := s.T()
	var resp listVulnerabilitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)

	// Invalid Order Key
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusBadRequest, &resp, "order_key", "foo", "order_direction", "asc")

	// EE Order Key is an invalid order key
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusBadRequest, &resp, "order_key", "cvss_score", "order_direction", "asc")

	// Exploit is an EE only filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusPaymentRequired, &resp, "exploit", "true")

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	s.Require().Empty(resp.Vulnerabilities)

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		OsqueryHostID:   new(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo1.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "windows",
	})
	require.NoError(t, err)

	err = s.ds.UpdateHostOperatingSystem(context.Background(), host.ID, fleet.OperatingSystem{
		Name:     "Windows 11 Enterprise 22H2",
		Version:  "10.0.19042.1234",
		Platform: "windows",
	})
	require.NoError(t, err)
	allos, err := s.ds.ListOperatingSystems(context.Background())
	require.NoError(t, err)
	var os fleet.OperatingSystem
	for _, o := range allos {
		if o.ID > os.ID {
			os = o
		}
	}

	err = s.ds.UpdateOSVersions(context.Background())
	require.NoError(t, err)

	_, err = s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID:              os.ID,
		CVE:               "CVE-2021-12345",
		ResolvedInVersion: new("10.0.19043.2013"),
	}, fleet.MSRCSource)
	require.NoError(t, err)

	res, err := s.ds.UpdateHostSoftware(context.Background(), host.ID, []fleet.Software{
		{Name: "Google Chrome", Version: "0.0.1", Source: "programs"},
	})
	require.NoError(t, err)
	sw := res.Inserted[0]

	_, err = s.ds.UpsertSoftwareCPEs(context.Background(), []fleet.SoftwareCPE{
		{
			SoftwareID: sw.ID,
			CPE:        "cpe:2.3:a:google:chrome:1.0.0:*:*:*:*:*:*:*:*",
		},
	})
	require.NoError(t, err)

	_, err = s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: sw.ID,
		CVE:        "CVE-2021-1235",
	}, fleet.NVDSource)
	require.NoError(t, err)

	err = s.ds.SyncHostsSoftware(context.Background(), time.Now())
	require.NoError(t, err)

	host2, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(strings.ReplaceAll(t.Name(), "/", "_") + "2"),
		OsqueryHostID:   new(strings.ReplaceAll(t.Name(), "/", "_") + "2"),
		UUID:            t.Name() + "2",
		Hostname:        t.Name() + "foo2.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "windows",
	})
	require.NoError(t, err)

	res2, err := s.ds.UpdateHostSoftware(context.Background(), host2.ID, []fleet.Software{
		{Name: "Firefox", Version: "0.0.1", Source: "programs"},
	})
	require.NoError(t, err)
	sw2 := res2.Inserted[0]

	// insert software vuln outside of host scope
	_, err = s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: sw2.ID,
		CVE:        "CVE-2021-1246",
	}, fleet.NVDSource)
	require.NoError(t, err)

	// insert CVEMeta
	knownCVE := "cve-2021-12999"
	mockTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	err = s.ds.InsertCVEMeta(context.Background(), []fleet.CVEMeta{
		{
			CVE:              "CVE-2021-12345",
			CVSSScore:        new(7.5),
			EPSSProbability:  new(0.5),
			CISAKnownExploit: new(true),
			Published:        new(mockTime),
			Description:      "Test CVE 2021-12345",
		},
		{
			CVE:              "CVE-2021-1235",
			CVSSScore:        new(5.4),
			EPSSProbability:  new(0.6),
			CISAKnownExploit: new(false),
			Published:        new(mockTime),
			Description:      "Test CVE 2021-1235",
		},
		{
			CVE:              "CVE-2021-1246",
			CVSSScore:        new(5.4),
			EPSSProbability:  new(0.6),
			CISAKnownExploit: new(false),
			Published:        new(mockTime),
			Description:      "Test CVE 2021-1246",
		},
		{
			CVE:              knownCVE,
			CVSSScore:        new(6.4),
			EPSSProbability:  new(0.61),
			CISAKnownExploit: new(true),
			Published:        new(mockTime),
			Description:      fmt.Sprintf("Test %s", knownCVE),
		},
	})
	require.NoError(t, err)

	err = s.ds.UpdateVulnerabilityHostCounts(context.Background(), 5)
	require.NoError(t, err)

	// test list
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	require.Empty(t, resp.Err)
	require.Len(s.T(), resp.Vulnerabilities, 3)
	require.Equal(t, resp.Count, uint(3))
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)

	expected := map[string]struct {
		fleet.CVEMeta
		HostCount   uint
		DetailsLink string
		Source      fleet.VulnerabilitySource
	}{
		"CVE-2021-12345": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-12345",
		},
		"CVE-2021-1235": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1235",
		},
		"CVE-2021-1246": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1246",
		},
	}

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE.CVE]
		require.True(t, ok, vuln.CVE.CVE)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Empty(t, vuln.CVSSScore)
	}

	// test list with matching query containing leading/trailing whitespace
	// TODO(jacob) - this may be another parsing bug
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "query", "  123	")
	require.Empty(t, resp.Err)
	require.Len(s.T(), resp.Vulnerabilities, 2)
	require.Equal(t, resp.Count, uint(2))
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)

	expected = map[string]struct {
		fleet.CVEMeta
		HostCount   uint
		DetailsLink string
		Source      fleet.VulnerabilitySource
	}{
		"CVE-2021-12345": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-12345",
		},
		"CVE-2021-1235": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1235",
		},
		// ...1246 should not match the query
	}

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Empty(t, vuln.CVSSScore)
	}

	// test list with non-matching query
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "query", "CVB")
	require.Empty(t, resp.Err)
	require.Len(s.T(), resp.Vulnerabilities, 0)
	require.Equal(t, resp.Count, uint(0))
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)

	// test with a known CVE that does not match on software/OS
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "query", knownCVE)
	require.Empty(t, resp.Err)
	assert.Len(s.T(), resp.Vulnerabilities, 0)
	assert.Equal(t, resp.Count, uint(0))
	assert.False(t, resp.Meta.HasPreviousResults)
	assert.False(t, resp.Meta.HasNextResults)

	// test with a substring of a known CVE -- results are returned
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "query", "CVE-2021-1234")
	require.Empty(t, resp.Err)
	assert.Len(s.T(), resp.Vulnerabilities, 1)
	assert.Equal(t, resp.Count, uint(1))
	assert.False(t, resp.Meta.HasPreviousResults)
	assert.False(t, resp.Meta.HasNextResults)
	_ = s.Do("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1234", nil, http.StatusNotFound)

	// Team 1 Filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", "1")
	require.Len(s.T(), resp.Vulnerabilities, 0)

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
	require.NoError(t, err)

	err = s.ds.UpdateVulnerabilityHostCounts(context.Background(), 5)
	require.NoError(t, err)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", fmt.Sprintf("%d", team.ID))
	require.Len(t, resp.Vulnerabilities, 2)
	require.Equal(t, uint(2), resp.Count)
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)
	require.Empty(t, resp.Err)

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Empty(t, vuln.CVSSScore)
	}

	// No filter (global)
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	require.Len(t, resp.Vulnerabilities, 3)
	require.Equal(t, uint(3), resp.Count)
	require.Equal(t, uint(1), resp.Vulnerabilities[0].HostsCount)
	require.Equal(t, uint(1), resp.Vulnerabilities[1].HostsCount)
	require.Equal(t, uint(1), resp.Vulnerabilities[2].HostsCount)

	// Team 0 Filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", "0")
	require.Len(t, resp.Vulnerabilities, 1)
	require.Equal(t, uint(1), resp.Count)
	require.Equal(t, "CVE-2021-1246", resp.Vulnerabilities[0].CVE.CVE)
	require.Equal(t, uint(1), resp.Vulnerabilities[0].HostsCount)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", "0")
	require.Len(t, resp.Vulnerabilities, 1)

	var gResp getVulnerabilityResponse
	// invalid cve
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/foobar", nil, http.StatusBadRequest, &gResp)

	// Valid CVE but not in team scope
	s.Do("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1246", nil, http.StatusNoContent, "team_id",
		fmt.Sprintf("%d", team.ID))

	// Valid CVE in "no team" scope
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1246", nil, http.StatusOK, &gResp, "team_id", "0")

	// Valid CVE not in "no team" scope
	s.Do("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-12345", nil, http.StatusNoContent, "team_id", "0")

	// Invalid TeamID
	s.Do("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-12345", nil, http.StatusForbidden, "team_id", "100")

	// Valid Global Request
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-12345", nil, http.StatusOK, &gResp)
	require.Empty(t, gResp.Err)
	require.Equal(t, "CVE-2021-12345", gResp.Vulnerability.CVE.CVE)
	require.Equal(t, uint(1), gResp.Vulnerability.HostsCount)
	require.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2021-12345", gResp.Vulnerability.DetailsLink)
	require.Empty(t, gResp.Vulnerability.Description)
	require.Empty(t, gResp.Vulnerability.CVSSScore)
	require.Empty(t, gResp.Vulnerability.CISAKnownExploit)
	require.Empty(t, gResp.Vulnerability.EPSSProbability)
	require.Empty(t, gResp.Vulnerability.CVEPublished)
	require.Len(t, gResp.OSVersions, 1)
	require.Equal(t, "Windows 11 Enterprise 22H2 10.0.19042.1234", gResp.OSVersions[0].Name)
	require.Equal(t, "Windows 11 Enterprise 22H2", gResp.OSVersions[0].NameOnly)
	require.Equal(t, "windows", gResp.OSVersions[0].Platform)
	require.Equal(t, "10.0.19042.1234", gResp.OSVersions[0].Version)
	require.Equal(t, 1, gResp.OSVersions[0].HostsCount)
	require.Equal(t, "10.0.19043.2013", *gResp.OSVersions[0].ResolvedInVersion)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1235", nil, http.StatusOK, &gResp)
	require.Empty(t, gResp.Err)
	require.Equal(t, "CVE-2021-1235", gResp.Vulnerability.CVE.CVE)
	require.Equal(t, uint(1), gResp.Vulnerability.HostsCount)
	require.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2021-1235", gResp.Vulnerability.DetailsLink)
	require.Empty(t, gResp.Vulnerability.Description)
	require.Empty(t, gResp.Vulnerability.CVSSScore)
	require.Empty(t, gResp.Vulnerability.CISAKnownExploit)
	require.Empty(t, gResp.Vulnerability.EPSSProbability)
	require.Empty(t, gResp.Vulnerability.CVEPublished)
	require.Len(t, gResp.Software, 1)
	require.Equal(t, "Google Chrome", gResp.Software[0].Name)
	require.Equal(t, "0.0.1", gResp.Software[0].Version)
	require.Equal(t, "programs", gResp.Software[0].Source)
	require.Equal(t, "cpe:2.3:a:google:chrome:1.0.0:*:*:*:*:*:*:*:*", gResp.Software[0].GenerateCPE)
	require.Equal(t, 1, gResp.Software[0].HostsCount)
}

func (s *integrationTestSuite) TestPingEndpoints() {
	t := s.T()

	s.DoRaw("HEAD", "/api/fleet/orbit/ping", nil, http.StatusOK)
	// unauthenticated works too
	s.DoRawNoAuth("HEAD", "/api/fleet/orbit/ping", nil, http.StatusOK)

	s.DoRaw("HEAD", "/api/fleet/device/ping", nil, http.StatusOK)
	// unauthenticated works too
	s.DoRawNoAuth("HEAD", "/api/fleet/device/ping", nil, http.StatusOK)

	// device authenticated ping
	createHostAndDeviceToken(t, s.ds, "ping-token")
	s.DoRaw("HEAD", fmt.Sprintf("/api/v1/fleet/device/%s/ping", "ping-token"), nil, http.StatusOK)
	s.DoRawNoAuth("HEAD", fmt.Sprintf("/api/v1/fleet/device/%s/ping", "ping-token"), nil, http.StatusOK)
	s.DoRaw("HEAD", fmt.Sprintf("/api/v1/fleet/device/%s/ping", "bozo-token"), nil, http.StatusUnauthorized)
	s.DoRawNoAuth("HEAD", fmt.Sprintf("/api/v1/fleet/device/%s/ping", "bozo-token"), nil, http.StatusUnauthorized)
}

func (s *integrationTestSuite) TestInitiateDeviceSSOFreeTier() {
	t := s.T()

	createHostAndDeviceToken(t, s.ds, "device-sso-token")

	// free tier: no InitiateDeviceSSO implementation beyond the license error,
	// same shape as every other premium-only device endpoint.
	res := s.DoRawNoAuth("POST", "/api/latest/fleet/device/device-sso-token/sso", nil, http.StatusPaymentRequired)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, fleet.ErrMissingLicense.Error())

	// invalid device token behaves like every other device-authenticated route
	s.DoRawNoAuth("POST", "/api/latest/fleet/device/bozo-token/sso", nil, http.StatusUnauthorized)
}

func (s *integrationTestSuite) TestMDMNotConfiguredEndpoints() {
	t := s.T()

	// create a host with device token to test device authenticated routes
	tkn := "D3V1C370K3N"
	h := createHostAndDeviceToken(t, s.ds, tkn)
	orbitKey := setOrbitEnrollment(t, h, s.ds)
	h.OrbitNodeKey = &orbitKey

	windowsOnly := windowsMDMConfigurationRequiredEndpoints()
	androidOnly := androidMDMConfigurationRequiredEndpoints()

	for _, route := range mdmConfigurationRequiredEndpoints() {
		var expectedErr fleet.ErrWithStatusCode = fleet.ErrMDMNotConfigured
		path := route.path
		if slices.Contains(windowsOnly, path) {
			expectedErr = fleet.ErrWindowsMDMNotConfigured
		} else if slices.Contains(androidOnly, path) {
			expectedErr = fleet.ErrAndroidMDMNotConfigured
		}

		if route.deviceAuthenticated {
			path = fmt.Sprintf(path, tkn)
		}

		// build the body of the request
		var params any
		var multipartBody *bytes.Buffer
		var headers map[string]string
		switch {
		case route.method == "POST" && route.path == "/api/fleet/orbit/setup_experience/status":
			params = fleet.GetOrbitSetupExperienceStatusRequest{
				OrbitNodeKey: *h.OrbitNodeKey,
			}

		case route.method == "POST" && route.path == "/api/latest/fleet/software/web_apps":
			multipartBody, headers = generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
				"title": {"Test App"},
				"url":   {"https://example.com"},
			})

		case route.method == "PATCH" && (route.path == "/api/latest/fleet/setup_experience" || route.path == "/api/latest/fleet/mdm/apple/setup"):
			// These routes don't require MDM because they can be used to change end-user auth, but they do require a license.
			expectedErr = fleet.ErrMissingLicense
		}

		var res *http.Response
		if multipartBody != nil {
			res = s.DoRawWithHeaders(route.method, path, multipartBody.Bytes(), expectedErr.StatusCode(), headers)
		} else {
			res = s.Do(route.method, path, params, expectedErr.StatusCode())
		}
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, expectedErr.Error(), fmt.Sprintf("%s %s", route.method, path))
	}

	fleetdmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Setenv("TEST_FLEETDM_API_URL", fleetdmSrv.URL)
	t.Cleanup(fleetdmSrv.Close)

	// Always accessible
	var reqCSRResp requestMDMAppleCSRResponse
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "a@b.c", Organization: "test"}, http.StatusOK, &reqCSRResp)
	s.Do("POST", "/api/latest/fleet/mdm/apple/dep/key_pair", nil, http.StatusOK)
}

func (s *integrationTestSuite) TestOrbitConfigNotifications() {
	t := s.T()
	ctx := context.Background()

	// set the enabled and configured flags,
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	origEnabledAndConfigured := appCfg.MDM.EnabledAndConfigured
	appCfg.MDM.EnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	defer func() {
		appCfg.MDM.EnabledAndConfigured = origEnabledAndConfigured
		err = s.ds.SaveAppConfig(ctx, appCfg)
		require.NoError(t, err)
	}()

	var resp fleet.OrbitGetConfigResponse
	// missing orbit key
	s.DoJSON("POST", "/api/fleet/orbit/config", nil, http.StatusUnauthorized, &resp)

	hNoMDM := createOrbitEnrolledHost(t, "darwin", "nomdm", s.ds)
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hNoMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	hSimpleMDM := createOrbitEnrolledHost(t, "darwin", "simplemdm", s.ds)
	err = s.ds.SetOrUpdateMDMData(context.Background(), hSimpleMDM.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "", false)
	require.NoError(t, err)
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hSimpleMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	// not yet assigned in ABM
	hFleetMDM := createOrbitEnrolledHost(t, "darwin", "fleetmdm", s.ds)
	err = s.ds.SetOrUpdateMDMData(context.Background(), hFleetMDM.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, "", false)
	require.NoError(t, err)

	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	// simulate ABM assignment
	encTok := uuid.NewString()
	abmToken, err := s.ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok), RenewAt: time.Now().Add(30 * 24 * time.Hour)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)
	err = s.ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*hFleetMDM}, abmToken.ID, make(map[uint]time.Time))
	require.NoError(t, err)
	err = s.ds.SetOrUpdateMDMData(context.Background(), hSimpleMDM.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "", false)
	require.NoError(t, err)
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.True(t, resp.Notifications.RenewEnrollmentProfile)

	// if the fleet mdm host is fully enrolled (not pending anymore), then the notification is false
	err = s.ds.SetOrUpdateMDMData(context.Background(), hFleetMDM.ID, false, true, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, "", false)
	require.NoError(t, err)
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	// the scripts orbit endpoints are accessible without license
	s.Do("POST", "/api/fleet/orbit/scripts/request", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusNotFound)
	s.Do("POST", "/api/fleet/orbit/scripts/result", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusBadRequest)
}

func (s *integrationTestSuite) TestTryingToEnrollWithTheWrongSecret() {
	t := s.T()
	ctx := context.Background()

	h, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:   uuid.New().String(),
		Platform:         "darwin",
		LastEnrolledAt:   time.Now(),
		DetailUpdatedAt:  time.Now(),
		RefetchRequested: true,
	})
	require.NoError(t, err)

	var resp endpointer.JsonError
	s.DoJSON("POST", "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
		EnrollSecret:   uuid.New().String(),
		HardwareUUID:   h.UUID,
		HardwareSerial: h.HardwareSerial,
	}, http.StatusUnauthorized, &resp)

	require.Equal(t, resp.Message, "Authentication failed")
}

func (s *integrationTestSuite) TestEnrollOrbitExistingHostNoSerialMatch() {
	t := s.T()
	ctx := context.Background()

	// create a host with minimal information and the serial, no uuid/osquery id
	// (as when created via DEP sync).
	dbZeroTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	h, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:   uuid.New().String(),
		Platform:         "darwin",
		LastEnrolledAt:   dbZeroTime,
		DetailUpdatedAt:  dbZeroTime,
		RefetchRequested: true,
	})
	require.NoError(t, err)

	// create an enroll secret
	secret := uuid.New().String()
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: secret}},
		},
	}, http.StatusOK, &applyResp)

	// enroll the host from orbit, it will NOT match the existing host since MDM
	// is not configured (it will only look for a match by osquery_host_id with
	// the provided uuid).
	var resp enrollOrbitResponse
	hostUUID := uuid.New().String()
	s.DoJSON("POST", "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
		EnrollSecret:   secret,
		HardwareUUID:   hostUUID, // will not match any existing host
		HardwareSerial: h.HardwareSerial,
	}, http.StatusOK, &resp)
	require.NotEmpty(t, resp.OrbitNodeKey)

	// fetch the host, it will NOT match the one created above
	orbitHost, err := s.ds.LoadHostByOrbitNodeKey(ctx, resp.OrbitNodeKey)
	require.NoError(t, err)
	require.NotEqual(t, h.ID, orbitHost.ID)

	// enroll the host from osquery, it should match the Orbit-enrolled host
	var osqueryResp contract.EnrollOsqueryAgentResponse

	// NOTE(mna): using an osquery_host_id that is NOT the host's UUID would not work,
	// because we haven't enabled lookup by UUID due to not having an index and possible
	// side-effects of this on host ingestion performance. However, this should not happen
	// anyway in MDM-enabled environments as we will recommend using the UUID as osquery
	// host identifier.
	// See https://github.com/fleetdm/fleet/issues/9033#issuecomment-1411150758

	osqueryID := hostUUID

	s.DoJSON("POST", "/api/osquery/enroll", contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   secret,
		HostIdentifier: osqueryID,
		HostDetails: map[string]map[string]string{
			"system_info": {
				"uuid":            hostUUID,
				"hardware_serial": h.HardwareSerial,
			},
		},
	}, http.StatusOK, &osqueryResp)
	require.NotEmpty(t, osqueryResp.NodeKey)

	// load the host by osquery node key, should match the orbit host
	got, err := s.ds.LoadHostByNodeKey(ctx, osqueryResp.NodeKey)
	require.NoError(t, err)
	require.Equal(t, orbitHost.ID, got.ID)
}

// this test can be deleted once the "v1" version is removed.
func (s *integrationTestSuite) TestAPIVersion_v1_2022_04() {
	t := s.T()

	// create a query that can be scheduled
	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery2",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Saved:          true,
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	// try to schedule that query on the endpoint that is deprecated
	// in that version
	gsParams := fleet.ScheduledQueryPayload{QueryID: new(qr.ID), Interval: new(uint(42))}
	res := s.DoRaw("POST", "/api/2022-04/fleet/global/schedule", jsonMustMarshal(t, gsParams), http.StatusNotFound)
	res.Body.Close()
	// use the correct version for that deprecated API
	createResp := globalScheduleQueryResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/schedule", gsParams, http.StatusOK, &createResp)
	require.NotZero(t, createResp.Scheduled.ID)

	// list the scheduled queries with the new endpoint, but the old version
	res = s.DoRaw("GET", "/api/v1/fleet/schedule", nil, http.StatusMethodNotAllowed)
	res.Body.Close()

	// list again, this time with the correct version
	gs := fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/2022-04/fleet/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)

	// delete using the old endpoint but on the wrong new version
	res = s.DoRaw("DELETE", fmt.Sprintf("/api/2022-04/fleet/global/schedule/%d", createResp.Scheduled.ID), nil, http.StatusNotFound)
	res.Body.Close()
	// properly delete with old endpoint and old version
	var delResp deleteGlobalScheduleResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/global/schedule/%d", createResp.Scheduled.ID), nil, http.StatusOK, &delResp)
}

type validationErrResp struct {
	Message string `json:"message"`
	Errors  []struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
	} `json:"errors"`
}

func setOrbitEnrollment(t *testing.T, h *fleet.Host, ds fleet.Datastore) string {
	orbitKey := uuid.New().String()
	_, err := ds.EnrollOrbit(
		context.Background(),
		fleet.WithEnrollOrbitHostInfo(fleet.OrbitHostInfo{
			HardwareUUID:   *h.OsqueryHostID,
			HardwareSerial: h.HardwareSerial,
		}),
		fleet.WithEnrollOrbitNodeKey(orbitKey),
		fleet.WithEnrollOrbitTeamID(h.TeamID),
	)
	require.NoError(t, err)
	err = ds.SetOrUpdateHostOrbitInfo(
		context.Background(), h.ID, "1.22.0", sql.NullString{String: "42", Valid: true}, sql.NullBool{Bool: true, Valid: true},
	)
	require.NoError(t, err)
	return orbitKey
}

func createOrbitEnrolledHost(t *testing.T, platform, suffix string, ds fleet.Datastore) *fleet.Host {
	name := t.Name() + suffix
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-time.Minute),
		OsqueryHostID:   new(name),
		NodeKey:         new(name),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", name),
		ComputerName:    name,
		HardwareSerial:  uuid.New().String(),
		Platform:        platform,
	})
	require.NoError(t, err)

	orbitKey := setOrbitEnrollment(t, h, ds)
	h.OrbitNodeKey = &orbitKey
	return h
}

// creates a session and returns it, its key is to be passed as authorization header.
func createSession(t *testing.T, uid uint, ds fleet.Datastore) *fleet.Session {
	ssn, err := ds.NewSession(context.Background(), uid, 64)
	require.NoError(t, err)

	return ssn
}

func (s *integrationTestSuite) cleanupQuery(queryID uint) {
	var delResp fleet.DeleteQueryByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", queryID), nil, http.StatusOK, &delResp)
}

func jsonMustMarshal(t testing.TB, v interface{}) []byte {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// starts a test web server that mocks responses to requests to external
// services with a valid payload (if the request is valid) or a status code
// error. It returns the URL to use to make requests to that server.
//
// For Jira, the project keys "qux" and "qux2" are supported.
// For Zendesk, the group IDs "122" and "123" are supported.
//
// The basic auth's user (or password for Zendesk) "ok" means that auth is
// allowed, while "fail" means unauthorized and anything else results in status
// 502.
func startExternalServiceWebServer(t *testing.T) string {
	// create a test http server to act as the Jira and Zendesk server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(501)
			return
		}

		switch r.URL.Path {
		case "/rest/api/2/project/qux":
			switch usr, _, _ := r.BasicAuth(); usr {
			case "ok":
				_, err := w.Write([]byte(jiraProjectResponsePayload))
				require.NoError(t, err)
			case "fail":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(502)
			}
		case "/rest/api/2/project/qux2":
			switch usr, _, _ := r.BasicAuth(); usr {
			case "ok":
				_, err := w.Write([]byte(jiraProjectResponsePayload))
				require.NoError(t, err)
			case "fail":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(502)
			}
		case "/api/v2/groups/122.json":
			switch _, pwd, _ := r.BasicAuth(); pwd {
			case "ok":
				_, err := w.Write([]byte(`{"group": {"id": 122,"name": "test122"}}`))
				require.NoError(t, err)
			case "fail":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(502)
			}
		case "/api/v2/groups/123.json":
			switch _, pwd, _ := r.BasicAuth(); pwd {
			case "ok":
				_, err := w.Write([]byte(`{"group": {"id": 123,"name": "test123"}}`))
				require.NoError(t, err)
			case "fail":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(502)
			}
		default:
			w.WriteHeader(502)
		}
	}))
	t.Cleanup(srv.Close)

	return srv.URL
}

const (
	// example response from the Jira docs
	jiraProjectResponsePayload = `{
  "self": "https://your-domain.atlassian.net/rest/api/2/project/EX",
  "id": "10000",
  "key": "EX",
  "description": "This project was created as an example for REST.",
  "lead": {
    "self": "https://your-domain.atlassian.net/rest/api/2/user?accountId=5b10a2844c20165700ede21g",
    "key": "",
    "accountId": "5b10a2844c20165700ede21g",
    "accountType": "atlassian",
    "name": "",
    "avatarUrls": {
      "48x48": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=48&s=48",
      "24x24": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=24&s=24",
      "16x16": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=16&s=16",
      "32x32": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=32&s=32"
    },
    "displayName": "Mia Krystof",
    "active": false
  },
  "components": [
    {
      "self": "https://your-domain.atlassian.net/rest/api/2/component/10000",
      "id": "10000",
      "name": "Component 1",
      "description": "This is a Jira component",
      "lead": {
        "self": "https://your-domain.atlassian.net/rest/api/2/user?accountId=5b10a2844c20165700ede21g",
        "key": "",
        "accountId": "5b10a2844c20165700ede21g",
        "accountType": "atlassian",
        "name": "",
        "avatarUrls": {
          "48x48": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=48&s=48",
          "24x24": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=24&s=24",
          "16x16": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=16&s=16",
          "32x32": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=32&s=32"
        },
        "displayName": "Mia Krystof",
        "active": false
      },
      "assigneeType": "PROJECT_LEAD",
      "assignee": {
        "self": "https://your-domain.atlassian.net/rest/api/2/user?accountId=5b10a2844c20165700ede21g",
        "key": "",
        "accountId": "5b10a2844c20165700ede21g",
        "accountType": "atlassian",
        "name": "",
        "avatarUrls": {
          "48x48": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=48&s=48",
          "24x24": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=24&s=24",
          "16x16": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=16&s=16",
          "32x32": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=32&s=32"
        },
        "displayName": "Mia Krystof",
        "active": false
      },
      "realAssigneeType": "PROJECT_LEAD",
      "realAssignee": {
        "self": "https://your-domain.atlassian.net/rest/api/2/user?accountId=5b10a2844c20165700ede21g",
        "key": "",
        "accountId": "5b10a2844c20165700ede21g",
        "accountType": "atlassian",
        "name": "",
        "avatarUrls": {
          "48x48": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=48&s=48",
          "24x24": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=24&s=24",
          "16x16": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=16&s=16",
          "32x32": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=32&s=32"
        },
        "displayName": "Mia Krystof",
        "active": false
      },
      "isAssigneeTypeValid": false,
      "project": "HSP",
      "projectId": 10000
    }
  ],
  "issueTypes": [
    {
      "self": "https://your-domain.atlassian.net/rest/api/2/issueType/3",
      "id": "3",
      "description": "A task that needs to be done.",
      "iconUrl": "https://your-domain.atlassian.net/secure/viewavatar?size=xsmall&avatarId=10299&avatarType=issuetype\",",
      "name": "Task",
      "subtask": false,
      "avatarId": 1,
      "hierarchyLevel": 0
    },
    {
      "self": "https://your-domain.atlassian.net/rest/api/2/issueType/1",
      "id": "1",
      "description": "A problem with the software.",
      "iconUrl": "https://your-domain.atlassian.net/secure/viewavatar?size=xsmall&avatarId=10316&avatarType=issuetype\",",
      "name": "Bug",
      "subtask": false,
      "avatarId": 10002,
      "entityId": "9d7dd6f7-e8b6-4247-954b-7b2c9b2a5ba2",
      "hierarchyLevel": 0,
      "scope": {
        "type": "PROJECT",
        "project": {
          "id": "10000",
          "key": "KEY",
          "name": "Next Gen Project"
        }
      }
    }
  ],
  "url": "https://www.example.com",
  "email": "from-jira@example.com",
  "assigneeType": "PROJECT_LEAD",
  "versions": [],
  "name": "Example",
  "roles": {
    "Developers": "https://your-domain.atlassian.net/rest/api/2/project/EX/role/10000"
  },
  "avatarUrls": {
    "48x48": "https://your-domain.atlassian.net/secure/projectavatar?size=large&pid=10000",
    "24x24": "https://your-domain.atlassian.net/secure/projectavatar?size=small&pid=10000",
    "16x16": "https://your-domain.atlassian.net/secure/projectavatar?size=xsmall&pid=10000",
    "32x32": "https://your-domain.atlassian.net/secure/projectavatar?size=medium&pid=10000"
  },
  "projectCategory": {
    "self": "https://your-domain.atlassian.net/rest/api/2/projectCategory/10000",
    "id": "10000",
    "name": "FIRST",
    "description": "First Project Category"
  },
  "simplified": false,
  "style": "classic",
  "properties": {
    "propertyKey": "propertyValue"
  },
  "insight": {
    "totalIssueCount": 100,
    "lastIssueUpdateTime": "2022-04-05T04:51:35.670+0000"
  }
}`
)

func (s *integrationTestSuite) TestDirectIngestScheduledQueryStats() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "Foobar",
	})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "Zoo",
	})
	require.NoError(t, err)
	globalHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(uuid.New().String()),
		NodeKey:         new(uuid.New().String()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.global", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	team1Host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(uuid.New().String()),
		NodeKey:         new(uuid.New().String()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.team", t.Name()),
		Platform:        "darwin",
		TeamID:          &team1.ID,
	})
	require.NoError(t, err)
	scheduledGlobalQuery, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "scheduled-global-query",
		TeamID:             nil,
		Interval:           10,
		Platform:           "darwin",
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from time;",
		Saved:              true,
	})
	require.NoError(t, err)
	nonScheduledGlobalQuery, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "non-scheduled-global-query",
		TeamID:             nil,
		Interval:           0,
		Platform:           "darwin",
		AutomationsEnabled: false,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from osquery_info;",
		Saved:              true,
	})
	require.NoError(t, err)
	scheduledTeam1Query1, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "scheduled-team1-query1",
		TeamID:             &team1.ID,
		Interval:           20,
		Platform:           "",
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from other;",
		Saved:              true,
	})
	require.NoError(t, err)
	scheduledTeam1Query2, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "scheduled-team1-query2",
		TeamID:             &team1.ID,
		Interval:           90,
		Platform:           "",
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from other;",
		Saved:              true,
	})
	require.NoError(t, err)
	// Create a non-scheduled query to test that we filter it out when providing
	// the queries in the osquery/config endpoint.
	_, err = s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "non-scheduled-team1-query",
		TeamID:             &team1.ID,
		Interval:           0,
		Platform:           "",
		AutomationsEnabled: false,
		Logging:            "snapshot",
		Description:        "foobar",
		Query:              "SELECT * from foobar;",
		Saved:              true,
	})
	require.NoError(t, err)
	// Create a scheduled query but on another team to test that we filter it
	// out when providing the queries in the osquery/config endpoint.
	_, err = s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "scheduled-team2-query",
		TeamID:             &team2.ID,
		Interval:           40,
		Platform:           "",
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from other;",
		Saved:              true,
	})
	require.NoError(t, err)
	// Create a legacy 2017 user pack with one query.
	userPack1TargetTeam1, err := s.ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "2017 Pack",
		Type:    nil,
		Teams:   []fleet.Target{{TargetID: team1.ID, Type: fleet.TargetTeam}},
		TeamIDs: []uint{team1.ID},
	})
	require.NoError(t, err)
	scheduledQueryOnPack1, err := s.ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     "scheduled-query-pack1",
		PackID:   userPack1TargetTeam1.ID,
		QueryID:  nonScheduledGlobalQuery.ID,
		Interval: 60,
		Snapshot: new(true),
		Removed:  new(true),
	})
	require.NoError(t, err)

	// Simulate the osquery instance of the global host calling the osquery/config endpoint
	// and test the returned scheduled queries.
	req := getClientConfigRequest{NodeKey: *globalHost.NodeKey}
	var resp getClientConfigResponse
	s.DoJSON("POST", "/api/osquery/config", req, http.StatusOK, &resp)
	packs := resp.Config["packs"].(map[string]interface{})
	require.Len(t, packs, 1)
	globalQueries := packs["Global"].(map[string]interface{})["queries"].(map[string]interface{})
	require.Len(t, globalQueries, 1)
	require.Contains(t, globalQueries, scheduledGlobalQuery.Name)

	// Simulate the osquery instance of the team host calling the osquery/config endpoint
	// and test the returned scheduled queries.
	req = getClientConfigRequest{NodeKey: *team1Host.NodeKey}
	resp = getClientConfigResponse{}
	s.DoJSON("POST", "/api/osquery/config", req, http.StatusOK, &resp)
	packs = resp.Config["packs"].(map[string]interface{})
	require.Len(t, packs, 3)
	globalQueries = packs["Global"].(map[string]interface{})["queries"].(map[string]interface{})
	require.Len(t, globalQueries, 1)
	require.Contains(t, globalQueries, scheduledGlobalQuery.Name)
	team1Queries := packs[fmt.Sprintf("team-%d", team1.ID)].(map[string]interface{})["queries"].(map[string]interface{})
	require.Len(t, team1Queries, 2)
	require.Contains(t, team1Queries, scheduledTeam1Query1.Name)
	require.Contains(t, team1Queries, scheduledTeam1Query2.Name)
	userPack1Queries := packs[userPack1TargetTeam1.Name].(map[string]interface{})["queries"].(map[string]interface{})
	require.Len(t, userPack1Queries, 1)
	require.Contains(t, userPack1Queries, scheduledQueryOnPack1.Name)

	// Now let's simulate a osquery instance running in the team host returning the
	// stats in the distributed/write (osquery_schedule table)
	rows := []map[string]string{
		{
			"name":              "pack/Global/scheduled-global-query",
			"query":             "SELECT * FROM time;",
			"interval":          "10",
			"executions":        "2",
			"last_executed":     "1693476753",
			"denylisted":        "0",
			"output_size":       "576",
			"wall_time":         "1",
			"wall_time_ms":      "2",
			"last_wall_time_ms": "3",
			"user_time":         "4",
			"last_user_time":    "5",
			"system_time":       "6",
			"last_system_time":  "7",
			"average_memory":    "8",
			"last_memory":       "9",
			"delimiter":         "/",
		},
		{
			"name":              "pack/2017 Pack/scheduled-query-pack1",
			"query":             "SELECT * FROM osquery_info;",
			"interval":          "60",
			"executions":        "20",
			"last_executed":     "1693476842",
			"denylisted":        "0",
			"output_size":       "9620",
			"wall_time":         "9",
			"wall_time_ms":      "8",
			"last_wall_time_ms": "7",
			"user_time":         "6",
			"last_user_time":    "5",
			"system_time":       "4",
			"last_system_time":  "3",
			"average_memory":    "2",
			"last_memory":       "1",
			"delimiter":         "/",
		},
		{
			"name":              fmt.Sprintf("pack/team-%d/scheduled-team1-query1", team1.ID),
			"query":             "SELECT * FROM other;",
			"interval":          "20",
			"executions":        "1",
			"last_executed":     "1693476561",
			"denylisted":        "0",
			"output_size":       "10",
			"wall_time":         "11",
			"wall_time_ms":      "12",
			"last_wall_time_ms": "13",
			"user_time":         "14",
			"last_user_time":    "15",
			"system_time":       "16",
			"last_system_time":  "17",
			"average_memory":    "18",
			"last_memory":       "19",
			"delimiter":         "/",
		},
		{
			"name":              fmt.Sprintf("pack/team-%d/scheduled-team1-query2", team1.ID),
			"query":             "SELECT * FROM other;",
			"interval":          "90",
			"executions":        "5",
			"last_executed":     "1693476666",
			"denylisted":        "0",
			"output_size":       "20",
			"wall_time":         "21",
			"wall_time_ms":      "22",
			"last_wall_time_ms": "23",
			"user_time":         "24",
			"last_user_time":    "25",
			"system_time":       "26",
			"last_system_time":  "27",
			"average_memory":    "28",
			"last_memory":       "29",
			"delimiter":         "/",
		},
	}

	appConfig, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{
		App: config.AppConfig{
			EnableScheduledQueryStats: true,
		},
	}, appConfig, &appConfig.Features, osquery_utils.Integrations{}, nil)
	task := async.NewTask(s.ds, nil, clock.C, nil)
	err = detailQueries["scheduled_query_stats"].DirectTaskIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		team1Host,
		task,
		rows,
	)
	require.NoError(t, err)

	// Check that the received stats were stored in the DB as expected.
	var scheduledQueriesStats []fleet.ScheduledQueryStats
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(
			context.Background(), q, &scheduledQueriesStats,
			`SELECT
				scheduled_query_id, q.name AS scheduled_query_name, average_memory, denylisted,
				executions, q.schedule_interval, last_executed,
				output_size, system_time, user_time, wall_time
			FROM scheduled_query_stats sqs
			JOIN queries q ON sqs.scheduled_query_id = q.id
			WHERE host_id = ?;`,
			team1Host.ID,
		)
	})
	require.Len(t, scheduledQueriesStats, 4)
	rowsMap := make(map[string]map[string]string)
	for _, row := range rows {
		parts := strings.Split(row["name"], "/")
		queryName := parts[len(parts)-1]
		// we need to map this because 2017 packs send the name of the schedule and not
		// the name of the query.
		if queryName == "scheduled-query-pack1" {
			queryName = "non-scheduled-global-query"
		}
		rowsMap[queryName] = row
	}
	for _, sqs := range scheduledQueriesStats {
		row := rowsMap[sqs.ScheduledQueryName]
		require.Equal(t, fmt.Sprint(sqs.AverageMemory), row["average_memory"])
		require.Equal(t, fmt.Sprint(sqs.Executions), row["executions"])
		interval := row["interval"]
		if sqs.ScheduledQueryName == "non-scheduled-global-query" {
			interval = "0" // this query has metrics because it runs on a pack.
		}
		require.Equal(t, strconv.FormatInt(int64(sqs.Interval), 10), interval)
		lastExecuted, err := strconv.ParseInt(row["last_executed"], 10, 64)
		require.NoError(t, err)
		require.WithinDuration(t, sqs.LastExecuted, time.Unix(lastExecuted, 0), 1*time.Second)
		require.Equal(t, fmt.Sprint(sqs.OutputSize), row["output_size"])
		require.Equal(t, fmt.Sprint(sqs.SystemTime), row["system_time"])
		require.Equal(t, fmt.Sprint(sqs.UserTime), row["user_time"])
		assert.Equal(t, fmt.Sprint(sqs.WallTime), row["wall_time_ms"])
	}

	// Now let's simulate a osquery instance running in the global host returning the
	// stats in the distributed/write (osquery_schedule table)
	rows = []map[string]string{
		{
			"name":              "pack/Global/scheduled-global-query",
			"query":             "SELECT * FROM time;",
			"interval":          "10",
			"executions":        "2",
			"last_executed":     "1693476753",
			"denylisted":        "0",
			"output_size":       "576",
			"wall_time":         "1",
			"wall_time_ms":      "2",
			"last_wall_time_ms": "3",
			"user_time":         "4",
			"last_user_time":    "5",
			"system_time":       "6",
			"last_system_time":  "7",
			"average_memory":    "8",
			"last_memory":       "9",
			"delimiter":         "/",
		},
	}

	err = detailQueries["scheduled_query_stats"].DirectTaskIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		globalHost,
		task,
		rows,
	)
	require.NoError(t, err)

	// Check that the received stats were stored in the DB as expected.
	scheduledQueriesStats = []fleet.ScheduledQueryStats{}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(
			context.Background(), q, &scheduledQueriesStats,
			`SELECT
				scheduled_query_id, q.name AS scheduled_query_name, average_memory, denylisted,
				executions, q.schedule_interval, last_executed,
				output_size, system_time, user_time, wall_time
			FROM scheduled_query_stats sqs
			JOIN queries q ON sqs.scheduled_query_id = q.id
			WHERE host_id = ?;`,
			globalHost.ID,
		)
	})
	require.Len(t, scheduledQueriesStats, 1)
	row := rows[0]
	parts := strings.Split(row["name"], "/")
	queryName := parts[len(parts)-1]
	sqs := scheduledQueriesStats[0]
	require.Equal(t, scheduledQueriesStats[0].ScheduledQueryName, queryName)
	require.Equal(t, fmt.Sprint(sqs.AverageMemory), row["average_memory"])
	require.Equal(t, fmt.Sprint(sqs.Executions), row["executions"])
	require.Equal(t, fmt.Sprint(sqs.Interval), row["interval"])
	lastExecuted, err := strconv.ParseInt(row["last_executed"], 10, 64)
	require.NoError(t, err)
	require.WithinDuration(t, sqs.LastExecuted, time.Unix(lastExecuted, 0), 1*time.Second)
	require.Equal(t, fmt.Sprint(sqs.OutputSize), row["output_size"])
	require.Equal(t, fmt.Sprint(sqs.SystemTime), row["system_time"])
	require.Equal(t, fmt.Sprint(sqs.UserTime), row["user_time"])
	require.Equal(t, fmt.Sprint(sqs.WallTime), row["wall_time_ms"])
}

// TestDirectIngestSoftwareWithLongFields tests that software with reported long fields
// are inserted properly and subsequent reports of the same software do not generate new
// entries in the `software` table. (It mainly tests the comparison between the currenly
// inserted software and the incoming software from a host.)
func (s *integrationTestSuite) TestDirectIngestSoftwareWithLongFields() {
	t := s.T()

	appConfig, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	appConfig.Features.EnableSoftwareInventory = true

	globalHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(uuid.New().String()),
		NodeKey:         new(uuid.New().String()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.global", t.Name()),
		Platform:        "windows",
	})
	require.NoError(t, err)

	// Simulate a osquery agent on Windows reporting a software row for Wireshark.
	rows := []map[string]string{
		{
			"name":           "Wireshark 4.0.8 64-bit",
			"version":        "4.0.8",
			"type":           "Program (Windows)",
			"source":         "programs",
			"vendor":         "The Wireshark developer community, https://www.wireshark.org",
			"installed_path": "C:\\Program Files\\Wireshark",
		},
	}
	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, appConfig, &appConfig.Features, osquery_utils.Integrations{}, nil)
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)

	// Check that the software was properly ingested.
	softwareQueryByName := "SELECT id, name, version, source, bundle_identifier, `release`, arch, vendor FROM software WHERE name = ?;"
	var wiresharkSoftware fleet.Software
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &wiresharkSoftware, softwareQueryByName, "Wireshark 4.0.8 64-bit")
	})
	require.NotZero(t, wiresharkSoftware.ID)
	require.Equal(t, "Wireshark 4.0.8 64-bit", wiresharkSoftware.Name)
	require.Equal(t, "4.0.8", wiresharkSoftware.Version)
	require.Equal(t, "programs", wiresharkSoftware.Source)
	require.Empty(t, wiresharkSoftware.BundleIdentifier)
	require.Empty(t, wiresharkSoftware.Release)
	require.Empty(t, wiresharkSoftware.Arch)
	require.Equal(t, "The Wireshark developer community, https://www.wireshark.org", wiresharkSoftware.Vendor)
	hostSoftwareInstalledPathsQuery := `SELECT installed_path FROM host_software_installed_paths WHERE software_id = ?;`
	var wiresharkSoftwareInstalledPath string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &wiresharkSoftwareInstalledPath, hostSoftwareInstalledPathsQuery, wiresharkSoftware.ID)
	})
	require.Equal(t, "C:\\Program Files\\Wireshark", wiresharkSoftwareInstalledPath)

	// We now check that the same software is not created again as a new row when it is received again during software ingestion.
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)
	var wiresharkSoftware2 fleet.Software
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &wiresharkSoftware2, softwareQueryByName, "Wireshark 4.0.8 64-bit")
	})
	require.NotZero(t, wiresharkSoftware2.ID)
	require.Equal(t, wiresharkSoftware.ID, wiresharkSoftware2.ID)

	// Simulate a osquery agent on Windows reporting a software row with a longer than 114 chars vendor field.
	rows = []map[string]string{
		{
			"name":           "Foobar" + strings.Repeat("A", fleet.SoftwareNameMaxLength),
			"version":        "4.0.8" + strings.Repeat("B", fleet.SoftwareVersionMaxLength),
			"type":           "Program (Windows)",
			"source":         "programs" + strings.Repeat("C", fleet.SoftwareSourceMaxLength),
			"vendor":         strings.Repeat("D", fleet.SoftwareVendorMaxLength+1),
			"installed_path": "C:\\Program Files\\Foobar",
			// Test UTF-8 encoded strings.
			"bundle_identifier": strings.Repeat("⌘", fleet.SoftwareBundleIdentifierMaxLength+1),
			"release":           strings.Repeat("F", fleet.SoftwareReleaseMaxLength-1) + "⌘⌘",
			"arch":              strings.Repeat("G", fleet.SoftwareArchMaxLength+1),
		},
	}

	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)

	var foobarSoftware fleet.Software
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &foobarSoftware, softwareQueryByName, "Foobar"+strings.Repeat("A", fleet.SoftwareNameMaxLength-6))
	})
	require.NotZero(t, foobarSoftware.ID)
	require.Equal(t, "Foobar"+strings.Repeat("A", fleet.SoftwareNameMaxLength-6), foobarSoftware.Name)
	require.Equal(t, "4.0.8"+strings.Repeat("B", fleet.SoftwareNameMaxLength-5), foobarSoftware.Version)
	require.Equal(t, "programs"+strings.Repeat("C", fleet.SoftwareSourceMaxLength-8), foobarSoftware.Source)
	// Vendor field is currenty trimmed with a different method (... appended at the end)
	require.Equal(t, strings.Repeat("D", fleet.SoftwareVendorMaxLength-3)+"...", foobarSoftware.Vendor)
	require.Equal(t, strings.Repeat("⌘", fleet.SoftwareBundleIdentifierMaxLength), foobarSoftware.BundleIdentifier)
	require.Equal(t, strings.Repeat("F", fleet.SoftwareReleaseMaxLength-1)+"⌘", foobarSoftware.Release)
	require.Equal(t, strings.Repeat("G", fleet.SoftwareArchMaxLength), foobarSoftware.Arch)

	// We now check that the same software with long (to be trimmed) fields is not created again as a new row.
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)

	var foobarSoftware2 fleet.Software
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &foobarSoftware2, softwareQueryByName, "Foobar"+strings.Repeat("A", fleet.SoftwareNameMaxLength-6))
	})
	require.Equal(t, foobarSoftware.ID, foobarSoftware2.ID)
}

func (s *integrationTestSuite) TestDirectIngestSoftwareWithInvalidFields() {
	t := s.T()

	appConfig, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	appConfig.Features.EnableSoftwareInventory = true

	globalHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(uuid.New().String()),
		NodeKey:         new(uuid.New().String()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.global", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// Ingesting software without name should not fail, but the software won't be inserted.
	rows := []map[string]string{
		{
			"version":        "4.0.8",
			"type":           "Program (Windows)",
			"source":         "programs",
			"vendor":         "The Wireshark developer community, https://www.wireshark.org",
			"installed_path": "C:\\Program Files\\Wireshark",
			"last_opened_at": "foobar",
		},
	}
	handler1 := logtestutils.NewTestHandler()
	logger1 := slog.New(handler1)
	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, appConfig, &appConfig.Features, osquery_utils.Integrations{}, nil)
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		logger1,
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)
	records1 := handler1.Records()
	require.NotEmpty(t, records1)
	// The "empty name" error is returned by SoftwareFromOsqueryRow and logged as an attribute
	// of the "failed to parse software row" debug message.
	var foundEmptyName bool
	for _, r := range records1 {
		if r.Level == slog.LevelDebug && r.Message == "failed to parse software row" {
			attrs := logtestutils.RecordAttrs(&r)
			if err, ok := attrs["err"]; ok {
				if errStr := fmt.Sprint(err); strings.Contains(errStr, "host reported software with empty name") {
					foundEmptyName = true
					break
				}
			}
		}
	}
	require.True(t, foundEmptyName, "expected a debug log about software with empty name")

	// Check that the software was not ingested.
	softwareQueryByVendor := "SELECT id, name, version, source, bundle_identifier, `release`, arch, vendor FROM software WHERE vendor = ?;"
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var wiresharkSoftware fleet.Software
		if sqlx.GetContext(context.Background(), q, &wiresharkSoftware, softwareQueryByVendor, "The Wireshark developer community, https://www.wireshark.org") != sql.ErrNoRows {
			return errors.New("expected no results")
		}
		return nil
	})

	// Ingesting software without source should not fail, but the software won't be inserted.
	rows = []map[string]string{
		{
			"name":           "Wireshark 4.0.8 64-bit",
			"version":        "4.0.8",
			"type":           "Program (Windows)",
			"vendor":         "The Wireshark developer community, https://www.wireshark.org",
			"installed_path": "C:\\Program Files\\Wireshark",
			"last_opened_at": "foobar",
		},
	}
	detailQueries = osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, appConfig, &appConfig.Features, osquery_utils.Integrations{}, nil)
	handler2 := logtestutils.NewTestHandler()
	logger2 := slog.New(handler2)
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		logger2,
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)
	records2 := handler2.Records()
	require.NotEmpty(t, records2)
	var foundEmptySource bool
	for _, r := range records2 {
		if r.Level == slog.LevelDebug && r.Message == "failed to parse software row" {
			attrs := logtestutils.RecordAttrs(&r)
			if err, ok := attrs["err"]; ok {
				if errStr := fmt.Sprint(err); strings.Contains(errStr, "host reported software with empty source") {
					foundEmptySource = true
					break
				}
			}
		}
	}
	require.True(t, foundEmptySource, "expected a debug log about software with empty source")

	// Check that the software was not ingested.
	softwareQueryByName := "SELECT id, name, version, source, bundle_identifier, `release`, arch, vendor FROM software WHERE name = ?;"
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var wiresharkSoftware fleet.Software
		if sqlx.GetContext(context.Background(), q, &wiresharkSoftware, softwareQueryByName, "Wireshark 4.0.8 64-bit") != sql.ErrNoRows {
			return errors.New("expected no results")
		}
		return nil
	})

	// Ingesting software with invalid last_opened_at should not fail (only log a debug error)
	rows = []map[string]string{
		{
			"name":           "Wireshark 4.0.8 64-bit",
			"version":        "4.0.8",
			"type":           "Program (Windows)",
			"source":         "programs",
			"vendor":         "The Wireshark developer community, https://www.wireshark.org",
			"installed_path": "C:\\Program Files\\Wireshark",
			"last_opened_at": "foobar",
		},
	}
	handler3 := logtestutils.NewTestHandler()
	logger3 := slog.New(handler3)
	detailQueries = osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, appConfig, &appConfig.Features, osquery_utils.Integrations{}, nil)
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		logger3,
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)
	records3 := handler3.Records()
	require.NotEmpty(t, records3)
	var foundInvalidTimestamp bool
	for _, r := range records3 {
		if r.Level == slog.LevelDebug && r.Message == "host reported software with invalid last opened timestamp" {
			foundInvalidTimestamp = true
			break
		}
	}
	require.True(t, foundInvalidTimestamp, "expected a debug log about software with invalid last opened timestamp")

	// Check that the software was properly ingested.
	var wiresharkSoftware fleet.Software
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &wiresharkSoftware, softwareQueryByName, "Wireshark 4.0.8 64-bit")
	})
	require.NotZero(t, wiresharkSoftware.ID)
}

func (s *integrationTestSuite) TestOrbitConfigExtensions() {
	t := s.T()
	ctx := context.Background()

	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	defer func() {
		err = s.ds.SaveAppConfig(ctx, appCfg)
		require.NoError(t, err)
	}()

	// Orbit client gets no extensions if extensions are not configured.
	orbitLinuxClient := createOrbitEnrolledHost(t, "linux", "foobar1", s.ds)
	resp := fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitLinuxClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.Extensions)

	// Attempt to add extensions (should succeed).
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
	"agent_options": {
		"config": {
			"options": {
				"pack_delimiter": "/",
				"logger_tls_period": 10,
				"distributed_plugin": "tls",
				"disable_distributed": false,
				"logger_tls_endpoint": "/api/osquery/log",
				"distributed_interval": 10,
				"distributed_tls_max_attempts": 3
			}
		},
		"extensions": {
			"hello_world_linux": {
				"channel": "stable",
				"platform": "linux"
			},
			"hello_mars_linux": {
				"channel": "stable",
				"platform": "linux"
			},
			"hello_world_macos": {
				"channel": "stable",
				"platform": "macos"
			}
		}
	}
}`), http.StatusOK)

	// Attempt to add labels to extensions (only available on premium).
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
  "agent_options": {
	"config": {
		"options": {
		"pack_delimiter": "/",
		"logger_tls_period": 10,
		"distributed_plugin": "tls",
		"disable_distributed": false,
		"logger_tls_endpoint": "/api/osquery/log",
		"distributed_interval": 10,
		"distributed_tls_max_attempts": 3
		}
	},
	"extensions": {
		"hello_world_linux": {
			"channel": "stable",
			"platform": "linux"
		},
		"hello_world_macos": {
			"labels": [
				"All hosts",
				"Some label"
			],
			"channel": "stable",
			"platform": "macos"
		},
		"hello_world_windows": {
			"channel": "stable",
			"platform": "windows"
		}
	}
  }
}`), http.StatusBadRequest)

	// Orbit client gets extensions configured for its platform.
	orbitDarwinClient := createOrbitEnrolledHost(t, "darwin", "foobar2", s.ds)
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitDarwinClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
    "hello_world_macos": {
      "platform": "macos",
      "channel": "stable"
    }
  }`, string(resp.Extensions))

	orbitWindowsClient := createOrbitEnrolledHost(t, "windows", "foobar3", s.ds)

	// Orbit client gets no extensions if none of the platforms target it.
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitWindowsClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.Extensions)

	// Orbit client gets the two extensions configured for its platform.
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitLinuxClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
	"hello_world_linux": {
		"channel": "stable",
		"platform": "linux"
	},
	"hello_mars_linux": {
		"channel": "stable",
		"platform": "linux"
	}
  }`, string(resp.Extensions))
}

func (s *integrationTestSuite) TestQueryReports() {
	t := s.T()
	ctx := context.Background()

	// Add mock expectations to the existing mock (the server already has a reference to it)
	counts := make(map[uint]int)
	s.lq.GetQueryResultsCountsOverride = func(queryIDs []uint) (map[uint]int, error) {
		return counts, nil
	}
	s.lq.SetQueryResultsCountOverride = func(queryID uint, count int) error {
		counts[queryID] = count
		return nil
	}
	s.lq.IncrQueryResultsCountsOverride = func(queryIDsToAmounts map[uint]int) error {
		for queryID, amount := range queryIDsToAmounts {
			counts[queryID] += amount
		}
		return nil
	}
	s.lq.DeleteQueryResultsCountOverride = func(queryID uint) error {
		delete(counts, queryID)
		return nil
	}
	defer func() {
		s.lq.GetQueryResultsCountsOverride = nil
		s.lq.SetQueryResultsCountOverride = nil
		s.lq.IncrQueryResultsCountsOverride = nil
		s.lq.DeleteQueryResultsCountOverride = nil
	}()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{
		ID:          43,
		Name:        "team2",
		Description: "desc team2",
	})
	require.NoError(t, err)

	host1Global, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("1"),
		UUID:            "1",
		Hostname:        "foo.local1",
		OsqueryHostID:   new("1"),
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		Platform:        "ubuntu",
	})
	require.NoError(t, err)

	host2Global, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("2"),
		UUID:            "2",
		Hostname:        "foo.local2",
		OsqueryHostID:   new("2"),
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "ubuntu",
	})
	require.NoError(t, err)

	host2Team1, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("3"),
		UUID:            "3",
		ComputerName:    "Foo Local3",
		Hostname:        "foo.local3",
		OsqueryHostID:   new("3"),
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "30-65-EC-6F-C4-60",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team1.ID, []uint{host2Team1.ID}))
	require.NoError(t, err)

	host1Team2, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("4"),
		UUID:            "4",
		ComputerName:    "Foo Local4",
		Hostname:        "foo.local4",
		OsqueryHostID:   new("4"),
		PrimaryIP:       "192.168.1.4",
		PrimaryMac:      "30-65-EC-6F-C4-61",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team2.ID, []uint{host1Team2.ID}))
	require.NoError(t, err)

	osqueryInfoQuery, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:               "Osquery info",
		Description:        "osquery_info table",
		Query:              "select * from osquery_info;",
		Saved:              true,
		Interval:           30,
		AutomationsEnabled: true,
		DiscardData:        false,
		TeamID:             nil,
		Logging:            fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	usbDevicesQuery, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:               "USB devices",
		Description:        "usb_devices table",
		Query:              "select * from usb_devices;",
		Saved:              true,
		Interval:           60,
		AutomationsEnabled: true,
		DiscardData:        false,
		TeamID:             new(team1.ID),
		Logging:            fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	// Should return no results.
	var gqrr fleet.GetQueryReportResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", usbDevicesQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.NoError(t, gqrr.Err)
	require.Equal(t, usbDevicesQuery.ID, gqrr.QueryID)
	require.NotNil(t, gqrr.Results)
	require.Len(t, gqrr.Results, 0)

	var ghqrr getHostQueryReportResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host1Global.ID, usbDevicesQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Equal(t, usbDevicesQuery.ID, ghqrr.QueryID)
	require.Equal(t, host1Global.ID, ghqrr.HostID)
	require.Nil(t, ghqrr.LastFetched)
	require.False(t, ghqrr.ReportClipped)
	require.NotNil(t, ghqrr.Results)
	require.Len(t, ghqrr.Results, 0)

	slreq := submitLogsRequest{
		NodeKey: *host2Team1.NodeKey,
		LogType: "result",
		Data: []json.RawMessage{
			json.RawMessage(`{
  "snapshot": [
    {
      "class": "239",
      "model": "HD Pro Webcam C920",
      "model_id": "0892",
      "protocol": "",
      "removable": "1",
      "serial": "zoobar",
      "subclass": "2",
      "usb_address": "3",
      "usb_port": "1",
      "vendor": "",
      "vendor_id": "046d",
      "version": "0.19"
    },
    {
      "class": "0",
      "model": "Apple Internal Keyboard / Trackpad",
      "model_id": "027e",
      "protocol": "",
      "removable": "0",
      "serial": "foobar",
      "subclass": "0",
      "usb_address": "8",
      "usb_port": "5",
      "vendor": "Apple Inc.",
      "vendor_id": "05ac",
      "version": "9.33"
    }
  ],
  "action": "snapshot",
  "name": "pack/team-` + usbDevicesQuery.TeamIDStr() + `/` + usbDevicesQuery.Name + `",
  "hostIdentifier": "` + *host2Team1.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 17:32:08 2023 UTC",
  "unixTime": 1696613528,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "` + host2Team1.UUID + `",
    "hostname": "` + host2Team1.Hostname + `"
  }
}`), json.RawMessage(`{
  "snapshot": [
    {
      "build_distro": "10.14",
      "build_platform": "darwin",
      "config_hash": "eed0d8296e5f90b790a23814a9db7a127b13498d",
      "config_valid": "1",
      "extensions": "active",
      "instance_id": "7f02ff0f-f8a7-4ba9-a1d2-66836b154f4a",
      "pid": "95637",
      "platform_mask": "21",
      "start_time": "1696611201",
      "uuid": "` + host2Team1.UUID + `",
      "version": "5.9.1",
      "watcher": "95636"
    }
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host2Team1.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:08:18 2023 UTC",
  "unixTime": 1696615698,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "` + host2Team1.UUID + `",
    "hostname": "` + host2Team1.Hostname + `"
  }
}`),
		},
	}
	slres := submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	slreq = submitLogsRequest{
		NodeKey: *host1Global.NodeKey,
		LogType: "result",
		Data: []json.RawMessage{
			json.RawMessage(`{
  "snapshot": [
    {
      "build_distro": "centos7",
      "build_platform": "linux",
      "config_hash": "eed0d8296e5f90b790a23814a9db7a127b13498d",
      "config_valid": "1",
      "extensions": "active",
      "instance_id": "e5799132-85ab-4cfa-89f3-03e0dd3c509a",
      "pid": "3574",
      "platform_mask": "9",
      "start_time": "1696502961",
      "uuid": "` + host1Global.UUID + `",
      "version": "5.9.2",
      "watcher": "3570"
    }
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host1Global.Hostname + `"
  }
}`),
		},
	}
	slres = submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	require.Equal(t, 2, counts[osqueryInfoQuery.ID])

	slreq = submitLogsRequest{
		NodeKey: *host1Team2.NodeKey,
		LogType: "result",
		Data: []json.RawMessage{
			json.RawMessage(`{
  "snapshot": [
    {
      "build_distro": "10.14",
      "build_platform": "darwin",
      "config_hash": "ca2bc81cd5e79132cb0f842c433ad7f84c056c12",
      "config_valid": "1",
      "extensions": "active",
      "instance_id": "975f2ce1-8672-4932-85f8-340272820e79",
      "pid": "1039",
      "platform_mask": "21",
      "start_time": "1733334052",
      "uuid": "` + host1Team2.UUID + `",
      "version": "5.14.1",
      "watcher": "1037"
    }
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Team2.OsqueryHostID + `",
  "calendarTime": "Mon Dec  16 13:28:00 2024 UTC",
  "unixTime": 1734377280,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "` + host1Team2.UUID + `",
    "hostname": "` + host1Team2.Hostname + `"
  }
}`),
		},
	}
	slres = submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	require.Equal(t, 3, counts[osqueryInfoQuery.ID])

	emptyslreq := submitLogsRequest{
		NodeKey: *host2Global.NodeKey,
		LogType: "result",
		Data: []json.RawMessage{
			json.RawMessage(`{
			  "snapshot": [],
			  "action": "snapshot",
			  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
			  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
			  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
			  "unixTime": 1696615984,
			  "epoch": 0,
			  "counter": 0,
			  "numerics": false,
			  "decorations": {
				"host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
				"hostname": "` + host1Global.Hostname + `"
			  }
			}`),
		},
	}
	emptyslres := submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", emptyslreq, http.StatusOK, &emptyslres)
	require.NoError(t, emptyslres.Err)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	// Count stays the same because error rows don't count against the limit.
	require.Equal(t, 3, counts[osqueryInfoQuery.ID])

	gqrr = fleet.GetQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", usbDevicesQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.NoError(t, gqrr.Err)
	require.Equal(t, usbDevicesQuery.ID, gqrr.QueryID)
	require.Len(t, gqrr.Results, 2)
	sort.Slice(gqrr.Results, func(i, j int) bool {
		// Let's just pick a known column of the query to sort.
		return gqrr.Results[i].Columns["usb_port"] < gqrr.Results[j].Columns["usb_port"]
	})
	require.Equal(t, host2Team1.ID, gqrr.Results[0].HostID)
	require.Equal(t, host2Team1.DisplayName(), gqrr.Results[0].Hostname)
	require.NotZero(t, gqrr.Results[0].LastFetched)
	require.Equal(t, map[string]string{
		"class":       "239",
		"model":       "HD Pro Webcam C920",
		"model_id":    "0892",
		"protocol":    "",
		"removable":   "1",
		"serial":      "zoobar",
		"subclass":    "2",
		"usb_address": "3",
		"usb_port":    "1",
		"vendor":      "",
		"vendor_id":   "046d",
		"version":     "0.19",
	}, gqrr.Results[0].Columns)
	require.Equal(t, host2Team1.ID, gqrr.Results[1].HostID)
	require.Equal(t, host2Team1.DisplayName(), gqrr.Results[1].Hostname)
	require.NotZero(t, gqrr.Results[1].LastFetched)
	require.Equal(t, map[string]string{
		"class":       "0",
		"model":       "Apple Internal Keyboard / Trackpad",
		"model_id":    "027e",
		"protocol":    "",
		"removable":   "0",
		"serial":      "foobar",
		"subclass":    "0",
		"usb_address": "8",
		"usb_port":    "5",
		"vendor":      "Apple Inc.",
		"vendor_id":   "05ac",
		"version":     "9.33",
	}, gqrr.Results[1].Columns)

	ghqrr = getHostQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host2Team1.ID, usbDevicesQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Equal(t, usbDevicesQuery.ID, ghqrr.QueryID)
	require.Equal(t, host2Team1.ID, ghqrr.HostID)
	require.NotNil(t, ghqrr.LastFetched)
	require.False(t, ghqrr.ReportClipped)
	require.Len(t, ghqrr.Results, 2)
	sort.Slice(gqrr.Results, func(i, j int) bool {
		// Let's just pick a known column of the query to sort.
		return gqrr.Results[i].Columns["usb_port"] < gqrr.Results[j].Columns["usb_port"]
	})
	require.Equal(t, map[string]string{
		"class":       "239",
		"model":       "HD Pro Webcam C920",
		"model_id":    "0892",
		"protocol":    "",
		"removable":   "1",
		"serial":      "zoobar",
		"subclass":    "2",
		"usb_address": "3",
		"usb_port":    "1",
		"vendor":      "",
		"vendor_id":   "046d",
		"version":     "0.19",
	}, ghqrr.Results[0].Columns)
	require.Equal(t, map[string]string{
		"class":       "0",
		"model":       "Apple Internal Keyboard / Trackpad",
		"model_id":    "027e",
		"protocol":    "",
		"removable":   "0",
		"serial":      "foobar",
		"subclass":    "0",
		"usb_address": "8",
		"usb_port":    "5",
		"vendor":      "Apple Inc.",
		"vendor_id":   "05ac",
		"version":     "9.33",
	}, ghqrr.Results[1].Columns)

	gqrr = fleet.GetQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.NoError(t, gqrr.Err)
	require.Equal(t, osqueryInfoQuery.ID, gqrr.QueryID)
	require.Len(t, gqrr.Results, 3)
	sort.Slice(gqrr.Results, func(i, j int) bool {
		// Let's just pick a known column of the query to sort.
		return gqrr.Results[i].Columns["version"] > gqrr.Results[j].Columns["version"]
	})
	require.Equal(t, host1Global.ID, gqrr.Results[0].HostID)
	require.Equal(t, host1Global.DisplayName(), gqrr.Results[0].Hostname)
	require.NotZero(t, gqrr.Results[0].LastFetched)
	require.Equal(t, map[string]string{
		"build_distro":   "centos7",
		"build_platform": "linux",
		"config_hash":    "eed0d8296e5f90b790a23814a9db7a127b13498d",
		"config_valid":   "1",
		"extensions":     "active",
		"instance_id":    "e5799132-85ab-4cfa-89f3-03e0dd3c509a",
		"pid":            "3574",
		"platform_mask":  "9",
		"start_time":     "1696502961",
		"uuid":           host1Global.UUID,
		"version":        "5.9.2",
		"watcher":        "3570",
	}, gqrr.Results[0].Columns)
	require.Equal(t, host2Team1.ID, gqrr.Results[1].HostID)
	require.Equal(t, host2Team1.DisplayName(), gqrr.Results[1].Hostname)
	require.NotZero(t, gqrr.Results[1].LastFetched)
	require.Equal(t, map[string]string{
		"build_distro":   "10.14",
		"build_platform": "darwin",
		"config_hash":    "eed0d8296e5f90b790a23814a9db7a127b13498d",
		"config_valid":   "1",
		"extensions":     "active",
		"instance_id":    "7f02ff0f-f8a7-4ba9-a1d2-66836b154f4a",
		"pid":            "95637",
		"platform_mask":  "21",
		"start_time":     "1696611201",
		"uuid":           host2Team1.UUID,
		"version":        "5.9.1",
		"watcher":        "95636",
	}, gqrr.Results[1].Columns)
	require.Equal(t, host1Team2.ID, gqrr.Results[2].HostID)
	require.Equal(t, host1Team2.DisplayName(), gqrr.Results[2].Hostname)
	require.NotZero(t, gqrr.Results[2].LastFetched)
	require.Equal(t, map[string]string{
		"build_distro":   "10.14",
		"build_platform": "darwin",
		"config_hash":    "ca2bc81cd5e79132cb0f842c433ad7f84c056c12",
		"config_valid":   "1",
		"extensions":     "active",
		"instance_id":    "975f2ce1-8672-4932-85f8-340272820e79",
		"pid":            "1039",
		"platform_mask":  "21",
		"start_time":     "1733334052",
		"uuid":           host1Team2.UUID,
		"version":        "5.14.1",
		"watcher":        "1037",
	}, gqrr.Results[2].Columns)

	gqrr = fleet.GetQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report?team_id=%d", osqueryInfoQuery.ID, team2.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.NoError(t, gqrr.Err)
	require.Equal(t, osqueryInfoQuery.ID, gqrr.QueryID)
	require.Len(t, gqrr.Results, 1)

	ghqrr = getHostQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host1Global.ID, osqueryInfoQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Equal(t, osqueryInfoQuery.ID, ghqrr.QueryID)
	require.Equal(t, host1Global.ID, ghqrr.HostID)
	require.NotNil(t, ghqrr.LastFetched)
	require.False(t, ghqrr.ReportClipped)
	require.Len(t, ghqrr.Results, 1)
	require.Equal(t, map[string]string{
		"build_distro":   "centos7",
		"build_platform": "linux",
		"config_hash":    "eed0d8296e5f90b790a23814a9db7a127b13498d",
		"config_valid":   "1",
		"extensions":     "active",
		"instance_id":    "e5799132-85ab-4cfa-89f3-03e0dd3c509a",
		"pid":            "3574",
		"platform_mask":  "9",
		"start_time":     "1696502961",
		"uuid":           host1Global.UUID,
		"version":        "5.9.2",
		"watcher":        "3570",
	}, ghqrr.Results[0].Columns)

	ghqrr = getHostQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host2Global.ID, osqueryInfoQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Equal(t, osqueryInfoQuery.ID, ghqrr.QueryID)
	require.Equal(t, host2Global.ID, ghqrr.HostID)
	require.NotNil(t, ghqrr.LastFetched)
	require.False(t, ghqrr.ReportClipped)
	require.Len(t, ghqrr.Results, 0)

	// verify that certain modifications to queries don't cause result deletion
	modifyQueryResp := fleet.ModifyQueryResponse{}
	updatedDesc := "Updated description"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{ID: osqueryInfoQuery.ID, QueryPayload: fleet.QueryPayload{Description: &updatedDesc}}, http.StatusOK, &modifyQueryResp)
	require.Equal(t, updatedDesc, modifyQueryResp.Query.Description)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 3)

	// now update the query and verify that results are deleted
	updatedQuery := "SELECT * FROM some_new_table;"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{ID: osqueryInfoQuery.ID, QueryPayload: fleet.QueryPayload{Query: &updatedQuery}}, http.StatusOK, &modifyQueryResp)
	require.Equal(t, updatedQuery, modifyQueryResp.Query.Query)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)

	// now update the platform and verify that results are deleted
	s.DoJSON(
		"PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{
			ID: osqueryInfoQuery.ID,
			QueryPayload: fleet.QueryPayload{
				Platform: new("linux"),
			},
		},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "linux", modifyQueryResp.Query.Platform)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	// now update the platform to the same value and verify that results are not deleted
	s.DoJSON(
		"PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{
			ID: osqueryInfoQuery.ID,
			QueryPayload: fleet.QueryPayload{
				Platform: new("linux"),
			},
		},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "linux", modifyQueryResp.Query.Platform)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)

	// now update the min_osquery_version and verify that results are deleted
	s.DoJSON(
		"PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{
			ID: osqueryInfoQuery.ID,
			QueryPayload: fleet.QueryPayload{
				MinOsqueryVersion: new("5.9.1"),
			},
		},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "5.9.1", modifyQueryResp.Query.MinOsqueryVersion)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.Equal(t, 2, counts[usbDevicesQuery.ID])
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	// now update the min_osquery_version to another value and verify that results are deleted
	s.DoJSON(
		"PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{
			ID: osqueryInfoQuery.ID,
			QueryPayload: fleet.QueryPayload{
				MinOsqueryVersion: new("5.11.0"),
			},
		},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "5.11.0", modifyQueryResp.Query.MinOsqueryVersion)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	// now update the min_osquery_version to the same value and verify that results are not deleted
	s.DoJSON(
		"PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{
			ID: osqueryInfoQuery.ID,
			QueryPayload: fleet.QueryPayload{
				MinOsqueryVersion: new("5.11.0"),
			},
		},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "5.11.0", modifyQueryResp.Query.MinOsqueryVersion)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)

	// now update the query via specs and change the min_osquery_version, results should be deleted.
	osqueryInfoQuerySpec := &fleet.QuerySpec{
		Name:               osqueryInfoQuery.Name,
		Description:        osqueryInfoQuery.Description,
		Query:              osqueryInfoQuery.Query,
		Interval:           osqueryInfoQuery.Interval,
		ObserverCanRun:     osqueryInfoQuery.ObserverCanRun,
		Platform:           osqueryInfoQuery.Platform,
		MinOsqueryVersion:  osqueryInfoQuery.MinOsqueryVersion,
		AutomationsEnabled: osqueryInfoQuery.AutomationsEnabled,
		Logging:            osqueryInfoQuery.Logging,
		DiscardData:        osqueryInfoQuery.DiscardData,
	}
	osqueryInfoQuerySpec.MinOsqueryVersion = "5.12.0"
	var applyResp fleet.ApplyQuerySpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{osqueryInfoQuerySpec},
	}, http.StatusOK, &applyResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)
	require.False(t, gqrr.ReportClipped)
	require.Equal(t, 0, counts[osqueryInfoQuery.ID]) // counter reset after min_osquery_version change

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.False(t, gqrr.ReportClipped)
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	// don't change platform or min_osquery_version and results should not be deleted
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{osqueryInfoQuerySpec},
	}, http.StatusOK, &applyResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.False(t, gqrr.ReportClipped)

	// now update the platform and results should be deleted.
	osqueryInfoQuerySpec.Platform = "darwin"
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{osqueryInfoQuerySpec},
	}, http.StatusOK, &applyResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)
	require.False(t, gqrr.ReportClipped)
	require.Equal(t, 0, counts[osqueryInfoQuery.ID]) // counter reset after platform change

	// Update logging type, which should cause results deletion
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", usbDevicesQuery.ID), fleet.ModifyQueryRequest{ID: usbDevicesQuery.ID, QueryPayload: fleet.QueryPayload{Logging: &fleet.LoggingDifferential}}, http.StatusOK, &modifyQueryResp)
	require.Equal(t, fleet.LoggingDifferential, modifyQueryResp.Query.Logging)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", usbDevicesQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)
	require.False(t, gqrr.ReportClipped)
	require.Equal(t, 0, counts[usbDevicesQuery.ID]) // counter reset after logging type change

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.False(t, gqrr.ReportClipped)
	require.Equal(t, 1, counts[osqueryInfoQuery.ID])

	discardData := true
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{ID: osqueryInfoQuery.ID, QueryPayload: fleet.QueryPayload{DiscardData: &discardData}}, http.StatusOK, &modifyQueryResp)
	require.True(t, modifyQueryResp.Query.DiscardData)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)
	require.False(t, gqrr.ReportClipped)
	require.Equal(t, 0, counts[osqueryInfoQuery.ID]) // counter reset after discardData=true

	// check that now that discardData is set, we don't add new results
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)
	require.False(t, gqrr.ReportClipped)

	// Verify row limit behavior with 10% buffer.
	// The system allows up to limit+10% rows before blocking new inserts.
	// This ensures the cleanup cron always has rows to delete, enabling rotation.
	discardData = false
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), fleet.ModifyQueryRequest{ID: osqueryInfoQuery.ID, QueryPayload: fleet.QueryPayload{DiscardData: &discardData}}, http.StatusOK, &modifyQueryResp)
	require.False(t, modifyQueryResp.Query.DiscardData)

	// Host1 submits exactly the max rows
	slreq = submitLogsRequest{
		NodeKey: *host1Global.NodeKey,
		LogType: "result",
		Data: []json.RawMessage{
			json.RawMessage(`{
  "snapshot": [` + results(fleet.DefaultMaxQueryReportRows, host1Global.UUID) + `
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host1Global.Hostname + `"
  }
}`),
		},
	}
	slres = submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	require.Equal(t, fleet.DefaultMaxQueryReportRows, counts[osqueryInfoQuery.ID])

	// Host1 submits same rows again (overwrite), should still have 1000 rows
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, fleet.DefaultMaxQueryReportRows)
	require.True(t, gqrr.ReportClipped)
	require.Equal(t, fleet.DefaultMaxQueryReportRows, counts[osqueryInfoQuery.ID]) // counter unchanged after overwrite

	ghqrr = getHostQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host1Global.ID, osqueryInfoQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Len(t, ghqrr.Results, fleet.DefaultMaxQueryReportRows)
	require.True(t, ghqrr.ReportClipped)

	// Host2 submits 1000 rows. Since we're at the limit but within the 10% buffer,
	// these rows are allowed. Total becomes 2000 rows (1000 from each host).
	slreq = submitLogsRequest{
		NodeKey: *host2Global.NodeKey,
		LogType: "result",
		Data: []json.RawMessage{
			json.RawMessage(`{
  "snapshot": [` + results(fleet.DefaultMaxQueryReportRows, host2Global.UUID) + `
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host2Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host2Global.Hostname + `"
  }
}`),
		},
	}
	slres = submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)

	// Now we have 2000 rows (1000 from host1, 1000 from host2)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, fleet.DefaultMaxQueryReportRows*2)
	require.True(t, gqrr.ReportClipped)
	require.Equal(t, fleet.DefaultMaxQueryReportRows*2, counts[osqueryInfoQuery.ID]) // counter is now 2000

	// Host1 tries to submit 1 row, but counter is now > limit+10%, so it's blocked
	slreq.NodeKey = *host1Global.NodeKey
	slreq.Data = []json.RawMessage{json.RawMessage(`{
  "snapshot": [` + results(1, host1Global.UUID) + `
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host1Global.Hostname + `"
  }
}`)}

	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	// Still 2000 rows since the submission was blocked
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, fleet.DefaultMaxQueryReportRows*2)
	require.True(t, gqrr.ReportClipped)
	require.Equal(t, fleet.DefaultMaxQueryReportRows*2, counts[osqueryInfoQuery.ID]) // counter unchanged (blocked)

	// Increase the limit so we can test further submissions
	appConfigSpec := map[string]map[string]int{
		"server_settings": {"query_report_cap": fleet.DefaultMaxQueryReportRows * 3},
	}
	s.Do("PATCH", "/api/latest/fleet/config", appConfigSpec, http.StatusOK)

	// With limit 3000, we have 2000 rows, which is below the limit, so not clipped
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, fleet.DefaultMaxQueryReportRows*2)
	require.False(t, gqrr.ReportClipped)

	// Now host1 can submit again since we're under the new limit+10%
	slreq.Data = []json.RawMessage{
		json.RawMessage(`{
  "snapshot": [` + results(500, host1Global.UUID) + `
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host1Global.Hostname + `"
  }
}`),
	}

	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)

	// Host1's 1000 rows were replaced with 500 rows, so total is now 1500 (500 from host1, 1000 from host2)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), fleet.GetQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1500)
	require.Equal(t, 1500, counts[osqueryInfoQuery.ID]) // counter unchanged (blocked)
	require.False(t, gqrr.ReportClipped)

	// TODO: Set global discard flag and verify that all data is gone.
}

// Creates a set of results for use in tests for Query Results.
func results(num int, hostID string) string {
	b := strings.Builder{}
	for i := range num {
		b.WriteString(`    {
      "build_distro": "centos7",
      "build_platform": "linux",
      "config_hash": "eed0d8296e5f90b790a23814a9db7a127b13498d",
      "config_valid": "1",
      "extensions": "active",
      "instance_id": "e5799132-85ab-4cfa-89f3-03e0dd3c509a",
      "pid": "3574",
      "platform_mask": "9",
      "start_time": "1696502961",
      "uuid": "` + hostID + `",
      "version": "5.9.2",
      "watcher": "3570"
    }`)
		if i != num-1 {
			b.WriteString(",")
		}
	}

	return b.String()
}

func (s *integrationTestSuite) TestAddingRemovingManualLabels() {
	t := s.T()
	ctx := context.Background()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	newGlobalUserFunc := func(email string, globalRole string) *fleet.User {
		user := &fleet.User{
			Name:       email,
			Email:      email,
			GlobalRole: &globalRole,
		}
		err = user.SetPassword(test.GoodPassword, 10, 10)
		require.NoError(t, err)
		user, err = s.ds.NewUser(context.Background(), user)
		require.NoError(t, err)
		return user
	}
	newTeamUserFunc := func(email string, team *fleet.Team, teamRole string) *fleet.User {
		user := &fleet.User{
			Name:  email,
			Email: email,
			Teams: []fleet.UserTeam{
				{
					Team: *team,
					Role: teamRole,
				},
			},
		}
		err = user.SetPassword(test.GoodPassword, 10, 10)
		require.NoError(t, err)
		user, err = s.ds.NewUser(context.Background(), user)
		require.NoError(t, err)
		return user
	}
	globalObserver := newGlobalUserFunc("global.observer@example.com", fleet.RoleObserver)
	teamAdmin := newTeamUserFunc("team.admin@example.com", team1, fleet.RoleAdmin)
	teamObserver := newTeamUserFunc("team.observer@example.com", team1, fleet.RoleObserver)

	newHostFunc := func(name string, teamID *uint) *fleet.Host {
		host, err := s.ds.NewHost(ctx, &fleet.Host{
			NodeKey:  new(name),
			UUID:     name,
			Hostname: "foo.local." + name,
			TeamID:   teamID,
		})
		require.NoError(t, err)
		require.NotNil(t, host)
		return host
	}
	host1 := newHostFunc("host1", nil)
	host2 := newHostFunc("host2", nil)
	teamHost2 := newHostFunc("teamHost2", &team1.ID)

	ls, err := s.ds.LabelIDsByName(ctx, []string{"All Hosts"}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, ls, 1)
	allHostsLabelID, ok := ls["All Hosts"]
	require.True(t, ok)
	require.NotZero(t, allHostsLabelID)

	dynamicLabel1, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                "dynamicLabel1",
		Query:               "SELECT 1;",
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	manualLabel1, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                "manualLabel1",
		Query:               "SELECT 2;",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)
	manualLabel2, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                "manualLabel2",
		Query:               "SELECT 3;",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)

	err = s.ds.RecordLabelQueryExecutions(context.Background(), host1, map[uint]*bool{allHostsLabelID: new(true)}, time.Now(), false)
	require.NoError(t, err)

	getHostLabels := func(host *fleet.Host) []string {
		var hostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
		var labels []string
		for _, label := range hostResp.Host.Labels {
			labels = append(labels, label.Name)
		}
		return labels
	}

	hostLabels1 := getHostLabels(host1)
	require.Len(t, hostLabels1, 1)
	require.Equal(t, "All Hosts", hostLabels1[0])

	// No labels or empty labels is a no-op.
	var addLabelsToHostResp addLabelsToHostResponse
	s.DoJSON(
		"POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID),
		json.RawMessage(`{}`), http.StatusOK, &addLabelsToHostResp,
	)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{},
	}, http.StatusOK, &addLabelsToHostResp)
	var removeLabelsFromHostResp fleet.RemoveLabelsFromHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{},
	}, http.StatusOK, &removeLabelsFromHostResp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{""},
	}, http.StatusOK, &addLabelsToHostResp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"", ""},
	}, http.StatusOK, &addLabelsToHostResp)

	// A dynamic buitin label should fail to be added.
	res := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"All Hosts"},
	}, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels are dynamic: \"All Hosts\". Dynamic labels can not be assigned to hosts manually.")
	// An inexistent label should fail to be added.
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"manualLabel2", "does not exist"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels not found: \"does not exist\". All labels must exist")
	// Multiple inexistent labels should fail to be added.
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"manualLabel2", "does not exist", "does not exist 2"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels not found: \"does not exist\", \"does not exist 2\". All labels must exist")
	// A dynamic non-builtin label should fail to be added.
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{dynamicLabel1.Name},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels are dynamic: \"dynamicLabel1\". Dynamic labels can not be assigned to hosts manually.")
	// Multiple dynamic labels should fail to be added.
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"All Hosts", dynamicLabel1.Name},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels are dynamic: \"All Hosts\", \"dynamicLabel1\". Dynamic labels can not be assigned to hosts manually.")

	// A dynamic builtin label should fail to be deleted.
	res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{"All Hosts"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't remove labels. Labels are dynamic: \"All Hosts\". Dynamic labels can not be assigned to hosts manually.")
	// An inexistent label should fail to be deleted.
	res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel2.Name, "does not exist"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't remove labels. Labels not found: \"does not exist\". All labels must exist")
	// Multiple inexistent labels should fail to be deleted.
	res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel2.Name, "does not exist", "does not exist 2"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't remove labels. Labels not found: \"does not exist\", \"does not exist 2\". All labels must exist")
	// Multiple dynamic labels should fail to be deleted.
	res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel2.Name, dynamicLabel1.Name, "All Hosts"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't remove labels. Labels are dynamic: \"All Hosts\", \"dynamicLabel1\". Dynamic labels can not be assigned to hosts manually.")

	// Add two manual labels to a host.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	// Add the same manual labels to a host should succeed.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &addLabelsToHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 3)
	require.Equal(t, "All Hosts", hostLabels1[0])
	require.Equal(t, manualLabel1.Name, hostLabels1[1])
	require.Equal(t, manualLabel2.Name, hostLabels1[2])
	hostLabels2 := getHostLabels(host2)
	require.Empty(t, hostLabels2)

	// Remove the two manual labels from the host.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)
	// Remove the same manual labels from the host again.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 1)
	require.Equal(t, "All Hosts", hostLabels1[0])

	// Add same label, should deduplicate.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 2)
	require.Equal(t, "All Hosts", hostLabels1[0])
	require.Equal(t, manualLabel1.Name, hostLabels1[1])

	// Adding an already added label should work.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &addLabelsToHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 3)
	require.Equal(t, "All Hosts", hostLabels1[0])
	require.Equal(t, manualLabel1.Name, hostLabels1[1])
	require.Equal(t, manualLabel2.Name, hostLabels1[2])

	// Delete same label, should deduplicate.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel1.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	// Deleting a non-member label (manualLabel1) should work.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 1)
	require.Equal(t, "All Hosts", hostLabels1[0])

	// Add to non-existent host
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", 999), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusNotFound, &addLabelsToHostResp)
	// Delete from non-existent host
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", 999), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusNotFound, &removeLabelsFromHostResp)

	// Add labels to team host.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)

	// A global observer should not be allowed to add/remove a label.
	oldToken := s.token
	s.token = s.getTestToken(globalObserver.Email, test.GoodPassword)
	t.Cleanup(func() {
		s.token = oldToken
	})
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &removeLabelsFromHostResp)

	// A team observer should not be allowed to add/remove a label.
	s.token = s.getTestToken(teamObserver.Email, test.GoodPassword)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &removeLabelsFromHostResp)

	// A team admin should not be allowed to add/remove a label for a global host.
	s.token = s.getTestToken(teamAdmin.Email, test.GoodPassword)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &removeLabelsFromHostResp)

	// A team admin should be allowed to add/remove a label for a team host.
	s.token = s.getTestToken(teamAdmin.Email, test.GoodPassword)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	teamHost2Labels := getHostLabels(teamHost2)
	require.Len(t, teamHost2Labels, 1)
	require.Equal(t, manualLabel1.Name, teamHost2Labels[0])
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)
	teamHost2Labels = getHostLabels(teamHost2)
	require.Empty(t, teamHost2Labels)
}

func (s *integrationTestSuite) TestDebugDB() {
	t := s.T()
	var response map[string]string
	s.DoJSON("GET", "/debug/db/locks", nil, http.StatusOK, &response)
	assert.Empty(t, response)

	var responseString string
	s.DoJSON("GET", "/debug/db/innodb-status", nil, http.StatusOK, &responseString)
	assert.Contains(t, responseString, "INNODB MONITOR OUTPUT")
}

func (s *integrationTestSuite) TestAutofillPolicies() {
	t := s.T()

	// Ensure AI features are enabled (may have been disabled by a previous test).
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"server_settings": {"ai_features_disabled": false}
	}`), http.StatusOK)

	startMockServer := func(t *testing.T) string {
		// create a test http server
		srv := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "POST" {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}
					switch r.URL.Path {
					case "/ok":
						var body map[string]interface{}
						err := json.NewDecoder(r.Body).Decode(&body)
						if err != nil {
							t.Log(err)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						_, _ = w.Write([]byte(`{"risks":"description", "whatWillProbablyHappenDuringMaintenance":"resolution"}`))
					case "/error":
						w.WriteHeader(http.StatusTeapot)
						_, _ = w.Write([]byte(`{}`))
					case "/badBody":
						_, _ = w.Write([]byte(`{bad json}`))
					case "/timeout":
						time.Sleep(2 * time.Second)
						_, _ = w.Write([]byte(`{"risks":"description", "whatWillProbablyHappenDuringMaintenance":"resolution"}`))
					default:
						w.WriteHeader(http.StatusNotFound)
					}
				},
			),
		)
		t.Cleanup(srv.Close)
		return srv.URL
	}
	mockUrl := startMockServer(t)
	originalUrl := getHumanInterpretationFromOsquerySqlUrl
	originalTimeout := getHumanInterpretationFromOsquerySqlTimeout
	t.Cleanup(
		func() {
			getHumanInterpretationFromOsquerySqlUrl = originalUrl
			getHumanInterpretationFromOsquerySqlTimeout = originalTimeout
		},
	)

	req := fleet.AutofillPoliciesRequest{
		SQL: "  ", // empty
	}
	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/ok"
	// empty sql
	resp := s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusBadRequest)
	assertBodyContains(t, resp, "cannot be empty")

	// good request
	req.SQL = "select 1"
	var res fleet.AutofillPoliciesResponse
	s.DoJSON("POST", "/api/latest/fleet/autofill/policy", req, http.StatusOK, &res)
	assert.Equal(t, "description", res.Description)
	assert.Equal(t, "resolution", res.Resolution)

	// good request with weird characters
	req.SQL = `select * from " with ' and "" \"`
	res = fleet.AutofillPoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/autofill/policy", req, http.StatusOK, &res)
	assert.Equal(t, "description", res.Description)
	assert.Equal(t, "resolution", res.Resolution)

	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/error"
	resp = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, resp, "error from human interpretation of osquery sql")

	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/badBody"
	resp = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, resp, "error unmarshaling response body from human interpretation of osquery sql")

	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/timeout"
	getHumanInterpretationFromOsquerySqlTimeout = 1 * time.Millisecond
	resp = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, resp, "error sending request to get human interpretation from osquery sql")

	// disable AI features
	appConfigSpec := map[string]map[string]bool{
		"server_settings": {"ai_features_disabled": true},
	}
	s.Do("PATCH", "/api/latest/fleet/config", appConfigSpec, http.StatusOK)
	resp = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusBadRequest)
	assertBodyContains(t, resp, "AI features are disabled")
}

func (s *integrationTestSuite) TestHostWithNoPoliciesClearsPolicyCounts() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "Zoobar"})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("foobar"),
		UUID:            "foobar",
		Hostname:        "com.foobar.local",
		Platform:        "linux",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:  "Barfoo",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)

	distributedWriteResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy.ID: new(false),
		},
	), http.StatusOK, &distributedWriteResp)

	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)
	require.Equal(t, uint64(1), listHostsResp.Hosts[0].FailingPoliciesCount)

	_, err = s.ds.DeleteTeamPolicies(ctx, team.ID, []uint{policy.ID})
	require.NoError(t, err)

	distributedWriteResp = submitDistributedQueryResultsResponse{}
	results := make(map[string]json.RawMessage)
	results[hostNoPoliciesWildcard] = json.RawMessage("{\"1\": \"1\"}")
	statuses := make(map[string]interface{})
	statuses[hostNoPoliciesWildcard] = 0
	s.DoJSON("POST", "/api/osquery/distributed/write", submitDistributedQueryResultsRequestShim{
		NodeKey:  *host.NodeKey,
		Results:  results,
		Statuses: statuses,
		Stats:    map[string]*fleet.Stats{},
	}, http.StatusOK, &distributedWriteResp)

	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)
	require.Equal(t, uint64(0), listHostsResp.Hosts[0].FailingPoliciesCount)
}

func (s *integrationTestSuite) TestSecretVariablesGitOps() {
	t := s.T()
	ctx := context.Background()

	u := &fleet.User{
		Name:  "GitOps",
		Email: "gitops1@example.com",
		// GitOps role is premium only, so we use the global admin role.
		GlobalRole: new(fleet.RoleAdmin),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(ctx, u)
	require.NoError(t, err)
	s.setTokenForTest(t, "gitops1@example.com", test.GoodPassword)

	// Empty request
	req := fleet.CreateSecretVariablesRequest{}
	var resp fleet.CreateSecretVariablesResponse
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &resp)

	// Secret variable name too long
	req = fleet.CreateSecretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  strings.Repeat("a", 256),
				Value: "value",
			},
		},
	}
	httpResp := s.Do("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, httpResp, "secret variable name is too long")

	// Secret variable name empty
	req = fleet.CreateSecretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "  ",
				Value: "value",
			},
		},
	}
	httpResp = s.Do("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, httpResp, "secret variable name cannot be empty")

	validName := strings.Repeat("G", 255)
	req = fleet.CreateSecretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "FLEET_SECRET_" + validName,
				Value: "value",
			},
		},
	}
	// Do dry run
	req.DryRun = true
	idBeforeDryRun := s.lastActivityMatches("", "", 0)
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &resp)

	secrets, err := s.ds.GetSecretVariables(ctx, []string{validName})
	require.NoError(t, err)
	require.Empty(t, secrets)
	// A dry run persists nothing, so it must not emit any activity.
	require.Equal(t, idBeforeDryRun, s.lastActivityMatches("", "", 0))

	// Do real run: creating the variable emits a created_custom_variable activity.
	req.DryRun = false
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &resp)
	secrets, err = s.ds.GetSecretVariables(ctx, []string{validName})
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, "value", secrets[0].Value)
	s.lastActivityMatches(
		fleet.ActivityCreatedCustomVariable{}.ActivityName(),
		fmt.Sprintf(`{"custom_variable_id":0,"custom_variable_name":%q}`, validName),
		0,
	)

	// Re-applying the same spec is a no-op and must not emit any activity.
	idAfterCreate := s.lastActivityMatches("", "", 0)
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &resp)
	require.Equal(t, idAfterCreate, s.lastActivityMatches("", "", 0))

	// Changing the value via the spec endpoint emits an updated_custom_variable activity.
	req.SecretVariables[0].Value = "new-value"
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &resp)
	secrets, err = s.ds.GetSecretVariables(ctx, []string{validName})
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, "new-value", secrets[0].Value)
	s.lastActivityMatches(
		fleet.ActivityUpdatedCustomVariable{}.ActivityName(),
		fmt.Sprintf(`{"custom_variable_name":%q}`, validName),
		0,
	)
}

func (s *integrationTestSuite) TestSecretVariables() {
	t := s.T()
	ctx := context.Background()

	// Create a single secret variable.
	var csvr fleet.CreateSecretVariableResponse
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "NAME1",
		Value: "value1",
	}, http.StatusOK, &csvr)
	firstVariableID := csvr.ID
	require.NotZero(t, firstVariableID)

	// List (no-filtering).
	var lsvr fleet.ListSecretVariablesResponse
	s.DoJSON("GET", "/api/latest/fleet/custom_variables", fleet.ListSecretVariablesRequest{}, http.StatusOK, &lsvr)
	require.Equal(t, lsvr.Count, 1)
	require.Len(t, lsvr.CustomVariables, 1)
	require.NotZero(t, lsvr.CustomVariables[0].ID)
	require.Equal(t, "NAME1", lsvr.CustomVariables[0].Name)
	require.NotZero(t, lsvr.CustomVariables[0].UpdatedAt)

	// Make sure we can access the value internally.
	secretVariables, err := s.ds.GetSecretVariables(ctx, []string{"NAME1"})
	require.NoError(t, err)
	require.Len(t, secretVariables, 1)
	require.Equal(t, "NAME1", secretVariables[0].Name)
	require.Equal(t, "value1", secretVariables[0].Value)
	require.NotZero(t, secretVariables[0].UpdatedAt)

	// Creating the same variable should fail with conflict.
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "NAME1",
		Value: "value1",
	}, http.StatusConflict, &csvr)

	// Creating a variable with invalid name should fail with 422.
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "lowercase",
		Value: "foobar",
	}, http.StatusUnprocessableEntity, &csvr)
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "",
		Value: "foobar",
	}, http.StatusUnprocessableEntity, &csvr)
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  strings.Repeat("HA ", 255/3+1),
		Value: "foobar",
	}, http.StatusUnprocessableEntity, &csvr)
	// No server private key configured, should fail with 400.
	testSetEmptyPrivateKey = true
	defer func() {
		testSetEmptyPrivateKey = false
	}()
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "NAME2",
		Value: "foobar",
	}, http.StatusBadRequest, &csvr)

	testSetEmptyPrivateKey = false

	// Creating a variable with empty value should fail with 422.
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "ANOTHER_NAME",
		Value: "",
	}, http.StatusUnprocessableEntity, &csvr)

	// Creating a second variable.
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "ANOTHER_NAME",
		Value: "value2",
	}, http.StatusOK, &csvr)
	secondVariableID := csvr.ID
	require.NotZero(t, secondVariableID)

	// List (no-filtering) with pagination (first page).
	lsvr = fleet.ListSecretVariablesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/custom_variables", nil, http.StatusOK, &lsvr, "per_page", "1", "page", "0")
	require.Equal(t, 2, lsvr.Count)
	require.NotNil(t, lsvr.Meta)
	require.False(t, lsvr.Meta.HasPreviousResults)
	require.True(t, lsvr.Meta.HasNextResults)
	require.Len(t, lsvr.CustomVariables, 1)
	require.Equal(t, secondVariableID, lsvr.CustomVariables[0].ID)
	require.Equal(t, "ANOTHER_NAME", lsvr.CustomVariables[0].Name)
	require.NotZero(t, lsvr.CustomVariables[0].UpdatedAt)
	// List (no-filtering) with pagination (second page).
	lsvr = fleet.ListSecretVariablesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/custom_variables", nil, http.StatusOK, &lsvr, "per_page", "1", "page", "1")
	require.Equal(t, 2, lsvr.Count)
	require.NotNil(t, lsvr.Meta)
	require.True(t, lsvr.Meta.HasPreviousResults)
	require.False(t, lsvr.Meta.HasNextResults)
	require.Len(t, lsvr.CustomVariables, 1)
	require.Equal(t, firstVariableID, lsvr.CustomVariables[0].ID)
	require.Equal(t, "NAME1", lsvr.CustomVariables[0].Name)
	require.NotZero(t, lsvr.CustomVariables[0].UpdatedAt)
	// List (no-filtering) with pagination (one page, two secrets).
	// Must be ordered alphabetically.
	lsvr = fleet.ListSecretVariablesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/custom_variables", nil, http.StatusOK, &lsvr, "per_page", "20", "page", "0")
	require.Equal(t, 2, lsvr.Count)
	require.NotNil(t, lsvr.Meta)
	require.False(t, lsvr.Meta.HasPreviousResults)
	require.False(t, lsvr.Meta.HasNextResults)
	require.Len(t, lsvr.CustomVariables, 2)
	require.Equal(t, secondVariableID, lsvr.CustomVariables[0].ID)
	require.Equal(t, "ANOTHER_NAME", lsvr.CustomVariables[0].Name)
	require.NotZero(t, lsvr.CustomVariables[0].UpdatedAt)
	require.Equal(t, firstVariableID, lsvr.CustomVariables[1].ID)
	require.Equal(t, "NAME1", lsvr.CustomVariables[1].Name)
	require.NotZero(t, lsvr.CustomVariables[1].UpdatedAt)

	// Test deletion of non-existent ID
	var dsvr fleet.DeleteSecretVariableResponse
	s.DoJSON("DELETE", "/api/latest/fleet/custom_variables/999", nil, http.StatusNotFound, &dsvr)

	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", firstVariableID), nil, http.StatusOK, &dsvr)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", secondVariableID), nil, http.StatusOK, &dsvr)

	// List after deletions should be empty.
	lsvr = fleet.ListSecretVariablesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/custom_variables", nil, http.StatusOK, &lsvr)
	require.Equal(t, 0, lsvr.Count)
	require.Empty(t, lsvr.CustomVariables)
}

func (s *integrationTestSuite) TestSecretVariablesInUse() {
	t := s.T()
	ctx := context.Background()

	foobarTeam, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Foobar",
	})
	require.NoError(t, err)

	// Create a single secret variable.
	var csvr fleet.CreateSecretVariableResponse
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "NAME1",
		Value: "value1",
	}, http.StatusOK, &csvr)
	firstVariableID := csvr.ID
	require.NotZero(t, firstVariableID)

	// Create Apple configuration profile in "No team" that uses the variable.
	appleProfile, err := s.ds.NewMDMAppleConfigProfile(
		ctx,
		fleet.MDMAppleConfigProfile{
			Name:         "Name0",
			Identifier:   "Identifier0",
			Mobileconfig: []byte("$FLEET_SECRET_NAME1"),
		},
		nil,
	)
	require.NoError(t, err)

	res := s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", firstVariableID), nil, http.StatusConflict)
	errorMsg := extractServerErrorText(res.Body)
	require.Contains(
		t,
		errorMsg,
		"Couldn't delete. NAME1 is used by the \"Name0\" configuration profile in the \"No team\" team. Please delete the configuration profile and try again.",
	)

	err = s.ds.DeleteMDMAppleConfigProfile(ctx, appleProfile.ProfileUUID)
	require.NoError(t, err)

	// Create Apple declaration in "Foobar" team that uses the variable.
	appleDeclaration, err := s.ds.NewMDMAppleDeclaration(ctx, &fleet.MDMAppleDeclaration{
		Identifier: "decl-1",
		Name:       "decl-1",
		RawJSON:    json.RawMessage(`{"Identifier": "${FLEET_SECRET_NAME1}"}`),
		TeamID:     &foobarTeam.ID,
	}, nil)
	require.NoError(t, err)

	res = s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", firstVariableID), nil, http.StatusConflict)
	errorMsg = extractServerErrorText(res.Body)
	require.Contains(
		t,
		errorMsg,
		"Couldn't delete. NAME1 is used by the \"decl-1\" configuration profile in the \"Foobar\" team. Please delete the configuration profile and try again.",
	)

	err = s.ds.DeleteMDMAppleDeclaration(ctx, appleDeclaration.DeclarationUUID)
	require.NoError(t, err)

	// Create Windows profile in "Foobar" team that uses the variable.
	windowsProfile, err := s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "zoo",
		TeamID: &foobarTeam.ID,
		SyncML: []byte("<Replace>$FLEET_SECRET_NAME1</Replace>"),
	}, nil)
	require.NoError(t, err)

	res = s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", firstVariableID), nil, http.StatusConflict)
	errorMsg = extractServerErrorText(res.Body)
	require.Contains(
		t,
		errorMsg,
		"Couldn't delete. NAME1 is used by the \"zoo\" configuration profile in the \"Foobar\" team. Please delete the configuration profile and try again.",
	)

	err = s.ds.DeleteMDMWindowsConfigProfile(ctx, windowsProfile.ProfileUUID)
	require.NoError(t, err)

	// Create a script in "No team" that uses a variable
	script, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "foobar.sh",
		ScriptContents: "echo $FLEET_SECRET_NAME1",
	})
	require.NoError(t, err)

	res = s.DoRaw("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", firstVariableID), nil, http.StatusConflict)
	errorMsg = extractServerErrorText(res.Body)
	require.Contains(
		t,
		errorMsg,
		"Couldn't delete. NAME1 is used by the \"foobar.sh\" script in the \"No team\" team. Please edit or delete the script and try again.",
	)

	err = s.ds.DeleteScript(ctx, script.ID)
	require.NoError(t, err)

	// Finally, delete now should work.
	var dsvr fleet.DeleteSecretVariableResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/custom_variables/%d", firstVariableID), nil, http.StatusOK, &dsvr)
}

func (s *integrationTestSuite) TestSecretVariablesPermissions() {
	t := s.T()
	ctx := context.Background()

	// Create a single secret variable.
	var csvr fleet.CreateSecretVariableResponse
	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "NAME1",
		Value: "foobar",
	}, http.StatusOK, &csvr)

	// Create a global observer user which should be allowed to read but not create secret variables.
	u := &fleet.User{
		Name:       "Observer",
		Email:      "observer@example.com",
		GlobalRole: new(fleet.RoleObserver),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(ctx, u)
	require.NoError(t, err)
	s.setTokenForTest(t, "observer@example.com", test.GoodPassword)

	s.DoJSON("POST", "/api/latest/fleet/custom_variables", fleet.CreateSecretVariableRequest{
		Name:  "NAME1",
		Value: "foobar",
	}, http.StatusForbidden, &csvr)

	// List (no-filtering) should work for non-admins.
	var lsvr fleet.ListSecretVariablesResponse
	s.DoJSON("GET", "/api/latest/fleet/custom_variables", fleet.ListSecretVariablesRequest{}, http.StatusOK, &lsvr)
	require.Equal(t, lsvr.Count, 1)
	require.Len(t, lsvr.CustomVariables, 1)
	require.NotZero(t, lsvr.CustomVariables[0].ID)
	require.Equal(t, "NAME1", lsvr.CustomVariables[0].Name)
	require.NotZero(t, lsvr.CustomVariables[0].UpdatedAt)
}

func (s *integrationTestSuite) TestAndroidHostUUIDPropagation() {
	t := s.T()
	ctx := context.Background()

	// Create an Android host with a specific UUID
	const expectedUUID = "TEST-UUID-12345-ANDROID"
	host := &fleet.AndroidHost{
		Host: &fleet.Host{
			Hostname:       "test-android-uuid",
			ComputerName:   "AndroidTestDevice",
			Platform:       "android",
			OSVersion:      "Android 15",
			Build:          "test-build-uuid",
			Memory:         2048,
			TeamID:         nil,
			HardwareSerial: "test-serial-uuid",
			HardwareModel:  "Pixel 8",
			HardwareVendor: "Google",
			UUID:           expectedUUID, // Set the UUID explicitly
		},
		Device: &android.Device{
			DeviceID:             strings.ReplaceAll(uuid.NewString(), "-", ""),
			EnterpriseSpecificID: new(expectedUUID),
			AppliedPolicyID:      new("1"),
			LastPolicySyncTime:   new(time.Now()),
		},
	}
	host.SetNodeKey(expectedUUID)

	// Create Android host
	androidHost, err := s.ds.NewAndroidHost(ctx, host, false)
	require.NoError(t, err)
	require.NotZero(t, androidHost.Host.ID)

	// Test 1: Get the host, verify UUID is present
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", androidHost.Host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host)
	require.Equal(t, expectedUUID, getHostResp.Host.UUID, "UUID should be returned in API response")
	require.Equal(t, "AndroidTestDevice", getHostResp.Host.ComputerName)

	// Test 2: Update the host, verify UUID is preserved
	updatedHost := androidHost
	updatedHost.Host.Hostname = "updated-android-hostname"
	updatedHost.Host.ComputerName = "UpdatedAndroidDevice"
	updatedHost.Host.OSVersion = "Android 16"
	updatedHost.Host.UUID = expectedUUID

	err = s.ds.UpdateAndroidHost(ctx, updatedHost, false, false)
	require.NoError(t, err)

	// Get the host again, verify UUID is still present
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", androidHost.Host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host)
	require.Equal(t, expectedUUID, getHostResp.Host.UUID, "UUID should be preserved after host update")
	require.Equal(t, "UpdatedAndroidDevice", getHostResp.Host.ComputerName)
	require.Equal(t, "Android 16", getHostResp.Host.OSVersion)

	// Test 3: List hosts, verify Android host UUID is included
	var listHostsResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsResp)

	// Find our Android host in the list
	var foundHost *fleet.HostResponse
	for _, h := range listHostsResp.Hosts {
		if h.ID == androidHost.Host.ID {
			foundHost = &h
			break
		}
	}
	require.NotNil(t, foundHost, "Android host should be in list response")
	require.Equal(t, expectedUUID, foundHost.UUID, "UUID should be present in list hosts response")

	// Test 4: AndroidHostLite returns UUID
	androidHostLite, err := s.ds.AndroidHostLite(ctx, expectedUUID)
	require.NoError(t, err)
	require.Equal(t, expectedUUID, androidHostLite.Host.UUID, "AndroidHostLite should return UUID")
}

func (s *integrationTestSuite) TestListAndroidHostsInLabel() {
	t := s.T()
	ctx := context.Background()

	hostIDs := createAndroidHosts(t, s.ds, 3, nil)
	notAndroidHost := createOrbitEnrolledHost(t, "darwin", "-4", s.ds)

	// list labels, has the built-in ones, capture All and Android
	var listResp fleet.ListLabelsResponse
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp)
	var allLblID, androidLblID uint
	for _, lbl := range listResp.Labels {
		switch lbl.Name {
		case fleet.BuiltinLabelNameAllHosts:
			allLblID = lbl.ID
		case fleet.BuiltinLabelNameAndroid:
			androidLblID = lbl.ID
		}
	}
	require.NotZero(t, allLblID)
	require.NotZero(t, androidLblID)

	err := s.ds.AddLabelsToHost(ctx, notAndroidHost.ID, []uint{allLblID})
	require.NoError(t, err)

	pluckHostIDs := func(hosts []fleet.HostResponse) []uint {
		ids := make([]uint, 0, len(hosts))
		for _, h := range hosts {
			ids = append(ids, h.ID)
		}
		return ids
	}

	// list hosts in all hosts
	var listHostsResp listHostsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", allLblID), nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, len(hostIDs)+1)
	wantIDs := append([]uint{notAndroidHost.ID}, hostIDs...)
	require.ElementsMatch(t, wantIDs, pluckHostIDs(listHostsResp.Hosts))

	// count hosts in label
	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "label_id", fmt.Sprint(allLblID))
	require.Equal(t, len(hostIDs)+1, countResp.Count)

	// list android hosts
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", androidLblID), nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, len(hostIDs))
	require.ElementsMatch(t, hostIDs, pluckHostIDs(listHostsResp.Hosts))

	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "label_id", fmt.Sprint(androidLblID))
	require.Equal(t, len(hostIDs), countResp.Count)
}

func (s *integrationTestSuite) TestAndroidHostStorageInAPI() {
	t := s.T()
	ctx := context.Background()

	// Android host with storage data
	hostID := createAndroidHostWithStorage(t, s.ds, nil)

	// individual host endpoint
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)

	require.NotNil(t, hostResp.Host)
	require.Equal(t, "android", hostResp.Host.Platform)

	// storage data is present in API response
	assert.Equal(t, 128.0, hostResp.Host.GigsTotalDiskSpace, "API should return total disk space")
	assert.Equal(t, 35.0, hostResp.Host.GigsDiskSpaceAvailable, "API should return available disk space")
	assert.InDelta(t, 27.34, hostResp.Host.PercentDiskSpaceAvailable, 0.1, "API should return disk space percentage")

	// list endpoint includes storage data
	var listResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp)

	var androidHost *fleet.HostResponse
	for _, host := range listResp.Hosts {
		if host.ID == hostID {
			androidHost = &host
			break
		}
	}

	require.NotNil(t, androidHost, "Android host should be in hosts list")
	require.Equal(t, "android", androidHost.Platform)

	// storage data in list endpoint
	assert.Equal(t, 128.0, androidHost.GigsTotalDiskSpace, "Host list should include total disk space")
	assert.Equal(t, 35.0, androidHost.GigsDiskSpaceAvailable, "Host list should include available disk space")
	assert.InDelta(t, 27.34, androidHost.PercentDiskSpaceAvailable, 0.1, "Host list should include disk space percentage")

	// clean up
	err := s.ds.DeleteHost(ctx, hostID)
	require.NoError(t, err)
}

func createAndroidHosts(t *testing.T, ds *mysql.Datastore, count int, teamID *uint) []uint {
	ids := make([]uint, 0, count)
	for i := range count {
		host := &fleet.AndroidHost{
			Host: &fleet.Host{
				Hostname:       fmt.Sprintf("hostname%d", i),
				ComputerName:   fmt.Sprintf("computer_name%d", i),
				Platform:       "android",
				OSVersion:      "Android 14",
				Build:          fmt.Sprintf("build%d", i),
				Memory:         1024,
				TeamID:         teamID,
				HardwareSerial: uuid.NewString(),
			},
			Device: &android.Device{
				DeviceID:             strings.ReplaceAll(uuid.NewString(), "-", ""), // Remove dashes to fit in VARCHAR(37)
				EnterpriseSpecificID: new(uuid.NewString()),
				AppliedPolicyID:      new("1"),
				LastPolicySyncTime:   new(time.Now().Add(-time.Hour)), // 1 hour ago
			},
		}
		host.SetNodeKey(*host.Device.EnterpriseSpecificID)
		ahost, err := ds.NewAndroidHost(context.Background(), host, false)
		require.NoError(t, err)
		ids = append(ids, ahost.Host.ID)
	}
	return ids
}

func createAndroidHostWithStorage(t *testing.T, ds *mysql.Datastore, teamID *uint) uint {
	return createAndroidHostForTest(t, ds, teamID, false)
}

// createAndroidHostForTest creates an android host. companyOwned=false stores it as BYO
// (is_personal_enrollment=true); companyOwned=true stores it as COBO.
func createAndroidHostForTest(t *testing.T, ds *mysql.Datastore, teamID *uint, companyOwned bool) uint {
	host := &fleet.AndroidHost{
		Host: &fleet.Host{
			Hostname:                  "android-storage-host",
			ComputerName:              "Android Storage Test Device",
			Platform:                  "android",
			OSVersion:                 "Android 14",
			Build:                     "UPB4.230623.005",
			Memory:                    8192, // 8GB RAM
			TeamID:                    teamID,
			HardwareSerial:            "STORAGE-TEST-" + uuid.NewString(),
			GigsTotalDiskSpace:        128.0, // 64GB system + 64GB external
			GigsDiskSpaceAvailable:    35.0,  // 10GB + 25GB available
			PercentDiskSpaceAvailable: 27.34, // 35/128 * 100
			UUID:                      uuid.NewString(),
		},
		Device: &android.Device{
			DeviceID:             strings.ReplaceAll(uuid.NewString(), "-", ""),
			EnterpriseSpecificID: new(uuid.NewString()),
			AppliedPolicyID:      new("1"),
			LastPolicySyncTime:   new(time.Now().Add(-time.Hour)),
		},
	}
	host.SetNodeKey(*host.Device.EnterpriseSpecificID)
	ahost, err := ds.NewAndroidHost(context.Background(), host, companyOwned)
	require.NoError(t, err)
	return ahost.Host.ID
}

func (s *integrationTestSuite) TestHostCertificates() {
	t := s.T()
	ctx := context.Background()

	token := "good_token"
	host := createOrbitEnrolledHost(t, "linux", "host1", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, token)

	// no certificate at the moment
	var certResp listHostCertificatesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/certificates", host.ID), nil, http.StatusOK, &certResp)
	require.Empty(t, certResp.Certificates)

	certResp = listHostCertificatesResponse{}
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/certificates", nil, http.StatusOK)
	err := json.NewDecoder(res.Body).Decode(&certResp)
	require.NoError(t, err)
	require.Empty(t, certResp.Certificates)

	// create some certs for that host
	certNames := []string{"a", "b", "c", "d", "e"}
	now := time.Now()
	// sorting by not_valid_after should get us "d", "c", "e", "a", "b"
	notValidAfterTimes := []time.Time{
		now.Add(time.Minute), now.Add(time.Hour),
		now.Add(time.Second), now.Add(time.Millisecond),
		now.Add(2 * time.Second),
	}
	certs := make([]*fleet.HostCertificateRecord, 0, len(certNames))
	for i, name := range certNames {
		sha1Sum := sha1.Sum([]byte(name)) // nolint:gosec
		certs = append(certs, &fleet.HostCertificateRecord{
			HostID:         host.ID,
			CommonName:     name,
			SHA1Sum:        sha1Sum[:],
			SubjectCountry: "s" + name,
			IssuerCountry:  "i" + name,
			NotValidBefore: now.Add(-24 * time.Hour), // 1 day ago
			NotValidAfter:  notValidAfterTimes[i],
			Source:         fleet.SystemHostCertificate,
		})
	}
	require.NoError(t, s.ds.UpdateHostCertificates(ctx, host.ID, host.UUID, certs, fleet.HostCertificateOriginOsquery, nil))

	// list all certs
	certResp = listHostCertificatesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/certificates", host.ID), nil, http.StatusOK, &certResp)
	require.Len(t, certResp.Certificates, len(certNames))
	for i, cert := range certResp.Certificates {
		want := certNames[i]
		require.Equal(t, want, cert.CommonName)
		require.NotNil(t, cert.Subject)
		require.Equal(t, "s"+want, cert.Subject.Country)
		require.NotNil(t, cert.Issuer)
		require.Equal(t, "i"+want, cert.Issuer.Country)
		require.Equal(t, fleet.SystemHostCertificate, cert.Source)
	}

	certResp = listHostCertificatesResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/certificates", nil, http.StatusOK)
	err = json.NewDecoder(res.Body).Decode(&certResp)
	require.NoError(t, err)
	require.Len(t, certResp.Certificates, len(certNames))
	for i, cert := range certResp.Certificates {
		want := certNames[i]
		require.Equal(t, want, cert.CommonName)
		require.NotNil(t, cert.Subject)
		require.Equal(t, "s"+want, cert.Subject.Country)
		require.NotNil(t, cert.Issuer)
		require.Equal(t, "i"+want, cert.Issuer.Country)
		require.Equal(t, fleet.SystemHostCertificate, cert.Source)
	}

	// non-existing host
	certResp = listHostCertificatesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/certificates", host.ID+1000), nil, http.StatusNotFound, &certResp)
	// for the device endpoint, the token is the authentication so if it doesn't
	// exist, the endpoint is unauthorized.
	certResp = listHostCertificatesResponse{}
	s.DoRawNoAuth("GET", "/api/latest/fleet/device/NO-SUCH-TOKEN/certificates", nil, http.StatusUnauthorized)

	pluckCertNames := func(certs []*fleet.HostCertificatePayload) []string {
		names := make([]string, 0, len(certs))
		for _, cert := range certs {
			names = append(names, cert.CommonName)
		}
		return names
	}

	// fails if order_key  is invalid
	certResp = listHostCertificatesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/certificates", host.ID), nil, http.StatusBadRequest, &certResp, "order_key", "no-such-column")

	certResp = listHostCertificatesResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/certificates", nil, http.StatusBadRequest, "order_key", "no-such-column")
	require.Contains(t, extractServerErrorText(res.Body), "invalid order key")

	// test the pagination options
	cases := []struct {
		queryParams []string
		wantNames   []string
		wantMeta    fleet.PaginationMetadata
	}{
		{queryParams: []string{"page", "0", "per_page", "2"}, wantNames: []string{"a", "b"}, wantMeta: fleet.PaginationMetadata{HasNextResults: true}},
		{queryParams: []string{"page", "1", "per_page", "2"}, wantNames: []string{"c", "d"}, wantMeta: fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true}},
		{queryParams: []string{"page", "2", "per_page", "2"}, wantNames: []string{"e"}, wantMeta: fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true}},
		{queryParams: []string{"page", "3", "per_page", "2"}, wantNames: []string{}, wantMeta: fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true}},
		{queryParams: []string{"page", "0", "per_page", "4", "order_direction", "desc"}, wantNames: []string{"e", "d", "c", "b"}, wantMeta: fleet.PaginationMetadata{HasNextResults: true}},
		{queryParams: []string{"page", "1", "per_page", "4", "order_direction", "desc"}, wantNames: []string{"a"}, wantMeta: fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true}},
		{queryParams: []string{"page", "0", "per_page", "3", "order_key", "not_valid_after"}, wantNames: []string{"d", "c", "e"}, wantMeta: fleet.PaginationMetadata{HasNextResults: true}},
		{queryParams: []string{"page", "1", "per_page", "3", "order_key", "not_valid_after"}, wantNames: []string{"a", "b"}, wantMeta: fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true}},
	}
	for _, c := range cases {
		t.Run(strings.Join(c.queryParams, "_"), func(t *testing.T) {
			certResp = listHostCertificatesResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/certificates", host.ID), nil, http.StatusOK, &certResp, c.queryParams...)
			require.Len(t, certResp.Certificates, len(c.wantNames))
			require.Equal(t, c.wantNames, pluckCertNames(certResp.Certificates))
			require.Equal(t, c.wantMeta, *certResp.Meta)

			certResp = listHostCertificatesResponse{}
			res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/certificates", nil, http.StatusOK, c.queryParams...)
			err = json.NewDecoder(res.Body).Decode(&certResp)
			require.NoError(t, err)
			require.Len(t, certResp.Certificates, len(c.wantNames))
			require.Equal(t, c.wantNames, pluckCertNames(certResp.Certificates))
			require.Equal(t, c.wantMeta, *certResp.Meta)
		})
	}
}

func (s *integrationTestSuite) TestConditionalAccessRequiresPremium() {
	// Microsoft compliance partner APIs should fail on Fleet Free (this suite
	// runs without a premium license).
	var r conditionalAccessMicrosoftCreateResponse
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftCreateRequest{
		MicrosoftTenantID: "foobar",
	}, http.StatusPaymentRequired, &r)
	var c conditionalAccessMicrosoftConfirmResponse
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft/confirm", conditionalAccessMicrosoftConfirmRequest{},
		http.StatusPaymentRequired, &c)
	var d conditionalAccessMicrosoftDeleteResponse
	s.DoJSON("DELETE", "/api/latest/fleet/conditional-access/microsoft", nil,
		http.StatusPaymentRequired, &d)
}

func (s *integrationTestSuite) TestUpdateHostCertificateTemplate() {
	t := s.T()
	ctx := context.Background()

	// Create a test team
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "Test Team"})
	require.NoError(t, err)
	teamID := team.ID

	// Create a test certificate authority
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      new("TestUpdateHostCertificateTemplate SCEP CA"),
		URL:       new("http://localhost:8080/scep"),
		Challenge: new("test-challenge"),
	})
	require.NoError(t, err)
	caID := ca.ID

	certTemplate := &fleet.CertificateTemplate{
		Name:                   "TestUpdateHostCertificateTemplate-Cert",
		TeamID:                 teamID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Subject 1",
	}
	savedTemplate, err := s.ds.CreateCertificateTemplate(ctx, certTemplate)
	require.NoError(t, err)
	require.NotNil(t, savedTemplate)

	orbitNodeKey := uuid.New().String()
	uuid := uuid.New().String()
	hostName := "test-update-host-certificate-template"

	// Create a host
	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		NodeKey:  &orbitNodeKey,
		UUID:     uuid,
		Hostname: hostName,
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	host.OrbitNodeKey = &orbitNodeKey
	require.NoError(t, s.ds.UpdateHost(ctx, host))

	certificateTemplateID := savedTemplate.ID

	// Delete the certificate after the test is done, so the team can be deleted.
	defer func() {
		// Clean up
		err = s.ds.DeleteCertificateTemplate(ctx, certificateTemplateID)
		require.NoError(t, err)
	}()

	// Create a record in host_certificate_templates using ad hoc SQL
	sql := `
INSERT INTO host_certificate_templates (
	host_uuid,
	certificate_template_id,
	status,
	fleet_challenge,
	operation_type,
	name
) VALUES (?, ?, ?, ?, ?, ?);
	`
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err = q.ExecContext(ctx, sql, host.UUID, certificateTemplateID, "pending", "some_challenge_value", "install", savedTemplate.Name)
		require.NoError(t, err)
		return nil
	})

	// Enable Android MDM and verify GetHost returns operation_type for certificate templates
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	origAndroidEnabled := appCfg.MDM.AndroidEnabledAndConfigured
	appCfg.MDM.AndroidEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	err = s.ds.SetAndroidEnabledAndConfigured(ctx, true)
	require.NoError(t, err)
	defer func() {
		appCfg.MDM.AndroidEnabledAndConfigured = origAndroidEnabled
		_ = s.ds.SaveAppConfig(ctx, appCfg)
		_ = s.ds.SetAndroidEnabledAndConfigured(ctx, origAndroidEnabled)
	}()

	// Verify GetHost returns operation_type for certificate templates
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 1)
	profile := (*getHostResp.Host.MDM.Profiles)[0]
	require.Equal(t, savedTemplate.Name, profile.Name)
	require.Equal(t, fleet.AndroidCertificateTemplateProfileID, profile.ProfileUUID)
	require.Equal(t, fleet.MDMOperationTypeInstall, profile.OperationType, "operation_type should be populated for certificate templates")

	// Test cases
	cases := []struct {
		name                    string
		templateID              uint
		newStatus               string
		newOperationType        *string
		detail                  *string
		expectedResponseStatus  int
		expectedResponseMessage string
		headers                 map[string]string
	}{
		{
			name:                   "Valid Update",
			templateID:             certificateTemplateID,
			newStatus:              "verified",
			detail:                 new("Certificate Verified"),
			expectedResponseStatus: http.StatusOK,
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
		{
			name:                    "Invalid Status",
			templateID:              certificateTemplateID,
			newStatus:               "invalid_status",
			expectedResponseStatus:  http.StatusUnprocessableEntity,
			expectedResponseMessage: "invalid status value",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
		{
			name:                    "Wrong node key",
			templateID:              certificateTemplateID,
			newStatus:               "verified",
			expectedResponseStatus:  http.StatusUnauthorized,
			expectedResponseMessage: "host certificate template not found",
			headers: map[string]string{
				"Authorization": "Node key wrong-node-key",
			},
		},
		{
			name:                    "With no auth headers",
			templateID:              certificateTemplateID,
			newStatus:               "verified",
			expectedResponseStatus:  http.StatusUnauthorized,
			expectedResponseMessage: "host certificate template not found",
		},
		{
			name:                    "Wrong Template ID",
			templateID:              9999,
			newStatus:               "verified",
			expectedResponseStatus:  http.StatusNotFound,
			expectedResponseMessage: "host certificate template not found",
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
		{
			name:                   "with operation_type install",
			templateID:             certificateTemplateID,
			newStatus:              "verified",
			expectedResponseStatus: http.StatusOK,
			newOperationType:       new("install"),
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
		{
			name:                   "with operation_type remove",
			templateID:             certificateTemplateID,
			newStatus:              "verified",
			expectedResponseStatus: http.StatusOK,
			newOperationType:       new("remove"),
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
		{
			name:                   "with operation_type empty string",
			templateID:             certificateTemplateID,
			newStatus:              "verified",
			expectedResponseStatus: http.StatusOK,
			newOperationType:       new(""),
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
		{
			name:                    "with invalid operation_type",
			templateID:              certificateTemplateID,
			newStatus:               "verified",
			expectedResponseStatus:  http.StatusUnprocessableEntity,
			expectedResponseMessage: "must be 'install' or 'remove'",
			newOperationType:        new("invalid_operation"),
			headers: map[string]string{
				"Authorization": fmt.Sprintf("Node key %s", orbitNodeKey),
			},
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("TestUpdateHostCertificateTemplate:%s", tc.name), func(t *testing.T) {
			req, err := json.Marshal(updateCertificateStatusRequest{
				Status:        tc.newStatus,
				Detail:        tc.detail,
				OperationType: tc.newOperationType,
			})
			require.NoError(t, err)

			resp := s.DoRawWithHeaders("PUT", fmt.Sprintf("/api/fleetd/certificates/%d/status", tc.templateID), req, tc.expectedResponseStatus, tc.headers)
			require.NoError(t, resp.Body.Close())
		})
	}
}

func (s *integrationTestSuite) TestDeleteCertificateTemplate() {
	t := s.T()
	ctx := context.Background()

	// Create a test team
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestDeleteCertificateTemplate Team"})
	require.NoError(t, err)
	teamID := team.ID

	// Create a test certificate authority
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      new("TestDeleteCertificateTemplate SCEP CA"),
		URL:       new("http://localhost:8080/scep"),
		Challenge: new("test-challenge"),
	})
	require.NoError(t, err)
	caID := ca.ID

	certTemplate := &fleet.CertificateTemplate{
		Name:                   "TestDeleteCertificateTemplate-Cert",
		TeamID:                 teamID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Subject",
	}
	savedTemplate, err := s.ds.CreateCertificateTemplate(ctx, certTemplate)
	require.NoError(t, err)
	require.NotNil(t, savedTemplate)
	certificateTemplateID := savedTemplate.ID
	certTemplateName := savedTemplate.Name

	// Create hosts with different certificate template statuses
	hostPending, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-template-host-pending",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostDelivered, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-template-host-delivered",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostVerified, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-template-host-verified",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostFailed, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-template-host-failed",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	// Insert host_certificate_templates with various statuses
	insertSQL := `
		INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, fleet_challenge, name)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		// Pending status - should be deleted
		_, err := q.ExecContext(ctx, insertSQL, hostPending.UUID, certificateTemplateID, "pending", "install", nil, certTemplateName)
		require.NoError(t, err)
		// Delivered status - should be updated to pending/remove
		_, err = q.ExecContext(ctx, insertSQL, hostDelivered.UUID, certificateTemplateID, "delivered", "install", "challenge1", certTemplateName)
		require.NoError(t, err)
		// Verified status - should be updated to pending/remove
		_, err = q.ExecContext(ctx, insertSQL, hostVerified.UUID, certificateTemplateID, "verified", "install", "challenge2", certTemplateName)
		require.NoError(t, err)
		// Failed status - should be deleted (never successfully installed)
		_, err = q.ExecContext(ctx, insertSQL, hostFailed.UUID, certificateTemplateID, "failed", "install", "challenge3", certTemplateName)
		require.NoError(t, err)
		return nil
	})

	// Enable Android MDM so GetHost returns certificate template profiles
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	origAndroidEnabled := appCfg.MDM.AndroidEnabledAndConfigured
	appCfg.MDM.AndroidEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	err = s.ds.SetAndroidEnabledAndConfigured(ctx, true)
	require.NoError(t, err)
	defer func() {
		appCfg.MDM.AndroidEnabledAndConfigured = origAndroidEnabled
		_ = s.ds.SaveAppConfig(ctx, appCfg)
		_ = s.ds.SetAndroidEnabledAndConfigured(ctx, origAndroidEnabled)
	}()

	// Helper to find the certificate template profile by name
	findProfile := func(profiles *[]fleet.HostMDMProfile, name string) *fleet.HostMDMProfile {
		if profiles == nil {
			return nil
		}
		for _, p := range *profiles {
			if p.Name == name {
				return &p
			}
		}
		return nil
	}

	// Verify the records exist before deletion via GetHost API
	var getHostResp getHostResponse
	for _, tc := range []struct {
		host           *fleet.Host
		hostName       string
		expectedStatus string
	}{
		{hostPending, "hostPending", string(fleet.CertificateTemplatePending)},
		{hostDelivered, "hostDelivered", string(fleet.CertificateTemplateDelivered)},
		{hostVerified, "hostVerified", string(fleet.CertificateTemplateVerified)},
		{hostFailed, "hostFailed", string(fleet.CertificateTemplateFailed)},
	} {
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
		require.NotNil(t, getHostResp.Host.MDM.Profiles, "%s should have MDM profiles before deletion", tc.hostName)

		profile := findProfile(getHostResp.Host.MDM.Profiles, certTemplateName)
		require.NotNil(t, profile, "%s should have certificate template profile %s before deletion", tc.hostName, certTemplateName)
		require.NotNil(t, profile.Status, "%s profile status should not be nil", tc.hostName)
		require.Equal(t, tc.expectedStatus, *profile.Status, "%s profile status should be %s before deletion", tc.hostName, tc.expectedStatus)
		require.Equal(t, fleet.MDMOperationTypeInstall, profile.OperationType, "%s profile operation_type should be install before deletion", tc.hostName)
	}

	// Delete the certificate template via API
	var deleteResp deleteCertificateTemplateResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/certificates/%d", certificateTemplateID), nil, http.StatusOK, &deleteResp)

	// After deletion:
	// - hostPending (pending/install) should have NO profile (record was deleted - never installed)
	// - hostFailed (failed/install) should have NO profile (record was deleted - never successfully installed)
	// - hostDelivered, hostVerified should have pending/remove profiles
	//   (kept for cron job to process removal from devices)

	// Verify hostPending has no profile after deletion (was pending/install, never installed)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostPending.ID), nil, http.StatusOK, &getHostResp)
	profile := findProfile(getHostResp.Host.MDM.Profiles, certTemplateName)
	require.Nil(t, profile, "hostPending should not have certificate template profile after deletion")

	// Verify hostFailed has no profile after deletion (was failed/install, never successfully installed)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostFailed.ID), nil, http.StatusOK, &getHostResp)
	profile = findProfile(getHostResp.Host.MDM.Profiles, certTemplateName)
	require.Nil(t, profile, "hostFailed should not have certificate template profile after deletion")

	// Verify hosts that had delivered/verified status now have pending/remove profiles
	for _, tc := range []struct {
		host     *fleet.Host
		hostName string
	}{
		{hostDelivered, "hostDelivered"},
		{hostVerified, "hostVerified"},
	} {
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
		profile := findProfile(getHostResp.Host.MDM.Profiles, certTemplateName)
		require.NotNil(t, profile, "%s should have pending remove profile after deletion", tc.hostName)
		require.NotNil(t, profile.Status, "%s profile status should not be nil", tc.hostName)
		require.Equal(t, string(fleet.CertificateTemplatePending), *profile.Status, "%s profile status should be pending after deletion", tc.hostName)
		require.Equal(t, fleet.MDMOperationTypeRemove, profile.OperationType, "%s profile operation_type should be remove after deletion", tc.hostName)
	}
}

func (s *integrationTestSuite) TestDeleteCertificateTemplateSpec() {
	t := s.T()
	ctx := context.Background()

	// Create a test team
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "TestDeleteCertificateTemplateSpec Team"})
	require.NoError(t, err)
	teamID := team.ID

	// Create a test certificate authority
	ca, err := s.ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      new("TestDeleteCertificateTemplateSpec SCEP CA"),
		URL:       new("http://localhost:8080/scep"),
		Challenge: new("test-challenge"),
	})
	require.NoError(t, err)
	caID := ca.ID

	// Create two certificate templates
	certTemplate1 := &fleet.CertificateTemplate{
		Name:                   "TestDeleteCertificateTemplateSpec-Cert1",
		TeamID:                 teamID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Subject 1",
	}
	savedTemplate1, err := s.ds.CreateCertificateTemplate(ctx, certTemplate1)
	require.NoError(t, err)
	require.NotNil(t, savedTemplate1)
	certTemplateID1 := savedTemplate1.ID
	certTemplateName1 := savedTemplate1.Name

	certTemplate2 := &fleet.CertificateTemplate{
		Name:                   "TestDeleteCertificateTemplateSpec-Cert2",
		TeamID:                 teamID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Subject 2",
	}
	savedTemplate2, err := s.ds.CreateCertificateTemplate(ctx, certTemplate2)
	require.NoError(t, err)
	require.NotNil(t, savedTemplate2)
	certTemplateID2 := savedTemplate2.ID
	certTemplateName2 := savedTemplate2.Name

	// Create hosts with different certificate template statuses
	hostPending, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-spec-host-pending",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostDelivered, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-spec-host-delivered",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostVerified, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-spec-host-verified",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	hostFailed, err := s.ds.NewHost(ctx, &fleet.Host{
		UUID:     uuid.New().String(),
		Hostname: "test-delete-cert-spec-host-failed",
		Platform: "android",
		TeamID:   &teamID,
	})
	require.NoError(t, err)

	// Insert host_certificate_templates with various statuses for both templates
	insertSQL := `
		INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, status, operation_type, fleet_challenge, name)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		// Template 1 - hosts with pending and delivered status
		_, err := q.ExecContext(ctx, insertSQL, hostPending.UUID, certTemplateID1, "pending", "install", nil, certTemplateName1)
		require.NoError(t, err)
		_, err = q.ExecContext(ctx, insertSQL, hostDelivered.UUID, certTemplateID1, "delivered", "install", "challenge1", certTemplateName1)
		require.NoError(t, err)

		// Template 2 - hosts with verified and failed status
		_, err = q.ExecContext(ctx, insertSQL, hostVerified.UUID, certTemplateID2, "verified", "install", "challenge2", certTemplateName2)
		require.NoError(t, err)
		_, err = q.ExecContext(ctx, insertSQL, hostFailed.UUID, certTemplateID2, "failed", "install", "challenge3", certTemplateName2)
		require.NoError(t, err)
		return nil
	})

	// Enable Android MDM so GetHost returns certificate template profiles
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	origAndroidEnabled := appCfg.MDM.AndroidEnabledAndConfigured
	appCfg.MDM.AndroidEnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	err = s.ds.SetAndroidEnabledAndConfigured(ctx, true)
	require.NoError(t, err)
	defer func() {
		appCfg.MDM.AndroidEnabledAndConfigured = origAndroidEnabled
		_ = s.ds.SaveAppConfig(ctx, appCfg)
		_ = s.ds.SetAndroidEnabledAndConfigured(ctx, origAndroidEnabled)
	}()

	// Helper to find the certificate template profile by name
	findProfile := func(profiles *[]fleet.HostMDMProfile, name string) *fleet.HostMDMProfile {
		if profiles == nil {
			return nil
		}
		for _, p := range *profiles {
			if p.Name == name {
				return &p
			}
		}
		return nil
	}

	// Verify the records exist before deletion via GetHost API
	var getHostResp getHostResponse
	for _, tc := range []struct {
		host           *fleet.Host
		hostName       string
		expectedStatus string
		templateName   string
	}{
		{hostPending, "hostPending", string(fleet.CertificateTemplatePending), certTemplateName1},
		{hostDelivered, "hostDelivered", string(fleet.CertificateTemplateDelivered), certTemplateName1},
		{hostVerified, "hostVerified", string(fleet.CertificateTemplateVerified), certTemplateName2},
		{hostFailed, "hostFailed", string(fleet.CertificateTemplateFailed), certTemplateName2},
	} {
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
		require.NotNil(t, getHostResp.Host.MDM.Profiles, "%s should have MDM profiles before deletion", tc.hostName)

		profile := findProfile(getHostResp.Host.MDM.Profiles, tc.templateName)
		require.NotNil(t, profile, "%s should have certificate template profile %s before deletion", tc.hostName, tc.templateName)
		require.NotNil(t, profile.Status, "%s profile status should not be nil", tc.hostName)
		require.Equal(t, tc.expectedStatus, *profile.Status, "%s profile status should be %s before deletion", tc.hostName, tc.expectedStatus)
		require.Equal(t, fleet.MDMOperationTypeInstall, profile.OperationType, "%s profile operation_type should be install before deletion", tc.hostName)
	}

	// Delete both certificate templates via spec endpoint (batch delete)
	var delBatchResp deleteCertificateTemplateSpecsResponse
	s.DoJSON("DELETE", "/api/latest/fleet/spec/certificates", map[string]any{
		"ids":     []uint{certTemplateID1, certTemplateID2},
		"team_id": teamID,
	}, http.StatusOK, &delBatchResp)

	// Verify certificate templates were deleted
	_, err = s.ds.GetCertificateTemplateById(ctx, certTemplateID1)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err), "certificate template 1 should be deleted")

	_, err = s.ds.GetCertificateTemplateById(ctx, certTemplateID2)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err), "certificate template 2 should be deleted")

	// After deletion:
	// - hostPending (pending/install) should have NO profile (record was deleted - never installed)
	// - hostFailed (failed/install) should have NO profile (record was deleted - never successfully installed)
	// - hostDelivered, hostVerified should have pending/remove profiles
	//   (kept for cron job to process removal from devices)

	// Verify hostPending has no profile after deletion (was pending/install, never installed)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostPending.ID), nil, http.StatusOK, &getHostResp)
	profile := findProfile(getHostResp.Host.MDM.Profiles, certTemplateName1)
	require.Nil(t, profile, "hostPending should not have certificate template profile after deletion")

	// Verify hostFailed has no profile after deletion (was failed/install, never successfully installed)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostFailed.ID), nil, http.StatusOK, &getHostResp)
	profile = findProfile(getHostResp.Host.MDM.Profiles, certTemplateName2)
	require.Nil(t, profile, "hostFailed should not have certificate template profile after deletion")

	// Verify hosts that had delivered/verified status now have pending/remove profiles
	for _, tc := range []struct {
		host         *fleet.Host
		hostName     string
		templateName string
	}{
		{hostDelivered, "hostDelivered", certTemplateName1},
		{hostVerified, "hostVerified", certTemplateName2},
	} {
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
		profile := findProfile(getHostResp.Host.MDM.Profiles, tc.templateName)
		require.NotNil(t, profile, "%s should have pending remove profile after deletion", tc.hostName)
		require.NotNil(t, profile.Status, "%s profile status should not be nil", tc.hostName)
		require.Equal(t, string(fleet.CertificateTemplatePending), *profile.Status, "%s profile status should be pending after deletion", tc.hostName)
		require.Equal(t, fleet.MDMOperationTypeRemove, profile.OperationType, "%s profile operation_type should be remove after deletion", tc.hostName)
	}
}

func (s *integrationTestSuite) TestOsqueryBodySizeLimit() {
	t := s.T()

	host := createOrbitEnrolledHost(t, "linux", "body-limit", s.ds)

	logLimit := int(fleet.DefaultMaxOsqueryLogWriteSize)
	distLimit := int(fleet.DefaultMaxOsqueryDistributedWriteSize)

	// Body over the per-route default must be rejected with 413. The padding
	// is inside a JSON string value so the body is syntactically valid up to
	// the point where the reader is cut off.
	logPrefix := fmt.Sprintf(`{"node_key":%q,"log_type":"status","data":["`, *host.NodeKey)
	logSuffix := `"]}`
	logPadSize := logLimit + 1 - len(logPrefix) - len(logSuffix)
	require.Positive(t, logPadSize, "padding must be positive")
	overLimitLog := []byte(logPrefix + strings.Repeat("x", logPadSize) + logSuffix)
	s.DoRawNoAuth("POST", "/api/osquery/log", overLimitLog, http.StatusRequestEntityTooLarge)

	// A well-formed body within the limit is accepted.
	withinLimitLog, err := json.Marshal(submitLogsRequest{
		NodeKey: *host.NodeKey,
		LogType: "status",
		Data:    []json.RawMessage{},
	})
	require.NoError(t, err)
	s.DoRawNoAuth("POST", "/api/osquery/log", withinLimitLog, http.StatusOK)

	// A truncated (malformed) body within the limit must NOT return 413.
	// Before the fix, io.ErrUnexpectedEOF from the JSON decoder was incorrectly
	// converted to PayloadTooLargeError even when the reader had not been exhausted.
	// The correct response is 400 Bad Request.
	truncatedLog := fmt.Appendf(nil, `{"node_key":%q,"log_type":"status","data":[`, *host.NodeKey) // missing closing ]}
	s.DoRawNoAuth("POST", "/api/osquery/log", truncatedLog, http.StatusBadRequest)

	// Body over the per-route default must be rejected with 413.
	distPrefix := fmt.Sprintf(`{"node_key":%q,"queries":{"q1":[{"data":"`, *host.NodeKey)
	distSuffix := `"}]},"statuses":{"q1":0},"messages":{},"stats":{}}`
	distPadSize := distLimit + 1 - len(distPrefix) - len(distSuffix)
	require.Positive(t, distPadSize, "padding must be positive")
	overLimitDist := []byte(distPrefix + strings.Repeat("x", distPadSize) + distSuffix)
	s.DoRawNoAuth("POST", "/api/osquery/distributed/write", overLimitDist, http.StatusRequestEntityTooLarge)

	// A well-formed body within the limit is accepted.
	withinLimitDist, err := json.Marshal(submitDistributedQueryResultsRequestShim{
		NodeKey:  *host.NodeKey,
		Results:  map[string]json.RawMessage{},
		Statuses: map[string]any{},
		Messages: map[string]string{},
		Stats:    map[string]*fleet.Stats{},
	})
	require.NoError(t, err)
	s.DoRawNoAuth("POST", "/api/osquery/distributed/write", withinLimitDist, http.StatusOK)

	// A truncated body within the limit must NOT return 413 (same false-positive guard).
	// io.ErrUnexpectedEOF from the bodyDecoder path is now wrapped as BadRequestErr → 400.
	truncatedDist := fmt.Appendf(nil, `{"node_key":%q,"queries":{"q1":[`, *host.NodeKey) // missing closing
	s.DoRawNoAuth("POST", "/api/osquery/distributed/write", truncatedDist, http.StatusBadRequest)

	s.Run("config overrides take effect in body-auth mode", func() {
		// Spin up a second server with custom per-route limits and
		// confirm bodies above the override are rejected while bodies
		// below are accepted.
		const customLimit = 2 * units.MiB

		cfg := config.TestConfig()
		cfg.Osquery.MaxLogWriteBodySize = customLimit
		cfg.Osquery.MaxDistributedWriteBodySize = customLimit

		_, customServer := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
			FleetConfig:         &cfg,
			SkipCreateTestUsers: true,
		})
		s.T().Cleanup(customServer.Close)
		ts := withServer{server: customServer}
		ts.s = &s.Suite

		logPad := customLimit + 1 - len(logPrefix) - len(logSuffix)
		s.Require().Positive(logPad)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", logPad)+logSuffix),
			http.StatusRequestEntityTooLarge)
		ts.DoRawNoAuth("POST", "/api/osquery/log", withinLimitLog, http.StatusOK)

		distPad := customLimit + 1 - len(distPrefix) - len(distSuffix)
		s.Require().Positive(distPad)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(distPrefix+strings.Repeat("x", distPad)+distSuffix),
			http.StatusRequestEntityTooLarge)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write", withinLimitDist, http.StatusOK)
	})

	s.Run("header-auth mode imposes no body size limit", func() {
		// In header-auth mode the per-route configs are intentionally
		// ignored AND no body size limit applies. A body well above the
		// global default (and well above any per-route default) must
		// succeed when authenticated via header.
		cfg := config.TestConfig()
		cfg.Osquery.AllowBodyAuthFallback = false
		cfg.Osquery.MaxLogWriteBodySize = 1 * units.MiB         // ignored
		cfg.Osquery.MaxDistributedWriteBodySize = 1 * units.MiB // ignored

		_, customServer := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
			FleetConfig:         &cfg,
			SkipCreateTestUsers: true,
		})
		s.T().Cleanup(customServer.Close)
		ts := withServer{server: customServer}
		ts.s = &s.Suite

		// 12 MiB body — over both the global limit (1 MiB) and any
		// per-route default. Must succeed because header-auth mode
		// applies no body size constraint.
		oversizedLog, err := json.Marshal(submitLogsRequest{
			NodeKey: *host.NodeKey,
			LogType: "status",
			Data:    []json.RawMessage{json.RawMessage(`"` + strings.Repeat("x", 12*1024*1024) + `"`)},
		})
		s.Require().NoError(err)
		ts.DoRawWithHeaders("POST", "/api/osquery/log", oversizedLog,
			http.StatusOK,
			map[string]string{"Authorization": "NodeKey " + *host.NodeKey})
	})

	s.Run("endpoint_request_size_overrides wins over the osquery per-route default in body-auth mode", func() {
		const overrideLimit = 15 * units.MiB

		oldOverrides := platform_http.EndpointRequestSizeOverrides
		s.T().Cleanup(func() { platform_http.EndpointRequestSizeOverrides = oldOverrides })
		platform_http.EndpointRequestSizeOverrides = map[string]int64{
			"/api/osquery/log":               overrideLimit,
			"/api/osquery/distributed/write": overrideLimit,
		}

		cfg := config.TestConfig()
		_, customServer := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
			FleetConfig:         &cfg,
			SkipCreateTestUsers: true,
		})
		s.T().Cleanup(customServer.Close)
		ts := withServer{server: customServer}
		ts.s = &s.Suite

		// Above the osquery route's own default (10MiB) but within the override (15MiB).
		// It must succeed, proving the override raised the effective limit.
		betweenPad := logLimit + 2*int(units.MiB) + 1 - len(logPrefix) - len(logSuffix)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", betweenPad)+logSuffix),
			http.StatusOK)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(logPrefix+strings.Repeat("x", betweenPad)+logSuffix),
			http.StatusOK)

		// Above the override itself. It must be rejected.
		aboveOverridePad := int(overrideLimit) + 1 - len(logPrefix) - len(logSuffix)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", aboveOverridePad)+logSuffix),
			http.StatusRequestEntityTooLarge)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(logPrefix+strings.Repeat("x", aboveOverridePad)+logSuffix),
			http.StatusRequestEntityTooLarge)
	})

	s.Run("endpoint_request_size_overrides wins over an explicitly configured osquery_max_*_body_size  when larger", func() {
		// osquery_max_log_write_body_size & osquery_max_distributed_write_body_size (deprecated but supported) are explicitly set.
		// The override must still win over it when larger, proving the two config sources compose via the same "largest wins" comparison.
		const (
			deprecatedLimit = 5 * units.MiB
			overrideLimit   = 15 * units.MiB
		)

		oldOverrides := platform_http.EndpointRequestSizeOverrides
		s.T().Cleanup(func() { platform_http.EndpointRequestSizeOverrides = oldOverrides })
		platform_http.EndpointRequestSizeOverrides = map[string]int64{
			"/api/osquery/log":               overrideLimit,
			"/api/osquery/distributed/write": overrideLimit,
		}

		cfg := config.TestConfig()
		cfg.Osquery.MaxLogWriteBodySize = deprecatedLimit
		cfg.Osquery.MaxDistributedWriteBodySize = deprecatedLimit

		_, customServer := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
			FleetConfig:         &cfg,
			SkipCreateTestUsers: true,
		})
		s.T().Cleanup(customServer.Close)
		ts := withServer{server: customServer}
		ts.s = &s.Suite

		// Above the explicit deprecated-config limit (5MiB) but within the override (15MiB)
		// It must succeed.
		aboveDeprecatedPad := int(deprecatedLimit) + int(units.MiB) + 1 - len(logPrefix) - len(logSuffix)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", aboveDeprecatedPad)+logSuffix),
			http.StatusOK)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(logPrefix+strings.Repeat("x", aboveDeprecatedPad)+logSuffix),
			http.StatusOK)

		// Above the override itself. It must be rejected.
		aboveOverridePad := int(overrideLimit) + 1 - len(logPrefix) - len(logSuffix)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", aboveOverridePad)+logSuffix),
			http.StatusRequestEntityTooLarge)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(logPrefix+strings.Repeat("x", aboveOverridePad)+logSuffix),
			http.StatusRequestEntityTooLarge)
	})

	s.Run("explicitly configured osquery_max_*_body_size wins when larger than endpoint_request_size_overrides", func() {
		// Reverse of above test.
		// A smaller override must not shrink the effective limit below an explicitly configured (larger) deprecated setting for the same path.
		const (
			deprecatedLimit = 15 * units.MiB
			overrideLimit   = 5 * units.MiB
		)

		oldOverrides := platform_http.EndpointRequestSizeOverrides
		s.T().Cleanup(func() { platform_http.EndpointRequestSizeOverrides = oldOverrides })
		platform_http.EndpointRequestSizeOverrides = map[string]int64{
			"/api/osquery/log":               overrideLimit,
			"/api/osquery/distributed/write": overrideLimit,
		}

		cfg := config.TestConfig()
		cfg.Osquery.MaxLogWriteBodySize = deprecatedLimit
		cfg.Osquery.MaxDistributedWriteBodySize = deprecatedLimit

		_, customServer := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
			FleetConfig:         &cfg,
			SkipCreateTestUsers: true,
		})
		s.T().Cleanup(customServer.Close)
		ts := withServer{server: customServer}
		ts.s = &s.Suite

		// Above the (smaller) override but within the explicit deprecated limit.
		// It must succeed, proving the override didn't shrink it.
		aboveOverridePad := int(overrideLimit) + int(units.MiB) + 1 - len(logPrefix) - len(logSuffix)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", aboveOverridePad)+logSuffix),
			http.StatusOK)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(logPrefix+strings.Repeat("x", aboveOverridePad)+logSuffix),
			http.StatusOK)

		// Above the deprecated limit itself. It must be rejected.
		aboveDeprecatedPad := int(deprecatedLimit) + 1 - len(logPrefix) - len(logSuffix)
		s.Require().Positive(aboveDeprecatedPad)
		ts.DoRawNoAuth("POST", "/api/osquery/log",
			[]byte(logPrefix+strings.Repeat("x", aboveDeprecatedPad)+logSuffix),
			http.StatusRequestEntityTooLarge)
		ts.DoRawNoAuth("POST", "/api/osquery/distributed/write",
			[]byte(logPrefix+strings.Repeat("x", aboveDeprecatedPad)+logSuffix),
			http.StatusRequestEntityTooLarge)
	})

	s.Run("endpoint_request_size_overrides does not apply in header-auth mode", func() {
		// Header-auth routes opt out of the size-limiting mechanism entirely (SkipRequestBodySizeLimit),
		// so a configured override for the same path must not reintroduce a limit there.
		const overrideLimit = 2 * units.MiB // well below the 12MiB body sent below

		oldOverrides := platform_http.EndpointRequestSizeOverrides
		s.T().Cleanup(func() { platform_http.EndpointRequestSizeOverrides = oldOverrides })
		platform_http.EndpointRequestSizeOverrides = map[string]int64{
			"/api/osquery/log":               overrideLimit,
			"/api/osquery/distributed/write": overrideLimit,
		}

		cfg := config.TestConfig()
		cfg.Osquery.AllowBodyAuthFallback = false

		_, customServer := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
			FleetConfig:         &cfg,
			SkipCreateTestUsers: true,
		})
		s.T().Cleanup(customServer.Close)
		ts := withServer{server: customServer}
		ts.s = &s.Suite

		oversizedLog, err := json.Marshal(submitLogsRequest{
			NodeKey: *host.NodeKey,
			LogType: "status",
			Data:    []json.RawMessage{json.RawMessage(`"` + strings.Repeat("x", 12*1024*1024) + `"`)},
		})
		s.Require().NoError(err)
		ts.DoRawWithHeaders("POST", "/api/osquery/log", oversizedLog,
			http.StatusOK,
			map[string]string{"Authorization": "NodeKey " + *host.NodeKey})
		ts.DoRawWithHeaders("POST", "/api/osquery/distributed/write", oversizedLog,
			http.StatusOK,
			map[string]string{"Authorization": "NodeKey " + *host.NodeKey})
	})
}

func (s *integrationTestSuite) TestLabelScopePremiumGate() {
	t := s.T()

	// One label suffices to exercise the gate.
	var lblResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.LabelPayload{Name: uuid.NewString(), Query: "SELECT 1"}, http.StatusOK, &lblResp)
	lbl := lblResp.Label

	// All policy label scope fields on GlobalPolicyRequest are tagged
	// `premium:"true"`, so the endpoint decoder rejects with 400 before reaching
	// the service handler.
	for _, req := range []fleet.GlobalPolicyRequest{
		{Name: "premium-include-all-" + t.Name(), Query: "SELECT 1", LabelsIncludeAll: []string{lbl.Name}},
		{Name: "premium-include-any-" + t.Name(), Query: "SELECT 1", LabelsIncludeAny: []string{lbl.Name}},
		{Name: "premium-exclude-all-" + t.Name(), Query: "SELECT 1", LabelsExcludeAll: []string{lbl.Name}},
		{Name: "premium-exclude-any-" + t.Name(), Query: "SELECT 1", LabelsExcludeAny: []string{lbl.Name}},
	} {
		s.DoJSON("POST", "/api/latest/fleet/policies", req, http.StatusBadRequest, &fleet.GlobalPolicyResponse{})
	}

	// A policy without any label scope can still be created on free-tier.
	var freeOK fleet.GlobalPolicyResponse
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
		Name:  "free-no-labels-" + t.Name(),
		Query: "SELECT 1",
	}, http.StatusOK, &freeOK)
	require.NotNil(t, freeOK.Policy)

	// Free-tier should reject PATCHing a policy to add any label scope (middleware → 400).
	for _, payload := range []fleet.ModifyPolicyPayload{
		{LabelsIncludeAll: []string{lbl.Name}},
		{LabelsIncludeAny: []string{lbl.Name}},
		{LabelsExcludeAll: []string{lbl.Name}},
		{LabelsExcludeAny: []string{lbl.Name}},
	} {
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", freeOK.Policy.ID), fleet.ModifyGlobalPolicyRequest{
			ModifyPolicyPayload: payload,
		}, http.StatusBadRequest, &fleet.ModifyGlobalPolicyResponse{})
	}

	for _, payload := range []fleet.QueryPayload{
		{Name: new("q-any-" + t.Name()), Query: new("SELECT 1"), Logging: new(fleet.LoggingSnapshot), LabelsIncludeAny: []string{lbl.Name}},
		{Name: new("q-all-" + t.Name()), Query: new("SELECT 1"), Logging: new(fleet.LoggingSnapshot), LabelsIncludeAll: []string{lbl.Name}},
	} {
		s.DoJSON("POST", "/api/latest/fleet/queries", payload, http.StatusPaymentRequired, &fleet.CreateQueryResponse{})
	}

	// PolicySpec label fields don't carry the `premium:"true"` middleware tag,
	// so the gate fires at the service layer and returns 402 (PaymentRequired).
	for _, spec := range []*fleet.PolicySpec{
		{Name: "spec-include-all-" + t.Name(), Query: "SELECT 1", LabelsIncludeAll: []string{lbl.Name}},
		{Name: "spec-include-any-" + t.Name(), Query: "SELECT 1", LabelsIncludeAny: []string{lbl.Name}},
		{Name: "spec-exclude-all-" + t.Name(), Query: "SELECT 1", LabelsExcludeAll: []string{lbl.Name}},
		{Name: "spec-exclude-any-" + t.Name(), Query: "SELECT 1", LabelsExcludeAny: []string{lbl.Name}},
	} {
		s.DoJSON("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
			Specs: []*fleet.PolicySpec{spec},
		}, http.StatusPaymentRequired, &fleet.ApplyPolicySpecsResponse{})
	}

	// Same for QuerySpec.
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{{
			Name:             "qspec-premium-gate-" + t.Name(),
			Query:            "SELECT 1",
			Logging:          fleet.LoggingSnapshot,
			LabelsIncludeAll: []string{lbl.Name},
		}},
	}, http.StatusPaymentRequired, &fleet.ApplyQuerySpecsResponse{})
}

func (s *integrationTestSuite) TestOrgLogoUpload() {
	t := s.T()

	pngImg := image.NewRGBA(image.Rect(0, 0, 1, 1))
	pngImg.Set(0, 0, color.RGBA{R: 0, G: 128, B: 0, A: 255})
	var pngBuf bytes.Buffer
	require.NoError(t, png.Encode(&pngBuf, pngImg))
	pngBytes := pngBuf.Bytes()

	buildLogoBody := func(filename string, content []byte) ([]byte, map[string]string) {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		fw, err := w.CreateFormFile("logo", filename)
		require.NoError(t, err)
		_, err = io.Copy(fw, bytes.NewReader(content))
		require.NoError(t, err)
		require.NoError(t, w.Close())
		return body.Bytes(), map[string]string{
			"Content-Type":  w.FormDataContentType(),
			"Accept":        "application/json",
			"Authorization": "Bearer " + s.token,
		}
	}

	// 1. Upload as admin: 200, AppConfig URL set to the Fleet-hosted serving
	// path, GET returns the bytes back with the right content type.
	body, headers := buildLogoBody("logo.png", pngBytes)
	s.DoRawWithHeaders("PUT", "/api/v1/fleet/logo?mode=light", body, http.StatusOK, headers)

	var acResp appConfigResponse
	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, acResp.OrgInfo.OrgLogoURLLightMode, "/api/latest/fleet/logo")
	require.Contains(t, acResp.OrgInfo.OrgLogoURLLightMode, "mode=light")
	// Deprecated key is in sync.
	require.Equal(t, acResp.OrgInfo.OrgLogoURLLightMode, acResp.OrgInfo.OrgLogoURLLightBackground)

	res := s.DoRawNoAuth("GET", "/api/latest/fleet/logo?mode=light", nil, http.StatusOK)
	gotBody, err := io.ReadAll(res.Body)
	require.NoError(t, res.Body.Close())
	require.NoError(t, err)
	require.Equal(t, pngBytes, gotBody)
	require.Equal(t, "image/png", res.Header.Get("Content-Type"))

	// 2. Upload a second mode (dark) as admin so the delete-lifecycle assertions
	// at the bottom can confirm modes are independent.
	body, headers = buildLogoBody("dark.png", pngBytes)
	s.DoRawWithHeaders("PUT", "/api/v1/fleet/logo?mode=dark", body, http.StatusOK, headers)

	// 3. Auth: a maintainer is rejected.
	maintainerEmail := "maintainer-logo@example.com"
	maintainerUser := &fleet.User{
		Name:       "Maintainer Logo",
		Email:      maintainerEmail,
		GlobalRole: new(fleet.RoleMaintainer),
	}
	require.NoError(t, maintainerUser.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(t.Context(), maintainerUser)
	require.NoError(t, err)

	s.token = s.getCachedUserToken(maintainerEmail, test.GoodPassword)
	body, headers = buildLogoBody("nope.png", pngBytes)
	s.DoRawWithHeaders("PUT", "/api/v1/fleet/logo?mode=light", body, http.StatusForbidden, headers)
	s.token = s.getTestAdminToken()

	// 4. A non-image payload is rejected at upload time.
	body, headers = buildLogoBody("not-an-image.png", []byte("plain text, definitely not a PNG"))
	s.DoRawWithHeaders("PUT", "/api/v1/fleet/logo?mode=light", body, http.StatusBadRequest, headers)

	// 5. DELETE clears the URL field and the GET endpoint returns 404 for
	// the affected mode while the other mode is unaffected.
	s.Do("DELETE", "/api/v1/fleet/logo", nil, http.StatusOK, "mode", "light")

	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	require.Empty(t, acResp.OrgInfo.OrgLogoURLLightMode)
	require.Empty(t, acResp.OrgInfo.OrgLogoURLLightBackground)
	require.Contains(t, acResp.OrgInfo.OrgLogoURLDarkMode, "/api/latest/fleet/logo")

	res = s.DoRawNoAuth("GET", "/api/latest/fleet/logo?mode=light", nil, http.StatusNotFound)
	require.NoError(t, res.Body.Close())

	// 6. DELETE is idempotent — a second DELETE for the same mode (now
	// empty) returns 200 with no error.
	s.Do("DELETE", "/api/v1/fleet/logo", nil, http.StatusOK, "mode", "light")

	// 7. DELETE on an external URL (no blob) clears the URL field.
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"org_info": {
			"org_logo_url": "https://placehold.co/100",
			"org_logo_url_dark_mode": "https://placehold.co/100",
			"org_logo_url_light_background": "https://placehold.co/200",
			"org_logo_url_light_mode": "https://placehold.co/200"
		}
	}`), http.StatusOK)

	s.Do("DELETE", "/api/v1/fleet/logo", nil, http.StatusOK, "mode", "dark")
	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	require.Empty(t, acResp.OrgInfo.OrgLogoURLDarkMode)
	require.Empty(t, acResp.OrgInfo.OrgLogoURL)
	require.Equal(t, "https://placehold.co/200", acResp.OrgInfo.OrgLogoURLLightMode)
	require.Equal(t, "https://placehold.co/200", acResp.OrgInfo.OrgLogoURLLightBackground)

	// 8. PATCH transitioning a Fleet-hosted URL to an external URL must
	// preserve the external URL and delete the previously-stored blob.
	body, headers = buildLogoBody("light2.png", pngBytes)
	s.DoRawWithHeaders("PUT", "/api/v1/fleet/logo?mode=light", body, http.StatusOK, headers)
	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, acResp.OrgInfo.OrgLogoURLLightMode, "/api/latest/fleet/logo")

	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"org_info": {
			"org_logo_url_light_mode": "https://placehold.co/300",
			"org_logo_url_light_background": "https://placehold.co/300"
		}
	}`), http.StatusOK)
	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	require.Equal(t, "https://placehold.co/300", acResp.OrgInfo.OrgLogoURLLightMode)
	require.Equal(t, "https://placehold.co/300", acResp.OrgInfo.OrgLogoURLLightBackground)

	// The orphan blob is actually gone: GET /logo?mode=light returns 404.
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/logo?mode=light", nil, http.StatusNotFound)
	require.NoError(t, res.Body.Close())

	// And the cleanup recorded a deleted_org_logo activity.
	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)
	var sawAutoCleanupActivity bool
	for _, a := range activities.Activities {
		if a.Type != "deleted_org_logo" || a.Details == nil {
			continue
		}
		var d struct {
			Mode string `json:"mode"`
		}
		if err := json.Unmarshal(*a.Details, &d); err == nil && d.Mode == string(fleet.OrgLogoModeLight) {
			sawAutoCleanupActivity = true
			break
		}
	}
	assert.True(t, sawAutoCleanupActivity, "auto-cleanup must emit a deleted_org_logo activity for the affected mode")
}

func (s *integrationTestSuite) TestOrbitDebugLoggingOnEnroll() {
	t := s.T()
	ctx := context.Background()

	// Reject above cap.
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "orbit": {"debug_logging_on_enroll_duration": 86401} }
	}`), http.StatusBadRequest, &acResp)

	// Reject negative.
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "orbit": {"debug_logging_on_enroll_duration": -1} }
	}`), http.StatusBadRequest, &acResp)

	// Reject duration string (must be seconds, integer).
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "orbit": {"debug_logging_on_enroll_duration": "1h"} }
	}`), http.StatusBadRequest, &acResp)

	// 1h global window (3600 seconds).
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "orbit": {"debug_logging_on_enroll_duration": 3600} }
	}`), http.StatusOK, &acResp)

	secret := uuid.New().String()
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: secret}},
		},
	}, http.StatusOK, &applyResp)

	beforeEnroll := time.Now()
	var enrollResp enrollOrbitResponse
	hostUUID := uuid.New().String()
	s.DoJSON("POST", "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
		EnrollSecret:   secret,
		HardwareUUID:   hostUUID,
		HardwareSerial: uuid.New().String(),
		Hostname:       "enroll-debug-stamped",
		Platform:       "linux",
	}, http.StatusOK, &enrollResp)
	require.NotEmpty(t, enrollResp.OrbitNodeKey)

	stampedHost, err := s.ds.LoadHostByOrbitNodeKey(ctx, enrollResp.OrbitNodeKey)
	require.NoError(t, err)
	require.NotNil(t, stampedHost.OrbitDebugUntil)
	require.WithinDuration(t, beforeEnroll.Add(time.Hour), *stampedHost.OrbitDebugUntil, time.Minute)

	// Clearing the option stops stamping.
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "orbit": {"debug_logging_on_enroll_duration": 0} }
	}`), http.StatusOK, &acResp)

	var enrollResp2 enrollOrbitResponse
	hostUUID2 := uuid.New().String()
	s.DoJSON("POST", "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
		EnrollSecret:   secret,
		HardwareUUID:   hostUUID2,
		HardwareSerial: uuid.New().String(),
		Hostname:       "enroll-debug-not-stamped",
		Platform:       "linux",
	}, http.StatusOK, &enrollResp2)

	unstampedHost, err := s.ds.LoadHostByOrbitNodeKey(ctx, enrollResp2.OrbitNodeKey)
	require.NoError(t, err)
	require.Nil(t, unstampedHost.OrbitDebugUntil)
}

func (s *integrationTestSuite) TestOsqueryConfigPackCacheLabelScopedQueries() {
	t := s.T()
	ctx := t.Context()

	// Two teams so each ordering gets its own (cold) cache key.
	teamA, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-A"})
	require.NoError(t, err)
	teamB, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-B"})
	require.NoError(t, err)

	newHost := func(name string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   new(uuid.New().String()),
			NodeKey:         new(uuid.New().String()),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s", name, t.Name()),
			Platform:        "darwin",
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}
	hostA1 := newHost("a1", &teamA.ID) // label member
	hostA2 := newHost("a2", &teamA.ID) // not a member
	hostB1 := newHost("b1", &teamB.ID) // label member
	hostB2 := newHost("b2", &teamB.ID) // not a member

	label, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:  t.Name() + "-label",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	for _, h := range []*fleet.Host{hostA1, hostB1} {
		err = s.ds.RecordLabelQueryExecutions(ctx, h, map[uint]*bool{label.ID: new(true)}, time.Now(), false)
		require.NoError(t, err)
	}

	scoped := []fleet.LabelIdent{{LabelName: label.Name}}

	unscopedA, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name: t.Name() + "-unscoped-A", TeamID: &teamA.ID, Interval: 30,
		AutomationsEnabled: true, Logging: fleet.LoggingSnapshot,
		Query: "SELECT * FROM time;", Saved: true,
	})
	require.NoError(t, err)
	scopedA, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name: t.Name() + "-scoped-A", TeamID: &teamA.ID, Interval: 30,
		AutomationsEnabled: true, Logging: fleet.LoggingSnapshot,
		Query: "SELECT * FROM time;", Saved: true,
		LabelsIncludeAny: scoped,
	})
	require.NoError(t, err)
	unscopedB, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name: t.Name() + "-unscoped-B", TeamID: &teamB.ID, Interval: 30,
		AutomationsEnabled: true, Logging: fleet.LoggingSnapshot,
		Query: "SELECT * FROM time;", Saved: true,
	})
	require.NoError(t, err)
	scopedB, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name: t.Name() + "-scoped-B", TeamID: &teamB.ID, Interval: 30,
		AutomationsEnabled: true, Logging: fleet.LoggingSnapshot,
		Query: "SELECT * FROM time;", Saved: true,
		LabelsIncludeAny: scoped,
	})
	require.NoError(t, err)

	teamQueries := func(host *fleet.Host, teamID uint) map[string]any {
		req := getClientConfigRequest{NodeKey: *host.NodeKey}
		var resp getClientConfigResponse
		s.DoJSON("POST", "/api/osquery/config", req, http.StatusOK, &resp)
		packs, ok := resp.Config["packs"].(map[string]any)
		require.True(t, ok, "expected a packs key in the osquery config")
		teamPack, ok := packs[fmt.Sprintf("team-%d", teamID)].(map[string]any)
		require.True(t, ok, "expected a team pack in the osquery config")
		return teamPack["queries"].(map[string]any)
	}

	// Team A: the label member calls GetClientConfig first (populates the cache).
	queries := teamQueries(hostA1, teamA.ID)
	require.Contains(t, queries, unscopedA.Name)
	require.Contains(t, queries, scopedA.Name)

	// The non-member must NOT receive the label-scoped query.
	queries = teamQueries(hostA2, teamA.ID)
	require.Contains(t, queries, unscopedA.Name)
	require.NotContains(t, queries, scopedA.Name,
		"host that is not a member of the label received the label-scoped query")

	// Team B: the non-member calls GetClientConfig first (populates the cache).
	queries = teamQueries(hostB2, teamB.ID)
	require.Contains(t, queries, unscopedB.Name)
	require.NotContains(t, queries, scopedB.Name,
		"host that is not a member of the label received the label-scoped query")

	// The label member must still receive its label-scoped query.
	queries = teamQueries(hostB1, teamB.ID)
	require.Contains(t, queries, unscopedB.Name)
	require.Contains(t, queries, scopedB.Name,
		"label member did not receive its label-scoped query")
}

func (s *integrationTestSuite) TestTeamPolicyResendConfigProfileRequiresPremium() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	profileUUID := fleet.MDMAppleProfileUUIDPrefix + uuid.NewString()

	// Create: rejected at decode, before the profile is ever looked up.
	res := s.Do("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team.ID),
		&fleet.TeamPolicyRequest{
			Name:        "premium resend",
			Query:       "SELECT 1;",
			Platform:    "darwin",
			ProfileUUID: new(profileUUID),
		}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "requires a premium license")

	// Modify: same decode-time gate.
	pol, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:  "premium resend patch",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)

	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, pol.ID),
		json.RawMessage(fmt.Sprintf(`{"profile_uuid": %q}`, profileUUID)), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "requires a premium license")

	// GitOps spec apply: explicit license check, so this one is a 402.
	res = s.Do("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{{
			Name:        "premium resend spec",
			Query:       "SELECT 1;",
			Platform:    "darwin",
			Team:        team.Name,
			ProfileUUID: new(profileUUID),
		}},
	}, http.StatusPaymentRequired)
	require.Contains(t, extractServerErrorText(res.Body), "Requires Fleet Premium license")

	// Without profile_uuid the same spec applies cleanly on a free license.
	s.Do("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{{
			Name:     "premium resend spec",
			Query:    "SELECT 1;",
			Platform: "darwin",
			Team:     team.Name,
		}},
	}, http.StatusOK)
}
