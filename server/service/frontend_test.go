package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServeFrontend(t *testing.T) {
	if !hasBuildTag("full") {
		t.Skip("This test requires running with -tags full")
	}
	logger := log.NewLogfmtLogger(os.Stdout)
	h := ServeFrontend("", false, logger)
	ts := httptest.NewServer(h)
	t.Cleanup(func() {
		ts.Close()
	})

	// Simulate a misconfigured osquery sending log requests to the root endpoint.
	requestBody := []byte(`
	{"data":[{"snapshot":[{"build_distro":"10.14","build_platform":"darwin","config_hash":"d8d220440ebea888f8704c4a0a5c1ced4ab601b5",
	"config_valid":"1","extensions":"active","instance_id":"522e6020-37de-460b-bb01-b76c77298f75","pid":"57456","platform_mask":"21",
	"start_time":"1707768989","uuid":"408F3B27-434F-4776-8538-DA394A3D545F","version":"5.11.0","watcher":"57455"}],"action":"snapshot",
	"name":"packFOOBARGlobalFOOBARQuery_50","hostIdentifier":"589966AE-074A-503B-B17B-54B05684A120","calendarTime":"Mon Feb 12 20:16:40 2024 UTC",
	"unixTime":1707769000,"epoch":0,"counter":0,"numerics":false,"decorations":{"host_uuid":"589966AE-074A-503B-B17B-54B05684A120",
	"hostname":"foobar.local"}},{"snapshot":[{"build_distro":"10.14","build_platform":"darwin",
	"config_hash":"d8d220440ebea888f8704c4a0a5c1ced4ab601b5","config_valid":"1","extensions":"active",
	"instance_id":"522e6020-37de-460b-bb01-b76c77298f75","pid":"57456","platform_mask":"21","start_time":"1707768989",
	"uuid":"408F3B27-434F-4776-8538-DA394A3D545F","version":"5.11.0","watcher":"57455"}],"action":"snapshot",
	"name":"packFOOBARGlobalFOOBARQuery_28","hostIdentifier": "589966AE-074A-503B-B17B-54B05684A120","calendarTime":"Mon Feb 12 20:16:41 2024 UTC",
	"unixTime":1707769001,"epoch":0,"counter":0,"numerics":false,"decorations":{"host_uuid":"408F3B27-434F-4776-8538-DA394A3D545F",
	"hostname":"foobar.local"}}],"log_type":"result","node_key":"J9pA1CmjydHGi0bqS1XkkR9pOJQJzoPA"}`)
	response, err := http.DefaultClient.Post(ts.URL, "", bytes.NewReader(requestBody))
	require.NoError(t, err)
	require.Equal(t, http.StatusMethodNotAllowed, response.StatusCode)
}

func TestServeEndUserEnrollOTA(t *testing.T) {
	if !hasBuildTag("full") {
		t.Skip("This test requires running with -tags full")
	}

	ds := new(mock.DataStore)
	ds.HasUsersFunc = func(ctx context.Context) (bool, error) {
		return true, nil
	}
	appCfg := &fleet.AppConfig{
		MDM: fleet.MDM{
			EnabledAndConfigured:        false,
			AndroidEnabledAndConfigured: false,
		},
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return appCfg, nil
	}

	svc, _ := newTestService(t, ds, nil, nil)
	premiumSvc, _ := newTestService(t, ds, nil, nil, &TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		Pool: redistest.SetupRedis(t, "frontend", false, false, false),
	})
	logger := log.NewLogfmtLogger(os.Stdout)
	h := ServeEndUserEnrollOTA(svc, "", ds, logger)
	premiumHandler := ServeEndUserEnrollOTA(premiumSvc, "", ds, logger)
	ts := httptest.NewServer(h)
	premiumTS := httptest.NewServer(premiumHandler)
	t.Cleanup(func() {
		ts.Close()
		premiumTS.Close()
	})
	noRedirectClient := fleethttp.NewClient(fleethttp.WithFollowRedir(false))

	makeEnrollRequest := func(enrollSecret string, premium bool) *http.Response {
		url := ts.URL
		if premium {
			url = premiumTS.URL
		}
		response, err := noRedirectClient.Get(url + "?enroll_secret=" + enrollSecret)
		require.NoError(t, err)
		assert.True(t, ds.AppConfigFuncInvoked)
		return response
	}

	validateEnrollPageIsReturned := func(response *http.Response, enrollSecret string, mdmEnabled bool) {
		require.Equal(t, http.StatusOK, response.StatusCode)
		// assert html is returned
		require.Equal(t, response.Header.Get("Content-Type"), "text/html; charset=utf-8")
		defer response.Body.Close()
		bodyBytes, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		bodyString := string(bodyBytes)
		require.Contains(t, bodyString, "api/v1/fleet/enrollment_profiles/ota?enroll_secret="+enrollSecret)
		require.Contains(t, bodyString, "/api/v1/fleet/android_enterprise/enrollment_token")
		require.Contains(t, bodyString, fmt.Sprintf(`const ANDROID_MDM_ENABLED = "%t" === "true";`, mdmEnabled))
		require.Contains(t, bodyString, fmt.Sprintf(`const MAC_MDM_ENABLED = "%t" == "true";`, mdmEnabled))
	}

	for _, enabled := range []bool{true, false} {
		t.Run(fmt.Sprintf("MDM enabled: %t", enabled), func(t *testing.T) {
			appCfg.MDM.EnabledAndConfigured = enabled
			appCfg.MDM.AndroidEnabledAndConfigured = enabled
			enrollSecret := "foo"

			response := makeEnrollRequest(enrollSecret, false)

			// assert it contains the content we expect
			validateEnrollPageIsReturned(response, enrollSecret, enabled)
		})
	}

	t.Run("sso in front", func(t *testing.T) {
		invalidSecret := "invalid"
		globalSecret := "global"
		teamSecret := "team"
		validTeamId := uint(1)
		ds.VerifyEnrollSecretFunc = func(ctx context.Context, enrollSecret string) (*fleet.EnrollSecret, error) {
			if enrollSecret == invalidSecret {
				return nil, ctxerr.Wrap(ctx, &common_mysql.NotFoundError{
					ResourceType: "EnrollSecret",
				}, "no matching secret found")
			}
			if enrollSecret == globalSecret {
				return &fleet.EnrollSecret{
					TeamID: nil,
				}, nil
			}
			if enrollSecret == teamSecret {
				return &fleet.EnrollSecret{
					TeamID: &validTeamId,
				}, nil
			}

			return nil, ctxerr.Errorf(ctx, "failure")
		}
		teamMdmConfig := &fleet.TeamMDM{
			MacOSSetup: fleet.MacOSSetup{},
		}
		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			if teamID != validTeamId {
				return nil, ctxerr.Errorf(ctx, "invalid team id")
			}

			return teamMdmConfig, nil
		}

		t.Run("if end user auth is configured", func(t *testing.T) {
			ssoUrl := "https://fake-sso.com/sso"
			appCfg.MDM.EndUserAuthentication = fleet.MDMEndUserAuthentication{
				SSOProviderSettings: fleet.SSOProviderSettings{
					EntityID: "fake-sso",
					IDPName:  "fake-sso",
					Metadata: fmt.Sprintf(`
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"
    entityID="https://fake-sso.com">
    <md:IDPSSODescriptor WantAuthnRequestsSigned="false"
        protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
        <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"
            Location="%s" />
    </md:IDPSSODescriptor>
</md:EntityDescriptor>
					`, ssoUrl),
				},
			}

			t.Run("but enroll secret is invalid", func(t *testing.T) {
				response := makeEnrollRequest(invalidSecret, true)

				require.Equal(t, http.StatusSeeOther, response.StatusCode)
				require.True(t, strings.HasPrefix(response.Header.Get("Location"), ssoUrl+"?SAMLRequest"))
				require.True(t, ds.VerifyEnrollSecretFuncInvoked)
			})

			t.Run("enroll secret matches a team with end user auth enabled", func(t *testing.T) {
				teamMdmConfig.MacOSSetup.EnableEndUserAuthentication = true

				response := makeEnrollRequest(teamSecret, true)

				require.Equal(t, http.StatusSeeOther, response.StatusCode)
				require.True(t, strings.HasPrefix(response.Header.Get("Location"), ssoUrl+"?SAMLRequest"))
				require.True(t, ds.VerifyEnrollSecretFuncInvoked)
			})

			t.Run("enroll secret matches a team with no end user auth do not show", func(t *testing.T) {
				appCfg.MDM.EnabledAndConfigured = true
				appCfg.MDM.AndroidEnabledAndConfigured = true
				teamMdmConfig.MacOSSetup.EnableEndUserAuthentication = false

				response := makeEnrollRequest(teamSecret, true)

				validateEnrollPageIsReturned(response, teamSecret, true)
			})

			// TODO(IB): Add test when coming back to page after SSO, that SSO is not prompted.
		})

		t.Run("is not shown if end user auth is not configured", func(t *testing.T) {
			appCfg.MDM = fleet.MDM{}
			response := makeEnrollRequest(globalSecret, true)

			validateEnrollPageIsReturned(response, globalSecret, false)
		})

		t.Run("is not checked if non-premium", func(t *testing.T) {
			// Arrange - set desired initial state as it persist across previous tests.
			ds.VerifyEnrollSecretFuncInvoked = false
			ds.TeamMDMConfigFuncInvoked = false
			appCfg.MDM = fleet.MDM{}

			// Act
			makeEnrollRequest(globalSecret, false)

			require.False(t, ds.VerifyEnrollSecretFuncInvoked)
			require.False(t, ds.TeamMDMConfigFuncInvoked)
		})
	})
}
