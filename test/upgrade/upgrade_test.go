package upgrade

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUpgradeAToB(t *testing.T) {
	f := NewFleet(t, "v4.15.0")

	client, err := f.Client()
	require.NoError(t, err)

	// enroll a host
	hostname, err := f.StartHost()
	require.NoError(t, err)

	// wait until host is enrolled and software is listed
	require.Eventually(t, func() bool {
		host, err := client.HostByIdentifier(hostname)
		if err != nil {
			t.Logf("get host: %v", err)
			return false
		}

		if len(host.Software) == 0 {
			return false
		}

		return true
	}, 5 * time.Minute, 5 * time.Second)

	err = f.Upgrade("v4.16.0")
	require.NoError(t, err)

	// enroll another host with the new version
	hostname, err = f.StartHost()
	require.NoError(t, err)

	// wait until host is enrolled and software is listed
	require.Eventually(t, func() bool {
		host, err := client.HostByIdentifier(hostname)
		if err != nil {
			t.Logf("get host: %v", err)
			return false
		}

		if len(host.Software) == 0 {
			return false
		}

		return true
	}, 5 * time.Minute, 5 * time.Second)
}
