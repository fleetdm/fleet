package service

import (
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/notifications/api"
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
		{2, api.EndUserNotificationReasonBadInvocation, false, false},
		{20, api.EndUserNotificationReasonBadConfiguration, false, false},
		{30, api.EndUserNotificationReasonPageLoadFailed, true, false},
		{31, api.EndUserNotificationReasonHTTPError, true, false},
		{40, api.EndUserNotificationReasonNoGUIUser, true, false},
		{41, api.EndUserNotificationReasonScreenLocked, true, false},
		{42, api.EndUserNotificationReasonNoDisplay, true, false},
		{70, api.EndUserNotificationReasonInternalError, true, false},
		{100, api.EndUserNotificationReasonDesktopMissing, true, true},
		{101, api.EndUserNotificationReasonDesktopTooOld, true, true},
		{9999, api.EndUserNotificationReasonUnexpectedFailure, true, false},
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
				assert.Equal(t, api.EndUserNotificationLongRetryInterval, *retryIn)
			} else {
				assert.Equal(t, api.EndUserNotificationShortRetryInterval, *retryIn)
			}
		})
	}
}
