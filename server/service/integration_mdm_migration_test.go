package service

// MDM migration tests for the MDM suite.
//
// Belongs here: migrating macOS hosts from another MDM to Fleet — the macOS migration settings, the
// migration webhook, the end-to-end migration flow, and silent migration edge cases.
//
// Does not belong here: Windows MDM migration (integration_mdm_windows_enrollment_test.go) and fresh
// enrollments (integration_mdm_apple_enrollment_test.go).

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestAppConfigMDMMacOSMigration() {
	t := s.T()

	checkDefaultAppConfig := func() {
		var ac appConfigResponse
		s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &ac)
		require.False(t, ac.MDM.MacOSMigration.Enable)
		require.Empty(t, ac.MDM.MacOSMigration.Mode)
		require.Empty(t, ac.MDM.MacOSMigration.WebhookURL)
	}
	checkDefaultAppConfig()

	var acResp appConfigResponse
	// missing webhook_url
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": true, "mode": "voluntary", "webhook_url": "" } }
  	}`), http.StatusUnprocessableEntity, &acResp)
	checkDefaultAppConfig()

	// invalid url scheme for webhook_url
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": true, "mode": "voluntary", "webhook_url": "ftp://example.com" } }
	}`), http.StatusUnprocessableEntity, &acResp)
	checkDefaultAppConfig()

	// invalid mode
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": true, "mode": "foobar", "webhook_url": "https://example.com" } }
  	}`), http.StatusUnprocessableEntity, &acResp)
	checkDefaultAppConfig()

	// valid request
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": true, "mode": "voluntary", "webhook_url": "https://example.com" } }
	}`), http.StatusOK, &acResp)

	// confirm new app config
	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.MDM.MacOSMigration.Enable)
	require.Equal(t, fleet.MacOSMigrationModeVoluntary, acResp.MDM.MacOSMigration.Mode)
	require.Equal(t, "https://example.com", acResp.MDM.MacOSMigration.WebhookURL)
}

func (s *integrationMDMTestSuite) TestMigrateMDMDeviceWebhook() {
	t := s.T()

	h := createHostAndDeviceToken(t, s.ds, "good-token")

	var webhookCalled bool
	webhookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled = true
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/test_mdm_migration":
			var payload fleet.MigrateMDMDeviceWebhookPayload
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			err = json.Unmarshal(b, &payload)
			assert.NoError(t, err)

			assert.Equal(t, h.ID, payload.Host.ID)
			assert.Equal(t, h.UUID, payload.Host.UUID)
			assert.Equal(t, h.HardwareSerial, payload.Host.HardwareSerial)

		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer webhookSrv.Close()

	// patch app config with webhook url
	acResp := fleet.AppConfig{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"macos_migration": {
				"enable": true,
				"mode": "voluntary",
				"webhook_url": "%s/test_mdm_migration"
			}
		}
	}`, webhookSrv.URL)), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.MacOSMigration.Enable)

	// expect errors when host is not eligible for migration
	isServer, enrolled, installedFromDEP := true, true, true
	mdmName := "ExampleMDM"
	mdmURL := "https://mdm.example.com"

	// host is a server so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, isServer, enrolled, mdmURL, installedFromDEP, mdmName, "", fleet.PersonalEnrollmentTypeNone))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	// host is not enrolled to MDM so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, !enrolled, mdmURL, installedFromDEP, mdmName, "", fleet.PersonalEnrollmentTypeNone))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	// host is already enrolled to Fleet MDM so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, enrolled, mdmURL, installedFromDEP, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	// up to this point, the refetch critical queries timestamp has not been set
	// on the host.
	h, err := s.ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	require.Nil(t, h.RefetchCriticalQueriesUntil)

	// host is enrolled to a third-party MDM but hasn't been assigned in
	// ABM yet, so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, enrolled, mdmURL, installedFromDEP, mdmName, "", fleet.PersonalEnrollmentTypeNone))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	s.enableABM(t.Name())

	// simulate that the device is assigned to Fleet in ABM
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/profile":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
			assert.NoError(t, err)
		case "/server/devices", "/devices/sync":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.DeviceResponse{
				Devices: []godep.Device{
					{
						SerialNumber: h.HardwareSerial,
						Model:        "Mac Mini",
						OS:           "osx",
						OpType:       "added",
					},
				},
			})
			assert.NoError(t, err)

		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var prof profileAssignmentReq
			assert.NoError(t, json.Unmarshal(b, &prof))
			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = map[string]string{
				prof.Devices[0]: string(fleet.DEPAssignProfileResponseSuccess),
			}
			encoder := json.NewEncoder(w)
			err = encoder.Encode(resp)
			assert.NoError(t, err)
		}
	}))
	s.runDEPSchedule()

	// hosts meets all requirements, webhook is run
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusNoContent)
	require.True(t, webhookCalled)
	webhookCalled = false

	// the refetch critical queries timestamp has been set in the future
	h, err = s.ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	require.NotNil(t, h.RefetchCriticalQueriesUntil)
	require.True(t, h.RefetchCriticalQueriesUntil.After(time.Now()))

	// calling again works but does not trigger the webhook, as it was called recently
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusNoContent)
	require.False(t, webhookCalled)

	// setting the refetch critical queries timestamp in the past triggers the webhook again
	h.RefetchCriticalQueriesUntil = new(time.Now().Add(-1 * time.Minute))
	err = s.ds.UpdateHost(context.Background(), h)
	require.NoError(t, err)

	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusNoContent)
	require.True(t, webhookCalled)
	webhookCalled = false

	// host is manually enrolled, which is allowed
	h.RefetchCriticalQueriesUntil = new(time.Now().Add(-1 * time.Minute))
	err = s.ds.UpdateHost(context.Background(), h)
	require.NoError(t, err)

	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, enrolled, mdmURL, !installedFromDEP, mdmName, "", fleet.PersonalEnrollmentTypeNone))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusNoContent)
	require.True(t, webhookCalled)
	webhookCalled = false

	// the refetch critical queries timestamp has been updated to the future
	h, err = s.ds.Host(context.Background(), h.ID)
	require.NoError(t, err)
	require.NotNil(t, h.RefetchCriticalQueriesUntil)
	require.True(t, h.RefetchCriticalQueriesUntil.After(time.Now()))

	require.NoError(t, s.ds.UpdateHostRefetchCriticalQueriesUntil(context.Background(), h.ID, nil))

	// bad token
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "bad-token"), nil, http.StatusUnauthorized)
	require.False(t, webhookCalled)

	// disable macos migration
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"macos_migration": {
				"enable": false,
				"mode": "voluntary",
				"webhook_url": ""
		      }
		}
	}`), http.StatusOK, &acResp)
	require.False(t, acResp.MDM.MacOSMigration.Enable)

	// expect error if macos migration is not configured
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)
}

func (s *integrationMDMTestSuite) TestMigrateMDMDeviceWebhookErrors() {
	t := s.T()

	h := createHostAndDeviceToken(t, s.ds, "good-token")

	var webhookCalled bool
	webhookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled = true
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer webhookSrv.Close()

	// patch app config with webhook url
	acResp := fleet.AppConfig{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"macos_migration": {
				"enable": true,
				"mode": "voluntary",
				"webhook_url": "%s/test_mdm_migration"
			}
		}
	}`, webhookSrv.URL)), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.MacOSMigration.Enable)

	isServer, enrolled, installedFromDEP := true, true, true
	mdmName := "ExampleMDM"
	mdmURL := "https://mdm.example.com"

	// host is enrolled to a third-party MDM but hasn't been assigned in
	// ABM yet, so migration is not allowed
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), h.ID, !isServer, enrolled, mdmURL, installedFromDEP, mdmName, "", fleet.PersonalEnrollmentTypeNone))
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.False(t, webhookCalled)

	s.enableABM(t.Name())
	// simulate that the device is assigned to Fleet in ABM
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/profile":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
			assert.NoError(t, err)
		case "/server/devices", "/devices/sync":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.DeviceResponse{
				Devices: []godep.Device{
					{
						SerialNumber: h.HardwareSerial,
						Model:        "Mac Mini",
						OS:           "osx",
						OpType:       "added",
					},
				},
			})
			assert.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var prof profileAssignmentReq
			assert.NoError(t, json.Unmarshal(b, &prof))
			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = map[string]string{
				prof.Devices[0]: string(fleet.DEPAssignProfileResponseSuccess),
			}
			encoder := json.NewEncoder(w)
			err = encoder.Encode(resp)
			assert.NoError(t, err)
		}
	}))
	s.runDEPSchedule()

	// hosts meets all requirements, webhook is run but returns an error, server should respond with
	// the same status code
	require.False(t, webhookCalled)
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "good-token"), nil, http.StatusBadRequest)
	require.True(t, webhookCalled)
}

func (s *integrationMDMTestSuite) TestMDMMigration() {
	t := s.T()
	ctx := context.Background()

	// enable migration
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": true, "mode": "voluntary", "webhook_url": "https://example.com" } }
	}`), http.StatusOK, &acResp)

	abmToken := s.enableABM(t.Name())

	checkMigrationResponses := func(host *fleet.Host, token string) {
		getDesktopResp := fleetDesktopResponse{}
		res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDesktopResp.Err)
		require.Zero(t, *getDesktopResp.FailingPolicies)
		require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, getDesktopResp.Notifications.RenewEnrollmentProfile)
		require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)                               //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
		require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground) //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
		require.Equal(t, acResp.OrgInfo.ContactURL, getDesktopResp.Config.OrgInfo.ContactURL)
		require.Equal(t, acResp.OrgInfo.OrgName, getDesktopResp.Config.OrgInfo.OrgName)
		require.Equal(t, acResp.MDM.MacOSMigration.Mode, getDesktopResp.Config.MDM.MacOSMigration.Mode)

		orbitConfigResp := fleet.OrbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.False(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)

		// simulate that the device is assigned to Fleet in ABM
		profileAssignmentStatusResponse := fleet.DEPAssignProfileResponseSuccess
		s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			switch r.URL.Path {
			case "/session":
				_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
			case "/profile":
				encoder := json.NewEncoder(w)
				err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
				assert.NoError(t, err)
			case "/server/devices", "/devices/sync":
				encoder := json.NewEncoder(w)
				err := encoder.Encode(godep.DeviceResponse{
					Devices: []godep.Device{
						{
							SerialNumber: host.HardwareSerial,
							Model:        "Mac Mini",
							OS:           "osx",
							OpType:       "added",
						},
					},
				})
				assert.NoError(t, err)
			case "/profile/devices":
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				var prof profileAssignmentReq
				assert.NoError(t, json.Unmarshal(b, &prof))
				var resp godep.ProfileResponse
				resp.ProfileUUID = prof.ProfileUUID
				resp.Devices = map[string]string{
					prof.Devices[0]: string(profileAssignmentStatusResponse),
				}
				encoder := json.NewEncoder(w)
				err = encoder.Encode(resp)
				assert.NoError(t, err)
			}
		}))

		cleanAssignmentStatus := func() {
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				stmt := `UPDATE host_dep_assignments
					 SET assign_profile_response = NULL,
					     response_updated_at = NULL,
					     profile_uuid = NULL
					 WHERE host_id = ?`
				_, err := q.ExecContext(context.Background(), stmt, host.ID)
				return err
			})
		}

		// simulate that the device is enrolled in a third-party MDM and DEP capable
		err := s.ds.SetOrUpdateMDMData(
			ctx,
			host.ID,
			false,
			true,
			"https://simplemdm.com",
			true,
			fleet.WellKnownMDMSimpleMDM,
			"",
			fleet.PersonalEnrollmentTypeNone,
		)
		require.NoError(t, err)

		// simulate a response before we have the chance to assign the profile
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDesktopResp.Err)
		require.Zero(t, *getDesktopResp.FailingPolicies)
		require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, getDesktopResp.Notifications.RenewEnrollmentProfile)
		require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)                               //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
		require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground) //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
		require.Equal(t, acResp.OrgInfo.ContactURL, getDesktopResp.Config.OrgInfo.ContactURL)
		require.Equal(t, acResp.OrgInfo.OrgName, getDesktopResp.Config.OrgInfo.OrgName)
		require.Equal(t, acResp.MDM.MacOSMigration.Mode, getDesktopResp.Config.MDM.MacOSMigration.Mode)
		orbitConfigResp = fleet.OrbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.False(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		cleanAssignmentStatus()

		// simulate a "FAILED" JSON profile assignment
		profileAssignmentStatusResponse = fleet.DEPAssignProfileResponseFailed
		s.runDEPSchedule()
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		orbitConfigResp = fleet.OrbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.False(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		require.NoError(t, s.ds.DeleteHostDEPAssignments(ctx, abmToken.ID, []string{host.HardwareSerial}))
		cleanAssignmentStatus()

		// simulate a "THROTTLED" JSON profile assignment
		profileAssignmentStatusResponse = fleet.DEPAssignProfileResponseThrottled
		s.runDEPSchedule()
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		orbitConfigResp = fleet.OrbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.False(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		require.NoError(t, s.ds.DeleteHostDEPAssignments(ctx, abmToken.ID, []string{host.HardwareSerial}))
		cleanAssignmentStatus()

		// simulate a "NOT_ACCESSIBLE" JSON profile assignment
		profileAssignmentStatusResponse = fleet.DEPAssignProfileResponseNotAccessible
		s.runDEPSchedule()
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.False(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		require.NoError(t, s.ds.DeleteHostDEPAssignments(ctx, abmToken.ID, []string{host.HardwareSerial}))
		cleanAssignmentStatus()

		// simulate a "SUCCESS" JSON profile assignment
		profileAssignmentStatusResponse = fleet.DEPAssignProfileResponseSuccess
		s.runDEPSchedule()
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.True(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.True(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)

		// simulate that the device needs to be enrolled in fleet, DEP capable
		err = s.ds.SetOrUpdateMDMData(
			ctx,
			host.ID,
			false,
			false,
			s.server.URL,
			true,
			fleet.WellKnownMDMFleet,
			"",
			fleet.PersonalEnrollmentTypeNone,
		)
		require.NoError(t, err)

		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDesktopResp.Err)
		require.Zero(t, *getDesktopResp.FailingPolicies)
		require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.True(t, getDesktopResp.Notifications.RenewEnrollmentProfile)
		require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)                               //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
		require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground) //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
		require.Equal(t, acResp.OrgInfo.ContactURL, getDesktopResp.Config.OrgInfo.ContactURL)
		require.Equal(t, acResp.OrgInfo.OrgName, getDesktopResp.Config.OrgInfo.OrgName)
		require.Equal(t, acResp.MDM.MacOSMigration.Mode, getDesktopResp.Config.MDM.MacOSMigration.Mode)

		orbitConfigResp = fleet.OrbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.False(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.True(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)

		// simulate that the device is manually enrolled into fleet, but DEP capable
		err = s.ds.SetOrUpdateMDMData(
			ctx,
			host.ID,
			false,
			true,
			s.server.URL,
			false,
			fleet.WellKnownMDMFleet,
			"",
			fleet.PersonalEnrollmentTypeNone,
		)
		require.NoError(t, err)
		mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
			SCEPChallenge: s.scepChallenge,
			SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
			MDMURL:        s.server.URL + apple_mdm.MDMPath,
		}, "MacBookPro16,1")
		mdmDevice.SerialNumber = host.HardwareSerial
		mdmDevice.UUID = host.UUID
		err = mdmDevice.Enroll()
		require.NoError(t, err)
		require.NoError(t, err)
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDesktopResp.Err)
		require.Zero(t, *getDesktopResp.FailingPolicies)
		require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, getDesktopResp.Notifications.RenewEnrollmentProfile)
		require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)                               //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
		require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground) //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
		require.Equal(t, acResp.OrgInfo.ContactURL, getDesktopResp.Config.OrgInfo.ContactURL)
		require.Equal(t, acResp.OrgInfo.OrgName, getDesktopResp.Config.OrgInfo.OrgName)
		require.Equal(t, acResp.MDM.MacOSMigration.Mode, getDesktopResp.Config.MDM.MacOSMigration.Mode)

		orbitConfigResp = fleet.OrbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.False(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)

		// simulate a host that was reset to factory, but fleet still has an mdm record active
		err = s.ds.SetOrUpdateMDMData(
			ctx,
			host.ID,
			false,
			true,
			s.server.URL,
			false,
			fleet.WellKnownMDMSimpleMDM,
			"",
			fleet.PersonalEnrollmentTypeNone,
		)
		require.NoError(t, err)
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDesktopResp.Err)
		require.Zero(t, *getDesktopResp.FailingPolicies)
		require.True(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, getDesktopResp.Notifications.RenewEnrollmentProfile)
		require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)                               //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
		require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground) //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
		require.Equal(t, acResp.OrgInfo.ContactURL, getDesktopResp.Config.OrgInfo.ContactURL)
		require.Equal(t, acResp.OrgInfo.OrgName, getDesktopResp.Config.OrgInfo.OrgName)
		require.Equal(t, acResp.MDM.MacOSMigration.Mode, getDesktopResp.Config.MDM.MacOSMigration.Mode)

		orbitConfigResp = fleet.OrbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.True(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)

		// simulate a device that is manually enrolled to 3rd party
		err = s.ds.SetOrUpdateMDMData(
			ctx,
			host.ID,
			false,
			true,
			"https://simplemdm.com",
			false,
			fleet.WellKnownMDMSimpleMDM,
			"",
			fleet.PersonalEnrollmentTypeNone,
		)
		require.NoError(t, err)
		getDesktopResp = fleetDesktopResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
		require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDesktopResp.Err)
		require.Zero(t, *getDesktopResp.FailingPolicies)
		require.True(t, getDesktopResp.Notifications.NeedsMDMMigration)
		require.False(t, getDesktopResp.Notifications.RenewEnrollmentProfile)
		require.Equal(t, acResp.OrgInfo.OrgLogoURL, getDesktopResp.Config.OrgInfo.OrgLogoURL)                               //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
		require.Equal(t, acResp.OrgInfo.OrgLogoURLLightBackground, getDesktopResp.Config.OrgInfo.OrgLogoURLLightBackground) //nolint:staticcheck // intentionally checks the deprecated field is still served to Fleet Desktop
		require.Equal(t, acResp.OrgInfo.ContactURL, getDesktopResp.Config.OrgInfo.ContactURL)
		require.Equal(t, acResp.OrgInfo.OrgName, getDesktopResp.Config.OrgInfo.OrgName)
		require.Equal(t, acResp.MDM.MacOSMigration.Mode, getDesktopResp.Config.MDM.MacOSMigration.Mode)

		orbitConfigResp = fleet.OrbitGetConfigResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &orbitConfigResp)
		require.True(t, orbitConfigResp.Notifications.NeedsMDMMigration)
		require.False(t, orbitConfigResp.Notifications.RenewEnrollmentProfile)

		// clean up nano tables
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), `
			DELETE FROM nano_enrollments WHERE id = ?
			`, host.UUID)
			return err
		})
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(), `
			DELETE FROM nano_devices WHERE id = ?
			`, host.UUID)
			return err
		})
	}

	token := "token_test_migration"
	host := createOrbitEnrolledHost(t, "darwin", "h", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, token)
	checkMigrationResponses(host, token)
	require.NoError(t, s.ds.DeleteHostDEPAssignments(ctx, abmToken.ID, []string{host.HardwareSerial}))

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team-1"})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm.ID, []uint{host.ID}))
	require.NoError(t, err)
	checkMigrationResponses(host, token)
}

func (s *integrationMDMTestSuite) TestSilentMigrationGotchas() {
	t := s.T()
	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "darwin", t.Name(), s.ds)
	// set the host as enrolled in a third-party MDM
	err := s.ds.SetOrUpdateMDMData(ctx, host.ID, true, true, "https://foo.com", false, fleet.WellKnownMDMSimpleMDM, "", fleet.PersonalEnrollmentTypeNone)
	require.NoError(t, err)

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host)
	require.NotNil(t, hostResp.Host.MDM.ConnectedToFleet)
	require.False(t, *hostResp.Host.MDM.ConnectedToFleet)

	// simulate that the device is assigned to Fleet in ABM
	s.enableABM(t.Name())
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/profile":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
			assert.NoError(t, err)
		case "/server/devices", "/devices/sync":
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.DeviceResponse{
				Devices: []godep.Device{
					{
						SerialNumber: host.HardwareSerial,
						Model:        "Mac Mini",
						OS:           "osx",
						OpType:       "added",
					},
				},
			})
			assert.NoError(t, err)

		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			var prof profileAssignmentReq
			assert.NoError(t, json.Unmarshal(b, &prof))
			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = map[string]string{
				prof.Devices[0]: string(fleet.DEPAssignProfileResponseSuccess),
			}
			encoder := json.NewEncoder(w)
			err = encoder.Encode(resp)
			assert.NoError(t, err)
		}
	}))
	s.runDEPSchedule()

	// enable migrations
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": true, "mode": "voluntary", "webhook_url": "https://example.com" } }
	}`), http.StatusOK, &acResp)

	// orbit config asks for a migration but not to renew enrollment profile
	resp := fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)
	require.True(t, resp.Notifications.NeedsMDMMigration)

	// simulate that's actually enrolled to Fleet under the hood
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	// by default the mdm test client will have a random uuid and serial, but we want
	// it to match with the previously created host
	mdmDevice.SerialNumber = host.HardwareSerial
	mdmDevice.UUID = host.UUID
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	// host response says that it's connected to Fleet (this just checks that the
	// device exists and is enabled in nano_enrollments, which it is thanks to
	// the mdmDevice.Enroll call above - the row is created during TokenUpdate).
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host)
	require.NotNil(t, hostResp.Host.MDM.ConnectedToFleet)
	require.True(t, *hostResp.Host.MDM.ConnectedToFleet)

	// orbit config asks for a migration because user migrations are enabled, but no ask to renew the enrollment profile.
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)
	require.True(t, resp.Notifications.NeedsMDMMigration)

	// set an enroll secret so the fleetd profile is delivered
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// trigger the profile cron
	s.awaitTriggerProfileSchedule(t)

	// explicitly run the worker, which will send the install fleetd command
	// because the device is ADE-enrolled (due to the simulation of it being
	// ingested from ABM)
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()

	var installEnterpriseCount int
	installs := [][]byte{}
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)

	for cmd != nil {
		if cmd.Command.RequestType == "DeclarativeManagement" {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
			continue
		}

		if cmd.Command.RequestType == "InstallEnterpriseApplication" {
			installEnterpriseCount++
		} else {
			require.Equal(t, "InstallProfile", cmd.Command.RequestType)
			t.Logf("Received InstallProfile command with payload:\n%s", string(cmd.Raw))
			installs = append(installs, cmd.Raw)
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	require.Equal(t, 1, installEnterpriseCount)
	require.Len(t, installs, 2) // fleetd profile and root cert profile

	// trigger the scep renewals cron
	cert, key, err := generateCertWithAPNsTopic()
	require.NoError(t, err)
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(s.T(), &fleetCfg, cert, key, "")
	scepLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	err = RenewSCEPCertificates(ctx, scepLogger, s.ds, &fleetCfg, s.mdmCommander, s.acmeSvc)
	require.NoError(t, err)

	// no new commands were enqueued
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// set the host as completely unenrolled
	err = s.ds.SetOrUpdateMDMData(ctx, host.ID, false, false, "", false, "", "", fleet.PersonalEnrollmentTypeNone)
	require.NoError(t, err)
	// orbit config asks to renew the enrollment profile, migration is not needed anymore so it's false
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &resp)
	require.True(t, resp.Notifications.RenewEnrollmentProfile)
	require.False(t, resp.Notifications.NeedsMDMMigration)

	// with migrations disabled, it still asks to renew the enrollment profile
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_migration": { "enable": false } }
	}`), http.StatusOK, &acResp)
	resp = fleet.OrbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &resp)
	require.True(t, resp.Notifications.RenewEnrollmentProfile)
	require.False(t, resp.Notifications.NeedsMDMMigration)
}
