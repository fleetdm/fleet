package types

import (
	"context"
	"time"

	"go.step.sm/crypto/jose"
)

type Directory struct {
	NewNonce   string `json:"newNonce"`
	NewAccount string `json:"newAccount"`
	NewOrder   string `json:"newOrder"`
	NewAuthz   string `json:"newAuthz,omitempty"`
	RevokeCert string `json:"revokeCert,omitempty"`
	KeyChange  string `json:"keyChange,omitempty"`
	Meta       Meta   `json:"meta"`
}

type Meta struct {
	TermsOfService          string   `json:"termsOfService,omitempty"`
	Website                 string   `json:"website,omitempty"`
	CaaIdentities           []string `json:"caaIdentities,omitempty"`
	ExternalAccountRequired bool     `json:"externalAccountRequired,omitempty"`
}

type Enrollment struct {
	ID             uint       `db:"id"`
	PathIdentifier string     `db:"path_identifier"`
	HostIdentifier string     `db:"host_identifier"`
	NotValidAfter  *time.Time `db:"not_valid_after"`
	Revoked        bool       `db:"revoked"`
}

// IsValid returns true if the enrollment is still valid
// (not revoked and not expired).
func (a *Enrollment) IsValid() bool {
	if a.NotValidAfter != nil && !a.NotValidAfter.IsZero() && time.Now().After(*a.NotValidAfter) {
		return false
	}
	return !a.Revoked
}

type Account struct {
	ID                   uint            `db:"id"`
	ACMEEnrollmentID     uint            `db:"acme_enrollment_id"`
	JSONWebKey           jose.JSONWebKey `db:"-"`
	JSONWebKeyThumbprint string          `db:"json_web_key_thumbprint"`
	Revoked              bool            `db:"revoked"`
}

type AccountResponse struct {
	CreatedAccount *Account `json:"-"`
	DidCreate      bool     `json:"-"`
	Status         string   `json:"status"`
	Contact        []string `json:"contact,omitempty"`
	Orders         string   `json:"orders"`
	Location       string   `json:"-"`
}

type Order struct {
	ID                        uint   `db:"id"`
	ACMEAccountID             uint   `db:"acme_account_id"`
	Finalized                 bool   `db:"finalized"`
	CertificateSigningRequest string `db:"certificate_signing_request"`
	// Identifiers is manually serialized to JSON when inserted, and should do the same
	// when read (or we could implement sql.Scanner).
	Identifiers             []Identifier `db:"-"`
	Status                  string       `db:"status"`
	IssuedCertificateSerial *uint        `db:"issued_certificate_serial"`

	// NotBefore and NotAfter must not be set, we capture them so we can validate
	// that they were indeed not provided.
	NotBefore *time.Time `db:"-"`
	NotAfter  *time.Time `db:"-"`
}

type OrderResponse struct {
	ID             uint         `json:"id"`
	Status         string       `json:"status"`
	Expires        *time.Time   `json:"expires,omitempty"`
	Identifiers    []Identifier `json:"identifiers"`
	Authorizations []string     `json:"authorizations"`
	Finalize       string       `json:"finalize"`
	Certificate    string       `json:"certificate,omitempty"`

	// Location is set in the header, pointing to the created order's URL.
	Location string `json:"-"`
}

type Authorization struct {
	ID          uint       `db:"id"`
	ACMEOrderID uint       `db:"acme_order_id"`
	Identifier  Identifier `db:"-"`
	Status      string     `db:"status"`
}

type AuthorizationResponse struct {
	Status     string              `json:"status"`
	Expires    *time.Time          `json:"expires,omitempty"`
	Identifier Identifier          `json:"identifier"`
	Challenges []ChallengeResponse `json:"challenges"`

	// Location is set in the header, pointing to the requested authorization's URL.
	Location string `json:"-"`
}

const (
	DeviceAttestationChallengeType string = "device-attest-01"
)

type Challenge struct {
	ID                  uint   `db:"id"`
	ACMEAuthorizationID uint   `db:"acme_authorization_id"`
	ChallengeType       string `db:"challenge_type"`
	Token               string `db:"token"`
	Status              string `db:"status"`
}

type ChallengeResponse struct {
	ChallengeType string `json:"type"`
	Status        string `json:"status"`
	Token         string `json:"token"`
	URL           string `json:"url"`

	// Validated is only set in the response when the challenge is valid and has been validated.
	Validated *time.Time `json:"validated,omitempty"`
}

const (
	IdentifierTypePermanentIdentifier = "permanent-identifier"
)

type AccountAuthenticatedRequest interface {
	SetEnrollmentAndAccount(enrollment *Enrollment, account *Account)
}

// The base struct for allowing arbitrary types to implement the interface above. It is important that these
// members not be serialized to/from JSON as they are meant to be set by the service after authentication and
// not by the client.
type AccountAuthenticatedRequestBase struct {
	Enrollment *Enrollment `json:"-"`
	Account    *Account    `json:"-"`
}

func (r *AccountAuthenticatedRequestBase) SetEnrollmentAndAccount(enrollment *Enrollment, account *Account) {
	r.Enrollment = enrollment
	r.Account = account
}

// Represents acme identifiers (not to be confused with enrollment identifiers)
// which, in our usecase, represent identifiers(e.g. serials) that hosts control.
type Identifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Datastore is the datastore interface for the ACME service module.
type Datastore interface {
	NewEnrollment(ctx context.Context, hostIdentifier string) (string, error)
	GetACMEEnrollment(ctx context.Context, pathIdentifier string) (*Enrollment, error)
	GetAccountByID(ctx context.Context, enrollmentID uint, accountID uint) (*Account, error)
	CreateAccount(ctx context.Context, account *Account, onlyReturnExisting bool) (*Account, bool, error)
	CreateOrder(ctx context.Context, order *Order, authorization *Authorization, challenge *Challenge) (*Order, error)
	GetOrderByID(ctx context.Context, accountID, orderID uint) (*Order, []*Authorization, error)
	ListAccountOrderIDs(ctx context.Context, accountID uint) ([]uint, error)
	GetAuthorizationByID(ctx context.Context, accountID uint, authorizationID uint) (*Authorization, error)
	GetChallengesByAuthorizationID(ctx context.Context, authorizationID uint) ([]*Challenge, error)
}
