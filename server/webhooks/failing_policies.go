package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/url"
	"path"
	"sort"
	"strconv"
	"time"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	fleethttp "github.com/fleetdm/fleet/v4/server/platform/http"
)

// recordWebhookFailedActivity records a failed_automation_webhook
// activity for every host in the failed batch, capturing the remote server's
// status code and response body when available. Failures to record are logged
// and swallowed so they don't mask the original webhook error.
func recordWebhookFailedActivity(
	ctx context.Context,
	newActivitySvc activity_api.NewActivityService,
	policy *fleet.Policy,
	batch []fleet.PolicySetHost,
	postErr error,
	logger *slog.Logger,
) {
	var statusCode int
	if sc, ok := errors.AsType[interface {
		error
		StatusCode() int
	}](postErr); ok {
		statusCode = sc.StatusCode()
	}

	errResponse := ""
	if b, ok := errors.AsType[interface {
		error
		Body() string
	}](postErr); ok {
		errResponse = b.Body()
	}
	if errResponse == "" {
		// network-level failures (e.g. connection refused) have no server
		// response; fall back to the (masked) error message.
		errResponse = fleethttp.MaskURLError(postErr).Error()
	}
	hostIDs := make([]uint, len(batch))
	for i, host := range batch {
		hostIDs[i] = host.ID
	}

	if err := newActivitySvc.NewActivity(ctx, nil, fleet.ActivityTypeFailedAutomationWebhook{
		PolicyID:      policy.ID,
		HostIDList:    hostIDs,
		StatusCode:    statusCode,
		ErrorResponse: errResponse,
	}); err != nil {
		logger.WarnContext(ctx, "failed to record webhook policy automation failure activity",
			"policy_id", policy.ID, "err", err)
	}
}

// recordWebhookRanActivity records a ran_automation_webhook activity
// for every host in a batch whose POST was accepted by the remote server.
// Failures to record are logged and swallowed so they don't affect the send.
func recordWebhookRanActivity(
	ctx context.Context,
	newActivitySvc activity_api.NewActivityService,
	policy *fleet.Policy,
	batch []fleet.PolicySetHost,
	logger *slog.Logger,
) {
	hostIDs := make([]uint, len(batch))
	for i, host := range batch {
		hostIDs[i] = host.ID
	}
	if err := newActivitySvc.NewActivity(ctx, nil, fleet.ActivityTypeRanAutomationWebhook{
		PolicyID:   policy.ID,
		HostIDList: hostIDs,
	}); err != nil {
		logger.WarnContext(ctx, "failed to record webhook policy automation queued activity",
			"policy_id", policy.ID, "err", err)
	}
}

// SendFailingPoliciesBatchedPOSTs sends a failing policy to the provided
// webhook URL. It sends in batches if hostBatchSize > 0. After a successful
// send, the corresponding hosts are removed from the failing policies set.
func SendFailingPoliciesBatchedPOSTs(
	ctx context.Context,
	policy *fleet.Policy,
	failingPoliciesSet fleet.FailingPolicySet,
	hostBatchSize int,
	serverURL *url.URL,
	webhookURL *url.URL,
	now time.Time,
	logger *slog.Logger,
	newActivitySvc activity_api.NewActivityService,
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

		failingHosts := make([]failingHost, len(batch))
		for i, host := range batch {
			failingHosts[i] = makeFailingHost(host, serverURL)
		}

		payload := failingPoliciesPayload{
			Timestamp:    now,
			Policy:       policy,
			FailingHosts: failingHosts,
		}
		logger.DebugContext(ctx, "sending failing policy batch", "payload", payload, "url", fleethttp.MaskSecretURLParams(webhookURL.String()), "batch", len(batch))

		// Marshal and duplicate renamed JSON keys (e.g. fleet_id → also team_id)
		// so that webhook consumers see both the new and deprecated field names.
		jsonBytes, err := json.Marshal(&payload)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshal failing policies payload")
		}
		if rules := endpointer.ExtractAliasRules(payload); len(rules) > 0 {
			jsonBytes = endpointer.DuplicateJSONKeys(jsonBytes, rules, endpointer.DuplicateJSONKeysOpts{Compact: true})
		}

		if err := fleethttp.PostJSONWithTimeout(ctx, webhookURL.String(), json.RawMessage(jsonBytes), logger); err != nil {
			recordWebhookFailedActivity(ctx, newActivitySvc, policy, batch, err, logger)
			return ctxerr.Wrapf(ctx, fleethttp.MaskURLError(err), "posting to %q", fleethttp.MaskSecretURLParams(webhookURL.String()))
		}
		recordWebhookRanActivity(ctx, newActivitySvc, policy, batch, logger)
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
