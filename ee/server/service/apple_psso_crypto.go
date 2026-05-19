package service

// PSSO crypto helpers. Implemented clean-room against Apple's
// ASAuthorizationProviderExtension* protocol surface and standard JOSE
// primitives. No third-party PSSO SDK or sample code is referenced.
//
// Cryptographic choices for the POC:
//   - Inbound JWTs from the Mac extension are ES256 (P-256). The kid in the
//     header points to a PEM stored in mdm_apple_psso_key_ids.
//   - "Asymmetric" JWE responses (key_request) use ECDH-ES with A256GCM,
//     wrapped to the device's encryption pubkey.
//   - "Symmetric" JWE responses (key_exchange, password_request) use
//     A256GCM with the content-encryption key derived from KeyExchangeKey
//     via HKDF-SHA256.
//
// TODO(apple-psso-spec): The exact claim names and the precise HKDF salt /
// info bindings in Apple's published spec should be confirmed before this
// POC ships to a real Mac. The names below ("key_exchange_key", "claims",
// etc.) are clean-room placeholders; if Apple's framework rejects them, this
// is the first place to look.

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	jose "github.com/go-jose/go-jose/v3"
	jwt "github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/hkdf"
)

// pssoRequestType is the claim discriminator on every inbound token JWT.
type pssoRequestType string

const (
	pssoRequestKey      pssoRequestType = "key_request"
	pssoRequestExchange pssoRequestType = "key_exchange"
	pssoRequestPassword pssoRequestType = "password_request"
)

// pssoTokenClaims models the union of claims an inbound token JWT can
// carry. Optional fields are populated only on certain request types; the
// dispatcher checks RequestType first.
type pssoTokenClaims struct {
	jwt.RegisteredClaims
	RequestType    pssoRequestType `json:"request_type"`
	RequestNonce   string          `json:"request_nonce,omitempty"`
	Username       string          `json:"username,omitempty"`
	EncryptedPwd   string          `json:"encrypted_password,omitempty"` // base64-A256GCM blob
	EncryptedNonce string          `json:"encrypted_nonce,omitempty"`    // for key_exchange handshake
}

// parsePSSOInboundJWT verifies the inbound compact JWS using the device's
// signing pubkey (resolved by kid) and returns the parsed claims plus the
// associated device record.
func (svc *Service) parsePSSOInboundJWT(ctx context.Context, jwtBytes []byte) (*pssoTokenClaims, *fleet.PSSODevice, error) {
	// First parse without verification to extract kid.
	unverified, _, err := jwt.NewParser(jwt.WithoutClaimsValidation()).ParseUnverified(string(jwtBytes), &pssoTokenClaims{})
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "parse inbound psso jwt header")
	}
	kid, _ := unverified.Header["kid"].(string)
	if kid == "" {
		return nil, nil, &fleet.BadRequestError{Message: "psso jwt missing kid header"}
	}

	device, keyID, err := svc.ds.GetPSSODeviceByKeyID(ctx, kid)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "look up psso device by kid")
	}
	if keyID.KeyType != fleet.PSSOKeyTypeSigning {
		return nil, nil, &fleet.BadRequestError{Message: "psso jwt kid does not reference a signing key"}
	}

	pub, err := parseECPublicKeyPEM([]byte(keyID.PEM))
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "parse device signing pubkey")
	}

	tok, err := jwt.ParseWithClaims(string(jwtBytes), &pssoTokenClaims{}, func(*jwt.Token) (any, error) {
		return pub, nil
	})
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "verify psso jwt signature")
	}
	claims, ok := tok.Claims.(*pssoTokenClaims)
	if !ok || !tok.Valid {
		return nil, nil, &fleet.BadRequestError{Message: "psso jwt claims invalid"}
	}
	return claims, device, nil
}

// parseECPublicKeyPEM decodes a PEM-encoded EC public key. Accepts either
// "PUBLIC KEY" (SPKI) or "EC PUBLIC KEY" wrappers.
func parseECPublicKeyPEM(pemBytes []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("psso: pem decode returned nil block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKIX pubkey: %w", err)
	}
	ec, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("psso: unexpected pubkey type %T (want *ecdsa.PublicKey)", pub)
	}
	return ec, nil
}

// buildAsymmetricJWE encrypts payload to deviceEncPub using JWE
// ECDH-ES + A256GCM. Used for the key_request response that delivers the
// initial KeyExchangeKey to the device.
func buildAsymmetricJWE(payload []byte, deviceEncPub *ecdsa.PublicKey, kid string) ([]byte, error) {
	enc, err := jose.NewEncrypter(
		jose.A256GCM,
		jose.Recipient{
			Algorithm: jose.ECDH_ES,
			Key:       deviceEncPub,
			KeyID:     kid,
		},
		(&jose.EncrypterOptions{}).WithContentType("application/platformsso-login-response+jwt"),
	)
	if err != nil {
		return nil, fmt.Errorf("build asymmetric encrypter: %w", err)
	}
	jwe, err := enc.Encrypt(payload)
	if err != nil {
		return nil, fmt.Errorf("encrypt asymmetric jwe: %w", err)
	}
	compact, err := jwe.CompactSerialize()
	if err != nil {
		return nil, fmt.Errorf("serialize asymmetric jwe: %w", err)
	}
	return []byte(compact), nil
}

// pssoSessionInfo is the HKDF info string distinguishing PSSO session keys
// from any other purpose KeyExchangeKey could be used for.
var pssoSessionInfo = []byte("fleetdm-psso-session-key-v1")

// deriveSessionKey returns a 32-byte AES-256 key derived from the device's
// KeyExchangeKey via HKDF-SHA256. The salt parameter binds the derivation
// to a specific request (typically the request_nonce) so each sign-in uses
// a distinct content-encryption key.
func deriveSessionKey(kek []byte, salt []byte) ([]byte, error) {
	r := hkdf.New(sha256.New, kek, salt, pssoSessionInfo)
	out := make([]byte, 32)
	if _, err := r.Read(out); err != nil {
		return nil, fmt.Errorf("hkdf read: %w", err)
	}
	return out, nil
}

// buildSymmetricJWE returns an A256GCM JWE of payload, keyed by sessionKey.
// Used for key_exchange and password_request responses where the device
// has already established a shared secret via the KeyExchangeKey handshake.
func buildSymmetricJWE(payload []byte, sessionKey []byte) ([]byte, error) {
	if len(sessionKey) != 32 {
		return nil, fmt.Errorf("psso: session key must be 32 bytes, got %d", len(sessionKey))
	}
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes-gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("rand nonce: %w", err)
	}
	ct := gcm.Seal(nil, nonce, payload, nil)
	// JOSE-compatible flat-JSON serialization keeps the result inspectable
	// for the POC. A real client may require compact form; switch when
	// confirmed against the extension's expectations.
	envelope := struct {
		Alg        string `json:"alg"`
		Enc        string `json:"enc"`
		IV         []byte `json:"iv"`
		Ciphertext []byte `json:"ciphertext"`
	}{
		Alg:        "dir",
		Enc:        "A256GCM",
		IV:         nonce,
		Ciphertext: ct,
	}
	return json.Marshal(envelope)
}

// decryptSymmetricBlob is the inverse of buildSymmetricJWE — used in
// password_request to decrypt the password the device sent under the
// previously-established session key.
func decryptSymmetricBlob(blob []byte, sessionKey []byte) ([]byte, error) {
	if len(sessionKey) != 32 {
		return nil, fmt.Errorf("psso: session key must be 32 bytes, got %d", len(sessionKey))
	}
	var envelope struct {
		IV         []byte `json:"iv"`
		Ciphertext []byte `json:"ciphertext"`
	}
	if err := json.Unmarshal(blob, &envelope); err != nil {
		return nil, fmt.Errorf("decode symmetric blob: %w", err)
	}
	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes-gcm: %w", err)
	}
	pt, err := gcm.Open(nil, envelope.IV, envelope.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("aes-gcm open: %w", err)
	}
	return pt, nil
}

// signServerJWT returns a signed compact JWS with the given claims, using
// Fleet's PSSO signing key. Used to wrap payloads that must be authenticated
// as coming from Fleet (e.g. claims responses).
func (svc *Service) signServerJWT(ctx context.Context, claims jwt.Claims) ([]byte, error) {
	key, kid, err := svc.getOrMintPSSOSigningKey(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load signing key for psso server jwt")
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = kid
	signed, err := tok.SignedString(key)
	if err != nil {
		return nil, fmt.Errorf("sign server jwt: %w", err)
	}
	return []byte(signed), nil
}
