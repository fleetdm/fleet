// Package pssocrypto holds the symmetric JOSE wire-format primitives for Apple
// Platform SSO (PSSO): the pieces both the Fleet server and a PSSO client (the
// macOS extension, and Fleet's MDM test/load simulator) must agree on byte for
// byte. Keeping them in one package means the two halves of every exchange are
// built and parsed by the same code, so the wire format can't drift.
//
// Implemented against Apple's ASAuthorizationProviderExtension* protocol surface
// and standard JOSE primitives. No PSSO SDK is involved — the Concat KDF is
// reused from go-jose's exported cipher package.
//
// Cryptographic choices:
//   - Inbound JWTs from the Mac extension are ES256 (P-256). The kid in the
//     header points to a registered device public key.
//   - JWE bodies use ECDH-ES with A256GCM, wrapped to the recipient's encryption
//     pubkey, binding the agreed key to Apple's apu/apv party-info.
//
// Server-only concerns (datastore lookups, Fleet's signing key, the opaque
// key_context sealing) deliberately stay in ee/server/service and call into the
// primitives here.
package pssocrypto

import (
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
	"time"

	jose "github.com/go-jose/go-jose/v3"
	josecipher "github.com/go-jose/go-jose/v3/cipher"
	jwt "github.com/golang-jwt/jwt/v4"
)

// Algorithms pinned across the PSSO protocol. SigningAlg is the only JWS
// algorithm the Secure Enclave-backed extension uses; EncryptionAlg /
// ContentEncryptionAlg are the JWE key-agreement and content-encryption algs.
const (
	SigningAlg           = "ES256"
	EncryptionAlg        = "ECDH-ES"
	ContentEncryptionAlg = "A256GCM"
)

// Grant types in the login-request JWT. With plaintext passwords Apple sends
// GrantTypePassword; when the password is encrypted into the embedded assertion
// it switches to the JWT-bearer grant and the password moves out of the
// top-level claim into the (encrypted) assertion.
const (
	GrantTypePassword  = "password"                                    //nolint:gosec // G101 not a credential, a grant type
	GrantTypeJWTBearer = "urn:ietf:params:oauth:grant-type:jwt-bearer" //nolint:gosec // G101 not a credential, a grant type
)

// JWE header `typ` media types. The first two are responses Fleet returns; the
// last is the embedded login assertion the device sends when password
// encryption is enabled.
const (
	TypLoginResponse           = "platformsso-login-response+jwt"
	TypKeyResponse             = "platformsso-key-response+jwt"
	TypEncryptedLoginAssertion = "platformsso-encrypted-login-assertion+jwt"
)

// RequestType is the claim discriminator on every inbound token JWT.
type RequestType string

const (
	RequestKey      RequestType = "key_request"
	RequestExchange RequestType = "key_exchange"
)

// ProtocolVersion is the PSSO protocol version the extension stamps on every
// request body ("version": "1.0").
const ProtocolVersion = "1.0"

// KeyPurposeUserUnlock is the only key purpose Fleet provisions today: the
// offline FileVault/keychain unlock key, sent as "key_purpose" on key requests.
const KeyPurposeUserUnlock = "user_unlock"

// JWTLeeway is the clock-skew tolerance applied to inbound JWT time claims. The
// default RegisteredClaims validation allows zero skew, so a Mac whose clock
// runs even a second ahead of the server gets "token used before issued" on
// every login.
const JWTLeeway = time.Minute

// TokenClaims models the union of claims an inbound token JWT can carry. The
// PSSO v2 Password login request identifies itself with GrantType=="password"
// and carries a plaintext Password plus a JWECrypto recipe describing how the
// response must be encrypted; key requests and key exchanges identify
// themselves via RequestType instead.
type TokenClaims struct {
	jwt.RegisteredClaims

	// Version is the PSSO protocol version ("1.0") stamped on every request.
	Version string `json:"version,omitempty"`

	// PSSO v2 Password login request.
	GrantType string `json:"grant_type,omitempty"`
	Password  string `json:"password,omitempty"` // plaintext; empty when the password is encrypted in Assertion
	// Assertion is the embedded login assertion. When the extension sets
	// loginRequestEncryptionPublicKey, Apple drops the plaintext Password claim
	// and instead places the password in this compact JWE
	// (typ platformsso-encrypted-login-assertion+jwt), encrypted to Fleet's
	// published encryption key. The outer JWT remains signed by the device key.
	Assertion    string     `json:"assertion,omitempty"`
	Username     string     `json:"username,omitempty"`
	Nonce        string     `json:"nonce,omitempty"`         // Apple session nonce, echoed in the response
	JWECrypto    *JWECrypto `json:"jwe_crypto,omitempty"`    // response-encryption recipe
	RequestNonce string     `json:"request_nonce,omitempty"` // Fleet-issued nonce from /nonce
	// RefreshToken is the current SSO refresh token the extension holds, carried
	// on login renewals and key requests. Fleet treats it as opaque.
	RefreshToken string `json:"refresh_token,omitempty"`

	// PSSO 2.0 key request / key exchange (request_type "key_request" /
	// "key_exchange", used during registration to provision the unlock key).
	RequestType    RequestType `json:"request_type,omitempty"`
	KeyPurpose     string      `json:"key_purpose,omitempty"`     // e.g. "user_unlock"
	OtherPublicKey string      `json:"other_publickey,omitempty"` // device DH public key (key_exchange)
	KeyContext     string      `json:"key_context,omitempty"`     // server-sealed provisioned key, echoed back
}

// Valid overrides the embedded RegisteredClaims validation to apply JWTLeeway to
// exp, iat, and nbf. jwt/v4 has no parser-level leeway option (that arrived in
// v5), so the claims type does it.
func (c *TokenClaims) Valid() error {
	now := time.Now()
	if !c.VerifyExpiresAt(now.Add(-JWTLeeway), false) {
		return jwt.ErrTokenExpired
	}
	if !c.VerifyIssuedAt(now.Add(JWTLeeway), false) {
		return jwt.ErrTokenUsedBeforeIssued
	}
	if !c.VerifyNotBefore(now.Add(JWTLeeway), false) {
		return jwt.ErrTokenNotValidYet
	}
	return nil
}

// JWECrypto is the jwe_crypto claim the extension sends to tell Fleet how to
// encrypt the login response: ECDH-ES key agreement to the device encryption key
// with A256GCM content encryption, binding the agreed key to the apu/apv
// party-info the device chose.
type JWECrypto struct {
	Alg string `json:"alg"`
	Enc string `json:"enc"`
	APU string `json:"apu,omitempty"`
	APV string `json:"apv,omitempty"`
}

// CanonicalizeKID normalizes a key ID to a stable comparison form. Apple's
// framework emits the JWT `kid` as base64 with padding (e.g. "…LZE="), while the
// extension registers its key IDs as base64url without padding ("…LZE"). Both
// encode the same SHA-256 bytes, so decode tolerantly (either alphabet, optional
// padding) and re-encode as raw base64url. Both the stored kid and the
// looked-up kid pass through this so the two encodings can't drift apart. If the
// value doesn't decode as base64 it's returned unchanged.
func CanonicalizeKID(kid string) string {
	t := strings.TrimRight(kid, "=")
	t = strings.ReplaceAll(t, "-", "+")
	t = strings.ReplaceAll(t, "_", "/")
	raw, err := base64.RawStdEncoding.DecodeString(t)
	if err != nil {
		return kid
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// RawECPoint returns the ANSI X9.63 uncompressed point (0x04 || X || Y) for a
// P-256 public key — the form the extension uses to represent its keys on the
// wire and to derive their kids.
func RawECPoint(pub *ecdsa.PublicKey) ([]byte, error) {
	ecdhPub, err := pub.ECDH()
	if err != nil {
		return nil, fmt.Errorf("pssocrypto: public key to ecdh: %w", err)
	}
	return ecdhPub.Bytes(), nil
}

// KIDFromRawECPoint returns the kid the extension registers a key under:
// base64url-nopad SHA-256 of the raw uncompressed point. This is the kid the
// server recomputes from a request's apv to resolve the device encryption key.
func KIDFromRawECPoint(pub *ecdsa.PublicKey) (string, error) {
	raw, err := RawECPoint(pub)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return CanonicalizeKID(base64.RawURLEncoding.EncodeToString(sum[:])), nil
}

// ParseECPublicKeyPEM decodes a PEM-wrapped P-256 public key. It accepts both
// DER-encoded SubjectPublicKeyInfo (the standard "PUBLIC KEY" body) and a raw
// ANSI X9.63 uncompressed point (0x04 || X || Y). The extension's keys arrive in
// the latter form: macOS SecKeyCopyExternalRepresentation returns the raw point
// for EC keys, which the extension PEM-wraps without converting to SPKI.
func ParseECPublicKeyPEM(pemBytes []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("pssocrypto: pem decode returned nil block")
	}
	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		ec, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("pssocrypto: unexpected pubkey type %T (want *ecdsa.PublicKey)", pub)
		}
		// Everything downstream assumes P-256: ES256 verification fails opaquely
		// on another curve, and go-jose's DeriveECDHES panics on a curve mismatch
		// when building the response JWE.
		if ec.Curve != elliptic.P256() {
			return nil, fmt.Errorf("pssocrypto: unsupported curve %s (want P-256)", ec.Curve.Params().Name)
		}
		return ec, nil
	}
	return ParseRawECPoint(block.Bytes)
}

// ParseRawECPoint parses a raw ANSI X9.63 uncompressed P-256 point into an
// ecdsa.PublicKey. crypto/ecdh validates the length and on-curve membership;
// round-tripping through SPKI yields the ecdsa type the JWT verifier and JWE
// encrypter expect without touching the deprecated raw coordinate fields.
func ParseRawECPoint(raw []byte) (*ecdsa.PublicKey, error) {
	key, err := ecdh.P256().NewPublicKey(raw)
	if err != nil {
		return nil, fmt.Errorf("pssocrypto: parse raw EC point: %w", err)
	}
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, fmt.Errorf("pssocrypto: marshal raw EC point to SPKI: %w", err)
	}
	pub, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("pssocrypto: parse SPKI from raw point: %w", err)
	}
	ec, ok := pub.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("pssocrypto: unexpected pubkey type %T (want *ecdsa.PublicKey)", pub)
	}
	return ec, nil
}

// BuildAsymmetricJWE encrypts payload to deviceEncPub using JWE
// ECDH-ES + A256GCM via go-jose's stock encrypter (empty apu/apv — see
// BuildPartyInfoJWE for the Apple-party-info variant the handlers use).
func BuildAsymmetricJWE(payload []byte, deviceEncPub *ecdsa.PublicKey, kid string) ([]byte, error) {
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
	APUPartyLabel = "APPLE"
	APVPartyLabel = "Apple"
)

// EncodeApplePartyInfo serializes fields as Apple's length-prefixed party-info
// blob: each field is a 4-byte big-endian length followed by its bytes.
func EncodeApplePartyInfo(fields ...[]byte) []byte {
	var b []byte
	var l [4]byte
	for _, f := range fields {
		//nolint:gosec // dismiss G115, party-info fields are small (labels, 65-byte EC points, nonces), never near 2^32
		binary.BigEndian.PutUint32(l[:], uint32(len(f)))
		b = append(b, l[:]...)
		b = append(b, f...)
	}
	return b
}

// ParseApplePartyInfo splits an Apple party-info blob back into its
// length-prefixed fields.
func ParseApplePartyInfo(raw []byte) ([][]byte, error) {
	var fields [][]byte
	for i := 0; i < len(raw); {
		if i+4 > len(raw) {
			return nil, errors.New("pssocrypto: truncated party-info length prefix")
		}
		n := int(binary.BigEndian.Uint32(raw[i:]))
		i += 4
		if i+n > len(raw) {
			return nil, errors.New("pssocrypto: party-info field overruns buffer")
		}
		fields = append(fields, raw[i:i+n])
		i += n
	}
	return fields, nil
}

// BuildAPV returns the base64url-encoded apv (PartyVInfo) party-info blob the
// device sends in its jwe_crypto recipe: "Apple" || rawEncKeyPoint || nonce.
func BuildAPV(encPub *ecdsa.PublicKey, nonce []byte) (string, error) {
	raw, err := RawECPoint(encPub)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(
		EncodeApplePartyInfo([]byte(APVPartyLabel), raw, nonce)), nil
}

// BuildPartyInfoJWE encrypts payload to the recipient's encryption public key as
// a compact JWE using ECDH-ES key agreement + A256GCM. typ is the JWE header
// media type — TypLoginResponse for login, TypKeyResponse for key/key-exchange
// responses, TypEncryptedLoginAssertion for the device's embedded password
// assertion.
//
// Apple's framework requires both apu and apv in the protected header and
// validates apu by recomputing it from the epk it sees. apv (PartyVInfo) is
// echoed verbatim from the request. apu (PartyUInfo) is built as
// "APPLE" || ephemeralPubKey — uppercase label, no nonce — per Apple's JWE
// login-response doc.
//
// The compact JWE is assembled by hand rather than via jose.NewEncrypter because
// go-jose's ECDH-ES key generator hardcodes empty apu/apv (see
// ecKeyGenerator.genKey) and exposes no way to set them. The Concat KDF itself
// is reused from go-jose's exported cipher package — no PSSO SDK is involved.
func BuildPartyInfoJWE(payload []byte, recipientPub *ecdsa.PublicKey, apvB64, typ string) ([]byte, error) {
	apvRaw, err := DecodeJOSEB64(apvB64)
	if err != nil {
		return nil, fmt.Errorf("decode apv: %w", err)
	}

	ephemeral, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ephemeral key: %w", err)
	}
	epkECDH, err := ephemeral.PublicKey.ECDH()
	if err != nil {
		return nil, fmt.Errorf("pssocrypto: ephemeral key to ecdh: %w", err)
	}
	apuRaw := EncodeApplePartyInfo([]byte(APUPartyLabel), epkECDH.Bytes())

	// ECDH-ES direct: the agreed key is the A256GCM content-encryption key, so
	// the Concat KDF algorithm ID is the content-encryption alg ("A256GCM").
	cek := josecipher.DeriveECDHES(ContentEncryptionAlg, apuRaw, apvRaw, ephemeral, recipientPub, 32)

	epkJSON, err := json.Marshal(&jose.JSONWebKey{Key: &ephemeral.PublicKey})
	if err != nil {
		return nil, fmt.Errorf("marshal epk: %w", err)
	}

	// No cty: the decrypted payload is a JSON object (OAuth token response or
	// key response), not a nested JWT.
	header := map[string]any{
		"alg": EncryptionAlg,
		"enc": ContentEncryptionAlg,
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
	// Compact JWE: protected.encrypted_key.iv.ciphertext.tag — encrypted_key is
	// empty for ECDH-ES direct key agreement.
	compact := protectedB64 + "." + "" + "." + enc(iv) + "." + enc(ct) + "." + enc(tag)
	return []byte(compact), nil
}

// DecryptPartyInfoJWE decrypts a compact ECDH-ES + A256GCM JWE built by
// BuildPartyInfoJWE, where recipientPriv is the static ECDH-ES recipient and the
// sender supplied the ephemeral epk in the header. go-jose reads epk/apu/apv from
// the protected header and runs the same Concat KDF to recover the
// content-encryption key. alg/enc are pinned; expectedTyp pins the JWE media
// type the caller requires (e.g. TypLoginResponse on the device, or
// TypEncryptedLoginAssertion on the server decrypting the embedded password).
func DecryptPartyInfoJWE(compact []byte, recipientPriv *ecdsa.PrivateKey, expectedTyp string) ([]byte, error) {
	protectedB64, _, ok := strings.Cut(string(compact), ".")
	if !ok {
		return nil, errors.New("pssocrypto: not a compact JWE")
	}
	protected, err := DecodeJOSEB64(protectedB64)
	if err != nil {
		return nil, fmt.Errorf("pssocrypto: decode protected header: %w", err)
	}
	var hdr struct {
		Alg string `json:"alg"`
		Enc string `json:"enc"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(protected, &hdr); err != nil {
		return nil, fmt.Errorf("pssocrypto: parse protected header: %w", err)
	}
	if hdr.Alg != EncryptionAlg || hdr.Enc != ContentEncryptionAlg {
		return nil, fmt.Errorf("pssocrypto: unsupported alg/enc %q/%q", hdr.Alg, hdr.Enc)
	}
	if expectedTyp != "" && hdr.Typ != expectedTyp {
		return nil, fmt.Errorf("pssocrypto: unexpected typ %q", hdr.Typ)
	}

	obj, err := jose.ParseEncrypted(string(compact))
	if err != nil {
		return nil, fmt.Errorf("pssocrypto: parse jwe: %w", err)
	}
	plaintext, err := obj.Decrypt(recipientPriv)
	if err != nil {
		return nil, fmt.Errorf("pssocrypto: decrypt jwe: %w", err)
	}
	return plaintext, nil
}

// BuildEmbeddedAssertionPlaintext returns the JSON plaintext a device encrypts
// into the embedded login assertion when password encryption is enabled. The
// username is taken from the signed outer JWT, not here, so only the password is
// carried. It is the inverse of ParseEmbeddedAssertionPassword.
func BuildEmbeddedAssertionPlaintext(password string) ([]byte, error) {
	return json.Marshal(map[string]string{"password": password})
}

// ParseEmbeddedAssertionPassword pulls the password out of a decrypted embedded
// login assertion. Apple's typ ends in "+jwt", so the plaintext is a JWT whose
// claims carry the password; a bare JSON claims object is also accepted. The
// username is taken from the signed outer JWT, not here. The assertion is
// encrypted-only — its integrity is covered by the outer signed JWT and the JWE
// GCM tag, so no inner signature is verified here.
func ParseEmbeddedAssertionPassword(plaintext []byte) (string, error) {
	s := strings.TrimSpace(string(plaintext))
	claimsJSON := []byte(s)
	if len(s) > 0 && s[0] != '{' {
		// Compact JWT: header.payload[.signature]; the claims are the payload.
		parts := strings.Split(s, ".")
		if len(parts) < 2 {
			return "", errors.New("pssocrypto: embedded assertion is not JSON or a compact JWT")
		}
		decoded, derr := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
		if derr != nil {
			return "", fmt.Errorf("pssocrypto: decode embedded assertion claims segment: %w", derr)
		}
		claimsJSON = decoded
	}
	var claims struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return "", fmt.Errorf("pssocrypto: parse embedded assertion claims: %w", err)
	}
	return claims.Password, nil
}

// DecodeJOSEB64 base64url-decodes a JOSE value, tolerating optional padding.
func DecodeJOSEB64(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
}

// DecodeBase64Flexible decodes standard or url base64, with or without padding —
// the device sends other_publickey as padded standard base64.
func DecodeBase64Flexible(s string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, errors.New("pssocrypto: value is not valid base64")
}

// ComputeECDHShared returns the raw ECDH shared secret (P-256 X coordinate, 32
// bytes) between priv and the uncompressed peer public point — the key field of
// a key-exchange response.
func ComputeECDHShared(priv *ecdsa.PrivateKey, peerRaw []byte) ([]byte, error) {
	ecdhPriv, err := priv.ECDH()
	if err != nil {
		return nil, fmt.Errorf("pssocrypto: private key to ecdh: %w", err)
	}
	peer, err := ecdh.P256().NewPublicKey(peerRaw)
	if err != nil {
		return nil, fmt.Errorf("pssocrypto: parse peer public key: %w", err)
	}
	return ecdhPriv.ECDH(peer)
}
