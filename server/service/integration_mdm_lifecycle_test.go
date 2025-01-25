package service

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/groob/plist"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

// NOTE: the mantra for lifecycle events is:
//   - Noah: When MDM is turned on, install fleetd, bootstrap package (if DEP),
//     and profiles. Don't clear host vitals (everything you see on the Host
//     details page)
//   - Noah: On re-enrollment, don't clear host vitals.
//   - Noah: On lock and wipe, don't clear host vitals.
//   - Noah: On delete, clear host vitals.

// NOTE: ADE lifecycle events are part of the integration_mdm_dep_test.go file

type mdmLifecycleAssertion[T any] func(t *testing.T, host *fleet.Host, device T)

func (s *integrationMDMTestSuite) TestTurnOnLifecycleEventsApple() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	s.setupLifecycleSettings()

	testCases := []struct {
		Name   string
		Action mdmLifecycleAssertion[*mdmtest.TestAppleMDMClient]
	}{
		{
			"wiped host turns on MDM",
			func(t *testing.T, host *fleet.Host, device *mdmtest.TestAppleMDMClient) {
				s.Do(
					"POST",
					fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID),
					nil,
					http.StatusOK,
				)

				cmd, err := device.Idle()
				require.NoError(t, err)
				for cmd != nil {
					cmd, err = device.Acknowledge(cmd.CommandUUID)
					require.NoError(t, err)
				}

				require.NoError(t, device.Enroll())
			},
		},
		{
			"locked host turns on MDM",
			func(t *testing.T, host *fleet.Host, device *mdmtest.TestAppleMDMClient) {
				s.Do(
					"POST",
					fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID),
					nil,
					http.StatusOK,
				)

				cmd, err := device.Idle()
				require.NoError(t, err)
				for cmd != nil {
					cmd, err = device.Acknowledge(cmd.CommandUUID)
					require.NoError(t, err)
				}

				require.NoError(t, device.Enroll())
			},
		},
		{
			"host turns on MDM features out of the blue",
			func(t *testing.T, host *fleet.Host, device *mdmtest.TestAppleMDMClient) {
				require.NoError(t, device.Enroll())
			},
		},
		{
			"IT admin turns off MDM for a host via the UI then host turns on MDM",
			func(t *testing.T, host *fleet.Host, device *mdmtest.TestAppleMDMClient) {
				originalPushMock := s.pushProvider.PushFunc
				defer func() { s.pushProvider.PushFunc = originalPushMock }()

				s.pushProvider.PushFunc = func(ctx context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
					res, err := mockSuccessfulPush(ctx, pushes)
					require.NoError(t, err)
					err = device.Checkout()
					require.NoError(t, err)
					return res, err
				}

				s.Do(
					"DELETE",
					fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", host.ID),
					nil,
					http.StatusOK,
				)

				require.NoError(t, device.Enroll())
			},
		},
		{
			"host is deleted then turns on MDM",
			func(t *testing.T, host *fleet.Host, device *mdmtest.TestAppleMDMClient) {
				var delResp deleteHostResponse
				s.DoJSON(
					"DELETE",
					fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID),
					nil,
					http.StatusOK,
					&delResp,
				)

				dupeClient := mdmtest.NewTestMDMClientAppleDirect(
					mdmtest.AppleEnrollInfo{
						SCEPChallenge: s.scepChallenge,
						SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
						MDMURL:        s.server.URL + apple_mdm.MDMPath,
					}, "MacBookPro16,1")
				dupeClient.UUID = device.UUID
				dupeClient.SerialNumber = device.SerialNumber
				dupeClient.Model = device.Model
				require.NoError(t, dupeClient.Enroll())

				*device = *dupeClient
			},
		},
		{
			"host is deleted in bulk then turns on MDM",
			func(t *testing.T, host *fleet.Host, device *mdmtest.TestAppleMDMClient) {
				req := deleteHostsRequest{
					IDs: []uint{host.ID},
				}
				resp := deleteHostsResponse{}
				s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusOK, &resp)

				dupeClient := mdmtest.NewTestMDMClientAppleDirect(
					mdmtest.AppleEnrollInfo{
						SCEPChallenge: s.scepChallenge,
						SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
						MDMURL:        s.server.URL + apple_mdm.MDMPath,
					}, "MacBookPro16,1")
				dupeClient.UUID = device.UUID
				dupeClient.SerialNumber = device.SerialNumber
				dupeClient.Model = device.Model
				require.NoError(t, dupeClient.Enroll())

				*device = *dupeClient
			},
		},
		{
			"host is deleted then osquery enrolls then turns on MDM",
			func(t *testing.T, host *fleet.Host, device *mdmtest.TestAppleMDMClient) {
				var delResp deleteHostResponse
				s.DoJSON(
					"DELETE",
					fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID),
					nil,
					http.StatusOK,
					&delResp,
				)

				var err error
				host.OsqueryHostID = ptr.String(t.Name())
				host, err = s.ds.NewHost(context.Background(), host)
				require.NoError(t, err)

				setOrbitEnrollment(t, host, s.ds)
				deviceToken := uuid.NewString()
				err = s.ds.SetOrUpdateDeviceAuthToken(context.Background(), host.ID, deviceToken)
				require.NoError(t, err)

				device.SetDesktopToken(deviceToken)
				require.NoError(t, device.Enroll())
			},
		},
	}

	assertAction := func(t *testing.T, host *fleet.Host, device *mdmtest.TestAppleMDMClient, action mdmLifecycleAssertion[*mdmtest.TestAppleMDMClient]) {
		fCmds, fSumm, fHostMDM := s.recordAppleHostStatus(host, device)

		action(t, host, device)

		// reload the host by identifier, tests might
		// delete hosts and create new records with different IDs
		var err error
		host, err = s.ds.HostByIdentifier(context.Background(), host.UUID)
		require.NoError(t, err)

		sCmds, sSumm, sHostMDM := s.recordAppleHostStatus(host, device)

		// post asssertions
		require.ElementsMatch(t, fCmds, sCmds)
		require.Equal(t, fSumm, sSumm)
		require.Equal(t, fHostMDM, sHostMDM)
	}

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			t.Run("manual enrollment", func(t *testing.T) {
				host, device := createHostThenEnrollMDM(s.ds, s.server.URL, t)
				assertAction(t, host, device, tt.Action)
			})

			t.Run("automatic enrollment", func(t *testing.T) {
				device := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, "")
				s.enableABM(t.Name())
				s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					encoder := json.NewEncoder(w)
					switch r.URL.Path {
					case "/session":
						_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
					case "/profile":
						err := encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
						require.NoError(t, err)
					case "/profile/devices":
						err := encoder.Encode(godep.ProfileResponse{
							ProfileUUID: "abc",
							Devices:     map[string]string{},
						})
						require.NoError(t, err)
					case "/server/devices", "/devices/sync":
						err := encoder.Encode(godep.DeviceResponse{
							Devices: []godep.Device{
								{
									SerialNumber: device.SerialNumber,
									Model:        device.Model,
									OS:           "osx",
									OpType:       "added",
								},
							},
						})
						require.NoError(t, err)
					}
				}))

				s.runDEPSchedule()
				depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
				device.SetDEPToken(depURLToken)
				var err error
				host, err := s.ds.HostByIdentifier(context.Background(), device.SerialNumber)
				require.NoError(t, err)
				require.NoError(t, device.Enroll())

				assertAction(t, host, device, tt.Action)
			})
		})
	}
}

func (s *integrationMDMTestSuite) TestTurnOnLifecycleEventsWindows() {
	t := s.T()
	s.setupLifecycleSettings()

	testCases := []struct {
		Name   string
		Action mdmLifecycleAssertion[*mdmtest.TestWindowsMDMClient]
	}{
		{
			"wiped host turns on MDM",
			func(t *testing.T, host *fleet.Host, device *mdmtest.TestWindowsMDMClient) {
				s.Do(
					"POST",
					fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID),
					nil,
					http.StatusNoContent,
				)

				status, err := s.ds.GetHostLockWipeStatus(context.Background(), host)
				require.NoError(t, err)

				cmds, err := device.StartManagementSession()
				require.NoError(t, err)

				// two status + the wipe command we enqueued
				require.Len(t, cmds, 3)
				wipeCmd := cmds[status.WipeMDMCommand.CommandUUID]
				require.NotNil(t, wipeCmd)
				require.Equal(t, wipeCmd.Verb, fleet.CmdExec)
				require.Len(t, wipeCmd.Cmd.Items, 1)
				require.EqualValues(t, "./Device/Vendor/MSFT/RemoteWipe/doWipeProtected", *wipeCmd.Cmd.Items[0].Target)

				msgID, err := device.GetCurrentMsgID()
				require.NoError(t, err)

				device.AppendResponse(fleet.SyncMLCmd{
					XMLName: xml.Name{Local: fleet.CmdStatus},
					MsgRef:  &msgID,
					CmdRef:  &status.WipeMDMCommand.CommandUUID,
					Cmd:     ptr.String("Exec"),
					Data:    ptr.String("200"),
					Items:   nil,
					CmdID:   fleet.CmdID{Value: uuid.NewString()},
				})
				cmds, err = device.SendResponse()
				require.NoError(t, err)
				// the ack of the message should be the only returned command
				require.Len(t, cmds, 1)

				// re-enroll
				require.NoError(t, device.Enroll())
			},
		},
		{
			"locked host turns on MDM",
			func(t *testing.T, host *fleet.Host, device *mdmtest.TestWindowsMDMClient) {
				s.Do(
					"POST",
					fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID),
					nil,
					http.StatusNoContent,
				)

				status, err := s.ds.GetHostLockWipeStatus(context.Background(), host)
				require.NoError(t, err)

				var orbitScriptResp orbitPostScriptResultResponse
				s.DoJSON(
					"POST",
					"/api/fleet/orbit/scripts/result",
					json.RawMessage(
						fmt.Sprintf(
							`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`,
							*host.OrbitNodeKey,
							status.LockScript.ExecutionID,
						),
					),
					http.StatusOK,
					&orbitScriptResp,
				)

				require.NoError(t, device.Enroll())
			},
		},
		{
			"host turns on MDM features out of the blue",
			func(t *testing.T, host *fleet.Host, device *mdmtest.TestWindowsMDMClient) {
				require.NoError(t, device.Enroll())
			},
		},
		{
			"host is deleted then osquery enrolls then turns on MDM",
			func(t *testing.T, host *fleet.Host, device *mdmtest.TestWindowsMDMClient) {
				var delResp deleteHostResponse
				s.DoJSON(
					"DELETE",
					fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID),
					nil,
					http.StatusOK,
					&delResp,
				)

				var err error
				host.OsqueryHostID = ptr.String(t.Name())
				host, err = s.ds.NewHost(context.Background(), host)
				require.NoError(t, err)

				orbitKey := setOrbitEnrollment(t, host, s.ds)
				host.OrbitNodeKey = &orbitKey
				if !strings.Contains(device.TokenIdentifier, "@") {
					device.TokenIdentifier = orbitKey
				}
				device.HardwareID = host.UUID
				device.DeviceID = host.UUID

				require.NoError(t, device.Enroll())
			},
		},
	}

	assertAction := func(t *testing.T, host *fleet.Host, device *mdmtest.TestWindowsMDMClient, action mdmLifecycleAssertion[*mdmtest.TestWindowsMDMClient]) {
		fCmds, fSumm, fHostMDM := s.recordWindowsHostStatus(host, device)

		action(t, host, device)

		// reload the host by identifier, tests might
		// delete hosts and create new records with different IDs
		var err error
		host, err = s.ds.HostByIdentifier(context.Background(), host.UUID)
		require.NoError(t, err)

		sCmds, sSumm, sHostMDM := s.recordWindowsHostStatus(host, device)

		// post asssertions
		require.Len(t, sCmds, len(fCmds))
		require.ElementsMatch(t, fCmds, sCmds)
		require.Equal(t, fSumm, sSumm)
		require.Equal(t, fHostMDM, sHostMDM)
	}

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			t.Run("programmatic enrollment", func(t *testing.T) {
				host, device := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
				err := s.ds.SetOrUpdateMDMData(context.Background(), host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "")
				require.NoError(t, err)
				assertAction(t, host, device, tt.Action)
			})

			t.Run("automatic enrollment", func(t *testing.T) {
				if strings.Contains(tt.Name, "wipe") {
					t.Skip("wipe tests are not supported for windows automatic enrollment until we fix #TODO")
				}

				err := s.ds.ApplyEnrollSecrets(context.Background(), nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
				require.NoError(t, err)

				host := createOrbitEnrolledHost(t, "windows", "windows_automatic", s.ds)

				azureMail := "foo.bar.baz@example.com"
				device := mdmtest.NewTestMDMClientWindowsAutomatic(s.server.URL, azureMail)
				device.HardwareID = host.UUID
				device.DeviceID = host.UUID
				require.NoError(t, device.Enroll())

				err = s.ds.SetOrUpdateMDMData(context.Background(), host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "")
				require.NoError(t, err)

				assertAction(t, host, device, tt.Action)
			})
		})
	}
}

// Hardcode response type because we are using a custom json marshaling so
// using getHostMDMResponse fails with "JSON unmarshaling is not supported for HostMDM".
type jsonMDM struct {
	EnrollmentStatus string `json:"enrollment_status"`
	ServerURL        string `json:"server_url"`
	Name             string `json:"name,omitempty"`
	ID               *uint  `json:"id,omitempty"`
}
type getHostMDMResponseTest struct {
	HostMDM *jsonMDM
	Err     error `json:"error,omitempty"`
}

func (s *integrationMDMTestSuite) recordWindowsHostStatus(
	host *fleet.Host,
	device *mdmtest.TestWindowsMDMClient,
) ([]fleet.ProtoCmdOperation, getHostMDMSummaryResponse, getHostMDMResponseTest) {
	t := s.T()

	var recordedCmds []fleet.ProtoCmdOperation
	cmds, err := device.StartManagementSession()
	require.NoError(t, err)

	msgID, err := device.GetCurrentMsgID()
	require.NoError(t, err)
	for _, c := range cmds {
		cmdID := c.Cmd.CmdID
		status := syncml.CmdStatusOK
		device.AppendResponse(fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdStatus},
			MsgRef:  &msgID,
			CmdRef:  &cmdID.Value,
			Cmd:     ptr.String(c.Verb),
			Data:    &status,
			Items:   nil,
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
		})
		c.Cmd.CmdID.Value = ""
		c.Cmd.CmdRef = nil
		recordedCmds = append(recordedCmds, c)
	}

	_, err = device.SendResponse()
	require.NoError(t, err)

	mdmAgg := getHostMDMSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/summary/mdm", nil, http.StatusOK, &mdmAgg)

	ghr := getHostMDMResponseTest{}
	s.DoJSON(
		"GET",
		fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", host.ID),
		nil,
		http.StatusOK,
		&ghr,
	)

	return recordedCmds, mdmAgg, ghr
}

func (s *integrationMDMTestSuite) recordAppleHostStatus(
	host *fleet.Host,
	device *mdmtest.TestAppleMDMClient,
) ([]*micromdm.CommandPayload, getHostMDMSummaryResponse, getHostMDMResponseTest) {
	t := s.T()

	s.runWorker()
	s.awaitTriggerProfileSchedule(t)

	var cmds []*micromdm.CommandPayload

	cmd, err := device.Idle()
	require.NoError(t, err)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload

		// command uuid is a random value, we only care that's set
		require.NotEmpty(t, cmd.CommandUUID)

		// strip the signature of the profiles so they can be easily compared
		if cmd.Command.RequestType == "InstallProfile" {
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			fullCmd.CommandUUID = ""
			p7, err := pkcs7.Parse(fullCmd.Command.InstallProfile.Payload)
			require.NoError(t, err)
			fullCmd.Command.InstallProfile.Payload = p7.Content
		}
		cmds = append(cmds, &fullCmd)

		cmd, err = device.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	mdmAgg := getHostMDMSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/summary/mdm", nil, http.StatusOK, &mdmAgg)

	ghr := getHostMDMResponseTest{}
	s.DoJSON(
		"GET",
		fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", host.ID),
		nil,
		http.StatusOK,
		&ghr,
	)

	return cmds, mdmAgg, ghr
}

func (s *integrationMDMTestSuite) setupLifecycleSettings() {
	t := s.T()
	ctx := context.Background()
	// add bootstrap package
	_ = s.ds.DeleteMDMAppleBootstrapPackage(ctx, 0)
	bp, err := os.ReadFile(filepath.Join("testdata", "bootstrap-packages", "signed.pkg"))
	require.NoError(t, err)
	s.uploadBootstrapPackage(
		&fleet.MDMAppleBootstrapPackage{Bytes: bp, Name: "pkg.pkg", TeamID: 0},
		http.StatusOK,
		"",
	)

	// enable disk encryption
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	  "mdm": { "macos_settings": {"enable_disk_encryption": true} }
  }`), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.EnableDiskEncryption.Value)

	// add profiles (windows, mac)
	s.Do(
		"POST",
		"/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "N1", Contents: mobileconfigForTest("N1", "I1")},
			{Name: "N2", Contents: syncMLForTest("./Foo/Bar")},
			{Name: "N3", Contents: declarationForTest("D1")},
		}},
		http.StatusNoContent,
	)
}

// Host is renewing SCEP certificates
func (s *integrationMDMTestSuite) TestLifecycleSCEPCertExpiration() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	// ensure there's a token for automatic enrollments
	s.enableABM(t.Name())
	s.runDEPSchedule()

	// add a device that's manually enrolled
	desktopToken := uuid.New().String()
	manualHost := createOrbitEnrolledHost(t, "darwin", "h1", s.ds)
	err := s.ds.SetOrUpdateDeviceAuthToken(context.Background(), manualHost.ID, desktopToken)
	require.NoError(t, err)
	manualEnrolledDevice := mdmtest.NewTestMDMClientAppleDesktopManual(s.server.URL, desktopToken)
	manualEnrolledDevice.UUID = manualHost.UUID
	manualEnrolledDevice.SerialNumber = manualHost.HardwareSerial
	err = manualEnrolledDevice.Enroll()
	require.NoError(t, err)

	// add a device that's automatically enrolled
	automaticHost := createOrbitEnrolledHost(t, "darwin", "h2", s.ds)
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	automaticEnrolledDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	automaticEnrolledDevice.UUID = automaticHost.UUID
	automaticEnrolledDevice.SerialNumber = automaticHost.HardwareSerial
	err = automaticEnrolledDevice.Enroll()
	require.NoError(t, err)

	// add a device that's automatically enrolled with a server ref
	automaticHostWithRef := createOrbitEnrolledHost(t, "darwin", "h3", s.ds)
	automaticEnrolledDeviceWithRef := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	automaticEnrolledDeviceWithRef.UUID = automaticHostWithRef.UUID
	automaticEnrolledDeviceWithRef.SerialNumber = automaticHostWithRef.HardwareSerial
	err = automaticEnrolledDeviceWithRef.Enroll()
	require.NoError(
		t,
		s.ds.SetOrUpdateMDMData(
			ctx,
			automaticHostWithRef.ID,
			false,
			true,
			s.server.URL,
			true,
			fleet.WellKnownMDMFleet,
			"foo",
		),
	)
	require.NoError(t, err)

	// add a device that was migrated from a third party mdm via
	// "touchless" migration
	migratedHost := createOrbitEnrolledHost(t, "darwin", "h4", s.ds)
	migratedDevice := mdmtest.NewTestMDMClientAppleDesktopManual(s.server.URL, desktopToken)
	migratedDevice.UUID = migratedHost.UUID
	migratedDevice.SerialNumber = migratedHost.HardwareSerial
	err = migratedDevice.Enroll()
	require.NoError(t, err)
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
                  UPDATE nano_enrollments
                  SET enrolled_from_migration = 1
                  WHERE id = ?
		`, migratedDevice.UUID)
		return err
	})

	// add global profiles
	globalProfiles := [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
	}
	s.Do(
		"POST",
		"/api/v1/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{Profiles: globalProfiles},
		http.StatusNoContent,
	)

	ackAllCommands := func(device *mdmtest.TestAppleMDMClient) {
		cmd, err := device.Idle()
		require.NoError(t, err)
		for cmd != nil {
			cmd, err = device.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
	}

	// ack all commands to install profiles
	ackAllCommands(manualEnrolledDevice)
	ackAllCommands(automaticEnrolledDevice)
	ackAllCommands(automaticEnrolledDeviceWithRef)
	ackAllCommands(migratedDevice)

	// simulate a device with two certificates by re-enrolling one of them
	err = manualEnrolledDevice.Enroll()
	require.NoError(t, err)
	ackAllCommands(manualEnrolledDevice)

	cert, key, err := generateCertWithAPNsTopic()
	require.NoError(t, err)
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(s.T(), &fleetCfg, cert, key, "")
	logger := kitlog.NewJSONLogger(os.Stdout)

	// run without expired certs, no command enqueued
	err = RenewSCEPCertificates(ctx, logger, s.ds, &fleetCfg, s.mdmCommander)
	require.NoError(t, err)
	cmd, err := manualEnrolledDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	cmd, err = automaticEnrolledDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	cmd, err = automaticEnrolledDeviceWithRef.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	cmd, err = migratedDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	expireCerts := func() {
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `
                  UPDATE nano_cert_auth_associations
                  SET cert_not_valid_after = DATE_SUB(CURDATE(), INTERVAL 1 YEAR)
                  WHERE id IN (?, ?, ?, ?)
		`, manualHost.UUID, automaticHost.UUID, automaticHostWithRef.UUID, migratedDevice.UUID)
			return err
		})
	}

	// expire all the certs we just created
	expireCerts()

	// generate a new config here so we can manipulate the certs.
	err = RenewSCEPCertificates(ctx, logger, s.ds, &fleetCfg, s.mdmCommander)
	require.NoError(t, err)

	checkRenewCertCommand := func(device *mdmtest.TestAppleMDMClient, enrollRef string, wantProfile string) {
		var renewCmd *mdm.Command
		cmd, err := device.Idle()
		require.NoError(t, err)
		for cmd != nil {
			if cmd.Command.RequestType == "InstallProfile" {
				renewCmd = cmd
			}
			cmd, err = device.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
		require.NotNil(t, renewCmd)
		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(renewCmd.Raw, &fullCmd))

		if wantProfile == "" {
			s.verifyEnrollmentProfile(fullCmd.Command.InstallProfile.Payload, enrollRef)
		} else {
			p7, err := pkcs7.Parse(fullCmd.Command.InstallProfile.Payload)
			require.NoError(t, err)
			rootCA := x509.NewCertPool()

			assets, err := s.ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
				fleet.MDMAssetCACert,
			}, nil)
			require.NoError(t, err)

			require.True(t, rootCA.AppendCertsFromPEM(assets[fleet.MDMAssetCACert].Value))
			require.NoError(t, p7.VerifyWithChain(rootCA))
			require.Equal(t, wantProfile, string(p7.Content))
		}
	}

	checkRenewCertCommand(manualEnrolledDevice, "", "")
	checkRenewCertCommand(automaticEnrolledDevice, "", "")
	checkRenewCertCommand(automaticEnrolledDeviceWithRef, "foo", "")

	// migrated device doesn't receive any commands because
	// `FLEET_SILENT_MIGRATION_ENROLLMENT_PROFILE` is not set
	_, err = migratedDevice.Idle()
	require.NoError(t, err)

	// set the env var, and run the cron
	t.Setenv("FLEET_SILENT_MIGRATION_ENROLLMENT_PROFILE", base64.StdEncoding.EncodeToString([]byte("<foo></foo>")))
	err = RenewSCEPCertificates(ctx, logger, s.ds, &fleetCfg, s.mdmCommander)
	require.NoError(t, err)
	checkRenewCertCommand(migratedDevice, "", "<foo></foo>")

	// another cron run shouldn't enqueue more commands
	err = RenewSCEPCertificates(ctx, logger, s.ds, &fleetCfg, s.mdmCommander)
	require.NoError(t, err)

	cmd, err = manualEnrolledDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	cmd, err = automaticEnrolledDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	cmd, err = automaticEnrolledDeviceWithRef.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	cmd, err = migratedDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// devices renew their SCEP cert by re-enrolling.
	require.NoError(t, manualEnrolledDevice.Enroll())
	require.NoError(t, automaticEnrolledDevice.Enroll())
	require.NoError(t, automaticEnrolledDeviceWithRef.Enroll())
	require.NoError(t, migratedDevice.Enroll())

	// no new commands are enqueued right after enrollment
	cmd, err = manualEnrolledDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	cmd, err = automaticEnrolledDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	cmd, err = automaticEnrolledDeviceWithRef.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	cmd, err = migratedDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// handle the case of a host being deleted, see https://github.com/fleetdm/fleet/issues/19149
	expireCerts()
	req := deleteHostsRequest{
		IDs: []uint{manualHost.ID},
	}
	resp := deleteHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusOK, &resp)
	err = RenewSCEPCertificates(ctx, logger, s.ds, &fleetCfg, s.mdmCommander)
	require.NoError(t, err)
	checkRenewCertCommand(automaticEnrolledDevice, "", "")
	checkRenewCertCommand(automaticEnrolledDeviceWithRef, "foo", "")

	// migrated device is still marked as migrated
	var stillMigrated bool
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &stillMigrated, `
                  SELECT enrolled_from_migration FROM nano_enrollments
                  WHERE id = ?
		`, migratedDevice.UUID)
	})
	require.True(t, stillMigrated)
}
