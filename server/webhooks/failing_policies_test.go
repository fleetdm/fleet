package webhooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerFailingPoliciesWebhookBasic(t *testing.T) {
	ds := new(mock.Store)

	requestBody := make([]string, 2)
	numRequests := 0

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
		requestBody[numRequests] = string(requestBodyBytes)
		numRequests++
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

	mockClock := time.Now()
	err = policies.TriggerFailingPoliciesAutomation(context.Background(), ds, kitlog.NewNopLogger(), failingPolicySet, func(pol *fleet.Policy, cfg policies.FailingPolicyAutomationConfig) error {
		serverURL, err := url.Parse(ac.ServerSettings.ServerURL)
		if err != nil {
			return err
		}
		return SendFailingPoliciesBatchedPOSTs(
			context.Background(), pol, failingPolicySet, cfg.HostBatchSize, serverURL, cfg.WebhookURL, mockClock, kitlog.NewNopLogger())
	})
	require.NoError(t, err)
	timestamp, err := mockClock.MarshalJSON()
	require.NoError(t, err)
	// Request body as defined in #2756.
	require.Equal(t, numRequests, 2)
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
        "resolution": "policy1 resolution",
        "platform": "darwin",
        "created_at": "0001-01-01T00:00:00Z",
        "updated_at": "0001-01-01T00:00:00Z",
        "passing_host_count": 0,
        "failing_host_count": 2,
        "host_count_updated_at": null,
		"critical": true,
		"calendar_events_enabled": false
    },
    "hosts": [
        {
            "id": 1,
            "hostname": "host1.example",
            "display_name": "display1",
            "url": "https://fleet.example.com/hosts/1"
        }     
    ]
}`, timestamp), requestBody[0])

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
        "resolution": "policy1 resolution",
        "platform": "darwin",
        "created_at": "0001-01-01T00:00:00Z",
        "updated_at": "0001-01-01T00:00:00Z",
        "passing_host_count": 0,
        "failing_host_count": 2,
        "host_count_updated_at": null,
		"critical": true,
		"calendar_events_enabled": false
    },
    "hosts": [
        {
            "id": 2,
            "hostname": "host2.example",
            "display_name": "display2",
            "url": "https://fleet.example.com/hosts/2"
        }     
    ]
}`, timestamp), requestBody[1])
	hosts, err := failingPolicySet.ListHosts(policyID1)
	require.NoError(t, err)
	assert.Empty(t, hosts)

	requestBody = make([]string, 0)
	numRequests = 0

	err = policies.TriggerFailingPoliciesAutomation(context.Background(), ds, kitlog.NewNopLogger(), failingPolicySet, func(pol *fleet.Policy, cfg policies.FailingPolicyAutomationConfig) error {
		serverURL, err := url.Parse(ac.ServerSettings.ServerURL)
		if err != nil {
			return err
		}
		return SendFailingPoliciesBatchedPOSTs(
			context.Background(), pol, failingPolicySet, cfg.HostBatchSize, serverURL, cfg.WebhookURL, mockClock, kitlog.NewNopLogger())
	})
	require.NoError(t, err)
	assert.Equal(t, len(requestBody), 0)
	assert.Equal(t, numRequests, 0)
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
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		if tid == teamID {
			return &fleet.Team{
				ID: teamID,
				Config: fleet.TeamConfig{
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

	now := time.Now()
	err = policies.TriggerFailingPoliciesAutomation(context.Background(), ds, kitlog.NewNopLogger(), failingPolicySet, func(pol *fleet.Policy, cfg policies.FailingPolicyAutomationConfig) error {
		serverURL, err := url.Parse(ac.ServerSettings.ServerURL)
		if err != nil {
			return err
		}
		return SendFailingPoliciesBatchedPOSTs(
			context.Background(), pol, failingPolicySet, cfg.HostBatchSize, serverURL, cfg.WebhookURL, now, kitlog.NewNopLogger())
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
        "resolution": "policy1 resolution",
        "platform": "darwin",
        "created_at": "0001-01-01T00:00:00Z",
        "updated_at": "0001-01-01T00:00:00Z",
        "passing_host_count": 0,
        "failing_host_count": 1,
        "host_count_updated_at": null,
		"critical": false,
		"calendar_events_enabled": true
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

	err = policies.TriggerFailingPoliciesAutomation(context.Background(), ds, kitlog.NewNopLogger(), failingPolicySet, func(pol *fleet.Policy, cfg policies.FailingPolicyAutomationConfig) error {
		serverURL, err := url.Parse(ac.ServerSettings.ServerURL)
		if err != nil {
			return err
		}
		return SendFailingPoliciesBatchedPOSTs(
			context.Background(), pol, failingPolicySet, cfg.HostBatchSize, serverURL, cfg.WebhookURL, now, kitlog.NewNopLogger())
	})
	require.NoError(t, err)
	assert.Empty(t, webhookBody)
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
			expRequestCount: 10,
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
			expRequestCount: 100000,
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

			err = SendFailingPoliciesBatchedPOSTs(
				context.Background(),
				p,
				failingPolicySet,
				tc.batchSize,
				serverURL,
				webhookURL,
				now,
				kitlog.NewNopLogger(),
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
