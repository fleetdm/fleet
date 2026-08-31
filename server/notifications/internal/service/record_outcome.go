package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/notifications/api"
	platform_errors "github.com/fleetdm/fleet/v4/server/platform/errors"
)

// notificationOutcomeForExitCode maps a notify exit code to a reason and a retry
// wait, nil meaning no retry. Codes come from
// apps/fleet-desktop-macos/FleetDesktop/cli.swift.
func notificationOutcomeForExitCode(exitCode int64) (reason string, retryIn *time.Duration) {
	shortRetry := api.EndUserNotificationShortRetryInterval
	longRetry := api.EndUserNotificationLongRetryInterval

	switch exitCode {
	case 0:
		return "", nil

	// Fleet asked for something impossible, so asking again changes nothing.
	case 2:
		return api.EndUserNotificationReasonBadInvocation, nil
	case 20:
		return api.EndUserNotificationReasonBadConfiguration, nil

	// Fleet Desktop couldn't display the notification.
	case 30:
		return api.EndUserNotificationReasonPageLoadFailed, &shortRetry
	case 31:
		return api.EndUserNotificationReasonHTTPError, &shortRetry
	case 40:
		return api.EndUserNotificationReasonNoGUIUser, &shortRetry
	case 41:
		return api.EndUserNotificationReasonScreenLocked, &shortRetry
	case 42:
		return api.EndUserNotificationReasonNoDisplay, &shortRetry
	// Fleet Desktop found a notification already on screen and would not put a
	// second one over it. Fleet's dispatcher tries not to let this happen, but
	// only the binary can see what is really there. It needs a Fleet Desktop
	// release to reach Fleet: cli.swift's ExitCode has no case for it yet.
	case 50:
		return api.EndUserNotificationReasonAnotherDisplayed, &shortRetry
	case 70:
		return api.EndUserNotificationReasonInternalError, &shortRetry

	// fleetd has its own scripts-disabled setting, which Fleet's bypass doesn't
	// reach, and Fleet couldn't build the notification's URL. Neither clears up on
	// its own. Values mirror server/fleet, which this context can't import.
	case -2:
		return api.EndUserNotificationReasonScriptsDisabled, &longRetry
	case -5:
		return api.EndUserNotificationReasonURLUnresolved, &longRetry

	// Fleet Desktop can't display a notification. Wait the long duration so an
	// admin or a patch policy has a chance to install a newer version.
	case 100:
		return api.EndUserNotificationReasonDesktopMissing, &longRetry
	case 101:
		return api.EndUserNotificationReasonDesktopTooOld, &longRetry

	default:
		return api.EndUserNotificationReasonUnexpectedFailure, &shortRetry
	}
}

func (s *Service) RecordOutcome(ctx context.Context, executionID string, exitCode int64, output string) error {
	if executionID == "" {
		return nil
	}

	notification, err := s.ds.GetEndUserNotificationByExecutionID(ctx, executionID)
	if err != nil {
		if platform_errors.IsNotFound(err) {
			return nil
		}
		return ctxerr.Wrap(ctx, err, "get end user notification for script result")
	}

	reason, retryIn := notificationOutcomeForExitCode(exitCode)
	outcome := api.NotificationOutcome{
		Displayed:   exitCode == 0,
		ExitCode:    exitCode,
		Reason:      reason,
		Output:      output,
		ExecutionID: executionID,
	}

	// Nothing caps how many attempts a notification gets. notification.AttemptCount
	// is recorded and would be read here, but expires_at is what stops the retries
	// today.
	var nextAttemptAt *time.Time
	if retryIn != nil {
		nextAttempt := time.Now().UTC().Add(*retryIn)
		nextAttemptAt = &nextAttempt
	}

	if err := s.ds.SetEndUserNotificationOutcome(ctx, notification.UUID, outcome, nextAttemptAt); err != nil {
		return ctxerr.Wrap(ctx, err, "set end user notification outcome")
	}

	kind, kindRegistered := s.kinds[notification.Kind]
	if !kindRegistered {
		s.logger.WarnContext(ctx, "no kind registered for end user notification",
			"kind", notification.Kind, "notification_uuid", notification.UUID)
		return nil
	}
	if err := kind.OnOutcome(ctx, notification, outcome); err != nil {
		return ctxerr.Wrap(ctx, err, "notification kind handling outcome")
	}
	return nil
}
