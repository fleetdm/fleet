package fleet

import "time"

type DayEndedError struct {
	Msg string
}

func (e DayEndedError) Error() string {
	return e.Msg
}

type Calendar interface {
	// Connect to calendar. This method must be called first. Currently, config must be a *GoogleCalendarConfig
	Connect(config any) (Calendar, error)
	// GetAndUpdateEvent retrieves the event with the given ID. If the event has been deleted, it schedules a new event and returns the new event.
	GetAndUpdateEvent(event *CalendarEvent, genBodyFn func() string) (updatedEvent *CalendarEvent, updated bool, err error)
	// CreateEvent creates a new event on the calendar on the given date. DayEndedError is returned if there is no time left on the given date to schedule event.
	CreateEvent(dateOfEvent time.Time, body string) (event *CalendarEvent, err error)
	// DeleteEvent deletes the event with the given ID.
	DeleteEvent(event *CalendarEvent) error
}
