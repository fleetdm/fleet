package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatistics(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ShouldSend", testStatisticsShouldSend},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testStatisticsShouldSend(t *testing.T, ds *Datastore) {
	// Create new host for test
	_, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   "M",
	})
	require.NoError(t, err)

	// Create new user for test
	_, err = ds.NewUser(context.Background(), &fleet.User{
		Password:                 []byte("foobar"),
		AdminForcedPasswordReset: false,
		Email:                    "baz@example.com",
		SSOEnabled:               false,
		GlobalRole:               ptr.String(fleet.RoleObserver),
	})
	require.NoError(t, err)

	// Create new team for test
	_, err = ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "footeam",
		Description: "team of foo",
	})
	require.NoError(t, err)

	// Create new global policy for test
	_, err = ds.NewGlobalPolicy(context.Background(), ptr.Uint(1), fleet.PolicyPayload{
		Name:        "testpolicy",
		Query:       "select 1;",
		Description: "test policy desc",
		Resolution:  "test policy resolution",
	})
	require.NoError(t, err)

	// Create new app config for test
	config, err := ds.NewAppConfig(context.Background(), &fleet.AppConfig{
		OrgInfo: fleet.OrgInfo{
			OrgName:    "Test",
			OrgLogoURL: "localhost:8080/logo.png",
		},
	})
	require.NoError(t, err)
	config.HostSettings.EnableSoftwareInventory = false
	config.HostSettings.EnableHostUsers = false
	config.VulnerabilitySettings.DatabasesPath = "foo/bar"
	config.WebhookSettings.HostStatusWebhook.Enable = true

	err = ds.SaveAppConfig(context.Background(), config)
	require.NoError(t, err)

	license := &fleet.LicenseInfo{Tier: "premium"}

	// First time running, we send statistics
	stats, shouldSend, err := ds.ShouldSendStatistics(context.Background(), fleet.StatisticsFrequency, license)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.NotEmpty(t, stats.AnonymousIdentifier)
	assert.NotEmpty(t, stats.FleetVersion)
	assert.Equal(t, stats.LicenseTier, "premium")
	assert.Equal(t, stats.NumHostsEnrolled, 1)
	assert.Equal(t, stats.NumUsers, 1)
	assert.Equal(t, stats.NumTeams, 1)
	assert.Equal(t, stats.NumPolicies, 1)
	assert.Equal(t, stats.SoftwareInventoryEnabled, false)
	assert.Equal(t, stats.SystemUsersEnabled, false)
	assert.Equal(t, stats.VulnDetectionEnabled, true)
	assert.Equal(t, stats.HostsStatusWebHookEnabled, true)

	firstIdentifier := stats.AnonymousIdentifier

	err = ds.RecordStatisticsSent(context.Background())
	require.NoError(t, err)

	// If we try right away, it shouldn't ask to send
	stats, shouldSend, err = ds.ShouldSendStatistics(context.Background(), fleet.StatisticsFrequency, license)
	require.NoError(t, err)
	assert.False(t, shouldSend)

	time.Sleep(2)

	_, err = ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "2",
		UUID:            "2",
		Hostname:        "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		OsqueryHostID:   "S",
	})
	require.NoError(t, err)

	// Lower the frequency to trigger an "outdated" sent
	stats, shouldSend, err = ds.ShouldSendStatistics(context.Background(), time.Millisecond, license)
	require.NoError(t, err)
	assert.True(t, shouldSend)
	assert.Equal(t, firstIdentifier, stats.AnonymousIdentifier)
	assert.Equal(t, stats.NumHostsEnrolled, 2)
}
