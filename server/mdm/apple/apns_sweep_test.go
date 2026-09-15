package apple_mdm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/nanopush"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

type fakeAPNsNotifier struct {
	calls [][]string
	err   error
}

func (f *fakeAPNsNotifier) SendNotifications(_ context.Context, ids []string) error {
	f.calls = append(f.calls, ids)
	return f.err
}

// sweepTestDS returns a mock datastore with MDM enabled and the sweep
// methods stubbed to a one-page pass; individual tests override pieces.
func sweepTestDS() *mock.Store {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
	}
	ds.GetMDMAppleAPNsSweepStateFunc = func(ctx context.Context) (*fleet.MDMAppleAPNsSweepState, error) {
		return nil, nil
	}
	ds.CountEnabledNanoEnrollmentsFunc = func(ctx context.Context) (int, error) {
		return 100, nil
	}
	ds.ListNanoEnrollmentIDsForAPNsSweepFunc = func(ctx context.Context, afterID string, batchSize int, silentFor time.Duration) ([]string, string, bool, error) {
		return nil, "", false, nil
	}
	ds.SetMDMAppleAPNsSweepStateFunc = func(ctx context.Context, state *fleet.MDMAppleAPNsSweepState) error {
		return nil
	}
	return ds
}

func TestSweepAPNsPushes(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)

	t.Run("no-op when MDM not enabled and configured", func(t *testing.T) {
		ds := sweepTestDS()
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		notifier := &fakeAPNsNotifier{}
		require.NoError(t, sweepAPNsPushes(ctx, ds, notifier, logger, time.Minute))
		require.Empty(t, notifier.calls)
		require.False(t, ds.GetMDMAppleAPNsSweepStateFuncInvoked)
	})

	t.Run("pass start computes clamped batch size", func(t *testing.T) {
		cases := []struct {
			name      string
			count     int
			interval  time.Duration
			wantBatch int
			wantWarn  bool
		}{
			{"scales to lap in a day", 100_000, time.Minute, 70, false},
			{"floored at one enrollment per tick", 10, time.Minute, 1, false},
			{"clamped down to the maximum with a warning", 5_000_000, time.Minute, 2000, true},
			{"slower ticks mean bigger pages", 40_000, time.Hour, 1667, false},
			{"intervals of a day or more floor at one tick per lap", 100, 25 * time.Hour, 100, false},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				ds := sweepTestDS()
				ds.CountEnabledNanoEnrollmentsFunc = func(ctx context.Context) (int, error) {
					return c.count, nil
				}
				var gotBatch int
				ds.ListNanoEnrollmentIDsForAPNsSweepFunc = func(ctx context.Context, afterID string, batchSize int, silentFor time.Duration) ([]string, string, bool, error) {
					gotBatch = batchSize
					require.Empty(t, afterID)
					require.Equal(t, 24*time.Hour, silentFor)
					return nil, "", false, nil
				}
				var buf strings.Builder
				warnLogger := slog.New(slog.NewTextHandler(&buf, nil))

				require.NoError(t, sweepAPNsPushes(ctx, ds, &fakeAPNsNotifier{}, warnLogger, c.interval))
				require.Equal(t, c.wantBatch, gotBatch)
				require.Equal(t, c.wantWarn, strings.Contains(buf.String(), "level=WARN"), buf.String())
			})
		}
	})

	t.Run("zero enrollments ends the tick without a walk", func(t *testing.T) {
		ds := sweepTestDS()
		ds.CountEnabledNanoEnrollmentsFunc = func(ctx context.Context) (int, error) {
			return 0, nil
		}
		notifier := &fakeAPNsNotifier{}
		require.NoError(t, sweepAPNsPushes(ctx, ds, notifier, logger, time.Minute))
		require.False(t, ds.ListNanoEnrollmentIDsForAPNsSweepFuncInvoked)
		require.Empty(t, notifier.calls)
	})

	t.Run("poisoned batch size self-heals to a fresh pass", func(t *testing.T) {
		ds := sweepTestDS()
		ds.GetMDMAppleAPNsSweepStateFunc = func(ctx context.Context) (*fleet.MDMAppleAPNsSweepState, error) {
			return &fleet.MDMAppleAPNsSweepState{Cursor: "enrollment-42", BatchSize: -5}, nil
		}
		var gotAfterID string
		var gotBatch int
		ds.ListNanoEnrollmentIDsForAPNsSweepFunc = func(ctx context.Context, afterID string, batchSize int, silentFor time.Duration) ([]string, string, bool, error) {
			gotAfterID = afterID
			gotBatch = batchSize
			return nil, "", false, nil
		}
		require.NoError(t, sweepAPNsPushes(ctx, ds, &fakeAPNsNotifier{}, logger, time.Minute))
		require.True(t, ds.CountEnabledNanoEnrollmentsFuncInvoked, "fresh pass recomputes the batch")
		require.Empty(t, gotAfterID)
		require.Equal(t, 1, gotBatch)
	})

	t.Run("mid-pass state is reused without recounting", func(t *testing.T) {
		ds := sweepTestDS()
		ds.GetMDMAppleAPNsSweepStateFunc = func(ctx context.Context) (*fleet.MDMAppleAPNsSweepState, error) {
			return &fleet.MDMAppleAPNsSweepState{Cursor: "enrollment-42", BatchSize: 7}, nil
		}
		ds.ListNanoEnrollmentIDsForAPNsSweepFunc = func(ctx context.Context, afterID string, batchSize int, silentFor time.Duration) ([]string, string, bool, error) {
			require.Equal(t, "enrollment-42", afterID)
			require.Equal(t, 7, batchSize)
			return nil, "enrollment-49", true, nil
		}
		require.NoError(t, sweepAPNsPushes(ctx, ds, &fakeAPNsNotifier{}, logger, time.Minute))
		require.False(t, ds.CountEnabledNanoEnrollmentsFuncInvoked)
	})

	t.Run("full page pushes eligible and advances the cursor", func(t *testing.T) {
		ds := sweepTestDS()
		ds.ListNanoEnrollmentIDsForAPNsSweepFunc = func(ctx context.Context, afterID string, batchSize int, silentFor time.Duration) ([]string, string, bool, error) {
			return []string{"e1", "e2"}, "e9", true, nil
		}
		var gotState *fleet.MDMAppleAPNsSweepState
		ds.SetMDMAppleAPNsSweepStateFunc = func(ctx context.Context, state *fleet.MDMAppleAPNsSweepState) error {
			gotState = state
			return nil
		}
		notifier := &fakeAPNsNotifier{}
		require.NoError(t, sweepAPNsPushes(ctx, ds, notifier, logger, time.Minute))
		require.Equal(t, [][]string{{"e1", "e2"}}, notifier.calls)
		require.NotNil(t, gotState)
		require.Equal(t, "e9", gotState.Cursor)
		require.Equal(t, 1, gotState.BatchSize, "batch size computed at pass start rides along")
	})

	t.Run("short page resets the state", func(t *testing.T) {
		ds := sweepTestDS()
		ds.ListNanoEnrollmentIDsForAPNsSweepFunc = func(ctx context.Context, afterID string, batchSize int, silentFor time.Duration) ([]string, string, bool, error) {
			return []string{"e1"}, "e1", false, nil
		}
		var setInvoked bool
		var gotState *fleet.MDMAppleAPNsSweepState
		ds.SetMDMAppleAPNsSweepStateFunc = func(ctx context.Context, state *fleet.MDMAppleAPNsSweepState) error {
			setInvoked = true
			gotState = state
			return nil
		}
		require.NoError(t, sweepAPNsPushes(ctx, ds, &fakeAPNsNotifier{}, logger, time.Minute))
		require.True(t, setInvoked)
		require.Nil(t, gotState)
	})

	t.Run("per-enrollment rejections are logged and never turn off MDM", func(t *testing.T) {
		ds := sweepTestDS()
		ds.ListNanoEnrollmentIDsForAPNsSweepFunc = func(ctx context.Context, afterID string, batchSize int, silentFor time.Duration) ([]string, string, bool, error) {
			return []string{"e1", "e2"}, "e9", true, nil
		}
		var setInvoked bool
		ds.SetMDMAppleAPNsSweepStateFunc = func(ctx context.Context, state *fleet.MDMAppleAPNsSweepState) error {
			setInvoked = true
			return nil
		}
		notifier := &fakeAPNsNotifier{err: &APNSDeliveryError{errorsByUUID: map[string]error{
			"e1": fmt.Errorf("push HTTP status: 410: %w", &nanopush.JSONPushError{Reason: APNSReasonUnregistered, Timestamp: 1}),
		}}}
		var buf strings.Builder
		infoLogger := slog.New(slog.NewTextHandler(&buf, nil))

		require.NoError(t, sweepAPNsPushes(ctx, ds, notifier, infoLogger, time.Minute))
		require.True(t, setInvoked, "rejections must not stall the lap")
		require.False(t, ds.MDMTurnOffFuncInvoked)
		require.Contains(t, buf.String(), APNSReasonUnregistered)
		require.Contains(t, buf.String(), "e1")
	})

	t.Run("provider-level failure leaves the cursor unadvanced", func(t *testing.T) {
		ds := sweepTestDS()
		ds.ListNanoEnrollmentIDsForAPNsSweepFunc = func(ctx context.Context, afterID string, batchSize int, silentFor time.Duration) ([]string, string, bool, error) {
			return []string{"e1"}, "e9", true, nil
		}
		notifier := &fakeAPNsNotifier{err: errors.New("dial tcp: connection refused")}
		err := sweepAPNsPushes(ctx, ds, notifier, logger, time.Minute)
		require.Error(t, err)
		require.False(t, ds.SetMDMAppleAPNsSweepStateFuncInvoked)
	})
}
