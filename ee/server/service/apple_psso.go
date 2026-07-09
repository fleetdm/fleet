package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/psso/regtoken"
	jose "github.com/go-jose/go-jose/v3"
)

// pssoServiceState caches the PSSO signing key, CA certificate, and password
// encryption key after first load. All are created in mdm_config_assets when the
// feature is first configured (bootstrapPSSOAssets, core side); this layer only
// loads them.
type pssoServiceState struct {
	mu            sync.Mutex
	signingKey    *ecdsa.PrivateKey
	kid           string
	caCert        *x509.Certificate
	encryptionKey *ecdsa.PrivateKey
	encKID        string
}

const (
	pssoSigningAlg = "ES256"
	// pssoEncryptionAlg is the JWK `alg` published for the password-encryption
	// key and the `alg` Apple uses in the embedded login-assertion JWE.
	pssoEncryptionAlg = "ECDH-ES"

	// The host app bundle ID is included alongside the extension's just in case;
	// PSSO validates against the extension, but listing both is harmless and matches
	// what the IdPs analyzed do.
	appBundleID       = "com.fleetdm.fleet-desktop"
	extensionBundleID = "com.fleetdm.fleet-desktop.pssoextension"

	fleetTeamID = "8VBZ3948LU"
)

// getPSSOSigningKey loads Fleet's PSSO signing key from mdm_config_assets,
// caching it after first use. The key (and CA) are created when the feature is
// first configured (bootstrapPSSOAssets); a missing key here means the feature
// isn't configured, so this never mints — it returns an error.
func (svc *Service) getPSSOSigningKey(ctx context.Context) (*ecdsa.PrivateKey, string, error) {
	svc.pssoState.mu.Lock()
	defer svc.pssoState.mu.Unlock()
	return svc.loadPSSOSigningKeyLocked(ctx)
}

// loadPSSOSigningKeyLocked is the cache-populating load shared by
// getPSSOSigningKey and getPSSOCA. Callers must hold pssoState.mu.
func (svc *Service) loadPSSOSigningKeyLocked(ctx context.Context) (*ecdsa.PrivateKey, string, error) {
	if svc.pssoState.signingKey != nil {
		return svc.pssoState.signingKey, svc.pssoState.kid, nil
	}
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetPSSOSigningKey},
		nil,
	)
	if err != nil {
		if isAssetNotFound(err) {
			return nil, "", ctxerr.Wrap(ctx, err, "psso signing key not found; configure the feature first")
		}
		return nil, "", ctxerr.Wrap(ctx, err, "get psso signing key asset")
	}
	asset, ok := assets[fleet.MDMAssetPSSOSigningKey]
	if !ok || len(asset.Value) == 0 {
		return nil, "", ctxerr.New(ctx, "psso signing key asset is empty")
	}
	key, kid, err := parsePSSOSigningKeyPEM(asset.Value)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "parse stored psso signing key")
	}
	svc.pssoState.signingKey = key
	svc.pssoState.kid = kid
	return key, kid, nil
}

// getPSSOEncryptionKey loads Fleet's PSSO password-encryption key from
// mdm_config_assets, caching it after first use. Like the signing key, it is
// created when the feature is first configured (bootstrapPSSOAssets) and never
// minted here; a missing key means the feature isn't configured.
func (svc *Service) getPSSOEncryptionKey(ctx context.Context) (*ecdsa.PrivateKey, string, error) {
	svc.pssoState.mu.Lock()
	defer svc.pssoState.mu.Unlock()
	if svc.pssoState.encryptionKey != nil {
		return svc.pssoState.encryptionKey, svc.pssoState.encKID, nil
	}
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetPSSOEncryptionKey},
		nil,
	)
	if err != nil {
		if isAssetNotFound(err) {
			return nil, "", ctxerr.Wrap(ctx, err, "psso encryption key not found; configure the feature first")
		}
		return nil, "", ctxerr.Wrap(ctx, err, "get psso encryption key asset")
	}
	asset, ok := assets[fleet.MDMAssetPSSOEncryptionKey]
	if !ok || len(asset.Value) == 0 {
		return nil, "", ctxerr.New(ctx, "psso encryption key asset is empty")
	}
	// The encryption key shares the signing key's PEM encoding and kid scheme
	// (base64url-nopad SHA-256 of the SPKI), which is the kid the extension
	// echoes back in the embedded assertion's JWE header.
	key, kid, err := parsePSSOSigningKeyPEM(asset.Value)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "parse stored psso encryption key")
	}
	svc.pssoState.encryptionKey = key
	svc.pssoState.encKID = kid
	return key, kid, nil
}

// getPSSOCA loads the PSSO CA: the signing key (which is also the CA's private
// key) and the self-signed CA certificate, caching the certificate after first
// use. Like the signing key, the CA is created at first configuration and is
// never minted here.
func (svc *Service) getPSSOCA(ctx context.Context) (*ecdsa.PrivateKey, *x509.Certificate, error) {
	svc.pssoState.mu.Lock()
	defer svc.pssoState.mu.Unlock()

	caKey, _, err := svc.loadPSSOSigningKeyLocked(ctx)
	if err != nil {
		return nil, nil, err
	}
	if svc.pssoState.caCert != nil {
		return caKey, svc.pssoState.caCert, nil
	}

	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetPSSOCACert},
		nil,
	)
	if err != nil {
		if isAssetNotFound(err) {
			return nil, nil, ctxerr.Wrap(ctx, err, "psso ca certificate not found; configure the feature first")
		}
		return nil, nil, ctxerr.Wrap(ctx, err, "get psso ca cert asset")
	}
	asset, ok := assets[fleet.MDMAssetPSSOCACert]
	if !ok || len(asset.Value) == 0 {
		return nil, nil, ctxerr.New(ctx, "psso ca cert asset is empty")
	}
	caCert, err := parsePSSOCACertPEM(asset.Value)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "parse stored psso ca cert")
	}
	svc.pssoState.caCert = caCert
	return caKey, caCert, nil
}

// parsePSSOCACertPEM decodes the stored PEM-wrapped PSSO CA certificate.
func parsePSSOCACertPEM(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("psso ca cert: pem decode returned nil block")
	}
	return x509.ParseCertificate(block.Bytes)
}

func parsePSSOSigningKeyPEM(pemBytes []byte) (*ecdsa.PrivateKey, string, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, "", errors.New("psso signing key: pem decode returned nil block")
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, "", err
	}
	kid, err := computeKID(&key.PublicKey)
	if err != nil {
		return nil, "", err
	}
	return key, kid, nil
}

// computeKID returns base64url-nopad SHA-256 of the SubjectPublicKeyInfo DER
// encoding of pub. Used only for Fleet's own signing key (JWKS/JWT kid).
// Device key kids are different: the extension computes them as SHA-256 of
// the raw X9.63 point bytes and submits them at registration.
func computeKID(pub *ecdsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

// isAssetNotFound reports whether err indicates that the requested
// mdm_config_assets row was absent.
func isAssetNotFound(err error) bool {
	if err == nil {
		return false
	}
	if fleet.IsNotFound(err) {
		return true
	}
	return errors.Is(err, sql.ErrNoRows)
}

// loadSecret / skipSecret are readable arguments for pssoSettingsIfConfigured's
// loadSecret parameter.
const (
	loadSecret = true
	skipSecret = false
)

// pssoSettingsIfConfigured resolves the Platform SSO settings for the current
// request, returning nil when the feature isn't configured. The public IdP
// fields come from AppConfig.MDM.AppleAccountProvisioning and the issuer is the
// Fleet server URL. The client secret lives in mdm_config_assets (a separate,
// uncached read + decrypt); only the token flow needs it, so pass skipSecret
// from the endpoints that don't (nonce, registration, JWKS, AASA) to avoid the
// extra read. Read per request so configuring, clearing, or repointing the IdP
// takes effect without a server restart.
func (svc *Service) pssoSettingsIfConfigured(ctx context.Context, loadSecret bool) (*fleet.PSSOSettings, error) {
	cfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load app config for psso")
	}
	aap := cfg.MDM.AppleAccountProvisioning
	if !aap.Configured() || cfg.ServerSettings.ServerURL == "" {
		return nil, nil
	}

	settings := &fleet.PSSOSettings{
		IssuerURL:   cfg.ServerSettings.ServerURL,
		IdPTokenURL: aap.OAuthIdPTokenURL.Value,
		IdPClientID: aap.OAuthIdPClientID.Value,
	}

	if loadSecret {
		secret, err := svc.pssoIdPClientSecret(ctx)
		if err != nil {
			return nil, err
		}
		if secret == "" {
			// Public config is present but the secret asset is missing: treat the
			// feature as not configured rather than attempting the ROPG flow with
			// empty credentials.
			return nil, nil
		}
		settings.IdPClientSecret = secret
	}

	return settings, nil
}

// pssoIdPClientSecret returns the stored OAuth IdP client secret for the macOS
// account provisioning feature, or "" if none is stored.
func (svc *Service) pssoIdPClientSecret(ctx context.Context) (string, error) {
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetAppleAccountProvisioningIdPClientSecret},
		nil,
	)
	if err != nil {
		if isAssetNotFound(err) {
			return "", nil
		}
		return "", ctxerr.Wrap(ctx, err, "get psso idp client secret asset")
	}
	asset, ok := assets[fleet.MDMAssetAppleAccountProvisioningIdPClientSecret]
	if !ok || len(asset.Value) == 0 {
		return "", nil
	}
	return string(asset.Value), nil
}

// errPSSONotConfigured is returned from the device-facing endpoints when the
// feature is disabled or missing required settings. Return it unwrapped (no
// ctxerr.Wrap) so errors.Is matches on pointer identity.
var errPSSONotConfigured = &fleet.BadRequestError{Message: "Platform SSO is not configured"}

// pssoNonceTTL is how long an issued nonce remains valid before it's
// rejected. Five minutes comfortably covers the extension's immediate
// nonce→token round trip.
const pssoNonceTTL = 5 * time.Minute

// PSSONonce mints a fresh 32-byte base64url nonce, persists it with a short
// TTL via the wired PSSONonceStore, and returns it to the caller. The
// extension embeds this nonce in its next token-request JWT, where it is
// consumed (single-use) to prevent replay.
func (svc *Service) PSSONonce(ctx context.Context) (string, error) {
	// skipauth: This is an unauthenticated endpoint hit by the Mac extension
	// before any user identity is established.
	svc.authz.SkipAuthorization(ctx)

	settings, err := svc.pssoSettingsIfConfigured(ctx, skipSecret)
	if err != nil {
		return "", err
	}
	if settings == nil {
		return "", errPSSONotConfigured
	}

	if svc.pssoNonceStore == nil {
		return "", ctxerr.New(ctx, "psso nonce store not configured")
	}

	var buf [32]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", ctxerr.Wrap(ctx, err, "generate psso nonce")
	}
	nonce := base64.RawURLEncoding.EncodeToString(buf[:])
	if err := svc.pssoNonceStore.Store(ctx, nonce, pssoNonceTTL); err != nil {
		return "", ctxerr.Wrap(ctx, err, "store psso nonce")
	}
	return nonce, nil
}

// consumePSSORequestNonce enforces the single-use request_nonce on token
// requests: the JWT must carry a nonce previously issued by PSSONonce, and
// consuming it must succeed exactly once. Any miss (absent claim, unknown or
// already-used nonce) rejects the request — this is the anti-replay control
// for the unauthenticated token endpoint.
func (svc *Service) consumePSSORequestNonce(ctx context.Context, requestNonce string) error {
	if requestNonce == "" {
		return &fleet.BadRequestError{Message: "psso token: missing request_nonce"}
	}
	if svc.pssoNonceStore == nil {
		return ctxerr.New(ctx, "psso nonce store not configured")
	}
	ok, err := svc.pssoNonceStore.Consume(ctx, requestNonce)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "consume psso request_nonce")
	}
	if !ok {
		return &fleet.BadRequestError{Message: "psso token: invalid or expired request_nonce"}
	}
	return nil
}

// PSSORegisterDevice consumes the device-key enrollment POST from the Mac
// extension: it resolves the enrolled host from the hardware device UUID and
// persists the device record plus its public key rows.
//
// Password-mode registration carries no OAuth code/state — the extension
// simply submits the public halves of its Secure Enclave signing and
// encryption keys. User identity is established later, on each password login
// at the token endpoint.
func (svc *Service) PSSORegisterDevice(ctx context.Context, req fleet.PSSODeviceRegistrationRequest) error {
	// skipauth: This is an unauthenticated device-initiated endpoint. The
	// device proves itself later by signing token requests with the signing
	// key registered here, verified against the kid.
	svc.authz.SkipAuthorization(ctx)

	settings, err := svc.pssoSettingsIfConfigured(ctx, skipSecret)
	if err != nil {
		return err
	}
	if settings == nil {
		return errPSSONotConfigured
	}

	if req.DeviceSigningKey == "" || req.DeviceEncryptionKey == "" || req.SigningKeyID == "" || req.EncryptionKeyID == "" {
		return &fleet.BadRequestError{Message: "missing required psso registration fields"}
	}
	if req.RegistrationToken == "" {
		return &fleet.BadRequestError{Message: "psso registration: missing registration token"}
	}

	// The registration token is what authenticates the device: it is a
	// Fleet-signed JWT delivered in the configuration profile and bound to a
	// specific host. Verify it with Fleet's PSSO signing key and take the host
	// UUID from the token's subject — the device-reported DeviceUUID is not
	// trusted for identity (an unauthenticated caller who guesses an enrolled
	// host's hardware UUID must not be able to register keys for it).
	signingKey, _, err := svc.getPSSOSigningKey(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load psso signing key for registration token validation")
	}
	hostUUID, err := regtoken.Validate(req.RegistrationToken, &signingKey.PublicKey, time.Now())
	if err != nil {
		return &fleet.BadRequestError{Message: "psso registration: invalid registration token", InternalErr: err}
	}

	// Reject unparseable key material up front: a bad PEM stored here would
	// otherwise only surface as opaque verification failures at every
	// subsequent login.
	signingPub, err := parseECPublicKeyPEM([]byte(req.DeviceSigningKey))
	if err != nil {
		return &fleet.BadRequestError{Message: "psso registration: signing key is not a valid P-256 public key"}
	}
	encryptionPub, err := parseECPublicKeyPEM([]byte(req.DeviceEncryptionKey))
	if err != nil {
		return &fleet.BadRequestError{Message: "psso registration: encryption key is not a valid P-256 public key"}
	}

	// The token endpoint resolves a device's host by looking its key up by kid,
	// so a caller free to pick an arbitrary kid could target and overwrite
	// another device's key row. Bind each kid to its key: recompute the expected
	// kid from the parsed public key and reject a submitted kid that doesn't
	// match. The result is already canonical (base64url, no padding), so it's
	// what we store below.
	signingKID, err := devicePSSOKID(signingPub)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "derive psso signing key id")
	}
	if signingKID != canonicalizeKID(req.SigningKeyID) {
		return &fleet.BadRequestError{Message: "psso registration: signing key id does not match signing key"}
	}
	encryptionKID, err := devicePSSOKID(encryptionPub)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "derive psso encryption key id")
	}
	if encryptionKID != canonicalizeKID(req.EncryptionKeyID) {
		return &fleet.BadRequestError{Message: "psso registration: encryption key id does not match encryption key"}
	}

	// PSSO requires a matching enrolled host; the registration is keyed by the
	// host UUID carried in the (validated) registration token.
	host, err := svc.ds.HostByUUID(ctx, hostUUID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return &fleet.BadRequestError{Message: fmt.Sprintf("psso registration: no enrolled host matches device UUID %q", hostUUID)}
		}
		return ctxerr.Wrap(ctx, err, "look up host by device uuid")
	}

	keys := []fleet.PSSOKey{
		{
			KID:     signingKID,
			KeyType: fleet.PSSOKeyTypeSigning,
			PEM:     req.DeviceSigningKey,
		},
		{
			KID:     encryptionKID,
			KeyType: fleet.PSSOKeyTypeEncryption,
			PEM:     req.DeviceEncryptionKey,
		},
	}
	if err := svc.ds.SetOrUpdatePSSODevice(ctx, host.UUID, keys); err != nil {
		return ctxerr.Wrap(ctx, err, "persist psso device registration")
	}
	return nil
}

// PSSOToken handles the per-sign-in token endpoint. It parses the inbound
// signed JWT, looks up the registered device by kid, verifies the signature,
// consumes the request_nonce, then dispatches on the JWT's claims and returns
// a JWE response in the Apple PSSO format.
func (svc *Service) PSSOToken(ctx context.Context, jwtBytes []byte) ([]byte, error) {
	// skipauth: This is an unauthenticated device-initiated endpoint; the
	// JWT signature against a known device signing pubkey is the auth.
	svc.authz.SkipAuthorization(ctx)

	settings, err := svc.pssoSettingsIfConfigured(ctx, loadSecret)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, errPSSONotConfigured
	}

	if len(jwtBytes) == 0 {
		return nil, &fleet.BadRequestError{Message: "psso token: empty request body"}
	}

	claims, signKey, err := svc.parsePSSOInboundJWT(ctx, jwtBytes)
	if err != nil {
		return nil, err
	}

	// Every token request, regardless of flow, must present a fresh
	// single-use nonce. Consume it before dispatching so a replayed JWS is
	// rejected before any IdP or key work happens.
	if err := svc.consumePSSORequestNonce(ctx, claims.RequestNonce); err != nil {
		return nil, err
	}

	// Key requests/exchanges carry a request_type and are dispatched first.
	switch claims.RequestType {
	case pssoRequestKey:
		return svc.handlePSSOKeyRequest(ctx, signKey.HostUUID, claims)
	case pssoRequestExchange:
		return svc.handlePSSOKeyExchange(ctx, signKey.HostUUID, claims)
	}

	// PSSO v2 Password login. grant_type=password carries a plaintext password;
	// when the extension enables password encryption Apple uses the JWT-bearer
	// grant and ships the password inside an encrypted embedded assertion. Both
	// land here and differ only in where handlePSSOPasswordLogin reads the
	// password from.
	if claims.GrantType == pssoGrantTypePassword || claims.GrantType == pssoGrantTypeJWTBearer {
		return svc.handlePSSOPasswordLogin(ctx, settings, signKey.HostUUID, claims)
	}
	return nil, &fleet.BadRequestError{Message: "psso token: unsupported grant_type/request_type"}
}

// PSSO grant types in the login request JWT. With plaintext passwords Apple
// sends grant_type=password; when the password is encrypted into the embedded
// assertion it switches to the JWT-bearer grant and the password moves out of
// the top-level claim into the (encrypted) assertion.
const (
	pssoGrantTypePassword  = "password"                                    //nolint:gosec // G101 not a credential, a grant type
	pssoGrantTypeJWTBearer = "urn:ietf:params:oauth:grant-type:jwt-bearer" //nolint:gosec // G101 not a credential, a grant type
)

// JWE header `typ` media types. The first two are responses Fleet returns; the
// last is the embedded login assertion the device sends when password
// encryption is enabled.
const (
	pssoTypLoginResponse           = "platformsso-login-response+jwt"
	pssoTypKeyResponse             = "platformsso-key-response+jwt"
	pssoTypEncryptedLoginAssertion = "platformsso-encrypted-login-assertion+jwt"
)

// pssoDefaultTokenTTL is the id_token / refresh_token lifetime used when the
// upstream IdP doesn't return an expires_in.
const pssoDefaultTokenTTL = time.Hour

// pssoAccountClaimPrefix namespaces the IdP claims Fleet forwards into the
// minted id_token so they can be referenced from the profile's
// TokenToUserMapping (e.g. mapping AccountName to a custom "accountUsername"
// claim for the macOS short name). Only claims whose names begin with this
// prefix (case-insensitive) cross the IdP -> Fleet-signed-token boundary; no
// registered OIDC/JWT claim uses it, so it can't collide with reserved claims.
const pssoAccountClaimPrefix = "account"

// pssoIDTokenIssuer returns the value the device validates the login-response
// id_token `iss` claim against. Apple's login configuration derives the issuer
// from the extension's configured hostname — a bare hostname with no scheme —
// so the configured IssuerURL is reduced to its host.
func pssoIDTokenIssuer(settings *fleet.PSSOSettings) string {
	// Hostname() (not Host) so a non-default port is dropped: the extension
	// derives the issuer from the BaseURL via Swift's URL.host, which excludes
	// the port. Returning Host here would mint iss with the port and the device
	// would reject the id_token on mismatch.
	if u, err := url.Parse(settings.IssuerURL); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return strings.TrimSuffix(settings.IssuerURL, "/")
}

// pssoIdPClientFromSettings builds the upstream IdP client for the password
// login flow from the current settings, so config changes apply without a
// restart. Returns the interface so an alternate backend (e.g. an LDAP bind
// client for IdPs that reject ROPG) can be selected here later. Tests fake
// the upstream IdP at the network boundary via PSSOOIDCROPGClient.HTTPClient.
func pssoIdPClientFromSettings(settings *fleet.PSSOSettings) fleet.PSSOIdPClient {
	return PSSOOIDCROPGClient{
		TokenURL:     settings.IdPTokenURL,
		ClientID:     settings.IdPClientID,
		ClientSecret: settings.IdPClientSecret,
		Scopes:       settings.IdPScopes,
	}
}

// buildPSSOIDTokenClaims assembles the claim set for the id_token Fleet mints
// and signs in the login response. It forwards the IdP's standard identity
// claims plus any namespaced "account*" custom claims (so the profile's
// TokenToUserMapping can map the macOS short name / full name to them), then
// sets Fleet's own iss/sub/aud/nonce/iat/exp last so a misconfigured or
// malicious IdP can never override the claims the device validates.
func buildPSSOIDTokenClaims(idpClaims *fleet.PSSOClaims, issuer, audience, nonce string, now time.Time, expiresIn int) jwt.MapClaims {
	out := jwt.MapClaims{
		"email":              idpClaims.Email,
		"name":               idpClaims.Name,
		"preferred_username": idpClaims.PreferredUsername,
	}
	for k, v := range idpClaims.Extra {
		if strings.HasPrefix(strings.ToLower(k), pssoAccountClaimPrefix) {
			out[k] = v
		}
	}
	out["iss"] = issuer
	out["sub"] = idpClaims.Subject
	out["aud"] = audience // request iss == the extension's clientID
	out["nonce"] = nonce
	out["iat"] = now.Unix()
	out["exp"] = now.Add(time.Duration(expiresIn) * time.Second).Unix()
	return out
}

// handlePSSOPasswordLogin services a PSSO v2 Password login request. The
// extension sends a signed JWT carrying the plaintext password (the JWS is the
// integrity/authenticity envelope; transport is TLS) and a jwe_crypto recipe.
// Fleet validates the password against the upstream IdP, then returns the
// resulting OIDC claims as a server-signed JWT wrapped in a JWE encrypted per
// that recipe.
func (svc *Service) handlePSSOPasswordLogin(ctx context.Context, settings *fleet.PSSOSettings, hostUUID string, claims *pssoTokenClaims) ([]byte, error) {
	if claims.JWECrypto == nil || claims.JWECrypto.APV == "" {
		return nil, &fleet.BadRequestError{Message: "psso password login: missing jwe_crypto recipe"}
	}
	if claims.JWECrypto.Alg != "ECDH-ES" || claims.JWECrypto.Enc != "A256GCM" {
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("psso password login: unsupported jwe_crypto %q/%q", claims.JWECrypto.Alg, claims.JWECrypto.Enc)}
	}

	username := claims.Username
	if username == "" {
		username = claims.Subject
	}

	password, err := svc.resolvePSSOLoginPassword(ctx, claims)
	if err != nil {
		return nil, err
	}
	if username == "" || password == "" {
		return nil, &fleet.BadRequestError{Message: "psso password login: missing username or password"}
	}

	idpClient := pssoIdPClientFromSettings(settings)
	idpClaims, err := idpClient.ValidatePasswordAndGetClaims(ctx, username, password)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "psso password validation")
	}

	recipientPub, err := svc.resolvePSSOEncryptionKey(ctx, hostUUID, claims.JWECrypto.APV)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "resolve device encryption pubkey")
	}

	// Per Apple's JWE login-response doc, the response id_token is verified by
	// the device against jwksEndpointURL (Fleet's JWKS). The upstream IdP's
	// id_token is signed by the IdP's key and would not verify there, so Fleet
	// mints its own id_token. The device validates: nonce == request nonce,
	// iss == the profile issuer (hostname, no scheme), aud contains the
	// clientID, iat in the past, exp in the future.
	issuer := pssoIDTokenIssuer(settings)
	expiresIn := idpClaims.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = int(pssoDefaultTokenTTL.Seconds())
	}
	now := time.Now()
	idTokenClaims := buildPSSOIDTokenClaims(idpClaims, issuer, claims.Issuer, claims.Nonce, now, expiresIn)
	idToken, err := svc.signServerJWT(ctx, idTokenClaims)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sign psso id_token")
	}

	refreshToken := idpClaims.RefreshToken
	if refreshToken == "" {
		// The device treats this as opaque; mint a placeholder when the IdP
		// didn't return one (e.g. offline_access not granted).
		var buf [32]byte
		if _, err := rand.Read(buf[:]); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "generate psso refresh token")
		}
		refreshToken = base64.RawURLEncoding.EncodeToString(buf[:])
	}

	// The JWE plaintext is the OAuth token response Apple expects, not a bare
	// JWT: id_token (verified), refresh_token (opaque, used for SSO renewal),
	// and the token lifetimes.
	payload, err := json.Marshal(map[string]any{
		"id_token":                 string(idToken),
		"refresh_token":            refreshToken,
		"token_type":               "Bearer",
		"expires_in":               expiresIn,
		"refresh_token_expires_in": expiresIn,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshal psso login response")
	}

	jwe, err := buildPSSOResponseJWE(payload, recipientPub, claims.JWECrypto.APV, pssoTypLoginResponse)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build psso login response jwe")
	}
	return jwe, nil
}

// resolvePSSOLoginPassword returns the plaintext password for a password login.
// With password encryption disabled it's the plaintext Password claim. With it
// enabled the Password claim is empty and the password lives in the encrypted
// embedded assertion, which Fleet decrypts with its PSSO encryption key. The
// username is always taken from the (signed) outer JWT, not the assertion.
func (svc *Service) resolvePSSOLoginPassword(ctx context.Context, claims *pssoTokenClaims) (string, error) {
	if claims.Password != "" {
		return claims.Password, nil
	}
	if claims.Assertion == "" {
		return "", nil
	}
	encKey, _, err := svc.getPSSOEncryptionKey(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "load psso encryption key")
	}
	plaintext, err := decryptPSSOInboundJWE([]byte(claims.Assertion), encKey)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "decrypt psso login assertion")
	}
	password, err := parseEmbeddedAssertionPassword(plaintext)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "parse psso login assertion")
	}
	return password, nil
}

// handlePSSOKeyRequest services a PSSO 2.0 key request (request_type
// "key_request", key_purpose "user_unlock"). Per Apple's "Supporting key
// requests and key exchange requests" doc, Fleet provisions a fresh EC256 key
// pair, certifies its public half, and returns {certificate, iat, exp,
// key_context} in a JWE (typ=platformsso-key-response+jwt) encrypted to the
// device. key_context carries the provisioned PRIVATE key, sealed under a
// server key, so the later key exchange can recover it statelessly.
func (svc *Service) handlePSSOKeyRequest(ctx context.Context, hostUUID string, claims *pssoTokenClaims) ([]byte, error) {
	if claims.JWECrypto == nil || claims.JWECrypto.APV == "" {
		return nil, &fleet.BadRequestError{Message: "psso key request: missing jwe_crypto recipe"}
	}
	encPub, err := svc.resolvePSSOEncryptionKey(ctx, hostUUID, claims.JWECrypto.APV)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "resolve device encryption pubkey")
	}

	provisioned, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generate provisioned key")
	}
	certDER, err := svc.issuePSSOProvisionedCertificate(ctx, &provisioned.PublicKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "issue psso provisioned certificate")
	}

	signingKey, _, err := svc.getPSSOSigningKey(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load psso signing key")
	}
	kcKey, err := deriveKeyContextKey(signingKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "derive key_context key")
	}
	keyContext, err := sealKeyContext(provisioned, hostUUID, pssoKeyPurposeUserUnlock, kcKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "seal key_context")
	}

	now := time.Now()
	payload, err := json.Marshal(map[string]any{
		"certificate": base64.RawURLEncoding.EncodeToString(certDER),
		"iat":         now.Unix(),
		"exp":         now.Add(5 * time.Minute).Unix(),
		"key_context": keyContext,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshal key_request payload")
	}

	jwe, err := buildPSSOResponseJWE(payload, encPub, claims.JWECrypto.APV, pssoTypKeyResponse)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build key_request JWE")
	}
	return jwe, nil
}

// issuePSSOProvisionedCertificate issues an X.509 certificate over a
// server-provisioned public key, signed by Fleet's persisted PSSO CA. This is
// the certificate returned in a key-request response; the device uses its public
// key for its half of the unlock-key Diffie-Hellman.
func (svc *Service) issuePSSOProvisionedCertificate(ctx context.Context, provisionedKey *ecdsa.PublicKey) ([]byte, error) {
	caKey, caCert, err := svc.getPSSOCA(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load psso ca for cert issuance")
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generate psso cert serial")
	}
	now := time.Now()
	devTmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "Fleet PSSO Device Key"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
	}
	devDER, err := x509.CreateCertificate(rand.Reader, devTmpl, caCert, provisionedKey, caKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create psso provisioned certificate")
	}
	return devDER, nil
}

// handlePSSOKeyExchange services a PSSO 2.0 key exchange (request_type
// "key_exchange"). The device sends its DH public key (other_publickey) plus
// the key_context Fleet issued during the key request. Fleet recovers the
// provisioned private key from key_context, computes the raw ECDH shared
// secret against other_publickey (this is the unlock key), and returns
// {iat, exp, key, key_context} in the same JWE envelope.
func (svc *Service) handlePSSOKeyExchange(ctx context.Context, hostUUID string, claims *pssoTokenClaims) ([]byte, error) {
	if claims.JWECrypto == nil || claims.JWECrypto.APV == "" {
		return nil, &fleet.BadRequestError{Message: "psso key exchange: missing jwe_crypto recipe"}
	}
	if claims.OtherPublicKey == "" || claims.KeyContext == "" {
		return nil, &fleet.BadRequestError{Message: "psso key exchange: missing other_publickey or key_context"}
	}

	signingKey, _, err := svc.getPSSOSigningKey(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load psso signing key")
	}
	kcKey, err := deriveKeyContextKey(signingKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "derive key_context key")
	}
	kc, provisioned, err := openKeyContext(claims.KeyContext, kcKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "open key_context")
	}
	// Bind the sealed key_context to the device: reject a context replayed by, or
	// fetched onto, any device other than the one it was issued to.
	if kc.HostUUID != hostUUID {
		return nil, &fleet.BadRequestError{Message: "psso key exchange: key_context host mismatch"}
	}
	if kc.KeyPurpose != pssoKeyPurposeUserUnlock {
		return nil, &fleet.BadRequestError{Message: "psso key exchange: unsupported key_context purpose"}
	}

	otherRaw, err := decodeBase64Flexible(claims.OtherPublicKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decode other_publickey")
	}
	shared, err := computeECDHShared(provisioned, otherRaw)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "compute key exchange shared secret")
	}

	encPub, err := svc.resolvePSSOEncryptionKey(ctx, hostUUID, claims.JWECrypto.APV)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "resolve device encryption pubkey")
	}

	now := time.Now()
	payload, err := json.Marshal(map[string]any{
		"iat":         now.Unix(),
		"exp":         now.Add(5 * time.Minute).Unix(),
		"key":         base64.StdEncoding.EncodeToString(shared),
		"key_context": claims.KeyContext,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshal key_exchange payload")
	}

	jwe, err := buildPSSOResponseJWE(payload, encPub, claims.JWECrypto.APV, pssoTypKeyResponse)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build key_exchange JWE")
	}
	return jwe, nil
}

// PSSOJWKS returns the JWKS JSON with Fleet's PSSO signing public key. When
// the feature is not configured it returns a 404 (not a 400 like the
// device-facing endpoints) so the endpoint is indistinguishable from absent.
func (svc *Service) PSSOJWKS(ctx context.Context) ([]byte, error) {
	// skipauth: This is an unauthenticated public endpoint serving only the
	// signing public key — there is no caller identity to authorize.
	svc.authz.SkipAuthorization(ctx)
	settings, err := svc.pssoSettingsIfConfigured(ctx, skipSecret)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, &notFoundError{}
	}

	key, kid, err := svc.getPSSOSigningKey(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load psso signing key")
	}

	encKey, encKID, err := svc.getPSSOEncryptionKey(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load psso encryption key")
	}

	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
		{
			Key:       &key.PublicKey,
			KeyID:     kid,
			Algorithm: pssoSigningAlg,
			Use:       "sig",
		},
		// The extension sets this key as loginRequestEncryptionPublicKey and
		// encrypts the password to it (ECDH-ES), so the password is never visible
		// to a TLS-terminating proxy.
		{
			Key:       &encKey.PublicKey,
			KeyID:     encKID,
			Algorithm: pssoEncryptionAlg,
			Use:       "enc",
		},
	}}
	return json.Marshal(jwks)
}

// pssoAASA mirrors the apple-app-site-association shape Apple's framework
// consumes for PSSO. PSSO validates the extension's authsrv: entitlement.
type pssoAASA struct {
	AuthSrv pssoAASAApps `json:"authsrv"`
}

type pssoAASAApps struct {
	Apps []string `json:"apps"`
}

// PSSOAASA returns the apple-app-site-association JSON Apple's framework
// uses to validate the extension's authsrv: entitlement against Fleet's
// hostname. Returns 404 when the feature is not configured. Note Apple's CDN
// caches this document for hours, so hosts may see a config change with a
// 6–24h delay.
func (svc *Service) PSSOAASA(ctx context.Context) ([]byte, error) {
	// skipauth: This is an unauthenticated public endpoint — Apple's
	// framework fetches it anonymously to validate the extension binding.
	svc.authz.SkipAuthorization(ctx)

	settings, err := svc.pssoSettingsIfConfigured(ctx, skipSecret)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, &notFoundError{}
	}

	ids := []string{fleetTeamID + "." + appBundleID, fleetTeamID + "." + extensionBundleID}
	doc := pssoAASA{
		AuthSrv: pssoAASAApps{
			Apps: ids,
		},
	}
	return json.Marshal(doc)
}
