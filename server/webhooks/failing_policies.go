package webhooks

import (
	"context"
	"database/sql"
	"errors"
	"path"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func TriggerFailingPoliciesWebhook(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	appConfig *fleet.AppConfig,
	failingPoliciesSet service.FailingPolicySet,
	now time.Time,
) error {
	if !appConfig.WebhookSettings.FailingPoliciesWebhook.Enable {
		return nil
	}

	level.Debug(logger).Log("enabled", "true")

	for _, policyID := range appConfig.WebhookSettings.FailingPoliciesWebhook.PolicyIDs {
		policy, err := ds.Policy(ctx, policyID)
		switch {
		case err == nil:
			// OK
		case errors.Is(err, sql.ErrNoRows):
			// TODO(lucas): Deal with deleted policies.
			continue
		default:
			return ctxerr.Wrapf(ctx, err, "failing to load failing policies set %d", policyID)
		}
		hostIDs, err := failingPoliciesSet.ListHosts(policyID)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "listing hosts for failing policies set %d", policyID)
		}
		failingHosts := make([]FailingHost, len(hostIDs))
		for i := range hostIDs {
			failingHosts[i] = makeFailingHost(hostIDs[i], appConfig.ServerSettings.ServerURL)
		}
		payload := FailingPoliciesPayload{
			Timestamp:    now,
			Policy:       policy,
			FailingHosts: failingHosts,
		}
		url := appConfig.WebhookSettings.FailingPoliciesWebhook.DestinationURL
		err = server.PostJSONWithTimeout(ctx, url, &payload)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "posting to '%s'", url)
		}
		if err := failingPoliciesSet.RemoveHosts(policyID, hostIDs); err != nil {
			return ctxerr.Wrapf(ctx, err, "removing hosts %v from failing policies set %d", hostIDs, policyID)
		}
	}
	return nil
}

type FailingPoliciesPayload struct {
	Timestamp    time.Time     `json:"timestamp"`
	Policy       *fleet.Policy `json:"policy"`
	FailingHosts []FailingHost `json:"hosts"`
}

type FailingHost struct {
	ID       uint   `json:"id"`
	Hostname string `json:"hostname"`
	URL      string `json:"url"`
}

func makeFailingHost(hostID uint, fleetServerURL string) FailingHost {
	return FailingHost{
		ID: hostID,
		// TODO(lucas): Currently hostname is empty.
		// Preload hostname into redis so that we don't have to perform hosts db lookup.
		Hostname: "todo",
		URL:      path.Join(fleetServerURL, "hosts", strconv.Itoa(int(hostID))),
	}
}
