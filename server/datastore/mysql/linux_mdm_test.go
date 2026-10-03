package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestLinuxDiskEncryptionSummary(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	// 5 new ubuntu hosts
	var ubuntuHosts []*fleet.Host
	for i := 0; i < 5; i++ {
		h := test.NewHost(t, ds, fmt.Sprintf("foo.local.%d", i), "1.1.1.1",
			fmt.Sprintf("%d", i), fmt.Sprintf("%d", i), time.Now(), test.WithPlatform("ubuntu"))
		ubuntuHosts = append(ubuntuHosts, h)
	}

	// 5 new fedora hosts
	var fedoraHosts []*fleet.Host
	for i := 5; i < 10; i++ {
		h := test.NewHost(t, ds, fmt.Sprintf("foo.local.%d", i), "1.1.1.1",
			fmt.Sprintf("%d", i), fmt.Sprintf("%d", i), time.Now(),
			test.WithOSVersion("Fedora Linux 38.0.0"), test.WithPlatform("rhel"))
		fedoraHosts = append(fedoraHosts, h)
	}

	// 5 macos hosts
	var macosHosts []*fleet.Host
	for i := 10; i < 15; i++ {
		h := test.NewHost(t, ds, fmt.Sprintf("foo.local.%d", i), "1.1.1.1",
			fmt.Sprintf("%d", i), fmt.Sprintf("%d", i), time.Now(), test.WithPlatform("darwin"))
		macosHosts = append(macosHosts, h)
	}

	// no teams tests =====
	summary, err := ds.GetLinuxDiskEncryptionSummary(ctx, nil)
	require.NoError(t, err)

	require.Equal(t, uint(0), summary.Verified)
	require.Equal(t, uint(10), summary.ActionRequired)
	require.Equal(t, uint(0), summary.Failed)

	// Add disk encryption keys

	// ubuntu
	keyArchived, err := ds.SetOrUpdateHostDiskEncryptionKey(ctx, ubuntuHosts[0], "base64_encrypted", "", nil)
	require.NoError(t, err)
	require.True(t, keyArchived)
	// fedora
	keyArchived, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, fedoraHosts[0], "base64_encrypted", "", nil)
	require.NoError(t, err)
	require.True(t, keyArchived)
	// macos
	keyArchived, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, macosHosts[0], "base64_encrypted", "", nil)
	require.NoError(t, err)
	require.True(t, keyArchived)

	summary, err = ds.GetLinuxDiskEncryptionSummary(ctx, nil)
	require.NoError(t, err)

	require.Equal(t, uint(2), summary.Verified)
	require.Equal(t, uint(8), summary.ActionRequired)
	require.Equal(t, uint(0), summary.Failed)

	// update ubuntu with key and client error
	keyArchived, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, ubuntuHosts[0], "base64_encrypted", "client error", nil)
	require.NoError(t, err)
	require.False(t, keyArchived)

	summary, err = ds.GetLinuxDiskEncryptionSummary(ctx, nil)
	require.NoError(t, err)

	require.Equal(t, uint(1), summary.Verified)
	require.Equal(t, uint(8), summary.ActionRequired)
	require.Equal(t, uint(1), summary.Failed)

	// add ubuntu with no key and client error
	keyArchived, err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, ubuntuHosts[1], "", "client error", nil)
	require.NoError(t, err)
	require.False(t, keyArchived)

	summary, err = ds.GetLinuxDiskEncryptionSummary(ctx, nil)
	require.NoError(t, err)

	require.Equal(t, uint(1), summary.Verified)
	require.Equal(t, uint(7), summary.ActionRequired)
	require.Equal(t, uint(2), summary.Failed)

	// move verified fedora host to team will remove existing key
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{fedoraHosts[0].ID}))
	require.NoError(t, err)

	// team summary
	summary, err = ds.GetLinuxDiskEncryptionSummary(ctx, &team.ID)
	require.NoError(t, err)

	require.Equal(t, uint(0), summary.Verified)
	require.Equal(t, uint(1), summary.ActionRequired)
	require.Equal(t, uint(0), summary.Failed)

	// no team summary
	summary, err = ds.GetLinuxDiskEncryptionSummary(ctx, nil)
	require.NoError(t, err)

	require.Equal(t, uint(0), summary.Verified)
	require.Equal(t, uint(7), summary.ActionRequired)
	require.Equal(t, uint(2), summary.Failed)

	// move all hosts to team
	for _, h := range ubuntuHosts {
		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{h.ID}))
		require.NoError(t, err)
	}

	for _, h := range fedoraHosts {
		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{h.ID}))
		require.NoError(t, err)
	}

	for _, h := range macosHosts {
		err = ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{h.ID}))
		require.NoError(t, err)
	}

	// team summary
	summary, err = ds.GetLinuxDiskEncryptionSummary(ctx, &team.ID)
	require.NoError(t, err)

	require.Equal(t, uint(0), summary.Verified)
	require.Equal(t, uint(10), summary.ActionRequired)
	require.Equal(t, uint(0), summary.Failed)

	// no team summary
	summary, err = ds.GetLinuxDiskEncryptionSummary(ctx, nil)
	require.NoError(t, err)

	require.Equal(t, uint(0), summary.Verified)
	require.Equal(t, uint(0), summary.ActionRequired)
	require.Equal(t, uint(0), summary.Failed)
}

func TestLinuxDiskEncryptionSupportedPlatforms(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	setAppConfigDiskEncryptionForTest(ac, true)
	require.NoError(t, ds.SaveAppConfig(ctx, ac))

	supported := []struct{ platform, osVersion string }{
		{"ubuntu", "Ubuntu 24.04.1 LTS"},
		{"zorin", "Zorin OS 17.2"},
		{"rhel", "Fedora Linux 41.0.0"},
		{"arch", "Arch Linux rolling"},
		{"archarm", "Arch Linux ARM rolling"},
		{"manjaro", "Manjaro Linux 25.0.0"},
		{"manjaro-arm", "Manjaro ARM 25.0.0"},
		{"cachyos", "CachyOS Linux rolling"},
		{"omarchy", "Omarchy 4.0.0"},
	}
	unsupported := []struct{ platform, osVersion string }{
		{"rhel", "CentOS Linux 7.9.2009"},
		{"debian", "Debian GNU/Linux 12"},
		{"pop", "Pop!_OS 22.04 LTS"},
		{"darwin", "macOS 15.1"},
	}

	var supportedIDs []uint
	for i, p := range append(supported, unsupported...) {
		h := test.NewHost(t, ds, fmt.Sprintf("luks.local.%d", i), "1.1.1.1", fmt.Sprintf("luks-%d", i), fmt.Sprintf("luks-%d", i),
			time.Now(), test.WithPlatform(p.platform), test.WithOSVersion(p.osVersion))
		if i < len(supported) {
			supportedIDs = append(supportedIDs, h.ID)
		}
	}

	summary, err := ds.GetLinuxDiskEncryptionSummary(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, fleet.MDMLinuxDiskEncryptionSummary{ActionRequired: uint(len(supported))}, summary)

	listIDs := func(opts fleet.HostListOptions) []uint {
		hosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, opts)
		require.NoError(t, err)
		ids := make([]uint, 0, len(hosts))
		for _, h := range hosts {
			ids = append(ids, h.ID)
		}
		return ids
	}
	require.ElementsMatch(t, supportedIDs, listIDs(fleet.HostListOptions{OSSettingsFilter: fleet.OSSettingsPending}))
	require.ElementsMatch(t, supportedIDs, listIDs(fleet.HostListOptions{OSSettingsDiskEncryptionFilter: fleet.DiskEncryptionActionRequired}))
}
