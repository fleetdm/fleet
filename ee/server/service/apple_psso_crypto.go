package service

// PSSO crypto helpers. Implemented against Apple's ASAuthorizationProviderExtension*
// protocol surface and standard JOSE primitives.
//
// Cryptographic choices:
//   - Inbound JWTs from the Mac extension are ES256 (P-256). We may need to allow more
//     algorithms in the future but this is what has been observed today. The kid in the
//     header points to a PEM stored in mdm_apple_psso_keys.
//   - JWE responses use ECDH-ES with A256GCM, wrapped to the device's registered encryption
//     pubkey (resolved from the request's apv).
//   - key_context blobs are sealed with A256GCM under a key derived from Fleet's PSSO signing
//     key via HKDF-SHA256 — no per-device server state is stored.

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
	"time"

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
)

// pssoTokenClaims models the union of claims an inbound token JWT can
// carry. The PSSO v2 Password login request identifies itself with
// GrantType=="password" and carries a plaintext Password plus a JWECrypto
// recipe describing how the response must be encrypted; key requests and
// key exchanges identify themselves via RequestType instead.
type pssoTokenClaims struct {
	jwt.RegisteredClaims

	// PSSO v2 Password login request.
	GrantType string `json:"grant_type,omitempty"`
	Password  string `json:"password,omitempty"` // plaintext; empty when the password is encrypted in Assertion
	// Assertion is the embedded login assertion. When the extension sets
	// loginRequestEncryptionPublicKey, Apple drops the plaintext Password claim
	// and instead places the password in this compact JWE
	// (typ platformsso-encrypted-login-assertion+jwt), encrypted to Fleet's
	// published encryption key. The outer JWT remains signed by the device key.
	Assertion    string         `json:"assertion,omitempty"`
	Username     string         `json:"username,omitempty"`
	Nonce        string         `json:"nonce,omitempty"`         // Apple session nonce, echoed in the response
	JWECrypto    *pssoJWECrypto `json:"jwe_crypto,omitempty"`    // response-encryption recipe
	RequestNonce string         `json:"request_nonce,omitempty"` // Fleet-issued nonce from /nonce

	// PSSO 2.0 key request / key exchange (request_type "key_request" /
	// "key_exchange", used during registration to provision the unlock key).
	RequestType    pssoRequestType `json:"request_type,omitempty"`
	OtherPublicKey string          `json:"other_publickey,omitempty"` // device DH public key (key_exchange)
	KeyContext     string          `json:"key_context,omitempty"`     // server-sealed provisioned key, echoed back
}

// pssoJWTLeeway is the clock-skew tolerance applied to inbound JWT time
// claims. The default RegisteredClaims validation allows zero skew, so a Mac
// whose clock runs even a second ahead of the server gets "token used before
// issued" on every login.
const pssoJWTLeeway = time.Minute

// Valid overrides the embedded RegisteredClaims validation to apply
// pssoJWTLeeway to exp, iat, and nbf. jwt/v4 has no parser-level leeway
// option (that arrived in v5), so the claims type does it.
func (c *pssoTokenClaims) Valid() error {
	now := time.Now()
	if !c.VerifyExpiresAt(now.Add(-pssoJWTLeeway), false) {
		return jwt.ErrTokenExpired
	}
	if !c.VerifyIssuedAt(now.Add(pssoJWTLeeway), false) {
		return jwt.ErrTokenUsedBeforeIssued
	}
	if !c.VerifyNotBefore(now.Add(pssoJWTLeeway), false) {
		return jwt.ErrTokenNotValidYet
	}
	return nil
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
// signing key row that matched (its HostUUID identifies the device).
func (svc *Service) parsePSSOInboundJWT(ctx context.Context, jwtBytes []byte) (*pssoTokenClaims, *fleet.PSSOKey, error) {
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

	signKey, err := svc.ds.GetPSSOKey(ctx, kid)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "look up psso key by kid")
	}
	if signKey.KeyType != fleet.PSSOKeyTypeSigning {
		return nil, nil, &fleet.BadRequestError{Message: "psso jwt kid does not reference a signing key"}
	}

	pub, err := parseECPublicKeyPEM([]byte(signKey.PEM))
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "parse device signing pubkey")
	}

	// Pin the algorithm to ES256 (the only alg the Secure Enclave-backed
	// extension signs with) and assert the ECDSA method in the keyfunc. Without
	// this, a future refactor returning a non-EC key could open an alg-confusion
	// forgery path even though golang-jwt's type assertions currently prevent it.
	tok, err := jwt.ParseWithClaims(string(jwtBytes), &pssoTokenClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("psso jwt: unexpected signing method %q", t.Method.Alg())
		}
		return pub, nil
	}, jwt.WithValidMethods([]string{pssoSigningAlg}))
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "verify psso jwt signature")
	}
	claims, ok := tok.Claims.(*pssoTokenClaims)
	if !ok || !tok.Valid {
		return nil, nil, &fleet.BadRequestError{Message: "psso jwt claims invalid"}
	}
	return claims, signKey, nil
}

// resolvePSSOEncryptionKey returns the registered encryption public key the
// response JWE must be wrapped to. The device names its encryption key inside
// the request's apv party-info blob ("Apple" || deviceEncKey || nonce), and
// the extension registered that key under kid = base64url(SHA-256(raw key
// bytes)) — so the kid is recomputed from apv and looked up. As a fallback
// against any re-encoding of the key by Apple's framework, the raw point is
// compared against each of the host's registered encryption keys. A key that
// resolves but belongs to a different host, or doesn't resolve at all, is
// rejected: responses are only ever encrypted to keys the host registered.
func (svc *Service) resolvePSSOEncryptionKey(ctx context.Context, hostUUID, apvB64 string) (*ecdsa.PublicKey, error) {
	apvRaw, err := decodeJOSEB64(apvB64)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decode apv")
	}
	fields, err := parseApplePartyInfo(apvRaw)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse apv party-info")
	}
	if len(fields) < 2 || string(fields[0]) != apvPartyLabel {
		return nil, &fleet.BadRequestError{Message: "psso: apv is not an Apple party-info blob"}
	}
	encKeyRaw := fields[1]

	sum := sha256.Sum256(encKeyRaw)
	kid := canonicalizeKID(base64.RawURLEncoding.EncodeToString(sum[:]))
	key, err := svc.ds.GetPSSOKey(ctx, kid)
	switch {
	case err == nil:
		if key.KeyType != fleet.PSSOKeyTypeEncryption || key.HostUUID != hostUUID {
			return nil, &fleet.BadRequestError{Message: "psso: apv key is not a registered encryption key for this device"}
		}
		return parseECPublicKeyPEM([]byte(key.PEM))
	case !fleet.IsNotFound(err):
		return nil, ctxerr.Wrap(ctx, err, "look up encryption key by apv kid")
	}

	apvPub, err := parseRawECPoint(encKeyRaw)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse apv encryption key")
	}
	hostKeys, err := svc.ds.ListPSSOKeys(ctx, hostUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list psso keys for apv fallback")
	}
	for _, k := range hostKeys {
		if k.KeyType != fleet.PSSOKeyTypeEncryption {
			continue
		}
		pub, err := parseECPublicKeyPEM([]byte(k.PEM))
		if err != nil {
			svc.logger.WarnContext(ctx, "psso: skipping unparseable registered encryption key", "kid", k.KID, "err", err)
			continue
		}
		if pub.Equal(apvPub) {
			return pub, nil
		}
	}
	return nil, &fleet.BadRequestError{Message: "psso: apv key is not a registered encryption key for this device"}
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
// ECDH-ES + A256GCM via go-jose's stock encrypter (empty apu/apv — see
// buildPSSOResponseJWE for the Apple-party-info variant the handlers use).
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
		//nolint:gosec // dismiss G115, party-info fields are small (labels, 65-byte EC points, nonces), never near 2^32
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

// decryptPSSOInboundJWE decrypts a compact JWE the device sent us — the embedded
// login assertion carrying the user's password. It is the inverse direction of
// buildPSSOResponseJWE: here Fleet is the static ECDH-ES recipient
// (recipientPriv) and the device supplied the ephemeral epk in the header.
// go-jose reads epk/apu/apv from the protected header and runs the same Concat
// KDF to recover the A256GCM content-encryption key (the same path the
// login-response round-trip test relies on). alg/enc/typ are pinned to what
// Apple emits for password encryption.
func decryptPSSOInboundJWE(compact []byte, recipientPriv *ecdsa.PrivateKey) ([]byte, error) {
	protectedB64, _, ok := strings.Cut(string(compact), ".")
	if !ok {
		return nil, errors.New("psso inbound jwe: not a compact JWE")
	}
	protected, err := decodeJOSEB64(protectedB64)
	if err != nil {
		return nil, fmt.Errorf("psso inbound jwe: decode protected header: %w", err)
	}
	var hdr struct {
		Alg string `json:"alg"`
		Enc string `json:"enc"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(protected, &hdr); err != nil {
		return nil, fmt.Errorf("psso inbound jwe: parse protected header: %w", err)
	}
	if hdr.Alg != pssoEncryptionAlg || hdr.Enc != "A256GCM" {
		return nil, fmt.Errorf("psso inbound jwe: unsupported alg/enc %q/%q", hdr.Alg, hdr.Enc)
	}
	if hdr.Typ != pssoTypEncryptedLoginAssertion {
		return nil, fmt.Errorf("psso inbound jwe: unexpected typ %q", hdr.Typ)
	}

	obj, err := jose.ParseEncrypted(string(compact))
	if err != nil {
		return nil, fmt.Errorf("psso inbound jwe: parse: %w", err)
	}
	plaintext, err := obj.Decrypt(recipientPriv)
	if err != nil {
		return nil, fmt.Errorf("psso inbound jwe: decrypt: %w", err)
	}
	return plaintext, nil
}

// parseEmbeddedAssertionPassword pulls the password out of a decrypted embedded
// login assertion. Apple's typ ends in "+jwt", so the plaintext is a JWT whose
// claims carry the password; a bare JSON claims object is also accepted. The
// username is taken from the signed outer JWT, not here. The assertion is
// encrypted-only — its integrity is covered by the outer signed JWT and the JWE
// GCM tag, so no inner signature is verified here.
func parseEmbeddedAssertionPassword(plaintext []byte) (string, error) {
	s := strings.TrimSpace(string(plaintext))
	claimsJSON := []byte(s)
	if len(s) > 0 && s[0] != '{' {
		// Compact JWT: header.payload[.signature]; the claims are the payload.
		parts := strings.Split(s, ".")
		if len(parts) < 2 {
			return "", errors.New("psso embedded assertion: not JSON or a compact JWT")
		}
		decoded, derr := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
		if derr != nil {
			return "", fmt.Errorf("psso embedded assertion: decode claims segment: %w", derr)
		}
		claimsJSON = decoded
	}
	var claims struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return "", fmt.Errorf("psso embedded assertion: parse claims: %w", err)
	}
	return claims.Password, nil
}

// decodeJOSEB64 base64url-decodes a JOSE value, tolerating optional padding.
func decodeJOSEB64(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
}

// deriveKeyContextKey derives the AES-256 key that seals key_context blobs,
// from Fleet's PSSO signing key. This lets the provisioned private key live
// statelessly inside the key_context the device round-trips between the key
// request and key exchange — no per-device server storage.
func deriveKeyContextKey(signingKey *ecdsa.PrivateKey) ([]byte, error) {
	ikm, err := x509.MarshalECPrivateKey(signingKey)
	if err != nil {
		return nil, err
	}
	return deriveSessionKey(ikm, []byte("fleetdm-psso-key-context-v1"))
}

// pssoKeyPurposeUserUnlock is the only key purpose Fleet provisions today: the
// offline FileVault/keychain unlock key. It's recorded in the sealed key_context
// so key exchange can validate it and future purposes can be distinguished.
const pssoKeyPurposeUserUnlock = "user_unlock"

// pssoKeyContext is the plaintext sealed into the opaque key_context blob that
// rides between a key request and its matching key exchange. Binding the host
// UUID lets key exchange reject a context replayed by, or fetched onto, any
// device other than the one it was issued to; key_purpose leaves room to
// provision other key types later without reusing a context across purposes.
type pssoKeyContext struct {
	HostUUID       string `json:"host_uuid"`
	KeyPurpose     string `json:"key_purpose"`
	ProvisionedKey string `json:"provisioned_key"` // base64 (std) DER of the EC private key
}

// sealKeyContext seals the provisioned EC private key, bound to the device and
// key purpose, into the opaque base64 key_context returned in a key-request
// response.
func sealKeyContext(provisioned *ecdsa.PrivateKey, hostUUID, keyPurpose string, kcKey []byte) (string, error) {
	der, err := x509.MarshalECPrivateKey(provisioned)
	if err != nil {
		return "", err
	}
	plaintext, err := json.Marshal(pssoKeyContext{
		HostUUID:       hostUUID,
		KeyPurpose:     keyPurpose,
		ProvisionedKey: base64.StdEncoding.EncodeToString(der),
	})
	if err != nil {
		return "", err
	}
	blob, err := buildSymmetricJWE(plaintext, kcKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(blob), nil
}

// openKeyContext reverses sealKeyContext, returning the sealed context metadata
// (for the caller to validate device/purpose binding) and the recovered
// provisioned private key the device echoed back in a key-exchange request.
func openKeyContext(keyContext string, kcKey []byte) (*pssoKeyContext, *ecdsa.PrivateKey, error) {
	blob, err := base64.StdEncoding.DecodeString(keyContext)
	if err != nil {
		return nil, nil, fmt.Errorf("decode key_context: %w", err)
	}
	plaintext, err := decryptSymmetricBlob(blob, kcKey)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt key_context: %w", err)
	}
	var kc pssoKeyContext
	if err := json.Unmarshal(plaintext, &kc); err != nil {
		return nil, nil, fmt.Errorf("unmarshal key_context: %w", err)
	}
	der, err := base64.StdEncoding.DecodeString(kc.ProvisionedKey)
	if err != nil {
		return nil, nil, fmt.Errorf("decode key_context provisioned_key: %w", err)
	}
	key, err := x509.ParseECPrivateKey(der)
	if err != nil {
		return nil, nil, fmt.Errorf("parse key_context provisioned_key: %w", err)
	}
	return &kc, key, nil
}

// computeECDHShared returns the raw ECDH shared secret (P-256 X coordinate, 32
// bytes) between priv and the uncompressed peer public point — the key field
// of a key-exchange response.
func computeECDHShared(priv *ecdsa.PrivateKey, peerRaw []byte) ([]byte, error) {
	ecdhPriv, err := priv.ECDH()
	if err != nil {
		return nil, fmt.Errorf("provisioned key to ecdh: %w", err)
	}
	peer, err := ecdh.P256().NewPublicKey(peerRaw)
	if err != nil {
		return nil, fmt.Errorf("parse other_publickey: %w", err)
	}
	return ecdhPriv.ECDH(peer)
}

// decodeBase64Flexible decodes standard or url base64, with or without padding
// — the device sends other_publickey as padded standard base64.
func decodeBase64Flexible(s string) ([]byte, error) {
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, errors.New("psso: value is not valid base64")
}

// pssoSessionInfo is the HKDF info string distinguishing PSSO-derived keys
// from any other derivation the same input keying material could feed.
var pssoSessionInfo = []byte("fleetdm-psso-session-key-v1")

// deriveSessionKey returns a 32-byte AES-256 key derived from ikm via
// HKDF-SHA256. The salt parameter binds the derivation to a purpose (e.g.
// the key_context info string in deriveKeyContextKey).
func deriveSessionKey(ikm []byte, salt []byte) ([]byte, error) {
	r := hkdf.New(sha256.New, ikm, salt, pssoSessionInfo)
	out := make([]byte, 32)
	if _, err := r.Read(out); err != nil {
		return nil, fmt.Errorf("hkdf read: %w", err)
	}
	return out, nil
}

// buildSymmetricJWE returns an A256GCM JWE of payload, keyed by sessionKey.
// Used to seal key_context blobs so the provisioned private key can
// round-trip statelessly between key_request and key_exchange.
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

// decryptSymmetricBlob is the inverse of buildSymmetricJWE — used to open
// the key_context blob a device echoes back in a key-exchange request.
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
	key, kid, err := svc.getPSSOSigningKey(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load signing key for psso server jwt")
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = kid
	signed, err := tok.SignedString(key)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "sign server jwt")
	}
	return []byte(signed), nil
}
