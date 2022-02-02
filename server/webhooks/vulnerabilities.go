package webhooks

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// TriggerVulnerabilitiesWebhook performs the webhook requests for vulnerabilities.
func TriggerVulnerabilitiesWebhook(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	recentVulns map[string][]string,
	appConfig *fleet.AppConfig,
	now time.Time,
) error {
	vulnConfig := appConfig.WebhookSettings.VulnerabilitiesWebhook
	if !vulnConfig.Enable {
		return nil
	}

	level.Debug(logger).Log("enabled", "true", "recentVulns", len(recentVulns))

	serverURL, err := url.Parse(appConfig.ServerSettings.ServerURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "invalid server url")
	}

	targetURL := vulnConfig.DestinationURL
	batchSize := vulnConfig.HostBatchSize

	for cve, cpes := range recentVulns {
		hosts, err := ds.HostsByCPEs(ctx, cpes)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get hosts by CPE")
		}

		for len(hosts) > 0 {
			limit := len(hosts)
			if batchSize > 0 && len(hosts) > batchSize {
				limit = batchSize
			}
			if err := sendVulnerabilityHostBatch(ctx, targetURL, cve, serverURL, hosts[:limit], now); err != nil {
				return ctxerr.Wrap(ctx, err, "send vulnerability host batch")
			}
			hosts = hosts[limit:]
		}
	}

	return nil
}

type vulnHostPayload struct {
	ID       uint   `json:"id"`
	Hostname string `json:"hostname"`
	URL      string `json:"url"`
}

func sendVulnerabilityHostBatch(ctx context.Context, targetURL, cve string, hostBaseURL *url.URL, hosts []*fleet.CPEHost, now time.Time) error {
	shortHosts := make([]*vulnHostPayload, len(hosts))
	for i, h := range hosts {
		hostURL := *hostBaseURL
		hostURL.Path = path.Join(hostURL.Path, "hosts", strconv.Itoa(int(h.ID)))
		shortHosts[i] = &vulnHostPayload{
			ID:       h.ID,
			Hostname: h.Hostname,
			URL:      hostURL.String(),
		}
	}

	payload := map[string]interface{}{
		"timestamp": now,
		"vulnerability": map[string]interface{}{
			"cve":            cve,
			"details_link":   fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cve),
			"hosts_affected": shortHosts,
		},
	}

	if err := server.PostJSONWithTimeout(ctx, targetURL, &payload); err != nil {
		return ctxerr.Wrapf(ctx, err, "posting to %s", targetURL)
	}
	return nil
}
