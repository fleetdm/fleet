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
	args VulnArgs,
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

	softwareIDsGroupedByCVE := make(map[string][]uint)
	for _, v := range args.Vulnerablities {
		softwareIDsGroupedByCVE[v.CVE] = append(softwareIDsGroupedByCVE[v.CVE], v.SoftwareID)
	}

	for _, v := range args.Vulnerablities {
		softwareIDs := softwareIDsGroupedByCVE[v.CVE]

		hosts, err := ds.HostsBySoftwareIDs(ctx, softwareIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get hosts by CPE")
		}

		for len(hosts) > 0 {
			limit := len(hosts)
			if batchSize > 0 && len(hosts) > batchSize {
				limit = batchSize
			}

			hostsPayload := getHostPayloadPart(serverURL, hosts[:limit])
			payload := getVulnPayload(v, args.Meta[v.CVE], args.IsPremium, hostsPayload)

			if err := sendVulnerabilityHostBatch(ctx, targetURL, payload, args.Time); err != nil {
				return ctxerr.Wrap(ctx, err, "send vulnerability host batch")
			}

			hosts = hosts[limit:]
		}
	}

	return nil
}

type hostPayloadPart struct {
	ID       uint   `json:"id"`
	Hostname string `json:"hostname"`
	URL      string `json:"url"`
}

func getHostPayloadPart(hostBaseURL *url.URL, hosts []*fleet.HostShort) []*hostPayloadPart {
	shortHosts := make([]*hostPayloadPart, len(hosts))
	for i, h := range hosts {
		hostURL := *hostBaseURL
		hostURL.Path = path.Join(hostURL.Path, "hosts", strconv.Itoa(int(h.ID)))
		shortHosts[i] = &hostPayloadPart{
			ID:       h.ID,
			Hostname: h.Hostname,
			URL:      hostURL.String(),
		}
	}
	return shortHosts
}

type payload struct {
	CVE              string             `json:"cve"`
	Link             string             `json:"details_link"`
	EPSSProbability  *float64           `json:"epss_probability,omitempty"`
	CVSSScore        *float64           `json:"cvss_score,omitempty"`
	CISAKnownExploit *bool              `json:"cisa_known_exploit,omitempty"`
	Hosts            []*hostPayloadPart `json:"hosts_affected"`
}

func getVulnPayload(vuln fleet.SoftwareVulnerability, meta fleet.CVEMeta, isPremium bool, hosts []*hostPayloadPart) payload {
	r := payload{
		CVE:   vuln.CVE,
		Link:  fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", vuln.CVE),
		Hosts: hosts,
	}

	if isPremium {
		r.EPSSProbability = meta.EPSSProbability
		r.CVSSScore = meta.CVSSScore
		r.CISAKnownExploit = meta.CISAKnownExploit
	}

	return r
}

func sendVulnerabilityHostBatch(ctx context.Context, targetURL string, vuln payload, now time.Time) error {
	payload := map[string]interface{}{
		"timestamp":     now,
		"vulnerability": vuln,
	}

	if err := server.PostJSONWithTimeout(ctx, targetURL, &payload); err != nil {
		return ctxerr.Wrapf(ctx, err, "posting to %s", targetURL)
	}
	return nil
}
