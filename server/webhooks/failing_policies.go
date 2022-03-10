package webhooks

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"path"
	"sort"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// TriggerFailingPoliciesWebhook performs the webhook requests for failing policies.
func TriggerFailingPoliciesWebhook(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	appConfig *fleet.AppConfig,
	failingPoliciesSet fleet.FailingPolicySet,
	now time.Time,
) error {
	serverURL, err := url.Parse(appConfig.ServerSettings.ServerURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "invalid server url")
	}

	globalSettings := appConfig.WebhookSettings.FailingPoliciesWebhook
	globalPolicyIDs := make(map[uint]struct{}, len(globalSettings.PolicyIDs))
	for _, policyID := range globalSettings.PolicyIDs {
		globalPolicyIDs[policyID] = struct{}{}
	}
	var globalWebhookURL *url.URL
	if globalSettings.Enable {
		globalWebhookURL, err = url.Parse(appConfig.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "parse global webhook url", "err", err)
		}
	}

	// team caches
	teamSettings := make(map[uint]fleet.FailingPoliciesWebhookSettings)
	teamPolicyIDs := make(map[uint]map[uint]struct{})
	getTeam := func(teamID uint) error {
		settings, ok := teamSettings[teamID]
		if ok {
			return nil
		}

		team, err := ds.Team(ctx, teamID)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "get team: %d", teamID)
		}

		settings = team.Config.WebhookSettings.FailingPoliciesWebhook
		teamSettings[teamID] = settings
		policyIDs := make(map[uint]struct{}, len(settings.PolicyIDs))
		for _, policyID := range settings.PolicyIDs {
			policyIDs[policyID] = struct{}{}
		}
		teamPolicyIDs[teamID] = policyIDs

		return nil
	}

	policySets, err := failingPoliciesSet.ListSets()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list policies set")
	}

	for _, policyID := range policySets {
		policy, err := ds.Policy(ctx, policyID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			level.Debug(logger).Log("msg", "skipping failing policy, deleted", "policyID", policyID)
			if err := failingPoliciesSet.RemoveSet(policy.ID); err != nil {
				level.Error(logger).Log("msg", "failed to remove policy from set", "policyID", policyID, "err", err)
			}
			continue
		case err != nil:
			return ctxerr.Wrapf(ctx, err, "get policy: %d", policyID)
		default:
			// Ok
		}

		if policy.TeamID != nil {
			// team policy
			err := getTeam(*policy.TeamID)
			switch {
			case errors.Is(err, sql.ErrNoRows):
				// shouldn't happen, unless the team was deleted after the policy was retrieved above
				level.Debug(logger).Log("msg", "team does not exist", "teamID", *policy.TeamID)
				continue
			case err != nil:
				level.Error(logger).Log("msg", "failed to get team", "teamID", *policy.TeamID, "err", err)
				continue
			}

			settings := teamSettings[*policy.TeamID]
			if !settings.Enable {
				continue
			}
			webhookURL, err := url.Parse(settings.DestinationURL)
			if err != nil {
				level.Error(logger).Log("msg", "failed to parse webhook url", "err", err)
				continue
			}
			_, ok := teamPolicyIDs[*policy.TeamID][policy.ID]
			if !ok {
				level.Debug(logger).Log("msg", "skipping failing policy, deleted from team policy IDs", "policyID", policyID)
				if err := failingPoliciesSet.RemoveSet(policy.ID); err != nil {
					level.Error(logger).Log("msg", "failed to remove policy from set", "policyID", policyID, "err", err)
				}
				continue
			}

			err = sendFailingPoliciesBatchedPOSTs(
				ctx,
				policy,
				failingPoliciesSet,
				settings.HostBatchSize,
				serverURL,
				webhookURL,
				now,
				logger,
			)
			if err != nil {
				level.Error(logger).Log("msg", "failed to send failing policies webhook requests", "policyID", policy.ID, "err", err)
			}

			continue
		}

		// global policy
		err = sendFailingPoliciesBatchedPOSTs(
			ctx,
			policy,
			failingPoliciesSet,
			globalSettings.HostBatchSize,
			serverURL,
			globalWebhookURL,
			now,
			logger,
		)
		if err != nil {
			level.Error(logger).Log("msg", "failed to send failing policies webhook requests", "policyID", policy.ID, "err", err)
		}
	}

	return nil
}

func sendFailingPoliciesBatchedPOSTs(
	ctx context.Context,
	policy *fleet.Policy,
	failingPoliciesSet fleet.FailingPolicySet,
	hostBatchSize int,
	serverURL *url.URL,
	webhookURL *url.URL,
	now time.Time,
	logger kitlog.Logger,
) error {
	hosts, err := failingPoliciesSet.ListHosts(policy.ID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "listing hosts for failing policies set %d", policy.ID)
	}
	if len(hosts) == 0 {
		level.Debug(logger).Log("msg", "no hosts", "policyID", policy.ID)
		return nil
	}
	sort.Slice(hosts, func(i, j int) bool {
		return hosts[i].ID < hosts[j].ID
	})

	if hostBatchSize == 0 {
		hostBatchSize = len(hosts)
	}
	for i := 0; i < len(hosts); i += hostBatchSize {
		end := i + hostBatchSize
		if end > len(hosts) {
			end = len(hosts)
		}
		batch := hosts[i:end]

		failingHosts := make([]FailingHost, len(batch))
		for i, host := range batch {
			failingHosts[i] = makeFailingHost(host, serverURL)
		}

		payload := FailingPoliciesPayload{
			Timestamp:    now,
			Policy:       policy,
			FailingHosts: failingHosts,
		}
		level.Debug(logger).Log("payload", payload, "url", webhookURL.String(), "batch", len(batch))
		if err := server.PostJSONWithTimeout(ctx, webhookURL.String(), &payload); err != nil {
			return ctxerr.Wrapf(ctx, err, "posting to %q", webhookURL)
		}
		if err := failingPoliciesSet.RemoveHosts(policy.ID, batch); err != nil {
			return ctxerr.Wrapf(ctx, err, "removing hosts %+v from failing policies set %d", batch, policy.ID)
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

func makeFailingHost(host fleet.PolicySetHost, serverURL *url.URL) FailingHost {
	u := *serverURL
	u.Path = path.Join(serverURL.Path, "hosts", strconv.FormatUint(uint64(host.ID), 10))
	return FailingHost{
		ID:       host.ID,
		Hostname: host.Hostname,
		URL:      serverURL.String(),
	}
}
