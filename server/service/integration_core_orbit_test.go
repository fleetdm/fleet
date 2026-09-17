package service

// Orbit (fleetd) tests for the core (no-license) suite.
//
// Belongs here: orbit config and its notifications, orbit extensions, orbit
// enrollment including serial-number mismatch handling, and debug logging on
// enroll.
//
// Does not belong here: the plain osquery enrollment and distributed endpoints
// (integration_core_osquery_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestOrbitConfigNotifications() {
	t := s.T()
	ctx := context.Background()

	// set the enabled and configured flags,
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	origEnabledAndConfigured := appCfg.MDM.EnabledAndConfigured
	appCfg.MDM.EnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	defer func() {
		appCfg.MDM.EnabledAndConfigured = origEnabledAndConfigured
		err = s.ds.SaveAppConfig(ctx, appCfg)
		require.NoError(t, err)
	}()

	var resp fleet.OrbitGetConfigResponse
	// missing orbit key
	s.DoJSON("POST", "/api/fleet/orbit/config", nil, http.StatusUnauthorized, &resp)

	hNoMDM := createOrbitEnrolledHost(t, "darwin", "nomdm", s.ds)
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hNoMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	hSimpleMDM := createOrbitEnrolledHost(t, "darwin", "simplemdm", s.ds)
	err = s.ds.SetOrUpdateMDMData(context.Background(), hSimpleMDM.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "", false)
	require.NoError(t, err)
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hSimpleMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	// not yet assigned in ABM
	hFleetMDM := createOrbitEnrolledHost(t, "darwin", "fleetmdm", s.ds)
	err = s.ds.SetOrUpdateMDMData(context.Background(), hFleetMDM.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, "", false)
	require.NoError(t, err)

	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	// simulate ABM assignment
	encTok := uuid.NewString()
	abmToken, err := s.ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok), RenewAt: time.Now().Add(30 * 24 * time.Hour)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)
	err = s.ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*hFleetMDM}, abmToken.ID, make(map[uint]time.Time))
	require.NoError(t, err)
	err = s.ds.SetOrUpdateMDMData(context.Background(), hSimpleMDM.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "", false)
	require.NoError(t, err)
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.True(t, resp.Notifications.RenewEnrollmentProfile)

	// if the fleet mdm host is fully enrolled (not pending anymore), then the notification is false
	err = s.ds.SetOrUpdateMDMData(context.Background(), hFleetMDM.ID, false, true, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, "", false)
	require.NoError(t, err)
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	// the scripts orbit endpoints are accessible without license
	s.Do("POST", "/api/fleet/orbit/scripts/request", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusNotFound)
	s.Do("POST", "/api/fleet/orbit/scripts/result", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusBadRequest)
}

func (s *integrationTestSuite) TestEnrollOrbitExistingHostNoSerialMatch() {
	t := s.T()
	ctx := context.Background()

	// create a host with minimal information and the serial, no uuid/osquery id
	// (as when created via DEP sync).
	dbZeroTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	h, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:   uuid.New().String(),
		Platform:         "darwin",
		LastEnrolledAt:   dbZeroTime,
		DetailUpdatedAt:  dbZeroTime,
		RefetchRequested: true,
	})
	require.NoError(t, err)

	// create an enroll secret
	secret := uuid.New().String()
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: secret}},
		},
	}, http.StatusOK, &applyResp)

	// enroll the host from orbit, it will NOT match the existing host since MDM
	// is not configured (it will only look for a match by osquery_host_id with
	// the provided uuid).
	var resp enrollOrbitResponse
	hostUUID := uuid.New().String()
	s.DoJSON("POST", "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
		EnrollSecret:   secret,
		HardwareUUID:   hostUUID, // will not match any existing host
		HardwareSerial: h.HardwareSerial,
	}, http.StatusOK, &resp)
	require.NotEmpty(t, resp.OrbitNodeKey)

	// fetch the host, it will NOT match the one created above
	orbitHost, err := s.ds.LoadHostByOrbitNodeKey(ctx, resp.OrbitNodeKey)
	require.NoError(t, err)
	require.NotEqual(t, h.ID, orbitHost.ID)

	// enroll the host from osquery, it should match the Orbit-enrolled host
	var osqueryResp contract.EnrollOsqueryAgentResponse

	// NOTE(mna): using an osquery_host_id that is NOT the host's UUID would not work,
	// because we haven't enabled lookup by UUID due to not having an index and possible
	// side-effects of this on host ingestion performance. However, this should not happen
	// anyway in MDM-enabled environments as we will recommend using the UUID as osquery
	// host identifier.
	// See https://github.com/fleetdm/fleet/issues/9033#issuecomment-1411150758

	osqueryID := hostUUID

	s.DoJSON("POST", "/api/osquery/enroll", contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   secret,
		HostIdentifier: osqueryID,
		HostDetails: map[string]map[string]string{
			"system_info": {
				"uuid":            hostUUID,
				"hardware_serial": h.HardwareSerial,
			},
		},
	}, http.StatusOK, &osqueryResp)
	require.NotEmpty(t, osqueryResp.NodeKey)

	// load the host by osquery node key, should match the orbit host
	got, err := s.ds.LoadHostByNodeKey(ctx, osqueryResp.NodeKey)
	require.NoError(t, err)
	require.Equal(t, orbitHost.ID, got.ID)
}

func (s *integrationTestSuite) TestOrbitConfigExtensions() {
	t := s.T()
	ctx := context.Background()

	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	defer func() {
		err = s.ds.SaveAppConfig(ctx, appCfg)
		require.NoError(t, err)
	}()

	// Orbit client gets no extensions if extensions are not configured.
	orbitLinuxClient := createOrbitEnrolledHost(t, "linux", "foobar1", s.ds)
	resp := fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitLinuxClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.Extensions)

	// Attempt to add extensions (should succeed).
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
	"agent_options": {
		"config": {
			"options": {
				"pack_delimiter": "/",
				"logger_tls_period": 10,
				"distributed_plugin": "tls",
				"disable_distributed": false,
				"logger_tls_endpoint": "/api/osquery/log",
				"distributed_interval": 10,
				"distributed_tls_max_attempts": 3
			}
		},
		"extensions": {
			"hello_world_linux": {
				"channel": "stable",
				"platform": "linux"
			},
			"hello_mars_linux": {
				"channel": "stable",
				"platform": "linux"
			},
			"hello_world_macos": {
				"channel": "stable",
				"platform": "macos"
			}
		}
	}
}`), http.StatusOK)

	// Attempt to add labels to extensions (only available on premium).
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
  "agent_options": {
	"config": {
		"options": {
		"pack_delimiter": "/",
		"logger_tls_period": 10,
		"distributed_plugin": "tls",
		"disable_distributed": false,
		"logger_tls_endpoint": "/api/osquery/log",
		"distributed_interval": 10,
		"distributed_tls_max_attempts": 3
		}
	},
	"extensions": {
		"hello_world_linux": {
			"channel": "stable",
			"platform": "linux"
		},
		"hello_world_macos": {
			"labels": [
				"All hosts",
				"Some label"
			],
			"channel": "stable",
			"platform": "macos"
		},
		"hello_world_windows": {
			"channel": "stable",
			"platform": "windows"
		}
	}
  }
}`), http.StatusBadRequest)

	// Orbit client gets extensions configured for its platform.
	orbitDarwinClient := createOrbitEnrolledHost(t, "darwin", "foobar2", s.ds)
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitDarwinClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
    "hello_world_macos": {
      "platform": "macos",
      "channel": "stable"
    }
  }`, string(resp.Extensions))

	orbitWindowsClient := createOrbitEnrolledHost(t, "windows", "foobar3", s.ds)

	// Orbit client gets no extensions if none of the platforms target it.
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitWindowsClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.Extensions)

	// Orbit client gets the two extensions configured for its platform.
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitLinuxClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
	"hello_world_linux": {
		"channel": "stable",
		"platform": "linux"
	},
	"hello_mars_linux": {
		"channel": "stable",
		"platform": "linux"
	}
  }`, string(resp.Extensions))
}

func (s *integrationTestSuite) TestOrbitDebugLoggingOnEnroll() {
	t := s.T()
	ctx := context.Background()

	// Reject above cap.
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "orbit": {"debug_logging_on_enroll_duration": 86401} }
	}`), http.StatusBadRequest, &acResp)

	// Reject negative.
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "orbit": {"debug_logging_on_enroll_duration": -1} }
	}`), http.StatusBadRequest, &acResp)

	// Reject duration string (must be seconds, integer).
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "orbit": {"debug_logging_on_enroll_duration": "1h"} }
	}`), http.StatusBadRequest, &acResp)

	// 1h global window (3600 seconds).
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "orbit": {"debug_logging_on_enroll_duration": 3600} }
	}`), http.StatusOK, &acResp)

	secret := uuid.New().String()
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: secret}},
		},
	}, http.StatusOK, &applyResp)

	beforeEnroll := time.Now()
	var enrollResp enrollOrbitResponse
	hostUUID := uuid.New().String()
	s.DoJSON("POST", "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
		EnrollSecret:   secret,
		HardwareUUID:   hostUUID,
		HardwareSerial: uuid.New().String(),
		Hostname:       "enroll-debug-stamped",
		Platform:       "linux",
	}, http.StatusOK, &enrollResp)
	require.NotEmpty(t, enrollResp.OrbitNodeKey)

	stampedHost, err := s.ds.LoadHostByOrbitNodeKey(ctx, enrollResp.OrbitNodeKey)
	require.NoError(t, err)
	require.NotNil(t, stampedHost.OrbitDebugUntil)
	require.WithinDuration(t, beforeEnroll.Add(time.Hour), *stampedHost.OrbitDebugUntil, time.Minute)

	// Clearing the option stops stamping.
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "orbit": {"debug_logging_on_enroll_duration": 0} }
	}`), http.StatusOK, &acResp)

	var enrollResp2 enrollOrbitResponse
	hostUUID2 := uuid.New().String()
	s.DoJSON("POST", "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
		EnrollSecret:   secret,
		HardwareUUID:   hostUUID2,
		HardwareSerial: uuid.New().String(),
		Hostname:       "enroll-debug-not-stamped",
		Platform:       "linux",
	}, http.StatusOK, &enrollResp2)

	unstampedHost, err := s.ds.LoadHostByOrbitNodeKey(ctx, enrollResp2.OrbitNodeKey)
	require.NoError(t, err)
	require.Nil(t, unstampedHost.OrbitDebugUntil)
}
