package service

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// TestAppleEntityAppliesToHost_DeclarationsShareSameDispatcher proves the
// shared dispatcher behaves identically when given a declaration vs a
// profile carrying the same label config. If this test breaks, drift has
// been introduced between profile and declaration label-membership
// evaluation — the whole point of the AppleLabeledEntity interface.
func TestAppleEntityAppliesToHost_DeclarationsShareSameDispatcher(t *testing.T) {
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
		ProfileUUID:   "aProf",
		TeamID:        0,
		IncludeMode:   fleet.AppleProfileIncludeAll,
		IncludeLabels: commonLabels,
		ExcludeLabels: commonExclude,
	}
	decl := &fleet.AppleDeclarationForReconcile{
		DeclarationUUID: "aDecl",
		TeamID:          0,
		IncludeMode:     fleet.AppleProfileIncludeAll,
		IncludeLabels:   commonLabels,
		ExcludeLabels:   commonExclude,
	}

	cases := []struct {
		name       string
		hostLabels map[uint]struct{}
		want       bool
	}{
		{"host has both required labels, not in exclude", map[uint]struct{}{1: {}, 2: {}}, true},
		{"host missing one required label", map[uint]struct{}{1: {}}, false},
		{"host in exclude label", map[uint]struct{}{1: {}, 2: {}, 9: {}}, false},
		{"host in neither include set", map[uint]struct{}{}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			profResult := appleEntityAppliesToHost(prof, host, c.hostLabels)
			declResult := appleEntityAppliesToHost(decl, host, c.hostLabels)
			require.Equal(t, c.want, profResult, "profile result mismatch")
			require.Equal(t, c.want, declResult, "declaration result mismatch")
			require.Equal(t, profResult, declResult,
				"PROFILE/DECLARATION DRIFT: same label config produced different results — the AppleLabeledEntity contract is broken")
		})
	}
}

func TestComputeAppleDeclarationDeltas(t *testing.T) {
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

	t.Run("desired but not present -> install diff", func(t *testing.T) {
		changed, rows := computeAppleDeclarationDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, nil, nil, declsByTeam,
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
		changed, rows := computeAppleDeclarationDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, nil, current, declsByTeam,
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
		changed, rows := computeAppleDeclarationDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, nil, current, declsByTeam,
		)
		require.ElementsMatch(t, []string{"uuid-A"}, changed)
		require.Len(t, rows, 1)
		require.Equal(t, fleet.MDMOperationTypeInstall, rows[0].OperationType)
		require.Equal(t, "tok1", rows[0].Token)
	})

	t.Run("not in desired -> remove diff", func(t *testing.T) {
		current := map[string][]*fleet.MDMAppleHostDeclaration{
			"uuid-A": {{
				HostUUID:        "uuid-A",
				DeclarationUUID: "aDeletedDecl",
				Identifier:      "com.deleted",
				Name:            "DeletedDecl",
				Token:           "old",
				OperationType:   fleet.MDMOperationTypeInstall,
				Status:          new(fleet.MDMDeliveryVerified),
			}},
		}
		changed, rows := computeAppleDeclarationDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, nil, current, declsByTeam,
		)
		require.ElementsMatch(t, []string{"uuid-A"}, changed)
		// Install for dGlobal + Remove for aDeletedDecl.
		require.Len(t, rows, 2)
		var hadRemove, hadInstall bool
		for _, r := range rows {
			switch r.OperationType {
			case fleet.MDMOperationTypeRemove:
				require.Equal(t, "aDeletedDecl", r.DeclarationUUID)
				hadRemove = true
			case fleet.MDMOperationTypeInstall:
				require.Equal(t, "aDeclGlobal", r.DeclarationUUID)
				hadInstall = true
			}
		}
		require.True(t, hadRemove)
		require.True(t, hadInstall)
	})

	t.Run("broken label declaration is not removed", func(t *testing.T) {
		brokenDecl := &fleet.AppleDeclarationForReconcile{
			DeclarationUUID: "aBrokenDecl",
			TeamID:          0,
			IncludeMode:     fleet.AppleProfileIncludeAll,
			IncludeLabels:   []fleet.AppleProfileLabelRef{{LabelID: nil}},
		}
		declByTeam := map[uint][]*fleet.AppleDeclarationForReconcile{0: {dGlobal, brokenDecl}}
		current := map[string][]*fleet.MDMAppleHostDeclaration{
			"uuid-A": {{
				HostUUID:        "uuid-A",
				DeclarationUUID: "aBrokenDecl",
				Token:           "tok",
				OperationType:   fleet.MDMOperationTypeInstall,
				Status:          new(fleet.MDMDeliveryVerified),
			}},
		}
		changed, rows := computeAppleDeclarationDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, nil, current, declByTeam,
		)
		// Only dGlobal install diff; broken decl is NOT in remove list.
		require.ElementsMatch(t, []string{"uuid-A"}, changed)
		require.Len(t, rows, 1)
		require.Equal(t, "aDeclGlobal", rows[0].DeclarationUUID)
		require.Equal(t, fleet.MDMOperationTypeInstall, rows[0].OperationType)
	})
}

func TestIsBrokenAppleDeclaration(t *testing.T) {
	good := &fleet.AppleDeclarationForReconcile{
		DeclarationUUID: "good",
		IncludeMode:     fleet.AppleProfileIncludeAll,
		IncludeLabels:   []fleet.AppleProfileLabelRef{{LabelID: new(uint(1))}},
	}
	broken := &fleet.AppleDeclarationForReconcile{
		DeclarationUUID: "broken",
		IncludeMode:     fleet.AppleProfileIncludeAll,
		IncludeLabels:   []fleet.AppleProfileLabelRef{{LabelID: nil}},
	}
	byTeam := map[uint][]*fleet.AppleDeclarationForReconcile{0: {good, broken}}

	require.False(t, isBrokenAppleDeclaration("good", byTeam))
	require.True(t, isBrokenAppleDeclaration("broken", byTeam))
	require.False(t, isBrokenAppleDeclaration("missing", byTeam))
}
