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
func notificationKindRegistry() map[string]fleet.NotificationKind {
	kinds := []fleet.NotificationKind{}

	registry := make(map[string]fleet.NotificationKind, len(kinds))
	for _, kind := range kinds {
		registry[kind.Name()] = kind
	}
	return registry
}

// notificationOutcomeForExitCode maps what the notification script reported to
// why it ended that way and how long to wait before trying again, where nil means
// no further attempt.
//
// The codes come from FleetDesktop's notify subcommand and the script that calls
// it, apps/fleet-desktop-macos/FleetDesktop/cli.swift.
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

// recordEndUserNotificationOutcome records how an attempt to display a
// notification ended, if the script result belongs to one, and hands the outcome
// to the kind that asked for it.
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

	kind, ok := svc.notificationKinds[notification.Kind]
	if !ok {
		svc.logger.WarnContext(ctx, "no kind registered for end user notification",
			"kind", notification.Kind, "notification_uuid", notification.UUID)
		return nil
	}
	if err := kind.OnOutcome(ctx, notification, outcome); err != nil {
		return ctxerr.Wrap(ctx, err, "notification kind handling outcome")
	}
	return nil
}

// applyEndUserNotificationAction records what the device reported, then hands the
// action to the kind. Fleet owns the timing of a delay so an end user cannot
// push a notification out forever; only the time it appeared comes from the
// device, which is the one thing Fleet cannot know.
func (svc *Service) applyEndUserNotificationAction(ctx context.Context, notification *fleet.EndUserNotification, action fleet.EndUserNotificationAction) error {
	if action.Action == nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("action", "is required"))
	}

	kind, ok := svc.notificationKinds[notification.Kind]
	if !ok {
		svc.logger.WarnContext(ctx, "no kind registered for end user notification",
			"kind", notification.Kind, "notification_uuid", notification.UUID)
	}

	switch *action.Action {
	case fleet.EndUserNotificationActionVerify:
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
		if ok {
			if err := kind.OnVerify(ctx, notification, displayedAt); err != nil {
				return ctxerr.Wrap(ctx, err, "notification kind handling verify")
			}
		}

	case fleet.EndUserNotificationActionDelay:
		nextAttemptAt := time.Now().UTC().Add(fleet.EndUserNotificationDelayInterval)
		if err := svc.ds.DelayEndUserNotification(ctx, notification.UUID, nextAttemptAt); err != nil {
			return ctxerr.Wrap(ctx, err, "delay end user notification")
		}
		if ok {
			if err := kind.OnDelay(ctx, notification, nextAttemptAt); err != nil {
				return ctxerr.Wrap(ctx, err, "notification kind handling delay")
			}
		}

	default:
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("action",
			fmt.Sprintf("%q is not something that can be done to a notification", *action.Action)))
	}

	return nil
}
