package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/psso/regtoken"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

// pssoMockIdP is a stand-in OAuth2 ROPG (Resource Owner Password Grant) token
// endpoint. Fleet's PSSO login flow POSTs grant_type=password here and reads the
// user's claims out of the returned id_token, which it does not signature-verify
// (it trusts the direct TLS channel), so the token can be signed with any key.
type pssoMockIdP struct {
	mu       sync.Mutex
	lastForm url.Values

	validUser, validPass string
	// idToken returns the id_token JSON value for a successful login.
	idToken func() string
}

func (m *pssoMockIdP) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		m.mu.Lock()
		m.lastForm = r.PostForm
		m.mu.Unlock()

		if r.FormValue("grant_type") != "password" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unsupported_grant_type"})
			return
		}
		if r.FormValue("username") != m.validUser || r.FormValue("password") != m.validPass {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant", "error_description": "bad credentials"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{ //nolint:gosec // G101: opaque test fixture refresh_token, not a real credential
			"id_token":      m.idToken(),
			"refresh_token": "idp-refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func (m *pssoMockIdP) lastGrantType() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastForm.Get("grant_type")
}

// TestApplePlatformSSO drives the full Apple Platform SSO server flow end to end
// with the reusable mdmtest device simulator: profile upload, MDM delivery of
// the (substituted) registration token, device registration, password login
// against a mocked IdP (plaintext and encrypted-on-the-wire), TokenToUserMapping
// claim forwarding, the offline-unlock key request/exchange, and the required
// error cases. The minted id_token is validated against Fleet's published JWKS.
func (s *integrationMDMTestSuite) TestApplePlatformSSO() {
	t := s.T()
	ctx := context.Background()

	const (
		clientID  = "test-psso-client-id"
		validUser = "fleetie@example.com"
		validPass = "correct-horse-battery-staple"
		idpSub    = "00ufleetiesubject"
		shortName = "fleetie"
		fullName  = "Fleetie Example"
	)

	// Mocked OAuth ROPG IdP. The id_token carries an "accountName" custom claim
	// (the account* prefix is what Fleet forwards into the minted id_token for the
	// profile's TokenToUserMapping to map to the macOS short name).
	idp := &pssoMockIdP{
		validUser: validUser,
		validPass: validPass,
		idToken: func() string {
			return mockOIDCIDToken(t, jwt.MapClaims{
				"sub":                idpSub,
				"email":              validUser,
				"name":               fullName,
				"preferred_username": shortName,
				"accountName":        shortName,
			})
		},
	}
	idpSrv := httptest.NewServer(idp.handler())
	t.Cleanup(idpSrv.Close)

	serverHost, err := url.Parse(s.server.URL)
	require.NoError(t, err)

	// Enroll a Mac in MDM and build a PSSO device on top of it. Enrollment
	// enqueues the post-enroll worker job, which must run before profiles flow.
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.awaitRunAppleMDMWorkerSchedule()
	dev, err := mdmtest.NewApplePSSODevice(mdmDevice, s.server.URL, clientID)
	require.NoError(t, err)

	// Before the feature is configured, the public PSSO endpoints are 404 so they
	// are indistinguishable from absent.
	status, _, err := dev.JWKSResponse()
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, status)
	status, _, err = dev.AASA()
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, status)

	s.enableApplePSSO(t, idpSrv.URL+"/token", clientID, "test-client-secret")

	// Configured: JWKS publishes a signing and an encryption key; AASA lists the
	// extension.
	sigPub, encPub, err := dev.JWKS()
	require.NoError(t, err)
	require.NotNil(t, sigPub)
	require.NotNil(t, encPub)
	status, aasaBody, err := dev.AASA()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Contains(t, string(aasaBody), "com.fleetdm.fleet-desktop.pssoextension")

	// Registration must present a valid Fleet-signed token bound to this host.
	t.Run("registration requires a valid token", func(t *testing.T) {
		dev.SetRegistrationToken("")
		require.Error(t, dev.Register(), "empty token must be rejected")

		dev.SetRegistrationToken("not-a-jwt")
		require.Error(t, dev.Register(), "garbage token must be rejected")

		wrongKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		forged, err := regtoken.Mint(wrongKey, host.UUID, time.Now())
		require.NoError(t, err)
		dev.SetRegistrationToken(forged)
		require.Error(t, dev.Register(), "token signed by a non-Fleet key must be rejected")

		gotDev, err := s.ds.GetPSSODevice(ctx, host.UUID)
		require.True(t, err != nil || gotDev == nil, "no device should be registered after failures")
	})

	// Upload the PSSO profile, reconcile, and have the device pull the substituted
	// registration token out of the delivered InstallProfile command.
	s.uploadApplePSSOProfile(serverHost.Host)
	// The per-host profile-processing key debounces reconciliation; clear it so
	// the next schedule run delivers the newly uploaded profile.
	require.NoError(t, s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+mdmDevice.UUID))
	s.awaitTriggerProfileSchedule(t)

	regToken := s.deliverApplePSSORegToken(t, mdmDevice, dev)
	require.NotEmpty(t, regToken)

	require.NoError(t, dev.Register())
	pssoDevice, err := s.ds.GetPSSODevice(ctx, host.UUID)
	require.NoError(t, err)
	require.NotNil(t, pssoDevice)
	keys, err := s.ds.ListPSSOKeys(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, keys, 2, "a signing and an encryption key are registered")

	t.Run("password login and id_token validated against jwks", func(t *testing.T) {
		res, err := dev.Login(validUser, validPass, mdmtest.PSSOLoginOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, res.IDToken)
		require.Equal(t, "password", idp.lastGrantType())

		// The plaintext password rode in the (signed) assertion claims.
		assertion := decodeJWSClaims(t, res.RawAssertion)
		require.Equal(t, validPass, assertion["password"])
		require.Equal(t, "password", assertion["grant_type"])

		// The device validates the response id_token against Fleet's published JWKS.
		claims, err := dev.ValidateIDToken(res.IDToken)
		require.NoError(t, err)
		require.Equal(t, clientID, claims["aud"])
		require.Equal(t, res.SessionNonce, claims["nonce"])
		require.Equal(t, serverHost.Hostname(), claims["iss"])
		require.Equal(t, idpSub, claims["sub"])
		require.Equal(t, validUser, claims["email"])
		require.Equal(t, fullName, claims["name"])
		require.Equal(t, shortName, claims["preferred_username"])
		// TokenToUserMapping: the account-prefixed claim is forwarded so the
		// profile can map the macOS short name to it.
		require.Equal(t, shortName, claims["accountName"])
	})

	t.Run("password encrypted on the wire", func(t *testing.T) {
		res, err := dev.Login(validUser, validPass, mdmtest.PSSOLoginOptions{EncryptOnWire: true})
		require.NoError(t, err)

		// No plaintext password on the wire — it rides inside an encrypted assertion.
		assertion := decodeJWSClaims(t, res.RawAssertion)
		require.NotContains(t, assertion, "password")
		require.NotEmpty(t, assertion["assertion"])
		require.Equal(t, "urn:ietf:params:oauth:grant-type:jwt-bearer", assertion["grant_type"])

		claims, err := dev.ValidateIDToken(res.IDToken)
		require.NoError(t, err)
		require.Equal(t, idpSub, claims["sub"])
	})

	t.Run("key request and key exchange", func(t *testing.T) {
		certDER, err := dev.KeyRequest()
		require.NoError(t, err)
		require.NotEmpty(t, certDER)

		// KeyExchange independently recomputes the ECDH against the provisioned
		// certificate's key and fails if it doesn't match the server's secret.
		shared, err := dev.KeyExchange()
		require.NoError(t, err)
		require.Len(t, shared, 32)
	})

	t.Run("invalid IdP credentials are rejected", func(t *testing.T) {
		_, err := dev.Login(validUser, "wrong-password", mdmtest.PSSOLoginOptions{})
		require.Error(t, err)
	})

	t.Run("assertion signed with the wrong key is rejected", func(t *testing.T) {
		wrongKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		_, err = dev.Login(validUser, validPass, mdmtest.PSSOLoginOptions{SigningKeyOverride: wrongKey})
		require.Error(t, err)
	})

	t.Run("request_nonce is single-use", func(t *testing.T) {
		nonce, err := dev.Nonce()
		require.NoError(t, err)
		_, err = dev.Login(validUser, validPass, mdmtest.PSSOLoginOptions{RequestNonceOverride: nonce})
		require.NoError(t, err)
		_, err = dev.Login(validUser, validPass, mdmtest.PSSOLoginOptions{RequestNonceOverride: nonce})
		require.Error(t, err, "replaying a consumed nonce must be rejected")
	})
}

// enableApplePSSO configures the macOS account-provisioning (Platform SSO)
// feature: it points the IdP at the mock token URL, stores the client secret,
// and bootstraps Fleet's PSSO signing/CA/encryption assets. The config is set
// directly (not via the API) so the mock IdP can use a plain-http URL — the
// API's https validation is covered by the appconfig tests. State is restored on
// cleanup so the shared suite isn't left with the feature enabled.
func (s *integrationMDMTestSuite) enableApplePSSO(t *testing.T, tokenURL, clientID, secret string) {
	ctx := context.Background()
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	orig := appCfg.MDM.AppleAccountProvisioning

	appCfg.MDM.AppleAccountProvisioning = fleet.AppleAccountProvisioning{
		OAuthIdPTokenURL: optjson.SetString(tokenURL),
		OAuthIdPClientID: optjson.SetString(clientID),
	}
	require.NoError(t, s.ds.SaveAppConfig(ctx, appCfg))
	require.NoError(t, s.ds.InsertMDMConfigAssets(ctx, []fleet.MDMConfigAsset{
		{Name: fleet.MDMAssetAppleAccountProvisioningIdPClientSecret, Value: []byte(secret)},
	}, nil))
	require.NoError(t, bootstrapPSSOAssets(ctx, s.ds))

	t.Cleanup(func() {
		appCfg, err := s.ds.AppConfig(context.Background())
		if err != nil {
			return
		}
		appCfg.MDM.AppleAccountProvisioning = orig
		_ = s.ds.SaveAppConfig(context.Background(), appCfg)
	})
}

// uploadApplePSSOProfile uploads the Fleet Platform SSO configuration profile
// (carrying the $FLEET_VAR_PSSO_DEVICE_REGISTRATION_TOKEN variable) as a no-team
// profile so it reconciles onto the enrolled host.
func (s *integrationMDMTestSuite) uploadApplePSSOProfile(serverHost string) {
	profile := strings.ReplaceAll(applePSSOProfileTemplate, "fleet.example.com", serverHost)
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{
		Profiles: []fleet.MDMProfileBatchPayload{
			{Name: "Fleet Platform SSO", Contents: []byte(profile)},
		},
	}, http.StatusNoContent)
}

// deliverApplePSSORegToken drains the host's pending MDM commands and extracts
// the substituted registration token from the delivered PSSO InstallProfile.
func (s *integrationMDMTestSuite) deliverApplePSSORegToken(t *testing.T, mdmDevice *mdmtest.TestAppleMDMClient, dev *mdmtest.TestApplePSSODevice) string {
	var token string
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "InstallProfile" {
			if tok, terr := dev.RegistrationTokenFromCommand(cmd); terr == nil && tok != "" {
				token = tok
			}
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.NotEmpty(t, token, "PSSO registration token was not delivered in an InstallProfile")
	return token
}

// mockOIDCIDToken builds an id_token JWT the mock IdP returns. Fleet reads the
// claims without verifying the signature, so any signing key works.
func mockOIDCIDToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	claims["iat"] = time.Now().Unix()
	claims["exp"] = time.Now().Add(time.Hour).Unix()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(key)
	require.NoError(t, err)
	return signed
}

// decodeJWSClaims decodes the claims segment of a compact JWS without verifying
// it, for asserting on what the device put on the wire.
func decodeJWSClaims(t *testing.T, compact string) map[string]any {
	t.Helper()
	parts := strings.Split(compact, ".")
	require.Len(t, parts, 3)
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	var claims map[string]any
	require.NoError(t, json.Unmarshal(raw, &claims))
	return claims
}

// applePSSOProfileTemplate is a Fleet Platform SSO v2 (com.apple.extensiblesso,
// UseSharedDeviceKeys) configuration profile whose RegistrationToken is the
// Fleet variable. "fleet.example.com" is replaced with the test server host.
const applePSSOProfileTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>ExtensionData</key>
			<dict>
				<key>BaseURL</key>
				<string>https://fleet.example.com</string>
			</dict>
			<key>ExtensionIdentifier</key>
			<string>com.fleetdm.fleet-desktop.pssoextension</string>
			<key>PayloadDisplayName</key>
			<string>Fleet Extensible Single Sign-On</string>
			<key>PayloadIdentifier</key>
			<string>com.apple.extensiblesso.AF68D4CF-1250-4FF4-AFFB-1176DB539C49</string>
			<key>PayloadType</key>
			<string>com.apple.extensiblesso</string>
			<key>PayloadUUID</key>
			<string>AF68D4CF-1250-4FF4-AFFB-1176DB539C49</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>PlatformSSO</key>
			<dict>
				<key>AuthenticationMethod</key>
				<string>Password</string>
				<key>UseSharedDeviceKeys</key>
				<true/>
				<key>EnableRegistrationDuringSetup</key>
				<true/>
			</dict>
			<key>RegistrationToken</key>
			<string>$FLEET_VAR_PSSO_DEVICE_REGISTRATION_TOKEN</string>
			<key>ScreenLockedBehavior</key>
			<string>DoNotHandle</string>
			<key>TeamIdentifier</key>
			<string>8VBZ3948LU</string>
			<key>Type</key>
			<string>Redirect</string>
			<key>URLs</key>
			<array>
				<string>https://fleet.example.com</string>
			</array>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>Fleet Platform SSO</string>
	<key>PayloadIdentifier</key>
	<string>com.fleetdm.platformsso.fleet.A72B07D0-2E08-45CE-9423-1FCAFFAEC390</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>A72B07D0-2E08-45CE-9423-1FCAFFAEC390</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`
