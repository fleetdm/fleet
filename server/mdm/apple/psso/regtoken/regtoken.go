// Package regtoken mints and validates the Apple Platform SSO device
// registration token.
//
// The token authenticates a device to Fleet's PSSO device registration
// endpoint. It is a Fleet-signed JWT (ES256, signed with the PSSO signing key),
// bound to a single host via its UUID (the `sub` claim), and locked to the
// device-registration use via a fixed audience so it cannot be replayed against
// any other endpoint.
package regtoken

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

const (
	// audience locks the token to device registration. Validation rejects any
	// token whose audience differs, so it cannot be presented to another flow.
	audience = "fleet-psso-device-registration"

	// DefaultValidity is the token lifetime. A long lifetime lets a device reuse
	// the same token across re-registrations without resending the profile.
	DefaultValidity = 5 * 365 * 24 * time.Hour

	signingMethod = "ES256"
)

// Mint returns a signed device registration token bound to hostUUID, valid for
// DefaultValidity from now.
func Mint(key *ecdsa.PrivateKey, hostUUID string, now time.Time) (string, error) {
	if key == nil {
		return "", errors.New("regtoken: nil signing key")
	}
	if hostUUID == "" {
		return "", errors.New("regtoken: empty host UUID")
	}

	claims := jwt.RegisteredClaims{
		Subject:   hostUUID,
		Audience:  jwt.ClaimStrings{audience},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(DefaultValidity)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	kid, err := computeKID(&key.PublicKey)
	if err != nil {
		return "", fmt.Errorf("regtoken: compute kid: %w", err)
	}
	tok.Header["kid"] = kid

	signed, err := tok.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("regtoken: sign: %w", err)
	}
	return signed, nil
}

// MintFromPEM parses a SEC1 EC private key PEM (Fleet's stored PSSO signing key)
// and mints a token for hostUUID. It exists so callers that hold the raw asset
// PEM (e.g. the datastore at command-delivery time) need not duplicate the key
// parsing or pull in the service layer.
func MintFromPEM(signingKeyPEM []byte, hostUUID string, now time.Time) (string, error) {
	key, err := parseECPrivateKeyPEM(signingKeyPEM)
	if err != nil {
		return "", err
	}
	return Mint(key, hostUUID, now)
}

// Validate verifies the token's ES256 signature against key, checks the
// expiry and audience against now, and returns the bound host UUID (the `sub`
// claim). A non-nil error means the token must be rejected.
func Validate(tokenString string, key *ecdsa.PublicKey, now time.Time) (string, error) {
	if key == nil {
		return "", errors.New("regtoken: nil verification key")
	}

	var claims jwt.RegisteredClaims
	// WithoutClaimsValidation skips golang-jwt's implicit time check (which uses
	// the package-global clock); we validate expiry explicitly against `now`
	// below. Signature verification still runs via the keyfunc.
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{signingMethod}),
		jwt.WithoutClaimsValidation(),
	)
	if _, err := parser.ParseWithClaims(tokenString, &claims, func(*jwt.Token) (any, error) {
		return key, nil
	}); err != nil {
		return "", fmt.Errorf("regtoken: parse: %w", err)
	}

	if !claims.VerifyExpiresAt(now, true) {
		return "", errors.New("regtoken: token expired or missing expiry")
	}
	// Reject a token whose issued-at is in the future (or absent). Fleet always
	// mints with iat=now, so a future iat indicates a malformed or tampered
	// token even though only Fleet's key can produce a valid signature.
	if !claims.VerifyIssuedAt(now, true) {
		return "", errors.New("regtoken: issued-at is in the future or missing")
	}
	if !claims.VerifyAudience(audience, true) {
		return "", errors.New("regtoken: wrong or missing audience")
	}
	if claims.Subject == "" {
		return "", errors.New("regtoken: missing subject")
	}
	return claims.Subject, nil
}

func parseECPrivateKeyPEM(pemBytes []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("regtoken: pem decode returned nil block")
	}
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("regtoken: parse ec private key: %w", err)
	}
	return key, nil
}

// computeKID returns base64url-nopad SHA-256 of the SubjectPublicKeyInfo DER
// encoding of pub, matching the kid Fleet uses for its PSSO signing key
// elsewhere (JWKS/JWT).
func computeKID(pub *ecdsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}
