package service

import (
	"bytes"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestAppleProfileHandlerIncludeAll(t *testing.T) {
	t.Run("no labels -> false", func(t *testing.T) {
		require.False(t, appleProfileHandlerIncludeAll(nil, map[uint]struct{}{1: {}}))
	})
	t.Run("broken label -> false", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{{LabelID: nil}}
		require.False(t, appleProfileHandlerIncludeAll(labels, map[uint]struct{}{1: {}}))
	})
	t.Run("host missing a label -> false", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1))},
			{LabelID: new(uint(2))},
		}
		require.False(t, appleProfileHandlerIncludeAll(labels, map[uint]struct{}{1: {}}))
	})
	t.Run("host has all labels -> true", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1))},
			{LabelID: new(uint(2))},
		}
		require.True(t, appleProfileHandlerIncludeAll(labels, map[uint]struct{}{1: {}, 2: {}}))
	})
}

func TestAppleProfileHandlerIncludeAny(t *testing.T) {
	t.Run("no labels -> false", func(t *testing.T) {
		require.False(t, appleProfileHandlerIncludeAny(nil, map[uint]struct{}{}))
	})
	t.Run("broken labels are ignored", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: nil},
			{LabelID: new(uint(2))},
		}
		require.True(t, appleProfileHandlerIncludeAny(labels, map[uint]struct{}{2: {}}))
	})
	t.Run("host in no label -> false", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1))},
			{LabelID: new(uint(2))},
		}
		require.False(t, appleProfileHandlerIncludeAny(labels, map[uint]struct{}{3: {}}))
	})
	t.Run("host in one label -> true", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1))},
			{LabelID: new(uint(2))},
		}
		require.True(t, appleProfileHandlerIncludeAny(labels, map[uint]struct{}{1: {}}))
	})
}

func TestAppleProfileHandlerExcludeAny(t *testing.T) {
	host := &fleet.AppleHostReconcileInfo{
		HostID:         100,
		UUID:           "h1",
		Platform:       "darwin",
		LabelUpdatedAt: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
	}

	t.Run("empty labels -> true (nothing to exclude)", func(t *testing.T) {
		require.True(t, appleProfileHandlerExcludeAny(nil, host, map[uint]struct{}{}))
	})

	t.Run("broken label -> false (never apply)", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{{LabelID: nil}}
		require.False(t, appleProfileHandlerExcludeAny(labels, host, map[uint]struct{}{}))
	})

	t.Run("host is in an excluded label -> false", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		}
		require.False(t, appleProfileHandlerExcludeAny(labels, host, map[uint]struct{}{1: {}}))
	})

	t.Run("host is not in any excluded label -> true", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			{LabelID: new(uint(2)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		}
		require.True(t, appleProfileHandlerExcludeAny(labels, host, map[uint]struct{}{99: {}}))
	})

	t.Run("dynamic label created after host's last scan -> false", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{
				LabelID:             new(uint(1)),
				CreatedAt:           time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				LabelMembershipType: 0,
			},
		}
		require.False(t, appleProfileHandlerExcludeAny(labels, host, map[uint]struct{}{}))
	})

	t.Run("manual label created after host's last scan -> still true", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{
				LabelID:             new(uint(1)),
				CreatedAt:           time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				LabelMembershipType: 1,
			},
		}
		require.True(t, appleProfileHandlerExcludeAny(labels, host, map[uint]struct{}{}))
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
		p := &fleet.AppleProfileForReconcile{TeamID: 5, IncludeMode: fleet.AppleProfileIncludeNone}
		require.False(t, appleProfileAppliesToHost(p, host, nil))
	})

	t.Run("global team matches nil team_id host", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{TeamID: 0, IncludeMode: fleet.AppleProfileIncludeNone}
		require.True(t, appleProfileAppliesToHost(p, host, nil))
	})

	t.Run("non-apple platform -> false", func(t *testing.T) {
		linuxHost := *host
		linuxHost.Platform = "linux"
		p := &fleet.AppleProfileForReconcile{TeamID: 0, IncludeMode: fleet.AppleProfileIncludeNone}
		require.False(t, appleProfileAppliesToHost(p, &linuxHost, nil))
	})
}

func TestAppleProfileAppliesToHost_CombinedIncludeAndExclude(t *testing.T) {
	host := &fleet.AppleHostReconcileInfo{
		HostID: 1, UUID: "h1", TeamID: nil, Platform: "darwin",
		LabelUpdatedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	t.Run("include-all + exclude-any: host passes both -> applies", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{
			TeamID:      0,
			IncludeMode: fleet.AppleProfileIncludeAll,
			IncludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(1))},
				{LabelID: new(uint(2))},
			},
			ExcludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(9)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		}
		hostLabels := map[uint]struct{}{1: {}, 2: {}} // in 1+2, not in 9
		require.True(t, appleProfileAppliesToHost(p, host, hostLabels))
	})

	t.Run("include-all + exclude-any: include fails -> does not apply", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{
			TeamID:      0,
			IncludeMode: fleet.AppleProfileIncludeAll,
			IncludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(1))},
				{LabelID: new(uint(2))},
			},
			ExcludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(9)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		}
		hostLabels := map[uint]struct{}{1: {}} // missing label 2
		require.False(t, appleProfileAppliesToHost(p, host, hostLabels))
	})

	t.Run("include-all + exclude-any: exclude fails -> does not apply", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{
			TeamID:      0,
			IncludeMode: fleet.AppleProfileIncludeAll,
			IncludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(1))},
			},
			ExcludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(9)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		}
		hostLabels := map[uint]struct{}{1: {}, 9: {}} // in label 9 -> excluded
		require.False(t, appleProfileAppliesToHost(p, host, hostLabels))
	})

	t.Run("include-any + exclude-any: any include matches, no exclude matches -> applies", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{
			TeamID:      0,
			IncludeMode: fleet.AppleProfileIncludeAny,
			IncludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(1))},
				{LabelID: new(uint(2))},
			},
			ExcludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(9)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		}
		hostLabels := map[uint]struct{}{2: {}} // matches include-any, not in exclude
		require.True(t, appleProfileAppliesToHost(p, host, hostLabels))
	})

	t.Run("include-any + exclude-any: no include matches -> does not apply", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{
			TeamID:      0,
			IncludeMode: fleet.AppleProfileIncludeAny,
			IncludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(1))},
				{LabelID: new(uint(2))},
			},
			ExcludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(9)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		}
		hostLabels := map[uint]struct{}{5: {}}
		require.False(t, appleProfileAppliesToHost(p, host, hostLabels))
	})

	t.Run("include-any + exclude-any: in exclude label -> does not apply even when include matches", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{
			TeamID:      0,
			IncludeMode: fleet.AppleProfileIncludeAny,
			IncludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(1))},
			},
			ExcludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(9)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		}
		hostLabels := map[uint]struct{}{1: {}, 9: {}}
		require.False(t, appleProfileAppliesToHost(p, host, hostLabels))
	})

	t.Run("exclude-only profile: applies when not in any exclude label", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{
			TeamID:      0,
			IncludeMode: fleet.AppleProfileIncludeNone,
			ExcludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(9)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		}
		require.True(t, appleProfileAppliesToHost(p, host, map[uint]struct{}{1: {}}))
	})

	t.Run("include-only profile: applies when include passes", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{
			TeamID:      0,
			IncludeMode: fleet.AppleProfileIncludeAll,
			IncludeLabels: []fleet.AppleProfileLabelRef{
				{LabelID: new(uint(1))},
			},
		}
		require.True(t, appleProfileAppliesToHost(p, host, map[uint]struct{}{1: {}}))
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
		IncludeMode:       fleet.AppleProfileIncludeNone,
	}
	pTeam7 := &fleet.AppleProfileForReconcile{
		ProfileUUID:       "aProfileTeam7",
		ProfileIdentifier: "com.example.team7",
		ProfileName:       "Team7",
		TeamID:            7,
		Checksum:          []byte("bbbb"),
		IncludeMode:       fleet.AppleProfileIncludeNone,
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

	t.Run("broken include label profile is not removed even when current row exists", func(t *testing.T) {
		brokenProf := &fleet.AppleProfileForReconcile{
			ProfileUUID:       "aBrokenLabel",
			ProfileIdentifier: "com.broken",
			TeamID:            0,
			IncludeMode:       fleet.AppleProfileIncludeAll,
			IncludeLabels:     []fleet.AppleProfileLabelRef{{LabelID: nil}},
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

	t.Run("broken exclude label profile is not removed (broken in either slice protects)", func(t *testing.T) {
		brokenProf := &fleet.AppleProfileForReconcile{
			ProfileUUID:       "aBrokenExclude",
			ProfileIdentifier: "com.broken.exclude",
			TeamID:            0,
			IncludeMode:       fleet.AppleProfileIncludeNone,
			ExcludeLabels:     []fleet.AppleProfileLabelRef{{LabelID: nil}},
		}
		profByTeam := map[uint][]*fleet.AppleProfileForReconcile{
			0: {pGlobal, brokenProf},
		}
		current := map[string][]*fleet.MDMAppleProfilePayload{
			"uuid-A": {{
				ProfileUUID:   "aBrokenExclude",
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
		IncludeMode:   fleet.AppleProfileIncludeAll,
		IncludeLabels: []fleet.AppleProfileLabelRef{{LabelID: new(uint(1))}, {LabelID: new(uint(2))}},
		Checksum:      []byte("c1"),
	}
	pExcludeAny := &fleet.AppleProfileForReconcile{
		ProfileUUID: "aExcludeAny", TeamID: 0,
		IncludeMode: fleet.AppleProfileIncludeNone,
		ExcludeLabels: []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(3)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		Checksum: []byte("c2"),
	}
	pCombined := &fleet.AppleProfileForReconcile{
		ProfileUUID: "aCombined", TeamID: 0,
		IncludeMode:   fleet.AppleProfileIncludeAll,
		IncludeLabels: []fleet.AppleProfileLabelRef{{LabelID: new(uint(1))}},
		ExcludeLabels: []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(3)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
		Checksum: []byte("c3"),
	}
	profByTeam := map[uint][]*fleet.AppleProfileForReconcile{0: {pIncludeAll, pExcludeAny, pCombined}}

	t.Run("host has both required labels, not in exclude -> all three install", func(t *testing.T) {
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
		_, hasCombined := uuids["aCombined"]
		require.True(t, hasIncludeAll)
		require.True(t, hasExcludeAny)
		require.True(t, hasCombined)
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
		_, hasCombined := uuids["aCombined"]
		require.False(t, hasIncludeAll)
		require.True(t, hasExcludeAny)
		// combined needs labels {1} AND not in {3}; host has {1} and not in 3 -> installs
		require.True(t, hasCombined)
	})

	t.Run("host is in the excluded label -> exclude-any and combined skipped", func(t *testing.T) {
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
		_, hasCombined := uuids["aCombined"]
		require.True(t, hasIncludeAll)
		require.False(t, hasExcludeAny)
		require.False(t, hasCombined)
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
		ProfileUUID:   "aGood",
		IncludeMode:   fleet.AppleProfileIncludeAll,
		IncludeLabels: []fleet.AppleProfileLabelRef{{LabelID: new(uint(1))}},
	}
	brokenInclude := &fleet.AppleProfileForReconcile{
		ProfileUUID:   "aBrokenInclude",
		IncludeMode:   fleet.AppleProfileIncludeAll,
		IncludeLabels: []fleet.AppleProfileLabelRef{{LabelID: nil}},
	}
	brokenExclude := &fleet.AppleProfileForReconcile{
		ProfileUUID:   "aBrokenExclude",
		IncludeMode:   fleet.AppleProfileIncludeNone,
		ExcludeLabels: []fleet.AppleProfileLabelRef{{LabelID: nil}},
	}
	byTeam := map[uint][]*fleet.AppleProfileForReconcile{
		0: {good, brokenInclude, brokenExclude},
	}

	require.False(t, isBrokenAppleProfile("aGood", byTeam))
	require.True(t, isBrokenAppleProfile("aBrokenInclude", byTeam))
	require.True(t, isBrokenAppleProfile("aBrokenExclude", byTeam))
	require.False(t, isBrokenAppleProfile("aMissing", byTeam))
}
