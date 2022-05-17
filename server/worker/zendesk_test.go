package worker

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

func TestZendeskRun(t *testing.T) {
	ds := new(mock.Store)
	ds.HostsByCVEFunc = func(ctx context.Context, cve string) ([]*fleet.HostShort, error) {
		return []*fleet.HostShort{
			{
				ID:       1,
				Hostname: "test",
			},
		}, nil
	}

	var expectedSubject, expectedDescription, expectedNotInDescription string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(501)
			return
		}
		if r.URL.Path != "/api/v2/tickets.json" {
			w.WriteHeader(502)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		if expectedSubject != "" {
			require.Contains(t, string(body), expectedSubject)
		}
		if expectedDescription != "" {
			require.Contains(t, string(body), expectedDescription)
		}
		if expectedNotInDescription != "" {
			require.NotContains(t, string(body), expectedNotInDescription)
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	client, err := externalsvc.NewZendeskTestClient(&externalsvc.ZendeskOptions{URL: srv.URL, GroupID: int64(123)})
	require.NoError(t, err)

	zendesk := &Zendesk{
		FleetURL:      "https://fleetdm.com",
		Datastore:     ds,
		Log:           kitlog.NewNopLogger(),
		ZendeskClient: client,
	}

	t.Run("vuln", func(t *testing.T) {
		expectedSubject = `"subject":"Vulnerability CVE-1234-5678 detected on 1 host(s)"`
		expectedDescription = `"group_id":123`
		expectedNotInDescription = ""
		err = zendesk.Run(context.Background(), json.RawMessage(`{"cve":"CVE-1234-5678"}`))
		require.NoError(t, err)
	})

	t.Run("failing global policy", func(t *testing.T) {
		expectedSubject = `"subject":"test-policy policy failed on 1 host(s)"`
		expectedDescription = "\\u0026policy_id=1\\u0026policy_response=failing" // ampersand gets rendered as \u0026 in json string
		expectedNotInDescription = "\\u0026team_id="
		err = zendesk.Run(context.Background(), json.RawMessage(`{"failing_policy":{"policy_id": 1, "policy_name": "test-policy", "hosts": [{"id": 123, "hostname": "host-123"}]}}`))
		require.NoError(t, err)
	})

	t.Run("failing team policy", func(t *testing.T) {
		expectedSubject = `"subject":"test-policy-2 policy failed on 2 host(s)"`
		expectedDescription = "\\u0026team_id=123\\u0026policy_id=2\\u0026policy_response=failing" // ampersand gets rendered as \u0026 in json string
		expectedNotInDescription = ""
		err = zendesk.Run(context.Background(), json.RawMessage(`{"failing_policy":{"policy_id": 2, "policy_name": "test-policy-2", "team_id": 123, "hosts": [{"id": 1, "hostname": "host-1"}, {"id": 2, "hostname": "host-2"}]}}`))
		require.NoError(t, err)
	})
}

func TestZendeskQueueVulnJobs(t *testing.T) {
	ds := new(mock.Store)
	ctx := context.Background()
	logger := kitlog.NewNopLogger()

	t.Run("success", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}
		err := QueueZendeskVulnJobs(ctx, ds, logger, map[string][]string{"CVE-1234-5678": nil})
		require.NoError(t, err)
		require.True(t, ds.NewJobFuncInvoked)
	})

	t.Run("failure", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return nil, io.EOF
		}
		err := QueueZendeskVulnJobs(ctx, ds, logger, map[string][]string{"CVE-1234-5678": nil})
		require.Error(t, err)
		require.ErrorIs(t, err, io.EOF)
		require.True(t, ds.NewJobFuncInvoked)
	})
}

func TestZendeskQueueFailingPolicyJob(t *testing.T) {
	ds := new(mock.Store)
	ctx := context.Background()
	logger := kitlog.NewNopLogger()

	t.Run("success global", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			require.NotContains(t, string(*job.Args), `"team_id"`)
			return job, nil
		}
		err := QueueZendeskFailingPolicyJob(ctx, ds, logger,
			&fleet.Policy{PolicyData: fleet.PolicyData{ID: 1, Name: "p1"}}, []fleet.PolicySetHost{{ID: 1, Hostname: "h1"}})
		require.NoError(t, err)
		require.True(t, ds.NewJobFuncInvoked)
	})

	t.Run("success team", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			require.Contains(t, string(*job.Args), `"team_id"`)
			return job, nil
		}
		err := QueueZendeskFailingPolicyJob(ctx, ds, logger,
			&fleet.Policy{PolicyData: fleet.PolicyData{ID: 1, Name: "p1", TeamID: ptr.Uint(2)}}, []fleet.PolicySetHost{{ID: 1, Hostname: "h1"}})
		require.NoError(t, err)
		require.True(t, ds.NewJobFuncInvoked)
	})

	t.Run("failure", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return nil, io.EOF
		}
		err := QueueZendeskFailingPolicyJob(ctx, ds, logger,
			&fleet.Policy{PolicyData: fleet.PolicyData{ID: 1, Name: "p1"}}, []fleet.PolicySetHost{{ID: 1, Hostname: "h1"}})
		require.Error(t, err)
		require.ErrorIs(t, err, io.EOF)
		require.True(t, ds.NewJobFuncInvoked)
	})
}
