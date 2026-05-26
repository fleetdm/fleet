package apple_mdm

import (
	"bytes"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestHandlerIncludeAll(t *testing.T) {
	t.Run("no labels -> false", func(t *testing.T) {
		require.False(t, HandlerIncludeAll(nil, map[uint]struct{}{1: {}}))
	})
	t.Run("broken label -> false", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{{LabelID: nil}}
		require.False(t, HandlerIncludeAll(labels, map[uint]struct{}{1: {}}))
	})
	t.Run("host missing a label -> false", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1))},
			{LabelID: new(uint(2))},
		}
		require.False(t, HandlerIncludeAll(labels, map[uint]struct{}{1: {}}))
	})
	t.Run("host has all labels -> true", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1))},
			{LabelID: new(uint(2))},
		}
		require.True(t, HandlerIncludeAll(labels, map[uint]struct{}{1: {}, 2: {}}))
	})
}

func TestHandlerIncludeAny(t *testing.T) {
	t.Run("no labels -> false", func(t *testing.T) {
		require.False(t, HandlerIncludeAny(nil, map[uint]struct{}{}))
	})
	t.Run("broken labels are ignored", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: nil},
			{LabelID: new(uint(2))},
		}
		require.True(t, HandlerIncludeAny(labels, map[uint]struct{}{2: {}}))
	})
	t.Run("host in no label -> false", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1))},
			{LabelID: new(uint(2))},
		}
		require.False(t, HandlerIncludeAny(labels, map[uint]struct{}{3: {}}))
	})
	t.Run("host in one label -> true", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1))},
			{LabelID: new(uint(2))},
		}
		require.True(t, HandlerIncludeAny(labels, map[uint]struct{}{1: {}}))
	})
}

func TestHandlerExcludeAny(t *testing.T) {
	host := &fleet.AppleHostReconcileInfo{
		HostID:         100,
		UUID:           "h1",
		Platform:       "darwin",
		LabelUpdatedAt: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
	}

	t.Run("empty labels -> false (nothing to exclude)", func(t *testing.T) {
		require.False(t, HandlerExcludeAny(nil, host, map[uint]struct{}{}))
	})
	t.Run("broken label -> true (exclude)", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{{LabelID: nil}}
		require.True(t, HandlerExcludeAny(labels, host, map[uint]struct{}{}))
	})
	t.Run("host is in an excluded label -> true", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		}
		require.True(t, HandlerExcludeAny(labels, host, map[uint]struct{}{1: {}}))
	})
	t.Run("host is not in any excluded label -> false", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{LabelID: new(uint(1)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			{LabelID: new(uint(2)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		}
		require.False(t, HandlerExcludeAny(labels, host, map[uint]struct{}{99: {}}))
	})
	t.Run("dynamic label created after host's last scan -> true (exclude)", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{
				LabelID:             new(uint(1)),
				CreatedAt:           time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				LabelMembershipType: int(fleet.LabelMembershipTypeDynamic),
			},
		}
		require.True(t, HandlerExcludeAny(labels, host, map[uint]struct{}{}))
	})
	t.Run("host vital label created after host's last scan -> false (include)", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{
				LabelID:             new(uint(1)),
				CreatedAt:           time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				LabelMembershipType: int(fleet.LabelMembershipTypeHostVitals),
			},
		}
		require.False(t, HandlerExcludeAny(labels, host, map[uint]struct{}{}))
	})
	t.Run("manual label created after host's last scan -> still false (include)", func(t *testing.T) {
		labels := []fleet.AppleProfileLabelRef{
			{
				LabelID:             new(uint(1)),
				CreatedAt:           time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
				LabelMembershipType: int(fleet.LabelMembershipTypeManual),
			},
		}
		require.False(t, HandlerExcludeAny(labels, host, map[uint]struct{}{}))
	})
}

func TestEntityAppliesToHost_TeamAndPlatformGates(t *testing.T) {
	host := &fleet.AppleHostReconcileInfo{
		HostID:   1,
		UUID:     "h1",
		TeamID:   nil,
		Platform: "darwin",
	}

	t.Run("wrong team -> false", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{TeamID: 5, IncludeMode: fleet.AppleProfileIncludeNone}
		require.False(t, EntityAppliesToHost(p, host, nil))
	})
	t.Run("global team matches nil team_id host", func(t *testing.T) {
		p := &fleet.AppleProfileForReconcile{TeamID: 0, IncludeMode: fleet.AppleProfileIncludeNone}
		require.True(t, EntityAppliesToHost(p, host, nil))
	})
	t.Run("non-apple platform -> false", func(t *testing.T) {
		linuxHost := *host
		linuxHost.Platform = "linux"
		p := &fleet.AppleProfileForReconcile{TeamID: 0, IncludeMode: fleet.AppleProfileIncludeNone}
		require.False(t, EntityAppliesToHost(p, &linuxHost, nil))
	})
}

// TestEntityAppliesToHost_DeclarationsShareSameDispatcher pins the
// drift-prevention contract: feeding the same label config to both a
// *AppleProfileForReconcile and a *AppleDeclarationForReconcile must
// produce the same applies-to-host result. If this test breaks, the
// AppleLabeledEntity interface is no longer the single source of truth.
func TestEntityAppliesToHost_DeclarationsShareSameDispatcher(t *testing.T) {
	host := &fleet.AppleHostReconcileInfo{
		HostID: 1, UUID: "h1", TeamID: nil, Platform: "darwin",
		LabelUpdatedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	commonLabels := []fleet.AppleProfileLabelRef{
		{LabelID: new(uint(1))},
		{LabelID: new(uint(2))},
	}
	commonExclude := []fleet.AppleProfileLabelRef{
		{LabelID: new(uint(9)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
	}

	prof := &fleet.AppleProfileForReconcile{
		ProfileUUID: "aProf", TeamID: 0,
		IncludeMode:   fleet.AppleProfileIncludeAll,
		IncludeLabels: commonLabels,
		ExcludeLabels: commonExclude,
	}
	decl := &fleet.AppleDeclarationForReconcile{
		DeclarationUUID: "aDecl", TeamID: 0,
		IncludeMode:   fleet.AppleProfileIncludeAll,
		IncludeLabels: commonLabels,
		ExcludeLabels: commonExclude,
	}

	cases := []struct {
		name       string
		hostLabels map[uint]struct{}
		want       bool
	}{
		{"both required + not in exclude", map[uint]struct{}{1: {}, 2: {}}, true},
		{"missing one required", map[uint]struct{}{1: {}}, false},
		{"in exclude", map[uint]struct{}{1: {}, 2: {}, 9: {}}, false},
		{"neither included", map[uint]struct{}{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pr := EntityAppliesToHost(prof, host, c.hostLabels)
			dr := EntityAppliesToHost(decl, host, c.hostLabels)
			require.Equal(t, c.want, pr, "profile result")
			require.Equal(t, c.want, dr, "declaration result")
			require.Equal(t, pr, dr,
				"PROFILE/DECLARATION DRIFT: same label config produced different results — AppleLabeledEntity contract broken")
		})
	}
}

func TestComputeReconcileDeltas(t *testing.T) {
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
	profilesByTeam := map[uint][]*fleet.AppleProfileForReconcile{0: {pGlobal}, 7: {pTeam7}}
	profilesWithBrokenLabel := map[string]struct{}{}

	t.Run("desired but not present -> install", func(t *testing.T) {
		toInstall, toRemove := ComputeReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA, hostB}, nil, nil, profilesByTeam, profilesWithBrokenLabel,
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
		toInstall, toRemove := ComputeReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, nil, current, profilesByTeam, profilesWithBrokenLabel,
		)
		require.Empty(t, toRemove)
		require.Len(t, toInstall, 1)
		require.True(t, bytes.Equal(toInstall[0].Checksum, []byte("aaaa")))
	})

	t.Run("not in desired -> remove (when not broken-label)", func(t *testing.T) {
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
		toInstall, toRemove := ComputeReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, nil, current, profilesByTeam, profilesWithBrokenLabel,
		)
		require.Len(t, toInstall, 1)
		require.Len(t, toRemove, 1)
		require.Equal(t, "aDeletedProfile", toRemove[0].ProfileUUID)
	})

	t.Run("broken label profile is not removed", func(t *testing.T) {
		brokenProf := &fleet.AppleProfileForReconcile{
			ProfileUUID:   "aBrokenLabel",
			TeamID:        0,
			IncludeMode:   fleet.AppleProfileIncludeAll,
			IncludeLabels: []fleet.AppleProfileLabelRef{{LabelID: nil}},
		}
		profByTeam := map[uint][]*fleet.AppleProfileForReconcile{0: {pGlobal, brokenProf}}
		current := map[string][]*fleet.MDMAppleProfilePayload{
			"uuid-A": {{
				ProfileUUID:   "aBrokenLabel",
				HostUUID:      "uuid-A",
				Checksum:      []byte("xxxx"),
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        new(fleet.MDMDeliveryVerified),
			}},
		}
		toInstall, toRemove := ComputeReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, nil, current, profByTeam, profilesWithBrokenLabel,
		)
		require.Len(t, toInstall, 1) // pGlobal still installs
		require.Empty(t, toRemove)
	})
}

func TestComputeDeclarationDeltas(t *testing.T) {
	hostA := &fleet.AppleHostReconcileInfo{
		HostID: 1, UUID: "uuid-A", TeamID: nil, Platform: "darwin",
		LabelUpdatedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	dGlobal := &fleet.AppleDeclarationForReconcile{
		DeclarationUUID:       "aDeclGlobal",
		DeclarationIdentifier: "com.example.decl.global",
		DeclarationName:       "GlobalDecl",
		TeamID:                0,
		Token:                 []byte("tok1"),
		IncludeMode:           fleet.AppleProfileIncludeNone,
	}
	declsByTeam := map[uint][]*fleet.AppleDeclarationForReconcile{0: {dGlobal}}
	declsWithBrokenLabel := map[string]struct{}{}

	t.Run("desired but not present -> install diff", func(t *testing.T) {
		changed, rows := ComputeDeclarationDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, nil, nil, declsByTeam, declsWithBrokenLabel,
		)
		require.ElementsMatch(t, []string{"uuid-A"}, changed)
		require.Len(t, rows, 1)
		require.Equal(t, fleet.MDMOperationTypeInstall, rows[0].OperationType)
		require.Equal(t, "aDeclGlobal", rows[0].DeclarationUUID)
	})

	t.Run("token matches and op=install,status=pending -> no diff", func(t *testing.T) {
		current := map[string][]*fleet.MDMAppleHostDeclaration{
			"uuid-A": {{
				HostUUID:        "uuid-A",
				DeclarationUUID: "aDeclGlobal",
				Token:           "tok1",
				OperationType:   fleet.MDMOperationTypeInstall,
				Status:          new(fleet.MDMDeliveryPending),
			}},
		}
		changed, rows := ComputeDeclarationDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, nil, current, declsByTeam, declsWithBrokenLabel,
		)
		require.Empty(t, changed)
		require.Empty(t, rows)
	})

	t.Run("token differs -> install diff", func(t *testing.T) {
		current := map[string][]*fleet.MDMAppleHostDeclaration{
			"uuid-A": {{
				HostUUID:        "uuid-A",
				DeclarationUUID: "aDeclGlobal",
				Token:           "OLD!",
				OperationType:   fleet.MDMOperationTypeInstall,
				Status:          new(fleet.MDMDeliveryVerified),
			}},
		}
		changed, rows := ComputeDeclarationDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, nil, current, declsByTeam, declsWithBrokenLabel,
		)
		require.ElementsMatch(t, []string{"uuid-A"}, changed)
		require.Len(t, rows, 1)
		require.Equal(t, "tok1", rows[0].Token)
	})
}
