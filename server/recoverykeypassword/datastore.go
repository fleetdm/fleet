package recoverykeypassword

import (
	"context"
	"time"
)

// HostRecoveryKeyPassword represents a recovery key password for a host.
type HostRecoveryKeyPassword struct {
	Password  string
	UpdatedAt time.Time
}

// Datastore defines the data access interface for recovery key passwords.
type Datastore interface {
	// SetHostRecoveryKeyPassword generates a new recovery key password,
	// encrypts it, and stores it for the given host. Returns the plaintext password.
	SetHostRecoveryKeyPassword(ctx context.Context, hostID uint) (string, error)

	// GetHostRecoveryKeyPassword retrieves and decrypts the recovery key password
	// for the given host.
	GetHostRecoveryKeyPassword(ctx context.Context, hostID uint) (*HostRecoveryKeyPassword, error)
}
