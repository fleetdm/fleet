package fleet

import "time"

type DayEndedError struct {
	Msg string
}

func (e DayEndedError) Error() string {
	return e.Msg
}

type UserCalendar interface {
	// Configure calendar. This method must be called first. Currently, config must be a *GoogleCalendarConfig
	Configure(config any) (UserCalendar, error)
	// GetAndUpdateEvent retrieves the event from the calendar.
	// If the event has been modified, it returns the updated event.
	// If the event has been deleted, it schedules a new event with given body callback and returns the new event.
	GetAndUpdateEvent(event *CalendarEvent, genBodyFn func() string) (updatedEvent *CalendarEvent, updated bool, err error)
	// CreateEvent creates a new event on the calendar on the given date. DayEndedError is returned if there is no time left on the given date to schedule event.
	CreateEvent(dateOfEvent time.Time, body string) (event *CalendarEvent, err error)
	// DeleteEvent deletes the event with the given ID.
	DeleteEvent(event *CalendarEvent) error
}
