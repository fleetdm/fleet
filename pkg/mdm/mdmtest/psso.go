package mdmtest

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/psso/pssocrypto"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	jose "github.com/go-jose/go-jose/v3"
	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
)

// HTTP paths for the Apple Platform SSO (PSSO) endpoints. These mirror the
// constants in server/service/apple_psso.go; the integration test and any
// load test exercise them against the real server, so drift fails fast.
const (
	pssoNoncePath        = "/api/mdm/apple/psso/nonce"
	pssoRegistrationPath = "/api/mdm/apple/psso/registration"
	pssoTokenPath        = "/api/mdm/apple/psso/token" //nolint:gosec // G101 false positive, this is a URL path
	pssoJWKSPath         = "/api/mdm/apple/psso/jwks"
	pssoAASAPath         = "/.well-known/apple-app-site-association"
)

// TestApplePSSODevice simulates the macOS side of Apple Platform SSO against a
// Fleet server: device registration, password login (against the proxied IdP),
// the offline-unlock key request and key exchange, plus validating Fleet's
// minted id_token against the published JWKS.
//
// It is the device half of the PSSO exchange and shares all wire-format crypto
// with the server via server/mdm/apple/psso/pssocrypto, so the two halves can't
// drift. It speaks real HTTP, so the same client drives both the in-process
// integration tests (httptest server) and load testing in osquery-perf (a
// remote server). A device is composed onto a TestAppleMDMClient: it reuses that
// client's device UUID (which the registration token is bound to) and reads the
// PSSO profile out of the InstallProfile command the MDM server delivers.
type TestApplePSSODevice struct {
	// mdm is the enrolled MDM client this PSSO device rides on. Its UUID is the
	// host the registration token is bound to.
	mdm *TestAppleMDMClient

	// serverURL is the Fleet base URL the PSSO endpoints hang off of.
	serverURL  string
	httpClient *http.Client

	// clientID is the IdP/extension client ID, sent as the assertion issuer and
	// echoed by Fleet as the id_token audience.
	clientID string

	// The device's Secure Enclave-equivalent keypairs and their kids (base64url
	// SHA-256 of the raw public point, matching how the real extension registers).
	signingKey    *ecdsa.PrivateKey
	encryptionKey *ecdsa.PrivateKey
	signingKID    string
	encryptionKID string

	// registrationToken is the Fleet-signed JWT delivered in the PSSO profile;
	// set via RegistrationTokenFromCommand or SetRegistrationToken.
	registrationToken string

	// keyContext and provisionedPub are captured from a key request so the
	// following key exchange can echo the context and independently verify the
	// returned shared secret.
	keyContext     string
	provisionedPub *ecdsa.PublicKey

	// username and refreshToken remember the most recent login identity so later
	// requests (key request/exchange) carry the same sub/username/refresh_token a
	// real extension would, making the traffic a closer facsimile.
	username     string
	refreshToken string
}

// PSSOLoginOptions tunes a Login call. The zero value is a plaintext-password
// login with the device's registered signing key and a freshly fetched nonce.
type PSSOLoginOptions struct {
	// EncryptOnWire models the extension's loginRequestEncryptionPublicKey
	// behavior: the password is sealed in an embedded ECDH-ES JWE encrypted to
	// Fleet's published encryption key instead of riding as a plaintext claim.
	EncryptOnWire bool

	// SigningKeyOverride signs the outer assertion with this key instead of the
	// device's registered signing key (the registered kid is kept). Used to
	// exercise the "device authenticating with the wrong key" rejection.
	SigningKeyOverride *ecdsa.PrivateKey

	// RequestNonceOverride uses this request_nonce verbatim instead of fetching a
	// fresh one. Used to exercise nonce replay (single-use) rejection.
	RequestNonceOverride string
}

// PSSOLoginResult is the decrypted login response plus the material a test needs
// to validate it.
type PSSOLoginResult struct {
	IDToken      string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	// SessionNonce is the Apple session nonce the device sent; Fleet echoes it as
	// the id_token `nonce` claim.
	SessionNonce string
	// RawAssertion is the signed outer JWS the device sent, so a caller can
	// confirm e.g. that no plaintext password appears on the wire.
	RawAssertion string
	// RawResponse is the decrypted JWE plaintext (the OAuth token-response JSON).
	RawResponse []byte
}

// NewApplePSSODevice builds a PSSO device on top of an enrolled MDM client. It
// generates the device's signing and encryption keypairs. fleetServerURL is the
// Fleet base URL; clientID is the IdP/extension client ID used as the assertion
// issuer.
func NewApplePSSODevice(mdmClient *TestAppleMDMClient, fleetServerURL, clientID string) (*TestApplePSSODevice, error) {
	signingKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate psso signing key: %w", err)
	}
	encryptionKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate psso encryption key: %w", err)
	}
	signingKID, err := pssocrypto.KIDFromRawECPoint(&signingKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("compute psso signing kid: %w", err)
	}
	encryptionKID, err := pssocrypto.KIDFromRawECPoint(&encryptionKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("compute psso encryption kid: %w", err)
	}
	return &TestApplePSSODevice{
		mdm:           mdmClient,
		serverURL:     strings.TrimRight(fleetServerURL, "/"),
		httpClient:    fleethttp.NewClient(),
		clientID:      clientID,
		signingKey:    signingKey,
		encryptionKey: encryptionKey,
		signingKID:    signingKID,
		encryptionKID: encryptionKID,
	}, nil
}

// UUID is the device UUID (the enrolled MDM client's), which the registration
// token is bound to.
func (c *TestApplePSSODevice) UUID() string { return c.mdm.UUID }

// SigningKID and EncryptionKID expose the registered key IDs for assertions.
func (c *TestApplePSSODevice) SigningKID() string    { return c.signingKID }
func (c *TestApplePSSODevice) EncryptionKID() string { return c.encryptionKID }

// SetRegistrationToken sets the registration token used by Register, e.g. to a
// deliberately invalid value for a negative test.
func (c *TestApplePSSODevice) SetRegistrationToken(token string) { c.registrationToken = token }

// RegistrationTokenFromCommand extracts the substituted RegistrationToken from a
// delivered InstallProfile command (the PSSO extension payload) and stores it
// for the next Register call. This is the real path: the token reaches the
// device only inside the MDM-delivered profile.
func (c *TestApplePSSODevice) RegistrationTokenFromCommand(cmd *mdm.Command) (string, error) {
	if cmd == nil || cmd.Command.RequestType != "InstallProfile" {
		return "", fmt.Errorf("psso: expected an InstallProfile command, got %v", cmd)
	}
	var full micromdm.CommandPayload
	if err := plist.Unmarshal(cmd.Raw, &full); err != nil {
		return "", fmt.Errorf("psso: unmarshal install profile command: %w", err)
	}
	if full.Command.InstallProfile == nil {
		return "", errors.New("psso: command has no InstallProfile payload")
	}
	raw := full.Command.InstallProfile.Payload
	// The mobileconfig may be PKCS7-signed; unwrap to the raw XML plist.
	if !bytes.HasPrefix(raw, []byte("<?xml")) {
		p7, err := pkcs7.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("psso: parse signed profile: %w", err)
		}
		raw = p7.Content
	}

	var profile struct {
		PayloadContent []map[string]any `plist:"PayloadContent"`
	}
	if err := plist.Unmarshal(raw, &profile); err != nil {
		return "", fmt.Errorf("psso: unmarshal profile: %w", err)
	}
	for _, payload := range profile.PayloadContent {
		if token, ok := payload["RegistrationToken"].(string); ok && token != "" {
			c.registrationToken = token
			return token, nil
		}
	}
	return "", errors.New("psso: no RegistrationToken found in delivered profile")
}

// Nonce fetches a single-use server nonce from the nonce endpoint.
func (c *TestApplePSSODevice) Nonce() (string, error) {
	form := url.Values{}
	form.Set("grant_type", "srv_challenge")
	status, body, err := c.postForm(pssoNoncePath, form)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("psso nonce: status %d: %s", status, body)
	}
	var resp struct {
		Nonce string `json:"Nonce"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("psso nonce: decode response: %w", err)
	}
	if resp.Nonce == "" {
		return "", fmt.Errorf("psso nonce: empty nonce (error %q)", resp.Error)
	}
	return resp.Nonce, nil
}

// Register submits the device's public keys and registration token to the
// registration endpoint. A successful registration is a 204.
func (c *TestApplePSSODevice) Register() error {
	signingPEM, err := spkiPEM(&c.signingKey.PublicKey)
	if err != nil {
		return err
	}
	encryptionPEM, err := spkiPEM(&c.encryptionKey.PublicKey)
	if err != nil {
		return err
	}
	form := url.Values{}
	form.Set("device_uuid", c.mdm.UUID)
	form.Set("device_signing_key", string(signingPEM))
	form.Set("device_encryption_key", string(encryptionPEM))
	form.Set("signing_key_id", c.signingKID)
	form.Set("encryption_key_id", c.encryptionKID)
	form.Set("registration_token", c.registrationToken)

	status, body, err := c.postForm(pssoRegistrationPath, form)
	if err != nil {
		return err
	}
	if status != http.StatusNoContent {
		return fmt.Errorf("psso registration: status %d: %s", status, body)
	}
	return nil
}

// JWKSResponse fetches the raw JWKS endpoint and returns its status and body,
// so a caller can assert e.g. a 404 when the feature is disabled.
func (c *TestApplePSSODevice) JWKSResponse() (int, []byte, error) {
	return c.get(pssoJWKSPath)
}

// AASA fetches the raw apple-app-site-association document and returns its status
// and body (a 404 when the feature is disabled).
func (c *TestApplePSSODevice) AASA() (int, []byte, error) {
	return c.get(pssoAASAPath)
}

// JWKS fetches and parses the published JWKS, returning Fleet's signing
// (use:sig) and encryption (use:enc) public keys.
func (c *TestApplePSSODevice) JWKS() (signing, encryption *ecdsa.PublicKey, err error) {
	status, body, err := c.get(pssoJWKSPath)
	if err != nil {
		return nil, nil, err
	}
	if status != http.StatusOK {
		return nil, nil, fmt.Errorf("psso jwks: status %d: %s", status, body)
	}
	var set jose.JSONWebKeySet
	if err := json.Unmarshal(body, &set); err != nil {
		return nil, nil, fmt.Errorf("psso jwks: decode: %w", err)
	}
	for _, k := range set.Keys {
		pub, ok := k.Key.(*ecdsa.PublicKey)
		if !ok {
			continue
		}
		switch k.Use {
		case "sig":
			signing = pub
		case "enc":
			encryption = pub
		}
	}
	if signing == nil || encryption == nil {
		return nil, nil, errors.New("psso jwks: missing signing or encryption key")
	}
	return signing, encryption, nil
}

// Login performs a password login: it fetches a nonce, builds and signs the
// assertion (encrypting the password when EncryptOnWire is set), posts it, and
// decrypts the login response.
func (c *TestApplePSSODevice) Login(username, password string, opts PSSOLoginOptions) (*PSSOLoginResult, error) {
	requestNonce, err := c.resolveRequestNonce(opts.RequestNonceOverride)
	if err != nil {
		return nil, err
	}
	sessionNonce := uuid.NewString()
	claims, err := c.baseClaims(requestNonce, sessionNonce)
	if err != nil {
		return nil, err
	}
	claims.Username = username
	claims.Subject = username
	claims.RefreshToken = c.refreshToken

	if opts.EncryptOnWire {
		_, serverEnc, err := c.JWKS()
		if err != nil {
			return nil, err
		}
		plaintext, err := pssocrypto.BuildEmbeddedAssertionPlaintext(password)
		if err != nil {
			return nil, err
		}
		// The embedded assertion is encrypted to Fleet's published key; the apv
		// party-info is bound to that recipient key, mirroring the response
		// direction.
		assertionAPV, err := pssocrypto.BuildAPV(serverEnc, []byte(sessionNonce))
		if err != nil {
			return nil, err
		}
		assertionJWE, err := pssocrypto.BuildPartyInfoJWE(plaintext, serverEnc, assertionAPV, pssocrypto.TypEncryptedLoginAssertion)
		if err != nil {
			return nil, err
		}
		claims.GrantType = pssocrypto.GrantTypeJWTBearer
		claims.Assertion = string(assertionJWE)
	} else {
		claims.GrantType = pssocrypto.GrantTypePassword
		claims.Password = password
	}

	signKey := c.signingKey
	if opts.SigningKeyOverride != nil {
		signKey = opts.SigningKeyOverride
	}
	assertion, err := c.signAssertion(claims, signKey)
	if err != nil {
		return nil, err
	}

	plaintext, err := c.token(assertion, pssocrypto.TypLoginResponse)
	if err != nil {
		return nil, err
	}
	var resp struct {
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(plaintext, &resp); err != nil {
		return nil, fmt.Errorf("psso login: decode response: %w", err)
	}
	// Remember this login's identity and refresh token so subsequent key
	// requests carry them like a real extension would.
	c.username = username
	c.refreshToken = resp.RefreshToken
	return &PSSOLoginResult{
		IDToken:      resp.IDToken,
		RefreshToken: resp.RefreshToken,
		TokenType:    resp.TokenType,
		ExpiresIn:    resp.ExpiresIn,
		SessionNonce: sessionNonce,
		RawAssertion: assertion,
		RawResponse:  plaintext,
	}, nil
}

// ValidateIDToken verifies the login-response id_token's signature against
// Fleet's published JWKS (the device's trust anchor) and returns its claims. A
// caller should additionally check iss/aud/nonce/sub against what it expects.
func (c *TestApplePSSODevice) ValidateIDToken(idToken string) (jwt.MapClaims, error) {
	signing, _, err := c.JWKS()
	if err != nil {
		return nil, err
	}
	claims := jwt.MapClaims{}
	if _, err := jwt.ParseWithClaims(idToken, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected id_token signing method %q", t.Method.Alg())
		}
		return signing, nil
	}, jwt.WithValidMethods([]string{pssocrypto.SigningAlg})); err != nil {
		return nil, fmt.Errorf("validate id_token against jwks: %w", err)
	}
	return claims, nil
}

// KeyRequest performs a PSSO key request: Fleet provisions and certifies an
// unlock keypair and returns the certificate plus an opaque key_context. The
// certificate's public key and the context are retained for the key exchange.
// The provisioned certificate DER is returned.
func (c *TestApplePSSODevice) KeyRequest() ([]byte, error) {
	requestNonce, err := c.Nonce()
	if err != nil {
		return nil, err
	}
	claims, err := c.baseClaims(requestNonce, uuid.NewString())
	if err != nil {
		return nil, err
	}
	claims.RequestType = pssocrypto.RequestKey
	c.applyKeyRequestIdentity(claims)

	assertion, err := c.signAssertion(claims, c.signingKey)
	if err != nil {
		return nil, err
	}
	plaintext, err := c.token(assertion, pssocrypto.TypKeyResponse)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Certificate string `json:"certificate"`
		KeyContext  string `json:"key_context"`
	}
	if err := json.Unmarshal(plaintext, &resp); err != nil {
		return nil, fmt.Errorf("psso key request: decode response: %w", err)
	}
	certDER, err := base64.RawURLEncoding.DecodeString(resp.Certificate)
	if err != nil {
		return nil, fmt.Errorf("psso key request: decode certificate: %w", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("psso key request: parse certificate: %w", err)
	}
	pub, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("psso key request: certificate key is %T, want *ecdsa.PublicKey", cert.PublicKey)
	}
	c.keyContext = resp.KeyContext
	c.provisionedPub = pub
	return certDER, nil
}

// KeyExchange performs a PSSO key exchange against the key_context from the
// preceding KeyRequest: it sends a fresh device DH public key, decrypts the
// returned shared secret, and independently recomputes ECDH(device_priv,
// provisioned_pub) to confirm the two sides agree. The shared secret is returned.
func (c *TestApplePSSODevice) KeyExchange() ([]byte, error) {
	if c.keyContext == "" || c.provisionedPub == nil {
		return nil, errors.New("psso: KeyExchange requires a prior KeyRequest")
	}
	deviceDH, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	dhRaw, err := pssocrypto.RawECPoint(&deviceDH.PublicKey)
	if err != nil {
		return nil, err
	}

	requestNonce, err := c.Nonce()
	if err != nil {
		return nil, err
	}
	claims, err := c.baseClaims(requestNonce, uuid.NewString())
	if err != nil {
		return nil, err
	}
	claims.RequestType = pssocrypto.RequestExchange
	claims.OtherPublicKey = base64.StdEncoding.EncodeToString(dhRaw)
	claims.KeyContext = c.keyContext
	c.applyKeyRequestIdentity(claims)

	assertion, err := c.signAssertion(claims, c.signingKey)
	if err != nil {
		return nil, err
	}
	plaintext, err := c.token(assertion, pssocrypto.TypKeyResponse)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(plaintext, &resp); err != nil {
		return nil, fmt.Errorf("psso key exchange: decode response: %w", err)
	}
	serverShared, err := base64.StdEncoding.DecodeString(resp.Key)
	if err != nil {
		return nil, fmt.Errorf("psso key exchange: decode key: %w", err)
	}

	// Cross-check: the unlock key is symmetric, so ECDH(device_priv,
	// provisioned_pub) must equal the server's ECDH(provisioned_priv, device_pub).
	provRaw, err := pssocrypto.RawECPoint(c.provisionedPub)
	if err != nil {
		return nil, err
	}
	deviceShared, err := pssocrypto.ComputeECDHShared(deviceDH, provRaw)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(serverShared, deviceShared) {
		return nil, errors.New("psso key exchange: server shared secret does not match device-computed ECDH")
	}
	return serverShared, nil
}

// baseClaims assembles the fields common to every token request: the registered
// issuer/iat/exp, the session and request nonces, and the jwe_crypto recipe
// (ECDH-ES + A256GCM bound to the device's encryption key via apv).
func (c *TestApplePSSODevice) baseClaims(requestNonce, sessionNonce string) (*pssocrypto.TokenClaims, error) {
	apv, err := pssocrypto.BuildAPV(&c.encryptionKey.PublicKey, []byte(sessionNonce))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &pssocrypto.TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    c.clientID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
		},
		Version:      pssocrypto.ProtocolVersion,
		Nonce:        sessionNonce,
		RequestNonce: requestNonce,
		JWECrypto: &pssocrypto.JWECrypto{
			Alg: pssocrypto.EncryptionAlg,
			Enc: pssocrypto.ContentEncryptionAlg,
			APV: apv,
		},
	}, nil
}

// applyKeyRequestIdentity stamps the key-purpose and the remembered login
// identity (username/sub/refresh_token) onto a key request or key exchange, as
// the real extension does once a user has signed in.
func (c *TestApplePSSODevice) applyKeyRequestIdentity(claims *pssocrypto.TokenClaims) {
	claims.KeyPurpose = pssocrypto.KeyPurposeUserUnlock
	claims.Username = c.username
	claims.Subject = c.username
	claims.RefreshToken = c.refreshToken
}

func (c *TestApplePSSODevice) resolveRequestNonce(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	return c.Nonce()
}

// signAssertion signs the outer JWS with the given key, always stamping the
// device's registered signing kid so the server resolves the right device.
func (c *TestApplePSSODevice) signAssertion(claims *pssocrypto.TokenClaims, key *ecdsa.PrivateKey) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = c.signingKID
	signed, err := tok.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("psso: sign assertion: %w", err)
	}
	return signed, nil
}

// token posts a signed assertion to the token endpoint and decrypts the JWE
// response to its plaintext, pinning the expected response media type.
func (c *TestApplePSSODevice) token(assertion, expectedTyp string) ([]byte, error) {
	form := url.Values{}
	form.Set("assertion", assertion)
	status, body, err := c.postForm(pssoTokenPath, form)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("psso token: status %d: %s", status, body)
	}
	plaintext, err := pssocrypto.DecryptPartyInfoJWE(body, c.encryptionKey, expectedTyp)
	if err != nil {
		return nil, fmt.Errorf("psso token: decrypt response: %w", err)
	}
	return plaintext, nil
}

// get issues a GET to a Fleet path and returns the status and body.
func (c *TestApplePSSODevice) get(path string) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.serverURL+path, nil)
	if err != nil {
		return 0, nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}

// postForm posts an x-www-form-urlencoded body and returns the status and body.
func (c *TestApplePSSODevice) postForm(path string, form url.Values) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodPost, c.serverURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}

// spkiPEM encodes a public key as a SubjectPublicKeyInfo "PUBLIC KEY" PEM, one of
// the forms the registration endpoint accepts.
func spkiPEM(pub *ecdsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("psso: marshal public key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}
