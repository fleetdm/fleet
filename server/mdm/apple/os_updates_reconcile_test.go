package apple_mdm

import (
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// TestComputeOSUpdatesTarget covers the in-memory decision logic that maps each
// host to its target OS version / deadline / resend flag. It exercises the
// branches independently of the datastore and GDMF: platform gate, removal when
// a host's team no longer has "latest", version selection, deadline derivation,
// and the unchanged-target short circuit.
func TestComputeOSUpdatesTarget(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	ctx := t.Context()

	const (
		macDevice = "Mac14,2"
		iosDevice = "iPhone15,2"
	)

	// Two macOS assets for the same device, higher version listed first so the
	// test proves the reconciler picks the max version, not the last one seen.
	updateAssets := map[string][]fleet.AppleSoftwareUpdateAsset{
		"macos": {
			{ProductVersion: "15.1", PostingDate: "2024-10-28", SupportedDevices: []string{macDevice}},
			{ProductVersion: "14.6.1", PostingDate: "2024-08-07", SupportedDevices: []string{macDevice}},
		},
		"ios": {
			{ProductVersion: "18.1", PostingDate: "2024-10-28", SupportedDevices: []string{iosDevice}},
		},
	}

	// deadline = posting date + deadlineDays. Posting dates parse as midnight UTC.
	macDeadline := time.Date(2024, 10, 30, 0, 0, 0, 0, time.UTC) // 2024-10-28 + 2d
	iosDeadline := time.Date(2024, 10, 30, 0, 0, 0, 0, time.UTC) // 2024-10-28 + 2d

	// teamsWithLatest is keyed by the three supported platforms; nil inner maps
	// mean "no team on that platform has latest configured".
	latest := func(darwin, ios, ipados map[uint]uint) map[string]map[uint]uint {
		return map[string]map[uint]uint{"darwin": darwin, "ios": ios, "ipados": ipados}
	}
	compute := func(host *fleet.AppleSoftwareUpdateHost, teams map[string]map[uint]uint) []*fleet.ComputedAppleSoftwareUpdateHost {
		return computeOSUpdatesTarget(ctx, logger, []*fleet.AppleSoftwareUpdateHost{host}, updateAssets, teams)
	}

	t.Run("macOS host in a team with latest gets highest version and posting-date deadline", func(t *testing.T) {
		host := &fleet.AppleSoftwareUpdateHost{HostUUID: "h1", Platform: "darwin", TeamID: 0, SoftwareUpdateDeviceID: macDevice}
		got := compute(host, latest(map[uint]uint{0: 2}, nil, nil))
		require.Len(t, got, 1)
		require.True(t, got[0].Resend)
		require.Equal(t, "15.1", got[0].TargetOSVersion)
		require.NotNil(t, got[0].TargetDeadline)
		require.True(t, macDeadline.Equal(*got[0].TargetDeadline), "want %s got %s", macDeadline, got[0].TargetDeadline)
		require.NotNil(t, got[0].ResolvedAt)
	})

	t.Run("iPadOS host resolves against the shared ios asset set", func(t *testing.T) {
		host := &fleet.AppleSoftwareUpdateHost{HostUUID: "h2", Platform: "ipados", TeamID: 3, SoftwareUpdateDeviceID: iosDevice}
		got := compute(host, latest(nil, nil, map[uint]uint{3: 2}))
		require.Len(t, got, 1)
		require.True(t, got[0].Resend)
		require.Equal(t, "18.1", got[0].TargetOSVersion)
		require.True(t, iosDeadline.Equal(*got[0].TargetDeadline))
	})

	t.Run("first_seen_at after posting date drives the deadline", func(t *testing.T) {
		firstSeen := time.Date(2024, 11, 5, 12, 0, 0, 0, time.UTC)
		assets := map[string][]fleet.AppleSoftwareUpdateAsset{
			"macos": {{ProductVersion: "15.1", PostingDate: "2024-10-28", FirstSeenAt: firstSeen, SupportedDevices: []string{macDevice}}},
		}
		host := &fleet.AppleSoftwareUpdateHost{HostUUID: "h3", Platform: "darwin", TeamID: 0, SoftwareUpdateDeviceID: macDevice}
		got := computeOSUpdatesTarget(ctx, logger, []*fleet.AppleSoftwareUpdateHost{host}, assets, latest(map[uint]uint{0: 2}, nil, nil))
		require.Len(t, got, 1)
		require.True(t, firstSeen.Add(2*24*time.Hour).Equal(*got[0].TargetDeadline))
	})

	t.Run("host whose team no longer has latest is cleared without resend", func(t *testing.T) {
		deadline := macDeadline
		host := &fleet.AppleSoftwareUpdateHost{
			HostUUID: "h4", Platform: "darwin", TeamID: 5, SoftwareUpdateDeviceID: macDevice,
			TargetOSVersion: "15.1", TargetDeadline: &deadline, ResolvedAt: &deadline,
		}
		// team 5 is not present in the darwin latest set.
		got := compute(host, latest(map[uint]uint{0: 2}, nil, nil))
		require.Len(t, got, 1)
		require.False(t, got[0].Resend)
		require.Empty(t, got[0].TargetOSVersion)
		require.Nil(t, got[0].TargetDeadline)
		require.Nil(t, got[0].ResolvedAt)
	})

	t.Run("unchanged target and deadline are skipped", func(t *testing.T) {
		deadline := macDeadline
		host := &fleet.AppleSoftwareUpdateHost{
			HostUUID: "h5", Platform: "darwin", TeamID: 0, SoftwareUpdateDeviceID: macDevice,
			TargetOSVersion: "15.1", TargetDeadline: &deadline,
		}
		got := compute(host, latest(map[uint]uint{0: 2}, nil, nil))
		require.Empty(t, got)
	})

	t.Run("unsupported platform is skipped", func(t *testing.T) {
		host := &fleet.AppleSoftwareUpdateHost{HostUUID: "h6", Platform: "tvos", TeamID: 0, SoftwareUpdateDeviceID: macDevice}
		got := compute(host, latest(map[uint]uint{0: 2}, nil, nil))
		require.Empty(t, got)
	})

	t.Run("no asset matches the host device id is skipped", func(t *testing.T) {
		host := &fleet.AppleSoftwareUpdateHost{HostUUID: "h7", Platform: "darwin", TeamID: 0, SoftwareUpdateDeviceID: "Mac99,9"}
		got := compute(host, latest(map[uint]uint{0: 2}, nil, nil))
		require.Empty(t, got)
	})

	t.Run("no assets for the platform is skipped", func(t *testing.T) {
		host := &fleet.AppleSoftwareUpdateHost{HostUUID: "h8", Platform: "darwin", TeamID: 0, SoftwareUpdateDeviceID: macDevice}
		got := computeOSUpdatesTarget(ctx, logger, []*fleet.AppleSoftwareUpdateHost{host}, map[string][]fleet.AppleSoftwareUpdateAsset{}, latest(map[uint]uint{0: 2}, nil, nil))
		require.Empty(t, got)
	})
}
