package webhooks

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
		require.NoError(t, err)
		requestBody = string(requestBodyBytes)
	}))
	defer ts.Close()

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
	err := failingPolicySet.AddHost(policyID1, 1)
	require.NoError(t, err)
	err = failingPolicySet.AddHost(policyID1, 2)
	require.NoError(t, err)

	mockClock := time.Now()
	err = TriggerFailingPoliciesWebhook(context.Background(), ds, kitlog.NewNopLogger(), ac, failingPolicySet, mockClock)
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
            "hostname": "todo",
            "url": "https:/fleet.example.com/hosts/1"
        },
        {
            "id": 2,
            "hostname": "todo",
            "url": "https:/fleet.example.com/hosts/2"
        }
    ]
}`, timestamp), requestBody)

	requestBody = ""

	err = TriggerHostStatusWebhook(context.Background(), ds, kitlog.NewNopLogger(), ac)
	require.NoError(t, err)
	assert.Empty(t, requestBody)
}
