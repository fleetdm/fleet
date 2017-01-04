package kolide

import (
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/stretchr/testify/assert"
)

func TestResetHosts(t *testing.T) {
	host := Host{}
	result := host.ResetPrimaryNetwork()
	assert.False(t, result)

	host.NetworkInterfaces = []*NetworkInterface{
		&NetworkInterface{
			ID:        1,
			IPAddress: "192.168.1.2",
		},
		&NetworkInterface{
			ID:        2,
			IPAddress: "192.168.1.3",
		},
	}

	result = host.ResetPrimaryNetwork()
	assert.True(t, result)
	assert.Equal(t, uint(1), *host.PrimaryNetworkInterfaceID)

	host.PrimaryNetworkInterfaceID = &host.NetworkInterfaces[1].ID
	result = host.ResetPrimaryNetwork()
	assert.False(t, result)
	assert.Equal(t, uint(2), *host.PrimaryNetworkInterfaceID)

	host.NetworkInterfaces = host.NetworkInterfaces[:1]
	result = host.ResetPrimaryNetwork()
	assert.True(t, result)
	assert.Equal(t, uint(1), *host.PrimaryNetworkInterfaceID)

	host.NetworkInterfaces = []*NetworkInterface{}
	result = host.ResetPrimaryNetwork()
	assert.True(t, result)
	assert.Nil(t, host.PrimaryNetworkInterfaceID)
}

func TestHostStatus(t *testing.T) {
	mockClock := clock.NewMockClock()

	host := Host{}

	host.SeenTime = mockClock.Now()
	assert.Equal(t, StatusOnline, host.Status(mockClock.Now()))

	host.SeenTime = mockClock.Now().Add(-1 * time.Minute)
	assert.Equal(t, StatusOnline, host.Status(mockClock.Now()))

	host.SeenTime = mockClock.Now().Add(-1 * time.Hour)
	assert.Equal(t, StatusOffline, host.Status(mockClock.Now()))

	host.SeenTime = mockClock.Now().Add(-35 * (24 * time.Hour)) // 35 days
	assert.Equal(t, StatusMIA, host.Status(mockClock.Now()))
}
