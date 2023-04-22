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
	hosts []fleet.HostVulnerabilitySummary,
	cve string,
	meta fleet.CVEMeta,
) fleetwebhooks.WebhookPayload {
	r := m.Mapper.GetPayload(hostBaseURL,
		hosts,
		cve,
		meta,
	)
	r.EPSSProbability = meta.EPSSProbability
	r.CVSSScore = meta.CVSSScore
	r.CISAKnownExploit = meta.CISAKnownExploit
	r.CVEPublished = meta.Published
	return r
}
