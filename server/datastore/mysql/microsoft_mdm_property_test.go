package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

// TestPBT_ScopedListingEquivalence is a property-based test for the scoped
// Windows MDM profile listings introduced for the cron's batched
// reconciliation. It asserts that for any subset of host UUIDs S,
//
//	ListMDMWindowsProfilesToInstallForHosts(S) ≡ filter(ListMDMWindowsProfilesToInstall(),  HostUUID ∈ S)
//	ListMDMWindowsProfilesToRemoveForHosts(S)  ≡ filter(ListMDMWindowsProfilesToRemove(),   HostUUID ∈ S)
//
// Background: the scoped listings reuse the same SQL templates as the
// global ones, with "TRUE" substituted for the host filter slots in the
// global form and an "IN (?)" predicate substituted in the scoped form.
// A subtle change to either query (e.g., adding a new join condition,
// reordering predicates, swapping LEFT and RIGHT joins) could break this
// equivalence in ways that single-subset tests would miss but the cron
// would surface as drifting visit sets.
//
// Run with more checks:
//
//	MYSQL_TEST=1 go test -run TestPBT_ScopedListingEquivalence \
//	  ./server/datastore/mysql/ -args -rapid.checks=2000
func TestPBT_ScopedListingEquivalence(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	// This property test exercises the production async path. The eager
	// hook is off by default, so no opt-in is needed.

	// Two teams plus the implicit "no team" (team_id=0). Hosts and
	// profiles span all three so both team-scoped and global predicates
	// participate in the desired-state JOIN.
	teamA, err := ds.NewTeam(ctx, &fleet.Team{Name: "pbt-equiv-A"})
	require.NoError(t, err)
	teamB, err := ds.NewTeam(ctx, &fleet.Team{Name: "pbt-equiv-B"})
	require.NoError(t, err)

	pGlobal := InsertWindowsProfileForTest(t, ds, 0)
	pTeamA := InsertWindowsProfileForTest(t, ds, teamA.ID)
	pTeamB := InsertWindowsProfileForTest(t, ds, teamB.ID)

	// 6 windows hosts, 2 per scope. All MDM-enrolled so the desired-state
	// predicate surfaces them.
	type hostSpec struct {
		name string
		ip   string
		team *uint
	}
	specs := []hostSpec{
		{"pbt-eq-noteam-1", "10.0.0.1", nil},
		{"pbt-eq-noteam-2", "10.0.0.2", nil},
		{"pbt-eq-teamA-1", "10.0.0.3", &teamA.ID},
		{"pbt-eq-teamA-2", "10.0.0.4", &teamA.ID},
		{"pbt-eq-teamB-1", "10.0.0.5", &teamB.ID},
		{"pbt-eq-teamB-2", "10.0.0.6", &teamB.ID},
	}
	hosts := make([]*fleet.Host, 0, len(specs))
	hostUUIDs := make([]string, 0, len(specs))
	for _, s := range specs {
		opts := []test.NewHostOption{test.WithPlatform("windows")}
		if s.team != nil {
			opts = append(opts, test.WithTeamID(*s.team))
		}
		h := test.NewHost(t, ds, s.name, s.ip, uuid.NewString(), uuid.NewString(), time.Now(), opts...)
		windowsEnroll(t, ds, h)
		hosts = append(hosts, h)
		hostUUIDs = append(hostUUIDs, h.UUID)
	}

	// Seed four verified install rows on (host, profile) pairs that
	// either are not, or will not be, in the host's desired state. Then
	// transfer two hosts. Each seeded row falls into the "host has a
	// row but desired state disagrees" bucket, which is the remove set;
	// the desired-state pairs that lack any actual row land in the
	// install set. The resulting population covers three axes the
	// property must discriminate on:
	//
	//   1. JOIN cardinality on the same profile across hosts.
	//      hosts[1] (no team) and hosts[2] (team A → B) both end up
	//      with pTeamA verified, and neither's desired state includes
	//      pTeamA — a "no team" host doesn't get team A's profile, and
	//      hosts[2] is now in team B. Both contribute a remove row for
	//      pTeamA, exercising JOIN cardinality on the remove side.
	//
	//   2. Multiple distinct profiles in the remove set.
	//      hosts[3] holds pGlobal verified while in team A, so pGlobal
	//      (a "no team" profile in Fleet's scoping) is in the remove
	//      set for hosts[3]. hosts[5] (team B → A) holds pTeamB
	//      verified but pTeamB is not in team A's desired state, so
	//      pTeamB is also in the remove set. Together with pTeamA
	//      from axis 1, the remove set spans three distinct profiles.
	//
	//   3. Hosts simultaneously in install and remove listings.
	//      Every seeded host (hosts[1], hosts[2], hosts[3], hosts[5])
	//      ends up with both an install-pending row (for the missing
	//      desired-state profile) and a remove-pending row (for the
	//      leftover seeded row), so a random subset including any of
	//      these exercises the property across both listings at once.
	installWindowsProfilesAsVerified(t, ds, []string{hosts[1].UUID}, []string{pTeamA})
	installWindowsProfilesAsVerified(t, ds, []string{hosts[2].UUID}, []string{pTeamA})
	installWindowsProfilesAsVerified(t, ds, []string{hosts[3].UUID}, []string{pGlobal})
	installWindowsProfilesAsVerified(t, ds, []string{hosts[5].UUID}, []string{pTeamB})

	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&teamB.ID, []uint{hosts[2].ID})))
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&teamA.ID, []uint{hosts[5].ID})))

	// Pin the seeded population's shape so a future change in the
	// listing queries is surfaced here rather than as a silent loss of
	// discriminative power inside the property check.
	globalInstall, err := ds.ListMDMWindowsProfilesToInstall(ctx)
	require.NoError(t, err)
	globalRemove, err := ds.ListMDMWindowsProfilesToRemove(ctx)
	require.NoError(t, err)

	// Expected install rows (6): each of the 6 desired-state pairs is
	// missing an actual row at the (host, profile) level (the seeded
	// rows are on different profiles than each host's desired state).
	//   hosts[0] no team → pGlobal
	//   hosts[1] no team → pGlobal
	//   hosts[2] team B  → pTeamB
	//   hosts[3] team A  → pTeamA
	//   hosts[4] team B  → pTeamB
	//   hosts[5] team A  → pTeamA
	require.Lenf(t, globalInstall, 6,
		"seeded install population has the wrong shape; got %d rows: %+v",
		len(globalInstall), globalInstall)

	// Expected remove rows (4): each seeded row falls outside its
	// host's desired state.
	//   hosts[1] pTeamA, hosts[2] pTeamA  ← shared profile, two hosts
	//   hosts[3] pGlobal
	//   hosts[5] pTeamB
	require.Lenf(t, globalRemove, 4,
		"seeded remove population has the wrong shape; got %d rows: %+v",
		len(globalRemove), globalRemove)
	removeHostsByProfile := map[string]map[string]bool{}
	for _, p := range globalRemove {
		if removeHostsByProfile[p.ProfileUUID] == nil {
			removeHostsByProfile[p.ProfileUUID] = map[string]bool{}
		}
		removeHostsByProfile[p.ProfileUUID][p.HostUUID] = true
	}
	require.Lenf(t, removeHostsByProfile[pTeamA], 2,
		"expected pTeamA remove rows on 2 distinct hosts; got %v", removeHostsByProfile[pTeamA])
	require.Lenf(t, removeHostsByProfile[pTeamB], 1,
		"expected pTeamB remove rows on 1 host; got %v", removeHostsByProfile[pTeamB])
	require.Lenf(t, removeHostsByProfile[pGlobal], 1,
		"expected pGlobal remove rows on 1 host; got %v", removeHostsByProfile[pGlobal])

	// At least one host must appear in both install and remove listings,
	// otherwise the property cannot exercise mixed-state hosts.
	removeHostSet := map[string]struct{}{}
	for _, p := range globalRemove {
		removeHostSet[p.HostUUID] = struct{}{}
	}
	mixedHosts := map[string]struct{}{}
	for _, p := range globalInstall {
		if _, ok := removeHostSet[p.HostUUID]; ok {
			mixedHosts[p.HostUUID] = struct{}{}
		}
	}
	require.GreaterOrEqualf(t, len(mixedHosts), 4,
		"expected at least 4 hosts with both install and remove pending rows; got %d", len(mixedHosts))

	rapid.Check(t, func(rt *rapid.T) {
		// Draw any subset (incl. empty, incl. with duplicates). Duplicate
		// host UUIDs in the input matter because they exercise the
		// sqlx.In expansion: the DB should de-duplicate via the JOIN.
		subset := rapid.SliceOf(rapid.SampledFrom(hostUUIDs)).Draw(rt, "subset")

		scopedInstall, err := ds.ListMDMWindowsProfilesToInstallForHosts(ctx, subset)
		require.NoError(rt, err)
		scopedRemove, err := ds.ListMDMWindowsProfilesToRemoveForHosts(ctx, subset)
		require.NoError(rt, err)

		subsetSet := make(map[string]struct{}, len(subset))
		for _, h := range subset {
			subsetSet[h] = struct{}{}
		}
		var expectedInstall []*fleet.MDMWindowsProfilePayload
		for _, p := range globalInstall {
			if _, ok := subsetSet[p.HostUUID]; ok {
				expectedInstall = append(expectedInstall, p)
			}
		}
		var expectedRemove []*fleet.MDMWindowsProfilePayload
		for _, p := range globalRemove {
			if _, ok := subsetSet[p.HostUUID]; ok {
				expectedRemove = append(expectedRemove, p)
			}
		}

		require.ElementsMatchf(rt, expectedInstall, scopedInstall,
			"scoped install ≠ filter(global, host ∈ subset); subset=%v", subset)
		require.ElementsMatchf(rt, expectedRemove, scopedRemove,
			"scoped remove ≠ filter(global, host ∈ subset); subset=%v", subset)
	})
}
