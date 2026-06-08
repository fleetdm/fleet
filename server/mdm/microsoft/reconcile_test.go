package microsoft_mdm

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// installSet / removeSet collapse the delta slices into sets of
// "hostUUID|profileUUID" keys so assertions don't depend on slice order.
func installSet(payloads []*fleet.MDMWindowsProfilePayload) map[string]struct{} {
	out := make(map[string]struct{}, len(payloads))
	for _, p := range payloads {
		out[p.HostUUID+"|"+p.ProfileUUID] = struct{}{}
	}
	return out
}

func key(hostUUID, profileUUID string) string { return hostUUID + "|" + profileUUID }

// TestComputeWindowsReconcileDeltasInstallRules covers each branch of the
// install WHERE clause (windowsProfilesToInstallQuery) as an independent
// subtest. The host is a no-team host with one global, label-less profile;
// only the current host_mdm_windows_profiles row varies between cases.
func TestComputeWindowsReconcileDeltasInstallRules(t *testing.T) {
	host := &fleet.WindowsHostReconcileInfo{HostID: 1, UUID: "h1", TeamID: nil}
	desiredChecksum := []byte("checksum-A")
	newer := time.Now()
	older := newer.Add(-time.Hour)

	profile := &fleet.WindowsProfileForReconcile{
		ProfileUUID: "p1",
		ProfileName: "Profile 1",
		TeamID:      0,
		Checksum:    desiredChecksum,
	}

	cases := []struct {
		name        string
		current     *fleet.MDMWindowsProfilePayload // nil => no current row
		profileMod  func(p *fleet.WindowsProfileForReconcile)
		wantInstall bool
	}{
		{
			name:        "no current row installs",
			current:     nil,
			wantInstall: true,
		},
		{
			name:        "matching install row does not reinstall",
			current:     &fleet.MDMWindowsProfilePayload{ProfileUUID: "p1", HostUUID: "h1", Checksum: desiredChecksum, OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryVerified)},
			wantInstall: false,
		},
		{
			name:        "checksum mismatch installs",
			current:     &fleet.MDMWindowsProfilePayload{ProfileUUID: "p1", HostUUID: "h1", Checksum: []byte("stale"), OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryVerified)},
			wantInstall: true,
		},
		{
			name: "secrets updated (both present, current older) installs",
			profileMod: func(p *fleet.WindowsProfileForReconcile) {
				p.SecretsUpdatedAt = &newer
			},
			current:     &fleet.MDMWindowsProfilePayload{ProfileUUID: "p1", HostUUID: "h1", Checksum: desiredChecksum, OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryVerified), SecretsUpdatedAt: &older},
			wantInstall: true,
		},
		{
			name: "desired has secrets but current has none does NOT install (IFNULL=FALSE)",
			profileMod: func(p *fleet.WindowsProfileForReconcile) {
				p.SecretsUpdatedAt = &newer
			},
			current:     &fleet.MDMWindowsProfilePayload{ProfileUUID: "p1", HostUUID: "h1", Checksum: desiredChecksum, OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryVerified), SecretsUpdatedAt: nil},
			wantInstall: false,
		},
		{
			name:        "install op with NULL status reinstalls",
			current:     &fleet.MDMWindowsProfilePayload{ProfileUUID: "p1", HostUUID: "h1", Checksum: desiredChecksum, OperationType: fleet.MDMOperationTypeInstall, Status: nil},
			wantInstall: true,
		},
		{
			name:        "install op pending does not reinstall",
			current:     &fleet.MDMWindowsProfilePayload{ProfileUUID: "p1", HostUUID: "h1", Checksum: desiredChecksum, OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryPending)},
			wantInstall: false,
		},
		{
			name:        "remove op NULL status flips back to install",
			current:     &fleet.MDMWindowsProfilePayload{ProfileUUID: "p1", HostUUID: "h1", Checksum: desiredChecksum, OperationType: fleet.MDMOperationTypeRemove, Status: nil},
			wantInstall: true,
		},
		{
			name:        "remove op pending flips back to install",
			current:     &fleet.MDMWindowsProfilePayload{ProfileUUID: "p1", HostUUID: "h1", Checksum: desiredChecksum, OperationType: fleet.MDMOperationTypeRemove, Status: new(fleet.MDMDeliveryPending)},
			wantInstall: true,
		},
		{
			name:        "remove op verifying does not flip back",
			current:     &fleet.MDMWindowsProfilePayload{ProfileUUID: "p1", HostUUID: "h1", Checksum: desiredChecksum, OperationType: fleet.MDMOperationTypeRemove, Status: new(fleet.MDMDeliveryVerifying)},
			wantInstall: false,
		},
		{
			name:        "remove op verified does not flip back",
			current:     &fleet.MDMWindowsProfilePayload{ProfileUUID: "p1", HostUUID: "h1", Checksum: desiredChecksum, OperationType: fleet.MDMOperationTypeRemove, Status: new(fleet.MDMDeliveryVerified)},
			wantInstall: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := *profile
			if tc.profileMod != nil {
				tc.profileMod(&p)
			}
			currentByHost := map[string][]*fleet.MDMWindowsProfilePayload{}
			if tc.current != nil {
				currentByHost["h1"] = []*fleet.MDMWindowsProfilePayload{tc.current}
			}

			toInstall, toRemove := ComputeWindowsReconcileDeltas(
				[]*fleet.WindowsHostReconcileInfo{host},
				nil,
				currentByHost,
				map[uint][]*fleet.WindowsProfileForReconcile{0: {&p}},
				nil,
			)
			require.Empty(t, toRemove)
			if tc.wantInstall {
				require.Contains(t, installSet(toInstall), key("h1", "p1"))
				// install payload carries the desired profile's content.
				require.Equal(t, p.Checksum, toInstall[0].Checksum)
			} else {
				require.Empty(t, toInstall)
			}
		})
	}
}

// TestComputeWindowsReconcileDeltasRemoveRules covers the remove WHERE clause
// (windowsProfilesToRemoveQuery): current rows with no desired-state match are
// removed, except rows already processing a remove and except broken-label
// profiles.
func TestComputeWindowsReconcileDeltasRemoveRules(t *testing.T) {
	host := &fleet.WindowsHostReconcileInfo{HostID: 1, UUID: "h1", TeamID: nil}

	cases := []struct {
		name       string
		current    *fleet.MDMWindowsProfilePayload
		broken     bool
		wantRemove bool
	}{
		{
			name:       "current install not desired is removed",
			current:    &fleet.MDMWindowsProfilePayload{ProfileUUID: "gone", HostUUID: "h1", ProfileName: "Gone", OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryVerified)},
			wantRemove: true,
		},
		{
			name:       "remove op with NULL status is (re)removed",
			current:    &fleet.MDMWindowsProfilePayload{ProfileUUID: "gone", HostUUID: "h1", OperationType: fleet.MDMOperationTypeRemove, Status: nil},
			wantRemove: true,
		},
		{
			name:       "remove op already in-flight is skipped",
			current:    &fleet.MDMWindowsProfilePayload{ProfileUUID: "gone", HostUUID: "h1", OperationType: fleet.MDMOperationTypeRemove, Status: new(fleet.MDMDeliveryPending)},
			wantRemove: false,
		},
		{
			name:       "broken-label profile is kept (not removed)",
			current:    &fleet.MDMWindowsProfilePayload{ProfileUUID: "gone", HostUUID: "h1", OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryVerified)},
			broken:     true,
			wantRemove: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var broken map[string]struct{}
			if tc.broken {
				broken = map[string]struct{}{"gone": {}}
			}
			toInstall, toRemove := ComputeWindowsReconcileDeltas(
				[]*fleet.WindowsHostReconcileInfo{host},
				nil,
				map[string][]*fleet.MDMWindowsProfilePayload{"h1": {tc.current}},
				nil, // no desired profiles
				broken,
			)
			require.Empty(t, toInstall)
			if tc.wantRemove {
				require.Len(t, toRemove, 1)
				require.Equal(t, "gone", toRemove[0].ProfileUUID)
				require.Equal(t, "h1", toRemove[0].HostUUID)
			} else {
				require.Empty(t, toRemove)
			}
		})
	}
}

// TestComputeWindowsReconcileDeltasTeamGating verifies that a host only
// receives profiles for its own team — team_id=0 is its own scope, a teamed
// host does not inherit global profiles.
func TestComputeWindowsReconcileDeltasTeamGating(t *testing.T) {
	globalProfile := &fleet.WindowsProfileForReconcile{ProfileUUID: "pg", ProfileName: "global", TeamID: 0, Checksum: []byte("c")}
	teamProfile := &fleet.WindowsProfileForReconcile{ProfileUUID: "pt", ProfileName: "team", TeamID: 5, Checksum: []byte("c")}

	noTeamHost := &fleet.WindowsHostReconcileInfo{HostID: 1, UUID: "h-global", TeamID: nil}
	teamedHost := &fleet.WindowsHostReconcileInfo{HostID: 2, UUID: "h-team", TeamID: new(uint(5))}

	profilesByTeam := map[uint][]*fleet.WindowsProfileForReconcile{
		0: {globalProfile},
		5: {teamProfile},
	}

	toInstall, toRemove := ComputeWindowsReconcileDeltas(
		[]*fleet.WindowsHostReconcileInfo{noTeamHost, teamedHost},
		nil,
		map[string][]*fleet.MDMWindowsProfilePayload{},
		profilesByTeam,
		nil,
	)
	require.Empty(t, toRemove)

	got := installSet(toInstall)
	require.Contains(t, got, key("h-global", "pg"))
	require.Contains(t, got, key("h-team", "pt"))
	require.NotContains(t, got, key("h-global", "pt"))
	require.NotContains(t, got, key("h-team", "pg"))
	require.Len(t, got, 2)
}

// TestComputeWindowsReconcileDeltasLabelMatrix confirms the compute routes the
// label gates through the shared dispatcher: include-all, include-any,
// exclude-any, combined include+exclude, broken labels, and dynamic-label
// timing. The handlers themselves are unit-tested in server/mdm/reconcile;
// here we assert the desired-state membership for representative cases.
func TestComputeWindowsReconcileDeltasLabelMatrix(t *testing.T) {
	hostLabels := map[uint]map[uint]struct{}{
		1: {10: {}, 11: {}}, // host 1 is a member of labels 10 and 11
	}
	host := &fleet.WindowsHostReconcileInfo{HostID: 1, UUID: "h1", TeamID: nil, LabelUpdatedAt: time.Now()}
	oldLabel := time.Now().Add(-24 * time.Hour)

	cases := []struct {
		name        string
		profile     *fleet.WindowsProfileForReconcile
		wantInstall bool
	}{
		{
			name: "include-all member of all installs",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				IncludeMode:   fleet.MDMProfileIncludeAll,
				IncludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(10))}, {LabelID: new(uint(11))}},
			},
			wantInstall: true,
		},
		{
			name: "include-all missing one does not install",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				IncludeMode:   fleet.MDMProfileIncludeAll,
				IncludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(10))}, {LabelID: new(uint(99))}},
			},
			wantInstall: false,
		},
		{
			name: "include-all with broken label does not install",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				IncludeMode:   fleet.MDMProfileIncludeAll,
				IncludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(10))}, {LabelID: nil}},
			},
			wantInstall: false,
		},
		{
			name: "include-any member of one installs",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				IncludeMode:   fleet.MDMProfileIncludeAny,
				IncludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(10))}, {LabelID: new(uint(99))}},
			},
			wantInstall: true,
		},
		{
			name: "include-any member of none does not install",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				IncludeMode:   fleet.MDMProfileIncludeAny,
				IncludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(98))}, {LabelID: new(uint(99))}},
			},
			wantInstall: false,
		},
		{
			name: "exclude-any non-member installs",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				ExcludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(99))}},
			},
			wantInstall: true,
		},
		{
			name: "exclude-any member does not install",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				ExcludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(10))}},
			},
			wantInstall: false,
		},
		{
			name: "exclude-any broken label does not install",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				ExcludeLabels: []fleet.MDMProfileLabelRef{{LabelID: nil}},
			},
			wantInstall: false,
		},
		{
			name: "include-all + exclude-any: in include, not in exclude installs",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				IncludeMode:   fleet.MDMProfileIncludeAll,
				IncludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(10))}},
				ExcludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(99))}},
			},
			wantInstall: true,
		},
		{
			name: "include-all + exclude-any: in include AND in exclude does not install",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				IncludeMode:   fleet.MDMProfileIncludeAll,
				IncludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(10))}},
				ExcludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(11))}},
			},
			wantInstall: false,
		},
		{
			name: "exclude-any dynamic label created after host scan disqualifies",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				// host is NOT a member of label 50, but the dynamic label was
				// created after the host's last label scan, so results are not
				// yet reported and the host is treated as excluded.
				ExcludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(50)), CreatedAt: time.Now().Add(time.Hour), LabelMembershipType: int(fleet.LabelMembershipTypeDynamic)}},
			},
			wantInstall: false,
		},
		{
			name: "exclude-any dynamic label created before host scan passes",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				ExcludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(50)), CreatedAt: oldLabel, LabelMembershipType: int(fleet.LabelMembershipTypeDynamic)}},
			},
			wantInstall: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			toInstall, toRemove := ComputeWindowsReconcileDeltas(
				[]*fleet.WindowsHostReconcileInfo{host},
				hostLabels,
				map[string][]*fleet.MDMWindowsProfilePayload{},
				map[uint][]*fleet.WindowsProfileForReconcile{0: {tc.profile}},
				nil,
			)
			require.Empty(t, toRemove)
			if tc.wantInstall {
				require.Contains(t, installSet(toInstall), key("h1", "p"))
			} else {
				require.Empty(t, toInstall)
			}
		})
	}
}
