package webhooks

import (
	"fmt"
	"net/url"
	"path"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// VulnMapper used for mapping vulnerabilities and their associated data into the payload that
// will be sent via thrid party webhooks.
type VulnMapper interface {
	GetPayload(*url.URL, []*fleet.HostShort, string, fleet.CVEMeta) WebhookPayload
}

type hostPayloadPart struct {
	ID          uint   `json:"id"`
	Hostname    string `json:"hostname"`
	DisplayName string `json:"display_name"`
	URL         string `json:"url"`
}

type WebhookPayload struct {
	CVE              string             `json:"cve"`
	Link             string             `json:"details_link"`
	EPSSProbability  *float64           `json:"epss_probability,omitempty"`   // Premium feature only
	CVSSScore        *float64           `json:"cvss_score,omitempty"`         // Premium feature only
	CISAKnownExploit *bool              `json:"cisa_known_exploit,omitempty"` // Premium feature only
	Hosts            []*hostPayloadPart `json:"hosts_affected"`
}

type Mapper struct{}

func NewMapper() VulnMapper {
	return &Mapper{}
}

func (m *Mapper) getHostPayloadPart(
	hostBaseURL *url.URL,
	hosts []*fleet.HostShort,
) []*hostPayloadPart {
	shortHosts := make([]*hostPayloadPart, len(hosts))
	for i, h := range hosts {
		hostURL := *hostBaseURL
		hostURL.Path = path.Join(hostURL.Path, "hosts", strconv.Itoa(int(h.ID)))
		shortHosts[i] = &hostPayloadPart{
			ID:          h.ID,
			Hostname:    h.Hostname,
			DisplayName: h.DisplayName,
			URL:         hostURL.String(),
		}
	}
	return shortHosts
}

func (m *Mapper) GetPayload(
	hostBaseURL *url.URL,
	hosts []*fleet.HostShort,
	cve string,
	meta fleet.CVEMeta,
) WebhookPayload {
	return WebhookPayload{
		CVE:   cve,
		Link:  fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", cve),
		Hosts: m.getHostPayloadPart(hostBaseURL, hosts),
	}
}
