package fleet

import (
	"context"
	"fmt"
	"time"
	_ "time/tzdata" // embed timezone information in the program

	"github.com/fleetdm/fleet/v4/server"
)

const (
	CalendarBodyStaticHeader   = "reserved this time to make some changes to your work computer"
	CalendarDefaultDescription = "needs to make sure your device meets the organization's requirements."
	CalendarDefaultResolution  = "During this maintenance window, you can expect updates to be applied automatically. Your device may be unavailable during this time."
)

type DayEndedError struct {
	Msg string
}

func (e DayEndedError) Error() string {
	return e.Msg
}

type UserCalendar interface {
	// Configure configures the connection to a user's calendar. Once configured,
	// CreateEvent, GetAndUpdateEvent and DeleteEvent reference the user's calendar.
	Configure(userEmail string) error
	// CreateEvent creates a new event on the calendar on the given date. DayEndedError is returned if there is no time left on the given date to schedule event.
	CreateEvent(dateOfEvent time.Time, genBodyFn func(conflict bool) (body string, ok bool, err error)) (event *CalendarEvent, err error)
	// GetAndUpdateEvent retrieves the event from the calendar.
	// If the event has been modified, it returns the updated event.
	// If the event has been deleted, it schedules a new event with given body callback and returns the new event.
	GetAndUpdateEvent(event *CalendarEvent, genBodyFn func(conflict bool) (body string, ok bool, err error)) (updatedEvent *CalendarEvent,
		updated bool, err error)
	// DeleteEvent deletes the event with the given ID.
	DeleteEvent(event *CalendarEvent) error
	// StopEventChannel stops the event's callback channel.
	StopEventChannel(event *CalendarEvent) error
	// Get retrieves the value of the given key from the event.
	Get(event *CalendarEvent, key string) (interface{}, error)
}

// Lock interface for managing distributed locks.
type Lock interface {
	AcquireLock(ctx context.Context, name string, value string, expireMs uint64) (result string, err error)
	ReleaseLock(ctx context.Context, name string, value string) (ok bool, err error)
	// Increment increments a counter stored in Redis. Creates a counter if it doesn't exist. Returns the new value of the counter.
	Increment(ctx context.Context, name string, expireMs uint64) (int64, error)
	Get(ctx context.Context, name string) (*string, error)
}

type CalendarWebhookPayload struct {
	Timestamp        time.Time            `json:"timestamp"`
	HostID           uint                 `json:"host_id"`
	HostDisplayName  string               `json:"host_display_name"`
	HostSerialNumber string               `json:"host_serial_number"`
	FailingPolicies  []PolicyCalendarData `json:"failing_policies,omitempty"`
	Error            string               `json:"error,omitempty"`
}

func FireCalendarWebhook(
	webhookURL string,
	hostID uint,
	hostHardwareSerial string,
	hostDisplayName string,
	failingCalendarPolicies []PolicyCalendarData,
	err string,
) error {
	if err := server.PostJSONWithTimeout(context.Background(), webhookURL, &CalendarWebhookPayload{
		Timestamp:        time.Now(),
		HostID:           hostID,
		HostDisplayName:  hostDisplayName,
		HostSerialNumber: hostHardwareSerial,
		FailingPolicies:  failingCalendarPolicies,
		Error:            err,
	}); err != nil {
		return fmt.Errorf("POST to %q: %w", server.MaskSecretURLParams(webhookURL), server.MaskURLError(err))
	}
	return nil
}
