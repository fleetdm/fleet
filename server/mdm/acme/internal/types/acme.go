package types

import (
	"context"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
	"go.step.sm/crypto/jose"
)

const (
	OrderStatusPending = "pending"
	OrderStatusReady   = "ready"
	OrderStatusValid   = "valid"
	OrderStatusInvalid = "invalid"

	AuthorizationStatusPending = "pending"
	AuthorizationStatusValid   = "valid"
	AuthorizationStatusInvalid = "invalid"

	ChallengeStatusPending = "pending"
	ChallengeStatusValid   = "valid"
	ChallengeStatusInvalid = "invalid"
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
	IssuedCertificateSerial *uint64      `db:"issued_certificate_serial"`

	// NotBefore and NotAfter must not be set, we capture them so we can validate
	// that they were indeed not provided.
	NotBefore *time.Time `db:"-"`
	NotAfter  *time.Time `db:"-"`
}

// IsReadyToFinalize returns an error if the order is not in a state where it can be finalized.
func (o Order) IsReadyToFinalize() error {
	if o.Status != OrderStatusReady || o.Finalized {
		extra := ""
		if o.Finalized {
			extra = " and order has already been finalized"
		}
		return OrderNotReadyError(fmt.Sprintf("Order is in status %s%s.", o.Status, extra))
	}

	return nil
}

// IsCertificateReady returns an error if the order is not in a state where the certificate can be retrieved.
func (o Order) IsCertificateReady() error {
	if !o.Finalized || o.Status != OrderStatusValid {
		if o.Status == OrderStatusInvalid {
			return OrderDoesNotExistError("Order is in invalid state, cannot get certificate")
		}
		return OrderNotFinalizedError("Order is not finalized/in valid state, cannot get certificate")
	}
	return nil
}

// ValidateOrderCreation validates that the order creation request is valid given the enrollment. It returns an error if the request is not valid.
func (o Order) ValidateOrderCreation(enrollment *Enrollment) error {
	// The "identifiers" passed as part of the newOrder request must be an array with a
	// single member of type "permanent-identifier" matching the serial specified in the
	// acme_enrollment that this enrollment was created for.
	if len(o.Identifiers) != 1 || o.Identifiers[0].Type != IdentifierTypePermanentIdentifier {
		return UnsupportedIdentifierError("A single identifier of type permanent-identifier must be provided in the order request")
	}
	if o.Identifiers[0].Value != enrollment.HostIdentifier {
		return RejectedIdentifierError("The identifier value does not match the host identifier for this enrollment")
	}

	// notBefore and notAfter, which are optional, must not be set because fleet is going
	// to control these and the Apple payload doesn't allow specification of them.
	if o.NotBefore != nil || o.NotAfter != nil {
		return MalformedError("notBefore and notAfter must not be set in the order request")
	}

	return nil
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
	// UpdatedAt is used as validated timestamp if the challenge is valid
	UpdatedAt time.Time `db:"updated_at"`
}

// ValidatedAt returns the time that the challenge was validated if it is valid, or nil if it is not valid.
func (c Challenge) ValidatedAt() *time.Time {
	if c.Status == ChallengeStatusValid {
		return &c.UpdatedAt
	}
	return nil
}

func (c *Challenge) MarkValid() {
	c.Status = ChallengeStatusValid
}

func (c *Challenge) MarkInvalid() {
	c.Status = ChallengeStatusInvalid
}

type ChallengeResponse struct {
	ChallengeType string `json:"type"`
	Status        string `json:"status"`
	Token         string `json:"token"`
	URL           string `json:"url"`

	// Validated is only set in the response when the challenge is valid and has been validated.
	Validated *time.Time `json:"validated,omitempty"`

	// Location is set in the header, pointing to the requested challenge's URL.
	Location string `json:"-"`
}

// https://www.w3.org/TR/webauthn-2/#sctn-attestation, but we don't use authData as per the ACME RFC.
type AttestationObject struct {
	Format               string          `cbor:"fmt"`
	AttestationStatement cbor.RawMessage `cbor:"attStmt"`
}

// https://www.w3.org/TR/webauthn-2/#sctn-apple-anonymous-attestation
type AppleDeviceAttestationStatement struct {
	X5C [][]byte `cbor:"x5c"`
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
	FinalizeOrder(ctx context.Context, orderID uint, csrPEM string, certSerial int64) error
	GetChallengesByAuthorizationID(ctx context.Context, authorizationID uint) ([]*Challenge, error)
	GetCertificatePEMByOrderID(ctx context.Context, accountID, orderID uint) (string, error)
	GetChallengeByID(ctx context.Context, accountID, challengeID uint) (*Challenge, error)
	// Update challenge handles updating the challenge status, and the authorization status as well as moving the order status.
	UpdateChallenge(ctx context.Context, challenge *Challenge) (*Challenge, error)
}
