package cron

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/calendar"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/go-kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const calendarConsumers = 18
const defaultDescription = "needs to make sure your device meets the organization's requirements."
const defaultResolution = "During this maintenance window, you can expect updates to be applied automatically. Your device may be unavailable during this time."

func NewCalendarSchedule(
	ctx context.Context,
	instanceID string,
	ds fleet.Datastore,
	interval time.Duration,
	logger kitlog.Logger,
) (*schedule.Schedule, error) {
	const (
		name = string(fleet.CronCalendar)
	)
	logger = kitlog.With(logger, "cron", name)
	s := schedule.New(
		ctx, name, instanceID, interval, ds, ds,
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
	domain := googleCalendarIntegrationConfig.Domain

	teams, err := ds.ListTeams(
		ctx, fleet.TeamFilter{
			User: &fleet.User{
				GlobalRole: ptr.String(fleet.RoleAdmin),
			},
		}, fleet.ListOptions{},
	)
	if err != nil {
		return fmt.Errorf("list teams: %w", err)
	}

	for _, team := range teams {
		if err := cronCalendarEventsForTeam(
			ctx, ds, googleCalendarIntegrationConfig, *team, appConfig.OrgInfo.OrgName, domain, logger,
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
	calendarConfig *fleet.GoogleCalendarIntegration,
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
		level.Debug(logger).Log(
			"msg", "skipping, no calendar policies",
			"team_id", team.ID,
		)
		return nil
	}

	logger = kitlog.With(logger, "team_id", team.ID)

	//
	// NOTEs:
	// 	- We ignore hosts that are passing all policies and do not have an associated email.
	//	- We get only one host per email that's failing policies (the one with lower host id).
	//	- On every host, we get only the first email that matches the domain (sorted lexicographically).
	//	- GetTeamHostsPolicyMemberships returns the hosts that are passing all policies and have a calendar event.
	//

	policyIDs := make([]uint, 0, len(policies))
	for _, policy := range policies {
		policyIDs = append(policyIDs, policy.ID)
	}
	hosts, err := ds.GetTeamHostsPolicyMemberships(ctx, domain, team.ID, policyIDs)
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
		"team_id", team.ID,
		"passing_hosts", len(passingHosts),
		"failing_hosts", len(failingHosts),
		"failing_hosts_without_associated_email", len(failingHostsWithoutAssociatedEmail),
	)

	// Remove calendar events from hosts that are passing the calendar policies.
	//
	// We execute this first to remove any calendar events for a user that is now passing
	// policies on one of its hosts, and possibly create a new calendar event if they have
	// another failing host on the same team.
	start := time.Now()
	removeCalendarEventsFromPassingHosts(ctx, ds, calendarConfig, passingHosts, logger)
	level.Debug(logger).Log(
		"msg", "passing_hosts", "took", time.Since(start),
	)

	// Process hosts that are failing calendar policies.
	start = time.Now()
	processCalendarFailingHosts(ctx, ds, calendarConfig, orgName, failingHosts, logger)
	level.Debug(logger).Log(
		"msg", "failing_hosts", "took", time.Since(start),
	)

	// At last, we want to log the hosts that are failing and don't have an associated email.
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
	calendarConfig *fleet.GoogleCalendarIntegration,
	orgName string,
	hosts []fleet.HostPolicyMembershipData,
	logger kitlog.Logger,
) {
	hosts = filterHostsWithSameEmail(hosts)

	hostsCh := make(chan fleet.HostPolicyMembershipData)
	var wg sync.WaitGroup
	// policyIDtoPolicy maps policy IDs to policies.
	// It is a cache to avoid querying the database for each host.
	policyIDtoPolicy := sync.Map{}

	for i := 0; i < calendarConsumers; i++ {
		wg.Add(+1)
		go func() {
			defer wg.Done()

			for host := range hostsCh {
				logger := log.With(logger, "host_id", host.HostID)

				hostCalendarEvent, calendarEvent, err := ds.GetHostCalendarEventByEmail(ctx, host.Email)

				expiredEvent := false
				if err == nil {
					if hostCalendarEvent.HostID != host.HostID {
						// This calendar event belongs to another host with this associated email,
						// thus we skip this entry.
						continue // continue with next host
					}
					if hostCalendarEvent.WebhookStatus == fleet.CalendarWebhookStatusPending || hostCalendarEvent.WebhookStatus == fleet.CalendarWebhookStatusRetry {
						// This can happen if the host went offline (and never returned results)
						// after setting the webhook as pending.
						continue // continue with next host
					}
					now := time.Now()
					webhookAlreadyFired := hostCalendarEvent.WebhookStatus == fleet.CalendarWebhookStatusSent
					if webhookAlreadyFired && sameDate(now, calendarEvent.StartTime) {
						// If the webhook already fired today and the policies are still failing
						// we give a grace period of one day for the host before we schedule a new event.
						continue // continue with next host
					}
					if calendarEvent.EndTime.Before(now) {
						expiredEvent = true
					}
				}

				userCalendar := createUserCalendarFromConfig(ctx, calendarConfig, logger)
				if err := userCalendar.Configure(host.Email); err != nil {
					level.Error(logger).Log("msg", "configure user calendar", "err", err)
					continue // continue with next host
				}

				switch {
				case err == nil && !expiredEvent:
					if err := processFailingHostExistingCalendarEvent(
						ctx, ds, userCalendar, orgName, hostCalendarEvent, calendarEvent, host, &policyIDtoPolicy, logger,
					); err != nil {
						level.Info(logger).Log("msg", "process failing host existing calendar event", "err", err)
						continue // continue with next host
					}
				case fleet.IsNotFound(err) || expiredEvent:
					if err := processFailingHostCreateCalendarEvent(
						ctx, ds, userCalendar, orgName, host, &policyIDtoPolicy, logger,
					); err != nil {
						level.Info(logger).Log("msg", "process failing host create calendar event", "err", err)
						continue // continue with next host
					}
				default:
					level.Error(logger).Log("msg", "get calendar event from db", "err", err)
					continue // continue with next host
				}
			}
		}()
	}

	for _, host := range hosts {
		hostsCh <- host
	}
	close(hostsCh)

	wg.Wait()
}

func filterHostsWithSameEmail(hosts []fleet.HostPolicyMembershipData) []fleet.HostPolicyMembershipData {
	minHostPerEmail := make(map[string]fleet.HostPolicyMembershipData)
	for _, host := range hosts {
		minHost, ok := minHostPerEmail[host.Email]
		if !ok {
			minHostPerEmail[host.Email] = host
			continue
		}
		if host.HostID < minHost.HostID {
			minHostPerEmail[host.Email] = host
		}
	}
	filtered := make([]fleet.HostPolicyMembershipData, 0, len(minHostPerEmail))
	for _, host := range minHostPerEmail {
		filtered = append(filtered, host)
	}
	return filtered
}

func processFailingHostExistingCalendarEvent(
	ctx context.Context,
	ds fleet.Datastore,
	calendar fleet.UserCalendar,
	orgName string,
	hostCalendarEvent *fleet.HostCalendarEvent,
	calendarEvent *fleet.CalendarEvent,
	host fleet.HostPolicyMembershipData,
	policyIDtoPolicy *sync.Map,
	logger kitlog.Logger,
) error {
	updatedEvent := calendarEvent
	updated := false
	now := time.Now()

	if shouldReloadCalendarEvent(now, calendarEvent, hostCalendarEvent) {
		var err error
		updatedEvent, _, err = calendar.GetAndUpdateEvent(
			calendarEvent, func(conflict bool) string {
				return generateCalendarEventBody(ctx, ds, orgName, host, policyIDtoPolicy, conflict, logger)
			},
		)
		if err != nil {
			return fmt.Errorf("get event calendar on db: %w", err)
		}
		// Even if fields haven't changed we want to update the calendar_events.updated_at below.
		updated = true
	}

	if updated {
		if err := ds.UpdateCalendarEvent(
			ctx,
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

	if err := ds.UpdateHostRefetchRequested(ctx, host.HostID, true); err != nil {
		return fmt.Errorf("refetch host: %w", err)
	}
	return nil
}

func shouldReloadCalendarEvent(now time.Time, calendarEvent *fleet.CalendarEvent, hostCalendarEvent *fleet.HostCalendarEvent) bool {
	// Check the user calendar every 30 minutes (and not every cron run)
	// to reduce load on both Fleet and the calendar service.
	if time.Since(calendarEvent.UpdatedAt) > 30*time.Minute {
		return true
	}
	// If the event is supposed to be happening now, we want to check if the user moved/deleted the
	// event on the last minute.
	if eventHappeningNow(now, calendarEvent) && hostCalendarEvent.WebhookStatus == fleet.CalendarWebhookStatusNone {
		return true
	}
	return false
}

func eventHappeningNow(now time.Time, calendarEvent *fleet.CalendarEvent) bool {
	return !now.Before(calendarEvent.StartTime) && now.Before(calendarEvent.EndTime)
}

func sameDate(t1 time.Time, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func processFailingHostCreateCalendarEvent(
	ctx context.Context,
	ds fleet.Datastore,
	userCalendar fleet.UserCalendar,
	orgName string,
	host fleet.HostPolicyMembershipData,
	policyIDtoPolicy *sync.Map,
	logger kitlog.Logger,
) error {
	calendarEvent, err := attemptCreatingEventOnUserCalendar(ctx, ds, orgName, host, userCalendar, policyIDtoPolicy, logger)
	if err != nil {
		return fmt.Errorf("create event on user calendar: %w", err)
	}
	if _, err := ds.CreateOrUpdateCalendarEvent(
		ctx, host.Email, calendarEvent.StartTime, calendarEvent.EndTime, calendarEvent.Data, host.HostID, fleet.CalendarWebhookStatusNone,
	); err != nil {
		return fmt.Errorf("create calendar event on db: %w", err)
	}
	return nil
}

func attemptCreatingEventOnUserCalendar(
	ctx context.Context,
	ds fleet.Datastore,
	orgName string,
	host fleet.HostPolicyMembershipData,
	userCalendar fleet.UserCalendar,
	policyIDtoPolicy *sync.Map,
	logger kitlog.Logger,
) (*fleet.CalendarEvent, error) {
	year, month, today := time.Now().Date()
	preferredDate := getPreferredCalendarEventDate(year, month, today)
	for {
		calendarEvent, err := userCalendar.CreateEvent(
			preferredDate, func(conflict bool) string {
				return generateCalendarEventBody(ctx, ds, orgName, host, policyIDtoPolicy, conflict, logger)
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
		// We are past the preferred date, so we move to next month and calculate again.
		month := month + 1
		if month == 13 {
			month = 1
			year += 1
		}
		return getPreferredCalendarEventDate(year, month, 1)
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
	default:
		// nextBusinessDay is 1
	}
	return date.AddDate(0, 0, nextBusinessDay)
}

func removeCalendarEventsFromPassingHosts(
	ctx context.Context,
	ds fleet.Datastore,
	calendarConfig *fleet.GoogleCalendarIntegration,
	hosts []fleet.HostPolicyMembershipData,
	logger kitlog.Logger,
) {
	hostIDsByEmail := make(map[string][]uint)
	for _, host := range hosts {
		hostIDsByEmail[host.Email] = append(hostIDsByEmail[host.Email], host.HostID)
	}
	type emailWithHosts struct {
		email   string
		hostIDs []uint
	}
	emails := make([]emailWithHosts, 0, len(hostIDsByEmail))
	for email, hostIDs := range hostIDsByEmail {
		emails = append(
			emails, emailWithHosts{
				email:   email,
				hostIDs: hostIDs,
			},
		)
	}

	emailsCh := make(chan emailWithHosts)
	var wg sync.WaitGroup

	for i := 0; i < calendarConsumers; i++ {
		wg.Add(+1)
		go func() {
			defer wg.Done()

			for email := range emailsCh {

				hostCalendarEvent, calendarEvent, err := ds.GetHostCalendarEventByEmail(ctx, email.email)
				switch {
				case err == nil:
					if ok := slices.Contains(email.hostIDs, hostCalendarEvent.HostID); !ok {
						// None of the hosts belong to this calendar event.
						continue
					}
				case fleet.IsNotFound(err):
					continue
				default:
					level.Error(logger).Log("msg", "get calendar event from DB", "err", err)
					continue
				}
				userCalendar := createUserCalendarFromConfig(ctx, calendarConfig, logger)
				if err := deleteCalendarEvent(ctx, ds, userCalendar, calendarEvent); err != nil {
					level.Error(logger).Log("msg", "delete user calendar event", "err", err)
					continue
				}
			}
		}()
	}

	for _, emailWithHostIDs := range emails {
		emailsCh <- emailWithHostIDs
	}
	close(emailsCh)

	wg.Wait()
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

func generateCalendarEventBody(
	ctx context.Context, ds fleet.Datastore, orgName string, host fleet.HostPolicyMembershipData, policyIDtoPolicy *sync.Map, conflict bool,
	logger kitlog.Logger,
) string {
	description, resolution := getCalendarEventDescriptionAndResolution(ctx, ds, orgName, host, policyIDtoPolicy, logger)

	conflictStr := ""
	if conflict {
		conflictStr = " because there was no remaining availability"
	}
	return fmt.Sprintf(
		`%s reserved this time to make some changes to your work computer%s.

Please leave your device on and connected to power.

<b>Why it matters</b>
%s

<b>What we'll do</b>
%s
`,
		orgName, conflictStr, description, resolution,
	)
}

func getCalendarEventDescriptionAndResolution(
	ctx context.Context, ds fleet.Datastore, orgName string, host fleet.HostPolicyMembershipData, policyIDtoPolicy *sync.Map,
	logger kitlog.Logger,
) (string, string) {
	getDefaultDescription := func() string {
		return fmt.Sprintf(`%s %s`, orgName, defaultDescription)
	}

	var description, resolution string
	policyIDs := strings.Split(host.FailingPolicyIDs, ",")
	if len(policyIDs) == 1 && policyIDs[0] != "" {
		var policy *fleet.PolicyLite
		policyAny, ok := policyIDtoPolicy.Load(policyIDs[0])
		if !ok {
			id, err := strconv.ParseUint(policyIDs[0], 10, 64)
			if err != nil {
				level.Error(logger).Log("msg", "parse policy id", "err", err)
				return getDefaultDescription(), defaultResolution
			}
			policy, err = ds.PolicyLite(ctx, uint(id))
			if err != nil {
				level.Error(logger).Log("msg", "get policy", "err", err)
				return getDefaultDescription(), defaultResolution
			}
			policyIDtoPolicy.Store(policyIDs[0], policy)
		} else {
			policy = policyAny.(*fleet.PolicyLite)
		}
		policyDescription := strings.TrimSpace(policy.Description)
		if policyDescription == "" || policy.Resolution == nil || strings.TrimSpace(*policy.Resolution) == "" {
			description = getDefaultDescription()
			resolution = defaultResolution
		} else {
			description = policyDescription
			resolution = strings.TrimSpace(*policy.Resolution)
		}
	} else {
		description = getDefaultDescription()
		resolution = defaultResolution
	}
	return description, resolution
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
	var calendarConfig *fleet.GoogleCalendarIntegration
	if len(appConfig.Integrations.GoogleCalendar) > 0 {
		calendarConfig = appConfig.Integrations.GoogleCalendar[0]
		userCalendar = createUserCalendarFromConfig(ctx, calendarConfig, logger)
	}

	// If global setting is disabled, we remove all calendar events from the DB
	// (we cannot delete the events from the user calendar because there's no configuration anymore).
	if userCalendar == nil {
		if err := deleteAllCalendarEvents(ctx, ds, nil, nil, logger); err != nil {
			return fmt.Errorf("delete all calendar events: %w", err)
		}
		// We've deleted all calendar events, nothing else to do.
		return nil
	}

	//
	// Feature is configured globally, but now we have to check team by team.
	//

	teams, err := ds.ListTeams(
		ctx, fleet.TeamFilter{
			User: &fleet.User{
				GlobalRole: ptr.String(fleet.RoleAdmin),
			},
		}, fleet.ListOptions{},
	)
	if err != nil {
		return fmt.Errorf("list teams: %w", err)
	}

	for _, team := range teams {
		if err := cleanupTeamCalendarEvents(ctx, ds, calendarConfig, *team, logger); err != nil {
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
	deleteCalendarEventsInParallel(ctx, ds, calendarConfig, outOfDateCalendarEvents, logger)
	return nil
}

func deleteAllCalendarEvents(
	ctx context.Context,
	ds fleet.Datastore,
	calendarConfig *fleet.GoogleCalendarIntegration,
	teamID *uint,
	logger kitlog.Logger,
) error {
	calendarEvents, err := ds.ListCalendarEvents(ctx, teamID)
	if err != nil {
		return fmt.Errorf("list calendar events: %w", err)
	}
	deleteCalendarEventsInParallel(ctx, ds, calendarConfig, calendarEvents, logger)
	return nil
}

func deleteCalendarEventsInParallel(
	ctx context.Context, ds fleet.Datastore, calendarConfig *fleet.GoogleCalendarIntegration, calendarEvents []*fleet.CalendarEvent,
	logger kitlog.Logger,
) {
	if len(calendarEvents) > 0 {
		calendarEventCh := make(chan *fleet.CalendarEvent)
		var wg sync.WaitGroup
		for i := 0; i < calendarConsumers; i++ {
			wg.Add(+1)
			go func() {
				defer wg.Done()
				for calEvent := range calendarEventCh {
					var userCalendar fleet.UserCalendar
					if calendarConfig != nil {
						userCalendar = createUserCalendarFromConfig(ctx, calendarConfig, logger)
					}
					if err := deleteCalendarEvent(ctx, ds, userCalendar, calEvent); err != nil {
						level.Error(logger).Log("msg", "delete user calendar event", "err", err)
						continue
					}
				}
			}()
		}
		for _, outOfDateCalendarEvent := range calendarEvents {
			calendarEventCh <- outOfDateCalendarEvent
		}
		close(calendarEventCh)
		wg.Wait()
	}
}

func cleanupTeamCalendarEvents(
	ctx context.Context,
	ds fleet.Datastore,
	calendarConfig *fleet.GoogleCalendarIntegration,
	team fleet.Team,
	logger kitlog.Logger,
) error {
	teamFeatureEnabled := team.Config.Integrations.GoogleCalendar != nil && team.Config.Integrations.GoogleCalendar.Enable

	if teamFeatureEnabled {
		policies, err := ds.GetCalendarPolicies(ctx, team.ID)
		if err != nil {
			return fmt.Errorf("get calendar policy ids: %w", err)
		}
		if len(policies) > 0 {
			// Feature is enabled and there are calendar policies configured, so nothing to do.
			return nil
		}
		// Feature is enabled but there are no calendar policies,
		// so we want to clean up all calendar events for the team.
	}

	return deleteAllCalendarEvents(ctx, ds, calendarConfig, &team.ID, logger)
}

func deleteCalendarEvent(
	ctx context.Context, ds fleet.Datastore, userCalendar fleet.UserCalendar, calendarEvent *fleet.CalendarEvent,
) error {
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
