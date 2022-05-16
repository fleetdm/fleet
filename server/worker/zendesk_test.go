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

	var expectedSubject string
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
		require.Contains(t, string(body), expectedSubject)
		require.Contains(t, string(body), `"group_id":123`)

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
		err = zendesk.Run(context.Background(), json.RawMessage(`{"cve":"CVE-1234-5678"}`))
		require.NoError(t, err)
	})

	t.Run("failing policy", func(t *testing.T) {
		expectedSubject = `"subject":"test-policy policy failed on 1 host(s)"`
		err = zendesk.Run(context.Background(), json.RawMessage(`{"failing_policy":{"policy_id": 1, "policy_name": "test-policy", "hosts": [{"id": 123, "hostname": "host-123"}]}}`))
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
