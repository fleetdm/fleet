package service

// Windows MDM tests for the MDM suite.
//
// Belongs here: enabling Windows MDM, the MS-MDE2 enrollment protocol (discovery, policy, security
// token, terms of use), automatic, Autopilot, Azure/Entra-initiated, and settings-app enrollment,
// re-enrollment and rekeying, unenrollment, enrollment-to-host mapping, and Windows MDM migration.
//
// Does not belong here: BitLocker (integration_mdm_disk_encryption_test.go), Windows MDM command
// results (integration_mdm_commands_queue_test.go), Windows profiles
// (integration_mdm_profiles_test.go), and user-scoped Windows profiles
// (integration_mdm_windows_userscope_test.go).

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/bindata"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Simulates a host being orbit enrolled first then an MDM enrollment coming via the settings app
func (s *integrationMDMTestSuite) createWindowsHostThenEnrollMDMViaSettingsApp(fleetServerURL, email, tenantID string) (*fleet.Host, *mdmtest.TestWindowsMDMClient) {
	host := createOrbitEnrolledHost(s.T(), "windows", uuid.NewString(), s.ds)
	mdmDevice := s.enrollWindowsMDMViaSettingsApp(fleetServerURL, email, tenantID)
	return host, mdmDevice
}

// Simulates a host fetching queries then reporting the queries required for an Azure-initiated Windows host to
// link up to its MDM enrollment.
func (s *integrationMDMTestSuite) simulateMDMWindowsQueries(t *testing.T, host *fleet.Host, mdmDevice *mdmtest.TestWindowsMDMClient) {
	s.lq.On("QueriesForHost", host.ID).Return(map[string]string{fmt.Sprintf("%d", host.ID): "select 1 from osquery;"}, nil)
	req := getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	var dqResp getDistributedQueriesResponse

	// Ensure we can read distributed queries for the host.
	err := s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)

	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_mdm_device_id_windows")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_mdm_windows")

	distributedReq := SubmitDistributedQueryResultsRequest{
		NodeKey: *host.NodeKey,
		Results: map[string][]map[string]string{
			"fleet_detail_query_mdm_windows": {
				{"aad_resource_id": s.server.URL, "discovery_service_url": s.server.URL + "/api/mdm/microsoft/discovery", "provider_id": "Fleet", "installation_type": "Client"},
			},
			"fleet_detail_query_mdm_device_id_windows": {
				{"name": "DeviceClientId", "data": mdmDevice.DeviceID},
			},
		},
		Statuses: map[string]fleet.OsqueryStatus{
			"fleet_detail_query_mdm_device_id_windows": 0,
			"fleet_detail_query_mdm_windows":           0,
		},
	}
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)
}

func (s *integrationMDMTestSuite) TestAppConfigWindowsMDM() {
	ctx := context.Background()
	t := s.T()

	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.WindowsEnabledAndConfigured = false
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.WindowsEnabledAndConfigured)

	// create a couple teams
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "1"})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "2"})
	require.NoError(t, err)

	// enable Windows MDM
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_enabled_and_configured": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	assert.False(t, acResp.MDM.WindowsMigrationEnabled)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledWindowsMDM{}.ActivityName(), `{}`, 0)

	// create some hosts - a Windows workstation in each team and no-team,
	// Windows server in no team, Windows workstation enrolled in a 3rd-party in
	// team 2, Windows workstation already enrolled in Fleet in no team, and a
	// macOS host in no team.
	metadataHosts := []struct {
		os            string
		suffix        string
		isServer      bool
		teamID        *uint
		enrolledName  string
		shouldEnroll  bool
		shouldMigrate bool
	}{
		{"windows", "win-no-team", false, nil, "", true, false},
		{"windows", "win-team-1", false, &tm1.ID, "", true, false},
		{"windows", "win-team-2", false, &tm2.ID, "", true, false},
		{"windows", "win-server", true, nil, "", false, false},                                      // is a server
		{"windows", "win-third-party", false, &tm2.ID, fleet.WellKnownMDMSimpleMDM, false, true},    // is enrolled in 3rd-party
		{"windows", "win-fleet", false, nil, fleet.WellKnownMDMFleet, false, false},                 // is already Fleet-enrolled
		{"darwin", "macos-no-team", false, nil, "", false, false},                                   // is not Windows
		{"windows", "win-server-third-party", true, nil, fleet.WellKnownMDMSimpleMDM, false, false}, // is enrolled in 3rd-party, but is a server
	}
	hostsBySuffix := make(map[string]*fleet.Host, len(metadataHosts))
	for _, meta := range metadataHosts {
		var host *fleet.Host
		if meta.os == "windows" && meta.enrolledName == fleet.WellKnownMDMFleet {
			// special-case to create a properly MDM-enrolled into Fleet host
			host = createOrbitEnrolledHost(t, meta.os, meta.suffix, s.ds)
			mdmDevice := mdmtest.NewTestMDMClientWindowsProgramatic(s.server.URL, *host.OrbitNodeKey)
			err := mdmDevice.Enroll()
			require.NoError(t, err)
			updated, err := s.ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, host.UUID, mdmDevice.DeviceID)
			require.NoError(t, err)
			// No update should have been made because the orbit enrolled path results in an immediately linked hostUUID->windows enrollment
			require.False(t, updated)
			err = s.ds.SetOrUpdateMDMData(ctx, host.ID, meta.isServer, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone)
			require.NoError(t, err)
		} else {
			host = createOrbitEnrolledHost(t, meta.os, meta.suffix, s.ds)
			createDeviceTokenForHost(t, s.ds, host.ID, meta.suffix)

			serverURL := "https://example.com"
			err := s.ds.SetOrUpdateMDMData(ctx, host.ID, meta.isServer, meta.enrolledName != "", serverURL, false, meta.enrolledName, "", fleet.PersonalEnrollmentTypeNone)
			require.NoError(t, err)
		}

		if meta.teamID != nil {
			err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(meta.teamID, []uint{host.ID}))
			require.NoError(t, err)
		}
		hostsBySuffix[meta.suffix] = host
	}

	// get the orbit config for each host, verify that only the expected ones
	// receive the "needs enrollment to Windows MDM" notification.
	for _, meta := range metadataHosts {
		var resp fleet.OrbitGetConfigResponse
		s.DoJSON("POST", "/api/fleet/orbit/config",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hostsBySuffix[meta.suffix].OrbitNodeKey)),
			http.StatusOK, &resp)
		require.Equal(t, meta.shouldEnroll, resp.Notifications.NeedsProgrammaticWindowsMDMEnrollment)
		require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)
		require.False(t, resp.Notifications.NeedsMDMMigration)
		if meta.shouldEnroll {
			require.Contains(t, resp.Notifications.WindowsMDMDiscoveryEndpoint, microsoft_mdm.MDE2DiscoveryPath)
		} else {
			require.Empty(t, resp.Notifications.WindowsMDMDiscoveryEndpoint)
		}
	}

	// Set an Entra tenant ID
	acResp = appConfigResponse{}
	// Should fail with invalid value
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_entra_tenant_ids": ["tenant-id-1234"] }
  }`), http.StatusUnprocessableEntity, &acResp)

	// Should pass and create activity with valid value
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_entra_tenant_ids": ["1a86b496-e2a4-43ef-ba00-20004e29b13b"] }
  }`), http.StatusOK, &acResp)
	s.lastActivityMatches(fleet.ActivityTypeAddedMicrosoftEntraTenant{}.ActivityName(), `{"tenant_id": "1a86b496-e2a4-43ef-ba00-20004e29b13b"}`, 0)

	// Should pass and create activity with multiple valid values
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_entra_tenant_ids": ["1a86b496-e2a4-43ef-ba00-20004e29b13b", "6dca58c4-c817-4730-831b-f3348931df05"] }
  }`), http.StatusOK, &acResp)
	s.lastActivityMatches(fleet.ActivityTypeAddedMicrosoftEntraTenant{}.ActivityName(), `{"tenant_id": "6dca58c4-c817-4730-831b-f3348931df05"}`, 0)

	// Should fail and not create a new activity with multiple valid values and one new invalid one
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_entra_tenant_ids": ["1a86b496-e2a4-43ef-ba00-20004e29b13b", "6dca58c4-c817-4730-831b-f3348931df05", "fleetie-was-here"] }
  }`), http.StatusUnprocessableEntity, &acResp)
	// Same activity as before
	s.lastActivityMatches(fleet.ActivityTypeAddedMicrosoftEntraTenant{}.ActivityName(), `{"tenant_id": "6dca58c4-c817-4730-831b-f3348931df05"}`, 0)

	// Should pass and create a deleted activity with less valid values than before
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_entra_tenant_ids": ["6dca58c4-c817-4730-831b-f3348931df05"] }
  }`), http.StatusOK, &acResp)
	s.lastActivityMatches(fleet.ActivityTypeDeletedMicrosoftEntraTenant{}.ActivityName(), `{"tenant_id": "1a86b496-e2a4-43ef-ba00-20004e29b13b"}`, 0)

	// enable Windows MDM manual enrollment
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "enable_turn_on_windows_mdm_manually": true }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	assert.True(t, acResp.MDM.EnableTurnOnWindowsMDMManually)
	// No hosts should be told to enroll or unenroll because (un)enrollment is all done by the end-user
	for _, meta := range metadataHosts {
		var resp fleet.OrbitGetConfigResponse
		s.DoJSON("POST", "/api/fleet/orbit/config",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hostsBySuffix[meta.suffix].OrbitNodeKey)),
			http.StatusOK, &resp)
		require.Equal(t, false, resp.Notifications.NeedsProgrammaticWindowsMDMEnrollment)
		require.Equal(t, false, resp.Notifications.NeedsMDMMigration)
		require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)
		require.Empty(t, resp.Notifications.WindowsMDMDiscoveryEndpoint)
	}

	// enable Windows MDM migration
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_migration_enabled": true, "enable_turn_on_windows_mdm_manually": false }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	assert.True(t, acResp.MDM.WindowsMigrationEnabled)
	s.lastActivityMatches(fleet.ActivityTypeEnabledWindowsMDMMigration{}.ActivityName(), `{}`, 0)

	// get the orbit config for each host, verify that only the expected ones
	// receive the "needs enrollment to Windows MDM" and "needs migration" notifications.
	// They still get enrollment notifications as we have not proceeded with enrollment.
	for _, meta := range metadataHosts {
		var resp fleet.OrbitGetConfigResponse
		s.DoJSON("POST", "/api/fleet/orbit/config",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hostsBySuffix[meta.suffix].OrbitNodeKey)),
			http.StatusOK, &resp)
		require.Equal(t, meta.shouldEnroll, resp.Notifications.NeedsProgrammaticWindowsMDMEnrollment)
		require.Equal(t, meta.shouldMigrate, resp.Notifications.NeedsMDMMigration)
		require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)
		if meta.shouldEnroll {
			require.Contains(t, resp.Notifications.WindowsMDMDiscoveryEndpoint, microsoft_mdm.MDE2DiscoveryPath)
		} else {
			require.Empty(t, resp.Notifications.WindowsMDMDiscoveryEndpoint)
		}
	}

	// turn on MDM for another host
	orbitHost, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// disable Microsoft MDM
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_enabled_and_configured": false }
  }`), http.StatusOK, &acResp)
	assert.False(t, acResp.MDM.WindowsEnabledAndConfigured)
	assert.False(t, acResp.MDM.WindowsMigrationEnabled)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledWindowsMDM{}.ActivityName(), `{}`, 0)

	// get the orbit config for that MDM-enrolled host returns true for the
	// unenrollment notification
	var resp fleet.OrbitGetConfigResponse
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitHost.OrbitNodeKey)),
		http.StatusOK, &resp)
	require.True(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)
	require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMEnrollment)
	require.False(t, resp.Notifications.NeedsMDMMigration)
	require.Empty(t, resp.Notifications.WindowsMDMDiscoveryEndpoint)

	// get the orbit config for each host, only the fleet-enrolled ones get the unenrollment,
	// and none get enrollment/migration (because MDM is now off).
	for _, meta := range metadataHosts {
		var resp fleet.OrbitGetConfigResponse
		s.DoJSON("POST", "/api/fleet/orbit/config",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hostsBySuffix[meta.suffix].OrbitNodeKey)),
			http.StatusOK, &resp)
		require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMEnrollment)
		require.False(t, resp.Notifications.NeedsMDMMigration)
		if meta.enrolledName == fleet.WellKnownMDMFleet {
			require.True(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)
		} else {
			require.False(t, resp.Notifications.NeedsProgrammaticWindowsMDMUnenrollment)
		}
		require.Empty(t, resp.Notifications.WindowsMDMDiscoveryEndpoint)
	}
}

func (s *integrationMDMTestSuite) TestValidDiscoveryRequest() {
	t := s.T()
	// Preparing the Discovery Request message. We test the historically-supported versions plus
	// newer ones (e.g. Windows 11 25H2 advertises "9.0") to confirm any version >= the minimum is
	// accepted.
	for _, requestVersion := range []string{"4.0", "5.0", "6.0", "7.0", "8.0", "9.0", "10.0"} {
		requestBytes := []byte(`
		 <s:Envelope xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:s="http://www.w3.org/2003/05/soap-envelope">
		   <s:Header>
		     <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/management/2012/01/enrollment/IDiscoveryService/Discover</a:Action>
		     <a:MessageID>urn:uuid:148132ec-a575-4322-b01b-6172a9cf8478</a:MessageID>
		     <a:ReplyTo>
		       <a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
		     </a:ReplyTo>
		     <a:To s:mustUnderstand="1">https://mdmwindows.com:443/EnrollmentServer/Discovery.svc</a:To>
		   </s:Header>
		   <s:Body>
		     <Discover xmlns="http://schemas.microsoft.com/windows/management/2012/01/enrollment">
		       <request xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
		         <EmailAddress>demo@mdmwindows.com</EmailAddress>
		         <RequestVersion>` + requestVersion + `</RequestVersion>
		         <DeviceType>CIMClient_Windows</DeviceType>
		         <ApplicationVersion>6.2.9200.2965</ApplicationVersion>
		         <OSEdition>48</OSEdition>
		         <AuthPolicies>
		           <AuthPolicy>OnPremise</AuthPolicy>
		           <AuthPolicy>Federated</AuthPolicy>
		         </AuthPolicies>
		       </request>
		     </Discover>
		   </s:Body>
		 </s:Envelope>`)

		resp := s.DoRaw("POST", microsoft_mdm.MDE2DiscoveryPath, requestBytes, http.StatusOK)

		resBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

		// Checking if SOAP response can be unmarshalled to an golang type
		var xmlType interface{}
		err = xml.Unmarshal(resBytes, &xmlType)
		require.NoError(t, err)

		// Checking if SOAP response contains a valid DiscoveryResponse message
		resSoapMsg := string(resBytes)
		require.True(t, s.isXMLTagPresent("DiscoverResult", resSoapMsg))
		require.True(t, s.isXMLTagContentPresent("AuthPolicy", resSoapMsg))
		require.True(t, s.isXMLTagContentPresent("EnrollmentVersion", resSoapMsg))
		require.True(t, s.isXMLTagContentPresent("EnrollmentPolicyServiceUrl", resSoapMsg))
		require.True(t, s.isXMLTagContentPresent("EnrollmentServiceUrl", resSoapMsg))
	}
}

func (s *integrationMDMTestSuite) TestInvalidDiscoveryRequest() {
	t := s.T()

	// Preparing the Discovery Request message
	requestBytes := []byte(`
		 <s:Envelope xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:s="http://www.w3.org/2003/05/soap-envelope">
		   <s:Header>
		     <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/management/2012/01/enrollment/IDiscoveryService/Discover</a:Action>
		     <a:ReplyTo>
		       <a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
		     </a:ReplyTo>
		     <a:To s:mustUnderstand="1">https://mdmwindows.com:443/EnrollmentServer/Discovery.svc</a:To>
		   </s:Header>
		   <s:Body>
		     <Discover xmlns="http://schemas.microsoft.com/windows/management/2012/01/enrollment">
		       <request xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
		         <EmailAddress>demo@mdmwindows.com</EmailAddress>
		         <RequestVersion>5.0</RequestVersion>
		         <DeviceType>CIMClient_Windows</DeviceType>
		         <ApplicationVersion>6.2.9200.2965</ApplicationVersion>
		         <OSEdition>48</OSEdition>
		         <AuthPolicies>
		           <AuthPolicy>OnPremise</AuthPolicy>
		           <AuthPolicy>Federated</AuthPolicy>
		         </AuthPolicies>
		       </request>
		     </Discover>
		   </s:Body>
		 </s:Envelope>`)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2DiscoveryPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid SoapFault message
	resSoapMsg := string(resBytes)

	require.True(t, s.isXMLTagPresent("s:fault", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:value", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:text", resSoapMsg))
	require.True(t, s.checkIfXMLTagContains("s:text", "invalid SOAP header: Header.MessageID", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestNoEmailDiscoveryRequest() {
	t := s.T()

	// Preparing the Discovery Request message
	requestBytes := []byte(`
		 <s:Envelope xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:s="http://www.w3.org/2003/05/soap-envelope">
		   <s:Header>
		     <a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/management/2012/01/enrollment/IDiscoveryService/Discover</a:Action>
		     <a:MessageID>urn:uuid:148132ec-a575-4322-b01b-6172a9cf8478</a:MessageID>
		     <a:ReplyTo>
		       <a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
		     </a:ReplyTo>
		     <a:To s:mustUnderstand="1">https://mdmwindows.com:443/EnrollmentServer/Discovery.svc</a:To>
		   </s:Header>
		   <s:Body>
		     <Discover xmlns="http://schemas.microsoft.com/windows/management/2012/01/enrollment">
		       <request xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
		         <EmailAddress></EmailAddress>
		         <RequestVersion>5.0</RequestVersion>
		         <DeviceType>CIMClient_Windows</DeviceType>
		         <ApplicationVersion>6.2.9200.2965</ApplicationVersion>
		         <OSEdition>48</OSEdition>
		         <AuthPolicies>
		           <AuthPolicy>OnPremise</AuthPolicy>
		           <AuthPolicy>Federated</AuthPolicy>
		         </AuthPolicies>
		       </request>
		     </Discover>
		   </s:Body>
		 </s:Envelope>`)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2DiscoveryPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid DiscoveryResponse message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("DiscoverResult", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("AuthPolicy", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("EnrollmentVersion", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("EnrollmentPolicyServiceUrl", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("EnrollmentServiceUrl", resSoapMsg))
	require.True(t, !s.isXMLTagContentPresent("AuthenticationServiceUrl", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestValidGetPoliciesRequestWithDeviceToken() {
	t := s.T()

	// create a new Host to get the UUID on the DB
	windowsHost := createOrbitEnrolledHost(t, "windows", "h1", s.ds)

	// Preparing the GetPolicies Request message
	encodedBinToken, err := fleet.GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, *windowsHost.OrbitNodeKey)
	require.NoError(t, err)

	requestBytes, err := s.newGetPoliciesMsg(true, encodedBinToken)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2PolicyPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid GetPoliciesResponse message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("GetPoliciesResponse", resSoapMsg))
	require.True(t, s.isXMLTagPresent("policyOIDReference", resSoapMsg))
	require.True(t, s.isXMLTagPresent("oIDReferenceID", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("validityPeriodSeconds", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("renewalPeriodSeconds", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("minimalKeyLength", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestValidGetPoliciesRequestWithAzureToken() {
	t := s.T()

	tenantID := uuid.New().String()

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "windows_entra_tenant_ids": ["`+tenantID+`"] } }`), http.StatusOK, &acResp)

	// Preparing the GetPolicies Request message with Azure JWT token
	// Preparing the SecurityToken Request message with Azure JWT token
	claims := &jwt.MapClaims{
		"upn":         "fleetie@example.com",
		"tid":         tenantID,
		"iss":         "https://sts.windows.net/" + tenantID + "/",
		"unique_name": "foo_bar",
		"scp":         "mdm_delegation",
		"aud":         s.server.URL + microsoft_mdm.MDE2PolicyPath,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = defaultFakeJWTKeyID
	tokenString, err := token.SignedString(s.jwtSigningKey)
	require.NoError(t, err)

	azureADTok := base64.URLEncoding.EncodeToString([]byte(tokenString))
	requestBytes, err := s.newGetPoliciesMsg(false, azureADTok)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2PolicyPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid GetPoliciesResponse message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("GetPoliciesResponse", resSoapMsg))
	require.True(t, s.isXMLTagPresent("policyOIDReference", resSoapMsg))
	require.True(t, s.isXMLTagPresent("oIDReferenceID", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("validityPeriodSeconds", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("renewalPeriodSeconds", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("minimalKeyLength", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestGetPoliciesRequestWithInvalidUUID() {
	t := s.T()

	// create a new Host to get the UUID on the DB
	_, err := s.ds.NewHost(context.Background(), &fleet.Host{
		ID:            1,
		OsqueryHostID: ptr.String("Desktop-ABCQWE"),
		NodeKey:       ptr.String("Desktop-ABCQWE"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", s.T().Name()),
		Platform:      "windows",
	})
	require.NoError(t, err)

	// Preparing the GetPolicies Request message
	encodedBinToken, err := fleet.GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, "not_exists")
	require.NoError(t, err)

	requestBytes, err := s.newGetPoliciesMsg(true, encodedBinToken)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2PolicyPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid SoapFault message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("s:fault", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:value", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:text", resSoapMsg))
	require.True(t, s.checkIfXMLTagContains("s:text", "host data cannot be found", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestGetPoliciesRequestWithNotElegibleHost() {
	t := s.T()

	// create a new Host to get the UUID on the DB
	linuxHost := createOrbitEnrolledHost(t, "linux", "h1", s.ds)

	// Preparing the GetPolicies Request message
	encodedBinToken, err := fleet.GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, *linuxHost.OrbitNodeKey)
	require.NoError(t, err)

	requestBytes, err := s.newGetPoliciesMsg(true, encodedBinToken)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2PolicyPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid SoapFault message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("s:fault", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:value", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:text", resSoapMsg))
	require.True(t, s.checkIfXMLTagContains("s:text", "host is not elegible for Windows MDM enrollment", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestValidRequestSecurityTokenRequestWithDeviceToken() {
	t := s.T()
	windowsHost := createOrbitEnrolledHost(t, "windows", "h1", s.ds)

	// Delete the host from the list of MDM enrolled devices if present
	_, _ = s.ds.MDMWindowsDeleteEnrolledDeviceOnReenrollment(t.Context(), enrollRequestHWDevID)

	// Preparing the RequestSecurityToken Request message
	encodedBinToken, err := fleet.GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, *windowsHost.OrbitNodeKey)
	require.NoError(t, err)

	requestBytes, err := s.newSecurityTokenMsg(encodedBinToken, true, false)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2EnrollPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid RequestSecurityTokenResponseCollection message
	resSoapMsg := string(resBytes)

	require.True(t, s.isXMLTagPresent("RequestSecurityTokenResponseCollection", resSoapMsg))
	require.True(t, s.isXMLTagPresent("DispositionMessage", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("TokenType", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("RequestID", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("BinarySecurityToken", resSoapMsg))

	// Checking if an activity was created for the enrollment
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeMDMEnrolled{}.ActivityName(),
		fmt.Sprintf(`{
			"mdm_platform": "microsoft",
			"host_id": %d,
			"host_serial": "%s",
			"installed_from_dep": false,
			"host_display_name": "%s",
			"enrollment_id": null,
			"platform": "windows"
		 }`, windowsHost.ID, windowsHost.HardwareSerial, windowsHost.DisplayName()),
		0,
	)

	expectedDeviceID := "AB157C3A18778F4FB21E2739066C1F27" // TODO: make the hard-coded deviceID in `s.newSecurityTokenMsg` configurable

	// Checking if the host uuid was set on mdm windows enrollments
	d, err := s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(context.Background(), expectedDeviceID)
	require.NoError(t, err)
	require.NotEmpty(t, d.HostUUID)
	require.Equal(t, windowsHost.UUID, d.HostUUID)
}

// TODO: Do we need integration tests for WindowsMDMAutomaticEnrollmentType flows?

func (s *integrationMDMTestSuite) TestValidRequestSecurityTokenRequestWithAzureToken() {
	t := s.T()

	tenantID := uuid.New().String()

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "windows_entra_tenant_ids": ["`+tenantID+`"] } }`), http.StatusOK, &acResp)

	// Preparing the SecurityToken Request message with Azure JWT token
	claims := &jwt.MapClaims{
		"upn":         "fleetie@example.com",
		"tid":         tenantID,
		"iss":         "https://sts.windows.net/" + tenantID + "/",
		"unique_name": "foo_bar",
		"scp":         "mdm_delegation",
		"aud":         s.server.URL,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = defaultFakeJWTKeyID
	tokenString, err := token.SignedString(s.jwtSigningKey)
	require.NoError(t, err)

	azureADTok := base64.URLEncoding.EncodeToString([]byte(tokenString))
	requestBytes, err := s.newSecurityTokenMsg(azureADTok, false, false)
	require.NoError(t, err)

	mdmEnrolledCountBefore := s.countActivitiesOfType(fleet.ActivityTypeMDMEnrolled{}.ActivityName())

	resp := s.DoRaw("POST", microsoft_mdm.MDE2EnrollPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid RequestSecurityTokenResponseCollection message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("RequestSecurityTokenResponseCollection", resSoapMsg))
	require.True(t, s.isXMLTagPresent("DispositionMessage", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("TokenType", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("RequestID", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("BinarySecurityToken", resSoapMsg))

	// No activity yet: an Entra enrollment carries neither a host UUID nor a serial. It is deferred to the first time
	// the enrollment is linked to a host (see the deferred-activity tests, and the enrollment is left unclaimed below).
	require.Equal(t, mdmEnrolledCountBefore, s.countActivitiesOfType(fleet.ActivityTypeMDMEnrolled{}.ActivityName()))

	expectedDeviceID := "AB157C3A18778F4FB21E2739066C1F27" // TODO: make the hard-coded deviceID in `s.newSecurityTokenMsg` configurable

	// Checking the host uuid was not set on mdm windows enrollments
	d, err := s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(context.Background(), expectedDeviceID)
	require.NoError(t, err)
	require.Empty(t, d.HostUUID)
	require.Nil(t, d.EnrolledActivityAt, "an unlinked enrollment must leave the activity claim for the linking path")
}

func (s *integrationMDMTestSuite) TestInvalidRequestSecurityTokenRequestWithMissingAdditionalContext() {
	t := s.T()

	// create a new Host to get the UUID on the DB
	windowsHost := createOrbitEnrolledHost(t, "windows", "h1", s.ds)

	// Preparing the RequestSecurityToken Request message
	encodedBinToken, err := fleet.GetEncodedBinarySecurityToken(fleet.WindowsMDMProgrammaticEnrollmentType, *windowsHost.OrbitNodeKey)
	require.NoError(t, err)

	requestBytes, err := s.newSecurityTokenMsg(encodedBinToken, true, true)
	require.NoError(t, err)

	resp := s.DoRaw("POST", microsoft_mdm.MDE2EnrollPath, requestBytes, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)

	// Checking if SOAP response can be unmarshalled to an golang type
	var xmlType interface{}
	err = xml.Unmarshal(resBytes, &xmlType)
	require.NoError(t, err)

	// Checking if SOAP response contains a valid SoapFault message
	resSoapMsg := string(resBytes)
	require.True(t, s.isXMLTagPresent("s:fault", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:value", resSoapMsg))
	require.True(t, s.isXMLTagContentPresent("s:text", resSoapMsg))
	require.True(t, s.checkIfXMLTagContains("s:text", "ContextItem item DeviceType is not present", resSoapMsg))
}

func (s *integrationMDMTestSuite) TestValidGetTOC() {
	t := s.T()

	// hacky check to make sure the assets were built (they require `-tags full`
	// when building the tests otherwise this test will always fail due to a
	// panic in server/bindata package)
	defer func() {
		if panicVal := recover(); panicVal != nil {
			s, ok := panicVal.(string)
			if ok && strings.Contains(s, "Assets may not be used when running Fleet as a library") {
				t.Skip("skipping, test will fail due to assets not built (requires '-tags full')")
			}
		}
	}()
	_, _ = bindata.Asset("check if assets are build")

	resp := s.DoRaw("GET", microsoft_mdm.MDE2TOSPath+"?api-version=1.0&redirect_uri=ms-appx-web%3a%2f%2fMicrosoft.AAD.BrokerPlugin&client-request-id=f2cf3127-1e80-4d73-965d-42a3b84bdb40", nil, http.StatusOK)

	resBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, resp.Header["Content-Type"], syncml.WebContainerContentType)

	resTOCcontent := string(resBytes)
	require.Contains(t, resTOCcontent, "Microsoft.AAD.BrokerPlugin")
	require.Contains(t, resTOCcontent, "IsAccepted=true")
	require.Contains(t, resTOCcontent, "OpaqueBlob=")
}

func (s *integrationMDMTestSuite) TestGetTOCRejectsUnsafeRedirectURI() {
	t := s.T()

	// The TOS page reflects redirect_uri into a window.location assignment, so a javascript:/data:/vbscript:
	// redirect_uri must be rejected rather than rendered, otherwise it enables reflected XSS (issue #16880).
	unsafeRedirectURIs := map[string]string{
		"javascript": "javascript:console.log(424281957)//",
		"data":       "data:text/html,<script>alert(1)</script>",
		"vbscript":   "vbscript:msgbox(1)",
	}

	for name, redirectURI := range unsafeRedirectURIs {
		t.Run(name, func(t *testing.T) {
			resp := s.DoRaw("GET", microsoft_mdm.MDE2TOSPath+"?api-version=1.0&redirect_uri="+url.QueryEscape(redirectURI)+
				"&client-request-id=f2cf3127-1e80-4d73-965d-42a3b84bdb40", nil, http.StatusOK)

			resBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			resContent := string(resBytes)

			// The response must be a SOAP fault rather than the rendered TOS page, and must not reflect the payload.
			require.Contains(t, resp.Header["Content-Type"], syncml.SoapContentType)
			require.True(t, s.isXMLTagPresent("s:fault", resContent))
			require.NotContains(t, resContent, redirectURI)
			require.NotContains(t, resContent, "Agree and continue")
			require.NotContains(t, resContent, "IsAccepted=true")
		})
	}
}

func (s *integrationMDMTestSuite) TestWindowsMDM() {
	t := s.T()
	orbitHost, d := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	cmdOneUUID := uuid.New().String()
	commandOne := &fleet.MDMWindowsCommand{
		CommandUUID: cmdOneUUID,
		RawCommand: []byte(fmt.Sprintf(`
                     <Exec>
                       <CmdID>%s</CmdID>
                       <Item>
                         <Target>
                           <LocURI>./Device/Vendor/MSFT/Reboot/RebootNow</LocURI>
                         </Target>
                         <Meta>
                           <Format xmlns="syncml:metinf">null</Format>
                           <Type>text/plain</Type>
                         </Meta>
                         <Data></Data>
                       </Item>
                     </Exec>
		`, cmdOneUUID)),
		TargetLocURI: "./Device/Vendor/MSFT/Reboot/RebootNow",
	}
	err := s.ds.MDMWindowsInsertCommandForHosts(context.Background(), []string{orbitHost.UUID}, commandOne)
	require.NoError(t, err)

	cmds, err := d.StartManagementSession()
	require.NoError(t, err)
	// 2 Status + 1 Exec
	require.Len(t, cmds, 3)
	receivedCmd := cmds[cmdOneUUID]
	require.NotNil(t, receivedCmd)
	require.Equal(t, receivedCmd.Verb, fleet.CmdExec)
	require.Len(t, receivedCmd.Cmd.Items, 1)
	require.EqualValues(t, "./Device/Vendor/MSFT/Reboot/RebootNow", *receivedCmd.Cmd.Items[0].Target)

	msgID, err := d.GetCurrentMsgID()
	require.NoError(t, err)

	d.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &cmdOneUUID,
		Cmd:     ptr.String("Exec"),
		Data:    ptr.String("200"),
		Items:   nil,
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	})
	cmds, err = d.SendResponse()
	require.NoError(t, err)
	// the ack of the message should be the only returned command
	require.Len(t, cmds, 1)

	cmdTwoUUID := uuid.New().String()
	commandTwo := &fleet.MDMWindowsCommand{
		CommandUUID: cmdTwoUUID,
		RawCommand: []byte(fmt.Sprintf(`
                    <Get>
                      <CmdID>%s</CmdID>
                      <Item>
                        <Target>
                          <LocURI>./Device/Vendor/MSFT/DMClient/Provider/DEMO%%20MDM/SignedEntDMID</LocURI>
                        </Target>
                      </Item>
                    </Get>
		`, cmdTwoUUID)),
		TargetLocURI: "./Device/Vendor/MSFT/DMClient/Provider/DEMO%%20MDM/SignedEntDMID",
	}
	err = s.ds.MDMWindowsInsertCommandForHosts(context.Background(), []string{orbitHost.UUID}, commandTwo)
	require.NoError(t, err)

	cmdThreeUUID := uuid.New().String()
	commandThree := &fleet.MDMWindowsCommand{
		CommandUUID: cmdThreeUUID,
		RawCommand: []byte(fmt.Sprintf(`
                    <Replace>
                       <CmdID>%s</CmdID>
                       <Item>
                         <Target>
                           <LocURI>./Device/Vendor/MSFT/DMClient/Provider/DEMO%%20MDM/SignedEntDMID</LocURI>
                         </Target>
                         <Meta>
                           <Type xmlns="syncml:metinf">text/plain</Type>
                           <Format xmlns="syncml:metinf">chr</Format>
                         </Meta>
                         <Data>1</Data>
                       </Item>
                    </Replace>
		`, cmdThreeUUID)),
		TargetLocURI: "./Device/Vendor/MSFT/DMClient/Provider/DEMO%%20MDM/SignedEntDMID",
	}
	err = s.ds.MDMWindowsInsertCommandForHosts(context.Background(), []string{orbitHost.UUID}, commandThree)
	require.NoError(t, err)

	cmdFourUUID := uuid.New().String()
	commandFour := &fleet.MDMWindowsCommand{
		CommandUUID: cmdFourUUID,
		RawCommand: []byte(fmt.Sprintf(`
                    <Add>
                       <CmdID>%s</CmdID>
                       <Item>
                         <Target>
                           <LocURI>./Vendor/MSFT/WiFi/Profile/MyNetwork/WlanXml</LocURI>
                         </Target>
                         <Meta>
                           <Type xmlns="syncml:metinf">text/plain</Type>
                           <Format xmlns="syncml:metinf">chr</Format>
                         </Meta>
                         <Data>
						   &lt;?xml version=&quot;1.0&quot;?&gt;&lt;WLANProfile
						   xmlns=&quot;http://contoso.com/provisioning/EapHostConfig&quot;&gt;&lt;EapMethod&gt;&lt;Type
						 </Data>
                       </Item>
                    </Add>
		`, cmdFourUUID)),
		TargetLocURI: "./Vendor/MSFT/WiFi/Profile/MyNetwork/WlanXml",
	}
	err = s.ds.MDMWindowsInsertCommandForHosts(context.Background(), []string{orbitHost.UUID}, commandFour)
	require.NoError(t, err)

	cmds, err = d.StartManagementSession()
	require.NoError(t, err)
	// two status + the three commands we enqueued
	require.Len(t, cmds, 5)
	receivedCmdTwo := cmds[cmdTwoUUID]
	require.NotNil(t, receivedCmdTwo)
	require.Equal(t, receivedCmdTwo.Verb, fleet.CmdGet)
	require.Len(t, receivedCmdTwo.Cmd.Items, 1)
	require.EqualValues(t, "./Device/Vendor/MSFT/DMClient/Provider/DEMO%20MDM/SignedEntDMID", *receivedCmdTwo.Cmd.Items[0].Target)

	receivedCmdThree := cmds[cmdThreeUUID]
	require.NotNil(t, receivedCmdThree)
	require.Equal(t, receivedCmdThree.Verb, fleet.CmdReplace)
	require.Len(t, receivedCmdThree.Cmd.Items, 1)
	require.EqualValues(t, "./Device/Vendor/MSFT/DMClient/Provider/DEMO%20MDM/SignedEntDMID", *receivedCmdThree.Cmd.Items[0].Target)

	receivedCmdFour := cmds[cmdFourUUID]
	require.NotNil(t, receivedCmdFour)
	require.Equal(t, receivedCmdFour.Verb, fleet.CmdAdd)
	require.Len(t, receivedCmdFour.Cmd.Items, 1)
	require.EqualValues(t, "./Vendor/MSFT/WiFi/Profile/MyNetwork/WlanXml", *receivedCmdFour.Cmd.Items[0].Target)

	// status 200 for command Two  (Get)
	d.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &cmdTwoUUID,
		Cmd:     ptr.String("Get"),
		Data:    ptr.String("200"),
		Items:   nil,
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	})
	// results for command two (Get)
	cmdTwoRespUUID := uuid.NewString()
	d.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdResults},
		MsgRef:  &msgID,
		CmdRef:  &cmdTwoUUID,
		Cmd:     ptr.String("Replace"),
		Data:    ptr.String("200"),
		Items: []fleet.CmdItem{
			{
				Source: ptr.String("./Device/Vendor/MSFT/DMClient/Provider/DEMO%20MDM/SignedEntDMID"),
				Data:   &fleet.RawXmlData{Content: "0"},
			},
		},
		CmdID: fleet.CmdID{Value: cmdTwoRespUUID},
	})
	// status 200 for command Three (Replace)
	d.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &cmdThreeUUID,
		Cmd:     ptr.String("Replace"),
		Data:    ptr.String("200"),
		Items:   nil,
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	})
	// status 200 for command Four (Add)
	d.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &cmdFourUUID,
		Cmd:     ptr.String("Add"),
		Data:    ptr.String("200"),
		Items:   nil,
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	})
	cmds, err = d.SendResponse()
	require.NoError(t, err)

	// the ack of the message should be the only returned command
	require.Len(t, cmds, 1)

	// check command results

	getCommandFullResult := func(cmdUUID string) []byte {
		var fullResult []byte
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &fullResult, `
			SELECT raw_response_gz
			FROM windows_mdm_responses wmr
			JOIN windows_mdm_command_results wmcr ON wmcr.response_id = wmr.id
			WHERE command_uuid = ?
			`, cmdUUID)
		})
		if len(fullResult) == 0 {
			return fullResult
		}
		gr, err := gzip.NewReader(bytes.NewReader(fullResult))
		require.NoError(t, err)
		defer gr.Close()
		out, err := io.ReadAll(gr)
		require.NoError(t, err)
		return out
	}

	var getMDMCmdResp getMDMCommandResultsResponse
	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", cmdOneUUID)
	require.Len(t, getMDMCmdResp.Results, 1)
	require.NotZero(t, getMDMCmdResp.Results[0].UpdatedAt)
	getMDMCmdResp.Results[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMCommandResult{
		HostUUID:    orbitHost.UUID,
		CommandUUID: cmdOneUUID,
		Status:      "200",
		RequestType: "./Device/Vendor/MSFT/Reboot/RebootNow",
		Result:      getCommandFullResult(cmdOneUUID),
		Payload:     commandOne.RawCommand,
		Hostname:    orbitHost.Hostname,
	}, getMDMCmdResp.Results[0])

	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", cmdTwoUUID)
	require.Len(t, getMDMCmdResp.Results, 1)
	require.NotZero(t, getMDMCmdResp.Results[0].UpdatedAt)
	getMDMCmdResp.Results[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMCommandResult{
		HostUUID:    orbitHost.UUID,
		CommandUUID: cmdTwoUUID,
		Status:      "200",
		RequestType: "./Device/Vendor/MSFT/DMClient/Provider/DEMO%%20MDM/SignedEntDMID",
		Result:      getCommandFullResult(cmdTwoUUID),
		Payload:     commandTwo.RawCommand,
		Hostname:    orbitHost.Hostname,
	}, getMDMCmdResp.Results[0])

	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", cmdThreeUUID)
	require.Len(t, getMDMCmdResp.Results, 1)
	require.NotZero(t, getMDMCmdResp.Results[0].UpdatedAt)
	getMDMCmdResp.Results[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMCommandResult{
		HostUUID:    orbitHost.UUID,
		CommandUUID: cmdThreeUUID,
		Status:      "200",
		RequestType: "./Device/Vendor/MSFT/DMClient/Provider/DEMO%%20MDM/SignedEntDMID",
		Result:      getCommandFullResult(cmdThreeUUID),
		Hostname:    orbitHost.Hostname,
		Payload:     commandThree.RawCommand,
	}, getMDMCmdResp.Results[0])

	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", cmdFourUUID)
	require.Len(t, getMDMCmdResp.Results, 1)
	require.NotZero(t, getMDMCmdResp.Results[0].UpdatedAt)
	getMDMCmdResp.Results[0].UpdatedAt = time.Time{}
	require.Equal(t, &fleet.MDMCommandResult{
		HostUUID:    orbitHost.UUID,
		CommandUUID: cmdFourUUID,
		Status:      "200",
		RequestType: "./Vendor/MSFT/WiFi/Profile/MyNetwork/WlanXml",
		Result:      getCommandFullResult(cmdFourUUID),
		Hostname:    orbitHost.Hostname,
		Payload:     commandFour.RawCommand,
	}, getMDMCmdResp.Results[0])
}

func (s *integrationMDMTestSuite) TestWindowsMDMCommandWithSecret() {
	t := s.T()
	orbitHost, d := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	secretValue := "abcd1234"
	req := fleet.CreateSecretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "FLEET_SECRET_DATA",
				Value: secretValue,
			},
		},
	}
	secretResp := fleet.CreateSecretVariablesResponse{}
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &secretResp)

	cmdOneUUID := uuid.New().String()
	commandOne := &fleet.MDMWindowsCommand{
		CommandUUID: cmdOneUUID,
		RawCommand: []byte(fmt.Sprintf(`
                     <Exec>
                       <CmdID>%s</CmdID>
                       <Item>
                         <Target>
                           <LocURI>./Device/Vendor/MSFT/Reboot/RebootNow</LocURI>
                         </Target>
                         <Meta>
                           <Format xmlns="syncml:metinf">null</Format>
                           <Type>text/plain</Type>
                         </Meta>
                         <Data>$FLEET_SECRET_DATA</Data>
                       </Item>
                     </Exec>
		`, cmdOneUUID)),
		TargetLocURI: "./Device/Vendor/MSFT/Reboot/RebootNow",
	}
	err := s.ds.MDMWindowsInsertCommandForHosts(context.Background(), []string{orbitHost.UUID}, commandOne)
	require.NoError(t, err)

	cmds, err := d.StartManagementSession()
	require.NoError(t, err)
	// 2 Status + 1 Exec
	require.Len(t, cmds, 3)
	receivedCmd := cmds[cmdOneUUID]
	require.NotNil(t, receivedCmd)
	require.Equal(t, receivedCmd.Verb, fleet.CmdExec)
	require.Len(t, receivedCmd.Cmd.Items, 1)
	require.EqualValues(t, "./Device/Vendor/MSFT/Reboot/RebootNow", *receivedCmd.Cmd.Items[0].Target)
	assert.EqualValues(t, secretValue, receivedCmd.Cmd.Items[0].Data.Content)

	msgID, err := d.GetCurrentMsgID()
	require.NoError(t, err)

	d.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &cmdOneUUID,
		Cmd:     ptr.String("Exec"),
		Data:    ptr.String("200"),
		Items:   nil,
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	})
	cmds, err = d.SendResponse()
	require.NoError(t, err)
	// the ack of the message should be the only returned command
	require.Len(t, cmds, 1)

	var getMDMCmdResp getMDMCommandResultsResponse
	s.DoJSON("GET", "/api/latest/fleet/commands/results", nil, http.StatusOK, &getMDMCmdResp, "command_uuid", cmdOneUUID)
	require.Len(t, getMDMCmdResp.Results, 1)
	// The secret value should not be exposed via the regular API.
	assert.NotContains(t, string(getMDMCmdResp.Results[0].Payload), secretValue)
	assert.Contains(t, string(getMDMCmdResp.Results[0].Payload), "$FLEET_SECRET_DATA")
}

func (s *integrationMDMTestSuite) TestWindowsAutomaticEnrollmentCommands() {
	t := s.T()
	ctx := context.Background()

	// define a global enroll secret
	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)

	tenantID := uuid.New().String()
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "windows_entra_tenant_ids": ["`+tenantID+`"] } }`), http.StatusOK, &acResp)

	azureMail := "foo.bar.baz@example.com"
	d := mdmtest.NewTestMDMClientWindowsAutomatic(s.server.URL, azureMail, mdmtest.TestWindowsMDMClientWithSigningKeyAndTenantID(s.jwtSigningKey, defaultFakeJWTKeyID, tenantID))
	require.NoError(t, d.Enroll())

	checkinAndAck := func(expectFleetdCmds bool) {
		cmds, err := d.StartManagementSession()
		require.NoError(t, err)

		// Find fleetd install commands by GUID. Autopilot devices also
		// receive ESP hold commands (Replace/Add on CSP nodes) which we
		// ignore here.
		var fleetdAddCmd, fleetdExecCmd fleet.ProtoCmdOperation
		for _, c := range cmds {
			if c.Cmd.GetTargetURI() == syncml.FleetdWindowsInstallerGUID {
				switch c.Verb {
				case "Add":
					fleetdAddCmd = c
				case "Exec":
					fleetdExecCmd = c
				}
			}
		}

		if !expectFleetdCmds {
			require.Empty(t, fleetdAddCmd.Cmd.CmdID.Value, "should not have fleetd Add command")
			require.Empty(t, fleetdExecCmd.Cmd.CmdID.Value, "should not have fleetd Exec command")
			return
		}

		require.NotEmpty(t, fleetdAddCmd.Cmd.CmdID.Value, "should have fleetd Add command")
		require.NotEmpty(t, fleetdExecCmd.Cmd.CmdID.Value, "should have fleetd Exec command")
		require.Len(t, fleetdExecCmd.Cmd.Items, 1)

		var installJob struct {
			Product struct {
				ContentURL  string `xml:"Download>ContentURLList>ContentURL"`
				FileHash    string `xml:"Validation>FileHash"`
				CommandLine string `xml:"Enforcement>CommandLine"`
			} `xml:"Product"`
		}
		err = xml.Unmarshal([]byte(fleetdExecCmd.Cmd.Items[0].Data.Content), &installJob)
		require.NoError(t, err)
		require.Equal(t, s.mockedDownloadFleetdmMeta.MSIURL, installJob.Product.ContentURL)
		require.Equal(t, s.mockedDownloadFleetdmMeta.MSISha256, installJob.Product.FileHash)

		// The device enrolled with a valid UPN (azureMail), so the command line
		// should include an EUA_TOKEN argument.
		require.Contains(t, installJob.Product.CommandLine, `EUA_TOKEN="`)
		require.Contains(t, installJob.Product.CommandLine, `FLEET_URL="`)
		require.Contains(t, installJob.Product.CommandLine, `FLEET_SECRET="`)

		// reply with success for both commands
		msgID, err := d.GetCurrentMsgID()
		require.NoError(t, err)

		d.AppendResponse(fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdStatus},
			MsgRef:  &msgID,
			CmdRef:  &fleetdAddCmd.Cmd.CmdID.Value,
			Cmd:     &fleetdAddCmd.Verb,
			Data:    ptr.String("200"),
			Items:   nil,
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
		})
		d.AppendResponse(fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdStatus},
			MsgRef:  &msgID,
			CmdRef:  &fleetdExecCmd.Cmd.CmdID.Value,
			Cmd:     &fleetdExecCmd.Verb,
			Data:    ptr.String("200"),
			Items:   nil,
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
		})
		cmds, err = d.SendResponse()
		require.NoError(t, err)

		// The ack response includes at least a Status command. Autopilot
		// devices also receive ESP hold commands (Replace/Add).
		var hasStatus bool
		for _, c := range cmds {
			if c.Verb == "Status" {
				hasStatus = true
				break
			}
		}
		require.True(t, hasStatus, "ack response should include a Status command")
	}

	// start a management session, will receive the install fleetd commands
	checkinAndAck(true)

	// start a new management session again, Fleetd is not reported as installed
	// so it receives the commands again
	checkinAndAck(true)

	// simulate fleetd installed and enrolled
	host := createOrbitEnrolledHost(t, "windows", "h1", s.ds)
	updated, err := s.ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, host.UUID, d.DeviceID)
	require.NoError(t, err)
	require.True(t, updated)
	err = s.ds.SetOrUpdateHostOrbitInfo(ctx, host.ID, "1.23", sql.NullString{}, sql.NullBool{})
	require.NoError(t, err)

	// start a new management session again, Fleetd is reported as installed so
	// it does not receive the commands
	checkinAndAck(false)
}

func (s *integrationMDMTestSuite) TestWindowsAutopilotESPCommands() {
	t := s.T()
	ctx := context.Background()

	// Shared setup: enroll secret, Entra tenant, and a team with no profiles or setup-experience items, so every release
	// gate passes at the first Active checkin. The scenarios verify the state transitions (Pending -> Active -> None)
	// without needing profile delivery.
	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)
	tenantID := uuid.New().String()
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "windows_entra_tenant_ids": ["`+tenantID+`"] } }`), http.StatusOK, &acResp)
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	// findUserRelease returns the user-scope ServerHasFinishedProvisioning Replace from a command batch, nil when absent.
	findUserRelease := func(cmds map[string]fleet.ProtoCmdOperation) *fleet.ProtoCmdOperation {
		for _, c := range cmds {
			uri := c.Cmd.GetTargetURI()
			if c.Verb == fleet.CmdReplace && strings.Contains(uri, "./User/") && strings.Contains(uri, "ServerHasFinishedProvisioning") {
				return &c
			}
		}
		return nil
	}
	// ackAll acks every non-Status command in the batch with 200, except the command with statusOverrideUUID
	// which is acked with the given status. Returns the server's response to the ack message.
	ackAll := func(t *testing.T, d *mdmtest.TestWindowsMDMClient, cmds map[string]fleet.ProtoCmdOperation, statusOverrideUUID, overrideStatus string) map[string]fleet.ProtoCmdOperation {
		ackMsgID, err := d.GetCurrentMsgID()
		require.NoError(t, err)
		for _, c := range cmds {
			if c.Verb == "Status" {
				continue
			}
			status := "200"
			if c.Cmd.CmdID.Value == statusOverrideUUID {
				status = overrideStatus
			}
			d.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &ackMsgID, CmdRef: &c.Cmd.CmdID.Value,
				Cmd: &c.Verb, Data: &status,
				CmdID: fleet.CmdID{Value: uuid.NewString()},
			})
		}
		resp, err := d.SendResponse()
		require.NoError(t, err)
		return resp
	}
	// awaiting returns the device's current awaiting_configuration state.
	awaiting := func(t *testing.T, d *mdmtest.TestWindowsMDMClient) fleet.WindowsMDMAwaitingConfiguration {
		enrolledDevice, err := s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, d.DeviceID)
		require.NoError(t, err)
		return enrolledDevice.AwaitingConfiguration
	}
	// enrollToActive enrolls a device via Autopilot (Automatic + InOOBE -> awaiting_configuration=Pending), acks the
	// first checkin (fleetd install and ESP hold commands), simulates fleetd on the team (orbit host + host_uuid link),
	// and advances the enrollment to Active. Returns a client ready to receive the release.
	enrollToActive := func(t *testing.T, azureMail, hostName string) *mdmtest.TestWindowsMDMClient {
		d := mdmtest.NewTestMDMClientWindowsAutomatic(s.server.URL, azureMail, mdmtest.TestWindowsMDMClientWithSigningKeyAndTenantID(s.jwtSigningKey, defaultFakeJWTKeyID, tenantID))
		require.NoError(t, d.Enroll())

		cmds, err := d.StartManagementSession()
		require.NoError(t, err)
		ackAll(t, d, cmds, "", "")

		host := createOrbitEnrolledHost(t, "windows", hostName, s.ds)
		s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
			TeamID:  &tm.ID,
			HostIDs: []uint{host.ID},
		}, http.StatusOK, &addHostsToTeamResponse{})
		updated, err := s.ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, host.UUID, d.DeviceID)
		require.NoError(t, err)
		require.True(t, updated)
		err = s.ds.SetOrUpdateHostOrbitInfo(ctx, host.ID, "1.23", sql.NullString{}, sql.NullBool{})
		require.NoError(t, err)

		// First management checkin after orbit links: device transitions to Active.
		_, err = d.StartManagementSession()
		require.NoError(t, err)
		require.Equal(t, fleet.WindowsMDMAwaitingConfigurationActive, awaiting(t, d))
		return d
	}

	// countSessionStartCmds counts the ESP hold bundles (by a command only the hold bundle carries) and the
	// DevDetail serial Gets in a response.
	countSessionStartCmds := func(cmds map[string]fleet.ProtoCmdOperation) (holds, devDetailGets int) {
		for _, c := range cmds {
			uri := c.Cmd.GetTargetURI()
			switch {
			case strings.HasSuffix(uri, "/FirstSyncStatus/SkipDeviceStatusPage"):
				holds++
			case c.Verb == fleet.CmdGet && uri == devDetailSMBIOSSerialNumberURI:
				devDetailGets++
			}
		}
		return holds, devDetailGets
	}
	// requireStatusOnly asserts a reply that lets the device end the session: the SyncHdr ack and no protocol
	// commands.
	requireStatusOnly := func(t *testing.T, cmds map[string]fleet.ProtoCmdOperation) {
		require.NotEmpty(t, cmds, "the SyncHdr Status ack is always sent")
		for _, c := range cmds {
			assert.Equal(t, fleet.CmdStatus, c.Verb, "unexpected %s %s mid-session", c.Verb, c.Cmd.GetTargetURI())
		}
	}

	t.Run("pending hold commands are only sent at session start", func(t *testing.T) {
		d := mdmtest.NewTestMDMClientWindowsAutomatic(s.server.URL, "esp-pending@example.com", mdmtest.TestWindowsMDMClientWithSigningKeyAndTenantID(s.jwtSigningKey, defaultFakeJWTKeyID, tenantID))
		require.NoError(t, d.Enroll())

		// The test client retries a challenged request under the same MsgID, so both MsgID 1 and the MsgID 2
		// ack are session start.
		cmds, err := d.StartManagementSession()
		require.NoError(t, err)
		holds, gets := countSessionStartCmds(cmds)
		require.Equal(t, 1, holds)
		require.Equal(t, 1, gets)
		cmds = ackAll(t, d, cmds, "", "")
		holds, gets = countSessionStartCmds(cmds)
		require.Equal(t, 1, holds)
		require.Equal(t, 1, gets)

		// MsgID 3 gets only Status acks.
		cmds = ackAll(t, d, cmds, "", "")
		requireStatusOnly(t, cmds)
		assert.Equal(t, fleet.WindowsMDMAwaitingConfigurationPending, awaiting(t, d))

		// The next session re-sends them while still unlinked.
		cmds, err = d.StartManagementSession()
		require.NoError(t, err)
		holds, gets = countSessionStartCmds(cmds)
		require.Equal(t, 1, holds)
		require.Equal(t, 1, gets)
	})

	t.Run("serial reply links and transitions mid-session", func(t *testing.T) {
		// The shape real devices follow: the DevDetail Get sent at session start is answered on MsgID 3, which
		// links the host and moves the enrollment to Active in that same message.
		host := createOrbitEnrolledHost(t, "windows", "esp-link", s.ds)
		d := mdmtest.NewTestMDMClientWindowsAutomatic(s.server.URL, "esp-link@example.com", mdmtest.TestWindowsMDMClientWithSigningKeyAndTenantID(s.jwtSigningKey, defaultFakeJWTKeyID, tenantID))
		require.NoError(t, d.Enroll())

		cmds, err := d.StartManagementSession()
		require.NoError(t, err)
		cmds = ackAll(t, d, cmds, "", "")
		holds, gets := countSessionStartCmds(cmds)
		require.Equal(t, 1, holds)
		require.Equal(t, 1, gets)

		// MsgID 3: ack the batch and answer the Get with the serial.
		msgID, err := d.GetCurrentMsgID()
		require.NoError(t, err)
		d.AppendResponse(fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdResults},
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
			MsgRef:  &msgID, CmdRef: new(fleet.FleetInternalCmdIDPrefix + "devdetail-smbios-serial"),
			Cmd:   new(fleet.CmdGet),
			Items: []fleet.CmdItem{{Source: new(devDetailSMBIOSSerialNumberURI), Data: &fleet.RawXmlData{Content: host.HardwareSerial}}},
		})
		cmds = ackAll(t, d, cmds, "", "")

		holds, gets = countSessionStartCmds(cmds)
		assert.Zero(t, holds, "no hold bundle once linked")
		assert.Zero(t, gets, "no Get once linked")
		var transition *fleet.ProtoCmdOperation
		for _, c := range cmds {
			if c.Verb == fleet.CmdReplace && strings.HasSuffix(c.Cmd.GetTargetURI(), "/DevicePreparation/PolicyProviders/"+syncml.DocProvisioningAppProviderID+"/InstallationState") {
				transition = &c
			}
		}
		require.NotNil(t, transition, "the reply must carry the DevicePreparation InstallationState=3")
		assert.Equal(t, fleet.WindowsMDMAwaitingConfigurationActive, awaiting(t, d))
		enrolledDevice, err := s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, d.DeviceID)
		require.NoError(t, err)
		assert.Equal(t, host.UUID, enrolledDevice.HostUUID)
	})

	t.Run("user-scope release rejected with 405 then retried until acked", func(t *testing.T) {
		d := enrollToActive(t, "esp-retry@example.com", "esp-h1")

		// All release gates pass: the release commands are sent, including the user-scope ServerHasFinishedProvisioning Replace.
		cmds, err := d.StartManagementSession()
		require.NoError(t, err)
		userRelease := findUserRelease(cmds)
		require.NotNil(t, userRelease, "should release device with the user-scope ServerHasFinishedProvisioning")

		// Sending the release must NOT complete the ESP on Fleet's side: during OOBE the device rejects
		// user-scope writes with SyncML 405 until its user MDM context initializes.
		assert.Equal(t, fleet.WindowsMDMAwaitingConfigurationActive, awaiting(t, d),
			"enrollment must stay Active until the user-scope release is acked 200")

		// The device 405s the user-scope Replace (user MDM context not ready) and acks everything else 200.
		afterNack := ackAll(t, d, cmds, userRelease.Cmd.CmdID.Value, "405")
		assert.Equal(t, fleet.WindowsMDMAwaitingConfigurationActive, awaiting(t, d),
			"a 405 on the user-scope release must NOT complete the ESP")

		// The server re-sends the user-scope Replace in its reply to the ack: the test client retries a
		// challenged request under the same MsgID, so its ack carries MsgID 2, within the session-start retry
		// gate (isOMADMSessionStart). Real devices ack on MsgID 3+ and get the retry at the next session
		// instead; that shape is covered by the "acked 405 mid-session" unit subtest.
		retry := findUserRelease(afterNack)
		require.NotNil(t, retry, "the user-scope release must be re-sent after a 405")

		// The user MDM context is now up: the device accepts the retried Replace. The 200 ack completes the ESP
		// within the same message exchange (results are recorded before the ESP handler runs).
		ackAll(t, d, map[string]fleet.ProtoCmdOperation{retry.Cmd.CmdID.Value: *retry}, "", "")
		assert.Equal(t, fleet.WindowsMDMAwaitingConfigurationNone, awaiting(t, d),
			"the 200 ack of the user-scope release completes the ESP")
	})

	t.Run("user-scope release acked immediately", func(t *testing.T) {
		// The zero-retry happy path: the device accepts the release on the first send.
		d := enrollToActive(t, "esp-happy@example.com", "esp-h2")

		cmds, err := d.StartManagementSession()
		require.NoError(t, err)
		require.NotNil(t, findUserRelease(cmds), "should release device with the user-scope ServerHasFinishedProvisioning")
		assert.Equal(t, fleet.WindowsMDMAwaitingConfigurationActive, awaiting(t, d),
			"enrollment must stay Active until the user-scope release is acked 200")

		// The device acks everything 200. Completion commits within the same message exchange, and the server
		// must not send another release attempt in its reply.
		afterAck := ackAll(t, d, cmds, "", "")
		assert.Nil(t, findUserRelease(afterAck), "no retry may follow a successful ack")
		assert.Equal(t, fleet.WindowsMDMAwaitingConfigurationNone, awaiting(t, d),
			"the 200 ack of the user-scope release completes the ESP")
	})
}

func (s *integrationMDMTestSuite) TestWindowsAzureInitiatedBadKeys() {
	t := s.T()
	ctx := context.Background()

	// define a global enroll secret
	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)

	// We could generate the key differently, this just ensures it's generated the same as the "happy path" tests
	_, badKey, err := apple_mdm.NewSCEPCACertKey()
	require.NoError(t, err)

	tenantID := uuid.New().String()
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "windows_entra_tenant_ids": ["`+tenantID+`"] } }`), http.StatusOK, &acResp)

	autopilotUserMail := "swan@example.com"

	// Bad key
	mdmDevice := mdmtest.NewTestMDMClientWindowsAutomatic(s.server.URL, autopilotUserMail, mdmtest.TestWindowsMDMClientWithSigningKeyAndTenantID(badKey, defaultFakeJWTKeyID, tenantID))
	err = mdmDevice.Enroll()
	require.Error(s.T(), err)

	// Good key but wrong ID
	mdmDevice = mdmtest.NewTestMDMClientWindowsAutomatic(s.server.URL, autopilotUserMail, mdmtest.TestWindowsMDMClientWithSigningKeyAndTenantID(s.jwtSigningKey, "bad-key-id", tenantID))
	err = mdmDevice.Enroll()
	require.Error(s.T(), err)

	// Happy path to ensure the setup is correct
	mdmDevice = mdmtest.NewTestMDMClientWindowsAutomatic(s.server.URL, autopilotUserMail, mdmtest.TestWindowsMDMClientWithSigningKeyAndTenantID(s.jwtSigningKey, defaultFakeJWTKeyID, tenantID))
	err = mdmDevice.Enroll()
	require.NoError(s.T(), err)
}

func (s *integrationMDMTestSuite) TestWindowsAzureInitiatedTenantIDs() {
	t := s.T()
	ctx := context.Background()

	// define a global enroll secret
	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)

	tenantID := uuid.New().String()
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "windows_entra_tenant_ids": ["`+tenantID+`"] } }`), http.StatusOK, &acResp)

	autopilotUserMail := "swan@example.com"

	// Bad entra tenant ID
	mdmDevice := mdmtest.NewTestMDMClientWindowsAutomatic(s.server.URL, autopilotUserMail, mdmtest.TestWindowsMDMClientWithSigningKeyAndTenantID(s.jwtSigningKey, defaultFakeJWTKeyID, uuid.New().String()))
	err = mdmDevice.Enroll()
	require.Error(s.T(), err)

	// Happy path to ensure the setup is correct
	mdmDevice = mdmtest.NewTestMDMClientWindowsAutomatic(s.server.URL, autopilotUserMail, mdmtest.TestWindowsMDMClientWithSigningKeyAndTenantID(s.jwtSigningKey, defaultFakeJWTKeyID, tenantID))
	err = mdmDevice.Enroll()
	require.NoError(s.T(), err)
}

func (s *integrationMDMTestSuite) TestWindowsAzureInitiatedEnrollmentAndMapping() {
	t := s.T()
	ctx := context.Background()

	// define a global enroll secret
	err := s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}})
	require.NoError(t, err)

	tenantID := uuid.New().String()
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "windows_entra_tenant_ids": ["`+tenantID+`"] } }`), http.StatusOK, &acResp)

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
	})
	require.NoError(t, err)

	// Enroll another host to ensure the wires don't get crossed somehow
	autopilotUserMail := "swan@example.com"
	autopilotDevice := s.enrollWindowsHostInMDMViaAutopilot(s.server.URL, autopilotUserMail, tenantID)
	require.NoError(t, autopilotDevice.Enroll())

	settingsAppUserMail := "fleetie@example.com"
	settingsAppHost, settingsAppDevice := s.createWindowsHostThenEnrollMDMViaSettingsApp(s.server.URL, settingsAppUserMail, tenantID)
	require.NoError(t, settingsAppDevice.Enroll())

	// Transfer the host to the team. Ensure it doesn't wind up in "No team" at the end
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{settingsAppHost.ID}}, http.StatusOK, &addHostsToTeamResponse{})

	checkinAndAck := func(device *mdmtest.TestWindowsMDMClient, expectFleetdCmds bool) {
		cmds, err := device.StartManagementSession()
		require.NoError(t, err)

		// Find fleetd install commands by GUID. Autopilot devices also
		// receive ESP hold commands (Replace/Add on CSP nodes) which we
		// ignore here.
		var fleetdAddCmd, fleetdExecCmd fleet.ProtoCmdOperation
		for _, c := range cmds {
			if c.Cmd.GetTargetURI() == syncml.FleetdWindowsInstallerGUID {
				switch c.Verb {
				case "Add":
					fleetdAddCmd = c
				case "Exec":
					fleetdExecCmd = c
				}
			}
		}

		if !expectFleetdCmds {
			require.Empty(t, fleetdAddCmd.Cmd.CmdID.Value, "should not have fleetd Add command")
			require.Empty(t, fleetdExecCmd.Cmd.CmdID.Value, "should not have fleetd Exec command")
			return
		}

		// Enrollment via settings app or autopilot always results in a fleetd install command, even if already installed,
		// as there's no way to tell at point of MDM enrollment if fleetd is already installed.
		require.NotEmpty(t, fleetdAddCmd.Cmd.CmdID.Value, "should have fleetd Add command")
		require.NotEmpty(t, fleetdExecCmd.Cmd.CmdID.Value, "should have fleetd Exec command")
		require.Len(t, fleetdExecCmd.Cmd.Items, 1)

		var installJob struct {
			Product struct {
				ContentURL  string `xml:"Download>ContentURLList>ContentURL"`
				FileHash    string `xml:"Validation>FileHash"`
				CommandLine string `xml:"Enforcement>CommandLine"`
			} `xml:"Product"`
		}
		err = xml.Unmarshal([]byte(fleetdExecCmd.Cmd.Items[0].Data.Content), &installJob)
		require.NoError(t, err)
		require.Equal(t, s.mockedDownloadFleetdmMeta.MSIURL, installJob.Product.ContentURL)
		require.Equal(t, s.mockedDownloadFleetdmMeta.MSISha256, installJob.Product.FileHash)
		// Test name is our simulated enroll secret
		require.Contains(t, installJob.Product.CommandLine, "FLEET_SECRET=\""+t.Name()+"\"")

		// reply with success for both commands
		msgID, err := device.GetCurrentMsgID()
		require.NoError(t, err)

		device.AppendResponse(fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdStatus},
			MsgRef:  &msgID,
			CmdRef:  &fleetdAddCmd.Cmd.CmdID.Value,
			Cmd:     &fleetdAddCmd.Verb,
			Data:    ptr.String("200"),
			Items:   nil,
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
		})
		device.AppendResponse(fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdStatus},
			MsgRef:  &msgID,
			CmdRef:  &fleetdExecCmd.Cmd.CmdID.Value,
			Cmd:     &fleetdExecCmd.Verb,
			Data:    ptr.String("200"),
			Items:   nil,
			CmdID:   fleet.CmdID{Value: uuid.NewString()},
		})
		cmds, err = device.SendResponse()
		require.NoError(t, err)

		// The ack response includes at least a Status command. Autopilot
		// devices also receive ESP hold commands (Replace/Add).
		var hasStatus bool
		for _, c := range cmds {
			if c.Verb == "Status" {
				hasStatus = true
				break
			}
		}
		require.True(t, hasStatus, "ack response should include a Status command")
	}

	// windowsEnrolledDetails is the mdm_enrolled payload expected for a linked Windows host.
	windowsEnrolledDetails := func(host *fleet.Host, installedFromDEP bool) string {
		return fmt.Sprintf(`{
			"mdm_platform": "microsoft",
			"host_id": %d,
			"host_serial": "%s",
			"installed_from_dep": %t,
			"host_display_name": "%s",
			"enrollment_id": null,
			"platform": "windows"
		 }`, host.ID, host.HardwareSerial, installedFromDEP, host.DisplayName())
	}

	// Neither Entra enrollment has an mdm_enrolled activity yet: the WSTEP exchange carries no serial and no host UUID,
	// so the activity waits for the enrollment to be linked to a host rather than being recorded without either.
	mdmEnrolledName := fleet.ActivityTypeMDMEnrolled{}.ActivityName()
	require.Zero(t, s.countHostActivitiesOfType(settingsAppHost.ID, mdmEnrolledName))

	// start a management session, will receive the install fleetd commands
	checkinAndAck(settingsAppDevice, true)

	// start a new management session again, Fleetd is not reported as installed
	// so it receives the commands again.
	// Note: This doesn't mean on a real host the actual install will run twice, review
	// Microsoft MDM code to see how this is done so that it only installs once
	checkinAndAck(settingsAppDevice, true)

	// Start a management session for the autopilot device then create the host
	checkinAndAck(autopilotDevice, true)

	autopilotHost := createOrbitEnrolledHost(t, "windows", uuid.NewString(), s.ds)

	// A new management session should still send the install commands for now
	checkinAndAck(autopilotDevice, true)

	// Simulate fleetd fetching queries and reporting back the device ID via MDM queries
	// for the "settings app" device
	s.simulateMDMWindowsQueries(t, settingsAppHost, settingsAppDevice)

	// The MDM Enrollment should now be linked to the host
	mdmEnrollment, err := s.ds.MDMWindowsGetEnrolledDeviceWithHostUUID(ctx, settingsAppHost.UUID)
	require.NoError(t, err)
	require.Equal(t, settingsAppDevice.DeviceID, mdmEnrollment.MDMDeviceID)

	// Linking happened through osquery's direct-ingest backstop, which cannot record an activity itself, so the
	// enrollment is still unannounced until the device's next management session.
	require.Zero(t, s.countHostActivitiesOfType(settingsAppHost.ID, mdmEnrolledName))

	// start a new management session again, Fleetd is reported as installed so
	// it does not receive the commands
	checkinAndAck(settingsAppDevice, false)

	// That session recorded the deferred activity, now carrying the host id and serial the Entra enrollment could not
	// know. This host enrolled from the settings app (not in OOBE), so it is a manual enrollment.
	// Asserted against the host's own timeline, which also proves the activity is linked to the host and not just in
	// the global feed.
	s.lastHostActivityMatches(settingsAppHost.ID, mdmEnrolledName, windowsEnrolledDetails(settingsAppHost, false), 0)
	require.Equal(t, 1, s.countHostActivitiesOfType(settingsAppHost.ID, mdmEnrolledName))

	// A further session must not record a second one. Assert the count, not just the payload: a duplicate would be
	// byte-identical and would slip past a details-only comparison.
	checkinAndAck(settingsAppDevice, false)
	require.Equal(t, 1, s.countHostActivitiesOfType(settingsAppHost.ID, mdmEnrolledName))

	// But the autopilot device still receives commands...
	checkinAndAck(autopilotDevice, true)

	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", settingsAppHost.ID), nil, http.StatusOK, &hostResp)
	// Host was not transferred
	require.NotNil(t, hostResp.Host.TeamID)
	require.Equal(t, team.ID, *hostResp.Host.TeamID)

	// Host enrolled via settings app and is marked as enrolled manually
	require.NotNil(t, hostResp.Host.MDM.EnrollmentStatus)
	require.Equal(t, "On (manual)", *hostResp.Host.MDM.EnrollmentStatus)

	// Host has the end user mapped correctly
	require.Len(t, hostResp.Host.EndUsers, 1)
	require.Equal(t, settingsAppUserMail, hostResp.Host.EndUsers[0].IdpUserName)

	// Autopilot host checks in, gets queries, reports back
	s.simulateMDMWindowsQueries(t, autopilotHost, autopilotDevice)

	mdmEnrollment, err = s.ds.MDMWindowsGetEnrolledDeviceWithHostUUID(ctx, autopilotHost.UUID)
	require.NoError(t, err)
	require.Equal(t, autopilotDevice.DeviceID, mdmEnrollment.MDMDeviceID)

	// Same for the Autopilot device, except this one enrolled during OOBE, so it is an automatic enrollment.
	checkinAndAck(autopilotDevice, false)
	s.lastHostActivityMatches(autopilotHost.ID, mdmEnrolledName, windowsEnrolledDetails(autopilotHost, true), 0)
	require.Equal(t, 1, s.countHostActivitiesOfType(autopilotHost.ID, mdmEnrolledName))

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", autopilotHost.ID), nil, http.StatusOK, &hostResp)

	// Host enrolled via Autopilot and is marked as enrolled automatically
	require.NotNil(t, hostResp.Host.MDM.EnrollmentStatus)
	require.Equal(t, "On (automatic)", *hostResp.Host.MDM.EnrollmentStatus)

	// Host has the end user mapped correctly
	require.Len(t, hostResp.Host.EndUsers, 1)
	require.Equal(t, autopilotUserMail, hostResp.Host.EndUsers[0].IdpUserName)

	// Re-run the queries
	s.simulateMDMWindowsQueries(t, autopilotHost, autopilotDevice)
	s.simulateMDMWindowsQueries(t, settingsAppHost, settingsAppDevice)

	// Just as a final sanity check the autopilot host is still marked as enrolled automatically
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", autopilotHost.ID), nil, http.StatusOK, &hostResp)

	// Host enrolled via Autopilot and is marked as enrolled automatically
	require.NotNil(t, hostResp.Host.MDM.EnrollmentStatus)
	require.Equal(t, "On (automatic)", *hostResp.Host.MDM.EnrollmentStatus)

	// And the Settings app host manually
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", settingsAppHost.ID), nil, http.StatusOK, &hostResp)

	// Host enrolled via settings app and is marked as enrolled manually
	require.NotNil(t, hostResp.Host.MDM.EnrollmentStatus)
	require.Equal(t, "On (manual)", *hostResp.Host.MDM.EnrollmentStatus)
}

func (s *integrationMDMTestSuite) TestValidManagementUnenrollRequest() {
	t := s.T()
	ctx := t.Context()

	// Target DeviceID to use
	_, mdmHost := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	deviceID := mdmHost.DeviceID

	// Checking if device was enrolled
	_, err := s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, deviceID)
	require.NoError(t, err)

	_, err = mdmHost.StartManagementSession()
	require.NoError(t, err)
	err = mdmHost.Unenroll()
	require.NoError(t, err)

	// Checking if device was unenrolled
	_, err = s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(context.Background(), deviceID)
	require.True(t, fleet.IsNotFound(err))
}

func (s *integrationMDMTestSuite) TestUpdateMDMWindowsEnrollmentsHostUUID() {
	ctx := context.Background()
	t := s.T()

	// simulate device that is MDM enrolled before fleetd is installed
	d := fleet.MDMWindowsEnrolledDevice{
		MDMDeviceID:            "test-device-id",
		MDMHardwareID:          "test-hardware-id",
		MDMDeviceState:         "ds",
		MDMDeviceType:          "dt",
		MDMDeviceName:          "dn",
		MDMEnrollType:          "et",
		MDMEnrollUserID:        "euid",
		MDMEnrollProtoVersion:  "epv",
		MDMEnrollClientVersion: "ecv",
		MDMNotInOOBE:           false,
		HostUUID:               "", // empty host uuid when created
	}
	require.NoError(t, s.ds.MDMWindowsInsertEnrolledDevice(ctx, &d))

	gotDevice, err := s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, d.MDMDeviceID)
	require.NoError(t, err)
	require.Empty(t, gotDevice.HostUUID)

	// create an enroll secret
	secret := uuid.New().String()
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: secret}},
		},
	}, http.StatusOK, &applyResp)

	// simulate fleetd installed and enrolled
	var resp enrollOrbitResponse
	hostUUID := uuid.New().String()
	hostSerial := "test-host-serial"
	s.DoJSON("POST", "/api/fleet/orbit/enroll", fleet.EnrollOrbitRequest{
		EnrollSecret:   secret,
		HardwareUUID:   hostUUID,
		HardwareSerial: hostSerial,
		Platform:       "windows",
	}, http.StatusOK, &resp)
	require.NotEmpty(t, resp.OrbitNodeKey)

	gotDevice, err = s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, d.MDMDeviceID)
	require.NoError(t, err)
	require.Empty(t, gotDevice.HostUUID)

	// simulate first report osquery host details
	updated, err := s.ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, hostUUID, d.MDMDeviceID)
	require.NoError(t, err)
	require.True(t, updated)

	// check that the host uuid was updated
	gotDevice, err = s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, d.MDMDeviceID)
	require.NoError(t, err)
	require.NotEmpty(t, gotDevice.HostUUID)
	require.Equal(t, hostUUID, gotDevice.HostUUID)
}

func (s *integrationMDMTestSuite) isXMLTagPresent(xmlTag string, payload string) bool {
	regex := fmt.Sprintf("<%s.*>", xmlTag)
	matched, err := regexp.MatchString(regex, payload)
	if err != nil {
		return false
	}

	return matched
}

func (s *integrationMDMTestSuite) isXMLTagContentPresent(xmlTag string, payload string) bool {
	regex := fmt.Sprintf("<%s.*>(.+)</%s.*>", xmlTag, xmlTag)
	matched, err := regexp.MatchString(regex, payload)
	if err != nil {
		return false
	}

	return matched
}

func (s *integrationMDMTestSuite) checkIfXMLTagContains(xmlTag string, xmlContent string, payload string) bool {
	regex := fmt.Sprintf("<%s.*>.*%s.*</%s.*>", xmlTag, xmlContent, xmlTag)

	matched, err := regexp.MatchString(regex, payload)
	if err != nil || !matched {
		return false
	}

	return true
}

func (s *integrationMDMTestSuite) newGetPoliciesMsg(deviceToken bool, encodedBinToken string) ([]byte, error) {
	if len(encodedBinToken) == 0 {
		return nil, errors.New("encodedBinToken is empty")
	}

	// JWT token by default
	tokType := syncml.BinarySecurityAzureEnroll
	if deviceToken {
		tokType = syncml.BinarySecurityDeviceEnroll
	}

	return []byte(`
			<s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://www.w3.org/2005/08/addressing" xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd" xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" xmlns:wst="http://docs.oasis-open.org/ws-sx/ws-trust/200512" xmlns:ac="http://schemas.xmlsoap.org/ws/2006/12/authorization">
			<s:Header>
				<a:Action s:mustUnderstand="1">http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy/IPolicy/GetPolicies</a:Action>
				<a:MessageID>urn:uuid:148132ec-a575-4322-b01b-6172a9cf8478</a:MessageID>
				<a:ReplyTo>
				<a:Address>http://www.w3.org/2005/08/addressing/anonymous</a:Address>
				</a:ReplyTo>
				<a:To s:mustUnderstand="1">https://mdmwindows.com/EnrollmentServer/Policy.svc</a:To>
				<wsse:Security s:mustUnderstand="1">
				<wsse:BinarySecurityToken ValueType="` + tokType + `" EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd#base64binary">` + encodedBinToken + `</wsse:BinarySecurityToken>
				</wsse:Security>
			</s:Header>
			<s:Body xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
				<GetPolicies xmlns="http://schemas.microsoft.com/windows/pki/2009/01/enrollmentpolicy">
				<client>
					<lastUpdate xsi:nil="true"/>
					<preferredLanguage xsi:nil="true"/>
				</client>
				<requestFilter xsi:nil="true"/>
				</GetPolicies>
			</s:Body>
			</s:Envelope>`), nil
}

func (s *integrationMDMTestSuite) TestWindowsFreshEnrollEmptyQuery() {
	t := s.T()
	host, _ := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// make sure we don't have any profiles
	s.Do(
		"POST",
		"/api/v1/fleet/mdm/profiles/batch",
		batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{}},
		http.StatusNoContent,
	)

	// Ensure we can read distributed queries for the host.
	err := s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)

	s.lq.On("QueriesForHost", host.ID).Return(map[string]string{fmt.Sprintf("%d", host.ID): "SELECT 1 FROM osquery;"}, nil)

	req := getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	// We no longer read profiles via osquery
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_mdm_config_profiles_windows")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_mdm_device_id_windows")
	require.NotEmpty(t, dqResp.Queries, "fleet_detail_query_mdm_device_id_windows")
}

func (s *integrationMDMTestSuite) TestWindowsMigrationEnabled() {
	t := s.T()

	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.False(t, acResp.MDM.WindowsMigrationEnabled)

	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_migration_enabled": true
		}
	}`), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.True(t, acResp.MDM.WindowsMigrationEnabled)
	s.lastActivityMatches(fleet.ActivityTypeEnabledWindowsMDMMigration{}.ActivityName(), "", 0)

	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_migration_enabled": false
		}
	}`), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.False(t, acResp.MDM.WindowsMigrationEnabled)
	s.lastActivityMatches(fleet.ActivityTypeDisabledWindowsMDMMigration{}.ActivityName(), "", 0)

	// set migrations back to true to see if they turn false when turning MDM off
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_migration_enabled": true
		}
	}`), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.True(t, acResp.MDM.WindowsMigrationEnabled)
	lastEnabledID := s.lastActivityMatches(fleet.ActivityTypeEnabledWindowsMDMMigration{}.ActivityName(), "", 0)

	// not providing any mdm update should leave the current values
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {}
	}`), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.True(t, acResp.MDM.WindowsMigrationEnabled)
	// no new activity was created
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledWindowsMDMMigration{}.ActivityName(), "", lastEnabledID)

	// set to true again does not generate a new activity, was already true
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_migration_enabled": true
		}
	}`), http.StatusOK, &acResp)
	require.True(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.True(t, acResp.MDM.WindowsMigrationEnabled)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledWindowsMDMMigration{}.ActivityName(), "", lastEnabledID)

	res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_enabled_and_configured": false,
			"windows_migration_enabled": true
		}
	}`), http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows MDM is not enabled")

	// turn off Windows MDM and try to enable migrations in a distinct call
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_enabled_and_configured": false
		}
	}`), http.StatusOK, &acResp)
	require.False(t, acResp.MDM.WindowsEnabledAndConfigured)
	require.False(t, acResp.MDM.WindowsMigrationEnabled)

	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"windows_migration_enabled": true
		}
	}`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows MDM is not enabled")
}

// for https://github.com/fleetdm/fleet/issues/29086
func (s *integrationMDMTestSuite) TestWipeWindowsReenrollAsNewHost() {
	t := s.T()
	ctx := context.Background()

	// create an MDM-enrolled Windows host
	host, winMDMClient := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// update its serial number to empty, to simulate the DreamQuest device (and
	// others) where this can happen
	host.HardwareSerial = ""
	err := s.ds.UpdateHost(ctx, host)
	require.NoError(t, err)

	// get the host's information
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	// wipe the host
	var wipeResp fleet.WipeHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusOK, &wipeResp)
	require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)
	require.Equal(t, fleet.DeviceStatusUnlocked, wipeResp.DeviceStatus)

	// refresh the host's status, it is unlocked, pending wipe
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, string(fleet.DeviceStatusUnlocked), *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, string(fleet.PendingActionWipe), *getHostResp.Host.MDM.PendingAction)

	status, err := s.ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	// simulate a successful wipe from the Windows device's MDM response
	cmds, err := winMDMClient.StartManagementSession()
	require.NoError(t, err)

	// two status + the wipe command we enqueued
	require.Len(t, cmds, 3)
	wipeCmd := cmds[status.WipeMDMCommand.CommandUUID]
	require.NotNil(t, wipeCmd)
	require.Equal(t, wipeCmd.Verb, fleet.CmdExec)
	require.Len(t, wipeCmd.Cmd.Items, 1)
	require.EqualValues(t, "./Device/Vendor/MSFT/RemoteWipe/doWipeProtected", *wipeCmd.Cmd.Items[0].Target)

	msgID, err := winMDMClient.GetCurrentMsgID()
	require.NoError(t, err)

	winMDMClient.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &status.WipeMDMCommand.CommandUUID,
		Cmd:     ptr.String("Exec"),
		Data:    ptr.String("200"),
		Items:   nil,
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	})
	cmds, err = winMDMClient.SendResponse()
	require.NoError(t, err)
	// the ack of the message should be the only returned command
	require.Len(t, cmds, 1)

	// refresh the host's status, it is wiped
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, string(fleet.DeviceStatusWiped), *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, string(fleet.PendingActionNone), *getHostResp.Host.MDM.PendingAction)

	// enroll a new host that will re-enroll as the same host for Windows MDM
	// (via the same hardware device ID).
	newHost := createOrbitEnrolledHost(t, "windows", uuid.NewString(), s.ds)
	require.NotEqual(t, host.ID, newHost.ID)

	// now re-enroll in MDM for the new host but with the same hardware device ID
	newHostDevice := mdmtest.NewTestMDMClientWindowsProgramatic(s.server.URL, *newHost.OrbitNodeKey)
	newHostDevice.HardwareID = winMDMClient.HardwareID
	err = newHostDevice.Enroll()
	require.NoError(t, err)
	updated, err := s.ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, newHost.UUID, newHostDevice.DeviceID)
	require.NoError(t, err)
	// The MDM Enrollment should have linked the host UUID to the Windows enrollment so there should have been no update
	require.False(t, updated)
	err = s.ds.SetOrUpdateMDMData(ctx, newHost.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone)
	require.NoError(t, err)

	// refresh the (old) host's status, it should not 500
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, string(fleet.DeviceStatusUnlocked), *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, string(fleet.PendingActionNone), *getHostResp.Host.MDM.PendingAction)

	// attempting to wipe the old host entry should fail, as it is not reported
	// as enrolled in Fleet MDM anymore
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusUnprocessableEntity, &wipeResp)

	// attempting to wipe the new host entry works
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", newHost.ID), nil, http.StatusOK, &wipeResp)
	require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)
	require.Equal(t, fleet.DeviceStatusUnlocked, wipeResp.DeviceStatus)
}

// TestWindowsMDMEnrollDoesNotReplaceAnotherHostsEnrollment covers fleetd enrollments that present the hardware ID of an
// existing enrollment they may not replace, because replacing it would delete that host's enrollment and its pending wipe.
func (s *integrationMDMTestSuite) TestWindowsMDMEnrollDoesNotReplaceAnotherHostsEnrollment() {
	t := s.T()
	ctx := t.Context()

	host, hostDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	var wipeResp fleet.WipeHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusOK, &wipeResp)
	require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)

	requireEnrollmentIntact := func(t *testing.T) {
		t.Helper()
		enrollment, err := s.ds.MDMWindowsGetEnrolledDeviceWithHardwareID(ctx, hostDevice.HardwareID)
		require.NoError(t, err)
		require.Equal(t, hostDevice.DeviceID, enrollment.MDMDeviceID)
		require.Equal(t, host.UUID, enrollment.HostUUID)
	}

	t.Run("another fleetd host presenting the hardware ID is refused", func(t *testing.T) {
		otherHost := createOrbitEnrolledHost(t, "windows", uuid.NewString(), s.ds)
		otherDevice := mdmtest.NewTestMDMClientWindowsProgramatic(s.server.URL, *otherHost.OrbitNodeKey)
		otherDevice.HardwareID = hostDevice.HardwareID
		require.ErrorContains(t, otherDevice.Enroll(), "SOAP fault")
		requireEnrollmentIntact(t)
	})

	t.Run("another fleetd host presenting the device ID is refused", func(t *testing.T) {
		// Management sessions load the newest enrollment holding the device ID, so a second one would take over the host.
		otherHost := createOrbitEnrolledHost(t, "windows", uuid.NewString(), s.ds)
		otherDevice := mdmtest.NewTestMDMClientWindowsProgramatic(s.server.URL, *otherHost.OrbitNodeKey)
		otherDevice.DeviceID = hostDevice.DeviceID
		require.ErrorContains(t, otherDevice.Enroll(), "SOAP fault")
		enrollment, err := s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, hostDevice.DeviceID)
		require.NoError(t, err)
		require.Equal(t, hostDevice.HardwareID, enrollment.MDMHardwareID)
		require.Equal(t, host.UUID, enrollment.HostUUID)
	})

	t.Run("a re-enrollment that fails provisioning leaves the enrollment in place", func(t *testing.T) {
		// fleetd only enrolls a host whose osquery reports MDM off.
		require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, host.ID, false, false, s.server.URL, false, fleet.WellKnownMDMFleet, "",
			fleet.PersonalEnrollmentTypeNone))
		badCSRDevice := mdmtest.NewTestMDMClientWindowsProgramatic(s.server.URL, *host.OrbitNodeKey,
			mdmtest.TestWindowsMDMClientWithMalformedCSR())
		badCSRDevice.HardwareID = hostDevice.HardwareID
		require.ErrorContains(t, badCSRDevice.Enroll(), "SOAP fault")
		requireEnrollmentIntact(t)
	})

	// The wipe is still pending and is delivered on the host's next management session.
	status, err := s.ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	require.True(t, status.IsPendingWipe())
	cmds, err := hostDevice.StartManagementSession()
	require.NoError(t, err)
	require.Contains(t, cmds, status.WipeMDMCommand.CommandUUID)
}

// TestWindowsMDMEntraEnrollmentBoundToEntraDevice covers Entra enrollments, which name their device only by a device-reported
// hardware ID: the Entra device ID signed into the access token decides whether an enrollment may replace the one holding that
// hardware ID, and a token without it cannot replace an enrollment bound to an Entra device.
func (s *integrationMDMTestSuite) TestWindowsMDMEntraEnrollmentBoundToEntraDevice() {
	t := s.T()
	ctx := t.Context()

	tenantID := uuid.NewString()
	s.DoJSON("PATCH", "/api/latest/fleet/config",
		json.RawMessage(`{ "mdm": { "windows_entra_tenant_ids": ["`+tenantID+`"] } }`), http.StatusOK, &appConfigResponse{})
	newEntraHost := func(hardwareID, entraDeviceID string) *mdmtest.TestWindowsMDMClient {
		d := mdmtest.NewTestMDMClientWindowsAutomatic(s.server.URL, "user@example.com", mdmtest.TestWindowsMDMClientNotInOOBE(),
			mdmtest.TestWindowsMDMClientWithSigningKeyAndTenantID(s.jwtSigningKey, defaultFakeJWTKeyID, tenantID))
		if hardwareID != "" {
			d.HardwareID = hardwareID
		}
		d.EntraDeviceID = entraDeviceID
		return d
	}

	requireEnrollment := func(t *testing.T, hardwareID, mdmDeviceID, entraDeviceID string) {
		t.Helper()
		enrollment, err := s.ds.MDMWindowsGetEnrolledDeviceWithHardwareID(ctx, hardwareID)
		require.NoError(t, err)
		require.Equal(t, mdmDeviceID, enrollment.MDMDeviceID)
		require.Equal(t, entraDeviceID, enrollment.EntraDeviceID)
	}

	victim := newEntraHost("", uuid.NewString())
	require.NoError(t, victim.Enroll())
	requireEnrollment(t, victim.HardwareID, victim.DeviceID, victim.EntraDeviceID)
	host := createOrbitEnrolledHost(t, "windows", uuid.NewString(), s.ds)
	_, err := s.ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, host.UUID, victim.DeviceID)
	require.NoError(t, err)

	t.Run("another host enrolling through Entra with the hardware ID is refused", func(t *testing.T) {
		require.ErrorContains(t, newEntraHost(victim.HardwareID, uuid.NewString()).Enroll(), "SOAP fault")
		requireEnrollment(t, victim.HardwareID, victim.DeviceID, victim.EntraDeviceID)
	})

	t.Run("a token without a deviceid claim cannot replace a bound enrollment", func(t *testing.T) {
		require.ErrorContains(t, newEntraHost(victim.HardwareID, "").Enroll(), "SOAP fault")
		requireEnrollment(t, victim.HardwareID, victim.DeviceID, victim.EntraDeviceID)

		// It still enrolls a hardware ID that no enrollment holds, unbound.
		unbound := newEntraHost("", "")
		require.NoError(t, unbound.Enroll())
		requireEnrollment(t, unbound.HardwareID, unbound.DeviceID, "")
	})

	t.Run("a fleetd enrollment of an Autopilot host is bound to its Autopilot Entra device ID", func(t *testing.T) {
		autopilotHost, fleetdDevice := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
		autopilotEntraDeviceID := uuid.NewString()
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `INSERT INTO host_autopilot_devices (host_id, autopilot_device_id, entra_device_id, tenant_id)
				VALUES (?, ?, ?, ?)`, autopilotHost.ID, uuid.NewString(), autopilotEntraDeviceID, tenantID)
			return err
		})

		require.ErrorContains(t, newEntraHost(fleetdDevice.HardwareID, uuid.NewString()).Enroll(), "SOAP fault")
		requireEnrollment(t, fleetdDevice.HardwareID, fleetdDevice.DeviceID, "")

		sameDevice := newEntraHost(fleetdDevice.HardwareID, autopilotEntraDeviceID)
		require.NoError(t, sameDevice.Enroll())
		requireEnrollment(t, fleetdDevice.HardwareID, sameDevice.DeviceID, autopilotEntraDeviceID)
	})

	t.Run("the same host re-enrolls through Entra", func(t *testing.T) {
		reenrolled := newEntraHost(victim.HardwareID, victim.EntraDeviceID)
		require.NoError(t, reenrolled.Enroll())
		requireEnrollment(t, victim.HardwareID, reenrolled.DeviceID, victim.EntraDeviceID)

		// a fleetd re-enrollment of its host keeps the Entra binding
		_, err := s.ds.UpdateMDMWindowsEnrollmentsHostUUID(ctx, host.UUID, reenrolled.DeviceID)
		require.NoError(t, err)
		// fleetd only enrolls a host whose osquery reports MDM off.
		require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, host.ID, false, false, s.server.URL, false, fleet.WellKnownMDMFleet, "",
			fleet.PersonalEnrollmentTypeNone))
		fleetdDevice := mdmtest.NewTestMDMClientWindowsProgramatic(s.server.URL, *host.OrbitNodeKey)
		fleetdDevice.HardwareID = victim.HardwareID
		require.NoError(t, fleetdDevice.Enroll())
		requireEnrollment(t, victim.HardwareID, fleetdDevice.DeviceID, victim.EntraDeviceID)

		require.ErrorContains(t, newEntraHost(victim.HardwareID, uuid.NewString()).Enroll(), "SOAP fault")
		requireEnrollment(t, victim.HardwareID, fleetdDevice.DeviceID, victim.EntraDeviceID)
	})
}

func (s *integrationMDMTestSuite) TestWindowsRekeyFlow() {
	t := s.T()
	ctx := t.Context()

	_, mdmHost := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)

	// Now we remove the credentials_hash and ack to simulate an existing enrollment to force rekeying
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE mdm_windows_enrollments SET credentials_hash = NULL, credentials_acknowledged = FALSE WHERE mdm_device_id = ?", mdmHost.DeviceID)
		return err
	})

	_, err := mdmHost.StartManagementSession()
	require.NoError(t, err)

	var updatedValues struct {
		CredentialsHash         *[]byte `db:"credentials_hash"`
		CredentialsAcknowledged bool    `db:"credentials_acknowledged"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		err := sqlx.GetContext(ctx, q, &updatedValues, "SELECT credentials_hash, credentials_acknowledged FROM mdm_windows_enrollments WHERE mdm_device_id = ?", mdmHost.DeviceID)
		return err
	})

	require.NotNil(t, updatedValues.CredentialsHash)
	require.True(t, updatedValues.CredentialsAcknowledged)
}
