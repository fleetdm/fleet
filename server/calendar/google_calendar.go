package calendar

import (
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
	"time"
)

// GoogleCalendar is an implementation of the Calendar interface that uses the
// Google Calendar API to manage events.
type GoogleCalendar struct {
	service *calendar.Service
	config  *Config
}

type eventDetails struct {
	CalendarID string `json:"calendarID"`
	Etag       string `json:"etag"`
}

func NewCalendarConn(config Config) (*GoogleCalendar, error) {
	// Create a new calendar service
	conf := &jwt.Config{
		Email: config.IntegrationConfig.Email,
		Scopes: []string{
			"https://www.googleapis.com/auth/calendar.events", "https://www.googleapis.com/auth/calendar.settings.readonly",
		},
		PrivateKey: []byte(config.IntegrationConfig.PrivateKey),
		TokenURL:   google.JWTTokenURL,
		Subject:    config.UserEmail,
	}
	client := conf.Client(config.Context)
	service, err := calendar.NewService(config.Context, option.WithHTTPClient(client))
	if err != nil {
		return nil, ctxerr.Wrap(config.Context, err, "creating Google calendar service")
	}

	// TODO: Get timezone

	gCal := &GoogleCalendar{
		service: service,
		config:  &config,
	}
	return gCal, nil
}

func (c *GoogleCalendar) GetEvent(e *Event) (event *Event, timeChanged bool, err error) {
	return nil, false, nil
}

func (c *GoogleCalendar) CreateEvent(dayInUsersTimezone time.Time, body string) (event *Event, err error) {
	// Get all events between 9 and 5
	// TODO: Test with event that starts earlier and ends later.
	// Figure out if we are in the middle of the day already.
	const maxResults = 100
	//list, err := c.service.Events.List("primary").EventTypes("default").MaxResults(maxResults).OrderBy("startTime").SingleEvents(true).TimeMin(time.Now().Format(time.RFC3339)).Do()
	//if err != nil {
	//	return nil, ctxerr.Wrap(c.config.Context, err, "retrieving list of Google calendar events")
	//}

	return nil, nil
}

func (c *GoogleCalendar) DeleteEvent(event *Event) error {
	return nil
}
