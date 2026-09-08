package service

// Label tests for the core (no-license) suite.
//
// Belongs here: label CRUD and the label spec endpoints, built-in label behaviour,
// listing a label's hosts, and adding/removing manual label membership on a host.
//
// Does not belong here: filtering the host list by label
// (integration_core_hosts_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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

func (s *integrationTestSuite) TestLabels() {
	t := s.T()

	// create some hosts to use for manual labels
	hosts := s.createHosts(t, "debian", "linux", "fedora", "darwin", "darwin", "darwin", "darwin")
	manualHosts := hosts[:3]
	lbl2Hosts := hosts[3:6]

	t.Run("Manual and Dynamic Labels", func(t *testing.T) {
		// list labels, has the built-in ones
		builtinsMap := fleet.ReservedLabelNames()
		var listResp fleet.ListLabelsResponse
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp)
		assert.NotEmpty(t, listResp.Labels)
		var builtinLbl fleet.Label
		for _, lbl := range listResp.Labels {
			_, ok := builtinsMap[lbl.Name]
			assert.True(t, ok)
			assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
			builtinLbl = lbl.Label
		}
		builtInsCount := len(listResp.Labels)
		require.Len(t, builtinsMap, builtInsCount)

		// labels summary has the built-in ones
		var summaryResp fleet.GetLabelsSummaryResponse
		s.DoJSON("GET", "/api/latest/fleet/labels/summary", nil, http.StatusOK, &summaryResp)
		assert.Len(t, summaryResp.Labels, builtInsCount)
		for _, lbl := range summaryResp.Labels {
			_, ok := builtinsMap[lbl.Name]
			assert.True(t, ok)
			assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
		}

		// create a label without name, an error
		var createResp fleet.CreateLabelResponse
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Query: "select 1"}, http.StatusUnprocessableEntity, &createResp)

		// create a label with both a query and hosts, error
		res := s.Do("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: t.Name(), Query: "select 1", Hosts: []string{manualHosts[0].UUID}}, http.StatusUnprocessableEntity)
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, `Only one of "criteria", "query" or "hosts/host_ids" can be included in the request.`)

		// create invalid label, conflicts with builtin name (case-insensitive)
		for n := range builtinsMap {
			s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: n, Query: "select 1"}, http.StatusUnprocessableEntity, &createResp)
			s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: strings.ToLower(n), Query: "select 1"}, http.StatusUnprocessableEntity, &createResp)
			s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: strings.ToUpper(n), Query: "select 1"}, http.StatusUnprocessableEntity, &createResp)
		}

		// try to create a label with an invalid platform
		s.DoJSON(
			"POST",
			"/api/latest/fleet/labels",
			&fleet.LabelPayload{
				Name:     "amazing label",
				Query:    "select 1",
				Platform: "bados",
			},
			http.StatusUnprocessableEntity,
			&createResp,
		)

		// create a label with the generic "linux" platform (matches all Linux distros)
		s.DoJSON(
			"POST",
			"/api/latest/fleet/labels",
			&fleet.LabelPayload{
				Name:     "linux label",
				Query:    "select 1",
				Platform: "linux",
			},
			http.StatusOK,
			&createResp,
		)
		assert.NotZero(t, createResp.Label.ID)
		assert.Equal(t, "linux", createResp.Label.Platform)
		linuxLbl := createResp.Label.Label

		// create a valid dynamic label
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: t.Name(), Query: "select 1"}, http.StatusOK, &createResp)
		assert.NotZero(t, createResp.Label.ID)
		assert.Equal(t, t.Name(), createResp.Label.Name)
		assert.Empty(t, createResp.Label.HostIDs)
		assert.False(t, createResp.Label.CreatedAt.IsZero())
		assert.WithinDuration(t, time.Now(), createResp.Label.CreatedAt, time.Minute)
		assert.False(t, createResp.Label.UpdatedAt.IsZero())
		assert.WithinDuration(t, time.Now(), createResp.Label.UpdatedAt, time.Minute)
		lbl1 := createResp.Label.Label

		// try to create a manual label with the same name
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: lbl1.Name, Hosts: []string{manualHosts[0].UUID}}, http.StatusConflict, &createResp)
		// try to create a dynamic label with the same name
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: lbl1.Name, Query: "select 2"}, http.StatusConflict, &createResp)

		// get the label
		var getResp fleet.GetLabelResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), nil, http.StatusOK, &getResp)
		assert.Equal(t, lbl1.ID, getResp.Label.ID)
		assert.Empty(t, getResp.Label.HostIDs)

		// get a non-existing label
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID+1), nil, http.StatusNotFound, &getResp)

		// create a valid manual label
		createResp = fleet.CreateLabelResponse{}
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: t.Name() + "manual", Hosts: []string{manualHosts[0].UUID, manualHosts[1].Hostname, *manualHosts[2].NodeKey}}, http.StatusOK, &createResp)
		assert.NotZero(t, createResp.Label.ID)
		assert.Equal(t, t.Name()+"manual", createResp.Label.Name)
		assert.ElementsMatch(t, []uint{manualHosts[0].ID, manualHosts[1].ID, manualHosts[2].ID}, createResp.Label.HostIDs)
		manualLbl1 := createResp.Label.Label

		// get the label
		getResp = fleet.GetLabelResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl1.ID), nil, http.StatusOK, &getResp)
		assert.Equal(t, manualLbl1.ID, getResp.Label.ID)
		assert.Equal(t, fleet.LabelTypeRegular, getResp.Label.LabelType)
		assert.Equal(t, fleet.LabelMembershipTypeManual, getResp.Label.LabelMembershipType)
		assert.ElementsMatch(t, []uint{manualHosts[0].ID, manualHosts[1].ID, manualHosts[2].ID}, getResp.Label.HostIDs)
		assert.Equal(t, 3, getResp.Label.HostCount)

		// create a valid empty manual label
		createResp = fleet.CreateLabelResponse{}
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: strings.ReplaceAll(t.Name(), "/", "_") + "manual2"}, http.StatusOK, &createResp)
		assert.NotZero(t, createResp.Label.ID)
		assert.Equal(t, strings.ReplaceAll(t.Name(), "/", "_")+"manual2", createResp.Label.Name)
		assert.Empty(t, createResp.Label.HostIDs)
		manualLbl2 := createResp.Label.Label

		// try to create a manual label with the same name
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: manualLbl2.Name, Hosts: []string{manualHosts[0].UUID}}, http.StatusConflict, &createResp)
		// try to create a dynamic label with the same name
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: manualLbl2.Name, Query: "select 2"}, http.StatusConflict, &createResp)

		// get the label
		getResp = fleet.GetLabelResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl2.ID), nil, http.StatusOK, &getResp)
		assert.Equal(t, manualLbl2.ID, getResp.Label.ID)
		assert.Equal(t, fleet.LabelTypeRegular, getResp.Label.LabelType)
		assert.Equal(t, fleet.LabelMembershipTypeManual, getResp.Label.LabelMembershipType)
		assert.Empty(t, getResp.Label.HostIDs)
		assert.Equal(t, 0, getResp.Label.HostCount)

		// get a non-existing label
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", 9999), nil, http.StatusNotFound, &getResp)

		// modify dynamic label lbl1
		var modResp fleet.ModifyLabelResponse
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), &fleet.ModifyLabelPayload{Name: new(t.Name() + "zzz")}, http.StatusOK, &modResp)
		assert.Equal(t, lbl1.ID, modResp.Label.ID)
		assert.Empty(t, modResp.Label.HostIDs)
		assert.NotEqual(t, lbl1.Name, modResp.Label.Name)

		// attempt to modify a label to a reserved name
		for n := range builtinsMap {
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), &fleet.ModifyLabelPayload{Name: new(n)}, http.StatusUnprocessableEntity, &modResp)
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), &fleet.ModifyLabelPayload{Name: new(strings.ToLower(n))}, http.StatusUnprocessableEntity, &modResp)
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), &fleet.ModifyLabelPayload{Name: new(strings.ToUpper(n))}, http.StatusUnprocessableEntity, &modResp)
		}

		// modify a non-existing label
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", 9999), &fleet.ModifyLabelPayload{Name: new("zzz")}, http.StatusNotFound, &modResp)
		// modify a built-in label
		res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", builtinLbl.ID), &fleet.ModifyLabelPayload{Name: new("zzz")}, http.StatusUnprocessableEntity)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "cannot modify built-in label")

		// modify manual label 1 without modifying its hosts
		modResp = fleet.ModifyLabelResponse{}
		newName := "modified_manual_label1"
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl1.ID), &fleet.ModifyLabelPayload{Name: &newName}, http.StatusOK,
			&modResp)
		assert.Equal(t, manualLbl1.ID, modResp.Label.ID)
		assert.Equal(t, fleet.LabelTypeRegular, modResp.Label.LabelType)
		assert.Equal(t, fleet.LabelMembershipTypeManual, modResp.Label.LabelMembershipType)
		assert.ElementsMatch(t, []uint{manualHosts[0].ID, manualHosts[1].ID, manualHosts[2].ID}, modResp.Label.HostIDs)
		assert.Equal(t, 3, modResp.Label.HostCount)
		assert.Equal(t, newName, modResp.Label.Name)

		// add a host with the same name as another host to manual label 2, confirm only one host is added
		sameName, err := s.ds.NewHost(context.Background(), &fleet.Host{
			HardwareSerial: "ABCDE",
			Hostname:       manualHosts[0].Hostname,
			Platform:       "darwin",
		})
		require.NoError(t, err)

		modResp = fleet.ModifyLabelResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl2.ID),
			&fleet.ModifyLabelPayload{Hosts: []string{sameName.HardwareSerial}}, http.StatusOK, &modResp)
		assert.Len(t, modResp.Label.HostIDs, 1)
		assert.NotEqual(t, manualHosts[0].ID, modResp.Label.HostIDs[0])
		assert.Equal(t, manualLbl2.ID, modResp.Label.ID)
		assert.Equal(t, fleet.LabelTypeRegular, modResp.Label.LabelType)
		assert.Equal(t, fleet.LabelMembershipTypeManual, modResp.Label.LabelMembershipType)
		assert.ElementsMatch(t, []uint{sameName.ID}, modResp.Label.HostIDs)
		assert.Equal(t, 1, modResp.Label.HostCount)

		// modify manual label 2 adding some hosts
		modResp = fleet.ModifyLabelResponse{}
		newName = "modified_manual_label2"
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl2.ID),
			&fleet.ModifyLabelPayload{Name: &newName, Hosts: []string{manualHosts[0].UUID}}, http.StatusOK, &modResp)
		assert.Equal(t, manualLbl2.ID, modResp.Label.ID)
		assert.Equal(t, fleet.LabelTypeRegular, modResp.Label.LabelType)
		assert.Equal(t, fleet.LabelMembershipTypeManual, modResp.Label.LabelMembershipType)
		assert.ElementsMatch(t, []uint{manualHosts[0].ID}, modResp.Label.HostIDs)
		assert.Equal(t, 1, modResp.Label.HostCount)
		assert.Equal(t, newName, modResp.Label.Name)
		manualLbl2.Name = newName

		// modify manual label 2 adding some hosts by ID
		modResp = fleet.ModifyLabelResponse{}
		newName = "modified_manual_label2"
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl2.ID),
			&fleet.ModifyLabelPayload{Name: &newName, HostIDs: []uint{manualHosts[1].ID, manualHosts[2].ID}}, http.StatusOK, &modResp)
		assert.Equal(t, manualLbl2.ID, modResp.Label.ID)
		assert.Equal(t, fleet.LabelTypeRegular, modResp.Label.LabelType)
		assert.Equal(t, fleet.LabelMembershipTypeManual, modResp.Label.LabelMembershipType)
		assert.ElementsMatch(t, []uint{manualHosts[1].ID, manualHosts[2].ID}, modResp.Label.HostIDs)
		assert.Equal(t, 2, modResp.Label.HostCount)
		assert.Equal(t, newName, modResp.Label.Name)
		manualLbl2.Name = newName

		// modify manual label 2 clearing its hosts
		modResp = fleet.ModifyLabelResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl2.ID), &fleet.ModifyLabelPayload{Hosts: []string{}, Description: new("desc")}, http.StatusOK, &modResp)
		assert.Equal(t, manualLbl2.ID, modResp.Label.ID)
		assert.Equal(t, "desc", modResp.Label.Description)
		assert.Empty(t, modResp.Label.HostIDs)
		assert.Equal(t, 0, modResp.Label.HostCount)

		// list labels
		dynamicLabels := []fleet.Label{lbl1, linuxLbl}
		manualLabels := []fleet.Label{manualLbl1, manualLbl2}
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(100))
		assert.Len(t, listResp.Labels, builtInsCount+len(dynamicLabels)+len(manualLabels))

		// labels summary
		s.DoJSON("GET", "/api/latest/fleet/labels/summary", nil, http.StatusOK, &summaryResp)
		assert.Len(t, summaryResp.Labels, builtInsCount+len(dynamicLabels)+len(manualLabels))

		// next page is empty
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "per_page", "100", "page", "1")
		assert.Empty(t, listResp.Labels)

		// list labels with invalid query params
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusBadRequest, &listResp, "per_page", strconv.Itoa(builtInsCount+1), "order_key", "id", "after", "1")
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusBadRequest, &listResp, "per_page", strconv.Itoa(builtInsCount+1), "query", "no match query for this endpoint")

		// create another dynamic label
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: strings.ReplaceAll(t.Name(), "/", "_"), Query: "select 1"}, http.StatusOK, &createResp)
		assert.NotZero(t, createResp.Label.ID)
		lbl2 := createResp.Label.Label
		dynamicLabels = append(dynamicLabels, lbl2)
		require.Len(t, dynamicLabels, 3) // to make linter happy (dynamicLabels is not used past this point)

		// add lbl2 hosts to that label
		for _, h := range lbl2Hosts {
			err := s.ds.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{lbl2.ID: new(true)}, time.Now(), false)
			require.NoError(t, err)
		}

		// list hosts in dynamic label lbl2
		var listHostsResp listHostsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp)
		assert.Len(t, listHostsResp.Hosts, len(lbl2Hosts))

		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "order_key", "id", "after", fmt.Sprintf("%d", lbl2Hosts[0].ID))
		assert.Len(t, listHostsResp.Hosts, 2)
		assert.Equal(t, lbl2Hosts[1].ID, listHostsResp.Hosts[0].ID)
		assert.Equal(t, lbl2Hosts[2].ID, listHostsResp.Hosts[1].ID)

		// a dynamic label's membership cannot be replaced, and an empty list is a
		// replacement too (it would clear all of its members)
		for _, payload := range []fleet.ModifyLabelPayload{
			{HostIDs: []uint{}},
			{HostIDs: []uint{lbl2Hosts[0].ID}},
			{Hosts: []string{}},
			{Hosts: []string{lbl2Hosts[0].UUID}},
		} {
			res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl2.ID), &payload, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, `"hosts" or "host_ids" can only be provided for a manual label`)
		}

		listHostsResp = listHostsResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp)
		assert.Len(t, listHostsResp.Hosts, len(lbl2Hosts))

		// list hosts in manual label 1
		listHostsResp = listHostsResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", manualLbl1.ID), nil, http.StatusOK, &listHostsResp, "order_key", "id")
		assert.Len(t, listHostsResp.Hosts, manualLbl1.HostCount)
		assert.Equal(t, manualHosts[0].ID, listHostsResp.Hosts[0].ID)
		assert.Equal(t, manualHosts[1].ID, listHostsResp.Hosts[1].ID)
		assert.Equal(t, manualHosts[2].ID, listHostsResp.Hosts[2].ID)

		// list hosts in manual label 2
		listHostsResp = listHostsResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", manualLbl2.ID), nil, http.StatusOK, &listHostsResp, "order_key", "id")
		assert.Empty(t, listHostsResp.Hosts)

		// list hosts in dynamic label 2 searching by display_name
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "order_key", "display_name", "order_direction", "desc")
		assert.Len(t, listHostsResp.Hosts, len(lbl2Hosts))
		// first in the list is the last one, as the names are ordered with the index
		// of creation, and vice-versa
		assert.Equal(t, lbl2Hosts[len(lbl2Hosts)-1].ID, listHostsResp.Hosts[0].ID)
		assert.Equal(t, lbl2Hosts[0].ID, listHostsResp.Hosts[len(lbl2Hosts)-1].ID)

		mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(
				context.Background(),
				`INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
				lbl2Hosts[0].ID, "a@b.c", "src1",
			)

			return err
		})

		// list hosts in label searching by email address
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "query", "a@b.c")
		assert.Len(t, listHostsResp.Hosts, 1)
		assert.Equal(t, lbl2Hosts[0].ID, listHostsResp.Hosts[0].ID)

		// list hosts in label searching by email address with leading/trailing whitespace
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "query", "    a@b.c   ")
		assert.Len(t, listHostsResp.Hosts, 1)
		assert.Equal(t, lbl2Hosts[0].ID, listHostsResp.Hosts[0].ID)

		// count hosts in label order by display_name
		var countResp countHostsResponse
		s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "label_id", fmt.Sprint(lbl2.ID), "order_key", "display_name", "order_direction", "desc")
		assert.Equal(t, len(lbl2Hosts), countResp.Count)

		// lists hosts in label without hosts
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl1.ID), nil, http.StatusOK, &listHostsResp)
		assert.Empty(t, listHostsResp.Hosts)

		// count hosts in label
		s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "label_id", fmt.Sprint(lbl1.ID))
		assert.Equal(t, 0, countResp.Count)

		// lists hosts in invalid label
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID+1), nil, http.StatusOK, &listHostsResp)
		assert.Empty(t, listHostsResp.Hosts)

		// set MDM information on a host
		require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), lbl2Hosts[0].ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "", false))
		var mdmID uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(context.Background(), q, &mdmID,
				`SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`, fleet.WellKnownMDMSimpleMDM, "https://simplemdm.com")
		})
		// generate aggregated stats
		require.NoError(t, s.ds.GenerateAggregatedMunkiAndMDM(context.Background()))

		// list host in label by mdm_id
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "mdm_id", fmt.Sprint(mdmID))
		require.Len(t, listHostsResp.Hosts, 1)
		assert.Nil(t, listHostsResp.Software)
		assert.Nil(t, listHostsResp.MunkiIssue)
		require.NotNil(t, listHostsResp.MDMSolution)
		assert.Equal(t, mdmID, listHostsResp.MDMSolution.ID)
		assert.Equal(t, fleet.WellKnownMDMSimpleMDM, listHostsResp.MDMSolution.Name)
		assert.Equal(t, "https://simplemdm.com", listHostsResp.MDMSolution.ServerURL)

		// invalid order_key returns 422 (sensitive columns must not be sortable to prevent
		// information disclosure via binary search extraction).
		for _, key := range []string{
			"node_key", "h.node_key",
			"orbit_node_key", "h.orbit_node_key",
			"invalid_column", "h.invalid_column",
			// computer_name is a valid field, but must not contain the table alias
			// (previous version of the endpoint allowed setting aliases here).
			"h.computer_name",
		} {
			res := s.Do("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusUnprocessableEntity, "order_key", key)
			errMsg := extractServerErrorText(res.Body)
			assert.Contains(t, errMsg, "invalid order_key")
			assert.Contains(t, errMsg, key)
		}

		// all allowed order_key values must be accepted by the endpoint and return the
		// hosts in lbl2.
		allowedOrderKeys := []string{
			"id", "osquery_host_id", "created_at", "updated_at", "detail_updated_at",
			"hostname", "uuid", "platform", "osquery_version", "os_version", "build",
			"platform_like", "code_name", "uptime", "memory", "cpu_type", "cpu_subtype",
			"cpu_brand", "cpu_physical_cores", "cpu_logical_cores",
			"hardware_vendor", "hardware_model", "hardware_version", "hardware_serial",
			"computer_name", "primary_ip_id", "distributed_interval", "logger_tls_period",
			"config_tls_refresh", "primary_ip", "primary_mac", "label_updated_at",
			"last_enrolled_at", "refetch_requested", "refetch_critical_queries_until",
			"team_id", "policy_updated_at", "public_ip",
			"gigs_disk_space_available", "percent_disk_space_available",
			"gigs_total_disk_space", "seen_time", "software_updated_at",
			"last_restarted_at", "timezone", "team_name",
			"failing_policies_count", "critical_vulnerabilities_count",
			"total_issues_count",
			"issues", // supported as alias for "total_issues_count"
			"device_mapping",
			"display_name",
		}
		for _, key := range allowedOrderKeys {
			listHostsResp = listHostsResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "order_key", key, "device_mapping", "true")
			assert.Len(t, listHostsResp.Hosts, len(lbl2Hosts), "order_key=%s", key)
		}

		// every allowed order_key must also accept the `after` cursor without
		// erroring — guards against SELECT-list aliases leaking into the WHERE
		// clause (MySQL disallows aliases in WHERE).
		for _, key := range allowedOrderKeys {
			listHostsResp = listHostsResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp,
				"order_key", key, "order_direction", "asc", "after", "0", "device_mapping", "true")
		}

		// issue-related order_keys are rejected when disable_issues=true.
		for _, key := range []string{"issues", "failing_policies_count", "critical_vulnerabilities_count", "total_issues_count"} {
			res := s.Do("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusBadRequest,
				"order_key", key, "disable_issues", "true")
			errMsg := extractServerErrorText(res.Body)
			assert.Contains(t, errMsg, "Invalid order_key")
			assert.Contains(t, errMsg, key)
		}

		// device_mapping order_key is rejected when device_mapping is not enabled.
		res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusBadRequest,
			"order_key", "device_mapping")
		errMsg = extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, "Invalid order_key")
		assert.Contains(t, errMsg, "device_mapping")

		// device_mapping=false is also rejected.
		res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusBadRequest,
			"order_key", "device_mapping", "device_mapping", "false")
		errMsg = extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, "Invalid order_key")
		assert.Contains(t, errMsg, "device_mapping")

		// delete a label by id
		var delIDResp fleet.DeleteLabelByIDResponse
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", linuxLbl.ID), nil, http.StatusOK, &delIDResp)
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", lbl1.ID), nil, http.StatusOK, &delIDResp)

		// delete a non-existing label by id
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", lbl2.ID+1), nil, http.StatusNotFound, &delIDResp)

		// delete a label by name
		var delResp fleet.DeleteLabelResponse
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(lbl2.Name)), nil, http.StatusOK, &delResp)

		// delete a non-existing label by name
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(lbl2.Name)), nil, http.StatusNotFound, &delResp)

		// delete a built-in label by name (case-insensitive)
		for n := range builtinsMap {
			for _, variant := range []string{n, strings.ToLower(n), strings.ToUpper(n)} {
				res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(variant)), nil, http.StatusUnprocessableEntity)
				errMsg = extractServerErrorText(res.Body)
				require.Contains(t, errMsg, "cannot delete built-in label")
			}
		}
		listResp = fleet.ListLabelsResponse{}
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp)
		var remainingBuiltIns int
		for _, lbl := range listResp.Labels {
			if _, ok := builtinsMap[lbl.Name]; ok {
				remainingBuiltIns++
			}
		}
		require.Equal(t, len(builtinsMap), remainingBuiltIns)

		// delete a manual label by id
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", manualLbl1.ID), nil, http.StatusOK, &delIDResp)

		// delete a manual label by name
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(manualLbl2.Name)), nil, http.StatusOK, &delResp)

		// list labels, only the built-ins remain
		s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(builtInsCount+1))
		assert.Len(t, listResp.Labels, builtInsCount)
		idsByName := make(map[string]uint, len(listResp.Labels))
		for _, lbl := range listResp.Labels {
			_, ok := builtinsMap[lbl.Name]
			assert.True(t, ok)
			assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
			idsByName[lbl.Name] = lbl.ID
		}

		// labels summary, only the built-ins remains
		s.DoJSON("GET", "/api/latest/fleet/labels/summary", nil, http.StatusOK, &summaryResp)
		assert.Len(t, summaryResp.Labels, builtInsCount)
		for _, lbl := range summaryResp.Labels {
			_, ok := builtinsMap[lbl.Name]
			assert.True(t, ok)
			assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
			assert.Equal(t, idsByName[lbl.Name], lbl.ID)
		}

		// host summary matches built-ins count
		var hostSummaryResp getHostSummaryResponse
		s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &hostSummaryResp)
		assert.Len(t, hostSummaryResp.BuiltinLabels, builtInsCount)
		for _, lbl := range hostSummaryResp.BuiltinLabels {
			_, ok := builtinsMap[lbl.Name]
			assert.True(t, ok)
			assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
			assert.Equal(t, idsByName[lbl.Name], lbl.ID)
		}

		require.Len(t, idsByName, len(builtinsMap))
		for name := range builtinsMap {
			id, ok := idsByName[name]
			require.True(t, ok)

			// attempt to delete by name
			s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(name)), nil, http.StatusUnprocessableEntity, &delResp)

			// attempt to delete by id
			s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", id), nil, http.StatusUnprocessableEntity, &delIDResp)
		}

		// A modify that changes membership but fails metadata validation (a
		// duplicate name) must roll back as a unit: the membership change must not
		// be committed on its own, and no edited_label activity must be recorded
		// for the failed request.
		createResp = fleet.CreateLabelResponse{}
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: "atomic_conflict_label"}, http.StatusOK, &createResp)
		conflictName := createResp.Label.Name

		createResp = fleet.CreateLabelResponse{}
		s.DoJSON("POST", "/api/latest/fleet/labels",
			&fleet.LabelPayload{Name: "atomic_target_label", HostIDs: []uint{manualHosts[0].ID, manualHosts[1].ID}}, http.StatusOK, &createResp)
		atomicLbl := createResp.Label.Label
		require.ElementsMatch(t, []uint{manualHosts[0].ID, manualHosts[1].ID}, createResp.Label.HostIDs)

		// watermark: id of the most recent activity before the failed request
		lastActID := s.lastActivityMatches("", "", 0)

		// rename to the conflicting name while also changing membership: the
		// duplicate name must fail the whole request with 409
		modResp = fleet.ModifyLabelResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", atomicLbl.ID),
			&fleet.ModifyLabelPayload{Name: &conflictName, HostIDs: []uint{manualHosts[2].ID}}, http.StatusConflict, &modResp)

		// name and membership must be unchanged (no partial commit)
		getResp = fleet.GetLabelResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", atomicLbl.ID), nil, http.StatusOK, &getResp)
		assert.Equal(t, atomicLbl.Name, getResp.Label.Name)
		assert.ElementsMatch(t, []uint{manualHosts[0].ID, manualHosts[1].ID}, getResp.Label.HostIDs)

		// no new activity must have been recorded for the failed request
		assert.Equal(t, lastActID, s.lastActivityMatches("", "", 0))

		// a valid rename that also changes membership must commit both and record
		// the edited_label activity
		modResp = fleet.ModifyLabelResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", atomicLbl.ID),
			&fleet.ModifyLabelPayload{Name: new("atomic_target_renamed"), HostIDs: []uint{manualHosts[2].ID}}, http.StatusOK, &modResp)
		assert.Equal(t, "atomic_target_renamed", modResp.Label.Name)
		assert.ElementsMatch(t, []uint{manualHosts[2].ID}, modResp.Label.HostIDs)
		require.Greater(t, s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedLabel{}.ActivityName(), "", 0), lastActID)
	})

	t.Run("IdP Labels", func(t *testing.T) {
		// Add some SCIM users
		mysqltest.ExecAdhocSQL(
			t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO scim_users (id, user_name, department) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?), (?, ?, ?), (?, ?, ?)",
					1,
					"no_groups_no_department",
					nil,
					2,
					"one_group_good_department",
					"department_good",
					3,
					"all_the_groups_good_department",
					"department_good",
					4,
					"wrong_groups_wrong_department",
					"department_other",
					5,
					"no_groups_with_department",
					"department_other_2",
				)
				return err
			},
		)
		// Add some SCIM groups
		mysqltest.ExecAdhocSQL(
			t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO scim_groups (id, display_name) VALUES (?, ?), (?, ?), (?, ?)",
					1,
					"group_good",
					2,
					"group_bad",
					3,
					"group_great",
				)
				return err
			},
		)
		// Add some SCIM group memberships
		mysqltest.ExecAdhocSQL(
			t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO scim_user_group (scim_user_id, group_id) VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)",
					2, 1, // "one_group" -> "group_good"
					3, 1, // "all_the_groups" -> "group_good"
					3, 2, // "all_the_groups" -> "group_bad"
					3, 3, // "all_the_groups" -> "group_great"
					4, 2, // "wrong_groups" -> "group_bad"
				)
				return err
			},
		)
		// Add some host->scim user mappings
		mysqltest.ExecAdhocSQL(
			t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					context.Background(),
					"INSERT INTO host_scim_user (host_id, scim_user_id) VALUES (?, ?), (?, ?), (?, ?), (?, ?), (?, ?), (?, ?)",
					hosts[0].ID, 1,
					hosts[1].ID, 2,
					hosts[2].ID, 3,
					hosts[3].ID, 2,
					hosts[4].ID, 4,
					hosts[6].ID, 5,
				)
				return err
			},
		)

		t.Run("IdP Group Label", func(t *testing.T) {
			// Create a label for an IdP group
			criteria := &fleet.HostVitalCriteria{
				Vital: new("end_user_idp_group"),
				Value: new("group_good"),
			}

			labelParams := fleet.CreateLabelRequest{
				LabelPayload: fleet.LabelPayload{
					Name:     "Test IdP Group Label",
					Criteria: criteria,
				},
			}

			labelResp := fleet.CreateLabelResponse{}
			s.DoJSON("POST", "/api/latest/fleet/labels", labelParams, http.StatusOK, &labelResp)
			require.NotNil(t, labelResp.Label)

			filter := fleet.TeamFilter{User: test.UserAdmin}
			label, _, err := s.ds.Label(context.Background(), labelResp.Label.Label.ID, filter)
			require.NoError(t, err)

			// Verify that the query and values are correct.
			// Test parsing the criteria
			query, queryValues, err := label.CalculateHostVitalsQuery()
			require.NoError(t, err)
			queryValuesJson, err := json.Marshal(queryValues)
			require.NoError(t, err)

			// Compare whitespace-normalized SQL: the IdP join fragment is a multi-line
			// raw string whose indentation is irrelevant to the query's meaning.
			assert.Equal(t,
				"SELECT %s FROM %s JOIN host_scim_user ON (hosts.id = host_scim_user.host_id) JOIN scim_users ON (host_scim_user.scim_user_id = scim_users.id) LEFT JOIN ( WITH RECURSIVE scim_user_group_expanded AS ( SELECT scim_user_id, group_id FROM scim_user_group WHERE scim_user_id IN (SELECT scim_user_id FROM host_scim_user) UNION SELECT e.scim_user_id, gg.parent_group_id AS group_id FROM scim_user_group_expanded e JOIN scim_group_group gg ON gg.child_group_id = e.group_id ) SELECT scim_user_id, group_id FROM scim_user_group_expanded ) scim_user_group ON (host_scim_user.scim_user_id = scim_user_group.scim_user_id) LEFT JOIN scim_groups ON (scim_user_group.group_id = scim_groups.id) WHERE scim_groups.display_name = ? GROUP BY hosts.id",
				strings.Join(strings.Fields(query), " "))
			assert.JSONEq(t, `["group_good"]`, string(queryValuesJson))

			// Update label membership.
			_, _, err = s.ds.UpdateLabelMembershipByHostCriteria(context.Background(), label)
			require.NoError(t, err)

			// Verify that the label has the correct hosts.
			// Check that the label has the correct hosts
			//
			// host 1 shouldn't be returned because its scim user has no groups
			// host 2 should be returned because its scim user has the "group_good" group
			// host 3 should be returned because it has a scim user with the "group_good" group
			// host 4 should be returned because it has a scim user with the "group_good" group
			// host 5 shouldn't be returned because its scim user only has the "group_bad" group
			// host 6 shouldn't be returned because it has no scim user
			//
			hostsInLabel, err := s.ds.ListHostsInLabel(context.Background(), filter, label.ID, fleet.HostListOptions{})
			require.NoError(t, err)
			require.Len(t, hostsInLabel, 3)
			require.ElementsMatch(t, []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID}, []uint{hostsInLabel[0].ID, hostsInLabel[1].ID, hostsInLabel[2].ID})

			// Check that the label has the correct host count
			label, _, err = s.ds.Label(context.Background(), labelResp.Label.Label.ID, filter)
			require.NoError(t, err)
			assert.Equal(t, 3, label.HostCount)
		})

		t.Run("IdP Department Label", func(t *testing.T) {
			// Create a label for an IdP department
			criteria := &fleet.HostVitalCriteria{
				Vital: new("end_user_idp_department"),
				Value: new("department_good"),
			}

			labelParams := fleet.CreateLabelRequest{
				LabelPayload: fleet.LabelPayload{
					Name:     "Test IdP Department Label",
					Criteria: criteria,
				},
			}

			labelResp := fleet.CreateLabelResponse{}
			s.DoJSON("POST", "/api/latest/fleet/labels", labelParams, http.StatusOK, &labelResp)
			require.NotNil(t, labelResp.Label)

			filter := fleet.TeamFilter{User: test.UserAdmin}
			label, _, err := s.ds.Label(context.Background(), labelResp.Label.Label.ID, filter)
			require.NoError(t, err)

			// Verify that the query and values are correct.
			// Test parsing the criteria
			query, queryValues, err := label.CalculateHostVitalsQuery()
			require.NoError(t, err)
			queryValuesJson, err := json.Marshal(queryValues)
			require.NoError(t, err)

			// Compare whitespace-normalized SQL (see the IdP Group Label subtest above).
			assert.Equal(t,
				"SELECT %s FROM %s JOIN host_scim_user ON (hosts.id = host_scim_user.host_id) JOIN scim_users ON (host_scim_user.scim_user_id = scim_users.id) LEFT JOIN ( WITH RECURSIVE scim_user_group_expanded AS ( SELECT scim_user_id, group_id FROM scim_user_group WHERE scim_user_id IN (SELECT scim_user_id FROM host_scim_user) UNION SELECT e.scim_user_id, gg.parent_group_id AS group_id FROM scim_user_group_expanded e JOIN scim_group_group gg ON gg.child_group_id = e.group_id ) SELECT scim_user_id, group_id FROM scim_user_group_expanded ) scim_user_group ON (host_scim_user.scim_user_id = scim_user_group.scim_user_id) LEFT JOIN scim_groups ON (scim_user_group.group_id = scim_groups.id) WHERE scim_users.department = ? GROUP BY hosts.id",
				strings.Join(strings.Fields(query), " "))
			assert.JSONEq(t, `["department_good"]`, string(queryValuesJson))

			// Update label membership.
			_, _, err = s.ds.UpdateLabelMembershipByHostCriteria(context.Background(), label)
			require.NoError(t, err)

			// Verify that the label has the correct hosts.
			hostsInLabel, err := s.ds.ListHostsInLabel(context.Background(), filter, label.ID, fleet.HostListOptions{})
			require.NoError(t, err)
			require.Len(t, hostsInLabel, 3)
			require.ElementsMatch(t, []uint{hosts[1].ID, hosts[2].ID, hosts[3].ID}, []uint{hostsInLabel[0].ID, hostsInLabel[1].ID, hostsInLabel[2].ID})

			// Check that the label has the correct host count
			label, _, err = s.ds.Label(context.Background(), labelResp.Label.Label.ID, filter)
			require.NoError(t, err)
			assert.Equal(t, 3, label.HostCount)

			t.Run("No Groups", func(t *testing.T) {
				// Create a label for IdP department to test users with department but no groups
				criteria := &fleet.HostVitalCriteria{
					Vital: new("end_user_idp_department"),
					Value: new("department_other_2"),
				}
				labelParams := fleet.CreateLabelRequest{
					LabelPayload: fleet.LabelPayload{
						Name:     "Test IdP Department Label 2",
						Criteria: criteria,
					},
				}

				labelResp := fleet.CreateLabelResponse{}
				s.DoJSON("POST", "/api/latest/fleet/labels", labelParams, http.StatusOK, &labelResp)
				require.NotNil(t, labelResp.Label)

				label, _, err = s.ds.Label(context.Background(), labelResp.Label.Label.ID, filter)
				require.NoError(t, err)

				// Update label membership.
				_, _, err = s.ds.UpdateLabelMembershipByHostCriteria(context.Background(), label)
				require.NoError(t, err)

				// Verify that the label has the correct hosts.
				hostsInLabel, err := s.ds.ListHostsInLabel(context.Background(), filter, label.ID, fleet.HostListOptions{})
				require.NoError(t, err)
				require.Len(t, hostsInLabel, 1)
				// host 7 is mapped to user 5 which matches the department but has no groups.
				require.ElementsMatch(t, []uint{hosts[6].ID}, []uint{hostsInLabel[0].ID})

				// Check that the label has the correct host count
				label, _, err = s.ds.Label(context.Background(), label.ID, filter)
				require.NoError(t, err)
				assert.Equal(t, 1, label.HostCount)
			})
		})
	})

	t.Run("Sort by order_key", func(t *testing.T) {
		ctx := context.Background()

		// Create three teams so each host has a distinct team_id and team_name.
		// Names sort A < B < C and team IDs are assigned in creation order.
		teamPrefix := strings.ReplaceAll(t.Name(), "/", "_")
		teamA, err := s.ds.NewTeam(ctx, &fleet.Team{Name: teamPrefix + "-team-A"})
		require.NoError(t, err)
		teamB, err := s.ds.NewTeam(ctx, &fleet.Team{Name: teamPrefix + "-team-B"})
		require.NoError(t, err)
		teamC, err := s.ds.NewTeam(ctx, &fleet.Team{Name: teamPrefix + "-team-C"})
		require.NoError(t, err)
		teamIDs := []*uint{&teamA.ID, &teamB.ID, &teamC.ID}

		// Each sortHost[i] is set up with field values that sort in the same direction
		// as the index: sortHost[0] is "smallest", sortHost[2] is "largest", for every
		// orderable field below. This means ASC order is always [h0, h1, h2] and DESC
		// order is always [h2, h1, h0], which keeps the per-field test cases compact.
		base := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
		platforms := []string{"darwin", "linux", "windows"}
		sortHosts := make([]*fleet.Host, 3)
		for i := range 3 {
			h, err := s.ds.NewHost(ctx, &fleet.Host{
				OsqueryHostID:               new(fmt.Sprintf("sort-osq-%d", i)),
				NodeKey:                     new(fmt.Sprintf("sort-nk-%d", i)),
				UUID:                        fmt.Sprintf("aaaaaaaa-0000-0000-0000-00000000000%d", i+1),
				Hostname:                    fmt.Sprintf("sort-host-%d", i),
				ComputerName:                fmt.Sprintf("sort-comp-%d", i),
				HardwareSerial:              fmt.Sprintf("sort-ser-%d", i),
				Platform:                    platforms[i],
				PlatformLike:                fmt.Sprintf("sort-plike-%d", i),
				OsqueryVersion:              fmt.Sprintf("%d.0.0", i+1),
				OSVersion:                   fmt.Sprintf("OS-%d.0", i+1),
				Uptime:                      time.Duration(i+1) * time.Hour,
				Memory:                      int64(i+1) * 1024,
				DistributedInterval:         uint(i+1) * 10,
				LoggerTLSPeriod:             uint(i+1) * 100,
				ConfigTLSRefresh:            uint(i+1) * 5,
				DetailUpdatedAt:             base.Add(time.Duration(i+1) * time.Hour),
				LabelUpdatedAt:              base.Add(time.Duration(i+1) * 2 * time.Hour),
				PolicyUpdatedAt:             base.Add(time.Duration(i+1) * 3 * time.Hour),
				RefetchCriticalQueriesUntil: new(base.Add(time.Duration(i+1) * 4 * time.Hour)),
				RefetchRequested:            i == 2, // only the "largest" host has refetch_requested = true
				TeamID:                      teamIDs[i],
			})
			require.NoError(t, err)
			sortHosts[i] = h
		}

		// Set columns not handled by NewHost via direct UPDATEs.
		for i, h := range sortHosts {
			mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(
					ctx, `
					UPDATE hosts SET
						created_at = ?,
						updated_at = ?,
						last_enrolled_at = ?,
						last_restarted_at = ?,
						primary_ip_id = ?,
						primary_ip = ?,
						primary_mac = ?,
						public_ip = ?,
						timezone = ?,
						build = ?,
						code_name = ?,
						cpu_type = ?,
						cpu_subtype = ?,
						cpu_brand = ?,
						cpu_physical_cores = ?,
						cpu_logical_cores = ?,
						hardware_vendor = ?,
						hardware_model = ?,
						hardware_version = ?
					WHERE id = ?`,
					base.Add(time.Duration(i+1)*5*time.Hour),
					base.Add(time.Duration(i+1)*6*time.Hour),
					base.Add(time.Duration(i+1)*7*time.Hour),
					base.Add(time.Duration(i+1)*8*time.Hour),
					uint(i+1)*100, //nolint:gosec // ignore G115
					fmt.Sprintf("10.0.0.%d", i+1),
					fmt.Sprintf("aa:bb:cc:00:00:0%d", i+1),
					fmt.Sprintf("8.0.0.%d", i+1),
					fmt.Sprintf("UTC-%d", i),
					fmt.Sprintf("sort-build-%d", i),
					fmt.Sprintf("sort-code-%d", i),
					fmt.Sprintf("sort-ct-%d", i),
					fmt.Sprintf("sort-cs-%d", i),
					fmt.Sprintf("sort-cb-%d", i),
					i+1,
					(i+1)*2,
					fmt.Sprintf("sort-hv-%d", i),
					fmt.Sprintf("sort-hm-%d", i),
					fmt.Sprintf("sort-hver-%d", i),
					h.ID,
				)
				return err
			})
		}

		// Populate joined tables (host_disks, host_seen_times, host_updates,
		// host_issues, host_emails) with values that also sort by host index.
		for i, h := range sortHosts {
			mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(ctx, `
					INSERT INTO host_disks (host_id, gigs_disk_space_available, percent_disk_space_available, gigs_total_disk_space)
					VALUES (?, ?, ?, ?)`,
					h.ID, float64((i+1)*100), float64((i+1)*10), float64((i+1)*1000))
				return err
			})
			mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(ctx,
					`UPDATE host_seen_times SET seen_time = ? WHERE host_id = ?`,
					base.Add(time.Duration(i+1)*9*time.Hour), h.ID)
				return err
			})
			mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(ctx,
					`INSERT INTO host_updates (host_id, software_updated_at) VALUES (?, ?)`,
					h.ID, base.Add(time.Duration(i+1)*10*time.Hour))
				return err
			})
			mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(ctx, `
					INSERT INTO host_issues (host_id, failing_policies_count, critical_vulnerabilities_count, total_issues_count)
					VALUES (?, ?, ?, ?)`,
					h.ID, uint(i+1), uint(i+1)*2, uint(i+1)*3) //nolint:gosec // ignore G115
				return err
			})
			mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
				_, err := db.ExecContext(ctx,
					`INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
					h.ID, fmt.Sprintf("sort-%d@example.com", i), "src")
				return err
			})
		}

		// Create a dynamic label and add the three hosts to it.
		var createResp fleet.CreateLabelResponse
		s.DoJSON("POST", "/api/latest/fleet/labels",
			&fleet.LabelPayload{Name: teamPrefix + "-sort", Query: "select 1"}, http.StatusOK, &createResp)
		sortLabel := createResp.Label.Label
		for _, h := range sortHosts {
			require.NoError(t, s.ds.RecordLabelQueryExecutions(ctx, h,
				map[uint]*bool{sortLabel.ID: new(true)}, time.Now(), false))
		}

		ascIDs := []uint{sortHosts[0].ID, sortHosts[1].ID, sortHosts[2].ID}
		descIDs := []uint{sortHosts[2].ID, sortHosts[1].ID, sortHosts[0].ID}
		labelHostsURL := fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", sortLabel.ID)

		// orderKeys lists every field for which sortHosts is set up to produce a
		// strict, deterministic ASC ordering of [h0, h1, h2]. Each field gets its
		// own subtest verifying both ASC and DESC orderings.
		orderKeys := []string{
			"id", "osquery_host_id", "created_at", "updated_at", "detail_updated_at",
			"hostname", "uuid", "platform", "osquery_version", "os_version", "build",
			"platform_like", "code_name", "uptime", "memory", "cpu_type", "cpu_subtype",
			"cpu_brand", "cpu_physical_cores", "cpu_logical_cores",
			"hardware_vendor", "hardware_model", "hardware_version", "hardware_serial",
			"computer_name", "primary_ip_id", "distributed_interval", "logger_tls_period",
			"config_tls_refresh", "primary_ip", "primary_mac", "label_updated_at",
			"last_enrolled_at", "refetch_critical_queries_until", "team_id",
			"policy_updated_at", "public_ip",
			"gigs_disk_space_available", "percent_disk_space_available",
			"gigs_total_disk_space", "seen_time", "software_updated_at",
			"last_restarted_at", "timezone", "team_name",
			"failing_policies_count", "critical_vulnerabilities_count",
			"total_issues_count", "issues",
			"device_mapping",
			"display_name",
		}
		for _, key := range orderKeys {
			t.Run(key, func(t *testing.T) {
				params := []string{"order_key", key, "order_direction", "asc"}
				if key == "device_mapping" {
					params = append(params, "device_mapping", "true")
				}

				var resp listHostsResponse
				s.DoJSON("GET", labelHostsURL, nil, http.StatusOK, &resp, params...)
				require.Len(t, resp.Hosts, 3)
				gotAsc := []uint{resp.Hosts[0].ID, resp.Hosts[1].ID, resp.Hosts[2].ID}
				assert.Equal(t, ascIDs, gotAsc, "asc order mismatch")

				params[3] = "desc"
				resp = listHostsResponse{}
				s.DoJSON("GET", labelHostsURL, nil, http.StatusOK, &resp, params...)
				require.Len(t, resp.Hosts, 3)
				gotDesc := []uint{resp.Hosts[0].ID, resp.Hosts[1].ID, resp.Hosts[2].ID}
				assert.Equal(t, descIDs, gotDesc, "desc order mismatch")
			})
		}

		// refetch_requested is a bool, so only h2 (true) has a unique value. ASC must
		// place h2 last; DESC must place h2 first. The order between h0 and h1 (both
		// false) is not deterministic, so we only assert the position of h2.
		t.Run("refetch_requested", func(t *testing.T) {
			var resp listHostsResponse
			s.DoJSON("GET", labelHostsURL, nil, http.StatusOK, &resp,
				"order_key", "refetch_requested", "order_direction", "asc")
			require.Len(t, resp.Hosts, 3)
			assert.Equal(t, sortHosts[2].ID, resp.Hosts[2].ID, "asc: h2 should be last")

			resp = listHostsResponse{}
			s.DoJSON("GET", labelHostsURL, nil, http.StatusOK, &resp,
				"order_key", "refetch_requested", "order_direction", "desc")
			require.Len(t, resp.Hosts, 3)
			assert.Equal(t, sortHosts[2].ID, resp.Hosts[0].ID, "desc: h2 should be first")
		})

		// Cursor pagination (`after`) injects the order_key into the WHERE
		// clause; SELECT-list aliases like team_name would error there. Verify
		// that paging through team_name with a cursor returns the expected
		// hosts and does not error.
		t.Run("team_name with after cursor", func(t *testing.T) {
			var resp listHostsResponse
			s.DoJSON("GET", labelHostsURL, nil, http.StatusOK, &resp,
				"order_key", "team_name", "order_direction", "asc",
				"after", teamA.Name, "per_page", "10")
			require.Len(t, resp.Hosts, 2)
			assert.Equal(t, sortHosts[1].ID, resp.Hosts[0].ID)
			assert.Equal(t, sortHosts[2].ID, resp.Hosts[1].ID)
		})
	})
}

// Sanity test to make sure fleet/labels/<all>/hosts and fleet/hosts return the same thing.
func (s *integrationTestSuite) TestListHostsByLabel() {
	t := s.T()

	lblIDs, err := s.ds.LabelIDsByName(context.Background(), []string{"All Hosts"}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, lblIDs, 1)
	labelID := lblIDs["All Hosts"]

	hosts := s.createHosts(t, "darwin")
	host := hosts[0]

	// Update label
	mysqltest.ExecAdhocSQL(
		t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(
				context.Background(),
				"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, (SELECT id FROM labels WHERE name = 'All Hosts' AND label_type = 1))",
				host.ID,
			)
			return err
		},
	)

	// set disk space information
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), host.ID, 10.0, 2.0, 500.0, nil)) // low disk

	// Update host fields
	host.Uptime = 30 * time.Second
	host.RefetchRequested = true
	host.OSVersion = "macOS 14.2"
	host.Build = "abc"
	host.PlatformLike = "darwin"
	host.CodeName = "sky"
	host.Memory = 1000
	host.CPUType = "arm64"
	host.CPUSubtype = "ARM64e"
	host.CPUBrand = "Apple M2 Pro"
	host.CPUPhysicalCores = 12
	host.CPULogicalCores = 14
	host.HardwareVendor = "Apple Inc."
	host.HardwareModel = "Mac14,10"
	host.HardwareVersion = "23"
	host.HardwareSerial = "ABC123"
	host.ComputerName = "MBP"
	host.PublicIP = "1.1.1.1"
	host.PrimaryIP = "10.10.10.10"
	host.PrimaryMac = "11:22:33"
	host.DistributedInterval = 10
	host.ConfigTLSRefresh = 9
	host.OsqueryVersion = "5.10"
	err = s.ds.UpdateHost(context.Background(), host)
	require.NoError(t, err)

	// Add team
	team, err := s.ds.NewTeam(
		context.Background(), &fleet.Team{
			Name: uuid.New().String(),
		},
	)
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

	// Add pack
	_, err = s.ds.NewPack(
		context.Background(), &fleet.Pack{
			Name: t.Name(),
			Hosts: []fleet.Target{
				{
					Type:     fleet.TargetHost,
					TargetID: hosts[0].ID,
				},
			},
		},
	)
	require.NoError(t, err)

	// Add policy
	qr, err := s.ds.NewQuery(
		context.Background(), &fleet.Query{
			Name:           t.Name(),
			Description:    "Some description",
			Query:          "select * from osquery;",
			ObserverCanRun: true,
			Logging:        fleet.LoggingSnapshot,
		},
	)
	require.NoError(t, err)

	gpParams := fleet.GlobalPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some global resolution",
	}
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NoError(
		t,
		errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{gpResp.Policy.ID: new(false)}, time.Now(), false, nil)),
	)

	// Add MDM info
	require.NoError(
		t,
		s.ds.SetOrUpdateMDMData(
			context.Background(), host.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "", false,
		),
	)

	// Add device mapping
	require.NoError(
		t, s.ds.ReplaceHostDeviceMapping(
			context.Background(), host.ID, []*fleet.HostDeviceMapping{
				{HostID: hosts[0].ID, Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
				{HostID: hosts[0].ID, Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
			}, fleet.DeviceMappingGoogleChromeProfiles,
		),
	)

	// Now do the actual API calls that we will compare.
	var hostsResp, labelsResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hostsResp, "device_mapping", "true")
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", labelID), nil, http.StatusOK, &labelsResp, "device_mapping", "true")

	// Converting to formatted JSON for easier diffs
	hostsJson, _ := json.MarshalIndent(hostsResp, "", "  ")
	labelsJson, _ := json.MarshalIndent(labelsResp, "", "  ")
	assert.JSONEq(t, string(hostsJson), string(labelsJson))

	// Do request with include_device_status, since it's an additional feature
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hostsResp, "device_mapping", "true", "include_device_status", "true")
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", labelID), nil, http.StatusOK, &labelsResp, "device_mapping", "true", "include_device_status", "true")

	// Converting to formatted JSON for easier diffs
	hostsJson, _ = json.MarshalIndent(hostsResp, "", "  ")
	labelsJson, _ = json.MarshalIndent(labelsResp, "", "  ")
	assert.JSONEq(t, string(hostsJson), string(labelsJson))
}

func (s *integrationTestSuite) TestLabelSpecs() {
	t := s.T()

	// list label specs, only those of the built-ins
	var listResp fleet.GetLabelSpecsResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/labels", nil, http.StatusOK, &listResp)
	assert.NotEmpty(t, listResp.Specs)
	for _, spec := range listResp.Specs {
		assert.Equal(t, fleet.LabelTypeBuiltIn, spec.LabelType)
	}
	builtInsCount := len(listResp.Specs)

	name := strings.ReplaceAll(t.Name(), "/", "_")
	// apply an invalid label spec - dynamic membership with host specified
	var applyResp fleet.ApplyLabelSpecsResponse
	s.DoJSON(
		"POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{
			Specs: []*fleet.LabelSpec{
				{
					Name:                name,
					Query:               "select 1",
					Platform:            "linux",
					LabelMembershipType: fleet.LabelMembershipTypeDynamic,
					Hosts:               []string{"abc"},
				},
			},
		}, http.StatusUnprocessableEntity, &applyResp,
	)

	// apply an invalid label spec - invalid platform
	s.DoJSON(
		"POST",
		"/api/latest/fleet/spec/labels",
		fleet.ApplyLabelSpecsRequest{
			Specs: []*fleet.LabelSpec{
				{
					Name:                name,
					Query:               "select 1",
					Platform:            "bados",
					LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				},
			},
		},
		http.StatusUnprocessableEntity,
		&applyResp,
	)

	// apply a valid label spec - generic "linux" platform
	s.DoJSON(
		"POST",
		"/api/latest/fleet/spec/labels",
		fleet.ApplyLabelSpecsRequest{
			Specs: []*fleet.LabelSpec{
				{
					Name:                name + "_linux",
					Query:               "select 1",
					Platform:            "linux",
					LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				},
			},
		},
		http.StatusOK,
		&applyResp,
	)

	// apply a valid label spec - manual membership without hosts specified (preserves existing membership)
	s.DoJSON(
		"POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{
			Specs: []*fleet.LabelSpec{
				{
					Name:                name,
					LabelMembershipType: fleet.LabelMembershipTypeManual,
				},
			},
		}, http.StatusOK, &applyResp,
	)

	// apply an invalid label spec - builtin label type
	s.DoJSON("POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{
		Specs: []*fleet.LabelSpec{
			{
				Name:                name,
				Query:               "select 1",
				Platform:            "linux",
				LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				LabelType:           fleet.LabelTypeBuiltIn,
			},
		},
	}, http.StatusUnprocessableEntity, &applyResp)

	// apply an invalid label spec - builtin label name
	for n := range fleet.ReservedLabelNames() {
		s.DoJSON("POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{
			Specs: []*fleet.LabelSpec{
				{
					Name:                n,
					Query:               "select 1",
					Platform:            "linux",
					LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				},
			},
		}, http.StatusUnprocessableEntity, &applyResp)
	}

	// apply a valid label spec
	s.DoJSON("POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{
		Specs: []*fleet.LabelSpec{
			{
				Name:                name,
				Query:               "select 1",
				Platform:            "centos",
				LabelMembershipType: fleet.LabelMembershipTypeDynamic,
			},
		},
	}, http.StatusOK, &applyResp)

	// list label specs, has the newly created ones
	s.DoJSON("GET", "/api/latest/fleet/spec/labels", nil, http.StatusOK, &listResp)
	assert.Len(t, listResp.Specs, builtInsCount+2)

	// get a specific label spec
	var getResp fleet.GetLabelSpecResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/spec/labels/%s", url.PathEscape(name)), nil, http.StatusOK, &getResp)
	assert.Equal(t, name, getResp.Spec.Name)
	assert.NotEqual(t, 0, getResp.Spec.ID)

	// the generic "linux" platform round-trips through the spec endpoints
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/spec/labels/%s", url.PathEscape(name+"_linux")), nil, http.StatusOK, &getResp)
	assert.Equal(t, "linux", getResp.Spec.Platform)

	// get a non-existing label spec
	s.DoJSON("GET", "/api/latest/fleet/spec/labels/zzz", nil, http.StatusNotFound, &getResp)
}

func (s *integrationTestSuite) TestAddingRemovingManualLabels() {
	t := s.T()
	ctx := context.Background()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	newGlobalUserFunc := func(email string, globalRole string) *fleet.User {
		user := &fleet.User{
			Name:       email,
			Email:      email,
			GlobalRole: &globalRole,
		}
		err = user.SetPassword(test.GoodPassword, 10, 10)
		require.NoError(t, err)
		user, err = s.ds.NewUser(context.Background(), user)
		require.NoError(t, err)
		return user
	}
	newTeamUserFunc := func(email string, team *fleet.Team, teamRole string) *fleet.User {
		user := &fleet.User{
			Name:  email,
			Email: email,
			Teams: []fleet.UserTeam{
				{
					Team: *team,
					Role: teamRole,
				},
			},
		}
		err = user.SetPassword(test.GoodPassword, 10, 10)
		require.NoError(t, err)
		user, err = s.ds.NewUser(context.Background(), user)
		require.NoError(t, err)
		return user
	}
	globalObserver := newGlobalUserFunc("global.observer@example.com", fleet.RoleObserver)
	teamAdmin := newTeamUserFunc("team.admin@example.com", team1, fleet.RoleAdmin)
	teamObserver := newTeamUserFunc("team.observer@example.com", team1, fleet.RoleObserver)

	newHostFunc := func(name string, teamID *uint) *fleet.Host {
		host, err := s.ds.NewHost(ctx, &fleet.Host{
			NodeKey:  new(name),
			UUID:     name,
			Hostname: "foo.local." + name,
			TeamID:   teamID,
		})
		require.NoError(t, err)
		require.NotNil(t, host)
		return host
	}
	host1 := newHostFunc("host1", nil)
	host2 := newHostFunc("host2", nil)
	teamHost2 := newHostFunc("teamHost2", &team1.ID)

	ls, err := s.ds.LabelIDsByName(ctx, []string{"All Hosts"}, fleet.TeamFilter{})
	require.NoError(t, err)
	require.Len(t, ls, 1)
	allHostsLabelID, ok := ls["All Hosts"]
	require.True(t, ok)
	require.NotZero(t, allHostsLabelID)

	dynamicLabel1, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                "dynamicLabel1",
		Query:               "SELECT 1;",
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	manualLabel1, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                "manualLabel1",
		Query:               "SELECT 2;",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)
	manualLabel2, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                "manualLabel2",
		Query:               "SELECT 3;",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)

	err = s.ds.RecordLabelQueryExecutions(context.Background(), host1, map[uint]*bool{allHostsLabelID: new(true)}, time.Now(), false)
	require.NoError(t, err)

	getHostLabels := func(host *fleet.Host) []string {
		var hostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
		var labels []string
		for _, label := range hostResp.Host.Labels {
			labels = append(labels, label.Name)
		}
		return labels
	}

	hostLabels1 := getHostLabels(host1)
	require.Len(t, hostLabels1, 1)
	require.Equal(t, "All Hosts", hostLabels1[0])

	// No labels or empty labels is a no-op.
	var addLabelsToHostResp addLabelsToHostResponse
	s.DoJSON(
		"POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID),
		json.RawMessage(`{}`), http.StatusOK, &addLabelsToHostResp,
	)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{},
	}, http.StatusOK, &addLabelsToHostResp)
	var removeLabelsFromHostResp fleet.RemoveLabelsFromHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{},
	}, http.StatusOK, &removeLabelsFromHostResp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{""},
	}, http.StatusOK, &addLabelsToHostResp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"", ""},
	}, http.StatusOK, &addLabelsToHostResp)

	// A dynamic buitin label should fail to be added.
	res := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"All Hosts"},
	}, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels are dynamic: \"All Hosts\". Dynamic labels can not be assigned to hosts manually.")
	// An inexistent label should fail to be added.
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"manualLabel2", "does not exist"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels not found: \"does not exist\". All labels must exist")
	// Multiple inexistent labels should fail to be added.
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"manualLabel2", "does not exist", "does not exist 2"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels not found: \"does not exist\", \"does not exist 2\". All labels must exist")
	// A dynamic non-builtin label should fail to be added.
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{dynamicLabel1.Name},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels are dynamic: \"dynamicLabel1\". Dynamic labels can not be assigned to hosts manually.")
	// Multiple dynamic labels should fail to be added.
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"All Hosts", dynamicLabel1.Name},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels are dynamic: \"All Hosts\", \"dynamicLabel1\". Dynamic labels can not be assigned to hosts manually.")

	// A dynamic builtin label should fail to be deleted.
	res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{"All Hosts"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't remove labels. Labels are dynamic: \"All Hosts\". Dynamic labels can not be assigned to hosts manually.")
	// An inexistent label should fail to be deleted.
	res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel2.Name, "does not exist"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't remove labels. Labels not found: \"does not exist\". All labels must exist")
	// Multiple inexistent labels should fail to be deleted.
	res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel2.Name, "does not exist", "does not exist 2"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't remove labels. Labels not found: \"does not exist\", \"does not exist 2\". All labels must exist")
	// Multiple dynamic labels should fail to be deleted.
	res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel2.Name, dynamicLabel1.Name, "All Hosts"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't remove labels. Labels are dynamic: \"All Hosts\", \"dynamicLabel1\". Dynamic labels can not be assigned to hosts manually.")

	// Add two manual labels to a host.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	// Add the same manual labels to a host should succeed.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &addLabelsToHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 3)
	require.Equal(t, "All Hosts", hostLabels1[0])
	require.Equal(t, manualLabel1.Name, hostLabels1[1])
	require.Equal(t, manualLabel2.Name, hostLabels1[2])
	hostLabels2 := getHostLabels(host2)
	require.Empty(t, hostLabels2)

	// Remove the two manual labels from the host.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)
	// Remove the same manual labels from the host again.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 1)
	require.Equal(t, "All Hosts", hostLabels1[0])

	// Add same label, should deduplicate.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 2)
	require.Equal(t, "All Hosts", hostLabels1[0])
	require.Equal(t, manualLabel1.Name, hostLabels1[1])

	// Adding an already added label should work.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &addLabelsToHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 3)
	require.Equal(t, "All Hosts", hostLabels1[0])
	require.Equal(t, manualLabel1.Name, hostLabels1[1])
	require.Equal(t, manualLabel2.Name, hostLabels1[2])

	// Delete same label, should deduplicate.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel1.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	// Deleting a non-member label (manualLabel1) should work.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 1)
	require.Equal(t, "All Hosts", hostLabels1[0])

	// Add to non-existent host
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", 999), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusNotFound, &addLabelsToHostResp)
	// Delete from non-existent host
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", 999), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusNotFound, &removeLabelsFromHostResp)

	// Add labels to team host.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)

	// A global observer should not be allowed to add/remove a label.
	oldToken := s.token
	s.token = s.getTestToken(globalObserver.Email, test.GoodPassword)
	t.Cleanup(func() {
		s.token = oldToken
	})
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &removeLabelsFromHostResp)

	// A team observer should not be allowed to add/remove a label.
	s.token = s.getTestToken(teamObserver.Email, test.GoodPassword)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &removeLabelsFromHostResp)

	// A team admin should not be allowed to add/remove a label for a global host.
	s.token = s.getTestToken(teamAdmin.Email, test.GoodPassword)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &removeLabelsFromHostResp)

	// A team admin should be allowed to add/remove a label for a team host.
	s.token = s.getTestToken(teamAdmin.Email, test.GoodPassword)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	teamHost2Labels := getHostLabels(teamHost2)
	require.Len(t, teamHost2Labels, 1)
	require.Equal(t, manualLabel1.Name, teamHost2Labels[0])
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)
	teamHost2Labels = getHostLabels(teamHost2)
	require.Empty(t, teamHost2Labels)
}

// TestLabelScopePremiumGate verifies that all policy label scope fields
// (include_any, include_all, exclude_any, exclude_all) are premium-gated on
// every entry point for the free-tier (core) server.
func (s *integrationTestSuite) TestLabelScopePremiumGate() {
	t := s.T()

	// One label suffices to exercise the gate.
	var lblResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.LabelPayload{Name: uuid.NewString(), Query: "SELECT 1"}, http.StatusOK, &lblResp)
	lbl := lblResp.Label

	// All policy label scope fields on GlobalPolicyRequest are tagged
	// `premium:"true"`, so the endpoint decoder rejects with 400 before reaching
	// the service handler.
	for _, req := range []fleet.GlobalPolicyRequest{
		{Name: "premium-include-all-" + t.Name(), Query: "SELECT 1", LabelsIncludeAll: []string{lbl.Name}},
		{Name: "premium-include-any-" + t.Name(), Query: "SELECT 1", LabelsIncludeAny: []string{lbl.Name}},
		{Name: "premium-exclude-all-" + t.Name(), Query: "SELECT 1", LabelsExcludeAll: []string{lbl.Name}},
		{Name: "premium-exclude-any-" + t.Name(), Query: "SELECT 1", LabelsExcludeAny: []string{lbl.Name}},
	} {
		s.DoJSON("POST", "/api/latest/fleet/policies", req, http.StatusBadRequest, &fleet.GlobalPolicyResponse{})
	}

	// A policy without any label scope can still be created on free-tier.
	var freeOK fleet.GlobalPolicyResponse
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
		Name:  "free-no-labels-" + t.Name(),
		Query: "SELECT 1",
	}, http.StatusOK, &freeOK)
	require.NotNil(t, freeOK.Policy)

	// Free-tier should reject PATCHing a policy to add any label scope (middleware → 400).
	for _, payload := range []fleet.ModifyPolicyPayload{
		{LabelsIncludeAll: []string{lbl.Name}},
		{LabelsIncludeAny: []string{lbl.Name}},
		{LabelsExcludeAll: []string{lbl.Name}},
		{LabelsExcludeAny: []string{lbl.Name}},
	} {
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", freeOK.Policy.ID), fleet.ModifyGlobalPolicyRequest{
			ModifyPolicyPayload: payload,
		}, http.StatusBadRequest, &fleet.ModifyGlobalPolicyResponse{})
	}

	for _, payload := range []fleet.QueryPayload{
		{Name: new("q-any-" + t.Name()), Query: new("SELECT 1"), Logging: new(fleet.LoggingSnapshot), LabelsIncludeAny: []string{lbl.Name}},
		{Name: new("q-all-" + t.Name()), Query: new("SELECT 1"), Logging: new(fleet.LoggingSnapshot), LabelsIncludeAll: []string{lbl.Name}},
	} {
		s.DoJSON("POST", "/api/latest/fleet/queries", payload, http.StatusPaymentRequired, &fleet.CreateQueryResponse{})
	}

	// PolicySpec label fields don't carry the `premium:"true"` middleware tag,
	// so the gate fires at the service layer and returns 402 (PaymentRequired).
	for _, spec := range []*fleet.PolicySpec{
		{Name: "spec-include-all-" + t.Name(), Query: "SELECT 1", LabelsIncludeAll: []string{lbl.Name}},
		{Name: "spec-include-any-" + t.Name(), Query: "SELECT 1", LabelsIncludeAny: []string{lbl.Name}},
		{Name: "spec-exclude-all-" + t.Name(), Query: "SELECT 1", LabelsExcludeAll: []string{lbl.Name}},
		{Name: "spec-exclude-any-" + t.Name(), Query: "SELECT 1", LabelsExcludeAny: []string{lbl.Name}},
	} {
		s.DoJSON("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
			Specs: []*fleet.PolicySpec{spec},
		}, http.StatusPaymentRequired, &fleet.ApplyPolicySpecsResponse{})
	}

	// Same for QuerySpec.
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", fleet.ApplyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{{
			Name:             "qspec-premium-gate-" + t.Name(),
			Query:            "SELECT 1",
			Logging:          fleet.LoggingSnapshot,
			LabelsIncludeAll: []string{lbl.Name},
		}},
	}, http.StatusPaymentRequired, &fleet.ApplyQuerySpecsResponse{})
}
