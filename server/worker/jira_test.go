package worker

import (
	"context"
	"encoding/json"
	"errors"
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

func TestJiraRun(t *testing.T) {
	ds := new(mock.Store)
	ds.HostsByCVEFunc = func(ctx context.Context, cve string) ([]*fleet.HostShort, error) {
		return []*fleet.HostShort{
			{
				ID:       1,
				Hostname: "test",
			},
		}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{Integrations: fleet.Integrations{
			Jira: []*fleet.JiraIntegration{
				{EnableSoftwareVulnerabilities: true, TeamJiraIntegration: fleet.TeamJiraIntegration{EnableFailingPolicies: true}},
			},
		}}, nil
	}
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		if tid != 123 {
			return nil, errors.New("unexpected team id")
		}
		return &fleet.Team{
			ID: 123,
			Config: fleet.TeamConfig{
				Integrations: fleet.TeamIntegrations{
					Jira: []*fleet.TeamJiraIntegration{
						{EnableFailingPolicies: true},
					},
				},
			},
		}, nil
	}

	var expectedSummary, expectedDescription, expectedNotInDescription string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(501)
			return
		}
		if r.URL.Path != "/rest/api/2/issue" {
			w.WriteHeader(502)
			return
		}

		// the request body is the JSON payload sent to Jira, i.e. the rendered templates
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		if expectedSummary != "" {
			require.Contains(t, string(body), expectedSummary)
		}
		if expectedDescription != "" {
			require.Contains(t, string(body), expectedDescription)
		}
		if expectedNotInDescription != "" {
			require.NotContains(t, string(body), expectedNotInDescription)
		}

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
		FleetURL:  "http://example.com",
		Datastore: ds,
		Log:       kitlog.NewNopLogger(),
		NewClientFunc: func(cfg fleet.TeamJiraIntegration) (JiraClient, error) {
			return client, nil
		},
	}

	t.Run("vuln", func(t *testing.T) {
		expectedSummary = `"summary":"Vulnerability CVE-1234-5678 detected on 1 host(s)"`
		expectedDescription, expectedNotInDescription = "", ""
		err = jira.Run(context.Background(), json.RawMessage(`{"cve":"CVE-1234-5678"}`))
		require.NoError(t, err)
	})

	t.Run("failing global policy", func(t *testing.T) {
		expectedSummary = `"summary":"test-policy policy failed on 0 host(s)"`
		expectedDescription = "\\u0026policy_id=1\\u0026policy_response=failing" // ampersand gets rendered as \u0026 in json string
		expectedNotInDescription = "\\u0026team_id="
		err = jira.Run(context.Background(), json.RawMessage(`{"failing_policy":{"policy_id": 1, "policy_name": "test-policy", "hosts": []}}`))
		require.NoError(t, err)
	})

	t.Run("failing team policy", func(t *testing.T) {
		expectedSummary = `"summary":"test-policy-2 policy failed on 2 host(s)"`
		expectedDescription = "\\u0026team_id=123\\u0026policy_id=2\\u0026policy_response=failing" // ampersand gets rendered as \u0026 in json string
		expectedNotInDescription = ""
		err = jira.Run(context.Background(), json.RawMessage(`{"failing_policy":{"policy_id": 2, "policy_name": "test-policy-2", "team_id": 123, "hosts": [{"id": 1, "hostname": "test-1"}, {"id": 2, "hostname": "test-2"}]}}`))
		require.NoError(t, err)
	})
}

func TestJiraQueueVulnJobs(t *testing.T) {
	ds := new(mock.Store)
	ctx := context.Background()
	logger := kitlog.NewNopLogger()

	t.Run("success", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}
		err := QueueJiraVulnJobs(ctx, ds, logger, map[string][]string{"CVE-1234-5678": nil})
		require.NoError(t, err)
		require.True(t, ds.NewJobFuncInvoked)
	})

	t.Run("failure", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return nil, io.EOF
		}
		err := QueueJiraVulnJobs(ctx, ds, logger, map[string][]string{"CVE-1234-5678": nil})
		require.Error(t, err)
		require.ErrorIs(t, err, io.EOF)
		require.True(t, ds.NewJobFuncInvoked)
	})
}

func TestJiraQueueFailingPolicyJob(t *testing.T) {
	ds := new(mock.Store)
	ctx := context.Background()
	logger := kitlog.NewNopLogger()

	t.Run("success global", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			require.NotContains(t, string(*job.Args), `"team_id"`)
			return job, nil
		}
		err := QueueJiraFailingPolicyJob(ctx, ds, logger,
			&fleet.Policy{PolicyData: fleet.PolicyData{ID: 1, Name: "p1"}}, []fleet.PolicySetHost{{ID: 1, Hostname: "h1"}})
		require.NoError(t, err)
		require.True(t, ds.NewJobFuncInvoked)
	})

	t.Run("success team", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			require.Contains(t, string(*job.Args), `"team_id"`)
			return job, nil
		}
		err := QueueJiraFailingPolicyJob(ctx, ds, logger,
			&fleet.Policy{PolicyData: fleet.PolicyData{ID: 1, Name: "p1", TeamID: ptr.Uint(2)}}, []fleet.PolicySetHost{{ID: 1, Hostname: "h1"}})
		require.NoError(t, err)
		require.True(t, ds.NewJobFuncInvoked)
	})

	t.Run("failure", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return nil, io.EOF
		}
		err := QueueJiraFailingPolicyJob(ctx, ds, logger,
			&fleet.Policy{PolicyData: fleet.PolicyData{ID: 1, Name: "p1"}}, []fleet.PolicySetHost{{ID: 1, Hostname: "h1"}})
		require.Error(t, err)
		require.ErrorIs(t, err, io.EOF)
		require.True(t, ds.NewJobFuncInvoked)
	})
}
