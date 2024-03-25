package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/calendar"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/go-kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func newCalendarSchedule(
	ctx context.Context,
	instanceID string,
	ds fleet.Datastore,
	logger kitlog.Logger,
) (*schedule.Schedule, error) {
	const (
		name            = string(fleet.CronCalendar)
		defaultInterval = 5 * time.Minute
	)
	logger = kitlog.With(logger, "cron", name)
	s := schedule.New(
		ctx, name, instanceID, defaultInterval, ds, ds,
		schedule.WithAltLockID("calendar"),
		schedule.WithLogger(logger),
		schedule.WithJob(
			"calendar_events_cleanup",
			func(ctx context.Context) error {
				return cronCalendarEventsCleanup(ctx, ds, logger)
			},
		),
		schedule.WithJob(
			"calendar_events",
			func(ctx context.Context) error {
				return cronCalendarEvents(ctx, ds, logger)
			},
		),
	)

	return s, nil
}

func cronCalendarEvents(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return fmt.Errorf("load app config: %w", err)
	}

	if len(appConfig.Integrations.GoogleCalendar) == 0 {
		return nil
	}
	googleCalendarIntegrationConfig := appConfig.Integrations.GoogleCalendar[0]
	calendar := createUserCalendarFromConfig(ctx, googleCalendarIntegrationConfig, logger)
	domain := googleCalendarIntegrationConfig.Domain

	teams, err := ds.ListTeams(ctx, fleet.TeamFilter{
		User: &fleet.User{
			GlobalRole: ptr.String(fleet.RoleAdmin),
		},
	}, fleet.ListOptions{})
	if err != nil {
		return fmt.Errorf("list teams: %w", err)
	}

	for _, team := range teams {
		if err := cronCalendarEventsForTeam(
			ctx, ds, calendar, *team, appConfig.OrgInfo.OrgName, domain, logger,
		); err != nil {
			level.Info(logger).Log("msg", "events calendar cron", "team_id", team.ID, "err", err)
		}
	}

	return nil
}

func createUserCalendarFromConfig(ctx context.Context, config *fleet.GoogleCalendarIntegration, logger kitlog.Logger) fleet.UserCalendar {
	googleCalendarConfig := calendar.GoogleCalendarConfig{
		Context:           ctx,
		IntegrationConfig: config,
		Logger:            log.With(logger, "component", "google_calendar"),
	}
	return calendar.NewGoogleCalendar(&googleCalendarConfig)
}

func cronCalendarEventsForTeam(
	ctx context.Context,
	ds fleet.Datastore,
	calendar fleet.UserCalendar,
	team fleet.Team,
	orgName string,
	domain string,
	logger kitlog.Logger,
) error {
	if team.Config.Integrations.GoogleCalendar == nil ||
		!team.Config.Integrations.GoogleCalendar.Enable {
		return nil
	}

	policies, err := ds.GetCalendarPolicies(ctx, team.ID)
	if err != nil {
		return fmt.Errorf("get calendar policy ids: %w", err)
	}

	if len(policies) == 0 {
		return nil
	}

	logger = kitlog.With(logger, "team_id", team.ID)

	//
	// NOTEs:
	// 	- We ignore hosts that are passing all policies and do not have an associated email.
	//	- We get only one host per email that's failing policies (the one with lower host id).
	//	- On every host, we get only the first email that matches the domain (sorted lexicographically).
	//

	policyIDs := make([]uint, 0, len(policies))
	for _, policy := range policies {
		policyIDs = append(policyIDs, policy.ID)
	}
	hosts, err := ds.GetHostsPolicyMemberships(ctx, domain, policyIDs)
	if err != nil {
		return fmt.Errorf("get team hosts failing policies: %w", err)
	}

	var (
		passingHosts                       []fleet.HostPolicyMembershipData
		failingHosts                       []fleet.HostPolicyMembershipData
		failingHostsWithoutAssociatedEmail []fleet.HostPolicyMembershipData
	)
	for _, host := range hosts {
		if host.Passing { // host is passing all configured policies
			if host.Email != "" {
				passingHosts = append(passingHosts, host)
			}
		} else { // host is failing some of the configured policies
			if host.Email == "" {
				failingHostsWithoutAssociatedEmail = append(failingHostsWithoutAssociatedEmail, host)
			} else {
				failingHosts = append(failingHosts, host)
			}
		}
	}
	level.Debug(logger).Log(
		"msg", "summary",
		"passing_hosts", len(passingHosts),
		"failing_hosts", len(failingHosts),
		"failing_hosts_without_associated_email", len(failingHostsWithoutAssociatedEmail),
	)

	if err := processCalendarFailingHosts(
		ctx, ds, calendar, orgName, failingHosts, logger,
	); err != nil {
		level.Info(logger).Log("msg", "processing failing hosts", "err", err)
	}

	// Remove calendar events from hosts that are passing the policies.
	if err := removeCalendarEventsFromPassingHosts(ctx, ds, calendar, passingHosts); err != nil {
		level.Info(logger).Log("msg", "removing calendar events from passing hosts", "err", err)
	}

	// At last we want to log the hosts that are failing and don't have an associated email.
	logHostsWithoutAssociatedEmail(
		domain,
		failingHostsWithoutAssociatedEmail,
		logger,
	)

	return nil
}

func processCalendarFailingHosts(
	ctx context.Context,
	ds fleet.Datastore,
	userCalendar fleet.UserCalendar,
	orgName string,
	hosts []fleet.HostPolicyMembershipData,
	logger kitlog.Logger,
) error {
	for _, host := range hosts {
		logger := log.With(logger, "host_id", host.HostID)

		hostCalendarEvent, calendarEvent, err := ds.GetHostCalendarEvent(ctx, host.HostID)

		expiredEvent := false
		webhookAlreadyFiredThisMonth := false
		if err == nil {
			now := time.Now()
			webhookAlreadyFired := hostCalendarEvent.WebhookStatus == fleet.CalendarWebhookStatusSent
			if webhookAlreadyFired && sameDate(now, calendarEvent.StartTime) {
				// If the webhook already fired today and the policies are still failing
				// we give a grace period of one day for the host before we schedule a new event.
				continue // continue with next host
			}
			webhookAlreadyFiredThisMonth = webhookAlreadyFired && sameMonth(now, calendarEvent.StartTime)
			if calendarEvent.EndTime.Before(time.Now()) {
				expiredEvent = true
			}
		}

		if err := userCalendar.Configure(host.Email); err != nil {
			return fmt.Errorf("configure user calendar: %w", err)
		}

		switch {
		case err == nil && !expiredEvent:
			if err := processFailingHostExistingCalendarEvent(
				ctx, ds, userCalendar, orgName, hostCalendarEvent, calendarEvent, host,
			); err != nil {
				level.Info(logger).Log("msg", "process failing host existing calendar event", "err", err)
				continue // continue with next host
			}
		case fleet.IsNotFound(err) || expiredEvent:
			if err := processFailingHostCreateCalendarEvent(
				ctx, ds, userCalendar, orgName, host, webhookAlreadyFiredThisMonth,
			); err != nil {
				level.Info(logger).Log("msg", "process failing host create calendar event", "err", err)
				continue // continue with next host
			}
		default:
			return fmt.Errorf("get calendar event: %w", err)
		}
	}

	return nil
}

func processFailingHostExistingCalendarEvent(
	ctx context.Context,
	ds fleet.Datastore,
	calendar fleet.UserCalendar,
	orgName string,
	hostCalendarEvent *fleet.HostCalendarEvent,
	calendarEvent *fleet.CalendarEvent,
	host fleet.HostPolicyMembershipData,
) error {
	updatedEvent := calendarEvent
	updated := false
	now := time.Now()

	// Check the user calendar every 30 minutes (and not every time)
	// to reduce load on both Fleet and the calendar service.
	if time.Since(calendarEvent.UpdatedAt) > 30*time.Minute {
		var err error
		updatedEvent, _, err = calendar.GetAndUpdateEvent(calendarEvent, func(conflict bool) string {
			return generateCalendarEventBody(orgName, host.HostDisplayName, conflict)
		})
		if err != nil {
			return fmt.Errorf("get event calendar on db: %w", err)
		}
		// Even if fields haven't changed we want to update the calendar_events.updated_at below.
		updated = true
		//
		// TODO(lucas): Check changing updatedEvent to UTC before consuming.
		//
	}

	if updated {
		if err := ds.UpdateCalendarEvent(ctx,
			calendarEvent.ID,
			updatedEvent.StartTime,
			updatedEvent.EndTime,
			updatedEvent.Data,
		); err != nil {
			return fmt.Errorf("updating event calendar on db: %w", err)
		}
	}

	eventInFuture := now.Before(updatedEvent.StartTime)
	if eventInFuture {
		// Nothing else to do as event is in the future.
		return nil
	}
	if now.After(updatedEvent.EndTime) {
		return fmt.Errorf(
			"unexpected event in the past: now=%s, start_time=%s, end_time=%s",
			now, updatedEvent.StartTime, updatedEvent.EndTime,
		)
	}

	//
	// Event happening now.
	//

	if hostCalendarEvent.WebhookStatus == fleet.CalendarWebhookStatusSent {
		return nil
	}

	online, err := isHostOnline(ctx, ds, host.HostID)
	if err != nil {
		return fmt.Errorf("host online check: %w", err)
	}
	if !online {
		// If host is offline then there's nothing to do.
		return nil
	}

	if err := ds.UpdateHostCalendarWebhookStatus(ctx, host.HostID, fleet.CalendarWebhookStatusPending); err != nil {
		return fmt.Errorf("update host calendar webhook status: %w", err)
	}

	// TODO(lucas): If this doesn't work at scale, then implement a special refetch
	// for policies only.
	if err := ds.UpdateHostRefetchRequested(ctx, host.HostID, true); err != nil {
		return fmt.Errorf("refetch host: %w", err)
	}
	return nil
}

func sameDate(t1 time.Time, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func sameMonth(t1 time.Time, t2 time.Time) bool {
	y1, m1, _ := t1.Date()
	y2, m2, _ := t2.Date()
	return y1 == y2 && m1 == m2
}

func processFailingHostCreateCalendarEvent(
	ctx context.Context,
	ds fleet.Datastore,
	userCalendar fleet.UserCalendar,
	orgName string,
	host fleet.HostPolicyMembershipData,
	webhookAlreadyFiredThisMonth bool,
) error {
	calendarEvent, err := attemptCreatingEventOnUserCalendar(orgName, host, userCalendar, webhookAlreadyFiredThisMonth)
	if err != nil {
		return fmt.Errorf("create event on user calendar: %w", err)
	}
	if _, err := ds.CreateOrUpdateCalendarEvent(ctx, host.Email, calendarEvent.StartTime, calendarEvent.EndTime, calendarEvent.Data, host.HostID, fleet.CalendarWebhookStatusNone); err != nil {
		return fmt.Errorf("create calendar event on db: %w", err)
	}
	return nil
}

func attemptCreatingEventOnUserCalendar(
	orgName string,
	host fleet.HostPolicyMembershipData,
	userCalendar fleet.UserCalendar,
	webhookAlreadyFiredThisMonth bool,
) (*fleet.CalendarEvent, error) {
	year, month, today := time.Now().Date()
	preferredDate := getPreferredCalendarEventDate(year, month, today, webhookAlreadyFiredThisMonth)
	for {
		calendarEvent, err := userCalendar.CreateEvent(
			preferredDate, func(conflict bool) string {
				return generateCalendarEventBody(orgName, host.HostDisplayName, conflict)
			},
		)
		var dee fleet.DayEndedError
		switch {
		case err == nil:
			return calendarEvent, nil
		case errors.As(err, &dee):
			preferredDate = addBusinessDay(preferredDate)
			continue
		default:
			return nil, fmt.Errorf("create event on user calendar: %w", err)
		}
	}
}

func getPreferredCalendarEventDate(
	year int, month time.Month, today int,
	webhookAlreadyFired bool,
) time.Time {
	const (
		// 3rd Tuesday of Month
		preferredWeekDay = time.Tuesday
		preferredOrdinal = 3
	)

	firstDayOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	offset := int(preferredWeekDay - firstDayOfMonth.Weekday())
	if offset < 0 {
		offset += 7
	}
	preferredDate := firstDayOfMonth.AddDate(0, 0, offset+(7*(preferredOrdinal-1)))
	if today > preferredDate.Day() {
		today_ := time.Date(year, month, today, 0, 0, 0, 0, time.UTC)
		if webhookAlreadyFired {
			nextMonth := today_.AddDate(0, 1, 0) // move to next month
			return getPreferredCalendarEventDate(nextMonth.Year(), nextMonth.Month(), 1, false)
		}
		preferredDate = addBusinessDay(today_)
	}
	return preferredDate
}

func addBusinessDay(date time.Time) time.Time {
	nextBusinessDay := 1
	switch weekday := date.Weekday(); weekday {
	case time.Friday:
		nextBusinessDay += 2
	case time.Saturday:
		nextBusinessDay += 1
	}
	return date.AddDate(0, 0, nextBusinessDay)
}

func removeCalendarEventsFromPassingHosts(
	ctx context.Context,
	ds fleet.Datastore,
	userCalendar fleet.UserCalendar,
	hosts []fleet.HostPolicyMembershipData,
) error {
	for _, host := range hosts {
		calendarEvent, err := ds.GetCalendarEvent(ctx, host.Email)
		switch {
		case err == nil:
			// OK
		case fleet.IsNotFound(err):
			continue
		default:
			return fmt.Errorf("get calendar event from DB: %w", err)
		}
		if err := deleteCalendarEvent(ctx, ds, userCalendar, calendarEvent); err != nil {
			return fmt.Errorf("delete user calendar event: %w", err)
		}
	}
	return nil
}

func logHostsWithoutAssociatedEmail(
	domain string,
	hosts []fleet.HostPolicyMembershipData,
	logger kitlog.Logger,
) {
	if len(hosts) == 0 {
		return
	}
	var hostIDs []uint
	for _, host := range hosts {
		hostIDs = append(hostIDs, host.HostID)
	}
	// Logging as debug because this might get logged every 5 minutes.
	level.Debug(logger).Log(
		"msg", fmt.Sprintf("no %s Google account associated with the hosts", domain),
		"host_ids", fmt.Sprintf("%+v", hostIDs),
	)
}

func generateCalendarEventBody(orgName, hostDisplayName string, conflict bool) string {
	conflictStr := ""
	if conflict {
		conflictStr = " because there was no remaining availability"
	}
	return fmt.Sprintf(`Please leave your computer on and connected to power.

Expect an automated restart.

%s reserved this time to fix %s%s.`, orgName, hostDisplayName, conflictStr,
	)
}

func isHostOnline(ctx context.Context, ds fleet.Datastore, hostID uint) (bool, error) {
	hostLite, err := ds.HostLiteByID(ctx, hostID)
	if err != nil {
		return false, fmt.Errorf("get host lite: %w", err)
	}
	status := (&fleet.Host{
		DistributedInterval: hostLite.DistributedInterval,
		ConfigTLSRefresh:    hostLite.ConfigTLSRefresh,
		SeenTime:            hostLite.SeenTime,
	}).Status(time.Now())

	switch status {
	case fleet.StatusOnline, fleet.StatusNew:
		return true, nil
	case fleet.StatusOffline, fleet.StatusMIA, fleet.StatusMissing:
		return false, nil
	default:
		return false, fmt.Errorf("unknown host status: %s", status)
	}
}

func cronCalendarEventsCleanup(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return fmt.Errorf("load app config: %w", err)
	}

	var userCalendar fleet.UserCalendar
	if len(appConfig.Integrations.GoogleCalendar) > 0 {
		googleCalendarIntegrationConfig := appConfig.Integrations.GoogleCalendar[0]
		userCalendar = createUserCalendarFromConfig(ctx, googleCalendarIntegrationConfig, logger)
	}

	// If global setting is disabled, we remove all calendar events from the DB
	// (we cannot delete the events from the user calendar because there's no configuration anymore).
	if userCalendar == nil {
		if err := deleteAllCalendarEvents(ctx, ds, nil, nil); err != nil {
			return fmt.Errorf("delete all calendar events: %w", err)
		}
		// We've deleted all calendar events, nothing else to do.
		return nil
	}

	//
	// Feature is configured globally, but now we have to check team by team.
	//

	teams, err := ds.ListTeams(ctx, fleet.TeamFilter{
		User: &fleet.User{
			GlobalRole: ptr.String(fleet.RoleAdmin),
		},
	}, fleet.ListOptions{})
	if err != nil {
		return fmt.Errorf("list teams: %w", err)
	}

	for _, team := range teams {
		if err := deleteTeamCalendarEvents(ctx, ds, userCalendar, *team); err != nil {
			level.Info(logger).Log("msg", "delete team calendar events", "team_id", team.ID, "err", err)
		}
	}

	//
	// Delete calendar events from DB that haven't been updated for a while
	// (e.g. host was transferred to another team or global).
	//

	outOfDateCalendarEvents, err := ds.ListOutOfDateCalendarEvents(ctx, time.Now().Add(-48*time.Hour))
	if err != nil {
		return fmt.Errorf("list out of date calendar events: %w", err)
	}
	for _, outOfDateCalendarEvent := range outOfDateCalendarEvents {
		if err := deleteCalendarEvent(ctx, ds, userCalendar, outOfDateCalendarEvent); err != nil {
			return fmt.Errorf("delete user calendar event: %w", err)
		}
	}

	return nil
}

func deleteAllCalendarEvents(
	ctx context.Context,
	ds fleet.Datastore,
	userCalendar fleet.UserCalendar,
	teamID *uint,
) error {
	calendarEvents, err := ds.ListCalendarEvents(ctx, teamID)
	if err != nil {
		return fmt.Errorf("list calendar events: %w", err)
	}
	for _, calendarEvent := range calendarEvents {
		if err := deleteCalendarEvent(ctx, ds, userCalendar, calendarEvent); err != nil {
			return fmt.Errorf("delete user calendar event: %w", err)
		}
	}
	return nil
}

func deleteTeamCalendarEvents(
	ctx context.Context,
	ds fleet.Datastore,
	userCalendar fleet.UserCalendar,
	team fleet.Team,
) error {
	if team.Config.Integrations.GoogleCalendar != nil &&
		team.Config.Integrations.GoogleCalendar.Enable {
		// Feature is enabled, nothing to cleanup.
		return nil
	}
	return deleteAllCalendarEvents(ctx, ds, userCalendar, &team.ID)
}

func deleteCalendarEvent(ctx context.Context, ds fleet.Datastore, userCalendar fleet.UserCalendar, calendarEvent *fleet.CalendarEvent) error {
	if userCalendar != nil {
		// Only delete events from the user's calendar if the event is in the future.
		if eventInFuture := time.Now().Before(calendarEvent.StartTime); eventInFuture {
			if err := userCalendar.Configure(calendarEvent.Email); err != nil {
				return fmt.Errorf("connect to user calendar: %w", err)
			}
			if err := userCalendar.DeleteEvent(calendarEvent); err != nil {
				return fmt.Errorf("delete calendar event: %w", err)
			}
		}
	}
	if err := ds.DeleteCalendarEvent(ctx, calendarEvent.ID); err != nil {
		return fmt.Errorf("delete db calendar event: %w", err)
	}
	return nil
}
