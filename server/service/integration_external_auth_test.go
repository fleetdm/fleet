package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// integrationExternalAuthTestSuite runs on Fleet Free (no license), which is the
// tier external identity authentication targets.
type integrationExternalAuthTestSuite struct {
	withServer
	suite.Suite
}

func TestIntegrationsExternalAuth(t *testing.T) {
	testingSuite := new(integrationExternalAuthTestSuite)
	testingSuite.withDS.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

func (s *integrationExternalAuthTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationExternalAuthTestSuite")
	cfg := config.TestConfig()
	cfg.Auth.JWTIssuers = []config.JWTIssuerConfig{{
		Name: "github", Issuer: "https://token.actions.githubusercontent.com", Audience: "fleet",
		SubjectClaim: "sub", AllowedAlgorithms: []string{"RS256"}, MaxTokenLifetime: 15 * time.Minute,
	}}
	users, server := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{
		FleetConfig: &cfg,
		Pool:        redistest.SetupRedis(s.T(), "integration_external_auth", false, false, false),
		DBConns:     s.dbConns,
	})
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
}

func (s *integrationExternalAuthTestSuite) TearDownTest() {
	s.token = s.getTestAdminToken()
}

func (s *integrationExternalAuthTestSuite) createAPIOnlyObserver(name string) (fleet.User, string) {
	var resp struct {
		User  fleet.User `json:"user"`
		Token string     `json:"token"`
	}
	s.DoJSON("POST", "/api/latest/fleet/users/api_only", map[string]any{
		"name": name, "global_role": fleet.RoleObserver,
	}, http.StatusOK, &resp)
	require.NotZero(s.T(), resp.User.ID)
	return resp.User, resp.Token
}

func (s *integrationExternalAuthTestSuite) TestExternalIdentitiesManagement() {
	t := s.T()
	bot, _ := s.createAPIOnlyObserver("bot")
	other, _ := s.createAPIOnlyObserver("other-bot")
	botID, otherID := bot.ID, other.ID
	path := fmt.Sprintf("/api/latest/fleet/users/api_only/%d/external_identities", botID)

	gh := func(sub string) map[string]string { return map[string]string{"issuer": "github", "subject": sub} }
	var resp externalIdentitiesResponse

	s.DoJSON("PUT", path, map[string]any{"external_identities": []any{gh("a"), gh("b")}}, http.StatusOK, &resp)
	require.Equal(t, []fleet.ExternalIdentity{{Issuer: "github", Subject: "a"}, {Issuer: "github", Subject: "b"}}, resp.ExternalIdentities)
	s.lastActivityOfTypeMatches("added_user_external_identity",
		fmt.Sprintf(`{"user_id": %d, "user_name": "bot", "user_email": %q, "issuer": "github", "subject": "b"}`, botID, bot.Email), 0)

	s.DoJSON("GET", path, nil, http.StatusOK, &resp)
	require.Len(t, resp.ExternalIdentities, 2)

	// A no-op PUT does not add activities.
	before := s.lastActivityOfTypeMatches("added_user_external_identity", "", 0)
	s.DoJSON("PUT", path, map[string]any{"external_identities": []any{gh("b"), gh("a")}}, http.StatusOK, &resp)
	require.Equal(t, before, s.lastActivityOfTypeMatches("added_user_external_identity", "", 0))

	// An identity can only be bound to one user.
	s.Do("PUT", fmt.Sprintf("/api/latest/fleet/users/api_only/%d/external_identities", otherID),
		map[string]any{"external_identities": []any{gh("a")}}, http.StatusConflict)

	s.Do("PUT", path, map[string]any{"external_identities": []any{map[string]string{"issuer": "unknown", "subject": "a"}}}, http.StatusUnprocessableEntity)
	s.Do("PUT", fmt.Sprintf("/api/latest/fleet/users/api_only/%d/external_identities", s.users[TestAdminUserEmail].ID),
		map[string]any{"external_identities": []any{gh("x")}}, http.StatusUnprocessableEntity)
	s.Do("PUT", "/api/latest/fleet/users/api_only/999999/external_identities",
		map[string]any{"external_identities": []any{gh("x")}}, http.StatusNotFound)

	// Clearing removes every binding and records one activity each.
	s.DoJSON("PUT", path, map[string]any{"external_identities": []any{}}, http.StatusOK, &resp)
	require.Empty(t, resp.ExternalIdentities)
	var acts listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &acts, "order_key", "id", "order_direction", "desc", "per_page", "10")
	var deleted int
	for _, a := range acts.Activities {
		if a.Type == "deleted_user_external_identity" {
			deleted++
			var d fleet.ActivityTypeDeletedUserExternalIdentity
			require.NoError(t, json.Unmarshal(*a.Details, &d))
			require.Equal(t, botID, d.UserID)
		}
	}
	require.Equal(t, 2, deleted)

	// Non-admins cannot read or write bindings.
	for _, email := range []string{TestMaintainerUserEmail, TestObserverUserEmail} {
		s.token = s.getTestToken(email, test.GoodPassword)
		s.Do("GET", path, nil, http.StatusForbidden)
		s.Do("PUT", path, map[string]any{"external_identities": []any{gh("z")}}, http.StatusForbidden)
	}
	s.token = s.getTestAdminToken()
	s.DoRawNoAuth("GET", path, nil, http.StatusUnauthorized)
}
