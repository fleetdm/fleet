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

	jira "github.com/andygrunwald/go-jira"
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
				{EnableSoftwareVulnerabilities: true, EnableFailingPolicies: true},
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
		NewClientFunc: func(opts *externalsvc.JiraOptions) (JiraClient, error) {
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

	t.Run("same vulnerability on multiple software only queue one job", func(t *testing.T) {
		var count int
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			count++
			return job, nil
		}
		vulns := []fleet.SoftwareVulnerability{{
			CVE:        "CVE-1234-5678",
			SoftwareID: 1,
		}, {
			CVE:        "CVE-1234-5678",
			SoftwareID: 2,
		}, {
			CVE:        "CVE-1234-5678",
			SoftwareID: 2,
		}, {
			CVE:        "CVE-1234-5678",
			SoftwareID: 3,
		}}

		err := QueueJiraVulnJobs(ctx, ds, logger, vulns)
		require.NoError(t, err)
		require.True(t, ds.NewJobFuncInvoked)
		require.Equal(t, 1, count)
	})

	t.Run("success", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}
		err := QueueJiraVulnJobs(ctx, ds, logger, []fleet.SoftwareVulnerability{{CVE: "CVE-1234-5678"}})
		require.NoError(t, err)
		require.True(t, ds.NewJobFuncInvoked)
	})

	t.Run("failure", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return nil, io.EOF
		}
		err := QueueJiraVulnJobs(ctx, ds, logger, []fleet.SoftwareVulnerability{{CVE: "CVE-1234-5678"}})
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
		ds.NewJobFuncInvoked = false
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
		ds.NewJobFuncInvoked = false
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
		ds.NewJobFuncInvoked = false
	})

	t.Run("no host", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}
		err := QueueJiraFailingPolicyJob(ctx, ds, logger,
			&fleet.Policy{PolicyData: fleet.PolicyData{ID: 1, Name: "p1"}}, []fleet.PolicySetHost{})
		require.NoError(t, err)
		require.False(t, ds.NewJobFuncInvoked)
		ds.NewJobFuncInvoked = false
	})
}

type mockJiraClient struct {
	opts externalsvc.JiraOptions
}

func (c *mockJiraClient) CreateJiraIssue(ctx context.Context, issue *jira.Issue) (*jira.Issue, error) {
	return &jira.Issue{}, nil
}

func (c *mockJiraClient) JiraConfigMatches(opts *externalsvc.JiraOptions) bool {
	return c.opts == *opts
}

func TestJiraRunClientUpdate(t *testing.T) {
	// test creation of client when config changes between 2 uses, and when integration is disabled.
	ds := new(mock.Store)

	var globalCount int
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		// failing policies is globally enabled
		globalCount++
		return &fleet.AppConfig{Integrations: fleet.Integrations{
			Jira: []*fleet.JiraIntegration{
				{ProjectKey: "0", EnableFailingPolicies: true},
				{ProjectKey: "1", EnableFailingPolicies: false}, // the team integration will use the project keys 1-3
				{ProjectKey: "2", EnableFailingPolicies: false},
				{ProjectKey: "3", EnableFailingPolicies: false},
			},
		}}, nil
	}

	teamCfg := &fleet.Team{
		ID: 123,
		Config: fleet.TeamConfig{
			Integrations: fleet.TeamIntegrations{
				Jira: []*fleet.TeamJiraIntegration{
					{ProjectKey: "1", EnableFailingPolicies: true},
				},
			},
		},
	}

	var teamCount int
	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		teamCount++

		if tid != 123 {
			return nil, errors.New("unexpected team id")
		}

		curCfg := *teamCfg

		jira0 := *teamCfg.Config.Integrations.Jira[0]
		// failing policies is enabled for team 123 the first time
		if jira0.ProjectKey == "1" {
			// the second time we change the project key
			jira0.ProjectKey = "2"
			teamCfg.Config.Integrations.Jira = []*fleet.TeamJiraIntegration{&jira0}
		} else if jira0.ProjectKey == "2" {
			// the third time we disable it altogether
			jira0.ProjectKey = "3"
			jira0.EnableFailingPolicies = false
			teamCfg.Config.Integrations.Jira = []*fleet.TeamJiraIntegration{&jira0}
		}
		return &curCfg, nil
	}

	var projectKeys []string
	jiraJob := &Jira{
		FleetURL:  "http://example.com",
		Datastore: ds,
		Log:       kitlog.NewNopLogger(),
		NewClientFunc: func(opts *externalsvc.JiraOptions) (JiraClient, error) {
			// keep track of project keys received in calls to NewClientFunc
			projectKeys = append(projectKeys, opts.ProjectKey)
			return &mockJiraClient{opts: *opts}, nil
		},
	}

	// run it globally - it is enabled and will not change
	err := jiraJob.Run(context.Background(), json.RawMessage(`{"failing_policy":{"policy_id": 1, "policy_name": "test-policy", "hosts": []}}`))
	require.NoError(t, err)

	// run it for team 123 a first time
	err = jiraJob.Run(context.Background(), json.RawMessage(`{"failing_policy":{"policy_id": 2, "policy_name": "test-policy-2", "team_id": 123, "hosts": []}}`))
	require.NoError(t, err)

	// run it globally again - it will reuse the cached client
	err = jiraJob.Run(context.Background(), json.RawMessage(`{"failing_policy":{"policy_id": 1, "policy_name": "test-policy", "hosts": []}}`))
	require.NoError(t, err)

	// run it for team 123 a second time
	err = jiraJob.Run(context.Background(), json.RawMessage(`{"failing_policy":{"policy_id": 2, "policy_name": "test-policy-2", "team_id": 123, "hosts": []}}`))
	require.NoError(t, err)

	// run it for team 123 a third time, this time integration is disabled
	err = jiraJob.Run(context.Background(), json.RawMessage(`{"failing_policy":{"policy_id": 2, "policy_name": "test-policy-2", "team_id": 123, "hosts": []}}`))
	require.NoError(t, err)

	// it should've created 3 clients - the global one, and the first 2 calls with team 123
	require.Equal(t, []string{"0", "1", "2"}, projectKeys)
	require.Equal(t, 5, globalCount) // app config is requested every time
	require.Equal(t, 3, teamCount)
}
