package service

import (
	"bytes"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestAppleProfileHandlerNoLabels(t *testing.T) {
	p := &fleet.AppleProfileForReconcile{LabelMode: fleet.AppleProfileLabelModeNone}
	require.True(t, appleProfileHandlerNoLabels(p))
}

func TestAppleProfileHandlerIncludeAll(t *testing.T) {
	t.Run("no labels -> false", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{}
		require.False(t, appleProfileHandlerIncludeAll(p, map[uint]struct{}{1: {}}))
	})
	t.Run("broken label -> false", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{Labels: []fleet.AppleProfileLabelRef{{LabelID: nil}}}
		require.False(t, appleProfileHandlerIncludeAll(p, map[uint]struct{}{1: {}}))
	})
	t.Run("host missing a label -> false", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{Labels: []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1))},
			{LabelID: new(uint(2))},
		}}
		require.False(t, appleProfileHandlerIncludeAll(p, map[uint]struct{}{1: {}}))
	})
	t.Run("host has all labels -> true", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{Labels: []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1))},
			{LabelID: new(uint(2))},
		}}
		require.True(t, appleProfileHandlerIncludeAll(p, map[uint]struct{}{1: {}, 2: {}}))
	})
}

func TestAppleProfileHandlerIncludeAny(t *testing.T) {
	t.Run("no labels -> false", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{}
		require.False(t, appleProfileHandlerIncludeAny(p, map[uint]struct{}{}))
	})
	t.Run("broken labels are ignored", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{Labels: []fleet.AppleProfileLabelRef{
			{LabelID: nil},
			{LabelID: new(uint(2))},
		}}
		require.True(t, appleProfileHandlerIncludeAny(p, map[uint]struct{}{2: {}}))
	})
	t.Run("host in no label -> false", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{Labels: []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1))},
			{LabelID: new(uint(2))},
		}}
		require.False(t, appleProfileHandlerIncludeAny(p, map[uint]struct{}{3: {}}))
	})
	t.Run("host in one label -> true", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{Labels: []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1))},
			{LabelID: new(uint(2))},
		}}
		require.True(t, appleProfileHandlerIncludeAny(p, map[uint]struct{}{1: {}}))
	})
}

func TestAppleProfileHandlerExcludeAny(t *testing.T) {
	host := &fleet.AppleHostReconcileInfo{
		HostID:         100,
		UUID:           "h1",
		Platform:       "darwin",
		LabelUpdatedAt: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
	}

	t.Run("no labels -> false", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{}
		require.False(t, appleProfileHandlerExcludeAny(p, host, map[uint]struct{}{}))
	})

	t.Run("broken label -> false (never apply)", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{Labels: []fleet.AppleProfileLabelRef{
			{LabelID: nil},
		}}
		require.False(t, appleProfileHandlerExcludeAny(p, host, map[uint]struct{}{}))
	})

	t.Run("host is in an excluded label -> false", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{Labels: []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		}}
		require.False(t, appleProfileHandlerExcludeAny(p, host, map[uint]struct{}{1: {}}))
	})

	t.Run("host is not in any excluded label -> true", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{Labels: []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			{LabelID: new(uint(2)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		}}
		require.True(t, appleProfileHandlerExcludeAny(p, host, map[uint]struct{}{99: {}}))
	})

	t.Run("dynamic label created after host's last scan -> false", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{Labels: []fleet.AppleProfileLabelRef{
			{
				LabelID:             new(uint(1)),
				CreatedAt:           time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				LabelMembershipType: 0,
			},
		}}
		require.False(t, appleProfileHandlerExcludeAny(p, host, map[uint]struct{}{}))
	})

	t.Run("manual label created after host's last scan -> still true", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{Labels: []fleet.AppleProfileLabelRef{
			{
				LabelID:             new(uint(1)),
				CreatedAt:           time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				LabelMembershipType: 1,
			},
		}}
		require.True(t, appleProfileHandlerExcludeAny(p, host, map[uint]struct{}{}))
	})
}

func TestAppleProfileAppliesToHost_TeamAndPlatformGates(t *testing.T) {
	host := &fleet.AppleHostReconcileInfo{
		HostID:   1,
		UUID:     "h1",
		TeamID:   nil,
		Platform: "darwin",
	}

	t.Run("wrong team -> false", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{TeamID: 5, LabelMode: fleet.AppleProfileLabelModeNone}
		require.False(t, appleProfileAppliesToHost(p, host, nil))
	})

	t.Run("global team matches nil team_id host", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{TeamID: 0, LabelMode: fleet.AppleProfileLabelModeNone}
		require.True(t, appleProfileAppliesToHost(p, host, nil))
	})

	t.Run("non-apple platform -> false", func(t *testing.T) {
		linuxHost := *host
		linuxHost.Platform = "linux"
		p := &fleet.AppleProfileForReconcile{TeamID: 0, LabelMode: fleet.AppleProfileLabelModeNone}
		require.False(t, appleProfileAppliesToHost(p, &linuxHost, nil))
	})
}

func TestComputeAppleReconcileDeltas(t *testing.T) {
	hostA := &fleet.AppleHostReconcileInfo{
		HostID: 1, UUID: "uuid-A", TeamID: nil, Platform: "darwin",
		LabelUpdatedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	hostB := &fleet.AppleHostReconcileInfo{
		HostID: 2, UUID: "uuid-B", TeamID: new(uint(7)), Platform: "darwin",
		LabelUpdatedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	pGlobal := &fleet.AppleProfileForReconcile{
		ProfileUUID:       "aProfileGlobal",
		ProfileIdentifier: "com.example.global",
		ProfileName:       "Global",
		TeamID:            0,
		Checksum:          []byte("aaaa"),
		LabelMode:         fleet.AppleProfileLabelModeNone,
	}
	pTeam7 := &fleet.AppleProfileForReconcile{
		ProfileUUID:       "aProfileTeam7",
		ProfileIdentifier: "com.example.team7",
		ProfileName:       "Team7",
		TeamID:            7,
		Checksum:          []byte("bbbb"),
		LabelMode:         fleet.AppleProfileLabelModeNone,
	}

	profilesByTeam := map[uint][]*fleet.AppleProfileForReconcile{
		0: {pGlobal},
		7: {pTeam7},
	}

	t.Run("desired but not present -> install", func(t *testing.T) {
		toInstall, toRemove := computeAppleReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA, hostB},
			nil,
			nil,
			profilesByTeam,
		)
		require.Empty(t, toRemove)
		require.Len(t, toInstall, 2)
		set := map[string]string{}
		for _, p := range toInstall {
			set[p.HostUUID] = p.ProfileUUID
		}
		require.Equal(t, "aProfileGlobal", set["uuid-A"])
		require.Equal(t, "aProfileTeam7", set["uuid-B"])
	})

	t.Run("checksum matches and op=install,status=pending -> no-op", func(t *testing.T) {
		current := map[string][]*fleet.MDMAppleProfilePayload{
			"uuid-A": {{
				ProfileUUID:       "aProfileGlobal",
				ProfileIdentifier: "com.example.global",
				ProfileName:       "Global",
				HostUUID:          "uuid-A",
				Checksum:          []byte("aaaa"),
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            new(fleet.MDMDeliveryPending),
			}},
		}
		toInstall, toRemove := computeAppleReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA},
			nil, current, profilesByTeam,
		)
		require.Empty(t, toInstall)
		require.Empty(t, toRemove)
	})

	t.Run("checksum differs -> install", func(t *testing.T) {
		current := map[string][]*fleet.MDMAppleProfilePayload{
			"uuid-A": {{
				ProfileUUID:   "aProfileGlobal",
				HostUUID:      "uuid-A",
				Checksum:      []byte("OLD!"),
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        new(fleet.MDMDeliveryVerified),
			}},
		}
		toInstall, toRemove := computeAppleReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA},
			nil, current, profilesByTeam,
		)
		require.Empty(t, toRemove)
		require.Len(t, toInstall, 1)
		require.True(t, bytes.Equal(toInstall[0].Checksum, []byte("aaaa")))
	})

	t.Run("op=install,status=NULL -> reinstall", func(t *testing.T) {
		current := map[string][]*fleet.MDMAppleProfilePayload{
			"uuid-A": {{
				ProfileUUID:   "aProfileGlobal",
				HostUUID:      "uuid-A",
				Checksum:      []byte("aaaa"),
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        nil,
			}},
		}
		toInstall, toRemove := computeAppleReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA},
			nil, current, profilesByTeam,
		)
		require.Empty(t, toRemove)
		require.Len(t, toInstall, 1)
	})

	t.Run("op=remove -> reinstall (host should get profile back)", func(t *testing.T) {
		current := map[string][]*fleet.MDMAppleProfilePayload{
			"uuid-A": {{
				ProfileUUID:   "aProfileGlobal",
				HostUUID:      "uuid-A",
				Checksum:      []byte("aaaa"),
				OperationType: fleet.MDMOperationTypeRemove,
				Status:        new(fleet.MDMDeliveryVerified),
			}},
		}
		toInstall, toRemove := computeAppleReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA},
			nil, current, profilesByTeam,
		)
		require.Empty(t, toRemove)
		require.Len(t, toInstall, 1)
	})

	t.Run("not in desired -> remove (when not a broken label profile)", func(t *testing.T) {
		current := map[string][]*fleet.MDMAppleProfilePayload{
			"uuid-A": {{
				ProfileUUID:       "aDeletedProfile",
				ProfileIdentifier: "com.deleted",
				HostUUID:          "uuid-A",
				Checksum:          []byte("xxxx"),
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            new(fleet.MDMDeliveryVerified),
			}},
		}
		toInstall, toRemove := computeAppleReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA},
			nil, current, profilesByTeam,
		)
		require.Len(t, toInstall, 1)
		require.Len(t, toRemove, 1)
		require.Equal(t, "aDeletedProfile", toRemove[0].ProfileUUID)
	})

	t.Run("remove already pending -> skip", func(t *testing.T) {
		current := map[string][]*fleet.MDMAppleProfilePayload{
			"uuid-A": {{
				ProfileUUID:   "aDeletedProfile",
				HostUUID:      "uuid-A",
				OperationType: fleet.MDMOperationTypeRemove,
				Status:        new(fleet.MDMDeliveryPending),
			}, {
				ProfileUUID:   "aProfileGlobal",
				HostUUID:      "uuid-A",
				Checksum:      []byte("aaaa"),
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        new(fleet.MDMDeliveryVerified),
			}},
		}
		toInstall, toRemove := computeAppleReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA},
			nil, current, profilesByTeam,
		)
		require.Empty(t, toInstall)
		require.Empty(t, toRemove)
	})

	t.Run("broken label profile is not removed even when current row exists", func(t *testing.T) {
		brokenProf := &fleet.AppleProfileForReconcile{
			ProfileUUID:       "aBrokenLabel",
			ProfileIdentifier: "com.broken",
			TeamID:            0,
			LabelMode:         fleet.AppleProfileLabelModeIncludeAll,
			Labels:            []fleet.AppleProfileLabelRef{{LabelID: nil}},
		}
		profByTeam := map[uint][]*fleet.AppleProfileForReconcile{
			0: {pGlobal, brokenProf},
		}
		current := map[string][]*fleet.MDMAppleProfilePayload{
			"uuid-A": {{
				ProfileUUID:   "aBrokenLabel",
				HostUUID:      "uuid-A",
				Checksum:      []byte("xxxx"),
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        new(fleet.MDMDeliveryVerified),
			}},
		}
		toInstall, toRemove := computeAppleReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA},
			nil, current, profByTeam,
		)
		require.Len(t, toInstall, 1)
		require.Empty(t, toRemove)
	})
}

func TestComputeAppleReconcileDeltas_LabelGates(t *testing.T) {
	host := &fleet.AppleHostReconcileInfo{
		HostID: 1, UUID: "uuid-A", TeamID: nil, Platform: "darwin",
		LabelUpdatedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	pIncludeAll := &fleet.AppleProfileForReconcile{
		ProfileUUID: "aIncludeAll", TeamID: 0,
		LabelMode: fleet.AppleProfileLabelModeIncludeAll,
		Labels:    []fleet.AppleProfileLabelRef{{LabelID: new(uint(1))}, {LabelID: new(uint(2))}},
		Checksum:  []byte("c1"),
	}
	pExcludeAny := &fleet.AppleProfileForReconcile{
		ProfileUUID: "aExcludeAny", TeamID: 0,
		LabelMode: fleet.AppleProfileLabelModeExcludeAny,
		Labels: []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(3)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		Checksum: []byte("c2"),
	}
	profByTeam := map[uint][]*fleet.AppleProfileForReconcile{0: {pIncludeAll, pExcludeAny}}

	t.Run("host has both required labels -> include-all installs", func(t *testing.T) {
		hostLabels := map[uint]map[uint]struct{}{
			1: {1: {}, 2: {}},
		}
		toInstall, _ := computeAppleReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{host}, hostLabels, nil, profByTeam,
		)
		uuids := map[string]struct{}{}
		for _, p := range toInstall {
			uuids[p.ProfileUUID] = struct{}{}
		}
		_, hasIncludeAll := uuids["aIncludeAll"]
		_, hasExcludeAny := uuids["aExcludeAny"]
		require.True(t, hasIncludeAll)
		require.True(t, hasExcludeAny)
	})

	t.Run("host missing one include-all label -> only exclude-any installs", func(t *testing.T) {
		hostLabels := map[uint]map[uint]struct{}{
			1: {1: {}},
		}
		toInstall, _ := computeAppleReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{host}, hostLabels, nil, profByTeam,
		)
		uuids := map[string]struct{}{}
		for _, p := range toInstall {
			uuids[p.ProfileUUID] = struct{}{}
		}
		_, hasIncludeAll := uuids["aIncludeAll"]
		_, hasExcludeAny := uuids["aExcludeAny"]
		require.False(t, hasIncludeAll)
		require.True(t, hasExcludeAny)
	})

	t.Run("host is in the excluded label -> exclude-any skipped", func(t *testing.T) {
		hostLabels := map[uint]map[uint]struct{}{
			1: {1: {}, 2: {}, 3: {}},
		}
		toInstall, _ := computeAppleReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{host}, hostLabels, nil, profByTeam,
		)
		uuids := map[string]struct{}{}
		for _, p := range toInstall {
			uuids[p.ProfileUUID] = struct{}{}
		}
		_, hasIncludeAll := uuids["aIncludeAll"]
		_, hasExcludeAny := uuids["aExcludeAny"]
		require.True(t, hasIncludeAll)
		require.False(t, hasExcludeAny)
	})
}

func TestAppleHostReconcileInfo_EffectiveTeamID(t *testing.T) {
	h := &fleet.AppleHostReconcileInfo{TeamID: nil}
	require.Equal(t, uint(0), h.EffectiveTeamID())

	h.TeamID = new(uint(42))
	require.Equal(t, uint(42), h.EffectiveTeamID())
}

func TestIsBrokenAppleProfile(t *testing.T) {
	good := &fleet.AppleProfileForReconcile{
		ProfileUUID: "aGood",
		LabelMode:   fleet.AppleProfileLabelModeIncludeAll,
		Labels:      []fleet.AppleProfileLabelRef{{LabelID: new(uint(1))}},
	}
	broken := &fleet.AppleProfileForReconcile{
		ProfileUUID: "aBroken",
		LabelMode:   fleet.AppleProfileLabelModeIncludeAll,
		Labels:      []fleet.AppleProfileLabelRef{{LabelID: nil}},
	}
	byTeam := map[uint][]*fleet.AppleProfileForReconcile{
		0: {good, broken},
	}

	require.False(t, isBrokenAppleProfile("aGood", byTeam))
	require.True(t, isBrokenAppleProfile("aBroken", byTeam))
	require.False(t, isBrokenAppleProfile("aMissing", byTeam))
}
