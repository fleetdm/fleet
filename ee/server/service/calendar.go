package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/calendar"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
)

var asyncCalendarProcessing bool
var asyncMutex sync.Mutex

func (svc *Service) CalendarWebhook(ctx context.Context, eventUUID string, channelID string, resourceState string) error {

	// We don't want the sender to cancel the context since we want to make sure we process the webhook.
	ctx = context.WithoutCancel(ctx)

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

	if resourceState == "sync" {
		// This is a sync notification, not a real event
		svc.authz.SkipAuthorization(ctx)
		return nil
	}

	// If the event was updated recently, we will ignore the callback.
	// If this was a legitimate update, then it will be caught by the next cron job run (or a future callback).
	recent, err := svc.distributedLock.Get(ctx, calendar.RecentUpdateKeyPrefix+eventUUID)
	if err != nil {
		return err
	}
	if recent != nil && *recent == calendar.RecentCalendarUpdateValue {
		svc.authz.SkipAuthorization(ctx)
		return nil
	}

	// In the common case, we get the lock right away and process the event.
	// Otherwise, we do additional validation to see if we need to process the event.
	lockValue, reserved, err := svc.getCalendarLock(ctx, eventUUID, false)
	if err != nil {
		return err
	}
	unlocked := false
	defer func() {
		if !unlocked && lockValue != "" {
			svc.releaseCalendarLock(ctx, eventUUID, lockValue)
		}
	}()

	eventDetails, err := svc.ds.GetCalendarEventDetailsByUUID(ctx, eventUUID)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		if fleet.IsNotFound(err) {
			// We could try to stop the channel callbacks here, but that may not be secure since we don't know if the request is legitimate
			level.Info(svc.logger).Log("msg", "Received calendar callback, but did not find corresponding event in database", "event_uuid",
				eventUUID, "channel_id", channelID)
			return nil
		}
		return err
	}
	if eventDetails.TeamID == nil {
		// Should not happen
		return fmt.Errorf("calendar event %s has no team ID", eventUUID)
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
		return ctxerr.Wrap(ctx, err, "get channel ID")
	}
	if savedChannelID != channelID {
		return authz.ForbiddenWithInternal(fmt.Sprintf("calendar channel ID mismatch: %s != %s", savedChannelID, channelID), nil, nil, nil)
	}

	// Now that we fully validated the request, try to get the lock again if we didn't get it the first time.
	// This time the event will be added to the queue if needed.
	if lockValue == "" {
		lockValue, reserved, err = svc.getCalendarLock(ctx, eventUUID, true)
		if err != nil {
			return err
		}
		if lockValue != "" {
			// We got the lock, so we can process the event. We need to refetch the event from DB, since it may have changed since the last fetch.
			eventDetails, err = svc.ds.GetCalendarEventDetailsByUUID(ctx, eventUUID)
			if err != nil {
				if fleet.IsNotFound(err) {
					// We found the event the first time, but it was deleted before we got the lock.
					level.Info(svc.logger).Log("msg", "Received calendar callback, but the event was just deleted", "event_uuid",
						eventUUID, "channel_id", channelID)
					return nil
				}
				return err
			}
		}
	}

	// If lock has been reserved by cron, we will need to re-process this event in case the calendar event was changed after the cron job read it.
	if lockValue == "" && !reserved {
		// We did not get a lock, so there is nothing to do here
		return nil
	}

	if !reserved {
		// Remove event from the queue so that we don't process this event again.
		// Note: This item can be added back to the queue while we are processing it.
		err = svc.distributedLock.RemoveFromSet(ctx, calendar.QueueKey, eventUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "remove calendar event from queue")
		}

		err = svc.processCalendarEvent(ctx, eventDetails, googleCalendarIntegrationConfig, userCalendar)
		if err != nil {
			return err
		}
		svc.releaseCalendarLock(ctx, eventUUID, lockValue)
		unlocked = true
	}

	// Now, we need to check if there are any events in the queue that need to be re-processed.
	asyncMutex.Lock()
	defer asyncMutex.Unlock()
	if !asyncCalendarProcessing {
		eventIDs, err := svc.distributedLock.GetSet(ctx, calendar.QueueKey)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get calendar event queue")
		}
		if len(eventIDs) > 0 {
			asyncCalendarProcessing = true
			go svc.processCalendarAsync(ctx, eventIDs)
		}
		return nil
	}

	return nil
}

func (svc *Service) processCalendarEvent(ctx context.Context, eventDetails *fleet.CalendarEventDetails,
	googleCalendarIntegrationConfig *fleet.GoogleCalendarIntegration, userCalendar fleet.UserCalendar) error {

	var generatedTag string
	// This flag indicates that calendar event should no longer exist, and we can stop watching it.
	stopChannel := false
	var genBodyFn fleet.CalendarGenBodyFn = func(conflict bool) (body string, ok bool, err error) {

		// This function is called when a new event is being created.
		var team *fleet.Team
		team, err = svc.ds.TeamWithoutExtras(ctx, *eventDetails.TeamID)
		if err != nil {
			return "", false, err
		}

		if team.Config.Integrations.GoogleCalendar == nil ||
			!team.Config.Integrations.GoogleCalendar.Enable {
			stopChannel = true
			return "", false, nil
		}

		var policies []fleet.PolicyCalendarData
		policies, err = svc.ds.GetCalendarPolicies(ctx, team.ID)
		if err != nil {
			return "", false, err
		}

		if len(policies) == 0 {
			stopChannel = true
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
			stopChannel = true
			return "", false, nil
		}
		host := hosts[0]
		if host.Passing { // host is passing all configured policies
			stopChannel = true
			return "", false, nil
		}
		if host.Email == "" {
			err = fmt.Errorf("host %d has no associated email", host.HostID)
			return "", false, err
		}

		body, generatedTag = calendar.GenerateCalendarEventBody(ctx, svc.ds, team.Name, host, &sync.Map{}, conflict, svc.logger)
		return body, true, nil
	}

	err := userCalendar.Configure(eventDetails.Email)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "configure calendar")
	}
	event, updated, err := userCalendar.GetAndUpdateEvent(&eventDetails.CalendarEvent, genBodyFn, fleet.CalendarGetAndUpdateEventOpts{})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get and update event")
	}
	if updated && event != nil {
		// Event was updated, so we set a flag.
		_, err = svc.distributedLock.SetIfNotExist(ctx, calendar.RecentUpdateKeyPrefix+event.UUID, calendar.RecentCalendarUpdateValue,
			uint64(calendar.RecentCalendarUpdateDuration.Milliseconds())) //nolint:gosec // dismiss G115
		if err != nil {
			return ctxerr.Wrap(ctx, err, "set recent update flag")
		}
		// Event was updated, so we need to save it
		if generatedTag != "" {
			err = event.SaveDataItems("body_tag", generatedTag)
		}
		if err != nil {
			return ctxerr.Wrap(ctx, err, "save calendar event body tag")
		}
		_, err = svc.ds.CreateOrUpdateCalendarEvent(ctx, event.UUID, event.Email, event.StartTime, event.EndTime, event.Data,
			event.TimeZone, eventDetails.HostID, fleet.CalendarWebhookStatusNone)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "create or update calendar event")
		}
		// Remove event from the queue (again) so that we don't process this event again in case we got a callback from the event change which we ourselves made.
		err = svc.distributedLock.RemoveFromSet(ctx, calendar.QueueKey, event.UUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "remove calendar event from queue")
		}

	}
	if stopChannel {
		// The cron job could have already stopped the channel. For example, if calendar was disabled.
		err = userCalendar.StopEventChannel(&eventDetails.CalendarEvent)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "stop event channel")
		}
	}

	return nil
}

func (svc *Service) releaseCalendarLock(ctx context.Context, eventUUID string, lockValue string) {
	ok, err := svc.distributedLock.ReleaseLock(ctx, calendar.LockKeyPrefix+eventUUID, lockValue)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to release calendar lock", "err", err)
	}
	if !ok {
		// If the lock was not released, it will expire on its own.
		level.Error(svc.logger).Log("msg", "Failed to release calendar lock", "event uuid", eventUUID, "lockValue", lockValue)
	}
}

func (svc *Service) getCalendarLock(ctx context.Context, eventUUID string, addToQueue bool) (lockValue string, reserved bool, err error) {
	// Check if lock has been reserved, which means we can't have it.
	reservedValue, err := svc.distributedLock.Get(ctx, calendar.ReservedLockKeyPrefix+eventUUID)
	if err != nil {
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
		lockAcquired, err = svc.distributedLock.SetIfNotExist(ctx, calendar.LockKeyPrefix+eventUUID, lockValue,
			calendar.DistributedLockExpireMs)
		if err != nil {
			return "", false, ctxerr.Wrap(ctx, err, "acquire calendar lock")
		}
	}
	if (!lockAcquired || reserved) && addToQueue {
		// Could not acquire lock, so we are already processing this event. In this case, we add the event to
		// the queue (actually a set) to indicate that we need to re-process the event.
		err = svc.distributedLock.AddToSet(ctx, calendar.QueueKey, eventUUID)
		if err != nil {
			return "", false, ctxerr.Wrap(ctx, err, "add calendar event to queue")
		}

		if reserved {
			// We flag the lock as reserved.
			return "", reserved, nil
		}

		// Try to acquire the lock again in case it was released while we were adding the event to the queue.
		lockAcquired, err = svc.distributedLock.SetIfNotExist(ctx, calendar.LockKeyPrefix+eventUUID, lockValue,
			calendar.DistributedLockExpireMs)
		if err != nil {
			return "", false, ctxerr.Wrap(ctx, err, "acquire calendar lock again")
		}
	}
	if !lockAcquired {
		// We could not acquire the lock, so we are done here.
		return "", reserved, nil
	}
	return lockValue, false, nil
}

func (svc *Service) processCalendarAsync(ctx context.Context, eventIDs []string) {
	defer func() {
		asyncMutex.Lock()
		asyncCalendarProcessing = false
		asyncMutex.Unlock()
	}()
	const minLoopTime = time.Second
	runTime := minLoopTime
	for {
		if len(eventIDs) == 0 {
			return
		}
		// We want to make sure we don't run this too often to reduce load on CPU/Redis, so we wait at least a second between runs.
		if runTime < minLoopTime && runTime > 0 {
			time.Sleep(minLoopTime - runTime)
		}
		start := svc.clock.Now()
		for _, eventUUID := range eventIDs {
			if ok := svc.processCalendarEventAsync(ctx, eventUUID); !ok {
				return
			}
		}
		end := svc.clock.Now()
		runTime = end.Sub(start)

		// Now we check whether there are any more events in the queue.
		var err error
		eventIDs, err = svc.distributedLock.GetSet(ctx, calendar.QueueKey)
		if err != nil {
			level.Error(svc.logger).Log("msg", "Failed to get calendar event queue", "err", err)
			return
		}
	}
}

func (svc *Service) processCalendarEventAsync(ctx context.Context, eventUUID string) bool {
	// If the event was updated recently, we will ignore it.
	// If this was a legitimate update, then it will be caught by the next cron job run (or a future callback).
	recent, err := svc.distributedLock.Get(ctx, calendar.RecentUpdateKeyPrefix+eventUUID)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to get recent update flag", "err", err)
		return false
	}
	if recent != nil && *recent == calendar.RecentCalendarUpdateValue {
		err = svc.distributedLock.RemoveFromSet(ctx, calendar.QueueKey, eventUUID)
		if err != nil {
			level.Error(svc.logger).Log("msg", "Failed to remove calendar event from queue", "err", err)
			return false
		}
		return true
	}

	lockValue, _, err := svc.getCalendarLock(ctx, eventUUID, false)
	if err != nil {
		level.Error(svc.logger).Log("msg", "Failed to get calendar lock", "err", err)
		return false
	}
	if lockValue == "" {
		// We did not get a lock, so there is nothing to do here
		return true
	}
	defer func() {
		svc.releaseCalendarLock(ctx, eventUUID, lockValue)
	}()

	// Remove event from the queue so that we don't process this event again.
	// Note: This item can be added back to the queue while we are processing it.
	err = svc.distributedLock.RemoveFromSet(ctx, calendar.QueueKey, eventUUID)
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
			// We found this event when the callback initially came in. So the event may have been removed or re-created since then.
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

	localConfig := &calendar.Config{
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
