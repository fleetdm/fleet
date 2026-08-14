// Package service provides the service implementation for the notifications
// bounded context.
package service

import (
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/notifications"
	"github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/fleetdm/fleet/v4/server/notifications/internal/types"
)

// Service is the notifications bounded context service implementation.
type Service struct {
	ds          types.Datastore
	scriptQueue notifications.ScriptQueueProvider
	logger      *slog.Logger

	// kinds is populated by RegisterKind before the server starts serving
	// requests, so it's never written to concurrently with a read.
	kinds map[string]api.NotificationKind
}

// NewService creates a new notifications service.
func NewService(
	ds types.Datastore,
	providers notifications.DataProviders,
	logger *slog.Logger,
) *Service {
	return &Service{
		ds:          ds,
		scriptQueue: providers,
		logger:      logger,
		kinds:       make(map[string]api.NotificationKind),
	}
}

// Ensure Service implements api.Service
var _ api.Service = (*Service)(nil)

// RegisterKind registers a kind of end user notification. Must be called
// before the server starts serving requests.
func (s *Service) RegisterKind(kind api.NotificationKind) {
	s.kinds[kind.Name()] = kind
}
