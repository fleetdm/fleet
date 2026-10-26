package service

// Policy automation tests for the enterprise (premium) suite.
//
// Belongs here: what happens when a policy fails — failing-policy and host activity webhooks,
// resetting automations, policy-triggered scripts and software installs (including label scoping,
// package selection, continuous mode, retries and failure limits), and configuration profile
// resends.
//
// Does not belong here: policy CRUD, specs, and scoping (integration_enterprise_policies_test.go),
// calendar events (integration_enterprise_calendar_test.go), and installs or scripts that a user
// or setup experience requests directly (integration_enterprise_software_installers_test.go,
// integration_enterprise_scripts_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/policies"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/fleetdm/fleet/v4/server/webhooks"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A failing-policy webhook automation batch is sent, records
// a ran_automation_webhook activity carrying host_ids, and that activity fires
// the per-fleet host activities webhook with the same details.
func (s *integrationEnterpriseTestSuite) TestFailingPolicyAutomationFiresHostActivitiesWebhook() {
	t := s.T()
	ctx := t.Context()

	// One receiver, two endpoints: the failing-policies destination and the
	// per-fleet host activities destination.
	type hostActivitiesWebhookPayload struct {
		Type    string          `json:"type"`
		Details json.RawMessage `json:"details"`
	}
	fpwCalled := make(chan struct{}, 1)
	hostActivitiesCalled := make(chan hostActivitiesWebhookPayload, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fpw":
			_, _ = io.Copy(io.Discard, r.Body)
			fpwCalled <- struct{}{}
		case "/host-activities":
			var p hostActivitiesWebhookPayload
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				t.Log(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			hostActivitiesCalled <- p
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new(t.Name() + "-key"),
		UUID:            t.Name() + "-uuid",
		Hostname:        "host-activities-webhook-host",
		Platform:        "ubuntu",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	pol, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:  "host-activities-webhook-failing-policy",
		Query: "SELECT 1 WHERE 0",
	})
	require.NoError(t, err)

	// Enable the per-fleet host activities webhook on the team.
	var tmResp teamResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		HostActivitiesWebhook: &fleet.HostActivitiesWebhookSettings{
			Enable:         true,
			DestinationURL: srv.URL + "/host-activities",
		},
	}}, http.StatusOK, &tmResp)

	// Send the failing-policies automation batch through the production
	// sender, with the real activity service as the activity sink.
	failingPolicySet := NewMemFailingPolicySet()
	require.NoError(t, failingPolicySet.AddHost(pol.ID, fleet.PolicySetHost{
		ID:       host.ID,
		Hostname: host.Hostname,
	}))
	policy, err := s.ds.Policy(ctx, pol.ID)
	require.NoError(t, err)
	serverURL, err := url.Parse("https://fleet.example.com")
	require.NoError(t, err)
	webhookURL, err := url.Parse(srv.URL + "/fpw")
	require.NoError(t, err)

	activitySvc := mysqltest.NewTestActivityService(t, s.ds)
	require.NoError(t, webhooks.SendFailingPoliciesBatchedPOSTs(
		ctx, policy, failingPolicySet, 0, serverURL, webhookURL, time.Now(),
		slog.New(slog.DiscardHandler), activitySvc,
	))

	// The failing-policies webhook received the batch POST.
	select {
	case <-fpwCalled:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for failing policies webhook")
	}

	// The per-fleet host activities webhook received the resulting
	// ran_automation_webhook activity with host_ids injected into the payload
	// details at fire time.
	select {
	case p := <-hostActivitiesCalled:
		require.Equal(t, "ran_automation_webhook", p.Type)
		var details struct {
			PolicyID uint   `json:"policy_id"`
			HostIDs  []uint `json:"host_ids"`
		}
		require.NoError(t, json.Unmarshal(p.Details, &details))
		assert.Equal(t, pol.ID, details.PolicyID)
		assert.Equal(t, []uint{host.ID}, details.HostIDs)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for host activities webhook")
	}

	// The stored activity does NOT carry host_ids: the list is webhook-only,
	// so API/feed responses stay lean and don't expose the batch's host IDs.
	s.lastActivityMatches(
		fleet.ActivityTypeRanAutomationWebhook{}.ActivityName(),
		fmt.Sprintf(`{"policy_id": %d}`, pol.ID),
		0)
}

// A failing GLOBAL policy batch spanning hosts of two fleets fires each
// fleet's host activities webhook once, and each delivery carries only the
// host IDs belonging to that fleet.
func (s *integrationEnterpriseTestSuite) TestGlobalPolicyAutomationFiresEachFleetsHostActivitiesWebhook() {
	t := s.T()
	ctx := t.Context()

	type hostActivitiesWebhookPayload struct {
		Type    string          `json:"type"`
		Details json.RawMessage `json:"details"`
	}
	received := make(chan struct {
		path string
		body hostActivitiesWebhookPayload
	}, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fpw" {
			_, _ = io.Copy(io.Discard, r.Body)
			return
		}
		var p hostActivitiesWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			t.Log(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		received <- struct {
			path string
			body hostActivitiesWebhookPayload
		}{path: r.URL.Path, body: p}
	}))
	t.Cleanup(srv.Close)

	newHost := func(name string, teamID uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         new(t.Name() + name + "-key"),
			UUID:            t.Name() + name + "-uuid",
			Hostname:        name,
			Platform:        "ubuntu",
			TeamID:          &teamID,
		})
		require.NoError(t, err)
		return h
	}

	teamA, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-a"})
	require.NoError(t, err)
	teamB, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-b"})
	require.NoError(t, err)
	hostA := newHost("global-batch-host-a", teamA.ID)
	hostB := newHost("global-batch-host-b", teamB.ID)

	var tmResp teamResponse
	for teamID, path := range map[uint]string{teamA.ID: "/fleet-a", teamB.ID: "/fleet-b"} {
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", teamID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
			HostActivitiesWebhook: &fleet.HostActivitiesWebhookSettings{
				Enable:         true,
				DestinationURL: srv.URL + path,
			},
		}}, http.StatusOK, &tmResp)
	}

	// Global policy (no team) failing on both hosts.
	gpol, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name:  "global-host-activities-webhook-failing-policy",
		Query: "SELECT 1 WHERE 0",
	})
	require.NoError(t, err)

	failingPolicySet := NewMemFailingPolicySet()
	for _, h := range []*fleet.Host{hostA, hostB} {
		require.NoError(t, failingPolicySet.AddHost(gpol.ID, fleet.PolicySetHost{
			ID:       h.ID,
			Hostname: h.Hostname,
		}))
	}
	policy, err := s.ds.Policy(ctx, gpol.ID)
	require.NoError(t, err)
	serverURL, err := url.Parse("https://fleet.example.com")
	require.NoError(t, err)
	webhookURL, err := url.Parse(srv.URL + "/fpw")
	require.NoError(t, err)

	activitySvc := mysqltest.NewTestActivityService(t, s.ds)
	require.NoError(t, webhooks.SendFailingPoliciesBatchedPOSTs(
		ctx, policy, failingPolicySet, 0, serverURL, webhookURL, time.Now(),
		slog.New(slog.DiscardHandler), activitySvc,
	))

	// One delivery per fleet destination, each scoped to that fleet's hosts.
	deliveries := make(map[string]hostActivitiesWebhookPayload, 2)
	for range 2 {
		select {
		case d := <-received:
			deliveries[d.path] = d.body
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for host activities webhooks")
		}
	}
	require.Len(t, deliveries, 2)
	wantHostIDs := map[string][]uint{
		"/fleet-a": {hostA.ID},
		"/fleet-b": {hostB.ID},
	}
	for _, path := range []string{"/fleet-a", "/fleet-b"} {
		body, ok := deliveries[path]
		require.True(t, ok, "expected a webhook POST to %s", path)
		require.Equal(t, "ran_automation_webhook", body.Type)
		var details struct {
			PolicyID uint   `json:"policy_id"`
			HostIDs  []uint `json:"host_ids"`
		}
		require.NoError(t, json.Unmarshal(body.Details, &details))
		assert.Equal(t, gpol.ID, details.PolicyID)
		// Each fleet's endpoint only sees its own hosts.
		assert.Equal(t, wantHostIDs[path], details.HostIDs)
	}
	// No third delivery: one request per destination, not per host.
	select {
	case d := <-received:
		t.Fatalf("unexpected extra webhook delivery to %s", d.path)
	case <-time.After(500 * time.Millisecond):
	}
}

func (s *integrationEnterpriseTestSuite) TestNoTeamFailingPolicyWebhookTrigger() {
	t := s.T()
	ctx := t.Context()

	// Track webhook calls
	webhookCalled := false
	var capturedPolicy *fleet.Policy

	// Create a host without a team (team_id = nil means no team)
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         new("no-team-host-key"),
		UUID:            "no-team-host-uuid",
		Hostname:        "no-team-host",
		PrimaryIP:       "192.168.1.100",
		PrimaryMac:      "00:11:22:33:44:55",
		Platform:        "ubuntu",
		OSVersion:       "Ubuntu 20.04",
		Build:           "",
		PlatformLike:    "debian",
		OsqueryVersion:  "5.5.0",
		TeamID:          nil, // nil means "No Team"
	})
	require.NoError(t, err)
	require.Nil(t, host.TeamID)

	// Create "No Team" policies (team_id = 0)
	noTeamPol1, err := s.ds.NewTeamPolicy(ctx, 0, nil, fleet.PolicyPayload{
		Name:        "no-team-failing-policy-1",
		Query:       "SELECT 1 WHERE 0", // Will always fail
		Description: "Policy 1 for hosts without a team",
		Resolution:  "Fix policy 1",
	})
	require.NoError(t, err)
	require.NotNil(t, noTeamPol1.TeamID)
	require.Equal(t, uint(0), *noTeamPol1.TeamID)

	noTeamPol2, err := s.ds.NewTeamPolicy(ctx, 0, nil, fleet.PolicyPayload{
		Name:        "no-team-failing-policy-2",
		Query:       "SELECT 1 WHERE 0", // Will also fail
		Description: "Policy 2 for hosts without a team",
		Resolution:  "Fix policy 2",
	})
	require.NoError(t, err)

	noTeamPol3, err := s.ds.NewTeamPolicy(ctx, 0, nil, fleet.PolicyPayload{
		Name:  "no-team-failing-policy-3",
		Query: "SELECT 1 WHERE 0", // Will also fail (not in webhook config)
	})
	require.NoError(t, err)

	// Configure webhook for "No Team" - only include pol1 and pol2
	var defaultTeamResp struct {
		Team *fleet.DefaultTeam `json:"team"` //nolint:apiparamcheck // test helper; matches server response shape
	}
	s.DoJSON("PATCH", "/api/latest/fleet/teams/0", fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "https://example.com/webhook",
			PolicyIDs:      []uint{noTeamPol1.ID, noTeamPol2.ID}, // pol3 is NOT included
			HostBatchSize:  100,
		},
	}}, http.StatusOK, &defaultTeamResp)
	require.True(t, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.Enable)

	// Record policy results - all fail
	_, err = s.ds.RecordPolicyQueryExecutions(ctx, host, map[uint]*bool{
		noTeamPol1.ID: new(false), // Fails and is in webhook config
		noTeamPol2.ID: new(false), // Fails and is in webhook config
		noTeamPol3.ID: new(false), // Fails but NOT in webhook config
	}, time.Now(), false, nil)
	require.NoError(t, err)

	// Initially, OutdatedAutomationBatch should be empty (policies haven't been triggered for automation yet)
	pfs, err := s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(t, err)
	require.Empty(t, pfs, "initially no policies should be marked for automation")

	// Reset with empty arrays - should not mark any policies
	var resetResp struct{}
	s.DoJSON("POST", "/api/latest/fleet/automations/reset", fleet.ResetAutomationRequest{
		TeamIDs:   nil,
		PolicyIDs: []uint{},
	}, http.StatusOK, &resetResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(t, err)
	require.Empty(t, pfs, "empty reset should not mark any policies")

	// Test that we can configure and retrieve the "No Team" webhook settings
	defaultTeamResp = struct {
		Team *fleet.DefaultTeam `json:"team"` //nolint:apiparamcheck // test helper; matches server response shape
	}{}
	s.DoJSON("GET", "/api/latest/fleet/teams/0", nil, http.StatusOK, &defaultTeamResp)
	require.Equal(t, uint(0), defaultTeamResp.Team.ID)
	require.Equal(t, fleet.ReservedNameNoTeam, defaultTeamResp.Team.Name)
	require.True(t, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "https://example.com/webhook", defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	require.Equal(t, []uint{noTeamPol1.ID, noTeamPol2.ID}, defaultTeamResp.Team.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)

	// Now reset by team ID 0 to mark policies for automation
	s.DoJSON("POST", "/api/latest/fleet/automations/reset", fleet.ResetAutomationRequest{
		TeamIDs:   []uint{0}, // Team ID 0 for "No Team"
		PolicyIDs: nil,
	}, http.StatusOK, &resetResp)

	// After reset, policies should be marked for automation
	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(t, err)
	require.Len(t, pfs, 2, "should have 2 policies marked for automation (pol1 and pol2, not pol3)")

	// Create a failing policy set and trigger the webhook
	failingPolicySet := NewMemFailingPolicySet()
	for _, pf := range pfs {
		err = failingPolicySet.AddHost(pf.PolicyID, pf.Host)
		require.NoError(t, err)
	}

	// Trigger the webhook automation with a custom sendFunc that captures the call
	err = policies.TriggerFailingPoliciesAutomation(ctx, s.ds, slog.New(slog.DiscardHandler), failingPolicySet,
		func(pol *fleet.Policy, cfg policies.FailingPolicyAutomationConfig) error {
			webhookCalled = true
			capturedPolicy = pol
			// Verify the config is correct
			require.NotNil(t, cfg.WebhookURL)
			require.Equal(t, "https://example.com/webhook", cfg.WebhookURL.String())
			require.Equal(t, 100, cfg.HostBatchSize)
			return nil
		})
	require.NoError(t, err)

	// Verify the webhook was called
	require.True(t, webhookCalled, "webhook should have been called for No Team failing policy")

	// Verify the captured policy is one of the configured policies
	require.NotNil(t, capturedPolicy)
	require.Contains(t, []string{"no-team-failing-policy-1", "no-team-failing-policy-2"}, capturedPolicy.Name)

	// Verify the policy has team_id = 0 for "No Team"
	require.NotNil(t, capturedPolicy.TeamID)
	require.Equal(t, uint(0), *capturedPolicy.TeamID, "team_id should be 0 for No Team")

	// Verify the failing policy set has the correct host
	hosts, err := failingPolicySet.ListHosts(capturedPolicy.ID)
	require.NoError(t, err)
	require.Len(t, hosts, 1, "should have one host")
	require.Equal(t, "no-team-host", hosts[0].Hostname)
}

func (s *integrationEnterpriseTestSuite) TestResetAutomation() {
	ctx := context.Background()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	s.Require().NoError(err)

	createPol1 := &fleet.TeamPolicyResponse{}
	createPol1Req := &fleet.TeamPolicyRequest{
		Query:       "query",
		Name:        "name1",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    true,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol1Req, http.StatusOK, &createPol1)

	createPol2 := &fleet.TeamPolicyResponse{}
	createPol2Req := &fleet.TeamPolicyRequest{
		Query:       "query",
		Name:        "name2",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol2Req, http.StatusOK, &createPol2)

	createPol3 := &fleet.TeamPolicyResponse{}
	createPol3Req := &fleet.TeamPolicyRequest{
		Query:       "query",
		Name:        "name3",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol3Req, http.StatusOK, &createPol3)

	var tmResp teamResponse
	// modify the team's config - enable the webhook
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team1.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamSpecWebhookSettings{
		FailingPoliciesWebhook: &fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://127/",
			PolicyIDs:      []uint{createPol1.Policy.ID, createPol2.Policy.ID},
			HostBatchSize:  12345,
		},
	}}, http.StatusOK, &tmResp)

	h1, err := s.ds.NewHost(ctx, &fleet.Host{})
	s.Require().NoError(err)

	_, err = s.ds.RecordPolicyQueryExecutions(ctx, h1, map[uint]*bool{
		createPol1.Policy.ID: new(false),
		createPol2.Policy.ID: new(false),
		createPol3.Policy.ID: new(false), // This policy is not activated for automation in config.
	}, time.Now(), false, nil)
	s.Require().NoError(err)

	pfs, err := s.ds.OutdatedAutomationBatch(ctx)
	s.Require().NoError(err)
	s.Require().Empty(pfs)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", fleet.ResetAutomationRequest{
		TeamIDs:   nil,
		PolicyIDs: []uint{},
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	s.Require().NoError(err)
	s.Require().Empty(pfs)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", fleet.ResetAutomationRequest{
		TeamIDs:   nil,
		PolicyIDs: []uint{createPol1.Policy.ID, createPol2.Policy.ID, createPol3.Policy.ID},
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	s.Require().NoError(err)
	s.Require().Len(pfs, 2)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", fleet.ResetAutomationRequest{
		TeamIDs:   []uint{team1.ID},
		PolicyIDs: nil,
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	s.Require().NoError(err)
	s.Require().Len(pfs, 2)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", fleet.ResetAutomationRequest{
		TeamIDs:   nil,
		PolicyIDs: []uint{createPol2.Policy.ID},
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	s.Require().NoError(err)
	s.Require().Len(pfs, 1)
}

func (s *integrationEnterpriseTestSuite) TestPolicyAutomationsSoftwareInstallers() {
	t := s.T()
	ctx := context.Background()
	test.CreateInsertGlobalVPPToken(t, s.ds)

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)

	newHost := func(name string, teamID *uint, platform string) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   new(t.Name() + name),
			NodeKey:         new(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        platform,
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}
	newFleetdHost := func(name string, teamID *uint, platform string) *fleet.Host {
		h := newHost(name, teamID, platform)
		orbitKey := setOrbitEnrollment(t, h, s.ds)
		h.OrbitNodeKey = &orbitKey
		return h
	}

	host0NoTeam := newFleetdHost("host1NoTeam", nil, "darwin")
	host1Team1 := newFleetdHost("host1Team1", &team1.ID, "darwin")
	host2Team1 := newFleetdHost("host2Team1", &team1.ID, "ubuntu")
	host3Team2 := newFleetdHost("host3Team2", &team2.ID, "windows")
	hostVanillaOsquery5Team1 := newHost("hostVanillaOsquery5Team2", &team1.ID, "darwin")

	// Upload dummy_installer.pkg to team1.
	pkgPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some pkg install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team1.ID,
	}
	s.uploadSoftwareInstaller(t, pkgPayload, http.StatusOK, "")
	// Get software title ID of the uploaded installer.
	resp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "DummyApp",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	dummyInstallerPkgTitleID := resp.SoftwareTitles[0].ID
	var dummyInstallerPkg struct {
		ID        uint   `db:"id"`
		UserID    *uint  `db:"user_id"`
		UserName  string `db:"user_name"`
		UserEmail string `db:"user_email"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&dummyInstallerPkg,
			`SELECT id, user_id, user_name, user_email FROM software_installers WHERE global_or_team_id = ? AND filename = ?`,
			team1.ID, "dummy_installer.pkg",
		)
	})
	dummyInstallerPkgInstallerID := dummyInstallerPkg.ID
	require.NotZero(t, dummyInstallerPkgInstallerID)
	require.NotNil(t, dummyInstallerPkg.UserID)
	globalAdmin, err := s.ds.UserByEmail(ctx, "admin1@example.com")
	require.NoError(t, err)
	require.Equal(t, globalAdmin.ID, *dummyInstallerPkg.UserID)
	require.Equal(t, "Test Name admin1@example.com", dummyInstallerPkg.UserName)
	require.Equal(t, "admin1@example.com", dummyInstallerPkg.UserEmail)

	// Upload ruby.deb to team1 by a user who will be deleted.
	u2 := &fleet.User{
		Name:       "admin team1",
		Email:      "admin_team1@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team1,
				Role: fleet.RoleAdmin,
			},
		},
	}
	require.NoError(t, u2.SetPassword(test.GoodPassword, 10, 10))
	adminTeam1, err := s.ds.NewUser(context.Background(), u2)
	require.NoError(t, err)
	rubyPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some deb install script",
		Filename:      "ruby.deb",
		TeamID:        &team1.ID,
	}
	adminTeam1Session, err := s.ds.NewSession(ctx, adminTeam1.ID, 64)
	require.NoError(t, err)
	adminToken := s.token
	t.Cleanup(func() {
		s.token = adminToken
	})
	s.token = adminTeam1Session.Key // nolint:nilaway // cannot be nil due to previous require
	s.uploadSoftwareInstaller(t, rubyPayload, http.StatusOK, "")
	s.token = adminToken
	err = s.ds.DeleteUser(ctx, adminTeam1.ID)
	require.NoError(t, err)
	// Get software title ID of the uploaded installer.
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ruby",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	rubyDebTitleID := resp.SoftwareTitles[0].ID
	var rubyDeb struct {
		ID        uint   `db:"id"`
		UserID    *uint  `db:"user_id"`
		UserName  string `db:"user_name"`
		UserEmail string `db:"user_email"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&rubyDeb,
			`SELECT id, user_id, user_name, user_email FROM software_installers WHERE global_or_team_id = ? AND filename = ?`,
			team1.ID, "ruby.deb",
		)
	})
	rubyDebInstallerID := rubyDeb.ID
	require.NotZero(t, rubyDebInstallerID)
	require.Nil(t, rubyDeb.UserID)
	require.Equal(t, "admin team1", rubyDeb.UserName)
	require.Equal(t, "admin_team1@example.com", rubyDeb.UserEmail)

	// Upload fleet-osquery.msi to team2.
	fleetOsqueryPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "some msi install script",
		Filename:      "fleet-osquery.msi",
		TeamID:        &team2.ID,
		// Set as Self-service to check that the generated host_software_installs
		// is generated with self_service=false and the activity has the correct
		// author (the admin that uploaded the installer).
		SelfService: true,
	}
	s.uploadSoftwareInstaller(t, fleetOsqueryPayload, http.StatusOK, "")
	// Get software title ID of the uploaded installer.
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "Fleet osquery",
		"team_id", fmt.Sprintf("%d", team2.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	fleetOsqueryMSITitleID := resp.SoftwareTitles[0].ID
	var fleetOsqueryMSIInstallerID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&fleetOsqueryMSIInstallerID,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`,
			team2.ID, "fleet-osquery.msi",
		)
	})
	require.NotZero(t, fleetOsqueryMSIInstallerID)

	// Create a VPP app to test that policies *can* be assigned to them (VPP automation is tested in MDM integration test)
	_, err = s.ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name:             "App123 " + t.Name(),
		BundleIdentifier: "bid_" + t.Name(),
		AdamID:           "adam_test_vpp2", // max 16 chars
		Platform:         fleet.MacOSPlatform,
	}, &team1.ID, nil)
	require.NoError(t, err)
	// Get software title ID of the uploaded VPP app.
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "App123",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].AppStoreApp)
	vppAppTitleID := resp.SoftwareTitles[0].ID

	// Populate software for host1Team1 (to have a software title
	// that doesn't have an associated installer)
	software := []fleet.Software{
		{Name: "Foobar", Version: "0.0.1", Source: "apps"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host1Team1.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, time.Now()))
	// Get software title ID of the software.
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "Foobar",
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.Nil(t, resp.SoftwareTitles[0].SoftwarePackage)
	foobarAppTitleID := resp.SoftwareTitles[0].ID

	// policy0AllTeams is a global policy that runs on all devices.
	policy0AllTeams, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name:     "policy0AllTeams",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	// policy1Team1 runs on macOS devices.
	policy1Team1, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:     "policy1Team1",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	// policy2Team1 runs on macOS and Linux devices.
	policy2Team1, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:     "policy2Team1",
		Query:    "SELECT 2;",
		Platform: "linux,darwin",
	})
	require.NoError(t, err)
	// policy3Team1 runs on all devices in team1 (will have no associated installers).
	policy3Team1, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:  "policy3Team1",
		Query: "SELECT 3;",
	})
	require.NoError(t, err)
	// policy4Team2 runs on Windows devices.
	policy4Team2, err := s.ds.NewTeamPolicy(ctx, team2.ID, nil, fleet.PolicyPayload{
		Name:     "policy4Team2",
		Query:    "SELECT 4;",
		Platform: "windows",
	})
	require.NoError(t, err)

	// Attempt to associate to an unknown software title.
	mtplr := fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: 999_999},
	}, http.StatusBadRequest, &mtplr)
	// Attempt to associate to a software title without associated installer.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: foobarAppTitleID},
	}, http.StatusBadRequest, &mtplr)
	// Associate a VPP app to the policy (which we'll immediately overwrite)
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: vppAppTitleID},
	}, http.StatusOK, &mtplr)
	// Associate dummy_installer.pkg to policy1Team1.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: dummyInstallerPkgTitleID},
	}, http.StatusOK, &mtplr)
	// Change name only (to test not setting a software_title_id).
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID),
		json.RawMessage(`{"name": "policy1Team1_Renamed"}`), http.StatusOK, &mtplr,
	)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.SoftwareInstallerID)
	require.Equal(t, dummyInstallerPkgInstallerID, *policy1Team1.SoftwareInstallerID)
	require.Equal(t, "policy1Team1_Renamed", policy1Team1.Name)
	// Explicit set to 0 to disable.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: 0},
	}, http.StatusOK, &mtplr)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Nil(t, policy1Team1.SoftwareInstallerID)

	// re-add software installer to policy1Team1
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: dummyInstallerPkgTitleID},
	}, http.StatusOK, &mtplr)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.SoftwareInstallerID)
	require.Equal(t, dummyInstallerPkgInstallerID, *policy1Team1.SoftwareInstallerID)
	// Set to null to disable
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), []byte(`{
			"software_title_id": null
	}`), http.StatusOK)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Nil(t, policy1Team1.SoftwareInstallerID)

	host1LastInstall, err := s.ds.GetHostLastInstallData(ctx, host1Team1.ID, dummyInstallerPkgInstallerID)
	require.NoError(t, err)
	require.Nil(t, host1LastInstall)

	// Add some results and stats that should be cleared after setting an installer again.
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(1), policy1Team1.FailingHostCount)
	passes := true
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&passes,
			`SELECT passes FROM policy_membership WHERE policy_id = ? AND host_id = ?`,
			policy1Team1.ID, host1Team1.ID,
		)
	})
	require.False(t, passes)

	// Back to associating dummy_installer.pkg to policy1Team1.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: dummyInstallerPkgTitleID},
	}, http.StatusOK, &mtplr)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.SoftwareInstallerID)
	require.Equal(t, dummyInstallerPkgInstallerID, *policy1Team1.SoftwareInstallerID)
	// Policy stats and membership should be cleared from policy1Team1.
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(0), policy1Team1.FailingHostCount)
	countBiggerThanZero := true
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&countBiggerThanZero,
			`SELECT COUNT(*) > 0 FROM policy_membership WHERE policy_id = ?`,
			policy1Team1.ID,
		)
	})
	require.False(t, countBiggerThanZero)

	// Add (again) some results and stats that should be cleared after changing an existing installer.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(1), policy1Team1.FailingHostCount)
	passes = true
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&passes,
			`SELECT passes FROM policy_membership WHERE policy_id = ? AND host_id = ?`,
			policy1Team1.ID, host1Team1.ID,
		)
	})
	require.False(t, passes)

	// Change the installer (temporarily to test that changing an installer will clear results)
	// Associate ruby.deb to policy1Team1.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: rubyDebTitleID},
	}, http.StatusOK, &mtplr)

	// After changing the installer, membership and stats should be cleared.
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.SoftwareInstallerID)
	require.Equal(t, rubyDebInstallerID, *policy1Team1.SoftwareInstallerID)
	// Policy stats and membership should be cleared from policy1Team1.
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(0), policy1Team1.FailingHostCount)
	countBiggerThanZero = true
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&countBiggerThanZero,
			`SELECT COUNT(*) > 0 FROM policy_membership WHERE policy_id = ?`,
			policy1Team1.ID,
		)
	})
	require.False(t, countBiggerThanZero)

	// Back to (again) associating dummy_installer.pkg to policy1Team1.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: dummyInstallerPkgTitleID},
	}, http.StatusOK, &mtplr)

	// Associate ruby.deb to policy2Team1.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy2Team1.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: rubyDebTitleID},
	}, http.StatusOK, &mtplr)

	// We use DoJSONWithoutAuth for distributed/write because we want the requests to not have the
	// current user's "Authorization: Bearer <API_TOKEN>" header.

	// host1Team1 fails all policies on the first report.
	// Failing policy1Team1 means an install request must be generated.
	// Failing policy2Team1 should not trigger a install request because it has a .deb attached to it (does not apply to macOS hosts).
	// Failing policy3Team1 should do nothing because it doesn't have any installers associated to it.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: new(false),
			policy2Team1.ID: new(false),
			policy3Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host1Team1.ID, dummyInstallerPkgInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.NotEmpty(t, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host1LastInstall.Status)
	prevExecutionID := host1LastInstall.ExecutionID

	// Request a manual installation on the host for the same installer, which should fail.
	var installResp installSoftwareResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install",
		host1Team1.ID, dummyInstallerPkgTitleID), nil, http.StatusBadRequest, &installResp)

	// Submit same results as before, which should not trigger a installation because the policy is already failing.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: new(false),
			policy2Team1.ID: new(false),
			policy3Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host1Team1.ID, dummyInstallerPkgInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.Equal(t, prevExecutionID, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host1LastInstall.Status)

	// Submit same results but policy1Team1 now passes,
	// and then submit again but policy1Team1 fails.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: new(true),
			policy2Team1.ID: new(false),
			policy3Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: new(false),
			policy2Team1.ID: new(false),
			policy3Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	// Another installation should not be triggered because the last installation is pending.
	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host1Team1.ID, dummyInstallerPkgInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
	require.Equal(t, prevExecutionID, host1LastInstall.ExecutionID)
	require.NotNil(t, host1LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host1LastInstall.Status)

	// host2Team1 is failing policy2Team1 and policy3Team1 policies.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			policy2Team1.ID: new(false),
			policy3Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	host2LastInstall, err := s.ds.GetHostLastInstallData(ctx, host2Team1.ID, rubyDebInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host2LastInstall)
	require.NotEmpty(t, host2LastInstall.ExecutionID)
	require.NotNil(t, host2LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host2LastInstall.Status)

	// Associate fleet-osquery.msi to policy4Team2.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team2.ID, policy4Team2.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: fleetOsqueryMSITitleID},
	}, http.StatusOK, &mtplr)

	// host3Team2 reports a failing result for policy4Team2, which should trigger an installation.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			policy4Team2.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	host3LastInstall, err := s.ds.GetHostLastInstallData(ctx, host3Team2.ID, fleetOsqueryMSIInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host3LastInstall)
	require.NotEmpty(t, host3LastInstall.ExecutionID)
	require.NotNil(t, host3LastInstall.Status)
	require.Equal(t, fleet.SoftwareInstallPending, *host3LastInstall.Status)
	host3LastInstallDetails, err := s.ds.GetSoftwareInstallDetails(ctx, host3LastInstall.ExecutionID)
	require.NoError(t, err)
	// Even if fleet-osquery.msi was uploaded as Self-service, it was installed by Fleet, so
	// host3LastInstallDetails.SelfService should be false.
	require.False(t, host3LastInstallDetails.SelfService)

	//
	// The following increase coverage of policies result processing in distributed/write.
	//

	// host3Team2 reports a passing result for policy0AllTeams which is a global policy.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			policy0AllTeams.ID: new(true),
		},
	), http.StatusOK, &distributedResp)

	// host0NoTeam reports a failing result for policy0AllTeams which is a global policy.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host0NoTeam,
		map[uint]*bool{
			policy0AllTeams.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	// host3Team2 reports a failing result for policy0AllTeams which is a global policy.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			policy0AllTeams.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	// Unassociate policy4Team2 from installer.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team2.ID, policy4Team2.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: 0},
	}, http.StatusOK, &mtplr)

	// host3Team2 reports a failing result for policy4Team2.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			policy4Team2.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	// Upcoming activities for host1Team1 should show the automatic installation of dummy_installer.pkg.
	// Check the author should be the admin that uploaded the installer.
	var listUpcomingAct listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host1Team1.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 1)
	require.Nil(t, listUpcomingAct.Activities[0].ActorID)
	require.Equal(t, "Fleet", *listUpcomingAct.Activities[0].ActorFullName)
	require.Nil(t, listUpcomingAct.Activities[0].ActorEmail)

	//
	// Finally have orbit install the packages and check activities.
	//

	// host1Team1 posts the installation result for dummy_installer.pkg.
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *host1Team1.OrbitNodeKey, host1LastInstall.ExecutionID)), http.StatusNoContent)
	var host1InstallerHash string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &host1InstallerHash,
			`SELECT si.storage_id FROM host_software_installs hsi JOIN software_installers si ON hsi.software_installer_id = si.id WHERE hsi.execution_id = ?`,
			host1LastInstall.ExecutionID)
	})
	s.lastActivityMatches(fleet.ActivityTypeInstalledSoftware{}.ActivityName(), fmt.Sprintf(`{
		"host_id": %d,
		"host_display_name": "%s",
		"software_title": "%s",
		"software_package": "%s",
		"hash_sha256": "%s",
		"self_service": false,
		"install_uuid": "%s",
		"status": "installed",
		"source": "apps",
		"policy_id": %d,
		"policy_name": "%s",
		"from_setup_experience": false,
		"patch_when_closed": false
	}`, host1Team1.ID, host1Team1.DisplayName(), "DummyApp", "dummy_installer.pkg", host1InstallerHash, host1LastInstall.ExecutionID, policy1Team1.ID, policy1Team1.Name), 0)

	var activityCount int
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&activityCount,
			`SELECT count(1) FROM activity_past`,
		)
	})
	// host2Team1 posts the installation result for ruby.deb (first attempt).
	// An activity is created for every failed attempt, including retries.
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 1,
			"install_script_output": "failed"
		}`, *host2Team1.OrbitNodeKey, host2LastInstall.ExecutionID)), http.StatusNoContent)

	// host3Team2 posts the installation result for fleet-osquery.msi.
	// An activity is created for every failed attempt, including retries.
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "ok",
			"install_script_exit_code": 1,
			"install_script_output": "failed"
		}`, *host3Team2.OrbitNodeKey, host3LastInstall.ExecutionID)), http.StatusNoContent)
	var activityNewCount int
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&activityNewCount,
			`SELECT count(1) FROM activity_past`,
		)
	})
	require.Equal(t, activityCount+2, activityNewCount, "an activity should be created for each failed install attempt (including retries)")

	// hostVanillaOsquery5Team1 sends policy results with failed policies with associated installers.
	// Fleet should not queue an install for vanilla osquery hosts.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		hostVanillaOsquery5Team1,
		map[uint]*bool{
			policy1Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	hostVanillaOsquery5Team1LastInstall, err := s.ds.GetHostLastInstallData(ctx, hostVanillaOsquery5Team1.ID, dummyInstallerPkgInstallerID)
	require.NoError(t, err)
	require.Nil(t, hostVanillaOsquery5Team1LastInstall)
}

func (s *integrationEnterpriseTestSuite) TestPolicyAutomationSoftwareInstallRetries() {
	t := s.T()
	ctx := context.Background()

	// Create a team
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	// Create a host
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", t.Name()),
		Platform:        "darwin",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// Upload a software installer
	pkgPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}
	s.uploadSoftwareInstaller(t, pkgPayload, http.StatusOK, "")

	// Get the software title and installer ID
	var resp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{},
		http.StatusOK, &resp, "query", "DummyApp", "team_id", fmt.Sprintf("%d", team.ID))
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	softwareTitleID := resp.SoftwareTitles[0].ID

	var installerID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerID,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`,
			team.ID, "dummy_installer.pkg")
	})
	require.NotZero(t, installerID)

	// Create a team policy
	policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:     t.Name(),
		Query:    "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Platform: "darwin",
	})
	require.NoError(t, err)

	// Associate software installer with policy
	var modifyResp fleet.ModifyTeamPolicyResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: softwareTitleID},
	}, http.StatusOK, &modifyResp)
	require.NotNil(t, modifyResp.Policy)
	policy, err = s.ds.Policy(ctx, policy.ID)
	require.NoError(t, err)
	require.NotNil(t, policy.SoftwareInstallerID)
	require.Equal(t, installerID, *policy.SoftwareInstallerID)

	submitPolicyResult := func(policyID uint, passes bool) {
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
			host,
			map[uint]*bool{
				policyID: new(passes),
			},
		), http.StatusOK, &distributedResp)
	}
	getPendingInstall := func() *fleet.HostSoftwareInstallerResult {
		var results []*fleet.HostSoftwareInstallerResult
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &results, `
				SELECT
					id,
					execution_id,
					host_id,
					software_installer_id,
					policy_id,
					install_script_exit_code,
					attempt_number
				FROM host_software_installs
				WHERE host_id = ? AND install_script_exit_code IS NULL
				ORDER BY id ASC
			`, host.ID)
		})
		if len(results) == 0 {
			return nil
		}
		return results[0]
	}
	submitInstallResult := func(installUUID string, exitCode int) {
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(
			fmt.Sprintf(`{
				"orbit_node_key": %q,
				"install_uuid": %q,
				"install_script_exit_code": %d,
				"install_script_output": "install output"
			}`, *host.OrbitNodeKey, installUUID, exitCode),
		), http.StatusNoContent)
	}
	getInstallResults := func() []*fleet.HostSoftwareInstallerResult {
		var results []*fleet.HostSoftwareInstallerResult
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &results, `
				SELECT id, execution_id, host_id, software_installer_id, policy_id, install_script_exit_code, attempt_number
				FROM host_software_installs
				WHERE host_id = ? AND software_installer_id = ? AND policy_id = ?
				ORDER BY id ASC
			`, host.ID, installerID, policy.ID)
		})
		return results
	}
	countActivities := func() int {
		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &count, `
				SELECT COUNT(*)
				FROM activity_past
				WHERE JSON_EXTRACT(details, '$.host_id') = ? AND JSON_EXTRACT(details, '$.policy_id') = ?
			`, host.ID, policy.ID)
		})
		return count
	}

	// Policy fails, triggers first install
	submitPolicyResult(policy.ID, false)

	// Wait for install to be queued
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingInstall := getPendingInstall()
		assert.NotNil(t, pendingInstall, "install should be queued")
		if pendingInstall != nil && pendingInstall.SoftwareInstallerID != nil {
			assert.Equal(t, installerID, *pendingInstall.SoftwareInstallerID)
		}
		if pendingInstall != nil && pendingInstall.PolicyID != nil {
			assert.Equal(t, policy.ID, *pendingInstall.PolicyID)
		}
	}, 5*time.Second, 100*time.Millisecond)

	// Fail the first attempt
	pendingInstall := getPendingInstall()
	require.NotNil(t, pendingInstall)
	installUUID1 := pendingInstall.InstallUUID
	submitInstallResult(installUUID1, 1) // exit code 1 = failure

	// Verify attempt 1 and retry
	results := getInstallResults()
	require.Len(t, results, 2)
	// First result: attempt 1
	require.NotNil(t, results[0].AttemptNumber)
	require.Equal(t, 1, *results[0].AttemptNumber)
	require.NotNil(t, results[0].InstallScriptExitCode)
	require.Equal(t, 1, *results[0].InstallScriptExitCode)
	// Queued retry (attempt_number is NULL until result is submitted)
	require.Nil(t, results[1].InstallScriptExitCode)
	require.Nil(t, results[1].AttemptNumber)
	// Activity is created for every attempt (including retries), not only the final one.
	require.Equal(t, 1, countActivities())

	// Get pending retry
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingInstall := getPendingInstall()
		assert.NotNil(t, pendingInstall, "retry should be queued")
		if pendingInstall != nil {
			assert.NotEqual(t, installUUID1, pendingInstall.InstallUUID, "should be a new execution")
		}
	}, 5*time.Second, 100*time.Millisecond)

	// Fail attempt 2
	pendingInstall = getPendingInstall()
	require.NotNil(t, pendingInstall)
	installUUID2 := pendingInstall.InstallUUID
	submitInstallResult(installUUID2, 1)

	// Verify attempt 2 and retry
	results = getInstallResults()
	require.Len(t, results, 3)
	require.NotNil(t, results[1].AttemptNumber)
	require.Equal(t, 2, *results[1].AttemptNumber)
	require.Nil(t, results[2].AttemptNumber)
	require.Equal(t, 2, countActivities())

	// Get second pending retry
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingInstall := getPendingInstall()
		assert.NotNil(t, pendingInstall, "second retry should be queued")
	}, 5*time.Second, 100*time.Millisecond)

	// Fail attempt 3
	pendingInstall = getPendingInstall()
	require.NotNil(t, pendingInstall)
	installUUID3 := pendingInstall.InstallUUID
	submitInstallResult(installUUID3, 1)

	// Verify attempt 3 is final
	results = getInstallResults()
	require.Len(t, results, 3, "should have exactly 3 completed attempts, no more retries queued")
	require.NotNil(t, results[2].AttemptNumber)
	require.Equal(t, 3, *results[2].AttemptNumber)
	require.NotNil(t, results[2].InstallScriptExitCode, "final attempt should be completed")

	// One activity per attempt — three failed attempts produce three activities.
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		assert.Equal(t, 3, countActivities(), "activity should be created for each retry attempt")
	}, 5*time.Second, 100*time.Millisecond)

	// No more retries, max reached
	time.Sleep(2 * time.Second)
	pendingInstall = getPendingInstall()
	require.Nil(t, pendingInstall)

	// Exactly 3 attempts
	results = getInstallResults()
	require.Len(t, results, 3)
	require.Equal(t, 1, *results[0].AttemptNumber)
	require.Equal(t, 2, *results[1].AttemptNumber)
	require.Equal(t, 3, *results[2].AttemptNumber)

	// Pass policy
	submitPolicyResult(policy.ID, true)
	// Fail policy to trigger first attempt
	submitPolicyResult(policy.ID, false)

	// Wait for first attempt
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingInstall := getPendingInstall()
		assert.NotNil(t, pendingInstall)
	}, 5*time.Second, 100*time.Millisecond)

	// Fail result
	pendingInstall = getPendingInstall()
	submitInstallResult(pendingInstall.InstallUUID, 1)

	// Verify retry is queued
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingInstall := getPendingInstall()
		assert.NotNil(t, pendingInstall)
	}, 5*time.Second, 100*time.Millisecond)

	// Make the policy pass (should stop retrying)
	submitPolicyResult(policy.ID, true)

	// Submit the pending retry result
	pendingInstall = getPendingInstall()
	submitInstallResult(pendingInstall.InstallUUID, 1)

	// Ensure no more retries (policy is now passing)
	time.Sleep(2 * time.Second)
	pendingInstall = getPendingInstall()
	require.Nil(t, pendingInstall)
	// Verify that all attempts were reset to 0 (policy passed, marking entire sequence as obsolete)
	results = getInstallResults()
	require.GreaterOrEqual(t, len(results), 4)
	require.NotNil(t, results[len(results)-1].AttemptNumber)
	require.Equal(t, 0, *results[len(results)-1].AttemptNumber)
	require.NotNil(t, results[len(results)-2].AttemptNumber)
	require.Equal(t, 0, *results[len(results)-2].AttemptNumber)
	cnt, err := s.ds.CountHostSoftwareInstallAttempts(ctx, host.ID, installerID, policy.ID)
	require.NoError(t, err)
	require.Equal(t, 0, cnt)

	// Verify that we can start a fresh retry sequence after the reset
	// Fail the policy again to trigger a new automation
	submitPolicyResult(policy.ID, false)

	// first attempt
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingInstall := getPendingInstall()
		assert.NotNil(t, pendingInstall, "should have queued a new first attempt")
	}, 5*time.Second, 100*time.Millisecond)

	// Submit failure for first attempt
	pendingInstall = getPendingInstall()
	submitInstallResult(pendingInstall.InstallUUID, 1)

	// Wait for retry to be queued
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingInstall := getPendingInstall()
		assert.NotNil(t, pendingInstall, "should have queued retry")
	}, 5*time.Second, 100*time.Millisecond)

	// Verify fresh sequence started at attempt_number = 1
	results = getInstallResults()
	require.GreaterOrEqual(t, len(results), 2, "should have at least completed attempt and pending retry")
	require.Nil(t, results[len(results)-1].AttemptNumber, "pending retry should have attempt_number = NULL")
	completedAttempt := results[len(results)-2]
	require.NotNil(t, completedAttempt.AttemptNumber)
	require.Equal(t, 1, *completedAttempt.AttemptNumber, "fresh sequence should start at attempt 1")

	cnt, err = s.ds.CountHostSoftwareInstallAttempts(ctx, host.ID, installerID, policy.ID)
	require.NoError(t, err)
	require.Equal(t, 2, cnt, "should count completed attempt and pending retry")

	// Verify database
	var attemptCounts struct {
		TotalRows     int `db:"total_rows"`
		ZeroCount     int `db:"zero_count"`
		PositiveCount int `db:"positive_count"`
		NullCount     int `db:"null_count"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &attemptCounts, `
			SELECT
				COUNT(*) as total_rows,
				SUM(CASE WHEN attempt_number = 0 THEN 1 ELSE 0 END) as zero_count,
				SUM(CASE WHEN attempt_number > 0 THEN 1 ELSE 0 END) as positive_count,
				SUM(CASE WHEN attempt_number IS NULL THEN 1 ELSE 0 END) as null_count
			FROM host_software_installs
			WHERE host_id = ? AND software_installer_id = ? AND policy_id = ?
		`, host.ID, installerID, policy.ID)
	})
	require.GreaterOrEqual(t, attemptCounts.TotalRows, 6, "should have at least 6 total rows (3 old + 2 new + 1 pending)")
	require.GreaterOrEqual(t, attemptCounts.ZeroCount, 5, "should have at least 5 rows with attempt_number=0 (old sequences)")
	require.Equal(t, 1, attemptCounts.PositiveCount, "should have exactly 1 row with attempt_number > 0 (new sequence)")
	require.Equal(t, 1, attemptCounts.NullCount, "should have exactly 1 row with attempt_number IS NULL (pending)")
}

func (s *integrationEnterpriseTestSuite) TestPolicyAutomationScriptActivitiesAcrossRetries() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", t.Name()),
		Platform:        "darwin",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// Saved script on the team (darwin host => non-.ps1 name so it isn't skipped).
	script, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "failing-policy-script.sh",
		ScriptContents: "echo failing",
		TeamID:         &team.ID,
	})
	require.NoError(t, err)

	// Team policy with the script as its failing-policy automation.
	policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:     t.Name(),
		Query:    "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Platform: "darwin",
	})
	require.NoError(t, err)

	var modifyResp fleet.ModifyTeamPolicyResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy.ID), fleet.ModifyTeamPolicyRequest{
		ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: script.ID},
	}, http.StatusOK, &modifyResp)
	policy, err = s.ds.Policy(ctx, policy.ID)
	require.NoError(t, err)
	require.NotNil(t, policy.ScriptID)
	require.Equal(t, script.ID, *policy.ScriptID)

	submitPolicyResult := func(passes bool) {
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
			host, map[uint]*bool{policy.ID: new(passes)},
		), http.StatusOK, &distributedResp)
	}
	// execution_id of the currently-pending (no exit code yet) script run, if any.
	getPendingExecutionID := func() string {
		var execIDs []string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &execIDs, `
				SELECT execution_id FROM host_script_results
				WHERE host_id = ? AND policy_id = ? AND exit_code IS NULL
				ORDER BY id ASC
			`, host.ID, policy.ID)
		})
		if len(execIDs) == 0 {
			return ""
		}
		return execIDs[0]
	}
	submitScriptResult := func(execID string, exitCode int) {
		var orbitPostScriptResp fleet.OrbitPostScriptResultResponse
		s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": %d, "output": "fail output"}`, *host.OrbitNodeKey, execID, exitCode)),
			http.StatusOK, &orbitPostScriptResp)
	}

	// Drive MaxPolicyAutomationRetries failing attempts. Each failed result records
	// a ran_script activity and queues the next retry (a NEW execution_id), up to the
	// cap. The policy stays failing for the host, so retries keep firing.
	submitPolicyResult(false)
	var execIDs []string
	for i := range fleet.MaxPolicyAutomationRetries {
		var execID string
		require.EventuallyWithT(t, func(c *assert.CollectT) {
			execID = getPendingExecutionID()
			assert.NotEmpty(c, execID, "a script run should be queued for attempt %d", i+1)
		}, 5*time.Second, 100*time.Millisecond)
		require.NotContains(t, execIDs, execID, "each retry must be a distinct execution")
		execIDs = append(execIDs, execID)
		submitScriptResult(execID, 1) // non-zero exit = failure
	}
	require.Len(t, execIDs, fleet.MaxPolicyAutomationRetries)

	ranScriptType := fleet.ActivityTypeRanScript{}.ActivityName()

	// The automation activities endpoint returns one ran_script row per attempt for
	// this host — multiple entries, not just the latest.
	var listResp fleet.ListPolicyAutomationActivitiesResponse
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		listResp = fleet.ListPolicyAutomationActivitiesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d/automation_activities", policy.ID),
			nil, http.StatusOK, &listResp)
		ranScript := 0
		for _, a := range listResp.Activities {
			if a.Type == ranScriptType {
				ranScript++
			}
		}
		assert.Equal(c, fleet.MaxPolicyAutomationRetries, ranScript, "one ran_script activity per retry attempt")
	}, 5*time.Second, 100*time.Millisecond)

	// Every ran_script row is for this host, marked as an error outcome, and is a
	// distinct activity (one per execution).
	seen := make(map[uint]struct{})
	for _, a := range listResp.Activities {
		if a.Type != ranScriptType {
			continue
		}
		require.Equal(t, host.ID, a.HostID)
		require.Equal(t, "error", a.Status)
		seen[a.ID] = struct{}{}
	}
	require.Len(t, seen, fleet.MaxPolicyAutomationRetries, "each retry attempt is a distinct activity")
}

func (s *integrationEnterpriseTestSuite) TestPolicyAutomationsSoftwareInstallersLabelScoping() {
	t := s.T()
	ctx := context.Background()
	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "linux",
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// Create a few labels
	var newLabelResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/v1/fleet/labels", fleet.LabelPayload{
		Name:  uuid.NewString(),
		Query: "SELECT 1",
	}, http.StatusOK, &newLabelResp)
	lbl1 := newLabelResp.Label

	newLabelResp = fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/v1/fleet/labels", fleet.LabelPayload{
		Name:  uuid.NewString(),
		Query: "SELECT 2",
	}, http.StatusOK, &newLabelResp)
	lbl2 := newLabelResp.Label

	newLabelResp = fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/v1/fleet/labels", fleet.LabelPayload{
		Name:  uuid.NewString(),
		Query: "SELECT 3",
	}, http.StatusOK, &newLabelResp)
	lbl3 := newLabelResp.Label

	// Add label1 and label2 to the host
	err = s.ds.RecordLabelQueryExecutions(context.Background(), host, map[uint]*bool{lbl1.ID: new(true), lbl2.ID: new(true)}, time.Now(), false)
	require.NoError(t, err)

	// upload software. Add label1 and label3 as "exclude any" labels.
	rubyPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "some deb install script",
		Filename:         "ruby.deb",
		TeamID:           nil,
		LabelsExcludeAny: []string{lbl1.Name, lbl3.Name},
		Platform:         "linux",
	}
	s.uploadSoftwareInstaller(t, rubyPayload, http.StatusOK, "")

	// Get software title ID of the uploaded installer.
	resp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ruby",
		"team_id", "0",
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	rubyDebTitleID := resp.SoftwareTitles[0].ID

	var rubyDetail getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", rubyDebTitleID), nil, http.StatusOK, &rubyDetail)
	require.NotNil(t, rubyDetail.SoftwareTitle)
	require.NotNil(t, rubyDetail.SoftwareTitle.SoftwarePackage)
	rubyInstallerID := rubyDetail.SoftwareTitle.SoftwarePackage.InstallerID

	policy1, err := s.ds.NewTeamPolicy(ctx, 0, nil, fleet.PolicyPayload{
		Name:     "policy1",
		Query:    "SELECT 1;",
		Platform: "linux",
	})
	require.NoError(t, err)

	mtplr := fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/0/policies/%d", policy1.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: rubyDebTitleID},
	}, http.StatusOK, &mtplr)

	host1LastInstall, err := s.ds.GetHostLastInstallData(ctx, host.ID, rubyInstallerID)
	require.NoError(t, err)
	require.Nil(t, host1LastInstall)

	// Send back a failed result for the policy.
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1, err = s.ds.Policy(ctx, policy1.ID)
	require.NoError(t, err)
	// Because the installer is not in scope, we do not mark the policy as failed.
	require.Equal(t, uint(0), policy1.PassingHostCount)
	require.Equal(t, uint(0), policy1.FailingHostCount)

	// No installation attempt, because we skipped due to label scoping
	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host.ID, rubyInstallerID)
	require.NoError(t, err)
	require.Nil(t, host1LastInstall)

	vimPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "some deb install script",
		Filename:         "vim.deb",
		TeamID:           nil,
		LabelsIncludeAny: []string{lbl1.Name, lbl2.Name},
		Platform:         "linux",
	}
	s.uploadSoftwareInstaller(t, vimPayload, http.StatusOK, "")

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "vim",
		"team_id", "0",
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	vimTitleID := resp.SoftwareTitles[0].ID

	var vimDetail getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", vimTitleID), nil, http.StatusOK, &vimDetail)
	require.NotNil(t, vimDetail.SoftwareTitle)
	require.NotNil(t, vimDetail.SoftwareTitle.SoftwarePackage)
	vimInstallerID := vimDetail.SoftwareTitle.SoftwarePackage.InstallerID

	policy2, err := s.ds.NewTeamPolicy(ctx, 0, nil, fleet.PolicyPayload{
		Name:     "policy2",
		Query:    "SELECT 2;",
		Platform: "linux",
	})
	require.NoError(t, err)

	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/0/policies/%d", policy2.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: vimTitleID},
	}, http.StatusOK, &mtplr)

	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host.ID, vimInstallerID)
	require.NoError(t, err)
	require.Nil(t, host1LastInstall)

	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy2.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy2, err = s.ds.Policy(ctx, policy2.ID)
	require.NoError(t, err)
	// Because the installer is in scope, we do mark the policy as failed.
	require.Equal(t, uint(0), policy2.PassingHostCount)
	require.Equal(t, uint(1), policy2.FailingHostCount)

	// We have an installation attempt for vim, because it's in scope
	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host.ID, vimInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)

	// --- include_all tests ---
	//
	// Use a dedicated team so we can reuse the ruby.deb and vim.deb testdata
	// files without conflicting with the no-team installers already uploaded above.
	var inclAllTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", fleet.Team{Name: t.Name() + "-include-all"}, http.StatusOK, &inclAllTeamResp)
	inclAllTeam := inclAllTeamResp.Team
	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&inclAllTeam.ID, []uint{host.ID}))
	require.NoError(t, err)

	// Host has lbl1 and lbl2. lbl3 is not on the host (lbl3 was never added
	// via RecordLabelQueryExecutions in this test above).

	// Upload ruby.deb with labels_include_all: [lbl1, lbl2].
	// Host has both labels, so it IS in scope and install should be attempted.
	rubyIncludeAllPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "some deb install script",
		Filename:         "ruby.deb",
		TeamID:           &inclAllTeam.ID,
		LabelsIncludeAll: []string{lbl1.Name, lbl2.Name},
		Platform:         "linux",
	}
	s.uploadSoftwareInstaller(t, rubyIncludeAllPayload, http.StatusOK, "")

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ruby",
		"team_id", fmt.Sprint(inclAllTeam.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	rubyInclAllTitleID := resp.SoftwareTitles[0].ID

	var rubyInclAllDetail getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", rubyInclAllTitleID), nil, http.StatusOK, &rubyInclAllDetail, "team_id", fmt.Sprint(inclAllTeam.ID))
	require.NotNil(t, rubyInclAllDetail.SoftwareTitle)
	require.NotNil(t, rubyInclAllDetail.SoftwareTitle.SoftwarePackage)
	rubyInclAllInstallerID := rubyInclAllDetail.SoftwareTitle.SoftwarePackage.InstallerID

	policy3, err := s.ds.NewTeamPolicy(ctx, inclAllTeam.ID, nil, fleet.PolicyPayload{
		Name:     "policy3",
		Query:    "SELECT 3;",
		Platform: "linux",
	})
	require.NoError(t, err)

	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", inclAllTeam.ID, policy3.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: rubyInclAllTitleID},
	}, http.StatusOK, &mtplr)

	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host.ID, rubyInclAllInstallerID)
	require.NoError(t, err)
	require.Nil(t, host1LastInstall)

	// Send back a failed result for policy3.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy3.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy3, err = s.ds.Policy(ctx, policy3.ID)
	require.NoError(t, err)
	// Host has all required labels, so the installer is in scope and the policy is marked failed.
	require.Equal(t, uint(0), policy3.PassingHostCount)
	require.Equal(t, uint(1), policy3.FailingHostCount)

	// Installation attempt was made for ruby, because host has all required labels.
	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host.ID, rubyInclAllInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)

	// Upload vim.deb with labels_include_all: [lbl1, lbl3].
	// Host has lbl1 but NOT lbl3, so it is NOT in scope and install should be skipped.
	vimIncludeAllPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "some deb install script",
		Filename:         "vim.deb",
		TeamID:           &inclAllTeam.ID,
		LabelsIncludeAll: []string{lbl1.Name, lbl3.Name},
		Platform:         "linux",
	}
	s.uploadSoftwareInstaller(t, vimIncludeAllPayload, http.StatusOK, "")

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "vim",
		"team_id", fmt.Sprint(inclAllTeam.ID),
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	vimInclAllTitleID := resp.SoftwareTitles[0].ID

	var vimInclAllDetail getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", vimInclAllTitleID), nil, http.StatusOK, &vimInclAllDetail, "team_id", fmt.Sprint(inclAllTeam.ID))
	require.NotNil(t, vimInclAllDetail.SoftwareTitle)
	require.NotNil(t, vimInclAllDetail.SoftwareTitle.SoftwarePackage)
	vimInclAllInstallerID := vimInclAllDetail.SoftwareTitle.SoftwarePackage.InstallerID

	policy4, err := s.ds.NewTeamPolicy(ctx, inclAllTeam.ID, nil, fleet.PolicyPayload{
		Name:     "policy4",
		Query:    "SELECT 4;",
		Platform: "linux",
	})
	require.NoError(t, err)

	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", inclAllTeam.ID, policy4.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: vimInclAllTitleID},
	}, http.StatusOK, &mtplr)

	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host.ID, vimInclAllInstallerID)
	require.NoError(t, err)
	require.Nil(t, host1LastInstall)

	// Send back a failed result for policy4.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy4.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy4, err = s.ds.Policy(ctx, policy4.ID)
	require.NoError(t, err)
	// Host is missing lbl3, so the installer is not in scope. Policy should not be counted as failed.
	require.Equal(t, uint(0), policy4.PassingHostCount)
	require.Equal(t, uint(0), policy4.FailingHostCount)

	// No installation attempt for vim, because host is missing one of the required labels.
	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host.ID, vimInclAllInstallerID)
	require.NoError(t, err)
	require.Nil(t, host1LastInstall)

	// Now add lbl3 to the host and re-run the policy failure. vim should now be in scope.
	err = s.ds.RecordLabelQueryExecutions(context.Background(), host, map[uint]*bool{lbl3.ID: new(true)}, time.Now(), false)
	require.NoError(t, err)

	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy4.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy4, err = s.ds.Policy(ctx, policy4.ID)
	require.NoError(t, err)
	// Host now has all required labels, so the policy is marked as failed.
	require.Equal(t, uint(0), policy4.PassingHostCount)
	require.Equal(t, uint(1), policy4.FailingHostCount)

	// Installation attempt was made for vim now that the host has all required labels.
	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host.ID, vimInclAllInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)
}

func (s *integrationEnterpriseTestSuite) TestPolicyAutomationLabelScopingRetrigger() {
	t := s.T()
	ctx := context.Background()
	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "linux",
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// Create a few labels
	var newLabelResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/v1/fleet/labels", fleet.LabelPayload{
		Name:  uuid.NewString(),
		Query: "SELECT 1",
	}, http.StatusOK, &newLabelResp)
	lbl1 := newLabelResp.Label

	newLabelResp = fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/v1/fleet/labels", fleet.LabelPayload{
		Name:  uuid.NewString(),
		Query: "SELECT 2",
	}, http.StatusOK, &newLabelResp)
	lbl2 := newLabelResp.Label

	newLabelResp = fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/v1/fleet/labels", fleet.LabelPayload{
		Name:  uuid.NewString(),
		Query: "SELECT 3",
	}, http.StatusOK, &newLabelResp)
	lbl3 := newLabelResp.Label

	// Add label1 and label2 to the host
	err = s.ds.RecordLabelQueryExecutions(context.Background(), host, map[uint]*bool{lbl1.ID: new(true), lbl2.ID: new(true)}, time.Now(), false)
	require.NoError(t, err)

	// upload software. Add label1 and label3 as "exclude any" labels.
	rubyPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:    "some deb install script",
		Filename:         "ruby.deb",
		TeamID:           nil,
		LabelsIncludeAny: []string{lbl1.Name, lbl3.Name},
		Platform:         "linux",
	}
	s.uploadSoftwareInstaller(t, rubyPayload, http.StatusOK, "")

	// Get software title ID of the uploaded installer.
	resp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ruby",
		"team_id", "0",
	)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotNil(t, resp.SoftwareTitles[0].SoftwarePackage)
	rubyDebTitleID := resp.SoftwareTitles[0].ID

	var rubyDetail getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", rubyDebTitleID), nil, http.StatusOK, &rubyDetail)
	require.NotNil(t, rubyDetail.SoftwareTitle)
	require.NotNil(t, rubyDetail.SoftwareTitle.SoftwarePackage)
	rubyInstallerID := rubyDetail.SoftwareTitle.SoftwarePackage.InstallerID

	policy1, err := s.ds.NewTeamPolicy(ctx, 0, nil, fleet.PolicyPayload{
		Name:     "policy1",
		Query:    "SELECT 1;",
		Platform: "linux",
	})
	require.NoError(t, err)

	mtplr := fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/0/policies/%d", policy1.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: rubyDebTitleID},
	}, http.StatusOK, &mtplr)

	// No install attempt yet
	host1LastInstall, err := s.ds.GetHostLastInstallData(ctx, host.ID, rubyInstallerID)
	require.NoError(t, err)
	require.Nil(t, host1LastInstall)

	// Send back a failed result for the policy.
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1, err = s.ds.Policy(ctx, policy1.ID)
	require.NoError(t, err)

	// Because the installer is in scope, the policy is failing
	require.Equal(t, uint(0), policy1.PassingHostCount)
	require.Equal(t, uint(1), policy1.FailingHostCount)

	// We've triggered an installation attempt
	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host.ID, rubyInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)

	// Update the installer's labels to "exclude any". This de-scopes the software.
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		InstallScript:     new("some install script"),
		PreInstallQuery:   new("some pre install query"),
		PostInstallScript: new("some post install script"),
		Filename:          "ruby.deb",
		TitleID:           rubyDebTitleID,
		TeamID:            nil,
		LabelsExcludeAny:  []string{lbl1.Name, lbl3.Name},
	}, http.StatusOK, "")

	// The update should clear out the installation attempt
	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host.ID, rubyInstallerID)
	require.NoError(t, err)
	require.Nil(t, host1LastInstall)

	// Update the installer's labels to be "include any" again. The software is now back in scope.
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		InstallScript:     new("some install script"),
		PreInstallQuery:   new("some pre install query"),
		PostInstallScript: new("some post install script"),
		Filename:          "ruby.deb",
		TitleID:           rubyDebTitleID,
		TeamID:            nil,
		LabelsIncludeAny:  []string{lbl1.Name, lbl3.Name},
	}, http.StatusOK, "")

	// Simulate a failure of the policy
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1, err = s.ds.Policy(ctx, policy1.ID)
	require.NoError(t, err)
	// Because the installer is in scope, the policy should be failing again
	require.Equal(t, uint(0), policy1.PassingHostCount)
	require.Equal(t, uint(1), policy1.FailingHostCount)

	// We have an installation attempt again; the policy automation has been re-triggered
	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host.ID, rubyInstallerID)
	require.NoError(t, err)
	require.NotNil(t, host1LastInstall)

	// Update the include any labels. The host doesn't have label2, so this means that the software
	// moved out of scope.
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		InstallScript:     new("some install script"),
		PreInstallQuery:   new("some pre install query"),
		PostInstallScript: new("some post install script"),
		Filename:          "ruby.deb",
		TitleID:           rubyDebTitleID,
		TeamID:            nil,
		LabelsIncludeAny:  []string{lbl2.Name},
	}, http.StatusOK, "")

	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host.ID, rubyInstallerID)
	require.NoError(t, err)
	require.Nil(t, host1LastInstall)

	// Update to exclude any with label 2. This moves the software back into scope. The policy
	// automation should re-trigger.
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		InstallScript:     new("some install script"),
		PreInstallQuery:   new("some pre install query"),
		PostInstallScript: new("some post install script"),
		Filename:          "ruby.deb",
		TitleID:           rubyDebTitleID,
		TeamID:            nil,
		LabelsExcludeAny:  []string{lbl2.Name},
	}, http.StatusOK, "")

	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1, err = s.ds.Policy(ctx, policy1.ID)
	require.NoError(t, err)
	// Because the installer is in scope, the policy should be failing again.
	require.Equal(t, uint(0), policy1.PassingHostCount)
	require.Equal(t, uint(1), policy1.FailingHostCount)

	// We have an installation attempt again.
	host1LastInstall, err = s.ds.GetHostLastInstallData(ctx, host.ID, rubyInstallerID)
	require.NoError(t, err)
	require.Nil(t, host1LastInstall)
}

func (s *integrationEnterpriseTestSuite) TestPolicyAutomationsScripts() {
	t := s.T()
	ctx := context.Background()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)

	newHost := func(name string, teamID *uint, platform string) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   new(t.Name() + name),
			NodeKey:         new(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        platform,
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}
	newFleetdHost := func(name string, teamID *uint, platform string) *fleet.Host {
		h := newHost(name, teamID, platform)
		orbitKey := setOrbitEnrollment(t, h, s.ds)
		h.OrbitNodeKey = &orbitKey
		return h
	}

	host1Team1 := newFleetdHost("host1Team1", &team1.ID, "darwin")
	host2Team1 := newFleetdHost("host2Team1", &team1.ID, "ubuntu")
	host3Team2 := newFleetdHost("host3Team2", &team2.ID, "windows")
	hostVanillaOsquery5Team1 := newHost("hostVanillaOsquery5Team2", &team1.ID, "darwin")

	// Upload script to team1.
	script, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "unix-script.sh",
		ScriptContents: "echo 'Hello World'",
		TeamID:         &team1.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, script.ID)

	// Upload winScript to team1.
	winScript, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "windows-script.ps1",
		ScriptContents: "beep boop I am a windoge",
		TeamID:         &team1.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, winScript.ID)

	// Upload script to team2.
	psScript, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "windows-script.ps1",
		ScriptContents: "beep boop I am a window",
		TeamID:         &team2.ID,
	})
	require.NoError(t, err)
	require.NotZero(t, psScript.ID)

	// craete a global policy that runs on all devices.
	_, err = s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name:     "policy0AllTeams",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	// policy1Team1 runs on macOS devices.
	policy1Team1, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:     "policy1Team1",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	// policy2Team1 runs on macOS and Linux devices.
	policy2Team1, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:     "policy2Team1",
		Query:    "SELECT 2;",
		Platform: "linux,darwin",
	})
	require.NoError(t, err)
	// policy3Team1 runs on all devices in team1 (will have no associated scripts).
	policy3Team1, err := s.ds.NewTeamPolicy(ctx, team1.ID, nil, fleet.PolicyPayload{
		Name:  "policy3Team1",
		Query: "SELECT 3;",
	})
	require.NoError(t, err)
	// policy4Team2 runs on Windows devices.
	policy4Team2, err := s.ds.NewTeamPolicy(ctx, team2.ID, nil, fleet.PolicyPayload{
		Name:     "policy4Team2",
		Query:    "SELECT 4;",
		Platform: "windows",
	})
	require.NoError(t, err)

	// Attempt to associate to an unknown script.
	mtplr := fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: 999_999},
	}, http.StatusBadRequest, &mtplr)
	// Associate first script to policy1Team1.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: script.ID},
	}, http.StatusOK, &mtplr)
	// Change name only (to test not setting a script_id).
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID),
		json.RawMessage(`{"name": "policy1Team1_Renamed"}`), http.StatusOK, &mtplr,
	)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.ScriptID)
	require.Equal(t, script.ID, *policy1Team1.ScriptID)
	require.Equal(t, "policy1Team1_Renamed", policy1Team1.Name)
	// Explicit set to 0 to disable.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: 0},
	}, http.StatusOK, &mtplr)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Nil(t, policy1Team1.ScriptID)

	// re-add script to policy1Team1.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: script.ID},
	}, http.StatusOK, &mtplr)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.ScriptID)
	require.Equal(t, script.ID, *policy1Team1.ScriptID)
	// set to null to disable
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), []byte(`{
		"script_id": null
	}`), http.StatusOK)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Nil(t, policy1Team1.ScriptID)

	// Add some results and stats that should be cleared after updating the script
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(1), policy1Team1.FailingHostCount)
	passes := true
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&passes,
			`SELECT passes FROM policy_membership WHERE policy_id = ? AND host_id = ?`,
			policy1Team1.ID, host1Team1.ID,
		)
	})
	require.False(t, passes)

	// Back to associating the script with policy1Team1.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: script.ID},
	}, http.StatusOK, &mtplr)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.ScriptID)
	require.Equal(t, script.ID, *policy1Team1.ScriptID)
	// Policy stats and membership should be cleared from policy1Team1.
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(0), policy1Team1.FailingHostCount)
	countBiggerThanZero := true
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&countBiggerThanZero,
			`SELECT COUNT(*) > 0 FROM policy_membership WHERE policy_id = ?`,
			policy1Team1.ID,
		)
	})
	require.False(t, countBiggerThanZero)

	// Add (again) some results and stats that should be cleared after changing an existing script.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(1), policy1Team1.FailingHostCount)
	passes = true
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&passes,
			`SELECT passes FROM policy_membership WHERE policy_id = ? AND host_id = ?`,
			policy1Team1.ID, host1Team1.ID,
		)
	})
	require.False(t, passes)

	// Change the script (temporarily to test that changing a script will clear results)
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: winScript.ID},
	}, http.StatusOK, &mtplr)

	// After changing the script, membership and stats should be cleared.
	policy1Team1, err = s.ds.Policy(ctx, policy1Team1.ID)
	require.NoError(t, err)
	require.NotNil(t, policy1Team1.ScriptID)
	require.Equal(t, winScript.ID, *policy1Team1.ScriptID)
	// Policy stats and membership should be cleared from policy1Team1.
	require.Equal(t, uint(0), policy1Team1.PassingHostCount)
	require.Equal(t, uint(0), policy1Team1.FailingHostCount)
	countBiggerThanZero = true
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q,
			&countBiggerThanZero,
			`SELECT COUNT(*) > 0 FROM policy_membership WHERE policy_id = ?`,
			policy1Team1.ID,
		)
	})
	require.False(t, countBiggerThanZero)

	// Back to (again) associating first script to policy1Team1.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: script.ID},
	}, http.StatusOK, &mtplr)

	// Associate winScript to policy2Team1.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, policy2Team1.ID), fleet.ModifyTeamPolicyRequest{
		ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: winScript.ID},
	}, http.StatusOK, &mtplr)

	// We use DoJSONWithoutAuth for distributed/write because we want the requests to not have the
	// current user's "Authorization: Bearer <API_TOKEN>" header.

	// host1Team1 fails all policies on the first report.
	// Failing policy1Team1 means a script run must be generated.
	// Failing policy2Team1 should not trigger a script run because it has a PowerShell script attached to it (doesn't apply to macOS).
	// Failing policy3Team1 should do nothing because it doesn't have any scripts associated to it.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: new(false),
			policy2Team1.ID: new(false),
			policy3Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	hostPendingScript, err := s.ds.IsExecutionPendingForHost(ctx, host1Team1.ID, script.ID)
	require.NoError(t, err)
	require.True(t, hostPendingScript)

	// Request a manual script execution on the host for the same script, which should fail.
	var scriptRunResp fleet.RunScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host1Team1.ID, ScriptID: &script.ID}, http.StatusConflict, &scriptRunResp)

	// Submit same results as before, which should not trigger a script run because the policy is already failing.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: new(false),
			policy2Team1.ID: new(false),
			policy3Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	hostPendingScript, err = s.ds.IsExecutionPendingForHost(ctx, host1Team1.ID, script.ID)
	require.NoError(t, err)
	require.True(t, hostPendingScript)

	// Submit same results but policy1Team1 now passes,
	// and then submit again but policy1Team1 fails.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: new(true),
			policy2Team1.ID: new(false),
			policy3Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host1Team1,
		map[uint]*bool{
			policy1Team1.ID: new(false),
			policy2Team1.ID: new(false),
			policy3Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	hostPendingScript, err = s.ds.IsExecutionPendingForHost(ctx, host1Team1.ID, script.ID)
	require.NoError(t, err)
	require.True(t, hostPendingScript)

	// host2Team1 is failing policy2Team1 (incompatible) and policy3Team1 (no script) policies; no scripts should be queued
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host2Team1,
		map[uint]*bool{
			policy2Team1.ID: new(false),
			policy3Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	hostPendingScript, err = s.ds.IsExecutionPendingForHost(ctx, host2Team1.ID, script.ID)
	require.NoError(t, err)
	require.False(t, hostPendingScript)

	// Associate psScript to policy4Team2.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team2.ID, policy4Team2.ID), fleet.ModifyTeamPolicyRequest{
		ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: psScript.ID},
	}, http.StatusOK, &mtplr)

	// host3Team2 reports a failing result for policy4Team2, which should trigger a script run.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			policy4Team2.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	host3PendingScripts, err := s.ds.ListPendingHostScriptExecutions(ctx, host3Team2.ID, false)
	require.NoError(t, err)
	require.Len(t, host3PendingScripts, 1)
	host3executionID := host3PendingScripts[0].ExecutionID

	// Dissociate policy4Team2 from script.
	mtplr = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team2.ID, policy4Team2.ID), fleet.ModifyTeamPolicyRequest{
		ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: 0},
	}, http.StatusOK, &mtplr)

	// host3Team2 reports a failing result for policy4Team2.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3Team2,
		map[uint]*bool{
			policy4Team2.ID: new(false),
		},
	), http.StatusOK, &distributedResp)

	// hostVanillaOsquery5Team1 sends policy results with failed policies with associated scripts.
	// Fleet should not queue scripts for vanilla osquery hosts.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		hostVanillaOsquery5Team1,
		map[uint]*bool{
			policy1Team1.ID: new(false),
		},
	), http.StatusOK, &distributedResp)
	hostPendingScripts, err := s.ds.ListPendingHostScriptExecutions(ctx, hostVanillaOsquery5Team1.ID, false)
	require.NoError(t, err)
	require.Empty(t, hostPendingScripts)

	// activity feed should show script run as pending, with "Fleet" as author, policy ID and name set in body
	var listResp listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host3Team2.ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 1)
	require.Nil(t, listResp.Activities[0].ActorEmail)
	require.Equal(t, "Fleet", *listResp.Activities[0].ActorFullName)
	require.Nil(t, listResp.Activities[0].ActorGravatar)
	require.Equal(t, "ran_script", listResp.Activities[0].Type)
	var activityJson map[string]any
	err = json.Unmarshal(*listResp.Activities[0].Details, &activityJson)
	require.NoError(t, err)
	require.InDelta(t, float64(policy4Team2.ID), activityJson["policy_id"], 0.001)
	require.Equal(t, "policy4Team2", activityJson["policy_name"])

	// post script result response
	var orbitPostScriptResp fleet.OrbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host3Team2.OrbitNodeKey, host3executionID)),
		http.StatusOK, &orbitPostScriptResp)

	// activity feed should show script run as completed
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host3Team2.ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 1)
	require.Empty(t, *listResp.Activities[0].ActorEmail) // actor email is blank rather than nil here 👀
	require.Equal(t, "Fleet", *listResp.Activities[0].ActorFullName)
	require.Nil(t, listResp.Activities[0].ActorGravatar)
	require.Equal(t, "ran_script", listResp.Activities[0].Type)
	err = json.Unmarshal(*listResp.Activities[0].Details, &activityJson)
	require.NoError(t, err)
	require.InDelta(t, float64(policy4Team2.ID), activityJson["policy_id"], 0.001)
	require.Equal(t, "policy4Team2", activityJson["policy_name"])
}

func (s *integrationEnterpriseTestSuite) TestPolicyAutomationScriptRetries() {
	t := s.T()
	ctx := context.Background()

	// Create a team
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	// Create a host
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", t.Name()),
		Platform:        "darwin",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// Create a script
	script, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "failing-script.sh",
		ScriptContents: "exit 1",
		TeamID:         &team.ID,
	})
	require.NoError(t, err)

	// Create a team policy
	policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:     t.Name(),
		Query:    "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Platform: "darwin",
	})
	require.NoError(t, err)

	// Associate script with policy
	var modifyResp fleet.ModifyTeamPolicyResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy.ID), fleet.ModifyTeamPolicyRequest{
		ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: script.ID},
	}, http.StatusOK, &modifyResp)
	require.NotNil(t, modifyResp.Policy)
	policy, err = s.ds.Policy(ctx, policy.ID)
	require.NoError(t, err)
	require.NotNil(t, policy.ScriptID)
	require.Equal(t, script.ID, *policy.ScriptID)

	submitPolicyResult := func(policyID uint, passes bool) {
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
			host,
			map[uint]*bool{
				policyID: new(passes),
			},
		), http.StatusOK, &distributedResp)
	}
	getPendingScript := func() *fleet.HostScriptResult {
		scripts, err := s.ds.ListPendingHostScriptExecutions(ctx, host.ID, false)
		require.NoError(t, err)
		if len(scripts) == 0 {
			return nil
		}
		return scripts[0]
	}
	submitScriptResult := func(executionID string, exitCode int) {
		var orbitPostScriptResp fleet.OrbitPostScriptResultResponse
		s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
			json.RawMessage(
				fmt.Sprintf(`
					{"orbit_node_key": %q, "execution_id": %q, "exit_code": %d, "output": "script output"}
				`, *host.OrbitNodeKey, executionID, exitCode),
			),
			http.StatusOK, &orbitPostScriptResp)
	}
	getScriptResults := func() []*fleet.HostScriptResult {
		var results []*fleet.HostScriptResult
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &results, `
				SELECT id, host_id, script_id, policy_id, exit_code, attempt_number, execution_id
				FROM host_script_results
				WHERE host_id = ? AND script_id = ? AND policy_id = ?
				ORDER BY id ASC
			`, host.ID, script.ID, policy.ID)
		})
		return results
	}
	countActivities := func() int {
		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &count, `
				SELECT COUNT(*)
				FROM activity_past
				WHERE JSON_EXTRACT(details, '$.host_id') = ? AND JSON_EXTRACT(details, '$.policy_id') = ?
			`, host.ID, policy.ID)
		})
		return count
	}

	// Policy fails, triggers first script execution
	submitPolicyResult(policy.ID, false)

	// Wait for script to be queued
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingScript := getPendingScript()
		assert.NotNil(t, pendingScript, "script should be queued")
		if pendingScript != nil && pendingScript.ScriptID != nil {
			assert.Equal(t, script.ID, *pendingScript.ScriptID)
		}
		if pendingScript != nil && pendingScript.PolicyID != nil {
			assert.Equal(t, policy.ID, *pendingScript.PolicyID)
		}
	}, 5*time.Second, 100*time.Millisecond)

	// Fail the attempt
	pendingScript := getPendingScript()
	require.NotNil(t, pendingScript)
	executionID1 := pendingScript.ExecutionID
	submitScriptResult(executionID1, 1)

	// Verify attempt 1 and retry
	results := getScriptResults()
	require.Len(t, results, 2)
	// First result: attempt 1
	require.NotNil(t, results[0].AttemptNumber)
	require.Equal(t, 1, *results[0].AttemptNumber)
	require.NotNil(t, results[0].ExitCode)
	require.Equal(t, int64(1), *results[0].ExitCode)
	// queued retry
	// Note: attempt_number is only calculated when the result is submitted, not when queued, so it will be NULL
	require.Nil(t, results[1].ExitCode)
	require.Nil(t, results[1].AttemptNumber)
	// Activity is created for every attempt (including retries), not only the final one.
	require.Equal(t, 1, countActivities())

	// Get pending retry
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingScript := getPendingScript()
		assert.NotNil(t, pendingScript, "retry should be queued")
		if pendingScript != nil {
			assert.NotEqual(t, executionID1, pendingScript.ExecutionID, "should be a new execution")
		}
	}, 5*time.Second, 100*time.Millisecond)

	// Fail attempt 2
	pendingScript = getPendingScript()
	require.NotNil(t, pendingScript)
	executionID2 := pendingScript.ExecutionID
	submitScriptResult(executionID2, 1)

	// Verify attempt 2 and retry
	results = getScriptResults()
	require.Len(t, results, 3)
	require.NotNil(t, results[1].AttemptNumber)
	require.Equal(t, 2, *results[1].AttemptNumber)
	require.Nil(t, results[2].AttemptNumber)
	require.Equal(t, 2, countActivities())

	// Get second pending retry
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingScript := getPendingScript()
		assert.NotNil(t, pendingScript, "second retry should be queued")
	}, 5*time.Second, 100*time.Millisecond)

	// Fail attempt 3
	pendingScript = getPendingScript()
	require.NotNil(t, pendingScript)
	executionID3 := pendingScript.ExecutionID
	submitScriptResult(executionID3, 1)

	// Verify attempt = 3
	results = getScriptResults()
	require.Len(t, results, 3, "should have exactly 3 completed attempts, no more retries queued")
	require.NotNil(t, results[2].AttemptNumber)
	require.Equal(t, 3, *results[2].AttemptNumber)
	require.NotNil(t, results[2].ExitCode, "final attempt should be completed")

	// One activity per attempt — three failed attempts produce three activities.
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		assert.Equal(t, 3, countActivities(), "activity should be created for each retry attempt")
	}, 5*time.Second, 100*time.Millisecond)

	// no more retries, max reached
	time.Sleep(2 * time.Second)
	pendingScript = getPendingScript()
	require.Nil(t, pendingScript)

	// exactly 3 attempts
	results = getScriptResults()
	require.Len(t, results, 3)
	require.Equal(t, 1, *results[0].AttemptNumber)
	require.Equal(t, 2, *results[1].AttemptNumber)
	require.Equal(t, 3, *results[2].AttemptNumber)

	// pass policy
	submitPolicyResult(policy.ID, true)
	// fail policy fails to trigger first attempt
	submitPolicyResult(policy.ID, false)

	// Wait for first attempt
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingScript := getPendingScript()
		assert.NotNil(t, pendingScript)
	}, 5*time.Second, 100*time.Millisecond)

	// fail result
	pendingScript = getPendingScript()
	submitScriptResult(pendingScript.ExecutionID, 1)

	// Verify retry is queued
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingScript := getPendingScript()
		assert.NotNil(t, pendingScript)
	}, 5*time.Second, 100*time.Millisecond)

	// Make the policy pass
	submitPolicyResult(policy.ID, true)

	// Submit the pending retry result
	pendingScript = getPendingScript()
	submitScriptResult(pendingScript.ExecutionID, 1)

	// Ensure no more retries
	time.Sleep(2 * time.Second)
	pendingScript = getPendingScript()
	require.Nil(t, pendingScript)
	// Verify that all attempts were reset to 0 (policy passed, marking entire sequence as obsolete)
	results = getScriptResults()
	require.GreaterOrEqual(t, len(results), 4)
	require.NotNil(t, results[len(results)-1].AttemptNumber)
	require.Equal(t, 0, *results[len(results)-1].AttemptNumber)
	require.NotNil(t, results[len(results)-2].AttemptNumber)
	require.Equal(t, 0, *results[len(results)-2].AttemptNumber)
	scnt, err := s.ds.CountHostScriptAttempts(ctx, host.ID, script.ID, policy.ID)
	require.NoError(t, err)
	require.Equal(t, 0, scnt)

	// Verify that we can start a fresh retry sequence after the reset
	// Fail the policy again to trigger a new automation
	submitPolicyResult(policy.ID, false)

	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingScript := getPendingScript()
		assert.NotNil(t, pendingScript, "should have queued a new first attempt")
	}, 5*time.Second, 100*time.Millisecond)

	// first attempt
	pendingScript = getPendingScript()
	submitScriptResult(pendingScript.ExecutionID, 1)

	// Wait for retry to be queued
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		pendingScript := getPendingScript()
		assert.NotNil(t, pendingScript, "should have queued retry")
	}, 5*time.Second, 100*time.Millisecond)

	// Verify fresh sequence started at attempt_number = 1
	results = getScriptResults()
	require.GreaterOrEqual(t, len(results), 2)
	require.Nil(t, results[len(results)-1].AttemptNumber)
	completedAttempt := results[len(results)-2]
	require.NotNil(t, completedAttempt.AttemptNumber)
	require.Equal(t, 1, *completedAttempt.AttemptNumber)

	// Count should be 2 (one completed, one pending)
	scnt, err = s.ds.CountHostScriptAttempts(ctx, host.ID, script.ID, policy.ID)
	require.NoError(t, err)
	require.Equal(t, 2, scnt)

	// Verify database state
	var attemptCounts struct {
		TotalRows     int `db:"total_rows"`
		ZeroCount     int `db:"zero_count"`
		PositiveCount int `db:"positive_count"`
		NullCount     int `db:"null_count"`
	}
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &attemptCounts, `
			SELECT
				COUNT(*) as total_rows,
				SUM(CASE WHEN attempt_number = 0 THEN 1 ELSE 0 END) as zero_count,
				SUM(CASE WHEN attempt_number > 0 THEN 1 ELSE 0 END) as positive_count,
				SUM(CASE WHEN attempt_number IS NULL THEN 1 ELSE 0 END) as null_count
			FROM host_script_results
			WHERE host_id = ? AND script_id = ? AND policy_id = ?
		`, host.ID, script.ID, policy.ID)
	})
	require.GreaterOrEqual(t, attemptCounts.TotalRows, 6)
	require.GreaterOrEqual(t, attemptCounts.ZeroCount, 5)
	require.Equal(t, 1, attemptCounts.PositiveCount)
	require.Equal(t, 1, attemptCounts.NullCount)
}

// TestTeamPolicyContinuousAutomationsCRUD exercises the create, modify, get, and
// GitOps spec-apply paths for the `continuous_automations_enabled` field on team
// policies, ensuring the value round-trips through every entry point.
func (s *integrationEnterpriseTestSuite) TestTeamPolicyContinuousAutomationsCRUD() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	// POST: default value is false.
	createDefault := &fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team.ID), &fleet.TeamPolicyRequest{
		Name:     "default",
		Query:    "SELECT 1;",
		Platform: "darwin",
	}, http.StatusOK, &createDefault)
	require.NotNil(t, createDefault.Policy)
	require.False(t, createDefault.Policy.ContinuousAutomationsEnabled)

	// POST: explicitly set to true at creation.
	createOn := &fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team.ID), &fleet.TeamPolicyRequest{
		Name:                         "on",
		Query:                        "SELECT 2;",
		Platform:                     "darwin",
		ContinuousAutomationsEnabled: true,
	}, http.StatusOK, &createOn)
	require.NotNil(t, createOn.Policy)
	require.True(t, createOn.Policy.ContinuousAutomationsEnabled)

	// GET (single + list) returns the field as persisted.
	getResp := &fleet.GetTeamPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, createOn.Policy.ID), nil, http.StatusOK, getResp)
	require.NotNil(t, getResp.Policy)
	require.True(t, getResp.Policy.ContinuousAutomationsEnabled)

	listResp := &fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team.ID), nil, http.StatusOK, listResp)
	require.Len(t, listResp.Policies, 2)
	for _, p := range listResp.Policies {
		switch p.Name {
		case "default":
			assert.False(t, p.ContinuousAutomationsEnabled)
		case "on":
			assert.True(t, p.ContinuousAutomationsEnabled)
		default:
			t.Fatalf("unexpected policy %q", p.Name)
		}
	}

	// PATCH: flip the default policy on, flip the explicit one off.
	patchOn := &fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, createDefault.Policy.ID),
		&fleet.ModifyTeamPolicyRequest{
			ContinuousAutomationsEnabled: new(true),
		}, http.StatusOK, patchOn)
	require.NotNil(t, patchOn.Policy)
	require.True(t, patchOn.Policy.ContinuousAutomationsEnabled)

	patchOff := &fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, createOn.Policy.ID),
		&fleet.ModifyTeamPolicyRequest{
			ContinuousAutomationsEnabled: new(false),
		}, http.StatusOK, patchOff)
	require.NotNil(t, patchOff.Policy)
	require.False(t, patchOff.Policy.ContinuousAutomationsEnabled)

	// PATCH without sending the field leaves it untouched (no implicit reset).
	patchUnrelated := &fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, createDefault.Policy.ID),
		json.RawMessage(`{"description": "unrelated edit"}`), http.StatusOK, patchUnrelated)
	require.NotNil(t, patchUnrelated.Policy)
	require.True(t, patchUnrelated.Policy.ContinuousAutomationsEnabled)

	// GitOps spec apply: create a new policy via /spec/policies and verify the
	// field is stored, then flip it on a subsequent apply.
	s.Do("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{
			{
				Name:                         "gitops-policy",
				Query:                        "SELECT 3;",
				Platform:                     "darwin",
				Team:                         team.Name,
				ContinuousAutomationsEnabled: true,
			},
		},
	}, http.StatusOK)

	listResp = &fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team.ID), nil, http.StatusOK, listResp)
	var gitopsPolicy *fleet.Policy
	for _, p := range listResp.Policies {
		if p.Name == "gitops-policy" {
			gitopsPolicy = p
			break
		}
	}
	require.NotNil(t, gitopsPolicy, "gitops-policy not found")
	require.True(t, gitopsPolicy.ContinuousAutomationsEnabled)

	s.Do("POST", "/api/latest/fleet/spec/policies", fleet.ApplyPolicySpecsRequest{
		Specs: []*fleet.PolicySpec{
			{
				Name:                         "gitops-policy",
				Query:                        "SELECT 3;",
				Platform:                     "darwin",
				Team:                         team.Name,
				ContinuousAutomationsEnabled: false,
			},
		},
	}, http.StatusOK)

	updated, err := s.ds.Policy(ctx, gitopsPolicy.ID)
	require.NoError(t, err)
	require.False(t, updated.ContinuousAutomationsEnabled)
}

// TestPolicyAutomationsContinuousScripts verifies that when a policy has
// continuous_automations_enabled=true, the attached script automation runs on
// every failing policy result — not just on the first failure or a pass→fail
// transition (the default behavior).
func (s *integrationEnterpriseTestSuite) TestPolicyAutomationsContinuousScripts() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", t.Name()),
		Platform:        "darwin",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// Use a dedicated script per policy.
	continuousScript, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "continuous.sh",
		ScriptContents: "echo continuous",
		TeamID:         &team.ID,
	})
	require.NoError(t, err)
	transitionScript, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "transition.sh",
		ScriptContents: "echo transition",
		TeamID:         &team.ID,
	})
	require.NoError(t, err)

	// Two policies on the same team: one with continuous automations, one
	// without. Each has its own script attached.
	continuousPolicy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:                         "continuous",
		Query:                        "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Platform:                     "darwin",
		ContinuousAutomationsEnabled: true,
	})
	require.NoError(t, err)
	transitionPolicy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:     "transition-only",
		Query:    "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Platform: "darwin",
	})
	require.NoError(t, err)

	// Attach the scripts via the API so the modify path is exercised.
	attach := func(policyID, scriptID uint) {
		var resp fleet.ModifyTeamPolicyResponse
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policyID), fleet.ModifyTeamPolicyRequest{
			ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: scriptID},
		}, http.StatusOK, &resp)
	}
	attach(continuousPolicy.ID, continuousScript.ID)
	attach(transitionPolicy.ID, transitionScript.ID)

	// Distributed writes carry a result for every policy in scope for the host,
	// mirroring how osquery reports: it runs all distributed queries from a read
	// and reports them together in one write. (Submitting a subset would make
	// the missing policies look out-of-scope and get their membership cleaned up.)
	submitPolicyResults := func(results map[uint]*bool) {
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
			host,
			results,
		), http.StatusOK, &distributedResp)
	}
	bothFail := map[uint]*bool{continuousPolicy.ID: new(false), transitionPolicy.ID: new(false)}
	bothPass := map[uint]*bool{continuousPolicy.ID: new(true), transitionPolicy.ID: new(true)}
	completePendingScripts := func() int {
		pending, err := s.ds.ListPendingHostScriptExecutions(ctx, host.ID, false)
		require.NoError(t, err)
		for _, hs := range pending {
			var orbitPostScriptResp fleet.OrbitPostScriptResultResponse
			s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
				json.RawMessage(fmt.Sprintf(
					`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`,
					*host.OrbitNodeKey, hs.ExecutionID,
				)), http.StatusOK, &orbitPostScriptResp)
		}
		return len(pending)
	}
	// countExecutionsFor returns the number of script executions queued for a
	// given (policy, script) pair on this host. Only one script activity is
	// activated at a time per host, and a queued script only appears in
	// host_script_results once activated — a script queued behind a pending one
	// shows up only after the pending one completes.
	countExecutionsFor := func(policyID, scriptID uint) int {
		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &count, `
				SELECT COUNT(*) FROM host_script_results
				WHERE host_id = ? AND script_id = ? AND policy_id = ?
			`, host.ID, scriptID, policyID)
		})
		return count
	}
	continuousCount := func() int { return countExecutionsFor(continuousPolicy.ID, continuousScript.ID) }
	transitionCount := func() int { return countExecutionsFor(transitionPolicy.ID, transitionScript.ID) }

	// assertCountStable polls the count for a window to catch delayed enqueues
	// that a single point-in-time check would miss.
	assertCountStable := func(expected int, countFn func() int, msgAndArgs ...any) {
		t.Helper()
		require.Never(t, func() bool {
			return countFn() != expected
		}, 2*time.Second, 100*time.Millisecond, msgAndArgs...)
	}

	// waitForCountsAndDrain waits for the expected per-policy execution counts,
	// completing pending scripts between polls (queued scripts only become
	// visible as the ones ahead of them complete), then drains any remainder so
	// the next step starts with no pending script executions.
	waitForCountsAndDrain := func(wantContinuous, wantTransition int, msgAndArgs ...any) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for {
			completePendingScripts()
			if continuousCount() == wantContinuous && transitionCount() == wantTransition {
				break
			}
			if time.Now().After(deadline) {
				require.Equal(t, wantContinuous, continuousCount(), msgAndArgs...)
				require.Equal(t, wantTransition, transitionCount(), msgAndArgs...)
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		for range 10 {
			if completePendingScripts() == 0 {
				return
			}
		}
		require.FailNow(t, "pending scripts did not drain")
	}

	// First failing results: pass→fail transition queues the script on both.
	submitPolicyResults(bothFail)
	waitForCountsAndDrain(1, 1, "first script run for both policies")

	// Second failing results (fail→fail): only the continuous policy re-queues.
	submitPolicyResults(bothFail)
	waitForCountsAndDrain(2, 1, "continuous policy fires on every failing result")
	assertCountStable(1, transitionCount, "default policy must not re-trigger on fail→fail")

	// Third failing results: continuous still re-triggers, default still does not.
	submitPolicyResults(bothFail)
	waitForCountsAndDrain(3, 1, "continuous policy fires on every failing result")
	assertCountStable(1, transitionCount)

	// Final: both policies pass. Neither should queue a script — passing
	// results never trigger automations, regardless of continuous mode.
	continuousBefore := continuousCount()
	transitionBefore := transitionCount()
	submitPolicyResults(bothPass)
	assertCountStable(continuousBefore, continuousCount, "continuous policy must not trigger script on passing result")
	assertCountStable(transitionBefore, transitionCount, "transition policy must not trigger script on passing result")
}

// TestPolicyAutomationsContinuousScriptRetryReset verifies that when a
// continuous-automation policy is still failing after its retry sequence has
// been exhausted (MaxPolicyAutomationRetries attempts), the next continuous
// re-fire resets the attempt counter so a fresh sequence of retries is
// available. Without the reset, attempt_number would just keep climbing past
// the cap and no retries would ever fire again.
func (s *integrationEnterpriseTestSuite) TestPolicyAutomationsContinuousScriptRetryReset() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", t.Name()),
		Platform:        "darwin",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	script, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "fail.sh",
		ScriptContents: "exit 1",
		TeamID:         &team.ID,
	})
	require.NoError(t, err)

	policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:                         "continuous-retry-reset",
		Query:                        "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Platform:                     "darwin",
		ContinuousAutomationsEnabled: true,
	})
	require.NoError(t, err)

	var mtplr fleet.ModifyTeamPolicyResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy.ID), fleet.ModifyTeamPolicyRequest{
		ScriptID: optjson.Any[uint]{Set: true, Valid: true, Value: script.ID},
	}, http.StatusOK, &mtplr)

	submitPolicyResult := func(passes bool) {
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
			host,
			map[uint]*bool{policy.ID: new(passes)},
		), http.StatusOK, &distributedResp)
	}

	getPendingScript := func() *fleet.HostScriptResult {
		t.Helper()
		var pending *fleet.HostScriptResult
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			scripts, err := s.ds.ListPendingHostScriptExecutions(ctx, host.ID, false)
			assert.NoError(t, err)
			assert.NotEmpty(t, scripts)
			if len(scripts) > 0 {
				pending = scripts[0]
			}
		}, 5*time.Second, 100*time.Millisecond)
		return pending
	}

	submitScriptFailure := func(executionID string) {
		s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
			json.RawMessage(fmt.Sprintf(
				`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 1, "output": "fail"}`,
				*host.OrbitNodeKey, executionID,
			)), http.StatusOK, &fleet.OrbitPostScriptResultResponse{})
	}

	type row struct {
		ExecutionID   string `db:"execution_id"`
		ExitCode      *int64 `db:"exit_code"`
		AttemptNumber *int   `db:"attempt_number"`
	}
	listAttempts := func() []row {
		var rows []row
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &rows, `
				SELECT execution_id, exit_code, attempt_number
				FROM host_script_results
				WHERE host_id = ? AND script_id = ? AND policy_id = ?
				ORDER BY id ASC
			`, host.ID, script.ID, policy.ID)
		})
		return rows
	}

	// Phase 1: pass→fail starts a sequence. Drive it to the cap by failing
	// every attempt. fleet.MaxPolicyAutomationRetries is 3, so the host
	// should see exactly 3 attempts: the initial run plus 2 auto-retries.
	submitPolicyResult(false)
	for i := 1; i <= fleet.MaxPolicyAutomationRetries; i++ {
		pending := getPendingScript()
		require.NotNil(t, pending)
		submitScriptFailure(pending.ExecutionID)
	}
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		rows := listAttempts()
		if !assert.Len(t, rows, fleet.MaxPolicyAutomationRetries) {
			return
		}
		// All rows should be completed (exit_code recorded) and form a
		// 1..MaxPolicyAutomationRetries attempt sequence.
		for i, r := range rows {
			if assert.NotNil(t, r.AttemptNumber) {
				assert.Equal(t, i+1, *r.AttemptNumber, "row %d attempt_number", i)
			}
			assert.NotNil(t, r.ExitCode, "row %d should have a recorded exit_code", i)
		}
	}, 5*time.Second, 100*time.Millisecond)

	// No further retry is queued once the cap is hit — without continuous
	// re-fire, the host_script_results count stays at MaxPolicyAutomationRetries.
	require.Never(t, func() bool {
		return len(listAttempts()) != fleet.MaxPolicyAutomationRetries
	}, 1*time.Second, 100*time.Millisecond, "no extra attempts queued after retry cap is hit")

	// Phase 2: the policy is still failing. A continuous re-fire must reset
	// the attempt counter — the new attempt should start at 1 and remain
	// retry-eligible, NOT inherit the previous sequence's cap.
	submitPolicyResult(false)
	pending := getPendingScript()
	require.NotNil(t, pending)
	submitScriptFailure(pending.ExecutionID)

	require.EventuallyWithT(t, func(t *assert.CollectT) {
		rows := listAttempts()
		// 3 old (reset to 0) + 1 new completed (attempt 1) + 1 new queued retry (NULL)
		if !assert.Len(t, rows, fleet.MaxPolicyAutomationRetries+2) {
			return
		}
		// Old sequence is marked attempt_number=0.
		for i := range fleet.MaxPolicyAutomationRetries {
			if assert.NotNil(t, rows[i].AttemptNumber) {
				assert.Equal(t, 0, *rows[i].AttemptNumber, "old row %d should be reset", i)
			}
		}
		// New sequence starts at 1, and a retry has been queued (attempt_number NULL until completed).
		if assert.NotNil(t, rows[fleet.MaxPolicyAutomationRetries].AttemptNumber) {
			assert.Equal(t, 1, *rows[fleet.MaxPolicyAutomationRetries].AttemptNumber, "new sequence first attempt")
		}
		assert.Nil(t, rows[fleet.MaxPolicyAutomationRetries+1].AttemptNumber, "retry should be queued (NULL) after fresh sequence")
	}, 5*time.Second, 100*time.Millisecond)
}

// TestPolicyAutomationsContinuousSoftwareInstaller verifies that
// continuous_automations_enabled=true causes a software install automation to
// fire on every failing policy result, not just on pass→fail transitions.
func (s *integrationEnterpriseTestSuite) TestPolicyAutomationsContinuousSoftwareInstaller() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", t.Name()),
		Platform:        "darwin",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// Upload a macOS installer to the team.
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "echo install",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}, http.StatusOK, "")

	titlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &titlesResp,
		"query", "DummyApp", "team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, titlesResp.SoftwareTitles, 1)
	titleID := titlesResp.SoftwareTitles[0].ID

	var installerID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerID,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`,
			team.ID, "dummy_installer.pkg",
		)
	})
	require.NotZero(t, installerID)

	continuousPolicy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:                         "continuous",
		Query:                        "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Platform:                     "darwin",
		ContinuousAutomationsEnabled: true,
	})
	require.NoError(t, err)
	transitionPolicy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:     "transition-only",
		Query:    "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Platform: "darwin",
	})
	require.NoError(t, err)

	for _, p := range []*fleet.Policy{continuousPolicy, transitionPolicy} {
		var resp fleet.ModifyTeamPolicyResponse
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, p.ID), fleet.ModifyTeamPolicyRequest{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: titleID},
		}, http.StatusOK, &resp)
	}

	// Distributed writes carry a result for every policy in scope for the host,
	// mirroring how osquery reports: it runs all distributed queries from a read
	// and reports them together in one write. (Submitting a subset would make
	// the missing policies look out-of-scope and get their membership cleaned up.)
	submitPolicyResults := func(results map[uint]*bool) {
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
			host,
			results,
		), http.StatusOK, &distributedResp)
	}
	bothFail := map[uint]*bool{continuousPolicy.ID: new(false), transitionPolicy.ID: new(false)}
	completePendingInstall := func() {
		last, err := s.ds.GetHostLastInstallData(ctx, host.ID, installerID)
		require.NoError(t, err)
		if last == nil || last.Status == nil || *last.Status != fleet.SoftwareInstallPending {
			return
		}
		s.Do("POST", "/api/fleet/orbit/software_install/result",
			json.RawMessage(fmt.Sprintf(`{
				"orbit_node_key": %q,
				"install_uuid": %q,
				"install_script_exit_code": 0,
				"install_script_output": "ok"
			}`, *host.OrbitNodeKey, last.ExecutionID)),
			http.StatusNoContent)
	}
	countInstallsFor := func(policyID uint) int {
		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &count, `
				SELECT COUNT(*) FROM host_software_installs
				WHERE host_id = ? AND software_installer_id = ? AND policy_id = ?
			`, host.ID, installerID, policyID)
		})
		return count
	}

	// First fail for the continuous policy (transition still passing): the
	// pass→fail transition queues an install. Both policies share the installer,
	// so their first failures are staggered across writes — a pending install
	// blocks queueing another one for the same installer.
	submitPolicyResults(map[uint]*bool{continuousPolicy.ID: new(false), transitionPolicy.ID: new(true)})
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		assert.Equal(t, 1, countInstallsFor(continuousPolicy.ID))
	}, 5*time.Second, 100*time.Millisecond)
	completePendingInstall()

	// Transition policy now fails for the first time (pass→fail): queues its
	// install. The continuous policy keeps failing (fail→fail) within the policy
	// update interval: it is throttled. A successful install requests a host
	// refetch that re-runs policies immediately, so without throttling this
	// would be a tight install→refetch→re-run loop. It must NOT re-queue until
	// the interval elapses.
	submitPolicyResults(bothFail)
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		assert.Equal(t, 1, countInstallsFor(transitionPolicy.ID))
	}, 5*time.Second, 100*time.Millisecond)
	require.Never(t, func() bool {
		return countInstallsFor(continuousPolicy.ID) != 1
	}, 2*time.Second, 100*time.Millisecond, "continuous policy must be throttled within the policy update interval")
	completePendingInstall()

	// Age the last successful install past the policy update interval; the next failing
	// result must re-queue (the retry happens on the next interval). The cooldown is
	// keyed per host+installer (GetHostLastInstallData), and both policies share this
	// installer, so age every install for it rather than only the continuous policy's.
	ageInstaller := func() {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `
				UPDATE host_software_installs SET updated_at = NOW() - INTERVAL 2 HOUR
				WHERE host_id = ? AND software_installer_id = ?
			`, host.ID, installerID)
			return err
		})
	}
	ageInstaller()
	submitPolicyResults(bothFail)
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		assert.Equal(t, 2, countInstallsFor(continuousPolicy.ID), "continuous policy must re-fire after the cooldown elapses")
	}, 5*time.Second, 100*time.Millisecond)
	completePendingInstall()

	// The transition policy kept failing across the writes above (fail→fail):
	// it must never have re-queued. Poll for a window so a delayed enqueue
	// doesn't slip past a one-shot check.
	require.Never(t, func() bool {
		return countInstallsFor(transitionPolicy.ID) != 1
	}, 2*time.Second, 100*time.Millisecond, "default policy must not re-trigger on fail→fail")

	// Final: both policies pass. Neither should queue an install — passing
	// results never trigger automations, regardless of continuous mode.
	continuousBefore := countInstallsFor(continuousPolicy.ID)
	transitionBefore := countInstallsFor(transitionPolicy.ID)
	submitPolicyResults(map[uint]*bool{continuousPolicy.ID: new(true), transitionPolicy.ID: new(true)})
	require.Never(t, func() bool {
		return countInstallsFor(continuousPolicy.ID) != continuousBefore
	}, 2*time.Second, 100*time.Millisecond, "continuous policy must not trigger install on passing result")
	require.Never(t, func() bool {
		return countInstallsFor(transitionPolicy.ID) != transitionBefore
	}, 2*time.Second, 100*time.Millisecond, "transition policy must not trigger install on passing result")
}

// TestPolicyAutomationsContinuousSoftwareInstallerRetryReset is the software
// install analog of TestPolicyAutomationsContinuousScriptRetryReset: it
// verifies that when a continuous-automation policy is still failing after its
// software-install retry sequence has been exhausted, the next continuous
// re-fire resets the attempt counter so a fresh retry sequence is available.
func (s *integrationEnterpriseTestSuite) TestPolicyAutomationsContinuousSoftwareInstallerRetryReset() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", t.Name()),
		Platform:        "darwin",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// Upload a macOS installer to the team.
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "exit 1",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}, http.StatusOK, "")

	titlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &titlesResp,
		"query", "DummyApp", "team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, titlesResp.SoftwareTitles, 1)
	titleID := titlesResp.SoftwareTitles[0].ID

	var installerID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerID,
			`SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`,
			team.ID, "dummy_installer.pkg",
		)
	})
	require.NotZero(t, installerID)

	policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:                         "continuous-retry-reset-installer",
		Query:                        "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Platform:                     "darwin",
		ContinuousAutomationsEnabled: true,
	})
	require.NoError(t, err)

	var mtplr fleet.ModifyTeamPolicyResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: titleID},
	}, http.StatusOK, &mtplr)

	submitPolicyResult := func(passes bool) {
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
			host,
			map[uint]*bool{policy.ID: new(passes)},
		), http.StatusOK, &distributedResp)
	}

	getPendingInstall := func() *fleet.HostLastInstallData {
		t.Helper()
		var pending *fleet.HostLastInstallData
		require.EventuallyWithT(t, func(t *assert.CollectT) {
			last, err := s.ds.GetHostLastInstallData(ctx, host.ID, installerID)
			assert.NoError(t, err)
			if !assert.NotNil(t, last) || !assert.NotNil(t, last.Status) {
				return
			}
			assert.Equal(t, fleet.SoftwareInstallPending, *last.Status)
			pending = last
		}, 5*time.Second, 100*time.Millisecond)
		return pending
	}

	submitInstallFailure := func(installUUID string) {
		s.Do("POST", "/api/fleet/orbit/software_install/result",
			json.RawMessage(fmt.Sprintf(`{
				"orbit_node_key": %q,
				"install_uuid": %q,
				"install_script_exit_code": 1,
				"install_script_output": "fail"
			}`, *host.OrbitNodeKey, installUUID)),
			http.StatusNoContent)
	}

	type row struct {
		ExecutionID   string `db:"execution_id"`
		ExitCode      *int64 `db:"install_script_exit_code"`
		AttemptNumber *int   `db:"attempt_number"`
	}
	listAttempts := func() []row {
		var rows []row
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &rows, `
				SELECT execution_id, install_script_exit_code, attempt_number
				FROM host_software_installs
				WHERE host_id = ? AND software_installer_id = ? AND policy_id = ?
				ORDER BY id ASC
			`, host.ID, installerID, policy.ID)
		})
		return rows
	}

	// Phase 1: pass→fail starts a sequence. Drive it to the cap by failing
	// every attempt. fleet.MaxPolicyAutomationRetries is 3, so the host
	// should see exactly 3 attempts: the initial run plus 2 auto-retries.
	submitPolicyResult(false)
	for i := 1; i <= fleet.MaxPolicyAutomationRetries; i++ {
		pending := getPendingInstall()
		require.NotNil(t, pending)
		submitInstallFailure(pending.ExecutionID)
	}
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		rows := listAttempts()
		if !assert.Len(t, rows, fleet.MaxPolicyAutomationRetries) {
			return
		}
		for i, r := range rows {
			if assert.NotNil(t, r.AttemptNumber) {
				assert.Equal(t, i+1, *r.AttemptNumber, "row %d attempt_number", i)
			}
			assert.NotNil(t, r.ExitCode, "row %d should have a recorded exit_code", i)
		}
	}, 5*time.Second, 100*time.Millisecond)

	// No further retry is queued once the cap is hit.
	require.Never(t, func() bool {
		return len(listAttempts()) != fleet.MaxPolicyAutomationRetries
	}, 1*time.Second, 100*time.Millisecond, "no extra install attempts after retry cap is hit")

	// Phase 2: continuous re-fire after the cap is reached must reset the
	// attempt counter so the new attempt starts at 1 and remains retry-eligible.
	submitPolicyResult(false)
	pending := getPendingInstall()
	require.NotNil(t, pending)
	submitInstallFailure(pending.ExecutionID)

	require.EventuallyWithT(t, func(t *assert.CollectT) {
		rows := listAttempts()
		// 3 old (reset to 0) + 1 new completed (attempt 1) + 1 new queued retry (NULL)
		if !assert.Len(t, rows, fleet.MaxPolicyAutomationRetries+2) {
			return
		}
		for i := range fleet.MaxPolicyAutomationRetries {
			if assert.NotNil(t, rows[i].AttemptNumber) {
				assert.Equal(t, 0, *rows[i].AttemptNumber, "old row %d should be reset", i)
			}
		}
		if assert.NotNil(t, rows[fleet.MaxPolicyAutomationRetries].AttemptNumber) {
			assert.Equal(t, 1, *rows[fleet.MaxPolicyAutomationRetries].AttemptNumber, "new sequence first attempt")
		}
		assert.Nil(t, rows[fleet.MaxPolicyAutomationRetries+1].AttemptNumber, "retry should be queued (NULL) after fresh sequence")
	}, 5*time.Second, 100*time.Millisecond)
}

func (s *integrationEnterpriseTestSuite) TestPolicyAutomationsFailedInstallsAreLimited() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", t.Name()),
		Platform:        "darwin",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// An installer that can never succeed, standing in for the per-user installer on a host with no logged-in user from the report.
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "exit 1",
		Filename:      "dummy_installer.pkg",
		TeamID:        &team.ID,
	}, http.StatusOK, "")

	titlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &titlesResp, "query", "DummyApp", "team_id", fmt.Sprintf("%d", team.ID))
	require.Len(t, titlesResp.SoftwareTitles, 1)
	titleID := titlesResp.SoftwareTitles[0].ID

	var installerID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerID, `SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, team.ID, "dummy_installer.pkg")
	})
	require.NotZero(t, installerID)

	require.NoError(t, s.attemptCounter.ResetAttempts(ctx, host.ID, installerID))

	policyIDs := make([]uint, 0, 2)
	for _, name := range []string{"continuous-a", "continuous-b"} {
		policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
			Name:                         name,
			Query:                        "SELECT 1 FROM osquery_info WHERE start_time < 0;",
			Platform:                     "darwin",
			ContinuousAutomationsEnabled: true,
		})
		require.NoError(t, err)

		var resp fleet.ModifyTeamPolicyResponse
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy.ID), fleet.ModifyTeamPolicyRequest{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: titleID},
		}, http.StatusOK, &resp)
		policyIDs = append(policyIDs, policy.ID)
	}

	// Distributed writes carry a result for every policy in scope for the host, the way osquery reports them together from one distributed read.
	submitPolicyResults := func(passes bool) {
		results := make(map[uint]*bool, len(policyIDs))
		for _, policyID := range policyIDs {
			results[policyID] = new(passes)
		}
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(host, results), http.StatusOK, &distributedResp)
	}

	// Empty when nothing was queued, which ends the drain loop below.
	pendingInstallUUID := func() string {
		last, err := s.ds.GetHostLastInstallData(ctx, host.ID, installerID)
		require.NoError(t, err)
		if last == nil || last.Status == nil || *last.Status != fleet.SoftwareInstallPending {
			return ""
		}
		return last.ExecutionID
	}

	// The install script always fails, so the exit code reported here is what decides the outcome.
	reportInstallResult := func(installUUID string, exitCode int) {
		s.Do("POST", "/api/fleet/orbit/software_install/result", fleet.OrbitPostSoftwareInstallResultRequest{
			OrbitNodeKey: *host.OrbitNodeKey,
			HostSoftwareInstallResultPayload: &fleet.HostSoftwareInstallResultPayload{
				HostID:                host.ID,
				InstallUUID:           installUUID,
				InstallScriptExitCode: new(exitCode),
				InstallScriptOutput:   new("done"),
			},
		}, http.StatusNoContent)
	}

	countInstalls := func() int {
		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &count, `SELECT COUNT(*) FROM host_software_installs WHERE host_id = ? AND software_installer_id = ?`, host.ID, installerID)
		})
		return count
	}

	// A failure counts.
	submitPolicyResults(false)
	installUUID := pendingInstallUUID()
	require.NotEmpty(t, installUUID)
	reportInstallResult(installUUID, 1)

	attempts, err := s.attemptCounter.CountAttempts(ctx, host.ID, installerID)
	require.NoError(t, err)
	require.Equal(t, 1, attempts)

	// A later success clears it, so a host that installs and regresses later starts over rather than staying at its old total.
	submitPolicyResults(false)
	installUUID = pendingInstallUUID()
	require.NotEmpty(t, installUUID)
	reportInstallResult(installUUID, 0)

	attempts, err = s.attemptCounter.CountAttempts(ctx, host.ID, installerID)
	require.NoError(t, err)
	require.Zero(t, attempts)

	// The success above puts continuous automations on cooldown for a policy update interval, which would
	// stop the run below from queueing anything. Dropping the rows lifts it and starts the count at zero.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM host_software_installs WHERE host_id = ? AND software_installer_id = ?`, host.ID, installerID)
		return err
	})
	installs := countInstalls()
	require.Zero(t, installs)

	// One report is one osquery check-in reporting both policies as failing. Report
	// once more than the limit allows, so the last report has to be turned away.
	for range fleet.MaxPolicyAutomationInstallAttempts + 1 {
		submitPolicyResults(false)

		// A report queues one install and each failure queues the next retry, so the
		// sequence is at most MaxPolicyAutomationRetries deep. Bounded so a sequence
		// that never ends fails the test instead of hanging it.
		drained := 0
		for range fleet.MaxPolicyAutomationRetries + 1 {
			retryUUID := pendingInstallUUID()
			if retryUUID == "" {
				break
			}
			reportInstallResult(retryUUID, 1)
			drained++
		}
		require.LessOrEqual(t, drained, fleet.MaxPolicyAutomationRetries)
	}

	installs = countInstalls()
	require.Equal(t, fleet.MaxPolicyAutomationInstallAttempts, installs)

	// The count is keyed on the host and installer, so both policies share it.
	attempts, err = s.attemptCounter.CountAttempts(ctx, host.ID, installerID)
	require.NoError(t, err)
	require.Equal(t, fleet.MaxPolicyAutomationInstallAttempts, attempts)

	// The distributed write queues the install before it returns, so a further failing
	// report either adds one immediately or not at all.
	submitPolicyResults(false)
	installs = countInstalls()
	require.Equal(t, fleet.MaxPolicyAutomationInstallAttempts, installs)

	// Editing the installer clears the count instead of waiting for it to expire.
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		InstallScript: new("exit 0"),
		Filename:      "dummy_installer.pkg",
		TitleID:       titleID,
		TeamID:        &team.ID,
	}, http.StatusOK, "")

	attempts, err = s.attemptCounter.CountAttempts(ctx, host.ID, installerID)
	require.NoError(t, err)
	require.Zero(t, attempts)

	// The count is back to zero, so a failing report queues an install again.
	installsBeforeReset := countInstalls()
	submitPolicyResults(false)
	installs = countInstalls()
	require.Greater(t, installs, installsBeforeReset)

	installUUID = pendingInstallUUID()
	require.NotEmpty(t, installUUID)
	reportInstallResult(installUUID, 1)

	attempts, err = s.attemptCounter.CountAttempts(ctx, host.ID, installerID)
	require.NoError(t, err)
	require.Equal(t, 1, attempts)
}

func (s *integrationEnterpriseTestSuite) TestPolicyAutomationsPreInstallFailuresDoNotCount() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name()),
		NodeKey:         new(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", t.Name()),
		Platform:        "darwin",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// The install script would succeed, so a failure can only come from the pre-install query.
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "echo ok",
		PreInstallQuery: "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Filename:        "dummy_installer.pkg",
		TeamID:          &team.ID,
	}, http.StatusOK, "")

	titlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &titlesResp, "query", "DummyApp", "team_id", fmt.Sprintf("%d", team.ID))
	require.Len(t, titlesResp.SoftwareTitles, 1)

	var installerID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installerID, `SELECT id FROM software_installers WHERE global_or_team_id = ? AND filename = ?`, team.ID, "dummy_installer.pkg")
	})
	require.NotZero(t, installerID)

	require.NoError(t, s.attemptCounter.ResetAttempts(ctx, host.ID, installerID))

	policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:                         "pre-install",
		Query:                        "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Platform:                     "darwin",
		ContinuousAutomationsEnabled: true,
	})
	require.NoError(t, err)

	var resp fleet.ModifyTeamPolicyResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: titlesResp.SoftwareTitles[0].ID},
	}, http.StatusOK, &resp)

	// Report more times than the limit allows, so a counted failure would have stopped the installs.
	for range fleet.MaxPolicyAutomationInstallAttempts + 2 {
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(host, map[uint]*bool{policy.ID: new(false)}), http.StatusOK, &distributedResp)

		// Report what fleetd reports when the pre-install query returns no rows: the
		// condition output only, with no install script exit code.
		// Bounded for the same reason as the limit test: the retry sequence is at most
		// MaxPolicyAutomationRetries deep, so a sequence that never ends fails here.
		drained := 0
		for range fleet.MaxPolicyAutomationRetries + 1 {
			last, err := s.ds.GetHostLastInstallData(ctx, host.ID, installerID)
			require.NoError(t, err)
			if last == nil || last.Status == nil || *last.Status != fleet.SoftwareInstallPending {
				break
			}
			s.Do("POST", "/api/fleet/orbit/software_install/result", fleet.OrbitPostSoftwareInstallResultRequest{
				OrbitNodeKey: *host.OrbitNodeKey,
				HostSoftwareInstallResultPayload: &fleet.HostSoftwareInstallResultPayload{
					HostID:                    host.ID,
					InstallUUID:               last.ExecutionID,
					PreInstallConditionOutput: new(""),
				},
			}, http.StatusNoContent)
			drained++
		}
		require.LessOrEqual(t, drained, fleet.MaxPolicyAutomationRetries)
	}

	// A failure before the package is fetched must not count against the limit.
	attempts, err := s.attemptCounter.CountAttempts(ctx, host.ID, installerID)
	require.NoError(t, err)
	require.Zero(t, attempts)

	var installs, ranInstallScript int
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		if err := sqlx.GetContext(ctx, q, &installs, `SELECT COUNT(*) FROM host_software_installs WHERE host_id = ? AND software_installer_id = ?`, host.ID, installerID); err != nil {
			return err
		}
		return sqlx.GetContext(ctx, q, &ranInstallScript, `SELECT COUNT(*) FROM host_software_installs WHERE host_id = ? AND software_installer_id = ? AND install_script_exit_code IS NOT NULL`, host.ID, installerID)
	})
	// The install script never runs, and installs keep being queued past the limit.
	require.Zero(t, ranInstallScript)
	require.Greater(t, installs, fleet.MaxPolicyAutomationInstallAttempts)
}

// TestPolicyAutomationSoftwarePackageSelection verifies a team policy defaults its install-software
// automation to the title's first-added package and honors an explicitly chosen software_package_id.
func (s *integrationEnterpriseTestSuite) TestPolicyAutomationSoftwarePackageSelection() {
	t := s.T()
	ctx := context.Background()

	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{Name: t.Name()}, http.StatusOK, &createTeamResp)
	teamID := createTeamResp.Team.ID
	user := s.users["admin1@example.com"]

	newPkg := func(storage, filename, version string) uint {
		tfr, err := fleet.NewTempFileReader(strings.NewReader("hello-"+storage), t.TempDir)
		require.NoError(t, err)
		id, _, err := s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
			InstallScript:    "install",
			UninstallScript:  "uninstall",
			InstallerFile:    tfr,
			StorageID:        storage,
			Filename:         filename,
			Title:            "PolicyMultiPkgApp",
			Version:          version,
			Source:           "deb_packages",
			Extension:        "deb",
			BundleIdentifier: "com.example.policymultipkg",
			UserID:           user.ID,
			TeamID:           &teamID,
			Platform:         "linux",
			ValidatedLabels:  &fleet.LabelIdentsWithScope{},
		})
		require.NoError(t, err)
		return id
	}
	packageA := newPkg("pol-storage-a", "pkgA.deb", "1.0")
	packageB := newPkg("pol-storage-b", "pkgB.deb", "2.0")
	require.Less(t, packageA, packageB)
	titleID := getSoftwareTitleID(t, s.ds, "PolicyMultiPkgApp", "deb_packages")

	// storedPackageID reads the DB column, which is still named software_installer_id.
	storedPackageID := func(policyID uint) uint {
		var ids []uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &ids, `SELECT software_installer_id FROM policies WHERE id = ?`, policyID)
		})
		require.Len(t, ids, 1)
		return ids[0]
	}

	// No package chosen → defaults to first-added.
	var defResp fleet.TeamPolicyResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", teamID), fleet.TeamPolicyRequest{
		Name:            "default first-added",
		Query:           "SELECT 1;",
		SoftwareTitleID: &titleID,
	}, http.StatusOK, &defResp)
	require.Equal(t, packageA, storedPackageID(defResp.Policy.ID))

	// Explicit package chosen → honored.
	var chosenResp fleet.TeamPolicyResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", teamID), fleet.TeamPolicyRequest{
		Name:                "explicit choice",
		Query:               "SELECT 2;",
		SoftwareTitleID:     &titleID,
		SoftwareInstallerID: &packageB,
	}, http.StatusOK, &chosenResp)
	require.Equal(t, packageB, storedPackageID(chosenResp.Policy.ID))

	// Modify the first policy to point at the chosen package.
	var modResp fleet.ModifyTeamPolicyResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", teamID, defResp.Policy.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID:     optjson.Any[uint]{Set: true, Valid: true, Value: titleID},
		SoftwareInstallerID: optjson.Any[uint]{Set: true, Valid: true, Value: packageB},
	}, http.StatusOK, &modResp)
	require.Equal(t, packageB, storedPackageID(defResp.Policy.ID))

	// A package_id that doesn't belong to the title is rejected.
	var badResp fleet.TeamPolicyResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", teamID), fleet.TeamPolicyRequest{
		Name:                "bad package",
		Query:               "SELECT 3;",
		SoftwareTitleID:     &titleID,
		SoftwareInstallerID: new(packageB + 100000),
	}, http.StatusBadRequest, &badResp)
}

func (s *integrationEnterpriseTestSuite) TestTeamPolicyResendConfigProfileCRUD() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)
	otherTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-other"})
	require.NoError(t, err)

	newAppleProf := func(name string, teamID *uint) *fleet.MDMAppleConfigProfile {
		prof, err := s.ds.NewMDMAppleConfigProfile(ctx, fleet.MDMAppleConfigProfile{
			Name:         name,
			Identifier:   "com.example." + name,
			Mobileconfig: []byte("<plist></plist>"),
			TeamID:       teamID,
		}, nil)
		require.NoError(t, err)
		return prof
	}
	appleProf := newAppleProf("resend-apple", &team.ID)
	otherAppleProf := newAppleProf("resend-other-team-apple", &otherTeam.ID)

	winProf, err := s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "resend-windows",
		SyncML: []byte("<Replace></Replace>"),
		TeamID: &team.ID,
	}, nil)
	require.NoError(t, err)

	policiesURL := fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team.ID)

	// resendProfileJSON pulls the raw resend_configuration_profile object out of a
	// policy response body, so the assertions cover the wire-format keys rather than
	// just the Go struct fields.
	resendProfileJSON := func(body map[string]any) map[string]any {
		policy, ok := body["policy"].(map[string]any)
		require.True(t, ok, "response has no policy object: %v", body)
		raw, ok := policy["resend_configuration_profile"]
		if !ok {
			return nil
		}
		obj, ok := raw.(map[string]any)
		require.True(t, ok, "resend_configuration_profile is not an object: %v", raw)
		return obj
	}

	requireProfileJSON := func(body map[string]any, wantUUID, wantName string) {
		obj := resendProfileJSON(body)
		require.NotNil(t, obj, "resend_configuration_profile missing from response")
		// Exactly these two keys, with these names, per the documented shape.
		assert.Equal(t, wantUUID, obj["profile_uuid"])
		assert.Equal(t, wantName, obj["name"])
		assert.Len(t, obj, 2, "unexpected keys in resend_configuration_profile: %v", obj)
	}

	// POST with an Apple profile: response carries the populated object.
	var createApple map[string]any
	s.DoJSON("POST", policiesURL, &fleet.TeamPolicyRequest{
		Name:        "apple resend",
		Query:       "SELECT 1;",
		Platform:    "darwin",
		ProfileUUID: new(appleProf.ProfileUUID),
	}, http.StatusOK, &createApple)
	requireProfileJSON(createApple, appleProf.ProfileUUID, appleProf.Name)
	applePolicyID := uint(createApple["policy"].(map[string]any)["id"].(float64))

	// The datastore column matches the prefix, and the other column stays NULL.
	stored, err := s.ds.Policy(ctx, applePolicyID)
	require.NoError(t, err)
	require.Equal(t, new(appleProf.ProfileUUID), stored.ResendAppleProfileUUID)
	require.Nil(t, stored.ResendWindowsProfileUUID)

	// POST with a Windows profile.
	var createWindows map[string]any
	s.DoJSON("POST", policiesURL, &fleet.TeamPolicyRequest{
		Name:        "windows resend",
		Query:       "SELECT 2;",
		Platform:    "windows",
		ProfileUUID: new(winProf.ProfileUUID),
	}, http.StatusOK, &createWindows)
	requireProfileJSON(createWindows, winProf.ProfileUUID, winProf.Name)
	winPolicyID := uint(createWindows["policy"].(map[string]any)["id"].(float64))

	stored, err = s.ds.Policy(ctx, winPolicyID)
	require.NoError(t, err)
	require.Equal(t, new(winProf.ProfileUUID), stored.ResendWindowsProfileUUID)
	require.Nil(t, stored.ResendAppleProfileUUID)

	// POST without a profile: the key is omitted entirely (omitempty).
	var createNone map[string]any
	s.DoJSON("POST", policiesURL, &fleet.TeamPolicyRequest{
		Name:     "no resend",
		Query:    "SELECT 3;",
		Platform: "darwin",
	}, http.StatusOK, &createNone)
	require.Nil(t, resendProfileJSON(createNone), "resend_configuration_profile should be omitted")

	// GET single and list both populate the object.
	var getResp map[string]any
	s.DoJSON("GET", fmt.Sprintf("%s/%d", policiesURL, applePolicyID), nil, http.StatusOK, &getResp)
	requireProfileJSON(getResp, appleProf.ProfileUUID, appleProf.Name)

	listResp := &fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", policiesURL, nil, http.StatusOK, listResp)
	require.Len(t, listResp.Policies, 3)
	for _, p := range listResp.Policies {
		switch p.Name {
		case "apple resend":
			require.NotNil(t, p.ResendConfigurationProfile)
			assert.Equal(t, appleProf.ProfileUUID, p.ResendConfigurationProfile.UUID)
			assert.Equal(t, appleProf.Name, p.ResendConfigurationProfile.Name)
		case "windows resend":
			require.NotNil(t, p.ResendConfigurationProfile)
			assert.Equal(t, winProf.ProfileUUID, p.ResendConfigurationProfile.UUID)
		case "no resend":
			assert.Nil(t, p.ResendConfigurationProfile)
		default:
			t.Fatalf("unexpected policy %q", p.Name)
		}
	}

	// The profiles automation filter selects exactly the two profile-backed policies.
	filtered := &fleet.ListTeamPoliciesResponse{}
	s.DoJSON("GET", policiesURL, nil, http.StatusOK, filtered, "automation_type", "profiles")
	require.Len(t, filtered.Policies, 2)
	names := []string{filtered.Policies[0].Name, filtered.Policies[1].Name}
	assert.ElementsMatch(t, []string{"apple resend", "windows resend"}, names)

	t.Run("patch switches platform and clears the other column", func(t *testing.T) {
		var patched map[string]any
		s.DoJSON("PATCH", fmt.Sprintf("%s/%d", policiesURL, applePolicyID),
			json.RawMessage(fmt.Sprintf(`{"profile_uuid": %q}`, winProf.ProfileUUID)),
			http.StatusOK, &patched)
		requireProfileJSON(patched, winProf.ProfileUUID, winProf.Name)

		stored, err := s.ds.Policy(ctx, applePolicyID)
		require.NoError(t, err)
		require.Equal(t, new(winProf.ProfileUUID), stored.ResendWindowsProfileUUID)
		require.Nil(t, stored.ResendAppleProfileUUID)

		// Put it back for the remaining subtests.
		s.DoJSON("PATCH", fmt.Sprintf("%s/%d", policiesURL, applePolicyID),
			json.RawMessage(fmt.Sprintf(`{"profile_uuid": %q}`, appleProf.ProfileUUID)),
			http.StatusOK, &patched)
		requireProfileJSON(patched, appleProf.ProfileUUID, appleProf.Name)
	})

	t.Run("patch without profile_uuid leaves the profile untouched", func(t *testing.T) {
		var patched map[string]any
		s.DoJSON("PATCH", fmt.Sprintf("%s/%d", policiesURL, applePolicyID),
			json.RawMessage(`{"description": "unrelated edit"}`), http.StatusOK, &patched)
		requireProfileJSON(patched, appleProf.ProfileUUID, appleProf.Name)
	})

	t.Run("patch with empty string unsets", func(t *testing.T) {
		var patched map[string]any
		s.DoJSON("PATCH", fmt.Sprintf("%s/%d", policiesURL, winPolicyID),
			json.RawMessage(`{"profile_uuid": ""}`), http.StatusOK, &patched)
		require.Nil(t, resendProfileJSON(patched))

		stored, err := s.ds.Policy(ctx, winPolicyID)
		require.NoError(t, err)
		require.Nil(t, stored.ResendAppleProfileUUID)
		require.Nil(t, stored.ResendWindowsProfileUUID)
	})

	t.Run("patch with null unsets", func(t *testing.T) {
		// Re-attach, then clear with an explicit null.
		var patched map[string]any
		s.DoJSON("PATCH", fmt.Sprintf("%s/%d", policiesURL, winPolicyID),
			json.RawMessage(fmt.Sprintf(`{"profile_uuid": %q}`, winProf.ProfileUUID)),
			http.StatusOK, &patched)
		requireProfileJSON(patched, winProf.ProfileUUID, winProf.Name)

		s.DoJSON("PATCH", fmt.Sprintf("%s/%d", policiesURL, winPolicyID),
			json.RawMessage(`{"profile_uuid": null}`), http.StatusOK, &patched)
		require.Nil(t, resendProfileJSON(patched))
	})

	t.Run("rejected UUID prefixes", func(t *testing.T) {
		cases := []struct {
			name       string
			uuid       string
			wantErrMsg string
		}{
			{
				name:       "apple declaration",
				uuid:       fleet.MDMAppleDeclarationUUIDPrefix + uuid.NewString(),
				wantErrMsg: fleet.CantResendAppleDeclarationProfilesMessage,
			},
			{
				name:       "android profile",
				uuid:       fleet.MDMAndroidProfileUUIDPrefix + uuid.NewString(),
				wantErrMsg: "has an invalid prefix",
			},
			{
				name:       "unknown prefix",
				uuid:       "z" + uuid.NewString(),
				wantErrMsg: "has an invalid prefix",
			},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				// On create.
				res := s.Do("POST", policiesURL, &fleet.TeamPolicyRequest{
					Name:        "rejected " + c.name,
					Query:       "SELECT 1;",
					Platform:    "darwin",
					ProfileUUID: new(c.uuid),
				}, http.StatusBadRequest)
				require.Contains(t, extractServerErrorText(res.Body), c.wantErrMsg)

				// And on modify.
				res = s.Do("PATCH", fmt.Sprintf("%s/%d", policiesURL, applePolicyID),
					json.RawMessage(fmt.Sprintf(`{"profile_uuid": %q}`, c.uuid)), http.StatusBadRequest)
				require.Contains(t, extractServerErrorText(res.Body), c.wantErrMsg)
			})
		}
	})

	t.Run("profile from another team is rejected", func(t *testing.T) {
		res := s.Do("POST", policiesURL, &fleet.TeamPolicyRequest{
			Name:        "cross team resend",
			Query:       "SELECT 1;",
			Platform:    "darwin",
			ProfileUUID: new(otherAppleProf.ProfileUUID),
		}, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), "does not belong to team ID")

		res = s.Do("PATCH", fmt.Sprintf("%s/%d", policiesURL, applePolicyID),
			json.RawMessage(fmt.Sprintf(`{"profile_uuid": %q}`, otherAppleProf.ProfileUUID)),
			http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), "does not belong to team ID")
	})

	t.Run("nonexistent profile is rejected", func(t *testing.T) {
		res := s.Do("POST", policiesURL, &fleet.TeamPolicyRequest{
			Name:        "missing resend",
			Query:       "SELECT 1;",
			Platform:    "darwin",
			ProfileUUID: new(fleet.MDMAppleProfileUUIDPrefix + uuid.NewString()),
		}, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), "does not exist")
	})

	t.Run("all fleets policy rejects a resend profile", func(t *testing.T) {
		globalPol, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
			Name:  "global no resend",
			Query: "SELECT 1;",
		})
		require.NoError(t, err)

		res := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", globalPol.ID),
			json.RawMessage(fmt.Sprintf(`{"profile_uuid": %q}`, appleProf.ProfileUUID)),
			http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), "cannot have profile_uuid set")
	})

	t.Run("gitops spec apply", func(t *testing.T) {
		specURL := "/api/latest/fleet/spec/policies"

		s.Do("POST", specURL, fleet.ApplyPolicySpecsRequest{
			Specs: []*fleet.PolicySpec{{
				Name:        "gitops resend",
				Query:       "SELECT 9;",
				Platform:    "darwin",
				Team:        team.Name,
				ProfileUUID: new(appleProf.ProfileUUID),
			}},
		}, http.StatusOK)

		byName := func(name string) *fleet.Policy {
			list := &fleet.ListTeamPoliciesResponse{}
			s.DoJSON("GET", policiesURL, nil, http.StatusOK, list)
			for _, p := range list.Policies {
				if p.Name == name {
					return p
				}
			}
			t.Fatalf("policy %q not found", name)
			return nil
		}

		applied := byName("gitops resend")
		require.NotNil(t, applied.ResendConfigurationProfile)
		require.Equal(t, appleProf.ProfileUUID, applied.ResendConfigurationProfile.UUID)

		// Omitting profile_uuid on a later apply unsets it, per the declarative
		// GitOps convention (unlike PATCH, where absent means "no change").
		s.Do("POST", specURL, fleet.ApplyPolicySpecsRequest{
			Specs: []*fleet.PolicySpec{{
				Name:     "gitops resend",
				Query:    "SELECT 9;",
				Platform: "darwin",
				Team:     team.Name,
			}},
		}, http.StatusOK)
		require.Nil(t, byName("gitops resend").ResendConfigurationProfile)

		// A spec with no team cannot carry a profile.
		res := s.Do("POST", specURL, fleet.ApplyPolicySpecsRequest{
			Specs: []*fleet.PolicySpec{{
				Name:        "gitops global resend",
				Query:       "SELECT 9;",
				Platform:    "darwin",
				ProfileUUID: new(appleProf.ProfileUUID),
			}},
		}, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), "cannot have profile_uuid set")

		// A profile owned by another team is rejected here too.
		res = s.Do("POST", specURL, fleet.ApplyPolicySpecsRequest{
			Specs: []*fleet.PolicySpec{{
				Name:        "gitops cross team resend",
				Query:       "SELECT 9;",
				Platform:    "darwin",
				Team:        team.Name,
				ProfileUUID: new(otherAppleProf.ProfileUUID),
			}},
		}, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), "does not belong to team ID")
	})
}
