package licensing

import (
	"crypto/ecdsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
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

func LoadLicense(licenseKey string) (*kolide.LicenseInfo, error) {
	// No license key
	if licenseKey == "" {
		return &kolide.LicenseInfo{Tier: "core"}, nil
	}

	token, err := jwt.ParseWithClaims(
		licenseKey,
		&jwt.MapClaims{},
		// Always use the same public key
		func(*jwt.Token) (interface{}, error) {
			return loadPublicKey()
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "parse license")
	}

	fmt.Println(token.Claims)

	return &kolide.LicenseInfo{Tier: "basic"}, nil
}
