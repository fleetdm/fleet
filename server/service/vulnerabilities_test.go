package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestListVulnerabilities(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	t.Run("list all vulnerabilities", func(t *testing.T) {
		expected := []fleet.VulnerabilityWithMetadata{
			{
				CVEMeta: fleet.CVEMeta{
					CVE:         "CVE-2019-1234",
					Description: "A vulnerability",
				},
				CreatedAt:          time.Now(),
				HostCount:          10,
				HostCountUpdatedAt: time.Now(),
			},
		}

		ds.ListVulnerabilitiesFunc = func(cxt context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, error) {
			return expected, nil
		}

		_, err := svc.ListVulnerabilities(ctx, fleet.VulnListOptions{})
		require.NoError(t, err)
	})
}
