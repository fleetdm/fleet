package service

// Server-side PSSO crypto. The symmetric JOSE wire-format primitives (party-info
// encoding, the ECDH-ES + A256GCM JWE build/decrypt, kid canonicalization, the
// inbound claim types) live in server/mdm/apple/psso/pssocrypto so the server and
// the PSSO client simulator share one implementation. What remains here is
// server-only: it touches the datastore (resolving a device's registered key by
// kid), Fleet's PSSO signing key, or the opaque key_context Fleet seals under a
// server key and the device round-trips verbatim.

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/psso/pssocrypto"
	jwt "github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/hkdf"
)

// parsePSSOInboundJWT verifies the inbound compact JWS using the device's
// signing pubkey (resolved by kid) and returns the parsed claims plus the
// signing key row that matched (its HostUUID identifies the device).
func (svc *Service) parsePSSOInboundJWT(ctx context.Context, jwtBytes []byte) (*pssocrypto.TokenClaims, *fleet.PSSOKey, error) {
	// First parse without verification to extract kid.
	unverified, _, err := jwt.NewParser(jwt.WithoutClaimsValidation()).ParseUnverified(string(jwtBytes), &pssocrypto.TokenClaims{})
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "parse inbound psso jwt header")
	}
	kid, _ := unverified.Header["kid"].(string)
	if kid == "" {
		return nil, nil, &fleet.BadRequestError{Message: "psso jwt missing kid header"}
	}
	kid = pssocrypto.CanonicalizeKID(kid)

	signKey, err := svc.ds.GetPSSOKey(ctx, kid)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "look up psso key by kid")
	}
	if signKey.KeyType != fleet.PSSOKeyTypeSigning {
		return nil, nil, &fleet.BadRequestError{Message: "psso jwt kid does not reference a signing key"}
	}

	pub, err := pssocrypto.ParseECPublicKeyPEM([]byte(signKey.PEM))
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "parse device signing pubkey")
	}

	// Pin the algorithm to ES256 (the only alg the Secure Enclave-backed
	// extension signs with) and assert the ECDSA method in the keyfunc. Without
	// this, a future refactor returning a non-EC key could open an alg-confusion
	// forgery path even though golang-jwt's type assertions currently prevent it.
	tok, err := jwt.ParseWithClaims(string(jwtBytes), &pssocrypto.TokenClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("psso jwt: unexpected signing method %q", t.Method.Alg())
		}
		return pub, nil
	}, jwt.WithValidMethods([]string{pssocrypto.SigningAlg}))
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "verify psso jwt signature")
	}
	claims, ok := tok.Claims.(*pssocrypto.TokenClaims)
	if !ok || !tok.Valid {
		return nil, nil, &fleet.BadRequestError{Message: "psso jwt claims invalid"}
	}
	return claims, signKey, nil
}

// resolvePSSOEncryptionKey returns the registered encryption public key the
// response JWE must be wrapped to. The device names its encryption key inside
// the request's apv party-info blob ("Apple" || deviceEncKey || nonce), and the
// extension registered that key under kid = base64url(SHA-256(raw key bytes)) —
// so the kid is recomputed from apv and looked up. As a fallback against any
// re-encoding of the key by Apple's framework, the raw point is compared against
// each of the host's registered encryption keys. A key that resolves but belongs
// to a different host, or doesn't resolve at all, is rejected: responses are only
// ever encrypted to keys the host registered.
func (svc *Service) resolvePSSOEncryptionKey(ctx context.Context, hostUUID, apvB64 string) (*ecdsa.PublicKey, error) {
	apvRaw, err := pssocrypto.DecodeJOSEB64(apvB64)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decode apv")
	}
	fields, err := pssocrypto.ParseApplePartyInfo(apvRaw)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parse apv party-info")
	}
	if len(fields) < 2 || string(fields[0]) != pssocrypto.APVPartyLabel {
		return nil, &fleet.BadRequestError{Message: "psso: apv is not an Apple party-info blob"}
	}
	encKeyRaw := fields[1]

	sum := sha256.Sum256(encKeyRaw)
	kid := pssocrypto.CanonicalizeKID(base64.RawURLEncoding.EncodeToString(sum[:]))
	key, err := svc.ds.GetPSSOKey(ctx, kid)
	switch {
	case err == nil:
		if key.KeyType != fleet.PSSOKeyTypeEncryption || key.HostUUID != hostUUID {
			return nil, &fleet.BadRequestError{Message: "psso: apv key is not a registered encryption key for this device"}
		}
		return pssocrypto.ParseECPublicKeyPEM([]byte(key.PEM))
	case !fleet.IsNotFound(err):
		return nil, ctxerr.Wrap(ctx, err, "look up encryption key by apv kid")
	}

	apvPub, err := pssocrypto.ParseRawECPoint(encKeyRaw)
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
		pub, err := pssocrypto.ParseECPublicKeyPEM([]byte(k.PEM))
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

// deriveKeyContextKey derives the AES-256 key that seals key_context blobs, from
// Fleet's PSSO signing key. This lets the provisioned private key live
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
const pssoKeyPurposeUserUnlock = pssocrypto.KeyPurposeUserUnlock

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

// pssoSessionInfo is the HKDF info string distinguishing PSSO-derived keys from
// any other derivation the same input keying material could feed.
var pssoSessionInfo = []byte("fleetdm-psso-session-key-v1")

// deriveSessionKey returns a 32-byte AES-256 key derived from ikm via
// HKDF-SHA256. The salt parameter binds the derivation to a purpose (e.g. the
// key_context info string in deriveKeyContextKey).
func deriveSessionKey(ikm []byte, salt []byte) ([]byte, error) {
	r := hkdf.New(sha256.New, ikm, salt, pssoSessionInfo)
	out := make([]byte, 32)
	if _, err := r.Read(out); err != nil {
		return nil, fmt.Errorf("hkdf read: %w", err)
	}
	return out, nil
}

// buildSymmetricJWE returns an A256GCM JWE of payload, keyed by sessionKey. Used
// to seal key_context blobs so the provisioned private key can round-trip
// statelessly between key_request and key_exchange.
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
	// JOSE-compatible flat-JSON serialization keeps the result inspectable for
	// the POC. A real client may require compact form; switch when confirmed
	// against the extension's expectations.
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

// decryptSymmetricBlob is the inverse of buildSymmetricJWE — used to open the
// key_context blob a device echoes back in a key-exchange request.
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
// Fleet's PSSO signing key. Used to wrap payloads that must be authenticated as
// coming from Fleet (e.g. claims responses).
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
