package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_microsoft_proxy"
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

type fakeComplianceProxy struct {
	ConditionalAccessMicrosoftProxy
	messageID            string
	getMessageStatusCall bool
}

func (f *fakeComplianceProxy) SetComplianceStatus(
	context.Context, string, string, string, string, bool, string, string, string, bool, time.Time,
) (*conditional_access_microsoft_proxy.SetComplianceStatusResponse, error) {
	return &conditional_access_microsoft_proxy.SetComplianceStatusResponse{MessageID: f.messageID}, nil
}

func (f *fakeComplianceProxy) GetMessageStatus(
	context.Context, string, string, string,
) (*conditional_access_microsoft_proxy.GetMessageStatusResponse, error) {
	f.getMessageStatusCall = true
	return &conditional_access_microsoft_proxy.GetMessageStatusResponse{Status: conditional_access_microsoft_proxy.MessageStatusCompleted}, nil
}

func TestSetHostConditionalAccessEmptyMessageID(t *testing.T) {
	newSvc := func(proxy *fakeComplianceProxy, acts *[]fleet.ActivityTypeFailedAutomationConditionalAccess) (*Service, *mock.Store) {
		ds := new(mock.Store)
		ds.ConditionalAccessMicrosoftGetFunc = func(context.Context) (*fleet.ConditionalAccessMicrosoftIntegration, error) {
			return &fleet.ConditionalAccessMicrosoftIntegration{TenantID: "tid", ProxyServerSecret: "s", SetupDone: true}, nil
		}
		ds.SetHostConditionalAccessStatusFunc = func(context.Context, uint, bool, bool) error { return nil }
		return &Service{
			ds:                              ds,
			logger:                          slog.New(slog.DiscardHandler),
			conditionalAccessMicrosoftProxy: proxy,
			activitySvc: &mock.MockActivityService{NewActivityFunc: func(_ context.Context, _ *activity_api.User, a fleet.ActivityDetails) error {
				if act, ok := a.(fleet.ActivityTypeFailedAutomationConditionalAccess); ok {
					*acts = append(*acts, act)
				}
				return nil
			}},
		}, ds
	}
	status := &fleet.HostConditionalAccessStatus{DeviceID: "dev", UserPrincipalName: "upn"}

	t.Run("macOS fails without polling", func(t *testing.T) {
		var acts []fleet.ActivityTypeFailedAutomationConditionalAccess
		proxy := &fakeComplianceProxy{}
		svc, ds := newSvc(proxy, &acts)

		err := svc.setHostConditionalAccess(7, "darwin", status, true, false, []uint{30})
		require.ErrorContains(t, err, "missing message ID")
		require.False(t, proxy.getMessageStatusCall)
		require.False(t, ds.SetHostConditionalAccessStatusFuncInvoked)
		require.Len(t, acts, 1)
		require.Equal(t, uint(30), acts[0].PolicyID)
		require.Equal(t, []uint{7}, acts[0].HostIDList)
	})

	t.Run("Windows doesn't use the message ID", func(t *testing.T) {
		var acts []fleet.ActivityTypeFailedAutomationConditionalAccess
		svc, ds := newSvc(&fakeComplianceProxy{}, &acts)

		require.NoError(t, svc.setHostConditionalAccess(7, "windows", status, true, true, []uint{30}))
		require.True(t, ds.SetHostConditionalAccessStatusFuncInvoked)
		require.Empty(t, acts)
	})
}
