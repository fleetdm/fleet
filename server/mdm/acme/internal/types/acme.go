package types

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
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

type ACMEEnrollment struct {
	ID             uint       `db:"id"`
	PathIdentifier string     `db:"path_identifier"`
	HostIdentifier string     `db:"host_identifier"`
	NotValidAfter  *time.Time `db:"not_valid_after"`
	Revoked        bool       `db:"revoked"`
}

// IsValid returns true if the enrollment is still valid
// (not revoked and not expired).
func (a *ACMEEnrollment) IsValid() bool {
	if a.NotValidAfter != nil && !a.NotValidAfter.IsZero() && time.Now().After(*a.NotValidAfter) {
		return false
	}
	return !a.Revoked
}

// AppleACMEBaseURL returns the base URL for the Apple ACME server, which is
// the Fleet server URL by default, but can be overridden by the FLEET_DEV_STEP_CA_SERVER
// environment variable for test purposes.
func AppleACMEBaseURL(serverURL string) string {
	if base := dev_mode.Env("FLEET_DEV_STEP_CA_SERVER"); base != "" {
		return base
	}
	return serverURL
}

// Datastore is the datastore interface for the ACME bounded context.
type Datastore interface {
	GetACMEEnrollment(ctx context.Context, pathIdentifier string) (*ACMEEnrollment, error)
}
