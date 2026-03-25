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
	EnrollmentID         uint            `db:"acme_enrollment_id"`
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
	ID             uint         `db:"id" json:"id"`
	AccountID      uint         `db:"account_id" json:"-"`
	Expires        time.Time    `db:"-" json:"expires"`
	Status         string       `db:"status" json:"status"`
	Identifiers    []Identifier `db:"-" json:"identifiers"`
	Authorizations []string     `db:"-" json:"authorizations"`
	Finalize       string       `db:"-" json:"finalize"`
}

type Authorization struct {
	ID         uint       `db:"id" json:"-"`
	OrderID    uint       `db:"order_id" json:"-"`
	Identifier Identifier `db:"-" json:"identifier"`
	// TODO: We should just set this to the overall Enrollment's expires value for now, I think.
	// we can always revisit later
	Expires    time.Time   `db:"-" json:"expires"`
	Status     string      `db:"status" json:"status"`
	Challenges []Challenge `db:"-" json:"challenges"`
}

type Challenge struct {
	ID              uint `db:"id" json:"-"`
	AuthorizationID uint `db:"authorization_id" json:"-"`

	Type   string `db:"challenge_type" json:"type"`
	Token  string `db:"token" json:"token"`
	Status string `db:"status" json:"status"`
	URL    string `db:"-" json:"url"`

	// TODO: We may need to add this to the db or we can use the challenge's updated_at. It
	// only needs to be returned if the challenge is status=valid, so we can set it in the
	// service when the challenge is validated.
	Validated *time.Time `db:"-" json:"validated,omitempty"`
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

// Datastore is the datastore interface for the ACME bounded context.
type Datastore interface {
	GetACMEEnrollment(ctx context.Context, pathIdentifier string) (*Enrollment, error)
	GetAccountByID(ctx context.Context, enrollmentID uint, accountID uint) (*Account, error)
	CreateAccount(ctx context.Context, account *Account, onlyReturnExisting bool) (*Account, bool, error)
}
