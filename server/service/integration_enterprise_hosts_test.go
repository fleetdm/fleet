package service

// Host tests for the enterprise (premium) suite.
//
// Belongs here: listing and filtering hosts (including by software on another fleet), host health,
// host device mapping from the IdP, a host's past activities, OS versions, and the vulnerabilities
// list.
//
// Does not belong here: host enrollment and agent behavior (integration_enterprise_orbit_osquery_test.go),
// a host's software (integration_enterprise_software_test.go), scripts
// (integration_enterprise_scripts_test.go), and MDM actions such as lock/wipe
// (integration_enterprise_mdm_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestListHosts() {
	t := s.T()

	// create a couple of hosts
	host1, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	host2, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name() + "2"),
		NodeKey:         new(t.Name() + "2"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sbar.local", t.Name()),
		Platform:        "linux",
	})
	require.NoError(t, err)
	host3, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name() + "3"),
		NodeKey:         new(t.Name() + "3"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sbaz.local", t.Name()),
		Platform:        "windows",
	})
	require.NoError(t, err)
	require.NotNil(t, host3)

	// set disk space information for some hosts (none provided for host3)
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), host1.ID, 10.0, 2.0, 500.0, nil))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), host2.ID, 32.0, 4.0, 1000.0, new(float64(1200.0))))

	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)
	require.Len(t, resp.Hosts, 3)

	allHostsLabel, err := s.ds.GetLabelSpec(context.Background(), fleet.TeamFilter{}, "All hosts")
	require.NoError(t, err)
	for _, h := range resp.Hosts {
		err = s.ds.RecordLabelQueryExecutions(
			context.Background(), h.Host, map[uint]*bool{allHostsLabel.ID: new(true)}, time.Now(), false,
		)
		require.NoError(t, err)
	}

	resp = listHostsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", allHostsLabel.ID), nil, http.StatusOK, &resp, "low_disk_space", "32")
	require.Len(t, resp.Hosts, 1)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "low_disk_space", "32")
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, host1.ID, resp.Hosts[0].ID)
	assert.InDelta(t, 10.0, resp.Hosts[0].GigsDiskSpaceAvailable, 0.001)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "low_disk_space", "100")
	require.Len(t, resp.Hosts, 2)

	// returns an error when the low_disk_space value is invalid (outside 1-100)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &resp, "low_disk_space", "101")
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &resp, "low_disk_space", "0")

	// counting hosts works with and without the filter too
	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp)
	require.Equal(t, 3, countResp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "low_disk_space", "32")
	require.Equal(t, 1, countResp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "low_disk_space", "100")
	require.Equal(t, 2, countResp.Count)

	// host summary returns counts for low disk space
	var summaryResp getHostSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "low_disk_space", "32")
	require.Equal(t, uint(3), summaryResp.TotalsHostsCount)
	require.NotNil(t, summaryResp.LowDiskSpaceCount)
	require.Equal(t, uint(1), *summaryResp.LowDiskSpaceCount)

	summaryResp = getHostSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "platform", "windows", "low_disk_space", "32")
	require.Equal(t, uint(1), summaryResp.TotalsHostsCount)
	require.NotNil(t, summaryResp.LowDiskSpaceCount)
	require.Equal(t, uint(0), *summaryResp.LowDiskSpaceCount)

	// all possible filters
	summaryResp = getHostSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "team_id", "1", "platform", "linux", "low_disk_space", "32")
	require.Equal(t, uint(0), summaryResp.TotalsHostsCount)
	require.NotNil(t, summaryResp.LowDiskSpaceCount)
	require.Equal(t, uint(0), *summaryResp.LowDiskSpaceCount)

	// without low_disk_space, does not return the count
	summaryResp = getHostSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "team_id", "1", "platform", "linux")
	require.Equal(t, uint(0), summaryResp.TotalsHostsCount)
	require.Nil(t, summaryResp.LowDiskSpaceCount)

	// Add a failing policy
	ctx := context.Background()
	qr, err := s.ds.NewQuery(
		ctx, &fleet.Query{
			Name:           "TestQueryEnterpriseTestListHosts",
			Description:    "Some description",
			Query:          "select * from osquery;",
			ObserverCanRun: true,
			Logging:        fleet.LoggingSnapshot,
		},
	)
	require.NoError(t, err)

	// add a global policy
	gpParams := fleet.GlobalPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some global resolution",
	}
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)

	// add a failing policy execution
	require.NoError(
		t, errOnly(s.ds.RecordPolicyQueryExecutions(
			ctx, host1,
			map[uint]*bool{gpResp.Policy.ID: new(false)}, time.Now(), false, nil,
		)),
	)

	// populate software for hosts
	now := time.Now()
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host1.ID, software)
	require.NoError(t, err)

	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host1, false))

	inserted, err := s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: host1.Software[0].ID,
		CVE:        "cve-123-123-123",
	}, fleet.NVDSource)
	require.NoError(t, err)
	require.True(t, inserted)

	vulnMeta := []fleet.CVEMeta{{
		CVE:              "cve-123-123-123",
		CVSSScore:        new(float64(9.8)),
		EPSSProbability:  new(float64(0.5)),
		CISAKnownExploit: new(true),
		Published:        &now,
		Description:      "a long description of the cve",
	}}

	require.NoError(t, s.ds.InsertCVEMeta(context.Background(), vulnMeta))
	ctx = license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
	require.NoError(t, s.ds.UpdateHostIssuesVulnerabilities(ctx))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_software", "true")
	require.Len(t, resp.Hosts, 3)
	for _, h := range resp.Hosts {
		if h.ID == host1.ID {
			require.NotEmpty(t, h.Software)
			require.Len(t, h.Software, 1)
			require.NotEmpty(t, h.Software[0].Vulnerabilities)

			s := &vulnMeta[0].Description
			require.Equal(t, &vulnMeta[0].CVSSScore, h.Software[0].Vulnerabilities[0].CVSSScore)
			require.Equal(t, &vulnMeta[0].EPSSProbability, h.Software[0].Vulnerabilities[0].EPSSProbability)
			require.Equal(t, &vulnMeta[0].CISAKnownExploit, h.Software[0].Vulnerabilities[0].CISAKnownExploit)
			require.Equal(t, &s, h.Software[0].Vulnerabilities[0].Description)
			assert.Equal(t, uint64(1), h.HostIssues.FailingPoliciesCount)
			assert.Equal(t, uint64(1), *h.HostIssues.CriticalVulnerabilitiesCount)
			assert.Equal(t, uint64(2), h.HostIssues.TotalIssuesCount)
		} else {
			assert.Zero(t, h.HostIssues.FailingPoliciesCount)
			assert.Zero(t, *h.HostIssues.CriticalVulnerabilitiesCount)
			assert.Zero(t, h.HostIssues.TotalIssuesCount)
		}
	}

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_software", "without_vulnerability_details")
	require.Len(t, resp.Hosts, 3)
	for _, h := range resp.Hosts {
		if h.ID == host1.ID {
			require.NotEmpty(t, h.Software)
			require.Len(t, h.Software, 1)
			require.NotEmpty(t, h.Software[0].Vulnerabilities)

			require.Nil(t, h.Software[0].Vulnerabilities[0].CVSSScore)
			require.Nil(t, h.Software[0].Vulnerabilities[0].EPSSProbability)
			require.Nil(t, h.Software[0].Vulnerabilities[0].CISAKnownExploit)
			require.Nil(t, h.Software[0].Vulnerabilities[0].Description)
			assert.Equal(t, uint64(1), h.HostIssues.FailingPoliciesCount)
			assert.Equal(t, uint64(1), *h.HostIssues.CriticalVulnerabilitiesCount)
			assert.Equal(t, uint64(2), h.HostIssues.TotalIssuesCount)
		} else {
			assert.Zero(t, h.HostIssues.FailingPoliciesCount)
			assert.Zero(t, *h.HostIssues.CriticalVulnerabilitiesCount)
			assert.Zero(t, h.HostIssues.TotalIssuesCount)
		}
	}

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_software", "false")
	require.Len(t, resp.Hosts, 3)
	for _, h := range resp.Hosts {
		require.Empty(t, h.Software)
	}

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", allHostsLabel.ID), nil, http.StatusOK, &resp)
	assert.Len(t, resp.Hosts, 3)
	for _, h := range resp.Hosts {
		if h.ID == host1.ID {
			assert.Equal(t, uint64(1), h.HostIssues.FailingPoliciesCount)
			assert.Equal(t, uint64(1), *h.HostIssues.CriticalVulnerabilitiesCount)
			assert.Equal(t, uint64(2), h.HostIssues.TotalIssuesCount)
		} else {
			assert.Zero(t, h.HostIssues.FailingPoliciesCount)
			assert.Zero(t, *h.HostIssues.CriticalVulnerabilitiesCount)
			assert.Zero(t, h.HostIssues.TotalIssuesCount)
		}
	}

	// Test ordering by issues
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "order_key", "issues",
	) // defaults to ascending order (lowest issues to most issues)
	require.Len(t, resp.Hosts, 3)
	assert.Equal(t, host1.ID, resp.Hosts[2].ID)
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", allHostsLabel.ID), nil, http.StatusOK, &resp, "order_key", "issues",
		"order_direction", "desc",
	)
	require.Len(t, resp.Hosts, 3)
	assert.Equal(t, host1.ID, resp.Hosts[0].ID)
}

func (s *integrationEnterpriseTestSuite) TestListHostsSoftwareVersionOnDifferentTeam() {
	t := s.T()
	ctx := context.Background()

	// create 2 teams
	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_team2"})
	require.NoError(t, err)

	// create 2 hosts
	h1, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   new(t.Name() + "h1"),
		NodeKey:         new(t.Name() + "h1"),
		UUID:            uuid.New().String(),
		Hostname:        t.Name() + "h1.local",
		Platform:        "darwin",
		TeamID:          &team1.ID,
	})
	require.NoError(t, err)

	h2, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   new(t.Name() + "h2"),
		NodeKey:         new(t.Name() + "h2"),
		UUID:            uuid.New().String(),
		Hostname:        t.Name() + "h2.local",
		Platform:        "darwin",
		TeamID:          &team2.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, h2.ID)

	// Install software only on h1 (team1).

	testSw := fleet.Software{Name: "UniqueApp", Version: "3.4.5", Source: "apps", BundleIdentifier: "com.unique.app"}
	_, err = s.ds.UpdateHostSoftware(ctx, h1.ID, []fleet.Software{
		testSw,
	})
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, h1, false))
	require.Len(t, h1.Software, 1)
	swVersionID := h1.Software[0].ID

	// Sync host counts so the team-scoped queries work.
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	// Filtering team1 by the software version returns the host and software including its bundle id.
	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp,
		"software_version_id", fmt.Sprint(swVersionID),
		"team_id", fmt.Sprint(team1.ID),
	)
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, h1.ID, resp.Hosts[0].ID)
	require.NotNil(t, resp.Software)
	assert.Equal(t, testSw.Name, resp.Software.Name)
	assert.Equal(t, testSw.Version, resp.Software.Version)
	assert.Equal(t, testSw.BundleIdentifier, resp.Software.BundleIdentifier)

	// Filtering team2 (no hosts with this software) returns 0 hosts but still
	// returns minimal software data
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp,
		"software_version_id", fmt.Sprint(swVersionID),
		"team_id", fmt.Sprint(team2.ID),
	)
	require.Empty(t, resp.Hosts)
	require.NotNil(t, resp.Software, "software metadata should be returned even when no hosts match on this team")
	assert.Equal(t, swVersionID, resp.Software.ID)
	assert.Equal(t, testSw.Name, resp.Software.Name)
	assert.Equal(t, testSw.Version, resp.Software.Version)
	assert.Empty(t, resp.Software.BundleIdentifier)

	// Filtering with a completely non-existent software_version_id → returns nil software.
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp,
		"software_version_id", "999999",
		"team_id", fmt.Sprint(team2.ID),
	)
	require.Empty(t, resp.Hosts)
	assert.Empty(t, resp.Software)
}

func (s *integrationEnterpriseTestSuite) TestListHostsSoftwareTitleOnDifferentTeam() {
	t := s.T()
	ctx := t.Context()

	// create 2 teams
	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_team2"})
	require.NoError(t, err)

	// create 1 host on team1
	h1, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   new(t.Name() + "h1"),
		NodeKey:         new(t.Name() + "h1"),
		UUID:            uuid.New().String(),
		Hostname:        t.Name() + "h1.local",
		Platform:        "darwin",
		TeamID:          &team1.ID,
	})
	require.NoError(t, err)

	// Install software only on h1 (team1).
	testSw := fleet.Software{Name: "UniqueTitleApp", Version: "3.4.5", Source: "apps", BundleIdentifier: "com.unique.titleapp"}
	_, err = s.ds.UpdateHostSoftware(ctx, h1.ID, []fleet.Software{testSw})
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, h1, false))
	require.Len(t, h1.Software, 1)
	require.NotNil(t, h1.Software[0].TitleID)
	titleID := *h1.Software[0].TitleID

	// Deliberately do NOT call SyncHostsSoftwareTitles here: the title's
	// entry in software_titles_host_counts (which SoftwareTitleByID relies
	// on) is only populated by that periodic sync, so skipping it
	// reproduces the up-to-~1h window between a host reporting new
	// software and the next sync run. The in-scope enrichment below must
	// still succeed immediately via a live (non-aggregated) fallback.

	// Filtering team1 (in-scope) by the software title returns the host and the title's name.
	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp,
		"software_title_id", fmt.Sprint(titleID),
		"team_id", fmt.Sprint(team1.ID),
	)
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, h1.ID, resp.Hosts[0].ID)
	require.NotNil(t, resp.SoftwareTitle)
	assert.Equal(t, testSw.Name, resp.SoftwareTitle.Name)

	// Filtering team2 (out-of-scope: the title isn't installed on any host on
	// this team) must not leak the title's name/display_name — software_title
	// should be omitted entirely, not backfilled from an unscoped lookup.
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp,
		"software_title_id", fmt.Sprint(titleID),
		"team_id", fmt.Sprint(team2.ID),
	)
	require.Empty(t, resp.Hosts)
	assert.Nil(t, resp.SoftwareTitle)
}

func (s *integrationEnterpriseTestSuite) TestHostHealth() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		OsqueryHostID:   new(t.Name() + "hostid1"),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "nodekey1"),
		UUID:            t.Name() + "uuid1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OSVersion:       "Mac OS X 10.14.6",
		Platform:        "darwin",
		CPUType:         "cpuType",
		TeamID:          new(team.ID),
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	passingTeamPolicy, err := s.ds.NewTeamPolicy(context.Background(), team.ID, nil, fleet.PolicyPayload{
		Name:       "Passing Global Policy",
		Query:      "select 1",
		Resolution: "Run this command to fix it",
	})
	require.NoError(t, err)

	failingTeamPolicy, err := s.ds.NewTeamPolicy(context.Background(), team.ID, nil, fleet.PolicyPayload{
		Name:       "Failing Global Policy",
		Query:      "select 1",
		Resolution: "Run this command to fix it",
		Critical:   true,
	})
	require.NoError(t, err)

	passingGlobalPolicy, err := s.ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{
		Name:       "Passing Global Policy",
		Query:      "select 1",
		Resolution: "Run this command to fix it",
	})
	require.NoError(t, err)

	failingGlobalPolicy, err := s.ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{
		Name:       "Failing Global Policy",
		Query:      "select 1",
		Resolution: "Run this command to fix it",
		Critical:   false,
	})
	require.NoError(t, err)

	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{failingGlobalPolicy.ID: new(false)}, time.Now(), false, nil)))
	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{passingGlobalPolicy.ID: new(true)}, time.Now(), false, nil)))
	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{failingTeamPolicy.ID: new(false)}, time.Now(), false, nil)))
	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{passingTeamPolicy.ID: new(true)}, time.Now(), false, nil)))

	hh := getHostHealthResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/health", host.ID), nil, http.StatusOK, &hh)
	require.Equal(t, host.ID, hh.HostID)
	assert.NotNil(t, hh.HostHealth)
	assert.Equal(t, host.OSVersion, hh.HostHealth.OsVersion)
	assert.Equal(t, 2, hh.HostHealth.FailingPoliciesCount)
	assert.Equal(t, new(int(1)), hh.HostHealth.FailingCriticalPoliciesCount)
	assert.Contains(t, hh.HostHealth.FailingPolicies, &fleet.HostHealthFailingPolicy{
		ID:         failingTeamPolicy.ID,
		Name:       failingTeamPolicy.Name,
		Resolution: failingTeamPolicy.Resolution,
		Critical:   new(true),
	})
	assert.Contains(t, hh.HostHealth.FailingPolicies, &fleet.HostHealthFailingPolicy{
		ID:         failingGlobalPolicy.ID,
		Name:       failingGlobalPolicy.Name,
		Resolution: failingGlobalPolicy.Resolution,
		Critical:   new(false),
	})
}

func (s *integrationEnterpriseTestSuite) TestListVulnerabilities() {
	t := s.T()
	var resp listVulnerabilitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)

	// Invalid Order Key
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusBadRequest, &resp, "order_key", "foo", "order_direction", "asc")

	// EE Only Order Key
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "order_key", "cvss_score", "order_direction", "asc")

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	s.Require().Empty(resp.Vulnerabilities)

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(strings.ReplaceAll(t.Name(), "/", "_") + "2"),
		OsqueryHostID:   new(strings.ReplaceAll(t.Name(), "/", "_") + "2"),
		UUID:            t.Name() + "2",
		Hostname:        t.Name() + "foo2.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "windows",
	})
	require.NoError(t, err)

	err = s.ds.UpdateHostOperatingSystem(context.Background(), host.ID, fleet.OperatingSystem{
		Name:     "Windows 11 Enterprise 22H2",
		Version:  "10.0.19042.1234",
		Platform: "windows",
	})
	require.NoError(t, err)
	allos, err := s.ds.ListOperatingSystems(context.Background())
	require.NoError(t, err)
	var os fleet.OperatingSystem
	for _, o := range allos {
		if o.ID > os.ID {
			os = o
		}
	}

	err = s.ds.UpdateOSVersions(context.Background())
	require.NoError(t, err)

	_, err = s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID:              os.ID,
		CVE:               "CVE-2021-1234",
		ResolvedInVersion: new("10.0.19043.2013"),
	}, fleet.MSRCSource)
	require.NoError(t, err)

	res, err := s.ds.UpdateHostSoftware(context.Background(), host.ID, []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	})
	require.NoError(t, err)
	sw := res.Inserted[0]

	_, err = s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: sw.ID,
		CVE:        "CVE-2021-1235",
	}, fleet.NVDSource)
	require.NoError(t, err)

	// insert CVEMeta
	mockTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	err = s.ds.InsertCVEMeta(context.Background(), []fleet.CVEMeta{
		{
			CVE:              "CVE-2021-1234",
			CVSSScore:        new(float64(7.5)),
			EPSSProbability:  new(float64(0.5)),
			CISAKnownExploit: new(true),
			Published:        &mockTime,
			Description:      "Test CVE 2021-1234",
		},
		{
			CVE:              "CVE-2021-1235",
			CVSSScore:        new(float64(5.4)),
			EPSSProbability:  new(float64(0.6)),
			CISAKnownExploit: new(false),
			Published:        new(mockTime),
			Description:      "Test CVE 2021-1235",
		},
	})
	require.NoError(t, err)

	err = s.ds.UpdateVulnerabilityHostCounts(context.Background(), 5)
	require.NoError(t, err)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	s.Require().Len(resp.Vulnerabilities, 2)
	require.Equal(t, uint(2), resp.Count)
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)
	require.NoError(t, resp.Err)

	expected := map[string]struct {
		fleet.CVE
		HostCount   uint
		DetailsLink string
		Source      fleet.VulnerabilitySource
	}{
		"CVE-2021-1234": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1234",
			CVE: fleet.CVE{
				CVE:              "CVE-2021-1234",
				CVSSScore:        new(new(7.5)),
				EPSSProbability:  new(new(0.5)),
				CISAKnownExploit: new(new(true)),
				CVEPublished:     new(new(mockTime)),
				Description:      new(new("Test CVE 2021-1234")),
			},
		},
		"CVE-2021-1235": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1235",
			CVE: fleet.CVE{
				CVE:              "CVE-2021-1235",
				CVSSScore:        new(new(5.4)),
				EPSSProbability:  new(new(0.6)),
				CISAKnownExploit: new(new(false)),
				CVEPublished:     new(new(mockTime)),
				Description:      new(new("Test CVE 2021-1235")),
			},
		},
	}

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Equal(t, expectedVuln.CVE.CVE, vuln.CVE.CVE)
	}

	// EE Exploit Filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "exploit", "true")
	require.Len(t, resp.Vulnerabilities, 1)
	require.Equal(t, "CVE-2021-1234", resp.Vulnerabilities[0].CVE.CVE)

	// Test Team Filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", "1")
	s.Require().Empty(resp.Vulnerabilities)

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID}))
	require.NoError(t, err)

	err = s.ds.UpdateVulnerabilityHostCounts(context.Background(), 5)
	require.NoError(t, err)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", fmt.Sprintf("%d", team.ID))
	require.Len(t, resp.Vulnerabilities, 2)
	require.Equal(t, uint(2), resp.Count)
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)
	require.NoError(t, resp.Err)

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Equal(t, expectedVuln.CVE.CVE, vuln.CVE.CVE)
	}

	var gResp getVulnerabilityResponse
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1234", nil, http.StatusOK, &gResp)
	require.NoError(t, gResp.Err)
	require.Equal(t, "CVE-2021-1234", gResp.Vulnerability.CVE.CVE)
	require.Equal(t, uint(1), gResp.Vulnerability.HostsCount)
	require.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2021-1234", gResp.Vulnerability.DetailsLink)
	require.Equal(t, new(new("Test CVE 2021-1234")), gResp.Vulnerability.Description)
	require.Equal(t, new(new(7.5)), gResp.Vulnerability.CVSSScore)
	require.Equal(t, new(new(true)), gResp.Vulnerability.CISAKnownExploit)
	require.Equal(t, new(new(0.5)), gResp.Vulnerability.EPSSProbability)
	require.Equal(t, new(new(mockTime)), gResp.Vulnerability.CVEPublished)
	require.Len(t, gResp.OSVersions, 1)
	require.Equal(t, "Windows 11 Enterprise 22H2 10.0.19042.1234", gResp.OSVersions[0].Name)
	require.Equal(t, "Windows 11 Enterprise 22H2", gResp.OSVersions[0].NameOnly)
	require.Equal(t, "windows", gResp.OSVersions[0].Platform)
	require.Equal(t, "10.0.19042.1234", gResp.OSVersions[0].Version)
	require.Equal(t, 1, gResp.OSVersions[0].HostsCount)
	require.Equal(t, "10.0.19043.2013", *gResp.OSVersions[0].ResolvedInVersion)

	// A matched CVE without cve_meta (e.g. published by MSRC before NVD) still resolves.
	_, err = s.ds.InsertOSVulnerability(t.Context(), fleet.OSVulnerability{
		OSID: os.ID,
		CVE:  "CVE-2021-1236",
	}, fleet.MSRCSource)
	require.NoError(t, err)
	err = s.ds.UpdateVulnerabilityHostCounts(t.Context(), 5)
	require.NoError(t, err)

	gResp = getVulnerabilityResponse{}
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1236", nil, http.StatusOK, &gResp)
	require.NoError(t, gResp.Err)
	require.Equal(t, fleet.CVE{
		CVE:         "CVE-2021-1236",
		DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1236",
	}, gResp.Vulnerability.CVE)
	require.Equal(t, uint(1), gResp.Vulnerability.HostsCount)
	require.Len(t, gResp.OSVersions, 1)
	require.Equal(t, "Windows 11 Enterprise 22H2 10.0.19042.1234", gResp.OSVersions[0].Name)

	// Known to Fleet via its match, but no hosts in "No team".
	s.Do("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1236", nil, http.StatusNoContent, "team_id", "0")
}

func (s *integrationEnterpriseTestSuite) TestOSVersions() {
	t := s.T()

	testOS := fleet.OperatingSystem{Name: "Windows 11 Pro", Version: "10.0.22621.2861", Arch: "x86_64", KernelVersion: "10.0.22621.2861", Platform: "windows"}

	hosts := s.createHosts(t)

	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)
	require.Len(t, resp.Hosts, len(hosts))

	// set operating system information on a host
	require.NoError(t, s.ds.UpdateHostOperatingSystem(context.Background(), hosts[0].ID, testOS))
	var osinfo struct {
		ID          uint `db:"id"`
		OSVersionID uint `db:"os_version_id"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &osinfo,
			`SELECT id, os_version_id FROM operating_systems WHERE name = ? AND version = ? AND arch = ? AND kernel_version = ? AND platform = ?`,
			testOS.Name, testOS.Version, testOS.Arch, testOS.KernelVersion, testOS.Platform)
	})
	require.Positive(t, osinfo.ID)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_name", testOS.Name, "os_version", testOS.Version)
	require.Len(t, resp.Hosts, 1)

	expected := resp.Hosts[0]
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_id", fmt.Sprintf("%d", osinfo.ID))
	require.Len(t, resp.Hosts, 1)
	require.Equal(t, expected, resp.Hosts[0])

	// generate aggregated stats
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))

	// insert OS Vulns
	_, err := s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID: osinfo.ID,
		CVE:  "CVE-2021-1234",
	}, fleet.MSRCSource)
	require.NoError(t, err)

	// insert CVE MEta
	vulnMeta := []fleet.CVEMeta{
		{
			CVE:              "CVE-2021-1234",
			CVSSScore:        new(float64(5.4)),
			EPSSProbability:  new(float64(0.5)),
			CISAKnownExploit: new(true),
			Published:        new(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
			Description:      "a long description of the cve",
		},
	}
	require.NoError(t, s.ds.InsertCVEMeta(context.Background(), vulnMeta))

	var osVersionsResp osVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
	require.Len(t, osVersionsResp.OSVersions, 1)
	require.Equal(t, 1, osVersionsResp.OSVersions[0].HostsCount)
	require.Equal(t, fmt.Sprintf("%s %s", testOS.Name, testOS.Version), osVersionsResp.OSVersions[0].Name)
	require.Equal(t, testOS.Name, osVersionsResp.OSVersions[0].NameOnly)
	require.Equal(t, testOS.Version, osVersionsResp.OSVersions[0].Version)
	require.Equal(t, testOS.Platform, osVersionsResp.OSVersions[0].Platform)
	require.Len(t, osVersionsResp.OSVersions[0].Vulnerabilities, 1)
	require.Equal(t, "CVE-2021-1234", osVersionsResp.OSVersions[0].Vulnerabilities[0].CVE)
	require.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2021-1234", osVersionsResp.OSVersions[0].Vulnerabilities[0].DetailsLink)
	require.InDelta(t, *vulnMeta[0].CVSSScore, **osVersionsResp.OSVersions[0].Vulnerabilities[0].CVSSScore, 0.001)
	require.InDelta(t, *vulnMeta[0].EPSSProbability, **osVersionsResp.OSVersions[0].Vulnerabilities[0].EPSSProbability, 0.001)
	require.Equal(t, *vulnMeta[0].CISAKnownExploit, **osVersionsResp.OSVersions[0].Vulnerabilities[0].CISAKnownExploit)
	require.Equal(t, *vulnMeta[0].Published, **osVersionsResp.OSVersions[0].Vulnerabilities[0].CVEPublished)
	require.Equal(t, vulnMeta[0].Description, **osVersionsResp.OSVersions[0].Vulnerabilities[0].Description)
	expectedOSVersion := osVersionsResp.OSVersions[0]

	var osVersionResp getOSVersionResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusOK, &osVersionResp)
	require.Equal(t, &expectedOSVersion, osVersionResp.OSVersion)

	// OS versions with invalid team
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusNotFound, &osVersionResp, "team_id",
		"99999",
	)

	// Create team and ask for the OS versions from the team (with no hosts) -- should get 404.
	tr := teamResponse{}
	s.DoJSON(
		"POST", "/api/latest/fleet/teams", createTeamRequest{
			Name: new("os_versions_team"),
		}, http.StatusOK, &tr,
	)
	osVersionResp = getOSVersionResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusNotFound, &osVersionResp, "team_id",
		fmt.Sprintf("%d", tr.Team.ID),
	)

	// return empty json if UpdateOSVersions cron hasn't run yet for new team
	team0, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "new team"})
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team0.ID, []uint{hosts[0].ID})))
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "team_id", fmt.Sprintf("%d", team0.ID))
	require.Empty(t, osVersionsResp.OSVersions)

	// return err if team_id is invalid
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusBadRequest, &osVersionsResp, "team_id", "invalid")

	// Create another team and a team user
	team1, err := s.ds.NewTeam(
		context.Background(), &fleet.Team{
			ID:          42,
			Name:        "team1-os_version",
			Description: "desc team1",
		},
	)
	require.NoError(t, err)
	// Create a new admin for team1.
	password := test.GoodPassword
	email := "admin-team1-os_version@example.com"
	u := &fleet.User{
		Name:       "admin team1",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team1,
				Role: fleet.RoleAdmin,
			},
		},
	}
	require.NoError(t, u.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)

	s.setTokenForTest(t, email, test.GoodPassword)

	// generate aggregated stats
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))
	// team1 user does not have access to team0 host
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
	assert.Empty(t, osVersionsResp.OSVersions)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusNotFound, &osVersionResp)

	// Move host from team0 to team1
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team1.ID, []uint{hosts[0].ID})))
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
	require.Len(t, osVersionsResp.OSVersions, 1)
	assert.Equal(t, expectedOSVersion, osVersionsResp.OSVersions[0])
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusOK, &osVersionResp)
	require.Equal(t, &expectedOSVersion, osVersionResp.OSVersion)

	// Team user is forbidden to access invalid team
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusForbidden, &osVersionResp, "team_id",
		"99999",
	)

	// team user doesn't have acess to "no team"
	osVersionsResp = osVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusForbidden, &osVersionsResp, "team_id", "0")
	require.Empty(t, osVersionsResp.OSVersions)

	// team_id=0 is supported and returns results for hosts in "no team"
	s.token = getTestAdminToken(t, s.server)
	// no hosts, the results are empty
	osVersionsResp = osVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "team_id", "0")
	require.Empty(t, osVersionsResp.OSVersions)
	osVersionsResp = osVersionsResponse{}
	// move the host to "no team" and update the stats
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(nil, []uint{hosts[0].ID}))) // nolint:nilaway // createHosts always returns at least one host
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "team_id", "0")
	require.Len(t, osVersionsResp.OSVersions, 1)
	osVersionResp = getOSVersionResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusOK, &osVersionResp, "team_id", "0")
	require.Equal(t, &expectedOSVersion, osVersionResp.OSVersion)
	require.Equal(t, 1, osVersionResp.OSVersion.HostsCount)
}

func (s *integrationEnterpriseTestSuite) TestHostDeviceMappingIDP() {
	t := s.T()
	ctx := context.Background()

	hosts := s.createHosts(t, "darwin")
	host := hosts[0]

	// Create a SCIM user for testing
	scimUser := &fleet.ScimUser{
		UserName: "test.user",
		Emails: []fleet.ScimUserEmail{
			{
				Email:   "scim.user@example.com",
				Primary: new(true),
			},
		},
	}

	createdUserID, err := s.ds.CreateScimUser(ctx, scimUser)
	require.NoError(t, err)
	defer func() { _, _ = s.ds.DeleteScimUser(ctx, createdUserID) }()

	// Test IDP device mapping with premium license and valid SCIM user
	var putResp putHostDeviceMappingResponse
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID),
		putHostDeviceMappingRequest{Email: "scim.user@example.com", Source: "idp"},
		http.StatusOK, &putResp)

	// Verify the IDP mapping was created and returned in device mappings
	require.Equal(t, "scim.user@example.com", putResp.DeviceMapping[0].Email)
	require.Equal(t, fleet.DeviceMappingMDMIdpAccounts, putResp.DeviceMapping[0].Source)

	// Verify the IDP mapping appears in the host's device mappings via getHostDeviceMapping
	var getResp listHostDeviceMappingResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID),
		nil, http.StatusOK, &getResp)
	require.Len(t, getResp.DeviceMapping, 1)
	require.Equal(t, "scim.user@example.com", getResp.DeviceMapping[0].Email)
	require.Equal(t, fleet.DeviceMappingMDMIdpAccounts, getResp.DeviceMapping[0].Source)

	// Test that IDP mapping appears correctly in host details via getHostEndpoint
	var hostResp getHostResponse
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+host.UUID, nil, http.StatusOK, &hostResp)

	// Find the IDP information in end_users field
	foundIdpInDetails := false
	for _, endUser := range hostResp.Host.EndUsers {
		// IDP users should have IdpUserName populated
		if endUser.IdpUserName == "test.user" {
			foundIdpInDetails = true
			// Verify other IDP fields if they exist
			assert.NotEmpty(t, endUser.IdpUserName, "IDP EndUser should have IdpUserName")
		}
	}
	assert.True(t, foundIdpInDetails, "Should find IDP user in host end_users")

	// Verify consistency between identifier and ID-based endpoints
	hostResp2 := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp2)

	// Find the same IDP information in the ID-based endpoint
	foundIdpInDetailsById := false
	for _, endUser := range hostResp2.Host.EndUsers {
		if endUser.IdpUserName == "test.user" {
			foundIdpInDetailsById = true
		}
	}
	assert.True(t, foundIdpInDetailsById, "Should find IDP user in ID-based host endpoint")

	// Test that IDP accepts any username, even if not a SCIM user - should replace existing
	// "scim.user@example.com" idp mapping
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID),
		putHostDeviceMappingRequest{Email: "any.username@example.com", Source: "idp"},
		http.StatusOK, &putResp)

	// Test that custom mappings still work alongside IDP
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID),
		putHostDeviceMappingRequest{Email: "custom.user@example.com", Source: "custom"},
		http.StatusOK, &putResp)

	// Verify current mappings: custom and latest IDP (replacement behavior)
	foundCustom := false
	foundCurrentIdp := false
	foundOldIdp := false
	for _, mapping := range putResp.DeviceMapping {
		if mapping.Email == "custom.user@example.com" && mapping.Source == fleet.DeviceMappingCustomReplacement {
			foundCustom = true
		}
		if mapping.Email == "any.username@example.com" && mapping.Source == fleet.DeviceMappingMDMIdpAccounts {
			foundCurrentIdp = true
		}
		if mapping.Email == "scim.user@example.com" && mapping.Source == fleet.DeviceMappingMDMIdpAccounts {
			foundOldIdp = true
		}
	}
	assert.True(t, foundCustom, "Should find custom mapping in device_mapping")
	assert.True(t, foundCurrentIdp, "Should find current IDP mapping (any.username) in device_mapping")
	assert.False(t, foundOldIdp, "Should NOT find old IDP mapping (scim.user) - replacement behavior")

	// Verify IDP appears in idp_username and custom appears in other_emails
	var finalHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), nil, http.StatusOK, &finalHostResp)

	require.Len(t, finalHostResp.Host.EndUsers, 1, "Should have exactly one end user")
	endUser := finalHostResp.Host.EndUsers[0]

	// Current IDP user (non-SCIM) should appear in idp_username field
	assert.Equal(t, "any.username@example.com", endUser.IdpUserName, "Current IDP user should be in idp_username")
	assert.Empty(t, endUser.IdpFullName, "Non-SCIM user should not have full name")

	// Verify custom mapping appears in other_emails
	foundCustomInOtherEmails := false
	foundIdpInOtherEmails := false
	for _, otherEmail := range endUser.OtherEmails {
		if otherEmail.Email == "custom.user@example.com" {
			foundCustomInOtherEmails = true
		}
		// Verify IDP emails do NOT appear in other_emails
		if otherEmail.Email == "any.username@example.com" {
			foundIdpInOtherEmails = true
		}
	}
	assert.True(t, foundCustomInOtherEmails, "Custom mapping should appear in other_emails")
	assert.False(t, foundIdpInOtherEmails, "IDP mappings should NOT appear in other_emails")

	// Test with non-SCIM IDP user only (should replace previous IDP mapping)
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID),
		putHostDeviceMappingRequest{Email: "nonscim@example.com", Source: "idp"},
		http.StatusOK, &putResp)

	// Verify device mapping API shows only the new IDP mapping (replacement behavior)
	foundNonScimIdp := false
	foundPreviousIdp := false
	for _, mapping := range putResp.DeviceMapping {
		if mapping.Email == "nonscim@example.com" && mapping.Source == fleet.DeviceMappingMDMIdpAccounts {
			foundNonScimIdp = true
		}
		if (mapping.Email == "scim.user@example.com" || mapping.Email == "any.username@example.com") && mapping.Source == fleet.DeviceMappingMDMIdpAccounts {
			foundPreviousIdp = true
		}
	}
	assert.True(t, foundNonScimIdp, "Should find new IDP mapping in device_mapping")
	assert.False(t, foundPreviousIdp, "Should NOT find old IDP mappings (replacement behavior)")

	var nonScimHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), nil, http.StatusOK, &nonScimHostResp)

	require.Len(t, nonScimHostResp.Host.EndUsers, 1, "Should have exactly one end user")
	endUser = nonScimHostResp.Host.EndUsers[0]

	// Non-SCIM IDP user should appear in idp_username field (no SCIM data)
	assert.Equal(t, "nonscim@example.com", endUser.IdpUserName, "Non-SCIM IDP user should be in idp_username")
	assert.Empty(t, endUser.IdpFullName, "Non-SCIM user should not have full name")

	// Verify IDP email doesn't appear in other_emails
	foundNonScimInOtherEmails := false
	for _, otherEmail := range endUser.OtherEmails {
		if otherEmail.Email == "nonscim@example.com" {
			foundNonScimInOtherEmails = true
		}
	}
	assert.False(t, foundNonScimInOtherEmails, "Non-SCIM IDP should NOT appear in other_emails")

	// test DELETE
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping/idp", host.ID), deleteHostIDPRequest{}, http.StatusNoContent)

	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+host.UUID, nil, http.StatusOK, &hostResp)
	require.Len(t, hostResp.Host.EndUsers, 1, "Should have exactly one end user")
	endUser = hostResp.Host.EndUsers[0]
	// Non-SCIM IDP user should now be replaced by user's username
	assert.Equal(t, "test.user", endUser.IdpUserName)

	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp2)
	require.Len(t, hostResp2.Host.EndUsers, 1, "Should have exactly one end user")
	endUser = hostResp.Host.EndUsers[0]

	// Non-SCIM IDP user should now be replaced by user's username
	assert.Equal(t, "test.user", endUser.IdpUserName)
}

// TestTeamAdminCanReadHostPastActivities is a regression test for issue #46009:
// after the OPA-tag fix, team-scoped users were still getting 403 on
// GET /api/_version_/fleet/hosts/{id}/activities for any host with a
// user-initiated past activity (locked_host being the common reproducer).
// The cause was Service.UsersByIDs gating user enrichment behind an authz
// check on an empty *fleet.User, which no team role can satisfy. The activity
// ACL adapter now bypasses that check by going directly to the datastore.
func (s *integrationEnterpriseTestSuite) TestTeamAdminCanReadHostPastActivities() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name:        t.Name(),
		Description: t.Name(),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, s.ds.DeleteTeam(context.Background(), team.ID))
	})

	teamAdminEmail := t.Name() + "_admin@example.com"
	teamAdmin := &fleet.User{
		Name:  teamAdminEmail,
		Email: teamAdminEmail,
		Teams: []fleet.UserTeam{
			{Team: *team, Role: fleet.RoleAdmin},
		},
	}
	require.NoError(t, teamAdmin.SetPassword(test.GoodPassword, 10, 10))
	teamAdmin, err = s.ds.NewUser(ctx, teamAdmin)
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   new(t.Name() + "_host"),
		NodeKey:         new(t.Name() + "_host"),
		UUID:            t.Name() + "_host",
		Hostname:        t.Name() + "_host.local",
		Platform:        "darwin",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	// Insert a host-linked past activity attributed to the team admin. This
	// mirrors what happens when the team admin locks one of their hosts —
	// the activity row has user_id set, which forces enrichWithUserData to
	// call the user lookup. Before the fix, that lookup went through
	// Service.UsersByIDs and 403'd the team admin viewing the activity feed.
	activitySvc := mysqltest.NewTestActivityService(t, s.ds)
	apiUser := &activity_api.User{ID: teamAdmin.ID, Name: teamAdmin.Name, Email: teamAdmin.Email}
	require.NoError(t, activitySvc.NewActivity(ctx, apiUser, fleet.ActivityTypeLockedHost{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
	}))

	t.Cleanup(func() { s.token = s.getTestAdminToken() })
	s.token = s.getTestToken(teamAdmin.Email, test.GoodPassword)

	var listResp listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 1)
	require.Equal(t, fleet.ActivityTypeLockedHost{}.ActivityName(), listResp.Activities[0].Type)
	require.NotNil(t, listResp.Activities[0].ActorEmail)
	require.Equal(t, teamAdmin.Email, *listResp.Activities[0].ActorEmail)
	require.NotNil(t, listResp.Activities[0].ActorFullName)
	require.Equal(t, teamAdmin.Name, *listResp.Activities[0].ActorFullName)
}

func (s *integrationEnterpriseTestSuite) TestHostGoogleDeviceID() {
	t := s.T()
	hosts := s.createHosts(t, "ios", "android", "darwin")
	iosHost, androidHost, macHost := hosts[0], hosts[1], hosts[2]

	idPath := func(h *fleet.Host) string { return fmt.Sprintf("/api/latest/fleet/hosts/%d/google_device_id", h.ID) }
	hostJSON := func(path string) map[string]any {
		res := s.Do("GET", path, nil, http.StatusOK)
		var body struct {
			Host map[string]any `json:"host"`
		}
		require.NoError(t, json.NewDecoder(res.Body).Decode(&body))
		return body.Host
	}
	storedID := func(h *fleet.Host) any {
		return hostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID))["google_device_id"]
	}
	requireErrorReason := func(res *http.Response, reason string) {
		var errBody struct {
			Errors []map[string]string `json:"errors"`
		}
		require.NoError(t, json.NewDecoder(res.Body).Decode(&errBody))
		require.Equal(t, []map[string]string{{"name": "google_device_id", "reason": reason}}, errBody.Errors)
	}
	activityDetails := func(h *fleet.Host, id string) string {
		return fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "google_device_id": %q}`, h.ID, h.DisplayName(), id)
	}

	// mobile hosts report "" until set, other platforms omit the field
	got := hostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", iosHost.ID))
	require.Contains(t, got, "google_device_id")
	require.Empty(t, got["google_device_id"])
	require.NotContains(t, hostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", macHost.ID)), "google_device_id")

	var putResp putHostGoogleDeviceIDResponse
	s.DoJSON("PUT", idPath(iosHost), map[string]any{"google_device_id": "device-a"}, http.StatusOK, &putResp)
	require.Equal(t, iosHost.ID, putResp.HostID)
	require.Equal(t, "device-a", putResp.GoogleDeviceID)
	activityID := s.lastActivityMatches("set_host_google_device_id", activityDetails(iosHost, "device-a"), 0)
	s.lastHostActivityMatches(iosHost.ID, "set_host_google_device_id", activityDetails(iosHost, "device-a"), activityID)

	// repeating the same value is idempotent and doesn't log another activity
	putResp = putHostGoogleDeviceIDResponse{}
	s.DoJSON("PUT", idPath(iosHost), map[string]any{"google_device_id": "device-a"}, http.StatusOK, &putResp)
	require.Equal(t, "device-a", putResp.GoogleDeviceID)
	s.lastActivityMatches("", "", activityID)

	require.Equal(t, "device-a", storedID(iosHost))
	require.Equal(t, "device-a", hostJSON("/api/latest/fleet/hosts/identifier/" + iosHost.UUID)["google_device_id"])

	// a different value is refused
	res := s.Do("PUT", idPath(iosHost), map[string]any{"google_device_id": "device-b"}, http.StatusConflict)
	requireErrorReason(res, "This host already has a different Google device ID.")
	require.Equal(t, "device-a", storedID(iosHost))

	// an ID can only be bound to one host
	res = s.Do("PUT", idPath(androidHost), map[string]any{"google_device_id": "device-a"}, http.StatusConflict)
	requireErrorReason(res, "This Google device ID is already used by another host.")
	require.Empty(t, storedID(androidHost))
	s.DoJSON("PUT", idPath(androidHost), map[string]any{"google_device_id": "device-c"}, http.StatusOK, &putResp)
	require.Equal(t, "device-c", hostJSON("/api/latest/fleet/hosts/identifier/" + androidHost.UUID)["google_device_id"])

	// only DELETE removes the ID, and it logs an activity
	res = s.Do("PUT", idPath(iosHost), map[string]any{"google_device_id": ""}, http.StatusUnprocessableEntity)
	name, _ := extractServerErrorNameReason(res.Body)
	require.Equal(t, "google_device_id", name)
	require.Equal(t, "device-a", storedID(iosHost))
	s.Do("DELETE", idPath(iosHost), nil, http.StatusNoContent)
	require.Empty(t, storedID(iosHost))
	activityID = s.lastActivityMatches("deleted_host_google_device_id", activityDetails(iosHost, "device-a"), 0)
	s.lastHostActivityMatches(iosHost.ID, "deleted_host_google_device_id", activityDetails(iosHost, "device-a"), activityID)
	// deleting again is a no-op without an activity
	s.Do("DELETE", idPath(iosHost), nil, http.StatusNoContent)
	s.lastActivityMatches("", "", activityID)
	s.DoJSON("PUT", idPath(iosHost), map[string]any{"google_device_id": "device-b"}, http.StatusOK, &putResp)
	require.Equal(t, "device-b", putResp.GoogleDeviceID)
	s.lastActivityMatches("set_host_google_device_id", activityDetails(iosHost, "device-b"), 0)

	// only iOS, iPadOS and Android hosts
	res = s.Do("PUT", idPath(macHost), map[string]any{"google_device_id": "device-d"}, http.StatusUnprocessableEntity)
	name, _ = extractServerErrorNameReason(res.Body)
	require.Equal(t, "google_device_id", name)

	res = s.Do("PUT", idPath(iosHost), map[string]any{}, http.StatusUnprocessableEntity)
	name, _ = extractServerErrorNameReason(res.Body)
	require.Equal(t, "google_device_id", name)
	require.Equal(t, "device-b", storedID(iosHost))

	// surrounding whitespace would store an ID that conflicts with the clean one later
	for _, bad := range []string{" device-b", "device-b\n", "   "} {
		res = s.Do("PUT", idPath(iosHost), map[string]any{"google_device_id": bad}, http.StatusUnprocessableEntity)
		name, _ = extractServerErrorNameReason(res.Body)
		require.Equal(t, "google_device_id", name)
	}
	require.Equal(t, "device-b", storedID(iosHost))

	// the limit is 255 characters, not bytes
	s.Do("DELETE", idPath(iosHost), nil, http.StatusNoContent)
	maxLenID := strings.Repeat("é", 255)
	s.DoJSON("PUT", idPath(iosHost), map[string]any{"google_device_id": maxLenID}, http.StatusOK, &putResp)
	require.Equal(t, maxLenID, storedID(iosHost))
	s.Do("DELETE", idPath(iosHost), nil, http.StatusNoContent)
	res = s.Do("PUT", idPath(iosHost), map[string]any{"google_device_id": strings.Repeat("a", 256)}, http.StatusUnprocessableEntity)
	name, _ = extractServerErrorNameReason(res.Body)
	require.Equal(t, "google_device_id", name)
	s.DoJSON("PUT", idPath(iosHost), map[string]any{"google_device_id": "device-b"}, http.StatusOK, &putResp)

	s.Do("PUT", "/api/latest/fleet/hosts/999999/google_device_id", map[string]any{"google_device_id": "device-d"}, http.StatusNotFound)
	s.Do("DELETE", "/api/latest/fleet/hosts/999999/google_device_id", nil, http.StatusNotFound)

	// an observer can read the host but not set or remove the ID
	defer func() { s.token = s.getTestAdminToken() }()
	observer := &fleet.User{Name: t.Name() + " observer", Email: "google-device-id-observer@example.com", GlobalRole: new(fleet.RoleObserver)}
	require.NoError(t, observer.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(t.Context(), observer)
	require.NoError(t, err)
	s.token = s.getTestToken(observer.Email, test.GoodPassword)
	s.Do("PUT", idPath(androidHost), map[string]any{"google_device_id": "device-c"}, http.StatusForbidden)
	s.Do("DELETE", idPath(androidHost), nil, http.StatusForbidden)

	// a team maintainer can only set or remove the ID on hosts of their own fleet
	s.token = s.getTestAdminToken()
	team, err := s.ds.NewTeam(t.Context(), &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.ds.DeleteTeam(context.Background(), team.ID)) })
	require.NoError(t, s.ds.AddHostsToTeam(t.Context(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{androidHost.ID})))
	teamMaintainer := &fleet.User{
		Name:  t.Name() + " team maintainer",
		Email: "google-device-id-team-maintainer@example.com",
		Teams: []fleet.UserTeam{{Team: *team, Role: fleet.RoleMaintainer}},
	}
	require.NoError(t, teamMaintainer.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(t.Context(), teamMaintainer)
	require.NoError(t, err)
	s.token = s.getTestToken(teamMaintainer.Email, test.GoodPassword)
	s.DoJSON("PUT", idPath(androidHost), map[string]any{"google_device_id": "device-c"}, http.StatusOK, &putResp)
	s.Do("PUT", idPath(iosHost), map[string]any{"google_device_id": "device-b"}, http.StatusForbidden)
	s.Do("DELETE", idPath(iosHost), nil, http.StatusForbidden)
	s.Do("DELETE", idPath(androidHost), nil, http.StatusNoContent)
	s.DoJSON("PUT", idPath(androidHost), map[string]any{"google_device_id": "device-c"}, http.StatusOK, &putResp)

	// the registration service's API-only user can set the ID but not remove it
	s.token = s.getTestAdminToken()
	var apiUserResp struct {
		Token string `json:"token"`
	}
	s.DoJSON("POST", "/api/latest/fleet/users/api_only", map[string]any{
		"name":        t.Name() + " registration service",
		"global_role": fleet.RoleMaintainer,
		"api_endpoints": []map[string]any{
			{"method": "GET", "path": "/api/v1/fleet/hosts/identifier/:identifier"},
			{"method": "PUT", "path": "/api/v1/fleet/hosts/:id/google_device_id"},
		},
	}, http.StatusOK, &apiUserResp)
	require.NotEmpty(t, apiUserResp.Token)
	s.token = apiUserResp.Token
	hostJSON("/api/latest/fleet/hosts/identifier/" + androidHost.UUID)
	s.DoJSON("PUT", idPath(androidHost), map[string]any{"google_device_id": "device-c"}, http.StatusOK, &putResp)
	s.Do("DELETE", idPath(androidHost), nil, http.StatusForbidden)
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", androidHost.ID), nil, http.StatusForbidden)
	s.token = s.getTestAdminToken()
	require.Equal(t, "device-c", storedID(androidHost))
}
