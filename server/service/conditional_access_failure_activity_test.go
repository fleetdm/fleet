package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

// proxyStatusErr mimics the conditional access proxy's StatusError, exposing
// the remote HTTP status code and response body.
type proxyStatusErr struct {
	code int
	body string
}

func (e *proxyStatusErr) Error() string   { return fmt.Sprintf("%d: %s", e.code, e.body) }
func (e *proxyStatusErr) StatusCode() int { return e.code }
func (e *proxyStatusErr) Body() string    { return e.body }

// TestRecordConditionalAccessFailureActivity verifies that a failed conditional
// access compliance push records one activity per conditional-access policy,
// capturing the remote status code and response body.
func TestRecordConditionalAccessFailureActivity(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	type recorded struct {
		acts []fleet.ActivityTypeFailedAutomationConditionalAccess
	}
	newRecorder := func(r *recorded) activity_api.NewActivityService {
		return &mock.MockActivityService{NewActivityFunc: func(_ context.Context, user *activity_api.User, activity fleet.ActivityDetails) error {
			require.Nil(t, user)
			act, ok := activity.(fleet.ActivityTypeFailedAutomationConditionalAccess)
			require.True(t, ok)
			r.acts = append(r.acts, act)
			return nil
		}}
	}

	t.Run("records one activity per policy with status and body", func(t *testing.T) {
		var r recorded
		err := &proxyStatusErr{code: 500, body: "upstream boom"}

		recordConditionalAccessFailureActivity(ctx, newRecorder(&r), 100, []uint{30, 31}, err, logger)

		require.Len(t, r.acts, 2)
		for _, act := range r.acts {
			require.Equal(t, []uint{100}, act.HostIDList)
			require.Equal(t, 500, act.StatusCode)
			require.Equal(t, "upstream boom", act.ErrorResponse)
		}
		require.Equal(t, uint(30), r.acts[0].PolicyID)
		require.Equal(t, uint(31), r.acts[1].PolicyID)
	})

	t.Run("falls back to error string when no body", func(t *testing.T) {
		var r recorded

		recordConditionalAccessFailureActivity(ctx, newRecorder(&r), 101, []uint{30}, errors.New("connection refused"), logger)

		require.Len(t, r.acts, 1)
		require.Equal(t, 0, r.acts[0].StatusCode)
		require.Equal(t, "connection refused", r.acts[0].ErrorResponse)
	})

	t.Run("no policies records nothing", func(t *testing.T) {
		var r recorded

		recordConditionalAccessFailureActivity(ctx, newRecorder(&r), 103, nil, &proxyStatusErr{code: 500}, logger)

		require.Empty(t, r.acts)
	})
}

// TestRecordSingleSignOnBlockedActivity verifies that a successful non-compliant
// compliance push records one ran_automation_conditional_access activity
// per conditional-access policy the host is failing.
func TestRecordSingleSignOnBlockedActivity(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	type recorded struct {
		acts []fleet.ActivityTypeRanAutomationConditionalAccess
	}
	newRecorder := func(r *recorded) activity_api.NewActivityService {
		return &mock.MockActivityService{NewActivityFunc: func(_ context.Context, user *activity_api.User, activity fleet.ActivityDetails) error {
			require.Nil(t, user)
			act, ok := activity.(fleet.ActivityTypeRanAutomationConditionalAccess)
			require.True(t, ok)
			r.acts = append(r.acts, act)
			return nil
		}}
	}

	t.Run("records one activity per policy", func(t *testing.T) {
		var r recorded

		recordSingleSignOnBlockedActivity(ctx, newRecorder(&r), 100, []uint{30, 31}, logger)

		require.Len(t, r.acts, 2)
		for _, act := range r.acts {
			require.Equal(t, []uint{100}, act.HostIDList)
		}
		require.Equal(t, uint(30), r.acts[0].PolicyID)
		require.Equal(t, uint(31), r.acts[1].PolicyID)
	})

	t.Run("no policies records nothing", func(t *testing.T) {
		var r recorded

		recordSingleSignOnBlockedActivity(ctx, newRecorder(&r), 103, nil, logger)

		require.Empty(t, r.acts)
	})

	t.Run("nil activity function is a no-op", func(t *testing.T) {
		require.NotPanics(t, func() {
			recordSingleSignOnBlockedActivity(ctx, nil, 104, []uint{30}, logger)
		})
	})
}
