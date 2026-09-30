package mysql

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

// openframeInventoryTenants seeds two tenants on one shared DB. Tenant A's host runs alpha 1.0 and
// shared 1.0; tenant B's host runs beta 1.0 and shared 2.0. CVEs: alpha 1.0 → CVE-2026-0001,
// beta 1.0 → CVE-2026-0002, shared 2.0 → CVE-2026-0003, B's OS → CVE-2026-0004.
type openframeInventoryTenants struct {
	teamA, teamB     *fleet.Team
	titleIDs         map[string]uint
	globalAdminScope fleet.TeamFilter
}

func seedOpenframeInventoryTenants(t *testing.T, ds *Datastore) openframeInventoryTenants {
	ctx := context.Background()

	teamA, err := ds.NewTeam(ctx, &fleet.Team{Name: "inv-tenant-a"})
	require.NoError(t, err)
	teamB, err := ds.NewTeam(ctx, &fleet.Team{Name: "inv-tenant-b"})
	require.NoError(t, err)

	hostA := test.NewHost(t, ds, "inv-host-a", "", "inv-key-a", "inv-uuid-a", time.Now())
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&teamA.ID, []uint{hostA.ID})))
	hostB := test.NewHost(t, ds, "inv-host-b", "", "inv-key-b", "inv-uuid-b", time.Now())
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&teamB.ID, []uint{hostB.ID})))

	_, err = ds.UpdateHostSoftware(ctx, hostA.ID, []fleet.Software{
		{Name: "alpha", Version: "1.0", Source: "apps"},
		{Name: "shared", Version: "1.0", Source: "apps"},
	})
	require.NoError(t, err)
	_, err = ds.UpdateHostSoftware(ctx, hostB.ID, []fleet.Software{
		{Name: "beta", Version: "1.0", Source: "apps"},
		{Name: "shared", Version: "2.0", Source: "apps"},
	})
	require.NoError(t, err)

	require.NoError(t, ds.UpdateHostOperatingSystem(ctx, hostA.ID, fleet.OperatingSystem{
		Name: "Microsoft Windows 11 Pro", Version: "10.0.22631.1", Arch: "x86_64", Platform: "windows",
	}))
	require.NoError(t, ds.UpdateHostOperatingSystem(ctx, hostB.ID, fleet.OperatingSystem{
		Name: "Microsoft Windows 11 Pro", Version: "10.0.22631.2", Arch: "x86_64", Platform: "windows",
	}))

	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.CleanupSoftwareTitles(ctx))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	globalAdminScope := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}, IncludeObserver: true}
	titles, _, _, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{}, globalAdminScope)
	require.NoError(t, err)
	titleIDs := make(map[string]uint, len(titles))
	versionIDs := make(map[string]uint)
	for _, title := range titles {
		titleIDs[title.Name] = title.ID
		for _, version := range title.Versions {
			versionIDs[title.Name+" "+version.Version] = version.ID
		}
	}

	for softwareKey, cve := range map[string]string{
		"alpha 1.0":  "CVE-2026-0001",
		"beta 1.0":   "CVE-2026-0002",
		"shared 2.0": "CVE-2026-0003",
	} {
		_, err := ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
			SoftwareID: versionIDs[softwareKey], CVE: cve,
		}, fleet.NVDSource)
		require.NoError(t, err)
	}

	osB, err := ds.GetHostOperatingSystem(ctx, hostB.ID)
	require.NoError(t, err)
	_, err = ds.InsertOSVulnerability(ctx, fleet.OSVulnerability{
		OSID: osB.ID, CVE: "CVE-2026-0004", ResolvedInVersion: ptr.String("10.0.22631.3"),
	}, fleet.MSRCSource)
	require.NoError(t, err)

	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))
	require.NoError(t, ds.UpdateOSVersions(ctx))
	require.NoError(t, ds.UpdateVulnerabilityHostCounts(ctx, 5))

	return openframeInventoryTenants{teamA: teamA, teamB: teamB, titleIDs: titleIDs, globalAdminScope: globalAdminScope}
}

func titleNames(titles []fleet.SoftwareTitleListResult) []string {
	names := make([]string, 0, len(titles))
	for _, title := range titles {
		names = append(names, title.Name)
	}
	sort.Strings(names)
	return names
}

// TestOpenframeSoftwareTitlesTeamFence verifies the OPENFRAME(mysql-multitenancy) fence on
// ListSoftwareTitles and SoftwareTitleByID: a pinned tenant sees only titles, versions and host
// counts from its own hosts, and an explicit team id cannot widen that. Runs only under MYSQL_TEST=1.
func TestOpenframeSoftwareTitlesTeamFence(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()
	tenants := seedOpenframeInventoryTenants(t, ds)

	all, count, _, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{}, tenants.globalAdminScope)
	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.Equal(t, []string{"alpha", "beta", "shared"}, titleNames(all))

	ctxA := fleet.NewOpenframeTeamContext(ctx, tenants.teamA.ID)
	own, count, _, err := ds.ListSoftwareTitles(ctxA, fleet.SoftwareTitleListOptions{}, tenants.globalAdminScope)
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.Equal(t, []string{"alpha", "shared"}, titleNames(own))
	for _, title := range own {
		require.EqualValues(t, 1, title.HostsCount, title.Name)
		require.EqualValues(t, 1, title.VersionsCount, title.Name)
	}

	widened, _, _, err := ds.ListSoftwareTitles(ctxA, fleet.SoftwareTitleListOptions{TeamID: &tenants.teamB.ID}, tenants.globalAdminScope)
	require.NoError(t, err)
	require.Equal(t, []string{"alpha", "shared"}, titleNames(widened))

	_, err = ds.SoftwareTitleByID(ctxA, tenants.titleIDs["beta"], nil, tenants.globalAdminScope)
	require.True(t, fleet.IsNotFound(err), "another tenant's title must not resolve, got %v", err)

	shared, err := ds.SoftwareTitleByID(ctxA, tenants.titleIDs["shared"], nil, tenants.globalAdminScope)
	require.NoError(t, err)
	require.Len(t, shared.Versions, 1)
	require.Equal(t, "1.0", shared.Versions[0].Version)
	require.EqualValues(t, 1, shared.HostsCount)

	// shared 2.0 carries CVE-2026-0003 but runs only on B's host: A's shared 1.0 must not qualify
	vulnerable, _, _, err := ds.ListSoftwareTitles(ctxA, fleet.SoftwareTitleListOptions{VulnerableOnly: true}, tenants.globalAdminScope)
	require.NoError(t, err)
	require.Equal(t, []string{"alpha"}, titleNames(vulnerable))

	byForeignCVE, _, _, err := ds.ListSoftwareTitles(ctxA,
		fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{MatchQuery: "CVE-2026-0003"}}, tenants.globalAdminScope)
	require.NoError(t, err)
	require.Empty(t, byForeignCVE)

	byOwnCVE, _, _, err := ds.ListSoftwareTitles(ctxA,
		fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{MatchQuery: "CVE-2026-0001"}}, tenants.globalAdminScope)
	require.NoError(t, err)
	require.Equal(t, []string{"alpha"}, titleNames(byOwnCVE))

	byOwnName, _, _, err := ds.ListSoftwareTitles(ctxA,
		fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{MatchQuery: "shared"}}, tenants.globalAdminScope)
	require.NoError(t, err)
	require.Equal(t, []string{"shared"}, titleNames(byOwnName))
	require.EqualValues(t, 1, byOwnName[0].VersionsCount)

	byForeignName, _, _, err := ds.ListSoftwareTitles(ctxA,
		fleet.SoftwareTitleListOptions{ListOptions: fleet.ListOptions{MatchQuery: "beta"}}, tenants.globalAdminScope)
	require.NoError(t, err)
	require.Empty(t, byForeignName)

	vulnerableByName, _, _, err := ds.ListSoftwareTitles(ctxA,
		fleet.SoftwareTitleListOptions{VulnerableOnly: true, ListOptions: fleet.ListOptions{MatchQuery: "shared"}}, tenants.globalAdminScope)
	require.NoError(t, err)
	require.Empty(t, vulnerableByName)

	ctxB := fleet.NewOpenframeTeamContext(ctx, tenants.teamB.ID)
	vulnerableForB, _, _, err := ds.ListSoftwareTitles(ctxB, fleet.SoftwareTitleListOptions{VulnerableOnly: true}, tenants.globalAdminScope)
	require.NoError(t, err)
	require.Equal(t, []string{"beta", "shared"}, titleNames(vulnerableForB))
}

// TestOpenframeSelectSoftwareTitlesSQLPinnedVersions verifies, without MySQL, that only a pinned
// request restricts the vulnerable/CVE-search software join to the team's own versions, so the
// unpinned statement stays upstream's.
func TestOpenframeSelectSoftwareTitlesSQLPinnedVersions(t *testing.T) {
	const teamVersions = "AND EXISTS (SELECT 1 FROM software_host_counts shc WHERE shc.software_id = s.id AND shc.team_id = 7 AND shc.global_stats = 0)"

	for _, opt := range []fleet.SoftwareTitleListOptions{
		{TeamID: ptr.Uint(7), VulnerableOnly: true, OpenframePinned: true},
		{TeamID: ptr.Uint(7), ListOptions: fleet.ListOptions{MatchQuery: "CVE-2026"}, OpenframePinned: true},
	} {
		pinned, _, err := selectSoftwareTitlesSQL(opt)
		require.NoError(t, err)
		require.Contains(t, pinned, teamVersions)

		opt.OpenframePinned = false
		unpinned, _, err := selectSoftwareTitlesSQL(opt)
		require.NoError(t, err)
		require.NotContains(t, unpinned, "software_host_counts shc")
	}
}

// TestOpenframeVulnerabilitiesTeamFence verifies the OPENFRAME(mysql-multitenancy) fence on
// ListVulnerabilities, CountVulnerabilities and Vulnerability: a pinned tenant sees only CVEs
// found on its own hosts. Runs only under MYSQL_TEST=1.
func TestOpenframeVulnerabilitiesTeamFence(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()
	tenants := seedOpenframeInventoryTenants(t, ds)

	all, _, err := ds.ListVulnerabilities(ctx, fleet.VulnListOptions{})
	require.NoError(t, err)
	require.Len(t, all, 4)

	ctxA := fleet.NewOpenframeTeamContext(ctx, tenants.teamA.ID)
	own, _, err := ds.ListVulnerabilities(ctxA, fleet.VulnListOptions{})
	require.NoError(t, err)
	require.Len(t, own, 1)
	require.Equal(t, "CVE-2026-0001", own[0].CVE.CVE)

	n, err := ds.CountVulnerabilities(ctxA, fleet.VulnListOptions{TeamID: &tenants.teamB.ID})
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	vuln, err := ds.Vulnerability(ctxA, "CVE-2026-0001", nil, false)
	require.NoError(t, err)
	require.EqualValues(t, 1, vuln.HostsCount)

	_, err = ds.Vulnerability(ctxA, "CVE-2026-0002", nil, false)
	require.True(t, fleet.IsNotFound(err), "another tenant's CVE must not resolve, got %v", err)
}

// TestOpenframeCVEDetailTeamFence verifies the OPENFRAME(mysql-multitenancy) fence on
// SoftwareByCVE and OSVersionsByCVE, which build the CVE detail page: a pinned tenant does not
// see another tenant's affected software or OS versions. Runs only under MYSQL_TEST=1.
func TestOpenframeCVEDetailTeamFence(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()
	tenants := seedOpenframeInventoryTenants(t, ds)

	software, _, err := ds.SoftwareByCVE(ctx, "CVE-2026-0003", nil)
	require.NoError(t, err)
	require.Len(t, software, 1)
	osVersions, _, err := ds.OSVersionsByCVE(ctx, "CVE-2026-0004", nil)
	require.NoError(t, err)
	require.Len(t, osVersions, 1)

	ctxA := fleet.NewOpenframeTeamContext(ctx, tenants.teamA.ID)
	software, _, err = ds.SoftwareByCVE(ctxA, "CVE-2026-0003", nil)
	require.NoError(t, err)
	require.Empty(t, software)
	osVersions, _, err = ds.OSVersionsByCVE(ctxA, "CVE-2026-0004", nil)
	require.NoError(t, err)
	require.Empty(t, osVersions)

	ctxB := fleet.NewOpenframeTeamContext(ctx, tenants.teamB.ID)
	software, _, err = ds.SoftwareByCVE(ctxB, "CVE-2026-0003", nil)
	require.NoError(t, err)
	require.Len(t, software, 1)
	require.Equal(t, "shared", software[0].Name)
}

// TestOpenframeVulnerabilityHostCountsUpdatedAt verifies the OPENFRAME(mysql-multitenancy)
// instance-wide recalculation time: zero before host counts were ever computed, the run time after,
// and the same value for a pinned tenant since it is instance metadata. Runs only under MYSQL_TEST=1.
func TestOpenframeVulnerabilityHostCountsUpdatedAt(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := context.Background()

	never, err := ds.VulnerabilityHostCountsUpdatedAt(ctx)
	require.NoError(t, err)
	require.True(t, never.IsZero())

	tenants := seedOpenframeInventoryTenants(t, ds)

	updatedAt, err := ds.VulnerabilityHostCountsUpdatedAt(ctx)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), updatedAt, 10*time.Minute)

	pinned, err := ds.VulnerabilityHostCountsUpdatedAt(fleet.NewOpenframeTeamContext(ctx, tenants.teamA.ID))
	require.NoError(t, err)
	require.Equal(t, updatedAt, pinned)
}
