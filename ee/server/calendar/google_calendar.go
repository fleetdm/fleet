package calendar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// The calendar package has the following features for testing:
// 1. High level UserCalendar interface and Low level GoogleCalendarAPI interface can have a custom implementations.
// 2. Setting "client_email" to "calendar-mock@example.com" in the API key will use a mock in-memory implementation GoogleCalendarMockAPI of GoogleCalendarAPI.
// 3. Setting FLEET_GOOGLE_CALENDAR_PLUS_ADDRESSING environment variable to "1" will strip the "plus addressing" from the user email, effectively allowing a single user
//    to create multiple events in the same calendar. This is useful for load testing. For example: john+test@example.com becomes john@example.com

const (
	eventTitle  = "💻🚫Downtime"
	startHour   = 9
	endHour     = 17
	eventLength = 30 * time.Minute
	calendarID  = "primary"
	mockEmail   = "calendar-mock@example.com"
	loadEmail   = "calendar-load@example.com"
)

var (
	calendarScopes = []string{
		"https://www.googleapis.com/auth/calendar.events",
		"https://www.googleapis.com/auth/calendar.settings.readonly",
	}
	plusAddressing      = os.Getenv("FLEET_GOOGLE_CALENDAR_PLUS_ADDRESSING") == "1"
	plusAddressingRegex = regexp.MustCompile(`\+.*@`)
)

type GoogleCalendarConfig struct {
	Context           context.Context
	IntegrationConfig *fleet.GoogleCalendarIntegration
	Logger            kitlog.Logger
	// Should be nil for production
	API GoogleCalendarAPI
}

// GoogleCalendar is an implementation of the UserCalendar interface that uses the
// Google Calendar API to manage events.
type GoogleCalendar struct {
	config            *GoogleCalendarConfig
	currentUserEmail  string
	adjustedUserEmail string
	location          *time.Location
}

func NewGoogleCalendar(config *GoogleCalendarConfig) *GoogleCalendar {
	switch {
	case config.API != nil:
		// Use the provided API.
	case config.IntegrationConfig.ApiKey[fleet.GoogleCalendarEmail] == loadEmail:
		config.API = &GoogleCalendarLoadAPI{Logger: config.Logger}
	case config.IntegrationConfig.ApiKey[fleet.GoogleCalendarEmail] == mockEmail:
		config.API = &GoogleCalendarMockAPI{config.Logger}
	default:
		config.API = &GoogleCalendarLowLevelAPI{logger: config.Logger}
	}
	return &GoogleCalendar{
		config: config,
	}
}

type GoogleCalendarAPI interface {
	Configure(ctx context.Context, serviceAccountEmail, privateKey, userToImpersonateEmail string) error
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
	logger  kitlog.Logger
}

// Configure creates a new Google Calendar service using the provided credentials.
func (lowLevelAPI *GoogleCalendarLowLevelAPI) Configure(
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

func adjustEmail(email string) string {
	if plusAddressing {
		return plusAddressingRegex.ReplaceAllString(email, "@")
	}
	return email
}

func (lowLevelAPI *GoogleCalendarLowLevelAPI) GetSetting(name string) (*calendar.Setting, error) {
	result, err := lowLevelAPI.withRetry(
		func() (any, error) {
			return lowLevelAPI.service.Settings.Get(name).Do()
		},
	)
	return result.(*calendar.Setting), err
}

func (lowLevelAPI *GoogleCalendarLowLevelAPI) CreateEvent(event *calendar.Event) (*calendar.Event, error) {
	result, err := lowLevelAPI.withRetry(
		func() (any, error) {
			return lowLevelAPI.service.Events.Insert(calendarID, event).Do()
		},
	)
	return result.(*calendar.Event), err
}

func (lowLevelAPI *GoogleCalendarLowLevelAPI) GetEvent(id, eTag string) (*calendar.Event, error) {
	result, err := lowLevelAPI.withRetry(
		func() (any, error) {
			return lowLevelAPI.service.Events.Get(calendarID, id).IfNoneMatch(eTag).Do()
		},
	)
	return result.(*calendar.Event), err
}

func (lowLevelAPI *GoogleCalendarLowLevelAPI) ListEvents(timeMin, timeMax string) (*calendar.Events, error) {
	result, err := lowLevelAPI.withRetry(
		func() (any, error) {
			// Default maximum number of events returned is 250, which should be sufficient for most calendars.
			return lowLevelAPI.service.Events.List(calendarID).
				EventTypes("default").
				OrderBy("startTime").
				SingleEvents(true).
				TimeMin(timeMin).
				TimeMax(timeMax).
				ShowDeleted(false).
				Do()
		},
	)
	return result.(*calendar.Events), err
}

func (lowLevelAPI *GoogleCalendarLowLevelAPI) DeleteEvent(id string) error {
	_, err := lowLevelAPI.withRetry(
		func() (any, error) {
			return nil, lowLevelAPI.service.Events.Delete(calendarID, id).Do()
		},
	)
	return err
}

func (lowLevelAPI *GoogleCalendarLowLevelAPI) withRetry(fn func() (any, error)) (any, error) {
	retryStrategy := backoff.NewExponentialBackOff()
	retryStrategy.MaxElapsedTime = 10 * time.Minute
	var result any
	err := backoff.Retry(
		func() error {
			var err error
			result, err = fn()
			if err != nil {
				if isRateLimited(err) {
					level.Debug(lowLevelAPI.logger).Log("msg", "rate limited by Google calendar API", "err", err)
					return err
				}
				return backoff.Permanent(err)
			}
			return nil
		}, retryStrategy,
	)
	return result, err
}

func (c *GoogleCalendar) Configure(userEmail string) error {
	adjustedUserEmail := adjustEmail(userEmail)
	err := c.config.API.Configure(
		c.config.Context, c.config.IntegrationConfig.ApiKey[fleet.GoogleCalendarEmail],
		c.config.IntegrationConfig.ApiKey[fleet.GoogleCalendarPrivateKey], adjustedUserEmail,
	)
	if err != nil {
		return ctxerr.Wrap(c.config.Context, err, "creating Google calendar service")
	}
	c.currentUserEmail = userEmail
	c.adjustedUserEmail = adjustedUserEmail
	// Clear the timezone offset so that it will be recalculated
	c.location = nil
	return nil
}

func (c *GoogleCalendar) GetAndUpdateEvent(event *fleet.CalendarEvent, genBodyFn func(conflict bool) string) (
	*fleet.CalendarEvent, bool, error,
) {
	// We assume that the Fleet event has not already ended. We will simply return it if it has not been modified.
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
	// http.StatusNotFound should be very rare -- Google keeps events for a while after they are deleted
	case isNotFound(err):
		deleted = true
	case err != nil:
		return nil, false, ctxerr.Wrap(c.config.Context, err, "retrieving Google calendar event")
	}
	if !deleted && gEvent.Status != "cancelled" {
		if details.ETag != "" && details.ETag == gEvent.Etag {
			// Event was not modified
			return event, false, nil
		}
		if gEvent.End == nil || (gEvent.End.DateTime == "" && gEvent.End.Date == "") {
			// We should not see this error. If we do, we can work around by treating event as deleted.
			return nil, false, ctxerr.Errorf(c.config.Context, "missing end date/time for Google calendar event: %s", gEvent.Id)
		}

		if gEvent.End.DateTime == "" {
			// User has modified the event to be an all-day event. All-day events are problematic because they depend on the user's timezone.
			// We won't handle all-day events at this time, and treat the event as deleted.
			err = c.DeleteEvent(event)
			if err != nil {
				level.Warn(c.config.Logger).Log("msg", "deleting Google calendar event which was changed to all-day event", "err", err)
			}
			deleted = true
		}

		var endTime *time.Time
		if !deleted {
			endTime, err = c.parseDateTime(gEvent.End)
			if err != nil {
				return nil, false, err
			}
			if !endTime.After(time.Now()) {
				// If event already ended, it is effectively deleted
				// Delete this event to prevent confusion. This operation should be rare.
				err = c.DeleteEvent(event)
				if err != nil {
					level.Warn(c.config.Logger).Log("msg", "deleting Google calendar event which is in the past", "err", err)
				}
				deleted = true
			}
		}
		if !deleted {
			if gEvent.Start == nil || (gEvent.Start.DateTime == "" && gEvent.Start.Date == "") {
				// We should not see this error. If we do, we can work around by treating event as deleted.
				return nil, false, ctxerr.Errorf(c.config.Context, "missing start date/time for Google calendar event: %s", gEvent.Id)
			}
			if gEvent.Start.DateTime == "" {
				// User has modified the event to be an all-day event. All-day events are problematic because they depend on the user's timezone.
				// We won't handle all-day events at this time, and treat the event as deleted.
				err = c.DeleteEvent(event)
				if err != nil {
					level.Warn(c.config.Logger).Log("msg", "deleting Google calendar event which was changed to all-day event", "err", err)
				}
				deleted = true
			}
		}
		if !deleted {
			startTime, err := c.parseDateTime(gEvent.Start)
			if err != nil {
				return nil, false, err
			}
			fleetEvent, err := c.googleEventToFleetEvent(*startTime, *endTime, gEvent)
			if err != nil {
				return nil, false, err
			}
			return fleetEvent, true, nil
		}
	}

	newStartDate := calculateNewEventDate(event.StartTime)

	fleetEvent, err := c.CreateEvent(newStartDate, genBodyFn)
	if err != nil {
		return nil, false, err
	}
	return fleetEvent, true, nil
}

func calculateNewEventDate(oldStartDate time.Time) time.Time {
	// Note: we do not handle time changes (daylight savings time, etc.) -- assuming 1 day is always 24 hours.
	newStartDate := oldStartDate.Add(24 * time.Hour)
	if newStartDate.Weekday() == time.Saturday {
		newStartDate = newStartDate.Add(48 * time.Hour)
	} else if newStartDate.Weekday() == time.Sunday {
		newStartDate = newStartDate.Add(24 * time.Hour)
	}
	return newStartDate
}

func (c *GoogleCalendar) parseDateTime(eventDateTime *calendar.EventDateTime) (*time.Time, error) {
	var t time.Time
	var err error
	if eventDateTime.TimeZone != "" {
		loc := getLocation(eventDateTime.TimeZone, c.config)
		t, err = time.ParseInLocation(time.RFC3339, eventDateTime.DateTime, loc)
	} else {
		t, err = time.Parse(time.RFC3339, eventDateTime.DateTime)
	}
	if err != nil {
		return nil, ctxerr.Wrap(
			c.config.Context, err, fmt.Sprintf("parsing Google calendar event time: %s", eventDateTime.DateTime),
		)
	}
	return &t, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var ae *googleapi.Error
	ok := errors.As(err, &ae)
	return ok && ae.Code == http.StatusNotFound
}

func isAlreadyDeleted(err error) bool {
	if err == nil {
		return false
	}
	var ae *googleapi.Error
	ok := errors.As(err, &ae)
	return ok && ae.Code == http.StatusGone
}

func isRateLimited(err error) bool {
	if err == nil {
		return false
	}
	var ae *googleapi.Error
	ok := errors.As(err, &ae)
	return ok && (ae.Code == http.StatusTooManyRequests ||
		(ae.Code == http.StatusForbidden &&
			(ae.Message == "Rate Limit Exceeded" || ae.Message == "User Rate Limit Exceeded" || ae.Message == "Calendar usage limits exceeded." || strings.HasPrefix(ae.Message, "Quota exceeded"))))
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
	// ETag is optional, but we need it to check if the event was modified
	return &details, nil
}

func (c *GoogleCalendar) CreateEvent(dayOfEvent time.Time, genBodyFn func(conflict bool) string) (*fleet.CalendarEvent, error) {
	return c.createEvent(dayOfEvent, genBodyFn, time.Now)
}

// createEvent creates a new event on the calendar on the given date. timeNow is a function that returns the current time.
// timeNow can be overwritten for testing
func (c *GoogleCalendar) createEvent(
	dayOfEvent time.Time, genBodyFn func(conflict bool) string, timeNow func() time.Time,
) (*fleet.CalendarEvent, error) {
	var err error
	if c.location == nil {
		c.location, err = getTimezone(c)
		if err != nil {
			return nil, err
		}
	}

	dayStart := time.Date(dayOfEvent.Year(), dayOfEvent.Month(), dayOfEvent.Day(), startHour, 0, 0, 0, c.location)
	dayEnd := time.Date(dayOfEvent.Year(), dayOfEvent.Month(), dayOfEvent.Day(), endHour, 0, 0, 0, c.location)

	now := timeNow().In(c.location)
	if dayEnd.Before(now) {
		// The workday has already ended.
		return nil, ctxerr.Wrap(c.config.Context, fleet.DayEndedError{Msg: "cannot schedule an event for a day that has already ended"})
	}

	// Adjust day start if workday already started
	if !dayStart.After(now) {
		dayStart = now.Truncate(eventLength)
		if dayStart.Before(now) {
			dayStart = dayStart.Add(eventLength)
		}
		if !dayStart.Before(dayEnd) {
			return nil, ctxerr.Wrap(c.config.Context, fleet.DayEndedError{Msg: "no time available for event"})
		}
	}
	eventStart := dayStart
	eventEnd := dayStart.Add(eventLength)

	events, err := c.config.API.ListEvents(dayStart.Format(time.RFC3339), dayEnd.Format(time.RFC3339))
	if err != nil {
		return nil, ctxerr.Wrap(c.config.Context, err, "listing Google calendar events")
	}
	var conflict bool
	for _, gEvent := range events.Items {
		// Ignore cancelled events
		if gEvent.Status == "cancelled" {
			continue
		}

		// Ignore all day events
		if gEvent.Start == nil || gEvent.Start.DateTime == "" || gEvent.End == nil || gEvent.End.DateTime == "" {
			continue
		}

		// Ignore events that the user has declined
		var declined bool
		for _, attendee := range gEvent.Attendees {
			if attendee.Email == c.adjustedUserEmail {
				// The user has declined the event, so this time is open for scheduling
				if attendee.ResponseStatus == "declined" {
					declined = true
					break
				}
			}
		}
		if declined {
			continue
		}

		// Ignore events that will end before our event
		endTime, err := c.parseDateTime(gEvent.End)
		if err != nil {
			return nil, err
		}
		if !endTime.After(eventStart) {
			continue
		}

		startTime, err := c.parseDateTime(gEvent.Start)
		if err != nil {
			return nil, err
		}

		if startTime.Before(eventEnd) {
			// Event occurs during our event, so we need to adjust.
			var isLastSlot bool
			eventStart, eventEnd, isLastSlot, conflict = adjustEventTimes(*endTime, dayEnd)
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
	event.Description = genBodyFn(conflict)
	event, err = c.config.API.CreateEvent(event)
	if err != nil {
		return nil, ctxerr.Wrap(c.config.Context, err, "creating Google calendar event")
	}

	// Convert Google event to Fleet event
	fleetEvent, err := c.googleEventToFleetEvent(eventStart, eventEnd, event)
	if err != nil {
		return nil, err
	}
	level.Debug(c.config.Logger).Log(
		"msg", "created Google calendar event", "user", c.adjustedUserEmail, "startTime", eventStart, "timezone", c.location.String(),
	)

	return fleetEvent, nil
}

func adjustEventTimes(endTime time.Time, dayEnd time.Time) (eventStart time.Time, eventEnd time.Time, isLastSlot bool, conflict bool) {
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
		conflict = true
	} else if eventEnd.Equal(dayEnd) {
		isLastSlot = true
	}
	return eventStart, eventEnd, isLastSlot, conflict
}

func getTimezone(gCal *GoogleCalendar) (*time.Location, error) {
	config := gCal.config
	setting, err := config.API.GetSetting("timezone")
	if err != nil {
		return nil, ctxerr.Wrap(config.Context, err, "retrieving Google calendar timezone")
	}

	return getLocation(setting.Value, config), nil
}

func getLocation(name string, config *GoogleCalendarConfig) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		// Could not load location, use EST
		level.Warn(config.Logger).Log("msg", "parsing Google calendar timezone", "timezone", name, "err", err)
		loc, _ = time.LoadLocation("America/New_York")
	}
	return loc
}

func (c *GoogleCalendar) googleEventToFleetEvent(startTime time.Time, endTime time.Time, event *calendar.Event) (
	*fleet.CalendarEvent, error,
) {
	fleetEvent := &fleet.CalendarEvent{}
	fleetEvent.StartTime = startTime
	fleetEvent.EndTime = endTime
	fleetEvent.Email = c.currentUserEmail
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
	details, err := c.unmarshalDetails(event)
	if err != nil {
		return err
	}
	err = c.config.API.DeleteEvent(details.ID)
	switch {
	case isAlreadyDeleted(err):
		return nil
	case err != nil:
		return ctxerr.Wrap(c.config.Context, err, "deleting Google calendar event")
	}
	return nil
}
