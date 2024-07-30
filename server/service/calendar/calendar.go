package calendar

// This package contains common calendar code used by cron and service packages.

import (
	"context"
	"crypto/sha256"
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
	DefaultEventBodyTag   = "default"

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

// PolicyLiteWithMeta is a wrapper around fleet.PolicyLite that includes a tag for policy's description/resolution.
type PolicyLiteWithMeta struct {
	fleet.PolicyLite
	Tag string
	mu  sync.Mutex
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
) (body string, tag string) {
	description, resolution, tag := getCalendarEventDescriptionAndResolution(ctx, ds, orgName, host, policyIDtoPolicy, logger)

	conflictStr := ""
	if conflict {
		conflictStr = fleet.CalendarEventConflictText
	}
	return fmt.Sprintf(`%s %s %s(%s).

Please leave your device on and connected to power.

<b>Why it matters</b>
%s

<b>What we'll do</b>
%s
`, orgName, fleet.CalendarBodyStaticHeader, conflictStr, host.HostDisplayName, description, resolution), tag
}

func getCalendarEventDescriptionAndResolution(ctx context.Context, ds fleet.Datastore, orgName string, host fleet.HostPolicyMembershipData,
	policyIDtoPolicy *sync.Map, logger kitlog.Logger,
) (description string, resolution string, tag string) {
	getDefaultDescription := func() string {
		return fmt.Sprintf(`%s %s`, orgName, fleet.CalendarDefaultDescription)
	}

	policyIDs := strings.Split(host.FailingPolicyIDs, ",")
	if len(policyIDs) == 1 && policyIDs[0] != "" {
		var policy *PolicyLiteWithMeta
		policyAny, ok := policyIDtoPolicy.Load(policyIDs[0])
		if !ok {
			id, err := strconv.ParseUint(policyIDs[0], 10, 64)
			if err != nil {
				level.Error(logger).Log("msg", "parse policy id", "err", err)
				return getDefaultDescription(), fleet.CalendarDefaultResolution, DefaultEventBodyTag
			}
			policyLite, err := ds.PolicyLite(ctx, uint(id))
			if err != nil {
				level.Error(logger).Log("msg", "get policy", "err", err)
				return getDefaultDescription(), fleet.CalendarDefaultResolution, DefaultEventBodyTag
			}
			policy = new(PolicyLiteWithMeta)
			policy.PolicyLite = *policyLite
			policyIDtoPolicy.Store(policyIDs[0], policy)
		} else {
			policy = policyAny.(*PolicyLiteWithMeta)
		}
		policyDescription := strings.TrimSpace(policy.Description)
		if policyDescription == "" || policy.Resolution == nil || strings.TrimSpace(*policy.Resolution) == "" {
			description = getDefaultDescription()
			resolution = fleet.CalendarDefaultResolution
			tag = DefaultEventBodyTag
		} else {
			description = policyDescription
			resolution = strings.TrimSpace(*policy.Resolution)
			policy.mu.Lock() // To make sure only one policy is reading/writing to the tag at a time.
			defer policy.mu.Unlock()
			if policy.Tag == "" {
				// Calculate a unique signature for the event body, which we will use to check if the event body has changed.
				policy.Tag = fmt.Sprintf("%x", sha256.Sum256([]byte(policy.Description+*policy.Resolution)))
			}
			tag = policy.Tag
		}
	} else {
		description = getDefaultDescription()
		resolution = fleet.CalendarDefaultResolution
		tag = DefaultEventBodyTag
	}
	return description, resolution, tag
}
