package calendar

// This package contains common calendar code used by cron and service packages.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/ee/server/calendar"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type CalendarConfig struct {
	config.CalendarConfig
	fleet.GoogleCalendarIntegration
	ServerURL string
}

func CreateUserCalendarFromConfig(ctx context.Context, config *CalendarConfig, logger kitlog.Logger) fleet.UserCalendar {
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
