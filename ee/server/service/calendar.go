package service

import (
	"context"
	"fmt"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/calendar"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"sync"
)

const (
	calendarLockKeyPrefix         = "calendar:lock:"
	calendarReservedLockKeyPrefix = "calendar:reserved:"
	calendarQueueKey              = "calendar:queue"
)

var asyncCalendarProcessing bool
var asyncMutex sync.Mutex

func (svc *Service) CalendarWebhook(ctx context.Context, eventUUID string, channelID string, resourceState string) error {

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return fmt.Errorf("load app config: %w", err)
	}

	if len(appConfig.Integrations.GoogleCalendar) == 0 {
		svc.authz.SkipAuthorization(ctx)
		level.Warn(svc.logger).Log("msg", "Received calendar callback, but Google Calendar integration is not configured")
		return nil
	}
	googleCalendarIntegrationConfig := appConfig.Integrations.GoogleCalendar[0]

	fmt.Printf("VICTOR callback - eventUUID: %s, channelID: %s\n", eventUUID, channelID)

	if resourceState == "sync" {
		// This is a sync notification, not a real event
		svc.authz.SkipAuthorization(ctx)
		return nil
	}

	eventDetails, err := svc.ds.GetCalendarEventDetailsByUUID(ctx, eventUUID)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		if fleet.IsNotFound(err) {
			// We could try to stop the channel callbacks here, but that may not be secure since we don't know if the request is legitimate
			level.Warn(svc.logger).Log("msg", "Received calendar callback, but did not find corresponding event in database", "event_uuid",
				eventUUID, "channel_id", channelID)
			return err
		}
		return err
	}
	if eventDetails.TeamID == nil {
		// Should not happen
		return fmt.Errorf("calendar event %s has no team ID", eventUUID)
	}

	localConfig := &calendar.CalendarConfig{
		GoogleCalendarIntegration: *googleCalendarIntegrationConfig,
		ServerURL:                 appConfig.ServerSettings.ServerURL,
	}
	userCalendar := calendar.CreateUserCalendarFromConfig(ctx, localConfig, svc.logger)

	// Authenticate request. We will use the channel ID for authentication.
	svc.authz.SkipAuthorization(ctx)
	savedChannelID, err := userCalendar.Get(&eventDetails.CalendarEvent, "channelID")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get channel ID")
	}
	if savedChannelID != channelID {
		return authz.ForbiddenWithInternal(fmt.Sprintf("calendar channel ID mismatch: %s != %s", savedChannelID, channelID), nil, nil, nil)
	}

	lockValue, err := svc.getCalendarLock(ctx, eventUUID, true)
	if err != nil {
		return err
	}
	if lockValue == "" {
		// We did not get a lock, so there is nothing to do here
		return nil
	}

	unlocked := false
	defer func() {
		if !unlocked {
			svc.releaseCalendarLock(ctx, eventUUID, lockValue)
		}
	}()

	err = svc.processCalendarEvent(ctx, eventDetails, googleCalendarIntegrationConfig, userCalendar)
	if err != nil {
		return err
	}
	svc.releaseCalendarLock(ctx, eventUUID, lockValue)
	unlocked = true

	// Now, we need to check if there are any events in the queue that need to be re-processed.
	asyncMutex.Lock()
	defer asyncMutex.Unlock()
	if !asyncCalendarProcessing {
		eventIDs, err := svc.distributedLock.GetSet(ctx, calendarQueueKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get calendar event queue")
		}
		if len(eventIDs) > 0 {
			asyncCalendarProcessing = true
			go svc.processCalendarAsync(context.WithoutCancel(ctx), eventIDs)
		}
		return nil
	}

	return nil
}

func (svc *Service) processCalendarEvent(ctx context.Context, eventDetails *fleet.CalendarEventDetails,
	googleCalendarIntegrationConfig *fleet.GoogleCalendarIntegration, userCalendar fleet.UserCalendar) error {

	genBodyFn := func(conflict bool) (body string, ok bool, err error) {

		// This function is called when a new event is being created.
		var team *fleet.Team
		team, err = svc.ds.TeamWithoutExtras(ctx, *eventDetails.TeamID)
		if err != nil {
			return "", false, err
		}

		if team.Config.Integrations.GoogleCalendar == nil ||
			!team.Config.Integrations.GoogleCalendar.Enable {
			return "", false, nil
		}

		var policies []fleet.PolicyCalendarData
		policies, err = svc.ds.GetCalendarPolicies(ctx, team.ID)
		if err != nil {
			return "", false, err
		}

		if len(policies) == 0 {
			return "", false, nil
		}

		policyIDs := make([]uint, 0, len(policies))
		for _, policy := range policies {
			policyIDs = append(policyIDs, policy.ID)
		}

		var hosts []fleet.HostPolicyMembershipData
		hosts, err = svc.ds.GetTeamHostsPolicyMemberships(ctx, googleCalendarIntegrationConfig.Domain, team.ID, policyIDs,
			&eventDetails.HostID)
		if err != nil {
			return "", false, err
		}
		if len(hosts) != 1 {
			return "", false, nil
		}
		host := hosts[0]
		if host.Passing { // host is passing all configured policies
			return "", false, nil
		}
		if host.Email == "" {
			err = fmt.Errorf("host %d has no associated email", host.HostID)
			return "", false, err
		}

		return calendar.GenerateCalendarEventBody(ctx, svc.ds, team.Name, host, &sync.Map{}, conflict, svc.logger), true, nil
	}

	err := userCalendar.Configure(eventDetails.Email)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "configure calendar")
	}
	event, updated, err := userCalendar.GetAndUpdateEvent(&eventDetails.CalendarEvent, genBodyFn)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get and update event")
	}
	if updated && event != nil {
		// Event was updated, so we need to save it
		_, err = svc.ds.CreateOrUpdateCalendarEvent(ctx, event.UUID, event.Email, event.StartTime, event.EndTime, event.Data,
			event.TimeZone, eventDetails.ID, fleet.CalendarWebhookStatusNone)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "create or update calendar event")
		}
	}

	return nil
}

func (svc *Service) releaseCalendarLock(ctx context.Context, eventUUID string, lockValue string) {
	ok, err := svc.distributedLock.ReleaseLock(ctx, calendarLockKeyPrefix+eventUUID, lockValue)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to release calendar lock", "err", err)
	}
	if !ok {
		// If the lock was not released, it will expire on its own.
		level.Warn(svc.logger).Log("msg", "Failed to release calendar lock")
	}
}

func (svc *Service) getCalendarLock(ctx context.Context, eventUUID string, addToQueue bool) (string, error) {
	// Check if lock has been reserved, which means we can't have it.
	reserved, err := svc.distributedLock.Get(ctx, calendarReservedLockKeyPrefix+eventUUID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get calendar reserved lock")
	}
	if reserved != nil {
		// We assume that the lock is reserved by cron, which will fully process this event. Nothing to do here.
		return "", nil
	}
	// Try to acquire the lock
	lockValue := uuid.New().String()
	result, err := svc.distributedLock.AcquireLock(ctx, calendarLockKeyPrefix+eventUUID, lockValue, 0)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "acquire calendar lock")
	}
	if result == "" && addToQueue {
		// Could not acquire lock, so we are already processing this event. In this case, we add the event to
		// the queue (actually a set) to indicate that we need to re-process the event.
		err = svc.distributedLock.AddToSet(ctx, calendarQueueKey, eventUUID)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "add calendar event to queue")
		}

		// Try to acquire the lock again in case it was released while we were adding the event to the queue.
		result, err = svc.distributedLock.AcquireLock(ctx, calendarLockKeyPrefix+eventUUID, lockValue, 0)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "acquire calendar lock again")
		}

		if result == "" {
			// We could not acquire the lock, so we are done here.
			return "", nil
		}
	}
	return lockValue, nil
}

func (svc *Service) processCalendarAsync(ctx context.Context, eventIDs []string) {
	defer func() {
		asyncMutex.Lock()
		asyncCalendarProcessing = false
		asyncMutex.Unlock()
	}()
	for {
		if len(eventIDs) == 0 {
			return
		}
		for _, eventUUID := range eventIDs {
			if ok := svc.processCalendarEventAsync(ctx, eventUUID); !ok {
				return
			}
		}

		// Now we check whether there are any more events in the queue.
		var err error
		eventIDs, err = svc.distributedLock.GetSet(ctx, calendarQueueKey)
		if err != nil {
			level.Error(svc.logger).Log("msg", "Failed to get calendar event queue", "err", err)
			return
		}
	}
}

func (svc *Service) processCalendarEventAsync(ctx context.Context, eventUUID string) bool {
	lockValue, err := svc.getCalendarLock(ctx, eventUUID, false)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to get calendar lock", "err", err)
		return false
	}
	if lockValue == "" {
		// We did not get a lock, so there is nothing to do here
		return true
	}
	defer svc.releaseCalendarLock(ctx, eventUUID, lockValue)

	// Remove event from the queue so that we don't process this event again.
	// Note: This item can be added back to the queue while we are processing it.
	err = svc.distributedLock.RemoveFromSet(ctx, calendarQueueKey, eventUUID)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to remove calendar event from queue", "err", err)
		return false
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to load app config", "err", err)
		return false
	}

	if len(appConfig.Integrations.GoogleCalendar) == 0 {
		// Google Calendar integration is not configured
		return true
	}
	googleCalendarIntegrationConfig := appConfig.Integrations.GoogleCalendar[0]

	eventDetails, err := svc.ds.GetCalendarEventDetailsByUUID(ctx, eventUUID)
	if err != nil {
		if fleet.IsNotFound(err) {
			// We found this event when the callback initially came in. So the event may have been removed from DB since then.
			return true
		}
		level.Error(svc.logger).Log("msg", "Failed to get calendar event details", "err", err)
		return false
	}
	if eventDetails.TeamID == nil {
		// Should not happen
		level.Error(svc.logger).Log("msg", "Calendar event has no team ID", "uuid", eventUUID)
		return false
	}

	localConfig := &calendar.CalendarConfig{
		GoogleCalendarIntegration: *googleCalendarIntegrationConfig,
		ServerURL:                 appConfig.ServerSettings.ServerURL,
	}
	userCalendar := calendar.CreateUserCalendarFromConfig(ctx, localConfig, svc.logger)

	err = svc.processCalendarEvent(ctx, eventDetails, googleCalendarIntegrationConfig, userCalendar)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to process calendar event", "err", err)
		return false
	}
	return true
}
