// Package recoverykeypassword provides recovery lock password management for macOS hosts.
package recoverykeypassword

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Service is the public interface for the recovery key password bounded context.
type Service interface {
	// Reconcile runs the reconciliation loop for recovery lock passwords.
	// This processes pending SetRecoveryLock commands and sends new ones to eligible hosts.
	// It is called by the cron job.
	Reconcile(ctx context.Context) error

	// NewResultsHandler returns an MDM command results handler for VerifyRecoveryLock commands.
	// This handler processes the device's response to verification commands.
	NewResultsHandler() fleet.MDMCommandResultsHandler
}

// MDMCommander defines the MDM operations needed by the recovery lock service.
type MDMCommander interface {
	// EnqueueCommand enqueues a raw MDM command for the given host UUIDs.
	EnqueueCommand(ctx context.Context, hostUUIDs []string, rawCommand string) error
	// SendNotifications sends APNs push notifications to wake up devices.
	SendNotifications(ctx context.Context, hostUUIDs []string) error
}

// service implements Service.
type service struct {
	ds        Datastore
	commander MDMCommander
	logger    *slog.Logger
}

// NewService creates a new service with the given dependencies.
// This is called by the bootstrap package to create the service.
func NewService(ds Datastore, commander MDMCommander, logger *slog.Logger) Service {
	return &service{
		ds:        ds,
		commander: commander,
		logger:    logger,
	}
}

// Reconcile implements Service.
func (s *service) Reconcile(ctx context.Context) error {
	return ReconcileRecoveryLockPasswords(ctx, s.ds, s.commander, s.logger)
}

// NewResultsHandler implements Service.
func (s *service) NewResultsHandler() fleet.MDMCommandResultsHandler {
	return NewVerifyRecoveryLockResultsHandler(s.ds, s.logger)
}
