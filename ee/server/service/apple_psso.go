package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
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
	bundleID1         = "com.fleetdm.pssotesting"
	bundleID2         = "com.fleetdm.pssotesting.extension"

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

// PSSORegisterComplete consumes the registration POST from the Mac
// extension. Validates the state nonce, looks up the host by device UUID,
// mints a KeyExchangeKey, and persists the device record + KeyID rows.
//
// For the POC, this trusts that the extension actually navigated through the
// upstream IdP — the OAuth code itself is not exchanged for tokens here.
// Adding code exchange + ID-token verification is a small follow-up but
// gated on getting the device-key persistence end-to-end first.
func (svc *Service) PSSORegisterComplete(ctx context.Context, req fleet.PSSORegisterRequest) error {
	// skipauth: This is an unauthenticated device-initiated endpoint; the
	// authorization is the device's possession of the OAuth code + nonce.
	svc.authz.SkipAuthorization(ctx)

	if svc.pssoNonceStore == nil {
		return ctxerr.New(ctx, "psso nonce store not configured")
	}

	if req.DeviceUUID == "" || req.DeviceSigningKey == "" || req.DeviceEncryptionKey == "" || req.SignKeyID == "" || req.EncKeyID == "" {
		return &fleet.BadRequestError{Message: "missing required psso register fields"}
	}

	// Validate state nonce (replay protection).
	if req.State == "" {
		return &fleet.BadRequestError{Message: "missing state nonce in psso register"}
	}
	ok, err := svc.pssoNonceStore.Consume(ctx, req.State)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "consume psso register state nonce")
	}
	if !ok {
		return fleet.NewAuthFailedError("psso register state nonce is expired or unknown")
	}

	// Resolve host_id from device UUID. PSSO requires a matching enrolled host.
	host, err := svc.ds.HostLiteByIdentifier(ctx, req.DeviceUUID)
	if err != nil {
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
	signKID := fleet.PSSOKeyID{
		KID:     req.SignKeyID,
		HostID:  host.ID,
		KeyType: fleet.PSSOKeyTypeSigning,
		PEM:     req.DeviceSigningKey,
	}
	encKID := fleet.PSSOKeyID{
		KID:     req.EncKeyID,
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

	switch claims.RequestType {
	case pssoRequestKey:
		return svc.handlePSSOKeyRequest(ctx, device)
	case pssoRequestExchange:
		return svc.handlePSSOKeyExchange(ctx, device, claims)
	case pssoRequestPassword:
		return svc.handlePSSOPasswordRequest(ctx, device, claims)
	default:
		return nil, &fleet.BadRequestError{Message: "psso token: unknown request_type"}
	}
}

// handlePSSOKeyRequest returns the device's KeyExchangeKey encrypted to its
// encryption pubkey. This is the v2 handshake: the device decrypts the KEK
// with the private half it holds in Secure Enclave, then derives session
// keys from the KEK for subsequent symmetric operations.
func (svc *Service) handlePSSOKeyRequest(ctx context.Context, device *fleet.PSSODevice) ([]byte, error) {
	encPub, err := parseECPublicKeyPEM([]byte(device.EncryptionKeyPEM))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse device encryption pubkey")
	}
	payload, err := json.Marshal(struct {
		KeyExchangeKey []byte `json:"key_exchange_key"`
	}{
		KeyExchangeKey: device.KeyExchangeKey,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshal key_request payload")
	}
	jwe, err := buildAsymmetricJWE(payload, encPub, "" /* kid filled by go-jose if encPub has KeyID */)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build key_request JWE")
	}
	return jwe, nil
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
	AuthSrv pssoAASAApps `json:"authsrv"`
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
