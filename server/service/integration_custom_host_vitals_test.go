package service

import (
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCustomHostVitalsCRUD exercises the full lifecycle through the HTTP stack:
// list (empty) -> create -> list/search -> update -> set a host value ->
// host detail surfaces it -> delete (cascades) -> list (empty), asserting the
// activity emitted at each mutating step.
func (s *integrationTestSuite) TestCustomHostVitalsCRUD() {
	t := s.T()

	// Initially empty.
	var listResp fleet.ListCustomHostVitalsResponse
	s.DoJSON("GET", "/api/latest/fleet/custom_host_vitals", nil, http.StatusOK, &listResp)
	require.Empty(t, listResp.CustomHostVitals)
	require.Equal(t, 0, listResp.Count)

	// Create.
	var createResp fleet.CreateCustomHostVitalResponse
	s.DoJSON("POST", "/api/latest/fleet/custom_host_vitals", fleet.CreateCustomHostVitalRequest{Name: "Asset tag"}, http.StatusOK, &createResp)
	require.NotNil(t, createResp.CustomHostVital)
	require.NotZero(t, createResp.CustomHostVital.ID)
	require.Equal(t, "Asset tag", createResp.CustomHostVital.Name)
	vitalID := createResp.CustomHostVital.ID
	s.lastActivityMatches(
		fleet.ActivityTypeCreatedCustomHostVital{}.ActivityName(),
		fmt.Sprintf(`{"custom_host_vital_id": %d, "custom_host_vital_name": "Asset tag"}`, vitalID),
		0,
	)

	// List shows the created definition.
	listResp = fleet.ListCustomHostVitalsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/custom_host_vitals", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.CustomHostVitals, 1)
	require.Equal(t, 1, listResp.Count)
	require.Equal(t, "Asset tag", listResp.CustomHostVitals[0].Name)

	// Duplicate name is rejected with a conflict.
	s.Do("POST", "/api/latest/fleet/custom_host_vitals", fleet.CreateCustomHostVitalRequest{Name: "Asset tag"}, http.StatusConflict)

	// Update (rename).
	var updateResp fleet.UpdateCustomHostVitalResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/custom_host_vitals/%d", vitalID), fleet.UpdateCustomHostVitalRequest{Name: "Asset ID"}, http.StatusOK, &updateResp)
	require.NotNil(t, updateResp.CustomHostVital)
	require.Equal(t, vitalID, updateResp.CustomHostVital.ID)
	require.Equal(t, "Asset ID", updateResp.CustomHostVital.Name)
	s.lastActivityMatches(
		fleet.ActivityTypeEditedCustomHostVital{}.ActivityName(),
		fmt.Sprintf(`{"custom_host_vital_id": %d, "custom_host_vital_name": "Asset ID"}`, vitalID),
		0,
	)

	// Search matches by name; a non-matching query returns nothing.
	listResp = fleet.ListCustomHostVitalsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/custom_host_vitals", nil, http.StatusOK, &listResp, "query", "asset")
	require.Len(t, listResp.CustomHostVitals, 1)
	listResp = fleet.ListCustomHostVitalsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/custom_host_vitals", nil, http.StatusOK, &listResp, "query", "nomatch")
	require.Empty(t, listResp.CustomHostVitals)

	// Before any value is set, the host detail still surfaces every definition
	// with an empty value.
	host := s.createHosts(t)[0]
	var preSetResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &preSetResp)
	require.Len(t, preSetResp.Host.CustomHostVitals, 1)
	assert.Equal(t, vitalID, preSetResp.Host.CustomHostVitals[0].CustomHostVitalID)
	assert.Equal(t, "Asset ID", preSetResp.Host.CustomHostVitals[0].Name)
	assert.Empty(t, preSetResp.Host.CustomHostVitals[0].Value)

	// Set a value for the vital on a host.
	s.Do("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/custom_host_vitals/%d", host.ID, vitalID), fleet.SetHostCustomHostVitalValueRequest{Value: "engineering"}, http.StatusOK)
	s.lastActivityMatches(
		fleet.ActivityTypeEditedCustomHostVitalValue{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "custom_host_vital_id": %d, "custom_host_vital_name": "Asset ID"}`, host.ID, host.DisplayName(), vitalID),
		0,
	)

	// Host detail surfaces the per-host value.
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.Len(t, hostResp.Host.CustomHostVitals, 1)
	assert.Equal(t, vitalID, hostResp.Host.CustomHostVitals[0].CustomHostVitalID)
	assert.Equal(t, "Asset ID", hostResp.Host.CustomHostVitals[0].Name)
	assert.Equal(t, "engineering", hostResp.Host.CustomHostVitals[0].Value)

	// Clearing the value (Save empty) is accepted and persists as an empty string.
	s.Do("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/custom_host_vitals/%d", host.ID, vitalID), fleet.SetHostCustomHostVitalValueRequest{Value: ""}, http.StatusOK)
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.Len(t, hostResp.Host.CustomHostVitals, 1)
	assert.Equal(t, vitalID, hostResp.Host.CustomHostVitals[0].CustomHostVitalID)
	assert.Empty(t, hostResp.Host.CustomHostVitals[0].Value)

	// Delete the definition; the per-host value cascades away.
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/custom_host_vitals/%d", vitalID), nil, http.StatusOK)
	s.lastActivityMatches(
		fleet.ActivityTypeDeletedCustomHostVital{}.ActivityName(),
		fmt.Sprintf(`{"custom_host_vital_id": %d, "custom_host_vital_name": "Asset ID"}`, vitalID),
		0,
	)

	// List is empty again.
	listResp = fleet.ListCustomHostVitalsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/custom_host_vitals", nil, http.StatusOK, &listResp)
	require.Empty(t, listResp.CustomHostVitals)
	require.Equal(t, 0, listResp.Count)

	// Host detail no longer surfaces the value.
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.Empty(t, hostResp.Host.CustomHostVitals)
}
