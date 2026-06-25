package apple_mdm

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	nanomdm_pushsvc "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push/service"
	"github.com/fleetdm/fleet/v4/server/mock"
	mdmmock "github.com/fleetdm/fleet/v4/server/mock/mdm"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/nanolib/log/stdlogfmt"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The include/exclude label handlers and the team + include/exclude logic of the dispatcher are platform-neutral and live in
// server/mdm/reconcile, which owns their exhaustive unit tests. The Apple tests below cover only what is Apple-specific: the
// platform gate, the threading of host fields (EffectiveTeamID, LabelUpdatedAt) through the Apple wrapper into the shared
// dispatcher, the declarations-share-the-dispatcher drift contract, and the Apple compute/execute logic.

func TestEntityAppliesToHost_AppleWrapper(t *testing.T) {
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
	t.Run("non-apple platform -> false (Apple-only platform gate)", func(t *testing.T) {
		linuxHost := *host
		linuxHost.Platform = "linux"
		p := &fleet.AppleProfileForReconcile{TeamID: 0, IncludeMode: fleet.AppleProfileIncludeNone}
		require.False(t, EntityAppliesToHost(p, &linuxHost, nil))
	})
	// The shared package tests the dynamic-label timing rule itself; this case
	// pins that the Apple wrapper threads host.LabelUpdatedAt into it, rather
	// than e.g. a zero time.
	t.Run("host.LabelUpdatedAt is threaded into the exclude-timing gate", func(t *testing.T) {
		dynamicExcLabel := fleet.AppleProfileLabelRef{
			LabelID:             new(uint(99)),
			LabelMembershipType: int(fleet.LabelMembershipTypeDynamic),
			CreatedAt:           time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC),
		}
		p := &fleet.AppleProfileForReconcile{
			TeamID:        0,
			IncludeMode:   fleet.AppleProfileIncludeAny,
			IncludeLabels: []fleet.AppleProfileLabelRef{{LabelID: new(uint(1))}},
			ExcludeLabels: []fleet.AppleProfileLabelRef{dynamicExcLabel},
		}
		// Host scanned before the dynamic exclude label was created -> disqualified.
		staleHost := &fleet.AppleHostReconcileInfo{
			HostID: 1, UUID: "h1", TeamID: nil, Platform: "darwin",
			LabelUpdatedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		}
		require.False(t, EntityAppliesToHost(p, staleHost, map[uint]struct{}{1: {}}))
		// Once the host's scan advances past the label's CreatedAt, it applies.
		freshHost := &fleet.AppleHostReconcileInfo{
			HostID: 1, UUID: "h1", TeamID: nil, Platform: "darwin",
			LabelUpdatedAt: time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
		}
		require.True(t, EntityAppliesToHost(p, freshHost, map[uint]struct{}{1: {}}))
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

	t.Run("combined include_all + exclude_any: host in include but also in exclude -> not desired", func(t *testing.T) {
		pCombined := &fleet.AppleProfileForReconcile{
			ProfileUUID:       "aCombined",
			ProfileIdentifier: "com.example.combined",
			ProfileName:       "Combined",
			TeamID:            0,
			Checksum:          []byte("cccc"),
			IncludeMode:       fleet.AppleProfileIncludeAll,
			IncludeLabels:     []fleet.AppleProfileLabelRef{{LabelID: new(uint(10))}},
			ExcludeLabels:     []fleet.AppleProfileLabelRef{{LabelID: new(uint(20)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}},
		}
		profByTeam := map[uint][]*fleet.AppleProfileForReconcile{0: {pGlobal, pCombined}}

		// Host in include label but also in exclude label -> combined profile not desired
		hostLabels := map[uint]map[uint]struct{}{
			hostA.HostID: {10: {}, 20: {}},
		}
		toInstall, toRemove := ComputeReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, hostLabels, nil, profByTeam, profilesWithBrokenLabel,
		)
		require.Empty(t, toRemove)
		require.Len(t, toInstall, 1)
		require.Equal(t, "aProfileGlobal", toInstall[0].ProfileUUID)
	})

	t.Run("combined include_all + exclude_any: host in include not in exclude -> desired", func(t *testing.T) {
		pCombined := &fleet.AppleProfileForReconcile{
			ProfileUUID:       "aCombined",
			ProfileIdentifier: "com.example.combined",
			ProfileName:       "Combined",
			TeamID:            0,
			Checksum:          []byte("cccc"),
			IncludeMode:       fleet.AppleProfileIncludeAll,
			IncludeLabels:     []fleet.AppleProfileLabelRef{{LabelID: new(uint(10))}},
			ExcludeLabels:     []fleet.AppleProfileLabelRef{{LabelID: new(uint(20)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}},
		}
		profByTeam := map[uint][]*fleet.AppleProfileForReconcile{0: {pGlobal, pCombined}}

		// Host in include label, not in exclude -> both profiles desired
		hostLabels := map[uint]map[uint]struct{}{
			hostA.HostID: {10: {}},
		}
		toInstall, toRemove := ComputeReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, hostLabels, nil, profByTeam, profilesWithBrokenLabel,
		)
		require.Empty(t, toRemove)
		require.Len(t, toInstall, 2)
	})

	t.Run("combined include_any + exclude_any: host in include but also in exclude -> not desired, removed if present", func(t *testing.T) {
		pCombined := &fleet.AppleProfileForReconcile{
			ProfileUUID:       "aCombined",
			ProfileIdentifier: "com.example.combined",
			ProfileName:       "Combined",
			TeamID:            0,
			Checksum:          []byte("cccc"),
			IncludeMode:       fleet.AppleProfileIncludeAny,
			IncludeLabels:     []fleet.AppleProfileLabelRef{{LabelID: new(uint(10))}, {LabelID: new(uint(11))}},
			ExcludeLabels:     []fleet.AppleProfileLabelRef{{LabelID: new(uint(20)), CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)}},
		}
		profByTeam := map[uint][]*fleet.AppleProfileForReconcile{0: {pGlobal, pCombined}}

		// Host in one include label AND in exclude label -> combined profile no longer desired, should be removed
		hostLabels := map[uint]map[uint]struct{}{
			hostA.HostID: {10: {}, 20: {}},
		}
		current := map[string][]*fleet.MDMAppleProfilePayload{
			"uuid-A": {{
				ProfileUUID:   "aCombined",
				HostUUID:      "uuid-A",
				Checksum:      []byte("cccc"),
				OperationType: fleet.MDMOperationTypeInstall,
				Status:        new(fleet.MDMDeliveryVerified),
			}},
		}
		toInstall, toRemove := ComputeReconcileDeltas(
			[]*fleet.AppleHostReconcileInfo{hostA}, hostLabels, current, profByTeam, profilesWithBrokenLabel,
		)
		require.Len(t, toInstall, 1) // pGlobal
		require.Len(t, toRemove, 1)
		require.Equal(t, "aCombined", toRemove[0].ProfileUUID)
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
		profilesWithBrokenLabel := map[string]struct{}{"aBrokenLabel": {}}
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

func TestMDMAppleExecuteReconcileBatch(t *testing.T) {
	ctx := context.Background()
	mdmStorage := &mdmmock.MDMAppleStore{}
	ds := new(mock.Store)
	kv := new(mock.AdvancedKVStore)
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		stdlogfmt.New(),
	)
	mdmConfig := config.MDMConfig{
		AppleSCEPCert: "../../service/testdata/server.pem",
		AppleSCEPKey:  "../../service/testdata/server.key",
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		_, pemCert, pemKey, err := mdmConfig.AppleSCEP()
		require.NoError(t, err)
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: pemCert},
			fleet.MDMAssetCAKey:  {Value: pemKey},
		}, nil
	}

	cmdr := NewMDMAppleCommander(mdmStorage, pusher)
	hostUUID1, hostUUID2 := "ABC-DEF", "GHI-JKL"
	hostUUID1UserEnrollment := hostUUID1 + ":user"
	contents1 := []byte("test-content-1")
	expectedContents1 := []byte("test-content-1") // used for Fleet variable substitution
	contents2 := []byte("test-content-2")
	contents4 := []byte("test-content-4")
	contents5 := []byte("test-contents-5")
	contents7 := []byte("test-contents-7")

	p1, p2, p3, p4, p5, p6, p7 := "a"+uuid.NewString(), "a"+uuid.NewString(), "a"+uuid.NewString(), "a"+uuid.NewString(), "a"+uuid.NewString(), "a"+uuid.NewString(), "a"+uuid.NewString()
	baseProfilesToInstall := []*fleet.MDMAppleProfilePayload{
		{ProfileUUID: p1, ProfileIdentifier: "com.add.profile", HostUUID: hostUUID1, Scope: fleet.PayloadScopeSystem},
		{ProfileUUID: p2, ProfileIdentifier: "com.add.profile.two", HostUUID: hostUUID1, Scope: fleet.PayloadScopeSystem},
		{ProfileUUID: p2, ProfileIdentifier: "com.add.profile.two", HostUUID: hostUUID2, Scope: fleet.PayloadScopeSystem},
		{ProfileUUID: p4, ProfileIdentifier: "com.add.profile.four", HostUUID: hostUUID2, Scope: fleet.PayloadScopeSystem},
		{ProfileUUID: p5, ProfileIdentifier: "com.add.profile.five", HostUUID: hostUUID1, Scope: fleet.PayloadScopeUser},
		{ProfileUUID: p5, ProfileIdentifier: "com.add.profile.five", HostUUID: hostUUID2, Scope: fleet.PayloadScopeUser},
		{ProfileUUID: p7, ProfileIdentifier: "com.add.profile.seven", HostUUID: hostUUID1, Scope: fleet.PayloadScopeUser},
		{ProfileUUID: p7, ProfileIdentifier: "com.add.profile.seven", HostUUID: hostUUID2, Scope: fleet.PayloadScopeUser, HostPlatform: "ios"},
	}
	baseProfilesToRemove := []*fleet.MDMAppleProfilePayload{
		{ProfileUUID: p3, ProfileIdentifier: "com.remove.profile", HostUUID: hostUUID1, Scope: fleet.PayloadScopeSystem},
		{ProfileUUID: p3, ProfileIdentifier: "com.remove.profile", HostUUID: hostUUID2, Scope: fleet.PayloadScopeSystem},
		{ProfileUUID: p6, ProfileIdentifier: "com.remove.profile.six", HostUUID: hostUUID1, Scope: fleet.PayloadScopeUser},
		{ProfileUUID: p6, ProfileIdentifier: "com.remove.profile.six", HostUUID: hostUUID2, Scope: fleet.PayloadScopeUser},
	}

	kv.MGetFunc = func(ctx context.Context, keys []string) (map[string]*string, error) {
		return map[string]*string{}, nil
	}

	ds.GetMDMAppleProfilesContentsFunc = func(ctx context.Context, profileUUIDs []string) (map[string]mobileconfig.Mobileconfig, error) {
		require.ElementsMatch(t, []string{p1, p2, p4, p5, p7}, profileUUIDs)
		// only those profiles that are to be installed
		return map[string]mobileconfig.Mobileconfig{
			p1: contents1,
			p2: contents2,
			p4: contents4,
			p5: contents5,
			p7: contents7,
		}, nil
	}

	ds.BulkDeleteMDMAppleHostsConfigProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleProfilePayload) error {
		require.ElementsMatch(t, payload, []*fleet.MDMAppleProfilePayload{{ProfileUUID: p6, ProfileIdentifier: "com.remove.profile.six", HostUUID: hostUUID2, Scope: fleet.PayloadScopeUser}})
		return nil
	}

	ds.GetNanoMDMUserEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		if hostUUID == hostUUID1 {
			return &fleet.NanoEnrollment{
				ID:               hostUUID1UserEnrollment,
				DeviceID:         hostUUID1,
				Type:             "User",
				Enabled:          true,
				TokenUpdateTally: 1,
			}, nil
		}
		// hostUUID2 has no user enrollment
		assert.Equal(t, hostUUID2, hostUUID)
		return nil, nil
	}

	mdmStorage.BulkDeleteHostUserCommandsWithoutResultsFunc = func(ctx context.Context, commandToIDs map[string][]string) error {
		require.Empty(t, commandToIDs)
		return nil
	}

	var enqueueFailForOp fleet.MDMOperationType
	var mu sync.Mutex
	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
		require.NotNil(t, cmd)
		require.NotEmpty(t, cmd.CommandUUID)

		switch cmd.Command.Command.RequestType {
		case "InstallProfile":

			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			// the p7 library doesn't support concurrent calls to Parse
			mu.Lock()
			pk7, err := pkcs7.Parse(fullCmd.Command.InstallProfile.Payload)
			mu.Unlock()
			require.NoError(t, err)

			if !bytes.Equal(pk7.Content, expectedContents1) && !bytes.Equal(pk7.Content, contents2) &&
				!bytes.Equal(pk7.Content, contents4) && !bytes.Equal(pk7.Content, contents5) && !bytes.Equal(pk7.Content, contents7) {
				require.Failf(t, "profile contents don't match", "expected to contain %s, %s or %s but got %s",
					expectedContents1, contents2, contents4, pk7.Content)
			}

			// may be called for a single host or both
			if len(id) == 2 {
				if bytes.Equal(pk7.Content, contents5) || bytes.Equal(pk7.Content, contents7) {
					require.ElementsMatch(t, []string{hostUUID1UserEnrollment, hostUUID2}, id)
				} else {
					require.ElementsMatch(t, []string{hostUUID1, hostUUID2}, id)
				}
			} else {
				require.Len(t, id, 1)
			}

		case "RemoveProfile":
			if len(id) == 1 {
				require.Equal(t, hostUUID1UserEnrollment, id[0])
			} else {
				require.ElementsMatch(t, []string{hostUUID1, hostUUID2}, id)
			}
			require.Contains(t, string(cmd.Raw), "com.remove.profile")
		}
		switch {
		case enqueueFailForOp == fleet.MDMOperationTypeInstall && cmd.Command.Command.RequestType == "InstallProfile":
			return nil, errors.New("enqueue error")
		case enqueueFailForOp == fleet.MDMOperationTypeRemove && cmd.Command.Command.RequestType == "RemoveProfile":
			return nil, errors.New("enqueue error")
		}
		return nil, nil
	}

	mdmStorage.RetrievePushInfoFunc = func(ctx context.Context, tokens []string) (map[string]*mdm.Push, error) {
		res := make(map[string]*mdm.Push, len(tokens))
		for _, t := range tokens {
			res[t] = &mdm.Push{
				PushMagic: "",
				Token:     []byte(t),
				Topic:     "",
			}
		}
		return res, nil
	}
	mdmStorage.RetrievePushCertFunc = func(ctx context.Context, topic string) (*tls.Certificate, string, error) {
		cert, err := tls.LoadX509KeyPair("../../service/testdata/server.pem", "../../service/testdata/server.key")
		return &cert, "", err
	}
	mdmStorage.IsPushCertStaleFunc = func(ctx context.Context, topic string, staleToken string) (bool, error) {
		return false, nil
	}
	mdmStorage.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		certPEM, err := os.ReadFile("../../service/testdata/server.pem")
		require.NoError(t, err)
		keyPEM, err := os.ReadFile("../../service/testdata/server.key")
		require.NoError(t, err)
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: certPEM},
			fleet.MDMAssetCAKey:  {Value: keyPEM},
		}, nil
	}

	var failedCall bool
	var failedCheck func([]*fleet.MDMAppleBulkUpsertHostProfilePayload)
	ds.BulkUpsertMDMAppleHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
		if failedCall {
			failedCheck(payload)
			return nil
		}

		// next call will be failed call, until reset
		failedCall = true

		// first time it is called, it is to set the status to pending and all
		// host profiles have a command uuid
		cmdUUIDByProfileUUIDInstall := make(map[string]string)
		cmdUUIDByProfileUUIDRemove := make(map[string]string)
		copies := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, len(payload))
		for i, p := range payload {
			// clear the command UUID (in a copy so that it does not affect the
			// pointed-to struct) from the payload for the subsequent checks
			copyp := *p
			copyp.CommandUUID = ""
			copies[i] = &copyp

			// Host with no user enrollment, so install fails
			if p.HostUUID == hostUUID2 && (p.ProfileUUID == p5 || p.ProfileUUID == p7) {
				continue
			}

			if p.OperationType == fleet.MDMOperationTypeInstall {
				existing, ok := cmdUUIDByProfileUUIDInstall[p.ProfileUUID]
				if ok {
					require.Equal(t, existing, p.CommandUUID)
				} else {
					cmdUUIDByProfileUUIDInstall[p.ProfileUUID] = p.CommandUUID
				}
			} else {
				require.Equal(t, fleet.MDMOperationTypeRemove, p.OperationType)
				existing, ok := cmdUUIDByProfileUUIDRemove[p.ProfileUUID]
				if ok {
					require.Equal(t, existing, p.CommandUUID)
				} else {
					cmdUUIDByProfileUUIDRemove[p.ProfileUUID] = p.CommandUUID
				}
			}

		}

		require.ElementsMatch(t, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
			{
				ProfileUUID:       p1,
				ProfileIdentifier: "com.add.profile",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p2,
				ProfileIdentifier: "com.add.profile.two",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p2,
				ProfileIdentifier: "com.add.profile.two",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p3,
				ProfileIdentifier: "com.remove.profile",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeRemove,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p3,
				ProfileIdentifier: "com.remove.profile",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeRemove,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p4,
				ProfileIdentifier: "com.add.profile.four",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			// This host has a user enrollment so the profile is sent to it
			{
				ProfileUUID:       p5,
				ProfileIdentifier: "com.add.profile.five",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeUser,
			},
			// This host has no user enrollment so the profile is errored
			{
				ProfileUUID:       p5,
				ProfileIdentifier: "com.add.profile.five",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Detail:            "This setting couldn't be enforced because the user channel doesn't exist for this host. Currently, Fleet creates the user channel for hosts that automatically enroll.",
				Status:            &fleet.MDMDeliveryFailed,
				Scope:             fleet.PayloadScopeUser,
			},
			// This host has a user enrollment so the profile is removed from it
			{
				ProfileUUID:       p6,
				ProfileIdentifier: "com.remove.profile.six",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeRemove,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeUser,
			},
			// Note that host2 has no user enrollment so the profile is not marked for removal
			// from it
			{
				ProfileUUID:       p7,
				ProfileIdentifier: "com.add.profile.seven",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeUser,
			},
			{
				ProfileUUID:       p7,
				ProfileIdentifier: "com.add.profile.seven",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryFailed,
				Detail:            "This setting couldn't be enforced because the user channel isn't available on iOS and iPadOS hosts.",
				Scope:             fleet.PayloadScopeUser,
			},
		}, copies)
		return nil
	}

	appCfg := &fleet.AppConfig{}
	appCfg.ServerSettings.ServerURL = "https://test.example.com"
	appCfg.MDM.EnabledAndConfigured = true

	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, includeSecrets bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{}, nil
	}

	ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, p []*fleet.MDMAppleConfigProfile) error {
		return nil
	}

	ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
		return []*fleet.EnrollSecret{}, nil
	}
	ds.GetMDMAppleReconcileCursorFunc = func(ctx context.Context) (string, error) {
		return "", nil
	}

	checkAndReset := func(t *testing.T, want bool, invoked *bool) {
		if want {
			assert.True(t, *invoked)
		} else {
			assert.False(t, *invoked)
		}
		*invoked = false
	}

	t.Run("success", func(t *testing.T) {
		var failedCount int
		failedCall = false
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			failedCount++
			require.Empty(t, payload)
		}
		_, err := ExecuteReconcileBatch(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), appCfg, 0, baseProfilesToInstall, baseProfilesToRemove)
		require.NoError(t, err)
		require.Equal(t, 1, failedCount)
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
		checkAndReset(t, true, &ds.GetNanoMDMUserEnrollmentFuncInvoked)
	})

	t.Run("fail enqueue remove ops", func(t *testing.T) {
		var failedCount int
		failedCall = false
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			failedCount++
			require.Len(t, payload, 3) // the 3 remove ops
			require.ElementsMatch(t, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
				{
					ProfileUUID:       p3,
					ProfileIdentifier: "com.remove.profile",
					HostUUID:          hostUUID1,
					OperationType:     fleet.MDMOperationTypeRemove,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeSystem,
				},
				{
					ProfileUUID:       p3,
					ProfileIdentifier: "com.remove.profile",
					HostUUID:          hostUUID2,
					OperationType:     fleet.MDMOperationTypeRemove,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeSystem,
				},
				{
					ProfileUUID:       p6,
					ProfileIdentifier: "com.remove.profile.six",
					HostUUID:          hostUUID1,
					OperationType:     fleet.MDMOperationTypeRemove,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeUser,
				},
			}, payload)
		}

		enqueueFailForOp = fleet.MDMOperationTypeRemove
		_, err := ExecuteReconcileBatch(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), appCfg, 0, baseProfilesToInstall, baseProfilesToRemove)
		require.NoError(t, err)
		require.Equal(t, 1, failedCount)
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
		checkAndReset(t, true, &ds.GetNanoMDMUserEnrollmentFuncInvoked)
	})

	t.Run("fail enqueue install ops", func(t *testing.T) {
		var failedCount int
		failedCall = false
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			failedCount++

			require.Len(t, payload, 6) // the 6 install ops
			require.ElementsMatch(t, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
				{
					ProfileUUID:       p1,
					ProfileIdentifier: "com.add.profile",
					HostUUID:          hostUUID1, OperationType: fleet.MDMOperationTypeInstall,
					Status:      nil,
					CommandUUID: "",
					Scope:       fleet.PayloadScopeSystem,
				},
				{
					ProfileUUID:       p2,
					ProfileIdentifier: "com.add.profile.two",
					HostUUID:          hostUUID1, OperationType: fleet.MDMOperationTypeInstall,
					Status:      nil,
					CommandUUID: "",
					Scope:       fleet.PayloadScopeSystem,
				},
				{
					ProfileUUID:       p2,
					ProfileIdentifier: "com.add.profile.two",
					HostUUID:          hostUUID2,
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeSystem,
				},
				{
					ProfileUUID:       p4,
					ProfileIdentifier: "com.add.profile.four",
					HostUUID:          hostUUID2,
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeSystem,
				},
				{
					ProfileUUID:       p5,
					ProfileIdentifier: "com.add.profile.five",
					HostUUID:          hostUUID1,
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeUser,
				},
				{
					ProfileUUID:       p7,
					ProfileIdentifier: "com.add.profile.seven",
					HostUUID:          hostUUID1,
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            nil,
					CommandUUID:       "",
					Scope:             fleet.PayloadScopeUser,
				},
			}, payload)
		}

		enqueueFailForOp = fleet.MDMOperationTypeInstall
		_, err := ExecuteReconcileBatch(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), appCfg, 0, baseProfilesToInstall, baseProfilesToRemove)
		require.NoError(t, err)
		require.Equal(t, 1, failedCount)
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
		checkAndReset(t, true, &ds.GetNanoMDMUserEnrollmentFuncInvoked)
	})

	ds.BulkDeleteMDMAppleHostsConfigProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleProfilePayload) error {
		require.Empty(t, payload)
		return nil
	}
	ds.BulkUpsertMDMAppleHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
		if failedCall {
			failedCheck(payload)
			return nil
		}

		// next call will be failed call, until reset
		failedCall = true

		// first time it is called, it is to set the status to pending and all
		// host profiles have a command uuid
		cmdUUIDByProfileUUIDInstall := make(map[string]string)
		cmdUUIDByProfileUUIDRemove := make(map[string]string)
		copies := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, len(payload))
		for i, p := range payload {
			// clear the command UUID (in a copy so that it does not affect the
			// pointed-to struct) from the payload for the subsequent checks
			copyp := *p
			copyp.CommandUUID = ""
			copies[i] = &copyp

			// Host with no user enrollment, so install fails
			if p.HostUUID == hostUUID2 && (p.ProfileUUID == p5 || p.ProfileUUID == p7) {
				continue
			}

			if p.OperationType == fleet.MDMOperationTypeInstall {
				existing, ok := cmdUUIDByProfileUUIDInstall[p.ProfileUUID]
				if ok {
					require.Equal(t, existing, p.CommandUUID)
				} else {
					cmdUUIDByProfileUUIDInstall[p.ProfileUUID] = p.CommandUUID
				}
			} else {
				require.Equal(t, fleet.MDMOperationTypeRemove, p.OperationType)
				existing, ok := cmdUUIDByProfileUUIDRemove[p.ProfileUUID]
				if ok {
					require.Equal(t, existing, p.CommandUUID)
				} else {
					cmdUUIDByProfileUUIDRemove[p.ProfileUUID] = p.CommandUUID
				}
			}
		}

		require.ElementsMatch(t, []*fleet.MDMAppleBulkUpsertHostProfilePayload{
			{
				ProfileUUID:       p1,
				ProfileIdentifier: "com.add.profile",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p2,
				ProfileIdentifier: "com.add.profile.two",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p2,
				ProfileIdentifier: "com.add.profile.two",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p4,
				ProfileIdentifier: "com.add.profile.four",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeSystem,
			},
			{
				ProfileUUID:       p5,
				ProfileIdentifier: "com.add.profile.five",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeUser,
			},
			// This host has no user enrollment so the profile is sent to the device enrollment
			{
				ProfileUUID:       p5,
				ProfileIdentifier: "com.add.profile.five",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryFailed,
				Detail:            "This setting couldn't be enforced because the user channel doesn't exist for this host. Currently, Fleet creates the user channel for hosts that automatically enroll.",
				Scope:             fleet.PayloadScopeUser,
			},
			{
				ProfileUUID:       p7,
				ProfileIdentifier: "com.add.profile.seven",
				HostUUID:          hostUUID1,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryPending,
				Scope:             fleet.PayloadScopeUser,
			},
			{
				ProfileUUID:       p7,
				ProfileIdentifier: "com.add.profile.seven",
				HostUUID:          hostUUID2,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            &fleet.MDMDeliveryFailed,
				Detail:            "This setting couldn't be enforced because the user channel isn't available on iOS and iPadOS hosts.",
				Scope:             fleet.PayloadScopeUser,
			},
		}, copies)
		return nil
	}

	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	ds.BulkUpsertMDMManagedCertificatesFunc = func(ctx context.Context, payload []*fleet.MDMManagedCertificate) error {
		assert.Empty(t, payload)
		return nil
	}

	t.Run("replace $FLEET_VAR_"+string(fleet.FleetVarNDESSCEPProxyURL), func(t *testing.T) {
		var upsertCount int
		failedCall = false
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			upsertCount++
			if upsertCount == 1 {
				// We update the profile with a new command UUID
				assert.Len(t, payload, 1, "at upsertCount %d", upsertCount)
			} else {
				assert.Empty(t, payload, "at upsertCount %d", upsertCount)
			}
		}
		enqueueFailForOp = ""
		newContents := "$FLEET_VAR_" + fleet.FleetVarNDESSCEPProxyURL
		originalContents1 := contents1
		originalExpectedContents1 := expectedContents1
		contents1 = []byte(newContents)
		expectedContents1 = []byte("https://test.example.com" + SCEPProxyPath + url.QueryEscape(fmt.Sprintf("%s,%s,NDES", hostUUID1, p1)))
		t.Cleanup(func() {
			contents1 = originalContents1
			expectedContents1 = originalExpectedContents1
		})
		_, err := ExecuteReconcileBatch(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), appCfg, 0, baseProfilesToInstall, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, upsertCount)
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
		checkAndReset(t, true, &ds.GetNanoMDMUserEnrollmentFuncInvoked)
	})

	t.Run("preprocessor fails on $FLEET_VAR_"+string(fleet.FleetVarHostEndUserEmailIDP), func(t *testing.T) {
		var failedCount int
		failedCall = false
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			failedCount++
			require.Len(t, payload, 8)
		}
		enqueueFailForOp = ""
		newContents := "$FLEET_VAR_" + fleet.FleetVarHostEndUserEmailIDP
		originalContents1 := contents1
		contents1 = []byte(newContents)
		t.Cleanup(func() {
			contents1 = originalContents1
		})
		ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
			return nil, errors.New("GetHostEmailsFuncError")
		}
		_, err := ExecuteReconcileBatch(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), appCfg, 0, baseProfilesToInstall, nil)
		require.ErrorContains(t, err, "GetHostEmailsFuncError")
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
		checkAndReset(t, true, &ds.GetNanoMDMUserEnrollmentFuncInvoked)
	})

	t.Run("bad $FLEET_VAR", func(t *testing.T) {
		var failedCount int
		failedCall = false
		var hostUUIDs []string
		failedCheck = func(payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) {
			if len(payload) > 0 {
				failedCount++
			}
			for _, p := range payload {
				assert.Equal(t, fleet.MDMDeliveryFailed, *p.Status)
				assert.Contains(t, p.Detail, "FLEET_VAR_BOZO")
				for i, hu := range hostUUIDs {
					if hu == p.HostUUID {
						// remove element
						hostUUIDs = append(hostUUIDs[:i], hostUUIDs[i+1:]...)
						break
					}
				}
			}
		}
		enqueueFailForOp = ""

		// All profiles will have bad contents
		badContents := "bad-content: $FLEET_VAR_BOZO"
		originalContents1 := contents1
		originalContents2 := contents2
		originalContents4 := contents4
		originalContents5 := contents5
		originalContents7 := contents7
		contents1 = []byte(badContents)
		contents2 = []byte(badContents)
		contents4 = []byte(badContents)
		contents5 = []byte(badContents)
		contents7 = []byte(badContents)
		t.Cleanup(func() {
			contents1 = originalContents1
			contents2 = originalContents2
			contents4 = originalContents4
			contents5 = originalContents5
			contents7 = originalContents7
		})

		hostUUIDs = make([]string, 0, len(baseProfilesToInstall))
		for _, p := range baseProfilesToInstall {
			// This host will error before this point - should not be updated by the variable failure
			if p.HostUUID == hostUUID2 && (p.ProfileUUID == p5 || p.ProfileUUID == p7) {
				continue
			}
			hostUUIDs = append(hostUUIDs, p.HostUUID)
		}

		_, err := ExecuteReconcileBatch(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), appCfg, 0, baseProfilesToInstall, nil)
		require.NoError(t, err)
		assert.Empty(t, hostUUIDs, "all host+profile combinations should be updated")
		require.Equal(t, 5, failedCount, "number of profiles with bad content")
		checkAndReset(t, true, &ds.GetMDMAppleProfilesContentsFuncInvoked)
		checkAndReset(t, true, &ds.BulkUpsertMDMAppleHostProfilesFuncInvoked)
		checkAndReset(t, true, &ds.GetNanoMDMUserEnrollmentFuncInvoked)
		// Check that individual updates were not done (bulk update should be done)
		checkAndReset(t, false, &ds.UpdateOrDeleteHostMDMAppleProfileFuncInvoked)
	})
}

func TestMDMAppleExecuteReconcileBatchCAThrottle(t *testing.T) {
	ctx := t.Context()
	mdmStorage := &mdmmock.MDMAppleStore{}
	ds := new(mock.Store)
	kv := new(mock.AdvancedKVStore)
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		stdlogfmt.New(),
	)
	mdmConfig := config.MDMConfig{
		AppleSCEPCert: "../../service/testdata/server.pem",
		AppleSCEPKey:  "../../service/testdata/server.key",
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		_, pemCert, pemKey, err := mdmConfig.AppleSCEP()
		require.NoError(t, err)
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: pemCert},
			fleet.MDMAssetCAKey:  {Value: pemKey},
		}, nil
	}

	cmdr := NewMDMAppleCommander(mdmStorage, pusher)
	hostUUIDs := []string{"host-1", "host-2", "host-3", "host-4", "host-5"}

	caProfileUUID := "a" + uuid.NewString()
	nonCAProfileUUID := "a" + uuid.NewString()
	caContent := []byte("profile with $FLEET_VAR_NDES_SCEP_CHALLENGE variable")
	nonCAContent := []byte("regular profile content")

	appCfg := &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}

	// Build toInstall: CA profile for 5 hosts + non-CA profile for 5 hosts
	var profilesToInstall []*fleet.MDMAppleProfilePayload
	for _, h := range hostUUIDs {
		profilesToInstall = append(profilesToInstall,
			&fleet.MDMAppleProfilePayload{ProfileUUID: caProfileUUID, ProfileIdentifier: "com.ca.profile", ProfileName: "CA Profile", HostUUID: h, Scope: fleet.PayloadScopeSystem},
			&fleet.MDMAppleProfilePayload{ProfileUUID: nonCAProfileUUID, ProfileIdentifier: "com.regular.profile", ProfileName: "Regular Profile", HostUUID: h, Scope: fleet.PayloadScopeSystem},
		)
	}

	ds.GetMDMAppleProfilesContentsFunc = func(ctx context.Context, profileUUIDs []string) (map[string]mobileconfig.Mobileconfig, error) {
		return map[string]mobileconfig.Mobileconfig{
			caProfileUUID:    caContent,
			nonCAProfileUUID: nonCAContent,
		}, nil
	}

	kv.MGetFunc = func(ctx context.Context, keys []string) (map[string]*string, error) {
		return make(map[string]*string), nil
	}

	ds.BulkDeleteMDMAppleHostsConfigProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleProfilePayload) error {
		return nil
	}

	ds.GetNanoMDMUserEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		return nil, nil
	}

	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, allCAs bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{}, nil
	}

	mdmStorage.BulkDeleteHostUserCommandsWithoutResultsFunc = func(ctx context.Context, commandToIDs map[string][]string) error {
		return nil
	}

	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
		return nil, nil
	}

	mdmStorage.RetrievePushInfoFunc = func(ctx context.Context, tokens []string) (map[string]*mdm.Push, error) {
		res := make(map[string]*mdm.Push, len(tokens))
		for _, t := range tokens {
			res[t] = &mdm.Push{
				PushMagic: "",
				Token:     []byte(t),
				Topic:     "",
			}
		}
		return res, nil
	}
	mdmStorage.RetrievePushCertFunc = func(ctx context.Context, topic string) (*tls.Certificate, string, error) {
		cert, err := tls.LoadX509KeyPair("../../service/testdata/server.pem", "../../service/testdata/server.key")
		return &cert, "", err
	}
	mdmStorage.IsPushCertStaleFunc = func(ctx context.Context, topic string, staleToken string) (bool, error) {
		return false, nil
	}
	mdmStorage.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		certPEM, err := os.ReadFile("../../service/testdata/server.pem")
		require.NoError(t, err)
		keyPEM, err := os.ReadFile("../../service/testdata/server.key")
		require.NoError(t, err)
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: certPEM},
			fleet.MDMAssetCAKey:  {Value: keyPEM},
		}, nil
	}

	ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
		return []*fleet.EnrollSecret{}, nil
	}

	ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, p []*fleet.MDMAppleConfigProfile) error {
		return nil
	}

	// Track upserted host profiles to verify throttling.
	// The first BulkUpsert call contains the profiles that will be sent;
	// subsequent calls are for reverting failures (empty).
	var upsertedProfiles []*fleet.MDMAppleBulkUpsertHostProfilePayload
	var bulkUpsertCallCount int
	ds.BulkUpsertMDMAppleHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
		bulkUpsertCallCount++
		if bulkUpsertCallCount == 1 {
			upsertedProfiles = payload
		}
		return nil
	}

	t.Run("limit=0 sends all profiles", func(t *testing.T) {
		upsertedProfiles = nil
		bulkUpsertCallCount = 0
		_, err := ExecuteReconcileBatch(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), appCfg, 0, profilesToInstall, nil)
		require.NoError(t, err)

		// All 10 host-profile pairs should be upserted (5 CA + 5 non-CA)
		var caCount, nonCACount int
		for _, p := range upsertedProfiles {
			if p.ProfileUUID == caProfileUUID {
				caCount++
			} else if p.ProfileUUID == nonCAProfileUUID {
				nonCACount++
			}
		}
		assert.Equal(t, 5, caCount, "all CA host-profile pairs should be sent when limit=0")
		assert.Equal(t, 5, nonCACount, "all non-CA host-profile pairs should be sent")
	})

	t.Run("limit=2 throttles CA profiles only", func(t *testing.T) {
		upsertedProfiles = nil
		bulkUpsertCallCount = 0
		_, err := ExecuteReconcileBatch(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), appCfg, 2, profilesToInstall, nil)
		require.NoError(t, err)

		// Should have 2 CA + 5 non-CA = 7 host-profile pairs upserted
		var caCount, nonCACount int
		for _, p := range upsertedProfiles {
			if p.ProfileUUID == caProfileUUID {
				caCount++
			} else if p.ProfileUUID == nonCAProfileUUID {
				nonCACount++
			}
		}
		assert.Equal(t, 2, caCount, "only 2 CA host-profile pairs should be sent when limit=2")
		assert.Equal(t, 5, nonCACount, "all non-CA host-profile pairs should still be sent")
	})

	t.Run("recently enrolled hosts bypass throttle", func(t *testing.T) {
		upsertedProfiles = nil
		bulkUpsertCallCount = 0

		recentEnrollTime := time.Now().Add(-30 * time.Minute)
		var recentProfilesToInstall []*fleet.MDMAppleProfilePayload
		for _, h := range hostUUIDs {
			recentProfilesToInstall = append(recentProfilesToInstall,
				&fleet.MDMAppleProfilePayload{
					ProfileUUID: caProfileUUID, ProfileIdentifier: "com.ca.profile", ProfileName: "CA Profile",
					HostUUID: h, Scope: fleet.PayloadScopeSystem, DeviceEnrolledAt: &recentEnrollTime,
				},
				&fleet.MDMAppleProfilePayload{
					ProfileUUID: nonCAProfileUUID, ProfileIdentifier: "com.regular.profile", ProfileName: "Regular Profile",
					HostUUID: h, Scope: fleet.PayloadScopeSystem, DeviceEnrolledAt: &recentEnrollTime,
				},
			)
		}

		_, err := ExecuteReconcileBatch(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), appCfg, 2, recentProfilesToInstall, nil)
		require.NoError(t, err)

		var caCount, nonCACount int
		for _, p := range upsertedProfiles {
			switch p.ProfileUUID {
			case caProfileUUID:
				caCount++
			case nonCAProfileUUID:
				nonCACount++
			}
		}
		assert.Equal(t, 5, caCount, "all CA host-profile pairs should be sent for recently enrolled hosts")
		assert.Equal(t, 5, nonCACount, "all non-CA host-profile pairs should be sent")
	})

	t.Run("removals are not throttled", func(t *testing.T) {
		upsertedProfiles = nil
		bulkUpsertCallCount = 0

		var profilesToRemove []*fleet.MDMAppleProfilePayload
		for _, h := range hostUUIDs {
			profilesToRemove = append(profilesToRemove,
				&fleet.MDMAppleProfilePayload{
					ProfileUUID: caProfileUUID, ProfileIdentifier: "com.ca.profile", ProfileName: "CA Profile",
					HostUUID: h, Scope: fleet.PayloadScopeSystem, OperationType: fleet.MDMOperationTypeInstall,
					Status: &fleet.MDMDeliveryVerifying, CommandUUID: uuid.NewString(),
				},
			)
		}

		_, err := ExecuteReconcileBatch(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), appCfg, 2, nil, profilesToRemove)
		require.NoError(t, err)

		var removeCount int
		for _, p := range upsertedProfiles {
			if p.ProfileUUID == caProfileUUID && p.OperationType == fleet.MDMOperationTypeRemove {
				removeCount++
			}
		}
		assert.Equal(t, 5, removeCount, "all CA profile removals should proceed regardless of throttle limit")
	})
}

func TestMDMAppleExecuteReconcileBatchSkipsHostBeingProcessed(t *testing.T) {
	ctx := t.Context()
	mdmStorage := &mdmmock.MDMAppleStore{}
	ds := new(mock.Store)
	kv := new(mock.AdvancedKVStore)
	pushFactory, _ := newMockAPNSPushProviderFactory()
	pusher := nanomdm_pushsvc.New(
		mdmStorage,
		mdmStorage,
		pushFactory,
		stdlogfmt.New(),
	)
	mdmConfig := config.MDMConfig{
		AppleSCEPCert: "../../service/testdata/server.pem",
		AppleSCEPKey:  "../../service/testdata/server.key",
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		_, pemCert, pemKey, err := mdmConfig.AppleSCEP()
		require.NoError(t, err)
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: pemCert},
			fleet.MDMAssetCAKey:  {Value: pemKey},
		}, nil
	}

	cmdr := NewMDMAppleCommander(mdmStorage, pusher)

	profileUUID := "a" + uuid.NewString()
	profileContent := []byte("regular profile content")
	blockedHostUUID := "host-blocked"
	nonSetupHostUUID := "host-non-setup"

	appCfg := &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}
	profilesToInstall := []*fleet.MDMAppleProfilePayload{
		{ProfileUUID: profileUUID, ProfileIdentifier: "com.test.profile", ProfileName: "Test Profile", HostUUID: blockedHostUUID, Scope: fleet.PayloadScopeSystem},
		{ProfileUUID: profileUUID, ProfileIdentifier: "com.test.profile", ProfileName: "Test Profile", HostUUID: nonSetupHostUUID, Scope: fleet.PayloadScopeSystem},
	}
	ds.GetMDMAppleProfilesContentsFunc = func(ctx context.Context, profileUUIDs []string) (map[string]mobileconfig.Mobileconfig, error) {
		return map[string]mobileconfig.Mobileconfig{profileUUID: profileContent}, nil
	}
	ds.BulkDeleteMDMAppleHostsConfigProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleProfilePayload) error {
		return nil
	}
	ds.GetNanoMDMUserEnrollmentFunc = func(ctx context.Context, hostUUID string) (*fleet.NanoEnrollment, error) {
		return nil, nil
	}
	ds.GetGroupedCertificateAuthoritiesFunc = func(ctx context.Context, allCAs bool) (*fleet.GroupedCertificateAuthorities, error) {
		return &fleet.GroupedCertificateAuthorities{}, nil
	}
	ds.AggregateEnrollSecretPerTeamFunc = func(ctx context.Context) ([]*fleet.EnrollSecret, error) {
		return []*fleet.EnrollSecret{}, nil
	}
	ds.BulkUpsertMDMAppleConfigProfilesFunc = func(ctx context.Context, p []*fleet.MDMAppleConfigProfile) error {
		return nil
	}
	mdmStorage.BulkDeleteHostUserCommandsWithoutResultsFunc = func(ctx context.Context, commandToIDs map[string][]string) error {
		return nil
	}
	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
		return nil, nil
	}
	mdmStorage.RetrievePushInfoFunc = func(ctx context.Context, tokens []string) (map[string]*mdm.Push, error) {
		res := make(map[string]*mdm.Push, len(tokens))
		for _, t := range tokens {
			res[t] = &mdm.Push{PushMagic: "", Token: []byte(t), Topic: ""}
		}
		return res, nil
	}
	mdmStorage.RetrievePushCertFunc = func(ctx context.Context, topic string) (*tls.Certificate, string, error) {
		cert, err := tls.LoadX509KeyPair("../../service/testdata/server.pem", "../../service/testdata/server.key")
		return &cert, "", err
	}
	mdmStorage.IsPushCertStaleFunc = func(ctx context.Context, topic string, staleToken string) (bool, error) {
		return false, nil
	}
	mdmStorage.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName,
		_ sqlx.QueryerContext,
	) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		certPEM, err := os.ReadFile("../../service/testdata/server.pem")
		require.NoError(t, err)
		keyPEM, err := os.ReadFile("../../service/testdata/server.key")
		require.NoError(t, err)
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetCACert: {Value: certPEM},
			fleet.MDMAssetCAKey:  {Value: keyPEM},
		}, nil
	}

	// Track what gets upserted and which hosts get commands enqueued
	var upsertedProfiles []*fleet.MDMAppleBulkUpsertHostProfilePayload
	var bulkUpsertCallCount int
	ds.BulkUpsertMDMAppleHostProfilesFunc = func(ctx context.Context, payload []*fleet.MDMAppleBulkUpsertHostProfilePayload) error {
		bulkUpsertCallCount++
		if bulkUpsertCallCount == 1 {
			upsertedProfiles = payload
		}
		return nil
	}

	// Simulate an in-memory KV store with TTL support
	kvStore := make(map[string]string)
	kv.MGetFunc = func(ctx context.Context, keys []string) (map[string]*string, error) {
		result := make(map[string]*string, len(keys))
		for _, k := range keys {
			if v, ok := kvStore[k]; ok {
				result[k] = &v
			} else {
				result[k] = nil
			}
		}
		return result, nil
	}

	// verify host marked as going through setup does not get profiles reconciled
	blockedKey := fleet.MDMProfileProcessingKeyPrefix + ":" + blockedHostUUID
	kvStore[blockedKey] = "1"

	upsertedProfiles = nil
	bulkUpsertCallCount = 0
	_, err := ExecuteReconcileBatch(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), appCfg, 0, profilesToInstall, nil)
	require.NoError(t, err)

	// Only the non setup host should have profiles with a pending status and command UUID;
	// the blocked host should have its status/command cleared.
	var pendingHosts []string
	var skippedHosts []string
	for _, p := range upsertedProfiles {
		if p.Status != nil && *p.Status == fleet.MDMDeliveryPending && p.CommandUUID != "" {
			pendingHosts = append(pendingHosts, p.HostUUID)
		} else if p.Status == nil && p.CommandUUID == "" {
			skippedHosts = append(skippedHosts, p.HostUUID)
		}
	}
	assert.Contains(t, pendingHosts, nonSetupHostUUID, "non setup host should have profiles enqueued")
	assert.NotContains(t, pendingHosts, blockedHostUUID, "blocked host should NOT have profiles enqueued")
	assert.Contains(t, skippedHosts, blockedHostUUID, "blocked host should be skipped with nil status")

	// expire the key, the host that didn't get profiles before should do now
	delete(kvStore, blockedKey) // simulate TTL expiry

	upsertedProfiles = nil
	bulkUpsertCallCount = 0
	_, err = ExecuteReconcileBatch(ctx, ds, cmdr, kv, slog.New(slog.DiscardHandler), appCfg, 0, profilesToInstall, nil)
	require.NoError(t, err)

	pendingHosts = nil
	for _, p := range upsertedProfiles {
		if p.Status != nil && *p.Status == fleet.MDMDeliveryPending && p.CommandUUID != "" {
			pendingHosts = append(pendingHosts, p.HostUUID)
		}
	}
	assert.Contains(t, pendingHosts, nonSetupHostUUID, "non setup host should still have profiles enqueued")
	assert.Contains(t, pendingHosts, blockedHostUUID, "previously blocked host should now have profiles enqueued after key expiry")
}
