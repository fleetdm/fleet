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
	CalendarEventConflictText  = "because there was no remaining availability "
	CalendarDefaultDescription = "needs to make sure your device meets the organization's requirements."
	CalendarDefaultResolution  = "During this maintenance window, you can expect updates to be applied automatically. Your device may be unavailable during this time."
)

type DayEndedError struct {
	Msg string
}

func (e DayEndedError) Error() string {
	return e.Msg
}

type CalendarGenBodyFn func(conflict bool) (body string, ok bool, err error)

type UserCalendar interface {
	// Configure configures the connection to a user's calendar. Once configured,
	// CreateEvent, GetAndUpdateEvent and DeleteEvent reference the user's calendar.
	Configure(userEmail string) error
	// CreateEvent creates a new event on the calendar on the given date. DayEndedError is returned if there is no time left on the given date to schedule event.
	CreateEvent(
		dateOfEvent time.Time,
		genBodyFn CalendarGenBodyFn,
		opts CalendarCreateEventOpts,
	) (event *CalendarEvent, err error)
	// GetAndUpdateEvent retrieves the event from the calendar.
	// If the event has been modified, it returns the updated event.
	// If the event has been deleted, it schedules a new event with given body callback and returns the new event.
	GetAndUpdateEvent(event *CalendarEvent, genBodyFn CalendarGenBodyFn,
		opts CalendarGetAndUpdateEventOpts) (updatedEvent *CalendarEvent,
		updated bool, err error)
	// UpdateEventBody updates the body of the calendar event and returns new ETag
	UpdateEventBody(event *CalendarEvent, genBodyFn CalendarGenBodyFn) (string, error)
	// DeleteEvent deletes the event with the given ID.
	DeleteEvent(event *CalendarEvent) error
	// StopEventChannel stops the event's callback channel.
	StopEventChannel(event *CalendarEvent) error
	// Get retrieves the value of the given key from the event.
	Get(event *CalendarEvent, key string) (interface{}, error)
}

// Lock interface for managing distributed locks.
type Lock interface {
	// SetIfNotExist attempts to set an item with the given key. value is the value to set for the key, which is used to release the lock.
	// expireMs is the time in milliseconds after which the lock is automatically released. expireMs=0 means a default expiration time is used.
	// Returns true if the lock was acquired, false otherwise.
	SetIfNotExist(ctx context.Context, key string, value string, expireMs uint64) (ok bool, err error)
	// ReleaseLock attempts to release a lock with the given key and value. If key does not exist or value does not match, the lock is not released.
	// Returns true if the lock was released, false otherwise.
	ReleaseLock(ctx context.Context, key string, value string) (ok bool, err error)
	// Get retrieves the value of the given key. If the key does not exist, nil is returned.
	Get(ctx context.Context, key string) (*string, error)
	// GetAndDelete retrieves the value of the given key and deletes the key. If the key does not exist, nil is returned.
	GetAndDelete(ctx context.Context, key string) (*string, error)
	// AddToSet adds the value to the set identified by the given key.
	AddToSet(ctx context.Context, key string, value string) error
	// RemoveFromSet removes the value from the set identified by the given key.
	RemoveFromSet(ctx context.Context, key string, value string) error
	// GetSet retrieves a slice of string values from the set identified by the given key.
	GetSet(ctx context.Context, key string) ([]string, error)
}

type CalendarCreateEventOpts struct {
	EventUUID  string
	ChannelID  string
	ResourceID string
}

type CalendarGetAndUpdateEventOpts struct {
	UpdateTimezone bool
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
