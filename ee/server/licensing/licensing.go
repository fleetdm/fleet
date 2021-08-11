package licensing

import (
	"crypto/ecdsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/golang-jwt/jwt/v4"
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
func LoadLicense(licenseKey string) (*fleet.LicenseInfo, error) {
	// No license key
	if licenseKey == "" {
		return &fleet.LicenseInfo{Tier: fleet.TierCore}, nil
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

	return &fleet.LicenseInfo{
		Tier:         claims.Tier,
		Organization: claims.Subject,
		DeviceCount:  claims.Devices,
		Expiration:   time.Unix(claims.ExpiresAt, 0),
		Note:         claims.Note,
	}, nil

}
