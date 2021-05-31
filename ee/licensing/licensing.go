package licensing

import (
	"crypto/ecdsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
)

const (
	expectedAlgorithm = "ES256"
	expectedIssuer    = "Fleet Device Management Inc."
)

//go:embed pubkey.pem
var pubKeyPEM []byte

// loadPublicKey loads the public key from pubkey.pem.
func loadPublicKey() (*ecdsa.PublicKey, error) {
	// pub, err := jwt.ParseECPublicKeyFromPEM(pubKeyPEM)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "parse public key")
	// }
	// return pub, nil
	block, _ := pem.Decode(pubKeyPEM)
	if block == nil {
		return nil, errors.New("no key block found in pem")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse ecdsa key")
	}

	if pub, ok := pub.(*ecdsa.PublicKey); ok {
		return pub, nil
	} else {
		return nil, errors.Errorf("%T is not *ecdsa.PublicKey", pub)
	}
}

// LoadLicense loads and validates the license key.
func LoadLicense(licenseKey string) (*kolide.LicenseInfo, error) {
	// No license key
	if licenseKey == "" {
		return &kolide.LicenseInfo{Tier: kolide.TierCore}, nil
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
		return nil, errors.Wrap(err, "parse license")
	}

	license, err := validate(parsedToken)
	if err != nil {
		return nil, errors.Wrap(err, "validate license")
	}

	return license, nil
}

type licenseClaims struct {
	jwt.StandardClaims
	Tier    string `json:"tier"`
	Devices int    `json:"devices"`
	Note    string `json:"note"`
}

func validate(token *jwt.Token) (*kolide.LicenseInfo, error) {
	if !token.Valid {
		// ParseWithClaims should error anyway, but double-check here
		return nil, errors.New("token invalid")
	}

	if token.Method.Alg() != expectedAlgorithm {
		return nil, errors.Errorf("unexpected algorithm %s", token.Method.Alg())
	}

	var claims *licenseClaims
	claims, ok := token.Claims.(*licenseClaims)
	if !ok || claims == nil {
		return nil, errors.Errorf("unexpected claims type %T", token.Claims)
	}

	if claims.Devices == 0 {
		return nil, errors.Errorf("missing devices")
	}

	if claims.Tier == "" {
		return nil, errors.Errorf("missing tier")
	}

	if claims.ExpiresAt == 0 {
		return nil, errors.Errorf("missing exp")
	}

	if claims.Issuer != expectedIssuer {
		return nil, errors.Errorf("unexpected issuer %s", claims.Issuer)
	}

	// We explicitly do not validate expiration at this time because we want to
	// allow some flexibility for expired tokens.

	return &kolide.LicenseInfo{
		Tier:         claims.Tier,
		Organization: claims.Subject,
		DeviceCount:  claims.Devices,
		Expiration:   time.Unix(claims.ExpiresAt, 0),
		Note:         claims.Note,
	}, nil

}
