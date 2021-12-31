package webhooks

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	kitlog "github.com/go-kit/kit/log"
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
				},
			}, nil
		}
		return nil, ctxerr.Wrap(ctx, sql.ErrNoRows)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestBodyBytes, err := ioutil.ReadAll(r.Body)
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

	failingPolicySet := service.NewMemFailingPolicySet()
	err := failingPolicySet.AddHost(policyID1, fleet.PolicySetHost{
		ID:       1,
		Hostname: "host1.example",
	})
	require.NoError(t, err)
	err = failingPolicySet.AddHost(policyID1, fleet.PolicySetHost{
		ID:       2,
		Hostname: "host2.example",
	})
	require.NoError(t, err)

	mockClock := time.Now()
	err = TriggerGlobalFailingPoliciesWebhook(context.Background(), ds, kitlog.NewNopLogger(), ac, failingPolicySet, mockClock)
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
        "resolution": "policy1 resolution",
        "platform": "darwin",
        "created_at": "0001-01-01T00:00:00Z",
        "updated_at": "0001-01-01T00:00:00Z",
        "passing_host_count": 0,
        "failing_host_count": 0
    },
    "hosts": [
        {
            "id": 1,
            "hostname": "host1.example",
            "url": "https://fleet.example.com/hosts/1"
        },
        {
            "id": 2,
            "hostname": "host2.example",
            "url": "https://fleet.example.com/hosts/2"
        }
    ]
}`, timestamp), requestBody)

	hosts, err := failingPolicySet.ListHosts(policyID1)
	require.NoError(t, err)
	assert.Empty(t, hosts)

	requestBody = ""

	err = TriggerGlobalFailingPoliciesWebhook(context.Background(), ds, kitlog.NewNopLogger(), ac, failingPolicySet, mockClock)
	require.NoError(t, err)
	assert.Empty(t, requestBody)
}

func TestSendBatchedPOSTs(t *testing.T) {
	allHosts := []uint{}
	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		var payload FailingPoliciesPayload
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
				ID:       uint(i + 1),
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
			err := sendFailingPoliciesBatchedPOSTs(context.Background(),
				p, failingPolicySet, postData{
					serverURL:  serverURL,
					now:        now,
					webhookURL: ts.URL,
				},
				tc.batchSize,
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

func TestFilterPolicies(t *testing.T) {
	ds := new(mock.Store)
	for _, tc := range []struct {
		name        string
		set         []uint
		cfg         []uint
		nonExisting []uint
		expIDs      []uint
	}{
		{
			name:        "one-non-configured",
			set:         []uint{1, 2, 6},
			cfg:         []uint{1, 2, 3},
			nonExisting: []uint{},
			expIDs:      []uint{1, 2},
		},
		{
			name:        "none-configured",
			set:         []uint{1, 2, 6},
			cfg:         []uint{},
			nonExisting: []uint{},
			expIDs:      []uint{},
		},
		{
			name:        "one-non-existing-and-one-non-configured",
			set:         []uint{1, 2, 6},
			cfg:         []uint{1, 2},
			nonExisting: []uint{1},
			expIDs:      []uint{2},
		},
		{
			name:        "empty-set",
			set:         []uint{},
			cfg:         []uint{1, 2},
			nonExisting: []uint{1},
			expIDs:      []uint{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			failingPoliciesSet := service.NewMemFailingPolicySet()
			for _, policyID := range tc.set {
				err := failingPoliciesSet.AddHost(policyID, fleet.PolicySetHost{ID: 1})
				require.NoError(t, err)
			}
			ds.PolicyFunc = func(ctx context.Context, id uint) (*fleet.Policy, error) {
				for _, nonID := range tc.nonExisting {
					if nonID == id {
						return nil, sql.ErrNoRows
					}
				}
				return &fleet.Policy{
					PolicyData: fleet.PolicyData{
						ID: id,
					},
				}, nil
			}
			policies, err := filterPolicies(
				context.Background(),
				ds,
				tc.cfg,
				failingPoliciesSet,
				kitlog.NewNopLogger(),
			)
			require.NoError(t, err)
			require.Len(t, policies, len(tc.expIDs))
			sets, err := failingPoliciesSet.ListSets()
			sort.Slice(sets, func(i, j int) bool {
				return sets[i] < sets[j]
			})
			sort.Slice(policies, func(i, j int) bool {
				return policies[i].ID < policies[j].ID
			})
			require.NoError(t, err)
			for i := range policies {
				require.Equal(t, tc.expIDs[i], policies[i].ID)
				require.Equal(t, sets[i], policies[i].ID)
			}
		})
	}
}
