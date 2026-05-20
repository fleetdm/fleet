package worker

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	"github.com/google/uuid"
	zendesk "github.com/nukosuke/go-zendesk/zendesk"
	"github.com/stretchr/testify/require"
)

// recordingMocks captures the side-effects of the policy-automation recording
// helpers wired onto a mock.Store. Used by the tests below to assert what got
// recorded.
type recordingMocks struct {
	recordedExecutions []fleet.PolicyRunRef
	createdType        fleet.PolicyAutomationType
	createdBatch       uuid.UUID
	statusUpdates      []automationStatusUpdate
	// hostsWithRunIDs controls whether GetFailingPolicyRunIDs returns a
	// non-empty map for the given host IDs (simulating the centralized hot-
	// path write having already happened). When false for all hosts, the
	// dispatch surface sees no executions to record and the orchestrator
	// returns uuid.Nil.
	hostsWithRunIDs map[uint]bool
}

type automationStatusUpdate struct {
	batchID    uuid.UUID
	outcomeErr error
}

// wireRecordingMocks stubs the policy-automation methods the worker invokes
// at queue-time and run-time and captures their inputs.
//
// foundFlags controls whether GetFailingPolicyRunIDs reports each input host
// as having a recorded run row (matches the order of `hosts` passed to the
// dispatcher). All-false → orchestrator gets an empty executions slice and
// returns uuid.Nil — simulating the case where the hot path failed to record.
func wireRecordingMocks(ds *mock.Store, foundFlags []bool) *recordingMocks {
	rm := &recordingMocks{hostsWithRunIDs: map[uint]bool{}}
	ds.GetFailingPolicyRunsFunc = func(ctx context.Context, policyIDs, hostIDs []uint) ([]fleet.PolicyRunRef, error) {
		var out []fleet.PolicyRunRef
		for i, hid := range hostIDs {
			found := true
			if i < len(foundFlags) {
				found = foundFlags[i]
			}
			rm.hostsWithRunIDs[hid] = found
			if found {
				out = append(out, fleet.PolicyRunRef{PolicyID: policyIDs[0], HostID: hid, RunID: uint(i + 1000)})
			}
		}
		return out, nil
	}
	ds.CreatePolicyAutomationExecutionsFunc = func(ctx context.Context, typ fleet.PolicyAutomationType, executions []fleet.PolicyRunRef) (uuid.UUID, error) {
		rm.recordedExecutions = append([]fleet.PolicyRunRef{}, executions...)
		rm.createdType = typ
		if len(executions) == 0 {
			return uuid.Nil, nil
		}
		rm.createdBatch = uuid.New()
		return rm.createdBatch, nil
	}
	ds.UpdatePolicyAutomationExecutionsFunc = func(ctx context.Context, batchID uuid.UUID, outcomeErr error) error {
		rm.statusUpdates = append(rm.statusUpdates, automationStatusUpdate{
			batchID:    batchID,
			outcomeErr: outcomeErr,
		})
		return nil
	}
	return rm
}

func TestJiraFailingPolicyRecordingLifecycle(t *testing.T) {
	t.Run("queue records pending and run finalizes success", func(t *testing.T) {
		ds := new(mock.Store)
		rm := wireRecordingMocks(ds, []bool{true, true})
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				Integrations: fleet.Integrations{Jira: []*fleet.JiraIntegration{{
					EnableFailingPolicies: true,
					URL:                   "https://jira.example.com",
					Username:              "u",
					APIToken:              "t",
					ProjectKey:            "FLEET",
				}}},
			}, nil
		}

		ctx := t.Context()
		logger := slog.New(slog.DiscardHandler)
		policy := &fleet.Policy{PolicyData: fleet.PolicyData{ID: 1, Name: "p1"}}
		hosts := []fleet.PolicySetHost{{ID: 1, Hostname: "h1"}, {ID: 2, Hostname: "h2"}}

		require.NoError(t, QueueJiraFailingPolicyJob(ctx, ds, logger, policy, hosts))
		require.Equal(t, fleet.PolicyAutomationJira, rm.createdType)
		require.Len(t, rm.recordedExecutions, 2)
		require.NotEqual(t, uuid.Nil, rm.createdBatch, "queue must produce a non-nil batch UUID")
		require.Empty(t, rm.statusUpdates, "queue should not finalize status")

		queuedBatch := rm.createdBatch

		j := &Jira{
			FleetURL:  "https://fleet.example.com",
			Datastore: ds,
			Log:       logger,
			NewClientFunc: func(_ *externalsvc.JiraOptions) (JiraClient, error) {
				return stubJiraClient{}, nil
			},
		}
		argsJSON, err := json.Marshal(jiraArgs{FailingPolicy: &failingPolicyArgs{
			PolicyID:                1,
			PolicyName:              "p1",
			Hosts:                   hosts,
			PolicyAutomationBatchID: queuedBatch,
		}})
		require.NoError(t, err)
		require.NoError(t, j.Run(ctx, argsJSON))

		require.Len(t, rm.statusUpdates, 1)
		require.NoError(t, rm.statusUpdates[0].outcomeErr)
	})

	t.Run("run finalizes failure when client returns error", func(t *testing.T) {
		ds := new(mock.Store)
		rm := wireRecordingMocks(ds, []bool{true})
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				Integrations: fleet.Integrations{Jira: []*fleet.JiraIntegration{{
					EnableFailingPolicies: true,
					URL:                   "https://jira.example.com",
					Username:              "u",
					APIToken:              "t",
					ProjectKey:            "FLEET",
				}}},
			}, nil
		}
		j := &Jira{
			FleetURL:  "https://fleet.example.com",
			Datastore: ds,
			Log:       slog.New(slog.DiscardHandler),
			NewClientFunc: func(_ *externalsvc.JiraOptions) (JiraClient, error) {
				return stubJiraClient{createErr: errors.New("jira boom")}, nil
			},
		}
		batch := uuid.New()
		argsJSON, err := json.Marshal(jiraArgs{FailingPolicy: &failingPolicyArgs{
			PolicyID:                1,
			PolicyName:              "p1",
			Hosts:                   []fleet.PolicySetHost{{ID: 1, Hostname: "h1"}},
			PolicyAutomationBatchID: batch,
		}})
		require.NoError(t, err)

		err = j.Run(t.Context(), argsJSON)
		require.Error(t, err)
		require.Len(t, rm.statusUpdates, 1)
		require.ErrorContains(t, rm.statusUpdates[0].outcomeErr, "jira boom")
	})

	t.Run("hot path didn't record any runs → orchestrator returns uuid.Nil, no finalize", func(t *testing.T) {
		ds := new(mock.Store)
		rm := wireRecordingMocks(ds, []bool{false, false})
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}

		policy := &fleet.Policy{PolicyData: fleet.PolicyData{ID: 5, Name: "p5"}}
		hosts := []fleet.PolicySetHost{{ID: 1, Hostname: "h1"}, {ID: 2, Hostname: "h2"}}
		require.NoError(t, QueueJiraFailingPolicyJob(t.Context(), ds, slog.New(slog.DiscardHandler), policy, hosts))

		require.Empty(t, rm.recordedExecutions, "no run_ids found → no executions to record")
		require.Equal(t, uuid.Nil, rm.createdBatch, "empty executions → uuid.Nil")
		require.Empty(t, rm.statusUpdates)
	})

	t.Run("integration disabled before run finalizes as failure", func(t *testing.T) {
		ds := new(mock.Store)
		rm := wireRecordingMocks(ds, []bool{true})
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			// No Jira integration configured → getClient returns nil.
			return &fleet.AppConfig{}, nil
		}
		j := &Jira{
			FleetURL:  "https://fleet.example.com",
			Datastore: ds,
			Log:       slog.New(slog.DiscardHandler),
			NewClientFunc: func(_ *externalsvc.JiraOptions) (JiraClient, error) {
				return stubJiraClient{}, nil
			},
		}
		batch := uuid.New()
		argsJSON, err := json.Marshal(jiraArgs{FailingPolicy: &failingPolicyArgs{
			PolicyID:                1,
			PolicyName:              "p1",
			Hosts:                   []fleet.PolicySetHost{{ID: 1}},
			PolicyAutomationBatchID: batch,
		}})
		require.NoError(t, err)

		require.NoError(t, j.Run(t.Context(), argsJSON))
		require.Len(t, rm.statusUpdates, 1)
		require.ErrorContains(t, rm.statusUpdates[0].outcomeErr, "disabled")
	})
}

// stubJiraClient implements the minimal JiraClient surface needed by tests.
type stubJiraClient struct {
	createErr error
}

func (s stubJiraClient) CreateJiraIssue(_ context.Context, issue *jira.Issue) (*jira.Issue, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	out := *issue
	out.ID = "10001"
	out.Key = "FLEET-1"
	return &out, nil
}

func (stubJiraClient) JiraConfigMatches(_ *externalsvc.JiraOptions) bool { return true }

// stubZendeskClient is the Zendesk twin of stubJiraClient.
type stubZendeskClient struct {
	createErr error
}

func (s stubZendeskClient) CreateZendeskTicket(_ context.Context, ticket *zendesk.Ticket) (*zendesk.Ticket, error) {
	if s.createErr != nil {
		return nil, s.createErr
	}
	out := *ticket
	out.ID = 4242
	return &out, nil
}

func (stubZendeskClient) ZendeskConfigMatches(_ *externalsvc.ZendeskOptions) bool { return true }

func TestZendeskFailingPolicyRecordingLifecycle(t *testing.T) {
	// This test mirrors TestJiraFailingPolicyRecordingLifecycle. Jira and
	// Zendesk share the same failingPolicyArgs shape and the same finalize
	// helper in runFailingPolicy, so a regression in one path is likely to
	// also need fixing in the other — parallel coverage catches drift.
	t.Run("queue records pending and run finalizes success", func(t *testing.T) {
		ds := new(mock.Store)
		rm := wireRecordingMocks(ds, []bool{true, true})
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				Integrations: fleet.Integrations{Zendesk: []*fleet.ZendeskIntegration{{
					EnableFailingPolicies: true,
					URL:                   "https://zendesk.example.com",
					Email:                 "u@example.com",
					APIToken:              "t",
					GroupID:               1,
				}}},
			}, nil
		}

		ctx := t.Context()
		logger := slog.New(slog.DiscardHandler)
		policy := &fleet.Policy{PolicyData: fleet.PolicyData{ID: 1, Name: "p1"}}
		hosts := []fleet.PolicySetHost{{ID: 1, Hostname: "h1"}, {ID: 2, Hostname: "h2"}}

		require.NoError(t, QueueZendeskFailingPolicyJob(ctx, ds, logger, policy, hosts))
		require.Equal(t, fleet.PolicyAutomationZendesk, rm.createdType)
		require.Len(t, rm.recordedExecutions, 2)
		require.NotEqual(t, uuid.Nil, rm.createdBatch)
		require.Empty(t, rm.statusUpdates, "queue should not finalize status")

		queuedBatch := rm.createdBatch

		z := &Zendesk{
			FleetURL:  "https://fleet.example.com",
			Datastore: ds,
			Log:       logger,
			NewClientFunc: func(_ *externalsvc.ZendeskOptions) (ZendeskClient, error) {
				return stubZendeskClient{}, nil
			},
		}
		argsJSON, err := json.Marshal(zendeskArgs{FailingPolicy: &failingPolicyArgs{
			PolicyID:                1,
			PolicyName:              "p1",
			Hosts:                   hosts,
			PolicyAutomationBatchID: queuedBatch,
		}})
		require.NoError(t, err)
		require.NoError(t, z.Run(ctx, argsJSON))

		require.Len(t, rm.statusUpdates, 1)
		require.NoError(t, rm.statusUpdates[0].outcomeErr)
	})

	t.Run("run finalizes failure when client returns error", func(t *testing.T) {
		ds := new(mock.Store)
		rm := wireRecordingMocks(ds, []bool{true})
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				Integrations: fleet.Integrations{Zendesk: []*fleet.ZendeskIntegration{{
					EnableFailingPolicies: true,
					URL:                   "https://zendesk.example.com",
					Email:                 "u@example.com",
					APIToken:              "t",
					GroupID:               1,
				}}},
			}, nil
		}
		z := &Zendesk{
			FleetURL:  "https://fleet.example.com",
			Datastore: ds,
			Log:       slog.New(slog.DiscardHandler),
			NewClientFunc: func(_ *externalsvc.ZendeskOptions) (ZendeskClient, error) {
				return stubZendeskClient{createErr: errors.New("zendesk boom")}, nil
			},
		}
		batch := uuid.New()
		argsJSON, err := json.Marshal(zendeskArgs{FailingPolicy: &failingPolicyArgs{
			PolicyID:                1,
			PolicyName:              "p1",
			Hosts:                   []fleet.PolicySetHost{{ID: 1, Hostname: "h1"}},
			PolicyAutomationBatchID: batch,
		}})
		require.NoError(t, err)

		err = z.Run(t.Context(), argsJSON)
		require.Error(t, err)
		require.Len(t, rm.statusUpdates, 1)
		require.ErrorContains(t, rm.statusUpdates[0].outcomeErr, "zendesk boom")
	})

	t.Run("hot path didn't record any runs → orchestrator returns uuid.Nil, no finalize", func(t *testing.T) {
		ds := new(mock.Store)
		rm := wireRecordingMocks(ds, []bool{false, false})
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}

		policy := &fleet.Policy{PolicyData: fleet.PolicyData{ID: 5, Name: "p5"}}
		hosts := []fleet.PolicySetHost{{ID: 1, Hostname: "h1"}, {ID: 2, Hostname: "h2"}}
		require.NoError(t, QueueZendeskFailingPolicyJob(t.Context(), ds, slog.New(slog.DiscardHandler), policy, hosts))

		require.Empty(t, rm.recordedExecutions, "no run_ids found → no executions to record")
		require.Equal(t, uuid.Nil, rm.createdBatch, "empty executions → uuid.Nil")
		require.Empty(t, rm.statusUpdates)
	})

	t.Run("integration disabled before run finalizes as failure", func(t *testing.T) {
		ds := new(mock.Store)
		rm := wireRecordingMocks(ds, []bool{true})
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		z := &Zendesk{
			FleetURL:  "https://fleet.example.com",
			Datastore: ds,
			Log:       slog.New(slog.DiscardHandler),
			NewClientFunc: func(_ *externalsvc.ZendeskOptions) (ZendeskClient, error) {
				return stubZendeskClient{}, nil
			},
		}
		batch := uuid.New()
		argsJSON, err := json.Marshal(zendeskArgs{FailingPolicy: &failingPolicyArgs{
			PolicyID:                1,
			PolicyName:              "p1",
			Hosts:                   []fleet.PolicySetHost{{ID: 1}},
			PolicyAutomationBatchID: batch,
		}})
		require.NoError(t, err)

		require.NoError(t, z.Run(t.Context(), argsJSON))
		require.Len(t, rm.statusUpdates, 1)
		require.ErrorContains(t, rm.statusUpdates[0].outcomeErr, "disabled")
	})
}

// TestFailingPolicyArgsBackwardCompatJSON guards against breaking workers that
// dequeue jobs queued by an older binary which didn't know about the
// PolicyAutomationBatchID field. The JSON below is the exact shape jobs took
// before this branch landed — unmarshaling must succeed and yield uuid.Nil so
// the deferred finalize becomes a no-op at the datastore layer.
func TestFailingPolicyArgsBackwardCompatJSON(t *testing.T) {
	oldArgsJSON := []byte(`{
		"failing_policy": {
			"policy_id": 7,
			"policy_name": "old-policy",
			"policy_critical": false,
			"hosts": [{"id": 11, "hostname": "h-old"}],
			"team_id": null
		}
	}`)

	t.Run("jira unmarshals old args with uuid.Nil batch", func(t *testing.T) {
		var args jiraArgs
		require.NoError(t, json.Unmarshal(oldArgsJSON, &args))
		require.NotNil(t, args.FailingPolicy)
		require.Equal(t, uuid.Nil, args.FailingPolicy.PolicyAutomationBatchID)
		require.Equal(t, uint(7), args.FailingPolicy.PolicyID)
		require.Equal(t, "old-policy", args.FailingPolicy.PolicyName)
		require.Len(t, args.FailingPolicy.Hosts, 1)
	})

	t.Run("zendesk unmarshals old args with uuid.Nil batch", func(t *testing.T) {
		var args zendeskArgs
		require.NoError(t, json.Unmarshal(oldArgsJSON, &args))
		require.NotNil(t, args.FailingPolicy)
		require.Equal(t, uuid.Nil, args.FailingPolicy.PolicyAutomationBatchID)
	})

	// End-to-end: a worker running new code against args queued by old code
	// must complete the integration call AND finalize with uuid.Nil (the
	// datastore method then no-ops, so no stray rows are written or updated).
	t.Run("jira run on old args completes and finalizes with uuid.Nil", func(t *testing.T) {
		ds := new(mock.Store)
		rm := wireRecordingMocks(ds, []bool{true})
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				Integrations: fleet.Integrations{Jira: []*fleet.JiraIntegration{{
					EnableFailingPolicies: true,
					URL:                   "https://jira.example.com",
					Username:              "u",
					APIToken:              "t",
					ProjectKey:            "FLEET",
				}}},
			}, nil
		}
		j := &Jira{
			FleetURL:  "https://fleet.example.com",
			Datastore: ds,
			Log:       slog.New(slog.DiscardHandler),
			NewClientFunc: func(_ *externalsvc.JiraOptions) (JiraClient, error) {
				return stubJiraClient{}, nil
			},
		}

		require.NoError(t, j.Run(t.Context(), oldArgsJSON))
		require.Len(t, rm.statusUpdates, 1)
		require.Equal(t, uuid.Nil, rm.statusUpdates[0].batchID, "old jobs must finalize with uuid.Nil so the DS no-ops")
		require.NoError(t, rm.statusUpdates[0].outcomeErr)
	})

	t.Run("zendesk run on old args completes and finalizes with uuid.Nil", func(t *testing.T) {
		ds := new(mock.Store)
		rm := wireRecordingMocks(ds, []bool{true})
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				Integrations: fleet.Integrations{Zendesk: []*fleet.ZendeskIntegration{{
					EnableFailingPolicies: true,
					URL:                   "https://zendesk.example.com",
					Email:                 "u@example.com",
					APIToken:              "t",
					GroupID:               1,
				}}},
			}, nil
		}
		z := &Zendesk{
			FleetURL:  "https://fleet.example.com",
			Datastore: ds,
			Log:       slog.New(slog.DiscardHandler),
			NewClientFunc: func(_ *externalsvc.ZendeskOptions) (ZendeskClient, error) {
				return stubZendeskClient{}, nil
			},
		}

		require.NoError(t, z.Run(t.Context(), oldArgsJSON))
		require.Len(t, rm.statusUpdates, 1)
		require.Equal(t, uuid.Nil, rm.statusUpdates[0].batchID, "old jobs must finalize with uuid.Nil so the DS no-ops")
		require.NoError(t, rm.statusUpdates[0].outcomeErr)
	})
}
