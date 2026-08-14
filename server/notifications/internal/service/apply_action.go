package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/notifications/api"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
)

// ApplyAction validates the device's report and carries out the action.
// Delay only happens if a kind is registered to decide it.
func (s *Service) ApplyAction(ctx context.Context, hostID uint, notificationUUID string, action api.EndUserNotificationAction) error {
	notification, err := s.GetNotificationForHost(ctx, hostID, notificationUUID)
	if err != nil {
		return err
	}

	if action.Action == nil {
		return ctxerr.Wrap(ctx, &platform_http.BadRequestError{Message: "action is required"})
	}

	kind, kindRegistered := s.kinds[notification.Kind]
	if !kindRegistered {
		s.logger.WarnContext(ctx, "no kind registered for end user notification",
			"kind", notification.Kind, "notification_uuid", notification.UUID)
	}

	switch *action.Action {
	case api.EndUserNotificationActionVerify:
		// TODO: end users should be able to send the time they displayed a
		// notification at. Currently Fleet uses the script queue to implement
		// notification so their displayed at time is set when the script
		// returns a successful code.

	case api.EndUserNotificationActionDelay:
		if !kindRegistered {
			return nil
		}
		if err := kind.OnDelay(ctx, notification); err != nil {
			return ctxerr.Wrap(ctx, err, "notification kind handling delay")
		}

	default:
		return ctxerr.Wrap(ctx, &platform_http.BadRequestError{
			Message: fmt.Sprintf("%q is not something that can be done to a notification", *action.Action),
		})
	}

	return nil
}
