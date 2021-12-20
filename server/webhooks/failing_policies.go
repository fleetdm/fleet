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

	globalPoliciesURL := appConfig.WebhookSettings.FailingPoliciesWebhook.DestinationURL
	if globalPoliciesURL == "" {
		level.Info(logger).Log("msg", "empty global destination_url")
		return nil
	}
	serverURL, err := url.Parse(appConfig.ServerSettings.ServerURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "invalid server url")
	}
	policies, err := filteredPolicies(ctx, ds, appConfig.WebhookSettings.FailingPoliciesWebhook.PolicyIDs, failingPoliciesSet, logger)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "filtering policies")
	}
	for _, policy := range policies {
		hosts, err := failingPoliciesSet.ListHosts(policy.ID)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "listing hosts for global failing policies set %d", policy.ID)
		}
		if len(hosts) == 0 {
			level.Debug(logger).Log("id", policy.ID, "msg", "no hosts")
			continue
		}
		if err := sendBatchedPOSTs(ctx, policy, hosts, failingPoliciesSet, postData{
			serverURL:  serverURL,
			now:        now,
			webhookURL: globalPoliciesURL,
		}, appConfig.WebhookSettings.FailingPoliciesWebhook.HostBatchSize, logger); err != nil {
			return ctxerr.Wrapf(ctx, err, "sending POSTs for policy set %d", policy.ID)
		}
	}
	return nil
}

type postData struct {
	serverURL  *url.URL
	now        time.Time
	webhookURL string
}

func sendBatchedPOSTs(
	ctx context.Context,
	policy *fleet.Policy,
	hosts []service.PolicySetHost,
	failingPoliciesSet service.FailingPolicySet,
	postData postData,
	hostBatchSize int,
	logger kitlog.Logger,
) error {
	if hostBatchSize == 0 {
		hostBatchSize = len(hosts)
	}
	sort.Slice(hosts, func(i, j int) bool {
		return hosts[i].ID < hosts[j].ID
	})
	for len(hosts) > 0 {
		j := hostBatchSize
		if l := len(hosts); j > l {
			j = l
		}
		batch := hosts[:j]
		failingHosts := make([]FailingHost, len(batch))
		for i := range batch {
			failingHosts[i] = makeFailingHost(batch[i], *postData.serverURL)
		}
		payload := FailingPoliciesPayload{
			Timestamp:    postData.now,
			Policy:       policy,
			FailingHosts: failingHosts[:j],
		}
		level.Debug(logger).Log("payload", payload, "url", postData.webhookURL, "batch", len(batch))
		if err := server.PostJSONWithTimeout(ctx, postData.webhookURL, &payload); err != nil {
			return ctxerr.Wrapf(ctx, err, "posting to '%s'", postData.webhookURL)
		}
		if err := failingPoliciesSet.RemoveHosts(policy.ID, batch); err != nil {
			return ctxerr.Wrapf(ctx, err, "removing hosts %+v from failing policies set %d", batch, policy.ID)
		}
		hosts = hosts[j:]
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

func makeFailingHost(host service.PolicySetHost, serverURL url.URL) FailingHost {
	serverURL.Path = path.Join(serverURL.Path, "hosts", strconv.Itoa(int(host.ID)))
	return FailingHost{
		ID:       host.ID,
		Hostname: host.Hostname,
		URL:      serverURL.String(),
	}
}

func filteredPolicies(
	ctx context.Context,
	ds fleet.Datastore,
	configuredPolicyIDs []uint,
	failingPoliciesSet service.FailingPolicySet,
	logger kitlog.Logger,
) ([]*fleet.Policy, error) {
	configuredPolicyIDsSet := make(map[uint]struct{})
	for _, policyID := range configuredPolicyIDs {
		configuredPolicyIDsSet[policyID] = struct{}{}
	}
	policySets, err := failingPoliciesSet.ListSets()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing global policies set")
	}
	var policies []*fleet.Policy
	var gcSet []uint
	for _, policyID := range policySets {
		if _, ok := configuredPolicyIDsSet[policyID]; !ok {
			level.Debug(logger).Log("msg", "skipping policy from set, not in config", "id", policyID)
			gcSet = append(gcSet, policyID)
			continue
		}
		switch policy, err := ds.Policy(ctx, policyID); {
		case err == nil:
			policies = append(policies, policy)
		case errors.Is(err, sql.ErrNoRows):
			level.Debug(logger).Log("msg", "skipping policy from set, deleted", "id", policyID)
			gcSet = append(gcSet, policyID)
		default:
			return nil, ctxerr.Wrapf(ctx, err, "failing to load global failing policies set %d", policyID)
		}
	}
	// Remove the policies that are present in the set but:
	//	- are not present in the config (user disabled automation for them), or,
	//	- do not exist anymore (user deleted the policy).
	for _, policyID := range gcSet {
		if err := failingPoliciesSet.RemoveSet(policyID); err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "removing global policy  %d from policy set", policyID)
		}
	}
	return policies, nil
}
