package webhooks

import (
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetwebhooks "github.com/fleetdm/fleet/v4/server/webhooks"
)

type Mapper struct {
	fleetwebhooks.Mapper
}

func NewMapper() fleetwebhooks.VulnMapper {
	return &Mapper{}
}

func (m *Mapper) GetPayload(
	hostBaseURL *url.URL,
	hosts []*fleet.HostShort,
	vuln fleet.SoftwareVulnerability,
	meta fleet.CVEMeta,
) fleetwebhooks.WebhookPayload {
	r := m.Mapper.GetPayload(hostBaseURL,
		hosts,
		vuln,
		meta,
	)
	r.EPSSProbability = meta.EPSSProbability
	r.CVSSScore = meta.CVSSScore
	r.CISAKnownExploit = meta.CISAKnownExploit
	return r
}
