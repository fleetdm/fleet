package service

// Policy tests for the core (no-license) suite.
//
// Belongs here: global and fleet policy CRUD (both the spec and proprietary
// endpoints), cross-fleet access rules and browsing, autofill, a host's policy
// membership, and the policy count bookkeeping kept on hosts.
//
// Does not belong here: the webhook/automation configuration that fires on policy
// failure (integration_core_activities_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestPolicyDeletionLogsActivity() {
	t := s.T()

	admin1 := s.users["admin1@example.com"]
	admin1.GravatarURL = "http://iii.com"
	err := s.ds.SaveUser(context.Background(), &admin1)
	require.NoError(t, err)

	testPolicies := []fleet.PolicyPayload{{
		Name:  "policy1",
		Query: "select * from time;",
	}, {
		Name:  "policy2",
		Query: "select * from osquery_info;",
	}}

	var policyIDs []uint
	for _, policy := range testPolicies {
		var resp fleet.GlobalPolicyResponse
		s.DoJSON("POST", "/api/latest/fleet/policies", policy, http.StatusOK, &resp)
		policyIDs = append(policyIDs, resp.Policy.PolicyData.ID)
	}

	// critical is premium only.
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.PolicyPayload{
		Name:     "policy3",
		Query:    "select * from time;",
		Critical: true,
	}, http.StatusBadRequest, new(struct{}))

	prevActivities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &prevActivities)
	require.GreaterOrEqual(t, len(prevActivities.Activities), 2)

	var deletePoliciesResp fleet.DeleteGlobalPoliciesResponse
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", fleet.DeleteGlobalPoliciesRequest{IDs: policyIDs}, http.StatusOK, &deletePoliciesResp)
	require.Len(t, deletePoliciesResp.Deleted, len(policyIDs))

	newActivities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &newActivities)
	require.Equal(t, len(newActivities.Activities), (len(prevActivities.Activities) + 2))

	var prevDeletes []*fleet.Activity
	for _, a := range prevActivities.Activities {
		if a.Type == "deleted_policy" {
			prevDeletes = append(prevDeletes, a)
		}
	}
	var newDeletes []*fleet.Activity
	for _, a := range newActivities.Activities {
		if a.Type == "deleted_policy" {
			newDeletes = append(newDeletes, a)
		}
	}
	require.Equal(t, len(newDeletes), (len(prevDeletes) + 2))

	type policyDetails struct {
		PolicyID   uint   `json:"policy_id"`
		PolicyName string `json:"policy_name"`
	}
	for _, id := range policyIDs {
		found := false
		for _, d := range newDeletes {
			var details policyDetails
			err := json.Unmarshal([]byte(*d.Details), &details)
			require.NoError(t, err)
			require.NotNil(t, details.PolicyID)
			if id == details.PolicyID {
				found = true
			}

		}
		require.True(t, found)
	}
	for _, p := range testPolicies {
		found := false
		for _, d := range newDeletes {
			var details policyDetails
			err := json.Unmarshal([]byte(*d.Details), &details)
			require.NoError(t, err)
			require.NotNil(t, details.PolicyName)
			if p.Name == details.PolicyName {
				found = true
			}

		}
		require.True(t, found)
	}
}

func (s *integrationTestSuite) TestGlobalPolicies() {
	t := s.T()

	// create 3 hosts
	for i := range 3 {
		_, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   new(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         new(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
		})
		require.NoError(t, err)
	}

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery3",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	// create a global policy
	gpParams := fleet.GlobalPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some global resolution",
	}
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, qr.Name, gpResp.Policy.Name)
	assert.Equal(t, qr.Query, gpResp.Policy.Query)
	assert.Equal(t, qr.Description, gpResp.Policy.Description)
	require.NotNil(t, gpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution", *gpResp.Policy.Resolution)

	// list global policies
	policiesResponse := fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, qr.Name, policiesResponse.Policies[0].Name)
	assert.Equal(t, qr.Query, policiesResponse.Policies[0].Query)
	assert.Equal(t, qr.Description, policiesResponse.Policies[0].Description)

	// invalid order_key returns 422
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusUnprocessableEntity, &fleet.ListGlobalPoliciesResponse{}, "order_key", "invalid")

	// Get an unexistent policy
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", 9999), nil, http.StatusNotFound)

	singlePolicyResponse := fleet.GetPolicyByIDResponse{}
	singlePolicyURL := fmt.Sprintf("/api/latest/fleet/policies/%d", policiesResponse.Policies[0].ID)
	s.DoJSON("GET", singlePolicyURL, nil, http.StatusOK, &singlePolicyResponse)
	assert.Equal(t, qr.Name, singlePolicyResponse.Policy.Name)
	assert.Equal(t, qr.Query, singlePolicyResponse.Policy.Query)
	assert.Equal(t, qr.Description, singlePolicyResponse.Policy.Description)

	listHostsURL := fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d", policiesResponse.Policies[0].ID)
	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 3)

	h1 := listHostsResp.Hosts[0]
	h2 := listHostsResp.Hosts[1]

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Empty(t, listHostsResp.Hosts)

	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: new(true)}, time.Now(), false, nil)))
	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false, nil)))

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	// count global policies
	cGPRes := fleet.CountGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &cGPRes)
	assert.Equal(t, 1, cGPRes.Count)

	// count global policies with matching search query
	cGPRes = fleet.CountGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &cGPRes, "query", "estQue")
	assert.Equal(t, 1, cGPRes.Count)

	// count global policies with matching search query containing leading/trailing whitespace
	cGPRes = fleet.CountGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &cGPRes, "query", " estQue    ")
	assert.Equal(t, 1, cGPRes.Count)

	// count global policies with non-matching search query
	cGPRes = fleet.CountGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &cGPRes, "query", "Query4")
	assert.Equal(t, 0, cGPRes.Count)

	// delete the policy
	deletePolicyParams := fleet.DeleteGlobalPoliciesRequest{IDs: []uint{policiesResponse.Policies[0].ID}}
	deletePolicyResp := fleet.DeleteGlobalPoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)

	policiesResponse = fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Empty(t, policiesResponse.Policies)
}

func (s *integrationTestSuite) TestGlobalPoliciesProprietary() {
	t := s.T()

	for i := range 3 {
		_, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   new(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         new(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
			Platform:        "darwin",
		})
		require.NoError(t, err)
	}

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery321",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	// Cannot set both QueryID and Query.
	gpParams0 := fleet.GlobalPolicyRequest{
		QueryID: &qr.ID,
		Query:   "select * from osquery;",
	}
	gpResp0 := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams0, http.StatusBadRequest, &gpResp0)
	require.Nil(t, gpResp0.Policy)

	gpParams := fleet.GlobalPolicyRequest{
		Name:        "TestQuery3",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some global resolution",
		Platform:    "darwin",
	}
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NotEmpty(t, gpResp.Policy.ID)
	assert.Equal(t, "TestQuery3", gpResp.Policy.Name)
	assert.Equal(t, "select * from osquery;", gpResp.Policy.Query)
	assert.Equal(t, "Some description", gpResp.Policy.Description)
	require.NotNil(t, gpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution", *gpResp.Policy.Resolution)
	assert.NotNil(t, gpResp.Policy.AuthorID)
	assert.Equal(t, "Test Name admin1@example.com", gpResp.Policy.AuthorName)
	assert.Equal(t, "admin1@example.com", gpResp.Policy.AuthorEmail)
	assert.Equal(t, "darwin", gpResp.Policy.Platform)

	response := s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), []byte(`{
		"name": "TestQuery4",
		"query": "select * from osquery_info;",
		"description": "Some description updated",
		"resolution": "some global resolution updated"
	}`), http.StatusOK)
	var mgpResp fleet.ModifyGlobalPolicyResponse
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	err = json.Unmarshal(responseBody, &mgpResp)
	require.NoError(t, err)

	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, "TestQuery4", mgpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", mgpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mgpResp.Policy.Description)
	require.NotNil(t, mgpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *mgpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mgpResp.Policy.Platform)
	assert.Equal(t, uint(0), mgpResp.Policy.FailingHostCount)
	assert.Equal(t, uint(0), mgpResp.Policy.PassingHostCount)

	ggpResp := fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), fleet.GetPolicyByIDRequest{}, http.StatusOK, &ggpResp)
	require.NotNil(t, ggpResp.Policy)
	assert.Equal(t, "TestQuery4", ggpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", ggpResp.Policy.Query)
	assert.Equal(t, "Some description updated", ggpResp.Policy.Description)
	require.NotNil(t, ggpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *ggpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mgpResp.Policy.Platform)
	assert.Equal(t, uint(0), mgpResp.Policy.FailingHostCount)
	assert.Equal(t, uint(0), mgpResp.Policy.PassingHostCount)

	policiesResponse := fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "TestQuery4", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery_info;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some global resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].FailingHostCount)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].PassingHostCount)

	listHostsURL := fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d", policiesResponse.Policies[0].ID)
	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 3)
	h1 := listHostsResp.Hosts[0]
	h2 := listHostsResp.Hosts[1]

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Empty(t, listHostsResp.Hosts)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=failing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Empty(t, listHostsResp.Hosts)

	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: new(true)}, time.Now(), false, nil)))
	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false, nil)))

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	response = s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), []byte(`{
		"query": "select * from users;"
	}`), http.StatusOK)
	responseBody, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	err = json.Unmarshal(responseBody, &mgpResp)
	require.NoError(t, err)

	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, "TestQuery4", mgpResp.Policy.Name)
	assert.Equal(t, "select * from users;", mgpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mgpResp.Policy.Description)
	require.NotNil(t, mgpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *mgpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mgpResp.Policy.Platform)
	assert.Equal(t, uint(0), mgpResp.Policy.FailingHostCount)
	assert.Equal(t, uint(0), mgpResp.Policy.PassingHostCount)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Empty(t, listHostsResp.Hosts)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=failing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Empty(t, listHostsResp.Hosts)

	policiesResponse = fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "TestQuery4", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from users;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some global resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].FailingHostCount)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].PassingHostCount)

	// Record query executions
	require.NoError(
		t, errOnly(s.ds.RecordPolicyQueryExecutions(
			context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: new(true)}, time.Now(), false, nil,
		)),
	)
	require.NoError(
		t, errOnly(s.ds.RecordPolicyQueryExecutions(
			context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false, nil,
		)),
	)
	// Update policy stats
	require.NoError(t, s.ds.UpdateHostPolicyCounts(context.Background()))

	// Fetch policy to make sure stats are updated
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].FailingHostCount)
	assert.Equal(t, uint(1), policiesResponse.Policies[0].PassingHostCount)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	// Modify the platform for the policy, which should clear the policy stats
	response = s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), []byte(`{
		"platform": "linux"
	}`), http.StatusOK)
	responseBody, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	err = json.Unmarshal(responseBody, &mgpResp)
	require.NoError(t, err)

	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, "TestQuery4", mgpResp.Policy.Name)
	assert.Equal(t, "select * from users;", mgpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mgpResp.Policy.Description)
	require.NotNil(t, mgpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *mgpResp.Policy.Resolution)
	assert.Equal(t, "linux", mgpResp.Policy.Platform)
	assert.Equal(t, uint(0), mgpResp.Policy.FailingHostCount)
	assert.Equal(t, uint(0), mgpResp.Policy.PassingHostCount)

	// Fetch policy to make sure stats are updated
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].FailingHostCount)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].PassingHostCount)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Empty(t, listHostsResp.Hosts)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=failing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Empty(t, listHostsResp.Hosts)

	deletePolicyParams := fleet.DeleteGlobalPoliciesRequest{IDs: []uint{policiesResponse.Policies[0].ID}}
	deletePolicyResp := fleet.DeleteGlobalPoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)

	policiesResponse = fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Empty(t, policiesResponse.Policies)
}

func (s *integrationTestSuite) TestTeamPoliciesProprietary() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1-policies",
		Description: "desc team1",
	})
	require.NoError(t, err)
	hosts := make([]uint, 2)
	for i := range 2 {
		h, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   new(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         new(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
			Platform:        "darwin",
		})
		require.NoError(t, err)
		hosts[i] = h.ID
	}
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team1.ID, hosts))
	require.NoError(t, err)

	tpName := "TestPolicy3"
	tpParams := fleet.TeamPolicyRequest{
		Name:        tpName,
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
		Platform:    "darwin",
	}
	tpResp := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	require.NotEmpty(t, tpResp.Policy.ID)
	assert.Equal(t, tpName, tpResp.Policy.Name)
	assert.Equal(t, "select * from osquery;", tpResp.Policy.Query)
	assert.Equal(t, "Some description", tpResp.Policy.Description)
	require.NotNil(t, tpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution", *tpResp.Policy.Resolution)
	assert.NotNil(t, tpResp.Policy.AuthorID)
	assert.Equal(t, "Test Name admin1@example.com", tpResp.Policy.AuthorName)
	assert.Equal(t, "admin1@example.com", tpResp.Policy.AuthorEmail)

	tpNameNew := "TestPolicy4"

	response := s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, tpResp.Policy.ID), []byte(fmt.Sprintf(`{
		"name": "%s",
		"query": "select * from osquery_info;",
		"description": "Some description updated",
		"resolution": "some team resolution updated"
	}`, tpNameNew)), http.StatusOK)
	var mtpResp fleet.ModifyGlobalPolicyResponse
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	err = json.Unmarshal(responseBody, &mtpResp)
	require.NoError(t, err)

	require.NotNil(t, mtpResp.Policy)
	assert.Equal(t, tpNameNew, mtpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", mtpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mtpResp.Policy.Description)
	require.NotNil(t, mtpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution updated", *mtpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mtpResp.Policy.Platform)

	gtpResp := fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, tpResp.Policy.ID), fleet.GetPolicyByIDRequest{}, http.StatusOK, &gtpResp)
	require.NotNil(t, gtpResp.Policy)
	assert.Equal(t, tpNameNew, gtpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", gtpResp.Policy.Query)
	assert.Equal(t, "Some description updated", gtpResp.Policy.Description)
	require.NotNil(t, gtpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution updated", *gtpResp.Policy.Resolution)
	assert.Equal(t, "darwin", gtpResp.Policy.Platform)

	policiesResponse := fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, tpNameNew, policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery_info;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some team resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)
	require.Empty(t, policiesResponse.InheritedPolicies)

	// test team policy count endpoint
	tpCountResp := fleet.CountTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tpCountResp)
	assert.Equal(t, 1, tpCountResp.Count)
	assert.Equal(t, 0, tpCountResp.InheritedPolicyCount)

	tpCountResp = fleet.CountTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tpCountResp, "query", tpNameNew)
	assert.Equal(t, 1, tpCountResp.Count)
	assert.Equal(t, 0, tpCountResp.InheritedPolicyCount)

	tpCountResp = fleet.CountTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tpCountResp, "query", " "+tpNameNew+" ")
	assert.Equal(t, 1, tpCountResp.Count)
	assert.Equal(t, 0, tpCountResp.InheritedPolicyCount)

	tpCountResp = fleet.CountTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tpCountResp, "query", " nomatch")
	assert.Equal(t, 0, tpCountResp.Count)
	assert.Equal(t, 0, tpCountResp.InheritedPolicyCount)

	listHostsURL := fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d", policiesResponse.Policies[0].ID)
	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 2)
	h1 := listHostsResp.Hosts[0]
	h2 := listHostsResp.Hosts[1]

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?team_id=%d&policy_id=%d&policy_response=passing", team1.ID, policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Empty(t, listHostsResp.Hosts)

	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: new(true)}, time.Now(), false, nil)))
	require.NoError(t, errOnly(s.ds.RecordPolicyQueryExecutions(context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false, nil)))

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?team_id=%d&policy_id=%d&policy_response=passing", team1.ID, policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	deletePolicyParams := fleet.DeleteTeamPoliciesRequest{IDs: []uint{policiesResponse.Policies[0].ID}}
	deletePolicyResp := fleet.DeleteTeamPoliciesResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", team1.ID), deletePolicyParams, http.StatusOK, &deletePolicyResp)

	policiesResponse = fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &policiesResponse)
	require.Empty(t, policiesResponse.Policies)
}

func (s *integrationTestSuite) TestTeamPoliciesProprietaryInvalid() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1-policies-2",
		Description: "desc team1",
	})
	require.NoError(t, err)

	tpParams := fleet.TeamPolicyRequest{
		Name:        "TestQuery3-Team",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
	}
	tpResp := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	teamPolicyID := tpResp.Policy.ID

	gpParams := fleet.GlobalPolicyRequest{
		Name:        "TestQuery3-Global",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some global resolution",
	}
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NotEmpty(t, gpResp.Policy.ID)
	globalPolicyID := gpResp.Policy.ID

	for _, tc := range []struct {
		tname      string
		testUpdate bool
		queryID    *uint
		name       string
		query      string
		platforms  string
	}{
		{
			tname:      "set both QueryID and Query",
			testUpdate: false,
			queryID:    new(uint(1)),
			name:       "Some name",
			query:      "select * from osquery;",
		},
		{
			tname:      "empty query",
			testUpdate: true,
			name:       "Some name",
			query:      "",
		},
		{
			tname:      "empty name",
			testUpdate: true,
			name:       "",
			query:      "select 1;",
		},
		{
			tname:      "empty with space",
			testUpdate: true,
			name:       " ", // #3704
			query:      "select 1;",
		},
		{
			tname:      "Invalid query",
			testUpdate: true,
			name:       "Invalid query",
			query:      "",
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			tpReq := fleet.TeamPolicyRequest{
				QueryID:  tc.queryID,
				Name:     tc.name,
				Query:    tc.query,
				Platform: tc.platforms,
			}
			tpResp := fleet.TeamPolicyResponse{}
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpReq, http.StatusBadRequest, &tpResp)
			require.Nil(t, tpResp.Policy)

			testUpdate := tc.queryID == nil

			if testUpdate {
				tpReq := fleet.ModifyTeamPolicyRequest{
					ModifyPolicyPayload: fleet.ModifyPolicyPayload{
						Name:  new(tc.name),
						Query: new(tc.query),
					},
				}
				tpResp := fleet.ModifyTeamPolicyResponse{}
				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, teamPolicyID), tpReq, http.StatusBadRequest, &tpResp)
				require.Nil(t, tpResp.Policy)
			}

			gpReq := fleet.GlobalPolicyRequest{
				QueryID:  tc.queryID,
				Name:     tc.name,
				Query:    tc.query,
				Platform: tc.platforms,
			}
			gpResp := fleet.GlobalPolicyResponse{}
			s.DoJSON("POST", "/api/latest/fleet/policies", gpReq, http.StatusBadRequest, &gpResp)
			require.Nil(t, tpResp.Policy)

			if testUpdate {
				gpReq := fleet.ModifyGlobalPolicyRequest{
					ModifyPolicyPayload: fleet.ModifyPolicyPayload{
						Name:  new(tc.name),
						Query: new(tc.query),
					},
				}
				gpResp := fleet.ModifyGlobalPolicyResponse{}
				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", globalPolicyID), gpReq, http.StatusBadRequest, &gpResp)
				require.Nil(t, tpResp.Policy)
			}
		})
	}
}

func (s *integrationTestSuite) TestHostDetailsPolicies() {
	t := s.T()

	hosts := s.createHosts(t)
	host1 := hosts[0]
	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "HostDetailsPolicies-Team",
		Description: "desc team1",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&team1.ID, []uint{host1.ID}))
	require.NoError(t, err)

	gpParams := fleet.GlobalPolicyRequest{
		Name:        "HostDetailsPolicies",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some global resolution",
	}
	gpResp := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NotEmpty(t, gpResp.Policy.ID)

	tpParams := fleet.TeamPolicyRequest{
		Name:        "HostDetailsPolicies-Team",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
	}
	tpResp := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	require.NotEmpty(t, tpResp.Policy.ID)

	_, err = s.ds.RecordPolicyQueryExecutions(
		context.Background(),
		host1,
		map[uint]*bool{gpResp.Policy.ID: new(true)},
		time.Now(),
		false,
		nil,
	)
	require.NoError(t, err)

	resp := s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host1.ID), nil, http.StatusOK)
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var r struct {
		Host *fleet.HostDetailResponse `json:"host"`
		Err  error                     `json:"error,omitempty"`
	}
	err = json.Unmarshal(b, &r)
	require.NoError(t, err)
	require.NoError(t, r.Err)
	hd := r.Host.HostDetail
	policies := *hd.Policies
	require.Len(t, policies, 2)
	// Policies that did not run are listed before passing policies
	// TODO(JVE): verify that this passes once JK merges his code
	require.True(t, reflect.DeepEqual(tpResp.Policy.PolicyData, policies[0].PolicyData))
	require.Empty(t, policies[0].Response) // policy didn't "run"

	require.True(t, reflect.DeepEqual(gpResp.Policy.PolicyData, policies[1].PolicyData))
	require.Equal(t, "pass", policies[1].Response)

	// Try to create a global policy with an existing name.
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusConflict, &gpResp)
	// Try to create a team policy with an existing name.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusConflict, &tpResp)
}

// TestGlobalPoliciesBrowsing tests that team users can browse (read) global policies (see #3722).
func (s *integrationTestSuite) TestGlobalPoliciesBrowsing() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team_for_global_policies_browsing",
		Description: "desc team1",
	})
	require.NoError(t, err)

	gpParams0 := fleet.GlobalPolicyRequest{
		Name:  "global policy",
		Query: "select * from osquery;",
	}
	gpResp0 := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams0, http.StatusOK, &gpResp0)
	require.NotNil(t, gpResp0.Policy)

	email := "team.observer@example.com"
	teamObserver := &fleet.User{
		Name:       "team observer user",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team,
				Role: fleet.RoleObserver,
			},
		},
	}
	password := test.GoodPassword
	require.NoError(t, teamObserver.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), teamObserver)
	require.NoError(t, err)

	oldToken := s.token
	s.token = s.getTestToken(email, password)
	t.Cleanup(func() {
		s.token = oldToken
	})

	policiesResponse := fleet.ListGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "global policy", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery;", policiesResponse.Policies[0].Query)
}

// TestGetPolicyByIDCrossTeamAccess verifies that GET /api/latest/fleet/policies/{id}
// returns a team policy to a user that has access to the team it belongs to and
// rejects the request with 403 when the requesting user has no role on that team.
// This guards the security fix for the "Cross-Team Policy Data Exposure" disclosure.
func (s *integrationTestSuite) TestGetPolicyByIDCrossTeamAccess() {
	t := s.T()
	ctx := context.Background()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_team2"})
	require.NoError(t, err)

	policy, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:  t.Name() + "_team1_policy",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)

	password := test.GoodPassword

	team1Observer := &fleet.User{
		Name:  "team1 observer",
		Email: t.Name() + "_team1_observer@example.com",
		Teams: []fleet.UserTeam{{Team: *team1, Role: fleet.RoleObserver}},
	}
	require.NoError(t, team1Observer.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(ctx, team1Observer)
	require.NoError(t, err)

	team2Observer := &fleet.User{
		Name:  "team2 observer",
		Email: t.Name() + "_team2_observer@example.com",
		Teams: []fleet.UserTeam{{Team: *team2, Role: fleet.RoleObserver}},
	}
	require.NoError(t, team2Observer.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(ctx, team2Observer)
	require.NoError(t, err)

	policyURL := fmt.Sprintf("/api/latest/fleet/policies/%d", policy.ID) // nolint:nilaway // cannot be nil due to previous require

	// A user with access to the policy's team can read it.
	s.setTokenForTest(t, team1Observer.Email, password)
	var okResp fleet.GetPolicyByIDResponse
	s.DoJSON("GET", policyURL, fleet.GetPolicyByIDRequest{}, http.StatusOK, &okResp)
	require.NotNil(t, okResp.Policy)
	assert.Equal(t, policy.ID, okResp.Policy.ID)
	require.NotNil(t, okResp.Policy.TeamID)
	assert.Equal(t, team1.ID, *okResp.Policy.TeamID)

	// A user with no role on the policy's team is forbidden from reading it.
	s.setTokenForTest(t, team2Observer.Email, password)
	s.Do("GET", policyURL, fleet.GetPolicyByIDRequest{}, http.StatusForbidden)
}

func (s *integrationTestSuite) TestTeamPoliciesTeamNotExists() {
	t := s.T()

	teamPoliciesResponse := fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", 9999999), nil, http.StatusNotFound, &teamPoliciesResponse)
	require.Empty(t, teamPoliciesResponse.Policies)

	deleteTeamPoliciesResp := fleet.DeleteTeamPoliciesResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", 9999999), fleet.DeleteTeamPoliciesRequest{IDs: []uint{1, 1000}}, http.StatusNotFound, &deleteTeamPoliciesResp)
}

func (s *integrationTestSuite) TestReenrollHostCleansPolicies() {
	t := s.T()
	ctx := context.Background()
	host := s.createHosts(t)[0]

	// set the enroll secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Empty(t, getHostResp.Host.Policies)

	// create a policy and make the host fail it
	pol, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: t.Name(), Query: "SELECT 1", Platform: host.FleetPlatform()})
	require.NoError(t, err)
	_, err = s.ds.RecordPolicyQueryExecutions(ctx, &fleet.Host{ID: host.ID}, map[uint]*bool{pol.ID: new(false)}, time.Now(), false, nil)
	require.NoError(t, err)

	// refetch the host details
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Len(t, *getHostResp.Host.Policies, 1)

	// re-enroll the host, but using a different platform
	j, err := json.Marshal(&contract.EnrollOsqueryAgentRequest{
		EnrollSecret:   t.Name(),
		HostIdentifier: *host.OsqueryHostID,
		HostDetails:    map[string](map[string]string){"os_version": map[string]string{"platform": "windows"}},
	})
	require.NoError(t, err)

	// prevent the enroll cooldown from being applied
	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(
			context.Background(),
			"UPDATE hosts SET last_enrolled_at = DATE_SUB(NOW(), INTERVAL '1' HOUR) WHERE id = ?",
			host.ID,
		)
		return err
	})
	var resp contract.EnrollOsqueryAgentResponse
	hres := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusOK)
	defer hres.Body.Close()
	require.NoError(t, json.NewDecoder(hres.Body).Decode(&resp))
	require.NotEmpty(t, resp.NodeKey)

	// refetch the host details
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)

	// policies should be gone
	require.Empty(t, getHostResp.Host.Policies)
}

func (s *integrationTestSuite) TestAutofillPolicies() {
	t := s.T()

	// Ensure AI features are enabled (may have been disabled by a previous test).
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"server_settings": {"ai_features_disabled": false}
	}`), http.StatusOK)

	startMockServer := func(t *testing.T) string {
		// create a test http server
		srv := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "POST" {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}
					switch r.URL.Path {
					case "/ok":
						var body map[string]any
						err := json.NewDecoder(r.Body).Decode(&body)
						if err != nil {
							t.Log(err)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						_, _ = w.Write([]byte(`{"risks":"description", "whatWillProbablyHappenDuringMaintenance":"resolution"}`))
					case "/error":
						w.WriteHeader(http.StatusTeapot)
						_, _ = w.Write([]byte(`{}`))
					case "/badBody":
						_, _ = w.Write([]byte(`{bad json}`))
					case "/timeout":
						time.Sleep(2 * time.Second)
						_, _ = w.Write([]byte(`{"risks":"description", "whatWillProbablyHappenDuringMaintenance":"resolution"}`))
					default:
						w.WriteHeader(http.StatusNotFound)
					}
				},
			),
		)
		t.Cleanup(srv.Close)
		return srv.URL
	}
	mockUrl := startMockServer(t)
	originalUrl := getHumanInterpretationFromOsquerySqlUrl
	originalTimeout := getHumanInterpretationFromOsquerySqlTimeout
	t.Cleanup(
		func() {
			getHumanInterpretationFromOsquerySqlUrl = originalUrl
			getHumanInterpretationFromOsquerySqlTimeout = originalTimeout
		},
	)

	req := fleet.AutofillPoliciesRequest{
		SQL: "  ", // empty
	}
	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/ok"
	// empty sql
	resp := s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusBadRequest)
	assertBodyContains(t, resp, "cannot be empty")

	// good request
	req.SQL = "select 1"
	var res fleet.AutofillPoliciesResponse
	s.DoJSON("POST", "/api/latest/fleet/autofill/policy", req, http.StatusOK, &res)
	assert.Equal(t, "description", res.Description)
	assert.Equal(t, "resolution", res.Resolution)

	// good request with weird characters
	req.SQL = `select * from " with ' and "" \"`
	res = fleet.AutofillPoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/autofill/policy", req, http.StatusOK, &res)
	assert.Equal(t, "description", res.Description)
	assert.Equal(t, "resolution", res.Resolution)

	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/error"
	resp = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, resp, "error from human interpretation of osquery sql")

	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/badBody"
	resp = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, resp, "error unmarshaling response body from human interpretation of osquery sql")

	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/timeout"
	getHumanInterpretationFromOsquerySqlTimeout = 1 * time.Millisecond
	resp = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, resp, "error sending request to get human interpretation from osquery sql")

	// disable AI features
	appConfigSpec := map[string]map[string]bool{
		"server_settings": {"ai_features_disabled": true},
	}
	s.Do("PATCH", "/api/latest/fleet/config", appConfigSpec, http.StatusOK)
	resp = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusBadRequest)
	assertBodyContains(t, resp, "AI features are disabled")
}

func (s *integrationTestSuite) TestHostWithNoPoliciesClearsPolicyCounts() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "Zoobar"})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("foobar"),
		UUID:            "foobar",
		Hostname:        "com.foobar.local",
		Platform:        "linux",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:  "Barfoo",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)

	distributedWriteResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy.ID: new(false),
		},
	), http.StatusOK, &distributedWriteResp)

	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)
	require.Equal(t, uint64(1), listHostsResp.Hosts[0].FailingPoliciesCount)

	_, err = s.ds.DeleteTeamPolicies(ctx, team.ID, []uint{policy.ID})
	require.NoError(t, err)

	distributedWriteResp = submitDistributedQueryResultsResponse{}
	results := make(map[string]json.RawMessage)
	results[hostNoPoliciesWildcard] = json.RawMessage("{\"1\": \"1\"}")
	statuses := make(map[string]any)
	statuses[hostNoPoliciesWildcard] = 0
	s.DoJSON("POST", "/api/osquery/distributed/write", submitDistributedQueryResultsRequestShim{
		NodeKey:  *host.NodeKey,
		Results:  results,
		Statuses: statuses,
		Stats:    map[string]*fleet.Stats{},
	}, http.StatusOK, &distributedWriteResp)

	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)
	require.Equal(t, uint64(0), listHostsResp.Hosts[0].FailingPoliciesCount)
}

func (s *integrationTestSuite) TestTeamPolicyResendConfigProfileRequiresPremium() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	profileUUID := fleet.MDMAppleProfileUUIDPrefix + uuid.NewString()

	// Create: rejected at decode, before the profile is ever looked up.
	res := s.Do("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team.ID),
		&fleet.TeamPolicyRequest{
			Name:        "premium resend",
			Query:       "SELECT 1;",
			Platform:    "darwin",
			ProfileUUID: new(profileUUID),
		}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "requires a premium license")

	// Modify: same decode-time gate.
	pol, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:  "premium resend patch",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)

	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, pol.ID),
		json.RawMessage(fmt.Sprintf(`{"profile_uuid": %q}`, profileUUID)), http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "requires a premium license")

	// GitOps spec apply: explicit license check, so this one is a 402.
	res = s.Do("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{{
			Name:        "premium resend spec",
			Query:       "SELECT 1;",
			Platform:    "darwin",
			Team:        team.Name,
			ProfileUUID: new(profileUUID),
		}},
	}, http.StatusPaymentRequired)
	require.Contains(t, extractServerErrorText(res.Body), "Requires Fleet Premium license")

	// Without profile_uuid the same spec applies cleanly on a free license.
	s.Do("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{{
			Name:     "premium resend spec",
			Query:    "SELECT 1;",
			Platform: "darwin",
			Team:     team.Name,
		}},
	}, http.StatusOK)
}
