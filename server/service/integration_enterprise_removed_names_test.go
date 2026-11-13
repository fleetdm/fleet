package service

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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

	reject := func(method, path, body, key string, query ...string) {
		t.Run(method+" "+path+" "+key, func(t *testing.T) {
			resp := s.DoRaw(method, "/api/latest/fleet"+path, []byte(body), http.StatusUnprocessableEntity, query...)
			var got struct {
				Message string              `json:"message"`
				Errors  []map[string]string `json:"errors"`
			}
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
			require.Equal(t, "Validation Failed", got.Message)
			require.Equal(t, []map[string]string{{
				"name": key, "reason": `The "` + key + `" parameter is not available anymore. What to use instead: ` + removednames.GuideURL,
			}}, got.Errors)
		})
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
		{"POST", "/software/app_store_apps", `{"app_store_id": "a", "team_id": 1}`, "team_id"},
		{"PATCH", "/software/titles/1/app_store_app", `{"team_id": 1}`, "team_id"},
		{"PATCH", "/fleets/1/secrets", `{"secrets": [{"secret": "a", "team_id": 1}]}`, "team_id"},
	} {
		reject(tc[0], tc[1], tc[2], tc[3])
	}

	// Keys inside the free-form host filters and the AB and VPP token arrays of the config, with force too, which skips strict
	// decoding.
	for _, tc := range [][]string{
		{"POST", "/hosts/transfer/filter", `{"fleet_id": null, "filters": {"team_id": 1}}`, "team_id"},
		{"POST", "/hosts/delete", `{"filters": {"team_id": 1}}`, "team_id"},
		{"PATCH", "/config", `{"mdm": {"apple_business": [{"organization_name": "a", "macos_team": "a"}]}}`, "macos_team"},
		{"PATCH", "/config?force=true", `{"mdm": {"apple_business": [{"organization_name": "a", "byod_team": "a"}]}}`, "byod_team"},
		{"PATCH", "/config?force=true", `{"mdm": {"volume_purchasing_program": [{"location": "a", "teams": ["a"]}]}}`, "teams"},
	} {
		reject(tc[0], tc[1], tc[2], tc[3])
	}

	// Multipart uploads: the old field is rejected whether or not fleet_id is also sent.
	for _, path := range []string{"/software/package", "/scripts", "/configuration_profiles", "/setup_experience/script", "/bootstrap"} {
		for _, fields := range []map[string]string{{"team_id": "0"}, {"fleet_id": "0", "team_id": "0"}} {
			t.Run("POST "+path+" form field team_id", func(t *testing.T) {
				var body bytes.Buffer
				w := multipart.NewWriter(&body)
				for name, value := range fields {
					require.NoError(t, w.WriteField(name, value))
				}
				require.NoError(t, w.Close())
				resp := s.DoRawWithHeaders("POST", "/api/latest/fleet"+path, body.Bytes(), http.StatusUnprocessableEntity,
					map[string]string{"Content-Type": w.FormDataContentType(), "Authorization": "Bearer " + s.token})
				var got struct {
					Errors []map[string]string `json:"errors"`
				}
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
				require.Equal(t, []map[string]string{{
					"name": "team_id", "reason": `The "team_id" parameter is not available anymore. What to use instead: ` + removednames.GuideURL,
				}}, got.Errors)
			})
		}
	}

	// Query parameters are rejected before the body is read, so these requests carry none. The new name next to the old one does not win.
	for _, tc := range [][]string{
		{"GET", "/labels"}, {"GET", "/labels/summary"}, {"GET", "/spec/labels"}, {"POST", "/spec/labels"},
		{"GET", "/reports"}, {"GET", "/reports/1/report"}, {"GET", "/spec/reports/a"},
		{"GET", "/scripts"}, {"GET", "/scripts/batch"}, {"POST", "/scripts/batch"},
		{"GET", "/setup_experience/software"}, {"GET", "/setup_experience/script"}, {"DELETE", "/setup_experience/script"},
		{"GET", "/certificates"},
		{"GET", "/hosts"}, {"GET", "/hosts/count"}, {"GET", "/hosts/report"}, {"GET", "/users"},
		{"GET", "/vulnerabilities"}, {"GET", "/vulnerabilities/CVE-2021-1234"},
		{"GET", "/software/fleet_maintained_apps"}, {"GET", "/software/fleet_maintained_apps/1"}, {"GET", "/software/app_store_apps"},
		{"GET", "/software/titles/1/icon"}, {"DELETE", "/software/titles/1/icon"},
	} {
		reject(tc[0], tc[1], "", "team_id", "fleet_id", "0", "team_id", "1")
	}
	reject("POST", "/scripts/batch", "", "team_name", "team_name", "a")
	reject("POST", "/software/app_store_apps/batch", "", "team_name", "team_name", "a")

	// The rejected create calls did not create anything, and the new name is accepted.
	require.Len(t, s.listReports(), reportsBefore)
}

func (s *integrationEnterpriseTestSuite) listReports() []fleet.Query {
	var listResp fleet.ListQueriesResponse
	s.DoJSON("GET", "/api/latest/fleet/reports", nil, http.StatusOK, &listResp, "per_page", "1000", "fleet_id", "0")
	return listResp.Queries
}
