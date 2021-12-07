package fleet

import (
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostStatus(t *testing.T) {
	mockClock := clock.NewMockClock()

	testCases := []struct {
		seenTime            time.Time
		distributedInterval uint
		configTLSRefresh    uint
		status              HostStatus
	}{
		{mockClock.Now().Add(-30 * time.Second), 10, 3600, StatusOnline},
		{mockClock.Now().Add(-75 * time.Second), 10, 3600, StatusOffline},
		{mockClock.Now().Add(-30 * time.Second), 3600, 10, StatusOnline},
		{mockClock.Now().Add(-75 * time.Second), 3600, 10, StatusOffline},

		{mockClock.Now().Add(-60 * time.Second), 60, 60, StatusOnline},
		{mockClock.Now().Add(-121 * time.Second), 60, 60, StatusOffline},

		{mockClock.Now().Add(-1 * time.Second), 10, 10, StatusOnline},
		{mockClock.Now().Add(-2 * time.Minute), 10, 10, StatusOffline},
		{mockClock.Now().Add(-31 * 24 * time.Hour), 10, 10, StatusMIA},

		// Ensure behavior is reasonable if we don't have the values
		{mockClock.Now().Add(-1 * time.Second), 0, 0, StatusOnline},
		{mockClock.Now().Add(-2 * time.Minute), 0, 0, StatusOffline},
		{mockClock.Now().Add(-31 * 24 * time.Hour), 0, 0, StatusMIA},
	}

	for _, tt := range testCases {
		t.Run("", func(t *testing.T) {
			// Save interval values
			h := Host{
				DistributedInterval: tt.distributedInterval,
				ConfigTLSRefresh:    tt.configTLSRefresh,
				SeenTime:            tt.seenTime,
			}

			assert.Equal(t, tt.status, h.Status(mockClock.Now()))
		})
	}
}

func TestHostIsNew(t *testing.T) {
	mockClock := clock.NewMockClock()

	host := Host{}

	host.CreatedAt = mockClock.Now().AddDate(0, 0, -1)
	assert.True(t, host.IsNew(mockClock.Now()))

	host.CreatedAt = mockClock.Now().AddDate(0, 0, -2)
	assert.False(t, host.IsNew(mockClock.Now()))
}

func TestPlatformFromHost(t *testing.T) {
	for _, tc := range []struct {
		host        string
		expPlatform string
	}{
		{
			host:        "unknown",
			expPlatform: "",
		},
		{
			host:        "",
			expPlatform: "",
		},
		{
			host:        "linux",
			expPlatform: "linux",
		},
		{
			host:        "ubuntu",
			expPlatform: "linux",
		},
		{
			host:        "debian",
			expPlatform: "linux",
		},
		{
			host:        "rhel",
			expPlatform: "linux",
		},
		{
			host:        "centos",
			expPlatform: "linux",
		},
		{
			host:        "sles",
			expPlatform: "linux",
		},
		{
			host:        "kali",
			expPlatform: "linux",
		},
		{
			host:        "gentoo",
			expPlatform: "linux",
		},
		{
			host:        "darwin",
			expPlatform: "darwin",
		},
		{
			host:        "windows",
			expPlatform: "windows",
		},
	} {
		fleetPlatform := PlatformFromHost(tc.host)
		require.Equal(t, tc.expPlatform, fleetPlatform)

	}
}
