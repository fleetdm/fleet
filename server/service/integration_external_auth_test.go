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
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth/externalauth/externalauthtest"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// integrationExternalAuthTestSuite runs on Fleet Free (no license), which is the
// tier external identity authentication targets.
type integrationExternalAuthTestSuite struct {
	withServer
	suite.Suite
	issuer *externalauthtest.Issuer
}

func TestIntegrationsExternalAuth(t *testing.T) {
	testingSuite := new(integrationExternalAuthTestSuite)
	testingSuite.withDS.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

func (s *integrationExternalAuthTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationExternalAuthTestSuite")
	s.issuer = externalauthtest.NewIssuer(s.T())
	cfg := config.TestConfig()
	cfg.Auth.JWTIssuers = []config.JWTIssuerConfig{{
		Name: "github", Issuer: s.issuer.URL(), Audience: "fleet",
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
	// The UI offers these names when adding a binding.
	require.Equal(t, []string{"github"}, resp.Issuers)

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

// TestExternalJWTAuth is the core QA flow from the design (section 12.3), on Fleet Free.
func (s *integrationExternalAuthTestSuite) TestExternalJWTAuth() {
	t := s.T()
	ctx := t.Context()
	observer, sessionToken := s.createAPIOnlyObserver("posture-reader")
	bind := func(userID uint, subs ...string) {
		ids := []map[string]string{}
		for _, sub := range subs {
			ids = append(ids, map[string]string{"issuer": "github", "subject": sub})
		}
		s.token = s.getTestAdminToken()
		s.Do("PUT", fmt.Sprintf("/api/latest/fleet/users/api_only/%d/external_identities", userID),
			map[string]any{"external_identities": ids}, http.StatusOK)
	}
	jwtFor := func(sub string) string { return s.issuer.Token(t, sub, "fleet", 5*time.Minute) }

	// Steps 3-4: bind, then a read endpoint succeeds with a minted token and no stored secret.
	bind(observer.ID, "repo:acme/posture:ref:refs/heads/main")
	s.token = jwtFor("repo:acme/posture:ref:refs/heads/main")
	s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusOK)
	var me getUserResponse
	s.DoJSON("GET", "/api/latest/fleet/me", nil, http.StatusOK, &me)
	require.Equal(t, observer.ID, me.User.ID)

	// Step 5: the observer role still applies.
	s.Do("POST", "/api/latest/fleet/labels", map[string]any{"name": "jwt-observer-label", "query": "select 1"}, http.StatusForbidden)

	// Logging out has no session to end.
	s.Do("POST", "/api/latest/fleet/logout", nil, http.StatusUnprocessableEntity)

	// Step 6: activities are attributed to the API-only user (a maintainer, so it can write).
	maint, _ := s.createAPIOnlyUser("label-writer", fleet.RoleMaintainer)
	bind(maint.ID, "label-writer-sub")
	s.token = jwtFor("label-writer-sub")
	s.Do("POST", "/api/latest/fleet/labels", map[string]any{"name": "jwt-maintainer-label", "query": "select 1"}, http.StatusOK)
	s.token = s.getTestAdminToken()
	var acts listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &acts, "order_key", "id", "order_direction", "desc", "per_page", "5")
	var found bool
	for _, a := range acts.Activities {
		if a.Type == "created_label" {
			require.NotNil(t, a.ActorID)
			require.Equal(t, maint.ID, *a.ActorID)
			found = true
			break
		}
	}
	require.True(t, found, "created_label activity not found")

	// Unbound subjects, wrong audiences and other issuers are rejected.
	// Subjects match byte-for-byte: a branch named MAIN or maín is not main.
	for _, lookalike := range []string{"repo:acme/posture:ref:refs/heads/MAIN", "repo:acme/posture:ref:refs/heads/maín", "repo:acme/posture:ref:refs/heads/main "} {
		s.token = jwtFor(lookalike)
		s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusUnauthorized)
	}
	s.token = jwtFor("not-bound")
	s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusUnauthorized)
	s.token = s.issuer.Token(t, "repo:acme/posture:ref:refs/heads/main", "some-other-app", 5*time.Minute)
	s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusUnauthorized)
	s.token = externalauthtest.NewIssuer(t).Token(t, "repo:acme/posture:ref:refs/heads/main", "fleet", 5*time.Minute)
	s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusUnauthorized)
	s.token = s.issuer.Token(t, "repo:acme/posture:ref:refs/heads/main", "fleet", time.Hour)
	s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusUnauthorized) // exceeds max_token_lifetime

	// Step 7: removing the binding revokes access immediately.
	bind(observer.ID)
	s.token = jwtFor("repo:acme/posture:ref:refs/heads/main")
	s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusUnauthorized)

	// Step 8: the session token issued at creation kept working throughout.
	s.token = sessionToken
	s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusOK)

	// A bound user that stops being API-only can no longer authenticate this way.
	bind(maint.ID, "label-writer-sub")
	u, err := s.ds.UserByID(ctx, maint.ID)
	require.NoError(t, err)
	u.APIOnly = false
	require.NoError(t, s.ds.SaveUser(ctx, u))
	s.token = jwtFor("label-writer-sub")
	s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusUnauthorized)
}

func (s *integrationExternalAuthTestSuite) createAPIOnlyUser(name, role string) (fleet.User, string) {
	s.token = s.getTestAdminToken()
	var resp struct {
		User  fleet.User `json:"user"`
		Token string     `json:"token"`
	}
	s.DoJSON("POST", "/api/latest/fleet/users/api_only", map[string]any{"name": name, "global_role": role}, http.StatusOK, &resp)
	require.NotZero(s.T(), resp.User.ID)
	return resp.User, resp.Token
}
