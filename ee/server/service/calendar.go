package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/calendar"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
)

var (
	asyncCalendarProcessing bool
	asyncMutex              sync.Mutex
)

func (svc *Service) CalendarWebhook(ctx context.Context, eventUUID string, channelID string, resourceState string) error {
	uuid := uuid.New().String()
	level.Debug(svc.logger).Log("msg", "CalendarWebhook", "requestID", uuid, "eventUUID:", eventUUID, "channelID:", channelID, "resourceState:", resourceState)

	// We don't want the sender to cancel the context since we want to make sure we process the webhook.
	ctx = context.WithoutCancel(ctx)

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return fmt.Errorf("%s load app config: %w", uuid, err)
	}

	if len(appConfig.Integrations.GoogleCalendar) == 0 {
		svc.authz.SkipAuthorization(ctx)
		level.Warn(svc.logger).Log("msg", "Received calendar callback, but Google Calendar integration is not configured", "requestID", uuid)
		return nil
	}
	googleCalendarIntegrationConfig := appConfig.Integrations.GoogleCalendar[0]

	if resourceState == "sync" {
		// This is a sync notification, not a real event
		svc.authz.SkipAuthorization(ctx)
		return nil
	}

	lockValue, reserved, err := svc.getCalendarLock(ctx, eventUUID, true, uuid)
	if err != nil {
		return fmt.Errorf("event %s requestID %s get calendar lock: %w", eventUUID, uuid, err)
	}
	// If lock has been reserved by cron, we will need to re-process this event in case the calendar event was changed after the cron job read it.
	if lockValue == "" && !reserved {
		level.Debug(svc.logger).Log("msg", "lock value empty, returning", "eventUUID", eventUUID, "requestID", uuid)
		// We did not get a lock, so there is nothing to do here
		return nil
	}

	eventDetails, err := svc.ds.GetCalendarEventDetailsByUUID(ctx, eventUUID)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		if fleet.IsNotFound(err) {
			// We could try to stop the channel callbacks here, but that may not be secure since we don't know if the request is legitimate
			level.Info(svc.logger).Log("msg", "Received calendar callback, but did not find corresponding event in database", "event_uuid",
				eventUUID, "channel_id", channelID, "requestID", uuid)
			return nil
		}
		return fmt.Errorf("requestID: %s get calendar event details by UUID %s: %w", uuid, eventUUID, err)
	}
	if eventDetails.TeamID == nil {
		// Should not happen
		return fmt.Errorf("%s calendar event %s has no team ID", uuid, eventUUID)
	}

	localConfig := &calendar.Config{
		GoogleCalendarIntegration: *googleCalendarIntegrationConfig,
		ServerURL:                 appConfig.ServerSettings.ServerURL,
	}
	userCalendar := calendar.CreateUserCalendarFromConfig(ctx, localConfig, svc.logger)

	// Authenticate request. We will use the channel ID for authentication.
	svc.authz.SkipAuthorization(ctx)
	savedChannelID, err := userCalendar.Get(&eventDetails.CalendarEvent, "channelID")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get channel ID", "requestID", uuid)
	}
	if savedChannelID != channelID {
		return authz.ForbiddenWithInternal(fmt.Sprintf("%s calendar channel ID mismatch: %s != %s", uuid, savedChannelID, channelID), nil, nil, nil)
	}

	if !reserved {
		unlocked := false
		defer func() {
			if !unlocked {
				svc.releaseCalendarLock(ctx, eventUUID, lockValue, uuid)
			}
		}()

		// Remove event from the queue so that we don't process this event again.
		// Note: This item can be added back to the queue while we are processing it.
		err = svc.distributedLock.RemoveFromSet(ctx, calendar.QueueKey, eventUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "remove calendar event from queue", "requestID", uuid, "eventUUID", eventUUID)
		}

		id := "requestID: " + uuid + "eventUUID: " + eventUUID
		err = svc.processCalendarEvent(ctx, eventDetails, googleCalendarIntegrationConfig, userCalendar, id)
		if err != nil {
			return err
		}
		svc.releaseCalendarLock(ctx, eventUUID, lockValue, uuid)
		unlocked = true
	}

	// Now, we need to check if there are any events in the queue that need to be re-processed.
	asyncMutex.Lock()
	defer asyncMutex.Unlock()
	if !asyncCalendarProcessing {
		level.Debug(svc.logger).Log("msg", "processing set", "requestID", uuid, "eventUUID", eventUUID)
		eventIDs, err := svc.distributedLock.GetSet(ctx, calendar.QueueKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get calendar event queue", "requestID", uuid, "eventUUID", eventUUID)
		}
		if len(eventIDs) > 0 {
			asyncCalendarProcessing = true
			level.Debug(svc.logger).Log("msg", "Processing calendar events asynchronously", "requestID", uuid, "eventUUID", eventUUID)
			go svc.processCalendarAsync(ctx, eventIDs, uuid)
		} else {
			level.Debug(svc.logger).Log("msg", "No calendar events in queue", "requestID", uuid, "eventUUID", eventUUID)
		}

		return nil
	}

	return nil
}

func (svc *Service) processCalendarEvent(ctx context.Context, eventDetails *fleet.CalendarEventDetails,
	googleCalendarIntegrationConfig *fleet.GoogleCalendarIntegration, userCalendar fleet.UserCalendar, requestID string,
) error {
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

		return calendar.GenerateCalendarEventBody(ctx, svc.ds, team.Name, host, &sync.Map{}, conflict, svc.logger) + " " + eventDetails.UUID, true, nil
	}

	err := userCalendar.Configure(eventDetails.Email)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "configure calendar", "ids:", requestID)
	}
	event, updated, err := userCalendar.GetAndUpdateEvent(&eventDetails.CalendarEvent, genBodyFn, requestID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get and update event", "ids:", requestID)
	}
	if updated && event != nil {
		// Event was updated, so we need to save it
		level.Debug(svc.logger).Log("msg", "ds.CreateOrUpdateCalendarEvent", "event.UUID", event.UUID, "ids", requestID)
		_, err = svc.ds.CreateOrUpdateCalendarEvent(ctx, event.UUID, event.Email, event.StartTime, event.EndTime, event.Data,
			event.TimeZone, eventDetails.HostID, fleet.CalendarWebhookStatusNone)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "create or update calendar event", "ids:", requestID)
		}
	}

	return nil
}

func (svc *Service) releaseCalendarLock(ctx context.Context, eventUUID string, lockValue, requestID string) {
	ok, err := svc.distributedLock.ReleaseLock(ctx, calendar.LockKeyPrefix+eventUUID, lockValue)
	if err != nil {
		level.Error(svc.logger).Log("requestID", requestID, "eventUUID", eventUUID, "msg", "Failed to release calendar lock", "err", err)
	}
	if !ok {
		// If the lock was not released, it will expire on its own.
		level.Warn(svc.logger).Log("requestID", requestID, "msg", "Failed to release calendar lock", "eventUUID", eventUUID, "requestID", requestID)
	}
	level.Debug(svc.logger).Log("msg", "Released calendar lock", "eventUUID", eventUUID, "requestID", requestID)
}

func (svc *Service) getCalendarLock(ctx context.Context, eventUUID string, addToQueue bool, requestID string) (lockValue string, reserved bool, err error) {
	// Check if lock has been reserved, which means we can't have it.
	reservedValue, err := svc.distributedLock.Get(ctx, calendar.ReservedLockKeyPrefix+eventUUID)
	if err != nil {
		level.Debug(svc.logger).Log("msg", "Failed to get reserved calendar lock", "err", err, "requestID", requestID, "eventUUID", eventUUID)
		return "", false, ctxerr.Wrap(ctx, err, "get calendar reserved lock")
	}
	reserved = reservedValue != nil
	if reserved && !addToQueue {
		// We flag the lock as reserved.
		return "", reserved, nil
	}
	var lockAcquired bool
	if !reserved {
		// Try to acquire the lock
		lockValue = uuid.New().String()
		lockAcquired, err = svc.distributedLock.AcquireLock(ctx, calendar.LockKeyPrefix+eventUUID, lockValue, 0)
		if err != nil {
			level.Error(svc.logger).Log("msg", "Failed to acquire calendar lock", "err", err, "requestID", requestID, "eventUUID", eventUUID)
			return "", false, ctxerr.Wrap(ctx, err, "acquire calendar lock")
		}
		level.Debug(svc.logger).Log("LockAquired", lockAcquired, "eventUUID", eventUUID, "requestID", requestID)

		if !lockAcquired {
			lockValue = ""
		}
	}
	if (!lockAcquired || reserved) && addToQueue {
		// Could not acquire lock, so we are already processing this event. In this case, we add the event to
		// the queue (actually a set) to indicate that we need to re-process the event.
		err = svc.distributedLock.AddToSet(ctx, calendar.QueueKey, eventUUID)
		if err != nil {
			level.Error(svc.logger).Log("msg", "Failed to add calendar event to queue", "err", err, "requestID", requestID, "eventUUID", eventUUID)
			return "", false, ctxerr.Wrap(ctx, err, "add calendar event to queue")
		}
		level.Debug(svc.logger).Log("msg", "Added calendar event to queue", "eventUUID", eventUUID, "requestID", requestID)

		if reserved {
			// We flag the lock as reserved.
			level.Debug(svc.logger).Log("msg", "Reserved calendar lock", "eventUUID", eventUUID, "requestID", requestID)
			return "", reserved, nil
		}

		// Try to acquire the lock again in case it was released while we were adding the event to the queue.
		lockAcquired, err = svc.distributedLock.AcquireLock(ctx, calendar.LockKeyPrefix+eventUUID, lockValue, 0)
		if err != nil {
			level.Error(svc.logger).Log("msg", "Failed to acquire calendar lock again", "err", err, "requestID", requestID, "eventUUID", eventUUID)
			return "", false, ctxerr.Wrap(ctx, err, "acquire calendar lock again")
		}
		level.Debug(svc.logger).Log("LockAquiredAgain", lockAcquired, "eventUUID", eventUUID, "requestID", requestID)

		if !lockAcquired {
			level.Debug(svc.logger).Log("msg", "Could not acquire calendar lock again", "eventUUID", eventUUID, "requestID", requestID)
			// We could not acquire the lock, so we are done here.
			return "", reserved, nil
		}
	}
	return lockValue, false, nil
}

func (svc *Service) processCalendarAsync(ctx context.Context, eventIDs []string, requestID string) {
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
			if ok := svc.processCalendarEventAsync(ctx, eventUUID, requestID); !ok {
				return
			}
		}

		// Now we check whether there are any more events in the queue.
		var err error
		eventIDs, err = svc.distributedLock.GetSet(ctx, calendar.QueueKey)
		if err != nil {
			level.Error(svc.logger).Log("msg", "Failed to get calendar event queue", "err", err, "requestID", requestID)
			return
		}
	}
}

func (svc *Service) processCalendarEventAsync(ctx context.Context, eventUUID, requestID string) bool {
	level.Debug(svc.logger).Log("func", "ProcessCalendarEventAsync", "msg", "event_uuid", eventUUID, "requestID", requestID)
	lockValue, _, err := svc.getCalendarLock(ctx, eventUUID, false, requestID)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to get calendar lock", "err", err, "requestID", requestID, "eventUUID", eventUUID)
		return false
	}
	if lockValue == "" {
		// We did not get a lock, so there is nothing to do here
		return true
	}
	defer svc.releaseCalendarLock(ctx, eventUUID, lockValue, requestID)

	// Remove event from the queue so that we don't process this event again.
	// Note: This item can be added back to the queue while we are processing it.
	err = svc.distributedLock.RemoveFromSet(ctx, calendar.QueueKey, eventUUID)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to remove calendar event from queue", "err", err, "requestID", requestID, "eventUUID", eventUUID)
		return false
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to load app config", "err", err, "requestID", requestID, "eventUUID", eventUUID)
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
			// We found this event when the callback initially came in. So the event may have been removed or re-created since then.
			return true
		}
		level.Error(svc.logger).Log("msg", "Failed to get calendar event details", "err", err, "requestID", requestID, "eventUUID", eventUUID)
		return false
	}
	if eventDetails.TeamID == nil {
		// Should not happen
		level.Error(svc.logger).Log("msg", "Calendar event has no team ID", "uuid", eventUUID, "requestID", requestID, "eventUUID", eventUUID)
		return false
	}

	localConfig := &calendar.Config{
		GoogleCalendarIntegration: *googleCalendarIntegrationConfig,
		ServerURL:                 appConfig.ServerSettings.ServerURL,
	}
	userCalendar := calendar.CreateUserCalendarFromConfig(ctx, localConfig, svc.logger)

	err = svc.processCalendarEvent(ctx, eventDetails, googleCalendarIntegrationConfig, userCalendar, requestID)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to process calendar event", "err", err, "requestID", requestID, "eventUUID", eventUUID)
		return false
	}
	return true
}
