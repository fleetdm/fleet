package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// NotificationUUIDForExecution returns the UUID of the notification that
// execution ID belongs to, or a not-found error if it isn't one.
func (s *Service) NotificationUUIDForExecution(ctx context.Context, executionID string) (string, error) {
	notification, err := s.ds.GetEndUserNotificationByExecutionID(ctx, executionID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get end user notification for execution")
	}
	return notification.UUID, nil
}
