package service

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/removednames"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// TestRemovedFleetNames checks that the old team names are rejected, not silently ignored, on the endpoints whose request types
// already use the fleet names. A request that still sends one would otherwise apply to No team or to every fleet.
func (s *integrationEnterpriseTestSuite) TestRemovedFleetNames() {
	t := s.T()
	reportsBefore := len(s.listReports())

	// rejectNamed expects the error to name the field param, which differs from key only when key is a removed value.
	rejectNamed := func(method, path, body, param, key string, query ...string) {
		t.Run(method+" "+path+" "+key, func(t *testing.T) {
			resp := s.DoRaw(method, "/api/latest/fleet"+path, []byte(body), http.StatusUnprocessableEntity, query...)
			var got struct {
				Message string              `json:"message"`
				Errors  []map[string]string `json:"errors"`
			}
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
			require.Equal(t, "Validation Failed", got.Message)
			require.Equal(t, []map[string]string{{
				"name": param, "reason": `The "` + key + `" parameter is not available anymore. What to use instead: ` + removednames.GuideURL,
			}}, got.Errors)
		})
	}
	reject := func(method, path, body, key string, query ...string) {
		rejectNamed(method, path, body, key, key, query...)
	}

	// Request body keys: fleet roles, selected fleets and the fleet of a created or targeted entity.
	for _, tc := range [][]string{
		{"POST", "/invites", `{"email": "a@example.com", "name": "a", "teams": []}`, "teams"},
		{"PATCH", "/invites/1", `{"teams": []}`, "teams"},
		{"POST", "/users/admin", `{"name": "a", "email": "a@example.com", "teams": []}`, "teams"},
		{"PATCH", "/users/1", `{"teams": []}`, "teams"},
		{"POST", "/users/api_only", `{"name": "a", "teams": []}`, "teams"},
		{"PATCH", "/users/api_only/1", `{"teams": []}`, "teams"},
		{"POST", "/targets", `{"selected": {"teams": [1]}}`, "teams"},
		{"POST", "/targets/count", `{"selected": {"teams": [1]}}`, "teams"},
		{"POST", "/spec/labels", `{"specs": [{"name": "a", "team_id": 1}]}`, "team_id"},
		{"POST", "/reports", `{"name": "removed names", "query": "SELECT 1", "team_id": 1}`, "team_id"},
		{"POST", "/spec/reports", `{"specs": [{"name": "a", "query": "SELECT 1", "team": "a"}]}`, "team"},
		{"POST", "/spec/policies", `{"specs": [{"name": "a", "query": "SELECT 1", "team": "a"}]}`, "team"},
		{"POST", "/automations/reset", `{"team_ids": [1]}`, "team_ids"},
		{"POST", "/scripts/run", `{"script_id": 1, "host_id": 1, "team_id": 1}`, "team_id"},
		{"POST", "/scripts/run/sync", `{"script_id": 1, "host_id": 1, "team_id": 1}`, "team_id"},
		{"PUT", "/setup_experience/software", `{"team_id": 1, "platform": "macos"}`, "team_id"},
		{"POST", "/certificates", `{"name": "a", "team_id": 1}`, "team_id"},
		{"DELETE", "/spec/certificates", `{"team_id": 1, "names": ["a"]}`, "team_id"},
		{"PATCH", "/config", `{"mdm": {"apple_bm_default_team": "a"}}`, "apple_bm_default_team"},
	} {
		reject(tc[0], tc[1], tc[2], tc[3])
	}

	// Query parameters are rejected before the body is read, so these requests carry none. The new name next to the old one does not win.
	for _, tc := range [][]string{
		{"GET", "/labels"}, {"GET", "/labels/summary"}, {"GET", "/spec/labels"}, {"POST", "/spec/labels"},
		{"GET", "/reports"}, {"GET", "/reports/1/report"}, {"GET", "/spec/reports/a"},
		{"GET", "/scripts"}, {"GET", "/scripts/batch"}, {"POST", "/scripts/batch"},
		{"GET", "/setup_experience/software"}, {"GET", "/setup_experience/script"}, {"DELETE", "/setup_experience/script"},
		{"GET", "/certificates"},
		{"GET", "/hosts"}, {"GET", "/hosts/count"}, {"GET", "/hosts/report"}, {"GET", "/users"},
	} {
		reject(tc[0], tc[1], "", "team_id", "fleet_id", "0", "team_id", "1")
	}
	reject("POST", "/scripts/batch", "", "team_name", "team_name", "a")

	// Removed sort keys are values of order_key, so the error names that parameter.
	for _, tc := range [][]string{
		{"/hosts", "team_name"}, {"/hosts", "team_id"}, {"/labels", "team_id"}, {"/labels/1/hosts", "team_name"},
		{"/fleets/0/policies", "team_id"}, {"/reports", "team_id"}, {"/configuration_profiles", "team_id"},
	} {
		rejectNamed("GET", tc[0], "", "order_key", tc[1], "order_key", tc[1])
	}

	// The rejected create calls did not create anything, and the new name is accepted.
	require.Len(t, s.listReports(), reportsBefore)
}

func (s *integrationEnterpriseTestSuite) listReports() []fleet.Query {
	var listResp fleet.ListQueriesResponse
	s.DoJSON("GET", "/api/latest/fleet/reports", nil, http.StatusOK, &listResp, "per_page", "1000", "fleet_id", "0")
	return listResp.Queries
}
