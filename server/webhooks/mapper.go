package webhooks

import (
	"fmt"
	"net/url"
	"path"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// WebhookMapper used for mapping vulnerabilities and their associated data into the payload that
// will be sent via thrid party webhooks.
type WebhookMapper interface {
	GetPayload(*url.URL, []*fleet.HostShort, fleet.SoftwareVulnerability, fleet.CVEMeta) WebhookPayload
}

type hostPayloadPart struct {
	ID        uint   `json:"id"`
	Hostname  string `json:"hostname"`
	URL       string `json:"url"`
	Platform  string `json:"platform"`
	OSVersion string `json:"os_version"`
}

type WebhookPayload struct {
	CVE              string             `json:"cve"`
	Link             string             `json:"details_link"`
	EPSSProbability  *float64           `json:"epss_probability,omitempty"`   // Premium feature only
	CVSSScore        *float64           `json:"cvss_score,omitempty"`         // Premium feature only
	CISAKnownExploit *bool              `json:"cisa_known_exploit,omitempty"` // Premium feature only
	Hosts            []*hostPayloadPart `json:"hosts_affected"`
}

type FreeMapper struct{}

func NewWebhookFreeMapper() WebhookMapper {
	return &FreeMapper{}
}

func (m *FreeMapper) getHostPayloadPart(
	hostBaseURL *url.URL,
	hosts []*fleet.HostShort,
) []*hostPayloadPart {
	shortHosts := make([]*hostPayloadPart, len(hosts))
	for i, h := range hosts {
		hostURL := *hostBaseURL
		hostURL.Path = path.Join(hostURL.Path, "hosts", strconv.Itoa(int(h.ID)))
		shortHosts[i] = &hostPayloadPart{
			ID:        h.ID,
			Hostname:  h.Hostname,
			URL:       hostURL.String(),
			Platform:  h.Platform,
			OSVersion: h.OSVersion,
		}
	}
	return shortHosts
}

func (m *FreeMapper) GetPayload(
	hostBaseURL *url.URL,
	hosts []*fleet.HostShort,
	vuln fleet.SoftwareVulnerability,
	meta fleet.CVEMeta,
) WebhookPayload {
	return WebhookPayload{
		CVE:   vuln.CVE,
		Link:  fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", vuln.CVE),
		Hosts: m.getHostPayloadPart(hostBaseURL, hosts),
	}
}
