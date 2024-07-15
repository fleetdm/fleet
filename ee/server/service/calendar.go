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

// RecentUpdateDuration is the duration during which we will ignore a calendar event callback if the event was just updated.
// This variable is exposed to be modified by unit tests.
var RecentUpdateDuration = -2 * time.Second

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
	now := time.Now()
	if eventDetails.UpdatedAt.After(now.Add(RecentUpdateDuration)) {
		// If the event was updated recently, we will ignore the callback.
		// This is to avoid reading stale data from Google Calendar after we just updated the event.
		// If this was a legitimate update, then it will be caught by the next cron job run (or the next callback).
		svc.authz.SkipAuthorization(ctx)
		return nil
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

	lockValue, reserved, err := svc.getCalendarLock(ctx, eventUUID, true)
	if err != nil {
		return err
	}
	// If lock has been reserved by cron, we will need to re-process this event in case the calendar event was changed after the cron job read it.
	if lockValue == "" && !reserved {
		// We did not get a lock, so there is nothing to do here
		return nil
	}

	if !reserved {
		unlocked := false
		defer func() {
			if !unlocked {
				svc.releaseCalendarLock(ctx, eventUUID, lockValue)
			}
		}()

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

	// This flag indicates that calendar event should no longer exist, and we can stop watching it.
	stopChannel := false
	genBodyFn := func(conflict bool) (body string, ok bool, err error) {

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
		level.Warn(svc.logger).Log("msg", "Failed to release calendar lock")
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
		lockAcquired, err = svc.distributedLock.AcquireLock(ctx, calendar.LockKeyPrefix+eventUUID, lockValue, 0)
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
		lockAcquired, err = svc.distributedLock.AcquireLock(ctx, calendar.LockKeyPrefix+eventUUID, lockValue, 0)
		if err != nil {
			return "", false, ctxerr.Wrap(ctx, err, "acquire calendar lock again")
		}

		if !lockAcquired {
			// We could not acquire the lock, so we are done here.
			return "", reserved, nil
		}
	}
	return lockValue, false, nil
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
		eventIDs, err = svc.distributedLock.GetSet(ctx, calendar.QueueKey)
		if err != nil {
			level.Error(svc.logger).Log("msg", "Failed to get calendar event queue", "err", err)
			return
		}
	}
}

func (svc *Service) processCalendarEventAsync(ctx context.Context, eventUUID string) bool {
	lockValue, _, err := svc.getCalendarLock(ctx, eventUUID, false)
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
