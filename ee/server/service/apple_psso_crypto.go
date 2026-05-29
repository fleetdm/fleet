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
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	jose "github.com/go-jose/go-jose/v3"
	josecipher "github.com/go-jose/go-jose/v3/cipher"
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
// carry. The real PSSO v2 Password login request identifies itself with
// GrantType=="password" and carries a plaintext Password plus a JWECrypto
// recipe describing how the response must be encrypted. The RequestType /
// Encrypted* fields belong to an earlier handshake model and are retained
// only so the legacy dispatch path still compiles.
type pssoTokenClaims struct {
	jwt.RegisteredClaims

	// PSSO v2 Password login request.
	GrantType    string         `json:"grant_type,omitempty"`
	Password     string         `json:"password,omitempty"`
	Username     string         `json:"username,omitempty"`
	Nonce        string         `json:"nonce,omitempty"`         // Apple session nonce, echoed in the response
	JWECrypto    *pssoJWECrypto `json:"jwe_crypto,omitempty"`    // response-encryption recipe
	RequestNonce string         `json:"request_nonce,omitempty"` // Fleet-issued nonce from /nonce

	// Legacy handshake model (unused by the Password flow).
	RequestType    pssoRequestType `json:"request_type,omitempty"`
	EncryptedPwd   string          `json:"encrypted_password,omitempty"`
	EncryptedNonce string          `json:"encrypted_nonce,omitempty"`
}

// pssoJWECrypto is the jwe_crypto claim the extension sends to tell Fleet how
// to encrypt the login response: ECDH-ES key agreement to the device
// encryption key with A256GCM content encryption, binding the agreed key to
// the apu/apv party-info the device chose.
type pssoJWECrypto struct {
	Alg string `json:"alg"`
	Enc string `json:"enc"`
	APU string `json:"apu,omitempty"`
	APV string `json:"apv,omitempty"`
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
	kid = canonicalizeKID(kid)

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

// canonicalizeKID normalizes a key ID to a stable comparison form. Apple's
// framework emits the JWT `kid` as base64 with padding (e.g. "…LZE="), while
// the extension registers its key IDs as base64url without padding ("…LZE").
// Both encode the same SHA-256 bytes, so decode tolerantly (either alphabet,
// optional padding) and re-encode as raw base64url. Both the stored kid and
// the looked-up kid pass through this so the two encodings can't drift apart.
// If the value doesn't decode as base64 it's returned unchanged.
func canonicalizeKID(kid string) string {
	t := strings.TrimRight(kid, "=")
	t = strings.ReplaceAll(t, "-", "+")
	t = strings.ReplaceAll(t, "_", "/")
	raw, err := base64.RawStdEncoding.DecodeString(t)
	if err != nil {
		return kid
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// parseECPublicKeyPEM decodes a PEM-wrapped P-256 public key. It accepts both
// DER-encoded SubjectPublicKeyInfo (the standard "PUBLIC KEY" body) and a raw
// ANSI X9.63 uncompressed point (0x04 || X || Y). The extension's keys arrive
// in the latter form: macOS SecKeyCopyExternalRepresentation returns the raw
// point for EC keys, which the extension PEM-wraps without converting to SPKI.
func parseECPublicKeyPEM(pemBytes []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("psso: pem decode returned nil block")
	}
	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		ec, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("psso: unexpected pubkey type %T (want *ecdsa.PublicKey)", pub)
		}
		return ec, nil
	}
	return parseRawECPoint(block.Bytes)
}

// parseRawECPoint parses a raw ANSI X9.63 uncompressed P-256 point into an
// ecdsa.PublicKey. crypto/ecdh validates the length and on-curve membership;
// round-tripping through SPKI yields the ecdsa type the JWT verifier and JWE
// encrypter expect without touching the deprecated raw coordinate fields.
func parseRawECPoint(raw []byte) (*ecdsa.PublicKey, error) {
	key, err := ecdh.P256().NewPublicKey(raw)
	if err != nil {
		return nil, fmt.Errorf("psso: parse raw EC point: %w", err)
	}
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, fmt.Errorf("psso: marshal raw EC point to SPKI: %w", err)
	}
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("psso: parse SPKI from raw point: %w", err)
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

// Apple's PSSO ECDH-ES party-info blobs are sequences of 4-byte big-endian
// length-prefixed fields. Per Apple's "Creating a JSON Web Encryption (JWE)
// login response" doc the two differ in both label case and contents:
//   - apv (PartyVInfo, the device): "Apple" || deviceEncKey || nonce — echoed
//     verbatim from the request.
//   - apu (PartyUInfo, the server):  "APPLE" || serverEphemeralKey — note the
//     uppercase label and the absence of a nonce.
const (
	apuPartyLabel = "APPLE"
	apvPartyLabel = "Apple"
)

// encodeApplePartyInfo serializes fields as Apple's length-prefixed party-info
// blob: each field is a 4-byte big-endian length followed by its bytes.
func encodeApplePartyInfo(fields ...[]byte) []byte {
	var b []byte
	var l [4]byte
	for _, f := range fields {
		binary.BigEndian.PutUint32(l[:], uint32(len(f)))
		b = append(b, l[:]...)
		b = append(b, f...)
	}
	return b
}

// parseApplePartyInfo splits an Apple party-info blob back into its
// length-prefixed fields.
func parseApplePartyInfo(raw []byte) ([][]byte, error) {
	var fields [][]byte
	for i := 0; i < len(raw); {
		if i+4 > len(raw) {
			return nil, errors.New("psso: truncated party-info length prefix")
		}
		n := int(binary.BigEndian.Uint32(raw[i:]))
		i += 4
		if i+n > len(raw) {
			return nil, errors.New("psso: party-info field overruns buffer")
		}
		fields = append(fields, raw[i:i+n])
		i += n
	}
	return fields, nil
}

// buildPSSOResponseJWE encrypts payload to the device's encryption public key
// as a compact JWE using ECDH-ES key agreement + A256GCM. typ is the JWE
// header media type — "platformsso-login-response+jwt" for login,
// "platformsso-key-response+jwt" for key/key-exchange responses.
//
// Apple's framework requires both apu and apv in the protected header and
// validates apu by recomputing it from the epk it sees. apv (PartyVInfo) is
// echoed verbatim from the request. apu (PartyUInfo) is built as
// "APPLE" || serverEphemeralPubKey — uppercase label, no nonce — per Apple's
// JWE login-response doc.
//
// The compact JWE is assembled by hand rather than via jose.NewEncrypter
// because go-jose's ECDH-ES key generator hardcodes empty apu/apv (see
// ecKeyGenerator.genKey) and exposes no way to set them. The Concat KDF itself
// is reused from go-jose's exported cipher package — no PSSO SDK is involved.
func buildPSSOResponseJWE(payload []byte, recipientPub *ecdsa.PublicKey, apvB64, typ string) ([]byte, error) {
	apvRaw, err := decodeJOSEB64(apvB64)
	if err != nil {
		return nil, fmt.Errorf("decode apv: %w", err)
	}

	ephemeral, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ephemeral key: %w", err)
	}
	epkECDH, err := ephemeral.PublicKey.ECDH()
	if err != nil {
		return nil, fmt.Errorf("psso: ephemeral key to ecdh: %w", err)
	}
	apuRaw := encodeApplePartyInfo([]byte(apuPartyLabel), epkECDH.Bytes())

	// ECDH-ES direct: the agreed key is the A256GCM content-encryption key, so
	// the Concat KDF algorithm ID is the content-encryption alg ("A256GCM").
	cek := josecipher.DeriveECDHES("A256GCM", apuRaw, apvRaw, ephemeral, recipientPub, 32)

	epkJSON, err := json.Marshal(&jose.JSONWebKey{Key: &ephemeral.PublicKey})
	if err != nil {
		return nil, fmt.Errorf("marshal epk: %w", err)
	}

	// No cty: the decrypted payload is a JSON object (OAuth token response or
	// key response), not a nested JWT.
	header := map[string]any{
		"alg": "ECDH-ES",
		"enc": "A256GCM",
		"epk": json.RawMessage(epkJSON),
		"typ": typ,
		"apu": base64.RawURLEncoding.EncodeToString(apuRaw),
		"apv": strings.TrimRight(apvB64, "="),
	}
	protected, err := json.Marshal(header)
	if err != nil {
		return nil, fmt.Errorf("marshal protected header: %w", err)
	}
	protectedB64 := base64.RawURLEncoding.EncodeToString(protected)

	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes-gcm: %w", err)
	}
	iv := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("rand iv: %w", err)
	}
	// JWE AAD for compact serialization is the ASCII base64url protected header.
	sealed := gcm.Seal(nil, iv, payload, []byte(protectedB64))
	ct := sealed[:len(sealed)-gcm.Overhead()]
	tag := sealed[len(sealed)-gcm.Overhead():]

	enc := base64.RawURLEncoding.EncodeToString
	// Compact JWE: protected.encrypted_key.iv.ciphertext.tag — encrypted_key
	// is empty for ECDH-ES direct key agreement.
	compact := protectedB64 + "." + "" + "." + enc(iv) + "." + enc(ct) + "." + enc(tag)
	return []byte(compact), nil
}

// decodeJOSEB64 base64url-decodes a JOSE value, tolerating optional padding.
func decodeJOSEB64(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
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
