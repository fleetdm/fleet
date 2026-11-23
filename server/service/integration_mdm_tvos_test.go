package service

import (
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestEnrollAndRefetchTvOS() {
	t := s.T()
	ctx := t.Context()

	host, mdmClient := s.createAppleMobileHostThenEnrollMDM("tvos")
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, "https://foo.com", true, "", "", fleet.PersonalEnrollmentTypeNone))

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, "tvos", hostResp.Host.Platform)
	require.NotEqual(t, host.LastEnrolledAt, hostResp.Host.LastEnrolledAt)

	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", host.ID), nil, http.StatusOK)

	cmd, err := mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
	cmd, err = mdmClient.AcknowledgeInstalledApplicationList(mdmClient.UUID, cmd.CommandUUID, []fleet.Software{
		{BundleIdentifier: "com.netflix.Netflix", Name: "Netflix", Version: "17.1.0", Source: "tvos_apps"},
	})
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "CertificateList", cmd.Command.RequestType)
	cmd, err = mdmClient.AcknowledgeCertificateList(mdmClient.UUID, cmd.CommandUUID, nil)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "DeviceInformation", cmd.Command.RequestType)
	require.NotContains(t, string(cmd.Raw), "BatteryLevel")
	require.NotContains(t, string(cmd.Raw), "DeviceCapacity")
	_, err = mdmClient.AcknowledgeDeviceInformationWithExtra(mdmClient.UUID, cmd.CommandUUID, "Lobby TV", "AppleTV14,1", "America/New_York", "26.4", "")
	require.NoError(t, err)

	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.False(t, hostResp.Host.RefetchRequested)
	require.Equal(t, "tvos", hostResp.Host.Platform)
	require.Equal(t, "Lobby TV", hostResp.Host.ComputerName)
	require.Equal(t, "tvOS", hostResp.Host.OSName)
	require.Equal(t, "26.4", hostResp.Host.OSVersionOnly)
	require.Len(t, hostResp.Host.Software, 1)
	require.Equal(t, "tvos_apps", hostResp.Host.Software[0].Source)

	var summaryResp getHostSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "platform", "tvos")
	require.EqualValues(t, 1, summaryResp.TotalsHostsCount)
	require.Len(t, summaryResp.Platforms, 1)
	require.Equal(t, "tvos", summaryResp.Platforms[0].Platform)

	res := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), fleet.CantLockTvOSHostsMessage)
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), fleet.CantUnlockTvOSHostsMessage)
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/clear_passcode", host.ID), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "ClearPasscode command is only available for iOS and iPadOS")

	// none of the refused actions queued a command
	cmd, err = mdmClient.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)
}

func (s *integrationMDMTestSuite) TestAppleTVEnrollsWithoutEndUserAuth() {
	t := s.T()
	ctx := t.Context()
	abmTok := s.enableABM(t.Name())
	s.setSkipWorkerJobs(t)
	// machine info blobs in this test are signed with a throwaway cert, not an
	// Apple device identity
	apple_mdm.SetMachineInfoVerificationForTest(t, false)

	staticToken := func() string {
		var token string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &token, `SELECT COALESCE(MAX(token), '') FROM mdm_apple_enrollment_profiles WHERE type = ?`, fleet.MDMAppleEnrollmentTypeAutomatic)
		})
		return token
	}
	if staticToken() == "" {
		require.NoError(t, apple_mdm.NewDEPService(s.ds, s.depStorage, s.logger).CreateDefaultAutomaticProfile(ctx))
	}
	token := staticToken()

	euaTeam, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:   t.Name() + " eua",
		Config: fleet.TeamConfig{MDM: fleet.TeamMDM{MacOSSetup: fleet.MacOSSetup{EnableEndUserAuthentication: true}}},
	})
	require.NoError(t, err)

	// deviceFamily is what Apple Business reported for the serial, product what the device says it is
	newDEPHost := func(deviceFamily, product string) (*fleet.Host, string) {
		t.Helper()
		host, err := s.ds.NewHost(ctx, &fleet.Host{
			Hostname:       "eua-dep-host-" + uuid.NewString()[:8],
			HardwareSerial: mdmtest.RandSerialNumber(),
			UUID:           uuid.NewString(),
			Platform:       "darwin",
			TeamID:         &euaTeam.ID,
		})
		require.NoError(t, err)
		require.NoError(t, s.ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*host}, abmTok.ID, nil))
		if deviceFamily != "" {
			require.NoError(t, s.ds.SetHostDEPAssignmentsDeviceFamily(ctx, abmTok.ID, deviceFamily, []string{host.HardwareSerial}))
		}
		di, err := mdmtest.EncodeDeviceInfo(fleet.MDMAppleMachineInfo{Serial: host.HardwareSerial, UDID: host.UUID, Product: product})
		require.NoError(t, err)
		return host, di
	}
	const rejectedActivity = "host_enrollment_rejected"

	appleTV, di := newDEPHost(fleet.DEPDeviceFamilyAppleTV, "AppleTV14,1")
	s.downloadAndVerifyEnrollmentProfile(t, optsDownloadEnrollProf{token: token, diParam: di})
	require.Zero(t, s.countHostActivitiesOfType(appleTV.ID, rejectedActivity))

	for _, tc := range []struct {
		name, deviceFamily, product string
	}{
		{"a Mac that Apple Business reported as a Mac", "Mac", "Mac15,7"},
		{"a device claiming to be an Apple TV that Apple Business reported as a Mac", "Mac", "AppleTV14,1"},
		{"a device claiming to be an Apple TV with no recorded device family", "", "AppleTV14,1"},
		{"a Mac whose serial is recorded as an Apple TV", fleet.DEPDeviceFamilyAppleTV, "Mac15,7"},
	} {
		_, di := newDEPHost(tc.deviceFamily, tc.product)
		res := s.DoRawNoAuth("GET", "/api/mdm/apple/enroll", nil, http.StatusForbidden, "token", token, "deviceinfo", di)
		require.Contains(t, extractServerErrorText(res.Body), "can't enroll right now", tc.name)
	}
}
