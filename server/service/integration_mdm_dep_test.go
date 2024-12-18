package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleetdbase"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/worker"
	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/groob/plist"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type profileAssignmentReq struct {
	ProfileUUID string   `json:"profile_uuid"`
	Devices     []string `json:"devices"`
}

func (s *integrationMDMTestSuite) TestDEPEnrollReleaseDeviceGlobal() {
	t := s.T()
	ctx := context.Background()
	s.enableABM(t.Name())
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "session123"}`))
		case "/account":
			_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "admin123", "org_name": "%s"}`, "foo")))
		case "/profile":
			w.WriteHeader(http.StatusOK)
			require.NoError(t, encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile123"}))
		}
	}))

	globalDevice := godep.Device{SerialNumber: uuid.New().String(), Model: "MacBook Pro", OS: "osx", OpType: "added"}

	// set an enroll secret, the Fleetd configuration profile will be installed
	// on the host
	enrollSecret := "test-release-dep-device"
	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: enrollSecret}})
	require.NoError(t, err)

	// add a valid bootstrap package
	b, err := os.ReadFile(filepath.Join("testdata", "bootstrap-packages", "signed.pkg"))
	require.NoError(t, err)
	signedPkg := b
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: signedPkg, Name: "pkg.pkg", TeamID: 0}, http.StatusOK, "")

	// add a custom setup assistant and ensure enable_release_device_manually is
	// false (the default)
	noTeamProf := `{"x": 1}`
	s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team",
		EnrollmentProfile: json.RawMessage(noTeamProf),
	}, http.StatusOK)
	payload := map[string]any{
		"enable_release_device_manually": false,
	}
	s.Do("PATCH", "/api/latest/fleet/setup_experience", json.RawMessage(jsonMustMarshal(t, payload)), http.StatusNoContent)

	// setup IdP so that AccountConfiguration profile is sent after DEP enrollment
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"end_user_authentication": {
					"entity_id": "https://localhost:8080",
					"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
					"idp_name": "SimpleSAML",
					"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
				},
				"macos_setup": {
					"enable_end_user_authentication": true
				}
			}
		}`), http.StatusOK, &acResp)
	require.NotEmpty(t, acResp.MDM.EndUserAuthentication)

	// TODO(mna): how/where to pass an enroll_reference so that
	// runPostDEPEnrollment sends an AccountConfiguration command?

	// add a global profile
	globalProfile := mobileconfigForTest("N1", "I1")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{globalProfile}}, http.StatusNoContent)

	// enable FileVault
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage([]byte(`{"mdm":{"macos_settings":{"enable_disk_encryption":true}}}`)), http.StatusOK)

	s.enableABM("fleet_ade_test")

	// add a setup experience script to run for no team
	extraArgs := make(map[string][]string)
	body, headers := generateNewScriptMultipartRequest(t,
		"script.sh", []byte(`echo "hello"`), s.token, extraArgs)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)

	// test manual and automatic release with the new setup experience flow
	for _, enableReleaseManually := range []bool{false, true} {
		t.Run(fmt.Sprintf("enableReleaseManually=%t;new_flow", enableReleaseManually), func(t *testing.T) {
			s.runDEPEnrollReleaseDeviceTest(t, globalDevice, enableReleaseManually, nil, "I1", false)
		})
	}
	// test manual and automatic release with the old worker flow
	for _, enableReleaseManually := range []bool{false, true} {
		t.Run(fmt.Sprintf("enableReleaseManually=%t;old_flow", enableReleaseManually), func(t *testing.T) {
			s.runDEPEnrollReleaseDeviceTest(t, globalDevice, enableReleaseManually, nil, "I1", true)
		})
	}

	// remove the setup experience script, run the new setup experience flow when
	// there is no setup experience item to process (so it is bypassed)
	s.Do("DELETE", "/api/latest/fleet/setup_experience/script", nil, http.StatusOK)
	for _, enableReleaseManually := range []bool{false, true} {
		t.Run(fmt.Sprintf("enableReleaseManually=%t;bypass_flow", enableReleaseManually), func(t *testing.T) {
			s.runDEPEnrollReleaseDeviceTest(t, globalDevice, enableReleaseManually, nil, "I1", false)
		})
	}
}

func (s *integrationMDMTestSuite) TestDEPEnrollReleaseDeviceTeam() {
	t := s.T()
	ctx := context.Background()

	// Set up a mock DEP Apple API
	s.enableABM(t.Name())
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "session123"}`))
		case "/account":
			_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "admin123", "org_name": "%s"}`, "foo")))
		case "/profile":
			require.NoError(t, encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile123"}))
		}
	}))

	teamDevice := godep.Device{SerialNumber: uuid.New().String(), Model: "MacBook Pro", OS: "osx", OpType: "added"}

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test-team-device-release"})
	require.NoError(t, err)

	// set an enroll secret, the Fleetd configuration profile will be installed
	// on the host
	enrollSecret := "test-release-dep-device-team"
	err = s.ds.ApplyEnrollSecrets(ctx, &tm.ID, []*fleet.EnrollSecret{{Secret: enrollSecret}})
	require.NoError(t, err)

	// add a valid bootstrap package
	b, err := os.ReadFile(filepath.Join("testdata", "bootstrap-packages", "signed.pkg"))
	require.NoError(t, err)
	signedPkg := b
	s.uploadBootstrapPackage(&fleet.MDMAppleBootstrapPackage{Bytes: signedPkg, Name: "pkg.pkg", TeamID: tm.ID}, http.StatusOK, "")

	// add a custom setup assistant and ensure enable_release_device_manually is
	// false (the default)
	teamProf := `{"y": 2}`
	s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team",
		EnrollmentProfile: json.RawMessage(teamProf),
	}, http.StatusOK)
	payload := map[string]any{
		"enable_release_device_manually": false,
	}
	s.Do("PATCH", "/api/latest/fleet/setup_experience", json.RawMessage(jsonMustMarshal(t, payload)), http.StatusNoContent)

	// setup IdP so that AccountConfiguration profile is sent after DEP enrollment
	var acResp appConfigResponse
	s.enableABM("fleet_ade_test")
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
			"mdm": {
			       "apple_business_manager": [{
			         "organization_name": %q,
			         "macos_team": %q,
			         "ios_team": %q,
			         "ipados_team": %q
			       }],
				"end_user_authentication": {
					"entity_id": "https://localhost:8080",
					"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
					"idp_name": "SimpleSAML",
					"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
				},
				"macos_setup": {
					"enable_end_user_authentication": true
				}
			}
		}`, "fleet_ade_test", tm.Name, tm.Name, tm.Name)), http.StatusOK, &acResp)
	require.NotEmpty(t, acResp.MDM.EndUserAuthentication)

	// TODO(mna): how/where to pass an enroll_reference so that
	// runPostDEPEnrollment sends an AccountConfiguration command?

	// add a team profile
	teamProfile := mobileconfigForTest("N2", "I2")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{teamProfile}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

	// enable FileVault
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings", json.RawMessage([]byte(fmt.Sprintf(`{"enable_disk_encryption":true,"team_id":%d}`, tm.ID))), http.StatusNoContent)

	// add a setup experience script to run for this team
	extraArgs := map[string][]string{
		"team_id": {fmt.Sprintf("%d", tm.ID)},
	}
	body, headers := generateNewScriptMultipartRequest(t,
		"script.sh", []byte(`echo "hello"`), s.token, extraArgs)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)

	// test manual and automatic release with the new setup experience flow
	for _, enableReleaseManually := range []bool{false, true} {
		t.Run(fmt.Sprintf("enableReleaseManually=%t;new_flow", enableReleaseManually), func(t *testing.T) {
			s.runDEPEnrollReleaseDeviceTest(t, teamDevice, enableReleaseManually, &tm.ID, "I2", false)
		})
	}
	// test manual and automatic release with the old worker flow
	for _, enableReleaseManually := range []bool{false, true} {
		t.Run(fmt.Sprintf("enableReleaseManually=%t;old_flow", enableReleaseManually), func(t *testing.T) {
			s.runDEPEnrollReleaseDeviceTest(t, teamDevice, enableReleaseManually, &tm.ID, "I2", true)
		})
	}

	// remove the setup experience script, run the new setup experience flow when
	// there is no setup experience item to process (so it is bypassed)
	s.Do("DELETE", "/api/latest/fleet/setup_experience/script", nil, http.StatusOK, "team_id", fmt.Sprint(tm.ID))
	for _, enableReleaseManually := range []bool{false, true} {
		t.Run(fmt.Sprintf("enableReleaseManually=%t;bypass_flow", enableReleaseManually), func(t *testing.T) {
			s.runDEPEnrollReleaseDeviceTest(t, teamDevice, enableReleaseManually, &tm.ID, "I2", false)
		})
	}
}

func (s *integrationMDMTestSuite) TestDEPEnrollReleaseIphoneTeam() {
	t := s.T()
	ctx := context.Background()

	// Set up a mock DEP Apple API
	s.enableABM(t.Name())
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "session123"}`))
		case "/account":
			_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "admin123", "org_name": "%s"}`, "foo")))
		case "/profile":
			require.NoError(t, encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile123"}))
		}
	}))

	teamDevice := godep.Device{SerialNumber: "IOS0_SERIAL", Model: "iPhone 16 Pro", OS: "ios", DeviceFamily: "iPhone", OpType: "added"}

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test-team-device-release"})
	require.NoError(t, err)

	enrollSecret := "test-release-dep-device-team"
	err = s.ds.ApplyEnrollSecrets(ctx, &tm.ID, []*fleet.EnrollSecret{{Secret: enrollSecret}})
	require.NoError(t, err)

	// add a custom setup assistant and ensure enable_release_device_manually is
	// false (the default)
	teamProf := `{"y": 2}`
	s.Do("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              "team",
		EnrollmentProfile: json.RawMessage(teamProf),
	}, http.StatusOK)
	payload := map[string]any{
		"enable_release_device_manually": false,
	}
	s.Do("PATCH", "/api/latest/fleet/setup_experience", json.RawMessage(jsonMustMarshal(t, payload)), http.StatusNoContent)

	var acResp appConfigResponse
	s.enableABM("fleet_ade_test")
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
			"mdm": {
			       "apple_business_manager": [{
			         "organization_name": %q,
			         "macos_team": %q,
			         "ios_team": %q,
			         "ipados_team": %q
			       }]
			}
		}`, "fleet_ade_test", tm.Name, tm.Name, tm.Name)), http.StatusOK, &acResp)

	// add a team profile
	teamProfile := mobileconfigForTest("N2", "I2")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{teamProfile}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

	for _, enableReleaseManually := range []bool{false, true} {
		t.Run(fmt.Sprintf("enableReleaseManually=%t", enableReleaseManually), func(t *testing.T) {
			s.runDEPEnrollReleaseDeviceTest(t, teamDevice, enableReleaseManually, &tm.ID, "I2", false)
		})
	}
}

func (s *integrationMDMTestSuite) runDEPEnrollReleaseDeviceTest(t *testing.T, device godep.Device, enableReleaseManually bool, teamID *uint, customProfileIdent string, useOldFleetdFlow bool) {
	ctx := context.Background()

	var isIphone bool
	if device.DeviceFamily == "iPhone" {
		isIphone = true
	}

	// set the enable release device manually option
	payload := map[string]any{
		"enable_release_device_manually": enableReleaseManually,
	}
	if teamID != nil {
		payload["team_id"] = *teamID
	}
	s.Do("PATCH", "/api/latest/fleet/setup_experience", json.RawMessage(jsonMustMarshal(t, payload)), http.StatusNoContent)

	// query all hosts - none yet
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Empty(t, listHostsRes.Hosts)

	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		return map[string]*push.Response{}, nil
	}

	s.mockDEPResponse("fleet_ade_test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			require.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()})
			require.NoError(t, err)
		case "/server/devices":
			err := encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{device}})
			require.NoError(t, err)
		case "/devices/sync":
			// This endpoint is polled over time to sync devices from
			// ABM, send a repeated serial and a new one
			err := encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{device}, Cursor: "foo"})
			require.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))

			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = make(map[string]string, len(prof.Devices))
			for _, device := range prof.Devices {
				resp.Devices[device] = string(fleet.DEPAssignProfileResponseSuccess)
			}
			err = encoder.Encode(resp)
			require.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	// trigger a profile sync
	s.runDEPSchedule()

	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	require.Equal(t, listHostsRes.Hosts[0].HardwareSerial, device.SerialNumber)
	enrolledHost := listHostsRes.Hosts[0].Host

	t.Cleanup(func() {
		// delete the enrolled host
		err := s.ds.DeleteHost(ctx, enrolledHost.ID)
		require.NoError(t, err)
	})

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	if isIphone {
		mdmDevice.Model = "iPhone 14,6"
	}
	mdmDevice.SerialNumber = device.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// check if it has setup experience items or not
	hasSetupExpItems := true
	_, err = s.ds.GetHostAwaitingConfiguration(ctx, mdmDevice.UUID)
	if fleet.IsNotFound(err) {
		hasSetupExpItems = false
	} else if err != nil {
		require.NoError(t, err)
	}

	// run the worker to process the DEP enroll request
	s.runWorker()
	// run the cron to assign configuration profiles
	s.awaitTriggerProfileSchedule(t)

	var cmds []*micromdm.CommandPayload
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {

		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

		// Can be useful for debugging
		// switch cmd.Command.RequestType {
		// case "InstallProfile":
		// 	fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType, string(fullCmd.Command.InstallProfile.Payload))
		// case "InstallEnterpriseApplication":
		// 	if fullCmd.Command.InstallEnterpriseApplication.ManifestURL != nil {
		// 		fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType, *fullCmd.Command.InstallEnterpriseApplication.ManifestURL)
		// 	} else {
		// 		fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType)
		// 	}
		// default:
		// 	fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType)
		// }

		cmds = append(cmds, &fullCmd)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	if isIphone {
		// expected commands: install CA, install profile (only the custom one),
		// not expected: account configuration, since enrollment_reference not set
		require.Len(t, cmds, 2)
	} else {
		// expected commands: install fleetd, install bootstrap, install CA, install profiles
		// (custom one, fleetd configuration, FileVault) (not expected: account
		// configuration, since enrollment_reference not set)
		require.Len(t, cmds, 6)
	}

	var installProfileCount, installEnterpriseCount, otherCount int
	var profileCustomSeen, profileFleetdSeen, profileFleetCASeen, profileFileVaultSeen bool
	for _, cmd := range cmds {
		switch cmd.Command.RequestType {
		case "InstallProfile":
			installProfileCount++
			if strings.Contains(string(cmd.Command.InstallProfile.Payload), //nolint:gocritic // ignore ifElseChain
				fmt.Sprintf("<string>%s</string>", customProfileIdent)) {
				profileCustomSeen = true
			} else if strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetdConfigPayloadIdentifier)) {
				profileFleetdSeen = true
			} else if strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetCARootConfigPayloadIdentifier)) {
				profileFleetCASeen = true
			} else if strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string", mobileconfig.FleetFileVaultPayloadIdentifier)) &&
				strings.Contains(string(cmd.Command.InstallProfile.Payload), "ForceEnableInSetupAssistant") {
				profileFileVaultSeen = true
			}

		case "InstallEnterpriseApplication":
			installEnterpriseCount++
		default:
			otherCount++
		}
	}

	if isIphone {
		require.Equal(t, 2, installProfileCount)
		require.Equal(t, 0, installEnterpriseCount)
		require.Equal(t, 0, otherCount)
		require.True(t, profileCustomSeen)
		require.False(t, profileFleetdSeen)
		require.True(t, profileFleetCASeen)
		require.False(t, profileFileVaultSeen)

		// for iDevices, fleetd is not installed so the rest of this test does not apply.
		if enableReleaseManually {
			// get the worker's pending job from the future, there should not be any
			// because it needs to be released manually
			pending, err := s.ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute))
			require.NoError(t, err)
			require.Empty(t, pending)
		} else {
			// otherwise the device release job should be enqueued
			pending, err := s.ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute))
			require.NoError(t, err)
			require.Len(t, pending, 1)
			require.Equal(t, "apple_mdm", pending[0].Name)
			require.Contains(t, string(*pending[0].Args), worker.AppleMDMPostDEPReleaseDeviceTask)

			// make the pending job ready to run immediately and run the job
			mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx, `UPDATE jobs SET not_before = ? WHERE id = ?`, time.Now().Add(-1*time.Minute).UTC(), pending[0].ID)
				return err
			})

			s.runWorker()

			// make the device process the commands, it should receive the
			// DeviceConfigured one.
			cmds = cmds[:0]
			cmd, err = mdmDevice.Idle()
			require.NoError(t, err)
			for cmd != nil {
				var fullCmd micromdm.CommandPayload
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
				cmds = append(cmds, &fullCmd)
				cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
				require.NoError(t, err)
			}

			require.Len(t, cmds, 1)
			var deviceConfiguredCount int
			for _, cmd := range cmds {
				switch cmd.Command.RequestType {
				case "DeviceConfigured":
					deviceConfiguredCount++
				default:
					otherCount++
				}
			}
			require.Equal(t, 1, deviceConfiguredCount)
			require.Equal(t, 0, otherCount)
		}
		return
	}

	require.Equal(t, 4, installProfileCount)
	require.Equal(t, 2, installEnterpriseCount)
	require.Equal(t, 0, otherCount)
	require.True(t, profileCustomSeen)
	require.True(t, profileFleetdSeen)
	require.True(t, profileFleetCASeen)
	require.True(t, profileFileVaultSeen)

	// simulate fleetd being installed and the host being orbit-enrolled now
	enrolledHost.OsqueryHostID = ptr.String(mdmDevice.UUID)
	orbitKey := setOrbitEnrollment(t, enrolledHost, s.ds)
	enrolledHost.OrbitNodeKey = &orbitKey

	// call the /config endpoint as fleetd would
	var orbitConfigResp orbitGetConfigResponse
	var caps fleet.CapabilityMap
	if useOldFleetdFlow {
		// important thing is that it doesn't have the CapabilitySetupExperience
		caps.PopulateFromString(string(fleet.CapabilityEscrowBuddy))
	} else {
		caps = fleet.GetOrbitClientCapabilities()
	}

	res := s.DoRawWithHeaders("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)),
		http.StatusOK, map[string]string{fleet.CapabilitiesHeader: caps.String()})
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &orbitConfigResp))
	if hasSetupExpItems {
		// should be notified of the setup experience flow
		require.True(t, orbitConfigResp.Notifications.RunSetupExperience)
	} else {
		// should bypass the setup experience flow
		require.False(t, orbitConfigResp.Notifications.RunSetupExperience)
	}

	if enableReleaseManually {
		// get the worker's pending job from the future, there should not be any
		// because it needs to be released manually
		pending, err := s.ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute))
		require.NoError(t, err)
		require.Empty(t, pending)
		return
	}

	if useOldFleetdFlow || !hasSetupExpItems {
		// there should be a Release Device pending job
		pending, err := s.ds.GetQueuedJobs(ctx, 2, time.Now().UTC().Add(time.Minute))
		require.NoError(t, err)
		require.Len(t, pending, 1)
		require.Equal(t, "apple_mdm", pending[0].Name)
		require.Contains(t, string(*pending[0].Args), worker.AppleMDMPostDEPReleaseDeviceTask)

		// calling the orbit config endpoint again does NOT enqueue a new job, and doesn't
		// return the RunSetupExperience notification anymore
		orbitConfigResp = orbitGetConfigResponse{}
		res := s.DoRawWithHeaders("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)),
			http.StatusOK, map[string]string{fleet.CapabilitiesHeader: caps.String()})
		b, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(b, &orbitConfigResp))
		require.False(t, orbitConfigResp.Notifications.RunSetupExperience)

		pending, err = s.ds.GetQueuedJobs(ctx, 2, time.Now().UTC().Add(time.Minute))
		require.NoError(t, err)
		require.Len(t, pending, 1)

		// make the pending job ready to run immediately and run the job
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE jobs SET not_before = ? WHERE id = ?`, time.Now().Add(-1*time.Minute).UTC(), pending[0].ID)
			return err
		})

		s.runWorker()

	} else {

		// there shouldn't be a Release Device pending job anymore
		pending, err := s.ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute))
		require.NoError(t, err)
		require.Len(t, pending, 0)

		// mark the setup experience script as done
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE setup_experience_status_results SET status = 'success' WHERE host_uuid = ?`, mdmDevice.UUID)
			return err
		})

		// call the /status endpoint to automatically release the host
		var statusResp getOrbitSetupExperienceStatusResponse
		s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	}

	// make the device process the commands, it should receive the
	// DeviceConfigured one.
	cmds = cmds[:0]
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
		cmds = append(cmds, &fullCmd)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	require.Len(t, cmds, 1)
	var deviceConfiguredCount int
	for _, cmd := range cmds {
		switch cmd.Command.RequestType {
		case "DeviceConfigured":
			deviceConfiguredCount++
		default:
			otherCount++
		}
	}
	require.Equal(t, 1, deviceConfiguredCount)
	require.Equal(t, 0, otherCount)
}

func (s *integrationMDMTestSuite) TestDEPProfileAssignment() {
	t := s.T()

	ctx := context.Background()
	devices := []godep.Device{
		{SerialNumber: uuid.New().String(), Model: "MacBook Pro", OS: "osx", OpType: "added"},
		{SerialNumber: uuid.New().String(), Model: "MacBook Mini", OS: "osx", OpType: "added"},
		{SerialNumber: uuid.New().String(), Model: "MacBook Mini", OS: "osx", OpType: ""},
		{SerialNumber: uuid.New().String(), Model: "MacBook Mini", OS: "osx", OpType: "modified"},
	}

	// set release device manually to true so there is no job enqueued at a later
	// time to release the device (this is not what this test is about)
	s.Do("PATCH", "/api/latest/fleet/setup_experience", json.RawMessage(jsonMustMarshal(t, map[string]any{
		"enable_release_device_manually": true,
	})), http.StatusNoContent)

	profileAssignmentReqs := []profileAssignmentReq{}

	// add global profiles
	globalProfile := mobileconfigForTest("N1", "I1")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{globalProfile}}, http.StatusNoContent)

	checkPostEnrollmentCommands := func(mdmDevice *mdmtest.TestAppleMDMClient, shouldReceive bool) {
		// run the worker to process the DEP enroll request
		s.runWorker()
		// run the worker to assign configuration profiles
		s.awaitTriggerProfileSchedule(t)

		var fleetdCmd, installProfileCmd *micromdm.CommandPayload
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			if fullCmd.Command.RequestType == "InstallEnterpriseApplication" &&
				fullCmd.Command.InstallEnterpriseApplication.ManifestURL != nil &&
				strings.Contains(*fullCmd.Command.InstallEnterpriseApplication.ManifestURL, fleetdbase.GetPKGManifestURL()) {
				fleetdCmd = &fullCmd
			} else if cmd.Command.RequestType == "InstallProfile" {
				installProfileCmd = &fullCmd
			}
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}

		if shouldReceive {
			// received request to install fleetd
			require.NotNil(t, fleetdCmd, "host didn't get a command to install fleetd")
			require.NotNil(t, fleetdCmd.Command, "host didn't get a command to install fleetd")

			// received request to install the global configuration profile
			require.NotNil(t, installProfileCmd, "host didn't get a command to install profiles")
			require.NotNil(t, installProfileCmd.Command, "host didn't get a command to install profiles")
		} else {
			require.Nil(t, fleetdCmd, "host got a command to install fleetd")
			require.Nil(t, installProfileCmd, "host got a command to install profiles")
		}
	}

	checkAssignProfileRequests := func(serial string, profUUID *string) {
		require.NotEmpty(t, profileAssignmentReqs)
		require.Len(t, profileAssignmentReqs, 1)
		require.Len(t, profileAssignmentReqs[0].Devices, 1)
		require.Equal(t, serial, profileAssignmentReqs[0].Devices[0])
		if profUUID != nil {
			require.Equal(t, *profUUID, profileAssignmentReqs[0].ProfileUUID)
		}
	}

	type hostDEPRow struct {
		HostID                uint       `db:"host_id"`
		ProfileUUID           string     `db:"profile_uuid"`
		AssignProfileResponse string     `db:"assign_profile_response"`
		ResponseUpdatedAt     time.Time  `db:"response_updated_at"`
		RetryJobID            uint       `db:"retry_job_id"`
		DeletedAt             *time.Time `db:"deleted_at"`
	}
	checkHostDEPAssignProfileResponses := func(deviceSerials []string, expectedProfileUUID string, expectedStatus fleet.DEPAssignProfileResponseStatus) map[string]hostDEPRow {
		bySerial := make(map[string]hostDEPRow, len(deviceSerials))
		for _, deviceSerial := range deviceSerials {
			mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				var dest hostDEPRow
				err := sqlx.GetContext(ctx, q, &dest, "SELECT host_id, assign_profile_response, profile_uuid, response_updated_at, retry_job_id, deleted_at FROM host_dep_assignments WHERE profile_uuid = ? AND host_id = (SELECT id FROM hosts WHERE hardware_serial = ?)", expectedProfileUUID, deviceSerial)
				require.NoError(t, err)
				require.Equal(t, string(expectedStatus), dest.AssignProfileResponse)
				bySerial[deviceSerial] = dest
				return nil
			})
		}
		return bySerial
	}

	checkPendingMacOSSetupAssistantJob := func(expectedTask string, expectedTeamID *uint, expectedSerials []string, expectedJobID uint) {
		pending, err := s.ds.GetQueuedJobs(context.Background(), 1, time.Time{})
		require.NoError(t, err)
		require.Len(t, pending, 1)
		require.Equal(t, "macos_setup_assistant", pending[0].Name)
		require.NotNil(t, pending[0].Args)
		var gotArgs struct {
			Task              string   `json:"task"`
			TeamID            *uint    `json:"team_id,omitempty"`
			HostSerialNumbers []string `json:"host_serial_numbers,omitempty"`
		}
		require.NoError(t, json.Unmarshal(*pending[0].Args, &gotArgs))
		require.Equal(t, expectedTask, gotArgs.Task)
		if expectedTeamID != nil {
			require.NotNil(t, gotArgs.TeamID)
			require.Equal(t, *expectedTeamID, *gotArgs.TeamID)
		} else {
			require.Nil(t, gotArgs.TeamID)
		}
		require.Equal(t, expectedSerials, gotArgs.HostSerialNumbers)

		if expectedJobID != 0 {
			require.Equal(t, expectedJobID, pending[0].ID)
		}
	}

	checkNoJobsPending := func() {
		pending, err := s.ds.GetQueuedJobs(context.Background(), 1, time.Time{})
		require.NoError(t, err)
		if len(pending) > 0 {
			t.Logf("found unexpected pending job %s scheduled for %v ('now' is %v):\n%s\n", pending[0].Name, pending[0].NotBefore, time.Now().UTC(), string(*pending[0].Args))
		}
		require.Empty(t, pending)
	}

	expectNoJobID := ptr.Uint(0) // used when expect no retry job
	checkHostCooldown := func(serial, profUUID string, status fleet.DEPAssignProfileResponseStatus, expectUpdatedAt *time.Time, expectRetryJobID *uint) hostDEPRow {
		bySerial := checkHostDEPAssignProfileResponses([]string{serial}, profUUID, status)
		d, ok := bySerial[serial]
		require.True(t, ok)
		if expectUpdatedAt != nil {
			require.Equal(t, *expectUpdatedAt, d.ResponseUpdatedAt)
		}
		if expectRetryJobID != nil {
			require.Equal(t, *expectRetryJobID, d.RetryJobID)
		}
		return d
	}

	checkListHostDEPError := func(serial string, expectStatus string, expectError bool) *fleet.HostResponse {
		listHostsRes := listHostsResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts?query=%s", serial), nil, http.StatusOK, &listHostsRes)
		require.Len(t, listHostsRes.Hosts, 1)
		require.Equal(t, serial, listHostsRes.Hosts[0].HardwareSerial)
		require.Equal(t, expectStatus, *listHostsRes.Hosts[0].MDM.EnrollmentStatus)
		require.Equal(t, expectError, listHostsRes.Hosts[0].MDM.DEPProfileError)

		return &listHostsRes.Hosts[0]
	}

	setAssignProfileResponseUpdatedAt := func(serial string, updatedAt time.Time) {
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE host_dep_assignments SET response_updated_at = ? WHERE host_id = (SELECT id FROM hosts WHERE hardware_serial = ?)`, updatedAt, serial)
			return err
		})
	}

	expectAssignProfileResponseFailed := ""        // set to device serial when testing the failed profile assignment flow
	expectAssignProfileResponseNotAccessible := "" // set to device serial when testing the not accessible profile assignment flow

	s.enableABM(t.Name())
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			require.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()})
			require.NoError(t, err)
		case "/server/devices":
			// This endpoint  is used to get an initial list of
			// devices, return a single device
			err := encoder.Encode(godep.DeviceResponse{Devices: devices[:1]})
			require.NoError(t, err)
		case "/devices/sync":
			// This endpoint is polled over time to sync devices from
			// ABM, send a repeated serial and a new one
			err := encoder.Encode(godep.DeviceResponse{Devices: devices, Cursor: "foo"})
			require.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))
			profileAssignmentReqs = append(profileAssignmentReqs, prof)
			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = make(map[string]string, len(prof.Devices))
			for _, device := range prof.Devices {
				switch device {
				case expectAssignProfileResponseNotAccessible:
					resp.Devices[device] = string(fleet.DEPAssignProfileResponseNotAccessible)
				case expectAssignProfileResponseFailed:
					resp.Devices[device] = string(fleet.DEPAssignProfileResponseFailed)
				default:
					resp.Devices[device] = string(fleet.DEPAssignProfileResponseSuccess)
				}
			}
			err = encoder.Encode(resp)
			require.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	// query all hosts
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Empty(t, listHostsRes.Hosts)

	// trigger a profile sync
	s.runDEPSchedule()

	// all hosts should be returned from the hosts endpoint
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, len(devices))
	var wantSerials []string
	var gotSerials []string
	for i, device := range devices {
		wantSerials = append(wantSerials, device.SerialNumber)
		gotSerials = append(gotSerials, listHostsRes.Hosts[i].HardwareSerial)
		// entries for all hosts should be created in the host_dep_assignments table
		_, err := s.ds.GetHostDEPAssignment(ctx, listHostsRes.Hosts[i].ID)
		require.NoError(t, err)
	}
	require.ElementsMatch(t, wantSerials, gotSerials)
	// called two times:
	// - one when we get the initial list of devices (/server/devices)
	// - one when we do the device sync (/device/sync)
	require.Len(t, profileAssignmentReqs, 2)
	require.Len(t, profileAssignmentReqs[0].Devices, 1)
	checkHostDEPAssignProfileResponses(profileAssignmentReqs[0].Devices, profileAssignmentReqs[0].ProfileUUID, fleet.DEPAssignProfileResponseSuccess)
	require.Len(t, profileAssignmentReqs[1].Devices, len(devices))
	checkHostDEPAssignProfileResponses(profileAssignmentReqs[1].Devices, profileAssignmentReqs[1].ProfileUUID, fleet.DEPAssignProfileResponseSuccess)
	// record the default profile to be used in other tests
	defaultProfileUUID := profileAssignmentReqs[1].ProfileUUID

	// create a new host
	nonDEPHost := createHostAndDeviceToken(t, s.ds, "not-dep")
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, len(devices)+1)

	// filtering by MDM status works
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts?mdm_enrollment_status=pending", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, len(devices))

	// searching by display name works
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts?query=%s", url.QueryEscape("MacBook Mini")), nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 3)
	for _, host := range listHostsRes.Hosts {
		require.Equal(t, "MacBook Mini", host.HardwareModel)
		require.Equal(t, host.DisplayName, fmt.Sprintf("MacBook Mini (%s)", host.HardwareSerial))
	}

	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		return map[string]*push.Response{}, nil
	}

	// Enroll one of the hosts
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = devices[0].SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// make sure the host gets post enrollment requests
	checkPostEnrollmentCommands(mdmDevice, true)

	// only one shows up as pending
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts?mdm_enrollment_status=pending", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, len(devices)-1)

	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities, "order_key", "created_at")
	found := false
	for _, activity := range activities.Activities {
		if activity.Type == "mdm_enrolled" &&
			strings.Contains(string(*activity.Details), devices[0].SerialNumber) {
			found = true
			require.Nil(t, activity.ActorID)
			require.Nil(t, activity.ActorFullName)
			require.JSONEq(
				t,
				fmt.Sprintf(
					`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": true, "mdm_platform": "apple"}`,
					devices[0].SerialNumber, devices[0].Model, devices[0].SerialNumber,
				),
				string(*activity.Details),
			)
		}
	}
	require.True(t, found)

	// add devices[1].SerialNumber to a team
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team
	var device1ID uint
	for _, h := range listHostsRes.Hosts {
		if h.HardwareSerial == devices[1].SerialNumber {
			err = s.ds.AddHostsToTeam(ctx, &team.ID, []uint{h.ID})
			require.NoError(t, err)
			device1ID = h.ID
			break
		}
	}

	// modify the response and trigger another sync to include:
	//
	// 1. A repeated device with "added"
	// 2. A repeated device with "modified"
	// 3. A device with "deleted"
	// 4. A new device
	deletedSerial := devices[2].SerialNumber
	addedSerial := uuid.New().String()
	devices = []godep.Device{
		{SerialNumber: devices[0].SerialNumber, Model: "MacBook Pro", OS: "osx", OpType: "added"},
		{SerialNumber: devices[1].SerialNumber, Model: "MacBook Mini", OS: "osx", OpType: "modified"},
		{SerialNumber: deletedSerial, Model: "MacBook Mini", OS: "osx", OpType: "deleted"},
		{SerialNumber: addedSerial, Model: "MacBook Mini", OS: "osx", OpType: "added"},
	}
	profileAssignmentReqs = []profileAssignmentReq{}

	// Enroll the host to be deleted. It will stay in Fleet after deletion from DEP.
	mdmDeviceToDelete := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDeviceToDelete.SerialNumber = deletedSerial
	require.NoError(t, mdmDeviceToDelete.Enroll())
	// make sure the host gets post enrollment requests
	checkPostEnrollmentCommands(mdmDeviceToDelete, true)

	s.runDEPSchedule()

	// all hosts should be returned from the hosts endpoint
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	// all previous devices + the manually added host + the new `addedSerial`
	wantSerials = append(wantSerials, devices[3].SerialNumber, nonDEPHost.HardwareSerial)
	require.Len(t, listHostsRes.Hosts, len(wantSerials))
	gotSerials = []string{}
	var deletedHostID uint
	var addedHostID uint
	var mdmDeviceID uint
	for _, device := range listHostsRes.Hosts {
		gotSerials = append(gotSerials, device.HardwareSerial)
		switch device.HardwareSerial {
		case deletedSerial:
			deletedHostID = device.ID
		case addedSerial:
			addedHostID = device.ID
		case mdmDevice.SerialNumber:
			mdmDeviceID = device.ID
		}
	}
	require.ElementsMatch(t, wantSerials, gotSerials)
	require.Len(t, profileAssignmentReqs, 3)

	// first request to get a list of profiles
	// TODO: seems like we're doing this request on each loop?
	require.Len(t, profileAssignmentReqs[0].Devices, 1)
	require.Equal(t, devices[0].SerialNumber, profileAssignmentReqs[0].Devices[0])
	checkHostDEPAssignProfileResponses(profileAssignmentReqs[0].Devices, profileAssignmentReqs[0].ProfileUUID, fleet.DEPAssignProfileResponseSuccess)

	// profileAssignmentReqs[1] and [2] can be in any order
	ix2Devices, ix1Device := 1, 2
	if len(profileAssignmentReqs[1].Devices) == 1 {
		ix2Devices, ix1Device = ix1Device, ix2Devices
	}

	// - existing device with "added"
	// - new device with "added"
	require.Len(t, profileAssignmentReqs[ix2Devices].Devices, 2, "%#+v", profileAssignmentReqs)
	require.ElementsMatch(t, []string{devices[0].SerialNumber, addedSerial}, profileAssignmentReqs[ix2Devices].Devices)
	checkHostDEPAssignProfileResponses(profileAssignmentReqs[ix2Devices].Devices, profileAssignmentReqs[ix2Devices].ProfileUUID, fleet.DEPAssignProfileResponseSuccess)

	// - existing device with "modified" and a different team (thus different profile request)
	require.Len(t, profileAssignmentReqs[ix1Device].Devices, 1)
	require.Equal(t, devices[1].SerialNumber, profileAssignmentReqs[ix1Device].Devices[0])
	checkHostDEPAssignProfileResponses(profileAssignmentReqs[ix1Device].Devices, profileAssignmentReqs[ix1Device].ProfileUUID, fleet.DEPAssignProfileResponseSuccess)

	// entries for all hosts except for the one with OpType = "deleted"
	assignment, err := s.ds.GetHostDEPAssignment(ctx, deletedHostID)
	require.NoError(t, err)
	require.NotZero(t, assignment.DeletedAt)

	_, err = s.ds.GetHostDEPAssignment(ctx, addedHostID)
	require.NoError(t, err)

	// send a TokenUpdate command, it shouldn't re-send the post-enrollment commands
	err = mdmDevice.TokenUpdate(false)
	require.NoError(t, err)
	checkPostEnrollmentCommands(mdmDevice, false)

	// enroll the device again, it should get the post-enrollment commands
	err = mdmDevice.Enroll()
	require.NoError(t, err)
	checkPostEnrollmentCommands(mdmDevice, true)

	// delete the device from Fleet
	var delResp deleteHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", mdmDeviceID), nil, http.StatusOK, &delResp)

	// the device comes back as pending
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts?query=%s", mdmDevice.UUID), nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	require.Equal(t, mdmDevice.SerialNumber, listHostsRes.Hosts[0].HardwareSerial)

	// we assign a DEP profile to the device
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runWorker()
	require.Equal(t, mdmDevice.SerialNumber, profileAssignmentReqs[0].Devices[0])
	checkHostDEPAssignProfileResponses(profileAssignmentReqs[0].Devices, profileAssignmentReqs[0].ProfileUUID, fleet.DEPAssignProfileResponseSuccess)

	// it should get the post-enrollment commands
	require.NoError(t, mdmDevice.Enroll())
	checkPostEnrollmentCommands(mdmDevice, true)

	// Delete a pending device from DEP
	addedModifiedDeletedSerial := uuid.NewString() // no-op
	deletedAddedSerial := devices[0].SerialNumber  // stay as is
	deletedSerial = devices[1].SerialNumber
	devices = []godep.Device{
		{SerialNumber: addedModifiedDeletedSerial, Model: "MacBook Pro", OS: "osx", OpType: "added", OpDate: time.Now()},
		{
			SerialNumber: addedModifiedDeletedSerial, Model: "MacBook Pro", OS: "osx", OpType: "modified",
			OpDate: time.Now().Add(time.Second),
		},
		{
			SerialNumber: addedModifiedDeletedSerial, Model: "MacBook Pro", OS: "osx", OpType: "deleted",
			OpDate: time.Now().Add(2 * time.Second),
		},
		{
			SerialNumber: addedModifiedDeletedSerial, Model: "MacBook Pro", OS: "osx", OpType: "added",
			OpDate: time.Now().Add(3 * time.Second),
		},
		{
			SerialNumber: addedModifiedDeletedSerial, Model: "MacBook Pro", OS: "osx", OpType: "deleted",
			OpDate: time.Now().Add(4 * time.Second),
		},

		{SerialNumber: deletedAddedSerial, Model: "MacBook Pro", OS: "osx", OpType: "deleted", OpDate: time.Now()},
		{SerialNumber: deletedAddedSerial, Model: "MacBook Pro", OS: "osx", OpType: "added", OpDate: time.Now().Add(time.Second)},
		{SerialNumber: deletedAddedSerial, Model: "MacBook Pro", OS: "osx", OpType: "added", OpDate: time.Now().Add(2 * time.Second)},

		{SerialNumber: deletedSerial, Model: "MacBook Mini", OS: "osx", OpType: "modified", OpDate: time.Now()},
		{SerialNumber: deletedSerial, Model: "MacBook Mini", OS: "osx", OpType: "modified", OpDate: time.Now().Add(time.Second)},
		{SerialNumber: deletedSerial, Model: "MacBook Mini", OS: "osx", OpType: "deleted", OpDate: time.Now().Add(2 * time.Second)},
	}
	profileAssignmentReqs = []profileAssignmentReq{}
	// Check that host display name is present for the device to be deleted; later we will check that it has been deleted
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var dest uint
		return sqlx.GetContext(ctx, q, &dest,
			"SELECT 1 FROM host_display_names WHERE host_id = ?", device1ID)
	})
	s.runDEPSchedule()
	// all hosts should be returned from the hosts endpoint
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	// all previous devices minus the pending deleted one
	wantSerials = slices.DeleteFunc(wantSerials, func(s string) bool { return s == deletedSerial })
	assert.Len(t, listHostsRes.Hosts, len(wantSerials))
	gotSerials = []string{}
	for _, device := range listHostsRes.Hosts {
		gotSerials = append(gotSerials, device.HardwareSerial)
	}
	assert.ElementsMatch(t, wantSerials, gotSerials)
	assert.Len(t, profileAssignmentReqs, 2)
	gotSerials = []string{}
	for _, req := range profileAssignmentReqs {
		assert.Len(t, req.Devices, 1)
		gotSerials = append(gotSerials, req.Devices...)
	}
	assert.ElementsMatch(t, []string{addedModifiedDeletedSerial, deletedAddedSerial}, gotSerials)
	// Check that host display name was deleted
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var dest uint
		return sqlx.GetContext(ctx, q, &dest,
			"SELECT 1 FROM host_display_names WHERE NOT EXISTS (SELECT 1 FROM host_display_names WHERE host_id = ?)", device1ID)
	})

	// delete all MDM info
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM host_mdm WHERE host_id = ?`, listHostsRes.Hosts[0].ID)
		return err
	})

	// it should still get the post-enrollment commands
	require.NoError(t, mdmDevice.Enroll())
	checkPostEnrollmentCommands(mdmDevice, true)

	// The user unenrolls from Fleet (e.g. was DEP enrolled but with `is_mdm_removable: true`
	// so the user removes the enrollment profile).
	err = mdmDevice.Checkout()
	require.NoError(t, err)

	// Simulate a refetch where we clean up the MDM data since the host is not enrolled anymore
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM host_mdm WHERE host_id = ?`, mdmDeviceID)
		return err
	})

	// Simulate fleetd re-enrolling automatically.
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	// The last activity should have `installed_from_dep=true`.
	s.lastActivityMatches(
		"mdm_enrolled",
		fmt.Sprintf(
			`{"host_serial": "%s", "host_display_name": "%s (%s)", "installed_from_dep": true, "mdm_platform": "apple"}`,
			mdmDevice.SerialNumber, mdmDevice.Model, mdmDevice.SerialNumber,
		),
		0,
	)

	// enroll a host into Fleet
	eHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		ID:             1,
		OsqueryHostID:  ptr.String("Desktop-ABCQWE"),
		NodeKey:        ptr.String("Desktop-ABCQWE"),
		UUID:           uuid.New().String(),
		Hostname:       fmt.Sprintf("%sfoo.local", s.T().Name()),
		Platform:       "darwin",
		HardwareSerial: uuid.New().String(),
	})
	require.NoError(t, err)

	// on team transfer, we don't assign a DEP profile to the device
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{eHost.ID}}, http.StatusOK)
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runWorker()
	require.Empty(t, profileAssignmentReqs)

	// assign the host in ABM
	devices = []godep.Device{
		{SerialNumber: eHost.HardwareSerial, Model: "MacBook Pro", OS: "osx", OpType: "modified"},
	}
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runDEPSchedule()
	require.NotEmpty(t, profileAssignmentReqs)
	require.Equal(t, eHost.HardwareSerial, profileAssignmentReqs[0].Devices[0])
	checkHostDEPAssignProfileResponses(profileAssignmentReqs[0].Devices, profileAssignmentReqs[0].ProfileUUID, fleet.DEPAssignProfileResponseSuccess)

	// report MDM info via osquery
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, eHost.ID, false, true, s.server.URL, true, fleet.WellKnownMDMFleet, ""))
	checkListHostDEPError(eHost.HardwareSerial, "On (automatic)", false)

	// transfer to "no team", we assign a DEP profile to the device
	profileAssignmentReqs = []profileAssignmentReq{}
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: nil, HostIDs: []uint{eHost.ID}}, http.StatusOK)
	s.runWorker()
	require.NotEmpty(t, profileAssignmentReqs)
	require.Equal(t, eHost.HardwareSerial, profileAssignmentReqs[0].Devices[0])
	checkHostDEPAssignProfileResponses(profileAssignmentReqs[0].Devices, profileAssignmentReqs[0].ProfileUUID, fleet.DEPAssignProfileResponseSuccess)
	checkListHostDEPError(eHost.HardwareSerial, "On (automatic)", false)

	// transfer to the team back again, we assign a DEP profile to the device again
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{eHost.ID}}, http.StatusOK)
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runWorker()
	require.NotEmpty(t, profileAssignmentReqs)
	require.Equal(t, eHost.HardwareSerial, profileAssignmentReqs[0].Devices[0])
	checkHostDEPAssignProfileResponses(profileAssignmentReqs[0].Devices, profileAssignmentReqs[0].ProfileUUID, fleet.DEPAssignProfileResponseSuccess)
	checkListHostDEPError(eHost.HardwareSerial, "On (automatic)", false)

	// transfer to "no team", but simulate a failed profile assignment
	expectAssignProfileResponseFailed = eHost.HardwareSerial
	profileAssignmentReqs = []profileAssignmentReq{}
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: nil, HostIDs: []uint{eHost.ID}}, http.StatusOK)
	checkPendingMacOSSetupAssistantJob("hosts_transferred", nil, []string{eHost.HardwareSerial}, 0)

	s.runIntegrationsSchedule()
	checkAssignProfileRequests(eHost.HardwareSerial, nil)
	profUUID := profileAssignmentReqs[0].ProfileUUID
	d := checkHostCooldown(eHost.HardwareSerial, profUUID, fleet.DEPAssignProfileResponseFailed, nil, expectNoJobID)
	require.NotZero(t, d.ResponseUpdatedAt)
	failedAt := d.ResponseUpdatedAt
	checkNoJobsPending()
	// list hosts shows dep profile error
	checkListHostDEPError(eHost.HardwareSerial, "On (automatic)", true)

	// run the integrations schedule during the cooldown period
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runIntegrationsSchedule()
	require.Empty(t, profileAssignmentReqs)                                                                           // no new request during cooldown
	checkHostCooldown(eHost.HardwareSerial, profUUID, fleet.DEPAssignProfileResponseFailed, &failedAt, expectNoJobID) // no change
	checkNoJobsPending()

	// create a new team
	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{
		Name:        t.Name() + "dummy",
		Description: "desc dummy",
	}, http.StatusOK, &tmResp)
	require.NotZero(t, createTeamResp.Team.ID)
	dummyTeam := tmResp.Team
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &dummyTeam.ID, HostIDs: []uint{eHost.ID}}, http.StatusOK)
	checkPendingMacOSSetupAssistantJob("hosts_transferred", &dummyTeam.ID, []string{eHost.HardwareSerial}, 0)

	s.runWorker()
	// expect no assign profile request during cooldown
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runIntegrationsSchedule()
	require.Empty(t, profileAssignmentReqs)                                                                           // screened for cooldown
	checkHostCooldown(eHost.HardwareSerial, profUUID, fleet.DEPAssignProfileResponseFailed, &failedAt, expectNoJobID) // no change
	checkNoJobsPending()

	// cooldown hosts are screened from update profile jobs that would assign profiles
	_, err = worker.QueueMacosSetupAssistantJob(ctx, s.ds, kitlog.NewNopLogger(), worker.MacosSetupAssistantUpdateProfile, &dummyTeam.ID, eHost.HardwareSerial)
	require.NoError(t, err)
	checkPendingMacOSSetupAssistantJob("update_profile", &dummyTeam.ID, []string{eHost.HardwareSerial}, 0)
	s.runIntegrationsSchedule()
	require.Empty(t, profileAssignmentReqs)                                                                           // screened for cooldown
	checkHostCooldown(eHost.HardwareSerial, profUUID, fleet.DEPAssignProfileResponseFailed, &failedAt, expectNoJobID) // no change
	checkNoJobsPending()

	// cooldown hosts are screened from delete profile jobs that would assign profiles
	_, err = worker.QueueMacosSetupAssistantJob(ctx, s.ds, kitlog.NewNopLogger(), worker.MacosSetupAssistantProfileDeleted, &dummyTeam.ID, eHost.HardwareSerial)
	require.NoError(t, err)
	checkPendingMacOSSetupAssistantJob("profile_deleted", &dummyTeam.ID, []string{eHost.HardwareSerial}, 0)
	s.runIntegrationsSchedule()
	require.Empty(t, profileAssignmentReqs)                                                                           // screened for cooldown
	checkHostCooldown(eHost.HardwareSerial, profUUID, fleet.DEPAssignProfileResponseFailed, &failedAt, expectNoJobID) // no change
	checkNoJobsPending()

	// // TODO: Restore this test when FIXME on DeleteTeam is addressed
	// s.Do("DELETE", fmt.Sprintf("/api/v1/fleet/teams/%d", dummyTeam.ID), nil, http.StatusOK)
	// checkPendingMacOSSetupAssistantJob("team_deleted", nil, []string{eHost.HardwareSerial}, 0)
	// s.runIntegrationsSchedule()
	// require.Empty(t, profileAssignmentReqs) // screened for cooldown
	// bySerial = checkHostDEPAssignProfileResponses([]string{eHost.HardwareSerial}, profUUID, fleet.DEPAssignProfileResponseFailed)
	// d, ok = bySerial[eHost.HardwareSerial]
	// require.True(t, ok)
	// require.Equal(t, failedAt, d.ResponseUpdatedAt)
	// require.Zero(t, d.RetryJobID) // cooling down so no retry job
	// checkNoJobsPending()

	// transfer back to no team, expect no assign profile request during cooldown
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: nil, HostIDs: []uint{eHost.ID}}, http.StatusOK)
	checkPendingMacOSSetupAssistantJob("hosts_transferred", nil, []string{eHost.HardwareSerial}, 0)
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runIntegrationsSchedule()
	require.Empty(t, profileAssignmentReqs)                                                                           // screened for cooldown
	checkHostCooldown(eHost.HardwareSerial, profUUID, fleet.DEPAssignProfileResponseFailed, &failedAt, expectNoJobID) // no change
	checkNoJobsPending()

	// simulate expired cooldown
	failedAt = failedAt.Add(-2 * time.Hour)
	setAssignProfileResponseUpdatedAt(eHost.HardwareSerial, failedAt)
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runIntegrationsSchedule()
	require.Empty(t, profileAssignmentReqs) // assign profile request will be made when the retry job is processed on the next worker run
	d = checkHostCooldown(eHost.HardwareSerial, profUUID, fleet.DEPAssignProfileResponseFailed, &failedAt, nil)
	require.NotZero(t, d.RetryJobID) // retry job created
	jobID := d.RetryJobID
	checkPendingMacOSSetupAssistantJob("hosts_cooldown", nil, []string{eHost.HardwareSerial}, jobID)

	// running the DEP schedule should not trigger a profile assignment request when the retry job is pending
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runDEPSchedule()
	require.Empty(t, profileAssignmentReqs)                                                                    // assign profile request will be made when the retry job is processed on the next worker run
	checkHostCooldown(eHost.HardwareSerial, profUUID, fleet.DEPAssignProfileResponseFailed, &failedAt, &jobID) // no change
	checkPendingMacOSSetupAssistantJob("hosts_cooldown", nil, []string{eHost.HardwareSerial}, jobID)
	checkListHostDEPError(eHost.HardwareSerial, "On (automatic)", true)

	// run the integration schedule and expect success
	expectAssignProfileResponseFailed = ""
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runIntegrationsSchedule()
	checkAssignProfileRequests(eHost.HardwareSerial, &profUUID)
	d = checkHostCooldown(eHost.HardwareSerial, profUUID, fleet.DEPAssignProfileResponseSuccess, nil, expectNoJobID) // retry job cleared
	require.True(t, d.ResponseUpdatedAt.After(failedAt))
	succeededAt := d.ResponseUpdatedAt
	checkNoJobsPending()
	checkListHostDEPError(eHost.HardwareSerial, "On (automatic)", false)

	// run the integrations schedule and expect no changes
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runIntegrationsSchedule()
	require.Empty(t, profileAssignmentReqs)
	checkHostCooldown(eHost.HardwareSerial, profUUID, fleet.DEPAssignProfileResponseSuccess, &succeededAt, expectNoJobID) // no change
	checkNoJobsPending()

	// ingest new device via DEP but the profile assignment fails
	serial := uuid.NewString()
	devices = []godep.Device{
		{SerialNumber: serial, Model: "MacBook Pro", OS: "osx", OpType: "added"},
	}
	expectAssignProfileResponseFailed = serial
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runDEPSchedule()
	checkAssignProfileRequests(serial, nil)
	profUUID = profileAssignmentReqs[0].ProfileUUID
	d = checkHostCooldown(serial, profUUID, fleet.DEPAssignProfileResponseFailed, nil, expectNoJobID)
	require.NotZero(t, d.ResponseUpdatedAt)
	failedAt = d.ResponseUpdatedAt
	checkNoJobsPending()
	h := checkListHostDEPError(serial, "Pending", true) // list hosts shows device pending and dep profile error

	// transfer to team, no profile assignment request is made during the cooldown period
	profileAssignmentReqs = []profileAssignmentReq{}
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{h.ID}}, http.StatusOK)
	checkPendingMacOSSetupAssistantJob("hosts_transferred", &team.ID, []string{serial}, 0)
	s.runIntegrationsSchedule()
	require.Empty(t, profileAssignmentReqs)                                                             // screened by cooldown
	checkHostCooldown(serial, profUUID, fleet.DEPAssignProfileResponseFailed, &failedAt, expectNoJobID) // no change
	checkNoJobsPending()

	// run the integrations schedule and expect no changes
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runIntegrationsSchedule()
	require.Empty(t, profileAssignmentReqs)
	checkHostCooldown(serial, profUUID, fleet.DEPAssignProfileResponseFailed, &failedAt, expectNoJobID) // no change
	checkNoJobsPending()

	// simulate expired cooldown
	failedAt = failedAt.Add(-2 * time.Hour)
	setAssignProfileResponseUpdatedAt(serial, failedAt)
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runIntegrationsSchedule()
	require.Empty(t, profileAssignmentReqs) // assign profile request will be made when the retry job is processed on the next worker run
	d = checkHostCooldown(serial, profUUID, fleet.DEPAssignProfileResponseFailed, &failedAt, nil)
	require.NotZero(t, d.RetryJobID) // retry job created
	jobID = d.RetryJobID
	checkPendingMacOSSetupAssistantJob("hosts_cooldown", &team.ID, []string{serial}, jobID)

	// run the inregration schedule and expect success
	expectAssignProfileResponseFailed = ""
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runIntegrationsSchedule()
	checkAssignProfileRequests(serial, nil)
	require.NotEqual(t, profUUID, profileAssignmentReqs[0].ProfileUUID) // retry job will use the current team profile instead
	profUUID = profileAssignmentReqs[0].ProfileUUID
	d = checkHostCooldown(serial, profUUID, fleet.DEPAssignProfileResponseSuccess, nil, expectNoJobID) // retry job cleared
	require.True(t, d.ResponseUpdatedAt.After(failedAt))
	checkNoJobsPending()
	// list hosts shows pending (because MDM detail query hasn't been reported) but dep profile
	// error has been cleared
	checkListHostDEPError(serial, "Pending", false)

	// ingest another device via DEP but the profile assignment is not accessible
	serial = uuid.NewString()
	devices = []godep.Device{
		{SerialNumber: serial, Model: "MacBook Pro", OS: "osx", OpType: "added"},
	}
	expectAssignProfileResponseNotAccessible = serial
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runDEPSchedule()
	require.Len(t, profileAssignmentReqs, 2) // FIXME: When new device is added in ABM, we see two profile assign requests when device is not accessible: first during the "fetch" phase, then during the "sync" phase
	expectProfileUUID := ""
	for _, req := range profileAssignmentReqs {
		require.Len(t, req.Devices, 1)
		require.Equal(t, serial, req.Devices[0])
		if expectProfileUUID == "" {
			expectProfileUUID = req.ProfileUUID
		} else {
			require.Equal(t, expectProfileUUID, req.ProfileUUID)
		}
		d := checkHostCooldown(serial, req.ProfileUUID, fleet.DEPAssignProfileResponseNotAccessible, nil, expectNoJobID) // not accessible responses aren't retried
		require.NotZero(t, d.ResponseUpdatedAt)
		failedAt = d.ResponseUpdatedAt
	}
	// list hosts shows device pending and no dep profile error for not accessible responses
	checkListHostDEPError(serial, "Pending", false)

	// no retry job for not accessible responses even if cooldown expires
	failedAt = failedAt.Add(-2 * time.Hour)
	setAssignProfileResponseUpdatedAt(serial, failedAt)
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runIntegrationsSchedule()
	require.Empty(t, profileAssignmentReqs)
	checkHostCooldown(serial, expectProfileUUID, fleet.DEPAssignProfileResponseNotAccessible, &failedAt, expectNoJobID) // no change
	checkNoJobsPending()

	// run with devices that already have valid and invalid profiles
	// assigned, we shouldn't re-assign the valid ones.
	devices = []godep.Device{
		{SerialNumber: uuid.NewString(), Model: "MacBook Pro", OS: "osx", OpType: "added", ProfileUUID: defaultProfileUUID},     // matches existing profile
		{SerialNumber: uuid.NewString(), Model: "MacBook Mini", OS: "osx", OpType: "modified", ProfileUUID: defaultProfileUUID}, // matches existing profile
		{SerialNumber: uuid.NewString(), Model: "MacBook Pro", OS: "osx", OpType: "added", ProfileUUID: "bar"},                  // doesn't match an existing profile
		{SerialNumber: uuid.NewString(), Model: "MacBook Mini", OS: "osx", OpType: "modified", ProfileUUID: "foo"},              // doesn't match an existing profile
		{SerialNumber: addedSerial, Model: "MacBook Pro", OS: "osx", OpType: "added", ProfileUUID: defaultProfileUUID},          // matches existing profile
		{SerialNumber: serial, Model: "MacBook Mini", OS: "osx", OpType: "modified", ProfileUUID: defaultProfileUUID},           // matches existing profile
	}
	expectAssignProfileResponseNotAccessible = ""
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runDEPSchedule()
	require.NotEmpty(t, profileAssignmentReqs)
	require.Len(t, profileAssignmentReqs[0].Devices, 2)
	require.ElementsMatch(t, []string{devices[2].SerialNumber, devices[3].SerialNumber}, profileAssignmentReqs[0].Devices)
	checkHostDEPAssignProfileResponses(profileAssignmentReqs[0].Devices, profileAssignmentReqs[0].ProfileUUID, fleet.DEPAssignProfileResponseSuccess)

	// run with only a device that already has the right profile, no errors and no assignments
	devices = []godep.Device{
		{SerialNumber: uuid.NewString(), Model: "MacBook Pro", OS: "osx", OpType: "added", ProfileUUID: defaultProfileUUID}, // matches existing profile
	}
	profileAssignmentReqs = []profileAssignmentReq{}
	s.runDEPSchedule()
	require.Empty(t, profileAssignmentReqs)
}

func (s *integrationMDMTestSuite) TestDEPProfileAssignmentWithMultipleABMs() {
	t := s.T()

	ctx := context.Background()
	type hostDEPRow struct {
		HostID                uint      `db:"host_id"`
		ProfileUUID           string    `db:"profile_uuid"`
		AssignProfileResponse string    `db:"assign_profile_response"`
		ResponseUpdatedAt     time.Time `db:"response_updated_at"`
		RetryJobID            uint      `db:"retry_job_id"`
	}
	checkHostDEPAssignProfileResponses := func(deviceSerials []string, expectedProfileUUID string,
		expectedStatus fleet.DEPAssignProfileResponseStatus,
	) map[string]hostDEPRow {
		bySerial := make(map[string]hostDEPRow, len(deviceSerials))
		for _, deviceSerial := range deviceSerials {
			mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				var dest hostDEPRow
				err := sqlx.GetContext(ctx, q, &dest,
					"SELECT host_id, assign_profile_response, profile_uuid, response_updated_at, retry_job_id FROM host_dep_assignments WHERE profile_uuid = ? AND host_id = (SELECT id FROM hosts WHERE hardware_serial = ?)",
					expectedProfileUUID, deviceSerial)
				require.NoError(t, err)
				require.Equal(t, string(expectedStatus), dest.AssignProfileResponse)
				bySerial[deviceSerial] = dest
				return nil
			})
		}
		return bySerial
	}

	devices := []godep.Device{
		{SerialNumber: uuid.New().String(), Model: "MacBook Pro M1", OS: "osx", OpType: "added", OpDate: time.Now()},
		{SerialNumber: uuid.New().String(), Model: "MacBook Mini M1", OS: "osx", OpType: "added", OpDate: time.Now()},
	}
	defaultOrgDevices := []godep.Device{
		{SerialNumber: uuid.New().String(), Model: "MacBook Mini M2", OS: "osx", OpType: "added", OpDate: time.Now()},
		{SerialNumber: uuid.New().String(), Model: "MacBook Pro M2", OS: "osx", OpType: "added", OpDate: time.Now()},
	}

	// set release device manually to true so there is no job enqueued at a later
	// time to release the device (this is not what this test is about)
	s.Do("PATCH", "/api/latest/fleet/setup_experience", json.RawMessage(jsonMustMarshal(t, map[string]any{
		"enable_release_device_manually": true,
	})), http.StatusNoContent)

	// set up multiple ABM tokens with different org names
	defaultOrgName := "default_" + t.Name()
	s.enableABM(defaultOrgName)
	tmOrgName := t.Name()
	s.enableABM(tmOrgName)

	// create a new team
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        tmOrgName,
		Description: "desc",
	})
	require.NoError(t, err)
	// set the default bm assignment for that token to that team
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_business_manager": [{
			  "organization_name": %q,
			  "macos_team": %q,
			  "ios_team": %q,
			  "ipados_team": %q
			}]
		}
	}`, tmOrgName, tm.Name, tm.Name, tm.Name)), http.StatusOK, &acResp)
	t.Cleanup(func() {
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"apple_business_manager": []
			}
		}`), http.StatusOK, &acResp)
	})
	tmProf := `{"profile_name": "Team Profile"}`
	var createResp createMDMAppleSetupAssistantResponse
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            &tm.ID,
		Name:              tmOrgName,
		EnrollmentProfile: json.RawMessage(tmProf),
	}, http.StatusOK, &createResp)
	assert.Equal(t, tm.ID, *createResp.TeamID)

	var teamProfileUUIDs []string
	var defaultProfileUUIDs []string
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			require.NoError(t, err)
		case "/profile":
			profileUUID := uuid.NewString()
			teamProfileUUIDs = append(teamProfileUUIDs, profileUUID)
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: profileUUID})
			require.NoError(t, err)
		case "/server/devices":
			// This endpoint  is used to get an initial list of
			// devices, return a single device
			err := encoder.Encode(godep.DeviceResponse{Devices: devices[:1]})
			require.NoError(t, err)
		case "/devices/sync":
			// This endpoint is polled over time to sync devices from
			// ABM, send a repeated serial and a new one
			err := encoder.Encode(godep.DeviceResponse{Devices: devices, Cursor: "foo"})
			require.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))
			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			assert.Contains(t, teamProfileUUIDs, resp.ProfileUUID)
			resp.Devices = make(map[string]string, len(prof.Devices))
			for _, device := range prof.Devices {
				resp.Devices[device] = string(fleet.DEPAssignProfileResponseSuccess)
			}
			err = encoder.Encode(resp)
			require.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	s.mockDEPResponse(defaultOrgName, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			require.NoError(t, err)
		case "/profile":
			profileUUID := uuid.NewString()
			defaultProfileUUIDs = append(defaultProfileUUIDs, profileUUID)
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: profileUUID})
			require.NoError(t, err)
		case "/server/devices":
			// This endpoint  is used to get an initial list of
			// devices, return a single device
			err := encoder.Encode(godep.DeviceResponse{Devices: defaultOrgDevices[:1]})
			require.NoError(t, err)
		case "/devices/sync":
			// This endpoint is polled over time to sync devices from
			// ABM, send a repeated serial and a new one
			err := encoder.Encode(godep.DeviceResponse{Devices: defaultOrgDevices, Cursor: "foo"})
			require.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))
			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			assert.Contains(t, defaultProfileUUIDs, resp.ProfileUUID)
			resp.Devices = make(map[string]string, len(prof.Devices))
			for _, device := range prof.Devices {
				resp.Devices[device] = string(fleet.DEPAssignProfileResponseSuccess)
			}
			err = encoder.Encode(resp)
			require.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	// query all hosts
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Empty(t, listHostsRes.Hosts)

	// trigger a profile sync
	s.runDEPSchedule()

	// all hosts should be returned from the hosts endpoint
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	numHosts := len(devices) + len(defaultOrgDevices)
	assert.Len(t, listHostsRes.Hosts, numHosts)
	defaultSerials := []string{defaultOrgDevices[0].SerialNumber, defaultOrgDevices[1].SerialNumber}
	teamSerials := []string{devices[0].SerialNumber, devices[1].SerialNumber}
	for _, host := range listHostsRes.Hosts {
		switch {
		case slices.Contains(defaultSerials, host.HardwareSerial):
			assert.Nil(t, host.TeamID)
		case slices.Contains(teamSerials, host.HardwareSerial):
			assert.NotNil(t, host.TeamID)
		default:
			t.Errorf("unexpected host serial %s", host.HardwareSerial)
		}
	}
	require.GreaterOrEqual(t, len(defaultProfileUUIDs), 1)
	require.Len(t, teamProfileUUIDs, 2)
	checkHostDEPAssignProfileResponses(defaultSerials, defaultProfileUUIDs[len(defaultProfileUUIDs)-1],
		fleet.DEPAssignProfileResponseSuccess)
	checkHostDEPAssignProfileResponses(teamSerials, teamProfileUUIDs[len(teamProfileUUIDs)-1], fleet.DEPAssignProfileResponseSuccess)

	// Delete the devices in one org, and add them to the other (x2)
	devices = []godep.Device{
		{SerialNumber: devices[0].SerialNumber, Model: "MacBook Pro M1", OS: "osx", OpType: "added", OpDate: time.Now()},
		{SerialNumber: devices[1].SerialNumber, Model: "MacBook Mini M1", OS: "osx", OpType: "deleted", OpDate: time.Now()},
		{
			SerialNumber: defaultOrgDevices[1].SerialNumber, Model: "MacBook Pro M2", OS: "osx", OpType: "added",
			OpDate: time.Now().Add(time.Microsecond),
		},
	}
	defaultOrgDevices = []godep.Device{
		{SerialNumber: defaultOrgDevices[0].SerialNumber, Model: "MacBook Mini M2", OS: "osx", OpType: "added", OpDate: time.Now()},
		{SerialNumber: defaultOrgDevices[1].SerialNumber, Model: "MacBook Pro M2", OS: "osx", OpType: "deleted", OpDate: time.Now()},
		{
			SerialNumber: devices[1].SerialNumber, Model: "MacBook Mini M1", OS: "osx", OpType: "added",
			OpDate: time.Now().Add(time.Microsecond),
		},
	}

	// trigger a profile sync
	s.runDEPSchedule()

	// all hosts should be returned from the hosts endpoint; the 2 deleted and re-added hosts should switch teams and profiles
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	assert.Len(t, listHostsRes.Hosts, numHosts)
	defaultSerials = []string{defaultOrgDevices[0].SerialNumber, defaultOrgDevices[2].SerialNumber}
	teamSerials = []string{devices[0].SerialNumber, devices[2].SerialNumber}
	for _, host := range listHostsRes.Hosts {
		switch {
		case slices.Contains(defaultSerials, host.HardwareSerial):
			assert.Nil(t, host.TeamID)
		case slices.Contains(teamSerials, host.HardwareSerial):
			assert.NotNil(t, host.TeamID)
		default:
			t.Errorf("unexpected host serial %s", host.HardwareSerial)
		}
	}
	checkHostDEPAssignProfileResponses(defaultSerials, defaultProfileUUIDs[len(defaultProfileUUIDs)-1],
		fleet.DEPAssignProfileResponseSuccess)
	checkHostDEPAssignProfileResponses(teamSerials, teamProfileUUIDs[len(teamProfileUUIDs)-1], fleet.DEPAssignProfileResponseSuccess)

	// Delete the devices
	devices = []godep.Device{
		{SerialNumber: devices[0].SerialNumber, Model: "MacBook Pro M1", OS: "osx", OpType: "modified", OpDate: time.Now()},
		{
			SerialNumber: devices[2].SerialNumber, Model: "MacBook Pro M2", OS: "osx", OpType: "deleted",
			OpDate: time.Now().Add(time.Microsecond),
		},
	}
	defaultOrgDevices = []godep.Device{
		{SerialNumber: defaultOrgDevices[0].SerialNumber, Model: "MacBook Mini M2", OS: "osx", OpType: "modified", OpDate: time.Now()},
		{
			SerialNumber: defaultOrgDevices[2].SerialNumber, Model: "MacBook Mini M1", OS: "osx", OpType: "deleted",
			OpDate: time.Now().Add(time.Microsecond),
		},
	}

	// trigger a profile sync
	s.runDEPSchedule()

	// 2 hosts should be gone
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	assert.Len(t, listHostsRes.Hosts, numHosts-2)
	defaultSerials = []string{defaultOrgDevices[0].SerialNumber}
	teamSerials = []string{devices[0].SerialNumber}
	for _, host := range listHostsRes.Hosts {
		switch {
		case slices.Contains(defaultSerials, host.HardwareSerial):
			assert.Nil(t, host.TeamID)
		case slices.Contains(teamSerials, host.HardwareSerial):
			assert.NotNil(t, host.TeamID)
		default:
			t.Errorf("unexpected host serial %s", host.HardwareSerial)
		}
	}
	checkHostDEPAssignProfileResponses(defaultSerials, defaultProfileUUIDs[len(defaultProfileUUIDs)-1],
		fleet.DEPAssignProfileResponseSuccess)
	checkHostDEPAssignProfileResponses(teamSerials, teamProfileUUIDs[len(teamProfileUUIDs)-1], fleet.DEPAssignProfileResponseSuccess)
}

func (s *integrationMDMTestSuite) TestDeprecatedDefaultAppleBMTeam() {
	t := s.T()

	s.enableABM(t.Name())

	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(s.T(), err)

	var acResp appConfigResponse

	defer func() {
		acResp = appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		    "mdm": {
			    "apple_bm_default_team": ""
		    }
		}`), http.StatusOK, &acResp)
		require.Empty(t, acResp.MDM.DeprecatedAppleBMDefaultTeam)
	}()

	// try to set an invalid team name
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"apple_bm_default_team": "xyz"
		}
	}`), http.StatusUnprocessableEntity, &acResp)

	// get the appconfig, nothing changed
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Empty(t, acResp.MDM.DeprecatedAppleBMDefaultTeam)

	// set to a valid team name
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_bm_default_team": %q
		}
	}`, tm.Name)), http.StatusOK, &acResp)
	require.Equal(t, tm.Name, acResp.MDM.DeprecatedAppleBMDefaultTeam)

	// get the appconfig, set to that team name
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Equal(t, tm.Name, acResp.MDM.DeprecatedAppleBMDefaultTeam)
}

func (s *integrationMDMTestSuite) TestSetupExperienceScript() {
	t := s.T()

	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	// create new team script
	var newScriptResp setSetupExperienceScriptResponse
	body, headers := generateNewScriptMultipartRequest(t,
		"script42.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)

	// test script secret validation
	body, headers = generateNewScriptMultipartRequest(t,
		"script.sh", []byte(`echo "$FLEET_SECRET_INVALID"`), s.token, map[string][]string{})
	s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusUnprocessableEntity, headers)

	// get team script metadata
	var getScriptResp getSetupExperienceScriptResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d", tm.ID), nil, http.StatusOK, &getScriptResp)
	require.Equal(t, "script42.sh", getScriptResp.Name)
	require.NotNil(t, getScriptResp.TeamID)
	require.Equal(t, tm.ID, *getScriptResp.TeamID)
	require.NotZero(t, getScriptResp.ID)
	require.NotZero(t, getScriptResp.CreatedAt)
	require.NotZero(t, getScriptResp.UpdatedAt)

	// get team script contents
	res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d&alt=media", tm.ID), nil, http.StatusOK)
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, `echo "hello"`, string(b))
	require.Equal(t, int64(len(`echo "hello"`)), res.ContentLength)
	require.Equal(t, fmt.Sprintf("attachment;filename=\"%s %s\"", time.Now().Format(time.DateOnly), "script42.sh"), res.Header.Get("Content-Disposition"))

	// try to create script with same name, should fail because already exists with this name for this team
	body, headers = generateNewScriptMultipartRequest(t,
		"script42.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusConflict, headers)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "already exists") // TODO: confirm expected error message with product/frontend

	// try to create with a different name for this team, should fail because another script already exists
	// for this team
	body, headers = generateNewScriptMultipartRequest(t,
		"different.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusConflict, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "already exists") // TODO: confirm expected error message with product/frontend

	// create no-team script
	body, headers = generateNewScriptMultipartRequest(t,
		"script42.sh", []byte(`echo "hello"`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)
	// // TODO: confirm if we will allow team_id=0 requests
	// noTeamID := uint(0) // TODO: confirm if we will allow team_id=0 requests
	// body, headers = generateNewScriptMultipartRequest(t,
	// 	"script42.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", noTeamID)}})

	// get no-team script metadata
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/script", nil, http.StatusOK, &getScriptResp)
	require.Equal(t, "script42.sh", getScriptResp.Name)
	require.Nil(t, getScriptResp.TeamID)
	require.NotZero(t, getScriptResp.ID)
	require.NotZero(t, getScriptResp.CreatedAt)
	require.NotZero(t, getScriptResp.UpdatedAt)
	// // TODO: confirm if we will allow team_id=0 requests
	// s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d", noTeamID), nil, http.StatusOK, &getScriptResp)

	// get no-team script contents
	res = s.Do("GET", "/api/latest/fleet/setup_experience/script?alt=media", nil, http.StatusOK)
	b, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, `echo "hello"`, string(b))
	require.Equal(t, int64(len(`echo "hello"`)), res.ContentLength)
	require.Equal(t, fmt.Sprintf("attachment;filename=\"%s %s\"", time.Now().Format(time.DateOnly), "script42.sh"), res.Header.Get("Content-Disposition"))
	// // TODO: confirm if we will allow team_id=0 requests
	// res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d&alt=media", noTeamID), nil, http.StatusOK)

	// delete the no-team script
	s.Do("DELETE", "/api/latest/fleet/setup_experience/script", nil, http.StatusOK)

	// try get the no-team script
	s.Do("GET", "/api/latest/fleet/setup_experience/script", nil, http.StatusNotFound)

	// try deleting the no-team script again
	s.Do("DELETE", "/api/latest/fleet/setup_experience/script", nil, http.StatusOK) // TODO: confirm if we want to return not found

	// // TODO: confirm if we will allow team_id=0 requests
	// s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/script/?team_id=%d", noTeamID), nil, http.StatusOK)

	// delete the team script
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d", tm.ID), nil, http.StatusOK)

	// try get the team script
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d", tm.ID), nil, http.StatusNotFound)

	// try deleting the team script again
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d", tm.ID), nil, http.StatusOK) // TODO: confirm if we want to return not found
}

func (s *integrationMDMTestSuite) createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript() (device godep.Device, host *fleet.Host, tm *fleet.Team) {
	t := s.T()
	ctx := context.Background()

	// enroll a device in a team with software to install and a script to execute
	s.enableABM("fleet-setup-experience")
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	teamDevice := godep.Device{SerialNumber: uuid.New().String(), Model: "MacBook Pro", OS: "osx", OpType: "added"}

	// add a team profile
	teamProfile := mobileconfigForTest("N1", "I1")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{teamProfile}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

	// add a macOS software to install
	payloadDummy := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "dummy_installer.pkg",
		Title:         "DummyApp.app",
		TeamID:        &tm.ID,
	}
	s.uploadSoftwareInstaller(t, payloadDummy, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, payloadDummy.Title, "apps")
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{TeamID: tm.ID, TitleIDs: []uint{titleID}}, http.StatusOK, &swInstallResp)

	// add a script to execute
	body, headers := generateNewScriptMultipartRequest(t,
		"script.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)

	// no bootstrap package, no custom setup assistant (those are already tested
	// in the DEPEnrollReleaseDevice tests).

	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		return map[string]*push.Response{}, nil
	}

	s.mockDEPResponse("fleet-setup-experience", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			require.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()})
			require.NoError(t, err)
		case "/server/devices":
			err := encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{teamDevice}})
			require.NoError(t, err)
		case "/devices/sync":
			// This endpoint is polled over time to sync devices from
			// ABM, send a repeated serial
			err := encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{teamDevice}, Cursor: "foo"})
			require.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))

			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = make(map[string]string, len(prof.Devices))
			for _, device := range prof.Devices {
				resp.Devices[device] = string(fleet.DEPAssignProfileResponseSuccess)
			}
			err = encoder.Encode(resp)
			require.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	// trigger a profile sync
	s.runDEPSchedule()

	// the (ghost) host now exists
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	require.Equal(t, listHostsRes.Hosts[0].HardwareSerial, teamDevice.SerialNumber)
	enrolledHost := listHostsRes.Hosts[0].Host
	enrolledHost.TeamID = &tm.ID

	// transfer it to the team
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &tm.ID, HostIDs: []uint{enrolledHost.ID}}, http.StatusOK)

	return teamDevice, enrolledHost, tm
}

func (s *integrationMDMTestSuite) TestSetupExperienceFlowWithSoftwareAndScriptAutoRelease() {
	t := s.T()
	ctx := context.Background()

	teamDevice, enrolledHost, _ := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = teamDevice.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// run the worker to process the DEP enroll request
	s.runWorker()
	// run the worker to assign configuration profiles
	s.awaitTriggerProfileSchedule(t)

	var cmds []*micromdm.CommandPayload
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {

		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

		// Can be useful for debugging
		// switch cmd.Command.RequestType {
		// case "InstallProfile":
		// 	fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType, string(fullCmd.Command.InstallProfile.Payload))
		// case "InstallEnterpriseApplication":
		// 	if fullCmd.Command.InstallEnterpriseApplication.ManifestURL != nil {
		// 		fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType, *fullCmd.Command.InstallEnterpriseApplication.ManifestURL)
		// 	} else {
		// 		fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType)
		// 	}
		// default:
		// 	fmt.Println(">>>> device received command: ", cmd.Command.RequestType)
		// }

		cmds = append(cmds, &fullCmd)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// expected commands: install fleetd (install enterprise), install profiles
	// (custom one, fleetd configuration, fleet CA root)
	require.Len(t, cmds, 4)
	var installProfileCount, installEnterpriseCount, otherCount int
	var profileCustomSeen, profileFleetdSeen, profileFleetCASeen, profileFileVaultSeen bool
	for _, cmd := range cmds {
		switch cmd.Command.RequestType {
		case "InstallProfile":
			installProfileCount++
			switch {
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), "<string>I1</string>"):
				profileCustomSeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetdConfigPayloadIdentifier)):
				profileFleetdSeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetCARootConfigPayloadIdentifier)):
				profileFleetCASeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string", mobileconfig.FleetFileVaultPayloadIdentifier)) &&
				strings.Contains(string(cmd.Command.InstallProfile.Payload), "ForceEnableInSetupAssistant"):
				profileFileVaultSeen = true
			}

		case "InstallEnterpriseApplication":
			installEnterpriseCount++
		default:
			otherCount++
		}
	}
	require.Equal(t, 3, installProfileCount)
	require.Equal(t, 1, installEnterpriseCount)
	require.Equal(t, 0, otherCount)
	require.True(t, profileCustomSeen)
	require.True(t, profileFleetdSeen)
	require.True(t, profileFleetCASeen)
	require.False(t, profileFileVaultSeen)

	// simulate fleetd being installed and the host being orbit-enrolled now
	enrolledHost.OsqueryHostID = ptr.String(mdmDevice.UUID)
	enrolledHost.UUID = mdmDevice.UUID
	orbitKey := setOrbitEnrollment(t, enrolledHost, s.ds)
	enrolledHost.OrbitNodeKey = &orbitKey

	// there shouldn't be a worker Release Device pending job (we don't release that way anymore)
	pending, err := s.ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute))
	require.NoError(t, err)
	require.Len(t, pending, 0)

	// call the /status endpoint, the software and script should be pending
	var statusResp getOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile
	var profNames []string
	var profStatuses []fleet.MDMDeliveryStatus
	for _, prof := range statusResp.Results.ConfigurationProfiles {
		profNames = append(profNames, prof.Name)
		profStatuses = append(profStatuses, prof.Status)
	}
	require.ElementsMatch(t, []string{"N1", "Fleetd configuration", "Fleet root certificate authority (CA)"}, profNames)
	require.ElementsMatch(t, []fleet.MDMDeliveryStatus{fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerifying}, profStatuses)

	// the software and script are still pending
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "DummyApp.app", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

	// The /setup_experience/status endpoint doesn't return the various IDs for executions, so pull
	// it out manually
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID)
	require.Len(t, results, 2)
	require.NoError(t, err)
	var installUUID string
	for _, r := range results {
		if r.HostSoftwareInstallsExecutionID != nil {
			installUUID = *r.HostSoftwareInstallsExecutionID
		}
	}

	require.NotEmpty(t, installUUID)

	// Need to get the software title to get the package name
	var getSoftwareTitleResp getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", *statusResp.Results.Software[0].SoftwareTitleID), nil, http.StatusOK, &getSoftwareTitleResp, "team_id", fmt.Sprintf("%d", *enrolledHost.TeamID))
	require.NotNil(t, getSoftwareTitleResp.SoftwareTitle)
	require.NotNil(t, getSoftwareTitleResp.SoftwareTitle.SoftwarePackage)

	debugPrintActivities := func(activities []*fleet.Activity) []string {
		var res []string
		for _, activity := range activities {
			res = append(res, fmt.Sprintf("%+v", activity))
		}
		return res
	}

	// Check upcoming activities: we should only have the software upcoming because we don't run the
	// script until after the software is done
	var hostActivitiesResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", enrolledHost.ID),
		nil, http.StatusOK, &hostActivitiesResp)

	expectedActivityDetail := fmt.Sprintf(`
	{
		"status": "pending_install",
		"host_id": %d,
		"policy_id": null,
		"policy_name": null,
		"install_uuid": "%s",
		"self_service": false,
		"software_title": "%s",
		"software_package": "%s",
		"host_display_name": "%s"
	}
	`, enrolledHost.ID, installUUID, getSoftwareTitleResp.SoftwareTitle.Name, getSoftwareTitleResp.SoftwareTitle.SoftwarePackage.Name, enrolledHost.DisplayName())
	require.Len(t, hostActivitiesResp.Activities, 1, "got activities: %v", debugPrintActivities(hostActivitiesResp.Activities))
	require.NotNil(t, hostActivitiesResp.Activities[0].Details)
	require.JSONEq(t, expectedActivityDetail, string(*hostActivitiesResp.Activities[0].Details))

	// no MDM command got enqueued due to the /status call (device not released yet)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	// Software is now running, script is still pending
	require.Equal(t, "DummyApp.app", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)

	// record a result for software installation
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`, *enrolledHost.OrbitNodeKey, installUUID)), http.StatusNoContent)

	// status still shows script as pending
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "DummyApp.app", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)

	// no MDM command got enqueued due to the /status call (device not released yet)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// Software is installed, now we should run the script
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Equal(t, "DummyApp.app", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Script.Status)

	// Get script exec ID
	results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID)
	require.Len(t, results, 2)
	require.NoError(t, err)
	var execID string
	for _, r := range results {
		if r.ScriptExecutionID != nil {
			execID = *r.ScriptExecutionID
		}
	}

	// Validate past activity for software install
	// For some reason the display name that's included in the `enrolledHost` is _slightly_
	// different than the expected value in the activities. Pulling the host directly gets the
	// correct display name.
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", enrolledHost.ID), nil, http.StatusOK, &getHostResp)

	expectedActivityDetail = fmt.Sprintf(`
{
  "host_id": %d,
  "host_display_name": "%s",
  "software_title": "%s",
  "software_package": "%s",
  "self_service": false,
  "install_uuid": "%s",
  "status": "installed",
  "policy_id": null,
  "policy_name": null
}
	`, enrolledHost.ID, getHostResp.Host.DisplayName, statusResp.Results.Software[0].Name, getSoftwareTitleResp.SoftwareTitle.SoftwarePackage.Name, installUUID)

	s.lastActivityMatches(fleet.ActivityTypeInstalledSoftware{}.ActivityName(), expectedActivityDetail, 0)

	// Validate upcoming activity for the script
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", enrolledHost.ID),
		nil, http.StatusOK, &hostActivitiesResp)

	expectedActivityDetail = fmt.Sprintf(`
{
	"async": true,
	"host_id": %d,
	"policy_id": null,
	"policy_name": null,
	"script_name": "%s",
	"host_display_name": "%s",
	"script_execution_id": "%s"
}
	`, enrolledHost.ID, statusResp.Results.Script.Name, enrolledHost.DisplayName(), execID)
	require.Len(t, hostActivitiesResp.Activities, 1, "got activities: %v", debugPrintActivities(hostActivitiesResp.Activities))
	require.NotNil(t, hostActivitiesResp.Activities[0].Details)
	require.JSONEq(t, expectedActivityDetail, string(*hostActivitiesResp.Activities[0].Details))

	// record a result for script execution
	var scriptResp orbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *enrolledHost.OrbitNodeKey, execID)),
		http.StatusOK, &scriptResp)

	// Get status again, now the script should be complete. This should also trigger the automatic
	// release of the device, as all setup experience steps are now complete.
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Equal(t, "DummyApp.app", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Script.Status)

	// check that the host received the device configured command automatically
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	cmds = cmds[:0]
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
		cmds = append(cmds, &fullCmd)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	require.Len(t, cmds, 1)
	var deviceConfiguredCount int
	for _, cmd := range cmds {
		switch cmd.Command.RequestType {
		case "DeviceConfigured":
			deviceConfiguredCount++
		default:
			otherCount++
		}
	}
	require.Equal(t, 1, deviceConfiguredCount)
	require.Equal(t, 0, otherCount)

	// Validate activity for script run
	expectedActivityDetail = fmt.Sprintf(`
{
	"async": true,
	"host_id": %d,
	"policy_id": null,
	"policy_name": null,
	"script_name": "%s",
	"host_display_name": "%s",
	"script_execution_id": "%s"
}
	`, enrolledHost.ID, statusResp.Results.Script.Name, getHostResp.Host.DisplayName, execID)

	s.lastActivityMatches(fleet.ActivityTypeRanScript{}.ActivityName(), expectedActivityDetail, 0)
}

func (s *integrationMDMTestSuite) TestSetupExperienceFlowWithSoftwareAndScriptForceRelease() {
	t := s.T()
	ctx := context.Background()

	teamDevice, enrolledHost, _ := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = teamDevice.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// run the worker to process the DEP enroll request
	s.runWorker()
	// run the worker to assign configuration profiles
	s.awaitTriggerProfileSchedule(t)

	var cmds []*micromdm.CommandPayload
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {

		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

		// Can be useful for debugging
		// switch cmd.Command.RequestType {
		// case "InstallProfile":
		// 	fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType, string(fullCmd.Command.InstallProfile.Payload))
		// case "InstallEnterpriseApplication":
		// 	if fullCmd.Command.InstallEnterpriseApplication.ManifestURL != nil {
		// 		fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType, *fullCmd.Command.InstallEnterpriseApplication.ManifestURL)
		// 	} else {
		// 		fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType)
		// 	}
		// default:
		// 	fmt.Println(">>>> device received command: ", cmd.Command.RequestType)
		// }

		cmds = append(cmds, &fullCmd)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// expected commands: install fleetd (install enterprise), install profiles
	// (custom one, fleetd configuration, fleet CA root)
	require.Len(t, cmds, 4)
	var installProfileCount, installEnterpriseCount, otherCount int
	var profileCustomSeen, profileFleetdSeen, profileFleetCASeen, profileFileVaultSeen bool
	for _, cmd := range cmds {
		switch cmd.Command.RequestType {
		case "InstallProfile":
			installProfileCount++
			switch {
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), "<string>I1</string>"):
				profileCustomSeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetdConfigPayloadIdentifier)):
				profileFleetdSeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetCARootConfigPayloadIdentifier)):
				profileFleetCASeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string", mobileconfig.FleetFileVaultPayloadIdentifier)) &&
				strings.Contains(string(cmd.Command.InstallProfile.Payload), "ForceEnableInSetupAssistant"):
				profileFileVaultSeen = true
			}

		case "InstallEnterpriseApplication":
			installEnterpriseCount++
		default:
			otherCount++
		}
	}
	require.Equal(t, 3, installProfileCount)
	require.Equal(t, 1, installEnterpriseCount)
	require.Equal(t, 0, otherCount)
	require.True(t, profileCustomSeen)
	require.True(t, profileFleetdSeen)
	require.True(t, profileFleetCASeen)
	require.False(t, profileFileVaultSeen)

	// simulate fleetd being installed and the host being orbit-enrolled now
	enrolledHost.OsqueryHostID = ptr.String(mdmDevice.UUID)
	orbitKey := setOrbitEnrollment(t, enrolledHost, s.ds)
	enrolledHost.OrbitNodeKey = &orbitKey

	// there shouldn't be a worker Release Device pending job (we don't release that way anymore)
	pending, err := s.ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute))
	require.NoError(t, err)
	require.Len(t, pending, 0)

	// call the /status endpoint, the software and script should be pending
	var statusResp getOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile
	var profNames []string
	var profStatuses []fleet.MDMDeliveryStatus
	for _, prof := range statusResp.Results.ConfigurationProfiles {
		profNames = append(profNames, prof.Name)
		profStatuses = append(profStatuses, prof.Status)
	}
	require.ElementsMatch(t, []string{"N1", "Fleetd configuration", "Fleet root certificate authority (CA)"}, profNames)
	require.ElementsMatch(t, []fleet.MDMDeliveryStatus{fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerifying}, profStatuses)

	// the software and script are still pending
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "DummyApp.app", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

	// no MDM command got enqueued due to the /status call (device not released yet)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// call the /status endpoint again but this time force the release
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "force_release": true}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)

	// the software and script have not completed yet
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Software[0].Status)

	// check that the host received the device configured command even if
	// software and script are still pending
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	cmds = cmds[:0]
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
		cmds = append(cmds, &fullCmd)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	require.Len(t, cmds, 1)
	var deviceConfiguredCount int
	for _, cmd := range cmds {
		switch cmd.Command.RequestType {
		case "DeviceConfigured":
			deviceConfiguredCount++
		default:
			otherCount++
		}
	}
	require.Equal(t, 1, deviceConfiguredCount)
	require.Equal(t, 0, otherCount)
}

func (s *integrationMDMTestSuite) TestReenrollingADEDeviceAfterRemovingItFromABM() {
	t := s.T()
	s.enableABM(t.Name())
	ctx := context.Background()

	checkPostEnrollmentCommands := func(mdmDevice *mdmtest.TestAppleMDMClient, shouldReceive bool) {
		// run the worker to process the DEP enroll request
		s.runWorker()
		// run the worker to assign configuration profiles
		s.awaitTriggerProfileSchedule(t)

		var fleetdCmd, installProfileCmd *micromdm.CommandPayload
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			if fullCmd.Command.RequestType == "InstallEnterpriseApplication" &&
				fullCmd.Command.InstallEnterpriseApplication.ManifestURL != nil &&
				strings.Contains(*fullCmd.Command.InstallEnterpriseApplication.ManifestURL, fleetdbase.GetPKGManifestURL()) {
				fleetdCmd = &fullCmd
			} else if cmd.Command.RequestType == "InstallProfile" {
				installProfileCmd = &fullCmd
			}
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}

		if shouldReceive {
			// received request to install fleetd
			require.NotNil(t, fleetdCmd, "host didn't get a command to install fleetd")
			require.NotNil(t, fleetdCmd.Command, "host didn't get a command to install fleetd")

			// received request to install the global configuration profile
			require.NotNil(t, installProfileCmd, "host didn't get a command to install profiles")
			require.NotNil(t, installProfileCmd.Command, "host didn't get a command to install profiles")
		} else {
			require.Nil(t, fleetdCmd, "host got a command to install fleetd")
			require.Nil(t, installProfileCmd, "host got a command to install profiles")
		}
	}

	devices := []godep.Device{
		{SerialNumber: uuid.New().String(), Model: "MacBook Pro", OS: "osx", OpType: "added"},
	}

	profileAssignmentReqs := []profileAssignmentReq{}

	type hostDEPRow struct {
		HostID                uint       `db:"host_id"`
		ProfileUUID           string     `db:"profile_uuid"`
		AssignProfileResponse string     `db:"assign_profile_response"`
		ResponseUpdatedAt     time.Time  `db:"response_updated_at"`
		RetryJobID            uint       `db:"retry_job_id"`
		DeletedAt             *time.Time `db:"deleted_at"`
	}
	checkHostDEPAssignProfileResponses := func(deviceSerials []string, expectedProfileUUID string, expectedStatus fleet.DEPAssignProfileResponseStatus) map[string]hostDEPRow {
		bySerial := make(map[string]hostDEPRow, len(deviceSerials))
		for _, deviceSerial := range deviceSerials {
			mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				var dest hostDEPRow
				err := sqlx.GetContext(ctx, q, &dest, "SELECT host_id, assign_profile_response, profile_uuid, response_updated_at, retry_job_id, deleted_at FROM host_dep_assignments WHERE profile_uuid = ? AND host_id = (SELECT id FROM hosts WHERE hardware_serial = ?)", expectedProfileUUID, deviceSerial)
				require.NoError(t, err)
				require.Equal(t, string(expectedStatus), dest.AssignProfileResponse)
				bySerial[deviceSerial] = dest
				return nil
			})
		}
		return bySerial
	}

	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			require.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()})
			require.NoError(t, err)
		case "/server/devices":
			// This endpoint  is used to get an initial list of
			// devices, return a single device
			err := encoder.Encode(godep.DeviceResponse{Devices: devices[:1]})
			require.NoError(t, err)
		case "/devices/sync":
			// This endpoint is polled over time to sync devices from
			// ABM, send a repeated serial and a new one
			err := encoder.Encode(godep.DeviceResponse{Devices: devices, Cursor: "foo"})
			require.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))
			profileAssignmentReqs = append(profileAssignmentReqs, prof)
			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = make(map[string]string, len(prof.Devices))
			for _, device := range prof.Devices {
				resp.Devices[device] = string(fleet.DEPAssignProfileResponseSuccess)
			}
			err = encoder.Encode(resp)
			require.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		return map[string]*push.Response{}, nil
	}

	// Enroll the host via ADE
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = devices[0].SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// Simulate an osquery enrollment too
	// set an enroll secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// simulate a matching host enrolling via osquery
	j, err := json.Marshal(&enrollAgentRequest{
		EnrollSecret:   t.Name(),
		HostIdentifier: mdmDevice.UUID,
	})
	require.NoError(t, err)
	var enrollResp enrollAgentResponse
	hres := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusOK)
	defer hres.Body.Close()
	require.NoError(t, json.NewDecoder(hres.Body).Decode(&enrollResp))
	require.NotEmpty(t, enrollResp.NodeKey)

	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	h := listHostsRes.Hosts[0]

	s.runDEPSchedule()

	// make sure the host gets post enrollment requests
	checkPostEnrollmentCommands(mdmDevice, true)

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", h.ID), getHostRequest{}, http.StatusOK, &hostResp)
	// 1 profile with fleetd configuration + 1 root CA config
	require.Len(t, *hostResp.Host.MDM.Profiles, 2)

	// Turn MDM off in the host
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", h.ID), nil, http.StatusOK)

	// profiles are removed and the host is no longer enrolled
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", h.ID), getHostRequest{}, http.StatusOK, &hostResp)
	require.Nil(t, hostResp.Host.MDM.Profiles)
	require.Equal(t, "", hostResp.Host.MDM.Name)

	err = mdmDevice.Checkout()
	require.NoError(t, err)

	// Simulate the device getting unassigned from Fleet in ABM
	devices = []godep.Device{
		{SerialNumber: mdmDevice.SerialNumber, Model: "MacBook Pro", OS: "osx", OpType: "deleted", OpDate: time.Now()},
	}

	s.runDEPSchedule()

	a := checkHostDEPAssignProfileResponses([]string{mdmDevice.SerialNumber}, profileAssignmentReqs[0].ProfileUUID, fleet.DEPAssignProfileResponseSuccess)
	require.NotZero(t, a[mdmDevice.SerialNumber].DeletedAt)

	// Now we add the device back into ABM
	profileAssignmentReqs = []profileAssignmentReq{}

	devices = []godep.Device{
		// In https://github.com/fleetdm/fleet/issues/23200, we saw a profileUUID being sent back on
		// the godep.Device in the response from ABM. We're not 100% sure why, but the fact that
		// this field is set was the source of the bug, which is why we're including it here.
		{SerialNumber: mdmDevice.SerialNumber, Model: "MacBook Pro", OS: "osx", OpType: "added", OpDate: time.Now(), ProfileUUID: a[mdmDevice.SerialNumber].ProfileUUID},
	}

	s.runDEPSchedule()

	a = checkHostDEPAssignProfileResponses([]string{mdmDevice.SerialNumber}, profileAssignmentReqs[0].ProfileUUID, fleet.DEPAssignProfileResponseSuccess)
	require.Nil(t, a[mdmDevice.SerialNumber].DeletedAt)

	err = mdmDevice.Enroll()
	require.NoError(t, err)

	// make sure the host gets post enrollment requests
	checkPostEnrollmentCommands(mdmDevice, true)
}

func (s *integrationMDMTestSuite) TestEnforceMiniumOSVersion() {
	t := s.T()
	s.enableABM(t.Name())

	latestMacOSVersion := "14.6.1" // this is the latest version in our test data (see ../mdm/apple/gdmf/testdata/gdmf.json)
	latestMacOSBuild := "23G93"    // this is the latest version in our test data (see ../mdm/apple/gdmf/testdata/gdmf.json)
	deadline := "2023-12-31"
	scepChallenge := "scepcha/><llenge"
	scepURL := s.server.URL + "/mdm/apple/scep"
	mdmURL := s.server.URL + "/mdm/apple/mdm"

	// for our tests, we'll crete two devices: devices[0] will be enrolled with no team and
	// devices[1] will be enrolled with a team (created later in this test)
	devices := []godep.Device{
		{SerialNumber: uuid.New().String(), Model: "MacBook Pro", OS: "osx", OpType: "added"},
		{SerialNumber: uuid.New().String(), Model: "MacBook Pro", OS: "osx", OpType: "added"},
	}
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			require.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()})
			require.NoError(t, err)
		case "/server/devices":
			// This endpoint  is used to get an initial list of
			// devices, return a single device
			err := encoder.Encode(godep.DeviceResponse{Devices: devices})
			require.NoError(t, err)
		case "/devices/sync":
			// This endpoint is polled over time to sync devices from
			// ABM, send a repeated serial and a new one
			err := encoder.Encode(godep.DeviceResponse{Devices: devices, Cursor: "foo"})
			require.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))
			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = make(map[string]string, len(prof.Devices))
			for _, device := range prof.Devices {
				resp.Devices[device] = string(fleet.DEPAssignProfileResponseSuccess)
			}
			err = encoder.Encode(resp)
			require.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	s.runDEPSchedule()

	// confirm that the devices were created
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 2)

	// create a team and add the second device to it
	team := &fleet.Team{
		Name:        t.Name() + "team1",
		Description: "desc team1",
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team
	for _, h := range listHostsRes.Hosts {
		if h.HardwareSerial == devices[1].SerialNumber {
			err := s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{h.ID})
			require.NoError(t, err)
			break
		}
	}

	// this helper function mocks the x-aspen-deviceinfo header that is sent by the device during
	// the enrollment
	encodeDeviceInfo := func(machineInfo fleet.MDMAppleMachineInfo) string {
		body, err := plist.Marshal(machineInfo)
		require.NoError(t, err)

		// body is expected to be a PKCS7 signed message, although we don't currently verify the signature
		signedData, err := pkcs7.NewSignedData(body)
		require.NoError(t, err)
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		crtBytes, err := depot.NewCACert().SelfSign(rand.Reader, key.Public(), key)
		require.NoError(t, err)
		crt, err := x509.ParseCertificate(crtBytes)
		require.NoError(t, err)
		require.NoError(t, signedData.AddSigner(crt, key, pkcs7.SignerInfoConfig{}))
		sig, err := signedData.Finish()
		require.NoError(t, err)

		return base64.StdEncoding.EncodeToString(sig)
	}

	// this helper function calls the /enroll endpoint with the supplied machineInfo (from the test
	// case) and checks for the expected response
	checkMDMEnrollEndpoint := func(machineInfo *fleet.MDMAppleMachineInfo, expectEnrollInfo *mdmtest.AppleEnrollInfo, expectSoftwareUpdate *fleet.MDMAppleSoftwareUpdateRequiredDetails) error {
		request, err := http.NewRequest("GET", s.server.URL+apple_mdm.EnrollPath+"?token="+loadEnrollmentProfileDEPToken(t, s.ds), nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		if machineInfo != nil {
			request.Header.Set("x-apple-aspen-deviceinfo", encodeDeviceInfo(*machineInfo))
		}

		// nolint:gosec // this client is used for testing only
		cc := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
			InsecureSkipVerify: true,
		}))
		response, err := cc.Do(request)
		if err != nil {
			return fmt.Errorf("send request: %w", err)
		}
		defer response.Body.Close()

		switch {
		case expectEnrollInfo != nil:
			require.Equal(t, http.StatusOK, response.StatusCode)
			body, err := io.ReadAll(response.Body)
			if err != nil {
				return fmt.Errorf("read body: %w", err)
			}
			rawProfile := body
			if !bytes.HasPrefix(rawProfile, []byte("<?xml")) {
				p7, err := pkcs7.Parse(body)
				if err != nil {
					return fmt.Errorf("enrollment profile is not XML nor PKCS7 parseable: %w", err)
				}
				err = p7.Verify()
				if err != nil {
					return err
				}
				rawProfile = p7.Content
			}
			enrollInfo, err := mdmtest.ParseEnrollmentProfile(rawProfile)
			if err != nil {
				return fmt.Errorf("parse enrollment profile: %w", err)
			}
			require.NotNil(t, enrollInfo)
			require.Equal(t, expectEnrollInfo, enrollInfo)

			return nil

		case expectSoftwareUpdate != nil:
			require.Equal(t, http.StatusForbidden, response.StatusCode)
			body, err := io.ReadAll(response.Body)
			if err != nil {
				return fmt.Errorf("read body: %w", err)
			}
			var sur fleet.MDMAppleSoftwareUpdateRequired
			require.NoError(t, json.Unmarshal(body, &sur))
			require.NotNil(t, sur)
			require.Equal(t, fleet.MDMAppleSoftwareUpdateRequiredCode, sur.Code)
			require.Equal(t, *expectSoftwareUpdate, sur.Details)

			return nil

		default:
			return fmt.Errorf("request error: %d, %s", response.StatusCode, response.Status)
		}
	}

	// this helper function calls the /mdm/sso endpoint with the supplied machineInfo (from the test
	// case) and checks for the expected response
	checkMDMSSOEndpoint := func(machineInfo *fleet.MDMAppleMachineInfo, expectSoftwareUpdate *fleet.MDMAppleSoftwareUpdateRequiredDetails) error {
		request, err := http.NewRequest("GET", s.server.URL+"/mdm/sso", nil)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		if machineInfo != nil {
			request.Header.Set("x-apple-aspen-deviceinfo", encodeDeviceInfo(*machineInfo))
		}

		// nolint:gosec // this client is used for testing only
		cc := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
			InsecureSkipVerify: true,
		}))
		response, err := cc.Do(request)
		if err != nil {
			return fmt.Errorf("send request: %w", err)
		}
		defer response.Body.Close()

		switch {
		case expectSoftwareUpdate != nil:
			require.Equal(t, http.StatusForbidden, response.StatusCode)
			body, err := io.ReadAll(response.Body)
			if err != nil {
				return fmt.Errorf("read body: %w", err)
			}
			var sur fleet.MDMAppleSoftwareUpdateRequired
			require.NoError(t, json.Unmarshal(body, &sur))
			require.NotNil(t, sur)
			require.Equal(t, fleet.MDMAppleSoftwareUpdateRequiredCode, sur.Code)
			require.Equal(t, *expectSoftwareUpdate, sur.Details)

			return nil

		case response.StatusCode == http.StatusOK:
			// this is the expected happy path based on the test server setup (note that the full
			// SSO callback flow is not being tested here, just the OS enforcement that is tied to
			// the `GET /mdm/sso` initial request)
			// https://github.com/fleetdm/fleet/blob/e62956924bbe041a1e75be2b0b7c6d1dd235a09d/server/service/testing_utils.go#L420-L421
			return nil

		default:
			return fmt.Errorf("request error: %d, %s", response.StatusCode, response.Status)
		}
	}

	// this helper function sets the minimum OS version for the team or no team
	setMinOSVersion := func(minVersion string, deadline string, teamID *uint) {
		raw := json.RawMessage(fmt.Sprintf(`{ "mdm": { "macos_updates": { "minimum_version": "%s", "deadline": "%s" } } }`, minVersion, deadline))
		if teamID == nil {
			acResp := appConfigResponse{}
			s.DoJSON("PATCH", "/api/latest/fleet/config", raw, http.StatusOK, &acResp)
			assert.NotNil(t, acResp.MDM.MacOSUpdates)
		} else {
			tcResp := teamResponse{}
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", *teamID), raw, http.StatusOK, &tcResp)
		}
	}

	// this helper function sets the enable end user authentication for the team or no team
	setEnableEndUserAuth := func(enable bool, teamID *uint) {
		raw := json.RawMessage(fmt.Sprintf(`{ "mdm": { "macos_setup": { "enable_end_user_authentication": %v } } }`, enable))
		if teamID == nil {
			acResp := appConfigResponse{}
			s.DoJSON("PATCH", "/api/latest/fleet/config", raw, http.StatusOK, &acResp)
		} else {
			tcResp := teamResponse{}
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", *teamID), raw, http.StatusOK, &tcResp)
		}
	}

	// configure idp settings
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"end_user_authentication": { "entity_id": "https://example.com", "idp_name": "example-idp", "metadata_url": "https://idp.example.com/metadata" }
			}
		}`), http.StatusOK, &acResp)

	t.Cleanup(func() {
		acResp := appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_updates": { "minimum_version": null, "deadline": null },
				"macos_setup": { "enable_end_user_authentication": false },
				"end_user_authentication": { "entity_id": "", "idp_name": "", "metadata_url": "", "issuer_uri": "", "metadata": "" }
			}
		}`), http.StatusOK, &acResp)
	})

	testCases := []struct {
		name           string
		machineInfo    *fleet.MDMAppleMachineInfo
		updateRequired *fleet.MDMAppleSoftwareUpdateRequiredDetails
		// err is reserved for future test cases; currently we aren't expecting errors with this endpoint
		// because product specs say to allow enrollment to proceed without software update so this
		// is here so we can be explicit about those expectations and allow for future test cases
		// that may need to check for errors
		err string
	}{
		{
			name: "device above latest",
			machineInfo: &fleet.MDMAppleMachineInfo{
				MDMCanRequestSoftwareUpdate: true,
				Product:                     "Mac15,7",
				OSVersion:                   "14.6.2",
				SupplementalBuildVersion:    "IRRELEVANT",
				SoftwareUpdateDeviceID:      "J516sAP",
			},
			updateRequired: nil,
		},
		{
			name: "device equal to latest",
			machineInfo: &fleet.MDMAppleMachineInfo{
				MDMCanRequestSoftwareUpdate: true,
				Product:                     "Mac15,7",
				OSVersion:                   latestMacOSVersion,
				SupplementalBuildVersion:    "IRRELEVANT",
				SoftwareUpdateDeviceID:      "J516sAP",
			},
			updateRequired: nil,
		},
		{
			name: "device below latest",
			machineInfo: &fleet.MDMAppleMachineInfo{
				MDMCanRequestSoftwareUpdate: true,
				Product:                     "Mac15,7",
				OSVersion:                   "14.4",
				SupplementalBuildVersion:    "IRRELEVANT",
				SoftwareUpdateDeviceID:      "J516sAP",
			},
			updateRequired: &fleet.MDMAppleSoftwareUpdateRequiredDetails{
				OSVersion:    latestMacOSVersion,
				BuildVersion: latestMacOSBuild,
			},
		},
		{
			name: "device below latest but MDM cannot request software update",
			machineInfo: &fleet.MDMAppleMachineInfo{
				MDMCanRequestSoftwareUpdate: false,
				Product:                     "Mac15,7",
				OSVersion:                   "14.4",
				SupplementalBuildVersion:    "IRRELEVANT",
				SoftwareUpdateDeviceID:      "J516sAP",
			},
			updateRequired: nil,
		},
		{
			name: "no match for software update device ID",
			machineInfo: &fleet.MDMAppleMachineInfo{
				MDMCanRequestSoftwareUpdate: true,
				Product:                     "Mac15,7",
				OSVersion:                   "14.4",
				SupplementalBuildVersion:    "IRRELEVANT",
				SoftwareUpdateDeviceID:      "INVALID",
			},
			updateRequired: nil,
			err:            "", // no error, allow enrollment to proceed without software update
		},
		{
			name:           "no machine info",
			machineInfo:    nil,
			updateRequired: nil,
			err:            "", // no error, allow enrollment to proceed without software update
		},
		{
			name: "cannot parse OS version",
			machineInfo: &fleet.MDMAppleMachineInfo{
				MDMCanRequestSoftwareUpdate: true,
				Product:                     "Mac15,7",
				OSVersion:                   "INVALID",
				SupplementalBuildVersion:    "IRRELEVANT",
				SoftwareUpdateDeviceID:      "J516sAP",
			},
			updateRequired: nil,
			err:            "", // no error, allow enrollment to proceed without software update
		},
	}

	t.Run("no team setting equal to latest", func(t *testing.T) {
		setMinOSVersion(latestMacOSVersion, deadline, nil)

		t.Run("sso disabled", func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					var mi fleet.MDMAppleMachineInfo
					if tc.machineInfo != nil {
						mi = *tc.machineInfo
						mi.Serial = devices[0].SerialNumber
					}
					var expectEnrollInfo *mdmtest.AppleEnrollInfo
					if tc.updateRequired == nil && tc.err == "" {
						expectEnrollInfo = &mdmtest.AppleEnrollInfo{
							SCEPChallenge: scepChallenge,
							SCEPURL:       scepURL,
							MDMURL:        mdmURL,
						}
					}

					err := checkMDMEnrollEndpoint(&mi, expectEnrollInfo, tc.updateRequired)
					if tc.err != "" {
						require.Error(t, err)
						require.Contains(t, err.Error(), tc.err)
					} else {
						require.NoError(t, err)
					}
				})
			}
		})

		t.Run("sso enabled", func(t *testing.T) {
			setEnableEndUserAuth(true, nil)

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					var mi fleet.MDMAppleMachineInfo
					if tc.machineInfo != nil {
						mi = *tc.machineInfo
						mi.Serial = devices[0].SerialNumber
					}

					err := checkMDMSSOEndpoint(&mi, tc.updateRequired)
					if tc.err != "" {
						require.Error(t, err)
						require.Contains(t, err.Error(), tc.err)
					} else {
						require.NoError(t, err)
					}
				})
			}
		})
	})

	t.Run("team setting equal to latest", func(t *testing.T) {
		setMinOSVersion(latestMacOSVersion, deadline, &team.ID)

		t.Run("sso disabled", func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					if tc.machineInfo != nil {
						tc.machineInfo.Serial = devices[1].SerialNumber
					}
					var expectEnrollInfo *mdmtest.AppleEnrollInfo
					if tc.updateRequired == nil && tc.err == "" {
						expectEnrollInfo = &mdmtest.AppleEnrollInfo{
							SCEPChallenge: "scepcha/><llenge",
							SCEPURL:       s.server.URL + "/mdm/apple/scep",
							MDMURL:        s.server.URL + "/mdm/apple/mdm",
						}
					}

					err := checkMDMEnrollEndpoint(tc.machineInfo, expectEnrollInfo, tc.updateRequired)
					if tc.err != "" {
						require.Error(t, err)
						require.Contains(t, err.Error(), tc.err)
					} else {
						require.NoError(t, err)
					}
				})
			}
		})

		t.Run("sso enabled", func(t *testing.T) {
			setEnableEndUserAuth(true, &team.ID)

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					if tc.machineInfo != nil {
						tc.machineInfo.Serial = devices[0].SerialNumber
					}

					err := checkMDMSSOEndpoint(tc.machineInfo, tc.updateRequired)
					if tc.err != "" {
						require.Error(t, err)
						require.Contains(t, err.Error(), tc.err)
					} else {
						require.NoError(t, err)
					}
				})
			}
		})
	})
}
