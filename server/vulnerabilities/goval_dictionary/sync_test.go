package goval_dictionary

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	"github.com/stretchr/testify/require"
)

func TestSync(t *testing.T) {
	t.Run("#whatToDownload", func(t *testing.T) {
		osVersions := fleet.OSVersions{
			CountsUpdatedAt: time.Now(),
			OSVersions: []fleet.OSVersion{
				{
					HostsCount: 1,
					Platform:   "ubuntu",
					Name:       "Ubuntu 20.4.0",
				},
				{
					HostsCount: 1,
					Platform:   "amzn",
					Name:       "Amazon Linux 2.0.0",
				},
				{
					HostsCount: 1,
					Platform:   "amzn",
					Name:       "Amazon Linux 2022.0.20221207",
				},
				{
					HostsCount: 1,
					Platform:   "amzn",
					Name:       "Amazon Linux 2023.5.20240624",
				},
			},
		}

		result := whatToDownload(&osVersions)
		require.ElementsMatch(t, []oval.Platform{"amzn_02", "amzn_2023"}, result)
	})

	t.Run("#whatToDownload skips host-supplied traversal platforms", func(t *testing.T) {
		osVersions := fleet.OSVersions{
			OSVersions: []fleet.OSVersion{
				{Platform: "amzn_01/../../pwned-a/f", Name: "Amazon Linux 1.0.0"},
				{Platform: "amzn", Name: "Amazon Linux 2023.0.0"},
				{Platform: "amzn_01/../fleet_goval_dictionary_amzn", Name: "Amazon Linux 2023.0.0"},
				{Platform: "rhel", Name: "Red Hat Enterprise Linux 9.0.0"},
			},
		}

		result := whatToDownload(&osVersions)
		require.ElementsMatch(t, []oval.Platform{"amzn_2023", "rhel_09"}, result)
	})
}
