package service

// iOS and iPadOS tests for the MDM suite.
//
// Belongs here: iOS/iPadOS enrollment (after DEP sync, OTA, account-driven user enrollment, BYOD with
// an IdP), refetching device information, actions on personal hosts, recreating deleted iPhones, and
// the installed application list command for BYOD devices.
//
// Does not belong here: VPP apps (integration_mdm_vpp_test.go), macOS enrollment
// (integration_mdm_apple_enrollment_test.go), and ADE release (integration_mdm_dep_test.go).

import (
	"bytes"
	"context"
	"crypto/x509"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/WatchBeam/clock"

	"github.com/fleetdm/fleet/v4/server/dev_mode"

	shared_mdm "github.com/fleetdm/fleet/v4/pkg/mdm"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/nanopush"
	mdmtesting "github.com/fleetdm/fleet/v4/server/mdm/testing_utils"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/redis_key_value"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestEnrollAfterDEPSyncIOSIPadOS() {
	t := s.T()

	h, _ := s.createAppleMobileHostThenEnrollMDM("ios")

	// fetch the host, it will match the one created above
	// (NOTE: cannot check the returned OrbitNodeKey, this field is not part of the response)
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, h.ID, hostResp.Host.ID)
	require.NotEqual(t, h.LastEnrolledAt, hostResp.Host.LastEnrolledAt)

	h, _ = s.createAppleMobileHostThenEnrollMDM("ipados")

	// fetch the host, it will match the one created above
	// (NOTE: cannot check the returned OrbitNodeKey, this field is not part of the response)
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, h.ID, hostResp.Host.ID)
	require.NotEqual(t, h.LastEnrolledAt, hostResp.Host.LastEnrolledAt)

	h, _ = s.createIpodHostThenEnrollMDM()

	// fetch the host, it will match the one created above
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, h.ID, hostResp.Host.ID)
	require.NotEqual(t, h.LastEnrolledAt, hostResp.Host.LastEnrolledAt)

	// list commands returns empty set -- no MDM commands should be queued at this point
	var listCmdResp listMDMAppleCommandsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp)
	require.Empty(t, listCmdResp.Results)
}

// assertHostMDMCertificates checks the certificates the host details API reports
// against the templates the device answered CertificateList with. A refetch on
// the device channel only ever sees the system keychain.
func (s *integrationMDMTestSuite) assertHostMDMCertificates(t *testing.T, hostID uint, templates []*x509.Certificate) {
	t.Helper()
	var resp listHostCertificatesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/certificates", hostID), nil, http.StatusOK, &resp)
	require.Len(t, resp.Certificates, len(templates))
	bySerial := make(map[string]*fleet.HostCertificatePayload, len(resp.Certificates))
	for _, c := range resp.Certificates {
		bySerial[c.Serial] = c
	}
	for _, tmpl := range templates {
		serial := tmpl.SerialNumber.Text(16)
		got, ok := bySerial[serial]
		require.True(t, ok, "no certificate reported with serial %s", serial)
		require.Equal(t, fleet.SystemHostCertificate, got.Source)
		require.Empty(t, got.Username)
		require.WithinDuration(t, tmpl.NotBefore, got.NotValidBefore, time.Second)
		require.WithinDuration(t, tmpl.NotAfter, got.NotValidAfter, time.Second)
		require.NotNil(t, got.Subject)
		require.Equal(t, tmpl.Subject.Organization[0], got.Subject.Organization)
	}
}

func (s *integrationMDMTestSuite) TestRefetchIOSIPadOS() {
	t := s.T()

	testCerts := []*x509.Certificate{mdmtesting.NewTestMDMAppleCertTemplate()}

	// Try to refetch host that is not MDM enrolled
	serialNumber := mdmtest.RandSerialNumber()
	fleetHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		HardwareSerial:   serialNumber,
		Platform:         "ipados",
		LastEnrolledAt:   time.Now(),
		DetailUpdatedAt:  time.Now(),
		RefetchRequested: true,
	})
	require.NoError(t, err)
	r := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", fleetHost.ID), nil, http.StatusUnprocessableEntity, "error",
		"host is not enrolled in MDM")
	assert.Contains(t, extractServerErrorText(r.Body), "Host does not have MDM turned on")

	// Enroll host
	host, mdmClient := s.createAppleMobileHostThenEnrollMDM("ios")
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), host.ID, false, true, "https://foo.com", true, "", "", fleet.PersonalEnrollmentTypeNone))

	s.enableABM(t.Name())
	abmTok, err := s.ds.GetABMTokenByOrgName(t.Context(), t.Name())
	require.NoError(t, err)
	s.Require().NoError(s.ds.UpsertMDMAppleHostDEPAssignments(t.Context(), []fleet.Host{*host}, abmTok.ID, nil))

	// Refetch host
	_ = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", host.ID), nil, http.StatusOK)
	const commandsSentPerRefetch = 3
	commandsSent := commandsSentPerRefetch

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, host.ID, hostResp.Host.ID)
	assert.True(t, hostResp.Host.RefetchRequested)

	commands, err := s.ds.GetHostMDMCommands(context.Background(), host.ID)
	require.NoError(t, err)
	require.Len(t, commands, commandsSent)
	requireTrackedRefetchCommands(t, commands, host.ID,
		fleet.RefetchAppsCommandUUIDPrefix, fleet.RefetchCertsCommandUUIDPrefix, fleet.RefetchDeviceCommandUUIDPrefix)

	// Since refetch is already queued up, doing another refetch is a no-op and will not add more MDM commands
	_ = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", host.ID), nil, http.StatusOK)

	// Check the MDM commands and send response
	cmd, err := mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)

	const deviceName = "My iPhone"
	expectedSoftware := []fleet.HostSoftwareEntry{
		{
			Software: fleet.Software{
				BundleIdentifier: "com.evernote.iPhone.Evernote",
				Name:             "Evernote",
				Version:          "10.98.0",
				Source:           "ios_apps",
			},
		},
	}
	require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
	cmd, err = mdmClient.AcknowledgeInstalledApplicationList(mdmClient.UUID, cmd.CommandUUID,
		[]fleet.Software{expectedSoftware[0].Software})
	require.NoError(t, err)
	require.Equal(t, "CertificateList", cmd.Command.RequestType)
	cmd, err = mdmClient.AcknowledgeCertificateList(mdmClient.UUID, cmd.CommandUUID, testCerts)
	require.NoError(t, err)
	require.Equal(t, "DeviceInformation", cmd.Command.RequestType)
	_, err = mdmClient.AcknowledgeDeviceInformation(mdmClient.UUID, cmd.CommandUUID, deviceName, "iPhone SE", "America/Los_Angeles")
	require.NoError(t, err)

	commands, err = s.ds.GetHostMDMCommands(context.Background(), host.ID)
	require.NoError(t, err)
	require.Empty(t, commands)

	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, host.ID, hostResp.Host.ID)
	assert.False(t, hostResp.Host.RefetchRequested)
	assert.Equal(t, deviceName, hostResp.Host.ComputerName)

	for index := range hostResp.Host.Software {
		hostResp.Host.Software[index].ID = 0
	}
	assert.ElementsMatch(t, expectedSoftware, hostResp.Host.Software)

	s.assertHostMDMCertificates(t, host.ID, testCerts)

	// Install the same app for iPadOS
	hostIPad, mdmClientIPad := s.createAppleMobileHostThenEnrollMDM("ipados")

	// Refetch host
	_ = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", hostIPad.ID), nil, http.StatusOK)
	commandsSent += commandsSentPerRefetch

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostIPad.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, hostIPad.ID, hostResp.Host.ID)
	assert.True(t, hostResp.Host.RefetchRequested)

	// Check the MDM commands and send response
	cmd, err = mdmClientIPad.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)

	const deviceNameIPad = "My iPad"
	expectedSoftware = []fleet.HostSoftwareEntry{
		{
			Software: fleet.Software{
				BundleIdentifier: "com.evernote.iPhone.Evernote",
				Name:             "Evernote",
				Version:          "10.98.0",
				Source:           "ipados_apps",
			},
		},
	}
	require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
	cmd, err = mdmClientIPad.AcknowledgeInstalledApplicationList(mdmClientIPad.UUID, cmd.CommandUUID,
		[]fleet.Software{expectedSoftware[0].Software})
	require.NoError(t, err)
	require.Equal(t, "CertificateList", cmd.Command.RequestType)
	cmd, err = mdmClientIPad.AcknowledgeCertificateList(mdmClientIPad.UUID, cmd.CommandUUID, testCerts)
	require.NoError(t, err)
	require.Equal(t, "DeviceInformation", cmd.Command.RequestType)
	cmd, err = mdmClientIPad.AcknowledgeDeviceInformation(mdmClientIPad.UUID, cmd.CommandUUID, deviceNameIPad, "iPad 10", "America/Los_Angeles")
	require.NoError(t, err)
	require.Nil(t, cmd)

	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostIPad.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, hostIPad.ID, hostResp.Host.ID)
	assert.False(t, hostResp.Host.RefetchRequested)
	assert.Equal(t, deviceNameIPad, hostResp.Host.ComputerName)

	for index := range hostResp.Host.Software {
		hostResp.Host.Software[index].ID = 0
	}
	assert.ElementsMatch(t, expectedSoftware, hostResp.Host.Software)

	s.assertHostMDMCertificates(t, hostIPad.ID, testCerts)

	hostIPod, mdmClientIPod := s.createIpodHostThenEnrollMDM()
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), hostIPod.ID, false, true, "https://foo.com", true, "", "", fleet.PersonalEnrollmentTypeNone))

	// Refetch host
	_ = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", hostIPod.ID), nil, http.StatusOK)
	commandsSent += commandsSentPerRefetch

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostIPod.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, hostIPod.ID, hostResp.Host.ID)
	assert.True(t, hostResp.Host.RefetchRequested)

	commands, err = s.ds.GetHostMDMCommands(context.Background(), hostIPod.ID)
	require.NoError(t, err)
	require.Len(t, commands, commandsSentPerRefetch)
	requireTrackedRefetchCommands(t, commands, hostIPod.ID,
		fleet.RefetchAppsCommandUUIDPrefix, fleet.RefetchCertsCommandUUIDPrefix, fleet.RefetchDeviceCommandUUIDPrefix)

	// Since refetch is already queued up, doing another refetch is a no-op and will not add more MDM commands
	_ = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", hostIPod.ID), nil, http.StatusOK)

	// Check the MDM commands and send response
	cmd, err = mdmClientIPod.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)

	const deviceNameIPod = "My iPod"
	expectedSoftware = []fleet.HostSoftwareEntry{
		{
			Software: fleet.Software{
				BundleIdentifier: "com.evernote.iPhone.Evernote",
				Name:             "Evernote",
				Version:          "10.98.0",
				Source:           "ios_apps",
			},
		},
	}
	require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
	cmd, err = mdmClientIPod.AcknowledgeInstalledApplicationList(mdmClientIPod.UUID, cmd.CommandUUID,
		[]fleet.Software{expectedSoftware[0].Software})
	require.NoError(t, err)
	require.Equal(t, "CertificateList", cmd.Command.RequestType)
	cmd, err = mdmClientIPod.AcknowledgeCertificateList(mdmClientIPod.UUID, cmd.CommandUUID, testCerts)
	require.NoError(t, err)
	require.Equal(t, "DeviceInformation", cmd.Command.RequestType)
	_, err = mdmClientIPod.AcknowledgeDeviceInformation(mdmClientIPod.UUID, cmd.CommandUUID, deviceNameIPod, "iPod Touch", "America/Los_Angeles")
	require.NoError(t, err)

	commands, err = s.ds.GetHostMDMCommands(context.Background(), hostIPod.ID)
	require.NoError(t, err)
	require.Empty(t, commands)

	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostIPod.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, hostIPod.ID, hostResp.Host.ID)
	assert.False(t, hostResp.Host.RefetchRequested)

	assert.Equal(t, deviceNameIPod, hostResp.Host.ComputerName)

	for index := range hostResp.Host.Software {
		hostResp.Host.Software[index].ID = 0
	}
	assert.ElementsMatch(t, expectedSoftware, hostResp.Host.Software)

	// TODO: add test for GET /hosts/:id/certificates endpoint, should match up with testCerts

	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))
	ctx := context.Background()
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	// Check that we have the correct software titles
	var resp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &resp, "query", "Evernote")
	expectedTitles := []fleet.SoftwareTitleListResult{
		{
			BundleIdentifier: ptr.String("com.evernote.iPhone.Evernote"),
			Name:             "Evernote",
			Source:           "ios_apps",
			HostsCount:       2,
			VersionsCount:    1,
		},
		{
			BundleIdentifier: ptr.String("com.evernote.iPhone.Evernote"),
			Name:             "Evernote",
			Source:           "ipados_apps",
			HostsCount:       1,
			VersionsCount:    1,
		},
	}
	require.Len(t, resp.SoftwareTitles, 2)
	// Cleaning up the response to make it easier to compare using ElementsMatch
	for index := range resp.SoftwareTitles {
		resp.SoftwareTitles[index].ID = 0
		assert.Len(t, resp.SoftwareTitles[index].Versions, 1)
		resp.SoftwareTitles[index].Versions = nil
	}
	assert.ElementsMatch(t, expectedTitles, resp.SoftwareTitles)

	// Test that software was deleted from device
	_ = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", host.ID), nil, http.StatusOK)
	commandsSent += commandsSentPerRefetch

	// Check the MDM commands and send response
	cmd, err = mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
	cmd, err = mdmClient.AcknowledgeInstalledApplicationList(mdmClient.UUID, cmd.CommandUUID, []fleet.Software{})
	require.NoError(t, err)
	require.Equal(t, "CertificateList", cmd.Command.RequestType)
	cmd, err = mdmClient.AcknowledgeCertificateList(mdmClient.UUID, cmd.CommandUUID, []*x509.Certificate{})
	require.NoError(t, err)
	require.Equal(t, "DeviceInformation", cmd.Command.RequestType)
	const deviceNameRenamed = "My new iPhone"
	cmd, err = mdmClient.AcknowledgeDeviceInformation(mdmClient.UUID, cmd.CommandUUID, deviceNameRenamed, "iPhone SE", "America/Los_Angeles")
	require.NoError(t, err)
	require.Nil(t, cmd)

	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, host.ID, hostResp.Host.ID)
	assert.False(t, hostResp.Host.RefetchRequested)
	assert.Equal(t, deviceNameRenamed, hostResp.Host.ComputerName)
	assert.Empty(t, hostResp.Host.Software)

	// TODO: add test for GET /hosts/:id/certificates endpoint, should be empty

	// Mark host as unenrolled and refetch.
	require.NoError(t, s.ds.UpdateMDMData(ctx, host.ID, false))
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.NoError(t, err)
	require.NotNil(t, hostResp.Host.MDM.EnrollmentStatus)
	assert.Equal(t, "Pending", *hostResp.Host.MDM.EnrollmentStatus)

	// Set iOS detail_updated_at as 2 hours in the past.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE hosts SET detail_updated_at = DATE_SUB(NOW(), INTERVAL 2 HOUR) WHERE id = ?`, host.ID)
		return err
	})
	trigger := triggerRequest{
		Name: string(fleet.CronAppleMDMIPhoneIPadRefetcher),
	}
	_ = s.Do("POST", "/api/latest/fleet/trigger", trigger, http.StatusOK)
	commandsSent += commandsSentPerRefetch

	// Wait until MDM commands are set up
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			commands, err = s.ds.GetHostMDMCommands(context.Background(), host.ID)
			require.NoError(t, err)
			if len(commands) == commandsSentPerRefetch {
				done <- struct{}{}
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Error("Timeout: MDM commands not queued up")
	}

	// Check the MDM commands and send response
	cmd, err = mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
	cmd, err = mdmClient.AcknowledgeInstalledApplicationList(mdmClient.UUID, cmd.CommandUUID, []fleet.Software{})
	require.NoError(t, err)
	require.Equal(t, "CertificateList", cmd.Command.RequestType)
	cmd, err = mdmClient.AcknowledgeCertificateList(mdmClient.UUID, cmd.CommandUUID, []*x509.Certificate{})
	require.Equal(t, "DeviceInformation", cmd.Command.RequestType)
	cmd, err = mdmClient.AcknowledgeDeviceInformation(mdmClient.UUID, cmd.CommandUUID, deviceNameRenamed, "iPhone SE", "America/Los_Angeles")
	require.NoError(t, err)
	require.Nil(t, cmd)

	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	assert.Equal(t, host.ID, hostResp.Host.ID)
	assert.False(t, hostResp.Host.RefetchRequested)
	require.NotNil(t, hostResp.Host.MDM.EnrollmentStatus)
	assert.Equal(t, "On (automatic)", *hostResp.Host.MDM.EnrollmentStatus)

	// TODO: add test for GET /hosts/:id/certificates endpoint, should be empty

	// list commands should return all the commands we sent
	var listCmdResp listMDMAppleCommandsResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple/commands", nil, http.StatusOK, &listCmdResp)
	require.Len(t, listCmdResp.Results, commandsSent)
}

func (s *integrationMDMTestSuite) TestOTAEnrollment() {
	t := s.T()

	// create a global enroll secret
	globalSecret := "global_secret"
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: globalSecret}},
		},
	}, http.StatusOK, &applyResp)

	reqBody := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PRODUCT</key>
	<string></string>
	<key>SERIAL</key>
	<string>foo</string>
	<key>UDID</key>
	<string></string>
	<key>VERSION</key>
	<string></string>
</dict>
</plist>`)

	// Setup signed req body, but also verify we can sign it.
	signedReqBody, cert, key := s.getSignedOTAEnrollmentBody(t, reqBody)

	// ensure fleet profiles
	s.awaitTriggerProfileSchedule(t)

	t.Run("errors", func(t *testing.T) {
		t.Run("if no enroll secret is provided", func(t *testing.T) {
			httpResp := s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment", reqBody, http.StatusForbidden)
			errMsg := extractServerErrorText(httpResp.Body)
			require.Contains(t, errMsg, "Couldn't install the profile. Invalid enroll secret. Please contact your IT admin.")
			require.NoError(t, httpResp.Body.Close())
		})

		t.Run("if no body is provided", func(t *testing.T) {
			httpResp := s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment?enroll_secret=foo", nil, http.StatusBadRequest)
			errMsg := extractServerErrorText(httpResp.Body)
			require.Contains(t, errMsg, "invalid request body")
			require.NoError(t, httpResp.Body.Close())
		})

		t.Run("if body is unsigned", func(t *testing.T) {
			httpResp := s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment?enroll_secret=foo", reqBody, http.StatusBadRequest)
			errMsg := extractServerErrorText(httpResp.Body)
			require.Contains(t, errMsg, "invalid request body")
			require.NoError(t, httpResp.Body.Close())
		})

		t.Run("if invalid apple signature", func(t *testing.T) {
			httpResp := s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment?enroll_secret=foo", signedReqBody, http.StatusForbidden)
			errMsg := extractServerErrorText(httpResp.Body)
			require.Contains(t, errMsg, "Couldn't install the profile. Invalid enroll secret. Please contact your IT admin.")
			require.NoError(t, httpResp.Body.Close())
		})

		t.Run("if invalid device signature", func(t *testing.T) {
			dev_mode.SetOverride("FLEET_DEV_MDM_APPLE_DISABLE_DEVICE_INFO_CERT_VERIFY", "1", t)
			apple_mdm.SetMachineInfoVerificationForTest(t, false)
			httpResp := s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment?enroll_secret=foo", signedReqBody, http.StatusForbidden)
			errMsg := extractServerErrorText(httpResp.Body)
			require.Contains(t, errMsg, "Couldn't install the profile. Invalid enroll secret. Please contact your IT admin.")
			require.NoError(t, httpResp.Body.Close())
		})

		t.Run("if machine info signature verification fails on phase 1", func(t *testing.T) {
			// the device-CA bypass makes the request look like OTA phase 1, but
			// machine info verification is NOT disabled, so the throwaway
			// signature must be rejected
			dev_mode.SetOverride("FLEET_DEV_MDM_APPLE_DISABLE_DEVICE_INFO_CERT_VERIFY", "1", t)
			httpResp := s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment?enroll_secret=foo", signedReqBody, http.StatusBadRequest)
			errMsg := extractServerErrorText(httpResp.Body)
			require.Contains(t, errMsg, "unable to verify deviceinfo signature")
			require.NoError(t, httpResp.Body.Close())
		})

		t.Run("if phase 2 signature is invalid", func(t *testing.T) {
			// a genuine Fleet-signed phase-2 body with its content tampered: the
			// signer cert still chains to Fleet, but the CMS signature no longer
			// matches, so it must be rejected at decode.
			raw, err := os.ReadFile("../mdm/apple/testdata/deviceinfo/ota-phase2-iphone.der")
			require.NoError(t, err)
			tampered := bytes.Clone(raw)
			// flip a printable byte inside the PRODUCT string so the plist still
			// parses but the signed messageDigest no longer matches
			i := bytes.Index(tampered, []byte("iPhone14,5"))
			require.Positive(t, i)
			tampered[i] ^= 0x01
			httpResp := s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment?enroll_secret=foo", tampered, http.StatusBadRequest)
			errMsg := extractServerErrorText(httpResp.Body)
			require.Contains(t, errMsg, "invalid request signature")
			require.NoError(t, httpResp.Body.Close())
		})

		t.Run("if serial is missing", func(t *testing.T) {
			signedData, err := pkcs7.NewSignedData([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>SERIAL</key>
	<string></string>
</dict>
</plist>`))
			require.NoError(t, err)
			require.NoError(t, signedData.AddSigner(cert, key, pkcs7.SignerInfoConfig{}))
			signedReqBody, err = signedData.Finish()
			require.NoError(t, err)
			httpResp := s.DoRawNoAuth("POST", "/api/latest/fleet/ota_enrollment?enroll_secret=foo", signedReqBody, http.StatusBadRequest)
			errMsg := extractServerErrorText(httpResp.Body)
			require.Contains(t, errMsg, "SERIAL is required")
			require.NoError(t, httpResp.Body.Close())
		})

		t.Run("if idp uuid does not match an account", func(t *testing.T) {
			hwModel := "MacBookPro16,1"
			mdmDevice := mdmtest.NewTestMDMClientAppleOTA(
				s.server.URL,
				globalSecret,
				hwModel,
				mdmtest.WithOTAIdpSession(s.mustBYODIdPSession(t, uuid.New().String())),
			)
			err := mdmDevice.Enroll()
			require.Error(t, err)
			require.Contains(t, err.Error(), "404 Not Found")
		})

		t.Run("if idp uuid is required but not set", func(t *testing.T) {
			var specResp applyTeamSpecsResponse
			teamSecret := "team_secret"
			teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "newteam", Secrets: &[]fleet.EnrollSecret{{Secret: teamSecret}}, MDM: fleet.TeamSpecMDM{MacOSSetup: fleet.MacOSSetup{EnableEndUserAuthentication: true}}}}}
			s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &specResp)

			hwModel := "MacBookPro16,1"
			mdmDevice := mdmtest.NewTestMDMClientAppleOTA(
				s.server.URL,
				"team_secret",
				hwModel,
			)
			err := mdmDevice.Enroll()
			require.Error(t, err)
			require.Contains(t, err.Error(), "403 Forbidden")
		})
	})

	t.Run("succeeds", func(t *testing.T) {
		verifySuccessfulOTAEnrollment := func(mdmDevice *mdmtest.TestAppleMDMClient, hwModel, platform string, enrollTime time.Time) getHostResponse {
			var hostByIdentifierResp getHostResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", mdmDevice.UUID), nil, http.StatusOK, &hostByIdentifierResp)
			require.Equal(t, hwModel, hostByIdentifierResp.Host.HardwareModel)
			require.Equal(t, platform, hostByIdentifierResp.Host.Platform)
			require.NotNil(t, hostByIdentifierResp.Host.LastMDMEnrolledAt)
			assert.GreaterOrEqual(t, *hostByIdentifierResp.Host.LastMDMEnrolledAt, enrollTime)
			require.NotNil(t, hostByIdentifierResp.Host.LastMDMCheckedInAt)
			assert.GreaterOrEqual(t, *hostByIdentifierResp.Host.LastMDMCheckedInAt, enrollTime)

			return hostByIdentifierResp
		}

		verifySuccessfulIdpAssociation := func(hostUUID, idpUUID string) {
			hostToIdpMap, err := s.ds.GetMDMIdPAccountsByHostUUIDs(context.Background(), []string{hostUUID})
			require.NoError(t, err)
			idpAccount := hostToIdpMap[hostUUID]
			require.NotNil(t, idpAccount)
			require.Equal(t, idpUUID, idpAccount.UUID)
		}

		t.Run("ota enrolling a macbook", func(t *testing.T) {
			hwModel := "MacBookPro16,1"
			mdmDevice := mdmtest.NewTestMDMClientAppleOTA(
				s.server.URL,
				globalSecret,
				hwModel,
			)
			enrollTime := time.Now().UTC().Truncate(time.Second)
			require.NoError(t, mdmDevice.Enroll())
			s.awaitRunAppleMDMWorkerSchedule()
			checkInstallFleetdCommandSent(t, mdmDevice, true)

			hostByIdentifierResp := verifySuccessfulOTAEnrollment(mdmDevice, hwModel, "darwin", enrollTime)
			require.Nil(t, hostByIdentifierResp.Host.TeamID)

			// Verify that OTA enrollment upserts the macOS entry for software_device_id.
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				var count int
				err := sqlx.GetContext(t.Context(), q, &count, `SELECT COUNT(*) FROM host_mdm_apple_os_updates WHERE host_uuid = ?`, mdmDevice.UUID)
				require.NoError(t, err)
				require.Equal(t, 1, count)
				return nil
			})
		})

		t.Run("ota enrolling an ipad", func(t *testing.T) {
			var specResp applyTeamSpecsResponse
			teamSecret := "team_secret"
			teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "newteam", Secrets: &[]fleet.EnrollSecret{{Secret: teamSecret}}}}}
			s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &specResp)

			hwModel := "iPad13,16"
			mdmDevice := mdmtest.NewTestMDMClientAppleOTA(
				s.server.URL,
				teamSecret,
				hwModel,
			)
			enrollTime := time.Now().UTC().Truncate(time.Second)
			require.NoError(t, mdmDevice.Enroll())
			s.awaitRunAppleMDMWorkerSchedule()
			checkInstallFleetdCommandSent(t, mdmDevice, false)

			hostByIdentifierResp := verifySuccessfulOTAEnrollment(mdmDevice, hwModel, "ipados", enrollTime)
			require.NotNil(t, hostByIdentifierResp.Host.TeamID)
			require.Equal(t, specResp.TeamIDsByName["newteam"], *hostByIdentifierResp.Host.TeamID)
		})

		t.Run("ota enrollment without an idp session does not associate host with idp account", func(t *testing.T) {
			hwModel := "MacBookPro16,1"
			mdmDevice := mdmtest.NewTestMDMClientAppleOTA(
				s.server.URL,
				globalSecret,
				hwModel,
			)
			enrollTime := time.Now().UTC().Truncate(time.Second)
			require.NoError(t, mdmDevice.Enroll())
			s.awaitRunAppleMDMWorkerSchedule()
			checkInstallFleetdCommandSent(t, mdmDevice, true)

			resp := verifySuccessfulOTAEnrollment(mdmDevice, hwModel, "darwin", enrollTime)
			account, err := s.ds.GetMDMIdPAccountByHostUUID(context.Background(), resp.Host.UUID)
			require.NoError(t, err)
			require.Nil(t, account) // We do not fail but return nil for both if not found, and we do not expect an entry here.
		})

		t.Run("ota enrollment with an idp session associates host with idp account", func(t *testing.T) {
			idpEmail := "test@example.com"
			err := s.ds.InsertMDMIdPAccount(context.Background(), &fleet.MDMIdPAccount{
				Username: "test",
				Email:    idpEmail,
			})
			require.NoError(t, err)
			idpAccount, err := s.ds.GetMDMIdPAccountByEmail(context.Background(), idpEmail)
			require.NoError(t, err)
			sessionID := s.mustBYODIdPSession(t, idpAccount.UUID)

			hwModel := "MacBookPro16,1"
			mdmDevice := mdmtest.NewTestMDMClientAppleOTA(
				s.server.URL,
				globalSecret,
				hwModel,
				mdmtest.WithOTAIdpSession(sessionID),
			)
			enrollTime := time.Now().UTC().Truncate(time.Second)
			require.NoError(t, mdmDevice.Enroll())
			s.awaitRunAppleMDMWorkerSchedule()
			checkInstallFleetdCommandSent(t, mdmDevice, true)

			resp := verifySuccessfulOTAEnrollment(mdmDevice, hwModel, "darwin", enrollTime)
			verifySuccessfulIdpAssociation(resp.Host.UUID, idpAccount.UUID)

			// the session is single-use: enrolling burned it
			_, err = shared_mdm.ValidateBYODIdPSession(t.Context(), redis_key_value.New(s.redisPool), clock.C, sessionID)
			require.ErrorAs(t, err, new(*fleet.AuthRequiredError))

			// a second device presenting the same session enrolls without an account
			secondDevice := mdmtest.NewTestMDMClientAppleOTA(
				s.server.URL,
				globalSecret,
				hwModel,
				mdmtest.WithOTAIdpSession(sessionID),
			)
			require.NoError(t, secondDevice.Enroll())
			s.awaitRunAppleMDMWorkerSchedule()
			resp = verifySuccessfulOTAEnrollment(secondDevice, hwModel, "darwin", enrollTime)
			hostToIdpMap, err := s.ds.GetMDMIdPAccountsByHostUUIDs(context.Background(), []string{resp.Host.UUID})
			require.NoError(t, err)
			require.Nil(t, hostToIdpMap[resp.Host.UUID])
		})

		t.Run("ota enrollment with a used idp session is refused where authentication is required", func(t *testing.T) {
			idpEmail := "used@example.com"
			err := s.ds.InsertMDMIdPAccount(context.Background(), &fleet.MDMIdPAccount{
				Username: "used",
				Email:    idpEmail,
			})
			require.NoError(t, err)
			idpAccount, err := s.ds.GetMDMIdPAccountByEmail(context.Background(), idpEmail)
			require.NoError(t, err)
			sessionID := s.mustBYODIdPSession(t, idpAccount.UUID)
			require.NoError(t, shared_mdm.ConsumeBYODIdPSession(t.Context(), redis_key_value.New(s.redisPool), clock.C, sessionID))

			var specResp applyTeamSpecsResponse
			teamSecret := "team_secret_used_session"
			teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "team used session", Secrets: &[]fleet.EnrollSecret{{Secret: teamSecret}}, MDM: fleet.TeamSpecMDM{MacOSSetup: fleet.MacOSSetup{EnableEndUserAuthentication: true}}}}}
			s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &specResp)

			mdmDevice := mdmtest.NewTestMDMClientAppleOTA(
				s.server.URL,
				teamSecret,
				"MacBookPro16,1",
				mdmtest.WithOTAIdpSession(sessionID),
			)
			err = mdmDevice.Enroll()
			require.Error(t, err)
			require.Contains(t, err.Error(), "403 Forbidden")
		})
	})
}

func (s *integrationMDMTestSuite) TestAppleMDMAccountDrivenUserEnrollment() {
	t := s.T()

	// We are going to login and do some SSO operations then do the actual enrollments during the
	// test. Why? Testing the SAML SSO flow requires specific hardcoded server values but the actual
	// server URL changes between runs, so once we've done the SSO operations we reset the server config
	// to the proper value and then fetch the enrollment profiles and do the enrollments.
	// TODO: Is there a better way to do this?
	originalServerUrl := s.server.URL
	s.setUpMDMSSO(t, true)
	abmToken := s.enableABM(t.Name())

	getSSOAccessToken := func(username, password, token string) string {
		ssoResult := s.LoginAccountDrivenEnrollUser(username, password, token)
		// check location for apple-remotemanagement-user-login://authentication-results?access-token=
		returnedLocation, err := ssoResult.Location()
		require.NoError(t, err)
		require.NotNil(t, returnedLocation)
		returnedLocationURL := returnedLocation.String()
		require.True(t, strings.HasPrefix(returnedLocationURL, "apple-remotemanagement-user-login://authentication-results?access-token="))
		accessToken := strings.Split(returnedLocationURL, "apple-remotemanagement-user-login://authentication-results?access-token=")[1]
		return accessToken
	}

	iPhoneAccessToken := getSSOAccessToken("sso_user", "user123#", string(abmToken.EnrollmentURLToken))
	iPadAccessToken := getSSOAccessToken("sso_user2", "user123#", string(abmToken.EnrollmentURLToken))
	oldUrlIphoneAccessToken := getSSOAccessToken("sso_user2", "user123#", "")

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"server_settings": {
			"server_url": "`+originalServerUrl+`"
		}
	}`), http.StatusOK, &acResp)

	iPhoneHwModel := "iPhone14,5"
	iPhoneMdmDevice := mdmtest.NewTestMDMClientAppleAccountDrivenUserEnrollment(
		s.server.URL,
		iPhoneHwModel,
		"", // No bearer token
	)
	require.ErrorContains(t, iPhoneMdmDevice.Enroll(), "401 Unauthorized")

	iPhoneMdmDevice = mdmtest.NewTestMDMClientAppleAccountDrivenUserEnrollment(
		s.server.URL,
		iPhoneHwModel,
		iPhoneAccessToken,
	)
	require.NoError(t, iPhoneMdmDevice.Enroll())
	assert.Equal(t, "sso_user@example.com", iPhoneMdmDevice.EnrollInfo.AssignedManagedAppleID)

	iPhoneHost, err := s.ds.HostByIdentifier(context.Background(), iPhoneMdmDevice.EnrollmentID())
	require.NoError(t, err)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeMDMEnrolled{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_serial": "%s", "enrollment_id": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false, "mdm_platform": "apple", "platform": "ios"}`, iPhoneHost.ID, iPhoneMdmDevice.EnrollmentID(), iPhoneMdmDevice.EnrollmentID(), iPhoneMdmDevice.Model, iPhoneMdmDevice.EnrollmentID()), 0)
	linkedIDPAccount, err := s.ds.GetMDMIdPAccountByHostUUID(context.Background(), iPhoneMdmDevice.EnrollmentID())
	require.NoError(t, err)
	require.NotNil(t, linkedIDPAccount)
	assert.Equal(t, "sso_user@example.com", linkedIDPAccount.Email)

	iPadHwModel := "iPad14,5"
	iPadMdmDevice := mdmtest.NewTestMDMClientAppleAccountDrivenUserEnrollment(
		s.server.URL,
		iPadHwModel,
		"", // No bearer token
	)
	require.ErrorContains(t, iPadMdmDevice.Enroll(), "401 Unauthorized")

	// create a team we can update the ABM token to default the iPad enrollment to
	team, err := s.ds.NewTeam(t.Context(), &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	// TODO: Update to call DS/SVC method
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(t.Context(), "UPDATE abm_tokens SET byod_default_team_id = ? WHERE id = ?", team.ID, abmToken.ID)
		return err
	})

	iPadMdmDevice = mdmtest.NewTestMDMClientAppleAccountDrivenUserEnrollment(
		s.server.URL,
		iPadHwModel,
		iPadAccessToken,
	)
	require.NoError(t, iPadMdmDevice.Enroll())
	assert.Equal(t, "sso_user2@example.com", iPadMdmDevice.EnrollInfo.AssignedManagedAppleID)

	iPadHost, err := s.ds.HostByIdentifier(context.Background(), iPadMdmDevice.EnrollmentID())
	require.NoError(t, err)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeMDMEnrolled{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_serial": "%s", "enrollment_id": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false, "mdm_platform": "apple", "platform": "ipados"}`, iPadHost.ID, iPadMdmDevice.EnrollmentID(), iPadMdmDevice.EnrollmentID(), iPadMdmDevice.Model, iPadMdmDevice.EnrollmentID()), 0)
	linkedIDPAccount, err = s.ds.GetMDMIdPAccountByHostUUID(context.Background(), iPadMdmDevice.EnrollmentID())
	require.NoError(t, err)
	require.NotNil(t, linkedIDPAccount)
	assert.Equal(t, linkedIDPAccount.Email, "sso_user2@example.com")

	oldUrlIphoneMdmDevice := mdmtest.NewTestMDMClientAppleAccountDrivenUserEnrollment(
		s.server.URL,
		"iPhone14,5",
		oldUrlIphoneAccessToken,
	)
	require.NoError(t, oldUrlIphoneMdmDevice.Enroll())
	assert.Equal(t, "sso_user2@example.com", oldUrlIphoneMdmDevice.EnrollInfo.AssignedManagedAppleID)

	oldUrlIphoneHost, err := s.ds.HostByIdentifier(context.Background(), oldUrlIphoneMdmDevice.EnrollmentID())
	require.NoError(t, err)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeMDMEnrolled{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_serial": "%s", "enrollment_id": "%s", "host_display_name": "%s (%s)", "installed_from_dep": false, "mdm_platform": "apple", "platform": "ios"}`, oldUrlIphoneHost.ID, oldUrlIphoneMdmDevice.EnrollmentID(), oldUrlIphoneMdmDevice.EnrollmentID(), oldUrlIphoneMdmDevice.Model, oldUrlIphoneMdmDevice.EnrollmentID()), 0)
	linkedIDPAccount, err = s.ds.GetMDMIdPAccountByHostUUID(context.Background(), oldUrlIphoneMdmDevice.EnrollmentID())
	require.NoError(t, err)
	require.NotNil(t, linkedIDPAccount)
	assert.Equal(t, "sso_user2@example.com", linkedIDPAccount.Email)

	// Account driven enrollment is not yet supported for macOS devices so we expect an error in
	// either case
	macbookHwModel := "Mac16,1"
	macbookMdmDevice := mdmtest.NewTestMDMClientAppleAccountDrivenUserEnrollment(
		s.server.URL,
		macbookHwModel,
		"", // No bearer token
	)

	// A different error than usual for no bearer token because the system will try
	// to initiate the SSO flow if the server returns a 401 unauthorized, and we don't
	// want that.
	require.ErrorContains(t, macbookMdmDevice.Enroll(), "400 Bad Request")

	macbookMdmDevice = mdmtest.NewTestMDMClientAppleAccountDrivenUserEnrollment(
		s.server.URL,
		macbookHwModel,
		iPadAccessToken, // Using iPad's access token. It doesn't matter because it should fail before checking it
	)
	require.ErrorContains(t, macbookMdmDevice.Enroll(), "400 Bad Request")

	// Fetch the hosts we enrolled and check their details. Specifically we want to verify that MDM
	// shows a personal enrollment via all 3 endpoints(list hosts, host details and host MDM
	// details) since they query for the information slightly differently
	listHostsRes := listHostsResponse{}
	var iPhoneHostID *uint
	var iPadHostID *uint
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 3)
	for _, host := range listHostsRes.Hosts {
		switch host.UUID {
		case iPhoneMdmDevice.EnrollmentID():
			assert.Equal(t, iPhoneHwModel, host.HardwareModel)
			assert.Equal(t, iPhoneMdmDevice.EnrollmentID(), host.UUID)
			assert.Equal(t, iPhoneMdmDevice.EnrollmentID(), host.HardwareSerial)
			require.NotNil(t, host.MDM.EnrollmentStatus)
			assert.Equal(t, fleet.MDMEnrollmentStatusPersonal, *host.MDM.EnrollmentStatus)
			assert.True(t, *host.MDM.ConnectedToFleet)
			assert.Nil(t, host.TeamID)
			id := host.ID
			iPhoneHostID = &id
		case iPadMdmDevice.EnrollmentID():
			assert.Equal(t, "ipados", host.Platform)
			assert.Equal(t, iPadMdmDevice.EnrollmentID(), host.UUID)
			assert.Equal(t, iPadMdmDevice.EnrollmentID(), host.HardwareSerial)
			assert.Equal(t, iPadHwModel, host.HardwareModel)
			require.NotNil(t, host.MDM.EnrollmentStatus)
			assert.Equal(t, fleet.MDMEnrollmentStatusPersonal, *host.MDM.EnrollmentStatus)
			assert.True(t, *host.MDM.ConnectedToFleet)
			assert.Equal(t, team.ID, *host.TeamID)
			id := host.ID
			iPadHostID = &id
		case oldUrlIphoneMdmDevice.EnrollmentID():
			// Primarily assert it was enrolled correctly and fallback to unassigned
			assert.Equal(t, "ios", host.Platform)
			assert.Equal(t, fleet.MDMEnrollmentStatusPersonal, *host.MDM.EnrollmentStatus)
			assert.True(t, *host.MDM.ConnectedToFleet)
			assert.Nil(t, host.TeamID)
		}
	}

	require.NotNil(t, iPhoneHostID, "iPhone host not found in the list of hosts")
	require.NotNil(t, iPadHostID, "iPad host not found in the list of hosts")

	// Confirm that host details endpoint contains the expected values for the iPhone
	getHostResp := getDeviceHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", *iPhoneHostID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.EnrollmentStatus)
	assert.Equal(t, fleet.MDMEnrollmentStatusPersonal, *getHostResp.Host.MDM.EnrollmentStatus)
	assert.True(t, *getHostResp.Host.MDM.ConnectedToFleet)
	assert.Equal(t, iPhoneHwModel, getHostResp.Host.HardwareModel)
	assert.Nil(t, getHostResp.Host.HostDetail.TeamID)

	// Confirm that the host MDM endpoint contains the expected values for the iPhone
	// using an inline struct because the definition of fleet.HostMDM makes it unusable here
	getHostMDMResponse := struct {
		EnrollmentStatus string `json:"enrollment_status"`
		ServerURL        string `json:"server_url"`
		Name             string `json:"name"`
		ID               uint   `json:"id"`
	}{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", *iPhoneHostID), nil, http.StatusOK, &getHostMDMResponse)
	assert.Equal(t, fleet.MDMEnrollmentStatusPersonal, getHostMDMResponse.EnrollmentStatus)
	assert.Equal(t, fleet.WellKnownMDMFleet, getHostMDMResponse.Name)

	// Confirm that host details endpoint contains the expected values for the iPad
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", *iPadHostID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.EnrollmentStatus)
	assert.Equal(t, fleet.MDMEnrollmentStatusPersonal, *getHostResp.Host.MDM.EnrollmentStatus)
	assert.True(t, *getHostResp.Host.MDM.ConnectedToFleet)
	assert.Equal(t, team.ID, *getHostResp.Host.TeamID)

	// Confirm that the host MDM endpoint contains the expected values for the iPad
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", *iPadHostID), nil, http.StatusOK, &getHostMDMResponse)
	assert.Equal(t, fleet.MDMEnrollmentStatusPersonal, getHostMDMResponse.EnrollmentStatus)
	assert.Equal(t, fleet.WellKnownMDMFleet, getHostMDMResponse.Name)

	for _, hostID := range []uint{*iPhoneHostID, *iPadHostID} {
		var macadminsData macadminsDataResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostID), nil, http.StatusOK, &macadminsData)
		require.NotNil(t, macadminsData.Macadmins)
		require.NotNil(t, macadminsData.Macadmins.MDM)
		assert.Equal(t, fleet.MDMEnrollmentStatusPersonal, macadminsData.Macadmins.MDM.EnrollmentStatus)
	}
}

func (s *integrationMDMTestSuite) TestAppleMDMActionsOnPersonalHost() {
	t := s.T()

	// We are going to login and do some SSO operations then do the actual enrollments during the
	// test. Why? Testing the SAML SSO flow requires specific hardcoded server values but the actual
	// server URL changes between runs, so once we've done the SSO operations we reset the server config
	// to the proper value and then fetch the enrollment profiles and do the enrollments.
	// TODO: Is there a better way to do this?
	originalServerUrl := s.server.URL
	s.setUpMDMSSO(t, true)
	abmToken := s.enableABM(t.Name())

	getSSOAccessToken := func(username, password string) string {
		ssoResult := s.LoginAccountDrivenEnrollUser(username, password, string(abmToken.EnrollmentURLToken))
		// check location for apple-remotemanagement-user-login://authentication-results?access-token=
		returnedLocation, err := ssoResult.Location()
		require.NoError(t, err)
		require.NotNil(t, returnedLocation)
		returnedLocationURL := returnedLocation.String()
		require.True(t, strings.HasPrefix(returnedLocationURL, "apple-remotemanagement-user-login://authentication-results?access-token="))
		accessToken := strings.Split(returnedLocationURL, "apple-remotemanagement-user-login://authentication-results?access-token=")[1]
		return accessToken
	}

	iPhoneAccessToken := getSSOAccessToken("sso_user", "user123#")

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"server_settings": {
			"server_url": "`+originalServerUrl+`"
		}
	}`), http.StatusOK, &acResp)

	iPhoneHwModel := "iPhone14,5"

	iPhoneMdmDevice := mdmtest.NewTestMDMClientAppleAccountDrivenUserEnrollment(
		s.server.URL,
		iPhoneHwModel,
		iPhoneAccessToken,
	)
	require.NoError(t, iPhoneMdmDevice.Enroll())
	assert.Equal(t, iPhoneMdmDevice.EnrollInfo.AssignedManagedAppleID, "sso_user@example.com")

	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	host := listHostsRes.Hosts[0]
	assert.Equal(t, host.UUID, iPhoneMdmDevice.EnrollmentID())
	require.NotNil(t, host.MDM.EnrollmentStatus)
	assert.Equal(t, fleet.MDMEnrollmentStatusPersonal, *host.MDM.EnrollmentStatus)
	assert.True(t, *host.MDM.ConnectedToFleet)

	// Confirm that locking or wiping the host fails with an appropriate error
	// Lock the host
	r := s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(r.Body), fleet.CantLockPersonalHostsMessage)

	// try to wipe the host
	r = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(r.Body), fleet.CantWipePersonalHostsMessage)

	r = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/clear_passcode", host.ID), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(r.Body), fleet.CantClearPasscodePersonalHostsMessage)

	// Confirm that turning off MDM for personal hosts are allowed - NEEDS to be last, to not turn off MDM for the other checks.
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", host.ID), nil, http.StatusNoContent)
}

// Test for #27231
func (s *integrationMDMTestSuite) TestRecreateDeletedIPhoneBYOD() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	// create a team with its enroll secret
	var specResp applyTeamSpecsResponse
	teamSecret := "team_secret"
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "newteam", Secrets: &[]fleet.EnrollSecret{{Secret: teamSecret}}}}}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &specResp)

	// enroll a BYOD iPhone
	model := "iPhone14,6"
	mdmDevice := mdmtest.NewTestMDMClientAppleOTA(s.server.URL, teamSecret, model)
	require.NoError(t, mdmDevice.Enroll())

	// get its corresponding host
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	host := listHostsRes.Hosts[0]
	require.Equal(t, mdmDevice.UUID, host.UUID)
	require.Equal(t, mdmDevice.SerialNumber, host.HardwareSerial)
	require.NotNil(t, host.TeamID)

	// delete the host
	var delResp deleteHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &delResp)

	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 0)

	// run the reviver cron job, will send a push notification to the deleted host
	oriPushFunc := s.pushProvider.PushFunc
	t.Cleanup(func() { s.pushProvider.PushFunc = oriPushFunc })

	var recordedPushes []*mdm.Push
	var pushMutex sync.Mutex
	s.pushProvider.PushFunc = func(ctx context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		pushMutex.Lock()
		recordedPushes = pushes
		pushMutex.Unlock()
		return mockSuccessfulPush(ctx, pushes)
	}
	err := apple_mdm.IOSiPadOSRevive(context.Background(), s.ds, s.mdmCommander, s.logger)
	require.NoError(t, err)
	pushMutex.Lock()
	require.Len(t, recordedPushes, 1)
	pushMutex.Unlock()

	// do an MDM sync (as the host would when triggered by the notification),
	// will re-create the host and move it to its enrollment team
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd) // no command to process

	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	require.Equal(t, mdmDevice.UUID, listHostsRes.Hosts[0].UUID)
	require.Equal(t, mdmDevice.SerialNumber, listHostsRes.Hosts[0].HardwareSerial)
	require.Equal(t, host.TeamID, listHostsRes.Hosts[0].TeamID)
}

// Test for #22391
func (s *integrationMDMTestSuite) TestRecreateDeletedIPhoneADE() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	device := godep.Device{SerialNumber: "IOS0_SERIAL", Model: "iPhone 16 Pro", OS: "ios", DeviceFamily: "iPhone", OpType: "added"}

	s.enableABM("fleet_ade_test")
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

	// host now exists in phantom (pending enrollment) form
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	host := listHostsRes.Hosts[0].Host
	require.Equal(t, device.SerialNumber, host.HardwareSerial)
	require.Nil(t, host.TeamID)
	require.Empty(t, host.UUID)
	require.NotNil(t, host.MDM.EnrollmentStatus)
	require.Equal(t, "Pending", *host.MDM.EnrollmentStatus)

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.Model = device.Model
	mdmDevice.SerialNumber = device.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// host now exists as a full host
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	host = listHostsRes.Hosts[0].Host
	require.Equal(t, device.SerialNumber, host.HardwareSerial)
	require.Nil(t, host.TeamID)
	require.NotEmpty(t, host.UUID)
	require.NotNil(t, host.MDM.EnrollmentStatus)
	require.Equal(t, "On (automatic)", *host.MDM.EnrollmentStatus)

	// delete the host
	var delResp deleteHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &delResp)

	// host still exists, but back in phantom (pending enrollment) form
	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	host = listHostsRes.Hosts[0].Host
	require.Equal(t, device.SerialNumber, host.HardwareSerial)
	require.Nil(t, host.TeamID)
	require.NotNil(t, host.MDM.EnrollmentStatus)
	require.Equal(t, "Pending", *host.MDM.EnrollmentStatus)

	// Some time later, the refetcher will send commands to refetch that host.
	// Mimic this by setting its detail_updated_at at least 1h in the past and
	// triggering the refetch cron job.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(), `UPDATE hosts SET detail_updated_at = DATE_SUB(NOW(), INTERVAL 2 HOUR) WHERE id = ?`, host.ID)
		return err
	})
	trigger := triggerRequest{
		Name: string(fleet.CronAppleMDMIPhoneIPadRefetcher),
	}
	s.Do("POST", "/api/latest/fleet/trigger", trigger, http.StatusOK)

	// Wait until MDM commands are set up
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			commands, err := s.ds.GetHostMDMCommands(context.Background(), host.ID)
			require.NoError(t, err)
			if len(commands) > 0 {
				done <- struct{}{}
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Error("Timeout: MDM commands not queued up")
	}

	// do an MDM sync of the expected commands, will re-enroll the host
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		switch cmd.Command.RequestType {
		case "InstalledApplicationList":
			cmd, err = mdmDevice.AcknowledgeInstalledApplicationList(mdmDevice.UUID, cmd.CommandUUID, []fleet.Software{})
			require.NoError(t, err)
		case "CertificateList":
			cmd, err = mdmDevice.AcknowledgeCertificateList(mdmDevice.UUID, cmd.CommandUUID, []*x509.Certificate{})
			require.NoError(t, err)
		case "DeviceInformation":
			cmd, err = mdmDevice.AcknowledgeDeviceInformation(mdmDevice.UUID, cmd.CommandUUID, "Test Name", "iPhone 16", "America/Los_Angeles")
			require.NoError(t, err)
		default:
			require.Fail(t, "unexpected command", cmd.Command.RequestType)
		}
	}

	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	host = listHostsRes.Hosts[0].Host
	require.Equal(t, device.SerialNumber, host.HardwareSerial)
	require.Nil(t, host.TeamID)
	require.NotEmpty(t, host.UUID)
	require.NotNil(t, host.MDM.EnrollmentStatus)
	require.Equal(t, "On (automatic)", *host.MDM.EnrollmentStatus)
}

func (s *integrationMDMTestSuite) TestBYODEnrollmentWithIdPEnabled() {
	t := s.T()

	if !hasBuildTag("full") {
		t.Skip("This test requires running with -tags full")
	}

	ctx := t.Context()
	s.setSkipWorkerJobs(t)

	s.setUpMDMSSO(t, true)

	// create a couple teams, one with IdP enabled and one without
	teamNoIdP, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team without idp"})
	require.NoError(t, err)
	teamIdP, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team with idp"})
	require.NoError(t, err)
	teamIdP.Config.MDM.MacOSSetup.EnableEndUserAuthentication = true
	_, err = s.ds.SaveTeam(ctx, teamIdP)
	require.NoError(t, err)

	err = s.ds.ApplyEnrollSecrets(ctx, &teamNoIdP.ID, []*fleet.EnrollSecret{{Secret: "noidp"}})
	require.NoError(t, err)
	err = s.ds.ApplyEnrollSecrets(ctx, &teamIdP.ID, []*fleet.EnrollSecret{{Secret: "idp"}})
	require.NoError(t, err)

	// try to BYOD-enroll in team without IdP enabled, should render the download
	// profile page immediately (200)
	res := s.DoRawNoAuth("GET", "/enroll", nil, http.StatusOK, "enroll_secret", "noidp")
	page, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Contains(t, string(page), "How to enroll your Android device to Fleet")

	// try to BYOD-enroll in team with IdP enabled, should redirect to SSO login
	res = s.DoRawNoAuth("GET", "/enroll", nil, http.StatusSeeOther, "enroll_secret", "idp")
	location := res.Header.Get("Location")
	require.NotEmpty(t, location)
	require.True(t, strings.HasPrefix(location, testSAMLIDPBaseURL+"/simplesaml/"))

	res = s.LoginOTAEnrollSSOUser("sso_user", "user123#", "idp")
	require.Equal(t, http.StatusSeeOther, res.StatusCode)
	location = res.Header.Get("Location")
	t.Logf("SSO login redirect location: %s", location)
	require.NotEmpty(t, location)
	require.True(t, strings.HasPrefix(location, "/enroll")) // expect to be redirect from /enroll page for BYOD

	var byodSession string
	for _, c := range res.Cookies() {
		if c.Name == shared_mdm.BYODIdpCookieName {
			byodSession = c.Value
		}
	}
	require.NotEmpty(t, byodSession)

	// a cookie that merely matches the enrollment reference in the query string
	// is not an authenticated session and still redirects to the SSO login.
	res = s.DoRawWithHeaders("GET", "/enroll", nil, http.StatusSeeOther,
		map[string]string{"Cookie": shared_mdm.BYODIdpCookieName + "=abc"}, "enroll_secret", "idp", "enrollment_reference", "abc")
	location = res.Header.Get("Location")
	require.NotEmpty(t, location)
	require.True(t, strings.HasPrefix(location, testSAMLIDPBaseURL+"/simplesaml/"))

	res = s.DoRawWithHeaders("GET", "/enroll", nil, http.StatusOK,
		map[string]string{"Cookie": shared_mdm.BYODIdpCookieName + "=" + byodSession}, "enroll_secret", "idp")
	page, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Contains(t, string(page), "How to enroll your Android device to Fleet")

	// try to BYOD-enroll with invalid enroll secret, should redirect to SSO
	// login as at least one team has it enabled
	res = s.DoRawNoAuth("GET", "/enroll", nil, http.StatusSeeOther, "enroll_secret", "no-such-secret")
	location = res.Header.Get("Location")
	require.NotEmpty(t, location)
	require.True(t, strings.HasPrefix(location, testSAMLIDPBaseURL+"/simplesaml/"))
}

// TestOTAEnrollSSOWithoutAppleDEPProfile verifies that OTA enrollment SSO
// (used by Android and BYOD iPhone/iPad) succeeds even when no Apple DEP
// automatic enrollment profile exists. This is a regression test for #45024
// where the SSO callback returned "missing profile" on Android-only instances.
func (s *integrationMDMTestSuite) TestOTAEnrollSSOWithoutAppleDEPProfile() {
	t := s.T()
	ctx := t.Context()

	s.setSkipWorkerJobs(t)
	s.setUpMDMSSO(t, false)

	// Create a team with IdP (end user authentication) enabled and an enroll secret.
	teamIdP, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team with idp for ota sso test"})
	require.NoError(t, err)
	teamIdP.Config.MDM.MacOSSetup.EnableEndUserAuthentication = true
	_, err = s.ds.SaveTeam(ctx, teamIdP)
	require.NoError(t, err)
	err = s.ds.ApplyEnrollSecrets(ctx, &teamIdP.ID, []*fleet.EnrollSecret{{Secret: "ota-sso-test"}}) //nolint:gosec // test credential
	require.NoError(t, err)

	// Remove any Apple DEP automatic enrollment profiles to simulate an
	// instance where Apple MDM is not configured (Android-only).
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "DELETE FROM mdm_apple_enrollment_profiles")
		return err
	})

	// Perform the full OTA enrollment SSO flow: GET /enroll → IdP login → callback.
	// Before the fix for #45024, this would fail with "missing profile" because the
	// callback tried to fetch the (now-deleted) Apple DEP enrollment profile.
	res := s.LoginOTAEnrollSSOUser("sso_user", "user123#", "ota-sso-test")
	require.Equal(t, http.StatusSeeOther, res.StatusCode)
	location := res.Header.Get("Location")
	require.NotEmpty(t, location)

	u, err := url.Parse(location)
	require.NoError(t, err)

	// The callback should redirect back to the /enroll page (not to ?error=true).
	require.True(t, strings.HasPrefix(u.Path, "/enroll"), "expected redirect to /enroll, got: %s", location)
	require.Empty(t, u.Query().Get("error"), "expected no error in redirect, got: %s", location)
	require.Empty(t, u.Query().Get("enrollment_reference"), "expected no enrollment_reference in redirect, got: %s", location)
	require.Equal(t, fleet.SSOInitiatorOTAEnroll, u.Query().Get("initiator"))
	var byodCookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == shared_mdm.BYODIdpCookieName {
			byodCookie = c
		}
	}
	require.NotNil(t, byodCookie, "expected BYOD IdP session cookie")
	require.NotEmpty(t, byodCookie.Value)
	require.Equal(t, int(shared_mdm.BYODIdPSessionTTL.Seconds()), byodCookie.MaxAge)
}

func (s *integrationMDMTestSuite) TestIOSiPadOSRefetch() {
	ctx := s.T().Context()

	successfulPushUUID := "successful-uuid"
	failedPushUUID := "failed-uuid"

	// Set up test data

	// Perform the MDM enrollment.
	mdmEnrollInfo := mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}
	model := "iPhone14,6"

	successfulHost, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:  mdmtest.RandSerialNumber(),
		UUID:            successfulPushUUID,
		Platform:        string(fleet.IOSPlatform),
		DetailUpdatedAt: time.Now().Add(-2 * time.Hour), // ensure refetch is needed
	})
	require.NoError(s.T(), err)

	successfulMdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, model)
	successfulMdmDevice.SerialNumber = successfulHost.HardwareSerial
	err = successfulMdmDevice.Enroll()
	require.NoError(s.T(), err)

	failedHost, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:  mdmtest.RandSerialNumber(),
		UUID:            failedPushUUID,
		Platform:        string(fleet.IOSPlatform),
		DetailUpdatedAt: time.Now().Add(-2 * time.Hour), // ensure refetch is needed
	})
	require.NoError(s.T(), err)

	failedMdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, model)
	failedMdmDevice.SerialNumber = failedHost.HardwareSerial
	err = failedMdmDevice.Enroll()
	require.NoError(s.T(), err)

	failedHostTokenInactive, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:  mdmtest.RandSerialNumber(),
		UUID:            failedPushUUID,
		Platform:        string(fleet.IOSPlatform),
		DetailUpdatedAt: time.Now().Add(-2 * time.Hour), // ensure refetch is needed
	})
	require.NoError(s.T(), err)

	failedMdmDeviceTokenInactive := mdmtest.NewTestMDMClientAppleDirect(mdmEnrollInfo, model)
	failedMdmDeviceTokenInactive.SerialNumber = failedHostTokenInactive.HardwareSerial
	err = failedMdmDeviceTokenInactive.Enroll()
	require.NoError(s.T(), err)

	originalPushFunc := s.pushProvider.PushFunc
	s.T().Cleanup(func() {
		s.pushProvider.PushFunc = originalPushFunc
	})
	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		require.Len(s.T(), pushes, 1)
		pushObject := pushes[0]
		switch pushObject.PushMagic {
		case "pushmagic" + successfulMdmDevice.SerialNumber:
			return map[string]*push.Response{
				pushObject.Token.String(): {
					Id:  successfulPushUUID,
					Err: nil,
				},
			}, nil
		case "pushmagic" + failedMdmDeviceTokenInactive.SerialNumber:
			return map[string]*push.Response{
				pushObject.Token.String(): {
					Id: failedPushUUID,
					// same shape the nanopush provider returns on a 410
					Err: fmt.Errorf("push HTTP status: 410: %w",
						&nanopush.JSONPushError{Reason: "Unregistered", Timestamp: time.Now().UnixMilli()}),
				},
			}, nil
		case "pushmagic" + failedMdmDevice.SerialNumber:
			return map[string]*push.Response{
				pushObject.Token.String(): {
					Id:  failedPushUUID,
					Err: errors.New("random apns error"),
				},
			}, nil
		}
		return nil, errors.New("unknown device")
	}

	err = apple_mdm.IOSiPadOSRefetch(ctx, s.ds, s.mdmCommander, s.logger, s.fleetSvc.NewActivity)
	require.NoError(s.T(), err) // Verify it not longer throws an error

	// Verify successful is still enrolled
	successfulHostMDM, err := s.ds.GetHostMDM(ctx, successfulHost.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), successfulHostMDM)
	require.True(s.T(), successfulHostMDM.Enrolled)

	// Verify random APNS error host is still enrolled
	failedHostMDM, err := s.ds.GetHostMDM(ctx, failedHost.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), failedHostMDM)
	require.True(s.T(), failedHostMDM.Enrolled)

	// Verify device token inactive host is no longer enrolled
	failedHostMDMTokenInactive, err := s.ds.GetHostMDM(ctx, failedHostTokenInactive.ID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), failedHostMDMTokenInactive)
	require.False(s.T(), failedHostMDMTokenInactive.Enrolled)
}

func (s *integrationMDMTestSuite) TestInstalledApplicationListCommandForBYODiDevices() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := t.Context()

	// create a manually-enrolled (BYOD) ios host
	hostBYOD, mdmClientBYOD := s.createAppleMobileHostThenEnrollMDM("ios")
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostBYOD.ID, false, true, s.server.URL, false, "", "", fleet.PersonalEnrollmentTypeNone))

	// Refetch host
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", hostBYOD.ID), nil, http.StatusOK)

	// check expected command types
	commands, err := s.ds.GetHostMDMCommands(ctx, hostBYOD.ID)
	require.NoError(t, err)
	require.Len(t, commands, 3)
	requireTrackedRefetchCommands(t, commands, hostBYOD.ID,
		fleet.RefetchAppsCommandUUIDPrefix, fleet.RefetchCertsCommandUUIDPrefix, fleet.RefetchDeviceCommandUUIDPrefix)

	checkExpectedCommands := func(mdmClient *mdmtest.TestAppleMDMClient, managedOnly bool, wantAppListCount int) {
		var installedAppListCount int
		cmd, err := mdmClient.Idle()
		require.NoError(t, err)
		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			switch cmd.Command.RequestType {
			case "InstalledApplicationList":
				installedAppListCount++
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
				require.NotNil(t, fullCmd.Command)
				require.NotNil(t, fullCmd.Command.InstalledApplicationList)
				require.Equal(t, managedOnly, fullCmd.Command.InstalledApplicationList.ManagedAppsOnly)

				cmd, err = mdmClient.AcknowledgeInstalledApplicationList(mdmClient.UUID, cmd.CommandUUID, []fleet.Software{})
				require.NoError(t, err)

			case "DeviceInformation":
				cmd, err = mdmClient.AcknowledgeDeviceInformation(mdmClient.UUID, cmd.CommandUUID, "My iPhone", "iPhone X", "America/Los_Angeles")
				require.NoError(t, err)

			default:
				cmd, err = mdmClient.Acknowledge(cmd.CommandUUID)
				require.NoError(t, err)
			}
		}
		require.Equal(t, wantAppListCount, installedAppListCount)
	}

	checkExpectedCommands(mdmClientBYOD, true, 1)

	// create a company-owned ios host
	hostDEP, mdmClientDEP := s.createAppleMobileHostThenEnrollMDM("ios")
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostDEP.ID, false, true, s.server.URL, true, "", "", fleet.PersonalEnrollmentTypeNone))

	// Refetch host
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", hostDEP.ID), nil, http.StatusOK)

	commands, err = s.ds.GetHostMDMCommands(ctx, hostDEP.ID)
	require.NoError(t, err)
	require.Len(t, commands, 3)
	requireTrackedRefetchCommands(t, commands, hostDEP.ID,
		fleet.RefetchAppsCommandUUIDPrefix, fleet.RefetchCertsCommandUUIDPrefix, fleet.RefetchDeviceCommandUUIDPrefix)

	checkExpectedCommands(mdmClientDEP, false, 1)

	// run the cron-based refetch, will not do anything as the devices were just refetched
	err = apple_mdm.IOSiPadOSRefetch(ctx, s.ds, s.mdmCommander, s.logger, s.fleetSvc.NewActivity)
	require.NoError(t, err)

	checkExpectedCommands(mdmClientBYOD, true, 0)
	checkExpectedCommands(mdmClientDEP, false, 0)

	// change the detail_updated_at of the devices to > 1h ago, and run the refetch again
	hostBYOD.DetailUpdatedAt = time.Now().Add(-24 * time.Hour)
	hostDEP.DetailUpdatedAt = time.Now().Add(-24 * time.Hour)
	require.NoError(t, s.ds.UpdateHost(ctx, hostBYOD))
	require.NoError(t, s.ds.UpdateHost(ctx, hostDEP))

	// run the cron-based refetch again, will enqueue the commands with the correct managed only flag
	err = apple_mdm.IOSiPadOSRefetch(ctx, s.ds, s.mdmCommander, s.logger, s.fleetSvc.NewActivity)
	require.NoError(t, err)

	checkExpectedCommands(mdmClientBYOD, true, 1)
	checkExpectedCommands(mdmClientDEP, false, 1)
}
