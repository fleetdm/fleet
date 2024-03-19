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
	googleCalendarConfig := calendar.GoogleCalendarConfig{
		Context:           ctx,
		IntegrationConfig: googleCalendarIntegrationConfig,
		Logger:            log.With(logger, "component", "google_calendar"),
	}
	calendar := calendar.NewGoogleCalendar(&googleCalendarConfig)
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
	// TODOs(lucas):
	//	- We need to rate limit calendar requests.
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

	// At last we want to notify the hosts that are failing and don't have an associated email.
	if err := fireWebhookForHostsWithoutAssociatedEmail(
		team.Config.Integrations.GoogleCalendar.WebhookURL,
		domain,
		failingHostsWithoutAssociatedEmail,
		logger,
	); err != nil {
		level.Info(logger).Log("msg", "webhook for hosts without associated email", "err", err)
	}

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
		if err := userCalendar.Configure(host.Email); err != nil {
			return fmt.Errorf("configure user calendar: %w", err)
		}

		hostCalendarEvent, calendarEvent, err := ds.GetHostCalendarEvent(ctx, host.HostID)

		deletedExpiredEvent := false
		if err == nil {
			if calendarEvent.EndTime.Before(time.Now()) {
				if err := ds.DeleteCalendarEvent(ctx, calendarEvent.ID); err != nil {
					level.Info(logger).Log("msg", "deleting existing expired calendar event", "err", err)
					continue // continue with next host
				}
				deletedExpiredEvent = true
			}
		}

		switch {
		case err == nil && !deletedExpiredEvent:
			if err := processFailingHostExistingCalendarEvent(
				ctx, ds, userCalendar, orgName, hostCalendarEvent, calendarEvent, host,
			); err != nil {
				level.Info(logger).Log("msg", "process failing host existing calendar event", "err", err)
				continue // continue with next host
			}
		case fleet.IsNotFound(err) || deletedExpiredEvent:
			if err := processFailingHostCreateCalendarEvent(
				ctx, ds, userCalendar, orgName, host,
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
	updatedEvent, updated, err := calendar.GetAndUpdateEvent(calendarEvent, func() string {
		return generateCalendarEventBody(orgName, host.HostDisplayName)
	})
	if err != nil {
		return fmt.Errorf("get event calendar on db: %w", err)
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
	now := time.Now()
	eventInFuture := now.Before(updatedEvent.StartTime)
	if eventInFuture {
		// If the webhook status was sent and event was moved to the future we set the status to pending.
		// This can happen if the admin wants to retry a remediation.
		if hostCalendarEvent.WebhookStatus == fleet.CalendarWebhookStatusSent {
			if err := ds.UpdateHostCalendarWebhookStatus(ctx, host.HostID, fleet.CalendarWebhookStatusPending); err != nil {
				return fmt.Errorf("update host calendar webhook status: %w", err)
			}
		}
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

func processFailingHostCreateCalendarEvent(
	ctx context.Context,
	ds fleet.Datastore,
	userCalendar fleet.UserCalendar,
	orgName string,
	host fleet.HostPolicyMembershipData,
) error {
	calendarEvent, err := attemptCreatingEventOnUserCalendar(orgName, host, userCalendar)
	if err != nil {
		return fmt.Errorf("create event on user calendar: %w", err)
	}
	if _, err := ds.NewCalendarEvent(ctx, host.Email, calendarEvent.StartTime, calendarEvent.EndTime, calendarEvent.Data, host.HostID); err != nil {
		return fmt.Errorf("create calendar event on db: %w", err)
	}
	return nil
}

func attemptCreatingEventOnUserCalendar(
	orgName string,
	host fleet.HostPolicyMembershipData,
	userCalendar fleet.UserCalendar,
) (*fleet.CalendarEvent, error) {
	// TODO(lucas): Where do we handle the following case (it seems CreateEvent needs to return no slot available for the requested day if there are none or too late):
	//
	// - If it’s the 3rd Tuesday of the month, create an event in the upcoming slot (if available).
	// For example, if it’s the 3rd Tuesday of the month at 10:07a, Fleet will look for an open slot starting at 10:30a.
	// - If it’s the 3rd Tuesday, Weds, Thurs, etc. of the month and it’s past the last slot, schedule the call for the next business day.
	year, month, today := time.Now().Date()
	preferredDate := getPreferredCalendarEventDate(year, month, today)
	body := generateCalendarEventBody(orgName, host.HostDisplayName)
	for {
		calendarEvent, err := userCalendar.CreateEvent(preferredDate, body)
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

func getPreferredCalendarEventDate(year int, month time.Month, today int) time.Time {
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
	calendar fleet.UserCalendar,
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

		if err := ds.DeleteCalendarEvent(ctx, calendarEvent.ID); err != nil {
			return fmt.Errorf("delete db calendar event: %w", err)
		}
		if err := calendar.Configure(host.Email); err != nil {
			return fmt.Errorf("connect to user calendar: %w", err)
		}
		if err := calendar.DeleteEvent(calendarEvent); err != nil {
			return fmt.Errorf("delete calendar event: %w", err)
		}
	}
	return nil
}

func fireWebhookForHostsWithoutAssociatedEmail(
	webhookURL string,
	domain string,
	hosts []fleet.HostPolicyMembershipData,
	logger kitlog.Logger,
) error {
	// TODO(lucas): We are firing these every 5 minutes...
	for _, host := range hosts {
		if err := fleet.FireCalendarWebhook(
			webhookURL,
			host.HostID, host.HostHardwareSerial, host.HostDisplayName, nil,
			fmt.Sprintf("No %s Google account associated with this host.", domain),
		); err != nil {
			level.Error(logger).Log(
				"msg", "fire webhook for hosts without associated email", "err", err,
			)
		}
	}
	return nil
}

func generateCalendarEventBody(orgName, hostDisplayName string) string {
	return fmt.Sprintf(`Please leave your computer on and connected to power.

Expect an automated restart.

%s reserved this time to fix %s.`, orgName, hostDisplayName,
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
