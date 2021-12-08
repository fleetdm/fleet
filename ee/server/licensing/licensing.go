package licensing

import (
	"crypto/ecdsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/golang-jwt/jwt/v4"
)

const (
	expectedAlgorithm = "ES256"
	expectedIssuer    = "Fleet Device Management Inc."
)

//go:embed pubkey.pem
var pubKeyPEM []byte

// loadPublicKey loads the public key from pubkey.pem.
func loadPublicKey() (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pubKeyPEM)
	if block == nil {
		return nil, errors.New("no key block found in pem")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ecdsa key: %w", err)
	}

	if pub, ok := pub.(*ecdsa.PublicKey); ok {
		return pub, nil
	}
	return nil, fmt.Errorf("%T is not *ecdsa.PublicKey", pub)
}

// LoadLicense loads and validates the license key.
func LoadLicense(licenseKey string) (*fleet.LicenseInfo, error) {
	// No license key
	if licenseKey == "" {
		return &fleet.LicenseInfo{Tier: fleet.TierFree}, nil
	}

	parsedToken, err := jwt.ParseWithClaims(
		licenseKey,
		&licenseClaims{},
		// Always use the same public key
		func(*jwt.Token) (interface{}, error) {
			return loadPublicKey()
		},
	)
	if err != nil {
		v, _ := err.(*jwt.ValidationError)

		// if the ONLY error is that it's expired, then we ignore it
		if v == nil || v.Errors != jwt.ValidationErrorExpired {
			return nil, fmt.Errorf("parse license: %w", err)
		}
		parsedToken.Valid = true
	}

	license, err := validate(parsedToken)
	if err != nil {
		return nil, fmt.Errorf("validate license: %w", err)
	}

	// Force premium license
	if license.IsPremium() {
		license.Tier = fleet.TierPremium
	}
	return license, nil
}

type licenseClaims struct {
	// jwt.StandardClaims includes validation for iat, nbf, and exp.
	jwt.StandardClaims
	Tier    string `json:"tier"`
	Devices int    `json:"devices"`
	Note    string `json:"note"`
}

func validate(token *jwt.Token) (*fleet.LicenseInfo, error) {
	// token.IssuedAt, token.ExpiresAt, token.NotBefore already validated by JWT
	// library.
	if !token.Valid {
		// ParseWithClaims should have errored already, but double-check here
		return nil, errors.New("token invalid")
	}

	if token.Method.Alg() != expectedAlgorithm {
		return nil, fmt.Errorf("unexpected algorithm %s", token.Method.Alg())
	}

	var claims *licenseClaims
	claims, ok := token.Claims.(*licenseClaims)
	if !ok || claims == nil {
		return nil, fmt.Errorf("unexpected claims type %T", token.Claims)
	}

	if claims.Devices == 0 {
		return nil, errors.New("missing devices")
	}

	if claims.Tier == "" {
		return nil, errors.New("missing tier")
	}

	if claims.ExpiresAt == 0 {
		return nil, errors.New("missing exp")
	}

	if claims.Issuer != expectedIssuer {
		return nil, fmt.Errorf("unexpected issuer %s", claims.Issuer)
	}

	return &fleet.LicenseInfo{
		Tier:         claims.Tier,
		Organization: claims.Subject,
		DeviceCount:  claims.Devices,
		Expiration:   time.Unix(claims.ExpiresAt, 0),
		Note:         claims.Note,
	}, nil

}
