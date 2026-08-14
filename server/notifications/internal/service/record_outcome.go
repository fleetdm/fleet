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

	// Nobody was there to see it, or the page did not load. Both pass.
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
	case 70:
		return api.EndUserNotificationReasonInternalError, &shortRetry

	// An admin has to deploy or upgrade Fleet Desktop, which will not happen in
	// the next half hour.
	case 100:
		return api.EndUserNotificationReasonDesktopMissing, &longRetry
	case 101:
		return api.EndUserNotificationReasonDesktopTooOld, &longRetry

	default:
		return api.EndUserNotificationReasonUnexpectedFailure, &shortRetry
	}
}

// RecordOutcome records the outcome of a script result that belongs to a
// notification, and hands it to the kind. executionID that doesn't belong to
// any notification is the overwhelming majority of script results, and is a
// no-op rather than an error.
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
