package calendar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"net/http"
	"time"
)

const (
	eventTitle  = "💻🚫Downtime"
	startHour   = 9
	endHour     = 17
	eventLength = 30 * time.Minute
	calendarID  = "primary"
)

var calendarScopes = []string{
	"https://www.googleapis.com/auth/calendar.events",
	"https://www.googleapis.com/auth/calendar.settings.readonly",
}

type GoogleCalendarConfig struct {
	Context           context.Context
	IntegrationConfig *fleet.GoogleCalendarIntegration
	UserEmail         string
	Logger            log.Logger
	// Should be nil for production
	API GoogleCalendarAPI
}

// GoogleCalendar is an implementation of the UserCalendar interface that uses the
// Google Calendar API to manage events.
type GoogleCalendar struct {
	config         *GoogleCalendarConfig
	timezoneOffset *int
}

type GoogleCalendarAPI interface {
	Connect(ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail string) error
	GetSetting(name string) (*calendar.Setting, error)
	ListEvents(timeMin, timeMax string) (*calendar.Events, error)
	CreateEvent(event *calendar.Event) (*calendar.Event, error)
	GetEvent(id, eTag string) (*calendar.Event, error)
	DeleteEvent(id string) error
}

type eventDetails struct {
	ID   string `json:"id"`
	ETag string `json:"etag"`
}

type GoogleCalendarLowLevelAPI struct {
	service *calendar.Service
}

// Connect creates a new Google Calendar service using the provided credentials.
func (lowLevelAPI *GoogleCalendarLowLevelAPI) Connect(
	ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail string,
) error {
	// Create a new calendar service
	conf := &jwt.Config{
		Email:      serviceAccountEmail,
		Scopes:     calendarScopes,
		PrivateKey: []byte(privateKey),
		TokenURL:   google.JWTTokenURL,
		Subject:    userToImpersonateEmail,
	}
	client := conf.Client(ctx)
	service, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}
	lowLevelAPI.service = service
	return nil
}

func (lowLevelAPI *GoogleCalendarLowLevelAPI) GetSetting(name string) (*calendar.Setting, error) {
	return lowLevelAPI.service.Settings.Get(name).Do()
}

func (lowLevelAPI *GoogleCalendarLowLevelAPI) CreateEvent(event *calendar.Event) (*calendar.Event, error) {
	return lowLevelAPI.service.Events.Insert(calendarID, event).Do()
}

func (lowLevelAPI *GoogleCalendarLowLevelAPI) GetEvent(id, eTag string) (*calendar.Event, error) {
	return lowLevelAPI.service.Events.Get(calendarID, id).IfNoneMatch(eTag).Do()
}

func (lowLevelAPI *GoogleCalendarLowLevelAPI) ListEvents(timeMin, timeMax string) (*calendar.Events, error) {
	// Default maximum number of events returned is 250, which should be sufficient for most calendars.
	return lowLevelAPI.service.Events.List(calendarID).EventTypes("default").OrderBy("startTime").SingleEvents(true).TimeMin(timeMin).TimeMax(timeMax).Do()
}

func (lowLevelAPI *GoogleCalendarLowLevelAPI) DeleteEvent(id string) error {
	return lowLevelAPI.service.Events.Delete(calendarID, id).Do()
}

func (c *GoogleCalendar) Configure(config any) (fleet.UserCalendar, error) {
	gConfig, ok := config.(*GoogleCalendarConfig)
	if !ok {
		return nil, errors.New("invalid Google calendar config")
	}
	if gConfig.API == nil {
		var lowLevelAPI GoogleCalendarAPI = &GoogleCalendarLowLevelAPI{}
		gConfig.API = lowLevelAPI
	}
	err := gConfig.API.Connect(
		gConfig.Context, gConfig.IntegrationConfig.Email, gConfig.IntegrationConfig.PrivateKey, gConfig.UserEmail,
	)
	if err != nil {
		return nil, ctxerr.Wrap(gConfig.Context, err, "creating Google calendar service")
	}

	gCal := &GoogleCalendar{
		config: gConfig,
	}

	return gCal, nil
}

func (c *GoogleCalendar) GetAndUpdateEvent(event *fleet.CalendarEvent, genBodyFn func() string) (*fleet.CalendarEvent, bool, error) {
	if c.config == nil {
		return nil, false, errors.New("the Google calendar is not connected. Please call Configure first")
	}
	if event.EndTime.Before(time.Now()) {
		return nil, false, ctxerr.Errorf(c.config.Context, "cannot get and update an event that has already ended: %s", event.EndTime)
	}
	details, err := c.unmarshalDetails(event)
	if err != nil {
		return nil, false, err
	}
	gEvent, err := c.config.API.GetEvent(details.ID, details.ETag)
	var deleted bool
	switch {
	// http.StatusNotModified is returned sometimes, but not always, so we need to check ETag explicitly later
	case googleapi.IsNotModified(err):
		return event, false, nil
	case isNotFound(err):
		deleted = true
	case err != nil:
		return nil, false, ctxerr.Wrap(c.config.Context, err, "retrieving Google calendar event")
	}
	if !deleted && gEvent.Status != "cancelled" {
		if details.ETag == gEvent.Etag {
			// Event was not modified
			return event, false, nil
		}
		endTime, err := time.Parse(time.RFC3339, gEvent.End.DateTime)
		if err != nil {
			return nil, false, ctxerr.Wrap(
				c.config.Context, err, fmt.Sprintf("parsing Google calendar event end time: %s", gEvent.End.DateTime),
			)
		}
		// If event already ended, it is effectively deleted
		if endTime.After(time.Now()) {
			startTime, err := time.Parse(time.RFC3339, gEvent.Start.DateTime)
			if err != nil {
				return nil, false, ctxerr.Wrap(
					c.config.Context, err, fmt.Sprintf("parsing Google calendar event start time: %s", gEvent.Start.DateTime),
				)
			}
			fleetEvent, err := c.googleEventToFleetEvent(startTime, endTime, gEvent)
			if err != nil {
				return nil, false, err
			}
			return fleetEvent, true, nil
		}
	}

	newStartDate := event.StartTime.Add(24 * time.Hour)
	if newStartDate.Weekday() == time.Saturday {
		newStartDate = newStartDate.Add(48 * time.Hour)
	} else if newStartDate.Weekday() == time.Sunday {
		newStartDate = newStartDate.Add(24 * time.Hour)
	}

	fleetEvent, err := c.CreateEvent(newStartDate, genBodyFn())
	if err != nil {
		return nil, false, err
	}
	return fleetEvent, true, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ae *googleapi.Error
	ok := errors.As(err, &ae)
	return ok && ae.Code == http.StatusNotFound
}

func (c *GoogleCalendar) unmarshalDetails(event *fleet.CalendarEvent) (*eventDetails, error) {
	var details eventDetails
	err := json.Unmarshal(event.Data, &details)
	if err != nil {
		return nil, ctxerr.Wrap(c.config.Context, err, "unmarshaling Google calendar event details")
	}
	if details.ID == "" {
		return nil, ctxerr.Errorf(c.config.Context, "missing Google calendar event ID")
	}
	if details.ETag == "" {
		return nil, ctxerr.Errorf(c.config.Context, "missing Google calendar event ETag")
	}
	return &details, nil
}

func (c *GoogleCalendar) CreateEvent(dayOfEvent time.Time, body string) (*fleet.CalendarEvent, error) {
	if c.config == nil {
		return nil, errors.New("the Google calendar is not connected. Please call Configure first")
	}
	if c.timezoneOffset == nil {
		err := getTimezone(c)
		if err != nil {
			return nil, err
		}
	}

	location := time.FixedZone("", *c.timezoneOffset)
	dayStart := time.Date(dayOfEvent.Year(), dayOfEvent.Month(), dayOfEvent.Day(), startHour, 0, 0, 0, location)
	dayEnd := time.Date(dayOfEvent.Year(), dayOfEvent.Month(), dayOfEvent.Day(), endHour, 0, 0, 0, location)

	now := time.Now().In(location)
	if dayEnd.Before(now) {
		// The workday has already ended.
		return nil, ctxerr.Wrap(c.config.Context, fleet.DayEndedError{Msg: "cannot schedule an event for a day that has already ended"})
	}

	// Adjust day start if workday already started
	if dayStart.Before(now) {
		dayStart = now.Truncate(eventLength)
		if dayStart.Before(now) {
			dayStart = dayStart.Add(eventLength)
		}
		if dayStart.Equal(dayEnd) {
			return nil, ctxerr.Wrap(c.config.Context, fleet.DayEndedError{Msg: "no time available for event"})
		}
	}
	eventStart := dayStart
	eventEnd := dayStart.Add(eventLength)

	searchStart := dayStart.Add(-24 * time.Hour)
	events, err := c.config.API.ListEvents(searchStart.Format(time.RFC3339), dayEnd.Format(time.RFC3339))
	if err != nil {
		return nil, ctxerr.Wrap(c.config.Context, err, "listing Google calendar events")
	}
	for _, gEvent := range events.Items {
		// Ignore cancelled events
		if gEvent.Status == "cancelled" {
			continue
		}

		// Ignore events that the user has declined
		var attending bool
		if len(gEvent.Attendees) == 0 {
			// No attendees, so we assume the user is attending
			attending = true
		} else {
			for _, attendee := range gEvent.Attendees {
				if attendee.Email == c.config.UserEmail {
					if attendee.ResponseStatus != "declined" {
						attending = true
					}
					break
				}
			}
		}
		if !attending {
			continue
		}

		// Ignore events that will end before our event
		endTime, err := time.Parse(time.RFC3339, gEvent.End.DateTime)
		if err != nil {
			return nil, ctxerr.Wrap(
				c.config.Context, err, fmt.Sprintf("parsing Google calendar event end time: %s", gEvent.End.DateTime),
			)
		}
		if endTime.Before(eventStart) || endTime.Equal(eventStart) {
			continue
		}

		startTime, err := time.Parse(time.RFC3339, gEvent.Start.DateTime)
		if err != nil {
			return nil, ctxerr.Wrap(
				c.config.Context, err, fmt.Sprintf("parsing Google calendar event start time: %s", gEvent.Start.DateTime),
			)
		}

		if startTime.Before(eventEnd) {
			// Event occurs during our event, so we need to adjust.
			fmt.Printf("VICTOR Adjusting event times due to %s: %s - %s\n", gEvent.Summary, eventStart, eventEnd)
			var isLastSlot bool
			eventStart, eventEnd, isLastSlot = adjustEventTimes(endTime, dayEnd)
			if isLastSlot {
				break
			}
			continue
		}
		// Since events are sorted by startTime, all subsequent events are after our event, so we can stop processing
		break
	}

	event := &calendar.Event{}
	event.Start = &calendar.EventDateTime{DateTime: eventStart.Format(time.RFC3339)}
	event.End = &calendar.EventDateTime{DateTime: eventEnd.Format(time.RFC3339)}
	event.Summary = eventTitle
	event.Description = body
	event, err = c.config.API.CreateEvent(event)
	if err != nil {
		return nil, ctxerr.Wrap(c.config.Context, err, "creating Google calendar event")
	}

	// Convert Google event to Fleet event
	fleetEvent, err := c.googleEventToFleetEvent(eventStart, eventEnd, event)
	if err != nil {
		return nil, err
	}
	level.Debug(c.config.Logger).Log("msg", "created Google calendar events", "user", c.config.UserEmail, "startTime", eventStart)
	fmt.Printf("VICTOR Created event with id:%s and ETag:%s\n", event.Id, event.Etag)

	return fleetEvent, nil
}

func adjustEventTimes(endTime time.Time, dayEnd time.Time) (eventStart time.Time, eventEnd time.Time, isLastSlot bool) {
	eventStart = endTime.Truncate(eventLength)
	if eventStart.Before(endTime) {
		eventStart = eventStart.Add(eventLength)
	}
	eventEnd = eventStart.Add(eventLength)
	// If we are at the end of the day, pick the last slot
	if eventEnd.After(dayEnd) {
		eventEnd = dayEnd
		eventStart = eventEnd.Add(-eventLength)
		isLastSlot = true
	}
	if eventEnd.Equal(dayEnd) {
		isLastSlot = true
	}
	return eventStart, eventEnd, isLastSlot
}

func getTimezone(gCal *GoogleCalendar) error {
	config := gCal.config
	setting, err := config.API.GetSetting("timezone")
	if err != nil {
		return ctxerr.Wrap(config.Context, err, "retrieving Google calendar timezone")
	}

	loc, err := time.LoadLocation(setting.Value)
	if err != nil {
		// Could not load location, use EST
		level.Warn(config.Logger).Log("msg", "parsing Google calendar timezone", "timezone", setting.Value, "err", err)
		loc, _ = time.LoadLocation("America/New_York")
	}
	_, timezoneOffset := time.Now().In(loc).Zone()
	gCal.timezoneOffset = &timezoneOffset
	return nil
}

func (c *GoogleCalendar) googleEventToFleetEvent(startTime time.Time, endTime time.Time, event *calendar.Event) (
	*fleet.CalendarEvent, error,
) {
	fleetEvent := &fleet.CalendarEvent{}
	fleetEvent.StartTime = startTime
	fleetEvent.EndTime = endTime
	fleetEvent.Email = c.config.UserEmail
	details := &eventDetails{
		ID:   event.Id,
		ETag: event.Etag,
	}
	detailsJson, err := json.Marshal(details)
	if err != nil {
		return nil, ctxerr.Wrap(c.config.Context, err, "marshaling Google calendar event details")
	}
	fleetEvent.Data = detailsJson
	return fleetEvent, nil
}

func (c *GoogleCalendar) DeleteEvent(event *fleet.CalendarEvent) error {
	if c.config == nil {
		return errors.New("the Google calendar is not connected. Please call Configure first")
	}
	details, err := c.unmarshalDetails(event)
	if err != nil {
		return err
	}
	err = c.config.API.DeleteEvent(details.ID)
	if err != nil {
		return ctxerr.Wrap(c.config.Context, err, "deleting Google calendar event")
	}
	return nil
}
