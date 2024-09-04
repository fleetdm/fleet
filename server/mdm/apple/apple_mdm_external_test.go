package apple_mdm_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestDEPService_RunAssigner(t *testing.T) {
	ctx := context.Background()
	ds := mysql.CreateMySQLDS(t)

	const abmTokenOrgName = "test_org"
	depStorage, err := ds.NewMDMAppleDEPStorage()
	require.NoError(t, err)

	setupTest := func(t *testing.T, depHandler http.HandlerFunc) *apple_mdm.DEPService {
		// start a server that will mock the Apple DEP API
		srv := httptest.NewServer(depHandler)
		t.Cleanup(srv.Close)
		t.Cleanup(func() { mysql.TruncateTables(t, ds) })

		err = depStorage.StoreConfig(ctx, abmTokenOrgName, &nanodep_client.Config{BaseURL: srv.URL})
		require.NoError(t, err)

		mysql.SetTestABMAssets(t, ds, abmTokenOrgName)

		logger := log.NewNopLogger()
		return apple_mdm.NewDEPService(ds, depStorage, logger)
	}

	t.Run("no custom profiles, no devices", func(t *testing.T) {
		start := time.Now().Truncate(time.Second)

		svc := setupTest(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			encoder := json.NewEncoder(w)
			switch r.URL.Path {
			case "/session":
				_, _ = w.Write([]byte(`{"auth_session_token": "session123"}`))
			case "/account":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "admin123", "org_name": "%s"}`, abmTokenOrgName)))
			case "/profile":
				err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile123"})
				require.NoError(t, err)
			case "/server/devices":
				err := encoder.Encode(godep.DeviceResponse{Devices: nil})
				require.NoError(t, err)
			case "/devices/sync":
				err := encoder.Encode(godep.DeviceResponse{Devices: nil})
				require.NoError(t, err)
			default:
				t.Errorf("unexpected request to %s", r.URL.Path)
			}
		})
		err := svc.RunAssigner(ctx)
		require.NoError(t, err)

		// the default profile was created
		defProf, err := ds.GetMDMAppleEnrollmentProfileByType(ctx, fleet.MDMAppleEnrollmentTypeAutomatic)
		require.NoError(t, err)
		require.NotNil(t, defProf)
		require.NotEmpty(t, defProf.Token)

		// a profile UUID was assigned for no-team
		profUUID, modTime, err := ds.GetMDMAppleDefaultSetupAssistant(ctx, nil, abmTokenOrgName)
		require.NoError(t, err)
		require.Equal(t, "profile123", profUUID)
		require.False(t, modTime.Before(start))

		// no team to assign to
		appCfg, err := ds.AppConfig(ctx)
		require.NoError(t, err)
		require.Empty(t, appCfg.MDM.DeprecatedAppleBMDefaultTeam)
		abmTok, err := ds.GetABMTokenByOrgName(ctx, abmTokenOrgName)
		require.NoError(t, err)
		require.Nil(t, abmTok.MacOSDefaultTeamID)
		require.Nil(t, abmTok.IPadOSDefaultTeamID)
		require.Nil(t, abmTok.IOSDefaultTeamID)

		// no teams, so no team-specific custom setup assistants
		teams, err := ds.ListTeams(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.ListOptions{})
		require.NoError(t, err)
		require.Empty(t, teams)

		// no no-team custom setup assistant
		_, err = ds.GetMDMAppleSetupAssistant(ctx, nil)
		require.ErrorIs(t, err, sql.ErrNoRows)

		// no host got created
		hosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{})
		require.NoError(t, err)
		require.Empty(t, hosts)
	})

	t.Run("no custom profiles, some devices", func(t *testing.T) {
		start := time.Now().Truncate(time.Second)

		devices := []godep.Device{
			{SerialNumber: "a", OpType: "added"},
			{SerialNumber: "b", OpType: "ignore"},
			{SerialNumber: "c", OpType: ""},
		}

		var assignCalled bool
		svc := setupTest(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			encoder := json.NewEncoder(w)
			switch r.URL.Path {
			case "/session":
				_, _ = w.Write([]byte(`{"auth_session_token": "session123"}`))
			case "/account":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "admin123", "org_name": "%s"}`, abmTokenOrgName)))
			case "/profile":
				err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile123"})
				require.NoError(t, err)
			case "/server/devices":
				err := encoder.Encode(godep.DeviceResponse{Devices: devices})
				require.NoError(t, err)
			case "/devices/sync":
				err := encoder.Encode(godep.DeviceResponse{Devices: devices})
				require.NoError(t, err)
			case "/profile/devices":
				assignCalled = true

				reqBody, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				var assignReq godep.Profile
				err = json.Unmarshal(reqBody, &assignReq)
				require.NoError(t, err)
				require.Equal(t, assignReq.ProfileUUID, "profile123")
				require.ElementsMatch(t, []string{"a", "c"}, assignReq.Devices)

				_, _ = w.Write([]byte(`{}`))
			default:
				t.Errorf("unexpected request to %s", r.URL.Path)
			}
		})
		err := svc.RunAssigner(ctx)
		require.NoError(t, err)
		require.True(t, assignCalled)

		// the default profile was created
		defProf, err := ds.GetMDMAppleEnrollmentProfileByType(ctx, fleet.MDMAppleEnrollmentTypeAutomatic)
		require.NoError(t, err)
		require.NotNil(t, defProf)
		require.NotEmpty(t, defProf.Token)

		// a profile UUID was assigned to no-team
		profUUID, modTime, err := ds.GetMDMAppleDefaultSetupAssistant(ctx, nil, abmTokenOrgName)
		require.NoError(t, err)
		require.Equal(t, "profile123", profUUID)
		require.False(t, modTime.Before(start))

		// a couple hosts were created (except the op_type ignored)
		hosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{})
		require.NoError(t, err)
		require.Len(t, hosts, 2)
		serials := make([]string, len(hosts))
		for i, h := range hosts {
			serials[i] = h.HardwareSerial
			require.Nil(t, h.TeamID, h.HardwareSerial)
		}
		require.ElementsMatch(t, []string{"a", "c"}, serials)
	})

	t.Run("a custom profile, some devices", func(t *testing.T) {
		start := time.Now().Truncate(time.Second)

		devices := []godep.Device{
			{SerialNumber: "a", OpType: "added"},
			{SerialNumber: "b", OpType: "ignore"},
			{SerialNumber: "c", OpType: ""},
		}

		var assignCalled bool
		svc := setupTest(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			encoder := json.NewEncoder(w)
			switch r.URL.Path {
			case "/session":
				_, _ = w.Write([]byte(`{"auth_session_token": "session123"}`))
			case "/account":
				_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "admin123", "org_name": "%s"}`, abmTokenOrgName)))
			case "/profile":
				reqBody, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				var defineReq godep.Profile
				err = json.Unmarshal(reqBody, &defineReq)
				require.NoError(t, err)

				if defineReq.ProfileName == "team" {
					err = encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile456"})
					require.NoError(t, err)
				} else {
					err = encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile123"})
					require.NoError(t, err)
				}
			case "/server/devices":
				err := encoder.Encode(godep.DeviceResponse{Devices: devices})
				require.NoError(t, err)
			case "/devices/sync":
				err := encoder.Encode(godep.DeviceResponse{Devices: devices})
				require.NoError(t, err)
			case "/profile/devices":
				assignCalled = true

				reqBody, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				var assignReq godep.Profile
				err = json.Unmarshal(reqBody, &assignReq)
				require.NoError(t, err)
				require.Equal(t, assignReq.ProfileUUID, "profile456")
				require.ElementsMatch(t, []string{"a", "c"}, assignReq.Devices)

				_, _ = w.Write([]byte(`{}`))
			default:
				t.Errorf("unexpected request to %s", r.URL.Path)
			}
		})

		// create a team
		tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "test_team"})
		require.NoError(t, err)

		// set that team as default assignment for new macOS devices
		tok, err := ds.GetABMTokenByOrgName(ctx, abmTokenOrgName)
		require.NoError(t, err)
		tok.MacOSDefaultTeamID = &tm.ID
		err = ds.SaveABMToken(ctx, tok)
		require.NoError(t, err)

		// create a custom setup assistant for that team
		tmAsst, err := ds.SetOrUpdateMDMAppleSetupAssistant(ctx, &fleet.MDMAppleSetupAssistant{
			TeamID:  &tm.ID,
			Name:    "test",
			Profile: json.RawMessage(`{"profile_name": "team"}`),
		})
		require.NoError(t, err)
		require.NotZero(t, tmAsst.ID)

		err = svc.RunAssigner(ctx)
		require.NoError(t, err)
		require.True(t, assignCalled)

		// the default profile was created
		defProf, err := ds.GetMDMAppleEnrollmentProfileByType(ctx, fleet.MDMAppleEnrollmentTypeAutomatic)
		require.NoError(t, err)
		require.NotNil(t, defProf)
		require.NotEmpty(t, defProf.Token)

		// a profile UUID was assigned to the team
		profUUID, modTime, err := ds.GetMDMAppleDefaultSetupAssistant(ctx, &tm.ID, abmTokenOrgName)
		require.NoError(t, err)
		require.Equal(t, "profile123", profUUID)
		require.False(t, modTime.Before(start))

		// the team-specific custom profile was registered
		tmAsst, err = ds.GetMDMAppleSetupAssistant(ctx, tmAsst.TeamID)
		require.NoError(t, err)
		require.False(t, tmAsst.UploadedAt.Before(start))
		profileUUID, modTime, err := ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, &tm.ID, abmTokenOrgName)
		require.NoError(t, err)
		require.Equal(t, "profile456", profileUUID)
		require.True(t, tmAsst.UploadedAt.Equal(modTime))

		// a couple hosts were created and assigned to the team (except the op_type ignored)
		hosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{})
		require.NoError(t, err)
		require.Len(t, hosts, 2)
		serials := make([]string, len(hosts))
		for i, h := range hosts {
			serials[i] = h.HardwareSerial
			require.NotNil(t, h.TeamID, h.HardwareSerial)
			require.Equal(t, tm.ID, *h.TeamID, h.HardwareSerial)
		}
		require.ElementsMatch(t, []string{"a", "c"}, serials)
	})
}
