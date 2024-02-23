// Package storage defines interfaces, types, data, and helpers related
// to storage and retrieval for MDM enrollments and commands.
package storage

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

// CheckinStore stores MDM check-in data.
type CheckinStore interface {
	StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error
	StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error
	StoreUserAuthenticate(r *mdm.Request, msg *mdm.UserAuthenticate) error
	Disable(r *mdm.Request) error
}

// CommandAndReportResultsStore stores and retrieves MDM command queue data.
type CommandAndReportResultsStore interface {
	StoreCommandReport(r *mdm.Request, report *mdm.CommandResults) error
	RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.Command, error)
	ClearQueue(r *mdm.Request) error
}

type BootstrapTokenStore interface {
	StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error
	RetrieveBootstrapToken(r *mdm.Request, msg *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error)
}

// ServiceStore stores & retrieves both command and check-in data.
type ServiceStore interface {
	CheckinStore
	CommandAndReportResultsStore
	BootstrapTokenStore
}

// PushStore stores and retrieves APNs push-related data.
type PushStore interface {
	RetrievePushInfo(context.Context, []string) (map[string]*mdm.Push, error)
}

// PushCertStore stores and retrieves APNs push certificates.
type PushCertStore interface {
	// IsPushCertStale asks a PushStore if the staleToken it has
	// is stale or not. The staleToken is returned from RetrievePushCert
	// and should turn stale (and return true) if the certificate has
	// changed—such as being renewed.
	IsPushCertStale(ctx context.Context, topic string, staleToken string) (bool, error)
	RetrievePushCert(ctx context.Context, topic string) (cert *tls.Certificate, staleToken string, err error)
	StorePushCert(ctx context.Context, pemCert, pemKey []byte) error
}

// CommandEnqueuer is able to enqueue MDM commands.
type CommandEnqueuer interface {
	EnqueueCommand(ctx context.Context, id []string, cmd *mdm.Command) (map[string]error, error)
}

// CertAuthStore stores and retrieves cert-to-enrollment associations.
type CertAuthStore interface {
	HasCertHash(r *mdm.Request, hash string) (bool, error)
	EnrollmentHasCertHash(r *mdm.Request, hash string) (bool, error)
	IsCertHashAssociated(r *mdm.Request, hash string) (bool, error)
	AssociateCertHash(r *mdm.Request, hash string, certNotValidAfter time.Time) error
}

// StoreMigrator retrieves MDM check-ins
type StoreMigrator interface {
	// RetrieveMigrationCheckins sends the (decoded) forms of
	// Authenticate and TokenUpdate messages to the provided channel.
	// Note that order matters: device channel TokenUpdate messages must
	// follow Authenticate messages and user channel TokenUpdates must
	// follow the device channel TokenUpdate.
	RetrieveMigrationCheckins(context.Context, chan<- interface{}) error
}

// TokenUpdateTallyStore retrieves the TokenUpdate tally (count) for an id
type TokenUpdateTallyStore interface {
	RetrieveTokenUpdateTally(ctx context.Context, id string) (int, error)
}
