package upgrade

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func enrollHost(t *testing.T, f *Fleet) string {
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

	return hostname
}

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

	enrollHost(t, f)

	err := f.Upgrade(versionB)
	require.NoError(t, err)

	// enroll another host with the new version
	enrollHost(t, f)
}
