package webhooks

import (
	"net/url"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestGetPaylaod(t *testing.T) {
	serverURL, err := url.Parse("http://mywebsite.com")
	require.NoError(t, err)

	vuln := fleet.SoftwareVulnerability{
		CVE:        "cve-1",
		SoftwareID: 1,
	}
	meta := fleet.CVEMeta{
		CVE:              "cve-1",
		CVSSScore:        ptr.Float64(1),
		EPSSProbability:  ptr.Float64(0.5),
		CISAKnownExploit: ptr.Bool(true),
	}

	sut := Mapper{}

	result := sut.GetPayload(serverURL, nil, vuln.CVE, meta)
	require.Empty(t, result.CISAKnownExploit)
	require.Empty(t, result.EPSSProbability)
	require.Empty(t, result.CVSSScore)
}
