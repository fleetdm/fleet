package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/fleetdm/fleet/v4/server/notifications/internal/types"
)

// GetNotificationForHost returns the notification, or a not-found error if it
// doesn't exist or belongs to a different host than hostID.
func (s *Service) GetNotificationForHost(ctx context.Context, hostID uint, notificationUUID string) (*api.EndUserNotification, error) {
	notification, err := s.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get end user notification")
	}

	// another host's notification is not found rather than forbidden, so the
	// uuid can't be probed
	if notification.HostID != hostID {
		return nil, ctxerr.Wrap(ctx, &types.NotFoundError{Identifier: notificationUUID}, "no notification found for this host")
	}

	return notification, nil
}
