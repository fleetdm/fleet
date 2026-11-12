package service

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRemovedFleetNames checks that the old team names are rejected, not silently ignored, on the endpoints whose request types
// already use the fleet names. A request that still sends one would otherwise apply to No team or to every fleet.
func (s *integrationEnterpriseTestSuite) TestRemovedFleetNames() {
	t := s.T()

	reportsBefore := len(s.listReports())

	for _, tc := range []struct {
		name, method, path, body string
		query                    []string
		key                      string
	}{
		// Users, invites and targets: the fleet roles and the selected fleets.
		{name: "invite create", method: "POST", path: "/invites", body: `{"email": "a@example.com", "name": "a", "teams": []}`, key: "teams"},
		{name: "invite modify", method: "PATCH", path: "/invites/1", body: `{"teams": []}`, key: "teams"},
		{name: "user create", method: "POST", path: "/users/admin", body: `{"name": "a", "email": "a@example.com", "teams": []}`, key: "teams"},
		{name: "user modify", method: "PATCH", path: "/users/1", body: `{"teams": []}`, key: "teams"},
		{name: "api-only user create", method: "POST", path: "/users/api_only", body: `{"name": "a", "teams": []}`, key: "teams"},
		{name: "api-only user modify", method: "PATCH", path: "/users/api_only/1", body: `{"teams": []}`, key: "teams"},
		{name: "targets", method: "POST", path: "/targets", body: `{"selected": {"teams": [1]}}`, key: "teams"},
		{name: "targets count", method: "POST", path: "/targets/count", body: `{"selected": {"teams": [1]}}`, key: "teams"},

		// Labels, reports and policies.
		{name: "labels list", method: "GET", path: "/labels", query: []string{"team_id", "1"}, key: "team_id"},
		{name: "labels summary", method: "GET", path: "/labels/summary", query: []string{"team_id", "1"}, key: "team_id"},
		{name: "label specs list", method: "GET", path: "/spec/labels", query: []string{"team_id", "1"}, key: "team_id"},
		{name: "label specs apply query", method: "POST", path: "/spec/labels", body: `{"specs": []}`, query: []string{"team_id", "1"}, key: "team_id"},
		{name: "label specs apply key", method: "POST", path: "/spec/labels", body: `{"specs": [{"name": "a", "team_id": 1}]}`, key: "team_id"},
		{name: "reports list", method: "GET", path: "/reports", query: []string{"team_id", "1"}, key: "team_id"},
		{name: "report results", method: "GET", path: "/reports/1/report", query: []string{"team_id", "1"}, key: "team_id"},
		{name: "report spec", method: "GET", path: "/spec/reports/a", query: []string{"team_id", "1"}, key: "team_id"},
		{name: "report create", method: "POST", path: "/reports", body: `{"name": "removed names", "query": "SELECT 1", "team_id": 1}`, key: "team_id"},
		{name: "report spec apply", method: "POST", path: "/spec/reports", body: `{"specs": [{"name": "a", "query": "SELECT 1", "team": "a"}]}`, key: "team"},
		{name: "policy spec apply", method: "POST", path: "/spec/policies", body: `{"specs": [{"name": "a", "query": "SELECT 1", "team": "a"}]}`, key: "team"},
		{name: "automations reset", method: "POST", path: "/automations/reset", body: `{"team_ids": [1]}`, key: "team_ids"},

		// Scripts.
		{name: "scripts list", method: "GET", path: "/scripts", query: []string{"team_id", "1"}, key: "team_id"},
		{name: "scripts batch set id", method: "POST", path: "/scripts/batch", body: `{"scripts": []}`, query: []string{"team_id", "1"}, key: "team_id"},
		{name: "scripts batch set name", method: "POST", path: "/scripts/batch", body: `{"scripts": []}`, query: []string{"team_name", "a"}, key: "team_name"},
		{name: "scripts batch executions", method: "GET", path: "/scripts/batch", query: []string{"team_id", "1"}, key: "team_id"},
		{name: "script run", method: "POST", path: "/scripts/run", body: `{"script_id": 1, "host_id": 1, "team_id": 1}`, key: "team_id"},
		{name: "script run sync", method: "POST", path: "/scripts/run/sync", body: `{"script_id": 1, "host_id": 1, "team_id": 1}`, key: "team_id"},

		// Setup experience and certificate templates.
		{name: "setup experience software list", method: "GET", path: "/setup_experience/software", query: []string{"team_id", "1"}, key: "team_id"},
		{name: "setup experience software set", method: "PUT", path: "/setup_experience/software", body: `{"team_id": 1, "platform": "macos"}`, key: "team_id"},
		{name: "setup experience script get", method: "GET", path: "/setup_experience/script", query: []string{"team_id", "1"}, key: "team_id"},
		{name: "setup experience script delete", method: "DELETE", path: "/setup_experience/script", query: []string{"team_id", "1"}, key: "team_id"},
		{name: "certificates create", method: "POST", path: "/certificates", body: `{"name": "a", "team_id": 1}`, key: "team_id"},
		{name: "certificates list", method: "GET", path: "/certificates", query: []string{"team_id", "1"}, key: "team_id"},
		{name: "certificate specs delete", method: "DELETE", path: "/spec/certificates", body: `{"team_id": 1, "names": ["a"]}`, key: "team_id"},

		// Both names together are rejected too, the new one does not win.
		{name: "reports list both", method: "GET", path: "/reports", query: []string{"fleet_id", "0", "team_id", "1"}, key: "team_id"},
		{name: "report create both", method: "POST", path: "/reports", body: `{"name": "both", "query": "SELECT 1", "fleet_id": 0, "team_id": 1}`, key: "team_id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := s.DoRaw(tc.method, "/api/latest/fleet"+tc.path, []byte(tc.body), http.StatusUnprocessableEntity, tc.query...)
			var got struct {
				Message string              `json:"message"`
				Errors  []map[string]string `json:"errors"`
			}
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
			require.Equal(t, "Validation Failed", got.Message)
			require.Len(t, got.Errors, 1)
			assert.Equal(t, tc.key, got.Errors[0]["name"])
			assert.Equal(t, `The "`+tc.key+`" parameter is not available anymore. What to use instead: `+removednames.GuideURL, got.Errors[0]["reason"])
		})
	}

	// The rejected create calls did not create anything.
	require.Len(t, s.listReports(), reportsBefore)

	// The new names are still accepted.
	var listResp fleet.ListQueriesResponse
	s.DoJSON("GET", "/api/latest/fleet/reports", nil, http.StatusOK, &listResp, "fleet_id", "0")
}

func (s *integrationEnterpriseTestSuite) listReports() []fleet.Query {
	var listResp fleet.ListQueriesResponse
	s.DoJSON("GET", "/api/latest/fleet/reports", nil, http.StatusOK, &listResp, "per_page", "1000")
	return listResp.Queries
}
