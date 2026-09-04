package service

// Host reporting and read-only host detail tests for the core (no-license) suite.
//
// Belongs here: reads of a single host's reported state (macadmins/Munki, batteries,
// disk encryption, maintenance window, iOS/iPadOS vitals, last-opened-at and
// software-updated-at timestamps), the hosts report CSV endpoint, host health, a
// host's past and upcoming activities, aggregated OS versions, and a host's report
// (scheduled query) results.
//
// Does not belong here: host CRUD, listing, filtering, or fleet assignment
// (integration_core_hosts_test.go), and anything creating or configuring the
// reports themselves rather than reading their results.

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestGetMacadminsData() {
	t := s.T()

	ctx := context.Background()

	hostAll, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   new("1"),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, hostAll)

	hostNothing, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "2"),
		UUID:            t.Name() + "2",
		Hostname:        t.Name() + "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		OsqueryHostID:   new("2"),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, hostNothing)

	hostOnlyMunki, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "3"),
		UUID:            t.Name() + "3",
		Hostname:        t.Name() + "foo.local3",
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "30-65-EC-6F-C4-5F",
		OsqueryHostID:   new("3"),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, hostOnlyMunki)

	hostOnlyMDM, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "4"),
		UUID:            t.Name() + "4",
		Hostname:        t.Name() + "foo.local4",
		PrimaryIP:       "192.168.1.4",
		PrimaryMac:      "30-65-EC-6F-C4-5A",
		OsqueryHostID:   new("4"),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, hostOnlyMDM)

	hostMDMNoID, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "5"),
		UUID:            t.Name() + "5",
		Hostname:        t.Name() + "foo.local5",
		PrimaryIP:       "192.168.1.5",
		PrimaryMac:      "30-65-EC-6F-D5-5A",
		OsqueryHostID:   new("5"),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, hostMDMNoID)

	// insert a host_mdm row for hostMDMNoID without any mdm_id
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_server) VALUES (?, ?, ?, ?, ?)`,
			hostMDMNoID.ID, true, "https://simplemdm.com", true, false)
		return err
	})

	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostAll.ID, false, true, "url", false, "", "", false))
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(ctx, hostAll.ID, "1.3.0", []string{"error1"}, []string{"warning1"}))

	macadminsData := macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostAll.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	assert.Equal(t, "url", macadminsData.Macadmins.MDM.ServerURL)
	assert.Equal(t, "On (manual)", macadminsData.Macadmins.MDM.EnrollmentStatus)
	assert.Nil(t, macadminsData.Macadmins.MDM.Name)
	require.NotNil(t, macadminsData.Macadmins.MDM.ID)
	assert.NotZero(t, *macadminsData.Macadmins.MDM.ID)
	assert.Equal(t, "1.3.0", macadminsData.Macadmins.Munki.Version)

	require.Len(t, macadminsData.Macadmins.MunkiIssues, 2)
	sort.Slice(macadminsData.Macadmins.MunkiIssues, func(i, j int) bool {
		l, r := macadminsData.Macadmins.MunkiIssues[i], macadminsData.Macadmins.MunkiIssues[j]
		return l.Name < r.Name
	})
	assert.NotZero(t, macadminsData.Macadmins.MunkiIssues[0].MunkiIssueID)
	assert.False(t, macadminsData.Macadmins.MunkiIssues[0].HostIssueCreatedAt.IsZero())
	assert.Equal(t, "error1", macadminsData.Macadmins.MunkiIssues[0].Name)
	assert.Equal(t, "error", macadminsData.Macadmins.MunkiIssues[0].IssueType)
	assert.Equal(t, "warning1", macadminsData.Macadmins.MunkiIssues[1].Name)
	assert.NotZero(t, macadminsData.Macadmins.MunkiIssues[1].MunkiIssueID)
	assert.False(t, macadminsData.Macadmins.MunkiIssues[1].HostIssueCreatedAt.IsZero())
	assert.Equal(t, "warning", macadminsData.Macadmins.MunkiIssues[1].IssueType)

	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostAll.ID, false, true, "https://simplemdm.com", true, fleet.WellKnownMDMSimpleMDM, "", false))
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(ctx, hostAll.ID, "1.5.0", []string{"error1"}, nil))

	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostAll.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	assert.Equal(t, "https://simplemdm.com", macadminsData.Macadmins.MDM.ServerURL)
	assert.Equal(t, "On (automatic)", macadminsData.Macadmins.MDM.EnrollmentStatus)
	require.NotNil(t, macadminsData.Macadmins.MDM.Name)
	assert.Equal(t, fleet.WellKnownMDMSimpleMDM, *macadminsData.Macadmins.MDM.Name)
	require.NotNil(t, macadminsData.Macadmins.MDM.ID)
	assert.NotZero(t, *macadminsData.Macadmins.MDM.ID)
	assert.Equal(t, "1.5.0", macadminsData.Macadmins.Munki.Version)
	require.Len(t, macadminsData.Macadmins.MunkiIssues, 1)
	assert.Equal(t, "error1", macadminsData.Macadmins.MunkiIssues[0].Name)

	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostAll.ID, false, false, "url2", false, "", "", false))

	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostAll.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	assert.Equal(t, "Off", macadminsData.Macadmins.MDM.EnrollmentStatus)
	assert.Nil(t, macadminsData.Macadmins.MDM.Name)
	require.NotNil(t, macadminsData.Macadmins.MDM.ID)
	assert.NotZero(t, *macadminsData.Macadmins.MDM.ID)
	assert.Len(t, macadminsData.Macadmins.MunkiIssues, 1)

	// nothing returns null
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostNothing.ID), nil, http.StatusOK, &macadminsData)
	require.Nil(t, macadminsData.Macadmins)

	// only munki info returns null on mdm
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(ctx, hostOnlyMunki.ID, "3.2.0", nil, []string{"warning1"}))
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostOnlyMunki.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	require.Nil(t, macadminsData.Macadmins.MDM)
	require.NotNil(t, macadminsData.Macadmins.Munki)
	assert.Equal(t, "3.2.0", macadminsData.Macadmins.Munki.Version)
	require.Len(t, macadminsData.Macadmins.MunkiIssues, 1)
	assert.Equal(t, "warning1", macadminsData.Macadmins.MunkiIssues[0].Name)

	// only mdm returns null on munki info
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostOnlyMDM.ID, false, true, "https://kandji.io", true, fleet.WellKnownMDMIru, "", false))
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostOnlyMDM.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	require.NotNil(t, macadminsData.Macadmins.MDM)
	require.NotNil(t, macadminsData.Macadmins.MDM.Name)
	assert.Equal(t, fleet.WellKnownMDMIru, *macadminsData.Macadmins.MDM.Name)
	require.NotNil(t, macadminsData.Macadmins.MDM.ID)
	assert.NotZero(t, *macadminsData.Macadmins.MDM.ID)
	require.Nil(t, macadminsData.Macadmins.Munki)
	require.Empty(t, macadminsData.Macadmins.MunkiIssues)
	assert.Equal(t, "https://kandji.io", macadminsData.Macadmins.MDM.ServerURL)
	assert.Equal(t, "On (automatic)", macadminsData.Macadmins.MDM.EnrollmentStatus)

	// host without mdm_id still works, returns nil id and unknown name
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostMDMNoID.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	require.NotNil(t, macadminsData.Macadmins.MDM)
	assert.Nil(t, macadminsData.Macadmins.MDM.Name)
	assert.Nil(t, macadminsData.Macadmins.MDM.ID)
	require.Nil(t, macadminsData.Macadmins.Munki)
	assert.Equal(t, "On (automatic)", macadminsData.Macadmins.MDM.EnrollmentStatus)

	// generate aggregated data
	require.NoError(t, s.ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	agg := getAggregatedMacadminsDataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/macadmins", nil, http.StatusOK, &agg)
	require.NotNil(t, agg.Macadmins)
	assert.NotZero(t, agg.Macadmins.CountsUpdatedAt)
	assert.Len(t, agg.Macadmins.MunkiVersions, 2)
	assert.ElementsMatch(t, agg.Macadmins.MunkiVersions, []fleet.AggregatedMunkiVersion{
		{
			HostMunkiInfo: fleet.HostMunkiInfo{Version: "1.5.0"},
			HostsCount:    1,
		},
		{
			HostMunkiInfo: fleet.HostMunkiInfo{Version: "3.2.0"},
			HostsCount:    1,
		},
	})
	require.Len(t, agg.Macadmins.MunkiIssues, 2)
	// ignore ids
	agg.Macadmins.MunkiIssues[0].ID = 0
	agg.Macadmins.MunkiIssues[1].ID = 0
	assert.ElementsMatch(t, agg.Macadmins.MunkiIssues, []fleet.AggregatedMunkiIssue{
		{
			MunkiIssue: fleet.MunkiIssue{
				Name:      "error1",
				IssueType: "error",
			},
			HostsCount: 1,
		},
		{
			MunkiIssue: fleet.MunkiIssue{
				Name:      "warning1",
				IssueType: "warning",
			},
			HostsCount: 1,
		},
	})
	assert.Equal(t, 0, agg.Macadmins.MDMStatus.EnrolledManualHostsCount)
	assert.Equal(t, 2, agg.Macadmins.MDMStatus.EnrolledAutomatedHostsCount)
	assert.Equal(t, 1, agg.Macadmins.MDMStatus.UnenrolledHostsCount)
	assert.Equal(t, 3, agg.Macadmins.MDMStatus.HostsCount)
	require.Len(t, agg.Macadmins.MDMSolutions, 2)
	for _, sol := range agg.Macadmins.MDMSolutions {
		switch sol.ServerURL {
		case "url2":
			assert.Equal(t, fleet.UnknownMDMName, sol.Name)
			assert.Equal(t, 1, sol.HostsCount)
		case "https://kandji.io":
			assert.Equal(t, fleet.WellKnownMDMIru, sol.Name)
			assert.Equal(t, 1, sol.HostsCount)
		default:
			require.Fail(t, fmt.Sprintf("unknown MDM server URL: %s", sol.ServerURL))
		}
	}

	// Delete Munki from host -- no munki, but issues stick.
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(ctx, hostAll.ID, "", []string{"error1", "error3"}, []string{}))
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostAll.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	assert.Equal(t, "Off", macadminsData.Macadmins.MDM.EnrollmentStatus)
	assert.Nil(t, macadminsData.Macadmins.MDM.Name)
	require.NotNil(t, macadminsData.Macadmins.MDM.ID)
	assert.NotZero(t, *macadminsData.Macadmins.MDM.ID)
	require.Nil(t, macadminsData.Macadmins.Munki)
	require.Len(t, macadminsData.Macadmins.MunkiIssues, 2)

	// Bring Munki back, with same issues.
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(ctx, hostAll.ID, "6.4", []string{"error1", "error3"}, []string{}))
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostAll.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	assert.Equal(t, "Off", macadminsData.Macadmins.MDM.EnrollmentStatus)
	assert.Nil(t, macadminsData.Macadmins.MDM.Name)
	require.NotNil(t, macadminsData.Macadmins.MDM.ID)
	assert.NotZero(t, *macadminsData.Macadmins.MDM.ID)
	assert.NotNil(t, macadminsData.Macadmins.Munki)
	require.NotNil(t, macadminsData.Macadmins.Munki.Version, "6.4")
	require.Len(t, macadminsData.Macadmins.MunkiIssues, 2)

	// Delete Munki from host without MDM -- nothing is returned
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(ctx, hostOnlyMunki.ID, "", nil, []string{}))
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostOnlyMunki.ID), nil, http.StatusOK, &macadminsData)
	require.Nil(t, macadminsData.Macadmins)

	// TODO: ideally we'd pull this out into its own function that specifically tests
	// the mdm summary endpoint. We can add additional tests for testing the platform
	// and team_id query params for this endpoint.
	mdmAgg := getHostMDMSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/summary/mdm", nil, http.StatusOK, &mdmAgg)
	assert.NotZero(t, mdmAgg.AggregatedMDMData.CountsUpdatedAt)

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team1" + t.Name(),
		Description: "desc team1",
	})
	require.NoError(t, err)

	agg = getAggregatedMacadminsDataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/macadmins", nil, http.StatusOK, &agg, "team_id", fmt.Sprint(team.ID))
	require.NotNil(t, agg.Macadmins)
	require.Empty(t, agg.Macadmins.MunkiVersions)
	require.Empty(t, agg.Macadmins.MunkiIssues)
	require.Empty(t, agg.Macadmins.MDMStatus)
	require.Empty(t, agg.Macadmins.MDMSolutions)

	agg = getAggregatedMacadminsDataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/macadmins", nil, http.StatusNotFound, &agg, "team_id", "9999999")

	// Hardcode response type because we are using a custom json marshaling so
	// using getHostMDMResponse fails with "JSON unmarshaling is not supported for HostMDM".
	type jsonMDM struct {
		EnrollmentStatus string `json:"enrollment_status"`
		ServerURL        string `json:"server_url"`
		Name             string `json:"name,omitempty"`
		ID               *uint  `json:"id,omitempty"`
	}
	type getHostMDMResponseTest struct {
		HostMDM *jsonMDM
		Err     error `json:"error,omitempty"`
	}
	ghr := getHostMDMResponseTest{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", hostNothing.ID), nil, http.StatusOK, &ghr)
	require.Nil(t, ghr.HostMDM)

	ghr = getHostMDMResponseTest{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", 999999), nil, http.StatusNotFound, &ghr)
	require.Nil(t, ghr.HostMDM)
}

func (s *integrationTestSuite) TestGetHostLastOpenedAt() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	today := time.Now()
	yesterday := today.Add(-24 * time.Hour)
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps", LastOpenedAt: &today},
		{Name: "baz", Version: "0.0.4", Source: "apps", LastOpenedAt: &yesterday},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Len(t, getHostResp.Host.Software, len(software))

	sort.Slice(getHostResp.Host.Software, func(l, r int) bool {
		lsw, rsw := getHostResp.Host.Software[l], getHostResp.Host.Software[r]
		return lsw.Name < rsw.Name
	})
	// bar, baz, foo, in this order
	wantTs := []time.Time{today, yesterday, {}}
	for i, want := range wantTs {
		sw := getHostResp.Host.Software[i]
		if want.IsZero() {
			require.Nil(t, sw.LastOpenedAt)
		} else {
			require.WithinDuration(t, want, *sw.LastOpenedAt, time.Second)
		}
	}

	// listing hosts does not return the last opened at timestamp, only the GET /hosts/{id} endpoint
	var listHostsResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsResp)

	var hostSeen bool
	for _, h := range listHostsResp.Hosts {
		if h.ID == host.ID {
			hostSeen = true
		}
		for _, sw := range h.Software {
			require.Nil(t, sw.LastOpenedAt)
		}
	}
	require.True(t, hostSeen)
}

func (s *integrationTestSuite) TestGetHostSoftwareUpdatedAt() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Empty(t, getHostResp.Host.Software)
	require.Equal(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)

	// Sleep for 1 second to have software_updated_at be bigger than created_at.
	time.Sleep(1 * time.Second)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Len(t, getHostResp.Host.Software, len(software))
	require.Greater(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp, "exclude_software", "true")
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.NotNil(t, getHostResp.Host.Software)
	require.Greater(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", *host.NodeKey), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Len(t, getHostResp.Host.Software, len(software))
	require.Greater(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", *host.NodeKey), nil, http.StatusOK, &getHostResp, "exclude_software", "true")
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Empty(t, getHostResp.Host.Software)
	require.Greater(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)
}

func (s *integrationTestSuite) TestHostsReportDownload() {
	t := s.T()
	ctx := context.Background()

	// create 3 hosts (deb, rhel, linux)
	hosts := s.createHosts(t)
	err := s.ds.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{
		{Name: t.Name(), LabelMembershipType: fleet.LabelMembershipTypeManual, Query: "select 1", Hosts: []string{hosts[2].Hostname}},
	})
	require.NoError(t, err)
	lids, err := s.ds.LabelIDsByName(context.Background(), []string{t.Name()}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, lids, 1)
	customLabelID := lids[t.Name()]

	// create a policy and make host[1] fail that policy
	pol, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: t.Name(), Query: "SELECT 1"})
	require.NoError(t, err)
	_, err = s.ds.RecordPolicyQueryExecutions(ctx, hosts[1], map[uint]*bool{pol.ID: new(false)}, time.Now(), false, nil) // nolint:nilaway // cannot be nil due to previous require
	require.NoError(t, err)

	// create some device mappings for host[2]
	err = s.ds.ReplaceHostDeviceMapping(ctx, hosts[2].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[2].ID, Email: "a@b.c", Source: "google_chrome_profiles"},
		{HostID: hosts[2].ID, Email: "b@b.c", Source: "google_chrome_profiles"},
	}, "google_chrome_profiles")
	require.NoError(t, err)

	// set disk space information for hosts [0] and [1]
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[0].ID, 1.0, 2.0, 500.0, new(600.0)))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[1].ID, 3.0, 4.0, 1000.0, new(1200.0)))

	// create software for host [0]
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, hosts[0].ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, hosts[0], false))

	var fooV1ID, fooTitleID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		err := sqlx.GetContext(context.Background(), q, &fooV1ID,
			`SELECT id FROM software WHERE name = ? AND source = ? AND version = ?`, "foo", "chrome_extensions", "0.0.1")
		if err != nil {
			return err
		}
		err = sqlx.GetContext(context.Background(), q, &fooTitleID,
			`SELECT id FROM software_titles WHERE name = ? AND source = ?`, "foo", "chrome_extensions")
		if err != nil {
			return err
		}
		return nil
	})

	res := s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusUnsupportedMediaType, "format", "gzip")
	var errs validationErrResp
	require.NoError(t, json.NewDecoder(res.Body).Decode(&errs))
	res.Body.Close()
	require.Len(t, errs.Errors, 1)
	assert.Equal(t, "format", errs.Errors[0].Name)

	// valid format, no column specified so all columns returned
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv")
	rows, err := csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, len(hosts)+1) // all hosts + header row
	assert.Len(t, rows[0], 58)         // total number of cols
	// Validate that both team_id and fleet_id columns are present.
	assert.Contains(t, rows[0], "team_id")
	assert.Contains(t, rows[0], "fleet_id")
	assert.Contains(t, rows[0], "team_name")
	assert.Contains(t, rows[0], "fleet_name")

	// hardware_marketing_name is emitted right after hardware_model, shifting
	// every subsequent column index by one.
	const (
		idCol        = 3
		issuesCol    = 47
		gigsDiskCol  = 43
		pctDiskCol   = 44
		gigsTotalCol = 45
	)

	// find the row for hosts[1], it should have issues=1 (1 failing policy) and the expected disk space
	for _, row := range rows[1:] {
		if row[idCol] == fmt.Sprint(hosts[1].ID) {
			assert.Equal(t, "1", row[issuesCol], row)
			assert.Equal(t, "3", row[gigsDiskCol], row)
			assert.Equal(t, "4", row[pctDiskCol], row)
			assert.Equal(t, "1000", row[gigsTotalCol], row)
		} else {
			assert.Equal(t, "0", row[issuesCol], row)
		}
	}

	// valid format, some columns
	res = s.DoRaw(
		"GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv",
		"columns", "hostname,gigs_disk_space_available,percent_disk_space_available,gigs_total_disk_space,team_id,team_name",
	)
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, len(hosts)+1)
	require.Contains(t, rows[0], "hostname") // first row contains headers
	assert.Contains(t, rows[0], "team_id")
	assert.Contains(t, rows[0], "fleet_id")
	assert.Contains(t, rows[0], "team_name")
	assert.Contains(t, rows[0], "fleet_name")
	require.Contains(t, res.Header, "Content-Disposition")
	require.Contains(t, res.Header, "Content-Type")
	require.Contains(t, res.Header, "X-Content-Type-Options")
	require.Contains(t, res.Header.Get("Content-Disposition"), "attachment;")
	require.Contains(t, res.Header.Get("Content-Type"), "text/csv")
	require.Contains(t, res.Header.Get("X-Content-Type-Options"), "nosniff")

	// requesting columns that don't include hardware_model or
	// hardware_marketing_name returns exactly the requested columns and neither
	// hardware field (hardware_marketing_name must not leak into the report).
	res = s.DoRaw(
		"GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv",
		"columns", "hostname,uuid,platform",
	)
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, len(hosts)+1)
	require.Equal(t, []string{"hostname", "uuid", "platform"}, rows[0])
	require.NotContains(t, rows[0], "hardware_model")
	require.NotContains(t, rows[0], "hardware_marketing_name")

	// pagination does not apply to this endpoint, it returns the complete list of hosts
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "page", "1", "per_page", "2", "columns", "hostname")
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, len(hosts)+1)

	// search criteria are applied
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "query", "local0", "columns", "hostname")
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, 2) // headers + matching host
	require.Contains(t, rows[1], hosts[0].Hostname)

	// search criteria including search query with leading/trailing whitespace are applied
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "query", "   local0 ", "columns", "hostname")
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, 2) // headers + matching host
	require.Contains(t, rows[1], hosts[0].Hostname)

	// with device mapping results
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "columns", "id,hostname,device_mapping")
	rawCSV, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Contains(t, string(rawCSV), `"a@b.c,b@b.c"`) // inside quotes because it contains a comma
	rows, err = csv.NewReader(bytes.NewReader(rawCSV)).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, len(hosts)+1)
	for _, row := range rows[1:] {
		if row[0] == fmt.Sprint(hosts[2].ID) {
			require.Equal(t, "a@b.c,b@b.c", row[2], row)
		} else {
			require.Empty(t, row[2], row)
		}
	}

	var putDMResp putHostDeviceMappingResponse
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[0].ID),
		putHostDeviceMappingRequest{Email: "=1+1", Source: "custom"}, http.StatusOK, &putDMResp)

	deviceMappingForHost0 := func(cols string) ([]string, string) {
		res := s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "columns", cols)
		rows, err := csv.NewReader(res.Body).ReadAll()
		res.Body.Close()
		require.NoError(t, err)
		require.Len(t, rows, len(hosts)+1)

		idIx, dmIx := -1, -1
		for i, hdr := range rows[0] {
			switch hdr {
			case "id":
				idIx = i
			case "device_mapping":
				dmIx = i
			}
		}
		require.NotEqual(t, -1, idIx)
		require.NotEqual(t, -1, dmIx)

		for _, row := range rows[1:] {
			if row[idIx] == fmt.Sprint(hosts[0].ID) {
				return rows[0], row[dmIx]
			}
		}
		t.Fatalf("no row found for host %d", hosts[0].ID)
		return nil, ""
	}

	reqCols := []string{"id", "display_name", "device_mapping"}
	hdr, cell := deviceMappingForHost0(strings.Join(reqCols, ","))
	require.Equal(t, reqCols, hdr)
	require.Equal(t, "'=1+1", cell)

	hdr, cell = deviceMappingForHost0("")
	require.Greater(t, len(hdr), len(reqCols))
	require.Equal(t, "'=1+1", cell)

	adminToken := s.token
	s.setTokenForTest(t, TestObserverUserEmail, test.GoodPassword)
	_, cell = deviceMappingForHost0(strings.Join(reqCols, ","))
	require.Equal(t, "'=1+1", cell)
	s.token = adminToken

	require.Len(t, putDMResp.DeviceMapping, 1)
	require.Equal(t, "=1+1", putDMResp.DeviceMapping[0].Email)

	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[0].ID),
		putHostDeviceMappingRequest{Email: ""}, http.StatusOK, &putDMResp)

	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "columns", "hostname", "label_id", fmt.Sprintf("%d", customLabelID))
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, 2) // headers + member host
	require.Contains(t, rows[1], hosts[2].Hostname)

	// with a label id and a search query with leading/trailing whitespace
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "columns", "hostname", "label_id", fmt.Sprintf("%d", customLabelID), "query", "  local2 ")
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, 2) // headers + member host
	// hosts[2] is both matched by the trimmed query and in the provided label
	require.Contains(t, rows[1], hosts[2].Hostname)

	// with a software version id
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "columns", "hostname", "software_version_id", fmt.Sprint(fooV1ID))
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, 2) // headers + member host
	require.Contains(t, rows[1], hosts[0].Hostname)

	// with a software title id
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "columns", "hostname", "software_title_id", fmt.Sprint(fooTitleID))
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, 2)                         // headers + member host
	require.Contains(t, rows[1], hosts[0].Hostname) // nolint:nilaway // createHosts always returns at least one host

	// valid format but an invalid column is provided
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusBadRequest, "format", "csv", "columns", "memory,hostname,status,nosuchcolumn")
	require.NoError(t, json.NewDecoder(res.Body).Decode(&errs))
	res.Body.Close()
	require.Len(t, errs.Errors, 1)
	require.Contains(t, errs.Errors[0].Reason, "nosuchcolumn")

	// valid format, valid columns, order is respected, sorted
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "order_key", "hostname", "order_direction", "desc", "columns", "memory,hostname,status")
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, len(hosts)+1)
	require.Equal(t, []string{"memory", "hostname", "status"}, rows[0]) // first row contains headers
	require.Len(t, rows[1], 3)
	// status is timing-dependent, ignore in the assertion
	require.Equal(t, []string{"0", "TestIntegrations/TestHostsReportDownloadfoo.local2"}, rows[1][:2])
	require.Len(t, rows[2], 3)
	require.Equal(t, []string{"0", "TestIntegrations/TestHostsReportDownloadfoo.local1"}, rows[2][:2])
	require.Len(t, rows[3], 3)
	require.Equal(t, []string{"0", "TestIntegrations/TestHostsReportDownloadfoo.local0"}, rows[3][:2])

	// invalid combinations of software filters
	s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusBadRequest, "software_title_id", "123", "software_id", "456")
	s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusBadRequest, "software_title_id", "123", "software_version_id", "456")
	s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusBadRequest, "software_id", "123", "software_version_id", "456")
	s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusBadRequest, "software_id", "123", "software_version_id", "456", "software_title_id", "789")
}

func (s *integrationTestSuite) TestHostsReportHardwareMarketingName() {
	t := s.T()
	ctx := context.Background()

	newHost := func(suffix, platform, model string) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			OsqueryHostID:   new(t.Name() + suffix),
			NodeKey:         new(t.Name() + suffix),
			UUID:            uuid.New().String(),
			Hostname:        t.Name() + suffix,
			Platform:        platform,
		})
		require.NoError(t, err)
		// hardware_model is not persisted by NewHost, so set it directly.
		mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(ctx, `UPDATE hosts SET hardware_model = ? WHERE id = ?`, model, h.ID)
			return err
		})
		return h
	}

	// Apple host whose model maps to a marketing name, plus a non-Apple host
	// with no mapping.
	mapped := newHost("-mapped", "darwin", "MacBookPro18,1")
	unmapped := newHost("-unmapped", "ubuntu", "Standard PC")

	res := s.DoRaw(
		"GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv",
		"columns", "hostname,hardware_model,hardware_marketing_name",
	)
	rows, err := csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, 3) // header + 2 hosts
	// columns are returned in the requested order
	require.Equal(t, []string{"hostname", "hardware_model", "hardware_marketing_name"}, rows[0])

	byHostname := make(map[string][]string, len(rows)-1)
	for _, row := range rows[1:] {
		byHostname[row[0]] = row
	}
	mappedRow, ok := byHostname[mapped.Hostname]
	require.True(t, ok)
	unmappedRow, ok := byHostname[unmapped.Hostname]
	require.True(t, ok)

	// Apple host: raw model plus the mapped marketing name.
	require.Equal(t, "MacBookPro18,1", mappedRow[1])
	require.Equal(t, fleet.AppleHardwareModelsToMarketingNames["MacBookPro18,1"], mappedRow[2])

	// Non-Apple host: raw model, empty marketing name.
	require.Equal(t, "Standard PC", unmappedRow[1])
	require.Empty(t, unmappedRow[2])
}

func (s *integrationTestSuite) TestGetHostBatteries() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	bats := []*fleet.HostBattery{
		{HostID: host.ID, SerialNumber: "a", CycleCount: 1, Health: "Normal"},
		{HostID: host.ID, SerialNumber: "b", CycleCount: 1002, Health: "Service recommended"},
	}
	require.NoError(t, s.ds.ReplaceHostBatteries(context.Background(), host.ID, bats))

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	// only cycle count and health are returned
	require.ElementsMatch(t, []*fleet.HostBattery{
		{CycleCount: 1, Health: "Normal"},
		{CycleCount: 1002, Health: "Service recommended"},
	}, *getHostResp.Host.Batteries)

	// same for get host by identifier
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", *host.NodeKey), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	// only cycle count and health are returned
	require.ElementsMatch(t, []*fleet.HostBattery{
		{CycleCount: 1, Health: "Normal"},
		{CycleCount: 1002, Health: "Service recommended"},
	}, *getHostResp.Host.Batteries)
}

func (s *integrationTestSuite) TestGetHostMaintenanceWindow() {
	t := s.T()
	ctx := context.Background()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	err = s.ds.ReplaceHostDeviceMapping(ctx, host.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host.ID,
			Email:  "foo@example.com",
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)

	startTime := time.Now().Add(time.Minute).In(time.UTC)
	endTime := startTime.Add(time.Minute * 30)
	testEvent := fleet.CalendarEvent{
		Email:     "foo@example.com",
		StartTime: startTime,
		EndTime:   endTime,
		Data:      []byte(`{}`),
		TimeZone:  nil,
		UUID:      uuid.New().String(),
	}

	dsEvent, err := s.ds.CreateOrUpdateCalendarEvent(ctx, testEvent.UUID, testEvent.Email, testEvent.StartTime, testEvent.EndTime,
		testEvent.Data, testEvent.TimeZone, host.ID, fleet.CalendarWebhookStatusNone)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	// DB methods don't allow nil timezone, since we only allow it for the edge case that the db has
	// just undergone a migration and the calendar_cron has not run to populate the new `time_zone`
	// column yet. This means we need to manually set the timezone to nil.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE calendar_events SET timezone = NULL WHERE id = ?", dsEvent.ID)
		return err
	})

	// GET host, check maintenance window
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	// Round to account for sub-second precision differences between DB and Go
	require.Equal(t, testEvent.StartTime.Round(time.Second), getHostResp.Host.MaintenanceWindow.StartsAt)
	require.Nil(t, getHostResp.Host.MaintenanceWindow.TimeZone)

	timeZone := "America/Argentina/Buenos_Aires"
	// get a time.Location from the timezone string
	tZLoc, err := time.LoadLocation(timeZone)
	require.NoError(t, err)

	// use the time.Location to update the start time for the timezone
	zonedStartsAt := startTime.In(tZLoc).Round(time.Second)

	// update the timezone
	_, err = s.ds.CreateOrUpdateCalendarEvent(ctx, testEvent.UUID, testEvent.Email, testEvent.StartTime, testEvent.EndTime, testEvent.Data,
		&timeZone, host.ID, fleet.CalendarWebhookStatusNone)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	// GET it again
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Equal(t, timeZone, *getHostResp.Host.MaintenanceWindow.TimeZone)

	// for equality comparison with original Go-derived start time, add a Location to the DB-derived start time, which only has an offset
	respStartsAt := getHostResp.Host.MaintenanceWindow.StartsAt
	respSAWithLoc, err := time.ParseInLocation("2006-01-02T15:04:05", respStartsAt.Format("2006-01-02T15:04:05"), tZLoc)
	require.NoError(t, err)

	require.Equal(t, zonedStartsAt, respSAWithLoc)
}

func (s *integrationTestSuite) TestHostByIdentifierSoftwareUpdatedAt() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", *host.NodeKey), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Equal(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)

	time.Sleep(1 * time.Second)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", *host.NodeKey), nil, http.StatusOK, &getHostResp)
	require.Greater(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)
}

func (s *integrationTestSuite) TestGetHostDiskEncryption() {
	t := s.T()

	// create Windows, mac and Linux hosts
	hostWin, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		OsqueryHostID:   new(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		Platform:        "windows",
	})
	require.NoError(t, err)

	hostMac, err := s.ds.NewHost(context.Background(), &fleet.Host{
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
		Platform:        "darwin",
	})
	require.NoError(t, err)

	hostLin, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(strings.ReplaceAll(t.Name(), "/", "_") + "3"),
		OsqueryHostID:   new(strings.ReplaceAll(t.Name(), "/", "_") + "3"),
		UUID:            t.Name() + "3",
		Hostname:        t.Name() + "foo3.local",
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "30-65-EC-6F-C4-60",
		Platform:        "linux",
	})
	require.NoError(t, err)

	listHostsDiskEncryption := func() map[uint]*bool {
		var listResp listHostsResponse
		s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "query", t.Name())
		byID := make(map[uint]*bool, len(listResp.Hosts))
		for _, h := range listResp.Hosts {
			byID[h.ID] = h.DiskEncryptionEnabled
		}
		return byID
	}

	// before any disk encryption is received, all hosts report NULL (even if
	// some have disk space information, i.e. an entry exists in host_disks).
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hostWin.ID, 44.5, 55.6, 90.0, nil))

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostWin.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostWin.ID, getHostResp.Host.ID)
	require.Nil(t, getHostResp.Host.DiskEncryptionEnabled)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostMac.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostMac.ID, getHostResp.Host.ID)
	require.Nil(t, getHostResp.Host.DiskEncryptionEnabled)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostLin.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostLin.ID, getHostResp.Host.ID)
	require.Nil(t, getHostResp.Host.DiskEncryptionEnabled)

	listed := listHostsDiskEncryption()
	require.Len(t, listed, 3)
	require.Nil(t, listed[hostWin.ID])
	require.Nil(t, listed[hostMac.ID])
	require.Nil(t, listed[hostLin.ID])

	// set encrypted for all hosts
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), hostWin.ID, true, nil))
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), hostMac.ID, true, nil))
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), hostLin.ID, true, nil))

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostWin.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostWin.ID, getHostResp.Host.ID)
	require.True(t, *getHostResp.Host.DiskEncryptionEnabled)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostMac.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostMac.ID, getHostResp.Host.ID)
	require.True(t, *getHostResp.Host.DiskEncryptionEnabled)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostLin.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostLin.ID, getHostResp.Host.ID)
	require.True(t, *getHostResp.Host.DiskEncryptionEnabled)

	listed = listHostsDiskEncryption()
	require.Equal(t, new(true), listed[hostWin.ID])
	require.Equal(t, new(true), listed[hostMac.ID])
	require.Equal(t, new(true), listed[hostLin.ID])

	// should succeed as we no longer require MDM to access this endpoint, as Linux encryption doesn't require MDM
	var profiles getMDMProfilesSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &profiles)
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &profiles)

	// set unencrypted for all hosts
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), hostWin.ID, false, nil))
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), hostMac.ID, false, nil))
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), hostLin.ID, false, nil))

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostWin.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostWin.ID, getHostResp.Host.ID)
	require.False(t, *getHostResp.Host.DiskEncryptionEnabled)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostMac.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostMac.ID, getHostResp.Host.ID)
	require.False(t, *getHostResp.Host.DiskEncryptionEnabled)

	// Linux may omit the field when false
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostLin.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostLin.ID, getHostResp.Host.ID)
	require.Nil(t, getHostResp.Host.DiskEncryptionEnabled)

	listed = listHostsDiskEncryption()
	require.Equal(t, new(false), listed[hostWin.ID])
	require.Equal(t, new(false), listed[hostMac.ID])
	require.Nil(t, listed[hostLin.ID])

	// the orbit endpoint to set the disk encryption key always fails in this
	// suite because MDM is not configured.
	orbitHost := createOrbitEnrolledHost(t, "windows", "diskenc", s.ds)
	res := s.Do("POST", "/api/fleet/orbit/disk_encryption_key", fleet.OrbitPostDiskEncryptionKeyRequest{
		OrbitNodeKey:  *orbitHost.OrbitNodeKey,
		EncryptionKey: []byte("testkey"),
	}, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.ErrWindowsMDMNotConfigured.Error())
}

// hostIOSVitalsJSONKeys are the JSON keys of the 29 iOS/iPadOS vitals fields
// added to fleet.Host: they must be fully omitted (not present, not null)
// from the host response for non-iOS/iPadOS hosts, or for a field that's
// absent from the host's host_mdm_apple_device_vitals row.
func (s *integrationTestSuite) TestGetHostIOSVitals() {
	t := s.T()
	ctx := t.Context()

	newHost := func(platform, uuidSuffix string) *fleet.Host {
		name := strings.ReplaceAll(t.Name(), "/", "_") + uuidSuffix
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         new(name),
			OsqueryHostID:   new(name),
			UUID:            name,
			Hostname:        name + ".local",
			PrimaryIP:       "192.168.1.1",
			PrimaryMac:      "30-65-EC-6F-C4-58",
			Platform:        platform,
		})
		require.NoError(t, err)
		return h
	}

	lastBackup := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	fullVitals := fleet.MDMAppleDeviceVitals{
		UDID:                          new("00008030-AAA"),
		ModelNumber:                   new("MNEP3LL/A"),
		ModemFirmwareVersion:          new("2.01.00"),
		SupplementalBuildVersion:      new("21E236"),
		SupplementalOSVersionExtra:    new("a"),
		BluetoothMAC:                  new("a4:83:e7:12:34:57"),
		WiFiMAC:                       new("a4:83:e7:12:34:58"),
		EASDeviceIdentifier:           new("3E2A1F9C"),
		ITunesStoreAccountHash:        new("a1b2c3"),
		PushToken:                     []byte("push-token-bytes"),
		BatteryLevel:                  new(0.87),
		CellularTechnology:            new(int64(1)),
		AppAnalyticsEnabled:           new(true),
		AwaitingConfiguration:         new(false),
		DataRoamingEnabled:            new(false),
		DiagnosticSubmissionEnabled:   new(true),
		IsCloudBackupEnabled:          new(true),
		IsDeviceLocatorServiceEnabled: new(true),
		IsDoNotDisturbInEffect:        new(false),
		IsMDMLostModeEnabled:          new(false),
		IsNetworkTethered:             new(false),
		ITunesStoreAccountIsActive:    new(true),
		PersonalHotspotEnabled:        new(false),
		LastCloudBackupDate:           &lastBackup,
		AccessibilitySettings: &fleet.MDMAppleAccessibilitySettings{
			VoiceOverEnabled: new(true),
			GrayscaleEnabled: new(false),
		},
		OrganizationInfo: &fleet.MDMAppleOrganizationInfo{
			OrganizationName: new("Acme Corp"),
		},
		MDMOptions: &fleet.MDMAppleDeviceVitalsMDMOptions{
			BootstrapTokenAllowed: new(true),
		},
		DevicePropertiesAttestation: [][]byte{[]byte("leaf-cert"), []byte("intermediate-cert")},
		ServiceSubscriptions: []fleet.MDMAppleServiceSubscription{
			{Slot: "CTSubscriptionSlotOne", ICCID: new("iccid-1")},
		},
	}

	// A fully populated iOS host returns all 29 fields, on both GET endpoints.
	fullHost := newHost("ios", "-full")
	require.NoError(t, s.ds.SetOrUpdateHostMDMAppleDeviceVitals(ctx, fullHost.UUID, fullVitals))

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", fullHost.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, "00008030-AAA", *getHostResp.Host.UDID)
	require.InDelta(t, 0.87, *getHostResp.Host.BatteryLevel, 0.001)
	require.True(t, *getHostResp.Host.AccessibilitySettings.VoiceOverEnabled)
	require.Len(t, getHostResp.Host.ServiceSubscriptions, 1)

	hostJSON := s.getHostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", fullHost.ID))
	for _, key := range hostIOSVitalsJSONKeys {
		assert.Contains(t, hostJSON, key, "expected key %q in response for fully populated iOS host", key)
	}
	// cellular_technology is stored as Apple's raw integer but returned as its
	// display label.
	assert.Equal(t, "GSM", hostJSON["cellular_technology"])

	// GET /hosts/identifier/:identifier funnels through the same datastore
	// loading path and must behave identically.
	var getByIdentifierResp getHostResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/identifier/"+fullHost.UUID, nil, http.StatusOK, &getByIdentifierResp)
	require.Equal(t, "00008030-AAA", *getByIdentifierResp.Host.UDID)

	identifierJSON := s.getHostJSON("/api/latest/fleet/hosts/identifier/" + fullHost.UUID)
	for _, key := range hostIOSVitalsJSONKeys {
		assert.Contains(t, identifierJSON, key, "expected key %q in identifier response for fully populated iOS host", key)
	}

	// A non-Apple-mobile host omits all 29 keys.
	macHost := newHost("darwin", "-macos")
	hostJSON = s.getHostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", macHost.ID))
	for _, key := range hostIOSVitalsJSONKeys {
		assert.NotContains(t, hostJSON, key, "did not expect key %q for a non-Apple-mobile host", key)
	}

	// A field absent from the side table row is omitted; other populated
	// fields are still present.
	partialHost := newHost("ipados", "-partial")
	partialVitals := fleet.MDMAppleDeviceVitals{
		UDID:         new("00008030-CCC"),
		BatteryLevel: new(0.5),
	}
	require.NoError(t, s.ds.SetOrUpdateHostMDMAppleDeviceVitals(ctx, partialHost.UUID, partialVitals))

	hostJSON = s.getHostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", partialHost.ID))
	assert.Contains(t, hostJSON, "udid")
	assert.Contains(t, hostJSON, "battery_level")
	assert.NotContains(t, hostJSON, "model_number")
	assert.NotContains(t, hostJSON, "accessibility_settings")
	assert.NotContains(t, hostJSON, "service_subscriptions")

	// An iOS host with no vitals row yet (hasn't refetched since this
	// shipped) omits all 29 keys, with no error.
	noRowHost := newHost("ios", "-no-row")
	hostJSON = s.getHostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", noRowHost.ID))
	for _, key := range hostIOSVitalsJSONKeys {
		assert.NotContains(t, hostJSON, key, "did not expect key %q for an iOS host with no vitals row yet", key)
	}
}

// hostAndroidVitalsJSONKeys are the JSON keys of the 15 Android vitals fields
// added to fleet.Host: they must be fully omitted (not present, not null) from
// the host response for non-Android hosts, or for a field that's absent from
// the host's host_mdm_android_device_vitals row.
var hostAndroidVitalsJSONKeys = []string{
	"adb_enabled", "passcode_protected", "play_protect_enabled", "encryption_type",
	"manufacturer", "security_update_version", "device_kernel_version",
	"bootloader_version", "system_update_status", "security_posture", "api_level",
	"security_posture_details", "telephony_infos", "imei", "meid",
}

func (s *integrationTestSuite) TestGetHostAndroidVitals() {
	t := s.T()
	ctx := t.Context()

	newHost := func(platform, uuidSuffix string) *fleet.Host {
		name := strings.ReplaceAll(t.Name(), "/", "_") + uuidSuffix
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         new(name),
			OsqueryHostID:   new(name),
			UUID:            name,
			Hostname:        name + ".local",
			PrimaryIP:       "192.168.1.1",
			PrimaryMac:      "30-65-EC-6F-C4-58",
			Platform:        platform,
		})
		require.NoError(t, err)
		return h
	}

	fullVitals := fleet.MDMAndroidDeviceVitals{
		AdbEnabled:            new(true),
		PasscodeProtected:     new(true),
		PlayProtectEnabled:    new(false),
		EncryptionType:        new("ACTIVE"),
		Manufacturer:          new("Google"),
		SecurityUpdateVersion: new("2026-05-01"),
		DeviceKernelVersion:   new("6.1.75-android14"),
		BootloaderVersion:     new("slider-1.4-12345678"),
		SystemUpdateStatus:    new("SECURITY_UPDATE_AVAILABLE"),
		SecurityPosture:       new("POTENTIALLY_COMPROMISED"),
		IMEI:                  new("A1000031212"),
		MEID:                  new("A00000292788E1"),
		APILevel:              new(int64(36)),
		SecurityPostureDetails: []fleet.MDMAndroidPostureDetail{
			{SecurityRisk: "COMPROMISED_OS", Advice: []string{"Factory reset the device"}},
		},
		TelephonyInfos: []fleet.MDMAndroidTelephonyInfo{
			{PhoneNumber: "+15555550100", CarrierName: "Acme Mobile"},
			{PhoneNumber: "+15555550101", CarrierName: "Acme Mobile"},
		},
	}

	// A fully populated Android host returns all 15 fields, on both GET endpoints.
	fullHost := newHost("android", "-full")
	require.NoError(t, s.ds.SetOrUpdateHostMDMAndroidDeviceVitals(ctx, fullHost.UUID, fullVitals))

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", fullHost.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, "Google", *getHostResp.Host.Manufacturer)
	require.Equal(t, int64(36), *getHostResp.Host.APILevel)
	require.True(t, *getHostResp.Host.AdbEnabled)
	require.Len(t, getHostResp.Host.SecurityPostureDetails, 1)
	require.Len(t, getHostResp.Host.TelephonyInfos, 2)
	require.Equal(t, "A1000031212", *getHostResp.Host.IMEI)
	require.Equal(t, "A00000292788E1", *getHostResp.Host.MEID)

	hostJSON := s.getHostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", fullHost.ID))
	for _, key := range hostAndroidVitalsJSONKeys {
		assert.Contains(t, hostJSON, key, "expected key %q in response for fully populated Android host", key)
	}
	// Enum values are returned as AMAPI's raw strings; mapping them to display
	// labels is the frontend's job.
	assert.Equal(t, "ACTIVE", hostJSON["encryption_type"])
	assert.Equal(t, "POTENTIALLY_COMPROMISED", hostJSON["security_posture"])

	// GET /hosts/identifier/:identifier funnels through the same datastore
	// loading path and must behave identically.
	var getByIdentifierResp getHostResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/identifier/"+fullHost.UUID, nil, http.StatusOK, &getByIdentifierResp)
	require.Equal(t, "Google", *getByIdentifierResp.Host.Manufacturer)

	identifierJSON := s.getHostJSON("/api/latest/fleet/hosts/identifier/" + fullHost.UUID)
	for _, key := range hostAndroidVitalsJSONKeys {
		assert.Contains(t, identifierJSON, key, "expected key %q in identifier response for fully populated Android host", key)
	}

	// A non-Android host omits all 15 keys, even though a row exists for it.
	// (manufacturer in particular is a plausible-sounding key to leak.)
	macHost := newHost("darwin", "-macos")
	require.NoError(t, s.ds.SetOrUpdateHostMDMAndroidDeviceVitals(ctx, macHost.UUID, fullVitals))
	hostJSON = s.getHostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", macHost.ID))
	for _, key := range hostAndroidVitalsJSONKeys {
		assert.NotContains(t, hostJSON, key, "did not expect key %q for a non-Android host", key)
	}

	// A field absent from the side table row is omitted; other populated
	// fields are still present.
	partialHost := newHost("android", "-partial")
	require.NoError(t, s.ds.SetOrUpdateHostMDMAndroidDeviceVitals(ctx, partialHost.UUID, fleet.MDMAndroidDeviceVitals{
		Manufacturer: new("Samsung"),
		APILevel:     new(int64(34)),
	}))

	hostJSON = s.getHostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", partialHost.ID))
	assert.Contains(t, hostJSON, "manufacturer")
	assert.Contains(t, hostJSON, "api_level")
	assert.NotContains(t, hostJSON, "adb_enabled")
	assert.NotContains(t, hostJSON, "security_posture_details")
	assert.NotContains(t, hostJSON, "telephony_infos")
	assert.NotContains(t, hostJSON, "imei")
	assert.NotContains(t, hostJSON, "meid")

	// An Android host with no vitals row yet (hasn't reported since this
	// shipped) omits all 15 keys, with no error.
	noRowHost := newHost("android", "-no-row")
	hostJSON = s.getHostJSON(fmt.Sprintf("/api/latest/fleet/hosts/%d", noRowHost.ID))
	for _, key := range hostAndroidVitalsJSONKeys {
		assert.NotContains(t, hostJSON, key, "did not expect key %q for an Android host with no vitals row yet", key)
	}
}

func (s *integrationTestSuite) TestOSVersions() {
	t := s.T()

	testOSes := []fleet.OperatingSystem{
		{Name: "macOS", Version: "14.1.2", Arch: "64bit", KernelVersion: "13.37", Platform: "darwin"},                             // os_version_id=1
		{Name: "macOS", Version: "13.2.1", Arch: "64bit", KernelVersion: "18.12", Platform: "darwin"},                             // os_version_id=2
		{Name: "macOS", Version: "13.2.1", Arch: "64bit", KernelVersion: "18.12", Platform: "darwin"},                             // os_version_id=2
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.1", Arch: "64bit", KernelVersion: "10.0.22000.1", Platform: "windows"}, // os_version_id=3
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.1", Arch: "64bit", KernelVersion: "10.0.22000.1", Platform: "windows"}, // os_version_id=3
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.1", Arch: "64bit", KernelVersion: "10.0.22000.1", Platform: "windows"}, // os_version_id=3
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.2", Arch: "64bit", KernelVersion: "10.0.22000.2", Platform: "windows"}, // os_version_id=4
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.2", Arch: "64bit", KernelVersion: "10.0.22000.2", Platform: "windows"}, // os_version_id=4
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.2", Arch: "ARM64", KernelVersion: "10.0.22000.2", Platform: "windows"}, // os_version_id=4
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.2", Arch: "ARM64", KernelVersion: "10.0.22000.2", Platform: "windows"}, // os_version_id=4
	}

	var platforms []string
	for _, os := range testOSes {
		platforms = append(platforms, os.Platform)
	}

	hosts := s.createHosts(t, platforms...)

	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)
	require.Len(t, resp.Hosts, len(hosts))

	// set operating system information on a host
	for i, os := range testOSes {
		require.NoError(t, s.ds.UpdateHostOperatingSystem(context.Background(), hosts[i].ID, os))
	}

	// get OS versions
	osv, err := s.ds.ListOperatingSystems(context.Background())
	require.NoError(t, err)

	osvMap := make(map[string]fleet.OperatingSystem)
	for _, os := range osv {
		key := fmt.Sprintf("%s %s %s", os.Name, os.Version, os.Arch)
		osvMap[key] = os
	}

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_name", testOSes[1].Name, "os_version", testOSes[1].Version)
	require.Len(t, resp.Hosts, 2)

	expected := hosts[1].Hostname
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_version_id", fmt.Sprintf("%d", osvMap["macOS 13.2.1 64bit"].OSVersionID))
	require.Len(t, resp.Hosts, 2)
	require.Equal(t, expected, resp.Hosts[0].Hostname)

	countResp := countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "os_version_id", fmt.Sprintf("%d", osvMap["macOS 13.2.1 64bit"].OSVersionID))
	require.Equal(t, 2, countResp.Count)

	// generate aggregated stats
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))

	// insert Vuln for Win x64
	_, err = s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID: osvMap["Windows 11 Pro 21H2 10.0.22000.2 64bit"].ID,
		CVE:  "CVE-2021-1234",
	}, fleet.MSRCSource)
	require.NoError(t, err)

	// insert duplicate Vuln for Win ARM64
	_, err = s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID: osvMap["Windows 11 Pro 21H2 10.0.22000.2 ARM64"].ID,
		CVE:  "CVE-2021-1234",
	}, fleet.MSRCSource)
	require.NoError(t, err)

	// insert different Vuln for Win ARM64
	_, err = s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID: osvMap["Windows 11 Pro 21H2 10.0.22000.2 ARM64"].ID,
		CVE:  "CVE-2021-5678",
	}, fleet.MSRCSource)
	require.NoError(t, err)

	assertOSVersion := func(t *testing.T, expected fleet.OSVersion, actual fleet.OSVersion) {
		require.Equal(t, expected.HostsCount, actual.HostsCount)
		require.Equal(t, expected.Name, actual.Name)
		require.Equal(t, expected.NameOnly, actual.NameOnly)
		require.Equal(t, expected.Version, actual.Version)
		require.Equal(t, expected.Platform, actual.Platform)
		require.Equal(t, expected.OSVersionID, actual.OSVersionID)
		require.Len(t, actual.Vulnerabilities, len(expected.Vulnerabilities))
		for i, vuln := range expected.Vulnerabilities {
			require.Equal(t, vuln.CVE, actual.Vulnerabilities[i].CVE)
			require.Equal(t, vuln.DetailsLink, actual.Vulnerabilities[i].DetailsLink)
			require.Greater(t, actual.Vulnerabilities[i].CreatedAt, time.Now().Add(-time.Hour)) // assert non-zero value
		}
	}

	var osVersionsResp osVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
	require.Len(t, osVersionsResp.OSVersions, 4) // different archs are grouped together

	// Default sort is by hosts count, descending
	expectedVersion := fleet.OSVersion{
		HostsCount:  4,
		Name:        "Windows 11 Pro 21H2 10.0.22000.2",
		NameOnly:    "Windows 11 Pro 21H2",
		Version:     "10.0.22000.2",
		Platform:    "windows",
		OSVersionID: osvMap["Windows 11 Pro 21H2 10.0.22000.2 ARM64"].OSVersionID,
		Vulnerabilities: fleet.Vulnerabilities{
			{
				CVE:         "CVE-2021-1234",
				DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1234",
			},
			{
				CVE:         "CVE-2021-5678", // vulns are aggregated by OS name and version
				DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-5678",
			},
		},
	}

	// Default sort is by hosts count, descending
	assertOSVersion(t, expectedVersion, osVersionsResp.OSVersions[0])

	// get OS version by id
	var osVersionResp getOSVersionResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osvMap["Windows 11 Pro 21H2 10.0.22000.2 ARM64"].OSVersionID), nil, http.StatusOK, &osVersionResp)
	assertOSVersion(t, expectedVersion, *osVersionResp.OSVersion)

	// invalid id returns a not-found error rather than an empty object
	s.DoJSON("GET", "/api/latest/fleet/os_versions/999", nil, http.StatusNotFound, &osVersionResp)

	// name and version filters
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "os_name", "Windows 11 Pro 21H2", "os_version", "10.0.22000.2")
	require.Len(t, osVersionsResp.OSVersions, 1)
	require.Equal(t, "Windows 11 Pro 21H2 10.0.22000.2", osVersionsResp.OSVersions[0].Name)
	require.Len(t, osVersionsResp.OSVersions[0].Vulnerabilities, 2)

	// name without version
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusBadRequest, &osVersionsResp, "os_name", "Windows 11 Pro 21H2")

	// version without name
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusBadRequest, &osVersionsResp, "os_version", "10.0.22000.1")

	// invalid order key
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusBadRequest, &osVersionsResp, "order_key", "nosuchkey")

	// ascending order by hosts count
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "order_key", "hosts_count", "order_direction", "asc")
	require.Equal(t, 1, osVersionsResp.OSVersions[0].HostsCount)
	require.Equal(t, "macOS 14.1.2", osVersionsResp.OSVersions[0].Name)

	// test pagination
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "page", "0", "per_page", "2")
	require.Len(t, osVersionsResp.OSVersions, 2)
	require.Equal(t, "Windows 11 Pro 21H2 10.0.22000.2", osVersionsResp.OSVersions[0].Name)
	require.Equal(t, "Windows 11 Pro 21H2 10.0.22000.1", osVersionsResp.OSVersions[1].Name)
	require.Equal(t, 4, osVersionsResp.Count)
	require.True(t, osVersionsResp.Meta.HasNextResults)
	require.False(t, osVersionsResp.Meta.HasPreviousResults)

	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "page", "1", "per_page", "2")
	require.Len(t, osVersionsResp.OSVersions, 2)
	require.Equal(t, "macOS 13.2.1", osVersionsResp.OSVersions[0].Name)
	require.Equal(t, "macOS 14.1.2", osVersionsResp.OSVersions[1].Name)
	require.Equal(t, 4, osVersionsResp.Count)
	require.False(t, osVersionsResp.Meta.HasNextResults)
	require.True(t, osVersionsResp.Meta.HasPreviousResults)

	// same results with team_id=0
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "page", "1", "per_page", "2", "team_id", "0")
	require.Len(t, osVersionsResp.OSVersions, 2)
	require.Equal(t, "macOS 13.2.1", osVersionsResp.OSVersions[0].Name)
	require.Equal(t, "macOS 14.1.2", osVersionsResp.OSVersions[1].Name)
	require.Equal(t, 4, osVersionsResp.Count)
	require.False(t, osVersionsResp.Meta.HasNextResults)
	require.True(t, osVersionsResp.Meta.HasPreviousResults)
}

func (s *integrationTestSuite) TestHostsReportWithPolicyResults() {
	t := s.T()
	ctx := context.Background()

	newHostFunc := func(name string) *fleet.Host {
		host, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         new(name),
			UUID:            name,
			Hostname:        "foo.local." + name,
		})
		require.NoError(t, err)
		require.NotNil(t, host)
		return host
	}

	hostCount := 10
	hosts := make([]*fleet.Host, 0, hostCount)
	for i := range hostCount {
		hosts = append(hosts, newHostFunc(fmt.Sprintf("h%d", i)))
	}

	globalPolicy0, err := s.ds.NewGlobalPolicy(ctx, &test.UserAdmin.ID, fleet.PolicyPayload{
		Name:  "foobar0",
		Query: "SELECT 0;",
	})
	require.NoError(t, err)
	globalPolicy1, err := s.ds.NewGlobalPolicy(ctx, &test.UserAdmin.ID, fleet.PolicyPayload{
		Name:  "foobar1",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	globalPolicy2, err := s.ds.NewGlobalPolicy(ctx, &test.UserAdmin.ID, fleet.PolicyPayload{
		Name:  "foobar2",
		Query: "SELECT 2;",
	})
	require.NoError(t, err)

	for i, host := range hosts {
		// All hosts pass the globalPolicy0
		_, err := s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{globalPolicy0.ID: new(true)}, time.Now(), false, nil)
		require.NoError(t, err)

		if i%2 == 0 {
			// Half of the hosts pass the globalPolicy1 and fail the globalPolicy2
			_, err := s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{globalPolicy1.ID: new(true)}, time.Now(), false, nil)
			require.NoError(t, err)
			_, err = s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{globalPolicy2.ID: new(false)}, time.Now(), false, nil)
			require.NoError(t, err)
		} else {
			// Half of the hosts pass the globalPolicy2 and fail the globalPolicy1
			_, err := s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{globalPolicy1.ID: new(false)}, time.Now(), false, nil)
			require.NoError(t, err)
			_, err = s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{globalPolicy2.ID: new(true)}, time.Now(), false, nil)
			require.NoError(t, err)
		}
	}

	// The hosts/report endpoint uses svc.ds.ListHosts with page=0, per_page=0, thus we are
	// testing the non optimized for pagination queries for failing policies calculation.
	res := s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv")
	rows1, err := csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows1, len(hosts)+1) // all hosts + header row
	assert.Len(t, rows1[0], 58)         // total number of cols

	var (
		idIdx     int
		issuesIdx int
	)
	for colIdx, column := range rows1[0] {
		switch column {
		case "issues":
			issuesIdx = colIdx
		case "id":
			idIdx = colIdx
		}
	}

	for i := 1; i < len(hosts)+1; i++ {
		row := rows1[i]
		require.Equal(t, "1", row[issuesIdx])
	}

	// Running with disable_issues=true (which overrides disable_failing_policies=false) disable the counting of failed policies for a host.
	// Thus, all "issues" values should be 0.
	res = s.DoRaw(
		"GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "disable_failing_policies", "false", "disable_issues",
		"true",
	)
	rows2, err := csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows2, len(hosts)+1) // all hosts + header row
	assert.Len(t, rows2[0], 58)         // total number of cols

	// Check that all hosts have 0 issues and that they match the previous call to `/hosts/report`.
	for i := 1; i < len(hosts)+1; i++ {
		row := rows2[i]
		require.Equal(t, "0", row[issuesIdx])
		row1 := rows1[i]
		require.Equal(t, row[idIdx], row1[idIdx])
	}

	for _, tc := range []struct {
		name      string
		args      []string
		checkRows func(t *testing.T, rows [][]string)
	}{
		{
			name: "get hosts that fail globalPolicy0",
			args: []string{"policy_id", fmt.Sprint(globalPolicy0.ID), "policy_response", "failing"},
			checkRows: func(t *testing.T, rows [][]string) {
				require.Len(t, rows, 1) // just header row, all hosts pass such policy.
			},
		},
		{
			name: "get hosts that pass globalPolicy0",
			args: []string{"policy_id", fmt.Sprint(globalPolicy0.ID), "policy_response", "passing"},
			checkRows: func(t *testing.T, rows [][]string) {
				require.Len(t, rows, len(hosts)+1) // all hosts + header row, all hosts pass such policy.
			},
		},
		{
			name: "get hosts that fail globalPolicy1",
			args: []string{"policy_id", fmt.Sprint(globalPolicy1.ID), "policy_response", "failing"},
			checkRows: func(t *testing.T, rows [][]string) {
				require.Len(t, rows, len(hosts)/2+1) // half of hosts + header row.
			},
		},
		{
			name: "get hosts that pass globalPolicy1",
			args: []string{"policy_id", fmt.Sprint(globalPolicy1.ID), "policy_response", "passing"},
			checkRows: func(t *testing.T, rows [][]string) {
				require.Len(t, rows, len(hosts)/2+1) // half of hosts + header row.
			},
		},
		{
			name: "get hosts that fail globalPolicy2",
			args: []string{"policy_id", fmt.Sprint(globalPolicy2.ID), "policy_response", "failing"},
			checkRows: func(t *testing.T, rows [][]string) {
				require.Len(t, rows, len(hosts)/2+1) // half of hosts + header row.
			},
		},
		{
			name: "get hosts that pass globalPolicy2",
			args: []string{"policy_id", fmt.Sprint(globalPolicy2.ID), "policy_response", "passing"},
			checkRows: func(t *testing.T, rows [][]string) {
				require.Len(t, rows, len(hosts)/2+1) // half of hosts + header row.
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, append(tc.args, "format", "csv")...)
			rows, err := csv.NewReader(res.Body).ReadAll()
			res.Body.Close()
			require.NoError(t, err)
			tc.checkRows(t, rows)
			// Test the same with "disable_issues=true" which should not change the result.
			res = s.DoRaw(
				"GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, append(tc.args, "format", "csv", "disable_issues", "true")...,
			)
			rows, err = csv.NewReader(res.Body).ReadAll()
			res.Body.Close()
			require.NoError(t, err)
			tc.checkRows(t, rows)
		})
	}
}

func (s *integrationTestSuite) TestHostHealth() {
	t := s.T()

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
		TeamID:          nil,
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		{Name: "baz", Version: "0.0.4", Source: "apps"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))

	soft1 := host.Software[0]
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	cpes := []fleet.SoftwareCPE{{SoftwareID: soft1.ID, CPE: "somecpe"}}
	_, err = s.ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	// Reload software so that 'GeneratedCPEID is set.
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))
	soft1 = host.Software[0]
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	inserted, err := s.ds.InsertSoftwareVulnerability(
		context.Background(), fleet.SoftwareVulnerability{
			SoftwareID: soft1.ID,
			CVE:        "cve-123-123-132",
		}, fleet.NVDSource,
	)
	require.NoError(t, err)
	require.True(t, inserted)

	passingPolicy, err := s.ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{
		Name:       "passing_policy",
		Query:      "select 1",
		Resolution: "Run this command to fix it",
	})
	require.NoError(t, err)

	failingPolicy, err := s.ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{
		Name:       "failing_policy",
		Query:      "select 0",
		Resolution: "Run this command to fix it",
	})
	require.NoError(t, err)

	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{failingPolicy.ID: new(false)}, time.Now(), false, nil)))
	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{passingPolicy.ID: new(true)}, time.Now(), false, nil)))

	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), host.ID, true, nil))

	// Get host health
	hh := getHostHealthResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/health", host.ID), nil, http.StatusOK, &hh)
	assert.Equal(t, host.ID, hh.HostID)
	assert.NotNil(t, hh.HostHealth)
	assert.Equal(t, host.OSVersion, hh.HostHealth.OsVersion)
	assert.Len(t, hh.HostHealth.VulnerableSoftware, 1)
	assert.Equal(t, fleet.HostHealthVulnerableSoftware{
		ID:      soft1.ID,
		Name:    soft1.Name,
		Version: soft1.Version,
	}, hh.HostHealth.VulnerableSoftware[0])
	assert.Equal(t, 1, hh.HostHealth.FailingPoliciesCount)
	assert.Nil(t, hh.HostHealth.FailingCriticalPoliciesCount)
	assert.Len(t, hh.HostHealth.FailingPolicies, 1)
	assert.Equal(t, &fleet.HostHealthFailingPolicy{
		ID:         failingPolicy.ID,
		Name:       failingPolicy.Name,
		Resolution: failingPolicy.Resolution,
		Critical:   nil,
	}, hh.HostHealth.FailingPolicies[0])
	assert.True(t, *hh.HostHealth.DiskEncryptionEnabled)
	// Check that the TeamID didn't make it into the response
	assert.Nil(t, hh.HostHealth.TeamID)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/health", 0), nil, http.StatusNotFound, &hh)

	resp := getHostHealthResponse{}
	host1, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		OsqueryHostID:   new(t.Name() + "hostid2"),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "nodekey2"),
		UUID:            t.Name() + "uuid2",
		Hostname:        t.Name() + "foo2.local",
		PrimaryIP:       "192.168.2.2",
		PrimaryMac:      "32-62-E2-62-C2-52",
		OSVersion:       "Mac OS X 10.14.2",
		Platform:        "darwin",
		CPUType:         "cpuType",
	})
	require.NoError(t, err)
	require.NotNil(t, host1)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/health", host1.ID), nil, http.StatusOK, &resp)
	assert.Equal(t, host1.ID, resp.HostID)
	assert.NotNil(t, resp.HostHealth)
	assert.Equal(t, host1.OSVersion, resp.HostHealth.OsVersion)
	assert.Nil(t, resp.HostHealth.DiskEncryptionEnabled)
	assert.Empty(t, resp.HostHealth.VulnerableSoftware)
	assert.Empty(t, resp.HostHealth.FailingPolicies)
	assert.Nil(t, resp.HostHealth.TeamID)
}

func (s *integrationTestSuite) TestHostPastActivities() {
	t := s.T()
	ctx := context.Background()
	user := s.users["admin1@example.com"]
	getDetails := func(a *fleet.Activity) fleet.ActivityTypeRanScript {
		var details fleet.ActivityTypeRanScript
		err := json.Unmarshal([]byte(*a.Details), &details)
		require.NoError(t, err)

		return details
	}

	host := createOrbitEnrolledHost(t, "linux", "", s.ds)
	err := s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now())
	require.NoError(t, err)

	// create a valid script execution request
	savedScript, err := s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         nil,
		Name:           "saved.sh",
		ScriptContents: "echo 'hello world'",
	})
	require.NoError(t, err)

	var runResp fleet.RunScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &savedScript.ID}, http.StatusAccepted, &runResp)
	require.Equal(t, host.ID, runResp.HostID)
	require.NotEmpty(t, runResp.ExecutionID)

	execID1 := runResp.ExecutionID

	result, err := s.ds.GetHostScriptExecutionResult(ctx, runResp.ExecutionID)
	require.NoError(t, err)
	require.Equal(t, host.ID, result.HostID)
	require.Equal(t, "echo 'hello world'", result.ScriptContents)
	require.Nil(t, result.ExitCode)

	var orbitPostScriptResp fleet.OrbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, result.ExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	var listResp listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &listResp)

	require.Len(t, listResp.Activities, 1)
	require.Equal(t, user.Email, *listResp.Activities[0].ActorEmail)
	require.Equal(t, user.Name, *listResp.Activities[0].ActorFullName)
	require.Equal(t, user.GravatarURL, *listResp.Activities[0].ActorGravatar)
	require.Equal(t, "ran_script", listResp.Activities[0].Type)
	d := getDetails(listResp.Activities[0])
	require.Equal(t, execID1, d.ScriptExecutionID)
	require.Equal(t, savedScript.Name, d.ScriptName)
	require.Equal(t, host.DisplayName(), d.HostDisplayName)
	require.Equal(t, host.ID, d.HostID)
	require.True(t, d.Async)

	// sleep to have the created_at timestamps differ
	time.Sleep(time.Second)

	// Execute another script in order to test query params
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo 'foobar'"}, http.StatusAccepted, &runResp)
	require.Equal(t, host.ID, runResp.HostID)
	require.NotEmpty(t, runResp.ExecutionID)

	execID2 := runResp.ExecutionID

	result, err = s.ds.GetHostScriptExecutionResult(ctx, runResp.ExecutionID)
	require.NoError(t, err)
	require.Equal(t, host.ID, result.HostID)
	require.Equal(t, "echo 'foobar'", result.ScriptContents)
	require.Nil(t, result.ExitCode)

	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, result.ExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &listResp, "page", "0", "per_page", "1")

	require.Len(t, listResp.Activities, 1)
	d = getDetails(listResp.Activities[0])

	require.Equal(t, execID2, d.ScriptExecutionID)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &listResp, "page", "1", "per_page", "1")

	require.Len(t, listResp.Activities, 1)
	d = getDetails(listResp.Activities[0])
	require.Equal(t, execID1, d.ScriptExecutionID)
}

func (s *integrationTestSuite) TestListHostUpcomingActivities() {
	t := s.T()
	ctx := context.Background()

	adminUser, err := s.ds.UserByEmail(ctx, "admin1@example.com")
	require.NoError(t, err)

	// there is already a datastore-layer test that verifies that correct values
	// are returned for users, saved scripts, etc. so this is more focused on
	// verifying that the service layer passes the proper options and the
	// rendering of the response.

	host1, err := s.ds.NewHost(ctx, &fleet.Host{
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

	// create script execution requests
	hsr, err := s.ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host1.ID, ScriptContents: "A", SyncRequest: true})
	require.NoError(t, err)
	h1A := hsr.ExecutionID
	hsr, err = s.ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host1.ID, ScriptContents: "B"})
	require.NoError(t, err)
	h1B := hsr.ExecutionID
	hsr, err = s.ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host1.ID, ScriptContents: "C"})
	require.NoError(t, err)
	h1C := hsr.ExecutionID
	hsr, err = s.ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host1.ID, ScriptContents: "D", SyncRequest: true})
	require.NoError(t, err)
	h1D := hsr.ExecutionID
	hsr, err = s.ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host1.ID, ScriptContents: "E"})
	require.NoError(t, err)
	h1E := hsr.ExecutionID

	// create a software installation request
	tfr1, err := fleet.NewTempFileReader(strings.NewReader("echo"), t.TempDir)
	require.NoError(t, err)
	sw1, _, err := s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install foo",
		InstallerFile:   tfr1,
		StorageID:       uuid.NewString(),
		Filename:        "foo.pkg",
		Title:           "foo",
		Source:          "apps",
		Version:         "0.0.1",
		UserID:          adminUser.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)
	s1Meta, err := s.ds.GetSoftwareInstallerMetadataByID(ctx, sw1)
	require.NoError(t, err)
	h1Foo, err := s.ds.InsertSoftwareInstallRequest(ctx, host1.ID, s1Meta.InstallerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)

	// force an order to the activities (h1A must be first as it was automatically activated)
	mysqltest.SetOrderedCreatedAtTimestamps(t, s.ds, time.Now(), "upcoming_activities", "execution_id", h1A, h1B, h1Foo, h1C, h1D, h1E)

	// modify the timestamp h1A and h1B to simulate an script that has been
	// pending for a long time (h1A is a sync script but that doesn't change
	// anything anymore, sync scripts are part of the queue like any other:
	// https://github.com/fleetdm/fleet/issues/22866#issuecomment-2575961141)
	mysqltest.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "UPDATE upcoming_activities SET created_at = ? WHERE execution_id = ?", time.Now().Add(-24*time.Hour), h1A)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "UPDATE upcoming_activities SET created_at = ? WHERE execution_id = ?", time.Now().Add(-23*time.Hour), h1B)
		return err
	})

	cases := []struct {
		queries   []string // alternate query name and value
		wantExecs []string
		wantMeta  *fleet.PaginationMetadata
	}{
		{
			wantExecs: []string{h1A, h1B, h1Foo, h1C, h1D, h1E},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2"},
			wantExecs: []string{h1A, h1B},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2", "page", "1"},
			wantExecs: []string{h1Foo, h1C},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "2", "page", "2"},
			wantExecs: []string{h1D, h1E},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "2", "page", "3"},
			wantExecs: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3"},
			wantExecs: []string{h1A, h1B, h1Foo},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "3", "page", "1"},
			wantExecs: []string{h1C, h1D, h1E},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3", "page", "2"},
			wantExecs: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%#v", c.queries), func(t *testing.T) {
			var listResp listHostUpcomingActivitiesResponse
			queryArgs := c.queries
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host1.ID), nil, http.StatusOK, &listResp, queryArgs...)

			require.Equal(t, uint(6), listResp.Count)
			require.Len(t, listResp.Activities, len(c.wantExecs))
			require.Equal(t, c.wantMeta, listResp.Meta)

			var gotExecs []string
			if len(listResp.Activities) > 0 {
				gotExecs = make([]string, len(listResp.Activities))
				for i, a := range listResp.Activities {
					require.Zero(t, a.ID)
					require.NotEmpty(t, a.UUID)
					require.Contains(t, []string{
						fleet.ActivityTypeRanScript{}.ActivityName(),
						fleet.ActivityTypeInstalledSoftware{}.ActivityName(),
					}, a.Type)

					var details map[string]any
					require.NotNil(t, a.Details)
					require.NoError(t, json.Unmarshal(*a.Details, &details))
					switch a.Type {
					case fleet.ActivityTypeRanScript{}.ActivityName():
						gotExecs[i] = details["script_execution_id"].(string)
					case fleet.ActivityTypeInstalledSoftware{}.ActivityName():
						gotExecs[i] = details["install_uuid"].(string)
					}
				}
			}
			require.Equal(t, c.wantExecs, gotExecs)
		})
	}

	// Test with a host that has no upcoming activities
	host2, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name() + "2"),
		NodeKey:         new(t.Name() + "2"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo2.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	var listResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host2.ID), nil, http.StatusOK, &listResp)
	require.Equal(t, uint(0), listResp.Count)
	require.Empty(t, listResp.Activities)
	require.Equal(t, &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false}, listResp.Meta)
}

func (s *integrationTestSuite) TestListHostReports() {
	t := s.T()
	ctx := t.Context()

	now := time.Now().UTC().Truncate(time.Second)
	earlier := now.Add(-time.Hour)

	// Create a global host (no team).
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        t.Name() + ".local",
		Platform:        "linux",
	})
	require.NoError(t, err)

	// Create two global queries that save results and one that discards them.
	admin := s.users[TestAdminUserEmail]
	qAlpha, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:        t.Name() + "_alpha",
		Query:       "SELECT 1",
		AuthorID:    &admin.ID,
		Saved:       true,
		DiscardData: false,
		Logging:     fleet.LoggingSnapshot,
		Description: "alpha description",
	})
	require.NoError(t, err)

	qBeta, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:        t.Name() + "_beta",
		Query:       "SELECT 2",
		AuthorID:    &admin.ID,
		Saved:       true,
		DiscardData: false,
		Logging:     fleet.LoggingSnapshot,
		Description: "beta description",
	})
	require.NoError(t, err)

	qDiscard, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:        t.Name() + "_discard",
		Query:       "SELECT 3",
		AuthorID:    &admin.ID,
		Saved:       true,
		DiscardData: true,
		Logging:     fleet.LoggingDifferential, // non-snapshot + discard_data=1 → "don't store results"
	})
	require.NoError(t, err)

	// Insert two result rows for qAlpha on the host (to test has_more_results and first_result).
	_, err = s.ds.OverwriteQueryResultRows(ctx, []*fleet.ScheduledQueryResultRow{
		{QueryID: qAlpha.ID, HostID: host.ID, LastFetched: earlier, Data: new(json.RawMessage(`{"col":"older"}`))},
		{QueryID: qAlpha.ID, HostID: host.ID, LastFetched: now, Data: new(json.RawMessage(`{"col":"newest"}`))},
	}, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	// Insert one result row for qDiscard (only appears when include_reports_dont_store_results=true).
	_, err = s.ds.OverwriteQueryResultRows(ctx, []*fleet.ScheduledQueryResultRow{
		{QueryID: qDiscard.ID, HostID: host.ID, LastFetched: now, Data: new(json.RawMessage(`{"col":"discarded"}`))},
	}, fleet.DefaultMaxQueryReportRows)
	require.NoError(t, err)

	url := fmt.Sprintf("/api/latest/fleet/hosts/%d/reports", host.ID)

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		var resp listHostReportsResponse
		s.DoJSONWithoutAuth("GET", url, nil, http.StatusUnauthorized, &resp)
	})

	t.Run("observer can read reports", func(t *testing.T) {
		s.setTokenForTest(t, TestObserverUserEmail, test.GoodPassword)
		var resp listHostReportsResponse
		s.DoJSON("GET", url, nil, http.StatusOK, &resp)
		require.NoError(t, resp.Err)
	})

	t.Run("admin can read reports", func(t *testing.T) {
		s.setTokenForTest(t, TestAdminUserEmail, test.GoodPassword)
		var resp listHostReportsResponse
		s.DoJSON("GET", url, nil, http.StatusOK, &resp)
		require.NoError(t, resp.Err)
	})

	t.Run("nonexistent host returns 404", func(t *testing.T) {
		var resp listHostReportsResponse
		s.DoJSON("GET", "/api/latest/fleet/hosts/99999999/reports", nil, http.StatusNotFound, &resp)
	})

	t.Run("default excludes dont-store-results queries", func(t *testing.T) {
		var resp listHostReportsResponse
		s.DoJSON("GET", url, nil, http.StatusOK, &resp, "order_key", "name")
		require.NoError(t, resp.Err)
		assert.Equal(t, 2, resp.Count)
		require.Len(t, resp.Reports, 2)
		assert.Equal(t, qAlpha.Name, resp.Reports[0].Name)
		assert.Equal(t, qBeta.Name, resp.Reports[1].Name)
		for _, r := range resp.Reports {
			assert.NotEqual(t, qDiscard.Name, r.Name)
		}
	})

	t.Run("include_reports_dont_store_results=true returns all queries", func(t *testing.T) {
		var resp listHostReportsResponse
		s.DoJSON("GET", url, nil, http.StatusOK, &resp, "include_reports_dont_store_results", "true", "order_key", "name")
		require.NoError(t, resp.Err)
		assert.Equal(t, 3, resp.Count)
		require.Len(t, resp.Reports, 3)
		assert.Equal(t, qAlpha.Name, resp.Reports[0].Name)
		assert.Equal(t, qBeta.Name, resp.Reports[1].Name)
		assert.Equal(t, qDiscard.Name, resp.Reports[2].Name)
	})

	t.Run("response fields are populated correctly", func(t *testing.T) {
		var resp listHostReportsResponse
		s.DoJSON("GET", url, nil, http.StatusOK, &resp, "order_key", "name")
		require.NoError(t, resp.Err)
		require.Len(t, resp.Reports, 2)

		alpha := resp.Reports[0]
		assert.Equal(t, qAlpha.ID, alpha.ReportID)
		assert.Equal(t, qAlpha.Name, alpha.Name)
		assert.Equal(t, "alpha description", alpha.Description)
		// first_result is the most recent row.
		require.NotNil(t, alpha.FirstResult)
		assert.Equal(t, "newest", alpha.FirstResult["col"])
		// last_fetched is the most recent timestamp.
		require.NotNil(t, alpha.LastFetched)
		assert.Equal(t, now.Unix(), alpha.LastFetched.Unix())
		// n_host_results is 2 because there are 2 rows.
		assert.Equal(t, 2, alpha.NHostResults)
		assert.False(t, alpha.ReportClipped)
		// qAlpha stores results (discard_data=false).
		assert.True(t, alpha.StoreResults)

		// qBeta has no results yet.
		beta := resp.Reports[1]
		assert.Equal(t, qBeta.ID, beta.ReportID)
		assert.Nil(t, beta.FirstResult)
		assert.Nil(t, beta.LastFetched)
		assert.Equal(t, 0, beta.NHostResults)
		// qBeta also stores results.
		assert.True(t, beta.StoreResults)
	})

	t.Run("store_results is false for discard queries", func(t *testing.T) {
		var resp listHostReportsResponse
		s.DoJSON("GET", url, nil, http.StatusOK, &resp, "include_reports_dont_store_results", "true", "order_key", "name")
		require.NoError(t, resp.Err)
		require.Len(t, resp.Reports, 3)
		// qDiscard has discard_data=true → store_results must be false.
		discard := resp.Reports[2]
		assert.Equal(t, qDiscard.Name, discard.Name)
		assert.False(t, discard.StoreResults)
	})

	t.Run("report_clipped when total results reach the cap", func(t *testing.T) {
		// Save the current cap before mutating.
		var originalConfig fleet.AppConfig
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &originalConfig)
		origCap := originalConfig.ServerSettings.QueryReportCap

		// Set the report cap to 2, which equals the number of rows already stored for qAlpha.
		s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{"server_settings":{"query_report_cap":2}}`), http.StatusOK)
		t.Cleanup(func() {
			s.DoRaw("PATCH", "/api/latest/fleet/config",
				fmt.Appendf(nil, `{"server_settings":{"query_report_cap":%d}}`, origCap),
				http.StatusOK)
		})

		var resp listHostReportsResponse
		s.DoJSON("GET", url, nil, http.StatusOK, &resp, "order_key", "name")
		require.NoError(t, resp.Err)
		require.Len(t, resp.Reports, 2)
		assert.True(t, resp.Reports[0].ReportClipped)  // qAlpha has 2 rows == cap of 2
		assert.False(t, resp.Reports[1].ReportClipped) // qBeta has 0 rows
	})

	t.Run("name search", func(t *testing.T) {
		var resp listHostReportsResponse
		s.DoJSON("GET", url, nil, http.StatusOK, &resp, "query", "alpha")
		require.NoError(t, resp.Err)
		assert.Equal(t, 1, resp.Count)
		require.Len(t, resp.Reports, 1)
		assert.Equal(t, qAlpha.Name, resp.Reports[0].Name)
	})

	t.Run("sort by name ascending and descending", func(t *testing.T) {
		var resp listHostReportsResponse

		// Ascending (A→Z).
		s.DoJSON("GET", url, nil, http.StatusOK, &resp, "order_key", "name", "order_direction", "asc")
		require.Len(t, resp.Reports, 2)
		assert.Equal(t, qAlpha.Name, resp.Reports[0].Name)
		assert.Equal(t, qBeta.Name, resp.Reports[1].Name)

		// Descending (Z→A).
		s.DoJSON("GET", url, nil, http.StatusOK, &resp, "order_key", "name", "order_direction", "desc")
		require.Len(t, resp.Reports, 2)
		assert.Equal(t, qBeta.Name, resp.Reports[0].Name)
		assert.Equal(t, qAlpha.Name, resp.Reports[1].Name)
	})

	t.Run("sort by last_fetched puts nulls last", func(t *testing.T) {
		var resp listHostReportsResponse

		// ASC: qAlpha (has results) before qBeta (no results / NULL).
		s.DoJSON("GET", url, nil, http.StatusOK, &resp, "order_key", "last_fetched", "order_direction", "asc")
		require.Len(t, resp.Reports, 2)
		assert.Equal(t, qAlpha.Name, resp.Reports[0].Name)
		assert.Equal(t, qBeta.Name, resp.Reports[1].Name)

		// DESC: same NULL-last behaviour.
		s.DoJSON("GET", url, nil, http.StatusOK, &resp, "order_key", "last_fetched", "order_direction", "desc")
		require.Len(t, resp.Reports, 2)
		assert.Equal(t, qAlpha.Name, resp.Reports[0].Name)
		assert.Equal(t, qBeta.Name, resp.Reports[1].Name)
	})

	t.Run("pagination", func(t *testing.T) {
		var resp listHostReportsResponse

		// Page 0, 1 item per page.
		s.DoJSON("GET", url, nil, http.StatusOK, &resp,
			"order_key", "name", "per_page", "1", "page", "0")
		require.NoError(t, resp.Err)
		assert.Equal(t, 2, resp.Count)
		require.Len(t, resp.Reports, 1)
		assert.Equal(t, qAlpha.Name, resp.Reports[0].Name)
		require.NotNil(t, resp.Meta)
		assert.True(t, resp.Meta.HasNextResults)
		assert.False(t, resp.Meta.HasPreviousResults)

		// Page 1, 1 item per page.
		s.DoJSON("GET", url, nil, http.StatusOK, &resp,
			"order_key", "name", "per_page", "1", "page", "1")
		require.NoError(t, resp.Err)
		assert.Equal(t, 2, resp.Count)
		require.Len(t, resp.Reports, 1)
		assert.Equal(t, qBeta.Name, resp.Reports[0].Name)
		require.NotNil(t, resp.Meta)
		assert.False(t, resp.Meta.HasNextResults)
		assert.True(t, resp.Meta.HasPreviousResults)
	})

	t.Run("default page size is 50", func(t *testing.T) {
		var resp listHostReportsResponse
		s.DoJSON("GET", url, nil, http.StatusOK, &resp)
		require.NoError(t, resp.Err)
		require.NotNil(t, resp.Meta)
		// With only 2 queries, both fit within the default page size of 50.
		assert.False(t, resp.Meta.HasNextResults)
		assert.Len(t, resp.Reports, 2)
	})

	t.Run("alt path /hosts/:id/queries works", func(t *testing.T) {
		altURL := fmt.Sprintf("/api/latest/fleet/hosts/%d/queries", host.ID)
		var resp listHostReportsResponse
		s.DoJSON("GET", altURL, nil, http.StatusOK, &resp, "order_key", "name")
		require.NoError(t, resp.Err)
		assert.Equal(t, 2, resp.Count)
	})

	t.Run("default sort is last_fetched descending (newest first)", func(t *testing.T) {
		// With no order params, the endpoint defaults to last_fetched DESC.
		// qAlpha has results (non-NULL last_fetched) so it comes first; qBeta
		// has no results (NULL) so it sorts last.
		var resp listHostReportsResponse
		s.DoJSON("GET", url, nil, http.StatusOK, &resp)
		require.NoError(t, resp.Err)
		require.Len(t, resp.Reports, 2)
		assert.Equal(t, qAlpha.Name, resp.Reports[0].Name)
		assert.Equal(t, qBeta.Name, resp.Reports[1].Name)
	})

	t.Run("report_id is present in JSON response", func(t *testing.T) {
		rawBody := s.DoRaw("GET", url, nil, http.StatusOK)
		var raw map[string]any
		require.NoError(t, json.NewDecoder(rawBody.Body).Decode(&raw))

		reports, ok := raw["reports"].([]any)
		require.True(t, ok)
		require.NotEmpty(t, reports)

		firstReport, ok := reports[0].(map[string]any)
		require.True(t, ok)

		_, hasReportID := firstReport["report_id"]
		assert.True(t, hasReportID, "expected key 'report_id' in response")
	})

	t.Run("free_tier_hides_include_all_reports", func(t *testing.T) {
		labelA, err := s.ds.NewLabel(ctx, &fleet.Label{Name: t.Name() + "-A", Query: "SELECT 1"})
		require.NoError(t, err)
		labelB, err := s.ds.NewLabel(ctx, &fleet.Label{Name: t.Name() + "-B", Query: "SELECT 1"})
		require.NoError(t, err)

		qIncludeAll, err := s.ds.NewQuery(ctx, &fleet.Query{
			Name:        t.Name() + "_include_all",
			Query:       "SELECT 1",
			AuthorID:    &admin.ID,
			Saved:       true,
			DiscardData: false,
			Logging:     fleet.LoggingSnapshot,
			LabelsIncludeAll: []fleet.LabelIdent{
				{LabelName: labelA.Name},
				{LabelName: labelB.Name},
			},
		})
		require.NoError(t, err)

		mkHost := func(suffix string) *fleet.Host {
			h, err := s.ds.NewHost(ctx, &fleet.Host{
				DetailUpdatedAt: time.Now(),
				LabelUpdatedAt:  time.Now(),
				PolicyUpdatedAt: time.Now(),
				SeenTime:        time.Now(),
				OsqueryHostID:   new(t.Name() + suffix),
				NodeKey:         new(t.Name() + suffix),
				UUID:            uuid.New().String(),
				Hostname:        t.Name() + "-" + suffix + ".local",
				Platform:        "linux",
			})
			require.NoError(t, err)
			return h
		}
		hostNone := mkHost("none")
		hostOnlyA := mkHost("onlyA")
		hostBoth := mkHost("both")

		require.NoError(t, s.ds.RecordLabelQueryExecutions(ctx, hostOnlyA, map[uint]*bool{labelA.ID: new(true)}, time.Now(), false))
		require.NoError(t, s.ds.RecordLabelQueryExecutions(ctx, hostBoth, map[uint]*bool{labelA.ID: new(true), labelB.ID: new(true)}, time.Now(), false))

		hasIncludeAllReport := func(hostID uint) bool {
			t.Helper()
			var resp listHostReportsResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/reports", hostID), nil, http.StatusOK, &resp, "include_reports_dont_store_results", "true")
			for _, r := range resp.Reports {
				if r.ReportID == qIncludeAll.ID {
					return true
				}
			}
			return false
		}

		// Free tier must hide include_all queries entirely from /hosts/:id/reports —
		// labels_include_all is premium-only and pre-existing rows (e.g., from a
		// tier downgrade) must not surface in the reports list, regardless of
		// label membership. Premium-tier behavior is exercised in the
		// enterprise integration test.
		assert.False(t, hasIncludeAllReport(hostNone.ID), "free tier must not surface include_all queries (no labels)")
		assert.False(t, hasIncludeAllReport(hostOnlyA.ID), "free tier must not surface include_all queries (subset of labels)")
		assert.False(t, hasIncludeAllReport(hostBoth.ID), "free tier must not surface include_all queries (all labels)")
	})
}
