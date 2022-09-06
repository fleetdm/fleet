package webhooks

import (
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetwebhooks "github.com/fleetdm/fleet/v4/server/webhooks"
)

type EEMapper struct {
	fleetwebhooks.FreeMapper
}

func NewWebhookEEMapper() fleetwebhooks.WebhookMapper {
	return &EEMapper{}
}

func (m *EEMapper) GetPayload(
	hostBaseURL *url.URL,
	hosts []*fleet.HostShort,
	vuln fleet.SoftwareVulnerability,
	meta fleet.CVEMeta,
) fleetwebhooks.WebhookPayload {
	r := m.FreeMapper.GetPayload(hostBaseURL,
		hosts,
		vuln,
		meta,
	)
	r.EPSSProbability = meta.EPSSProbability
	r.CVSSScore = meta.CVSSScore
	r.CISAKnownExploit = meta.CISAKnownExploit
	return r
}
