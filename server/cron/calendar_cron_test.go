package cron

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/calendar"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestGetPreferredCalendarEventDate(t *testing.T) {
	t.Parallel()
	date := func(year int, month time.Month, day int) time.Time {
		return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	}
	for _, tc := range []struct {
		name      string
		year      int
		month     time.Month
		daysStart int
		daysEnd   int

		expected time.Time
	}{
		{
			name:      "March 2024 (before 3rd Tuesday)",
			year:      2024,
			month:     3,
			daysStart: 1,
			daysEnd:   19,

			expected: date(2024, 3, 19),
		},
		{
			name:      "March 2024 (past 3rd Tuesday)",
			year:      2024,
			month:     3,
			daysStart: 20,
			daysEnd:   31,

			expected: date(2024, 4, 16),
		},
		{
			name:      "April 2024 (before 3rd Tuesday)",
			year:      2024,
			month:     4,
			daysStart: 1,
			daysEnd:   16,

			expected: date(2024, 4, 16),
		},
		{
			name:      "April 2024 (after 3rd Tuesday)",
			year:      2024,
			month:     4,
			daysStart: 17,
			daysEnd:   30,

			expected: date(2024, 5, 21),
		},
		{
			name:      "May 2024 (before 3rd Tuesday)",
			year:      2024,
			month:     5,
			daysStart: 1,
			daysEnd:   21,

			expected: date(2024, 5, 21),
		},
		{
			name:      "May 2024 (after 3rd Tuesday)",
			year:      2024,
			month:     5,
			daysStart: 22,
			daysEnd:   31,

			expected: date(2024, 6, 18),
		},
		{
			name:      "Dec 2024 (before 3rd Tuesday)",
			year:      2024,
			month:     12,
			daysStart: 1,
			daysEnd:   17,

			expected: date(2024, 12, 17),
		},
		{
			name:      "Dec 2024 (after 3rd Tuesday)",
			year:      2024,
			month:     12,
			daysStart: 18,
			daysEnd:   31,

			expected: date(2025, 1, 21),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for day := tc.daysStart; day <= tc.daysEnd; day++ {
				actual := getPreferredCalendarEventDate(tc.year, tc.month, day)
				require.NotEqual(t, actual.Weekday(), time.Saturday)
				require.NotEqual(t, actual.Weekday(), time.Sunday)
				require.Equal(t, tc.expected, actual)
			}
		})
	}
}

// TestEventForDifferentHost tests case when event exists, but for a different host. Nothing should happen.
// The old event will eventually be cleaned up by the cleanup job, and afterward a new event will be created.
func TestEventForDifferentHost(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ctx := context.Background()
	logger := kitlog.With(kitlog.NewLogfmtLogger(os.Stdout))
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			Integrations: fleet.Integrations{
				GoogleCalendar: []*fleet.GoogleCalendarIntegration{
					{},
				},
			},
		}, nil
	}
	teamID1 := uint(1)
	ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
		return []*fleet.Team{
			{
				ID: teamID1,
				Config: fleet.TeamConfig{
					Integrations: fleet.TeamIntegrations{
						GoogleCalendar: &fleet.TeamGoogleCalendarIntegration{
							Enable: true,
						},
					},
				},
			},
		}, nil
	}
	policyID1 := uint(10)
	ds.GetCalendarPoliciesFunc = func(ctx context.Context, teamID uint) ([]fleet.PolicyCalendarData, error) {
		require.Equal(t, teamID1, teamID)
		return []fleet.PolicyCalendarData{
			{
				ID:   policyID1,
				Name: "Policy 1",
			},
		}, nil
	}
	hostID1 := uint(100)
	hostID2 := uint(101)
	userEmail1 := "user@example.com"
	ds.GetTeamHostsPolicyMembershipsFunc = func(
		ctx context.Context, domain string, teamID uint, policyIDs []uint,
	) ([]fleet.HostPolicyMembershipData, error) {
		require.Equal(t, teamID1, teamID)
		require.Equal(t, []uint{policyID1}, policyIDs)
		return []fleet.HostPolicyMembershipData{
			{
				HostID:  hostID1,
				Email:   userEmail1,
				Passing: false,
			},
		}, nil
	}
	// Return an existing event, but for a different host
	eventTime := time.Now().Add(time.Hour)
	ds.GetHostCalendarEventByEmailFunc = func(ctx context.Context, email string) (*fleet.HostCalendarEvent, *fleet.CalendarEvent, error) {
		require.Equal(t, userEmail1, email)
		calEvent := &fleet.CalendarEvent{
			ID:        1,
			Email:     email,
			StartTime: eventTime,
			EndTime:   eventTime,
		}
		hcEvent := &fleet.HostCalendarEvent{
			ID:              1,
			HostID:          hostID2,
			CalendarEventID: 1,
			WebhookStatus:   fleet.CalendarWebhookStatusNone,
		}
		return hcEvent, calEvent, nil
	}

	err := cronCalendarEvents(ctx, ds, logger)
	require.NoError(t, err)
}

func TestCalendarEventsMultipleHosts(t *testing.T) {
	ds := new(mock.Store)
	ctx := context.Background()
	logger := kitlog.With(kitlog.NewLogfmtLogger(os.Stdout))
	t.Cleanup(func() {
		calendar.ClearMockEvents()
	})

	//
	// Test setup
	//
	//	team1:
	//
	//	policyID1 (calendar)
	//	policyID2 (calendar)
	//
	// 	hostID1 has user1@example.com not passing policies.
	//	hostID2 has user2@example.com passing policies.
	//	hostID3 does not have example.com email and is not passing policies.
	//	hostID4 does not have example.com email and is passing policies.
	//

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			Integrations: fleet.Integrations{
				GoogleCalendar: []*fleet.GoogleCalendarIntegration{
					{
						Domain: "example.com",
						ApiKey: map[string]string{
							fleet.GoogleCalendarEmail: "calendar-mock@example.com",
						},
					},
				},
			},
		}, nil
	}

	teamID1 := uint(1)
	ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
		return []*fleet.Team{
			{
				ID: teamID1,
				Config: fleet.TeamConfig{
					Integrations: fleet.TeamIntegrations{
						GoogleCalendar: &fleet.TeamGoogleCalendarIntegration{
							Enable:     true,
							WebhookURL: "https://foo.example.com",
						},
					},
				},
			},
		}, nil
	}

	policyID1 := uint(10)
	policyID2 := uint(11)
	ds.GetCalendarPoliciesFunc = func(ctx context.Context, teamID uint) ([]fleet.PolicyCalendarData, error) {
		require.Equal(t, teamID1, teamID)
		return []fleet.PolicyCalendarData{
			{
				ID:   policyID1,
				Name: "Policy 1",
			},
			{
				ID:   policyID2,
				Name: "Policy 2",
			},
		}, nil
	}

	hostID1, userEmail1 := uint(100), "user1@example.com"
	hostID2, userEmail2 := uint(101), "user2@example.com"
	hostID3 := uint(102)
	hostID4 := uint(103)

	ds.GetTeamHostsPolicyMembershipsFunc = func(
		ctx context.Context, domain string, teamID uint, policyIDs []uint,
	) ([]fleet.HostPolicyMembershipData, error) {
		require.Equal(t, "example.com", domain)
		require.Equal(t, teamID1, teamID)
		require.Equal(t, []uint{policyID1, policyID2}, policyIDs)
		return []fleet.HostPolicyMembershipData{
			{
				HostID:  hostID1,
				Email:   userEmail1,
				Passing: false,
			},
			{
				HostID:  hostID2,
				Email:   userEmail2,
				Passing: true,
			},
			{
				HostID:  hostID3,
				Email:   "", // because it does not belong to example.com
				Passing: false,
			},
			{
				HostID:  hostID4,
				Email:   "", // because it does not belong to example.com
				Passing: true,
			},
		}, nil
	}

	ds.GetHostCalendarEventByEmailFunc = func(ctx context.Context, email string) (*fleet.HostCalendarEvent, *fleet.CalendarEvent, error) {
		return nil, nil, notFoundErr{}
	}

	var eventsMu sync.Mutex
	calendarEvents := make(map[string]*fleet.CalendarEvent)
	hostCalendarEvents := make(map[uint]*fleet.HostCalendarEvent)

	ds.CreateOrUpdateCalendarEventFunc = func(ctx context.Context,
		email string,
		startTime, endTime time.Time,
		data []byte,
		hostID uint,
		webhookStatus fleet.CalendarWebhookStatus,
	) (*fleet.CalendarEvent, error) {
		require.Equal(t, hostID1, hostID)
		require.Equal(t, userEmail1, email)
		require.Equal(t, fleet.CalendarWebhookStatusNone, webhookStatus)
		require.NotEmpty(t, data)
		require.NotZero(t, startTime)
		require.NotZero(t, endTime)

		eventsMu.Lock()
		calendarEventID := uint(len(calendarEvents) + 1)
		calendarEvents[email] = &fleet.CalendarEvent{
			ID:        calendarEventID,
			Email:     email,
			StartTime: startTime,
			EndTime:   endTime,
			Data:      data,
		}
		hostCalendarEventID := uint(len(hostCalendarEvents) + 1)
		hostCalendarEvents[hostID] = &fleet.HostCalendarEvent{
			ID:              hostCalendarEventID,
			HostID:          hostID,
			CalendarEventID: calendarEventID,
			WebhookStatus:   webhookStatus,
		}
		eventsMu.Unlock()
		return nil, nil
	}

	err := cronCalendarEvents(ctx, ds, logger)
	require.NoError(t, err)

	eventsMu.Lock()
	require.Len(t, calendarEvents, 1)
	require.Len(t, hostCalendarEvents, 1)
	eventsMu.Unlock()

	createdCalendarEvents := calendar.ListGoogleMockEvents()
	require.Len(t, createdCalendarEvents, 1)
}

type notFoundErr struct{}

func (n notFoundErr) IsNotFound() bool {
	return true
}

func (n notFoundErr) Error() string {
	return "not found"
}

func TestCalendarEvents1KHosts(t *testing.T) {
	ds := new(mock.Store)
	ctx := context.Background()
	var logger kitlog.Logger
	if os.Getenv("CALENDAR_TEST_LOGGING") != "" {
		logger = kitlog.With(kitlog.NewLogfmtLogger(os.Stdout))
	} else {
		logger = kitlog.NewNopLogger()
	}
	t.Cleanup(func() {
		calendar.ClearMockEvents()
	})

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			Integrations: fleet.Integrations{
				GoogleCalendar: []*fleet.GoogleCalendarIntegration{
					{
						Domain: "example.com",
						ApiKey: map[string]string{
							fleet.GoogleCalendarEmail: "calendar-mock@example.com",
						},
					},
				},
			},
		}, nil
	}

	teamID1 := uint(1)
	teamID2 := uint(2)
	teamID3 := uint(3)
	teamID4 := uint(4)
	teamID5 := uint(5)
	ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
		return []*fleet.Team{
			{
				ID: teamID1,
				Config: fleet.TeamConfig{
					Integrations: fleet.TeamIntegrations{
						GoogleCalendar: &fleet.TeamGoogleCalendarIntegration{
							Enable:     true,
							WebhookURL: "https://foo.example.com",
						},
					},
				},
			},
			{
				ID: teamID2,
				Config: fleet.TeamConfig{
					Integrations: fleet.TeamIntegrations{
						GoogleCalendar: &fleet.TeamGoogleCalendarIntegration{
							Enable:     true,
							WebhookURL: "https://foo.example.com",
						},
					},
				},
			},
			{
				ID: teamID3,
				Config: fleet.TeamConfig{
					Integrations: fleet.TeamIntegrations{
						GoogleCalendar: &fleet.TeamGoogleCalendarIntegration{
							Enable:     true,
							WebhookURL: "https://foo.example.com",
						},
					},
				},
			},
			{
				ID: teamID4,
				Config: fleet.TeamConfig{
					Integrations: fleet.TeamIntegrations{
						GoogleCalendar: &fleet.TeamGoogleCalendarIntegration{
							Enable:     true,
							WebhookURL: "https://foo.example.com",
						},
					},
				},
			},
			{
				ID: teamID5,
				Config: fleet.TeamConfig{
					Integrations: fleet.TeamIntegrations{
						GoogleCalendar: &fleet.TeamGoogleCalendarIntegration{
							Enable:     true,
							WebhookURL: "https://foo.example.com",
						},
					},
				},
			},
		}, nil
	}

	policyID1 := uint(10)
	policyID2 := uint(11)
	policyID3 := uint(12)
	policyID4 := uint(13)
	policyID5 := uint(14)
	policyID6 := uint(15)
	policyID7 := uint(16)
	policyID8 := uint(17)
	policyID9 := uint(18)
	policyID10 := uint(19)
	ds.GetCalendarPoliciesFunc = func(ctx context.Context, teamID uint) ([]fleet.PolicyCalendarData, error) {
		switch teamID {
		case teamID1:
			return []fleet.PolicyCalendarData{
				{
					ID:   policyID1,
					Name: "Policy 1",
				},
				{
					ID:   policyID2,
					Name: "Policy 2",
				},
			}, nil
		case teamID2:
			return []fleet.PolicyCalendarData{
				{
					ID:   policyID3,
					Name: "Policy 3",
				},
				{
					ID:   policyID4,
					Name: "Policy 4",
				},
			}, nil
		case teamID3:
			return []fleet.PolicyCalendarData{
				{
					ID:   policyID5,
					Name: "Policy 5",
				},
				{
					ID:   policyID6,
					Name: "Policy 6",
				},
			}, nil
		case teamID4:
			return []fleet.PolicyCalendarData{
				{
					ID:   policyID7,
					Name: "Policy 7",
				},
				{
					ID:   policyID8,
					Name: "Policy 8",
				},
			}, nil
		case teamID5:
			return []fleet.PolicyCalendarData{
				{
					ID:   policyID9,
					Name: "Policy 9",
				},
				{
					ID:   policyID10,
					Name: "Policy 10",
				},
			}, nil
		default:
			return nil, notFoundErr{}
		}
	}

	hosts := make([]fleet.HostPolicyMembershipData, 0, 1000)
	for i := 0; i < 1000; i++ {
		hosts = append(hosts, fleet.HostPolicyMembershipData{
			Email:              fmt.Sprintf("user%d@example.com", i),
			Passing:            i%2 == 0,
			HostID:             uint(i),
			HostDisplayName:    fmt.Sprintf("display_name%d", i),
			HostHardwareSerial: fmt.Sprintf("serial%d", i),
		})
	}

	ds.GetTeamHostsPolicyMembershipsFunc = func(
		ctx context.Context, domain string, teamID uint, policyIDs []uint,
	) ([]fleet.HostPolicyMembershipData, error) {
		var start, end int
		switch teamID {
		case teamID1:
			start, end = 0, 200
		case teamID2:
			start, end = 200, 400
		case teamID3:
			start, end = 400, 600
		case teamID4:
			start, end = 600, 800
		case teamID5:
			start, end = 800, 1000
		}
		return hosts[start:end], nil
	}

	ds.GetHostCalendarEventByEmailFunc = func(ctx context.Context, email string) (*fleet.HostCalendarEvent, *fleet.CalendarEvent, error) {
		return nil, nil, notFoundErr{}
	}

	eventsCreated := 0
	var eventsCreatedMu sync.Mutex

	eventPerHost := make(map[uint]*fleet.CalendarEvent)

	ds.CreateOrUpdateCalendarEventFunc = func(ctx context.Context,
		email string,
		startTime, endTime time.Time,
		data []byte,
		hostID uint,
		webhookStatus fleet.CalendarWebhookStatus,
	) (*fleet.CalendarEvent, error) {
		require.Equal(t, fmt.Sprintf("user%d@example.com", hostID), email)
		eventsCreatedMu.Lock()
		eventsCreated += 1
		eventPerHost[hostID] = &fleet.CalendarEvent{
			ID:        hostID,
			Email:     email,
			StartTime: startTime,
			EndTime:   endTime,
			Data:      data,
			UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
				CreateTimestamp: fleet.CreateTimestamp{
					CreatedAt: time.Now(),
				},
				UpdateTimestamp: fleet.UpdateTimestamp{
					UpdatedAt: time.Now(),
				},
			},
		}
		eventsCreatedMu.Unlock()
		require.Equal(t, fleet.CalendarWebhookStatusNone, webhookStatus)
		require.NotEmpty(t, data)
		require.NotZero(t, startTime)
		require.NotZero(t, endTime)
		// Currently, the returned calendar event is unused.
		return nil, nil
	}

	err := cronCalendarEvents(ctx, ds, logger)
	require.NoError(t, err)

	createdCalendarEvents := calendar.ListGoogleMockEvents()
	require.Equal(t, eventsCreated, 500)
	require.Len(t, createdCalendarEvents, 500)

	hosts = make([]fleet.HostPolicyMembershipData, 0, 1000)
	for i := 0; i < 1000; i++ {
		hosts = append(hosts, fleet.HostPolicyMembershipData{
			Email:              fmt.Sprintf("user%d@example.com", i),
			Passing:            true,
			HostID:             uint(i),
			HostDisplayName:    fmt.Sprintf("display_name%d", i),
			HostHardwareSerial: fmt.Sprintf("serial%d", i),
		})
	}

	ds.GetHostCalendarEventByEmailFunc = func(ctx context.Context, email string) (*fleet.HostCalendarEvent, *fleet.CalendarEvent, error) {
		hostID, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(email, "user"), "@example.com"))
		require.NoError(t, err)
		if hostID%2 == 0 {
			return nil, nil, notFoundErr{}
		}
		require.Contains(t, eventPerHost, uint(hostID))
		return &fleet.HostCalendarEvent{
			ID:              uint(hostID),
			HostID:          uint(hostID),
			CalendarEventID: uint(hostID),
			WebhookStatus:   fleet.CalendarWebhookStatusNone,
		}, eventPerHost[uint(hostID)], nil
	}

	ds.DeleteCalendarEventFunc = func(ctx context.Context, calendarEventID uint) error {
		return nil
	}

	err = cronCalendarEvents(ctx, ds, logger)
	require.NoError(t, err)

	createdCalendarEvents = calendar.ListGoogleMockEvents()
	require.Len(t, createdCalendarEvents, 0)
}
