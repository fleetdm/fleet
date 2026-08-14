package service

import (
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationOutcomeForExitCode(t *testing.T) {
	cases := []struct {
		exitCode  int64
		reason    string
		wantRetry bool
		longRetry bool
	}{
		{0, "", false, false},
		{2, fleet.EndUserNotificationReasonBadInvocation, false, false},
		{20, fleet.EndUserNotificationReasonBadConfiguration, false, false},
		{30, fleet.EndUserNotificationReasonPageLoadFailed, true, false},
		{31, fleet.EndUserNotificationReasonHTTPError, true, false},
		{40, fleet.EndUserNotificationReasonNoGUIUser, true, false},
		{41, fleet.EndUserNotificationReasonScreenLocked, true, false},
		{42, fleet.EndUserNotificationReasonNoDisplay, true, false},
		{70, fleet.EndUserNotificationReasonInternalError, true, false},
		{100, fleet.EndUserNotificationReasonDesktopMissing, true, true},
		{101, fleet.EndUserNotificationReasonDesktopTooOld, true, true},
		{9999, fleet.EndUserNotificationReasonUnexpectedFailure, true, false},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("exit_%d", c.exitCode), func(t *testing.T) {
			reason, retryIn := notificationOutcomeForExitCode(c.exitCode)
			assert.Equal(t, c.reason, reason)

			if !c.wantRetry {
				assert.Nil(t, retryIn)
				return
			}
			require.NotNil(t, retryIn)
			if c.longRetry {
				assert.Equal(t, fleet.EndUserNotificationLongRetryInterval, *retryIn)
			} else {
				assert.Equal(t, fleet.EndUserNotificationShortRetryInterval, *retryIn)
			}
		})
	}
}
