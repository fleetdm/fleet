package service

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/xml"
	"io"
	"log/slog"
	"net/url"
	"testing"

	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/sso"
	"github.com/stretchr/testify/require"
)

// mdmSSOTestMetadata is valid SAML IdP metadata with an HTTP-Redirect
// SingleSignOnService binding so that InitiateMDMSSO produces a redirect URL
// carrying an inflatable SAMLRequest.
const mdmSSOTestMetadata = `<?xml version="1.0"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="test-idp">
  <md:IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>MIIDXTCCAkWgAwIBAgIJALmVVuDWu4NYMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwHhcNMTYxMjMxMTQzNDQ3WhcNNDgwNjI1MTQzNDQ3WjBFMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzUCFozgNb1h1M0jzNRSCjhOBnR+uVbVpaWfXYIR+AhWDdEe5ryY+CgavOg8bfLybyzFdehlYdDRgkedEB/GjG8aJw06l0qF4jDOAw0kEygWCu2mcH7XOxRt+YAH3TVHa/Hu1W3WjzkobqqqLQ8gkKWWM27fOgAZ6GieaJBN6VBSMMcPey3HWLBmc+TYJmv1dbaO2jHhKh8pfKw0W12VM8P1PIO8gv4Phu/uuJYieBWKixBEyy0lHjyixYFCR12xdh4CA47q958ZRGnnDUGFVE1QhgRacJCOZ9bd5t9mr8KLaVBYTCJo5ERE8jymab5dPqe5qKfJsCZiqWglbjUo9twIDAQABo1AwTjAdBgNVHQ4EFgQUxpuwcs/CYQOyui+r1G+3KxBNhxkwHwYDVR0jBBgwFoAUxpuwcs/CYQOyui+r1G+3KxBNhxkwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAAiWUKs/2x/viNCKi3Y6blEuCtAGhzOOZ9EjrvJ8+COH3Rag3tVBWrcBZ3/uhhPq5gy9lqw4OkvEws99/5jFsX1FJ6MKBgqfuy7yh5s1YfM0ANHYczMmYpZeAcQf2CGAaVfwTTfSlzNLsF2lW/ly7yapFzlYSJLGoVE+OHEu8g5SlNACUEfkXw+5Eghh+KzlIN7R6Q7r2ixWNFBC/jWf7NKUfJyX8qIG5md1YUeT6GBW9Bm2/1/RiO24JTaYlfLdKK9TYb8sG5B+OLab2DImG99CJ25RkAcSobWNF5zD0O6lgOo3cEdB/ksCq3hmtlC/DlLZ/D8CJ+7VuZnS1rR2naQ==</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.example.com/sso"/>
  </md:IDPSSODescriptor>
</md:EntityDescriptor>`

func inflateMDMAuthnRequest(t *testing.T, s string) *saml.AuthnRequest {
	t.Helper()

	decoded, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)

	r := flate.NewReader(bytes.NewReader(decoded))
	defer r.Close()

	var req saml.AuthnRequest
	require.NoError(t, xml.NewDecoder(r).Decode(&req))
	return &req
}

func mdmSSOTestAppConfig(serverURL string, idpConfigured bool) *fleet.AppConfig {
	ac := &fleet.AppConfig{ServerSettings: fleet.ServerSettings{ServerURL: serverURL}}
	if idpConfigured {
		ac.MDM.EndUserAuthentication.SSOProviderSettings = fleet.SSOProviderSettings{
			EntityID: "fleet",
			IDPName:  "TestIDP",
			Metadata: mdmSSOTestMetadata,
		}
	}
	return ac
}

func newMDMSSOTestService(t *testing.T, appConfig *fleet.AppConfig, cfg config.FleetConfig) *Service {
	t.Helper()

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)

	ds := new(mock.Store)
	ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) { return appConfig, nil }

	return &Service{
		ds:              ds,
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		authz:           authorizer,
		config:          cfg,
		ssoSessionStore: sso.NewSessionStore(redistest.NopRedis()),
	}
}

func TestInitiateMDMSSOACSURLWithURLPrefix(t *testing.T) {
	// With url_prefix set, the MDM ACS callback URL must carry the subpath exactly
	// once, regardless of whether server_url was configured with or without the
	// subpath. The latter is the configuration older deployments may have used.
	testCases := []struct {
		name      string
		serverURL string
	}{
		{
			name:      "server_url includes the subpath",
			serverURL: "https://fleet.example.com/apps/fleet",
		},
		{
			name:      "server_url omits the subpath",
			serverURL: "https://fleet.example.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.TestConfig()
			cfg.Server.URLPrefix = "/apps/fleet"

			svc := newMDMSSOTestService(t, mdmSSOTestAppConfig(tc.serverURL, true), cfg)

			_, _, idpURL, err := svc.InitiateMDMSSO(t.Context(), "", "", "")
			require.NoError(t, err)
			require.NotEmpty(t, idpURL)

			parsed, err := url.Parse(idpURL)
			require.NoError(t, err)
			encoded := parsed.Query().Get("SAMLRequest")
			require.NotEmpty(t, encoded)

			authReq := inflateMDMAuthnRequest(t, encoded)
			require.NotNil(t, authReq.AssertionConsumerServiceURL)
			require.Equal(t,
				"https://fleet.example.com/apps/fleet/api/v1/fleet/mdm/sso/callback",
				authReq.AssertionConsumerServiceURL,
			)
		})
	}
}

func TestInitiateMDMSSOSetsNoRelayState(t *testing.T) {
	svc := newMDMSSOTestService(t,
		mdmSSOTestAppConfig("https://fleet.example.com", true), config.TestConfig())

	for _, initiator := range []string{
		fleet.SSOInitiatorOTAEnroll,
		fleet.SSOInitiatorOrbitSetupExperience,
		fleet.SSOInitiatorAppleMDMSSO,
		fleet.SSOInitiatorAccountDrivenEnroll,
		fleet.SSOInitiatorAccountDrivenEnroll + ":cf2b9a1e4d7c8f36b05e91a2d4c7e830f16b5a92",
	} {
		t.Run(initiator, func(t *testing.T) {
			_, _, idpURL, err := svc.InitiateMDMSSO(t.Context(), initiator, "", "host-uuid-1")
			require.NoError(t, err)

			parsed, err := url.Parse(idpURL)
			require.NoError(t, err)
			require.Empty(t, parsed.Query().Get("RelayState"))
		})
	}
}

func TestDeviceSSOErrorURL(t *testing.T) {
	require.Equal(t,
		"https://fleet.example.com/device/abc123?sso_error=sso_disabled",
		deviceSSOErrorURL("https://fleet.example.com/device/abc123", "sso_disabled"))

	// an existing query string is preserved
	require.Equal(t,
		"https://fleet.example.com/device/abc123?setup_only=1&sso_error=sso_disabled",
		deviceSSOErrorURL("https://fleet.example.com/device/abc123?setup_only=1", "sso_disabled"))
}

func TestMDMSSOCallbackEarlyFailureRedirects(t *testing.T) {
	testCases := []struct {
		name          string
		idpConfigured bool
		wantRedirect  string
	}{
		{
			name:          "expired handshake",
			idpConfigured: true,
			wantRedirect:  "/mdm/sso/callback?error=true&reason=session_expired",
		},
		{
			name:          "any other early failure",
			idpConfigured: false,
			wantRedirect:  "/mdm/sso/callback?error=true",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newMDMSSOTestService(t,
				mdmSSOTestAppConfig("https://fleet.example.com", tc.idpConfigured), config.TestConfig())
			redirectURL, byodCookie, deviceSessionID, deviceSessionDuration := svc.MDMSSOCallback(
				t.Context(), "does-not-exist", []byte("<x/>"))

			require.Equal(t, tc.wantRedirect, redirectURL)
			require.Empty(t, byodCookie)
			require.Empty(t, deviceSessionID)
			require.Zero(t, deviceSessionDuration)

			require.Contains(t, []string{
				apple_mdm.FleetUISSOCallbackError,
				apple_mdm.FleetUISSOCallbackSessionExpired,
			}, redirectURL, "early-failure redirect must be one the endpoint can rewrite")
		})
	}
}
