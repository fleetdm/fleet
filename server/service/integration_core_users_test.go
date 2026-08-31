package service

// User management tests for the core (no-license) suite.
//
// Belongs here: user creation and modification incl. field validation, global role
// assignment, API-only users and their tokens, the user roles spec endpoint,
// invites and accepting them, and changing a user's own email.
//
// Does not belong here: authenticating a user, sessions, or password reset flows
// (integration_core_sessions_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/ghodss/yaml"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"
)

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
		Password:   &userRawPwd,
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
	assert.Empty(t, user.Teams)

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
	require.Empty(t, listResp.Invites)

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
	require.Empty(t, listResp.Invites)

	// list invites, next page is empty
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp, "page", "1", "per_page", "2")
	require.Empty(t, listResp.Invites)

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
	require.Empty(t, verify.GlobalRole.String)
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
	require.Empty(t, listResp.Invites)

	// delete a now non-existing invite
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), nil, http.StatusNotFound, &delResp)

	// create user from never used but deleted invite
	s.DoJSON("POST", "/api/latest/fleet/users", fleet.UserPayload{
		Name:        new("Full Name"),
		Password:    new(test.GoodPassword),
		Email:       new(inv.Email),
		InviteToken: &deletedInviteToken,
	}, http.StatusNotFound, &createFromInviteResp)
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
				InviteToken: &invite.Token,
			},
			http.StatusUnprocessableEntity,
		},
		{
			"empty email",
			fleet.UserPayload{
				Name:        new("Name"),
				Password:    &test.GoodPassword,
				Email:       new(""),
				InviteToken: &invite.Token,
			},
			http.StatusUnprocessableEntity,
		},
		{
			"empty password",
			fleet.UserPayload{
				Name:        new("Name"),
				Password:    new(""),
				Email:       new("a@b.c"),
				InviteToken: &invite.Token,
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
				InviteToken: &invite.Token,
			},
			http.StatusUnprocessableEntity,
		},
		{
			"api_endpoints not accepted",
			fleet.UserPayload{
				Name:         new("Name"),
				Password:     &test.GoodPassword,
				Email:        new("a@b.c"),
				InviteToken:  &invite.Token,
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
			InviteToken: &invite.Token,
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
			InviteToken: &invite.Token,
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
			InviteToken: &invite.Token,
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
			InviteToken: &invite.Token,
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
			InviteToken: &invite.Token,
		}, http.StatusUnprocessableEntity, &resp)

		_, err := s.ds.UserByEmail(ctx, email)
		require.True(t, fleet.IsNotFound(err), "expected no user to be created, got err: %v", err)
	})
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
	assert.Empty(t, listResp.Users)

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
	assert.Empty(t, getMeResp.User.Teams)
	assert.Empty(t, getMeResp.AvailableTeams)

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
	assert.Equal(t, &fleet.UserSettings{HiddenHostColumns: []string{"osquery_version"}}, getResp.Settings)

	resp = s.DoRawWithHeaders("GET", "/api/latest/fleet/me", []byte(""), http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ssn.Key),
	}, "include_ui_settings", "true")
	err = json.NewDecoder(resp.Body).Decode(&getMeResp)
	require.NoError(t, err)
	assert.Equal(t, uint(1), getMeResp.User.ID)
	assert.NotNil(t, getMeResp.User.GlobalRole)
	assert.Nil(t, getMeResp.User.Settings)
	assert.Equal(t, &fleet.UserSettings{HiddenHostColumns: []string{"osquery_version"}}, getResp.Settings)

	// modify user ui settings, check they are returned modified
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", 1), json.RawMessage(`{
		"settings": {
			"hidden_host_columns": ["hostname", "osquery_version"]}
	}`), http.StatusOK, &modResp)

	// get session user with ui settings, should now be modified, two endpoints
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", 1), nil, http.StatusOK, &getResp, "include_ui_settings", "true")
	assert.Equal(t, uint(1), getResp.User.ID)
	assert.Nil(t, getResp.User.Settings)
	assert.Equal(t, &fleet.UserSettings{HiddenHostColumns: []string{"hostname", "osquery_version"}}, getResp.Settings)

	resp = s.DoRawWithHeaders("GET", "/api/latest/fleet/me", []byte(""), http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ssn.Key),
	}, "include_ui_settings", "true")
	err = json.NewDecoder(resp.Body).Decode(&getMeResp)
	require.NoError(t, err)
	assert.Equal(t, uint(1), getMeResp.User.ID)
	assert.NotNil(t, getMeResp.User.GlobalRole)
	assert.Nil(t, getMeResp.User.Settings)
	assert.Equal(t, &fleet.UserSettings{HiddenHostColumns: []string{"hostname", "osquery_version"}}, getMeResp.Settings)

	// modify user ui settings, empty array, check they are returned correctly
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", 1), json.RawMessage(`{
		"settings": {
			"hidden_host_columns": []}
	}`), http.StatusOK, &modResp)

	// get session user with ui settings, should now be modified, two endpoints
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", 1), nil, http.StatusOK, &getResp, "include_ui_settings", "true")
	assert.Equal(t, uint(1), getResp.User.ID)
	assert.Nil(t, getResp.User.Settings)
	assert.Equal(t, &fleet.UserSettings{HiddenHostColumns: []string{}}, getResp.Settings)

	resp = s.DoRawWithHeaders("GET", "/api/latest/fleet/me", []byte(""), http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ssn.Key),
	}, "include_ui_settings", "true")
	err = json.NewDecoder(resp.Body).Decode(&getMeResp)
	require.NoError(t, err)
	assert.Equal(t, uint(1), getMeResp.User.ID)
	assert.NotNil(t, getMeResp.User.GlobalRole)
	assert.Nil(t, getMeResp.User.Settings)
	assert.Equal(t, &fleet.UserSettings{HiddenHostColumns: []string{}}, getMeResp.Settings)

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
	assert.Empty(t, loginResp.User.Teams)
	assert.Empty(t, loginResp.AvailableTeams)

	// get that user from `/users` endpoint and check that teams info is empty
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, u.ID, getResp.User.ID)
	assert.Empty(t, getResp.User.Teams)
	assert.Empty(t, getResp.AvailableTeams)

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
	require.NoError(t, err)
	user, err = s.ds.NewUser(context.Background(), user)
	require.NoError(t, err)

	// try to change email with an invalid token
	var changeResp changeEmailResponse
	s.DoJSON("GET", "/api/latest/fleet/email/change/invalidtoken", nil, http.StatusNotFound, &changeResp)

	// create a valid token for the test user
	err = s.ds.PendingEmailChange(context.Background(), user.ID, "testchangeemail2@example.com", "validtoken")
	require.NoError(t, err)

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
		NewPassword: &newRawPwd,
	}, http.StatusUnprocessableEntity, &modResp)

	// as the user: set new password with an invalid current password
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		NewPassword: &newRawPwd,
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
		NewPassword: &newRawPwd,
		Password:    new(testUsers["user2"].PlaintextPassword),
	}, http.StatusForbidden, &modResp)

	s.token = s.getTestAdminToken()

	// as an admin, set a new email, name and password without a current password
	newRawPwd = userRawPwd + "4"
	modResp = modifyUserResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		SSOEnabled:  new(false),
		NewPassword: &newRawPwd,
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
