package service

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io"
	"log/slog"
	"net/url"
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/crewjam/saml"
	shared_mdm "github.com/fleetdm/fleet/v4/pkg/mdm"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mock"
	mockredis "github.com/fleetdm/fleet/v4/server/mock/redis"
	svcmock "github.com/fleetdm/fleet/v4/server/mock/service"
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

func newMDMSSOTestService(t *testing.T, appConfig *fleet.AppConfig, cfg config.FleetConfig) (*Service, *mock.Store) {
	t.Helper()

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)

	ds := new(mock.Store)
	ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) { return appConfig, nil }

	svcMock := &svcmock.Service{}
	svcMock.NewActivityFunc = func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error { return nil }

	kvs, _ := inMemoryKeyValueStore()

	return &Service{
		Service:         svcMock,
		ds:              ds,
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		authz:           authorizer,
		config:          cfg,
		clock:           clock.NewMockClock(),
		keyValueStore:   kvs,
		ssoSessionStore: sso.NewSessionStore(redistest.NopRedis()),
	}, ds
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

			svc, _ := newMDMSSOTestService(t, mdmSSOTestAppConfig(tc.serverURL, true), cfg)

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
	svc, _ := newMDMSSOTestService(t,
		mdmSSOTestAppConfig("https://fleet.example.com", true), config.TestConfig())

	// Each initiator names its own host UUID and only setup_experience's is
	// seeded, so this also fails if the pending-prompt precondition is ever
	// widened past setup_experience.
	hostUUIDFor := func(initiator string) string { return "host-uuid-for-" + initiator }
	seedEndUserAuthPrompt(t, svc, hostUUIDFor(fleet.SSOInitiatorOrbitSetupExperience))

	for _, initiator := range []string{
		fleet.SSOInitiatorOTAEnroll,
		fleet.SSOInitiatorOrbitSetupExperience,
		fleet.SSOInitiatorAppleMDMSSO,
		fleet.SSOInitiatorAccountDrivenEnroll,
		fleet.SSOInitiatorAccountDrivenEnroll + ":cf2b9a1e4d7c8f36b05e91a2d4c7e830f16b5a92",
	} {
		t.Run(initiator, func(t *testing.T) {
			_, _, idpURL, err := svc.InitiateMDMSSO(t.Context(), initiator, "", hostUUIDFor(initiator))
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
			svc, _ := newMDMSSOTestService(t,
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

// seedEndUserAuthPrompt puts the service in the state a device is in after Fleet
// answered its enroll request with END_USER_AUTH_REQUIRED.
func seedEndUserAuthPrompt(t *testing.T, svc *Service, hostUUID string) {
	t.Helper()
	require.NoError(t, shared_mdm.RecordEndUserAuthPrompt(
		t.Context(), svc.keyValueStore, hostUUID, svc.clock.Now()))
}

func TestInitiateMDMSSOSetupExperienceRequiresPendingPrompt(t *testing.T) {
	// A host UUID is not a secret, so the unauthenticated setup experience SSO
	// flow may only be started for a device Fleet just told to authenticate.
	// Otherwise any IdP user can bind any host to their own account.
	testCases := []struct {
		name string
		// pendingFor is the host UUID Fleet answered END_USER_AUTH_REQUIRED for.
		pendingFor  string
		hostUUID    string
		storeFails  bool
		wantRefused bool
	}{
		{
			name:        "device fleet prompted",
			pendingFor:  "host-uuid-1",
			hostUUID:    "host-uuid-1",
			wantRefused: false,
		},
		{
			name:        "some other host uuid",
			pendingFor:  "host-uuid-1",
			hostUUID:    "victim-uuid",
			wantRefused: true,
		},
		{
			name:        "no host uuid at all",
			pendingFor:  "host-uuid-1",
			hostUUID:    "",
			wantRefused: true,
		},
		{
			name:        "store failure fails closed",
			pendingFor:  "host-uuid-1",
			hostUUID:    "host-uuid-1",
			storeFails:  true,
			wantRefused: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newMDMSSOTestService(t,
				mdmSSOTestAppConfig("https://fleet.example.com", true), config.TestConfig())
			seedEndUserAuthPrompt(t, svc, tc.pendingFor)
			if tc.storeFails {
				svc.keyValueStore = &mockredis.KeyValueStore{
					GetFunc: func(_ context.Context, _ string) (*string, error) {
						return nil, errors.New("redis is down")
					},
				}
			}

			_, _, idpURL, err := svc.InitiateMDMSSO(
				t.Context(), fleet.SSOInitiatorOrbitSetupExperience, "", tc.hostUUID)
			if tc.wantRefused {
				require.Error(t, err)
				require.Empty(t, idpURL)
				return
			}
			require.NoError(t, err)
			require.NotEmpty(t, idpURL)
		})
	}
}

func TestBindHostToIdPAccountFromSSO(t *testing.T) {
	const hostUUID = "host-uuid-1"
	acct := &fleet.MDMIdPAccount{UUID: "acct-uuid-new", Email: "new@example.com"}

	newBindTestService := func(t *testing.T) (*Service, *mock.Store, *[]fleet.ActivityDetails) {
		t.Helper()

		svc, ds := newMDMSSOTestService(t,
			mdmSSOTestAppConfig("https://fleet.example.com", true), config.TestConfig())
		ds.GetMDMIdPAccountByUUIDFunc = func(_ context.Context, uuid string) (*fleet.MDMIdPAccount, error) {
			return &fleet.MDMIdPAccount{UUID: uuid, Email: "old@example.com"}, nil
		}

		var activities []fleet.ActivityDetails
		svc.Service.(*svcmock.Service).NewActivityFunc = func(
			_ context.Context, _ *fleet.User, activity fleet.ActivityDetails,
		) error {
			activities = append(activities, activity)
			return nil
		}
		return svc, ds, &activities
	}

	testCases := []struct {
		name string
		// prompted is whether Fleet is still waiting on this device's end user.
		// Once the device has enrolled the prompt is gone, and a sign-in that
		// started before that may no longer overwrite what enrollment settled on.
		prompted bool
		// previousAcctUUID is the binding the datastore reports the host had.
		previousAcctUUID string
		wantReplace      bool
		// wantActivity is nil when nothing is recorded.
		wantActivity fleet.ActivityDetails
	}{
		{
			// The ordinary first enrollment: no account is linked to the UUID,
			// and no host row exists for it either.
			name:        "binds a prompted device that has nothing linked yet",
			prompted:    true,
			wantReplace: true,
			wantActivity: fleet.ActivityTypeBoundHostToIdPAccount{
				HostUUID: hostUUID, IdPEmail: "new@example.com",
			},
		},
		{
			name:             "replaces while fleet is still waiting on the device",
			prompted:         true,
			previousAcctUUID: "acct-uuid-old",
			wantReplace:      true,
			wantActivity: fleet.ActivityTypeBoundHostToIdPAccount{
				HostUUID: hostUUID, IdPEmail: "new@example.com", ReplacedIdPEmail: "old@example.com",
			},
		},
		{
			// The refusal is the signature of a replayed sign-in, so it is
			// recorded rather than only logged.
			name:             "does not take over the binding of a host that already enrolled",
			previousAcctUUID: "acct-uuid-old",
			wantActivity: fleet.ActivityTypeRefusedHostIdPAccountChange{
				HostUUID: hostUUID, IdPEmail: "new@example.com", ExistingIdPEmail: "old@example.com",
			},
		},
		{
			name: "still fills in a missing binding after enrollment",
			wantActivity: fleet.ActivityTypeBoundHostToIdPAccount{
				HostUUID: hostUUID, IdPEmail: "new@example.com",
			},
		},
		{
			name:             "records nothing when the binding is unchanged",
			previousAcctUUID: acct.UUID,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc, ds, activities := newBindTestService(t)
			if tc.prompted {
				seedEndUserAuthPrompt(t, svc, hostUUID)
			}
			ds.AssociateHostMDMIdPAccountFromSSOFunc = func(
				_ context.Context, gotHostUUID string, gotAcctUUID string, replaceExisting bool,
			) (string, error) {
				require.Equal(t, hostUUID, gotHostUUID)
				require.Equal(t, acct.UUID, gotAcctUUID)
				require.Equal(t, tc.wantReplace, replaceExisting)
				return tc.previousAcctUUID, nil
			}

			require.NoError(t, svc.bindHostToIdPAccountFromSSO(t.Context(), hostUUID, acct))
			require.True(t, ds.AssociateHostMDMIdPAccountFromSSOFuncInvoked)

			if tc.wantActivity == nil {
				require.Empty(t, *activities)
				return
			}
			require.Equal(t, []fleet.ActivityDetails{tc.wantActivity}, *activities)
		})
	}

	t.Run("fails closed when the prompt store is unreadable", func(t *testing.T) {
		svc, ds, _ := newBindTestService(t)
		svc.keyValueStore = &mockredis.KeyValueStore{
			GetFunc: func(_ context.Context, _ string) (*string, error) {
				return nil, errors.New("redis is down")
			},
		}
		ds.AssociateHostMDMIdPAccountFromSSOFunc = func(
			_ context.Context, _ string, _ string, _ bool,
		) (string, error) {
			t.Fatal("must not write a binding it cannot decide the rule for")
			return "", nil
		}

		require.Error(t, svc.bindHostToIdPAccountFromSSO(t.Context(), hostUUID, acct))
	})
}
