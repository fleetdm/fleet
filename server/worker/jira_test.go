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

func TestJiraRun(t *testing.T) {
	ds := new(mock.Store)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(501)
			return
		}
		if r.URL.Path != "/rest/api/2/issue" {
			w.WriteHeader(502)
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), `"summary":"CVE-1234-test detected on hosts"`)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`
{
  "id": "10000",
  "key": "ED-24",
  "self": "https://your-domain.atlassian.net/rest/api/2/issue/10000",
  "transition": {
    "status": 200,
    "errorCollection": {
      "errorMessages": [],
      "errors": {}
    }
  }
}`))
	}))
	defer srv.Close()

	client, err := externalsvc.NewJiraClient(&externalsvc.JiraOptions{BaseURL: srv.URL})
	require.NoError(t, err)

	jira := &Jira{
		FleetURL:   "http://example.com",
		Datastore:  ds,
		Log:        kitlog.NewNopLogger(),
		JiraClient: client,
	}
	err = jira.Run(context.Background(), json.RawMessage(`{"cve":"CVE-1234-test","cpes":[]}`))
	require.NoError(t, err)
}

func TestJiraQueueJobs(t *testing.T) {
	ds := new(mock.Store)
	ctx := context.Background()
	logger := kitlog.NewNopLogger()

	t.Run("success", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}
		err := QueueJiraJobs(ctx, ds, logger, map[string][]string{"CVE-1234-test": {"a", "b"}})
		require.NoError(t, err)
		require.True(t, ds.NewJobFuncInvoked)
	})

	t.Run("failure", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return nil, io.EOF
		}
		err := QueueJiraJobs(ctx, ds, logger, map[string][]string{"CVE-1234-test": {"a", "b"}})
		require.Error(t, err)
		require.ErrorIs(t, err, io.EOF)
		require.True(t, ds.NewJobFuncInvoked)
	})
}
