package upgrade

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUpgradeAToB(t *testing.T) {
	versionA := os.Getenv("FLEET_VERSION_A")
	if versionA == "" {
		t.Skip("Missing environment variable FLEET_VERSION_A")
	}

	versionB := os.Getenv("FLEET_VERSION_B")
	if versionB == "" {
		t.Skip("Missing environment variable FLEET_VERSION_B")
	}

	f := NewFleet(t, versionA)

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
	}, 5*time.Minute, 5*time.Second)

	err = f.Upgrade(versionB)
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
	}, 5*time.Minute, 5*time.Second)
}
