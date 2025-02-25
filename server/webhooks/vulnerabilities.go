package webhooks

import (
	"context"
	"net/url"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// TriggerVulnerabilitiesWebhook performs the webhook requests for vulnerabilities.
func TriggerVulnerabilitiesWebhook(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	args VulnArgs,
	mapper VulnMapper,
) error {
	vulnConfig := args.AppConfig.WebhookSettings.VulnerabilitiesWebhook

	if !vulnConfig.Enable {
		return nil
	}

	level.Debug(logger).Log("enabled", "true", "recentVulns", len(args.Vulnerablities))

	serverURL, err := url.Parse(args.AppConfig.ServerSettings.ServerURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "invalid server url")
	}

	targetURL := vulnConfig.DestinationURL
	batchSize := vulnConfig.HostBatchSize

	cveGrouped := make(map[string][]uint)
	for _, v := range args.Vulnerablities {
		cveGrouped[v.GetCVE()] = append(cveGrouped[v.GetCVE()], v.Affected())
	}

	for cve, sIDs := range cveGrouped {
		hosts, err := ds.HostVulnSummariesBySoftwareIDs(ctx, sIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get hosts by software ids")
		}

		for len(hosts) > 0 {
			limit := len(hosts)
			if batchSize > 0 && len(hosts) > batchSize {
				limit = batchSize
			}
			payload := mapper.GetPayload(serverURL, hosts[:limit], cve, args.Meta[cve])
			if err := sendVulnerabilityHostBatch(ctx, targetURL, payload, args.Time); err != nil {
				return ctxerr.Wrap(ctx, err, "send vulnerability host batch")
			}
			hosts = hosts[limit:]
		}
	}

	return nil
}

func sendVulnerabilityHostBatch(ctx context.Context, targetURL string, vuln WebhookPayload, now time.Time) error {
	payload := map[string]interface{}{
		"timestamp":     now,
		"vulnerability": vuln,
	}

	if err := server.PostJSONWithTimeout(ctx, targetURL, &payload); err != nil {
		return ctxerr.Wrapf(ctx, err, "posting to %s", targetURL)
	}
	return nil
}
