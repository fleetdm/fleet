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
	pssoSigningCurve = "P-256"
	pssoSigningAlg   = "ES256"

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
// encoding of pub. This matches the kid format the extension sends with its
// JWTs (SHA-256 of the public key bytes, base64'd).
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

// SetPSSONonceStore wires the Redis-backed PSSO nonce store. Intended to be
// called from cmd/fleet right after eeservice.NewService so the POC doesn't
// have to expand the NewService signature for an optional collaborator.
func (svc *Service) SetPSSONonceStore(store fleet.PSSONonceStore) {
	svc.pssoNonceStore = store
}

// SetPSSOIdPClient wires the upstream IdP client (a generic OIDC ROPG
// client in production, the deterministic stub in tests). Same rationale as
// SetPSSONonceStore.
func (svc *Service) SetPSSOIdPClient(client fleet.PSSOIdPClient) {
	svc.pssoIdPClient = client
}

// pssoNonceTTL is how long an issued nonce remains valid before it's
// rejected. Five minutes covers both registration (browser round-trip
// through the upstream IdP) and sign-in (extension immediate use).
const pssoNonceTTL = 5 * time.Minute

// PSSONonce mints a fresh 32-byte base64url nonce, persists it with a short
// TTL via the wired PSSONonceStore, and returns it to the caller. The
// extension embeds this nonce in subsequent JWT claims to prevent replay.
func (svc *Service) PSSONonce(ctx context.Context) (string, error) {
	// skipauth: This is an unauthenticated endpoint hit by the Mac extension
	// before any user identity is established.
	svc.authz.SkipAuthorization(ctx)

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

// PSSORegisterBegin builds the redirect URL the Mac extension's WebView
// should follow to start the upstream IdP's OAuth code flow. The returned URL
// embeds a fresh server-issued nonce in the `state` parameter so we can
// detect replay when the extension calls PSSORegisterComplete.
func (svc *Service) PSSORegisterBegin(ctx context.Context) (string, error) {
	// skipauth: This is an unauthenticated endpoint hit by the Mac extension's
	// WebView before user identity exists.
	svc.authz.SkipAuthorization(ctx)

	cfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "load app config for psso register")
	}
	pcfg := cfg.PSSOSettings
	if pcfg == nil || pcfg.IdPAuthorizeURL == "" || pcfg.IdPClientID == "" || pcfg.IssuerURL == "" {
		return "", &fleet.BadRequestError{Message: "PSSO is not configured: idp_authorize_url, idp_client_id, and issuer_url are required"}
	}

	state, err := svc.PSSONonce(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "issue psso register state nonce")
	}

	scopes := pcfg.IdPScopes
	if scopes == "" {
		scopes = defaultOIDCScopes
	}

	params := url.Values{}
	params.Set("client_id", pcfg.IdPClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", pcfg.IssuerURL+"/mdm/apple/psso/register")
	params.Set("scope", scopes)
	params.Set("state", state)

	sep := "?"
	if strings.Contains(pcfg.IdPAuthorizeURL, "?") {
		sep = "&"
	}
	return pcfg.IdPAuthorizeURL + sep + params.Encode(), nil
}

// PSSORegisterComplete consumes the device-key enrollment POST from the Mac
// extension: it resolves the enrolled host from the hardware device UUID,
// mints a KeyExchangeKey, and persists the device record + KeyID rows.
//
// Password-mode registration carries no OAuth code/state — the extension
// simply submits the public halves of its Secure Enclave signing and
// encryption keys. User identity is established later, on each password login
// at the token endpoint.
func (svc *Service) PSSORegisterComplete(ctx context.Context, req fleet.PSSORegisterRequest) error {
	// skipauth: This is an unauthenticated device-initiated endpoint. The
	// device proves itself later by signing token requests with the signing
	// key registered here, verified against the kid.
	svc.authz.SkipAuthorization(ctx)

	if req.DeviceUUID == "" || req.DeviceSigningKey == "" || req.DeviceEncryptionKey == "" || req.SignKeyID == "" || req.EncKeyID == "" {
		return &fleet.BadRequestError{Message: "missing required psso register fields"}
	}

	// Resolve host_id from device UUID. PSSO requires a matching enrolled host
	// since the device record is keyed by host_id.
	host, err := svc.ds.HostLiteByIdentifier(ctx, req.DeviceUUID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return &fleet.BadRequestError{Message: fmt.Sprintf("psso register: no enrolled host matches device UUID %q", req.DeviceUUID)}
		}
		return ctxerr.Wrap(ctx, err, "look up host by device uuid")
	}

	// Mint a 32-byte KeyExchangeKey. This is the v2 secret returned to the
	// device on its first key_request and reused for symmetric session keys
	// thereafter.
	var kek [32]byte
	if _, err := rand.Read(kek[:]); err != nil {
		return ctxerr.Wrap(ctx, err, "generate key exchange key")
	}

	device := fleet.PSSODevice{
		HostID:           host.ID,
		DeviceUUID:       req.DeviceUUID,
		SigningKeyPEM:    req.DeviceSigningKey,
		EncryptionKeyPEM: req.DeviceEncryptionKey,
		KeyExchangeKey:   kek[:],
	}
	// Store kids in canonical form so the token endpoint's lookup (which
	// canonicalizes the JWT's kid) matches regardless of base64 padding or
	// alphabet differences between the extension and Apple's framework.
	signKID := fleet.PSSOKeyID{
		KID:     canonicalizeKID(req.SignKeyID),
		HostID:  host.ID,
		KeyType: fleet.PSSOKeyTypeSigning,
		PEM:     req.DeviceSigningKey,
	}
	encKID := fleet.PSSOKeyID{
		KID:     canonicalizeKID(req.EncKeyID),
		HostID:  host.ID,
		KeyType: fleet.PSSOKeyTypeEncryption,
		PEM:     req.DeviceEncryptionKey,
	}
	if err := svc.ds.SetOrUpdatePSSODevice(ctx, device, signKID, encKID); err != nil {
		return ctxerr.Wrap(ctx, err, "persist psso device registration")
	}
	return nil
}

// PSSOToken handles the per-sign-in token endpoint. It parses the inbound
// signed JWT, looks up the registered device by kid, verifies the signature,
// then dispatches on the JWT's request_type claim and returns a JWE
// response in the Apple PSSO format.
func (svc *Service) PSSOToken(ctx context.Context, jwtBytes []byte) ([]byte, error) {
	// skipauth: This is an unauthenticated device-initiated endpoint; the
	// JWT signature against a known device signing pubkey is the auth.
	svc.authz.SkipAuthorization(ctx)

	if len(jwtBytes) == 0 {
		return nil, &fleet.BadRequestError{Message: "psso token: empty request body"}
	}

	claims, device, err := svc.parsePSSOInboundJWT(ctx, jwtBytes)
	if err != nil {
		return nil, err
	}

	// PSSO v2 Password login: a single grant_type=password round trip carrying
	// a plaintext password and a jwe_crypto response recipe.
	if claims.GrantType == pssoGrantTypePassword {
		return svc.handlePSSOPasswordLogin(ctx, device, claims)
	}

	// Legacy request_type handshake model — retained but not exercised by the
	// Password flow.
	switch claims.RequestType {
	case pssoRequestKey:
		return svc.handlePSSOKeyRequest(ctx, device, claims)
	case pssoRequestExchange:
		return svc.handlePSSOKeyExchange(ctx, device, claims)
	case pssoRequestPassword:
		return svc.handlePSSOPasswordRequest(ctx, device, claims)
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
// from the profile's IssuerHostname — a bare hostname with no scheme — so the
// configured IssuerURL is reduced to its host.
func (svc *Service) pssoIDTokenIssuer(ctx context.Context) (string, error) {
	cfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "load app config for psso issuer")
	}
	if cfg.PSSOSettings == nil || cfg.PSSOSettings.IssuerURL == "" {
		return "", ctxerr.New(ctx, "psso issuer_url not configured")
	}
	if u, err := url.Parse(cfg.PSSOSettings.IssuerURL); err == nil && u.Host != "" {
		return u.Host, nil
	}
	return strings.TrimSuffix(cfg.PSSOSettings.IssuerURL, "/"), nil
}

// handlePSSOPasswordLogin services a PSSO v2 Password login request. The
// extension sends a signed JWT carrying the plaintext password (the JWS is the
// integrity/authenticity envelope; transport is TLS) and a jwe_crypto recipe.
// Fleet validates the password against the upstream IdP, then returns the
// resulting OIDC claims as a server-signed JWT wrapped in a JWE encrypted per
// that recipe.
func (svc *Service) handlePSSOPasswordLogin(ctx context.Context, device *fleet.PSSODevice, claims *pssoTokenClaims) ([]byte, error) {
	if svc.pssoIdPClient == nil {
		return nil, ctxerr.New(ctx, "psso idp client not configured")
	}
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

	// Best-effort single-use nonce check. request_nonce is the value Fleet
	// issued from /mdm/apple/psso/nonce that the extension echoes here. It is
	// not hard-enforced for the POC: the exact nonce the AppSSOAgent replays is
	// still being confirmed end-to-end, so a miss is logged rather than
	// rejected to keep password validation testable. Enforce before GA.
	if claims.RequestNonce != "" && svc.pssoNonceStore != nil {
		ok, err := svc.pssoNonceStore.Consume(ctx, claims.RequestNonce)
		if err != nil {
			svc.logger.WarnContext(ctx, "psso password login: nonce consume error", "err", err)
		} else if !ok {
			svc.logger.WarnContext(ctx, "psso password login: request_nonce not recognized", "request_nonce", claims.RequestNonce)
		}
	}

	idpClaims, err := svc.pssoIdPClient.ValidatePasswordAndGetClaims(ctx, username, claims.Password)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "psso password validation")
	}

	recipientPub, err := parseECPublicKeyPEM([]byte(device.EncryptionKeyPEM))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse device encryption pubkey")
	}

	// Per Apple's JWE login-response doc, the response id_token is verified by
	// the device against jwksEndpointURL (Fleet's JWKS). The upstream IdP's
	// id_token is signed by the IdP's key and would not verify there, so Fleet
	// mints its own id_token. The device validates: nonce == request nonce,
	// iss == the profile issuer (hostname, no scheme), aud contains the
	// clientID, iat in the past, exp in the future.
	issuer, err := svc.pssoIDTokenIssuer(ctx)
	if err != nil {
		return nil, err
	}
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
// requests and key exchange requests" doc, the response is a JWE
// (typ=platformsso-key-response+jwt) whose decrypted body is
// {certificate, iat, exp}: an X.509 certificate over the device's provisioned
// public key, issued by Fleet. The device verifies/stores this to enable the
// SSO unlock key.
func (svc *Service) handlePSSOKeyRequest(ctx context.Context, device *fleet.PSSODevice, claims *pssoTokenClaims) ([]byte, error) {
	if claims.JWECrypto == nil || claims.JWECrypto.APV == "" {
		return nil, &fleet.BadRequestError{Message: "psso key request: missing jwe_crypto recipe"}
	}
	encPub, err := parseECPublicKeyPEM([]byte(device.EncryptionKeyPEM))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse device encryption pubkey")
	}

	certDER, err := svc.issuePSSODeviceCertificate(ctx, encPub)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "issue psso device certificate")
	}

	now := time.Now()
	payload, err := json.Marshal(map[string]any{
		"certificate": base64.RawURLEncoding.EncodeToString(certDER),
		"iat":         now.Unix(),
		"exp":         now.Add(5 * time.Minute).Unix(),
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

// issuePSSODeviceCertificate issues an X.509 certificate over the device's
// provisioned public key, signed by Fleet's PSSO signing key acting as a CA.
// This is the certificate returned in a key-request response.
func (svc *Service) issuePSSODeviceCertificate(ctx context.Context, deviceKey *ecdsa.PublicKey) ([]byte, error) {
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
	devDER, err := x509.CreateCertificate(rand.Reader, devTmpl, caCert, deviceKey, caKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create psso device certificate")
	}
	return devDER, nil
}

// handlePSSOKeyExchange validates a symmetric handshake and returns a
// session JWE. For the POC, this is a minimal "you've proven you can
// derive the same session key" round trip.
func (svc *Service) handlePSSOKeyExchange(_ context.Context, device *fleet.PSSODevice, claims *pssoTokenClaims) ([]byte, error) {
	sessionKey, err := deriveSessionKey(device.KeyExchangeKey, []byte(claims.RequestNonce))
	if err != nil {
		return nil, fmt.Errorf("derive session key: %w", err)
	}
	payload, err := json.Marshal(struct {
		Status string `json:"status"`
	}{Status: "ok"})
	if err != nil {
		return nil, fmt.Errorf("marshal key_exchange payload: %w", err)
	}
	return buildSymmetricJWE(payload, sessionKey)
}

// handlePSSOPasswordRequest decrypts the password the device sent under
// the previously-established session key, validates it against the
// upstream IdP via the wired PSSOIdPClient, and returns the resulting
// claims as a JWT-inside-JWE.
func (svc *Service) handlePSSOPasswordRequest(ctx context.Context, device *fleet.PSSODevice, claims *pssoTokenClaims) ([]byte, error) {
	if svc.pssoIdPClient == nil {
		return nil, ctxerr.New(ctx, "psso idp client not configured")
	}
	if claims.Username == "" || claims.EncryptedPwd == "" {
		return nil, &fleet.BadRequestError{Message: "psso password_request missing username or encrypted_password"}
	}

	sessionKey, err := deriveSessionKey(device.KeyExchangeKey, []byte(claims.RequestNonce))
	if err != nil {
		return nil, fmt.Errorf("derive session key: %w", err)
	}
	pwdPlain, err := decryptSymmetricBlob([]byte(claims.EncryptedPwd), sessionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt password blob: %w", err)
	}

	idpClaims, err := svc.pssoIdPClient.ValidatePasswordAndGetClaims(ctx, claims.Username, string(pwdPlain))
	if err != nil {
		return nil, err
	}

	// Wrap the OIDC-shaped claims in a server-signed JWT, then JWE-wrap the
	// JWT under the session key.
	innerToken, err := svc.signServerJWT(ctx, jwt.MapClaims{
		"sub":                idpClaims.Subject,
		"email":              idpClaims.Email,
		"name":               idpClaims.Name,
		"preferred_username": idpClaims.PreferredUsername,
	})
	if err != nil {
		return nil, err
	}
	return buildSymmetricJWE(innerToken, sessionKey)
}

// PSSOJWKS returns the JWKS JSON with Fleet's PSSO signing public key.
func (svc *Service) PSSOJWKS(ctx context.Context) ([]byte, error) {
	// skipauth: This is an unauthenticated public endpoint serving only the
	// signing public key — there is no caller identity to authorize.
	svc.authz.SkipAuthorization(ctx)

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
// hostname.
func (svc *Service) PSSOAASA(ctx context.Context) ([]byte, error) {
	// skipauth: This is an unauthenticated public endpoint — Apple's
	// framework fetches it anonymously to validate the extension binding.
	svc.authz.SkipAuthorization(ctx)
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
