package service

// Conditional access tests for the enterprise (premium) suite.
//
// Belongs here: Microsoft Entra and Okta conditional access setup, policies, compliance result
// processing, platform validation, bypass, related activities, and the conditional access flags
// in device endpoint responses.
//
// Does not belong here: other Fleet Desktop device endpoints (integration_enterprise_desktop_test.go)
// and policy CRUD (integration_enterprise_policies_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_microsoft_proxy"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testOktaValidCert is a valid PEM-encoded x509 certificate used by tests that
// configure Okta conditional access in AppConfig.
const testOktaValidCert = `-----BEGIN CERTIFICATE-----
MIIDqDCCApCgAwIBAgIGAZXsT7aXMA0GCSqGSIb3DQEBCwUAMIGUMQswCQYDVQQGEwJVUzETMBEG
A1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzENMAsGA1UECgwET2t0YTEU
MBIGA1UECwwLU1NPUHJvdmlkZXIxFTATBgNVBAMMDGRldi01Nzk4ODEyOTEcMBoGCSqGSIb3DQEJ
ARYNaW5mb0Bva3RhLmNvbTAeFw0yNTAzMzExMzA1NDFaFw0zNTAzMzExMzA2NDFaMIGUMQswCQYD
VQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5jaXNjbzENMAsG
A1UECgwET2t0YTEUMBIGA1UECwwLU1NPUHJvdmlkZXIxFTATBgNVBAMMDGRldi01Nzk4ODEyOTEc
MBoGCSqGSIb3DQEJARYNaW5mb0Bva3RhLmNvbTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBAIS/AMr00GVLTHWnufTZg9sWjJhEkEawLoSRtMPZhJRPi/8rKsKk0fiYK6YKHpiY+iL4kle0
NHVMAQhk6vC4wmiaKMy8iEZxJB2gWLO/Xk6b+Vaa1Fu4xg+wWb61ue46HGRhvhHG3eHtz8NOLao4
2DRCjbghCv+qRDcfgei/IrrTUmSJDUMXNMtaQbg+dOMeQRbgfkz2x6LI/TeBKghIGHIYRKzebcH6
kr1XtgVapG+X6NccjL4FmIvfITpOK6+B3wdszEH5HUicMdZEt/8yLO00kJZhxRVCvK0LbzYEHFx5
ftIyBCB6iwIZ9eECf4p87UxOfe0AD0NAdm/BR+dr1psCAwEAATANBgkqhkiG9w0BAQsFAAOCAQEA
Wzh9U6/I5G/Uy/BoMTv3lBsbS6h7OGUE2kOTX5YF3+t4EKlGNHNHx1CcOa7kKb1Cpagnu3UfThly
nMVWcUemsnhjN+6DeTGpqX/GGpQ22YKIZbqFm90jS+CtLQQsi0ciU7w4d981T2I7oRs9yDk+A2ZF
9yf8wGi6ocy4EC00dCJ7DoSui6HdYiQWk60K4w7LPqtvx2bPPK9j+pmAbuLmHPAQ4qyccDZVDOaP
umSer90UyfV6FkY8/nfrqDk6tE8RyabI3o48Q4m12RoYcA3sZ3Ba3A4CzP7Q0uUFD6nMTqgq4ZeV
FqU+KJOed6qlzj7qy+u5l6CQeajLGdjUxFlFyw==
-----END CERTIFICATE-----`

// clearOktaConditionalAccess clears all Okta conditional access configuration fields.
// This is useful for test cleanup to ensure tests don't interfere with each other.
func (s *integrationEnterpriseTestSuite) clearOktaConditionalAccess() {
	body := map[string]any{
		"conditional_access": map[string]any{
			"okta_idp_id":                         nil,
			"okta_assertion_consumer_service_url": nil,
			"okta_audience_uri":                   nil,
			"okta_certificate":                    nil,
			"bypass_disabled":                     false,
		},
	}
	b, err := json.Marshal(body)
	require.NoError(s.T(), err)
	s.DoRaw("PATCH", "/api/latest/fleet/config", b, http.StatusOK)
}

// TestDeviceHostConditionalAccessFeatures tests the EnableConditionalAccess and
// EnableConditionalAccessBypass flags in the device endpoint response for hosts WITH teams.
// EnableConditionalAccess requires: host has team + global Okta configured + team conditional access enabled.
// EnableConditionalAccessBypass is controlled solely by AppConfig.ConditionalAccess.BypassEnabled().
func (s *integrationEnterpriseTestSuite) TestDeviceHostConditionalAccessFeatures() {
	t := s.T()
	ctx := t.Context()

	s.clearOktaConditionalAccess()
	// Clean up Okta conditional access config at the end
	t.Cleanup(func() {
		s.clearOktaConditionalAccess()
	})

	// Create a test team
	team, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:        "team-conditional-access-test",
		Description: "Test team for conditional access",
	})
	require.NoError(t, err)

	// Create a host with device token and assign to team
	token := "conditional_access_features_test_token"
	host := createHostAndDeviceToken(t, s.ds, token)
	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
	require.NoError(t, err)

	// Helper to call the device endpoint and return the response
	getDeviceHost := func() getDeviceHostResponse {
		var resp getDeviceHostResponse
		res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK)
		err := json.NewDecoder(res.Body).Decode(&resp)
		require.NoError(t, err)
		res.Body.Close()
		return resp
	}

	// Test case 1: No global Okta configured, team conditional access not enabled
	// Expected: EnableConditionalAccess=false, EnableConditionalAccessBypass=false (no ConditionalAccess in AppConfig)
	t.Run("no_okta_no_team_config", func(t *testing.T) {
		resp := getDeviceHost()
		require.NoError(t, resp.Err)
		assert.False(t, resp.GlobalConfig.Features.EnableConditionalAccess, "should be false when Okta not configured")
		assert.True(t, resp.GlobalConfig.Features.EnableConditionalAccessBypass, "should be true when no ConditionalAccess config")
	})

	// Configure Okta at global level (without bypass settings first)
	validCert := `-----BEGIN CERTIFICATE-----
MIICpDCCAYwCCQDU+pQ4P2GH3jANBgkqhkiG9w0BAQsFADAUMRIwEAYDVQQDDAls
b2NhbGhvc3QwHhcNMjMxMjA2MTYyOTQ0WhcNMjQxMjA1MTYyOTQ0WjAUMRIwEAYD
VQQDDAlsb2NhbGhvc3QwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC8
fKEMF69sJR8Ky7Vrt3EfJvL3NlVPgj/OAkFhpKgL9QPrjAAY9qdmkLuU5eBPpSvB
jAqGUEBLHKGUKx8kIJxoBguq5WIWwIBq1b3cmPbUXtDi7GzqoUqPSfPWMzLCrmpl
N5RYPu/pRWg9M4vI2XdhVFaDOE6X1sXhNqYfr7TNbOfxDQ0VPjpNqHY+kiEAqmcr
tJzuJFYN1y8eyevZj4VGTS/dZ3HYHWBpZ6xpVoZ6LWDqdmLPLQkp2ceGLCvoFaG8
TaMMfz3dfHnMvI9o7IrT8kCbLqVoxhLk1FwT+iLLL9v0rhOUbg/mvNT6BQM0P7rA
wDOx1kIauDmJQVWe9nYPAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAChAhSvUqH+u
wksgUXqBpQt7OPQhYmpCtq16eIYb6+LzX1FnTuMdaiT4q9FBnJVNcS2ofxhkv/67
d7r5SjMqFvCnSZujddP0TGo5Z7wfPhXKG5+X2GQMIX7agh7lfx5y5F5I/TsTQ+DS
AbKIDRWaHD5hPjUobJKtBdZxicXZj7e/K1mPxyQu9K3j/wPqIGkKwmEQQNWw/mFH
A9XUANZP+7e9EbV6AMtJPA/vmg3mSFqX+N2xLBFvpfhxdPMXLDOO3EHKQ8IMWPOC
A3smrFnIVFrVeLPn47FnPVP8HzT8dcMBwGKOGANW1VAMEwlZXHdVlRaGVqd9FbxS
KSCy+VfKBn4=
-----END CERTIFICATE-----`

	s.DoRaw("PATCH", "/api/latest/fleet/config", fmt.Appendf(nil, `{
		"conditional_access": {
			"okta_idp_id": "https://www.okta.com/saml2/service-provider/test",
			"okta_assertion_consumer_service_url": "https://dev-test.okta.com/sso/saml2/test",
			"okta_audience_uri": "https://www.okta.com/saml2/service-provider/test",
			"okta_certificate": %q
		}
	}`, validCert), http.StatusOK)

	// Test case 2: Global Okta configured, team conditional access NOT enabled
	// Expected: EnableConditionalAccess=false (team not enabled), EnableConditionalAccessBypass=true (default)
	t.Run("okta_configured_team_not_enabled", func(t *testing.T) {
		resp := getDeviceHost()
		require.NoError(t, resp.Err)
		assert.False(t, resp.GlobalConfig.Features.EnableConditionalAccess, "should be false when team conditional access not enabled")
		assert.True(t, resp.GlobalConfig.Features.EnableConditionalAccessBypass, "should be true when bypass not explicitly disabled")
	})

	// Enable conditional access on the team
	var tmResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"integrations": map[string]any{
			"conditional_access_enabled": true,
		},
	}, http.StatusOK, &tmResp)

	// Test case 3: Global Okta configured, team conditional access enabled
	// Expected: EnableConditionalAccess=true, EnableConditionalAccessBypass=true (default)
	t.Run("okta_configured_team_enabled", func(t *testing.T) {
		resp := getDeviceHost()
		require.NoError(t, resp.Err)
		assert.True(t, resp.GlobalConfig.Features.EnableConditionalAccess, "should be true when Okta configured and team enabled")
		assert.True(t, resp.GlobalConfig.Features.EnableConditionalAccessBypass, "should be true when bypass not explicitly disabled")
	})

	// Disable bypass explicitly
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"conditional_access": {
			"bypass_disabled": true
		}
	}`), http.StatusOK)

	// Test case 4: Bypass explicitly disabled
	// Expected: EnableConditionalAccess=true, EnableConditionalAccessBypass=false
	t.Run("bypass_explicitly_disabled", func(t *testing.T) {
		resp := getDeviceHost()
		require.NoError(t, resp.Err)
		assert.True(t, resp.GlobalConfig.Features.EnableConditionalAccess, "should still be true")
		assert.False(t, resp.GlobalConfig.Features.EnableConditionalAccessBypass, "should be false when bypass explicitly disabled")
	})

	// Re-enable bypass explicitly
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"conditional_access": {
			"bypass_disabled": false
		}
	}`), http.StatusOK)

	// Test case 5: Bypass explicitly enabled
	// Expected: EnableConditionalAccess=true, EnableConditionalAccessBypass=true
	t.Run("bypass_explicitly_enabled", func(t *testing.T) {
		resp := getDeviceHost()
		require.NoError(t, resp.Err)
		assert.True(t, resp.GlobalConfig.Features.EnableConditionalAccess, "should still be true")
		assert.True(t, resp.GlobalConfig.Features.EnableConditionalAccessBypass, "should be true when bypass explicitly enabled")
	})

	// Disable conditional access on the team
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"integrations": map[string]any{
			"conditional_access_enabled": false,
		},
	}, http.StatusOK, &tmResp)

	// Test case 6: Global Okta configured, team conditional access disabled
	// Expected: EnableConditionalAccess=false, EnableConditionalAccessBypass=true
	t.Run("okta_configured_team_disabled", func(t *testing.T) {
		resp := getDeviceHost()
		require.NoError(t, resp.Err)
		assert.False(t, resp.GlobalConfig.Features.EnableConditionalAccess, "should be false when team disabled")
		assert.True(t, resp.GlobalConfig.Features.EnableConditionalAccessBypass, "should still be true since global bypass is enabled")
	})
}

func genDistributedReqWithEntraIDDetails(host *fleet.Host, deviceID, userPrincipalName string) submitDistributedQueryResultsRequestShim {
	results := make(map[string]json.RawMessage)
	results["fleet_detail_query_conditional_access_microsoft_device_id"] = json.RawMessage(fmt.Sprintf(`[{"device_id": "%s", "user_principal_name": "%s"}]`, deviceID, userPrincipalName))
	return submitDistributedQueryResultsRequestShim{
		NodeKey:  *host.NodeKey,
		Results:  results,
		Statuses: make(map[string]interface{}),
		Messages: make(map[string]string),
		Stats:    map[string]*fleet.Stats{},
	}
}

func genDistributedReqWithEntraIDDetailsForWindows(host *fleet.Host, deviceID string) submitDistributedQueryResultsRequestShim {
	results := make(map[string]json.RawMessage)
	results["fleet_detail_query_conditional_access_microsoft_device_id_windows"] = json.RawMessage(fmt.Sprintf(`[{"device_id": "%s"}]`, deviceID))
	return submitDistributedQueryResultsRequestShim{
		NodeKey:  *host.NodeKey,
		Results:  results,
		Statuses: make(map[string]any),
		Messages: make(map[string]string),
		Stats:    map[string]*fleet.Stats{},
	}
}

const (
	testEntraTenantID      = "6f1c2a3b-4d5e-4f60-8a7b-9c0d1e2f3a4b"
	otherTestEntraTenantID = "0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d"
)

func (s *integrationEnterpriseTestSuite) TestConditionalAccessBasicSetup() {
	t := s.T()
	t.Cleanup(func() {
		s.clearOktaConditionalAccess()
	})

	var acResp appConfigResponse

	// Test global maintainer fails to create the integration.
	u := &fleet.User{
		Name:       "test maintainer",
		Email:      "maintainer@example.com",
		GlobalRole: new(fleet.RoleMaintainer),
	}
	password := test.GoodPassword
	require.NoError(t, u.SetPassword(password, 10, 10))
	_, err := s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)
	s.token = s.getTestToken("maintainer@example.com", password)
	var r conditionalAccessMicrosoftCreateResponse
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftCreateRequest{}, http.StatusForbidden, &r)
	var c conditionalAccessMicrosoftConfirmResponse
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft/confirm", conditionalAccessMicrosoftConfirmRequest{}, http.StatusForbidden, &c)
	var d conditionalAccessMicrosoftDeleteResponse
	s.DoJSON("DELETE", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftDeleteRequest{}, http.StatusForbidden, &d)

	// Restore token for global admin.
	s.token = s.getTestAdminToken()

	// Tenant IDs that are not canonical GUIDs are rejected.
	for _, invalidTenantID := range []string{
		"",
		"foobar",
		testEntraTenantID + "&fleetServerSecret=smuggled",
		"{" + testEntraTenantID + "}",
		"urn:uuid:" + testEntraTenantID,
		strings.ReplaceAll(testEntraTenantID, "-", ""),
	} {
		res := s.Do("POST", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftCreateRequest{
			MicrosoftTenantID: invalidTenantID,
		}, http.StatusUnprocessableEntity)
		require.Contains(t, extractServerErrorText(res.Body), "must be a valid Microsoft Entra tenant ID (GUID)", invalidTenantID)
	}

	// Setup integration.
	mockedConditionalAccessMicrosoftProxyInstance.getResponse = &conditional_access_microsoft_proxy.GetResponse{
		TenantID:        testEntraTenantID,
		SetupDone:       false,
		AdminConsentURL: "https://example.com",
	}
	mockedConditionalAccessMicrosoftProxyInstance.createResponse = &conditional_access_microsoft_proxy.CreateResponse{
		TenantID: testEntraTenantID,
		Secret:   "secret",
	}
	r = conditionalAccessMicrosoftCreateResponse{}
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftCreateRequest{
		MicrosoftTenantID: testEntraTenantID,
	}, http.StatusOK, &r)
	require.Equal(t, "https://example.com", r.MicrosoftAuthenticationURL)

	// UI uses the /config endpoint to know the status of the integration.
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.NotNil(t, acResp.ConditionalAccess)
	require.Equal(t, testEntraTenantID, acResp.ConditionalAccess.MicrosoftEntraTenantID)
	require.False(t, acResp.ConditionalAccess.MicrosoftEntraConnectionConfigured)

	// Confirm should return that the setup is not done.
	c = conditionalAccessMicrosoftConfirmResponse{}
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft/confirm", conditionalAccessMicrosoftConfirmRequest{}, http.StatusOK, &c)
	require.False(t, c.ConfigurationCompleted)

	// Confirm now should succeed.
	mockedConditionalAccessMicrosoftProxyInstance.getResponse = &conditional_access_microsoft_proxy.GetResponse{
		TenantID:  testEntraTenantID,
		SetupDone: true,
	}
	c = conditionalAccessMicrosoftConfirmResponse{}
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft/confirm", conditionalAccessMicrosoftConfirmRequest{}, http.StatusOK, &c)
	require.True(t, c.ConfigurationCompleted)
	// Confirm again should succeed because integration is done.
	c = conditionalAccessMicrosoftConfirmResponse{}
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft/confirm", conditionalAccessMicrosoftConfirmRequest{}, http.StatusOK, &c)
	require.True(t, c.ConfigurationCompleted)
	// Create will succeed if using the same tenant ID.
	r = conditionalAccessMicrosoftCreateResponse{}
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftCreateRequest{
		MicrosoftTenantID: testEntraTenantID,
	}, http.StatusOK, &r)
	// Create will should fail if using the a different tenant ID (if the setup is done).
	r = conditionalAccessMicrosoftCreateResponse{}
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftCreateRequest{
		MicrosoftTenantID: otherTestEntraTenantID,
	}, http.StatusBadRequest, &r)

	// Test app config returns that the configuration is done.
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.NotNil(t, acResp.ConditionalAccess)
	require.Equal(t, testEntraTenantID, acResp.ConditionalAccess.MicrosoftEntraTenantID)
	require.True(t, acResp.ConditionalAccess.MicrosoftEntraConnectionConfigured)

	// Delete endpoint.
	mockedConditionalAccessMicrosoftProxyInstance.deleteResponse = &conditional_access_microsoft_proxy.DeleteResponse{}
	d = conditionalAccessMicrosoftDeleteResponse{}
	s.DoJSON("DELETE", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftDeleteRequest{}, http.StatusOK, &d)
	// Deleting again should fail with bad request.
	d = conditionalAccessMicrosoftDeleteResponse{}
	s.DoJSON("DELETE", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftDeleteRequest{}, http.StatusBadRequest, &d)

	// Test app config returns that the integration was deleted.
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	// ConditionalAccess is always initialized, but Entra fields should be empty
	require.NotNil(t, acResp.ConditionalAccess)
	require.Equal(t, "", acResp.ConditionalAccess.MicrosoftEntraTenantID)
	require.False(t, acResp.ConditionalAccess.MicrosoftEntraConnectionConfigured)

	// Create again with a different tenant.
	mockedConditionalAccessMicrosoftProxyInstance.getResponse = &conditional_access_microsoft_proxy.GetResponse{
		TenantID:        otherTestEntraTenantID,
		SetupDone:       false,
		AdminConsentURL: "https://example.com",
	}
	mockedConditionalAccessMicrosoftProxyInstance.createResponse = &conditional_access_microsoft_proxy.CreateResponse{
		TenantID: otherTestEntraTenantID,
		Secret:   "secret",
	}
	r = conditionalAccessMicrosoftCreateResponse{}
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftCreateRequest{
		MicrosoftTenantID: otherTestEntraTenantID,
	}, http.StatusOK, &r)

	// Test app config returns that the new integration was created.
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.NotNil(t, acResp.ConditionalAccess)
	require.Equal(t, otherTestEntraTenantID, acResp.ConditionalAccess.MicrosoftEntraTenantID)
	require.False(t, acResp.ConditionalAccess.MicrosoftEntraConnectionConfigured)

	// Simulate a not found error on the proxy (should allow deletion to start over).
	mockedConditionalAccessMicrosoftProxyInstance.deleteErr = &notFoundError{}
	d = conditionalAccessMicrosoftDeleteResponse{}
	s.DoJSON("DELETE", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftDeleteRequest{}, http.StatusOK, &d)

	// Test app config returns that the configuration is gone.
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	// ConditionalAccess is always initialized, but Entra fields should be empty
	require.NotNil(t, acResp.ConditionalAccess)
	require.Equal(t, "", acResp.ConditionalAccess.MicrosoftEntraTenantID)
	require.False(t, acResp.ConditionalAccess.MicrosoftEntraConnectionConfigured)
}

func (s *integrationEnterpriseTestSuite) TestConditionalAccessPolicies() {
	t := s.T()

	// Setup integration.
	mockedConditionalAccessMicrosoftProxyInstance.getResponse = &conditional_access_microsoft_proxy.GetResponse{
		TenantID:        testEntraTenantID,
		SetupDone:       false,
		AdminConsentURL: "https://example.com",
	}
	mockedConditionalAccessMicrosoftProxyInstance.createResponse = &conditional_access_microsoft_proxy.CreateResponse{
		TenantID: testEntraTenantID,
		Secret:   "secret",
	}
	var r conditionalAccessMicrosoftCreateResponse
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftCreateRequest{
		MicrosoftTenantID: testEntraTenantID,
	}, http.StatusOK, &r)
	mockedConditionalAccessMicrosoftProxyInstance.getResponse = &conditional_access_microsoft_proxy.GetResponse{
		TenantID:  testEntraTenantID,
		SetupDone: true,
	}
	var c conditionalAccessMicrosoftConfirmResponse
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft/confirm", conditionalAccessMicrosoftConfirmRequest{}, http.StatusOK, &c)
	require.True(t, c.ConfigurationCompleted)

	t1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	var pr fleet.TeamPolicyResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", t1.ID), fleet.TeamPolicyRequest{
		Query:                    "SELECT 1;",
		Name:                     "Compliance check 1",
		Platform:                 "darwin,windows",
		ConditionalAccessEnabled: true,
	}, http.StatusOK, &pr)
	cp1 := pr.Policy
	pr = fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", t1.ID), fleet.TeamPolicyRequest{
		Query:                    "SELECT 2;",
		Name:                     "Compliance check 2",
		ConditionalAccessEnabled: true,
		Platform:                 "darwin,windows",
	}, http.StatusOK, &pr)
	cp2 := pr.Policy
	pr = fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", t1.ID), fleet.TeamPolicyRequest{
		Query:                    "SELECT 3;",
		Name:                     "Other policy",
		Platform:                 "darwin,windows",
		ConditionalAccessEnabled: false,
	}, http.StatusOK, &pr)
	p3 := pr.Policy

	ctx := context.Background()
	newHost := func(name string, teamID *uint, platform string, osVersion string) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   new(t.Name() + name),
			NodeKey:         new(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        platform,
			TeamID:          teamID,

			OSVersion: osVersion,
		})
		require.NoError(t, err)
		return h
	}

	//
	// Test A: host h1 in team t1.
	//

	h1 := newHost("h1", &t1.ID, "darwin", "macOS 15.7.3")
	orbitKey := setOrbitEnrollment(t, h1, s.ds)
	h1.OrbitNodeKey = &orbitKey

	windowsHost1 := newHost("windowsHost1", &t1.ID, "windows", "Windows 10 Pro 22H2 10.0.19045.6456")
	windowsHostorbitKey := setOrbitEnrollment(t, windowsHost1, s.ds)
	windowsHost1.OrbitNodeKey = &windowsHostorbitKey

	// Feature is disabled on the host's team.
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithEntraIDDetails(h1, "entraDeviceID", "entraUserPrincipalName"), http.StatusOK, &distributedResp)
	_, err = s.ds.LoadHostConditionalAccessStatus(ctx, h1.ID)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	// Enable feature on team.
	var tmResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", t1.ID), map[string]any{
		"integrations": map[string]any{
			"conditional_access_enabled": true,
		},
	}, http.StatusOK, &tmResp)

	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithEntraIDDetails(h1, "entraDeviceID", "entraUserPrincipalName"), http.StatusOK, &distributedResp)
	h1s, err := s.ds.LoadHostConditionalAccessStatus(ctx, h1.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceID", h1s.DeviceID)
	require.Equal(t, "entraUserPrincipalName", h1s.UserPrincipalName)
	require.Nil(t, h1s.Managed)
	require.Nil(t, h1s.Compliant)

	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithEntraIDDetailsForWindows(windowsHost1, "entraDeviceIDWindows"), http.StatusOK, &distributedResp)
	wh1s, err := s.ds.LoadHostConditionalAccessStatus(ctx, windowsHost1.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceIDWindows", wh1s.DeviceID)
	require.Equal(t, "", wh1s.UserPrincipalName)
	require.Nil(t, wh1s.Managed)
	require.Nil(t, wh1s.Compliant)

	// override value to reduce test time.
	conditionalAccessSetWaitTime = 250 * time.Millisecond

	var (
		setDeviceName string
		setOSName     string
		setOSVersion  string
	)
	mockedConditionalAccessMicrosoftProxyInstance.setComplianceStatusFunc = func(
		ctx context.Context,
		tenantID string, secret string,
		deviceID string,
		userPrincipalName string,
		mdmEnrolled bool,
		deviceName, osName, osVersion string,
		compliant bool,
		lastCheckInTime time.Time,
	) (*conditional_access_microsoft_proxy.SetComplianceStatusResponse, error) {
		setDeviceName = deviceName
		setOSName = osName
		setOSVersion = osVersion
		return &conditional_access_microsoft_proxy.SetComplianceStatusResponse{
			MessageID: "messageID",
		}, nil
	}

	setDone := make(chan struct{})

	mockedConditionalAccessMicrosoftProxyInstance.getMessageStatusFunc = func(
		ctx context.Context,
		tenantID string,
		secret string,
		messageID string,
	) (*conditional_access_microsoft_proxy.GetMessageStatusResponse, error) {
		close(setDone)
		return &conditional_access_microsoft_proxy.GetMessageStatusResponse{
			MessageID: "messageID",
			Status:    "Completed",
		}, nil
	}

	// Post policy results for the macOS host.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		h1,
		map[uint]*bool{
			cp1.ID: new(true),
			cp2.ID: new(true),
			p3.ID:  new(false),
		},
	), http.StatusOK, &distributedResp)

	select {
	case <-setDone:
		time.Sleep(1 * time.Second)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for compliance to be set")
	}

	h1s, err = s.ds.LoadHostConditionalAccessStatus(ctx, h1.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceID", h1s.DeviceID)
	require.Equal(t, "entraUserPrincipalName", h1s.UserPrincipalName)
	require.NotNil(t, h1s.Managed)
	require.False(t, *h1s.Managed) // not MDM enrolled
	require.NotNil(t, h1s.Compliant)
	require.True(t, *h1s.Compliant) // the two configured policies are passing

	// Check fields to set on Entra are correct.
	require.Equal(t, h1.Hostname, setDeviceName)
	require.Equal(t, "15.7.3", setOSVersion)
	require.Equal(t, "macOS", setOSName)

	// Post policy results for the windows host.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		windowsHost1,
		map[uint]*bool{
			cp1.ID: new(true),
			cp2.ID: new(true),
			p3.ID:  new(false),
		},
	), http.StatusOK, &distributedResp)

	time.Sleep(1 * time.Second)

	wh1s, err = s.ds.LoadHostConditionalAccessStatus(ctx, windowsHost1.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceIDWindows", wh1s.DeviceID)
	require.Equal(t, "", wh1s.UserPrincipalName)
	require.NotNil(t, wh1s.Managed)
	require.False(t, *wh1s.Managed) // not MDM enrolled
	require.NotNil(t, wh1s.Compliant)
	require.True(t, *wh1s.Compliant) // the two configured policies are passing

	// Check fields to set on Entra are correct.
	require.Equal(t, windowsHost1.Hostname, setDeviceName)
	require.Equal(t, "10.0.19045.6456", setOSVersion)
	require.Equal(t, "windows", setOSName)

	// Enroll macOS host to MDM to update managed status.
	err = s.ds.SetOrUpdateMDMData(ctx,
		h1.ID, false, true /* enrolled */, s.server.URL, false, /* installedFromDEP */
		"Fleet" /* MDM name */, "" /* fleetEnrollmentRef */, fleet.PersonalEnrollmentTypeNone, /* isPersonalEnrollment */
	)
	require.NoError(t, err)

	setDone = make(chan struct{})

	// Publish same policy results.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		h1,
		map[uint]*bool{
			cp1.ID: new(true),
			cp2.ID: new(true),
			p3.ID:  new(false),
		},
	), http.StatusOK, &distributedResp)

	select {
	case <-setDone:
		time.Sleep(1 * time.Second)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for compliance to be set")
	}

	h1s, err = s.ds.LoadHostConditionalAccessStatus(ctx, h1.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceID", h1s.DeviceID)
	require.Equal(t, "entraUserPrincipalName", h1s.UserPrincipalName)
	require.NotNil(t, h1s.Managed)
	require.True(t, *h1s.Managed) // now should be MDM enrolled
	require.NotNil(t, h1s.Compliant)
	require.True(t, *h1s.Compliant) // the two configured policies are passing

	// Enroll Windows host to MDM to update managed status.
	err = s.ds.SetOrUpdateMDMData(ctx,
		windowsHost1.ID, false, true /* enrolled */, s.server.URL, false, /* installedFromDEP */
		"Fleet" /* MDM name */, "" /* fleetEnrollmentRef */, fleet.PersonalEnrollmentTypeNone, /* isPersonalEnrollment */
	)
	require.NoError(t, err)

	// Publish same policy results.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		windowsHost1,
		map[uint]*bool{
			cp1.ID: new(true),
			cp2.ID: new(true),
			p3.ID:  new(false),
		},
	), http.StatusOK, &distributedResp)

	time.Sleep(1 * time.Second)

	wh1s, err = s.ds.LoadHostConditionalAccessStatus(ctx, windowsHost1.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceIDWindows", wh1s.DeviceID)
	require.Equal(t, "", wh1s.UserPrincipalName)
	require.NotNil(t, wh1s.Managed)
	require.True(t, *wh1s.Managed) // now should be MDM enrolled
	require.NotNil(t, wh1s.Compliant)
	require.True(t, *wh1s.Compliant) // the two configured policies are passing

	setDone = make(chan struct{})

	// Now the macOS host is failing a compliance policy.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		h1,
		map[uint]*bool{
			cp1.ID: new(true),
			cp2.ID: new(false),
			p3.ID:  new(false),
		},
	), http.StatusOK, &distributedResp)

	select {
	case <-setDone:
		time.Sleep(1 * time.Second)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for compliance to be set")
	}

	h1s, err = s.ds.LoadHostConditionalAccessStatus(ctx, h1.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceID", h1s.DeviceID)
	require.Equal(t, "entraUserPrincipalName", h1s.UserPrincipalName)
	require.NotNil(t, h1s.Managed)
	require.True(t, *h1s.Managed)
	require.NotNil(t, h1s.Compliant)
	require.False(t, *h1s.Compliant) // now the macOS host is non-compliant

	// Now the Windows host is failing a compliance policy.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		windowsHost1,
		map[uint]*bool{
			cp1.ID: new(true),
			cp2.ID: new(false),
			p3.ID:  new(false),
		},
	), http.StatusOK, &distributedResp)

	time.Sleep(1 * time.Second)

	wh1s, err = s.ds.LoadHostConditionalAccessStatus(ctx, windowsHost1.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceIDWindows", wh1s.DeviceID)
	require.Equal(t, "", wh1s.UserPrincipalName)
	require.NotNil(t, wh1s.Managed)
	require.True(t, *wh1s.Managed)
	require.NotNil(t, wh1s.Compliant)
	require.False(t, *wh1s.Compliant) // now the Windows host is non-compliant

	// Now nothing changes so there's no compliance operation on the proxy.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		h1,
		map[uint]*bool{
			cp1.ID: new(true),
			cp2.ID: new(false),
			p3.ID:  new(false),
		},
	), http.StatusOK, &distributedResp)

	time.Sleep(2 * time.Second)

	h1s, err = s.ds.LoadHostConditionalAccessStatus(ctx, h1.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceID", h1s.DeviceID)
	require.Equal(t, "entraUserPrincipalName", h1s.UserPrincipalName)
	require.NotNil(t, h1s.Managed)
	require.True(t, *h1s.Managed)
	require.NotNil(t, h1s.Compliant)
	require.False(t, *h1s.Compliant)

	// Now nothing changes on the Windows host so there's no compliance operation on the proxy.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		windowsHost1,
		map[uint]*bool{
			cp1.ID: new(true),
			cp2.ID: new(false),
			p3.ID:  new(false),
		},
	), http.StatusOK, &distributedResp)

	time.Sleep(5 * time.Second)

	wh1s, err = s.ds.LoadHostConditionalAccessStatus(ctx, windowsHost1.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceIDWindows", wh1s.DeviceID)
	require.Equal(t, "", wh1s.UserPrincipalName)
	require.NotNil(t, wh1s.Managed)
	require.True(t, *wh1s.Managed)
	require.NotNil(t, wh1s.Compliant)
	require.False(t, *wh1s.Compliant)

	//
	// Test B: host h2 in "No team".
	//

	h2 := newHost("h2", nil, "darwin", "macOS 26.3")
	orbitKey2 := setOrbitEnrollment(t, h2, s.ds)
	h2.OrbitNodeKey = &orbitKey2

	// "No team" configuration for conditional access is in global config.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"integrations": {
			"conditional_access_enabled": true
		}
	}`), http.StatusOK)
	// Test that by not setting it it's not disabled.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"integrations": {}
	}`), http.StatusOK)
	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.True(t, acResp.Integrations.ConditionalAccessEnabled.Set)
	require.True(t, acResp.Integrations.ConditionalAccessEnabled.Value)

	pr = fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", fleet.PolicyNoTeamID), fleet.TeamPolicyRequest{
		Query:                    "SELECT 1;",
		Name:                     "Compliance check 1",
		ConditionalAccessEnabled: true,
		Platform:                 "darwin,windows",
	}, http.StatusOK, &pr)
	cp1 = pr.Policy
	pr = fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", fleet.PolicyNoTeamID), fleet.TeamPolicyRequest{
		Query:                    "SELECT 2;",
		Name:                     "Other 1",
		ConditionalAccessEnabled: false,
	}, http.StatusOK, &pr)
	p2 := pr.Policy
	pr = fleet.TeamPolicyResponse{}

	// Enroll to MDM to update managed status.
	err = s.ds.SetOrUpdateMDMData(ctx,
		h2.ID, false, true /* enrolled */, s.server.URL, false, /* installedFromDEP */
		"Fleet" /* MDM name */, "" /* fleetEnrollmentRef */, fleet.PersonalEnrollmentTypeNone, /* isPersonalEnrollment */
	)
	require.NoError(t, err)

	// Ingest device ID and user principal name.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithEntraIDDetails(h2, "entraDeviceID2", "entraUserPrincipalName2"), http.StatusOK, &distributedResp)
	h2s, err := s.ds.LoadHostConditionalAccessStatus(ctx, h2.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceID2", h2s.DeviceID)
	require.Equal(t, "entraUserPrincipalName2", h2s.UserPrincipalName)
	require.Nil(t, h2s.Managed)
	require.Nil(t, h2s.Compliant)

	setDone = make(chan struct{})

	// Now the host is failing a the compliance policy.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		h2,
		map[uint]*bool{
			cp1.ID: new(false),
			p2.ID:  new(false),
		},
	), http.StatusOK, &distributedResp)

	select {
	case <-setDone:
		time.Sleep(1 * time.Second)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for compliance to be set")
	}

	h2s, err = s.ds.LoadHostConditionalAccessStatus(ctx, h2.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceID2", h2s.DeviceID)
	require.Equal(t, "entraUserPrincipalName2", h2s.UserPrincipalName)
	require.NotNil(t, h2s.Managed)
	require.True(t, *h2s.Managed)
	require.NotNil(t, h2s.Compliant)
	require.False(t, *h2s.Compliant) // host is non-compliant

	// Delete compliance policy, now there should be no compliance policies so host should be compliant.
	_, err = s.ds.DeleteTeamPolicies(ctx, fleet.PolicyNoTeamID, []uint{cp1.ID})
	require.NoError(t, err)

	setDone = make(chan struct{})

	// Now the host is failing a compliance policy.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		h2,
		map[uint]*bool{
			p2.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	select {
	case <-setDone:
		time.Sleep(1 * time.Second)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for compliance to be set")
	}

	h2s, err = s.ds.LoadHostConditionalAccessStatus(ctx, h2.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceID2", h2s.DeviceID)
	require.Equal(t, "entraUserPrincipalName2", h2s.UserPrincipalName)
	require.NotNil(t, h2s.Managed)
	require.True(t, *h2s.Managed)
	require.NotNil(t, h2s.Compliant)
	require.True(t, *h2s.Compliant) // now the host is compliant

	// A change of device ID and user principal name should update and clear the managed and compliant values.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithEntraIDDetails(h2, "entraDeviceID3", "entraUserPrincipalName3"), http.StatusOK, &distributedResp)
	h2s, err = s.ds.LoadHostConditionalAccessStatus(ctx, h2.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceID3", h2s.DeviceID)
	require.Equal(t, "entraUserPrincipalName3", h2s.UserPrincipalName)
	require.Nil(t, h2s.Managed)
	require.Nil(t, h2s.Compliant)
}

func (s *integrationEnterpriseTestSuite) TestConditionalAccessPoliciesEntraResultsProcessedFirst() {
	t := s.T()

	// Setup integration.
	mockedConditionalAccessMicrosoftProxyInstance.getResponse = &conditional_access_microsoft_proxy.GetResponse{
		TenantID:        testEntraTenantID,
		SetupDone:       false,
		AdminConsentURL: "https://example.com",
	}
	mockedConditionalAccessMicrosoftProxyInstance.createResponse = &conditional_access_microsoft_proxy.CreateResponse{
		TenantID: testEntraTenantID,
		Secret:   "secret",
	}
	var r conditionalAccessMicrosoftCreateResponse
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftCreateRequest{
		MicrosoftTenantID: testEntraTenantID,
	}, http.StatusOK, &r)
	mockedConditionalAccessMicrosoftProxyInstance.getResponse = &conditional_access_microsoft_proxy.GetResponse{
		TenantID:  testEntraTenantID,
		SetupDone: true,
	}
	var c conditionalAccessMicrosoftConfirmResponse
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft/confirm", conditionalAccessMicrosoftConfirmRequest{}, http.StatusOK, &c)
	require.True(t, c.ConfigurationCompleted)

	t1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	// Create two conditional access policies.
	var pr fleet.TeamPolicyResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", t1.ID), fleet.TeamPolicyRequest{
		Query:                    "SELECT 1;",
		Name:                     "Compliance check 1",
		ConditionalAccessEnabled: true,
		Platform:                 "darwin,windows",
	}, http.StatusOK, &pr)
	cp1 := pr.Policy
	pr = fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", t1.ID), fleet.TeamPolicyRequest{
		Query:                    "SELECT 2;",
		Name:                     "Compliance check 2",
		ConditionalAccessEnabled: true,
		Platform:                 "darwin,windows",
	}, http.StatusOK, &pr)
	cp2 := pr.Policy

	ctx := context.Background()
	newHost := func(name string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   new(t.Name() + name),
			NodeKey:         new(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        "darwin",
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}

	h1 := newHost("h1", &t1.ID)
	orbitKey := setOrbitEnrollment(t, h1, s.ds)
	h1.OrbitNodeKey = &orbitKey

	// Enable feature on team.
	var tmResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", t1.ID), map[string]any{
		"integrations": map[string]any{
			"conditional_access_enabled": true,
		},
	}, http.StatusOK, &tmResp)

	var (
		compliantValue *bool
		deviceIDValue  *string
	)
	// override value to reduce test time.
	conditionalAccessSetWaitTime = 250 * time.Millisecond
	mockedConditionalAccessMicrosoftProxyInstance.setComplianceStatusFunc = func(
		ctx context.Context,
		tenantID string, secret string,
		deviceID string,
		userPrincipalName string,
		mdmEnrolled bool,
		deviceName, osName, osVersion string,
		compliant bool,
		lastCheckInTime time.Time,
	) (*conditional_access_microsoft_proxy.SetComplianceStatusResponse, error) {
		deviceIDValue = &deviceID
		compliantValue = &compliant
		return &conditional_access_microsoft_proxy.SetComplianceStatusResponse{
			MessageID: "messageID",
		}, nil
	}

	setDone := make(chan struct{})

	mockedConditionalAccessMicrosoftProxyInstance.getMessageStatusFunc = func(
		ctx context.Context,
		tenantID string,
		secret string,
		messageID string,
	) (*conditional_access_microsoft_proxy.GetMessageStatusResponse, error) {
		close(setDone)
		return &conditional_access_microsoft_proxy.GetMessageStatusResponse{
			MessageID: "messageID",
			Status:    "Completed",
		}, nil
	}

	// Check that before the distributed/write there's no device information.
	_, err = s.ds.LoadHostConditionalAccessStatus(ctx, h1.ID)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	// Host sends policy results AND entra device information in the same distributed/write
	rawJSON := fmt.Sprintf(`{
		"node_key": "%[1]s",
		"queries": {
			"%[2]s": [{"1": "1"}],
			"%[6]s": [],
			"%[3]s": [{"device_id": "%[4]s", "user_principal_name": "%[5]s"}]
		},
		"messages": {},
		"statuses": {
			"%[2]s": 0,
			"%[6]s": 0,
			"%[3]s": 0
		}
	}`, *h1.NodeKey,
		hostPolicyQueryPrefix+fmt.Sprint(cp1.ID),
		hostDetailQueryPrefix+"conditional_access_microsoft_device_id", "entraDeviceID", "entraUserPrincipalName",
		hostPolicyQueryPrefix+fmt.Sprint(cp2.ID),
	)
	s.DoRaw("POST", "/api/osquery/distributed/write", []byte(rawJSON), http.StatusOK)

	// Test that the device ID was ingested prior to the policy evaluation
	// meaning that on the one single distributed write it will send the compliant request to Entra.
	select {
	case <-setDone:
		time.Sleep(1 * time.Second)
		require.NotNil(t, compliantValue)
		require.False(t, *compliantValue)
		require.NotNil(t, deviceIDValue)
		require.Equal(t, "entraDeviceID", *deviceIDValue)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for compliance to be set")
	}

	h1s, err := s.ds.LoadHostConditionalAccessStatus(ctx, h1.ID)
	require.NoError(t, err)
	require.Equal(t, "entraDeviceID", h1s.DeviceID)
	require.Equal(t, "entraUserPrincipalName", h1s.UserPrincipalName)
	require.NotNil(t, h1s.Managed)
	require.False(t, *h1s.Managed) // not MDM enrolled
	require.NotNil(t, h1s.Compliant)
	require.False(t, *h1s.Compliant) // one policy is failing
}

func (s *integrationEnterpriseTestSuite) TestAppConfigOktaConditionalAccess() {
	t := s.T()
	t.Cleanup(func() {
		s.clearOktaConditionalAccess()
	})

	// Helper function to build Okta conditional access payloads
	// Passing nil for a field will emit null in the JSON
	oktaPayload := func(idp, acs, aud, cert *string, bypass *bool) []byte {
		body := map[string]any{
			"conditional_access": map[string]any{
				"okta_idp_id":                         idp,
				"okta_assertion_consumer_service_url": acs,
				"okta_audience_uri":                   aud,
				"okta_certificate":                    cert,
				"bypass_disabled":                     bypass,
			},
		}
		b, err := json.Marshal(body)
		require.NoError(t, err)
		return b
	}

	validCert := testOktaValidCert

	// Test values
	idp := "https://www.okta.com/saml2/service-provider/spaubuaqdunfbsmoxyhl"
	acs := "https://dev-579.okta.com/sso/saml2/0oaqu0748bP2ELUlJ5d7"
	aud := "https://www.okta.com/saml2/service-provider/spaubuaqdunfbsmoxyhl"
	invalidURL := "not-a-valid-url"
	invalidCert := "not-a-valid-pem-certificate"
	trueBool := true
	falseBool := false

	// GET config should return empty Okta fields initially
	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	// ConditionalAccess should always be present; initially Okta fields should be empty/not valid
	require.NotNil(t, acResp.ConditionalAccess)
	require.False(t, acResp.ConditionalAccess.OktaIDPID.Valid)
	require.False(t, acResp.ConditionalAccess.OktaAssertionConsumerServiceURL.Valid)
	require.False(t, acResp.ConditionalAccess.OktaAudienceURI.Valid)
	require.False(t, acResp.ConditionalAccess.OktaCertificate.Valid)
	require.False(t, acResp.ConditionalAccess.BypassDisabled.Valid)

	// Test 1: Try to set only some Okta fields (should fail validation)
	s.DoRaw("PATCH", "/api/latest/fleet/config", oktaPayload(&idp, nil, nil, nil, nil), http.StatusUnprocessableEntity)

	// Test 2: Try to set 3 out of 4 fields (should fail validation)
	s.DoRaw("PATCH", "/api/latest/fleet/config", oktaPayload(&idp, &acs, &aud, nil, nil), http.StatusUnprocessableEntity)

	// Test 3: Set all 4 Okta fields with valid values (should succeed)
	s.DoRaw("PATCH", "/api/latest/fleet/config", oktaPayload(&idp, &acs, &aud, &validCert, &trueBool), http.StatusOK)

	// GET config should now return the Okta fields
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.ConditionalAccess.OktaIDPID.Valid)
	assert.Equal(t, idp, acResp.ConditionalAccess.OktaIDPID.Value)
	assert.Equal(t, acs, acResp.ConditionalAccess.OktaAssertionConsumerServiceURL.Value)
	assert.Equal(t, aud, acResp.ConditionalAccess.OktaAudienceURI.Value)
	assert.Equal(t, validCert, acResp.ConditionalAccess.OktaCertificate.Value)
	assert.True(t, acResp.ConditionalAccess.BypassDisabled.Valid)
	assert.True(t, acResp.ConditionalAccess.BypassDisabled.Value)

	// Verify activity was created for adding Okta config
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeAddedConditionalAccessOkta{}.ActivityName(),
		"",
		0,
	)

	// Test 4: Try to set invalid URL for assertion_consumer_service_url (should fail)
	s.DoRaw("PATCH", "/api/latest/fleet/config", oktaPayload(&idp, &invalidURL, &aud, &validCert, &falseBool), http.StatusUnprocessableEntity)

	// Test 5: Try to set invalid PEM certificate (should fail)
	s.DoRaw("PATCH", "/api/latest/fleet/config", oktaPayload(&idp, &acs, &aud, &invalidCert, &falseBool), http.StatusUnprocessableEntity)

	// Test 6: Clear all Okta configuration by setting all fields to null
	// First verify Okta is currently configured (from Test 3)
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.ConditionalAccess.OktaIDPID.Valid, "Okta should be configured before Test 6")

	// Now clear it
	s.DoRaw("PATCH", "/api/latest/fleet/config", oktaPayload(nil, nil, nil, nil, nil), http.StatusOK)

	// Verify all fields are now null/empty
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	// ConditionalAccess should always be present; after clearing, Okta fields should not be valid
	require.NotNil(t, acResp.ConditionalAccess)
	assert.False(t, acResp.ConditionalAccess.OktaIDPID.Valid)
	assert.False(t, acResp.ConditionalAccess.OktaAssertionConsumerServiceURL.Valid)
	assert.False(t, acResp.ConditionalAccess.OktaAudienceURI.Valid)
	assert.False(t, acResp.ConditionalAccess.OktaCertificate.Valid)
	assert.False(t, acResp.ConditionalAccess.BypassDisabled.Valid)

	// Verify activity was created for deleting Okta config
	// Note: Using lastActivityOfTypeMatches which searches the last 10 activities
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDeletedConditionalAccessOkta{}.ActivityName(),
		"", // no details for this activity type
		0,  // don't check specific ID
	)

	// Test 7: Verify an unrelated config change doesn't affect Okta settings when Okta is configured
	// First, set up Okta config again
	s.DoRaw("PATCH", "/api/latest/fleet/config", oktaPayload(&idp, &acs, &aud, &validCert, &trueBool), http.StatusOK)

	// Make an unrelated change (e.g., org_name)
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"org_info": {
			"org_name": "test-okta-config"
		}
	}`), http.StatusOK, &acResp)
	require.Equal(t, "test-okta-config", acResp.OrgInfo.OrgName)

	// Verify Okta settings are still there
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Equal(t, idp, acResp.ConditionalAccess.OktaIDPID.Value)
	assert.Equal(t, acs, acResp.ConditionalAccess.OktaAssertionConsumerServiceURL.Value)
	assert.Equal(t, aud, acResp.ConditionalAccess.OktaAudienceURI.Value)
	assert.Equal(t, validCert, acResp.ConditionalAccess.OktaCertificate.Value)

	// Test 8: Verify activity is created when editing Okta config
	// Change one of the fields (e.g., the IDP ID)
	newIdp := "https://www.okta.com/saml2/service-provider/newvalue123"
	s.DoRaw("PATCH", "/api/latest/fleet/config", oktaPayload(&newIdp, &acs, &aud, &validCert, &trueBool), http.StatusOK)

	// Verify the field was changed
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Equal(t, newIdp, acResp.ConditionalAccess.OktaIDPID.Value)

	// Verify activity was created for editing Okta config
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeAddedConditionalAccessOkta{}.ActivityName(),
		"",
		0,
	)

	// Test 9: Set configuration with multiple certificate blocks
	// This tests that the validation logic properly handles multiple PEM-encoded certificates
	// Use a second different certificate to simulate a real-world scenario (e.g., certificate chain)
	secondValidCert := `-----BEGIN CERTIFICATE-----
MIIGHzCCBAegAwIBAgIBATANBgkqhkiG9w0BAQsFADBSMQswCQYDVQQGEwJVUzEL
MAkGA1UECBMCTlkxETAPBgNVBAcTCE5ldyBZb3JrMRAwDgYDVQQKEwdFeGFtcGxl
MREwDwYDVQQDEwhncm9vYi1jYTAeFw0xNjEwMjQxMzExMTdaFw0yNjEwMjIxMzEx
MTdaMFIxETAPBgNVBAMTCGdyb29iLWNhMQswCQYDVQQGEwJVUzERMA8GA1UEBxMI
TmV3IFlvcmsxEDAOBgNVBAoTB0V4YW1wbGUxCzAJBgNVBAgTAk5ZMIICIjANBgkq
hkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAuU8t6vcXvlqW1/zGW1qArwFBRySBLvlD
jkb5I9GduARzokAipsfzJeSor9/e+TPd4pYD0m70q7uKOYxmEB42vtcYQVcWPeUt
6r1PT9PVGpJcq7ZNcKRmbLbddnesQCGZadxoDu8m1p/QRMA5BMEL7Hbly9jeYNo+
nLa1vKMyakxEj8nzVtwrHcwCrtC9BrYzui6HdPKZcB6Vnuf7edrjEQCgeOuLq0Hq
f9/6CtIsxJ6ynewjciSKj8kqsxmCfoV02D1xAJm75ABA26gIyamB3yY886w5LhJa
CGPHoJ4821FvnYd3slH2a/et5TBqxtsb/H1WDcRkXsrwYKyei+ANarYyQ6ZYKrjS
LXLBMSJiySEUYE33bCRjR196ggYhRTh5sZ7bf/uAA9ny5Dm5oaEy0e8NPVOCKkSE
kQ+n+gTiY0BZ4gpX2uYgqNsJ7KOk6V1ZAAc1tEXypxByPjglbbhHSWqCNWkLbZ27
17g5h9fOquh8rAUZaZJ+UWtGMB/ncFvpqfgS5ATAFUQ432Pd4yVHLNFkv6Wf6+WY
9ncp2BBcEhA3cfpNYBE1vv9tW5kf61/y/VHcQZKzWRLkDYgiw5TK5KSj4FHLID9W
UcwYYRVjJkXSEw1CUkZ6D9KfBktdPDLSAvvLB0TcexOrOMH8zCbgJe/NdPb4RVjq
Pke400JgEeMCAwEAAaOB/zCB/DCBggYDVR0jBHsweYAUJ8IQqDs7T5s8sf/2EsXx
O5f5JeShVqRUMFIxCzAJBgNVBAYTAlVTMQswCQYDVQQIEwJOWTERMA8GA1UEBxMI
TmV3IFlvcmsxEDAOBgNVBAoTB0V4YW1wbGUxETAPBgNVBAMTCGdyb29iLWNhggkA
yt8kNI+5nUMwEgYDVR0TAQH/BAgwBgEB/wIBADAdBgNVHSUEFjAUBggrBgEFBQcD
AgYIKwYBBQUHAwEwDgYDVR0PAQH/BAQDAgFGMBMGA1UdEQQMMAqCCGdyb29iLWNh
MB0GA1UdDgQWBBQkiMZe6VX0KyMynOOqrzBV1E/IozANBgkqhkiG9w0BAQsFAAOC
AgEAiFWITGT9vMvHXjKDfUarul7nP/hGC6nCOJLmZMl3sTTv2DpLvcXAia9WipTL
aXcID/ZAxTVKtRWpTJowEFQDkGAN3Qcf+apyJF3+ZPhoH1Ma+ZfnF0dl3qF7pr5+
I8NWCAzb8p+8g0nlHMrHONDT5yUdzwt/Dp9OFMENTh0hPcEuOc0IVePQ+yNLrSgI
d++nOcz9iEHP78QWXwsbduA9e+1aWKICxyFsM+HmpYKc/ehggEoPq5sY/DPZ/QIG
WNTkmgrlds77D17kj8D1g1EtZ6aZmxLyWFJhuZNMQLHgGK44qupANevPm8VM4tFj
4qGPqTL4iqH9KAEUsXQOWrobQiMzp/VGRqNZ0ttoA82h/xPe8Px84h0wyneTkT8D
oCHn/NS8U7VPYij1Ch2FY+4ZmrhyIw90fTs7QDP7PCv27xM1ZUyhfDQ7pdiwPbR9
kPTw3me7EBR2si8lx4f6O6yDavZVRVh9zI9ZBzA9HQIp4ayLO58Tpm2DTJ1cxVot
UFqwfMz6SBdChsGzMRcl4fop6u99TEZIXbqXYdDE5P3eTl6wLx6yxLazkREdLn98
mH6v0Wtbra/Ck/QjLbGj3zg/PGfpDiMwXFRCwTn+YjUFmNN/XYyfjqEaR7TbJ3H+
qcznMoapfGAjRwaheTlWbzyUh57ToALyx3xQbzqYIxiQCzY=
-----END CERTIFICATE-----`
	validCertMultiple := validCert + "\n" + secondValidCert
	s.DoRaw("PATCH", "/api/latest/fleet/config", oktaPayload(&idp, &acs, &aud, &validCertMultiple, &trueBool), http.StatusOK)

	// Verify the configuration is saved with multiple certificates
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.ConditionalAccess.OktaCertificate.Valid)
	assert.Equal(t, validCertMultiple, acResp.ConditionalAccess.OktaCertificate.Value)
	require.True(t, acResp.ConditionalAccess.BypassDisabled.Valid)
	assert.True(t, acResp.ConditionalAccess.BypassDisabled.Value)

	// Change only if bypass is disabled
	s.DoRaw("PATCH", "/api/latest/fleet/config", oktaPayload(&idp, &acs, &aud, &validCertMultiple, &falseBool), http.StatusOK)

	// Verify the was changed
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.ConditionalAccess.BypassDisabled.Valid)
	assert.False(t, acResp.ConditionalAccess.BypassDisabled.Value)

	// Verify activity was created for editing Okta config
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeUpdateConditionalAccessBypass{}.ActivityName(),
		"{\"bypass_disabled\": false}",
		0,
	)

	// Change only if bypass is disabled
	s.DoRaw("PATCH", "/api/latest/fleet/config", oktaPayload(&idp, &acs, &aud, &validCertMultiple, &trueBool), http.StatusOK)

	// Verify the was changed
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.ConditionalAccess.BypassDisabled.Valid)
	assert.True(t, acResp.ConditionalAccess.BypassDisabled.Value)

	// Verify activity was created for editing Okta config
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeUpdateConditionalAccessBypass{}.ActivityName(),
		"{\"bypass_disabled\": true}",
		0,
	)
}

// TestGitOpsModeToggleDoesNotCreateConditionalAccessActivity verifies that
// toggling GitOps mode while Okta conditional access is already configured
// (with bypass_disabled=true) does not generate a spurious
// update_conditional_access_bypass activity. The PATCH payload only touches
// the gitops block, so the conditional-access bypass setting is unchanged.
func (s *integrationEnterpriseTestSuite) TestGitOpsModeToggleDoesNotCreateConditionalAccessActivity() {
	t := s.T()
	ctx := t.Context()
	t.Cleanup(func() {
		s.clearOktaConditionalAccess()
		// Also reset gitops config to a clean state.
		s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"gitops": { "gitops_mode_enabled": false, "repository_url": "" }
		}`), http.StatusOK)
	})

	validCert := testOktaValidCert

	// Configure Okta conditional access with bypass_disabled=true. This puts
	// the appconfig in the state required to trigger the buggy code path.
	oktaBody := map[string]any{
		"conditional_access": map[string]any{
			"okta_idp_id":                         "https://www.okta.com/saml2/service-provider/spaubuaqdunfbsmoxyhl",
			"okta_assertion_consumer_service_url": "https://dev-579.okta.com/sso/saml2/0oaqu0748bP2ELUlJ5d7",
			"okta_audience_uri":                   "https://www.okta.com/saml2/service-provider/spaubuaqdunfbsmoxyhl",
			"okta_certificate":                    validCert,
			"bypass_disabled":                     true,
		},
	}
	oktaBodyBytes, err := json.Marshal(oktaBody)
	require.NoError(t, err)
	s.DoRaw("PATCH", "/api/latest/fleet/config", oktaBodyBytes, http.StatusOK)

	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.ConditionalAccess.BypassDisabled.Valid)
	require.True(t, acResp.ConditionalAccess.BypassDisabled.Value, "bypass_disabled must be true to reproduce the bug")

	// Seed a host_conditional_access row. The buggy branch in ModifyAppConfig
	// calls ConditionalAccessClearBypasses, which deletes every row in this
	// table — so if the bug fires, this row disappears.
	host := createHostAndDeviceToken(t, s.ds, fmt.Sprintf("bypass-row-%s", uuid.New().String()))
	require.NoError(t, s.ds.ConditionalAccessBypassDevice(ctx, host.ID))

	// Toggle GitOps mode ON. The PATCH body intentionally omits
	// conditional_access — this is exactly what the Change management page
	// sends (frontend/pages/admin/IntegrationsPage/cards/ChangeManagement).
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"gitops": { "gitops_mode_enabled": true, "repository_url": "https://example.com/repo" }
	}`), http.StatusOK)

	// The seeded bypass row must survive the toggle — if the buggy branch
	// ran it would have called ConditionalAccessClearBypasses.
	bypassedAt, err := s.ds.ConditionalAccessConsumeBypass(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, bypassedAt, "host_conditional_access row was cleared by an unrelated gitops PATCH")

	// The most recent activity must be enabled_gitops_mode — NOT
	// update_conditional_access_bypass. The buggy code emits the bypass
	// activity after the gitops one, so checking only the last activity
	// is sufficient.
	s.lastActivityMatches(fleet.ActivityTypeEnabledGitOpsMode{}.ActivityName(), "", 0)

	// Re-seed the bypass row and toggle GitOps mode OFF. Same assertions.
	require.NoError(t, s.ds.ConditionalAccessBypassDevice(ctx, host.ID))
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"gitops": { "gitops_mode_enabled": false, "repository_url": "https://example.com/repo" }
	}`), http.StatusOK)

	bypassedAt, err = s.ds.ConditionalAccessConsumeBypass(ctx, host.ID)
	require.NoError(t, err)
	require.NotNil(t, bypassedAt, "host_conditional_access row was cleared by an unrelated gitops PATCH")

	s.lastActivityMatches(fleet.ActivityTypeDisabledGitOpsMode{}.ActivityName(), "", 0)

	// Sanity check: a real change to bypass_disabled still emits the activity.
	flipBody := map[string]any{
		"conditional_access": map[string]any{
			"okta_idp_id":                         "https://www.okta.com/saml2/service-provider/spaubuaqdunfbsmoxyhl",
			"okta_assertion_consumer_service_url": "https://dev-579.okta.com/sso/saml2/0oaqu0748bP2ELUlJ5d7",
			"okta_audience_uri":                   "https://www.okta.com/saml2/service-provider/spaubuaqdunfbsmoxyhl",
			"okta_certificate":                    validCert,
			"bypass_disabled":                     false,
		},
	}
	flipBodyBytes, err := json.Marshal(flipBody)
	require.NoError(t, err)
	s.DoRaw("PATCH", "/api/latest/fleet/config", flipBodyBytes, http.StatusOK)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeUpdateConditionalAccessBypass{}.ActivityName(),
		`{"bypass_disabled": false}`,
		0,
	)
}

func (s *integrationEnterpriseTestSuite) TestConditionalAccessBypass() {
	t := s.T()
	ctx := t.Context()

	validCert := testOktaValidCert

	t.Run("bypass succeeds when enabled", func(t *testing.T) {
		// Create a unique host for this subtest
		token := fmt.Sprintf("bypass-enabled-%s", uuid.New().String())
		host := createHostAndDeviceToken(t, s.ds, token)

		var bypassResp bypassConditionalAccessResponse
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token),
			nil, http.StatusOK, &bypassResp)
		require.Nil(t, bypassResp.Err)

		// Verify the bypass was recorded
		bypassedAt, err := s.ds.ConditionalAccessConsumeBypass(ctx, host.ID)
		require.NoError(t, err)
		require.NotNil(t, bypassedAt, "bypass should have been recorded")
	})

	t.Run("activity created with default idp_full_name", func(t *testing.T) {
		// Create a unique host for this subtest
		token := fmt.Sprintf("activity-default-%s", uuid.New().String())
		host := createHostAndDeviceToken(t, s.ds, token)

		var bypassResp bypassConditionalAccessResponse
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token),
			nil, http.StatusOK, &bypassResp)
		require.Nil(t, bypassResp.Err)

		// Verify the activity was created with default idp_full_name when no SCIM user exists
		s.lastActivityOfTypeMatches(
			fleet.ActivityTypeHostBypassedConditionalAccess{}.ActivityName(),
			fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "idp_full_name": "An end user"}`, host.ID, host.DisplayName()),
			0,
		)
	})

	t.Run("activity includes actual IDP name when SCIM user present", func(t *testing.T) {
		// Create a host with SCIM user
		token := fmt.Sprintf("activity-scim-%s", uuid.New().String())
		host, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   new(token),
			NodeKey:         new(token),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s-scim.local", token),
			HardwareSerial:  uuid.New().String(),
			Platform:        "darwin",
		})
		require.NoError(t, err)
		createDeviceTokenForHost(t, s.ds, host.ID, token)

		// Create a SCIM user and map it to the host
		var scimUserID int64
		mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
			res, err := db.ExecContext(ctx,
				`INSERT INTO scim_users (user_name, given_name, family_name, active) VALUES (?, ?, ?, ?)`,
				"john.doe@example.com", "John", "Doe", 1)
			if err != nil {
				return err
			}
			scimUserID, err = res.LastInsertId()
			return err
		})
		mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(ctx,
				`INSERT INTO host_scim_user (host_id, scim_user_id) VALUES (?, ?)`,
				host.ID, scimUserID)
			return err
		})

		// Call bypass for host with SCIM user
		s.DoRawNoAuth("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token),
			nil, http.StatusOK)

		// Verify the activity includes the actual IDP full name
		s.lastActivityOfTypeMatches(
			fleet.ActivityTypeHostBypassedConditionalAccess{}.ActivityName(),
			fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "idp_full_name": "John Doe"}`, host.ID, host.DisplayName()),
			0,
		)
	})

	t.Run("calling bypass again creates new bypass", func(t *testing.T) {
		// Create a unique host for this subtest
		token := fmt.Sprintf("bypass-again-%s", uuid.New().String())
		host := createHostAndDeviceToken(t, s.ds, token)

		// First bypass
		var bypassResp bypassConditionalAccessResponse
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token),
			nil, http.StatusOK, &bypassResp)
		require.Nil(t, bypassResp.Err)

		// Consume the first bypass
		bypassedAt, err := s.ds.ConditionalAccessConsumeBypass(ctx, host.ID)
		require.NoError(t, err)
		require.NotNil(t, bypassedAt)

		// Call bypass again - should work (creates a new bypass)
		bypassResp = bypassConditionalAccessResponse{}
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token),
			nil, http.StatusOK, &bypassResp)
		require.Nil(t, bypassResp.Err)

		// Verify the new bypass was recorded
		bypassedAt, err = s.ds.ConditionalAccessConsumeBypass(ctx, host.ID)
		require.NoError(t, err)
		require.NotNil(t, bypassedAt, "new bypass should have been recorded")
	})

	t.Run("bypass fails when disabled", func(t *testing.T) {
		// Create a unique host for this subtest
		token := fmt.Sprintf("bypass-disabled-%s", uuid.New().String())
		_ = createHostAndDeviceToken(t, s.ds, token)

		// Disable bypass via app config
		var acResp appConfigResponse
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"conditional_access": {
				"bypass_disabled": true
			}
		}`), http.StatusOK, &acResp)

		t.Cleanup(func() {
			// Re-enable bypass for other tests
			s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
				"conditional_access": {
					"bypass_disabled": false
				}
			}`), http.StatusOK, &acResp)
		})

		// Verify bypass is now disabled
		acResp = appConfigResponse{}
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
		require.True(t, acResp.ConditionalAccess.BypassDisabled.Valid)
		require.True(t, acResp.ConditionalAccess.BypassDisabled.Value)

		// Bypass should fail when bypass is disabled (403 Forbidden)
		var bypassResp bypassConditionalAccessResponse
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token),
			nil, http.StatusForbidden, &bypassResp)
	})

	t.Run("bypass succeeds after re-enabling", func(t *testing.T) {
		// Create a unique host for this subtest
		token := fmt.Sprintf("bypass-reenable-%s", uuid.New().String())
		host := createHostAndDeviceToken(t, s.ds, token)

		t.Cleanup(func() {
			_, _ = s.ds.ConditionalAccessConsumeBypass(ctx, host.ID)
			// Ensure bypass is enabled for other tests
			var acResp appConfigResponse
			s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
				"conditional_access": {
					"bypass_disabled": false
				}
			}`), http.StatusOK, &acResp)
		})

		// Disable bypass
		var acResp appConfigResponse
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"conditional_access": {
				"bypass_disabled": true
			}
		}`), http.StatusOK, &acResp)

		// Verify it's disabled
		var bypassResp bypassConditionalAccessResponse
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token),
			nil, http.StatusForbidden, &bypassResp)

		// Re-enable bypass
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"conditional_access": {
				"bypass_disabled": false
			}
		}`), http.StatusOK, &acResp)

		// Bypass should succeed after re-enabling
		bypassResp = bypassConditionalAccessResponse{}
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token),
			nil, http.StatusOK, &bypassResp)
		require.Nil(t, bypassResp.Err)

		// Verify the bypass was recorded
		bypassedAt, err := s.ds.ConditionalAccessConsumeBypass(ctx, host.ID)
		require.NoError(t, err)
		require.NotNil(t, bypassedAt, "bypass should have been recorded after re-enabling")
	})

	t.Run("toggling bypass_disabled clears all bypasses", func(t *testing.T) {
		// Set up Okta configuration (required for bypass toggle detection)
		var acResp appConfigResponse
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
			"conditional_access": {
				"okta_idp_id": "https://www.okta.com/saml2/service-provider/test",
				"okta_assertion_consumer_service_url": "https://dev-test.okta.com/sso/saml2/test",
				"okta_audience_uri": "https://www.okta.com/saml2/service-provider/test",
				"okta_certificate": %q,
				"bypass_disabled": false
			}
		}`, validCert)), http.StatusOK, &acResp)

		// Create two hosts with unique tokens
		token1 := fmt.Sprintf("bypass-clear-1-%s", uuid.New().String())
		token2 := fmt.Sprintf("bypass-clear-2-%s", uuid.New().String())
		host1 := createHostAndDeviceToken(t, s.ds, token1)
		host2, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   new(token2),
			NodeKey:         new(token2),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.local", token2),
			HardwareSerial:  uuid.New().String(),
			Platform:        "darwin",
		})
		require.NoError(t, err)
		createDeviceTokenForHost(t, s.ds, host2.ID, token2)

		// Create bypass records for both hosts
		var bypassResp bypassConditionalAccessResponse
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token1),
			nil, http.StatusOK, &bypassResp)
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token2),
			nil, http.StatusOK, &bypassResp)

		// Verify both bypasses exist by checking the count directly
		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, db, &count, "SELECT COUNT(*) FROM host_conditional_access WHERE host_id IN (?, ?)", host1.ID, host2.ID)
		})
		require.GreaterOrEqual(t, count, 2, "at least both bypass records should exist")

		// Toggle bypass_disabled to true - this should clear all bypasses
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"conditional_access": {
				"bypass_disabled": true
			}
		}`), http.StatusOK, &acResp)

		// Verify all bypasses were cleared
		mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, db, &count, "SELECT COUNT(*) FROM host_conditional_access WHERE host_id IN (?, ?)", host1.ID, host2.ID)
		})
		require.Equal(t, 0, count, "all bypass records should be cleared when bypass_disabled is toggled")

		// Toggle back to false - should also clear any bypasses (though there are none)
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"conditional_access": {
				"bypass_disabled": false
			}
		}`), http.StatusOK, &acResp)
	})

	t.Run("ConditionalAccessBypassedAt does not consume bypass", func(t *testing.T) {
		token := fmt.Sprintf("bypassed-at-%s", uuid.New().String())
		host := createHostAndDeviceToken(t, s.ds, token)

		bypassedAtBefore, err := s.ds.ConditionalAccessBypassedAt(ctx, host.ID)
		require.NoError(t, err)
		require.Nil(t, bypassedAtBefore)

		var bypassResp bypassConditionalAccessResponse
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token),
			nil, http.StatusOK, &bypassResp)
		require.Nil(t, bypassResp.Err)

		bypassedAtAfter, err := s.ds.ConditionalAccessBypassedAt(ctx, host.ID)
		require.NoError(t, err)
		require.NotNil(t, bypassedAtAfter)

		bypassedAtAgain, err := s.ds.ConditionalAccessBypassedAt(ctx, host.ID)
		require.NoError(t, err)
		require.NotNil(t, bypassedAtAgain)
		require.Equal(t, *bypassedAtAfter, *bypassedAtAgain)

		consumedBypass, err := s.ds.ConditionalAccessConsumeBypass(ctx, host.ID)
		require.NoError(t, err)
		require.NotNil(t, consumedBypass)
		require.Equal(t, *bypassedAtAfter, *consumedBypass)

		bypassedAtAfterConsume, err := s.ds.ConditionalAccessBypassedAt(ctx, host.ID)
		require.NoError(t, err)
		require.Nil(t, bypassedAtAfterConsume)
	})

	t.Run("device host endpoint returns correct bypass state", func(t *testing.T) {
		token := fmt.Sprintf("device-endpoint-bypass-%s", uuid.New().String())
		host := createHostAndDeviceToken(t, s.ds, token)

		getDeviceHostResp := getDeviceHostResponse{}
		res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK)
		err := json.NewDecoder(res.Body).Decode(&getDeviceHostResp)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDeviceHostResp.Err)
		require.Equal(t, host.ID, getDeviceHostResp.Host.ID)
		require.False(t, getDeviceHostResp.Host.ConditionalAccessBypassed)

		var bypassResp bypassConditionalAccessResponse
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token),
			nil, http.StatusOK, &bypassResp)
		require.Nil(t, bypassResp.Err)

		getDeviceHostResp = getDeviceHostResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK)
		err = json.NewDecoder(res.Body).Decode(&getDeviceHostResp)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDeviceHostResp.Err)
		require.True(t, getDeviceHostResp.Host.ConditionalAccessBypassed)

		_, err = s.ds.ConditionalAccessConsumeBypass(ctx, host.ID)
		require.NoError(t, err)

		getDeviceHostResp = getDeviceHostResponse{}
		res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK)
		err = json.NewDecoder(res.Body).Decode(&getDeviceHostResp)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
		require.NoError(t, getDeviceHostResp.Err)
		require.False(t, getDeviceHostResp.Host.ConditionalAccessBypassed)
	})

	t.Run("bypass fails when host has failing critical policy", func(t *testing.T) {
		token := fmt.Sprintf("bypass-critical-%s", uuid.New().String())
		host := createHostAndDeviceToken(t, s.ds, token)

		// Assign host to a team
		team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: fmt.Sprintf("ca-blocking-team-%s", uuid.New().String())})
		require.NoError(t, err)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

		adminUser := s.users["admin1@example.com"]
		policy, err := s.ds.NewTeamPolicy(ctx, team.ID, &adminUser.ID, fleet.PolicyPayload{
			Name:                     fmt.Sprintf("ca-critical-%s", uuid.New().String()),
			Query:                    "select 1;",
			Critical:                 true,
			ConditionalAccessEnabled: true,
		})
		require.NoError(t, err)

		// Record a failing result for this policy on the host
		_, err = s.ds.RecordPolicyQueryExecutions(ctx, host, map[uint]*bool{policy.ID: new(false)}, time.Now(), false, nil)
		require.NoError(t, err)

		// Bypass should fail with 400 Bad Request
		var bypassResp bypassConditionalAccessResponse
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token),
			nil, http.StatusBadRequest, &bypassResp)

		// Verify no bypass row was created
		bypassedAt, err := s.ds.ConditionalAccessBypassedAt(ctx, host.ID)
		require.NoError(t, err)
		require.Nil(t, bypassedAt)
	})

	t.Run("bypass allowed when only non-CA critical policy is failing", func(t *testing.T) {
		token := fmt.Sprintf("bypass-non-ca-%s", uuid.New().String())
		host := createHostAndDeviceToken(t, s.ds, token)

		// Assign host to a team
		team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: fmt.Sprintf("non-ca-team-%s", uuid.New().String())})
		require.NoError(t, err)
		require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

		adminUser := s.users["admin1@example.com"]

		// CA policy — passing
		caPolicy, err := s.ds.NewTeamPolicy(ctx, team.ID, &adminUser.ID, fleet.PolicyPayload{
			Name:                     fmt.Sprintf("ca-policy-%s", uuid.New().String()),
			Query:                    "select 1;",
			Critical:                 true,
			ConditionalAccessEnabled: true,
		})
		require.NoError(t, err)

		// Non-CA critical policy — failing
		nonCAPolicy, err := s.ds.NewTeamPolicy(ctx, team.ID, &adminUser.ID, fleet.PolicyPayload{
			Name:     fmt.Sprintf("non-ca-policy-%s", uuid.New().String()),
			Query:    "select 1;",
			Critical: true,
		})
		require.NoError(t, err)

		_, err = s.ds.RecordPolicyQueryExecutions(ctx, host, map[uint]*bool{
			caPolicy.ID:    new(true),  // passing
			nonCAPolicy.ID: new(false), // failing
		}, time.Now(), false, nil)
		require.NoError(t, err)

		// Bypass must succeed: the only failing policy is not CA-enabled
		var bypassResp bypassConditionalAccessResponse
		s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/device/%s/bypass_conditional_access", token),
			nil, http.StatusOK, &bypassResp)
		require.Nil(t, bypassResp.Err)

		bypassedAt, err := s.ds.ConditionalAccessConsumeBypass(ctx, host.ID)
		require.NoError(t, err)
		require.NotNil(t, bypassedAt)
	})
}

func (s *integrationEnterpriseTestSuite) TestConditionalAccessPlatformValidation() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team_ca_platform_" + t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	//
	// 1. Create team policy — invalid platforms
	//

	// conditional_access_enabled=true with platform="linux" should fail
	res := s.Do("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), map[string]any{
		"name":                       "ca-linux",
		"query":                      "SELECT 1;",
		"platform":                   "linux",
		"conditional_access_enabled": true,
	}, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `"conditional_access_enabled" is only valid on "darwin" and "windows" policies`)
	res.Body.Close()

	// conditional_access_enabled=true with empty platform (all platforms) should fail
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), map[string]any{
		"name":                       "ca-all-platforms",
		"query":                      "SELECT 1;",
		"platform":                   "",
		"conditional_access_enabled": true,
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `"conditional_access_enabled" is only valid on "darwin" and "windows" policies`)
	res.Body.Close()

	// conditional_access_enabled=true with platform="darwin" should succeed
	var pr fleet.TeamPolicyResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), fleet.TeamPolicyRequest{
		Name:                     "ca-darwin",
		Query:                    "SELECT 1;",
		Platform:                 "darwin",
		ConditionalAccessEnabled: true,
	}, http.StatusOK, &pr)
	darwinPolicyID := pr.Policy.ID

	// conditional_access_enabled=true with platform="windows" should succeed
	pr = fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), fleet.TeamPolicyRequest{
		Name:                     "ca-windows",
		Query:                    "SELECT 2;",
		Platform:                 "windows",
		ConditionalAccessEnabled: true,
	}, http.StatusOK, &pr)
	windowsPolicyID := pr.Policy.ID

	// conditional_access_enabled=true with platform="darwin,windows" should succeed
	pr = fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), fleet.TeamPolicyRequest{
		Name:                     "ca-darwin-windows",
		Query:                    "SELECT 3;",
		Platform:                 "darwin,windows",
		ConditionalAccessEnabled: true,
	}, http.StatusOK, &pr)

	// conditional_access_enabled=false with platform="linux" should succeed (no restriction when disabled)
	pr = fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), fleet.TeamPolicyRequest{
		Name:                     "no-ca-linux",
		Query:                    "SELECT 4;",
		Platform:                 "linux",
		ConditionalAccessEnabled: false,
	}, http.StatusOK, &pr)
	linuxPolicyID := pr.Policy.ID

	//
	// 2. Modify team policy — platform validation on update
	//

	// Modify a darwin policy to enable conditional_access_enabled — should succeed (already darwin)
	patchResp := &fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, darwinPolicyID), fleet.ModifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			ConditionalAccessEnabled: new(true),
		},
	}, http.StatusOK, patchResp)

	// Modify a linux policy to enable conditional_access_enabled — should fail
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, linuxPolicyID), map[string]any{
		"conditional_access_enabled": true,
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `"conditional_access_enabled" is only valid on "darwin" and "windows" policies`)
	res.Body.Close()

	// Modify a darwin conditional_access policy to change platform to linux — should fail
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, darwinPolicyID), map[string]any{
		"platform": "linux",
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `"conditional_access_enabled" is only valid on "darwin" and "windows" policies`)
	res.Body.Close()

	// Modify a windows conditional_access policy to change platform to darwin — should succeed
	patchResp = &fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, windowsPolicyID), fleet.ModifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Platform: new("darwin"),
		},
	}, http.StatusOK, patchResp)

	//
	// 3. Modify global policy — conditional_access_enabled not allowed
	//

	// Create a global policy
	createGlobal := &fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", &fleet.GlobalPolicyRequest{
		Name:     "global-policy-ca-test",
		Query:    "SELECT 1;",
		Platform: "darwin",
	}, http.StatusOK, createGlobal)
	globalPolicyID := createGlobal.Policy.ID

	// Try to enable conditional_access_enabled on a global policy — should fail
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", globalPolicyID), map[string]any{
		"conditional_access_enabled": true,
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `"All fleets" policy cannot have conditional_access_enabled set`)
	res.Body.Close()

	//
	// 4. Apply policy specs — platform validation
	//

	// Team spec with conditional_access_enabled=true and platform="linux" should fail
	applyResp := fleet.ApplyPolicySpecsResponse{}
	res = s.Do("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{
			{
				Name:                     "spec-ca-linux",
				Query:                    "SELECT 1;",
				Team:                     team1.Name,
				Platform:                 "linux",
				ConditionalAccessEnabled: true,
				Type:                     fleet.PolicyTypeDynamic,
			},
		},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `"conditional_access_enabled" is only valid on "darwin" and "windows" policies`)
	res.Body.Close()

	// Global spec (no team) with conditional_access_enabled=true should fail
	res = s.Do("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{
			{
				Name:                     "spec-ca-global",
				Query:                    "SELECT 1;",
				Platform:                 "darwin",
				ConditionalAccessEnabled: true,
				Type:                     fleet.PolicyTypeDynamic,
			},
		},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `"All fleets" policy cannot have conditional_access_enabled set`)
	res.Body.Close()

	// Team spec with conditional_access_enabled=true and platform="darwin" should succeed
	s.DoJSON("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{
			{
				Name:                     "spec-ca-darwin",
				Query:                    "SELECT 1;",
				Team:                     team1.Name,
				Platform:                 "darwin",
				ConditionalAccessEnabled: true,
				Type:                     fleet.PolicyTypeDynamic,
			},
		},
	}, http.StatusOK, &applyResp)

	// Team spec with conditional_access_enabled=true and platform="windows" should succeed
	s.DoJSON("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{
			{
				Name:                     "spec-ca-windows",
				Query:                    "SELECT 2;",
				Team:                     team1.Name,
				Platform:                 "windows",
				ConditionalAccessEnabled: true,
				Type:                     fleet.PolicyTypeDynamic,
			},
		},
	}, http.StatusOK, &applyResp)
}
