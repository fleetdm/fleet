package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestRemovedMDMPaths() {
	t := s.T()
	ctx := t.Context()

	newSessionToken := func(user *fleet.User) string {
		require.NoError(t, user.SetPassword(test.GoodPassword, 10, 10))
		user, err := s.ds.NewUser(ctx, user)
		require.NoError(t, err)
		session, err := s.ds.NewSession(ctx, user.ID, 64)
		require.NoError(t, err)
		return session.Key
	}
	observerToken := newSessionToken(&fleet.User{
		Name: "removed paths observer", Email: "removed-paths-observer@example.com", GlobalRole: new(fleet.RoleObserver),
	})
	apiOnlyToken := newSessionToken(&fleet.User{
		Name: "removed paths api", Email: "removed-paths-api@example.com", GlobalRole: new(fleet.RoleAdmin), APIOnly: true,
		APIEndpoints: []fleet.APIEndpointRef{{Method: "GET", Path: "/api/v1/fleet/hosts"}},
	})

	callers := []struct {
		name    string
		headers map[string]string
	}{
		{"no token", nil},
		{"admin", bearer(s.token)},
		{"observer", bearer(observerToken)},
		{"restricted API-only user", bearer(apiOnlyToken)},
	}
	removed := []struct{ method, path, endpoint string }{
		{"GET", "/api/latest/fleet/mdm/commands?host_identifier=x", "GET /mdm/commands"},
		{"POST", "/api/v1/fleet/mdm/hosts/1/lock", "POST /mdm/hosts/{id}/lock"},
		{"PATCH", "/api/latest/fleet/mdm/apple/settings", "PATCH /mdm/apple/settings"},
		{"GET", "/api/latest/fleet/mdm/profiles/summary", "GET /mdm/profiles/summary"},
		{"DELETE", "/api/v1/fleet/mdm/profiles/abc", "DELETE /mdm/profiles/{profile_uuid}"},
		{"POST", "/api/latest/fleet/mdm/bootstrap", "POST /mdm/bootstrap"},
		{"GET", "/api/latest/fleet/mdm/apple/bootstrap?token=x", "GET /mdm/apple/bootstrap"},
		{"GET", "/api/latest/fleet/mdm/setup/eula/x", "GET /mdm/setup/eula/{token}"},
		{"POST", "/api/latest/fleet/mdm/apple/request_csr", "POST /mdm/apple/request_csr"},
	}
	for _, caller := range callers {
		for _, c := range removed {
			res := s.DoRawWithHeaders(c.method, c.path, nil, http.StatusGone, caller.headers)
			var body struct {
				Message string `json:"message"`
				Errors  []struct {
					Reason string `json:"reason"`
				} `json:"errors"`
			}
			require.NoError(t, json.NewDecoder(res.Body).Decode(&body), "%s: %s %s", caller.name, c.method, c.path)
			res.Body.Close()
			require.Len(t, body.Errors, 1)
			assert.Equal(t, fmt.Sprintf(`The "%s" endpoint is not available anymore. What to use instead: %s`, c.endpoint, removednames.GuideURL),
				body.Errors[0].Reason, "%s: %s %s", caller.name, c.method, c.path)
		}
	}

	// Other methods on partly removed paths are still served.
	s.DoRawWithHeaders("GET", "/api/latest/fleet/mdm/apple/request_csr", nil, http.StatusForbidden, bearer(observerToken))
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{}},
		http.StatusNoContent, "dry_run", "true")

	// Paths and methods that were never routes are not reported as removed.
	s.DoRaw("GET", "/api/latest/fleet/mdm/not_a_route", nil, http.StatusNotFound)
	s.DoRaw("POST", "/api/latest/fleet/mdm/hosts/abc/lock", nil, http.StatusNotFound)
	s.DoRaw("PUT", "/api/latest/fleet/mdm/commands", nil, http.StatusMethodNotAllowed)
}
