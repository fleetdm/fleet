package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	kitlog "github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/groob/plist"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
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

func (s *integrationMDMTestSuite) createLifecycleHosts() (map[string]*fleet.Host, map[string]*mdmtest.TestAppleMDMClient, map[string]*mdmtest.TestWindowsMDMClient) {
	// create a windows mdm turned on host, programmatic
	// create a windows mdm turned on host, automatic
	// create a macOS mdm turned on host, manual
	// create a macOS mdm turned on host, automatic
	return map[string]*fleet.Host{}, map[string]*mdmtest.TestAppleMDMClient{}, map[string]*mdmtest.TestWindowsMDMClient{}
}

func (s *integrationMDMTestSuite) setupLifecycleSettings() {
}

// A host is wiped from the Fleet UI, then re-enrolls
func (s *integrationMDMTestSuite) TestLifecycleWiped() {
	t := s.T()
	ctx := context.Background()
	s.setupLifecycleSettings()
	hosts, appleDevices, _ := s.createLifecycleHosts()

	for key, host := range hosts {
		s.Do(
			"POST",
			fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID),
			nil,
			http.StatusNoContent,
		)

		switch host.Platform {
		case "darwin":
			cmd, err := appleDevices[key].Idle()
			require.NoError(t, err)
			for cmd != nil {
				cmd, err = appleDevices[key].Acknowledge(cmd.CommandUUID)
				require.NoError(t, err)
			}
		case "windows":
			status, err := s.ds.GetHostLockWipeStatus(ctx, host)
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
		}

	}
}

// A host is locked from the Fleet UI, then re-enrolls
func (s *integrationMDMTestSuite) TestLifecycleLocked() {
	t := s.T()
	ctx := context.Background()
	s.setupLifecycleSettings()
	hosts, appleDevices, _ := s.createLifecycleHosts()

	for key, host := range hosts {
		s.Do(
			"POST",
			fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID),
			nil,
			http.StatusNoContent,
		)

		switch host.Platform {
		case "darwin":
			cmd, err := appleDevices[key].Idle()
			require.NoError(t, err)
			for cmd != nil {
				cmd, err = appleDevices[key].Acknowledge(cmd.CommandUUID)
				require.NoError(t, err)
			}
		case "windows":
			status, err := s.ds.GetHostLockWipeStatus(ctx, host)
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
		}

	}
}

// A host turns-on MDM features out of the blue (could be wiped physically without us
// knowing, could be an unenrollment that didn't send a CheckOut message, etc)
func (s *integrationMDMTestSuite) TestLifecycleReTurnOnMDM() {
	t := s.T()
	s.setupLifecycleSettings()
	_, appleDevices, windowsDevices := s.createLifecycleHosts()

	// TODO: assert all verified

	for _, d := range appleDevices {
		require.NoError(t, d.Enroll())
	}

	for _, d := range windowsDevices {
		require.NoError(t, d.Enroll())
	}

	// TODO: assert all pending
	// TODO: assert bootstrap package sent for all DEP
	// TODO: assert fleetd sent

}

// IT admin turns off MDM for a host via the UI
func (s *integrationMDMTestSuite) TestLifecycleTurnOffViaUI() {
	t := s.T()
	s.setupLifecycleSettings()
	hosts, appleDevices, windowsDevices := s.createLifecycleHosts()

	for _, host := range hosts {
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", host.ID), nil, http.StatusOK)
	}

	// TODO: assert no profiles

	for _, d := range appleDevices {
		require.NoError(t, d.Enroll())
	}

	for _, d := range windowsDevices {
		require.NoError(t, d.Enroll())
	}

	// TODO: assert all pending
	// TODO: assert bootstrap package sent for all DEP
	// TODO: assert fleetd sent

}

// Host turns off MDM features (eg: manual enrollment profile is removed)
func (s *integrationMDMTestSuite) TestLifecycleTurnOffInDevice() {
	t := s.T()
	s.setupLifecycleSettings()
	_, appleDevices, _ := s.createLifecycleHosts()

	for _, d := range appleDevices {
		require.NoError(t, d.Checkout())
	}

	// TODO: assert no profiles

	for _, d := range appleDevices {
		require.NoError(t, d.Enroll())
	}

	// TODO: assert all pending
	// TODO: assert bootstrap package sent for all DEP
	// TODO: assert fleetd sent
}

// Host is osquery enrolled, then turns on MDM features
func (s *integrationMDMTestSuite) TestLifecycleTurnOnOsqueryFirst() {
}

// Host turns on MDM features before enrolling in osquery (ADE, Windows Azure,
// manual enrollment with a config profile given by the IT admin)
func (s *integrationMDMTestSuite) TestLifecycleTurnOnMDMFirst() {
}

// Host migrates from third-party MDM to Fleet
func (s *integrationMDMTestSuite) TestLifecycleMigrate() {
}

// Host is deleted
func (s *integrationMDMTestSuite) TestLifecycleHostDeleted() {
}

// Host is renewing SCEP certificates
func (s *integrationMDMTestSuite) TestLifecycleSCEPCertExpiration() {
	t := s.T()
	ctx := context.Background()
	// ensure there's a token for automatic enrollments
	s.mockDEPResponse(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
	}))
	s.runDEPSchedule()

	// add a device that's manually enrolled
	desktopToken := uuid.New().String()
	manualHost := createOrbitEnrolledHost(t, "darwin", "h1", s.ds)
	err := s.ds.SetOrUpdateDeviceAuthToken(context.Background(), manualHost.ID, desktopToken)
	require.NoError(t, err)
	manualEnrolledDevice := mdmtest.NewTestMDMClientAppleDesktopManual(s.server.URL, desktopToken)
	manualEnrolledDevice.UUID = manualHost.UUID
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
	// ack all commands to install profiles
	cmd, err := manualEnrolledDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		cmd, err = manualEnrolledDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	cmd, err = automaticEnrolledDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		cmd, err = automaticEnrolledDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	cmd, err = automaticEnrolledDeviceWithRef.Idle()
	require.NoError(t, err)
	for cmd != nil {
		cmd, err = automaticEnrolledDeviceWithRef.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	cert, key, err := generateCertWithAPNsTopic()
	require.NoError(t, err)
	fleetCfg := config.TestConfig()
	config.SetTestMDMConfig(s.T(), &fleetCfg, cert, key, testBMToken, "")
	logger := kitlog.NewJSONLogger(os.Stdout)

	// run without expired certs, no command enqueued
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

	// expire all the certs we just created
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `
                  UPDATE nano_cert_auth_associations
                  SET cert_not_valid_after = DATE_SUB(CURDATE(), INTERVAL 1 YEAR)
                  WHERE id IN (?, ?, ?)
		`, manualHost.UUID, automaticHost.UUID, automaticHostWithRef.UUID)
		return err
	})

	// generate a new config here so we can manipulate the certs.
	err = RenewSCEPCertificates(ctx, logger, s.ds, &fleetCfg, s.mdmCommander)
	require.NoError(t, err)

	checkRenewCertCommand := func(device *mdmtest.TestAppleMDMClient, enrollRef string) {
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
		s.verifyEnrollmentProfile(fullCmd.Command.InstallProfile.Payload, enrollRef)
	}

	checkRenewCertCommand(manualEnrolledDevice, "")
	checkRenewCertCommand(automaticEnrolledDevice, "")
	checkRenewCertCommand(automaticEnrolledDeviceWithRef, "foo")

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

	// devices renew their SCEP cert by re-enrolling.
	require.NoError(t, manualEnrolledDevice.Enroll())
	require.NoError(t, automaticEnrolledDevice.Enroll())
	require.NoError(t, automaticEnrolledDeviceWithRef.Enroll())

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
}
