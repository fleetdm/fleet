package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/fleetdm/fleet/v4/server/notifications/internal/types"
)

func (s *Service) ApplyAction(ctx context.Context, hostID uint, notificationUUID string, action api.EndUserNotificationAction) (*api.NotificationView, error) {
	notification, err := s.notificationForHost(ctx, hostID, notificationUUID)
	if err != nil {
		return nil, err
	}

	if action.Action == nil {
		return nil, ctxerr.Wrap(ctx, &types.InvalidArgumentError{Name: "action", Reason: "is required"})
	}

	kind, kindRegistered := s.kinds[notification.Kind]
	if !kindRegistered {
		s.logger.WarnContext(ctx, "no kind registered for end user notification",
			"kind", notification.Kind, "notification_uuid", notification.UUID)
		return nil, ctxerr.Wrap(ctx, &types.NotFoundError{Identifier: notificationUUID}, "no kind to act on notification")
	}

	switch *action.Action {
	case api.EndUserNotificationActionVerify:
		// TODO: end users should be able to send the time they displayed a
		// notification at. Currently Fleet uses the script queue to implement
		// notification so their displayed at time is set when the script
		// returns a successful code.

	case api.EndUserNotificationActionDelay:
		if err := kind.OnDelay(ctx, notification); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "notification kind handling delay")
		}

	default:
		if err := kind.OnAction(ctx, notification, *action.Action); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "notification kind handling action")
		}
	}

	updated, err := s.ds.GetEndUserNotificationByUUID(ctx, notificationUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reload end user notification after action")
	}
	return s.render(ctx, updated)
}
