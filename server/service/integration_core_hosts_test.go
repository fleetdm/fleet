package service

// Host lifecycle and host listing tests for the core (no-license) suite.
//
// Belongs here: listing, counting, searching and filtering hosts; bulk delete;
// fleet assignment; the hosts summary; get-by-identifier; host detail staleness
// bookkeeping; human-device mapping (including IdP-sourced); device tokens and the
// device page URL; reuse of the same host row on re-enrollment; and the software
// listed on a host's detail.
//
// Does not belong here: reads of a host's reported state such as vitals, health,
// activities or the report CSV (integration_core_hosts_reports_test.go), label
// membership (integration_core_labels_test.go), and the osquery/orbit enrollment
// protocol itself (integration_core_osquery_test.go, integration_core_orbit_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestBulkDeleteHostsFromTeam() {
	t := s.T()

	hosts := s.createHosts(t)

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	p, err := s.ds.NewPack(context.Background(), &fleet.Pack{
		Name: t.Name(),
		Hosts: []fleet.Target{
			{
				Type:     fleet.TargetHost,
				TargetID: hosts[0].ID,
			},
		},
	})
	require.NoError(t, err)

	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team1.ID, []uint{hosts[0].ID})))

	req := deleteHostsRequest{
		Filters: &map[string]any{"team_id": float64(team1.ID)},
	}
	resp := deleteHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusOK, &resp)

	_, err = s.ds.Host(context.Background(), hosts[0].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[1].ID)
	require.NoError(t, err)
	_, err = s.ds.Host(context.Background(), hosts[2].ID)
	require.NoError(t, err)

	err = s.ds.DeleteHosts(context.Background(), []uint{hosts[1].ID, hosts[2].ID})
	require.NoError(t, err)

	newP, err := s.ds.Pack(context.Background(), p.ID)
	require.NoError(t, err)
	require.Empty(t, newP.Hosts)
	require.NoError(t, s.ds.DeletePack(context.Background(), newP.Name))
}

func (s *integrationTestSuite) TestBulkDeleteHostsInLabel() {
	t := s.T()

	hosts := s.createHosts(t)

	label := &fleet.Label{
		Name:  "foo",
		Query: "select * from foo;",
	}
	label, err := s.ds.NewLabel(context.Background(), label)
	require.NoError(t, err)

	require.NoError(t, s.ds.RecordLabelQueryExecutions(context.Background(), hosts[1], map[uint]*bool{label.ID: new(true)}, time.Now(), false))
	require.NoError(t, s.ds.RecordLabelQueryExecutions(context.Background(), hosts[2], map[uint]*bool{label.ID: new(true)}, time.Now(), false))

	req := deleteHostsRequest{
		Filters: &map[string]any{"label_id": float64(label.ID)},
	}
	resp := deleteHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusOK, &resp)

	_, err = s.ds.Host(context.Background(), hosts[0].ID)
	require.NoError(t, err)
	_, err = s.ds.Host(context.Background(), hosts[1].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[2].ID)
	require.Error(t, err)

	err = s.ds.DeleteHosts(context.Background(), []uint{hosts[0].ID})
	require.NoError(t, err)
}

func (s *integrationTestSuite) TestBulkDeleteHostByIDs() {
	t := s.T()

	hosts := s.createHosts(t)

	req := deleteHostsRequest{
		IDs: []uint{hosts[0].ID, hosts[1].ID},
	}
	resp := deleteHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusOK, &resp)

	_, err := s.ds.Host(context.Background(), hosts[0].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[1].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[2].ID)
	require.NoError(t, err)

	err = s.ds.DeleteHosts(context.Background(), []uint{hosts[2].ID})
	require.NoError(t, err)
}

func (s *integrationTestSuite) TestBulkDeleteHostByIDsWithTimeout() {
	t := s.T()

	hosts := s.createHosts(t, "debian")

	req := deleteHostsRequest{
		IDs: []uint{hosts[0].ID},
	}
	resp := deleteHostsResponse{}
	originalTimeout := deleteHostsTimeout
	deleteHostsTimeout = 0
	deleteHostsSkipAuthorization = true
	defer func() {
		deleteHostsTimeout = originalTimeout
		deleteHostsSkipAuthorization = false
	}()
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusAccepted, &resp)

	// Make sure the host was actually deleted.
	deleteDone := make(chan bool)
	go func() {
		for {
			_, err := s.ds.Host(context.Background(), hosts[0].ID)
			if err != nil {
				deleteDone <- true
				break
			}
		}
	}()
	select {
	case <-deleteDone:
		return
	case <-time.After(2 * time.Second):
		t.Log("http.StatusAccepted (202) means that delete should continue in the background, but we did not see the host deleted after 2 seconds.")
		t.Error("Timeout: delete did not occur.")
	}
}

func (s *integrationTestSuite) TestBulkDeleteHostsAll() {
	t := s.T()

	hosts := s.createHosts(t)

	// All hosts should be deleted when an empty filter is specified
	req := deleteHostsRequest{
		Filters: &map[string]any{},
	}
	resp := deleteHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusOK, &resp)

	_, err := s.ds.Host(context.Background(), hosts[0].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[1].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[2].ID)
	require.Error(t, err)
}

func (s *integrationTestSuite) TestBulkDeleteHostsErrors() {
	t := s.T()

	hosts := s.createHosts(t)

	req := deleteHostsRequest{
		IDs:     []uint{hosts[0].ID, hosts[1].ID},
		Filters: &map[string]any{"label_id": float64(1)},
	}
	resp := deleteHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusBadRequest, &resp)

	req = deleteHostsRequest{}
	// No ids or filter specified
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusBadRequest, &resp)
}

func (s *integrationTestSuite) TestHostsCount() {
	t := s.T()

	hosts := s.createHosts(t, "darwin", "darwin", "darwin")

	// set disk space information for some hosts
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[0].ID, 10.0, 2.0, 500.0, nil))  // low disk
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[1].ID, 40.0, 4.0, 1000.0, nil)) // not low disk

	label := &fleet.Label{
		Name:  t.Name() + "foo",
		Query: "select * from foo;",
	}
	label, err := s.ds.NewLabel(context.Background(), label)
	require.NoError(t, err)

	require.NoError(t, s.ds.RecordLabelQueryExecutions(context.Background(), hosts[0], map[uint]*bool{label.ID: new(true)}, time.Now(), false))

	req := countHostsRequest{}
	resp := countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"additional_info_filters", "*",
	)
	assert.Equal(t, 3, resp.Count)

	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"additional_info_filters", "*",
		"label_id", fmt.Sprint(label.ID),
	)
	assert.Equal(t, 1, resp.Count)

	// there are 3 hosts, whos names end with ...local0, ...local1, ...local2
	// query by host name

	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"query", "local0",
	)
	assert.Equal(t, 1, resp.Count)

	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"query", "local",
	)
	assert.Equal(t, 3, resp.Count)

	// query by host name with leading/trailing whitespace
	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"query", " local0  ",
	)
	assert.Equal(t, 1, resp.Count)

	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"query", " local  ",
	)
	assert.Equal(t, 3, resp.Count)

	// query by host name leading/trailing whitespace and label
	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"label_id", fmt.Sprint(label.ID),
		"query", "   local0	",
	)
	assert.Equal(t, 1, resp.Count)

	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"label_id", fmt.Sprint(label.ID),
		// only host 0 has the label
		"query", "   local1	",
	)
	assert.Equal(t, 0, resp.Count)

	// filter by low_disk_space criteria is ignored (premium-only filter)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "low_disk_space", "32")
	require.Equal(t, len(hosts), resp.Count)
	// but it is still validated for a correct value when provided (as that happens in a middleware before the handler)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &resp, "low_disk_space", "123456")

	// filter by MDM criteria without any host having such information
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_id", fmt.Sprint(999))
	require.Equal(t, 0, resp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_enrollment_status", "manual")
	require.Equal(t, 0, resp.Count)

	// set MDM information on a host
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), hosts[1].ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "", false))
	// also create server with MDM information, which is ignored.
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), hosts[2].ID, true, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "", false))
	var mdmID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &mdmID,
			`SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`, fleet.WellKnownMDMSimpleMDM, "https://simplemdm.com")
	})

	// set MDM information for another host installed from DEP and pending enrollment to Fleet MDM
	pendingMDMHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		Platform:       "darwin",
		HardwareSerial: "532141num832",
		HardwareModel:  "MacBook Pro",
	})
	require.NoError(t, err)
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), pendingMDMHost.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, "", false))

	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_id", fmt.Sprint(mdmID))
	require.Equal(t, 1, resp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_enrollment_status", "manual")
	require.Equal(t, 1, resp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_enrollment_status", "automatic")
	require.Equal(t, 0, resp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_enrollment_status", "unenrolled")
	require.Equal(t, 0, resp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_enrollment_status", "manual", "mdm_id", fmt.Sprint(mdmID))
	require.Equal(t, 1, resp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_enrollment_status", "pending")
	require.Equal(t, 1, resp.Count)

	// get the host's MDM info
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", pendingMDMHost.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, pendingMDMHost.ID, hostResp.Host.ID)
	require.Equal(t, "Pending", *hostResp.Host.MDM.EnrollmentStatus)
	require.Equal(t, "https://fleetdm.com", *hostResp.Host.MDM.ServerURL)

	// no macos_settings is returned when MDM is not configured
	require.Nil(t, hostResp.Host.MDM.MacOSSettings)
}

func (s *integrationTestSuite) TestListHosts() {
	t := s.T()

	hosts := s.createHosts(t, "darwin", "darwin", "darwin")

	// set disk space information for some hosts
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[0].ID, 10.0, 2.0, 500.0, nil))  // low disk
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[1].ID, 40.0, 4.0, 1000.0, nil)) // not low disk

	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)
	require.Len(t, resp.Hosts, len(hosts))
	for _, h := range resp.Hosts {
		switch h.ID {
		case hosts[0].ID:
			assert.InDelta(t, 10.0, h.GigsDiskSpaceAvailable, 0.001)
			assert.InDelta(t, 2.0, h.PercentDiskSpaceAvailable, 0.001)
		case hosts[1].ID:
			assert.InDelta(t, 40.0, h.GigsDiskSpaceAvailable, 0.001)
			assert.InDelta(t, 4.0, h.PercentDiskSpaceAvailable, 0.001)
		}
		assert.Equal(t, h.SoftwareUpdatedAt, h.CreatedAt)
	}

	// setting the low_disk_space criteria is ignored (premium-only)
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "low_disk_space", "32")
	require.Len(t, resp.Hosts, len(hosts))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "per_page", "1")
	require.Len(t, resp.Hosts, 1)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "order_key", "id", "after", fmt.Sprint(hosts[1].ID))
	require.Len(t, resp.Hosts, len(hosts)-2)

	// invalid order_key returns 422
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusUnprocessableEntity, &resp, "order_key", "invalid_column")

	time.Sleep(1 * time.Second)

	// create some software for various hosts
	host2 := hosts[2]
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err := s.ds.UpdateHostSoftware(context.Background(), host2.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host2, false))

	host1 := hosts[1]
	software = []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.1.0", Source: "application"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host1.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host1, false))

	host0 := hosts[0]
	software = []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.2.0", Source: "not_application"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host0.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host0, false))

	err = s.ds.SyncHostsSoftware(context.Background(), time.Now())
	require.NoError(t, err)
	err = s.ds.SyncHostsSoftwareTitles(context.Background(), time.Now())
	require.NoError(t, err)

	var fooV1ID, fooV2ID, barAppTitleID, fooTitleID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		err := sqlx.GetContext(context.Background(), q, &fooV1ID,
			`SELECT id FROM software WHERE name = ? AND source = ? AND version = ?`, "foo", "chrome_extensions", "0.0.1")
		if err != nil {
			return err
		}
		err = sqlx.GetContext(context.Background(), q, &fooV2ID,
			`SELECT id FROM software WHERE name = ? AND source = ? AND version = ?`, "foo", "chrome_extensions", "0.0.2")
		if err != nil {
			return err
		}
		err = sqlx.GetContext(context.Background(), q, &barAppTitleID,
			`SELECT id FROM software_titles WHERE name = ? AND source = ?`, "bar", "application")
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

	// foo v0.0.1 is only installed on host2
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_id", fmt.Sprint(fooV1ID))
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, host2.ID, resp.Hosts[0].ID)
	assert.Equal(t, "foo", resp.Software.Name)
	assert.Greater(t, resp.Hosts[0].SoftwareUpdatedAt, resp.Hosts[0].CreatedAt)
	assert.Nil(t, resp.SoftwareTitle)

	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_id", fmt.Sprint(fooV1ID))
	require.Equal(t, 1, countResp.Count)

	// foo v0.0.2 is installed on hosts 0 and 1
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_version_id", fmt.Sprint(fooV2ID))
	require.Len(t, resp.Hosts, 2)
	require.ElementsMatch(t, []uint{host0.ID, host1.ID}, []uint{resp.Hosts[0].ID, resp.Hosts[1].ID})

	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_version_id", fmt.Sprint(fooV2ID))
	require.Equal(t, 2, countResp.Count)

	// bar/application title is only on host1
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_title_id", fmt.Sprint(barAppTitleID))
	require.Len(t, resp.Hosts, 1)
	require.ElementsMatch(t, []uint{host1.ID}, []uint{resp.Hosts[0].ID})
	assert.Equal(t, "bar", resp.SoftwareTitle.Name)
	assert.Equal(t, "application", resp.SoftwareTitle.Source)
	assert.Equal(t, uint(1), resp.SoftwareTitle.HostsCount)
	require.Len(t, resp.SoftwareTitle.Versions, 1)
	assert.Equal(t, "0.1.0", resp.SoftwareTitle.Versions[0].Version)
	assert.Nil(t, resp.Software)

	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_title_id", fmt.Sprint(barAppTitleID))
	require.Equal(t, 1, countResp.Count)

	// foo title is on all 3 hosts
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_title_id", fmt.Sprint(fooTitleID))
	require.Len(t, resp.Hosts, 3)
	require.ElementsMatch(t, []uint{host0.ID, host1.ID, host2.ID}, []uint{resp.Hosts[0].ID, resp.Hosts[1].ID, resp.Hosts[2].ID})

	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_title_id", fmt.Sprint(fooTitleID))
	require.Equal(t, 3, countResp.Count)

	// verify invalid combinations of software filters
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, &resp, "software_title_id", fmt.Sprint(fooTitleID), "software_id", fmt.Sprint(fooV1ID))
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, &resp, "software_title_id", fmt.Sprint(fooTitleID), "software_version_id", fmt.Sprint(fooV1ID))
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, &resp, "software_id", fmt.Sprint(fooV1ID), "software_version_id", fmt.Sprint(fooV1ID))
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, &resp, "software_id", fmt.Sprint(fooV1ID), "software_version_id", fmt.Sprint(fooV1ID), "software_title_id", fmt.Sprint(fooTitleID))
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &countResp, "software_title_id", fmt.Sprint(fooTitleID), "software_id", fmt.Sprint(fooV1ID))
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &countResp, "software_title_id", fmt.Sprint(fooTitleID), "software_version_id", fmt.Sprint(fooV1ID))
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &countResp, "software_id", fmt.Sprint(fooV1ID), "software_version_id", fmt.Sprint(fooV1ID))
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &countResp, "software_id", fmt.Sprint(fooV1ID), "software_version_id", fmt.Sprint(fooV1ID), "software_title_id", fmt.Sprint(fooTitleID))

	user1 := test.NewUser(t, s.ds, "Alice", "alice@example.com", true)
	q := test.NewQuery(t, s.ds, nil, "query1", "select 1", 0, true)
	defer s.cleanupQuery(q.ID)
	globalPolicy0, err := s.ds.NewGlobalPolicy(
		context.Background(), &user1.ID, fleet.PolicyPayload{
			QueryID: &q.ID,
		},
	)
	require.NoError(t, err)

	require.NoError(
		t,
		errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{globalPolicy0.ID: new(false)}, time.Now(), false, nil)),
	)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_id", fmt.Sprint(fooV1ID))
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, uint64(1), resp.Hosts[0].HostIssues.FailingPoliciesCount)
	assert.Equal(t, uint64(1), resp.Hosts[0].HostIssues.TotalIssuesCount)
	assert.Nil(t, resp.Hosts[0].HostIssues.CriticalVulnerabilitiesCount)

	resp = listHostsResponse{}
	// disable_failing_policies has been deprecated and is no longer documented; it is an alias for disable_issues
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_version_id", fmt.Sprint(fooV1ID), "disable_failing_policies", "true")
	require.Len(t, resp.Hosts, 1)
	assert.Zero(t, resp.Hosts[0].HostIssues.FailingPoliciesCount)
	assert.Zero(t, resp.Hosts[0].HostIssues.TotalIssuesCount)
	assert.Nil(t, resp.Hosts[0].HostIssues.CriticalVulnerabilitiesCount)

	resp = listHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_version_id", fmt.Sprint(fooV1ID), "disable_issues", "true",
	)
	require.Len(t, resp.Hosts, 1)
	assert.Zero(t, resp.Hosts[0].HostIssues.FailingPoliciesCount)
	assert.Zero(t, resp.Hosts[0].HostIssues.TotalIssuesCount)
	assert.Nil(t, resp.Hosts[0].HostIssues.CriticalVulnerabilitiesCount)

	// filter by MDM criteria without any host having such information
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_id", fmt.Sprint(999))
	require.Empty(t, resp.Hosts)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_enrollment_status", "manual")
	require.Empty(t, resp.Hosts)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)
	// and same by munki issue id
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "munki_issue_id", fmt.Sprint(999))
	require.Empty(t, resp.Hosts)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)

	// set MDM information on a host
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), host2.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "", false))
	var mdmID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &mdmID,
			`SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`, fleet.WellKnownMDMSimpleMDM, "https://simplemdm.com")
	})

	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)

	// set MDM information for another host installed from DEP and pending enrollment to Fleet MDM
	pendingMDMHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		Platform:       "darwin",
		HardwareSerial: "532141num832",
		HardwareModel:  "MacBook Pro",
	})
	require.NoError(t, err)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(), "INSERT INTO mobile_device_management_solutions (name, server_url) VALUES ('https://fleetdm.com', 'Fleet')")
		require.NoError(t, err)
		return err
	})
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), pendingMDMHost.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, "", false))

	// generate aggregated stats
	require.NoError(t, s.ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_enrollment_status", "pending")
	require.Len(t, resp.Hosts, 1)
	require.Equal(t, "532141num832", resp.Hosts[0].HardwareSerial)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MunkiIssue)
	require.Nil(t, resp.MDMSolution) // MDM solution is included only if `mdm_id` query param is specified`

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_enrollment_status", "manual")
	require.Len(t, resp.Hosts, 1)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_enrollment_status", "automatic")
	require.Empty(t, resp.Hosts)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_enrollment_status", "unenrolled")
	require.Empty(t, resp.Hosts)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_id", fmt.Sprint(mdmID))
	require.Len(t, resp.Hosts, 1)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MunkiIssue)
	require.NotNil(t, resp.MDMSolution)
	assert.Equal(t, mdmID, resp.MDMSolution.ID)
	assert.Equal(t, fleet.WellKnownMDMSimpleMDM, resp.MDMSolution.Name)
	assert.Equal(t, "https://simplemdm.com", resp.MDMSolution.ServerURL)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_id", fmt.Sprint(mdmID), "mdm_enrollment_status", "manual")
	require.Len(t, resp.Hosts, 1)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MunkiIssue)
	assert.NotNil(t, resp.MDMSolution)
	assert.Equal(t, mdmID, resp.MDMSolution.ID)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, &resp, "mdm_enrollment_status", "invalid-status")

	// Filter by inexistent software.
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_id", fmt.Sprint(9999))
	require.Empty(t, resp.Hosts)
	assert.Nil(t, resp.Software)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_version_id", fmt.Sprint(9999))
	require.Empty(t, resp.Hosts)
	assert.Nil(t, resp.Software)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_title_id", fmt.Sprint(9999))
	require.Empty(t, resp.Hosts)
	assert.Nil(t, resp.SoftwareTitle)

	// Filter by non-existent team.
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, &resp, "team_id", fmt.Sprint(9999))

	// set munki information on a host
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(context.Background(), host2.ID, "1.2.3", []string{"err"}, []string{"warn"}))
	var errMunkiID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &errMunkiID,
			`SELECT id FROM munki_issues WHERE name = 'err' AND issue_type = 'error'`)
	})
	// generate aggregated stats
	require.NoError(t, s.ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "munki_issue_id", fmt.Sprint(errMunkiID))
	require.Len(t, resp.Hosts, 1)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	require.NotNil(t, resp.MunkiIssue)
	assert.Equal(t, fleet.MunkiIssue{
		ID:        errMunkiID,
		Name:      "err",
		IssueType: "error",
	}, *resp.MunkiIssue)

	// filters can be combined, no problem
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "munki_issue_id", fmt.Sprint(errMunkiID), "mdm_id", fmt.Sprint(mdmID))
	require.Len(t, resp.Hosts, 1)
	assert.Nil(t, resp.Software)
	assert.NotNil(t, resp.MDMSolution)
	assert.NotNil(t, resp.MunkiIssue)

	// set operating system information on a host
	testOS := fleet.OperatingSystem{Name: "fooOS", Version: "4.2", Arch: "64bit", KernelVersion: "13.37", Platform: "bar"}
	require.NoError(t, s.ds.UpdateHostOperatingSystem(context.Background(), host2.ID, testOS))
	var osID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &osID,
			`SELECT id FROM operating_systems WHERE name = ? AND version = ?`, "fooOS", "4.2")
	})
	require.Positive(t, osID)

	// generate aggregated stats
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_name", testOS.Name, "os_version", testOS.Version)
	require.Len(t, resp.Hosts, 1)

	expected := resp.Hosts[0]
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_id", fmt.Sprintf("%d", osID))
	require.Len(t, resp.Hosts, 1)
	require.Equal(t, expected, resp.Hosts[0])

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_name", "unknownOS", "os_version", "4.2")
	require.Empty(t, resp.Hosts)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_id", fmt.Sprintf("%d", osID+1337))
	require.Empty(t, resp.Hosts)

	// populate software for hosts
	now := time.Now()

	inserted, err := s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: host2.Software[0].ID,
		CVE:        "cve-123-123-123",
	}, fleet.NVDSource)
	require.NoError(t, err)
	require.True(t, inserted)

	require.NoError(t, s.ds.InsertCVEMeta(context.Background(), []fleet.CVEMeta{{
		CVE:              "cve-123-123-123",
		CVSSScore:        new(5.4),
		EPSSProbability:  new(0.5),
		CISAKnownExploit: new(true),
		Published:        &now,
		Description:      "a long description of the cve",
	}}))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_software", "true")
	require.Len(t, resp.Hosts, 4)
	for _, h := range resp.Hosts {
		if h.ID == hosts[2].ID {
			require.NotEmpty(t, h.Software)
			require.Len(t, h.Software, 1)
			require.NotEmpty(t, h.Software[0].Vulnerabilities)

			// all these should be nil because this isn't Premium
			require.Nil(t, h.Software[0].Vulnerabilities[0].CVSSScore)
			require.Nil(t, h.Software[0].Vulnerabilities[0].EPSSProbability)
			require.Nil(t, h.Software[0].Vulnerabilities[0].CISAKnownExploit)
			require.Nil(t, h.Software[0].Vulnerabilities[0].CVEPublished)
			require.Nil(t, h.Software[0].Vulnerabilities[0].Description)
			require.Nil(t, h.Software[0].Vulnerabilities[0].ResolvedInVersion)
		}
		assert.Nil(t, h.Policies)
	}

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_software", "false", "populate_policies", "false")
	require.Len(t, resp.Hosts, 4)
	for _, h := range resp.Hosts {
		require.Empty(t, h.Software)
		assert.Nil(t, h.Policies)
	}

	// Populate policies for hosts. One policy was created earlier.
	ctx := context.Background()
	globalPolicy1, err := s.ds.NewGlobalPolicy(
		ctx, &test.UserAdmin.ID, fleet.PolicyPayload{
			Name:  "foobar0",
			Query: "SELECT 0;",
		},
	)
	require.NoError(t, err)

	for _, host := range hosts {
		// All hosts pass the globalPolicy1
		_, err := s.ds.RecordPolicyQueryExecutions(
			context.Background(), host, map[uint]*bool{globalPolicy1.ID: new(true)}, time.Now(), false, nil,
		)
		require.NoError(t, err)
	}

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_policies", "true")
	require.Len(t, resp.Hosts, len(hosts)+1) // +1 for the pending MDM host
	for _, h := range resp.Hosts {
		if h.ID == hosts[0].ID {
			policies := *h.Policies
			require.Len(t, policies, 2)
			assert.Equal(t, globalPolicy0.Name, policies[0].Name)
			assert.Empty(t, policies[0].Response)
			assert.Equal(t, globalPolicy1.Name, policies[1].Name)
			assert.Equal(t, "pass", policies[1].Response)
		} else if h.ID == hosts[2].ID {
			policies := *h.Policies
			require.Len(t, policies, 2)
			assert.Equal(t, globalPolicy0.Name, policies[0].Name)
			assert.Equal(t, "fail", policies[0].Response)
			assert.Equal(t, globalPolicy1.Name, policies[1].Name)
			assert.Equal(t, "pass", policies[1].Response)
		}
	}

	// there are 3 hosts, whos names end with ...local0, ...local1, ...local2
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "query", "local0")
	require.Len(t, resp.Hosts, 1)
	require.Contains(t, resp.Hosts[0].Hostname, "local0")
	resp = listHostsResponse{}
	// now with leading/trailing whitespace
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "query", " local0 ")
	require.Len(t, resp.Hosts, 1)
	require.Contains(t, resp.Hosts[0].Hostname, "local0")

	// Add users to hosts
	users := []fleet.HostUser{
		{
			Uid:       1,
			Username:  "root",
			Type:      "local",
			GroupName: "root",
			Shell:     "/bin/sh",
		},
		{
			Uid:       1001,
			Username:  "username",
			Type:      "local",
			GroupName: "usergroup",
			Shell:     "/bin/sh",
		},
	}
	err = s.ds.SaveHostUsers(ctx, host0.ID, users)
	require.NoError(t, err)

	// Add labels to host
	label1, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "First Label"})
	require.NoError(t, err)
	label2, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "Second Label"})
	require.NoError(t, err)

	err = s.ds.AddLabelsToHost(ctx, host0.ID, []uint{label1.ID, label2.ID})
	require.NoError(t, err)

	// Without "populate_users" and "populate_labels" query params, no users or labels
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "query", "local0")
	require.Len(t, resp.Hosts, 1)
	require.Contains(t, resp.Hosts[0].Hostname, "local0")
	require.Empty(t, resp.Hosts[0].Users)
	require.Empty(t, resp.Hosts[0].Labels)

	// With "populate_users" query param
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "query", "local0", "populate_users", "true")
	require.Len(t, resp.Hosts, 1)
	require.Contains(t, resp.Hosts[0].Hostname, "local0")
	require.Len(t, resp.Hosts[0].Users, 2)
	require.Equal(t, resp.Hosts[0].Users[0], users[0])
	require.Equal(t, resp.Hosts[0].Users[1], users[1])

	// With "populate_labels" query param
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "query", "local0", "populate_labels", "true")
	require.Len(t, resp.Hosts, 1)
	require.Contains(t, resp.Hosts[0].Hostname, "local0")
	require.Len(t, resp.Hosts[0].Labels, 2)
	require.Equal(t, label1.Name, resp.Hosts[0].Labels[0].Name)
	require.Equal(t, label2.Name, resp.Hosts[0].Labels[1].Name)

	// With "include_device_status" query param
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "query", "local0", "include_device_status", "true")
	require.Len(t, resp.Hosts, 1)
	require.Contains(t, resp.Hosts[0].Hostname, "local0")
	require.Equal(t, string(fleet.DeviceStatusUnlocked), *resp.Hosts[0].MDM.DeviceStatus)
	require.Equal(t, string(fleet.PendingActionNone), *resp.Hosts[0].MDM.PendingAction)
}

func (s *integrationTestSuite) TestListHostsPopulateSoftwareWithInstalledPaths() {
	t := s.T()
	ctx := context.Background()

	// Create a host for this test
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "1"),
		OsqueryHostID:   new(t.Name() + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.10",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	// Create software with installed paths and signature information
	software := []fleet.Software{
		{
			Name:             "Google Chrome.app",
			Version:          "121.0.6167.160",
			Source:           "chrome_extensions",
			ExtensionID:      "test-extension-id",
			ExtensionFor:     "chrome",
			BundleIdentifier: "com.google.Chrome",
		},
	}
	hostSoftware, err := s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.Len(t, hostSoftware.CurrInstalled(), 1)

	// Add installed paths and signature information
	swPaths := map[string]struct{}{}
	testCdHash := "abc123hash"
	testExecHash := "def456hash"
	testExecPath := "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
	for _, s := range software {
		pathItems := [][5]string{
			{"/Applications/Google Chrome.app", "EQHXZ8M8AV", testCdHash, testExecHash, testExecPath},
			{"/Users/test/Applications/Google Chrome.app", "", "", "", ""},
		}
		for _, pathItem := range pathItems {
			path := pathItem[0]
			teamIdentifier := pathItem[1]
			cdHash := pathItem[2]
			eHash := pathItem[3]
			ePath := pathItem[4]
			key := fmt.Sprintf(
				"%s%s%s%s%s%s%s%s%s%s%s",
				path, fleet.SoftwareFieldSeparator, teamIdentifier, fleet.SoftwareFieldSeparator, cdHash, fleet.SoftwareFieldSeparator, eHash, fleet.SoftwareFieldSeparator, ePath, fleet.SoftwareFieldSeparator, s.ToUniqueStr(),
			)
			swPaths[key] = struct{}{}
		}
	}
	err = s.ds.UpdateHostSoftwareInstalledPaths(ctx, host.ID, swPaths, hostSoftware)
	require.NoError(t, err)

	// Sync to ensure counts are updated
	err = s.ds.SyncHostsSoftware(ctx, time.Now().UTC())
	require.NoError(t, err)

	// Test: GET /api/latest/fleet/hosts with populate_software=true
	var listResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "populate_software", "true")

	// Find our test host in the response
	var testHost *fleet.HostResponse
	for i, h := range listResp.Hosts {
		if h.ID == host.ID {
			testHost = &listResp.Hosts[i]
			break
		}
	}
	require.NotNil(t, testHost, "test host not found in response")

	// Verify software is populated
	require.NotEmpty(t, testHost.Software, "software should be populated")
	require.Len(t, testHost.Software, 1, "expected 1 software entry")

	// Verify the software entry has the expected fields
	sw := testHost.Software[0]
	assert.Equal(t, "Google Chrome.app", sw.Name)
	assert.Equal(t, "121.0.6167.160", sw.Version)
	assert.Equal(t, "chrome_extensions", sw.Source)
	assert.Equal(t, "test-extension-id", sw.ExtensionID)
	assert.Equal(t, "chrome", sw.ExtensionFor)
	assert.Equal(t, "chrome", sw.Browser) // backward compatibility field
	assert.Equal(t, "com.google.Chrome", sw.BundleIdentifier)

	// Verify installed_paths is populated and not empty
	assert.NotEmpty(t, sw.InstalledPaths, "installed_paths should be populated")
	assert.Len(t, sw.InstalledPaths, 2, "expected 2 installed paths")
	assert.Contains(t, sw.InstalledPaths, "/Applications/Google Chrome.app")
	assert.Contains(t, sw.InstalledPaths, "/Users/test/Applications/Google Chrome.app")

	// Verify signature_information is populated
	assert.NotEmpty(t, sw.PathSignatureInformation, "signature_information should be populated")
	assert.Len(t, sw.PathSignatureInformation, 2, "expected 2 signature information entries")

	// Sort by installed path for consistent ordering
	sort.Slice(sw.PathSignatureInformation, func(i, j int) bool {
		return sw.PathSignatureInformation[i].InstalledPath < sw.PathSignatureInformation[j].InstalledPath
	})

	// Verify first signature information (system-level with team identifier)
	sigInfo0 := sw.PathSignatureInformation[0]
	assert.Equal(t, "/Applications/Google Chrome.app", sigInfo0.InstalledPath)
	assert.Equal(t, "EQHXZ8M8AV", sigInfo0.TeamIdentifier)
	assert.NotNil(t, sigInfo0.CDHashSHA256)
	assert.Equal(t, testCdHash, *sigInfo0.CDHashSHA256)
	assert.NotNil(t, *sigInfo0.ExecutableSHA256)
	assert.Equal(t, testExecHash, *sigInfo0.ExecutableSHA256)
	assert.NotNil(t, *sigInfo0.ExecutablePath)
	assert.Equal(t, testExecPath, *sigInfo0.ExecutablePath)

	// Verify second signature information (user-level without team identifier)
	sigInfo1 := sw.PathSignatureInformation[1]
	assert.Equal(t, "/Users/test/Applications/Google Chrome.app", sigInfo1.InstalledPath)
	assert.Empty(t, sigInfo1.TeamIdentifier)
	assert.Nil(t, sigInfo1.CDHashSHA256)
	assert.Nil(t, sigInfo1.ExecutableSHA256)
	assert.Nil(t, sigInfo1.ExecutablePath)

	// Also verify the JSON marshaling by checking the raw JSON response
	rawResp := s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, "populate_software", "true")
	defer rawResp.Body.Close()
	body, err := io.ReadAll(rawResp.Body)
	require.NoError(t, err)

	// Verify the JSON contains our expected fields
	assert.Contains(t, string(body), "installed_paths", "JSON should contain installed_paths field")
	assert.Contains(t, string(body), "signature_information", "JSON should contain signature_information field")
	assert.Contains(t, string(body), "/Applications/Google Chrome.app", "JSON should contain the installed path")
	assert.Contains(t, string(body), "EQHXZ8M8AV", "JSON should contain the team identifier")
	assert.Contains(t, string(body), "abc123hash", "JSON should contain the hash")
}

func (s *integrationTestSuite) TestListHostsPopulateEndUsers() {
	t := s.T()
	ctx := context.Background()

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "1"),
		OsqueryHostID:   new(t.Name() + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// ReplaceHostDeviceMapping rejects mixed sources, so write one source at a time.
	require.NoError(t, s.ds.ReplaceHostDeviceMapping(ctx, host.ID, []*fleet.HostDeviceMapping{
		{HostID: host.ID, Email: "anna@acme.com", Source: fleet.DeviceMappingMDMIdpAccounts},
	}, fleet.DeviceMappingMDMIdpAccounts))
	require.NoError(t, s.ds.ReplaceHostDeviceMapping(ctx, host.ID, []*fleet.HostDeviceMapping{
		{HostID: host.ID, Email: "anna@example.com", Source: "google_chrome_profiles"},
	}, "google_chrome_profiles"))

	findHost := func(resp listHostsResponse) *fleet.HostResponse {
		for i, h := range resp.Hosts {
			if h.ID == host.ID {
				return &resp.Hosts[i]
			}
		}
		return nil
	}

	// end users are omitted unless populate_end_users is set
	var listResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp)
	got := findHost(listResp)
	require.NotNil(t, got)
	require.Empty(t, got.EndUsers)

	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "populate_end_users", "true")
	got = findHost(listResp)
	require.NotNil(t, got)
	require.Len(t, got.EndUsers, 1)
	assert.Equal(t, "anna@acme.com", got.EndUsers[0].IdpUserName)
	require.Len(t, got.EndUsers[0].OtherEmails, 1)
	assert.Equal(t, "anna@example.com", got.EndUsers[0].OtherEmails[0].Email)
	assert.Equal(t, "google_chrome_profiles", got.EndUsers[0].OtherEmails[0].Source)

	s.Do("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, "populate_end_users", "foo").Body.Close()
}

func (s *integrationTestSuite) TestGetHostSummary() {
	t := s.T()
	ctx := context.Background()

	hosts := s.createHosts(t)

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)

	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team1.ID, []uint{hosts[0].ID})))

	// set disk space information for hosts [0] and [1]
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[0].ID, 1.0, 2.0, 500.0, new(600.0)))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[1].ID, 3.0, 4.0, 1000.0, new(1200.0)))

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[0].ID), nil, http.StatusOK, &getHostResp)
	assert.InDelta(t, 1.0, getHostResp.Host.GigsDiskSpaceAvailable, 0.001)
	assert.InDelta(t, 2.0, getHostResp.Host.PercentDiskSpaceAvailable, 0.001)
	assert.InDelta(t, 500.0, getHostResp.Host.GigsTotalDiskSpace, 0.001)
	assert.Equal(t, new(600.0), getHostResp.Host.GigsAllDiskSpace)

	var resp getHostSummaryResponse

	// no team filter
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp)
	require.Equal(t, resp.TotalsHostsCount, uint(len(hosts)))
	require.Nil(t, resp.LowDiskSpaceCount)
	require.Len(t, resp.Platforms, 3)
	gotPlatforms, wantPlatforms := make([]string, 3), []string{"linux", "debian", "rhel"}
	for i, p := range resp.Platforms {
		gotPlatforms[i] = p.Platform
		// each platform has a count of 1
		require.Equal(t, uint(1), p.HostsCount)
	}
	require.ElementsMatch(t, wantPlatforms, gotPlatforms)
	require.Nil(t, resp.TeamID)
	require.Equal(t, uint(3), resp.AllLinuxCount)
	assert.NotEmpty(t, resp.BuiltinLabels)
	for _, lbl := range resp.BuiltinLabels {
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
	}
	builtinsCount := len(resp.BuiltinLabels)

	// host summary builtin labels match list labels response
	var listResp fleet.ListLabelsResponse
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp)
	assert.NotEmpty(t, listResp.Labels)
	for _, lbl := range listResp.Labels {
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
	}
	assert.Equal(t, len(listResp.Labels), builtinsCount)

	// 'after' param is not supported for labels
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusBadRequest, &listResp, "order_key", "id", "after", "1")

	// ordering by host_count when include_host_counts=false is rejected
	res := s.Do("GET", "/api/latest/fleet/labels", nil, http.StatusBadRequest, "order_key", "host_count", "include_host_counts", "false")
	require.Contains(t, extractServerErrorText(res.Body), "Invalid order_key (host_count cannot be ordered when they are disabled)")

	// ordering by host_count with include_host_counts=true is allowed
	listResp = fleet.ListLabelsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "order_key", "host_count", "include_host_counts", "true")

	// ordering by host_count without include_host_counts (default true) is allowed
	listResp = fleet.ListLabelsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "order_key", "host_count")

	// include_host_counts=false with a different order_key is allowed
	listResp = fleet.ListLabelsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "order_key", "name", "include_host_counts", "false")

	// team filter, no host
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp, "team_id", fmt.Sprint(team2.ID))
	require.Equal(t, uint(0), resp.TotalsHostsCount)
	require.Empty(t, resp.Platforms)
	require.Equal(t, uint(0), resp.AllLinuxCount)
	require.Equal(t, team2.ID, *resp.TeamID)

	// team filter, one host, low_disk_count is ignored as not premium
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp, "team_id", fmt.Sprint(team1.ID), "low_disk_space", "2")
	require.Equal(t, uint(1), resp.TotalsHostsCount)
	require.Nil(t, resp.LowDiskSpaceCount)
	require.Len(t, resp.Platforms, 1)
	require.Equal(t, "debian", resp.Platforms[0].Platform)
	require.Equal(t, uint(1), resp.Platforms[0].HostsCount)
	require.Equal(t, uint(1), resp.AllLinuxCount)
	require.Equal(t, team1.ID, *resp.TeamID)

	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp, "team_id", fmt.Sprint(team1.ID), "platform", "linux")
	require.Equal(t, uint(1), resp.TotalsHostsCount)
	require.Equal(t, "debian", resp.Platforms[0].Platform)
	require.Equal(t, uint(1), resp.AllLinuxCount)

	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp, "platform", "rhel")
	require.Equal(t, uint(1), resp.TotalsHostsCount)
	require.Equal(t, "rhel", resp.Platforms[0].Platform)
	require.Equal(t, uint(1), resp.AllLinuxCount)

	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp, "platform", "linux")
	require.Equal(t, uint(3), resp.TotalsHostsCount)
	require.Equal(t, uint(3), resp.AllLinuxCount)
	require.Len(t, resp.Platforms, 3)
	for i, p := range resp.Platforms {
		gotPlatforms[i] = p.Platform
		// each platform has a count of 1
		require.Equal(t, uint(1), p.HostsCount)
	}
	require.ElementsMatch(t, wantPlatforms, gotPlatforms)

	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp, "platform", "darwin")
	require.Equal(t, uint(0), resp.TotalsHostsCount)
	require.Equal(t, uint(0), resp.AllLinuxCount)
	require.Empty(t, resp.Platforms)

	// invalid low_disk_space value is still validated and results in error
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusBadRequest, &resp, "low_disk_space", "1234")
}

func (s *integrationTestSuite) TestHostDetailsUpdatesStaleHostIssues() {
	t := s.T()
	ctx := context.Background()

	// create host
	hosts := s.createHosts(t, "linux")
	host := hosts[0]

	staleIssuesCount, freshIssueCount := uint64(500), uint64(0)
	// create host_issues for it with stale data
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO host_issues (host_id, total_issues_count) VALUES (?, ?)`, host.ID, staleIssuesCount)
		return err
	})

	// hit endpoint: host issues should still be stale, since last updated was less than a minute ago
	hostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)

	require.Equal(t, hostResp.Host.HostIssues.TotalIssuesCount, staleIssuesCount)

	// set updated_at to longer than minute ago
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`UPDATE host_issues SET updated_at = ? WHERE host_id = ?`, time.Now().Add(-2*time.Minute), host.ID)
		return err
	})
	// hit endpoint: should have been updated this time
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, hostResp.Host.HostIssues.TotalIssuesCount, freshIssueCount)
}

func (s *integrationTestSuite) TestHostsAddToTeam() {
	t := s.T()

	ctx := context.Background()

	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: uuid.New().String(),
	})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: uuid.New().String(),
	})
	require.NoError(t, err)

	hosts := s.createHosts(t)
	var refetchResp refetchHostResponse
	// refetch existing
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", hosts[0].ID), nil, http.StatusOK, &refetchResp)
	require.NoError(t, refetchResp.Err)

	// refetch unknown
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", hosts[2].ID+1), nil, http.StatusNotFound, &refetchResp)

	// get by identifier unknown
	var getResp getHostResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/identifier/no-such-host", nil, http.StatusNotFound, &getResp)

	// get by identifier valid
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", hosts[0].UUID), nil, http.StatusOK, &getResp)
	require.Equal(t, hosts[0].ID, getResp.Host.ID)
	require.Nil(t, getResp.Host.TeamID)

	// assign host0 and host1 to team 1
	var addResp addHostsToTeamResponse
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &tm1.ID,
		HostIDs: []uint{hosts[0].ID, hosts[1].ID},
	}, http.StatusOK, &addResp)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeTransferredHostsToTeam{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "team_id": %d, "team_name": %q, "host_ids": [%d, %d], "host_display_names": [%q, %q]}`,
			tm1.ID, tm1.Name, tm1.ID, tm1.Name, hosts[0].ID, hosts[1].ID, hosts[0].DisplayName(), hosts[1].DisplayName()),
		0,
	)

	// transferring a mix of real and non-existent host IDs must not record the
	// fabricated IDs in the activity: only hosts that actually exist are logged.
	nonExistentHostID := hosts[2].ID + 1000
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &tm1.ID,
		HostIDs: []uint{hosts[0].ID, nonExistentHostID},
	}, http.StatusOK, &addResp)
	mixedActivityID := s.lastActivityOfTypeMatches(
		fleet.ActivityTypeTransferredHostsToTeam{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "team_id": %d, "team_name": %q, "host_ids": [%d], "host_display_names": [%q]}`,
			tm1.ID, tm1.Name, tm1.ID, tm1.Name, hosts[0].ID, hosts[0].DisplayName()),
		0,
	)

	// transferring only non-existent host IDs must not record any activity: the
	// latest transferred_hosts activity is still the mixed transfer above.
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &tm1.ID,
		HostIDs: []uint{nonExistentHostID},
	}, http.StatusOK, &addResp)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeTransferredHostsToTeam{}.ActivityName(), "", mixedActivityID)

	// check that hosts are now part of team 1
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[0].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm1.ID, *getResp.Host.TeamID)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", hosts[0].UUID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm1.ID, *getResp.Host.TeamID)
	require.NotNil(t, getResp.Host.TeamName)
	require.Equal(t, tm1.Name, *getResp.Host.TeamName)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[1].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm1.ID, *getResp.Host.TeamID)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[2].ID), nil, http.StatusOK, &getResp)
	require.Nil(t, getResp.Host.TeamID)

	// assign host2 to team 2 with filter
	var addfResp addHostsToTeamByFilterResponse
	req := addHostsToTeamByFilterRequest{
		TeamID:  &tm2.ID,
		Filters: &map[string]any{"query": hosts[2].Hostname},
	}

	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer/filter", req, http.StatusOK, &addfResp)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeTransferredHostsToTeam{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "fleet_id": %d, "fleet_name": %q, "host_ids": [%d], "host_display_names": [%q]}`,
			tm2.ID, tm2.Name, tm2.ID, tm2.Name, hosts[2].ID, hosts[2].DisplayName()),
		0,
	)

	// check that host2 is now part of team 2
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[2].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm2.ID, *getResp.Host.TeamID)

	// get all hosts label
	lblIDs, err := s.ds.LabelIDsByName(context.Background(), []string{"All Hosts"}, fleet.TeamFilter{})
	require.NoError(t, err)
	labelID := lblIDs["All Hosts"]

	// Add label to host0
	err = s.ds.RecordLabelQueryExecutions(context.Background(), hosts[0], map[uint]*bool{labelID: new(true)}, time.Now(), false)
	require.NoError(t, err)

	// offline status filter request should not move hosts
	req = addHostsToTeamByFilterRequest{
		TeamID:  &tm2.ID,
		Filters: &map[string]any{"status": "offline", "label_id": float64(labelID)},
	}
	var hostsResp listHostsResponse
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer/filter", req, http.StatusOK, &addfResp)
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hostsResp)
	require.Len(t, hostsResp.Hosts, 3)
	require.Equal(t, tm1.ID, *hostsResp.Hosts[0].TeamID)
	require.Equal(t, tm1.ID, *hostsResp.Hosts[1].TeamID)
	require.Equal(t, tm2.ID, *hostsResp.Hosts[2].TeamID)

	// assign host0 to team 2 with filter
	req = addHostsToTeamByFilterRequest{
		TeamID:  &tm2.ID,
		Filters: &map[string]any{"status": "online", "label_id": float64(labelID)},
	}
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer/filter", req, http.StatusOK, &addfResp)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[0].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm2.ID, *getResp.Host.TeamID)

	// status filter request should not delete hosts
	dreq := deleteHostsRequest{
		Filters: &map[string]any{"status": "offline", "label_id": float64(labelID)},
	}
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer/filter", req, http.StatusOK, &dreq)
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hostsResp)
	require.Len(t, hostsResp.Hosts, 3)

	// delete host 0 with filter
	dreq = deleteHostsRequest{
		Filters: &map[string]any{"status": "online", "label_id": float64(labelID)},
	}
	var delHostsResp deleteHostsResponse
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", dreq, http.StatusOK, &delHostsResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[0].ID), nil, http.StatusNotFound, &getResp)

	// delete non-existing host
	var delResp deleteHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[2].ID+1), nil, http.StatusNotFound, &delResp)

	// assign host 1 to no team
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  nil,
		HostIDs: []uint{hosts[1].ID},
	}, http.StatusOK, &addResp)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeTransferredHostsToTeam{}.ActivityName(),
		fmt.Sprintf(`{"team_id": null, "team_name": null, "fleet_id": null, "fleet_name": null, "host_ids": [%d], "host_display_names": [%q]}`,
			hosts[1].ID, hosts[1].DisplayName()),
		0,
	)

	// list the hosts
	var listResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "per_page", "3")
	require.Len(t, listResp.Hosts, 2)
	ids := []uint{listResp.Hosts[0].ID, listResp.Hosts[1].ID}
	require.ElementsMatch(t, ids, []uint{hosts[1].ID, hosts[2].ID})
}

func (s *integrationTestSuite) TestGetHostByIdentifier() {
	t := s.T()
	ctx := context.Background()

	hosts := make([]*fleet.Host, 6)
	for i := range hosts {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			Hostname:       fmt.Sprintf("test-host%d-name", i),
			OsqueryHostID:  new(fmt.Sprintf("osquery-%d", i)),
			NodeKey:        new(fmt.Sprintf("nodekey-%d", i)),
			UUID:           fmt.Sprintf("test-uuid-%d", i),
			Platform:       "darwin",
			HardwareSerial: fmt.Sprintf("serial-%d", i),
		})
		require.NoError(t, err)
		hosts[i] = h
	}

	var resp getHostResponse
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/osquery-1", nil, http.StatusOK, &resp)
	require.Equal(t, hosts[1].ID, resp.Host.ID)

	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/serial-2", nil, http.StatusOK, &resp)
	require.Equal(t, hosts[2].ID, resp.Host.ID)

	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/nodekey-3", nil, http.StatusOK, &resp)
	require.Equal(t, hosts[3].ID, resp.Host.ID)

	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/test-uuid-4", nil, http.StatusOK, &resp)
	require.Equal(t, hosts[4].ID, resp.Host.ID)

	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/test-host5-name", nil, http.StatusOK, &resp)
	require.Equal(t, hosts[5].ID, resp.Host.ID)

	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/no-such-host", nil, http.StatusNotFound, &resp)
}

func (s *integrationTestSuite) TestHostDeviceMapping() {
	t := s.T()
	ctx := context.Background()

	orbitHost := createOrbitEnrolledHost(t, "windows", "device_mapping", s.ds)
	hosts := s.createHosts(t)

	// get host device mappings of invalid host
	var listResp listHostDeviceMappingResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[2].ID+1), nil, http.StatusNotFound, &listResp)

	// existing host but none yet
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[0].ID), nil, http.StatusOK, &listResp)
	require.Empty(t, listResp.DeviceMapping)
	var hostResp getHostResponse
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+hosts[0].UUID, nil, http.StatusOK, &hostResp)
	assert.Empty(t, hostResp.Host.EndUsers)
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hosts[0].ID), nil, http.StatusOK, &hostResp)
	assert.Empty(t, hostResp.Host.EndUsers)

	// create a custom mapping of a non-existing host
	var putResp putHostDeviceMappingResponse
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[2].ID+1), nil, http.StatusNotFound, &putResp)

	// create some google mappings
	require.NoError(t, s.ds.ReplaceHostDeviceMapping(ctx, hosts[0].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[0].ID, Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{HostID: hosts[0].ID, Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
	}, fleet.DeviceMappingGoogleChromeProfiles))

	// create a custom mapping
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[0].ID), putHostDeviceMappingRequest{Email: "c@b.c"}, http.StatusOK, &putResp)
	require.Equal(t, hosts[0].ID, putResp.HostID)
	require.ElementsMatch(t, putResp.DeviceMapping, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "c@b.c", Source: fleet.DeviceMappingCustomReplacement},
	})

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[0].ID), nil, http.StatusOK, &listResp)
	require.Equal(t, hosts[0].ID, listResp.HostID)
	require.ElementsMatch(t, listResp.DeviceMapping, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "c@b.c", Source: fleet.DeviceMappingCustomReplacement},
	})
	// Check that mappings show up in host details
	hostResp = getHostResponse{}
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+hosts[0].UUID, nil, http.StatusOK, &hostResp)
	checkEndUser := func() {
		require.Len(t, hostResp.Host.EndUsers, 1)
		endUser := hostResp.Host.EndUsers[0]
		assert.Empty(t, endUser.IdpUserName)
		assert.Nil(t, endUser.IdpInfoUpdatedAt)
		assert.Empty(t, endUser.IdpID)
		assert.Empty(t, endUser.IdpFullName)
		assert.Empty(t, endUser.IdpGroups)
		require.Len(t, endUser.OtherEmails, 3)
		othersByEmail := make(map[string]string, 3)
		for _, otherEmail := range endUser.OtherEmails {
			othersByEmail[otherEmail.Email] = otherEmail.Source
		}
		source, ok := othersByEmail["a@b.c"]
		require.True(t, ok)
		assert.Equal(t, fleet.DeviceMappingGoogleChromeProfiles, source)
		source, ok = othersByEmail["b@b.c"]
		require.True(t, ok)
		assert.Equal(t, fleet.DeviceMappingGoogleChromeProfiles, source)
		source, ok = othersByEmail["c@b.c"]
		require.True(t, ok)
		assert.Equal(t, fleet.DeviceMappingCustomReplacement, source)
	}
	checkEndUser()
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hosts[0].ID), nil, http.StatusOK, &hostResp)
	checkEndUser()

	// other host still has none
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[1].ID), nil, http.StatusOK, &listResp)
	require.Empty(t, listResp.DeviceMapping)

	var listHosts listHostsResponse
	// list hosts response includes device mappings
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts)
	require.Len(t, listHosts.Hosts, len(hosts)+1)
	hostsByID := make(map[uint]fleet.HostResponse)
	for _, h := range listHosts.Hosts {
		hostsByID[h.ID] = h
	}
	var dm []*fleet.HostDeviceMapping

	// device mapping for host 1
	host1 := hosts[0]
	require.NotNil(t, *hostsByID[host1.ID].DeviceMapping)

	err := json.Unmarshal(*hostsByID[host1.ID].DeviceMapping, &dm)
	require.NoError(t, err)
	require.ElementsMatch(t, dm, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "c@b.c", Source: fleet.DeviceMappingCustomReplacement},
	})

	// no device mapping for other hosts
	assert.Nil(t, hostsByID[hosts[1].ID].DeviceMapping)
	assert.Nil(t, hostsByID[hosts[2].ID].DeviceMapping)
	assert.Nil(t, hostsByID[orbitHost.ID].DeviceMapping)

	// update custom email for hosts[0]
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[0].ID), putHostDeviceMappingRequest{Email: "d@b.c"}, http.StatusOK, &putResp)
	require.Equal(t, hosts[0].ID, putResp.HostID)
	require.ElementsMatch(t, putResp.DeviceMapping, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "d@b.c", Source: fleet.DeviceMappingCustomReplacement},
	})

	// create a custom_installer email for orbit host
	s.Do("PUT", "/api/fleet/orbit/device_mapping", fleet.OrbitPutDeviceMappingRequest{
		OrbitNodeKey: *orbitHost.OrbitNodeKey,
		Email:        "e@b.c",
	}, http.StatusOK)

	// search host by email address finds the corresponding host
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "a@b.c")
	require.Len(t, listHosts.Hosts, 1)
	require.Equal(t, host1.ID, listHosts.Hosts[0].ID)
	require.NotNil(t, listHosts.Hosts[0].DeviceMapping)

	err = json.Unmarshal(*listHosts.Hosts[0].DeviceMapping, &dm)
	require.NoError(t, err)
	require.ElementsMatch(t, putResp.DeviceMapping, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "d@b.c", Source: fleet.DeviceMappingCustomReplacement},
	})

	// search host by the custom email address finds the corresponding host
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "d@b.c")
	require.Len(t, listHosts.Hosts, 1)
	require.Equal(t, hosts[0].ID, listHosts.Hosts[0].ID)

	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "e@b.c")
	require.Len(t, listHosts.Hosts, 1)
	require.Equal(t, orbitHost.ID, listHosts.Hosts[0].ID)

	// override the custom email for the orbit host
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", orbitHost.ID), putHostDeviceMappingRequest{Email: "f@b.c"}, http.StatusOK, &putResp)

	// update the custom_installer email for orbit host, will get ignored (because a custom_override exists)
	s.Do("PUT", "/api/fleet/orbit/device_mapping", fleet.OrbitPutDeviceMappingRequest{
		OrbitNodeKey: *orbitHost.OrbitNodeKey,
		Email:        "g@b.c",
	}, http.StatusOK)

	// searching by the old custom installer email doesn't work anymore
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "e@b.c")
	require.Empty(t, listHosts.Hosts)

	// searching by the new custom email address finds it
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "f@b.c")
	require.Len(t, listHosts.Hosts, 1)
	require.Equal(t, orbitHost.ID, listHosts.Hosts[0].ID)

	// searching by a never-used email returns nothing
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "Z@b.c")
	require.Empty(t, listHosts.Hosts)
}

func (s *integrationTestSuite) TestHostDeviceMappingIDP() {
	t := s.T()
	hosts := s.createHosts(t)
	host := hosts[0]

	// Test 1: Test invalid source parameter validation
	var putResp putHostDeviceMappingResponse
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID),
		putHostDeviceMappingRequest{Email: "test@example.com", Source: "invalid"},
		http.StatusUnprocessableEntity, &putResp)

	// Test 2: Test endpoint routing - empty source defaults to custom (should work without premium)
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID),
		putHostDeviceMappingRequest{Email: "default@example.com"},
		http.StatusOK, &putResp)

	// Find the new mapping in the response
	var foundCustom bool
	for _, mapping := range putResp.DeviceMapping {
		if mapping.Email == "default@example.com" {
			assert.Equal(t, fleet.DeviceMappingCustomReplacement, mapping.Source)
			foundCustom = true
			break
		}
	}
	assert.True(t, foundCustom, "Should find the default custom mapping")

	// Test 3: Explicit custom source should work without premium
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID),
		putHostDeviceMappingRequest{Email: "custom@example.com", Source: "custom"},
		http.StatusOK, &putResp)

	// Find the custom mapping in the response
	var foundExplicitCustom bool
	for _, mapping := range putResp.DeviceMapping {
		if mapping.Email == "custom@example.com" {
			assert.Equal(t, fleet.DeviceMappingCustomReplacement, mapping.Source)
			foundExplicitCustom = true
			break
		}
	}
	assert.True(t, foundExplicitCustom, "Should find the explicit custom mapping")

	// Test 4: Verify custom mappings appear in host details via getHostEndpoint
	var hostResp getHostResponse
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/"+host.UUID, nil, http.StatusOK, &hostResp)

	// Should have at least 1 end user with device mappings
	require.GreaterOrEqual(t, len(hostResp.Host.EndUsers), 1)

	// Find mappings by checking OtherEmails in EndUsers
	foundMappings := make(map[string]string) // email -> source
	for _, endUser := range hostResp.Host.EndUsers {
		for _, otherEmail := range endUser.OtherEmails {
			foundMappings[otherEmail.Email] = otherEmail.Source
		}
	}

	// Verify that we have at least one custom mapping
	// (the exact emails present may vary based on how the system consolidates mappings)
	hasCustomMapping := false
	for email, source := range foundMappings {
		if source == fleet.DeviceMappingCustomReplacement {
			hasCustomMapping = true
			t.Logf("Found custom mapping: %s -> %s", email, source)
		}
	}
	assert.True(t, hasCustomMapping, "Should find at least one custom mapping in host details")

	// Verify that if we find specific mappings, they have the correct source
	if source, found := foundMappings["default@example.com"]; found {
		assert.Equal(t, fleet.DeviceMappingCustomReplacement, source)
	}
	if source, found := foundMappings["custom@example.com"]; found {
		assert.Equal(t, fleet.DeviceMappingCustomReplacement, source)
	}

	// Also test the ID-based endpoint
	hostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)

	// Verify mappings are consistent between identifier and ID endpoints
	foundMappingsById := make(map[string]string) // email -> source
	for _, endUser := range hostResp.Host.EndUsers {
		for _, otherEmail := range endUser.OtherEmails {
			foundMappingsById[otherEmail.Email] = otherEmail.Source
		}
	}
	assert.Equal(t, foundMappings, foundMappingsById, "Host details should be consistent between identifier and ID endpoints")

	// Test 5: IDP source validation (requires Fleet Premium)
	// This test verifies that the endpoint rejects IDP requests appropriately on free tier
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID),
		putHostDeviceMappingRequest{Email: "idp.user1@example.com", Source: "idp"},
		http.StatusPaymentRequired, &putResp)

	// Test 6: Delete IDP endpoint rejects request on Fleet Free
	var delResp deleteHostIDPResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping/idp", host.ID),
		deleteHostIDPRequest{},
		http.StatusPaymentRequired, &delResp)
}

func (s *integrationTestSuite) TestListHostsDeviceMappingSize() {
	t := s.T()
	ctx := context.Background()
	hosts := s.createHosts(t)

	testSize := 50
	var mappings []*fleet.HostDeviceMapping
	for range testSize {
		testEmail, _ := server.GenerateRandomText(14)
		mappings = append(mappings, &fleet.HostDeviceMapping{HostID: hosts[0].ID, Email: testEmail, Source: "google_chrome_profiles"})
	}

	require.NoError(t, s.ds.ReplaceHostDeviceMapping(ctx, hosts[0].ID, mappings, "google_chrome_profiles"))

	var listHosts listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts)

	hostsByID := make(map[uint]fleet.HostResponse)
	for _, h := range listHosts.Hosts {
		hostsByID[h.ID] = h
	}
	require.NotNil(t, *hostsByID[hosts[0].ID].DeviceMapping)

	var dm []*fleet.HostDeviceMapping
	err := json.Unmarshal(*hostsByID[hosts[0].ID].DeviceMapping, &dm)
	require.NoError(t, err)
	require.Len(t, dm, testSize)
}

func (s *integrationTestSuite) TestSearchHosts() {
	t := s.T()
	ctx := context.Background()

	hosts := s.createHosts(t)

	// set disk space information for hosts [0] and [1]
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[0].ID, 1.0, 2.0, 500.0, new(600.0)))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[1].ID, 3.0, 4.0, 1000.0, new(1200.0)))

	// no search criteria
	var searchResp searchHostsResponse
	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{}, http.StatusOK, &searchResp)
	require.Len(t, searchResp.Hosts, len(hosts)) // no request params
	for _, h := range searchResp.Hosts {
		switch h.ID {
		case hosts[0].ID:
			assert.InDelta(t, 1.0, h.GigsDiskSpaceAvailable, 0.001)
			assert.InDelta(t, 2.0, h.PercentDiskSpaceAvailable, 0.001)
			assert.InDelta(t, 500.0, h.GigsTotalDiskSpace, 0.001)
			assert.Equal(t, new(600.0), h.GigsAllDiskSpace)
		case hosts[1].ID:
			assert.InDelta(t, 3.0, h.GigsDiskSpaceAvailable, 0.001)
			assert.InDelta(t, 4.0, h.PercentDiskSpaceAvailable, 0.001)
			assert.InDelta(t, 1000.0, h.GigsTotalDiskSpace, 0.001)
			assert.Equal(t, new(1200.0), h.GigsAllDiskSpace)
		}
		assert.Equal(t, h.SoftwareUpdatedAt, h.CreatedAt)
	}

	searchResp = searchHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{ExcludedHostIDs: []uint{}}, http.StatusOK, &searchResp)
	require.Len(t, searchResp.Hosts, len(hosts)) // no omitted host id

	searchResp = searchHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{ExcludedHostIDs: []uint{hosts[1].ID}}, http.StatusOK, &searchResp)
	require.Len(t, searchResp.Hosts, len(hosts)-1) // one omitted host id

	searchResp = searchHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{MatchQuery: "foo.local1"}, http.StatusOK, &searchResp)
	require.Len(t, searchResp.Hosts, 1)
	require.Contains(t, searchResp.Hosts[0].Hostname, "foo.local1")

	// Update software and check that the software_updated_at is updated for the host returned by the search.
	time.Sleep(1 * time.Second)
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err := s.ds.UpdateHostSoftware(context.Background(), hosts[0].ID, software)
	require.NoError(t, err)
	searchResp = searchHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{MatchQuery: "foo.local0"}, http.StatusOK, &searchResp)
	require.Len(t, searchResp.Hosts, 1)
	require.Greater(t, searchResp.Hosts[0].SoftwareUpdatedAt, searchResp.Hosts[0].CreatedAt)

	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(
			context.Background(),
			`INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
			hosts[0].ID, "a@b.c", "src1",
		)

		return err
	})

	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{MatchQuery: "a@b.c"}, http.StatusOK, &searchResp)
	require.Len(t, searchResp.Hosts, 1)

	// search for non-existent email, shouldn't get anything back
	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{MatchQuery: "not@found.com"}, http.StatusOK, &searchResp)
	require.Empty(t, searchResp.Hosts)
}

func (s *integrationTestSuite) TestHostDeviceToken() {
	t := s.T()
	type response struct {
		Err string `json:"error"`
	}

	orbitHost := createOrbitEnrolledHost(t, "windows", "device_token", s.ds)

	// Write empty token
	body := fleet.SetOrUpdateDeviceTokenRequest{
		OrbitNodeKey:    *orbitHost.OrbitNodeKey,
		DeviceAuthToken: "",
	}
	s.DoJSON("POST", "/api/fleet/orbit/device_token", body, http.StatusBadRequest, &response{})

	// Use illegal characters
	body = fleet.SetOrUpdateDeviceTokenRequest{
		OrbitNodeKey:    *orbitHost.OrbitNodeKey,
		DeviceAuthToken: "../.",
	}
	s.DoJSON("POST", "/api/fleet/orbit/device_token", body, http.StatusBadRequest, &response{})

	// Write bad node key
	body = fleet.SetOrUpdateDeviceTokenRequest{
		OrbitNodeKey:    "",
		DeviceAuthToken: "token",
	}
	s.DoJSON("POST", "/api/fleet/orbit/device_token", body, http.StatusUnauthorized, &response{})

	// Write a good token.
	body = fleet.SetOrUpdateDeviceTokenRequest{
		OrbitNodeKey:    *orbitHost.OrbitNodeKey,
		DeviceAuthToken: "token",
	}
	s.DoJSON("POST", "/api/fleet/orbit/device_token", body, http.StatusOK, &response{})

	// Try to write the token again for a different host.
	// First write a valid token.
	orbitHost2 := createOrbitEnrolledHost(t, "darwin", "device_token2", s.ds)
	body = fleet.SetOrUpdateDeviceTokenRequest{
		OrbitNodeKey:    *orbitHost2.OrbitNodeKey,
		DeviceAuthToken: "token2",
	}
	s.DoJSON("POST", "/api/fleet/orbit/device_token", body, http.StatusOK, &response{})

	// Now write a duplicate token, which will result in a conflict with the first host.
	body = fleet.SetOrUpdateDeviceTokenRequest{
		OrbitNodeKey:    *orbitHost2.OrbitNodeKey,
		DeviceAuthToken: "token",
	}
	s.DoJSON("POST", "/api/fleet/orbit/device_token", body, http.StatusConflict, &response{})
}

func (s *integrationTestSuite) TestHostSoftwareWithTeamIdentifier() {
	t := s.T()
	ctx := context.Background()

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		NodeKey:       new(t.Name()),
		OsqueryHostID: new(t.Name()),
		UUID:          t.Name(),
		Hostname:      t.Name() + "foo.local",
		Platform:      "darwin",
	})
	require.NoError(t, err)

	safariApp := fleet.Software{
		Name:             "Safari.app",
		BundleIdentifier: "com.apple.safari",
		Version:          "18.1",
		Source:           "apps",
	}
	googleChromeApp := fleet.Software{
		Name:             "Google Chrome.app",
		BundleIdentifier: "com.google.Chrome",
		Version:          "130.0.6723.117",
		Source:           "apps",
	}
	ghCli := fleet.Software{
		Name:   "gh",
		Source: "homebrew_packages",
	}

	// Update the host's software.
	software := []fleet.Software{
		safariApp, googleChromeApp, ghCli,
	}
	hostSoftware, err := s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.Len(t, hostSoftware.CurrInstalled(), 3)

	// Update the host's software installed paths for the software above.
	// Google Chrome.app will have two installed paths one with team identifier set
	// the other one set to empty.
	swPaths := map[string]struct{}{}
	testCdHash := "e5b4ca9dd782162e526b95b2a37b25a55ddc8fdb"
	testExecHash := "f5b4ca9dd782162e526b95b2a37b25a55ddc8fdb"
	testExecPath := "/some/path/Google Chrome.app/Contents/MacOS/Google Chrome"
	for _, s := range software {
		pathItems := [][5]string{{fmt.Sprintf("/some/path/%s", s.Name), "", "", "", ""}}
		if s.Name == "Safari.app" {
			pathItems = [][5]string{
				{fmt.Sprintf("/some/path/%s", s.Name), "", testCdHash, testExecHash, testExecPath},
			}
		}
		if s.Name == "Google Chrome.app" {
			pathItems = [][5]string{
				{fmt.Sprintf("/some/path/%s", s.Name), "EQHXZ8M8AV", "", "", ""},
				{fmt.Sprintf("/some/other/path/%s", s.Name), "", "", "", ""},
			}
		}
		for _, pathItem := range pathItems {
			path := pathItem[0]
			teamIdentifier := pathItem[1]
			cdHash := pathItem[2]
			execHash := pathItem[3]
			execPath := pathItem[4]
			key := fmt.Sprintf(
				"%s%s%s%s%s%s%s%s%s%s%s",
				path, fleet.SoftwareFieldSeparator, teamIdentifier, fleet.SoftwareFieldSeparator, cdHash, fleet.SoftwareFieldSeparator, execHash, fleet.SoftwareFieldSeparator, execPath, fleet.SoftwareFieldSeparator, s.ToUniqueStr(),
			)
			swPaths[key] = struct{}{}
		}
	}
	err = s.ds.UpdateHostSoftwareInstalledPaths(ctx, host.ID, swPaths, hostSoftware)
	require.NoError(t, err)

	hostsCountTs := time.Now().UTC()
	err = s.ds.SyncHostsSoftware(context.Background(), hostsCountTs)
	require.NoError(t, err)

	getHostSoftwareResp := getHostSoftwareResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID),
		nil, http.StatusOK, &getHostSoftwareResp,
		"per_page", "5", "page", "0", "order_key", "name", "order_direction", "desc",
	)
	require.Len(t, getHostSoftwareResp.Software, 3)
	require.Equal(t, "Safari.app", getHostSoftwareResp.Software[0].Name)
	require.Len(t, getHostSoftwareResp.Software[0].InstalledVersions, 1)
	require.Len(t, getHostSoftwareResp.Software[0].InstalledVersions[0].InstalledPaths, 1)
	require.Equal(t, "/some/path/Safari.app", getHostSoftwareResp.Software[0].InstalledVersions[0].InstalledPaths[0])
	require.Len(t, getHostSoftwareResp.Software[0].InstalledVersions[0].SignatureInformation, 1)
	require.Equal(t, "/some/path/Safari.app", getHostSoftwareResp.Software[0].InstalledVersions[0].SignatureInformation[0].InstalledPath)
	require.Empty(t, getHostSoftwareResp.Software[0].InstalledVersions[0].SignatureInformation[0].TeamIdentifier)
	require.Equal(t, testCdHash, *getHostSoftwareResp.Software[0].InstalledVersions[0].SignatureInformation[0].CDHashSHA256)
	require.Equal(t, testExecHash, *getHostSoftwareResp.Software[0].InstalledVersions[0].SignatureInformation[0].ExecutableSHA256)
	require.Equal(t, testExecPath, *getHostSoftwareResp.Software[0].InstalledVersions[0].SignatureInformation[0].ExecutablePath)

	require.Equal(t, "Google Chrome.app", getHostSoftwareResp.Software[1].Name)
	require.Len(t, getHostSoftwareResp.Software[1].InstalledVersions, 1)
	require.Len(t, getHostSoftwareResp.Software[1].InstalledVersions[0].InstalledPaths, 2)
	slices.Sort(getHostSoftwareResp.Software[1].InstalledVersions[0].InstalledPaths)
	require.Equal(t, "/some/other/path/Google Chrome.app", getHostSoftwareResp.Software[1].InstalledVersions[0].InstalledPaths[0])
	require.Equal(t, "/some/path/Google Chrome.app", getHostSoftwareResp.Software[1].InstalledVersions[0].InstalledPaths[1])
	require.Len(t, getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation, 2)
	sort.Slice(getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation, func(i, j int) bool {
		return getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[i].InstalledPath < getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[j].InstalledPath
	})
	require.Equal(t, "/some/other/path/Google Chrome.app", getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[0].InstalledPath)
	require.Empty(t, getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[0].TeamIdentifier)
	require.Nil(t, getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[0].CDHashSHA256)
	require.Nil(t, getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[0].ExecutableSHA256)
	require.Nil(t, getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[0].ExecutablePath)
	require.Nil(t, getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[1].CDHashSHA256)
	require.Nil(t, getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[1].ExecutableSHA256)
	require.Nil(t, getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[1].ExecutablePath)
	require.Equal(t, "/some/path/Google Chrome.app", getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[1].InstalledPath)
	require.Equal(t, "EQHXZ8M8AV", getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[1].TeamIdentifier)

	require.Equal(t, "gh", getHostSoftwareResp.Software[2].Name)
	require.Len(t, getHostSoftwareResp.Software[2].InstalledVersions, 1)
	require.Len(t, getHostSoftwareResp.Software[2].InstalledVersions[0].InstalledPaths, 1)
	require.Equal(t, "/some/path/gh", getHostSoftwareResp.Software[2].InstalledVersions[0].InstalledPaths[0])
	require.Nil(t, getHostSoftwareResp.Software[2].InstalledVersions[0].SignatureInformation)
}

func (s *integrationTestSuite) TestHostReenrollWithSameHostRowRefetchOsquery() {
	t := s.T()

	// create a mac, linux and windows host
	host1 := createOrbitEnrolledHost(t, "darwin", "host1", s.ds)
	host2 := createOrbitEnrolledHost(t, "linux", "host2", s.ds)
	host3 := createOrbitEnrolledHost(t, "windows", "host3", s.ds)

	// set a chrome profile for each host
	for i, h := range []*fleet.Host{host1, host2, host3} {
		distributedReq := submitDistributedQueryResultsRequestShim{
			NodeKey: *h.NodeKey,
			Results: map[string]json.RawMessage{
				hostDetailQueryPrefix + "google_chrome_profiles": json.RawMessage(fmt.Sprintf(
					`[{"email": "%s"}]`, fmt.Sprintf("user%d@example.com", i),
				)),
			},
			Statuses: map[string]any{
				hostDistributedQueryPrefix + "google_chrome_profiles": 0,
			},
			Messages: map[string]string{},
			Stats:    map[string]*fleet.Stats{},
		}
		distributedResp := submitDistributedQueryResultsResponse{}
		s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)
	}

	oldHosts := make([]fleet.Host, 3)
	for i, h := range []*fleet.Host{host1, host2, host3} {
		var hostResponse getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostResponse)
		require.False(t, hostResponse.Host.RefetchRequested)
		require.Len(t, hostResponse.Host.EndUsers, 1)
		require.Len(t, hostResponse.Host.EndUsers[0].OtherEmails, 1)
		require.Equal(t, "google_chrome_profiles", hostResponse.Host.EndUsers[0].OtherEmails[0].Source)
		oldHosts[i] = hostResponse.Host.Host
	}

	// do an orbit re-enrollment of the hosts, should set refetch requested
	orbitKey := setOrbitEnrollment(t, host1, s.ds)
	host1.OrbitNodeKey = &orbitKey
	orbitKey = setOrbitEnrollment(t, host2, s.ds)
	host2.OrbitNodeKey = &orbitKey
	orbitKey = setOrbitEnrollment(t, host3, s.ds)
	host3.OrbitNodeKey = &orbitKey

	for i, h := range []*fleet.Host{host1, host2, host3} {
		var hostResponse getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostResponse)
		require.True(t, hostResponse.Host.RefetchRequested)
		require.Len(t, hostResponse.Host.EndUsers, 1)
		require.Len(t, hostResponse.Host.EndUsers[0].OtherEmails, 1)
		require.Equal(t, "google_chrome_profiles", hostResponse.Host.EndUsers[0].OtherEmails[0].Source)
		require.Equal(t, oldHosts[i].ID, h.ID)
	}

	// send a response for the refetch request
	for _, h := range []*fleet.Host{host1, host2, host3} {
		distributedReq := submitDistributedQueryResultsRequestShim{
			NodeKey: *h.NodeKey,
			Results: map[string]json.RawMessage{
				hostDetailQueryPrefix + "google_chrome_profiles": json.RawMessage(`[]`),
			},
			Statuses: map[string]any{
				hostDistributedQueryPrefix + "google_chrome_profiles": 0,
			},
			Messages: map[string]string{},
			Stats:    map[string]*fleet.Stats{},
		}
		distributedResp := submitDistributedQueryResultsResponse{}
		s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)
	}

	for i, h := range []*fleet.Host{host1, host2, host3} {
		var hostResponse getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h.ID), nil, http.StatusOK, &hostResponse)
		require.False(t, hostResponse.Host.RefetchRequested)
		require.Empty(t, hostResponse.Host.EndUsers)
		require.Equal(t, oldHosts[i].ID, h.ID)
	}
}

func (s *integrationTestSuite) TestHostDeviceURL() {
	t := s.T()
	ctx := context.Background()

	// Pin the server URL so the assertion below is stable. Restore the original
	// on cleanup so other tests aren't affected.
	origAC, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	origServerURL := origAC.ServerSettings.ServerURL
	// Trailing slash exercises the TrimRight in HostDeviceURL.
	origAC.ServerSettings.ServerURL = "https://fleet.example.com/"
	require.NoError(t, s.ds.SaveAppConfig(ctx, origAC))
	t.Cleanup(func() {
		ac, err := s.ds.AppConfig(ctx)
		require.NoError(t, err)
		ac.ServerSettings.ServerURL = origServerURL
		require.NoError(t, s.ds.SaveAppConfig(ctx, ac))
	})

	const freshToken = "my-device-link-fresh-token" //nolint:gosec // G101 false positive, test fixture value
	host := createOrbitEnrolledHost(t, "linux", "device-url-host", s.ds)
	createDeviceTokenForHost(t, s.ds, host.ID, freshToken)

	retrievedActivity := fleet.ActivityTypeRetrievedHostMyDeviceURL{}.ActivityName()
	expectedActivityDetails := fmt.Sprintf(
		`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName(),
	)

	// Case 1: host has a fresh token → it should be reused as-is.
	var resp getHostDeviceURLResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_url", host.ID), nil, http.StatusOK, &resp)
	require.Equal(t, host.ID, resp.HostID)
	require.Equal(t, "https://fleet.example.com/device/"+freshToken, resp.DeviceURL)
	// Each successful retrieval logs an audit activity tied to the host.
	s.lastActivityOfTypeMatches(retrievedActivity, expectedActivityDetails, 0)

	// A second call also reuses the same token (still fresh).
	var resp2 getHostDeviceURLResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_url", host.ID), nil, http.StatusOK, &resp2)
	require.Equal(t, resp.DeviceURL, resp2.DeviceURL)

	// Case 2: host has a token but it's expired. Push updated_at into the
	// past, then expect a freshly generated token (different from the stale
	// one).
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE host_device_auth SET updated_at = DATE_SUB(NOW(), INTERVAL 2 HOUR) WHERE host_id = ?`, host.ID)
		return err
	})

	var respExpired getHostDeviceURLResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_url", host.ID), nil, http.StatusOK, &respExpired)
	require.True(t, strings.HasPrefix(respExpired.DeviceURL, "https://fleet.example.com/device/"), "URL: %s", respExpired.DeviceURL)
	require.NotEqual(t, resp.DeviceURL, respExpired.DeviceURL, "expected stale token to be rotated")

	regenToken, err := s.ds.GetDeviceAuthToken(ctx, host.ID)
	require.NoError(t, err)
	require.NotEqual(t, freshToken, regenToken)
	require.Equal(t, "https://fleet.example.com/device/"+regenToken, respExpired.DeviceURL)

	// Case 3: host has never had a token row → Fleet generates one.
	hostNoToken := createOrbitEnrolledHost(t, "linux", "device-url-no-token", s.ds)
	var respNew getHostDeviceURLResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_url", hostNoToken.ID), nil, http.StatusOK, &respNew)
	require.True(t, strings.HasPrefix(respNew.DeviceURL, "https://fleet.example.com/device/"), "URL: %s", respNew.DeviceURL)
	newlyMintedToken, err := s.ds.GetDeviceAuthToken(ctx, hostNoToken.ID)
	require.NoError(t, err)
	require.Equal(t, "https://fleet.example.com/device/"+newlyMintedToken, respNew.DeviceURL)

	// Unknown host ID: 404.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_url", host.ID+9999), nil, http.StatusNotFound, &resp)

	// iOS and iPadOS hosts have no device auth token; their URL is the host
	// UUID landing on the self-service tab, matching the Web Clip profile in
	// docs/solutions/ios-ipados.
	iosHost := createOrbitEnrolledHost(t, "ios", "device-url-ios", s.ds)
	var iosResp getHostDeviceURLResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_url", iosHost.ID), nil, http.StatusOK, &iosResp)
	require.Equal(t, "https://fleet.example.com/device/"+iosHost.UUID+"/self-service", iosResp.DeviceURL)
	ipadHost := createOrbitEnrolledHost(t, "ipados", "device-url-ipad", s.ds)
	var ipadResp getHostDeviceURLResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_url", ipadHost.ID), nil, http.StatusOK, &ipadResp)
	require.Equal(t, "https://fleet.example.com/device/"+ipadHost.UUID+"/self-service", ipadResp.DeviceURL)

	// Android and ChromeOS have no My device page at all, so the endpoint explains
	// that rather than minting a URL that leads nowhere. See #48439.
	for _, platform := range []string{"android", "chrome", "CrOS"} {
		unsupportedHost := createOrbitEnrolledHost(t, platform, "device-url-"+platform, s.ds)
		res := s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_url", unsupportedHost.ID), nil, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), fleet.MyDeviceURLUnsupportedPlatformMessage, "platform %s", platform)
	}

	// Non-global-admin roles: 403. Switch tokens, then restore admin token at end.
	defer func() { s.token = s.getTestAdminToken() }()

	t.Run("global maintainer is forbidden", func(t *testing.T) {
		s.setTokenForTest(t, TestMaintainerUserEmail, test.GoodPassword)
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_url", host.ID), nil, http.StatusForbidden, &resp)
	})
	t.Run("global observer is forbidden", func(t *testing.T) {
		s.setTokenForTest(t, TestObserverUserEmail, test.GoodPassword)
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_url", host.ID), nil, http.StatusForbidden, &resp)
	})
}
