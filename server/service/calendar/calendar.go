package calendar

// This package contains common calendar code used by cron and service packages.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/calendar"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

const (
	LockKeyPrefix         = "calendar:lock:"
	ReservedLockKeyPrefix = "calendar:reserved:"
	RecentUpdateKeyPrefix = "calendar:recent_update:"
	QueueKey              = "calendar:queue"

	// DistributedLockExpireMs is the time Redis will hold the lock before automatically releasing it.
	// Our current max retry time for calendar API is 10 minutes, and multiple API calls (with their own retry timing) can be made during event processing.
	// If a Fleet server gets the lock and is then shut down before releasing the lock, the next server may need to wait this long
	// before getting the lock.
	DistributedLockExpireMs = 20 * 60 * 1000
	// ReserveLockExpireMs is used by cron job to guarantee that it gets the next lock.
	ReserveLockExpireMs = 2 * DistributedLockExpireMs

	// RecentCalendarUpdateValue is the value stored in Redis to indicate that a calendar event was recently updated.
	RecentCalendarUpdateValue = "1"
)

// RecentCalendarUpdateDuration is the duration during which we will ignore a calendar event callback if the event in DB was just updated by a previous callback.
// This reduces CPU load and Google API load. If we update the event, Google calendar may send a callback which we don't need to process.
// We are using Redis instead of updated_at timestamp in DB because the calendar cron job may update the timestamp even when the event did not change, which could
// cause us to miss a legitimate update.
// This variable is exposed so that it can be modified by unit tests.
var RecentCalendarUpdateDuration = 10 * time.Second

type Config struct {
	config.CalendarConfig
	fleet.GoogleCalendarIntegration
	ServerURL string
}

func CreateUserCalendarFromConfig(ctx context.Context, config *Config, logger kitlog.Logger) fleet.UserCalendar {
	googleCalendarConfig := calendar.GoogleCalendarConfig{
		Context:           ctx,
		IntegrationConfig: &config.GoogleCalendarIntegration,
		ServerURL:         config.ServerURL,
		Logger:            kitlog.With(logger, "component", "google_calendar"),
	}
	return calendar.NewGoogleCalendar(&googleCalendarConfig)
}

func GenerateCalendarEventBody(ctx context.Context, ds fleet.Datastore, orgName string, host fleet.HostPolicyMembershipData,
	policyIDtoPolicy *sync.Map, conflict bool, logger kitlog.Logger,
) string {
	description, resolution := getCalendarEventDescriptionAndResolution(ctx, ds, orgName, host, policyIDtoPolicy, logger)

	conflictStr := ""
	if conflict {
		conflictStr = "because there was no remaining availability "
	}
	return fmt.Sprintf(`%s %s %s(%s).

Please leave your device on and connected to power.

<b>Why it matters</b>
%s

<b>What we'll do</b>
%s
`, orgName, fleet.CalendarBodyStaticHeader, conflictStr, host.HostDisplayName, description, resolution)
}

func getCalendarEventDescriptionAndResolution(ctx context.Context, ds fleet.Datastore, orgName string, host fleet.HostPolicyMembershipData,
	policyIDtoPolicy *sync.Map, logger kitlog.Logger,
) (string, string) {
	getDefaultDescription := func() string {
		return fmt.Sprintf(`%s %s`, orgName, fleet.CalendarDefaultDescription)
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
				return getDefaultDescription(), fleet.CalendarDefaultResolution
			}
			policy, err = ds.PolicyLite(ctx, uint(id))
			if err != nil {
				level.Error(logger).Log("msg", "get policy", "err", err)
				return getDefaultDescription(), fleet.CalendarDefaultResolution
			}
			policyIDtoPolicy.Store(policyIDs[0], policy)
		} else {
			policy = policyAny.(*fleet.PolicyLite)
		}
		policyDescription := strings.TrimSpace(policy.Description)
		if policyDescription == "" || policy.Resolution == nil || strings.TrimSpace(*policy.Resolution) == "" {
			description = getDefaultDescription()
			resolution = fleet.CalendarDefaultResolution
		} else {
			description = policyDescription
			resolution = strings.TrimSpace(*policy.Resolution)
		}
	} else {
		description = getDefaultDescription()
		resolution = fleet.CalendarDefaultResolution
	}
	return description, resolution
}
