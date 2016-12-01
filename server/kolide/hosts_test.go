package kolide

import (
	"testing"

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
