package service

// MDM SSO and IdP tests for the MDM suite.
//
// Belongs here: SSO for Fleet users and for end users during MDM enrollment, device SSO, binding SSO
// sessions to setup experience hosts, SCIM-provided user data, and keeping manually mapped IdP users
// across re-enrollment.
//
// Does not belong here: BYOD enrollment with an IdP (integration_mdm_ios_ipados_test.go), IdP
// variables in host name templates (integration_mdm_host_name_template_test.go), and Platform SSO
// (integration_mdm_apple_psso_test.go).

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestSSO() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	// machine info blobs in this test are signed with a throwaway cert, not an
	// Apple device identity
	apple_mdm.SetMachineInfoVerificationForTest(t, false)

	lastSubmittedProfile := &godep.Profile{}
	mdmDevice, wantSettings := s.setUpEndUserAuthentication(t, lastSubmittedProfile, true)
	di, err := mdmtest.EncodeDeviceInfo(fleet.MDMAppleMachineInfo{
		Serial: mdmDevice.SerialNumber,
		UDID:   mdmDevice.UUID,
	})
	require.NoError(t, err)

	// patch without specifying the mdm sso settings fields and an unrelated
	// field, should not remove them
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusOK, &acResp)
	assert.Equal(t, wantSettings, acResp.MDM.EndUserAuthentication.SSOProviderSettings)

	s.runWorker()

	// patch with explicitly empty mdm sso settings fields, would remove
	// them but this is a dry-run
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "",
				"idp_name": "",
				"metadata_url": ""
			},
			"macos_setup": {
				"enable_end_user_authentication": false
			}
		}
	}`), http.StatusOK, &acResp, "dry_run", "true")
	assert.Equal(t, wantSettings, acResp.MDM.EndUserAuthentication.SSOProviderSettings)

	s.runWorker()

	// patch with explicitly empty mdm sso settings fields, fails because end user auth is still enabled
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "",
				"idp_name": "",
				"metadata_url": ""
			}
		}
	}`), http.StatusUnprocessableEntity, &acResp)

	// patch with explicitly empty mdm sso settings fields and disabled end user auth, removes them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"end_user_authentication": {
				"entity_id": "",
				"idp_name": "",
				"metadata_url": ""
			},
			"macos_setup": {
				"enable_end_user_authentication": false
			}
		}
	}`), http.StatusOK, &acResp)
	assert.Empty(t, acResp.MDM.EndUserAuthentication.SSOProviderSettings)

	s.runWorker()
	assert.Empty(t, lastSubmittedProfile.ConfigurationWebURL)
	assert.NotEmpty(t, lastSubmittedProfile.URL)

	// "mdm.test.com" entity ID is defined in `tools/saml/config.php`.
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"server_settings": {
			"server_url": "https://localhost:8080"
		},
		"mdm": {
			"end_user_authentication": {
				"entity_id": "mdm.test.com",
				"idp_name": "SimpleSAML",
				"metadata_url": "%s"
			},
			"macos_setup": {
				"enable_end_user_authentication": true,
				"lock_end_user_info": true
			}
		}
	}`, testSAMLIDPMetadataURL)), http.StatusOK, &acResp)

	s.runWorker()
	require.Contains(t, lastSubmittedProfile.URL, acResp.ServerSettings.ServerURL+"/mdm/sso")
	require.Equal(t, acResp.ServerSettings.ServerURL+"/mdm/sso", lastSubmittedProfile.ConfigurationWebURL)

	res := s.LoginMDMSSOUser("sso_user", "user123#", di)
	require.NotEmpty(t, res.Header.Get("Location"))
	require.Equal(t, http.StatusSeeOther, res.StatusCode)

	u, err := url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q := u.Query()
	user1EnrollRef := s.mdmIdPAccountUUID(t, "sso_user@example.com")
	// without an EULA uploaded
	require.False(t, q.Has("eula_token"))
	require.True(t, q.Has("profile_token"))
	require.False(t, q.Has("enrollment_reference"))
	require.False(t, q.Has("error"))
	// the url retrieves a valid profile
	s.downloadAndVerifyEnrollmentProfile(t, optsDownloadEnrollProf{
		basePath: "/api/mdm/apple/enroll",
		token:    q.Get("profile_token"),
		diParam:  di,
	})

	// IdP info stored is accurate for the account
	s.checkStoredIdPInfo(t, user1EnrollRef, "sso_user", "SSO User 1", "sso_user@example.com")

	// the profile token is a one-time token, not the static automatic enrollment token
	staticProf, err := s.ds.GetMDMAppleEnrollmentProfileByType(t.Context(), fleet.MDMAppleEnrollmentTypeAutomatic)
	require.NoError(t, err)
	require.NotEqual(t, staticProf.Token, q.Get("profile_token"))
	s.checkOneTimeDEPEnrollmentToken(t, di, q.Get("profile_token"))

	boundUser1 := fmt.Sprintf(`{"host_uuid": %q, "idp_email": "sso_user@example.com"}`, mdmDevice.UUID)
	boundActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeBoundHostToIdPAccount{}.ActivityName(), boundUser1, 0)

	res = s.LoginMDMSSOUser("sso_user", "user123#", di)
	require.NotEmpty(t, res.Header.Get("Location"))
	require.Equal(t, http.StatusSeeOther, res.StatusCode)

	u, err = url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q = u.Query()
	// without an EULA uploaded
	require.False(t, q.Has("eula_token"))
	require.True(t, q.Has("profile_token"))
	require.False(t, q.Has("enrollment_reference"))
	require.False(t, q.Has("error"))
	// the url retrieves a valid profile
	s.downloadAndVerifyEnrollmentProfile(t, optsDownloadEnrollProf{
		basePath: "/api/mdm/apple/enroll",
		token:    q.Get("profile_token"),
		diParam:  di,
	})

	// IdP info stored is accurate for the account
	s.checkStoredIdPInfo(t, user1EnrollRef, "sso_user", "SSO User 1", "sso_user@example.com")

	// re-enrolling as the same account logs nothing new
	s.lastActivityOfTypeMatches(fleet.ActivityTypeBoundHostToIdPAccount{}.ActivityName(), boundUser1, boundActID)

	res = s.LoginMDMSSOUser("sso_user", "user123#", di)
	require.NotEmpty(t, res.Header.Get("Location"))
	require.Equal(t, http.StatusSeeOther, res.StatusCode)

	// upload an EULA
	pdfBytes := []byte("%PDF-1.pdf-contents")
	pdfName := "eula.pdf"
	s.uploadEULA(&fleet.MDMEULA{Bytes: pdfBytes, Name: pdfName}, http.StatusOK, "")

	res = s.LoginMDMSSOUser("sso_user", "user123#", di)
	require.NotEmpty(t, res.Header.Get("Location"))
	require.Equal(t, http.StatusSeeOther, res.StatusCode)
	u, err = url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q = u.Query()
	// with an EULA uploaded, all values are present
	require.True(t, q.Has("eula_token"))
	require.True(t, q.Has("profile_token"))
	require.False(t, q.Has("enrollment_reference"))
	require.False(t, q.Has("error"))
	// the url retrieves a valid profile
	prof := s.downloadAndVerifyEnrollmentProfile(t, optsDownloadEnrollProf{
		basePath: "/api/mdm/apple/enroll",
		token:    q.Get("profile_token"),
		diParam:  di,
	})

	// the url retrieves a valid EULA
	resp := s.DoRaw("GET", "/api/latest/fleet/setup_experience/eula/"+q.Get("eula_token"), nil, http.StatusOK)
	require.EqualValues(t, len(pdfBytes), resp.ContentLength)
	require.Equal(t, "application/pdf", resp.Header.Get("content-type"))
	require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, pdfBytes, respBytes)

	// IdP info stored is accurate for the account
	s.checkStoredIdPInfo(t, user1EnrollRef, "sso_user", "SSO User 1", "sso_user@example.com")

	enrollURL := ""
	scepURL := ""
	for _, p := range prof.PayloadContent {
		switch p.PayloadType {
		case "com.apple.security.scep":
			scepURL = p.PayloadContent.URL
		case "com.apple.mdm":
			enrollURL = p.ServerURL
		}
	}
	require.NotEmpty(t, enrollURL)
	require.NotEmpty(t, scepURL)

	// enroll the device using the provided profile
	// we're using localhost for SSO because that's how the local
	// SimpleSAML server is configured, and s.server.URL changes between
	// test runs.
	mdmDevice.EnrollInfo.MDMURL = strings.Replace(enrollURL, "https://localhost:8080", s.server.URL, 1)
	mdmDevice.EnrollInfo.SCEPURL = strings.Replace(scepURL, "https://localhost:8080", s.server.URL, 1)
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	// Enroll generated the TokenUpdate request to Fleet and enqueued the
	// Post-DEP enrollment job, it needs to be processed.
	s.awaitRunAppleMDMWorkerSchedule()

	// ask for commands and verify that we get AccountConfiguration
	var accCmd *mdm.Command
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "AccountConfiguration" {
			accCmd = cmd
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.NotNil(t, accCmd)
	require.NotNil(t, accCmd.Command)

	var fullAccCmd *micromdm.CommandPayload
	require.NoError(t, plist.Unmarshal(accCmd.Raw, &fullAccCmd))
	require.True(t, fullAccCmd.Command.AccountConfiguration.LockPrimaryAccountInfo) // nolint:nilaway // set by the previous plist.Unmarshal
	require.Equal(t, "SSO User 1", fullAccCmd.Command.AccountConfiguration.PrimaryAccountFullName)
	require.Equal(t, "sso_user", fullAccCmd.Command.AccountConfiguration.PrimaryAccountUserName)

	// report host details for the device
	var hostResp getHostResponse
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+mdmDevice.UUID, nil, http.StatusOK, &hostResp)
	assert.Len(t, hostResp.Host.EndUsers, 1)
	assert.Empty(t, hostResp.Host.EndUsers[0].IdpFullName)                         // full name is not recorded for mdm flow until scim sync
	assert.Equal(t, "sso_user@example.com", hostResp.Host.EndUsers[0].IdpUserName) // email is the same as the one used to enroll
	hostID := hostResp.Host.ID
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)
	assert.Len(t, hostResp.Host.EndUsers, 1)
	assert.Empty(t, hostResp.Host.EndUsers[0].IdpFullName)                         // full name is not recorded for mdm flow until scim sync
	assert.Equal(t, "sso_user@example.com", hostResp.Host.EndUsers[0].IdpUserName) // email is the same as the one used to enroll

	ac, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)

	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, ac, &ac.Features, osquery_utils.Integrations{}, nil)

	// simulate osquery reporting mdm information
	rows := []map[string]string{
		{
			"enrolled":           "true",
			"installed_from_dep": "true",
			"server_url":         "https://test.example.com?enrollment_reference=" + user1EnrollRef,
			"payload_identifier": apple_mdm.FleetPayloadIdentifier,
		},
	}
	err = detailQueries["mdm"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		&fleet.Host{ID: hostResp.Host.ID},
		s.ds,
		rows,
	)
	require.NoError(t, err)

	hostResp = getHostResponse{}
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+mdmDevice.UUID, nil, http.StatusOK, &hostResp)
	checkEndUser := func() {
		require.Len(t, hostResp.Host.EndUsers, 1)
		endUser := hostResp.Host.EndUsers[0]
		assert.Equal(t, "sso_user@example.com", endUser.IdpUserName)
		assert.Nil(t, endUser.IdpInfoUpdatedAt)
		assert.Empty(t, endUser.IdpID)
		assert.Empty(t, endUser.OtherEmails)
		assert.Empty(t, endUser.IdpFullName)
		assert.Empty(t, endUser.IdpGroups)
	}
	checkEndUser()
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)
	checkEndUser()

	// sumulate osquery reporting chrome extension information
	rows = []map[string]string{
		{"email": "g1@example.com"},
		{"email": "g2@example.com"},
	}
	err = detailQueries["google_chrome_profiles"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		&fleet.Host{ID: hostResp.Host.ID},
		s.ds,
		rows,
	)
	require.NoError(t, err)

	hostResp = getHostResponse{}
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+mdmDevice.UUID, nil, http.StatusOK, &hostResp)
	checkEndUser = func() {
		require.Len(t, hostResp.Host.EndUsers, 1)
		endUser := hostResp.Host.EndUsers[0]
		assert.Equal(t, "sso_user@example.com", endUser.IdpUserName)
		assert.Nil(t, endUser.IdpInfoUpdatedAt)
		assert.Empty(t, endUser.IdpID)
		assert.Empty(t, endUser.IdpFullName)
		assert.Empty(t, endUser.IdpGroups)
		require.Len(t, endUser.OtherEmails, 2)
		othersByEmail := make(map[string]string, 2)
		for _, otherEmail := range endUser.OtherEmails {
			othersByEmail[otherEmail.Email] = otherEmail.Source
		}
		source, ok := othersByEmail["g1@example.com"]
		require.True(t, ok)
		require.Equal(t, fleet.DeviceMappingGoogleChromeProfiles, source)
		source, ok = othersByEmail["g2@example.com"]
		require.True(t, ok)
		require.Equal(t, fleet.DeviceMappingGoogleChromeProfiles, source)
	}
	checkEndUser()
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)
	checkEndUser()

	// host device mapping includes the SSO user and the chrome extension users
	var dmResp listHostDeviceMappingResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/device_mapping", hostResp.Host.ID), nil, http.StatusOK, &dmResp)
	require.Len(t, dmResp.DeviceMapping, 3)
	sourceByEmail := make(map[string]string, 3)
	for _, dm := range dmResp.DeviceMapping {
		sourceByEmail[dm.Email] = dm.Source
	}
	source, ok := sourceByEmail["sso_user@example.com"]
	require.True(t, ok)
	require.Equal(t, fleet.DeviceMappingMDMIdpAccounts, source)
	source, ok = sourceByEmail["g1@example.com"]
	require.True(t, ok)
	require.Equal(t, "google_chrome_profiles", source)
	source, ok = sourceByEmail["g2@example.com"]
	require.True(t, ok)
	require.Equal(t, "google_chrome_profiles", source)

	// list hosts can filter on mdm idp email
	var hostsResp listHostsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts?query=%s&device_mapping=true", url.QueryEscape("sso_user@example.com")), nil, http.StatusOK, &hostsResp)
	require.Len(t, hostsResp.Hosts, 1)
	gotHost := hostsResp.Hosts[0]
	require.Equal(t, hostResp.Host.ID, gotHost.ID)
	require.NotNil(t, gotHost.DeviceMapping)
	var dm []fleet.HostDeviceMapping
	require.NoError(t, json.Unmarshal(*gotHost.DeviceMapping, &dm))
	require.Len(t, dm, 3)

	// reporting google chrome profiles only clears chrome profiles from device mapping
	err = detailQueries["google_chrome_profiles"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		&fleet.Host{ID: hostResp.Host.ID},
		s.ds,
		[]map[string]string{},
	)
	require.NoError(t, err)
	dmResp = listHostDeviceMappingResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/device_mapping", hostResp.Host.ID), nil, http.StatusOK, &dmResp)
	require.Len(t, dmResp.DeviceMapping, 1)
	require.Equal(t, "sso_user@example.com", dmResp.DeviceMapping[0].Email)
	require.Equal(t, fleet.DeviceMappingMDMIdpAccounts, dmResp.DeviceMapping[0].Source)
	hostsResp = listHostsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts?query=%s&device_mapping=true", url.QueryEscape("sso_user@example.com")), nil, http.StatusOK, &hostsResp)
	require.Len(t, hostsResp.Hosts, 1)
	gotHost = hostsResp.Hosts[0]
	require.Equal(t, hostResp.Host.ID, gotHost.ID)
	require.NotNil(t, gotHost.DeviceMapping)
	dm = []fleet.HostDeviceMapping{}
	require.NoError(t, json.Unmarshal(*gotHost.DeviceMapping, &dm))
	require.Len(t, dm, 1)
	require.Equal(t, "sso_user@example.com", dm[0].Email)
	require.Equal(t, fleet.DeviceMappingMDMIdpAccounts, dm[0].Source)

	// enrolling a different user works without problems
	res = s.LoginMDMSSOUser("sso_user2", "user123#", di)
	require.NotEmpty(t, res.Header.Get("Location"))
	require.Equal(t, http.StatusSeeOther, res.StatusCode)
	u, err = url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q = u.Query()
	user2EnrollRef := s.mdmIdPAccountUUID(t, "sso_user2@example.com")
	require.True(t, q.Has("eula_token"))
	require.True(t, q.Has("profile_token"))
	require.False(t, q.Has("enrollment_reference"))
	require.False(t, q.Has("error"))
	// the enrollment reference is different to the one used for the previous user
	require.NotEqual(t, user1EnrollRef, user2EnrollRef)
	// the url retrieves a valid profile
	s.downloadAndVerifyEnrollmentProfile(t, optsDownloadEnrollProf{
		basePath: "/api/mdm/apple/enroll",
		token:    q.Get("profile_token"),
		diParam:  di,
	})

	// the url retrieves a valid EULA
	resp = s.DoRaw("GET", "/api/latest/fleet/setup_experience/eula/"+q.Get("eula_token"), nil, http.StatusOK)
	require.EqualValues(t, len(pdfBytes), resp.ContentLength)
	require.Equal(t, "application/pdf", resp.Header.Get("content-type"))
	respBytes, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, pdfBytes, respBytes)

	// IdP info stored is accurate for the account
	s.checkStoredIdPInfo(t, user2EnrollRef, "sso_user2", "SSO User 2", "sso_user2@example.com")
	s.lastActivityOfTypeMatches(fleet.ActivityTypeBoundHostToIdPAccount{}.ActivityName(),
		fmt.Sprintf(`{"host_uuid": %q, "idp_email": "sso_user2@example.com", "replaced_idp_email": "sso_user@example.com"}`, mdmDevice.UUID), 0)

	// the automatic enrollment token is refused while the host's fleet requires
	// end user authentication, and the link is kept
	res = s.DoRawWithHeaders("GET", "/api/mdm/apple/enroll", nil, http.StatusUnauthorized, nil,
		"token", staticProf.Token, "deviceinfo", di)
	res.Body.Close()
	linked, err := s.ds.GetMDMIdPAccountByHostUUID(t.Context(), mdmDevice.UUID)
	require.NoError(t, err)
	require.NotNil(t, linked)
	require.Equal(t, "sso_user2@example.com", linked.Email)

	// without end user authentication, enrolling with the automatic enrollment
	// token (no account) removes the link
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {"macos_setup": {"enable_end_user_authentication": false}}
	}`), http.StatusOK, &appConfigResponse{})
	s.downloadAndVerifyEnrollmentProfile(t, optsDownloadEnrollProf{
		basePath: "/api/mdm/apple/enroll",
		token:    staticProf.Token,
		diParam:  di,
	})
	s.lastActivityOfTypeMatches(fleet.ActivityTypeUnboundHostFromIdPAccount{}.ActivityName(),
		fmt.Sprintf(`{"host_uuid": %q, "idp_email": "sso_user2@example.com"}`, mdmDevice.UUID), 0)
	unlinked, err := s.ds.GetMDMIdPAccountByHostUUID(t.Context(), mdmDevice.UUID)
	require.NoError(t, err)
	require.Nil(t, unlinked)
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {"macos_setup": {"enable_end_user_authentication": true}}
	}`), http.StatusOK, &appConfigResponse{})

	// changing the server URL also updates the remote DEP profile
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
                "server_settings": {"server_url": "https://example.com"}
	}`), http.StatusOK, &acResp)

	s.runWorker()
	require.Contains(t, lastSubmittedProfile.URL, "https://example.com/mdm/sso")
	require.Equal(t, "https://example.com/mdm/sso", lastSubmittedProfile.ConfigurationWebURL)

	// Hitting the callback with an invalid SAMLResponse returns to the UI with an error.
	rawSSOResp := `InvalidXML`
	samlResponse := base64.StdEncoding.EncodeToString([]byte(rawSSOResp))
	res = s.DoRawNoAuth("POST", "/api/v1/fleet/mdm/sso/callback?SAMLResponse="+url.QueryEscape(samlResponse), nil, http.StatusSeeOther)
	require.NotEmpty(t, res.Header.Get("Location"))
	u, err = url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q = u.Query()
	require.False(t, q.Has("eula_token"))
	require.False(t, q.Has("profile_token"))
	require.False(t, q.Has("enrollment_reference"))
	require.True(t, q.Has("error"))
	// No session cookie is sent, so the session lookup fails before the
	// SAMLResponse is ever parsed.
	require.Equal(t, "session_expired", q.Get("reason"))

	// hitting the callback with an invalid session id redirects the user to the UI
	rawSSOResp = `<?xml version="1.0" encoding="UTF-8"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" Destination="https://localhost:8080/api/v1/fleet/sso/callback" ID="_7822b394622740aa92878ca6c7d1a28c53e80ec5ef" InResponseTo="4982b430-73e1-4ad2-885a-4a775a91f820" IssueInstant="2017-04-27T15:03:16.747Z" Version="2.0">
</samlp:Response>`
	samlResponse = base64.StdEncoding.EncodeToString([]byte(rawSSOResp))
	res = s.DoRawNoAuth("POST", "/api/v1/fleet/mdm/sso/callback?SAMLResponse="+url.QueryEscape(samlResponse), nil, http.StatusSeeOther)
	require.NotEmpty(t, res.Header.Get("Location"))
	u, err = url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q = u.Query()
	require.False(t, q.Has("eula_token"))
	require.False(t, q.Has("profile_token"))
	require.False(t, q.Has("enrollment_reference"))
	require.True(t, q.Has("error"))
	// No session cookie was sent, so the session lookup is what failed. The UI
	// uses this to tell the end user their sign-in timed out.
	require.Equal(t, "session_expired", q.Get("reason"))

	// With a live session, a SAMLResponse that can't be verified is a different
	// failure. The user hasn't run out of time, so the timeout message would be
	// wrong -- this stays generic.
	prevCookieSecure := cookieSecure
	t.Cleanup(func() { cookieSecure = prevCookieSecure })
	cookieSecure = false
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := fleethttp.NewClient(fleethttp.WithFollowRedir(false), fleethttp.WithCookieJar(jar))

	var resIni initiateSSOResponse
	iniRes := s.doWithClient(client, "POST", "/api/v1/fleet/mdm/sso", []byte(`{}`), http.StatusOK, nil)
	require.NoError(t, json.NewDecoder(iniRes.Body).Decode(&resIni))
	require.NoError(t, resIni.Error())

	res = s.doWithClient(client, "POST", "/api/v1/fleet/mdm/sso/callback", nil, http.StatusSeeOther, nil,
		"SAMLResponse", base64.StdEncoding.EncodeToString([]byte(`InvalidXML`)))
	u, err = url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q = u.Query()
	require.True(t, q.Has("error"))
	require.False(t, q.Has("reason"))
}

func (s *integrationMDMTestSuite) TestDeviceSSO() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	newHostWithToken := func(suffix string) (*fleet.Host, string) {
		h, err := s.ds.NewHost(t.Context(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   new(t.Name() + "-osquery" + suffix),
			NodeKey:         new(t.Name() + "-nodekey" + suffix),
			UUID:            strings.ToUpper(uuid.NewString()),
			Hostname:        fmt.Sprintf("%sfoo%s.local", t.Name(), suffix),
			HardwareSerial:  mdmtest.RandSerialNumber(),
			Platform:        "darwin",
		})
		require.NoError(t, err)
		token := uuid.New().String()
		require.NoError(t, s.ds.SetOrUpdateDeviceAuthToken(t.Context(), h.ID, token))
		return h, token
	}

	fleetHost, deviceToken := newHostWithToken("-a")
	otherHost, otherDeviceToken := newHostWithToken("-b")

	res := s.DoRawNoAuth("POST", "/api/v1/fleet/device/"+deviceToken+"/sso", nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "is not enabled")

	for _, c := range res.Cookies() {
		require.NotEqual(t, cookieNameSSOSession, c.Name, "error response must not touch the handshake cookie")
	}

	s.setUpMDMSSO(t, false)
	defer s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"fleet_desktop": { "sso_enabled": false }
	}`), http.StatusOK, &appConfigResponse{})

	// enable fleet_desktop.sso_enabled
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"fleet_desktop": { "sso_enabled": true }
	}`), http.StatusOK, &acResp)
	require.True(t, acResp.FleetDesktop.SSOEnabled)

	client := s.newSSOTestClient()
	idpURL := s.InitiateDeviceSSO(client, deviceToken)

	// the handshake cookie has to land in the jar, or the callback below cannot
	// resolve the SSO session
	serverURL, err := url.Parse(s.server.URL)
	require.NoError(t, err)
	var gotHandshakeCookie bool
	for _, c := range client.Jar.Cookies(serverURL) {
		if c.Name == cookieNameSSOSession {
			gotHandshakeCookie = true
		}
	}
	require.True(t, gotHandshakeCookie, "expected the SSO handshake cookie to be set")

	forgedBody, err := json.Marshal(initiateMDMSSORequest{
		Initiator: fleet.SSOInitiatorFleetDesktop,
		HostUUID:  fleetHost.UUID,
	})
	require.NoError(t, err)
	s.DoRawNoAuth("POST", "/api/v1/fleet/mdm/sso", forgedBody, http.StatusBadRequest)

	// the gate is on, so the device page is refused until the end user signs in
	requireSSORequired(t, s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+deviceToken, nil, http.StatusUnauthorized), true)

	// complete *this* flow: sign in at the IdP and post the assertion back to the
	// existing MDM SSO callback, on the same client that holds the handshake cookie
	res = s.CompleteDeviceSSO(client, idpURL, "sso_user", "user123#")
	require.Equal(t, "https://localhost:8080/device/"+deviceToken, res.Header.Get("Location"))

	var gotSessionCookie *http.Cookie
	var handshakeCookieDeleted bool
	for _, c := range res.Cookies() {
		switch c.Name {
		case cookieNameDeviceSSOSession:
			gotSessionCookie = c
		case cookieNameSSOSession:
			handshakeCookieDeleted = c.MaxAge < 0
		}
	}
	require.NotNil(t, gotSessionCookie, "expected the device SSO session cookie to be set")
	require.NotEmpty(t, gotSessionCookie.Value)
	require.True(t, handshakeCookieDeleted, "expected the handshake cookie to be deleted")

	// the cookie must be unreadable from JS, scoped to the whole site so the
	// device page's API calls carry it, and live exactly as long as the session
	require.True(t, gotSessionCookie.HttpOnly)
	require.Equal(t, "/", gotSessionCookie.Path)
	require.Equal(t, http.SameSiteLaxMode, gotSessionCookie.SameSite)
	require.Equal(t, int(s.fleetCfg.Session.Duration.Seconds()), gotSessionCookie.MaxAge)

	deviceSSOSession := func(sessionID string) fleet.DeviceSSOSession {
		t.Helper()
		raw, err := s.keyValueStore.Get(t.Context(), "device_sso_session:"+sessionID)
		require.NoError(t, err)
		require.NotNil(t, raw, "expected a stored device SSO session")
		var sess fleet.DeviceSSOSession
		require.NoError(t, json.Unmarshal([]byte(*raw), &sess))
		return sess
	}

	// the minted session is bound to the host whose token initiated the flow,
	// which is what stops one device's cookie from unlocking another's page
	session := deviceSSOSession(gotSessionCookie.Value)
	require.Equal(t, fleetHost.ID, session.HostID)
	require.NotEqual(t, otherHost.ID, session.HostID)

	// the session also records which IdP identity signed in
	ssoUserAcct, err := s.ds.GetMDMIdPAccountByEmail(t.Context(), "sso_user@example.com")
	require.NoError(t, err)
	require.Equal(t, ssoUserAcct.UUID, session.IdPAccountUUID)

	// a session initiated from a different host's token binds to that host
	otherRes := s.LoginDeviceSSOUser("sso_user", "user123#", otherDeviceToken)
	var otherSessionCookie *http.Cookie
	for _, c := range otherRes.Cookies() {
		if c.Name == cookieNameDeviceSSOSession {
			otherSessionCookie = c
		}
	}
	require.NotNil(t, otherSessionCookie)
	require.NotEqual(t, gotSessionCookie.Value, otherSessionCookie.Value)
	require.Equal(t, otherHost.ID, deviceSSOSession(otherSessionCookie.Value).HostID)

	// the fleet_desktop branch never associates the IdP account with the host
	// -- that link belongs to enrollment flows, not a Fleet Desktop sign-in.
	var idpAssociationCount int
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &idpAssociationCount,
			`SELECT COUNT(*) FROM host_mdm_idp_accounts WHERE host_uuid IN (?, ?)`, fleetHost.UUID, otherHost.UUID)
	})
	assert.Zero(t, idpAssociationCount)

	// A failed callback sends the end user back to the device page with a
	// reason, not to the admin SSO callback route, and mints nothing.
	requireNoSessionMinted := func(res *http.Response, deviceToken, wantReason, msg string) {
		t.Helper()
		require.Equal(t, "https://localhost:8080/device/"+deviceToken+"?sso_error="+wantReason,
			res.Header.Get("Location"), msg)
		for _, c := range res.Cookies() {
			require.NotEqualf(t, cookieNameDeviceSSOSession, c.Name, "no session cookie expected: %s", msg)
		}
	}

	// if the host is gone by the time the IdP calls back, the callback errors
	// out instead of minting a dangling session
	deletedClient := s.newSSOTestClient()
	deletedIdPURL := s.InitiateDeviceSSO(deletedClient, otherDeviceToken)
	require.NoError(t, s.ds.DeleteHost(t.Context(), otherHost.ID))
	requireNoSessionMinted(
		s.CompleteDeviceSSO(deletedClient, deletedIdPURL, "sso_user", "user123#"),
		otherDeviceToken, "server_error", "host no longer exists",
	)

	// disabling the feature between initiation and callback also stops a
	// session from being minted -- otherwise it would outlive the change by its
	// whole TTL
	_, thirdToken := newHostWithToken("-c")
	disabledClient := s.newSSOTestClient()
	disabledIdPURL := s.InitiateDeviceSSO(disabledClient, thirdToken)
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"fleet_desktop": { "sso_enabled": false }
	}`), http.StatusOK, &appConfigResponse{})
	requireNoSessionMinted(
		s.CompleteDeviceSSO(disabledClient, disabledIdPURL, "sso_user", "user123#"),
		thirdToken, "sso_disabled", "feature disabled mid-flow",
	)

	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"fleet_desktop": { "sso_enabled": true }
	}`), http.StatusOK, &appConfigResponse{})

	_, fourthToken := newHostWithToken("-d")
	expiredClient := s.newSSOTestClient()
	expiredIdPURL := s.InitiateDeviceSSO(expiredClient, fourthToken)
	s.ExpireSSOHandshake(expiredClient)
	expiredRes := s.CompleteDeviceSSO(expiredClient, expiredIdPURL, "sso_user", "user123#")
	require.Equal(t, "/device/sso-error?reason=session_expired", expiredRes.Header.Get("Location"))
	for _, c := range expiredRes.Cookies() {
		require.NotEqual(t, cookieNameDeviceSSOSession, c.Name)
	}

	_, fifthToken := newHostWithToken("-e")
	forgedRelayClient := s.newSSOTestClient()
	forgedRelayIdPURL := s.InitiateDeviceSSO(forgedRelayClient, fifthToken)
	s.ExpireSSOHandshake(forgedRelayClient)
	forgedRelayRes := s.CompleteDeviceSSOWithRelayState(forgedRelayClient, forgedRelayIdPURL,
		"sso_user", "user123#", "https://evil.example.com")
	require.Equal(t, "/mdm/sso/callback?error=true&reason=session_expired", forgedRelayRes.Header.Get("Location"))
}

// TestMDMSSOSetupExperienceHostBinding covers which IdP account the Orbit Setup
// Experience SSO flow (Linux/Windows) leaves a host UUID bound to.
//
// The flow is unauthenticated and the device's host UUID travels in the SSO
// request data, so the binding rules only hold as a pair: while Fleet is still
// waiting on that device's end user a sign-in may create or replace the binding,
// and once Fleet is no longer waiting a sign-in may fill in a missing binding but
// never take one over. Host UUIDs are not secret, so without the first
// precondition any IdP user could claim any host and have the victim's fleetd
// enroll silently attributed to them.
func (s *integrationMDMTestSuite) TestMDMSSOSetupExperienceHostBinding() {
	t := s.T()
	// Skip up here rather than letting newSSOTestClient do it: it skips the
	// suite's T, which Go rejects from inside a subtest, and setUpMDMSSO below
	// needs the IdP's metadata anyway.
	if _, ok := os.LookupEnv("SAML_IDP_TEST"); !ok {
		t.Skip("SSO tests are disabled")
	}
	s.setSkipWorkerJobs(t)
	ctx := context.Background()

	s.setUpMDMSSO(t, false)

	t.Run("a host uuid fleet is not waiting on cannot start the flow", func(t *testing.T) {
		initiate := func(hostUUID string) {
			t.Helper()
			body, err := json.Marshal(initiateMDMSSORequest{
				Initiator: fleet.SSOInitiatorOrbitSetupExperience,
				HostUUID:  hostUUID,
			})
			require.NoError(t, err)
			s.DoRawNoAuth("POST", "/api/v1/fleet/mdm/sso", body, http.StatusUnauthorized)
		}

		victimUUID := uuid.NewString()
		initiate(victimUUID)
		initiate("")

		acct, err := s.ds.GetMDMIdPAccountByHostUUID(ctx, victimUUID)
		require.NoError(t, err)
		require.Nil(t, acct)
	})

	t.Run("a sign-in binds the host, and a later one re-points it", func(t *testing.T) {
		hostUUID := uuid.NewString()

		res := s.LoginMDMSSOUserSetupExperience("sso_user", "user123#", hostUUID)
		require.Equal(t, http.StatusSeeOther, res.StatusCode)
		loc, err := url.Parse(res.Header.Get("Location"))
		require.NoError(t, err)
		require.False(t, loc.Query().Has("error"), "first setup-experience SSO login should succeed")

		acct, err := s.ds.GetMDMIdPAccountByHostUUID(ctx, hostUUID)
		require.NoError(t, err)
		require.NotNil(t, acct)
		require.Equal(t, "sso_user@example.com", acct.Email)
		// The account UUID must not be the host UUID: keying the account row on
		// the host UUID is what used to break the re-point below.
		require.NotEqual(t, hostUUID, acct.UUID)
		firstAcctUUID := acct.UUID

		s.lastActivityOfTypeMatches(
			fleet.ActivityTypeBoundHostToIdPAccount{}.ActivityName(),
			fmt.Sprintf(`{"host_uuid": %q, "idp_email": "sso_user@example.com"}`, hostUUID),
			0,
		)

		// The SAME host UUID signs in as a DIFFERENT IdP user, which is what a
		// device re-enrolling after a wipe does, and what an end user correcting
		// a wrong-account sign-in does. mdmSSOHandleCallbackAuth once keyed the
		// mdm_idp_accounts row on the host UUID, so this collided on the primary
		// key: ON DUPLICATE KEY UPDATE left the old email in place and the
		// read-back by email returned not-found, surfacing as "retrieving new
		// account data from IdP". The account UUID is now DB-generated, matching
		// the Apple flow, so the second login find-or-creates by email instead.
		// TestSSO re-enrolls a second user too, but through the Apple initiator
		// with no host UUID, where the UUID was always generated -- which is
		// exactly why it never caught that.
		res = s.LoginMDMSSOUserSetupExperience("sso_user2", "user123#", hostUUID)
		require.Equal(t, http.StatusSeeOther, res.StatusCode)
		loc, err = url.Parse(res.Header.Get("Location"))
		require.NoError(t, err)
		require.False(t, loc.Query().Has("error"),
			"re-enrollment SSO with a different IdP email must not fail")

		acct, err = s.ds.GetMDMIdPAccountByHostUUID(ctx, hostUUID)
		require.NoError(t, err)
		require.NotNil(t, acct)
		require.Equal(t, "sso_user2@example.com", acct.Email)
		require.NotEqual(t, firstAcctUUID, acct.UUID, "host should be re-pointed to the new account")

		s.lastActivityOfTypeMatches(
			fleet.ActivityTypeBoundHostToIdPAccount{}.ActivityName(),
			fmt.Sprintf(`{"host_uuid": %q, "idp_email": "sso_user2@example.com", "replaced_idp_email": "sso_user@example.com"}`, hostUUID),
			0,
		)
	})

	t.Run("a sign-in that outlived the prompt cannot take over an existing binding", func(t *testing.T) {
		hostUUID := uuid.NewString()

		// The device enrolls with one account...
		res := s.LoginMDMSSOUserSetupExperience("sso_user", "user123#", hostUUID)
		require.Equal(t, http.StatusSeeOther, res.StatusCode)

		// ...and a second sign-in completes after fleetd has enrolled, which is
		// what clears the prompt.
		res = s.LoginMDMSSOUserSetupExperience("sso_user2", "user123#", hostUUID, func() {
			s.ClearEndUserAuthPrompt(hostUUID)
		})
		require.Equal(t, http.StatusSeeOther, res.StatusCode)

		acct, err := s.ds.GetMDMIdPAccountByHostUUID(ctx, hostUUID)
		require.NoError(t, err)
		require.NotNil(t, acct)
		require.Equal(t, "sso_user@example.com", acct.Email,
			"the binding the enrollment settled on must survive a late sign-in")

		s.lastActivityOfTypeMatches(
			fleet.ActivityTypeRefusedHostIdPAccountChange{}.ActivityName(),
			fmt.Sprintf(`{"host_uuid": %q, "idp_email": "sso_user2@example.com", "existing_idp_email": "sso_user@example.com"}`, hostUUID),
			0,
		)
	})
}

func (s *integrationMDMTestSuite) mdmIdPAccountUUID(t *testing.T, email string) string {
	acc, err := s.ds.GetMDMIdPAccountByEmail(t.Context(), email)
	require.NoError(t, err)
	return acc.UUID
}

func (s *integrationMDMTestSuite) checkStoredIdPInfo(t *testing.T, uuid, username, fullname, email string) {
	acc, err := s.ds.GetMDMIdPAccountByUUID(context.Background(), uuid)
	require.NoError(t, err)
	require.Equal(t, username, acc.Username)
	require.Equal(t, fullname, acc.Fullname)
	require.Equal(t, email, acc.Email)
}

func (s *integrationMDMTestSuite) TestSSOWithSCIM() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	// machine info blobs in this test are signed with a throwaway cert, not an
	// Apple device identity
	apple_mdm.SetMachineInfoVerificationForTest(t, false)
	ctx := context.Background()

	lastSubmittedProfile := &godep.Profile{}
	mdmDevice, _ := s.setUpEndUserAuthentication(t, lastSubmittedProfile, false)
	di, err := mdmtest.EncodeDeviceInfo(fleet.MDMAppleMachineInfo{
		Serial: mdmDevice.SerialNumber,
		UDID:   mdmDevice.UUID,
	})
	require.NoError(t, err)

	res := s.LoginMDMSSOUser("sso_user_no_displayname", "user123#", di)
	require.NotEmpty(t, res.Header.Get("Location"))
	require.Equal(t, http.StatusSeeOther, res.StatusCode)

	u, err := url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q := u.Query()
	user1EnrollRef := s.mdmIdPAccountUUID(t, "sso_user_no_displayname@example.com")
	// the url retrieves a valid profile
	prof := s.downloadAndVerifyEnrollmentProfile(t, optsDownloadEnrollProf{
		basePath: "/api/mdm/apple/enroll",
		token:    q.Get("profile_token"),
		diParam:  di,
	})

	// IdP info stored is accurate for the account
	s.checkStoredIdPInfo(t, user1EnrollRef, "sso_user_no_displayname", "", "sso_user_no_displayname@example.com")

	enrollURL := ""
	scepURL := ""
	for _, p := range prof.PayloadContent {
		switch p.PayloadType {
		case "com.apple.security.scep":
			scepURL = p.PayloadContent.URL
		case "com.apple.mdm":
			enrollURL = p.ServerURL
		}
	}
	require.NotEmpty(t, enrollURL)
	require.NotEmpty(t, scepURL)

	// enroll the device using the provided profile
	// we're using localhost for SSO because that's how the local
	// SimpleSAML server is configured, and s.server.URL changes between
	// test runs.
	mdmDevice.EnrollInfo.MDMURL = strings.Replace(enrollURL, "https://localhost:8080", s.server.URL, 1)
	mdmDevice.EnrollInfo.SCEPURL = strings.Replace(scepURL, "https://localhost:8080", s.server.URL, 1)
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	// Add matching SCIM user with a display name (givenName + familyName)
	createUserPayload := map[string]any{
		"schemas":    []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName":   "some_other_username",
		"externalID": "external_id",
		"name": map[string]any{
			"givenName":  "SCIM",
			"familyName": "User",
		},
		"emails": []map[string]any{
			{
				"value":   "sso_user_no_displayname@example.com",
				"type":    "work",
				"primary": true,
			},
		},
	}
	displayName := "SCIM User"
	var createResp map[string]any
	s.DoJSON("POST", "/api/latest/fleet/scim/Users", createUserPayload, http.StatusCreated, &createResp)
	// Verify the created user
	assert.Equal(t, "some_other_username", createResp["userName"])
	scimUserID, ok := createResp["id"]
	require.True(t, ok)

	// Enroll generated the TokenUpdate request to Fleet and enqueued the
	// Post-DEP enrollment job, it needs to be processed.
	s.awaitRunAppleMDMWorkerSchedule()

	// Simulate TTL expiration
	require.NoError(t, s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+mdmDevice.UUID))

	// ask for commands and verify that we get AccountConfiguration
	var accCmd *mdm.Command
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "AccountConfiguration" {
			accCmd = cmd
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.NotNil(t, accCmd)
	require.NotNil(t, accCmd.Command)

	var fullAccCmd *micromdm.CommandPayload
	require.NoError(t, plist.Unmarshal(accCmd.Raw, &fullAccCmd))
	// We setup the test with Lock End User Account info disabled, so this should be false
	require.NotNil(t, fullAccCmd)
	require.NotNil(t, fullAccCmd.Command)
	require.NotNil(t, fullAccCmd.Command.AccountConfiguration)
	assert.False(t, fullAccCmd.Command.AccountConfiguration.LockPrimaryAccountInfo)
	assert.Equal(t, displayName, fullAccCmd.Command.AccountConfiguration.PrimaryAccountFullName)
	assert.Equal(t, "sso_user_no_displayname", fullAccCmd.Command.AccountConfiguration.PrimaryAccountUserName)

	// Report host details for the device. At this point, we should have fully linked the SCIM data with the user.
	var hostResp getHostResponse
	checkEndUser := func() {
		require.Len(t, hostResp.Host.EndUsers, 1)
		endUser := hostResp.Host.EndUsers[0]
		assert.Equal(t, "some_other_username", endUser.IdpUserName)
		assert.NotNil(t, endUser.IdpInfoUpdatedAt)
		assert.Equal(t, "external_id", endUser.IdpID)
		assert.Equal(t, displayName, endUser.IdpFullName)
		assert.Empty(t, endUser.IdpGroups)
		assert.Empty(t, endUser.OtherEmails)
	}
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+mdmDevice.UUID, nil, http.StatusOK, &hostResp)
	assert.Len(t, hostResp.Host.EndUsers, 1)
	checkEndUser()

	hostID := hostResp.Host.ID
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)
	assert.Len(t, hostResp.Host.EndUsers, 1)
	checkEndUser()

	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)
	checkEndUser()

	ac, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)

	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, ac, &ac.Features, osquery_utils.Integrations{}, nil)

	// simulate osquery reporting mdm information, doesn't change anything
	rows := []map[string]string{
		{
			"enrolled":           "true",
			"installed_from_dep": "true",
			"server_url":         "https://test.example.com?enrollment_reference=" + user1EnrollRef,
			"payload_identifier": apple_mdm.FleetPayloadIdentifier,
		},
	}
	err = detailQueries["mdm"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		&fleet.Host{ID: hostResp.Host.ID},
		s.ds,
		rows,
	)
	require.NoError(t, err)

	hostResp = getHostResponse{}
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+mdmDevice.UUID, nil, http.StatusOK, &hostResp)
	checkEndUser()
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)
	checkEndUser()

	// sumulate osquery reporting chrome extension information
	rows = []map[string]string{
		{"email": "g1@example.com"},
		{"email": "g2@example.com"},
	}
	err = detailQueries["google_chrome_profiles"].DirectIngestFunc(
		context.Background(),
		slog.New(slog.DiscardHandler),
		&fleet.Host{ID: hostResp.Host.ID},
		s.ds,
		rows,
	)
	require.NoError(t, err)

	// mark any pending command as acknowledged at this point
	s.awaitTriggerProfileSchedule(t)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// add two profiles to no team, one that uses the IdP username and one that uses groups
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: mobileconfigForTest("N1", "I1", string(fleet.FleetVarHostEndUserIDPUsername))},
		{Name: "N2", Contents: mobileconfigForTest("N2", "I2", string(fleet.FleetVarHostEndUserIDPGroups))},
	}}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	// one will be failed (no IdP groups), and the other will succeed and be sent to the host
	// ask for commands and verify that we get InstallProfile for the username N1 profile
	var installProfCmd *mdm.Command
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "InstallProfile" {
			installProfCmd = cmd
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.NotNil(t, installProfCmd)
	var decodedCmd struct {
		Command struct {
			Payload []byte
		}
	}
	err = plist.Unmarshal(installProfCmd.Raw, &decodedCmd)
	require.NoError(t, err)
	// variable was replaced with the IdP username
	require.Contains(t, string(decodedCmd.Command.Payload), "some_other_username")

	hostResp = getHostResponse{}
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+mdmDevice.UUID, nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host.MDM.Profiles)
	var expectedCount int
	for _, prof := range *hostResp.Host.MDM.Profiles {
		switch prof.Name {
		case "N1": // profile with username
			expectedCount++
			require.NotNil(t, prof.Status)
			require.EqualValues(t, fleet.MDMDeliveryVerifying, *prof.Status)
		case "N2": // profile with group
			expectedCount++
			require.NotNil(t, prof.Status)
			require.EqualValues(t, fleet.MDMDeliveryFailed, *prof.Status)
		}
	}
	require.Equal(t, 2, expectedCount)

	// Also add a group to the SCIM user
	createGroup1Payload := map[string]any{
		"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
		"displayName": "Test Group 1",
		"members": []map[string]any{
			{
				"value": scimUserID,
			},
		},
	}
	var createGroup1Resp map[string]any
	s.DoJSON("POST", "/api/latest/fleet/scim/Groups", createGroup1Payload, http.StatusCreated, &createGroup1Resp)

	// this marked profile N2 as pending for the host as the groups changed
	hostResp = getHostResponse{}
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+mdmDevice.UUID, nil, http.StatusOK, &hostResp)
	require.NotNil(t, hostResp.Host.MDM.Profiles)
	expectedCount = 0
	for _, prof := range *hostResp.Host.MDM.Profiles {
		switch prof.Name {
		case "N1": // profile with username
			expectedCount++
			require.NotNil(t, prof.Status)
			require.EqualValues(t, fleet.MDMDeliveryVerifying, *prof.Status)
		case "N2": // profile with group
			expectedCount++
			require.NotNil(t, prof.Status)
			require.EqualValues(t, fleet.MDMDeliveryPending, *prof.Status)
		}
	}
	require.Equal(t, 2, expectedCount)

	// reconcile profiles
	s.awaitTriggerProfileSchedule(t)

	// ask for commands and verify that we get InstallProfile for the group N2 profile
	installProfCmd = nil
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "InstallProfile" {
			installProfCmd = cmd
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.NotNil(t, installProfCmd)
	err = plist.Unmarshal(installProfCmd.Raw, &decodedCmd)
	require.NoError(t, err)
	// variable was replaced with the IdP group
	require.Contains(t, string(decodedCmd.Command.Payload), "Test Group 1")

	hostResp = getHostResponse{}
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+mdmDevice.UUID, nil, http.StatusOK, &hostResp)

	// check that both profiles are now delivered
	require.NotNil(t, hostResp.Host.MDM.Profiles)
	expectedCount = 0
	for _, prof := range *hostResp.Host.MDM.Profiles {
		switch prof.Name {
		case "N1": // profile with username
			expectedCount++
			require.NotNil(t, prof.Status)
			require.EqualValues(t, fleet.MDMDeliveryVerifying, *prof.Status)
		case "N2": // profile with group
			expectedCount++
			require.NotNil(t, prof.Status)
			require.EqualValues(t, fleet.MDMDeliveryVerifying, *prof.Status)
		}
	}
	require.Equal(t, 2, expectedCount)

	checkEndUser = func() {
		require.Len(t, hostResp.Host.EndUsers, 1)
		endUser := hostResp.Host.EndUsers[0]
		assert.Equal(t, "some_other_username", endUser.IdpUserName) // This is a username, not an email.
		assert.NotNil(t, endUser.IdpInfoUpdatedAt)
		assert.Equal(t, "external_id", endUser.IdpID)
		assert.Equal(t, displayName, endUser.IdpFullName)
		require.Equal(t, []string{"Test Group 1"}, endUser.IdpGroups)
		require.Len(t, endUser.OtherEmails, 2)
		othersByEmail := make(map[string]string, 2)
		for _, otherEmail := range endUser.OtherEmails {
			othersByEmail[otherEmail.Email] = otherEmail.Source
		}
		source, ok := othersByEmail["g1@example.com"]
		require.True(t, ok)
		require.Equal(t, fleet.DeviceMappingGoogleChromeProfiles, source)
		source, ok = othersByEmail["g2@example.com"]
		require.True(t, ok)
		require.Equal(t, fleet.DeviceMappingGoogleChromeProfiles, source)
	}
	checkEndUser()
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)
	checkEndUser()

	// renew the mdm enrollment profile with a user that isn't in SCIM
	res = s.LoginMDMSSOUser("sso_user2", "user123#", di)
	require.NotEmpty(t, res.Header.Get("Location"))
	require.Equal(t, http.StatusSeeOther, res.StatusCode)
	u, err = url.Parse(res.Header.Get("Location"))
	require.NoError(t, err)
	q = u.Query()
	user2EnrollRef := s.mdmIdPAccountUUID(t, "sso_user2@example.com")
	require.True(t, q.Has("profile_token"))
	require.False(t, q.Has("enrollment_reference"))
	require.False(t, q.Has("error"))
	// the enrollment reference is not same as the one used for the previous user
	require.NotEqual(t, user1EnrollRef, user2EnrollRef)
	// the url retrieves a valid profile
	prof = s.downloadAndVerifyEnrollmentProfile(t, optsDownloadEnrollProf{
		basePath: "/api/mdm/apple/enroll",
		token:    q.Get("profile_token"),
		diParam:  di,
	})

	// IdP info stored is accurate for the account
	s.checkStoredIdPInfo(t, user2EnrollRef, "sso_user2", "SSO User 2", "sso_user2@example.com")

	enrollURL = ""
	scepURL = ""
	for _, p := range prof.PayloadContent {
		switch p.PayloadType {
		case "com.apple.security.scep":
			scepURL = p.PayloadContent.URL
		case "com.apple.mdm":
			enrollURL = p.ServerURL
		}
	}
	require.NotEmpty(t, enrollURL)
	require.NotEmpty(t, scepURL)

	// enroll the device using the provided profile
	mdmDevice.EnrollInfo.MDMURL = strings.Replace(enrollURL, "https://localhost:8080", s.server.URL, 1)
	mdmDevice.EnrollInfo.SCEPURL = strings.Replace(scepURL, "https://localhost:8080", s.server.URL, 1)
	err = mdmDevice.Enroll()
	require.NoError(t, err)

	hostResp = getHostResponse{}
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+mdmDevice.UUID, nil, http.StatusOK, &hostResp)
	checkEndUser = func() {
		require.Len(t, hostResp.Host.EndUsers, 1)
		endUser := hostResp.Host.EndUsers[0]
		assert.Equal(t, "sso_user2@example.com", endUser.IdpUserName)
		assert.Empty(t, endUser.IdpInfoUpdatedAt)
		assert.Empty(t, endUser.IdpID)
		assert.Empty(t, endUser.IdpFullName)
		assert.Empty(t, endUser.IdpGroups)
		assert.Len(t, endUser.OtherEmails, 2) // prior google chrome profiles are still associated
	}
	checkEndUser()
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)
	checkEndUser()

	// TODO: add tests for other scenarios:
	// - re-enroll via SSO flow with a different SCIM user (IdP user should be updated to include all SCIM info)
	// - re-enroll via non-SSO flow (IdP user should be entirely empty)
	// - multiple SCIM integrations with overlapping emails (how would that even work?)
	// - SSO flow configured to use IdP different from SCIM but with overlapping emails (how would that even work?)
}

// If you call this function see note on setUpMDMSSO() about server URL changes
func (s *integrationMDMTestSuite) setUpEndUserAuthentication(t *testing.T, lastSubmittedProfile *godep.Profile, lockEndUserInfo bool) (*mdmtest.TestAppleMDMClient,
	fleet.SSOProviderSettings,
) {
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
	}, "MacBookPro16,1")
	s.enableABM(t.Name())
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/profile":
			*lastSubmittedProfile = godep.Profile{}
			rawProfile, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			err = json.Unmarshal(rawProfile, lastSubmittedProfile)
			assert.NoError(t, err)
			encoder := json.NewEncoder(w)
			err = encoder.Encode(godep.ProfileResponse{ProfileUUID: "abc"})
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
		case "/server/devices", "/devices/sync":
			// This endpoint  is used to get an initial list of
			// devices, return a single device
			encoder := json.NewEncoder(w)
			err := encoder.Encode(godep.DeviceResponse{
				Devices: []godep.Device{
					{
						SerialNumber: mdmDevice.SerialNumber,
						Model:        mdmDevice.Model,
						OS:           "osx",
						OpType:       "added",
					},
				},
			})
			assert.NoError(t, err)
		}
	}))

	// sync the list of ABM devices
	s.runDEPSchedule()

	acResp := s.setUpMDMSSO(t, lockEndUserInfo)

	// trigger the worker to process the job and wait for result before continuing.
	s.runWorker()

	// check that the last submitted DEP profile has been updated accordingly
	require.Contains(t, lastSubmittedProfile.URL, acResp.ServerSettings.ServerURL+"/mdm/sso")
	require.Equal(t, acResp.ServerSettings.ServerURL+"/mdm/sso", lastSubmittedProfile.ConfigurationWebURL)
	return mdmDevice, acResp.MDM.EndUserAuthentication.SSOProviderSettings
}

// checkOneTimeDEPEnrollmentToken checks that usedToken, a one-time token
// already redeemed by the device that presented deviceInfo, can't be redeemed
// again, and that fresh one-time tokens are refused for the wrong device or
// after they expire.
func (s *integrationMDMTestSuite) checkOneTimeDEPEnrollmentToken(t *testing.T, deviceInfo, usedToken string) {
	requireRefused := func(token, di string) {
		t.Helper()
		res := s.DoRawNoAuth("GET", "/api/mdm/apple/enroll", nil, http.StatusUnauthorized, "token", token, "deviceinfo", di)
		require.Contains(t, extractServerErrorText(res.Body), "Authentication failed")
	}
	newToken := func() string {
		t.Helper()
		res := s.LoginMDMSSOUser("sso_user", "user123#", deviceInfo)
		require.Equal(t, http.StatusSeeOther, res.StatusCode)
		u, err := url.Parse(res.Header.Get("Location"))
		require.NoError(t, err)
		return u.Query().Get("profile_token")
	}
	parsed, _, err := apple_mdm.ParseDeviceinfo(deviceInfo)
	require.NoError(t, err)
	otherDeviceInfo, err := mdmtest.EncodeDeviceInfo(fleet.MDMAppleMachineInfo{
		Serial: "OTHER" + parsed.Serial,
		UDID:   "other-" + parsed.UDID,
	})
	require.NoError(t, err)

	// reused
	requireRefused(usedToken, deviceInfo)

	// redeemed by another device, which uses it up for the right device too
	token := newToken()
	requireRefused(token, otherDeviceInfo)
	requireRefused(token, deviceInfo)

	// expired
	token = newToken()
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(t.Context(), `UPDATE mdm_apple_dep_enrollment_challenges SET expires_at = NOW(6) - INTERVAL 1 SECOND WHERE challenge = ?`, token)
		return err
	})
	requireRefused(token, deviceInfo)

	// a fresh one-time token still works
	token = newToken()
	s.downloadAndVerifyEnrollmentProfile(t, optsDownloadEnrollProf{token: token, diParam: deviceInfo})

	// initiating the flow requires the device's deviceinfo
	s.DoRawNoAuth("POST", "/api/v1/fleet/mdm/sso", []byte(`{"initiator": "mdm_sso"}`), http.StatusBadRequest)
}

func (s *integrationMDMTestSuite) TestReenrollKeepsManuallyMappedIdPUser() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	const userName = "manual.mapped@example.com"
	var createResp map[string]any
	s.DoJSON("POST", "/api/latest/fleet/scim/Users", map[string]any{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": userName,
		"name":     map[string]any{"givenName": "Manual", "familyName": "Mapped"},
		"emails":   []map[string]any{{"value": userName, "type": "work", "primary": true}},
		"active":   true,
	}, http.StatusCreated, &createResp)

	// Manual enrollment, no end-user authentication: the host has no IdP account.
	mdmDevice := mdmtest.NewTestMDMClientAppleDirect(mdmtest.AppleEnrollInfo{
		SCEPChallenge: s.scepChallenge,
		SCEPURL:       s.server.URL + apple_mdm.SCEPPath,
		MDMURL:        s.server.URL + apple_mdm.MDMPath,
	}, "MacBookPro16,1")
	require.NoError(t, mdmDevice.Enroll())

	var hostResp getHostResponse
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+mdmDevice.UUID, nil, http.StatusOK, &hostResp)
	hostID := hostResp.Host.ID

	var putResp putHostDeviceMappingResponse
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hostID),
		putHostDeviceMappingRequest{Email: userName, Source: fleet.DeviceMappingIDP}, http.StatusOK, &putResp)

	checkEndUser := func() {
		hostResp = getHostResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hostID), nil, http.StatusOK, &hostResp)
		require.Len(t, hostResp.Host.EndUsers, 1)
		assert.Equal(t, userName, hostResp.Host.EndUsers[0].IdpUserName)
		assert.Equal(t, "Manual Mapped", hostResp.Host.EndUsers[0].IdpFullName)
	}
	checkEndUser()

	// Authenticate + TokenUpdate again, as a SCEP renewal or `profiles renew -type enrollment` does.
	require.NoError(t, mdmDevice.Reenroll())
	checkEndUser()
}
