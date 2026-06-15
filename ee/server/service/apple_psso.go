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
	jose "github.com/go-jose/go-jose/v3"
)

// pssoServiceState holds the lazily-loaded PSSO signing key.
//
// TODO(psso bootstrap): the current lazy-mint behavior runs on the first call
// to any method that reaches getOrMintPSSOSigningKey — most commonly
// PSSOJWKS, which is an unauthenticated public endpoint. That means an
// unauthenticated GET triggers a write + KMS roundtrip if the key doesn't
// exist yet. Acceptable for POC but worth revisiting before GA. Alternatives
// to consider:
//   - mint when an admin enables PSSO via AppConfig.PSSOSettings.Enabled = true
//   - mint on the first device registration request
//   - explicit `fleetctl psso bootstrap` step
type pssoServiceState struct {
	mu         sync.Mutex
	signingKey *ecdsa.PrivateKey
	kid        string
}

const (
	pssoSigningAlg = "ES256"

	// TODO: It's not clear if we need the overall app bundle ID or not either. We'll add it just in case
	bundleID1 = "com.fleetdm.pssotesting"
	bundleID2 = "com.fleetdm.pssotesting.extension"

	// TODO:  Not sure if I actually need to use the team or my private user one so we'll define
	// both for now...
	teamID1 = "5K28R5ZUK5"
	teamID2 = "B34KW9D28L"
)

// getOrMintPSSOSigningKey returns Fleet's PSSO signing key, loading it from
// mdm_config_assets or minting+persisting a fresh one if not present.
func (svc *Service) getOrMintPSSOSigningKey(ctx context.Context) (*ecdsa.PrivateKey, string, error) {
	svc.pssoState.mu.Lock()
	defer svc.pssoState.mu.Unlock()

	if svc.pssoState.signingKey != nil {
		return svc.pssoState.signingKey, svc.pssoState.kid, nil
	}

	// Try load.
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetPSSOSigningKey},
		nil,
	)
	if err == nil {
		asset, ok := assets[fleet.MDMAssetPSSOSigningKey]
		if ok && len(asset.Value) > 0 {
			key, kid, err := parsePSSOSigningKeyPEM(asset.Value)
			if err != nil {
				return nil, "", ctxerr.Wrap(ctx, err, "parse stored psso signing key")
			}
			svc.pssoState.signingKey = key
			svc.pssoState.kid = kid
			return key, kid, nil
		}
	} else if !isAssetNotFound(err) {
		return nil, "", ctxerr.Wrap(ctx, err, "get psso signing key asset")
	}

	// Mint a fresh key and persist.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "generate psso signing key")
	}
	pemBytes, kid, err := encodePSSOSigningKeyPEM(key)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "encode psso signing key")
	}
	if err := svc.ds.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{
		Name:  fleet.MDMAssetPSSOSigningKey,
		Value: pemBytes,
	}); err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "persist psso signing key")
	}
	svc.pssoState.signingKey = key
	svc.pssoState.kid = kid
	return key, kid, nil
}

// encodePSSOSigningKeyPEM serializes a P-256 private key to PEM and returns
// the bytes plus the kid (base64url-nopad SHA-256 of the DER-encoded public
// key).
func encodePSSOSigningKeyPEM(key *ecdsa.PrivateKey) ([]byte, string, error) {
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, "", err
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	kid, err := computeKID(&key.PublicKey)
	if err != nil {
		return nil, "", err
	}
	return pemBytes, kid, nil
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
// mdm_config_assets row was absent. The datastore returns a partial-result
// error in that case.
func isAssetNotFound(err error) bool {
	if err == nil {
		return false
	}
	// fleet.IsNotFound catches the typed not-found case; the partial-result
	// error from GetAllMDMConfigAssetsByName matches via string content.
	if fleet.IsNotFound(err) {
		return true
	}
	return errors.Is(err, sql.ErrNoRows)
}

// pssoSettingsIfConfigured returns the PSSO settings from the current
// AppConfig when the feature is enabled and carries everything the flows
// need; otherwise it returns nil. Read per request so enabling, disabling,
// or repointing the IdP takes effect without a server restart.
func (svc *Service) pssoSettingsIfConfigured(ctx context.Context) (*fleet.PSSOSettings, error) {
	cfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load app config for psso")
	}
	s := cfg.PSSOSettings
	if s == nil || !s.Enabled || s.IssuerURL == "" || s.IdPTokenURL == "" || s.IdPClientID == "" || s.IdPClientSecret == "" {
		return nil, nil
	}
	return s, nil
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

	settings, err := svc.pssoSettingsIfConfigured(ctx)
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

	settings, err := svc.pssoSettingsIfConfigured(ctx)
	if err != nil {
		return err
	}
	if settings == nil {
		return errPSSONotConfigured
	}

	if req.DeviceUUID == "" || req.DeviceSigningKey == "" || req.DeviceEncryptionKey == "" || req.SigningKeyID == "" || req.EncryptionKeyID == "" {
		return &fleet.BadRequestError{Message: "missing required psso registration fields"}
	}

	// Reject unparseable key material up front: a bad PEM stored here would
	// otherwise only surface as opaque verification failures at every
	// subsequent login.
	if _, err := parseECPublicKeyPEM([]byte(req.DeviceSigningKey)); err != nil {
		return &fleet.BadRequestError{Message: "psso registration: signing key is not a valid P-256 public key"}
	}
	if _, err := parseECPublicKeyPEM([]byte(req.DeviceEncryptionKey)); err != nil {
		return &fleet.BadRequestError{Message: "psso registration: encryption key is not a valid P-256 public key"}
	}

	// PSSO requires a matching enrolled host; the registration is keyed by the
	// host's UUID.
	host, err := svc.ds.HostByUUID(ctx, req.DeviceUUID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return &fleet.BadRequestError{Message: fmt.Sprintf("psso registration: no enrolled host matches device UUID %q", req.DeviceUUID)}
		}
		return ctxerr.Wrap(ctx, err, "look up host by device uuid")
	}

	// Store kids in canonical form so the token endpoint's lookup (which
	// canonicalizes the JWT's kid) matches regardless of base64 padding or
	// alphabet differences between the extension and Apple's framework.
	keys := []fleet.PSSOKey{
		{
			KID:     canonicalizeKID(req.SigningKeyID),
			KeyType: fleet.PSSOKeyTypeSigning,
			PEM:     req.DeviceSigningKey,
		},
		{
			KID:     canonicalizeKID(req.EncryptionKeyID),
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

	settings, err := svc.pssoSettingsIfConfigured(ctx)
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

	// PSSO v2 Password login: a single grant_type=password round trip carrying
	// a plaintext password and a jwe_crypto response recipe.
	if claims.GrantType == pssoGrantTypePassword {
		return svc.handlePSSOPasswordLogin(ctx, settings, signKey.HostUUID, claims)
	}

	switch claims.RequestType {
	case pssoRequestKey:
		return svc.handlePSSOKeyRequest(ctx, signKey.HostUUID, claims)
	case pssoRequestExchange:
		return svc.handlePSSOKeyExchange(ctx, signKey.HostUUID, claims)
	default:
		return nil, &fleet.BadRequestError{Message: "psso token: unsupported grant_type/request_type"}
	}
}

// pssoGrantTypePassword is the grant_type the extension sends in the login
// request JWT for Password-method PSSO.
const pssoGrantTypePassword = "password"

// JWE header `typ` media types for the responses Fleet returns.
const (
	pssoTypLoginResponse = "platformsso-login-response+jwt"
	pssoTypKeyResponse   = "platformsso-key-response+jwt"
)

// pssoDefaultTokenTTL is the id_token / refresh_token lifetime used when the
// upstream IdP doesn't return an expires_in.
const pssoDefaultTokenTTL = time.Hour

// pssoIDTokenIssuer returns the value the device validates the login-response
// id_token `iss` claim against. Apple's login configuration derives the issuer
// from the extension's configured hostname — a bare hostname with no scheme —
// so the configured IssuerURL is reduced to its host.
func pssoIDTokenIssuer(settings *fleet.PSSOSettings) string {
	if u, err := url.Parse(settings.IssuerURL); err == nil && u.Host != "" {
		return u.Host
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
	if username == "" || claims.Password == "" {
		return nil, &fleet.BadRequestError{Message: "psso password login: missing username or password"}
	}

	idpClient := pssoIdPClientFromSettings(settings)
	idpClaims, err := idpClient.ValidatePasswordAndGetClaims(ctx, username, claims.Password)
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
	idToken, err := svc.signServerJWT(ctx, jwt.MapClaims{
		"iss":                issuer,
		"sub":                idpClaims.Subject,
		"aud":                claims.Issuer, // request iss == the extension's clientID
		"nonce":              claims.Nonce,
		"iat":                now.Unix(),
		"exp":                now.Add(time.Duration(expiresIn) * time.Second).Unix(),
		"email":              idpClaims.Email,
		"name":               idpClaims.Name,
		"preferred_username": idpClaims.PreferredUsername,
	})
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

	signingKey, _, err := svc.getOrMintPSSOSigningKey(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load psso signing key")
	}
	kcKey, err := deriveKeyContextKey(signingKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "derive key_context key")
	}
	keyContext, err := sealKeyContext(provisioned, kcKey)
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
// server-provisioned public key, signed by Fleet's PSSO signing key acting as
// a CA. This is the certificate returned in a key-request response; the device
// uses its public key for its half of the unlock-key Diffie-Hellman.
func (svc *Service) issuePSSOProvisionedCertificate(ctx context.Context, provisionedKey *ecdsa.PublicKey) ([]byte, error) {
	caKey, _, err := svc.getOrMintPSSOSigningKey(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load psso signing key for cert issuance")
	}

	now := time.Now()
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Fleet PSSO CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create psso ca certificate")
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse psso ca certificate")
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generate psso cert serial")
	}
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

	signingKey, _, err := svc.getOrMintPSSOSigningKey(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load psso signing key")
	}
	kcKey, err := deriveKeyContextKey(signingKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "derive key_context key")
	}
	provisioned, err := openKeyContext(claims.KeyContext, kcKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "open key_context")
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

	settings, err := svc.pssoSettingsIfConfigured(ctx)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, &notFoundError{}
	}

	key, kid, err := svc.getOrMintPSSOSigningKey(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load psso signing key")
	}

	jwk := jose.JSONWebKey{
		Key:       &key.PublicKey,
		KeyID:     kid,
		Algorithm: pssoSigningAlg,
		Use:       "sig",
	}
	jwks := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}
	return json.Marshal(jwks)
}

// pssoAASAEntry mirrors the apple-app-site-association shape Apple's
// framework consumes for PSSO. Only webcredentials.apps is required.
type pssoAASA struct {
	WebCredentials pssoAASAApps `json:"webcredentials"`
	AuthSrv        pssoAASAApps `json:"authsrv"`
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

	settings, err := svc.pssoSettingsIfConfigured(ctx)
	if err != nil {
		return nil, err
	}
	if settings == nil {
		return nil, &notFoundError{}
	}

	ids := []string{teamID1 + "." + bundleID1, teamID2 + "." + bundleID1, teamID1 + "." + bundleID2, teamID2 + "." + bundleID2}
	doc := pssoAASA{
		WebCredentials: pssoAASAApps{
			Apps: ids,
		},
		AuthSrv: pssoAASAApps{
			Apps: ids,
		},
	}
	return json.Marshal(doc)
}
