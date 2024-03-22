package main

import (
	"context"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	kitlog "github.com/go-kit/log"
	"os"
	"testing"
	"time"

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
