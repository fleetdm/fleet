package kolide

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

const (
	LicenseGracePeriod = time.Hour * 24 * 60 // 60 days
	hostLimitUnlimited = 0
)

type LicenseStore interface {
	// SaveLicense saves jwt formatted customer license information
	SaveLicense(tokenString, publicKey string) (*License, error)
	// License returns a structure with the jwt customer license if it exists.
	License() (*License, error)
	// LicensePublicKey gets the public key associated with this license
	LicensePublicKey(tokenString string) (string, error)
	// RevokeLicense sets revoked status of license
	RevokeLicense(revoked bool) error
}

type LicenseService interface {
	// License returns details of a customer license that determine authorization
	// to use the Kolide product. If the customer has not uploaded a token,
	// the license Token will be nil.
	License(ctx context.Context) (*License, error)

	// SaveLicense writes jwt token string to database after performing
	// validation
	SaveLicense(ctx context.Context, jwtToken string) (*License, error)
}

// Contains information needed to extract customer license particulars.
type License struct {
	UpdateTimestamp
	ID uint
	// Token is a jwt token
	Token *string `db:"token"`
	// PublicKey is used to validate the Token and extract claims
	PublicKey string `db:"key"`
	// Revoked if true overrides a license that might otherwise be valid
	Revoked bool
	// HostCount is the count of enrolled hosts
	HostCount uint `db:"-"`
}

// LicenseClaims contains information about the rights of a customer to
// use the Kolide product
type Claims struct {
	LicenseUUID      string
	OrganizationName string
	OrganizationUUID string
	// HostLimit the maximum number of hosts that a customer can use. 0 is unlimited.
	HostLimit int
	// Evaluation indicates that Kolide can be used for eval only.
	Evaluation bool
	// ExpiresAt time when license expires
	ExpiresAt time.Time
	// HostCount number of enrolled hosts
	HostCount int
}

// Expired returns true if the license is expired
func (c *Claims) Expired(current time.Time) bool {
	if c.Evaluation && c.ExpiresAt.Before(current) {
		return true
	}
	if !c.Evaluation && c.ExpiresAt.Add(LicenseGracePeriod).Before(current) {
		return true
	}
	return false
}

// CanEnrollHost returns true if the user is licensed to enroll additional
// hosts
func (c *Claims) CanEnrollHost() bool {
	if c.HostLimit == hostLimitUnlimited {
		return true
	}
	if c.HostCount <= c.HostLimit {
		return true
	}
	return false
}

// Claims returns information contained in the jwt license token
func (l *License) Claims() (*Claims, error) {
	if l.Token == nil {
		return nil, errors.New("license missing")
	}
	token, err := jwt.Parse(*l.Token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(l.PublicKey))
		if err != nil {
			return nil, errors.Wrap(err, "reading license token")
		}
		return key, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "reading licence token")
	}
	var result Claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		result.LicenseUUID = claims["license_uuid"].(string)
		result.OrganizationName = claims["organization_name"].(string)
		result.OrganizationUUID = claims["organization_uuid"].(string)
		result.HostLimit = int(claims["host_limit"].(float64))
		result.Evaluation = claims["evaluation"].(bool)
		expiry, err := time.Parse(time.RFC3339, claims["expires_at"].(string))
		if err != nil {
			return nil, err
		}
		result.ExpiresAt = expiry
	} else {
		return nil, errors.New("license token is not valid")
	}
	result.HostCount = int(l.HostCount)
	return &result, nil
}

// LicenseChecker allows checking that a license is valid by calling in to
// a remote URL.
type LicenseChecker interface {
	RunLicenseCheck(ctx context.Context)
}
