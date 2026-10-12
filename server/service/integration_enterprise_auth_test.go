package service

// User, SSO, and API-only user tests for the enterprise (premium) suite.
//
// Belongs here: user creation and role changes by fleet admins, invited user MFA, SSO JIT
// provisioning and IdP-initiated login, and API-only users (creation, modification, endpoint
// restrictions, GitOps endpoint access).
//
// Does not belong here: Fleet Desktop / device SSO (integration_enterprise_desktop_test.go) and
// the GitOps settings and actions themselves (integration_enterprise_gitops_test.go).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	apiendpoints "github.com/fleetdm/fleet/v4/server/api_endpoints"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"
)

// Skipping admin-created users because we don't have email fully set up in integration tests
func (s *integrationEnterpriseTestSuite) TestInvitedUserMFA() {
	t := s.T()

	// create valid invite
	createInviteReq := createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      new("some@email.com"),
		Name:       new("some name"),
		GlobalRole: null.StringFrom(fleet.RoleAdmin),
		MFAEnabled: new(true),
		SSOEnabled: new(true),
	}}
	createInviteResp := createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusConflict, &createInviteResp)
	createInviteReq.SSOEnabled = nil
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
	require.NotNil(t, createInviteResp.Invite)
	require.NotZero(t, createInviteResp.Invite.ID)
	validInvite := *createInviteResp.Invite

	// create user from valid invite - the token was not returned via the
	// response's json, must get it from the db
	inv, err := s.ds.Invite(context.Background(), validInvite.ID)
	require.NoError(t, err)

	// verify the token with valid invite
	var verifyInvResp verifyInviteResponse
	s.DoJSON("GET", "/api/latest/fleet/invites/"+inv.Token, nil, http.StatusOK, &verifyInvResp)
	require.Equal(t, validInvite.ID, verifyInvResp.Invite.ID)

	var createFromInviteResp createUserResponse
	s.DoJSON("POST", "/api/latest/fleet/users", fleet.UserPayload{
		Name:        new("Full Name"),
		Password:    new(test.GoodPassword),
		Email:       new(inv.Email),
		InviteToken: new(inv.Token),
	}, http.StatusOK, &createFromInviteResp)
	require.True(t, createFromInviteResp.User.MFAEnabled)

	// create an invite with SSO, swap to MFA
	createInviteReq = createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      new("a@b.d"),
		Name:       new("some other name"),
		GlobalRole: null.StringFrom(fleet.RoleAdmin),
		SSOEnabled: new(true),
	}}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
	validInvite = *createInviteResp.Invite
	var updateInviteResp updateInviteResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), updateInviteRequest{
		InvitePayload: fleet.InvitePayload{MFAEnabled: new(true), SSOEnabled: new(false)},
	}, http.StatusOK, &updateInviteResp)
	require.True(t, updateInviteResp.Invite.MFAEnabled)
}

// A fleet-admin must not be able to create a user with roles in teams they don't administer,
// even when one team in the payload is their own.
//
// Both the admin create path (POST /users) and the API-only create
// path (POST /users/api_only) funnel through svc.CreateUser, so both are
// exercised here.
func (s *integrationEnterpriseTestSuite) TestTeamAdminCannotCreateUserInOtherTeams() {
	t := s.T()
	ctx := context.Background()

	// team1 is administered by our caller; team2 is not.
	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_team2"})
	require.NoError(t, err)
	defer func() {
		s.token = s.getTestAdminToken()
		require.NoError(t, s.ds.DeleteTeam(ctx, team1.ID))
		require.NoError(t, s.ds.DeleteTeam(ctx, team2.ID))
	}()

	// Create a user who is an admin of team1 only.
	teamAdminEmail := t.Name() + "_team1_admin@example.com"
	teamAdmin := &fleet.User{
		Name:  teamAdminEmail,
		Email: teamAdminEmail,
		Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleAdmin}},
	}
	require.NoError(t, teamAdmin.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, teamAdmin)
	require.NoError(t, err)

	// Act as the team1 admin for the rest of the test.
	s.token = s.getTestToken(teamAdmin.Email, test.GoodPassword)

	var resp createUserResponse

	// Admin create path: a payload spanning team1 (allowed) and team2 (not
	// allowed) must be rejected wholesale, not partially honored.
	s.DoJSON("POST", "/api/latest/fleet/users/admin", fleet.UserPayload{
		Name:     new(t.Name() + "_crossteam"),
		Email:    new(t.Name() + "_crossteam@example.com"),
		Password: new(test.GoodPassword),
		Teams: &[]fleet.UserTeam{
			{Team: *team1, Role: fleet.RoleAdmin},
			{Team: *team2, Role: fleet.RoleAdmin},
		},
	}, http.StatusForbidden, &resp)

	// The forbidden request must not have created the user.
	_, err = s.ds.UserByEmail(ctx, t.Name()+"_crossteam@example.com")
	require.True(t, fleet.IsNotFound(err))

	// API-only create path inherits the same authorization, so it must reject
	// the cross-team payload too.
	s.DoJSON("POST", "/api/latest/fleet/users/api_only", createAPIOnlyUserRequest{
		Name: new(t.Name() + "_crossteam_api"),
		Fleets: &[]fleetsPayload{
			{ID: team1.ID, Role: fleet.RoleAdmin},
			{ID: team2.ID, Role: fleet.RoleAdmin},
		},
	}, http.StatusForbidden, &resp)

	// Sanity check: the team1 admin can still create a user scoped entirely to
	// their own team, so the fix doesn't over-restrict the legitimate path.
	resp = createUserResponse{}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", fleet.UserPayload{
		Name:     new(t.Name() + "_team1only"),
		Email:    new(t.Name() + "_team1only@example.com"),
		Password: new(test.GoodPassword),
		Teams: &[]fleet.UserTeam{
			{Team: *team1, Role: fleet.RoleAdmin},
		},
	}, http.StatusOK, &resp)
	require.NotNil(t, resp.User)
}

func (s *integrationEnterpriseTestSuite) TestTeamAdminCannotEscalateViaEmptyGlobalRole() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	otherTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_other"})
	require.NoError(t, err)
	defer func() {
		s.token = s.getTestAdminToken()
		require.NoError(t, s.ds.DeleteTeam(ctx, team.ID))
		require.NoError(t, s.ds.DeleteTeam(ctx, otherTeam.ID))
	}()
	otherTeamLabel, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                t.Name() + "_other_label",
		Query:               "SELECT 1",
		TeamID:              &otherTeam.ID,
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)

	teamAdminEmail := t.Name() + "_team_admin@example.com"
	teamAdmin := &fleet.User{
		Name:  teamAdminEmail,
		Email: teamAdminEmail,
		Teams: []fleet.UserTeam{{Team: *team, Role: fleet.RoleAdmin}},
	}
	require.NoError(t, teamAdmin.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(ctx, teamAdmin)
	require.NoError(t, err)

	s.token = s.getTestToken(teamAdmin.Email, test.GoodPassword)

	var createResp createUserResponse
	s.DoJSON("POST", "/api/latest/fleet/users/admin", map[string]any{
		"name":                        "svc",
		"email":                       t.Name() + "_svc@example.com",
		"password":                    test.GoodPassword,
		"global_role":                 "",
		"teams":                       []map[string]any{{"id": team.ID, "role": fleet.RoleAdmin}},
		"api_only":                    true,
		"admin_forced_password_reset": false,
	}, http.StatusUnprocessableEntity, &createResp)
	_, err = s.ds.UserByEmail(ctx, t.Name()+"_svc@example.com")
	require.True(t, fleet.IsNotFound(err))

	createResp = createUserResponse{}
	s.DoJSON("POST", "/api/latest/fleet/users/api_only", map[string]any{
		"name":        t.Name() + "_svc_api",
		"global_role": "",
		"fleets":      []map[string]any{{"id": team.ID, "role": fleet.RoleAdmin}},
	}, http.StatusUnprocessableEntity, &createResp)
	require.Nil(t, createResp.User)
	require.Empty(t, createResp.Token)

	var selfModResp modifyUserResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", teamAdmin.ID), map[string]any{
		"global_role": "",
		"teams":       []map[string]any{{"id": team.ID, "role": fleet.RoleAdmin}},
	}, http.StatusUnprocessableEntity, &selfModResp)

	plantedEmail := t.Name() + "_planted@example.com"
	planted := &fleet.User{
		Name:  plantedEmail,
		Email: plantedEmail,
		Teams: []fleet.UserTeam{{Team: *team, Role: fleet.RoleAdmin}},
	}
	require.NoError(t, planted.SetPassword(test.GoodPassword, 10, 10))
	planted, err = s.ds.NewUser(ctx, planted)
	require.NoError(t, err)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE users SET global_role = '' WHERE id = ?`, planted.ID)
		return err
	})

	s.token = s.getTestToken(planted.Email, test.GoodPassword)

	var labelsResp fleet.ListLabelsResponse
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusForbidden, &labelsResp, "team_id", fmt.Sprint(otherTeam.ID))
	labelsResp = fleet.ListLabelsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &labelsResp)
	for _, lbl := range labelsResp.Labels {
		require.NotEqual(t, otherTeamLabel.ID, lbl.ID)
	}

	var meResp getUserResponse
	s.DoJSON("GET", "/api/latest/fleet/me", nil, http.StatusOK, &meResp)
	require.Len(t, meResp.AvailableTeams, 1)
	require.Equal(t, team.ID, meResp.AvailableTeams[0].ID)

	globalAdminEmail := t.Name() + "_global_admin@example.com"
	globalAdmin := &fleet.User{
		Name:       globalAdminEmail,
		Email:      globalAdminEmail,
		GlobalRole: new(fleet.RoleAdmin),
	}
	require.NoError(t, globalAdmin.SetPassword(test.GoodPassword, 10, 10))
	globalAdmin, err = s.ds.NewUser(ctx, globalAdmin)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, s.ds.DeleteUser(ctx, globalAdmin.ID))
	}()

	// First step of demoting a global admin: getting them onto a team the caller administers.
	var teamUsersResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d/users", team.ID), map[string]any{
		"users": []map[string]any{{"id": globalAdmin.ID, "role": fleet.RoleObserver}},
	}, http.StatusForbidden, &teamUsersResp)
	globalAdmin, err = s.ds.UserByID(ctx, globalAdmin.ID)
	require.NoError(t, err)
	require.Empty(t, globalAdmin.Teams)

	// Even if a global admin ends up with a membership row on the caller's
	// team, a team admin must not be able to demote them through it.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `INSERT INTO user_teams (user_id, team_id, role) VALUES (?, ?, ?)`,
			globalAdmin.ID, team.ID, fleet.RoleObserver)
		return err
	})
	var modResp modifyUserResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", globalAdmin.ID), map[string]any{
		"teams": []map[string]any{{"id": team.ID, "role": fleet.RoleObserver}},
	}, http.StatusForbidden, &modResp)
	globalAdmin, err = s.ds.UserByID(ctx, globalAdmin.ID)
	require.NoError(t, err)
	require.NotNil(t, globalAdmin.GlobalRole)
	require.Equal(t, fleet.RoleAdmin, *globalAdmin.GlobalRole)

	modResp = modifyUserResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", planted.ID), map[string]any{
		"global_role": fleet.RoleAdmin,
	}, http.StatusForbidden, &modResp)

	planted, err = s.ds.UserByID(ctx, planted.ID)
	require.NoError(t, err)
	require.NotNil(t, planted.GlobalRole)
	require.Empty(t, *planted.GlobalRole)
}

func (s *integrationEnterpriseTestSuite) TestSSOJITProvisioning() {
	t := s.T()

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"server_settings": {
			"server_url": "https://localhost:8080"
		},
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"idp_name": "SimpleSAML",
			"metadata_url": "%s",
			"enable_jit_provisioning": false
		}
	}`, testSAMLIDPMetadataURL)), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.False(t, acResp.SSOSettings.EnableJITProvisioning)

	// users can't be created if SSO is disabled
	body := s.LoginSSOUser("sso_user", "user123#")
	require.Contains(t, body, "/login?status=account_invalid")
	// ensure theresn't a user in the DB
	_, err := s.ds.UserByEmail(context.Background(), "sso_user@example.com")
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// If enable_jit_provisioning is enabled Roles won't be updated for existing SSO users.

	// enable JIT provisioning
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"idp_name": "SimpleSAML",
			"metadata_url": "%s",
			"enable_jit_provisioning": true
		}
	}`, testSAMLIDPMetadataURL)), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.True(t, acResp.SSOSettings.EnableJITProvisioning)

	// a new user is created and redirected accordingly
	body = s.LoginSSOUser("sso_user", "user123#")
	// a successful redirect has this content
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user, err := s.ds.UserByEmail(context.Background(), "sso_user@example.com")
	require.NoError(t, err)
	require.Equal(t, "sso_user@example.com", user.Email)

	// a new activity item is created
	activitiesResp := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	require.NotEmpty(t, activitiesResp.Activities)
	require.Condition(t, func() bool {
		for _, a := range activitiesResp.Activities {
			if (a.Type == fleet.ActivityTypeUserAddedBySSO{}.ActivityName()) && *a.ActorEmail == "sso_user@example.com" {
				return true
			}
		}
		return false
	})

	// Test that roles are not updated for an existing user when SSO attributes are not set.

	// Change role to global admin first.
	user.GlobalRole = new("admin")
	err = s.ds.SaveUser(context.Background(), user)
	require.NoError(t, err)
	// Login should NOT change the role to the default (global observer) because SSO attributes
	// are not set for this user (see ../../tools/saml/users.php).
	body = s.LoginSSOUser("sso_user", "user123#")
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user, err = s.ds.UserByEmail(context.Background(), "sso_user@example.com")
	require.NoError(t, err)
	require.NotNil(t, user.GlobalRole)
	require.Equal(t, *user.GlobalRole, "admin")
	require.Equal(t, "SSO User 1", user.Name)

	// A user with pre-configured roles can be created
	// see `tools/saml/users.php` for details.
	body = s.LoginSSOUser("sso_user_3_global_admin", "user123#")
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user3, err := s.ds.UserByEmail(context.Background(), "sso_user_3_global_admin@example.com")
	require.NoError(t, err)
	require.Equal(t, "sso_user_3_global_admin@example.com", user3.Email)
	require.Equal(t, "SSO User 3", user3.Name)
	require.NotNil(t, user3.GlobalRole)
	require.Equal(t, fleet.RoleAdmin, *user3.GlobalRole)

	// Test that roles are updated for an existing user when SSO attributes are set.

	// Change role to global maintainer first.
	user3, err = s.ds.UserByEmail(context.Background(), "sso_user_3_global_admin@example.com")
	require.NoError(t, err)
	require.Equal(t, "sso_user_3_global_admin@example.com", user3.Email)
	user3.GlobalRole = new("maintainer")
	err = s.ds.SaveUser(context.Background(), user3)
	require.NoError(t, err)

	// Login should change the role to the configured role in the SSO attributes (global admin).
	body = s.LoginSSOUser("sso_user_3_global_admin", "user123#")
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user3, err = s.ds.UserByEmail(context.Background(), "sso_user_3_global_admin@example.com")
	require.NoError(t, err)
	require.Equal(t, "sso_user_3_global_admin@example.com", user3.Email)
	require.Equal(t, "SSO User 3", user3.Name)
	require.NotNil(t, user3.GlobalRole)
	require.Equal(t, fleet.RoleAdmin, *user3.GlobalRole)

	// We cannot use NewTeam and must use adhoc SQL because the teams.id is
	// auto-incremented and other tests cause it to be different than what we need (ID=1).
	var execErr error
	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, execErr = db.ExecContext(context.Background(), `INSERT INTO teams (id, name) VALUES (1, 'Foobar'), (2, 'Hollatchaboi') ON DUPLICATE KEY UPDATE name = VALUES(name);`)
		return execErr
	})
	require.NoError(t, execErr)

	// Create a team for the test below.
	_, err = s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team_" + t.Name(),
		Description: "desc team_" + t.Name(),
	})
	require.NoError(t, err)

	// A user with pre-configured roles can be created,
	// see `tools/saml/users.php` for details.
	body = s.LoginSSOUser("sso_user_4_team_maintainer", "user123#")
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user4, err := s.ds.UserByEmail(context.Background(), "sso_user_4_team_maintainer@example.com")
	require.NoError(t, err)
	require.Equal(t, "sso_user_4_team_maintainer@example.com", user4.Email)
	require.Equal(t, "SSO User 4", user4.Name)
	require.Nil(t, user4.GlobalRole)
	require.Len(t, user4.Teams, 2)
	require.Equal(t, uint(1), user4.Teams[0].ID)
	require.Equal(t, fleet.RoleMaintainer, user4.Teams[0].Role)
	require.Equal(t, uint(2), user4.Teams[1].ID)
	require.Equal(t, fleet.RoleObserver, user4.Teams[1].Role)

	// A user with pre-configured roles can be created,
	// see `tools/saml/users.php` for details.
	// This user has the following configuration (the last two are ignored by Fleet):
	//	- 'FLEET_JIT_USER_ROLE_TEAM_1' => 'admin',
	//	- 'FLEET_JIT_USER_ROLE_GLOBAL' => 'null',
	//	- 'FLEET_JIT_USER_ROLE_TEAM_2' => 'null',
	body = s.LoginSSOUser("sso_user_5_team_admin", "user123#")
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user5, err := s.ds.UserByEmail(context.Background(), "sso_user_5_team_admin@example.com")
	require.NoError(t, err)
	assert.Equal(t, "sso_user_5_team_admin@example.com", user5.Email)
	assert.Equal(t, "SSO User 5", user5.Name)
	require.Nil(t, user5.GlobalRole)
	require.Len(t, user5.Teams, 1)
	require.Equal(t, uint(1), user5.Teams[0].ID)
	require.Equal(t, fleet.RoleAdmin, user5.Teams[0].Role)

	// A user with pre-configured roles can be created,
	// see `tools/saml/users.php` for details.
	// This user has the following configuration (all ignored by Fleet thus added as global observer):
	//	- 'FLEET_JIT_USER_ROLE_GLOBAL' => 'null',
	//	- 'FLEET_JIT_USER_ROLE_TEAM_1' => 'null',
	body = s.LoginSSOUser("sso_user_6_global_observer", "user123#")
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user6, err := s.ds.UserByEmail(context.Background(), "sso_user_6_global_observer@example.com")
	require.NoError(t, err)
	assert.Equal(t, "sso_user_6_global_observer@example.com", user6.Email)
	assert.Equal(t, "SSO User 6", user6.Name)
	require.NotNil(t, user6.GlobalRole)
	require.Equal(t, fleet.RoleObserver, *user6.GlobalRole)

	// A user with a global technician role can be created via JIT provisioning,
	// see `tools/saml/users.php` for details.
	body = s.LoginSSOUser("sso_user_8_global_technician", "user123#")
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user8, err := s.ds.UserByEmail(context.Background(), "sso_user_8_global_technician@example.com")
	require.NoError(t, err)
	assert.Equal(t, "sso_user_8_global_technician@example.com", user8.Email)
	assert.Equal(t, "SSO User 8", user8.Name)
	require.NotNil(t, user8.GlobalRole)
	require.Equal(t, fleet.RoleTechnician, *user8.GlobalRole)
}

func (s *integrationEnterpriseTestSuite) TestSSOIdPInitiatedLogin() {
	t := s.T()

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
        "server_settings": {
          "server_url": "https://localhost:8080"
        },
		"sso_settings": {
			"enable_sso": true,
			"enable_jit_provisioning": true,
			"enable_sso_idp_login": true,
			"entity_id": "sso.test.com",
			"idp_name": "SimpleSAML",
			"metadata_url": "%s"
		}
	}`, testSAMLIDPMetadataURL)), http.StatusOK, &acResp)
	require.NotNil(t, acResp)

	body := s.LoginSSOUserIDPInitiated("sso_user2", "user123#", "sso.test.com")
	require.Contains(t, body, "Redirecting to Fleet at / ...")
}

func (s *integrationEnterpriseTestSuite) TestListAPIEndpoints() {
	t := s.T()
	defer func() { s.token = s.getTestAdminToken() }()

	// unauthenticated request is rejected
	s.DoRawNoAuth("GET", "/api/latest/fleet/rest_api", nil, http.StatusUnauthorized)

	// non-admin global roles are forbidden
	for _, role := range []string{fleet.RoleObserver, fleet.RoleObserverPlus, fleet.RoleMaintainer, fleet.RoleGitOps, fleet.RoleTechnician} {
		u := &fleet.User{
			Name:       "test " + role,
			Email:      role + "-api-endpoints@example.com",
			GlobalRole: new(role),
		}
		require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
		_, err := s.ds.NewUser(context.Background(), u)
		require.NoError(t, err)
		s.token = s.getTestToken(u.Email, test.GoodPassword)
		s.Do("GET", "/api/latest/fleet/rest_api", nil, http.StatusForbidden)
	}

	// non-admin team roles are forbidden
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "api-endpoints-test-team"})
	require.NoError(t, err)
	for _, role := range []string{fleet.RoleObserver, fleet.RoleObserverPlus, fleet.RoleMaintainer, fleet.RoleGitOps, fleet.RoleTechnician} {
		u := &fleet.User{
			Name:  "team " + role,
			Email: "team-" + role + "-api-endpoints@example.com",
			Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team.ID}, Role: role}},
		}
		require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
		_, err := s.ds.NewUser(context.Background(), u)
		require.NoError(t, err)
		s.token = s.getTestToken(u.Email, test.GoodPassword)
		s.Do("GET", "/api/latest/fleet/rest_api", nil, http.StatusForbidden)
	}

	// team admin is allowed
	teamAdmin := &fleet.User{
		Name:  "team admin",
		Email: "team-admin-api-endpoints@example.com",
		Teams: []fleet.UserTeam{{Team: fleet.Team{ID: team.ID}, Role: fleet.RoleAdmin}},
	}
	require.NoError(t, teamAdmin.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), teamAdmin)
	require.NoError(t, err)
	s.token = s.getTestToken(teamAdmin.Email, test.GoodPassword)
	s.Do("GET", "/api/latest/fleet/rest_api", nil, http.StatusOK)

	// restore admin token
	s.token = s.getTestAdminToken()

	// global admin can list API endpoints
	var resp listAPIEndpointsResponse
	s.DoJSON("GET", "/api/latest/fleet/rest_api", nil, http.StatusOK, &resp)
	require.NotEmpty(t, resp.APIEndpoints)
	require.NoError(t, resp.Err)
	require.NotEmpty(t, resp.APIEndpoints[0].Method)
	require.NotEmpty(t, resp.APIEndpoints[0].Path)
	require.NotEmpty(t, resp.APIEndpoints[0].DisplayName)
}

func (s *integrationEnterpriseTestSuite) TestCreateAPIOnlyUserPremium() {
	t := s.T()

	// Create a team to use for fleet-scoped assignments.
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: t.Name() + "_team"})
	require.NoError(t, err)

	type apiEndpoint struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	}
	type teamEntry struct {
		ID   uint   `json:"id"`
		Role string `json:"role"`
	}
	type createAPIOnlyUserResponse struct {
		User struct {
			ID           uint          `json:"id"`
			Name         string        `json:"name"`
			Email        string        `json:"email"`
			APIOnly      bool          `json:"api_only"`
			GlobalRole   *string       `json:"global_role"`
			APIEndpoints []apiEndpoint `json:"api_endpoints"`
			Teams        []teamEntry   `json:"teams"` //nolint:apiparamcheck // test helper; matches server response shape
		} `json:"user"`
		Token  string              `json:"token"`
		Err    string              `json:"error,omitempty"`
		Errors []map[string]string `json:"errors,omitempty"`
	}

	cases := []struct {
		name       string
		body       map[string]any
		wantStatus int
		verify     func(t *testing.T, resp createAPIOnlyUserResponse)
	}{
		// --- Validation still enforced under premium ---
		{
			name: "missing name",
			body: map[string]any{
				"fleets": []map[string]any{{"id": team.ID, "role": "observer"}},
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "neither global_role nor fleets",
			body:       map[string]any{"name": "Premium User"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "global_role and fleets together",
			body: map[string]any{
				"name":        "Premium User",
				"global_role": "observer",
				"fleets":      []map[string]any{{"id": team.ID, "role": "observer"}},
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid api_endpoint not in catalog",
			body: map[string]any{
				"name":        "Premium User",
				"global_role": "observer",
				"api_endpoints": []map[string]any{
					{"method": "GET", "path": "/api/v1/fleet/nonexistent/endpoint"},
				},
			},
			wantStatus: http.StatusUnprocessableEntity,
			verify: func(t *testing.T, resp createAPIOnlyUserResponse) {
				require.Len(t, resp.Errors, 1)
				require.Contains(t, resp.Errors[0]["reason"], "|GET|/api/v1/fleet/nonexistent/endpoint|")
			},
		},
		{
			name: "wildcard mixed with other entries",
			body: map[string]any{
				"name":        "Premium User",
				"global_role": "observer",
				"api_endpoints": []map[string]any{
					{"method": "*", "path": "*"},
					{"method": "GET", "path": "/api/v1/fleet/config"},
				},
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "allow only a limited number of api_endpoints",
			body: func() map[string]any {
				// One more than the catalog size to trigger the limit.
				eps := make([]map[string]any, len(apiendpoints.GetAPIEndpoints())+1)
				for i := range eps {
					eps[i] = map[string]any{"method": "GET", "path": fmt.Sprintf("/api/v1/fleet/path/%d", i)}
				}
				return map[string]any{
					"name":          "Premium User",
					"global_role":   "observer",
					"api_endpoints": eps,
				}
			}(),
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "nil api_endpoints grants full access",
			body: map[string]any{
				"name":        "Full Access API User",
				"global_role": "observer",
			},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, resp createAPIOnlyUserResponse) {
				require.NotEmpty(t, resp.Token)
				require.NotZero(t, resp.User.ID)
				require.True(t, resp.User.APIOnly)
				require.Empty(t, resp.User.APIEndpoints)
			},
		},
		{
			name: "global_role with specific api_endpoints",
			body: map[string]any{
				"name":        "Global API User",
				"global_role": "observer",
				"api_endpoints": []map[string]any{
					{"method": "GET", "path": "/api/v1/fleet/config"},
					{"method": "GET", "path": "/api/v1/fleet/version"},
				},
			},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, resp createAPIOnlyUserResponse) {
				require.NotEmpty(t, resp.Token)
				require.NotZero(t, resp.User.ID)
				require.Equal(t, "Global API User", resp.User.Name)
				require.NotEmpty(t, resp.User.Email)
				require.True(t, resp.User.APIOnly)
				require.NotNil(t, resp.User.GlobalRole)
				require.Equal(t, "observer", *resp.User.GlobalRole)
				require.Len(t, resp.User.APIEndpoints, 2)
			},
		},
		{
			name: "fleet-scoped assignment without api_endpoints grants full access",
			body: map[string]any{
				"name":   "Team API User",
				"fleets": []map[string]any{{"id": team.ID, "role": "observer"}},
			},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, resp createAPIOnlyUserResponse) {
				require.NotEmpty(t, resp.Token)
				require.NotZero(t, resp.User.ID)
				require.Equal(t, "Team API User", resp.User.Name)
				require.NotEmpty(t, resp.User.Email)
				require.True(t, resp.User.APIOnly)
				require.Len(t, resp.User.Teams, 1)
				require.Equal(t, team.ID, resp.User.Teams[0].ID)
				require.Equal(t, "observer", resp.User.Teams[0].Role)
				require.Empty(t, resp.User.APIEndpoints) // nil = full access
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

func (s *integrationEnterpriseTestSuite) TestModifyAPIOnlyUserPremium() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: t.Name() + "_team"})
	require.NoError(t, err)

	nonAPIUser := &fleet.User{
		Name:       "Non API User",
		Email:      "non-api-patch@example.com",
		GlobalRole: new(fleet.RoleObserver),
	}
	require.NoError(t, nonAPIUser.SetPassword(test.GoodPassword, 10, 10))
	nonAPIUser, err = s.ds.NewUser(context.Background(), nonAPIUser)
	require.NoError(t, err)

	type patchResp struct {
		User struct {
			ID           uint    `json:"id"`
			Name         string  `json:"name"`
			Email        string  `json:"email"`
			APIOnly      bool    `json:"api_only"`
			GlobalRole   *string `json:"global_role"`
			APIEndpoints []struct {
				Method string `json:"method"`
				Path   string `json:"path"`
			} `json:"api_endpoints"`
			Teams []struct {
				ID   uint   `json:"id"`
				Role string `json:"role"`
			} `json:"teams"` //nolint:apiparamcheck // test helper; matches server response shape
		} `json:"user"`
		Err string `json:"error,omitempty"`
	}

	var createResp struct {
		User struct {
			ID uint `json:"id"`
		} `json:"user"`
		Token string `json:"token"`
	}
	s.DoJSON("POST", "/api/latest/fleet/users/api_only", map[string]any{
		"name":        "Patch Target",
		"global_role": "observer",
		"api_endpoints": []map[string]any{
			{"method": "GET", "path": "/api/v1/fleet/version"},
		},
	}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)
	uid := createResp.User.ID
	patchURL := fmt.Sprintf("/api/latest/fleet/users/api_only/%d", uid)
	nonAPIURL := fmt.Sprintf("/api/latest/fleet/users/api_only/%d", nonAPIUser.ID)

	// Cases are order-dependent: success cases mutate the same user sequentially.
	cases := []struct {
		name       string
		url        string
		body       map[string]any
		wantStatus int
		verify     func(t *testing.T, resp patchResp)
	}{
		// --- Validation errors ---
		{
			name:       "nonexistent user",
			url:        "/api/latest/fleet/users/api_only/999999",
			body:       map[string]any{"name": "Ghost"},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "non-API-only user",
			url:        nonAPIURL,
			body:       map[string]any{"name": "New Name"},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid api_endpoint not in catalog",
			url:  patchURL,
			body: map[string]any{
				"api_endpoints": []map[string]any{
					{"method": "GET", "path": "/api/v1/fleet/nonexistent/endpoint"},
				},
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "wildcard mixed with other entries",
			url:  patchURL,
			body: map[string]any{
				"api_endpoints": []map[string]any{
					{"method": "*", "path": "*"},
					{"method": "GET", "path": "/api/v1/fleet/version"},
				},
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "more than 100 api_endpoints",
			url:  patchURL,
			body: func() map[string]any {
				eps := make([]map[string]any, 101)
				for i := range eps {
					eps[i] = map[string]any{"method": "GET", "path": fmt.Sprintf("/api/v1/fleet/path/%d", i)}
				}
				return map[string]any{"api_endpoints": eps}
			}(),
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "empty api_endpoints",
			url:        patchURL,
			body:       map[string]any{"api_endpoints": []map[string]any{}},
			wantStatus: http.StatusUnprocessableEntity,
		},
		// --- Successful mutations (order matters — each one builds on prior state) ---
		{
			name:       "update name",
			url:        patchURL,
			body:       map[string]any{"name": "Patched Name"},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, resp patchResp) {
				require.Equal(t, "Patched Name", resp.User.Name)
				require.True(t, resp.User.APIOnly)
			},
		},
		{
			name:       "update global_role",
			url:        patchURL,
			body:       map[string]any{"global_role": "admin"},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, resp patchResp) {
				require.NotNil(t, resp.User.GlobalRole)
				require.Equal(t, "admin", *resp.User.GlobalRole)
			},
		},
		{
			name: "global_role and fleets together",
			url:  patchURL,
			body: map[string]any{
				"global_role": "observer",
				"fleets":      []map[string]any{{"id": team.ID, "role": "observer"}},
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "assign to fleet clears global_role",
			url:  patchURL,
			body: map[string]any{
				"fleets": []map[string]any{{"id": team.ID, "role": "observer"}},
			},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, resp patchResp) {
				require.Len(t, resp.User.Teams, 1)
				require.Equal(t, team.ID, resp.User.Teams[0].ID)
				require.Equal(t, "observer", resp.User.Teams[0].Role)
				require.Nil(t, resp.User.GlobalRole, "global_role should be cleared when switching to fleet role")
			},
		},
		{
			name: "update api_endpoints to specific endpoints",
			url:  patchURL,
			body: map[string]any{
				"api_endpoints": []map[string]any{
					{"method": "GET", "path": "/api/v1/fleet/config"},
					{"method": "GET", "path": "/api/v1/fleet/version"},
				},
			},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, resp patchResp) {
				require.Len(t, resp.User.APIEndpoints, 2)
			},
		},
		{
			name:       "null api_endpoints resets to full access",
			url:        patchURL,
			body:       map[string]any{"api_endpoints": nil},
			wantStatus: http.StatusOK,
			verify: func(t *testing.T, resp patchResp) {
				require.Empty(t, resp.User.APIEndpoints)
			},
		},
		{
			name:       "empty array is invalid",
			url:        patchURL,
			body:       map[string]any{"api_endpoints": []map[string]any{}},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var resp patchResp
			s.DoJSON("PATCH", tc.url, tc.body, tc.wantStatus, &resp)
			if tc.verify != nil {
				tc.verify(t, resp)
			}
		})
	}
}

func (s *integrationEnterpriseTestSuite) TestGetUserReturnsAPIEndpoints() {
	t := s.T()

	// Shared response type for GET /users/:id and items in GET /users.
	type userJSON struct {
		ID           uint `json:"id"`
		APIOnly      bool `json:"api_only"`
		APIEndpoints []struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"api_endpoints"`
	}
	type getUserResp struct {
		User userJSON `json:"user"`
		Err  string   `json:"error,omitempty"`
	}
	type listUsersResp struct {
		Users []userJSON `json:"users"`
		Err   string     `json:"error,omitempty"`
	}

	// Create an API-only user with specific catalog endpoints.
	var createResp getUserResp
	s.DoJSON("POST", "/api/latest/fleet/users/api_only", map[string]any{
		"name":        "GET Endpoint User",
		"global_role": "observer",
		"api_endpoints": []map[string]any{
			{"method": "GET", "path": "/api/v1/fleet/config"},
			{"method": "GET", "path": "/api/v1/fleet/version"},
		},
	}, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)
	apiUserID := createResp.User.ID

	// GET /users/:id should include api_endpoints.
	var getResp getUserResp
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", apiUserID), nil, http.StatusOK, &getResp)
	require.True(t, getResp.User.APIOnly)
	require.Len(t, getResp.User.APIEndpoints, 2)

	// GET /users should include api_endpoints for the API-only user.
	var listResp listUsersResp
	s.DoJSON("GET", "/api/latest/fleet/users", nil, http.StatusOK, &listResp)
	var found *userJSON
	for i := range listResp.Users {
		if listResp.Users[i].ID == apiUserID {
			found = &listResp.Users[i]
			break
		}
	}
	require.NotNil(t, found, "API-only user should appear in list")
	require.Len(t, found.APIEndpoints, 2)

	// A regular (non-API-only) user should have no api_endpoints in either response.
	// Create a fresh regular user rather than relying on seed data.
	var regularUserResp getUserResp
	s.DoJSON("POST", "/api/latest/fleet/users/admin", fleet.UserPayload{
		Name:       new("Regular User"),
		Email:      new("regular-user-get-test@example.com"),
		Password:   new(test.GoodPassword),
		GlobalRole: new(fleet.RoleObserver),
	}, http.StatusOK, &regularUserResp)
	require.NotZero(t, regularUserResp.User.ID)

	var getAdminResp getUserResp
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", regularUserResp.User.ID), nil, http.StatusOK, &getAdminResp)
	require.False(t, getAdminResp.User.APIOnly)
	require.Empty(t, getAdminResp.User.APIEndpoints)

	var getUsersResp listUsersResp
	s.DoJSON("GET", "/api/latest/fleet/users", nil, http.StatusOK, &getUsersResp)
	var foundRegular *userJSON
	for i := range getUsersResp.Users {
		if getUsersResp.Users[i].ID == regularUserResp.User.ID {
			foundRegular = &getUsersResp.Users[i]
			break
		}
	}
	require.NotNil(t, foundRegular, "regular user should appear in list")
	require.False(t, foundRegular.APIOnly)
	require.Empty(t, foundRegular.APIEndpoints)
}

func (s *integrationEnterpriseTestSuite) TestAPIOnlyUserEndpointMiddleware() {
	t := s.T()

	defer func() { s.token = s.getTestAdminToken() }()

	createAPIOnlyUser := func(name string, endpoints []map[string]any) string {
		prev := s.token
		s.token = s.getTestAdminToken()
		defer func() { s.token = prev }()

		body := map[string]any{
			"name":        name,
			"global_role": "observer",
		}
		if endpoints != nil {
			body["api_endpoints"] = endpoints
		}
		var createResp struct {
			Token string `json:"token"`
		}
		s.DoJSON("POST", "/api/latest/fleet/users/api_only", body, http.StatusOK, &createResp)
		require.NotEmpty(t, createResp.Token)
		return createResp.Token
	}

	// With no endpoint restrictions the api_only middleware skips entirely, so
	// the user can reach any registered route — gated only by role-based authz
	// further down the chain.
	t.Run("no restrictions skips the middleware", func(t *testing.T) {
		s.token = createAPIOnlyUser("api-only-mw-no-restrictions", nil)

		s.Do("GET", "/api/latest/fleet/version", nil, http.StatusOK)
		s.Do("GET", "/api/latest/fleet/config", nil, http.StatusOK)
		s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusOK)
	})

	// requireRestrictionDeniedBody asserts the 403 body carries the distinct
	// endpoint-restriction message, distinguishing it from role-based denials.
	requireRestrictionDeniedBody := func(t *testing.T, res *http.Response) {
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), auth.EndpointRestrictionDeniedMessage)
	}

	// For api-only users with restrictions, requests to paths not in the API
	// endpoint catalog are rejected by the middleware before reaching the
	// service layer.
	t.Run("non-catalog path is rejected for restricted users", func(t *testing.T) {
		s.token = createAPIOnlyUser("api-only-mw-non-catalog-restricted", []map[string]any{
			{"method": "GET", "path": "/api/v1/fleet/version"},
		})
		res := s.Do("PATCH", "/api/latest/fleet/users/api_only/1", map[string]any{"name": "x"}, http.StatusForbidden)
		requireRestrictionDeniedBody(t, res)
	})

	// With endpoint restrictions, only explicitly allowed endpoints are reachable.
	t.Run("endpoint restrictions limit access to the allowed list", func(t *testing.T) {
		s.token = createAPIOnlyUser("api-only-mw-restricted", []map[string]any{
			{"method": "GET", "path": "/api/v1/fleet/version"},
		})

		// The only allowed endpoint returns 200.
		s.Do("GET", "/api/latest/fleet/version", nil, http.StatusOK)

		// These are in the catalog but not in the user's allow list.
		res := s.Do("GET", "/api/latest/fleet/config", nil, http.StatusForbidden)
		requireRestrictionDeniedBody(t, res)
		res = s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusForbidden)
		requireRestrictionDeniedBody(t, res)
	})

	// Non-api-only users must not be affected by the middleware at all.
	t.Run("non-api-only user is unaffected", func(t *testing.T) {
		s.token = s.getTestAdminToken()

		s.Do("GET", "/api/latest/fleet/version", nil, http.StatusOK)
		s.Do("GET", "/api/latest/fleet/config", nil, http.StatusOK)
		s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusOK)
	})
}

// TestAPIOnlyUserCanReachGitOpsEndpoints verifies that the API endpoint
// catalog includes every route invoked by fleetctl gitops and
// fleetctl generate-gitops. The test runs as an api-only admin user so any
// 403 observed here comes from the api_only middleware (catalog miss), never
// from service-level authz. Regression test for #44279.
func (s *integrationEnterpriseTestSuite) TestAPIOnlyUserCanReachGitOpsEndpoints() {
	t := s.T()
	defer func() { s.token = s.getTestAdminToken() }()

	assertNot403 := func(verb, path string, body any) {
		t.Helper()
		var raw []byte
		if body != nil {
			j, err := json.Marshal(body)
			require.NoError(t, err)
			raw = j
		}
		req, err := http.NewRequest(verb, s.server.URL+path, bytes.NewReader(raw))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+s.token)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.NotEqualf(t, http.StatusForbidden, resp.StatusCode,
			"%s %s returned 403 for an api-only admin user; the route is likely missing from server/api_endpoints/api_endpoints.yml",
			verb, path)
	}

	// Create an api-only admin (no endpoint restrictions). Admins pass every
	// service-level authz check, so the only remaining source of 403 is the
	// api_only middleware's catalog check — exactly what we want to verify.
	var createResp struct {
		Token string `json:"token"`
	}
	s.DoJSON("POST", "/api/latest/fleet/users/api_only", map[string]any{
		"name":        "api-only-admin-gitops-catalog",
		"global_role": fleet.RoleAdmin,
	}, http.StatusOK, &createResp)
	require.NotEmpty(t, createResp.Token)
	s.token = createResp.Token

	// Endpoints invoked by fleetctl gitops / generate-gitops. Status codes vary
	// per endpoint based on payload validity; we only care that 403 never comes
	// from the catalog check.
	emptySpecs := map[string]any{"specs": []any{}}
	assertNot403("GET", "/api/latest/fleet/me", nil)
	assertNot403("GET", "/api/latest/fleet/config", nil)
	assertNot403("GET", "/api/latest/fleet/version", nil)
	assertNot403("GET", "/api/latest/fleet/fleets", nil)
	assertNot403("GET", "/api/latest/fleet/spec/labels", nil)
	assertNot403("POST", "/api/latest/fleet/spec/labels", emptySpecs)
	assertNot403("GET", "/api/latest/fleet/spec/enroll_secret", nil)
	assertNot403("GET", "/api/latest/fleet/spec/certificate_authorities", nil)
	assertNot403("POST", "/api/latest/fleet/spec/certificate_authorities", emptySpecs)
	assertNot403("GET", "/api/latest/fleet/abm_tokens/count", nil)
	assertNot403("POST", "/api/latest/fleet/spec/policies", emptySpecs)
	assertNot403("POST", "/api/latest/fleet/spec/reports", emptySpecs)
	assertNot403("POST", "/api/latest/fleet/spec/fleets", emptySpecs)
	assertNot403("PUT", "/api/latest/fleet/spec/secret_variables", map[string]any{"secret_variables": []any{}})
	assertNot403("GET", "/api/latest/fleet/policies", nil)
	assertNot403("GET", "/api/latest/fleet/configuration_profiles", nil)
	assertNot403("GET", "/api/latest/fleet/scripts", nil)
	assertNot403("GET", "/api/latest/fleet/software/titles", nil)
	assertNot403("GET", "/api/latest/fleet/software/fleet_maintained_apps", nil)
	assertNot403("GET", "/api/latest/fleet/setup_experience/script", nil)
	assertNot403("GET", "/api/latest/fleet/vpp_tokens", nil)
	assertNot403("GET", "/api/latest/fleet/certificates", nil)
}

func (s *integrationEnterpriseTestSuite) TestAPIOnlyGitOpsUserWithEndpointRestrictions() {
	t := s.T()
	defer func() { s.token = s.getTestAdminToken() }()

	allowedEndpoints := []map[string]any{
		{"method": "GET", "path": "/api/v1/fleet/me"},
		{"method": "GET", "path": "/api/v1/fleet/config"},
		{"method": "GET", "path": "/api/v1/fleet/spec/labels"},
		{"method": "GET", "path": "/api/v1/fleet/abm_tokens/count"},
		{"method": "POST", "path": "/api/v1/fleet/spec/policies"},
	}

	var createResp struct {
		Token string `json:"token"`
	}
	s.DoJSON("POST", "/api/latest/fleet/users/api_only", map[string]any{
		"name":          "api-only-gitops-restricted",
		"global_role":   fleet.RoleGitOps,
		"api_endpoints": allowedEndpoints,
	}, http.StatusOK, &createResp)
	require.NotEmpty(t, createResp.Token)
	s.token = createResp.Token

	// Allowed endpoints reach the handler.
	s.Do("GET", "/api/latest/fleet/me", nil, http.StatusOK)
	s.Do("GET", "/api/latest/fleet/config", nil, http.StatusOK)
	s.Do("GET", "/api/latest/fleet/spec/labels", nil, http.StatusOK)
	s.Do("GET", "/api/latest/fleet/abm_tokens/count", nil, http.StatusOK)
	// Apply a real policy spec; an empty specs list short-circuits before authz
	// in checkPolicySpecAuthorization and surfaces as 500, which would mask the
	// middleware behavior we want to verify.
	s.Do("POST", "/api/latest/fleet/spec/policies", map[string]any{
		"specs": []map[string]any{
			{"name": t.Name() + "-policy", "query": "SELECT 1;", "platform": "darwin"},
		},
	}, http.StatusOK)

	// Other gitops endpoints are in the catalog but not in the allow list, so
	// the middleware rejects them with 403.
	s.Do("GET", "/api/latest/fleet/version", nil, http.StatusForbidden)
	s.Do("GET", "/api/latest/fleet/fleets", nil, http.StatusForbidden)
	s.Do("POST", "/api/latest/fleet/spec/labels", map[string]any{"specs": []any{}}, http.StatusForbidden)
	s.Do("POST", "/api/latest/fleet/spec/reports", map[string]any{"specs": []any{}}, http.StatusForbidden)
	s.Do("POST", "/api/latest/fleet/spec/fleets", map[string]any{"specs": []any{}}, http.StatusForbidden)
}
