package service

// GitOps tests for the enterprise (premium) suite.
//
// Belongs here: GitOps mode and exceptions settings, actions taken by GitOps users, and GitOps
// user authorization for specific endpoints such as the org logo upload.
//
// Does not belong here: API-only user endpoint restrictions (integration_enterprise_auth_test.go)
// and the spec endpoints themselves (fleets, policies, scripts, software each have their own file).

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestGitOpsModeConfig() {
	t := s.T()

	res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"gitops": { "gitops_mode_enabled": true, "repository_url": "" }
	  }`), http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "Repository URL is required when GitOps mode is enabled")

	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"gitops": { "gitops_mode_enabled": true, "repository_url": "a.b.cc" }
	  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "Git repository URL must include protocol (e.g. https://)")

	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"gitops": { "gitops_mode_enabled": true, "repository_url": "https://a.b.cc" }
	  }`), http.StatusOK)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEnabledGitOpsMode{}.ActivityName(), "", 0)

	// turn off, persists repo url
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"gitops": { "gitops_mode_enabled": false, "repository_url": "https://a.b.cc" }
	  }`), http.StatusOK)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeDisabledGitOpsMode{}.ActivityName(), "", 0)

	// Make sure URL isn't cleared after another setting is changed
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"org_info": {"org_name": "Changes"}
	  }`), http.StatusOK)
	config, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "https://a.b.cc", config.GitOpsConfig.RepositoryURL)
}

func (s *integrationEnterpriseTestSuite) TestGitOpsExceptionsConfig() {
	t := s.T()

	// Start from a known state: GitOps mode on, all exceptions disabled.
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"gitops": {
			"gitops_mode_enabled": true,
			"repository_url": "https://example.com/repo",
			"exceptions": { "labels": false, "software": false, "secrets": false }
		}
	}`), http.StatusOK)

	// Enable two exceptions (labels and software); secrets stays false.
	// Two enabled_gitops_exception activities expected — one per flipped field.
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"gitops": { "exceptions": { "labels": true, "software": true, "secrets": false } }
	}`), http.StatusOK)
	enabledExceptions := s.listRecentExceptionActivities(fleet.ActivityTypeEnabledGitOpsException{}.ActivityName())
	assert.Contains(t, enabledExceptions, "labels", "expected activity for enabled labels exception")
	assert.Contains(t, enabledExceptions, "software", "expected activity for enabled software exception")
	assert.NotContains(t, enabledExceptions, "secrets", "no activity expected for unchanged secrets exception")

	config, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	assert.True(t, config.GitOpsConfig.Exceptions.Labels)
	assert.True(t, config.GitOpsConfig.Exceptions.Software)
	assert.False(t, config.GitOpsConfig.Exceptions.Secrets)
	assert.True(t, config.GitOpsConfig.GitopsModeEnabled)
	assert.Equal(t, "https://example.com/repo", config.GitOpsConfig.RepositoryURL)

	// Partial update — only change one exception, others should persist.
	// Capture the last activity id before the update so we can verify no
	// activity is recorded for the fields that did not change.
	lastIDBefore := s.lastActivityMatches("", "", 0)
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"gitops": { "exceptions": { "software": false } }
	}`), http.StatusOK)
	lastID := s.lastActivityMatches(
		fleet.ActivityTypeDisabledGitOpsException{}.ActivityName(),
		`{"exception": "software"}`, 0)
	assert.Greater(t, lastID, lastIDBefore, "a new activity should have been recorded")

	// Save app config with no exception changes — no new exception activity should be recorded.
	lastIDBefore = s.lastActivityMatches("", "", 0)
	s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"gitops": { "exceptions": { "labels": true, "software": false, "secrets": false } }
	}`), http.StatusOK)
	lastIDAfter := s.lastActivityMatches("", "", 0)
	assert.Equal(t, lastIDBefore, lastIDAfter, "no activity should have been recorded when exceptions are unchanged")

	config, err = s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	assert.True(t, config.GitOpsConfig.Exceptions.Labels, "labels should persist")
	assert.False(t, config.GitOpsConfig.Exceptions.Software, "software should be updated")
	assert.False(t, config.GitOpsConfig.Exceptions.Secrets, "secrets should persist")
	assert.True(t, config.GitOpsConfig.GitopsModeEnabled)
	assert.Equal(t, "https://example.com/repo", config.GitOpsConfig.RepositoryURL)

	// Verify exceptions appear in GET response
	var getResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &getResp)
	assert.True(t, getResp.GitOpsConfig.Exceptions.Labels)
	assert.False(t, getResp.GitOpsConfig.Exceptions.Software)
	assert.False(t, getResp.GitOpsConfig.Exceptions.Secrets)
	assert.True(t, getResp.GitOpsConfig.GitopsModeEnabled)
	assert.Equal(t, "https://example.com/repo", getResp.GitOpsConfig.RepositoryURL)
}

// listRecentExceptionActivities returns the set of exception names found among
// the recent activities of the given type (enabled_gitops_exception or
// disabled_gitops_exception). Used to assert on exception activities without
// depending on ordering when multiple are emitted by a single PATCH.
func (s *integrationEnterpriseTestSuite) listRecentExceptionActivities(activityType string) map[string]struct{} {
	t := s.T()
	var listActivities listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK,
		&listActivities, "order_key", "id", "order_direction", "desc", "per_page", "10")
	found := map[string]struct{}{}
	for _, act := range listActivities.Activities {
		if act.Type != activityType || act.Details == nil {
			continue
		}
		var d struct {
			Exception string `json:"exception"`
		}
		require.NoError(t, json.Unmarshal([]byte(*act.Details), &d))
		found[d.Exception] = struct{}{}
	}
	return found
}

// TestGitOpsUserActions tests the (non-MDM) permissions listed in ../../docs/Using\ Fleet/manage-access.md
func (s *integrationEnterpriseTestSuite) TestGitOpsUserActions() {
	t := s.T()
	ctx := context.Background()

	s.lq.SetQueryResultsCountOverride = func(queryID uint, count int) error {
		return nil
	}
	//
	// Setup test data.
	// All setup actions are authored by a global admin.
	//

	admin, err := s.ds.UserByEmail(ctx, "admin1@example.com")
	require.NoError(t, err)
	h1, err := s.ds.NewHost(ctx, &fleet.Host{
		NodeKey:  new(t.Name() + "1"),
		UUID:     t.Name() + "1",
		Hostname: strings.ReplaceAll(t.Name()+"foo.local", "/", "_"),
	})
	require.NoError(t, err)
	t1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Foo",
	})
	require.NoError(t, err)
	t2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Bar",
	})
	require.NoError(t, err)
	t3, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Zoo",
	})
	require.NoError(t, err)
	team1Host, err := s.ds.NewHost(ctx, &fleet.Host{
		NodeKey:  new(t.Name() + "2"),
		UUID:     t.Name() + "2",
		Hostname: strings.ReplaceAll(t.Name()+"zoo.local", "/", "_"),
		TeamID:   &t1.ID,
	})
	require.NoError(t, err)
	globalHost, err := s.ds.NewHost(ctx, &fleet.Host{
		NodeKey:  new(t.Name() + "3"),
		UUID:     t.Name() + "3",
		Hostname: strings.ReplaceAll(t.Name()+"global.local", "/", "_"),
	})
	require.NoError(t, err)
	acr := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false
			}
		}
	}`), http.StatusOK, &acr)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acr)
	require.False(t, acr.WebhookSettings.VulnerabilitiesWebhook.Enable)
	q1, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:    "Foo",
		Query:   "SELECT * from time;",
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	ggsr := getGlobalScheduleResponse{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &ggsr)
	require.NoError(t, ggsr.Err)
	cpar := createPackResponse{}
	var userPackID uint
	s.DoJSON("POST", "/api/latest/fleet/packs", createPackRequest{
		PackPayload: fleet.PackPayload{
			Name:     new("Foobar"),
			Disabled: new(false),
		},
	}, http.StatusOK, &cpar)
	userPackID = cpar.Pack.Pack.ID
	require.NotZero(t, userPackID)
	cur := createUserResponse{}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", createUserRequest{
		UserPayload: fleet.UserPayload{
			Email:      new("foo42@example.com"),
			Password:   new("p4ssw0rd.123"),
			Name:       new("foo42"),
			GlobalRole: new("maintainer"),
		},
	}, http.StatusOK, &cur)
	maintainer := cur.User
	var carveBeginResp fleet.CarveBeginResponse
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *h1.NodeKey,
		BlockCount: 3,
		BlockSize:  3,
		CarveSize:  8,
		CarveId:    "c1",
		RequestId:  "r1",
	}, http.StatusOK, &carveBeginResp)
	require.NotEmpty(t, carveBeginResp.SessionId)
	lcr := fleet.ListCarvesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/carves", fleet.ListCarvesRequest{}, http.StatusOK, &lcr)
	require.NotEmpty(t, lcr.Carves)
	carveID := lcr.Carves[0].ID
	// Create the global GitOps user we'll use in tests.
	u := &fleet.User{
		Name:       "GitOps",
		Email:      "gitops1@example.com",
		GlobalRole: new(fleet.RoleGitOps),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)
	// Create a GitOps user for team t1 we'll use in tests.
	u2 := &fleet.User{
		Name:       "GitOps 2",
		Email:      "gitops2@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *t1,
				Role: fleet.RoleGitOps,
			},
			{
				Team: *t3,
				Role: fleet.RoleGitOps,
			},
		},
	}
	require.NoError(t, u2.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u2)
	require.NoError(t, err)
	gp2, err := s.ds.NewGlobalPolicy(ctx, &admin.ID, fleet.PolicyPayload{
		Name:  "Zoo",
		Query: "SELECT 0;",
	})
	require.NoError(t, err)
	t2p, err := s.ds.NewTeamPolicy(ctx, t2.ID, &admin.ID, fleet.PolicyPayload{
		Name:  "Zoo2",
		Query: "SELECT 2;",
	})
	require.NoError(t, err)
	// Create some test user to test moving from/to teams.
	u3 := &fleet.User{
		Name:       "Test Foo Observer",
		Email:      "test-foo-observer@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *t1,
				Role: fleet.RoleObserver,
			},
		},
	}
	require.NoError(t, u3.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u3)
	require.NoError(t, err)
	manualLabel1, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                "manualLabel1",
		Query:               "SELECT 2;",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)

	//
	// Start running permission tests with user gitops1.
	//
	s.setTokenForTest(t, "gitops1@example.com", test.GoodPassword)

	// Attempt to retrieve activities, should fail.
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusForbidden, &listActivitiesResponse{})

	// Attempt to retrieve hosts, should fail.
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusForbidden, &listHostsResponse{})

	// Attempt to retrieve a host by identifier should succeed, but return the host id
	// and nothing else: GitOps is denied on every other host read endpoint, and only
	// needs the id here to resolve an identifier for the Puppet module's
	// profile pre-assignment. Asserting on the raw body since decoding into a
	// response struct wouldn't catch extra fields.
	resp := s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", h1.Hostname), hostByIdentifierRequest{}, http.StatusOK)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, fmt.Sprintf(`{"host":{"id":%d}}`, h1.ID), string(body))

	// Attempt to filter hosts using labels, should fail (label ID 6 is the builtin label "All Hosts")
	s.DoJSON("GET", "/api/latest/fleet/labels/6/hosts", nil, http.StatusOK, &listHostsResponse{})

	// Attempt to delete hosts, should fail.
	s.DoJSON("DELETE", "/api/latest/fleet/hosts/1", nil, http.StatusForbidden, &deleteHostResponse{})

	// Attempt to transfer host from global to a team, should allow.
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &t1.ID,
		HostIDs: []uint{h1.ID},
	}, http.StatusOK, &addHostsToTeamResponse{})

	// The host is now on a team, but the identifier endpoint still returns the id only.
	resp = s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", h1.Hostname), hostByIdentifierRequest{}, http.StatusOK)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.JSONEq(t, fmt.Sprintf(`{"host":{"id":%d}}`, h1.ID), string(body))

	// Attempt to create a label, should allow.
	clr := fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.CreateLabelRequest{
		LabelPayload: fleet.LabelPayload{
			Name:  "foo",
			Query: "SELECT 1;",
		},
	}, http.StatusOK, &clr)

	// Attempt to modify a label, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", clr.Label.ID), fleet.ModifyLabelRequest{
		ModifyLabelPayload: fleet.ModifyLabelPayload{
			Name: new("foo2"),
		},
	}, http.StatusOK, &fleet.ModifyLabelResponse{})

	// Attempt to get a label, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", clr.Label.ID), fleet.GetLabelRequest{}, http.StatusOK, &fleet.GetLabelResponse{})

	// Attempt to list all labels, should fail.
	s.DoJSON("GET", "/api/latest/fleet/labels", fleet.ListLabelsRequest{}, http.StatusOK, &fleet.ListLabelsResponse{})

	// Attempt to delete a label, should allow.
	s.DoJSON("DELETE", "/api/latest/fleet/labels/foo2", fleet.DeleteLabelRequest{}, http.StatusOK, &fleet.DeleteLabelResponse{})

	// Listing software is allowed for gitops so they can reconcile software state.
	s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareRequest{}, http.StatusOK, &listSoftwareVersionsResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusOK, &listSoftwareResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software/count", countSoftwareRequest{}, http.StatusOK, &countSoftwareResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &listSoftwareTitlesResponse{})
	// Getting a single software title or version still requires Host:list, which gitops doesn't have.
	s.DoJSON("GET", "/api/latest/fleet/software/titles/1", getSoftwareTitleRequest{}, http.StatusForbidden, &getSoftwareTitleResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software/1", getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software/versions/1", getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResponse{})

	// Attempt to read app config, should pass.
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &appConfigResponse{})

	// Attempt to write app config, should allow.
	acr = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "https://foobar.example.com"
			}
		}
	}`), http.StatusOK, &acr)
	require.True(t, acr.AppConfig.WebhookSettings.VulnerabilitiesWebhook.Enable)
	require.Equal(t, "https://foobar.example.com", acr.AppConfig.WebhookSettings.VulnerabilitiesWebhook.DestinationURL)

	// Attempt to add/remove manual labels to/from a host.
	var addLabelsToHostResp addLabelsToHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", h1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	var removeLabelsFromHostResp fleet.RemoveLabelsFromHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", h1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	// Attempt to run live queries synchronously, should fail.
	s.DoJSON("GET", "/api/latest/fleet/queries/run", runLiveQueryRequest{
		HostIDs:  []uint{h1.ID},
		QueryIDs: []uint{q1.ID},
	}, http.StatusForbidden, &runLiveQueryResponse{},
	)

	// Attempt to run live queries asynchronously (new unsaved query), should fail.
	s.DoJSON("POST", "/api/latest/fleet/queries/run", createDistributedQueryCampaignRequest{
		QuerySQL: "SELECT * FROM time;",
		Selected: fleet.HostTargets{
			HostIDs: []uint{h1.ID},
		},
	}, http.StatusForbidden, &runLiveQueryResponse{})

	// Attempt to run live queries asynchronously (saved query), should fail.
	s.DoJSON("POST", "/api/latest/fleet/queries/run", createDistributedQueryCampaignRequest{
		QueryID: new(q1.ID),
		Selected: fleet.HostTargets{
			HostIDs: []uint{h1.ID},
		},
	}, http.StatusForbidden, &runLiveQueryResponse{})

	// Attempt to create queries, should allow.
	cqr := fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.CreateQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  new("foo4"),
			Query: new("SELECT * from osquery_info;"),
		},
	}, http.StatusOK, &cqr)
	cqr2 := fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.CreateQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  new("foo5"),
			Query: new("SELECT * from os_version;"),
		},
	}, http.StatusOK, &cqr2)
	cqr3 := fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.CreateQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  new("foo6"),
			Query: new("SELECT * from processes;"),
		},
	}, http.StatusOK, &cqr3)
	cqr4 := fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.CreateQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  new("foo7"),
			Query: new("SELECT * from managed_policies;"),
		},
	}, http.StatusOK, &cqr4)

	// Attempt to edit queries, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", cqr.Query.ID), fleet.ModifyQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  new("foo4"),
			Query: new("SELECT * FROM system_info;"),
		},
	}, http.StatusOK, &fleet.ModifyQueryResponse{})

	// Attempt to view a query, should work.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d", cqr.Query.ID), fleet.GetQueryRequest{}, http.StatusOK, &fleet.GetQueryResponse{})

	// Attempt to list all queries, should work.
	s.DoJSON("GET", "/api/latest/fleet/queries", fleet.ListQueriesRequest{}, http.StatusOK, &fleet.ListQueriesResponse{})

	// Attempt to delete queries, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", cqr.Query.ID), fleet.DeleteQueryByIDRequest{}, http.StatusOK, &fleet.DeleteQueryByIDResponse{})
	s.DoJSON("POST", "/api/latest/fleet/queries/delete", fleet.DeleteQueriesRequest{IDs: []uint{cqr2.Query.ID}}, http.StatusOK, &fleet.DeleteQueriesResponse{})
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/%s", cqr3.Query.Name), fleet.DeleteQueryRequest{}, http.StatusOK, &fleet.DeleteQueryResponse{})

	// Attempt to add a query to a user pack, should allow.
	sqr := fleet.ScheduleQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", fleet.ScheduleQueryRequest{
		PackID:   userPackID,
		QueryID:  cqr4.Query.ID,
		Interval: 60,
	}, http.StatusOK, &sqr)

	// Attempt to edit a scheduled query in the global schedule, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sqr.Scheduled.ID), fleet.ModifyScheduledQueryRequest{
		ScheduledQueryPayload: fleet.ScheduledQueryPayload{
			Interval: new(uint(30)),
		},
	}, http.StatusOK, &fleet.ScheduleQueryResponse{})

	// Attempt to remove a query from the global schedule, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sqr.Scheduled.ID), fleet.DeleteScheduledQueryRequest{}, http.StatusOK, &fleet.DeleteScheduledQueryResponse{})

	// Attempt to read the global schedule, should allow.
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &getGlobalScheduleResponse{})

	// Attempt to create a pack, should allow.
	cpr := createPackResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs", createPackRequest{
		PackPayload: fleet.PackPayload{
			Name: new("foo8"),
		},
	}, http.StatusOK, &cpr)

	// Attempt to edit a pack, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/%d", cpr.Pack.ID), modifyPackRequest{
		PackPayload: fleet.PackPayload{
			Name: new("foo9"),
		},
	}, http.StatusOK, &modifyPackResponse{})

	// Attempt to read a pack, should allow.
	// This is an exception to the "write only" nature of gitops (packs can be viewed by gitops).
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d", cpr.Pack.ID), nil, http.StatusOK, &getPackResponse{})

	// Attempt to delete a pack, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/id/%d", cpr.Pack.ID), deletePackRequest{}, http.StatusOK, &deletePackResponse{})

	// Attempt to create a global policy, should allow.
	gplr := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
		Name:  "foo9",
		Query: "SELECT * from plist;",
	}, http.StatusOK, &gplr)

	// Attempt to edit a global policy, should allow.
	mgplr := fleet.ModifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gplr.Policy.ID), fleet.ModifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Query: new("SELECT * from plist WHERE path = 'foo';"),
		},
	}, http.StatusOK, &mgplr)

	// Attempt to read a global policy, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gplr.Policy.ID), fleet.GetPolicyByIDRequest{}, http.StatusOK,
		&fleet.GetPolicyByIDResponse{},
	)

	// Attempt to delete a global policy, should allow.
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", fleet.DeleteGlobalPoliciesRequest{
		IDs: []uint{gplr.Policy.ID},
	}, http.StatusOK, &fleet.DeleteGlobalPoliciesResponse{})

	// Attempt to create a team policy, should allow.
	tplr := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/team/%d/policies", t1.ID), fleet.TeamPolicyRequest{
		Name:  "foo10",
		Query: "SELECT * from file;",
	}, http.StatusOK, &tplr)

	// Attempt to edit a team policy, should allow.
	mtplr := fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", t1.ID, tplr.Policy.ID), fleet.ModifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Query: new("SELECT * from file WHERE path = 'foo';"),
		},
	}, http.StatusOK, &mtplr)

	// Attempt to view a team policy, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/team/%d/policies/%d", t1.ID, tplr.Policy.ID), fleet.GetTeamPolicyByIDRequest{}, http.StatusOK,
		&fleet.GetTeamPolicyByIDResponse{},
	)

	// Attempt to delete a team policy, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", t1.ID), fleet.DeleteTeamPoliciesRequest{
		IDs: []uint{tplr.Policy.ID},
	}, http.StatusOK, &fleet.DeleteTeamPoliciesResponse{})

	// Attempt to create a user, should fail.
	s.DoJSON("POST", "/api/latest/fleet/users/admin", createUserRequest{
		UserPayload: fleet.UserPayload{
			Email:      new("foo10@example.com"),
			Name:       new("foo10"),
			GlobalRole: new("admin"),
		},
	}, http.StatusForbidden, &createUserResponse{})

	// Attempt to modify a user, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", admin.ID), modifyUserRequest{
		UserPayload: fleet.UserPayload{
			GlobalRole: new("observer"),
		},
	}, http.StatusForbidden, &modifyUserResponse{})

	// Attempt to view a user, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", admin.ID), getUserRequest{}, http.StatusForbidden, &getUserResponse{})

	// Attempt to delete a user, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/users/%d", admin.ID), deleteUserRequest{}, http.StatusForbidden, &deleteUserResponse{})

	// Attempt to add users to team, should fail (gitops cannot manage team membership).
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t1.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *maintainer,
				Role: "admin",
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to delete users from team, should fail (gitops cannot manage team membership).
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t1.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *maintainer,
				Role: "admin",
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to create a team, should allow.
	tr := teamResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: new("foo11"),
		},
	}, http.StatusOK, &tr)

	// Attempt to edit a team, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tr.Team.ID), modifyTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: new("foo12"),
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to edit a team's agent options, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tr.Team.ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusOK, &teamResponse{})

	// Attempt to view a team, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", tr.Team.ID), getTeamRequest{}, http.StatusForbidden, &teamResponse{})

	// Attempt to delete a team, should allow.
	dtr := deleteTeamResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", tr.Team.ID), deleteTeamRequest{}, http.StatusOK, &dtr)

	// Attempt to create/edit enroll secrets, should allow.
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{
				{
					Secret: "foo400",
					TeamID: nil,
				},
				{
					Secret: "foo500",
					TeamID: new(t1.ID),
				},
			},
		},
	}, http.StatusOK, &applyEnrollSecretSpecResponse{})

	// Attempt to get enroll secrets, should fail.
	s.DoJSON("GET", "/api/latest/fleet/spec/enroll_secret", nil, http.StatusForbidden, &getEnrollSecretSpecResponse{})

	// Attempt to get team enroll secret, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", t1.ID), teamEnrollSecretsRequest{}, http.StatusForbidden, &teamEnrollSecretsResponse{})

	// Attempt to list carved files, should fail.
	s.DoJSON("GET", "/api/latest/fleet/carves", fleet.ListCarvesRequest{}, http.StatusForbidden, &fleet.ListCarvesResponse{})

	// Attempt to get a carved file, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d", carveID), fleet.GetCarveRequest{}, http.StatusForbidden, &fleet.GetCarveResponse{})

	// Attempt to search hosts, should fail.
	s.DoJSON("POST", "/api/latest/fleet/targets", fleet.SearchTargetsRequest{
		MatchQuery: "foo",
		QueryID:    &q1.ID,
	}, http.StatusForbidden, &fleet.SearchTargetsResponse{})

	// Attempt to count target hosts, should fail.
	s.DoJSON("POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{
		Selected: fleet.HostTargets{
			HostIDs:  []uint{h1.ID},
			LabelIDs: []uint{clr.Label.ID},
			TeamIDs:  []uint{t1.ID},
		},
		QueryID: &q1.ID,
	}, http.StatusForbidden, &fleet.CountTargetsResponse{})

	//
	// Start running permission tests with user gitops2 (which is a GitOps use for team t1).
	//

	s.setTokenForTest(t, "gitops2@example.com", test.GoodPassword)

	// Attempt to create queries in global domain, should fail.
	tcqr := fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.CreateQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  new("foo600"),
			Query: new("SELECT * from orbit_info;"),
		},
	}, http.StatusForbidden, &tcqr)

	// Attempt to create queries in its team, should allow.
	tcqr = fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.CreateQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:   new("foo600"),
			Query:  new("SELECT * from orbit_info;"),
			TeamID: &t1.ID,
		},
	}, http.StatusOK, &tcqr)

	// Attempt to edit own query, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", tcqr.Query.ID), fleet.ModifyQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  new("foo4"),
			Query: new("SELECT * FROM system_info;"),
		},
	}, http.StatusOK, &fleet.ModifyQueryResponse{})

	// Attempt to delete own query, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", tcqr.Query.ID), fleet.DeleteQueryByIDRequest{}, http.StatusOK, &fleet.DeleteQueryByIDResponse{})

	// Attempt to edit query created by somebody else, should fail. A team
	// GitOps user can't read global queries either (unlike every other team
	// role), so this reports not-found rather than forbidden: a forbidden
	// would confirm the global query exists to a caller with no view of it.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", cqr4.Query.ID), fleet.ModifyQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  new("foo4"),
			Query: new("SELECT * FROM system_info;"),
		},
	}, http.StatusNotFound, &fleet.ModifyQueryResponse{})

	// Attempt to delete query created by somebody else, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", cqr4.Query.ID), fleet.DeleteQueryByIDRequest{}, http.StatusNotFound, &fleet.DeleteQueryByIDResponse{})

	// Attempt to read the global schedule, should fail.
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusForbidden, &getGlobalScheduleResponse{})

	// Attempt to read the team's schedule, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", t1.ID), getTeamScheduleRequest{}, http.StatusOK,
		&getTeamScheduleResponse{},
	)

	// Attempt to read other team's schedule, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", t2.ID), getTeamScheduleRequest{}, http.StatusForbidden, &getTeamScheduleResponse{})

	// Attempt to add a query to a user pack, should fail.
	tsqr := fleet.ScheduleQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", fleet.ScheduleQueryRequest{
		PackID:   userPackID,
		QueryID:  cqr4.Query.ID,
		Interval: 60,
	}, http.StatusForbidden, &tsqr)

	// Attempt to add a query to the team's schedule, should allow.
	cqrt1 := fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.CreateQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:   new("foo8"),
			Query:  new("SELECT * from managed_policies;"),
			TeamID: &t1.ID,
		},
	}, http.StatusOK, &cqrt1)
	ttsqr := teamScheduleQueryResponse{}
	// Add a schedule with the deprecated APIs (by referencing a global query).
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", t1.ID), teamScheduleQueryRequest{
		ScheduledQueryPayload: fleet.ScheduledQueryPayload{
			QueryID:  new(q1.ID),
			Interval: new(uint(60)),
		},
	}, http.StatusOK, &ttsqr)

	// Attempt to remove a query from the team's schedule, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule/%d", t1.ID, ttsqr.Scheduled.ID), deleteTeamScheduleRequest{}, http.StatusOK, &deleteTeamScheduleResponse{})

	// Attempt to add/remove a manual label from a team host, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", team1Host.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", team1Host.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	// Attempt to add/remove a manual label from a global host, should not allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", globalHost.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", globalHost.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &removeLabelsFromHostResp)

	// Attempt to read the global schedule, should fail.
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusForbidden, &getGlobalScheduleResponse{})

	// Attempt to read a global policy, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gp2.ID), fleet.GetPolicyByIDRequest{}, http.StatusForbidden, &fleet.GetPolicyByIDResponse{})

	// Attempt to delete a global policy, should fail.
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", fleet.DeleteGlobalPoliciesRequest{
		IDs: []uint{gp2.ID},
	}, http.StatusForbidden, &fleet.DeleteGlobalPoliciesResponse{})

	// Attempt to create a team policy, should allow.
	ttplr := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/team/%d/policies", t1.ID), fleet.TeamPolicyRequest{
		Name:  "foo1000",
		Query: "SELECT * from file;",
	}, http.StatusOK, &ttplr)

	// Attempt to edit a team policy, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", t1.ID, ttplr.Policy.ID), fleet.ModifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Query: new("SELECT * from file WHERE path = 'foobar';"),
		},
	}, http.StatusOK, &fleet.ModifyTeamPolicyResponse{})

	// Attempt to edit another team's policy, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", t2.ID, t2p.ID), fleet.ModifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Query: new("SELECT * from file WHERE path = 'foobar';"),
		},
	}, http.StatusForbidden, &fleet.ModifyTeamPolicyResponse{})

	// Attempt to view a team policy, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/team/%d/policies/%d", t1.ID, ttplr.Policy.ID), fleet.GetTeamPolicyByIDRequest{}, http.StatusOK,
		&fleet.GetTeamPolicyByIDResponse{},
	)

	// Attempt to view another team's policy, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/team/%d/policies/%d", t2.ID, t2p.ID), fleet.GetTeamPolicyByIDRequest{}, http.StatusForbidden, &fleet.GetTeamPolicyByIDResponse{})

	// Attempt to delete a team policy, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", t1.ID), fleet.DeleteTeamPoliciesRequest{
		IDs: []uint{ttplr.Policy.ID},
	}, http.StatusOK, &fleet.DeleteTeamPoliciesResponse{})

	// Attempt to edit own team, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", t1.ID), modifyTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: new("foo123456"),
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to edit another team, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", t2.ID), modifyTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: new("foo123456"),
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to edit own team's agent options, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", t1.ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusOK, &teamResponse{})

	// Attempt to edit another team's agent options, should fail.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", t2.ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusForbidden, &teamResponse{})

	// Attempt to add users from team it owns to another team it owns, should fail (gitops cannot manage team membership).
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t3.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u3,
				Role: "maintainer",
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to delete users from team it owns, should fail (gitops cannot manage team membership).
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t3.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u3,
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to add users to another team it doesn't own, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t2.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u3,
				Role: "maintainer",
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to delete users from team it doesn't own, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t2.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u2,
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to search hosts, should fail.
	s.DoJSON("POST", "/api/latest/fleet/targets", fleet.SearchTargetsRequest{
		MatchQuery: "foo",
		QueryID:    &q1.ID,
	}, http.StatusForbidden, &fleet.SearchTargetsResponse{})

	// Attempt to count target hosts, should fail.
	s.DoJSON("POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{
		Selected: fleet.HostTargets{
			HostIDs:  []uint{h1.ID},
			LabelIDs: []uint{clr.Label.ID},
			TeamIDs:  []uint{t1.ID},
		},
		QueryID: &q1.ID,
	}, http.StatusForbidden, &fleet.CountTargetsResponse{})

	// Listing software titles for the team it owns is allowed.
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &listSoftwareTitlesResponse{}, "team_id", fmt.Sprint(t1.ID))
	// Listing software titles globally (no team) is forbidden for team gitops.
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusForbidden, &listSoftwareTitlesResponse{})
	// Listing software titles for a team it does not own is forbidden.
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusForbidden, &listSoftwareTitlesResponse{}, "team_id", fmt.Sprint(t2.ID))
}

func (s *integrationEnterpriseTestSuite) TestOrgLogoUploadGitOpsAuth() {
	t := s.T()

	pngImg := image.NewRGBA(image.Rect(0, 0, 1, 1))
	pngImg.Set(0, 0, color.RGBA{R: 0, G: 128, B: 0, A: 255})
	var pngBuf bytes.Buffer
	require.NoError(t, png.Encode(&pngBuf, pngImg))
	pngBytes := pngBuf.Bytes()

	gitopsEmail := "gitops-logo-enterprise@example.com"
	gitopsUser := &fleet.User{
		Name:       "GitOps Logo",
		Email:      gitopsEmail,
		GlobalRole: new(fleet.RoleGitOps),
	}
	require.NoError(t, gitopsUser.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(t.Context(), gitopsUser)
	require.NoError(t, err)

	s.token = s.getCachedUserToken(gitopsEmail, test.GoodPassword)
	defer func() { s.token = s.getTestAdminToken() }()

	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, err := w.CreateFormFile("logo", "dark.png")
	require.NoError(t, err)
	_, err = io.Copy(fw, bytes.NewReader(pngBytes))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	s.DoRawWithHeaders("PUT", "/api/v1/fleet/logo?mode=dark", body.Bytes(), http.StatusOK, map[string]string{
		"Content-Type":  w.FormDataContentType(),
		"Accept":        "application/json",
		"Authorization": "Bearer " + s.token,
	})

	var acResp appConfigResponse
	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, acResp.OrgInfo.OrgLogoURLDarkMode, "/api/latest/fleet/logo")

	s.token = s.getTestAdminToken()
	s.Do("DELETE", "/api/v1/fleet/logo", nil, http.StatusOK, "mode", "dark")
}
