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
			ds := new(mock.Store)

			authorizer, err := authz.NewAuthorizer()
			require.NoError(t, err)

			cfg := config.TestConfig()
			cfg.Server.URLPrefix = "/apps/fleet"

			svc := &Service{
				ds:              ds,
				logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
				authz:           authorizer,
				config:          cfg,
				ssoSessionStore: sso.NewSessionStore(redistest.NopRedis()),
			}

			appConfig := &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: tc.serverURL,
				},
			}
			appConfig.MDM.EndUserAuthentication.SSOProviderSettings = fleet.SSOProviderSettings{
				EntityID: "fleet",
				IDPName:  "TestIDP",
				Metadata: mdmSSOTestMetadata,
			}
			ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
				return appConfig, nil
			}

			_, _, idpURL, err := svc.InitiateMDMSSO(context.Background(), "", "", "")
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
	// Failures before the SSO session loads can't know the initiator or the
	// original URL -- both live in the session -- so the only signal left is the
	// relay state the IdP echoed back.
	newSvc := func(t *testing.T, idpConfigured bool) *Service {
		t.Helper()

		authorizer, err := authz.NewAuthorizer()
		require.NoError(t, err)

		appConfig := &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{ServerURL: "https://fleet.example.com"},
		}
		if idpConfigured {
			appConfig.MDM.EndUserAuthentication.SSOProviderSettings = fleet.SSOProviderSettings{
				EntityID: "fleet",
				IDPName:  "TestIDP",
				Metadata: mdmSSOTestMetadata,
			}
		}

		ds := new(mock.Store)
		ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) { return appConfig, nil }

		return &Service{
			ds:     ds,
			logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			authz:  authorizer,
			config: config.TestConfig(),
			// Nothing was ever stored, so Fullfill fails exactly as it does once
			// the handshake session has expired.
			ssoSessionStore: sso.NewSessionStore(redistest.NopRedis()),
		}
	}

	testCases := []struct {
		name          string
		relayState    string
		idpConfigured bool
		wantRedirect  string
	}{
		{
			name:          "fleet desktop, expired handshake",
			relayState:    fleet.SSOInitiatorFleetDesktop,
			idpConfigured: true,
			wantRedirect:  "/device/sso-error?reason=session_expired",
		},
		{
			name:          "fleet desktop, other failure",
			relayState:    fleet.SSOInitiatorFleetDesktop,
			idpConfigured: false,
			wantRedirect:  "/device/sso-error?reason=error",
		},
		{
			name:          "no relay state keeps the admin callback route",
			relayState:    "",
			idpConfigured: true,
			wantRedirect:  "/mdm/sso/callback?error=true&reason=session_expired",
		},
		{
			name:          "enrollment initiator keeps the admin callback route",
			relayState:    fleet.SSOInitiatorAppleMDMSSO,
			idpConfigured: true,
			wantRedirect:  "/mdm/sso/callback?error=true&reason=session_expired",
		},
		{
			name:          "no relay state, other failure",
			relayState:    "",
			idpConfigured: false,
			wantRedirect:  "/mdm/sso/callback?error=true",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc := newSvc(t, tc.idpConfigured)
			redirectURL, byodCookie, deviceSessionID, deviceSessionDuration := svc.MDMSSOCallback(
				t.Context(), "does-not-exist", tc.relayState, []byte("<x/>"))

			require.Equal(t, tc.wantRedirect, redirectURL)
			require.Empty(t, byodCookie)
			require.Empty(t, deviceSessionID)
			require.Zero(t, deviceSessionDuration)
		})
	}
}
