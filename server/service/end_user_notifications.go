package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// notificationKindRegistry is every kind of end user notification Fleet knows how
// to act on, by the name stored in end_user_notifications.kind. A kind is added
// here and nowhere else: core resolves the column through this map rather than
// switching on it, so adding one touches no delivery code.
func notificationKindRegistry(ds fleet.Datastore) map[string]fleet.NotificationKind {
	kinds := []fleet.NotificationKind{
		&patchNotificationKind{ds: ds},
	}

	registry := make(map[string]fleet.NotificationKind, len(kinds))
	for _, kind := range kinds {
		registry[kind.Name()] = kind
	}
	return registry
}

// notificationOutcomeForExitCode maps a notify exit code to a reason and a retry
// wait, nil meaning no retry. Codes come from
// apps/fleet-desktop-macos/FleetDesktop/cli.swift.
func notificationOutcomeForExitCode(exitCode int64) (reason string, retryIn *time.Duration) {
	shortRetry := fleet.EndUserNotificationShortRetryInterval
	longRetry := fleet.EndUserNotificationLongRetryInterval

	switch exitCode {
	case 0:
		return "", nil

	// Fleet asked for something impossible, so asking again changes nothing.
	case 2:
		return fleet.EndUserNotificationReasonBadInvocation, nil
	case 20:
		return fleet.EndUserNotificationReasonBadConfiguration, nil

	// Nobody was there to see it, or the page did not load. Both pass.
	case 30:
		return fleet.EndUserNotificationReasonPageLoadFailed, &shortRetry
	case 31:
		return fleet.EndUserNotificationReasonHTTPError, &shortRetry
	case 40:
		return fleet.EndUserNotificationReasonNoGUIUser, &shortRetry
	case 41:
		return fleet.EndUserNotificationReasonScreenLocked, &shortRetry
	case 42:
		return fleet.EndUserNotificationReasonNoDisplay, &shortRetry
	case 70:
		return fleet.EndUserNotificationReasonInternalError, &shortRetry

	// An admin has to deploy or upgrade Fleet Desktop, which will not happen in
	// the next half hour.
	case 100:
		return fleet.EndUserNotificationReasonDesktopMissing, &longRetry
	case 101:
		return fleet.EndUserNotificationReasonDesktopTooOld, &longRetry

	default:
		return fleet.EndUserNotificationReasonUnexpectedFailure, &shortRetry
	}
}

// recordEndUserNotificationOutcome records the outcome of a script result that
// belongs to a notification, and hands it to the kind.
func (svc *Service) recordEndUserNotificationOutcome(ctx context.Context, result *fleet.HostScriptResultPayload) error {
	if result == nil || result.ExecutionID == "" {
		return nil
	}

	notification, err := svc.ds.GetEndUserNotificationByExecutionID(ctx, result.ExecutionID)
	if err != nil {
		if fleet.IsNotFound(err) {
			// the overwhelming majority of script results, which have nothing to
			// do with notifications
			return nil
		}
		return ctxerr.Wrap(ctx, err, "get end user notification for script result")
	}

	exitCode := int64(result.ExitCode)
	reason, retryIn := notificationOutcomeForExitCode(exitCode)
	outcome := fleet.NotificationOutcome{
		Displayed:   exitCode == 0,
		ExitCode:    exitCode,
		Reason:      reason,
		Output:      result.Output,
		ExecutionID: result.ExecutionID,
	}

	var nextAttemptAt *time.Time
	if retryIn != nil {
		nextAttempt := time.Now().UTC().Add(*retryIn)
		nextAttemptAt = &nextAttempt
	}

	if err := svc.ds.SetEndUserNotificationOutcome(ctx, notification.UUID, outcome, nextAttemptAt); err != nil {
		return ctxerr.Wrap(ctx, err, "set end user notification outcome")
	}

	kind, kindRegistered := svc.notificationKinds[notification.Kind]
	if !kindRegistered {
		svc.logger.WarnContext(ctx, "no kind registered for end user notification",
			"kind", notification.Kind, "notification_uuid", notification.UUID)
		return nil
	}
	if err := kind.OnOutcome(ctx, notification, outcome); err != nil {
		return ctxerr.Wrap(ctx, err, "notification kind handling outcome")
	}
	return nil
}

// applyEndUserNotificationAction validates the device's report and carries out
// the action. Verify always writes displayed_at, since other logic depends on
// it; delay only happens if a kind is registered to decide it.
func (svc *Service) applyEndUserNotificationAction(ctx context.Context, notification *fleet.EndUserNotification, action fleet.EndUserNotificationAction) error {
	if action.Action == nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("action", "is required"))
	}

	kind, kindRegistered := svc.notificationKinds[notification.Kind]
	if !kindRegistered {
		svc.logger.WarnContext(ctx, "no kind registered for end user notification",
			"kind", notification.Kind, "notification_uuid", notification.UUID)
	}

	switch *action.Action {
	case fleet.EndUserNotificationActionVerify:
		// Always verifies displayed_at for the notification. This case is currently unused, as
		// we currently trigger script runs to display notifications and set when they were
		// displayed when the script result is saved.
		displayedAt := time.Now().UTC()
		if action.DisplayedAt != nil {
			displayedAt = action.DisplayedAt.UTC()
		}
		// a device that reports a time in the future would postpone whatever the
		// kind counts from it
		if displayedAt.After(time.Now().UTC()) {
			displayedAt = time.Now().UTC()
		}
		if err := svc.ds.VerifyEndUserNotification(ctx, notification.UUID, displayedAt); err != nil {
			return ctxerr.Wrap(ctx, err, "verify end user notification")
		}
		if kindRegistered {
			if err := kind.OnVerify(ctx, notification, displayedAt); err != nil {
				return ctxerr.Wrap(ctx, err, "notification kind handling verify")
			}
		}

	case fleet.EndUserNotificationActionDelay:
		if !kindRegistered {
			return nil
		}
		if err := kind.OnDelay(ctx, notification); err != nil {
			return ctxerr.Wrap(ctx, err, "notification kind handling delay")
		}

	default:
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("action",
			fmt.Sprintf("%q is not something that can be done to a notification", *action.Action)))
	}

	return nil
}
