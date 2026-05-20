package webhooks

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"path"
	"sort"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/fleetdm/fleet/v4/server/platform/http"
)

// SendFailingPoliciesBatchedPOSTs sends a failing policy to the provided
// webhook URL. It sends in batches if hostBatchSize > 0. After a successful
// send, the corresponding hosts are removed from the failing policies set.
//
// For each batch, it also records a policy_run per host (passed=false) and
// a policy_automation_executions row (one per newly-inserted run) so the
// policy status page can surface the webhook outcome. Hosts whose run was
// already recorded on a prior flip are silently skipped from the recording
// path (the webhook still fires for them — that behavior is unchanged).
func SendFailingPoliciesBatchedPOSTs(
	ctx context.Context,
	ds fleet.Datastore,
	policy *fleet.Policy,
	failingPoliciesSet fleet.FailingPolicySet,
	hostBatchSize int,
	serverURL *url.URL,
	webhookURL *url.URL,
	now time.Time,
	logger *slog.Logger,
) error {
	hosts, err := failingPoliciesSet.ListHosts(policy.ID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "listing hosts for failing policies set %d", policy.ID)
	}
	if len(hosts) == 0 {
		logger.DebugContext(ctx, "no hosts", "policyID", policy.ID)
		return nil
	}
	// The count may be out of date since it is only updated during the hourly cleanups_then_aggregation cron.
	// Take care of the case where the count is less than the actual number of hosts we are returning.
	hostsCount := uint(len(hosts))
	if hostsCount > policy.FailingHostCount {
		policy.FailingHostCount = hostsCount
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

		hostIDs := make([]uint, len(batch))
		for i, h := range batch {
			hostIDs[i] = h.ID
		}
		refs, lookupErr := ds.GetFailingPolicyRuns(ctx, []uint{policy.ID}, hostIDs)
		if lookupErr != nil {
			logger.ErrorContext(ctx, "failed to look up policy_run IDs", "policyID", policy.ID, "err", lookupErr)
		}
		batchID, err := ds.CreatePolicyAutomationExecutions(ctx, fleet.PolicyAutomationWebhook, refs)
		if err != nil {
			logger.ErrorContext(ctx, "failed to record policy automation execution rows", "policyID", policy.ID, "err", err)
		}

		failingHosts := make([]failingHost, len(batch))
		for i, host := range batch {
			failingHosts[i] = makeFailingHost(host, serverURL)
		}

		payload := failingPoliciesPayload{
			Timestamp:    now,
			Policy:       policy,
			FailingHosts: failingHosts,
		}
		logger.DebugContext(ctx, "sending failing policy batch", "payload", payload, "url", server.MaskSecretURLParams(webhookURL.String()), "batch", len(batch))

		// Marshal and duplicate renamed JSON keys (e.g. fleet_id → also team_id)
		// so that webhook consumers see both the new and deprecated field names.
		jsonBytes, err := json.Marshal(&payload)
		if err != nil {
			if updErr := ds.UpdatePolicyAutomationExecutions(ctx, batchID, err); updErr != nil {
				logger.ErrorContext(ctx, "failed to update policy automation executions status", "batch_id", batchID.String(), "err", updErr)
			}
			return ctxerr.Wrap(ctx, err, "marshal failing policies payload")
		}
		if rules := endpointer.ExtractAliasRules(payload); len(rules) > 0 {
			jsonBytes = endpointer.DuplicateJSONKeys(jsonBytes, rules, endpointer.DuplicateJSONKeysOpts{Compact: true})
		}

		if err := http.PostJSONWithTimeout(ctx, webhookURL.String(), json.RawMessage(jsonBytes), logger); err != nil {
			maskedErr := http.MaskURLError(err)
			if updErr := ds.UpdatePolicyAutomationExecutions(ctx, batchID, maskedErr); updErr != nil {
				logger.ErrorContext(ctx, "failed to update policy automation executions status", "batch_id", batchID.String(), "err", updErr)
			}
			return ctxerr.Wrapf(ctx, maskedErr, "posting to %q", http.MaskSecretURLParams(webhookURL.String()))
		}
		if updErr := ds.UpdatePolicyAutomationExecutions(ctx, batchID, nil); updErr != nil {
			logger.ErrorContext(ctx, "failed to update policy automation executions status", "batch_id", batchID.String(), "err", updErr)
		}

		if err := failingPoliciesSet.RemoveHosts(policy.ID, batch); err != nil {
			return ctxerr.Wrapf(ctx, err, "removing hosts %+v from failing policies set %d", batch, policy.ID)
		}
	}
	return nil
}

type failingPoliciesPayload struct {
	Timestamp    time.Time     `json:"timestamp"`
	Policy       *fleet.Policy `json:"policy"`
	FailingHosts []failingHost `json:"hosts"`
}

type failingHost struct {
	ID          uint   `json:"id"`
	Hostname    string `json:"hostname"`
	DisplayName string `json:"display_name"`
	URL         string `json:"url"`
}

func makeFailingHost(host fleet.PolicySetHost, serverURL *url.URL) failingHost {
	u := *serverURL
	u.Path = path.Join(serverURL.Path, "hosts", strconv.FormatUint(uint64(host.ID), 10))
	return failingHost{
		ID:          host.ID,
		Hostname:    host.Hostname,
		DisplayName: host.DisplayName,
		URL:         u.String(),
	}
}
