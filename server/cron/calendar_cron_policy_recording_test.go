package cron

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/calendar"
	"github.com/fleetdm/fleet/v4/server/test/automationtest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// stubLock is an inline fleet.Lock implementation: every method is a no-op
// success unless overridden. The defaults are tuned for the calendar cron's
// existing-event path so a single SetIfNotExist returns lockAcquired=true.
type stubLock struct {
	setIfNotExist func(ctx context.Context, key, value string, expireMs uint64) (bool, error)
	removeFromSet func(ctx context.Context, key, value string) error
}

func (s stubLock) SetIfNotExist(ctx context.Context, key, value string, expireMs uint64) (bool, error) {
	if s.setIfNotExist != nil {
		return s.setIfNotExist(ctx, key, value, expireMs)
	}
	return true, nil
}

func (stubLock) ReleaseLock(ctx context.Context, key, value string) (bool, error) {
	return true, nil
}
func (stubLock) Get(ctx context.Context, key string) (*string, error)          { return nil, nil }
func (stubLock) GetAndDelete(ctx context.Context, key string) (*string, error) { return nil, nil }
func (stubLock) AddToSet(ctx context.Context, key, value string) error         { return nil }
func (s stubLock) RemoveFromSet(ctx context.Context, key, value string) error {
	if s.removeFromSet != nil {
		return s.removeFromSet(ctx, key, value)
	}
	return nil
}
func (stubLock) GetSet(ctx context.Context, key string) ([]string, error) { return nil, nil }

// stubUserCalendar is an inline fleet.UserCalendar implementation. Only the
// methods called by processFailingHostExistingCalendarEvent are configurable;
// the rest panic if invoked so accidental coverage gaps surface loudly.
type stubUserCalendar struct {
	createEvent       func(dateOfEvent time.Time, genBodyFn fleet.CalendarGenBodyFn, opts fleet.CalendarCreateEventOpts) (*fleet.CalendarEvent, error)
	updateEventBody   func(event *fleet.CalendarEvent, genBodyFn fleet.CalendarGenBodyFn) (string, error)
	getAndUpdateEvent func(event *fleet.CalendarEvent, genBodyFn fleet.CalendarGenBodyFn, opts fleet.CalendarGetAndUpdateEventOpts) (*fleet.CalendarEvent, bool, error)
	stopEventChannel  func(event *fleet.CalendarEvent) error
}

func (stubUserCalendar) Configure(userEmail string) error { panic("unexpected Configure call") }
func (s stubUserCalendar) CreateEvent(dateOfEvent time.Time, genBodyFn fleet.CalendarGenBodyFn, opts fleet.CalendarCreateEventOpts) (*fleet.CalendarEvent, error) {
	if s.createEvent == nil {
		panic("unexpected CreateEvent call")
	}
	return s.createEvent(dateOfEvent, genBodyFn, opts)
}

func (s stubUserCalendar) GetAndUpdateEvent(event *fleet.CalendarEvent, genBodyFn fleet.CalendarGenBodyFn, opts fleet.CalendarGetAndUpdateEventOpts) (*fleet.CalendarEvent, bool, error) {
	if s.getAndUpdateEvent == nil {
		panic("unexpected GetAndUpdateEvent call")
	}
	return s.getAndUpdateEvent(event, genBodyFn, opts)
}

func (s stubUserCalendar) UpdateEventBody(event *fleet.CalendarEvent, genBodyFn fleet.CalendarGenBodyFn) (string, error) {
	if s.updateEventBody == nil {
		panic("unexpected UpdateEventBody call")
	}
	return s.updateEventBody(event, genBodyFn)
}
func (stubUserCalendar) DeleteEvent(*fleet.CalendarEvent) error { panic("unexpected DeleteEvent call") }
func (s stubUserCalendar) StopEventChannel(event *fleet.CalendarEvent) error {
	if s.stopEventChannel != nil {
		return s.stopEventChannel(event)
	}
	return nil
}

func (stubUserCalendar) Get(*fleet.CalendarEvent, string) (any, error) {
	panic("unexpected Get call")
}

// recordingCalendarHost is the HostPolicyMembershipData used by the calendar
// recording-lifecycle tests. A single failing policy keeps the path through
// getBodyTag short.
func recordingCalendarHost() fleet.HostPolicyMembershipData {
	return fleet.HostPolicyMembershipData{
		Email:            "user@example.com",
		HostID:           42,
		FailingPolicyIDs: "1",
	}
}

// recordingPolicyMap returns a sync.Map pre-populated with policy ID 1 mapped
// to a PolicyLiteWithMeta carrying the given tag, so getBodyTag doesn't fall
// through to ds.PolicyLite.
func recordingPolicyMap(tag string) *sync.Map {
	m := &sync.Map{}
	m.Store(uint(1), &calendar.PolicyLiteWithMeta{Tag: tag})
	return m
}

// captureFinalize installs an UpdatePolicyAutomationExecutionsStatusByBatch
// stub that captures the outcomeErr from the most recent call. Returns a
// pointer the caller can dereference after the system-under-test has run:
// `*finalizeErr == nil` means Success was finalized, non-nil means Failure
// (with the message in `(*finalizeErr).Error()`). Typical usage:
//
//	automationtest.StubNoopRecording(ds)
//	finalizeErr := captureFinalize(ds) // overrides the Update stub above
func captureFinalize(ds *mock.Store) *error {
	var captured error
	ds.UpdatePolicyAutomationExecutionsFunc = func(_ context.Context, _ uuid.UUID, outcomeErr error) error {
		captured = outcomeErr
		return nil
	}
	return &captured
}

// TestProcessFailingHostExistingCalendarEventRecording locks in the lifecycle
// guarantees of the deferred finalize in processFailingHostExistingCalendarEvent:
//
//  1. Every path that calls ensureRecorded() must eventually transition the
//     batch out of 'pending' — including the IsNotFound short-circuit.
//  2. Error paths after ensureRecorded() finalize with Failure.
//  3. Paths that never reach ensureRecorded() never touch the recording tables.
func TestProcessFailingHostExistingCalendarEventRecording(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	// Pre-populated policy tag is "tag-B"; the test sets the calendar event's
	// body_tag to "tag-A" for a mismatch (subtests A and B) or "tag-B" for a
	// match (C).
	newCalendarEvent := func(bodyTag string) *fleet.CalendarEvent {
		ev := &fleet.CalendarEvent{
			ID:        1,
			UUID:      "event-uuid",
			Email:     "user@example.com",
			StartTime: time.Now().Add(1 * time.Hour),
			EndTime:   time.Now().Add(2 * time.Hour),
		}
		require.NoError(t, ev.SaveDataItems("body_tag", bodyTag))
		ev.UpdatedAt = time.Now()
		return ev
	}

	hostCalEvent := &fleet.HostCalendarEvent{
		ID:              1,
		HostID:          42,
		CalendarEventID: 1,
		WebhookStatus:   fleet.CalendarWebhookStatusNone,
	}

	t.Run("body-tag mismatch, UpdateEventBody ok, reload returns NotFound, deferred finalize records Success", func(t *testing.T) {
		ds := new(mock.Store)
		automationtest.StubNoopRecording(ds)
		finalizeErr := captureFinalize(ds)
		ds.GetCalendarEventFunc = func(ctx context.Context, email string) (*fleet.CalendarEvent, error) {
			return nil, notFoundErr{}
		}

		userCal := stubUserCalendar{
			updateEventBody: func(*fleet.CalendarEvent, fleet.CalendarGenBodyFn) (string, error) {
				return "new-etag", nil
			},
		}
		cfg := &calendar.Config{}
		cfg.SetAlwaysReloadEvent(true)

		err := processFailingHostExistingCalendarEvent(
			t.Context(), ds, stubLock{}, userCal, "org",
			hostCalEvent, newCalendarEvent("tag-A"), recordingCalendarHost(),
			recordingPolicyMap("tag-B"), cfg, logger,
		)
		require.NoError(t, err)
		require.True(t, ds.UpdatePolicyAutomationExecutionsFuncInvoked, "deferred finalize must run even on IsNotFound short-circuit")
		require.NoError(t, *finalizeErr, "IsNotFound is a benign exit, not a failure")
	})

	t.Run("body-tag mismatch, UpdateEventBody fails, deferred finalize records Failure", func(t *testing.T) {
		ds := new(mock.Store)
		automationtest.StubNoopRecording(ds)
		finalizeErr := captureFinalize(ds)

		userCal := stubUserCalendar{
			updateEventBody: func(*fleet.CalendarEvent, fleet.CalendarGenBodyFn) (string, error) {
				return "", errors.New("calendar API down")
			},
		}

		err := processFailingHostExistingCalendarEvent(
			t.Context(), ds, stubLock{}, userCal, "org",
			hostCalEvent, newCalendarEvent("tag-A"), recordingCalendarHost(),
			recordingPolicyMap("tag-B"), &calendar.Config{}, logger,
		)
		require.Error(t, err)
		require.True(t, ds.UpdatePolicyAutomationExecutionsFuncInvoked, "deferred finalize must run after UpdateEventBody failure")
		require.ErrorContains(t, *finalizeErr, "calendar API down")
	})

	t.Run("body-tag matches and no reload needed, recording functions are never called", func(t *testing.T) {
		ds := new(mock.Store)
		ds.CreatePolicyAutomationExecutionsFunc = func(ctx context.Context, typ fleet.PolicyAutomationType, runs []fleet.PolicyRunRef) (uuid.UUID, error) {
			t.Fatalf("RecordPolicyAutomationBatch should not be called on a no-op path")
			return uuid.Nil, nil
		}
		ds.UpdatePolicyAutomationExecutionsFunc = func(ctx context.Context, batchID uuid.UUID, outcomeErr error) error {
			t.Fatalf("UpdatePolicyAutomationExecutionsStatusByBatch should not be called on a no-op path")
			return nil
		}

		// AlwaysReloadEvent=false (default) plus a recently-updated event with
		// a future start time means shouldReloadCalendarEvent returns false too;
		// the function falls through to the "Event happening now" check, sees
		// the event is in the future, and returns nil — no userCalendar calls.
		cfg := &calendar.Config{}

		err := processFailingHostExistingCalendarEvent(
			t.Context(), ds, stubLock{}, stubUserCalendar{}, "org",
			hostCalEvent, newCalendarEvent("tag-B"), recordingCalendarHost(),
			recordingPolicyMap("tag-B"), cfg, logger,
		)
		require.NoError(t, err)
	})
}

// TestProcessFailingHostCreateCalendarEventRecording locks in the recording
// lifecycle of the *new-event* path:
//
//  1. CreateEvent fails        → Finalize ran with Failure.
//  2. CreateOrUpdateCalendarEvent fails → Finalize ran with Failure.
//  3. Both succeed             → Finalize ran with Success.
func TestProcessFailingHostCreateCalendarEventRecording(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	// Successful CreateEvent stub returns a fixed event for the SUT to write.
	createEventSuccess := func(time.Time, fleet.CalendarGenBodyFn, fleet.CalendarCreateEventOpts) (*fleet.CalendarEvent, error) {
		return &fleet.CalendarEvent{
			ID:        1,
			UUID:      "event-uuid",
			Email:     "user@example.com",
			StartTime: time.Now().Add(1 * time.Hour),
			EndTime:   time.Now().Add(2 * time.Hour),
		}, nil
	}

	t.Run("CreateEvent fails, Finalize records Failure", func(t *testing.T) {
		ds := new(mock.Store)
		automationtest.StubNoopRecording(ds)
		finalizeErr := captureFinalize(ds)

		userCal := stubUserCalendar{
			createEvent: func(time.Time, fleet.CalendarGenBodyFn, fleet.CalendarCreateEventOpts) (*fleet.CalendarEvent, error) {
				return nil, errors.New("calendar API quota exceeded")
			},
		}
		err := processFailingHostCreateCalendarEvent(t.Context(), ds, userCal, "org", recordingCalendarHost(), recordingPolicyMap("tag-X"), logger)
		require.Error(t, err)
		require.True(t, ds.UpdatePolicyAutomationExecutionsFuncInvoked)
		require.ErrorContains(t, *finalizeErr, "calendar API quota exceeded")
	})

	t.Run("CreateOrUpdateCalendarEvent fails, Finalize records Failure", func(t *testing.T) {
		ds := new(mock.Store)
		automationtest.StubNoopRecording(ds)
		finalizeErr := captureFinalize(ds)
		ds.CreateOrUpdateCalendarEventFunc = func(ctx context.Context, evUUID, email string, startTime, endTime time.Time, data []byte, timeZone *string, hostID uint, webhookStatus fleet.CalendarWebhookStatus) (*fleet.CalendarEvent, error) {
			return nil, errors.New("db write failure")
		}

		userCal := stubUserCalendar{createEvent: createEventSuccess}
		err := processFailingHostCreateCalendarEvent(t.Context(), ds, userCal, "org", recordingCalendarHost(), recordingPolicyMap("tag-X"), logger)
		require.Error(t, err)
		require.True(t, ds.UpdatePolicyAutomationExecutionsFuncInvoked)
		require.ErrorContains(t, *finalizeErr, "db write failure")
	})

	t.Run("both succeed, Finalize records Success", func(t *testing.T) {
		ds := new(mock.Store)
		automationtest.StubNoopRecording(ds)
		finalizeErr := captureFinalize(ds)
		ds.CreateOrUpdateCalendarEventFunc = func(ctx context.Context, evUUID, email string, startTime, endTime time.Time, data []byte, timeZone *string, hostID uint, webhookStatus fleet.CalendarWebhookStatus) (*fleet.CalendarEvent, error) {
			return &fleet.CalendarEvent{ID: 1, UUID: evUUID, Email: email, StartTime: startTime, EndTime: endTime}, nil
		}

		userCal := stubUserCalendar{createEvent: createEventSuccess}
		err := processFailingHostCreateCalendarEvent(t.Context(), ds, userCal, "org", recordingCalendarHost(), recordingPolicyMap("tag-X"), logger)
		require.NoError(t, err)
		require.True(t, ds.UpdatePolicyAutomationExecutionsFuncInvoked)
		require.NoError(t, *finalizeErr)
	})
}
