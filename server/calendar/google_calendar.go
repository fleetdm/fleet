package calendar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log/level"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"time"
)

const eventTitle = "💻🚫Downtime"

// GoogleCalendar is an implementation of the Calendar interface that uses the
// Google Calendar API to manage events.
type GoogleCalendar struct {
	config         *GoogleCalendarConfig
	timezoneOffset *int
}

type GoogleCalendarAPI interface {
	Connect(ctx context.Context, email, privateKey, subject string) error
	GetSetting(name string) (*calendar.Setting, error)
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
func (lowLevelAPI *GoogleCalendarLowLevelAPI) Connect(ctx context.Context, email, privateKey, subject string) error {
	// Create a new calendar service
	conf := &jwt.Config{
		Email: email,
		Scopes: []string{
			"https://www.googleapis.com/auth/calendar.events", "https://www.googleapis.com/auth/calendar.settings.readonly",
		},
		PrivateKey: []byte(privateKey),
		TokenURL:   google.JWTTokenURL,
		Subject:    subject,
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
	return lowLevelAPI.service.Events.Insert("primary", event).Do()
}

func (lowLevelAPI *GoogleCalendarLowLevelAPI) GetEvent(id, eTag string) (*calendar.Event, error) {
	return lowLevelAPI.service.Events.Get("primary", id).IfNoneMatch(eTag).Do()
}

func (lowLevelAPI *GoogleCalendarLowLevelAPI) DeleteEvent(id string) error {
	return lowLevelAPI.service.Events.Delete("primary", id).Do()
}

func NewCalendar(config GoogleCalendarConfig) (*GoogleCalendar, error) {
	if config.API == nil {
		var lowLevelAPI GoogleCalendarAPI = &GoogleCalendarLowLevelAPI{}
		config.API = lowLevelAPI
	}
	err := config.API.Connect(
		config.Context, config.IntegrationConfig.Email, config.IntegrationConfig.PrivateKey, config.UserEmail,
	)
	if err != nil {
		return nil, ctxerr.Wrap(config.Context, err, "creating Google calendar service")
	}

	gCal := &GoogleCalendar{
		config: &config,
	}

	return gCal, nil
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

func (c *GoogleCalendar) GetEvent(event *fleet.CalendarEvent) (*fleet.CalendarEvent, bool, error) {
	details, err := c.unmarshalDetails(event)
	if err != nil {
		return nil, false, err
	}
	gEvent, err := c.config.API.GetEvent(details.ID, details.ETag)
	if googleapi.IsNotModified(err) {
		return event, true, nil
	}
	if err != nil {
		return nil, false, ctxerr.Wrap(c.config.Context, err, "retrieving Google calendar event")
	}

	startTime, err := time.Parse(time.RFC3339, gEvent.Start.DateTime)
	if err != nil {
		return nil, false, ctxerr.Wrap(
			c.config.Context, err, fmt.Sprintf("parsing Google calendar event start time: %s", gEvent.Start.DateTime),
		)
	}
	endTime, err := time.Parse(time.RFC3339, gEvent.End.DateTime)
	if err != nil {
		return nil, false, ctxerr.Wrap(
			c.config.Context, err, fmt.Sprintf("parsing Google calendar event end time: %s", gEvent.End.DateTime),
		)
	}

	// TODO: If event has been deleted or moved to the past, create a new event on the next day.

	fleetEvent, err := c.googleEventToFleetEvent(startTime, endTime, gEvent)
	if err != nil {
		return nil, false, err
	}
	return fleetEvent, false, nil
}

func (c *GoogleCalendar) unmarshalDetails(event *fleet.CalendarEvent) (*eventDetails, error) {
	var details eventDetails
	err := json.Unmarshal(event.Data, &details)
	if err != nil {
		return nil, ctxerr.Wrap(c.config.Context, err, "unmarshaling Google calendar event details")
	}
	if details.ID == "" {
		return nil, errors.New("missing Google calendar event ID")
	}
	if details.ETag == "" {
		return nil, errors.New("missing Google calendar event ETag")
	}
	return &details, nil
}

func (c *GoogleCalendar) CreateEvent(dayOfEvent time.Time, body string) (*fleet.CalendarEvent, error) {
	if c.timezoneOffset == nil {
		err := getTimezone(c)
		if err != nil {
			return nil, err
		}
	}
	// TODO: Get all events between 9 and 5
	// TODO: Test with event that starts earlier and ends later.
	// TODO: Figure out if we are in the middle of the day already.
	startTime := time.Date(dayOfEvent.Year(), dayOfEvent.Month(), dayOfEvent.Day(), 9, 0, 0, 0, time.FixedZone("", *c.timezoneOffset))
	endTime := time.Date(dayOfEvent.Year(), dayOfEvent.Month(), dayOfEvent.Day(), 9, 30, 0, 0, time.FixedZone("", *c.timezoneOffset))
	event := &calendar.Event{}
	event.Start = &calendar.EventDateTime{DateTime: startTime.Format(time.RFC3339)}
	event.End = &calendar.EventDateTime{DateTime: endTime.Format(time.RFC3339)}
	event.Summary = eventTitle
	event.Description = body
	event, err := c.config.API.CreateEvent(event)
	if err != nil {
		return nil, ctxerr.Wrap(c.config.Context, err, "creating Google calendar event")
	}

	// Convert Google event to Fleet event
	fleetEvent, err := c.googleEventToFleetEvent(startTime, endTime, event)
	if err != nil {
		return nil, err
	}

	return fleetEvent, nil
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
