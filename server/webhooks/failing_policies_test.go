package webhooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/policies"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/test/automationtest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerFailingPoliciesWebhookBasic(t *testing.T) {
	ds := new(mock.Store)

	requestBody := ""

	policyID1 := uint(1)
	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		if id == policyID1 {
			return &fleet.Policy{
				PolicyData: fleet.PolicyData{
					ID:          policyID1,
					Name:        "policy1",
					Query:       "select 42",
					Description: "policy1 description",
					AuthorID:    ptr.Uint(1),
					AuthorName:  "Alice",
					AuthorEmail: "alice@example.com",
					TeamID:      nil,
					Resolution:  ptr.String("policy1 resolution"),
					Platform:    "darwin",
					Critical:    true,
					Type:        "dynamic",
				},
			}, nil
		}
		return nil, ctxerr.Wrap(ctx, sql.ErrNoRows)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestBodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		requestBody = string(requestBodyBytes)
	}))
	t.Cleanup(func() {
		ts.Close()
	})

	ac := &fleet.AppConfig{
		WebhookSettings: fleet.WebhookSettings{
			FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
				Enable:         true,
				DestinationURL: ts.URL,
				PolicyIDs:      []uint{1, 3},
			},
		},
		ServerSettings: fleet.ServerSettings{
			ServerURL: "https://fleet.example.com",
		},
	}

	ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
		return ac, nil
	}

	failingPolicySet := service.NewMemFailingPolicySet()
	err := failingPolicySet.AddHost(policyID1, fleet.PolicySetHost{
		ID:          1,
		Hostname:    "host1.example",
		DisplayName: "display1",
	})
	require.NoError(t, err)
	err = failingPolicySet.AddHost(policyID1, fleet.PolicySetHost{
		ID:          2,
		Hostname:    "host2.example",
		DisplayName: "display2",
	})
	require.NoError(t, err)

	automationtest.StubNoopRecording(ds)

	mockClock := time.Now()
	err = policies.TriggerFailingPoliciesAutomation(context.Background(), ds, slog.New(slog.DiscardHandler), failingPolicySet, func(pol *fleet.Policy, cfg policies.FailingPolicyAutomationConfig) error {
		serverURL, err := url.Parse(ac.ServerSettings.ServerURL)
		if err != nil {
			return err
		}
		return SendFailingPoliciesBatchedPOSTs(
			context.Background(), ds, pol, failingPolicySet, cfg.HostBatchSize, serverURL, cfg.WebhookURL, mockClock, slog.New(slog.DiscardHandler))
	})
	require.NoError(t, err)
	timestamp, err := mockClock.MarshalJSON()
	require.NoError(t, err)
	// Request body as defined in #2756.
	require.JSONEq(
		t, fmt.Sprintf(`{
    "timestamp": %s,
    "policy": {
        "id": 1,
        "name": "policy1",
        "query": "select 42",
        "description": "policy1 description",
        "author_id": 1,
        "author_name": "Alice",
        "author_email": "alice@example.com",
        "team_id": null,
		"fleet_id": null,
        "resolution": "policy1 resolution",
        "platform": "darwin",
        "created_at": "0001-01-01T00:00:00Z",
        "updated_at": "0001-01-01T00:00:00Z",
        "passing_host_count": 0,
        "failing_host_count": 2,
        "host_count_updated_at": null,
		"critical": true,
		"calendar_events_enabled": false,
		"conditional_access_enabled": false,
		"type": "dynamic"
    },
    "hosts": [
        {
            "id": 1,
            "hostname": "host1.example",
            "display_name": "display1",
            "url": "https://fleet.example.com/hosts/1"
        },
        {
            "id": 2,
            "hostname": "host2.example",
            "display_name": "display2",
            "url": "https://fleet.example.com/hosts/2"
        }
    ]
}`, timestamp), requestBody)

	hosts, err := failingPolicySet.ListHosts(policyID1)
	require.NoError(t, err)
	assert.Empty(t, hosts)

	requestBody = ""

	err = policies.TriggerFailingPoliciesAutomation(context.Background(), ds, slog.New(slog.DiscardHandler), failingPolicySet, func(pol *fleet.Policy, cfg policies.FailingPolicyAutomationConfig) error {
		serverURL, err := url.Parse(ac.ServerSettings.ServerURL)
		if err != nil {
			return err
		}
		return SendFailingPoliciesBatchedPOSTs(
			context.Background(), ds, pol, failingPolicySet, cfg.HostBatchSize, serverURL, cfg.WebhookURL, mockClock, slog.New(slog.DiscardHandler))
	})
	require.NoError(t, err)
	assert.Empty(t, requestBody)
}

func TestTriggerFailingPoliciesWebhookTeam(t *testing.T) {
	// webhook server
	webhookBody := ""
	webhookCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookCalled = true
		requestBodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		webhookBody = string(requestBodyBytes)
	}))
	t.Cleanup(func() {
		ts.Close()
	})

	ds := new(mock.Store)

	teamID := uint(1)

	policiesByID := map[uint]*fleet.Policy{
		1: {
			PolicyData: fleet.PolicyData{
				ID:                    1,
				Name:                  "policy1",
				Query:                 "select 1",
				Description:           "policy1 description",
				AuthorID:              ptr.Uint(1),
				AuthorName:            "Alice",
				AuthorEmail:           "alice@example.com",
				TeamID:                &teamID,
				Resolution:            ptr.String("policy1 resolution"),
				Platform:              "darwin",
				CalendarEventsEnabled: true,
				Type:                  "dynamic",
			},
		},
		2: {
			PolicyData: fleet.PolicyData{
				ID:          2,
				Name:        "policy2",
				Query:       "select 2",
				Description: "policy2 description",
				AuthorID:    ptr.Uint(1),
				AuthorName:  "Alice",
				AuthorEmail: "alice@example.com",
				TeamID:      &teamID,
				Resolution:  ptr.String("policy2 resolution"),
				Platform:    "darwin",
				Type:        "dynamic",
			},
		},
		3: {
			PolicyData: fleet.PolicyData{
				ID:          2,
				Name:        "policy3",
				Query:       "select 3",
				Description: "policy3 description",
				AuthorID:    ptr.Uint(1),
				AuthorName:  "Alice",
				AuthorEmail: "alice@example.com",
				TeamID:      nil, // global policy
				Resolution:  ptr.String("policy3 resolution"),
				Platform:    "darwin",
				Type:        "dynamic",
			},
		},
	}

	ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
		policy, ok := policiesByID[id]
		if !ok {
			return nil, ctxerr.Wrap(ctx, sql.ErrNoRows)
		}
		return policy, nil
	}
	ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
		if tid == teamID {
			return &fleet.TeamLite{
				ID: teamID,
				Config: fleet.TeamConfigLite{
					WebhookSettings: fleet.TeamWebhookSettings{
						FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
							Enable:         true,
							DestinationURL: ts.URL,
							PolicyIDs:      []uint{1},
						},
					},
				},
			}, nil
		}
		return nil, ctxerr.Wrap(ctx, sql.ErrNoRows)
	}

	ac := &fleet.AppConfig{
		ServerSettings: fleet.ServerSettings{
			ServerURL: "https://fleet.example.com",
		},
	}

	ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
		return ac, nil
	}

	failingPolicySet := service.NewMemFailingPolicySet()
	err := failingPolicySet.AddHost(1, fleet.PolicySetHost{
		ID:          1,
		Hostname:    "host1",
		DisplayName: "display1",
	})
	require.NoError(t, err)
	err = failingPolicySet.AddHost(2, fleet.PolicySetHost{
		ID:          2,
		Hostname:    "host2",
		DisplayName: "display2",
	})
	require.NoError(t, err)

	automationtest.StubNoopRecording(ds)

	now := time.Now()
	err = policies.TriggerFailingPoliciesAutomation(context.Background(), ds, slog.New(slog.DiscardHandler), failingPolicySet, func(pol *fleet.Policy, cfg policies.FailingPolicyAutomationConfig) error {
		serverURL, err := url.Parse(ac.ServerSettings.ServerURL)
		if err != nil {
			return err
		}
		return SendFailingPoliciesBatchedPOSTs(
			context.Background(), ds, pol, failingPolicySet, cfg.HostBatchSize, serverURL, cfg.WebhookURL, now, slog.New(slog.DiscardHandler))
	})
	require.NoError(t, err)

	timestamp, err := now.MarshalJSON()
	require.NoError(t, err)

	// Request body as defined in #2756.
	require.True(t, webhookCalled, "webhook was not called")
	require.JSONEq(
		t, fmt.Sprintf(`{
    "timestamp": %s,
    "policy": {
        "id": 1,
        "name": "policy1",
        "query": "select 1",
        "description": "policy1 description",
        "author_id": 1,
        "author_name": "Alice",
        "author_email": "alice@example.com",
        "team_id": 1,
		"fleet_id": 1,
        "resolution": "policy1 resolution",
        "platform": "darwin",
        "created_at": "0001-01-01T00:00:00Z",
        "updated_at": "0001-01-01T00:00:00Z",
        "passing_host_count": 0,
        "failing_host_count": 1,
        "host_count_updated_at": null,
		"critical": false,
		"calendar_events_enabled": true,
		"conditional_access_enabled": false,
		"type": "dynamic"
    },
    "hosts": [
        {
            "id": 1,
            "hostname": "host1",
            "display_name": "display1",
            "url": "https://fleet.example.com/hosts/1"
        }
    ]
}`, timestamp), webhookBody)

	hosts, err := failingPolicySet.ListHosts(1)
	require.NoError(t, err)
	assert.Empty(t, hosts)

	webhookBody = ""

	err = policies.TriggerFailingPoliciesAutomation(context.Background(), ds, slog.New(slog.DiscardHandler), failingPolicySet, func(pol *fleet.Policy, cfg policies.FailingPolicyAutomationConfig) error {
		serverURL, err := url.Parse(ac.ServerSettings.ServerURL)
		if err != nil {
			return err
		}
		return SendFailingPoliciesBatchedPOSTs(
			context.Background(), ds, pol, failingPolicySet, cfg.HostBatchSize, serverURL, cfg.WebhookURL, now, slog.New(slog.DiscardHandler))
	})
	require.NoError(t, err)
	assert.Empty(t, webhookBody)
}

func TestSendFailingPoliciesRecordsAutomationStatus(t *testing.T) {
	t.Run("successful POST records success per host", func(t *testing.T) {
		ds := new(mock.Store)

		// The webhook now looks up run IDs via GetFailingPolicyRunIDs and calls
		// CreatePolicyAutomationExecutionsFunc with execution rows pointing at them. We
		// stub both at that boundary; the per-method behavior (transition
		// upsert, execution-row build) is covered by the MySQL integration
		// tests.
		ds.GetFailingPolicyRunsFunc = func(ctx context.Context, policyIDs, hostIDs []uint) ([]fleet.PolicyRunRef, error) {
			out := make([]fleet.PolicyRunRef, 0, len(policyIDs)*len(hostIDs))
			idx := uint(100)
			for _, pid := range policyIDs {
				for _, hid := range hostIDs {
					out = append(out, fleet.PolicyRunRef{PolicyID: pid, HostID: hid, RunID: idx})
					idx++
				}
			}
			return out, nil
		}
		var recordedExecutions []fleet.PolicyRunRef
		var createdBatch uuid.UUID
		ds.CreatePolicyAutomationExecutionsFunc = func(ctx context.Context, typ fleet.PolicyAutomationType, executions []fleet.PolicyRunRef) (uuid.UUID, error) {
			require.Equal(t, fleet.PolicyAutomationWebhook, typ)
			recordedExecutions = append(recordedExecutions, executions...)
			createdBatch = uuid.New()
			return createdBatch, nil
		}
		var finalErr error
		var finalBatch uuid.UUID
		ds.UpdatePolicyAutomationExecutionsFunc = func(ctx context.Context, batchID uuid.UUID, outcomeErr error) error {
			finalBatch = batchID
			finalErr = outcomeErr
			return nil
		}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(ts.Close)
		webhookURL, err := url.Parse(ts.URL)
		require.NoError(t, err)
		serverURL, err := url.Parse("https://fleet.example.com")
		require.NoError(t, err)

		set := service.NewMemFailingPolicySet()
		policyID := uint(42)
		require.NoError(t, set.AddHost(policyID, fleet.PolicySetHost{ID: 1, Hostname: "h1"}))
		require.NoError(t, set.AddHost(policyID, fleet.PolicySetHost{ID: 2, Hostname: "h2"}))

		policy := &fleet.Policy{PolicyData: fleet.PolicyData{ID: policyID, Name: "p"}}
		err = SendFailingPoliciesBatchedPOSTs(
			context.Background(), ds, policy, set, 0, serverURL, webhookURL, time.Now(), slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)

		require.Len(t, recordedExecutions, 2)
		require.Equal(t, policyID, recordedExecutions[0].PolicyID)
		require.Equal(t, policyID, recordedExecutions[1].PolicyID)
		require.NotEqual(t, uuid.Nil, createdBatch, "a non-nil batch UUID must be assigned for newly-inserted runs")
		require.NoError(t, finalErr)
		require.Equal(t, createdBatch, finalBatch, "the finalize call must target the same batch UUID created at INSERT time")
	})

	t.Run("POST failure records failure with error message", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetFailingPolicyRunsFunc = func(ctx context.Context, policyIDs, hostIDs []uint) ([]fleet.PolicyRunRef, error) {
			out := make([]fleet.PolicyRunRef, 0, len(hostIDs))
			for i, hid := range hostIDs {
				out = append(out, fleet.PolicyRunRef{PolicyID: policyIDs[0], HostID: hid, RunID: uint(i + 200)})
			}
			return out, nil
		}
		ds.CreatePolicyAutomationExecutionsFunc = func(ctx context.Context, typ fleet.PolicyAutomationType, executions []fleet.PolicyRunRef) (uuid.UUID, error) {
			return uuid.New(), nil
		}
		var finalErr error
		ds.UpdatePolicyAutomationExecutionsFunc = func(ctx context.Context, batchID uuid.UUID, outcomeErr error) error {
			finalErr = outcomeErr
			return nil
		}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(ts.Close)
		webhookURL, err := url.Parse(ts.URL)
		require.NoError(t, err)
		serverURL, err := url.Parse("https://fleet.example.com")
		require.NoError(t, err)

		set := service.NewMemFailingPolicySet()
		policyID := uint(7)
		require.NoError(t, set.AddHost(policyID, fleet.PolicySetHost{ID: 9, Hostname: "h9"}))
		policy := &fleet.Policy{PolicyData: fleet.PolicyData{ID: policyID, Name: "p"}}

		err = SendFailingPoliciesBatchedPOSTs(
			context.Background(), ds, policy, set, 0, serverURL, webhookURL, time.Now(), slog.New(slog.DiscardHandler),
		)
		require.Error(t, err, "POST 500 should propagate as an error")
		require.Error(t, finalErr, "non-nil outcomeErr (Failure) must be passed to Finalize on POST failure")
	})

	t.Run("hot path didn't record any runs → orchestrator gets empty input, no execution rows but POST still fires", func(t *testing.T) {
		ds := new(mock.Store)
		// Simulate the case where the centralized osquery hot-path write
		// failed (e.g. transient DB error). GetFailingPolicyRunIDs returns
		// empty, the dispatcher builds zero executions, RecordPolicyAutomationBatch
		// returns uuid.Nil. Webhook still POSTs.
		ds.GetFailingPolicyRunsFunc = func(ctx context.Context, policyIDs, hostIDs []uint) ([]fleet.PolicyRunRef, error) {
			return nil, nil
		}
		batchCalled := false
		ds.CreatePolicyAutomationExecutionsFunc = func(ctx context.Context, typ fleet.PolicyAutomationType, executions []fleet.PolicyRunRef) (uuid.UUID, error) {
			batchCalled = true
			require.Empty(t, executions, "no run_ids found → no executions to record")
			return uuid.Nil, nil
		}
		updateCalled := false
		ds.UpdatePolicyAutomationExecutionsFunc = func(ctx context.Context, batchID uuid.UUID, outcomeErr error) error {
			updateCalled = true
			return nil
		}

		var requestCount int
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requestCount++
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(ts.Close)
		webhookURL, err := url.Parse(ts.URL)
		require.NoError(t, err)
		serverURL, err := url.Parse("https://fleet.example.com")
		require.NoError(t, err)

		set := service.NewMemFailingPolicySet()
		policyID := uint(123)
		require.NoError(t, set.AddHost(policyID, fleet.PolicySetHost{ID: 11, Hostname: "h11"}))
		policy := &fleet.Policy{PolicyData: fleet.PolicyData{ID: policyID, Name: "p"}}

		err = SendFailingPoliciesBatchedPOSTs(
			context.Background(), ds, policy, set, 0, serverURL, webhookURL, time.Now(), slog.New(slog.DiscardHandler),
		)
		require.NoError(t, err)
		require.True(t, batchCalled, "the orchestrator must still be called even on a re-flip — it's the orchestrator's job to detect the no-op")
		// updateCalled may be true with batchID=uuid.Nil — the inlined
		// finalize at each webhook site calls the datastore unconditionally
		// and the datastore method short-circuits internally on uuid.Nil.
		// The contract that matters is "no UI signal for re-flips," which is
		// enforced by the orchestrator returning uuid.Nil and the datastore's
		// Update being a no-op against that batchID (verified at the MySQL
		// integration test layer).
		_ = updateCalled
		require.Equal(t, 1, requestCount, "webhook POST must still fire on re-flip even though no execution row is recorded")
	})
}

func TestSendBatchedPOSTs(t *testing.T) {
	allHosts := []uint{}
	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		var payload failingPoliciesPayload
		err = json.Unmarshal(b, &payload)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		for _, host := range payload.FailingHosts {
			allHosts = append(allHosts, host.ID)
		}
		requestCount++
	}))
	t.Cleanup(func() {
		ts.Close()
	})
	p := &fleet.Policy{
		PolicyData: fleet.PolicyData{
			ID:          1,
			Name:        "policy1",
			Query:       "select 42",
			Description: "policy1 description",
			AuthorID:    ptr.Uint(1),
			AuthorName:  "Alice",
			AuthorEmail: "alice@example.com",
			TeamID:      nil,
			Resolution:  ptr.String("policy1 resolution"),
			Platform:    "darwin",
		},
	}

	makeHosts := func(c int) []fleet.PolicySetHost {
		hosts := make([]fleet.PolicySetHost, c)
		for i := 0; i < len(hosts); i++ {
			hosts[i] = fleet.PolicySetHost{
				ID:       uint(i + 1), //nolint:gosec // dismiss G115
				Hostname: fmt.Sprintf("hostname-%d", i+1),
			}
		}
		return hosts
	}

	now := time.Now()
	serverURL, err := url.Parse("https://fleet.example.com")
	require.NoError(t, err)

	for _, tc := range []struct {
		name            string
		hostCount       int
		batchSize       int
		expRequestCount int
	}{
		{
			name:            "no-batching",
			hostCount:       10,
			batchSize:       0,
			expRequestCount: 1,
		},
		{
			name:            "one-host-no-batching",
			hostCount:       1,
			batchSize:       0,
			expRequestCount: 1,
		},
		{
			name:            "batching-by-one",
			hostCount:       10,
			batchSize:       1,
			expRequestCount: 10,
		},
		{
			name:            "batch-matches-host-count",
			hostCount:       10,
			batchSize:       10,
			expRequestCount: 1,
		},
		{
			name:            "batch-last-with-one",
			hostCount:       10,
			batchSize:       9,
			expRequestCount: 2,
		},
		{
			name:            "batch-bigger-than-host-count",
			hostCount:       10,
			batchSize:       11,
			expRequestCount: 1,
		},
		{
			name:            "100k-hosts-no-batching",
			hostCount:       100000,
			batchSize:       0,
			expRequestCount: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			allHosts = []uint{}
			requestCount = 0
			hosts := makeHosts(tc.hostCount)
			failingPolicySet := service.NewMemFailingPolicySet()
			for _, host := range hosts {
				err := failingPolicySet.AddHost(p.ID, host)
				require.NoError(t, err)
			}

			webhookURL, err := url.Parse(ts.URL)
			require.NoError(t, err)

			ds := new(mock.Store)
			automationtest.StubNoopRecording(ds)

			err = SendFailingPoliciesBatchedPOSTs(
				context.Background(),
				ds,
				p,
				failingPolicySet,
				tc.batchSize,
				serverURL,
				webhookURL,
				now,
				slog.New(slog.DiscardHandler),
			)
			require.NoError(t, err)
			require.Len(t, allHosts, tc.hostCount)
			for i := range allHosts {
				require.Equal(t, allHosts[i], hosts[i].ID)
			}
			require.Equal(t, tc.expRequestCount, requestCount)
			setHosts, err := failingPolicySet.ListHosts(p.ID)
			require.NoError(t, err)
			assert.Empty(t, setHosts)
		})
	}
}
