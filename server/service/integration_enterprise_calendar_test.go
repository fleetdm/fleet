package service

// Calendar integration tests for the enterprise (premium) suite.
//
// Belongs here: calendar events created for hosts failing policies, moving hosts between fleets,
// the calendar webhook callback, and event body updates.
//
// Does not belong here: other policy automations (integration_enterprise_policy_automations_test.go)
// and fleet integration settings in general (integration_enterprise_teams_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/calendar"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	commonCalendar "github.com/fleetdm/fleet/v4/server/service/calendar"
	"github.com/fleetdm/fleet/v4/server/service/redis_lock"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	googleCalendar "google.golang.org/api/calendar/v3"
)

func (s *integrationEnterpriseTestSuite) TestCalendarEvents() {
	t := s.T()
	ctx := context.Background()
	t.Cleanup(func() {
		calendar.ClearMockEvents()
		calendar.ClearMockChannels()
	})
	currentAppCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = s.ds.SaveAppConfig(ctx, currentAppCfg)
		require.NoError(t, err)
	})

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team2",
	})
	require.NoError(t, err)

	newHost := func(name string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   new(t.Name() + name),
			NodeKey:         new(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        "darwin",
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}

	host1Team1 := newHost("host1", &team1.ID)
	host2Team1 := newHost("host2", &team1.ID)
	host3Team2 := newHost("host3", &team2.ID)
	host4Team2 := newHost("host4", &team2.ID)
	_ = newHost("host5", nil) // global host

	team1Policy1Calendar, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy1Calendar",
			Query:                 "SELECT 1;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)
	team1Policy2, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy2",
			Query:                 "SELECT 2;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)
	team2Policy1Calendar, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team2Policy1Calendar",
			Query:                 "SELECT 3;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)
	team2Policy2, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team2Policy2",
			Query:                 "SELECT 4;",
			CalendarEventsEnabled: false,
		},
	)
	require.NoError(t, err)
	globalPolicy, err := s.ds.NewGlobalPolicy(
		ctx, nil, fleet.PolicyPayload{
			Name:                  "globalPolicy",
			Query:                 "SELECT 5;",
			CalendarEventsEnabled: false,
		},
	)
	require.NoError(t, err)

	// host1Team1 is failing a calendar policy and not a non-calendar policy (no results for global).
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(false),
			team1Policy2.ID:         new(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host2Team1 is passing the calendar policy but not the non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(true),
			team1Policy2.ID:         new(false),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host3Team2 is passing team2Policy1Calendar and failing the global policy
	// (not results for team2Policy2).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			team2Policy1Calendar.ID: new(true),
			team2Policy2.ID:         nil,
			globalPolicy.ID:         new(false),
		},
	), http.StatusOK, &distributedResp)

	// host4Team2 is not returning results for the calendar policy, failing the non-calendar
	// policy and passing the global policy.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host4Team2,
		map[uint]*bool{
			team2Policy1Calendar.ID: nil,
			team2Policy2.ID:         new(false),
			globalPolicy.ID:         new(true),
		},
	), http.StatusOK, &distributedResp)

	// Trigger the calendar cron with the global feature is disabled.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// No calendar events were created.
	allCalendarEvents, err := s.ds.ListCalendarEvents(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, allCalendarEvents)

	// Set global configuration for the calendar feature.
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.Integrations.GoogleCalendar = []*fleet.GoogleCalendarIntegration{
		{
			Domain: "example.com",
			ApiKey: fleet.GoogleCalendarApiKey{Values: map[string]string{
				fleet.GoogleCalendarEmail: "calendar-mock@example.com",
			}},
		},
	}
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	time.Sleep(2 * time.Second) // Wait 2 seconds for the app config cache to clear.

	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// No calendar events were created because we are missing enabling it on the teams.
	allCalendarEvents, err = s.ds.ListCalendarEvents(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, allCalendarEvents)

	// Run distributed/write for host4Team2 again, it should not attempt to trigger the webhook because
	// it's disabled for the teams.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host4Team2,
		map[uint]*bool{
			team2Policy1Calendar.ID: nil,
			team2Policy2.ID:         new(false),
			globalPolicy.ID:         new(true),
		},
	), http.StatusOK, &distributedResp)

	var (
		team1Fired   int
		team1FiredMu sync.Mutex
	)

	team1WebhookFired := make(chan struct{})
	team1WebhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		requestBodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		t.Logf("team1 webhook request: %s\n", requestBodyBytes)
		team1FiredMu.Lock()
		team1Fired++
		team1WebhookFired <- struct{}{}
		team1FiredMu.Unlock()
	}))
	t.Cleanup(func() {
		team1WebhookServer.Close()
	})

	team1.Config.Integrations.GoogleCalendar = &fleet.TeamGoogleCalendarIntegration{
		Enable:     true,
		WebhookURL: team1WebhookServer.URL,
	}
	team1, err = s.ds.SaveTeam(ctx, team1)
	require.NoError(t, err)

	var (
		team2Fired   int
		team2FiredMu sync.Mutex
	)

	team2WebhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		requestBodyBytes, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		t.Logf("team2 webhook request: %s\n", requestBodyBytes)
		team2FiredMu.Lock()
		team2Fired++
		team2FiredMu.Unlock()
	}))
	t.Cleanup(func() {
		team2WebhookServer.Close()
	})

	team2.Config.Integrations.GoogleCalendar = &fleet.TeamGoogleCalendarIntegration{
		Enable:     true,
		WebhookURL: team2WebhookServer.URL,
	}
	_, err = s.ds.SaveTeam(ctx, team2)
	require.NoError(t, err)

	//
	// Same distributed/write as before but they should not fire yet.
	//

	// host1Team1 is failing a calendar policy and not a non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(false),
			team1Policy2.ID:         new(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host2Team1 is passing the calendar policy but not the non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(true),
			team1Policy2.ID:         new(false),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host3Team2 is passing team2Policy1Calendar and failing the global policy
	// (not results for team2Policy2).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			team2Policy1Calendar.ID: new(true),
			team2Policy2.ID:         nil,
			globalPolicy.ID:         new(false),
		},
	), http.StatusOK, &distributedResp)

	// host4Team2 is not returning results for the calendar policy, failing the non-calendar
	// policy and passing the global policy.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host4Team2,
		map[uint]*bool{
			team2Policy1Calendar.ID: nil,
			team2Policy2.ID:         new(false),
			globalPolicy.ID:         new(true),
		},
	), http.StatusOK, &distributedResp)

	team1FiredMu.Lock()
	require.Zero(t, team1Fired)
	team1FiredMu.Unlock()

	team2FiredMu.Lock()
	require.Zero(t, team2Fired)
	team2FiredMu.Unlock()

	// Trigger the calendar cron, global feature enabled, team1 enabled, team2 not yet enabled
	// and hosts do not have an associated email yet.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	team1CalendarEvents, err := s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Empty(t, team1CalendarEvents)

	// Add an email but of another domain.
	err = s.ds.ReplaceHostDeviceMapping(ctx, host1Team1.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host1Team1.ID,
			Email:  "user@other.com",
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)

	// Trigger the calendar cron, global feature enabled, team1 enabled, team2 not yet enabled
	// and hosts do not have an associated email for the domain yet.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Empty(t, team1CalendarEvents)

	err = s.ds.ReplaceHostDeviceMapping(ctx, host1Team1.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host1Team1.ID,
			Email:  "user1@example.com",
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)

	// Trigger the calendar cron, global feature enabled, team1 enabled, team2 not yet enabled
	// and host1Team1 has a domain email associated.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// An event should be generated for host1Team1
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	require.NotZero(t, team1CalendarEvents[0].ID)
	require.Equal(t, "user1@example.com", team1CalendarEvents[0].Email)
	require.NotZero(t, team1CalendarEvents[0].StartTime)
	require.NotZero(t, team1CalendarEvents[0].EndTime)

	calendar.SetMockEventsToNow()

	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		// Update updated_at so the event gets updated (the event is updated regularly)
		_, err := db.ExecContext(ctx,
			`UPDATE calendar_events SET updated_at = DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 25 HOUR) WHERE id = ?`, team1CalendarEvents[0].ID)
		if err != nil {
			return err
		}
		// Set host1Team1 as online.
		if _, err := db.ExecContext(ctx,
			`UPDATE host_seen_times SET seen_time = CURRENT_TIMESTAMP WHERE host_id = ?`, host1Team1.ID); err != nil {
			return err
		}
		return nil
	})

	// Trigger the calendar cron, global feature enabled, team1 enabled, team2 not yet enabled
	// and host1Team1 has a domain email associated.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// Check that refetch on the host was set.
	host, err := s.ds.Host(ctx, host1Team1.ID)
	require.NoError(t, err)
	require.True(t, host.RefetchRequested)

	// host1Team1 is failing a calendar policy and not a non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(false),
			team1Policy2.ID:         new(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host2Team1 is passing the calendar policy but not the non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(true),
			team1Policy2.ID:         new(false),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	select {
	case <-team1WebhookFired:
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for team1 webhook to fire")
	}

	// Trigger again, nothing should fire as webhook has already fired.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	team1FiredMu.Lock()
	require.Equal(t, 1, team1Fired)
	team1FiredMu.Unlock()
	team2FiredMu.Lock()
	require.Equal(t, 0, team2Fired)
	team2FiredMu.Unlock()

	// Make host1Team1 pass all policies.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(true),
			team1Policy2.ID:         new(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// Trigger calendar should cleanup the events.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// Events in the user calendar should not be cleaned up because they are not in the future.
	mockEvents := calendar.ListGoogleMockEvents()
	require.NotEmpty(t, mockEvents)

	// Event should be cleaned up from our database.
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Empty(t, team1CalendarEvents)
}

func (s *integrationEnterpriseTestSuite) TestCalendarEventsTransferringHosts() {
	t := s.T()
	ctx := context.Background()
	t.Cleanup(func() {
		calendar.ClearMockEvents()
		calendar.ClearMockChannels()
	})
	currentAppCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = s.ds.SaveAppConfig(ctx, currentAppCfg)
		require.NoError(t, err)
	})

	// Set global configuration for the calendar feature.
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.Integrations.GoogleCalendar = []*fleet.GoogleCalendarIntegration{
		{
			Domain: "example.com",
			ApiKey: fleet.GoogleCalendarApiKey{Values: map[string]string{
				fleet.GoogleCalendarEmail: "calendar-mock@example.com",
			}},
		},
	}
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	time.Sleep(2 * time.Second) // Wait 2 seconds for the app config cache to clear.

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team2",
	})
	require.NoError(t, err)

	team1.Config.Integrations.GoogleCalendar = &fleet.TeamGoogleCalendarIntegration{
		Enable:     true,
		WebhookURL: "https://eoo.example.com",
	}
	team1, err = s.ds.SaveTeam(ctx, team1)
	require.NoError(t, err)
	team2.Config.Integrations.GoogleCalendar = &fleet.TeamGoogleCalendarIntegration{
		Enable:     true,
		WebhookURL: "https://foo.example.com",
	}
	team2, err = s.ds.SaveTeam(ctx, team2)
	require.NoError(t, err)

	newHost := func(name string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   new(t.Name() + name),
			NodeKey:         new(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        "darwin",
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}

	host1 := newHost("host1", &team1.ID)
	err = s.ds.ReplaceHostDeviceMapping(ctx, host1.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host1.ID,
			Email:  "user1@example.com",
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)

	team1Policy1, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy1",
			Query:                 "SELECT 1;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)
	team2Policy1, err := s.ds.NewTeamPolicy(
		ctx, team2.ID, nil, fleet.PolicyPayload{
			Name:                  "team2Policy1",
			Query:                 "SELECT 2;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)

	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1,
		map[uint]*bool{
			team1Policy1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	team1CalendarEvents, err := s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)

	// Check the calendar was created on the DB.
	hostCalendarEvent, calendarEvent, err := s.ds.GetHostCalendarEventByEmail(ctx, "user1@example.com")
	require.NoError(t, err)

	// Transfer host to team2.
	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team2.ID, []uint{host1.ID}))
	require.NoError(t, err)

	// host1 is failing team2's policy too.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1,
		map[uint]*bool{
			team2Policy1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// Check the calendar event entry was reused.
	hostCalendarEvent2, calendarEvent2, err := s.ds.GetHostCalendarEventByEmail(ctx, "user1@example.com")
	require.NoError(t, err)
	require.Equal(t, calendarEvent2.ID, calendarEvent.ID)
	require.Equal(t, hostCalendarEvent2.CalendarEventID, hostCalendarEvent.CalendarEventID)

	// Transfer host to global.
	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(nil, []uint{host1.ID}))
	require.NoError(t, err)

	// Move event to two days ago (to clean up the calendar event)
	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(ctx,
			`UPDATE calendar_events SET updated_at = DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 49 HOUR) WHERE id = ?`, team1CalendarEvents[0].ID)
		if err != nil {
			return err
		}
		return nil
	})

	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// Calendar event is cleaned up.
	_, _, err = s.ds.GetHostCalendarEventByEmail(ctx, "user1@example.com")
	require.True(t, fleet.IsNotFound(err))
}

func triggerAndWait(ctx context.Context, t *testing.T, ds fleet.Datastore, s *schedule.Schedule, timeout time.Duration) {
	// Following code assumes (for simplicity) only triggered runs.
	stats, err := ds.GetLatestCronStats(ctx, s.Name())
	require.NoError(t, err)
	var previousRunID int
	if len(stats) > 0 {
		previousRunID = stats[0].ID
	}

	var didTrigger bool
	for range 10 {
		_, didTrigger, err = s.Trigger(ctx)
		require.NoError(t, err)
		if didTrigger {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.True(t, didTrigger, "profile schedule did not trigger after 1 second of retries")

	timeoutCh := time.After(timeout)
	for {
		stats, err := ds.GetLatestCronStats(ctx, s.Name())
		require.NoError(t, err)
		if len(stats) > 0 && stats[0].ID > previousRunID && stats[0].Status == fleet.CronStatsStatusCompleted {
			t.Logf("cron %s:%d done", s.Name(), stats[0].ID)
			return
		}
		select {
		case <-timeoutCh:
			t.Logf("timeout waiting for schedule %s to complete", s.Name())
			t.Fail()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func (s *integrationEnterpriseTestSuite) TestCalendarCallback() {
	t := s.T()
	ctx := context.Background()
	t.Cleanup(func() {
		calendar.ClearMockEvents()
		calendar.ClearMockChannels()
	})
	currentAppCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = s.ds.SaveAppConfig(ctx, currentAppCfg)
		require.NoError(t, err)
	})

	origRecentUpdateDuration := commonCalendar.RecentCalendarUpdateDuration
	commonCalendar.RecentCalendarUpdateDuration = 1 * time.Millisecond
	t.Cleanup(func() {
		commonCalendar.RecentCalendarUpdateDuration = origRecentUpdateDuration
	})

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	newHost := func(name string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   new(t.Name() + name),
			NodeKey:         new(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        "darwin",
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}

	host1Team1 := newHost("host1", &team1.ID)
	host2Team1 := newHost("host2", &team1.ID)
	_ = newHost("host5", nil) // global host

	team1Policy1Calendar, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy1Calendar",
			Query:                 "SELECT 1;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)
	team1Policy2Calendar, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy2Calendar",
			Query:                 "SELECT 2;",
			CalendarEventsEnabled: true,
		},
	)
	require.NoError(t, err)
	globalPolicy, err := s.ds.NewGlobalPolicy(
		ctx, nil, fleet.PolicyPayload{
			Name:                  "globalPolicy",
			Query:                 "SELECT 5;",
			CalendarEventsEnabled: false,
		},
	)
	require.NoError(t, err)

	genDistributedReqWithPolicyResults := func(host *fleet.Host, policyResults map[uint]*bool) submitDistributedQueryResultsRequestShim {
		var (
			results  = make(map[string]json.RawMessage)
			statuses = make(map[string]interface{})
			messages = make(map[string]string)
		)
		for policyID, policyResult := range policyResults {
			distributedQueryName := hostPolicyQueryPrefix + fmt.Sprint(policyID)
			switch {
			case policyResult == nil:
				results[distributedQueryName] = json.RawMessage(`[]`)
				statuses[distributedQueryName] = 1
				messages[distributedQueryName] = "policy failed execution"
			case *policyResult:
				results[distributedQueryName] = json.RawMessage(`[{"1": "1"}]`)
				statuses[distributedQueryName] = 0
			case !*policyResult:
				results[distributedQueryName] = json.RawMessage(`[]`)
				statuses[distributedQueryName] = 0
			}
		}
		return submitDistributedQueryResultsRequestShim{
			NodeKey:  *host.NodeKey,
			Results:  results,
			Statuses: statuses,
			Messages: messages,
			Stats:    map[string]*fleet.Stats{},
		}
	}

	// host1Team1 is failing a calendar policy and not a non-calendar policy (no results for global).
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(false),
			team1Policy2Calendar.ID: new(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host2Team1 is passing the calendar policy but not the non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(true),
			team1Policy2Calendar.ID: new(false),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// Set global configuration for the calendar feature.
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.Integrations.GoogleCalendar = []*fleet.GoogleCalendarIntegration{
		{
			Domain: "example.com",
			ApiKey: fleet.GoogleCalendarApiKey{Values: map[string]string{
				fleet.GoogleCalendarEmail: calendar.MockEmail,
			}},
		},
	}
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	time.Sleep(2 * time.Second) // Wait 2 seconds for the app config cache to clear.

	team1.Config.Integrations.GoogleCalendar = &fleet.TeamGoogleCalendarIntegration{
		Enable:     true,
		WebhookURL: "https://example.com",
	}
	team1, err = s.ds.SaveTeam(ctx, team1)
	require.NoError(t, err)

	// Add email mapping for host1Team1
	const user1Email = "user1@example.com"
	err = s.ds.ReplaceHostDeviceMapping(ctx, host1Team1.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host1Team1.ID,
			Email:  user1Email,
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)
	assert.Equal(t, 0, calendar.MockChannelsCount())

	// Trigger the calendar cron, global feature enabled, team1 enabled
	// and host1Team1 has a domain email associated.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// An event should be generated for host1Team1
	team1CalendarEvents, err := s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	event := team1CalendarEvents[0]
	require.NotZero(t, event.ID)
	require.Equal(t, user1Email, event.Email)
	require.NotZero(t, event.StartTime)
	require.NotZero(t, event.EndTime)
	require.NotEmpty(t, event.UUID)
	bodyTag := event.GetBodyTag()
	assert.NotEmpty(t, bodyTag)
	assert.Equal(t, 1, calendar.MockChannelsCount())

	// Get channel ID
	type eventDetails struct {
		ChannelID string `json:"channel_id"`
		BodyTag   string `json:"body_tag"`
		ETag      string `json:"etag"`
	}
	var details eventDetails
	err = json.Unmarshal(event.Data, &details)
	require.NoError(t, err)

	// Send a sync command
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "sync",
	})

	// Send a regular callback with bad channel ID
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusForbidden, map[string]string{
		"X-Goog-Channel-Id":     "bad",
		"X-Goog-Resource-State": "exists",
	})

	// Send a regular callback
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "exists",
	})

	// Delete the event on the calendar
	calendar.ClearMockEvents()

	// Grab the distributed lock for this event
	distributedLock := redis_lock.NewLock(s.redisPool)
	lockValue := uuid.New().String()
	result, err := distributedLock.SetIfNotExist(ctx, commonCalendar.LockKeyPrefix+event.UUID, lockValue, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// This callback should put the event processing in a queue for async processing. It does not start async
	// processing because it assumes another server is handling this webhook, and that server will start
	// async processing.
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "exists",
	})

	uuids, err := distributedLock.GetSet(ctx, commonCalendar.QueueKey)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{event.UUID}, uuids)
	// The calendar should still be empty since event hasn't processed yet
	assert.Zero(t, len(calendar.ListGoogleMockEvents()))
	// We clear the queue
	assert.NoError(t, distributedLock.RemoveFromSet(ctx, commonCalendar.QueueKey, event.UUID))

	// We release the normal lock, but grab the reserve lock instead
	ok, err := distributedLock.ReleaseLock(ctx, commonCalendar.LockKeyPrefix+event.UUID, lockValue)
	require.NoError(t, err)
	assert.True(t, ok)
	result, err = distributedLock.SetIfNotExist(ctx, commonCalendar.ReservedLockKeyPrefix+event.UUID, lockValue, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	// This callback should put the event processing in a queue for async processing, AND start the async processing
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "exists",
	})

	uuids, err = distributedLock.GetSet(ctx, commonCalendar.QueueKey)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{event.UUID}, uuids)
	// The calendar should still be empty since event hasn't processed yet
	assert.Zero(t, len(calendar.ListGoogleMockEvents()))

	// We grab the normal lock again.
	lockValue2 := uuid.New().String()
	result, err = distributedLock.SetIfNotExist(ctx, commonCalendar.LockKeyPrefix+event.UUID, lockValue2, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	// We release the reserve lock.
	ok, err = distributedLock.ReleaseLock(ctx, commonCalendar.ReservedLockKeyPrefix+event.UUID, lockValue)
	require.NoError(t, err)
	assert.True(t, ok)
	// We release the normal lock.
	ok, err = distributedLock.ReleaseLock(ctx, commonCalendar.LockKeyPrefix+event.UUID, lockValue2)
	require.NoError(t, err)
	assert.True(t, ok)

	done := make(chan struct{})
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
			require.NoError(t, err)
			// Event should be rescheduled on a future date/time
			if len(team1CalendarEvents) == 1 && team1CalendarEvents[0].UUID == event.UUID &&
				team1CalendarEvents[0].StartTime.After(event.StartTime) {
				done <- struct{}{}
				return
			}
		}
	}()
	select {
	case <-done: // All good
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for calendar event processing")
	}

	eventRecreated := team1CalendarEvents[0]
	assert.NotZero(t, eventRecreated.ID)
	assert.Equal(t, user1Email, eventRecreated.Email)
	assert.NotZero(t, eventRecreated.StartTime)
	assert.NotZero(t, eventRecreated.EndTime)
	assert.NotEmpty(t, eventRecreated.UUID)
	assert.NotEqual(t, event.StartTime, eventRecreated.StartTime)
	assert.NotEqual(t, event.EndTime, eventRecreated.EndTime)
	assert.Equal(t, 1, calendar.MockChannelsCount())
	assert.Equal(t, 1, len(calendar.ListGoogleMockEvents()))

	// The previous event UUID should not work anymore, but API returns OK because this is a common occurrence.
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "exists",
	})

	err = json.Unmarshal(eventRecreated.Data, &details)
	require.NoError(t, err)
	assert.NotEmpty(t, details.BodyTag)
	bodyTag = details.BodyTag

	// New event callback should work
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+eventRecreated.UUID, []byte(""), http.StatusOK,
		map[string]string{
			"X-Goog-Channel-Id":     details.ChannelID,
			"X-Goog-Resource-State": "exists",
		})

	// Update the time of the event
	events := calendar.ListGoogleMockEvents()
	require.Len(t, events, 1)
	for _, e := range events {
		st, err := time.Parse(time.RFC3339, e.Start.DateTime)
		require.NoError(t, err)
		newStartTime := st.Add(5 * time.Minute).Format(time.RFC3339)
		e.Start.DateTime = newStartTime
	}

	// New event callback should cause the time to be updated in the DB
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+eventRecreated.UUID, []byte(""), http.StatusOK,
		map[string]string{
			"X-Goog-Channel-Id":     details.ChannelID,
			"X-Goog-Resource-State": "exists",
		})

	// Check that the time was updated in the DB
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	eventUpdated := team1CalendarEvents[0]
	assert.NotZero(t, eventUpdated.ID)
	assert.Equal(t, user1Email, eventUpdated.Email)
	assert.Equal(t, eventRecreated.UUID, eventUpdated.UUID)
	assert.Greater(t, eventUpdated.StartTime, eventRecreated.StartTime)
	assert.Equal(t, eventRecreated.EndTime, eventUpdated.EndTime)
	assert.Equal(t, 1, calendar.MockChannelsCount())
	assert.Equal(t, bodyTag, eventRecreated.GetBodyTag())

	// Change the body contents of event.
	events = calendar.ListGoogleMockEvents()
	require.Len(t, events, 1)
	eTag := "description change etag"
	for _, e := range events {
		e.Etag = eTag
		e.Description = "new description"
	}
	// New event callback should cause Etag to update but Body tag to remain the same
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+eventRecreated.UUID, []byte(""), http.StatusOK,
		map[string]string{
			"X-Goog-Channel-Id":     details.ChannelID,
			"X-Goog-Resource-State": "exists",
		})
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	eventDescUpdated := team1CalendarEvents[0]
	err = json.Unmarshal(eventDescUpdated.Data, &details)
	require.NoError(t, err)
	assert.Equal(t, bodyTag, details.BodyTag)
	assert.Equal(t, eTag, details.ETag)

	// Update the time of the event again
	events = calendar.ListGoogleMockEvents()
	require.Len(t, events, 1)
	for _, e := range events {
		st, err := time.Parse(time.RFC3339, e.Start.DateTime)
		require.NoError(t, err)
		newStartTime := st.Add(5 * time.Minute).Format(time.RFC3339)
		e.Start.DateTime = newStartTime
		e.Etag += "1"
	}

	// Grab the lock
	event = eventUpdated
	lockValue = uuid.New().String()
	result, err = distributedLock.SetIfNotExist(ctx, commonCalendar.LockKeyPrefix+event.UUID, lockValue, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, result)

	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		// Update updated_at so the event gets updated (the event is updated regularly)
		_, err := db.ExecContext(ctx,
			`UPDATE calendar_events SET updated_at = DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 25 HOUR) WHERE id = ?`, event.ID)
		return err
	})

	// Trigger the calendar cron async. It should wait for the lock and set reserve lock.
	go triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 10*time.Second)
	done = make(chan struct{})
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			reserveLock, err := distributedLock.Get(ctx, commonCalendar.ReservedLockKeyPrefix+event.UUID)
			require.NoError(t, err)
			if reserveLock != nil {
				done <- struct{}{}
				return
			}
		}
	}()
	select {
	case <-done: // All good
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for cron to set reserve lock")
	}

	// Release the normal lock
	ok, err = distributedLock.ReleaseLock(ctx, commonCalendar.LockKeyPrefix+event.UUID, lockValue)
	require.NoError(t, err)
	assert.True(t, ok)

	// Wait for the event to update
	done = make(chan struct{})
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
			require.NoError(t, err)
			if len(team1CalendarEvents) == 1 && team1CalendarEvents[0].UUID == event.UUID &&
				team1CalendarEvents[0].StartTime.After(event.StartTime) {
				err = json.Unmarshal(team1CalendarEvents[0].Data, &details)
				require.NoError(t, err)
				assert.NotEqual(t, eTag, details.ETag, "ETag should have updated")
				done <- struct{}{}
				return
			}
		}
	}()
	select {
	case <-done: // All good
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for event to update during cron")
	}

	// Delete the event on the calendar
	calendar.ClearMockEvents()

	// Make host1Team1 pass all policies.
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(true),
			team1Policy2Calendar.ID: new(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// We set a flag that event was updated recently. Callback shouldn't do anything since event was updated recently
	_, err = distributedLock.SetIfNotExist(ctx, commonCalendar.RecentUpdateKeyPrefix+event.UUID, commonCalendar.RecentCalendarUpdateValue,
		1000)
	require.NoError(t, err)
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+eventRecreated.UUID, []byte(""), http.StatusOK,
		map[string]string{
			"X-Goog-Channel-Id":     details.ChannelID,
			"X-Goog-Resource-State": "exists",
		})
	assert.Equal(t, 1, calendar.MockChannelsCount())

	// Callback should work, but only clear the callback channel. Event in DB will be deleted on the next cron run.
	_, err = distributedLock.ReleaseLock(ctx, commonCalendar.RecentUpdateKeyPrefix+event.UUID, commonCalendar.RecentCalendarUpdateValue)
	require.NoError(t, err)
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+eventRecreated.UUID, []byte(""), http.StatusOK,
		map[string]string{
			"X-Goog-Channel-Id":     details.ChannelID,
			"X-Goog-Resource-State": "exists",
		})
	assert.Equal(t, 0, calendar.MockChannelsCount())

	previousEvent := team1CalendarEvents[0]
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	assert.Equal(t, previousEvent, team1CalendarEvents[0])

	err = s.ds.DeleteHost(ctx, host1Team1.ID)
	require.NoError(t, err)
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+eventRecreated.UUID, []byte(""), http.StatusOK,
		map[string]string{
			"X-Goog-Channel-Id":     details.ChannelID,
			"X-Goog-Resource-State": "exists",
		})

	// Trigger calendar should cleanup the events
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)
	assert.Equal(t, 0, calendar.MockChannelsCount())

	// Event should be cleaned up from our database.
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	assert.Empty(t, team1CalendarEvents)
}

func (s *integrationEnterpriseTestSuite) TestCalendarEventBodyUpdate() {
	t := s.T()
	ctx := context.Background()
	t.Cleanup(func() {
		calendar.ClearMockEvents()
		calendar.ClearMockChannels()
	})
	currentAppCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		err = s.ds.SaveAppConfig(ctx, currentAppCfg)
		require.NoError(t, err)
	})
	origRecentUpdateDuration := commonCalendar.RecentCalendarUpdateDuration
	commonCalendar.RecentCalendarUpdateDuration = 1 * time.Millisecond
	t.Cleanup(func() {
		commonCalendar.RecentCalendarUpdateDuration = origRecentUpdateDuration
	})

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	newHost := func(name string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   new(t.Name() + name),
			NodeKey:         new(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        "darwin",
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}

	host1Team1 := newHost("host1", &team1.ID)
	host2Team1 := newHost("host2", &team1.ID)
	_ = newHost("host5", nil) // global host

	team1Policy1Calendar, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy1Calendar",
			Query:                 "SELECT 1;",
			CalendarEventsEnabled: true,
			Description:           "team1Policy1CalendarDescription",
			Resolution:            "team1Policy1CalendarResolution",
		},
	)
	require.NoError(t, err)
	team1Policy2Calendar, err := s.ds.NewTeamPolicy(
		ctx, team1.ID, nil, fleet.PolicyPayload{
			Name:                  "team1Policy2Calendar",
			Query:                 "SELECT 2;",
			CalendarEventsEnabled: true,
			Description:           "team1Policy2CalendarDescription",
			Resolution:            "team1Policy2CalendarResolution",
		},
	)
	require.NoError(t, err)
	globalPolicy, err := s.ds.NewGlobalPolicy(
		ctx, nil, fleet.PolicyPayload{
			Name:                  "globalPolicy",
			Query:                 "SELECT 5;",
			CalendarEventsEnabled: false,
			Description:           "globalPolicyDescription",
			Resolution:            "globalPolicyResolution",
		},
	)
	require.NoError(t, err)

	genDistributedReqWithPolicyResults := func(host *fleet.Host, policyResults map[uint]*bool) submitDistributedQueryResultsRequestShim {
		var (
			results  = make(map[string]json.RawMessage)
			statuses = make(map[string]interface{})
			messages = make(map[string]string)
		)
		for policyID, policyResult := range policyResults {
			distributedQueryName := hostPolicyQueryPrefix + fmt.Sprint(policyID)
			switch {
			case policyResult == nil:
				results[distributedQueryName] = json.RawMessage(`[]`)
				statuses[distributedQueryName] = 1
				messages[distributedQueryName] = "policy failed execution"
			case *policyResult:
				results[distributedQueryName] = json.RawMessage(`[{"1": "1"}]`)
				statuses[distributedQueryName] = 0
			case !*policyResult:
				results[distributedQueryName] = json.RawMessage(`[]`)
				statuses[distributedQueryName] = 0
			}
		}
		return submitDistributedQueryResultsRequestShim{
			NodeKey:  *host.NodeKey,
			Results:  results,
			Statuses: statuses,
			Messages: messages,
			Stats:    map[string]*fleet.Stats{},
		}
	}

	// host1Team1 is failing a calendar policy and not a non-calendar policy (no results for global).
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(false),
			team1Policy2Calendar.ID: new(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// host2Team1 is passing the calendar policy but not the non-calendar policy (no results for global).
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(true),
			team1Policy2Calendar.ID: new(false),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)

	// Set global configuration for the calendar feature.
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.Integrations.GoogleCalendar = []*fleet.GoogleCalendarIntegration{
		{
			Domain: "example.com",
			ApiKey: fleet.GoogleCalendarApiKey{Values: map[string]string{
				fleet.GoogleCalendarEmail: calendar.MockEmail,
			}},
		},
	}
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	time.Sleep(2 * time.Second) // Wait 2 seconds for the app config cache to clear.

	team1.Config.Integrations.GoogleCalendar = &fleet.TeamGoogleCalendarIntegration{
		Enable:     true,
		WebhookURL: "https://example.com",
	}
	team1, err = s.ds.SaveTeam(ctx, team1)
	require.NoError(t, err)

	// Add email mapping for host1Team1
	const user1Email = "user1@example.com"
	err = s.ds.ReplaceHostDeviceMapping(ctx, host1Team1.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host1Team1.ID,
			Email:  user1Email,
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)
	assert.Equal(t, 0, calendar.MockChannelsCount())

	// Trigger the calendar cron, global feature enabled, team1 enabled
	// and host1Team1 has a domain email associated.
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	// An event should be generated for host1Team1
	team1CalendarEvents, err := s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	event := team1CalendarEvents[0]
	require.NotZero(t, event.ID)
	require.Equal(t, user1Email, event.Email)
	require.NotZero(t, event.StartTime)
	require.NotZero(t, event.EndTime)
	require.NotEmpty(t, event.UUID)
	assert.Equal(t, 1, calendar.MockChannelsCount())

	getEvents := func() []*googleCalendar.Event {
		calEvents := calendar.ListGoogleMockEvents()
		calEventValues := make([]*googleCalendar.Event, 0, len(calEvents))
		for _, v := range calEvents {
			calEventValues = append(calEventValues, v)
		}
		return calEventValues
	}

	calEvents := getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, team1Policy1Calendar.Description)
	assert.Contains(t, calEvents[0].Description, *team1Policy1Calendar.Resolution)

	// Remove resolution from policy
	team1Policy1Calendar.Resolution = nil
	require.NoError(t, s.ds.SavePolicy(ctx, team1Policy1Calendar, false, false))
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, fleet.CalendarDefaultDescription)
	assert.Contains(t, calEvents[0].Description, fleet.CalendarDefaultResolution)

	// Put resolution back
	team1Policy1Calendar.Resolution = new("putResolutionBack")
	require.NoError(t, s.ds.SavePolicy(ctx, team1Policy1Calendar, false, false))
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, team1Policy1Calendar.Description)
	assert.Contains(t, calEvents[0].Description, *team1Policy1Calendar.Resolution)

	// Change resolution
	team1Policy1Calendar.Resolution = new("changeResolution")
	require.NoError(t, s.ds.SavePolicy(ctx, team1Policy1Calendar, false, false))
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, team1Policy1Calendar.Description)
	assert.Contains(t, calEvents[0].Description, *team1Policy1Calendar.Resolution)

	// Cause another policy to fail
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(false),
			team1Policy2Calendar.ID: new(false),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, fleet.CalendarDefaultDescription)
	assert.Contains(t, calEvents[0].Description, fleet.CalendarDefaultResolution)

	// Cause the other policy to pass
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			team1Policy1Calendar.ID: new(false),
			team1Policy2Calendar.ID: new(true),
			globalPolicy.ID:         nil,
		},
	), http.StatusOK, &distributedResp)
	triggerAndWait(ctx, t, s.ds, s.calendarSchedule, 5*time.Second)

	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, team1Policy1Calendar.Description)
	assert.Contains(t, calEvents[0].Description, *team1Policy1Calendar.Resolution)

	// Get channel ID
	type eventDetails struct {
		ChannelID string `json:"channel_id"`
	}
	var details eventDetails
	err = json.Unmarshal(event.Data, &details)
	require.NoError(t, err)

	// Delete the event on the calendar
	calendar.ClearMockEvents()

	// Send a regular callback
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "exists",
	})

	// Make sure event body is correct after event was recreated
	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, team1Policy1Calendar.Description)
	assert.Contains(t, calEvents[0].Description, *team1Policy1Calendar.Resolution)

	// Get the new event
	team1CalendarEvents, err = s.ds.ListCalendarEvents(ctx, &team1.ID)
	require.NoError(t, err)
	require.Len(t, team1CalendarEvents, 1)
	event = team1CalendarEvents[0]
	err = json.Unmarshal(event.Data, &details)
	require.NoError(t, err)

	// Remove description from policy
	team1Policy1Calendar.Description = " "
	require.NoError(t, s.ds.SavePolicy(ctx, team1Policy1Calendar, false, false))
	// Delete the event on the calendar
	calendar.ClearMockEvents()

	// Send a regular callback
	_ = s.DoRawWithHeaders("POST", "/api/v1/fleet/calendar/webhook/"+event.UUID, []byte(""), http.StatusOK, map[string]string{
		"X-Goog-Channel-Id":     details.ChannelID,
		"X-Goog-Resource-State": "exists",
	})

	// Make sure event body is correct after event was recreated
	calEvents = getEvents()
	require.Len(t, calEvents, 1)
	assert.Contains(t, calEvents[0].Description, fleet.CalendarDefaultDescription)
	assert.Contains(t, calEvents[0].Description, fleet.CalendarDefaultResolution)
}
