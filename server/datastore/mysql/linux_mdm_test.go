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
	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, ubuntuHosts[0], "base64_encrypted", "", nil)
	require.NoError(t, err)
	// fedora
	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, fedoraHosts[0], "base64_encrypted", "", nil)
	require.NoError(t, err)
	// macos
	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, macosHosts[0], "base64_encrypted", "", nil)
	require.NoError(t, err)

	summary, err = ds.GetLinuxDiskEncryptionSummary(ctx, nil)
	require.NoError(t, err)

	require.Equal(t, uint(2), summary.Verified)
	require.Equal(t, uint(8), summary.ActionRequired)
	require.Equal(t, uint(0), summary.Failed)

	// update ubuntu with key and client error
	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, ubuntuHosts[0], "base64_encrypted", "client error", nil)
	require.NoError(t, err)

	summary, err = ds.GetLinuxDiskEncryptionSummary(ctx, nil)
	require.NoError(t, err)

	require.Equal(t, uint(1), summary.Verified)
	require.Equal(t, uint(8), summary.ActionRequired)
	require.Equal(t, uint(1), summary.Failed)

	// add ubuntu with no key and client error
	err = ds.SetOrUpdateHostDiskEncryptionKey(ctx, ubuntuHosts[1], "", "client error", nil)
	require.NoError(t, err)

	summary, err = ds.GetLinuxDiskEncryptionSummary(ctx, nil)
	require.NoError(t, err)

	require.Equal(t, uint(1), summary.Verified)
	require.Equal(t, uint(7), summary.ActionRequired)
	require.Equal(t, uint(2), summary.Failed)

	// move verified fedora host to team will remove existing key
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	err = ds.AddHostsToTeam(ctx, &team.ID, []uint{fedoraHosts[0].ID})
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
		err = ds.AddHostsToTeam(ctx, &team.ID, []uint{h.ID})
		require.NoError(t, err)
	}

	for _, h := range fedoraHosts {
		err = ds.AddHostsToTeam(ctx, &team.ID, []uint{h.ID})
		require.NoError(t, err)
	}

	for _, h := range macosHosts {
		err = ds.AddHostsToTeam(ctx, &team.ID, []uint{h.ID})
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
