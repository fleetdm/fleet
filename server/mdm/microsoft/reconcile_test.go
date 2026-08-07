package microsoft_mdm

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// deltaSet collapses a delta slice (install or remove) into a set of "hostUUID|profileUUID" keys so assertions don't depend on
// slice order.
func deltaSet(payloads []*fleet.MDMWindowsProfilePayload) map[string]struct{} {
	out := make(map[string]struct{}, len(payloads))
	for _, p := range payloads {
		out[p.HostUUID+"|"+p.ProfileUUID] = struct{}{}
	}
	return out
}

func key(hostUUID, profileUUID string) string { return hostUUID + "|" + profileUUID }

// TestComputeWindowsReconcileDeltasInstallRules covers install scenarios. The host is a no-team host with one global, label-less
// profile; only the current host_mdm_windows_profiles row varies between cases.
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
				require.Contains(t, deltaSet(toInstall), key("h1", "p1"))
				// install payload carries the desired profile's content.
				require.Equal(t, p.Checksum, toInstall[0].Checksum)
			} else {
				require.Empty(t, toInstall)
			}
		})
	}
}

// TestComputeWindowsReconcileDeltasRemoveRules covers the remove scenarios: current rows with no desired-state match are removed,
// except rows already processing a remove and except broken-label profiles.
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
			current:    &fleet.MDMWindowsProfilePayload{ProfileUUID: "gone", HostUUID: "h1", ProfileName: "Gone", OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryVerified), Detail: "prior detail", CommandUUID: "cmd-1"},
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
				got := toRemove[0]
				require.Equal(t, "gone", got.ProfileUUID)
				require.Equal(t, "h1", got.HostUUID)
				// Remove payloads carry these fields straight from the current row, matching the legacy remove query's SELECT list.
				require.Equal(t, tc.current.ProfileName, got.ProfileName)
				require.Equal(t, tc.current.OperationType, got.OperationType)
				require.Equal(t, tc.current.Status, got.Status)
				require.Equal(t, tc.current.Detail, got.Detail)
				require.Equal(t, tc.current.CommandUUID, got.CommandUUID)
			} else {
				require.Empty(t, toRemove)
			}
		})
	}
}

// TestComputeWindowsReconcileDeltasTeamGating verifies that a host only receives profiles for its own team; team_id=0 is its own
// scope, a teamed host does not inherit global profiles.
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

	got := deltaSet(toInstall)
	require.Contains(t, got, key("h-global", "pg"))
	require.Contains(t, got, key("h-team", "pt"))
	require.NotContains(t, got, key("h-global", "pt"))
	require.NotContains(t, got, key("h-team", "pg"))
	require.Len(t, got, 2)
}

// TestComputeWindowsReconcileDeltasLabelMatrix confirms the compute routes the label gates through the shared dispatcher:
// include-all, include-any, exclude-any, combined include+exclude, broken labels, and dynamic-label timing. The handlers
// themselves are unit-tested in server/mdm/reconcile; here we assert the desired-state membership for representative cases.
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
			name: "exclude-any dynamic label created after host scan withholds (profile not on host)",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				// host is NOT a member of label 50, but the dynamic label was created after the host's last label scan, so results are not yet
				// reported. The profile is not on the host, so the unknown membership keeps it withheld. (The on-host preservation side is
				// covered in TestComputeWindowsReconcileDeltasUnknownLabelPreservation.)
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
		{
			name: "include-any + exclude-any: in an include, not in exclude installs",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				IncludeMode:   fleet.MDMProfileIncludeAny,
				IncludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(10))}, {LabelID: new(uint(99))}},
				ExcludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(98))}},
			},
			wantInstall: true,
		},
		{
			name: "include-any + exclude-any: in an include AND in exclude does not install",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				IncludeMode:   fleet.MDMProfileIncludeAny,
				IncludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(10))}},
				ExcludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(11))}},
			},
			wantInstall: false,
		},
		{
			// Deliberate cross-platform decision (see server/mdm/reconcile): the exclude-any timing safeguard applies only to dynamic labels.
			// A host_vitals exclude label created after the host's last scan does NOT disqualify, so the profile still installs, unlike a
			// dynamic label in the same situation above.
			name: "exclude-any host_vitals label created after host scan still installs",
			profile: &fleet.WindowsProfileForReconcile{ProfileUUID: "p", TeamID: 0, Checksum: []byte("c"),
				ExcludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(50)), CreatedAt: time.Now().Add(time.Hour), LabelMembershipType: int(fleet.LabelMembershipTypeHostVitals)}},
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
				require.Contains(t, deltaSet(toInstall), key("h1", "p"))
			} else {
				require.Empty(t, toInstall)
			}
		})
	}
}

// TestComputeWindowsReconcileDeltasMultipleProfilesPerHost exercises the per-host loop building install AND remove sets in a
// single call: one host whose team has a profile to install and a profile already in the desired/installed state (no-op), while
// also carrying a current row for a profile no longer desired (remove). This is the realistic shape the
// single-profile/single-direction cases above don't cover.
func TestComputeWindowsReconcileDeltasMultipleProfilesPerHost(t *testing.T) {
	host := &fleet.WindowsHostReconcileInfo{HostID: 1, UUID: "h1", TeamID: nil}
	checksum := []byte("c")

	profilesByTeam := map[uint][]*fleet.WindowsProfileForReconcile{
		0: {
			{ProfileUUID: "p-install", ProfileName: "Install", TeamID: 0, Checksum: checksum},
			{ProfileUUID: "p-noop", ProfileName: "NoOp", TeamID: 0, Checksum: checksum},
		},
	}
	currentByHost := map[string][]*fleet.MDMWindowsProfilePayload{
		"h1": {
			// already installed and matching -> neither install nor remove.
			{ProfileUUID: "p-noop", HostUUID: "h1", Checksum: checksum, OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryVerified)},
			// installed but no longer desired (not in profilesByTeam) -> remove.
			{ProfileUUID: "p-remove", HostUUID: "h1", ProfileName: "Remove", Checksum: checksum, OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryVerified)},
		},
	}

	toInstall, toRemove := ComputeWindowsReconcileDeltas(
		[]*fleet.WindowsHostReconcileInfo{host},
		nil,
		currentByHost,
		profilesByTeam,
		nil,
	)

	install := deltaSet(toInstall)
	require.Len(t, install, 1)
	require.Contains(t, install, key("h1", "p-install"))
	require.NotContains(t, install, key("h1", "p-noop"))

	remove := deltaSet(toRemove)
	require.Len(t, remove, 1)
	require.Contains(t, remove, key("h1", "p-remove"))
	require.NotContains(t, remove, key("h1", "p-noop"))
}

// TestComputeWindowsReconcileDeltasUnknownLabelPreservation covers the state-preservation rule for dynamic labels the host has
// not evaluated yet (label created after the host's last label scan): the host's current profile state is preserved — kept when
// installed, withheld when not — until the host reports label results and membership becomes authoritative (see #47865).
func TestComputeWindowsReconcileDeltasUnknownLabelPreservation(t *testing.T) {
	scannedAt := time.Now().Add(-time.Hour)
	staleHost := &fleet.WindowsHostReconcileInfo{HostID: 1, UUID: "h1", TeamID: nil, LabelUpdatedAt: scannedAt}
	freshHost := &fleet.WindowsHostReconcileInfo{HostID: 1, UUID: "h1", TeamID: nil, LabelUpdatedAt: time.Now().Add(time.Hour)}
	unknownAt := time.Now() // after staleHost's scan, before freshHost's

	checksum := []byte("c")
	installedRow := &fleet.MDMWindowsProfilePayload{
		ProfileUUID: "p", HostUUID: "h1", Checksum: checksum,
		OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryVerified),
	}

	excProfile := &fleet.WindowsProfileForReconcile{ProfileUUID: "p", ProfileName: "P", TeamID: 0, Checksum: checksum,
		ExcludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(50)), CreatedAt: unknownAt, LabelMembershipType: int(fleet.LabelMembershipTypeDynamic)}},
	}
	incProfile := &fleet.WindowsProfileForReconcile{ProfileUUID: "p", ProfileName: "P", TeamID: 0, Checksum: checksum,
		IncludeMode: fleet.MDMProfileIncludeAll,
		IncludeLabels: []fleet.MDMProfileLabelRef{
			{LabelID: new(uint(10))},
			{LabelID: new(uint(50)), CreatedAt: unknownAt, LabelMembershipType: int(fleet.LabelMembershipTypeDynamic)},
		},
	}
	// host is a confirmed member of the pre-existing include label 10 only.
	memberOf10 := map[uint]map[uint]struct{}{1: {10: {}}}

	cases := []struct {
		name        string
		host        *fleet.WindowsHostReconcileInfo
		hostLabels  map[uint]map[uint]struct{}
		profile     *fleet.WindowsProfileForReconcile
		installed   bool
		wantInstall bool
		wantRemove  bool
	}{
		{
			name: "unknown exclude label keeps installed profile",
			host: staleHost, profile: excProfile, installed: true,
		},
		{
			name: "unknown exclude label withholds uninstalled profile",
			host: staleHost, profile: excProfile,
		},
		{
			name: "exclude membership confirmed after scan removes profile",
			host: freshHost, hostLabels: map[uint]map[uint]struct{}{1: {50: {}}}, profile: excProfile, installed: true,
			wantRemove: true,
		},
		{
			name: "exclude non-membership confirmed after scan installs profile",
			host: freshHost, profile: excProfile,
			wantInstall: true,
		},
		{
			name: "unknown include-all label keeps installed profile",
			host: staleHost, hostLabels: memberOf10, profile: incProfile, installed: true,
		},
		{
			name: "unknown include-all label withholds uninstalled profile",
			host: staleHost, hostLabels: memberOf10, profile: incProfile,
		},
		{
			name: "include-all non-membership confirmed after scan removes profile",
			host: freshHost, hostLabels: memberOf10, profile: incProfile, installed: true,
			wantRemove: true,
		},
		{
			name: "include-all with confirmed non-membership of another label removes despite unknown label",
			host: staleHost, hostLabels: map[uint]map[uint]struct{}{1: {}}, profile: incProfile, installed: true,
			wantRemove: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			currentByHost := map[string][]*fleet.MDMWindowsProfilePayload{}
			if tc.installed {
				currentByHost["h1"] = []*fleet.MDMWindowsProfilePayload{installedRow}
			}
			toInstall, toRemove := ComputeWindowsReconcileDeltas(
				[]*fleet.WindowsHostReconcileInfo{tc.host},
				tc.hostLabels,
				currentByHost,
				map[uint][]*fleet.WindowsProfileForReconcile{0: {tc.profile}},
				nil,
			)
			if tc.wantInstall {
				require.Contains(t, deltaSet(toInstall), key("h1", "p"))
			} else {
				require.Empty(t, toInstall)
			}
			if tc.wantRemove {
				require.Contains(t, deltaSet(toRemove), key("h1", "p"))
			} else {
				require.Empty(t, toRemove)
			}
		})
	}
}

// TestDesiredWindowsProfileUUIDsByHost covers the per-host desired-state map that the reconciler uses to protect a removed profile's
// LocURIs from being deleted on hosts where another still-applicable profile enforces them. It must apply the same team-gating and
// per-host label rules as ComputeWindowsReconcileDeltas: a label-scoped profile appears only for the hosts it actually matches, and a
// host with no applicable profiles is omitted from the map entirely.
func TestDesiredWindowsProfileUUIDsByHost(t *testing.T) {
	// host 1 is a member of label 10; the other hosts are members of nothing.
	hostLabels := map[uint]map[uint]struct{}{
		1: {10: {}},
	}

	pGlobal := &fleet.WindowsProfileForReconcile{ProfileUUID: "p-global", ProfileName: "global", TeamID: 0, Checksum: []byte("c")}
	pLabeled := &fleet.WindowsProfileForReconcile{ProfileUUID: "p-labeled", ProfileName: "labeled", TeamID: 0, Checksum: []byte("c"),
		IncludeMode:   fleet.MDMProfileIncludeAny,
		IncludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(10))}},
	}
	pTeam := &fleet.WindowsProfileForReconcile{ProfileUUID: "p-team", ProfileName: "team", TeamID: 5, Checksum: []byte("c")}

	profilesByTeam := map[uint][]*fleet.WindowsProfileForReconcile{
		0: {pGlobal, pLabeled},
		5: {pTeam},
	}

	labeledHost := &fleet.WindowsHostReconcileInfo{HostID: 1, UUID: "h-labeled", TeamID: nil, LabelUpdatedAt: time.Now()}
	plainHost := &fleet.WindowsHostReconcileInfo{HostID: 2, UUID: "h-plain", TeamID: nil, LabelUpdatedAt: time.Now()}
	teamHost := &fleet.WindowsHostReconcileInfo{HostID: 3, UUID: "h-team", TeamID: new(uint(5)), LabelUpdatedAt: time.Now()}
	emptyHost := &fleet.WindowsHostReconcileInfo{HostID: 4, UUID: "h-empty", TeamID: new(uint(9)), LabelUpdatedAt: time.Now()}

	out := DesiredWindowsProfileUUIDsByHost(
		[]*fleet.WindowsHostReconcileInfo{labeledHost, plainHost, teamHost, emptyHost},
		hostLabels,
		nil,
		profilesByTeam,
	)

	// h-empty is in a team with no profiles, so it must not appear in the map at all.
	require.Len(t, out, 3)
	require.NotContains(t, out, "h-empty")

	// The labeled host gets both the global profile and the label-scoped one; the plain host (same team, not in the label) gets
	// only the global profile. This differential is exactly what makes the LocURI protection per host.
	require.ElementsMatch(t, []string{"p-global", "p-labeled"}, out["h-labeled"])
	require.ElementsMatch(t, []string{"p-global"}, out["h-plain"])

	// Team gating: the teamed host sees only its own team's profile, never the no-team profiles.
	require.ElementsMatch(t, []string{"p-team"}, out["h-team"])

	// A profile kept on a host only through unknown-label preservation is still desired there, so its LocURIs stay protected —
	// consistent with ComputeWindowsReconcileDeltas not removing it.
	pUnknownExc := &fleet.WindowsProfileForReconcile{ProfileUUID: "p-exc", ProfileName: "exc", TeamID: 0, Checksum: []byte("c"),
		ExcludeLabels: []fleet.MDMProfileLabelRef{{LabelID: new(uint(50)), CreatedAt: time.Now().Add(time.Hour), LabelMembershipType: int(fleet.LabelMembershipTypeDynamic)}},
	}
	currentByHost := map[string][]*fleet.MDMWindowsProfilePayload{
		"h-labeled": {{ProfileUUID: "p-exc", HostUUID: "h-labeled", OperationType: fleet.MDMOperationTypeInstall, Status: new(fleet.MDMDeliveryVerified)}},
	}
	out = DesiredWindowsProfileUUIDsByHost(
		[]*fleet.WindowsHostReconcileInfo{labeledHost, plainHost},
		hostLabels,
		currentByHost,
		map[uint][]*fleet.WindowsProfileForReconcile{0: {pUnknownExc}},
	)
	require.ElementsMatch(t, []string{"p-exc"}, out["h-labeled"])
	require.NotContains(t, out, "h-plain")
}
