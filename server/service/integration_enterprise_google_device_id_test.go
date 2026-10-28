package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestHostGoogleDeviceID() {
	t := s.T()
	hosts := s.createHosts(t, "ios", "android", "darwin")
	iosHost, androidHost, macHost := hosts[0], hosts[1], hosts[2]

	idPath := func(h *fleet.Host) string { return fmt.Sprintf("/api/latest/fleet/hosts/%d/google_device_id", h.ID) }
	hostJSON := func(path string) map[string]any {
		res := s.Do("GET", path, nil, http.StatusOK)
		var body struct {
			Host map[string]any `json:"host"`
		}
		require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
		return body.Host
	}
	storedID := func(h *fleet.Host) any {
		return hostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID))["google_device_id"]
	}
	requireErrorReason := func(res *http.Response, reason string) {
		var errBody struct {
			Errors []map[string]string `json:"errors"`
		}
		require.NoError(t, json.NewDecoder(res.Body).Decode(&errBody))
		require.Equal(t, []map[string]string{{"name": "google_device_id", "reason": reason}}, errBody.Errors)
	}
	activityDetails := func(h *fleet.Host, id string) string {
		return fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "google_device_id": %q}`, h.ID, h.DisplayName(), id)
	}

	// mobile hosts report "" until set, other platforms omit the field
	got := hostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", iosHost.ID))
	require.Contains(t, got, "google_device_id")
	require.Empty(t, got["google_device_id"])
	require.NotContains(t, hostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", macHost.ID)), "google_device_id")

	var putResp putHostGoogleDeviceIDResponse
	s.DoJSON("PUT", idPath(iosHost), map[string]any{"google_device_id": "device-a"}, http.StatusOK, &putResp)
	require.Equal(t, iosHost.ID, putResp.HostID)
	require.Equal(t, "device-a", putResp.GoogleDeviceID)
	activityID := s.lastActivityMatches("set_host_google_device_id", activityDetails(iosHost, "device-a"), 0)
	s.lastHostActivityMatches(iosHost.ID, "set_host_google_device_id", activityDetails(iosHost, "device-a"), activityID)

	// repeating the same value is idempotent and doesn't log another activity
	putResp = putHostGoogleDeviceIDResponse{}
	s.DoJSON("PUT", idPath(iosHost), map[string]any{"google_device_id": "device-a"}, http.StatusOK, &putResp)
	require.Equal(t, "device-a", putResp.GoogleDeviceID)
	s.lastActivityMatches("", "", activityID)

	require.Equal(t, "device-a", storedID(iosHost))
	require.Equal(t, "device-a", hostJSON("/api/latest/fleet/hosts/identifier/" + iosHost.UUID)["google_device_id"])

	// a different value is refused
	res := s.Do("PUT", idPath(iosHost), map[string]any{"google_device_id": "device-b"}, http.StatusConflict)
	requireErrorReason(res, "This host already has a different Google device ID.")
	require.Equal(t, "device-a", storedID(iosHost))

	// an ID can only be bound to one host
	res = s.Do("PUT", idPath(androidHost), map[string]any{"google_device_id": "device-a"}, http.StatusConflict)
	requireErrorReason(res, "This Google device ID is already used by another host.")
	require.Empty(t, storedID(androidHost))
	s.DoJSON("PUT", idPath(androidHost), map[string]any{"google_device_id": "device-c"}, http.StatusOK, &putResp)
	require.Equal(t, "device-c", hostJSON("/api/latest/fleet/hosts/identifier/" + androidHost.UUID)["google_device_id"])

	// only DELETE removes the ID, and it logs an activity
	res = s.Do("PUT", idPath(iosHost), map[string]any{"google_device_id": ""}, http.StatusUnprocessableEntity)
	name, _ := extractServerErrorNameReason(res.Body)
	require.Equal(t, "google_device_id", name)
	require.Equal(t, "device-a", storedID(iosHost))
	s.Do("DELETE", idPath(iosHost), nil, http.StatusNoContent)
	require.Empty(t, storedID(iosHost))
	activityID = s.lastActivityMatches("deleted_host_google_device_id", activityDetails(iosHost, "device-a"), 0)
	s.lastHostActivityMatches(iosHost.ID, "deleted_host_google_device_id", activityDetails(iosHost, "device-a"), activityID)
	// deleting again is a no-op without an activity
	s.Do("DELETE", idPath(iosHost), nil, http.StatusNoContent)
	s.lastActivityMatches("", "", activityID)
	s.DoJSON("PUT", idPath(iosHost), map[string]any{"google_device_id": "device-b"}, http.StatusOK, &putResp)
	require.Equal(t, "device-b", putResp.GoogleDeviceID)
	s.lastActivityMatches("set_host_google_device_id", activityDetails(iosHost, "device-b"), 0)

	// only iOS, iPadOS and Android hosts
	res = s.Do("PUT", idPath(macHost), map[string]any{"google_device_id": "device-d"}, http.StatusUnprocessableEntity)
	name, _ = extractServerErrorNameReason(res.Body)
	require.Equal(t, "google_device_id", name)

	res = s.Do("PUT", idPath(iosHost), map[string]any{}, http.StatusUnprocessableEntity)
	name, _ = extractServerErrorNameReason(res.Body)
	require.Equal(t, "google_device_id", name)
	require.Equal(t, "device-b", storedID(iosHost))

	// surrounding whitespace would store an ID that conflicts with the clean one later
	for _, bad := range []string{" device-b", "device-b\n", "   "} {
		res = s.Do("PUT", idPath(iosHost), map[string]any{"google_device_id": bad}, http.StatusUnprocessableEntity)
		name, _ = extractServerErrorNameReason(res.Body)
		require.Equal(t, "google_device_id", name)
	}
	require.Equal(t, "device-b", storedID(iosHost))

	// the limit is 255 characters, not bytes
	s.Do("DELETE", idPath(iosHost), nil, http.StatusNoContent)
	maxLenID := strings.Repeat("é", 255)
	s.DoJSON("PUT", idPath(iosHost), map[string]any{"google_device_id": maxLenID}, http.StatusOK, &putResp)
	require.Equal(t, maxLenID, storedID(iosHost))
	s.Do("DELETE", idPath(iosHost), nil, http.StatusNoContent)
	res = s.Do("PUT", idPath(iosHost), map[string]any{"google_device_id": strings.Repeat("a", 256)}, http.StatusUnprocessableEntity)
	name, _ = extractServerErrorNameReason(res.Body)
	require.Equal(t, "google_device_id", name)
	s.DoJSON("PUT", idPath(iosHost), map[string]any{"google_device_id": "device-b"}, http.StatusOK, &putResp)

	s.Do("PUT", "/api/latest/fleet/hosts/999999/google_device_id", map[string]any{"google_device_id": "device-d"}, http.StatusNotFound)
	s.Do("DELETE", "/api/latest/fleet/hosts/999999/google_device_id", nil, http.StatusNotFound)

	// an observer can read the host but not set or remove the ID
	defer func() { s.token = s.getTestAdminToken() }()
	observer := &fleet.User{Name: t.Name() + " observer", Email: "google-device-id-observer@example.com", GlobalRole: new(fleet.RoleObserver)}
	require.NoError(t, observer.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(t.Context(), observer)
	require.NoError(t, err)
	s.token = s.getTestToken(observer.Email, test.GoodPassword)
	s.Do("PUT", idPath(androidHost), map[string]any{"google_device_id": "device-c"}, http.StatusForbidden)
	s.Do("DELETE", idPath(androidHost), nil, http.StatusForbidden)

	// a team maintainer can only set or remove the ID on hosts of their own fleet
	s.token = s.getTestAdminToken()
	team, err := s.ds.NewTeam(t.Context(), &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.ds.DeleteTeam(context.Background(), team.ID)) })
	require.NoError(t, s.ds.AddHostsToTeam(t.Context(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{androidHost.ID})))
	teamMaintainer := &fleet.User{
		Name:  t.Name() + " team maintainer",
		Email: "google-device-id-team-maintainer@example.com",
		Teams: []fleet.UserTeam{{Team: *team, Role: fleet.RoleMaintainer}},
	}
	require.NoError(t, teamMaintainer.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(t.Context(), teamMaintainer)
	require.NoError(t, err)
	s.token = s.getTestToken(teamMaintainer.Email, test.GoodPassword)
	s.DoJSON("PUT", idPath(androidHost), map[string]any{"google_device_id": "device-c"}, http.StatusOK, &putResp)
	s.Do("PUT", idPath(iosHost), map[string]any{"google_device_id": "device-b"}, http.StatusForbidden)
	s.Do("DELETE", idPath(iosHost), nil, http.StatusForbidden)
	s.Do("DELETE", idPath(androidHost), nil, http.StatusNoContent)
	s.DoJSON("PUT", idPath(androidHost), map[string]any{"google_device_id": "device-c"}, http.StatusOK, &putResp)

	// the registration service's API-only user can set the ID but not remove it
	s.token = s.getTestAdminToken()
	var apiUserResp struct {
		Token string `json:"token"`
	}
	s.DoJSON("POST", "/api/latest/fleet/users/api_only", map[string]any{
		"name":        t.Name() + " registration service",
		"global_role": fleet.RoleMaintainer,
		"api_endpoints": []map[string]any{
			{"method": "GET", "path": "/api/v1/fleet/hosts/identifier/:identifier"},
			{"method": "PUT", "path": "/api/v1/fleet/hosts/:id/google_device_id"},
		},
	}, http.StatusOK, &apiUserResp)
	require.NotEmpty(t, apiUserResp.Token)
	s.token = apiUserResp.Token
	hostJSON("/api/latest/fleet/hosts/identifier/" + androidHost.UUID)
	s.DoJSON("PUT", idPath(androidHost), map[string]any{"google_device_id": "device-c"}, http.StatusOK, &putResp)
	s.Do("DELETE", idPath(androidHost), nil, http.StatusForbidden)
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", androidHost.ID), nil, http.StatusForbidden)
	s.token = s.getTestAdminToken()
	require.Equal(t, "device-c", storedID(androidHost))
}
