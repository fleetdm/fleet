package apple_mdm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/nanopush"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestAPNSReason(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "bare JSONPushError",
			err:  &nanopush.JSONPushError{Reason: APNSReasonUnregistered, Timestamp: 1725000000000},
			want: APNSReasonUnregistered,
		},
		{
			name: "wrapped like the provider wraps it",
			err:  fmt.Errorf("push HTTP status: 410: %w", &nanopush.JSONPushError{Reason: APNSReasonUnregistered}),
			want: APNSReasonUnregistered,
		},
		{
			name: "double-wrapped",
			err:  fmt.Errorf("outer: %w", fmt.Errorf("push HTTP status: 400: %w", &nanopush.JSONPushError{Reason: APNSReasonBadDeviceToken})),
			want: APNSReasonBadDeviceToken,
		},
		{
			name: "transport error",
			err:  errors.New("dial tcp: connection refused"),
			want: "",
		},
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, APNSReason(c.err))
		})
	}
}

func TestTurnOffMDMIfAPNSFailed(t *testing.T) {
	ctx := t.Context()
	noActivity := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error { return nil }

	apnsErr := func(err error) *APNSDeliveryError {
		return &APNSDeliveryError{errorsByUUID: map[string]error{"host-uuid-1": err}}
	}

	cases := []struct {
		name            string
		err             error
		wantHandled     bool
		wantTurnOffCall bool
	}{
		{
			name:            "Unregistered turns off MDM",
			err:             apnsErr(fmt.Errorf("push HTTP status: 410: %w", &nanopush.JSONPushError{Reason: APNSReasonUnregistered, Timestamp: 1725000000000})),
			wantHandled:     true,
			wantTurnOffCall: true,
		},
		{
			name:            "BadDeviceToken does not turn off MDM",
			err:             apnsErr(fmt.Errorf("push HTTP status: 400: %w", &nanopush.JSONPushError{Reason: APNSReasonBadDeviceToken})),
			wantHandled:     true,
			wantTurnOffCall: false,
		},
		{
			name:            "TooManyRequests does not turn off MDM",
			err:             apnsErr(fmt.Errorf("push HTTP status: 429: %w", &nanopush.JSONPushError{Reason: APNSReasonTooManyRequests})),
			wantHandled:     true,
			wantTurnOffCall: false,
		},
		{
			name:            "5xx does not turn off MDM",
			err:             apnsErr(fmt.Errorf("push HTTP status: 503: %w", &nanopush.JSONPushError{Reason: "InternalServerError"})),
			wantHandled:     true,
			wantTurnOffCall: false,
		},
		{
			name:            "transport error does not turn off MDM",
			err:             apnsErr(errors.New("dial tcp: connection refused")),
			wantHandled:     true,
			wantTurnOffCall: false,
		},
		{
			name:            "non-APNs error is not handled",
			err:             errors.New("some datastore error"),
			wantHandled:     false,
			wantTurnOffCall: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.MDMTurnOffFunc = func(ctx context.Context, uuid string) ([]*fleet.User, []fleet.ActivityDetails, error) {
				return nil, nil, nil
			}

			handled, err := turnOffMDMIfAPNSFailed(ctx, ds, c.err, slog.New(slog.DiscardHandler), noActivity)
			require.NoError(t, err)
			require.Equal(t, c.wantHandled, handled)
			require.Equal(t, c.wantTurnOffCall, ds.MDMTurnOffFuncInvoked)
		})
	}
}
