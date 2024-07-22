package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	jira "github.com/andygrunwald/go-jira"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestJiraRun(t *testing.T) {
	ds := new(mock.Store)
	ds.HostsByCVEFunc = func(ctx context.Context, cve string) ([]fleet.HostVulnerabilitySummary, error) {
		return []fleet.HostVulnerabilitySummary{
			{
				ID:       1,
				Hostname: "test",
				SoftwareInstalledPaths: []string{
					"/some/path/1",
					"/some/path/2",
				},
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

	var expectedSummary, expectedNotInDescription string
	var expectedDescription []string
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
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		if expectedSummary != "" {
			require.Contains(t, string(body), expectedSummary)
		}
		if len(expectedDescription) != 0 {
			for _, s := range expectedDescription {
				require.Contains(t, string(body), s)
			}
		}
		if expectedNotInDescription != "" {
			fmt.Println(string(body))
			require.NotContains(t, string(body), expectedNotInDescription)
		}

		w.WriteHeader(http.StatusCreated)
		_, err = w.Write([]byte(`
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
		require.NoError(t, err)
	}))
	defer srv.Close()

	client, err := externalsvc.NewJiraClient(&externalsvc.JiraOptions{BaseURL: srv.URL})
	require.NoError(t, err)

	cases := []struct {
		desc                     string
		licenseTier              string
		payload                  string
		expectedSummary          string
		expectedDescription      []string
		expectedNotInDescription string
	}{
		{
			"vuln free",
			fleet.TierFree,
			`{"vulnerability":{"cve":"CVE-1234-5678"}}`,
			`"summary":"Vulnerability CVE-1234-5678 detected on 1 host(s)"`,
			[]string{
				"Affected hosts:",
				"https://fleetdm.com/hosts/1",
				"** /some/path/1",
				"** /some/path/2",
			},
			"Probability of exploit",
		},
		{
			"vuln with scores free",
			fleet.TierFree,
			`{"vulnerability":{"cve":"CVE-1234-5678","epss_probability":3.4,"cvss_score":50,"cisa_known_exploit":true}}`,
			`"summary":"Vulnerability CVE-1234-5678 detected on 1 host(s)"`,
			[]string{
				"Affected hosts:",
				"https://fleetdm.com/hosts/1",
				"** /some/path/1",
				"** /some/path/2",
			},
			"Probability of exploit",
		},
		{
			"failing global policy",
			fleet.TierFree,
			`{"failing_policy":{"policy_id": 1, "policy_name": "test-policy", "hosts": []}}`,
			`"summary":"test-policy policy failed on 0 host(s)"`,
			[]string{"\\u0026policy_id=1\\u0026policy_response=failing"},
			"\\u0026team_id=",
		},
		{
			"failing team policy",
			fleet.TierPremium,
			`{"failing_policy":{"policy_id": 2, "policy_name": "test-policy-2", "team_id": 123, "hosts": [{"id": 1, "hostname": "test-1"}, {"id": 2, "hostname": "test-2"}]}}`,
			`"summary":"test-policy-2 policy failed on 2 host(s)"`,
			[]string{"\\u0026team_id=123\\u0026policy_id=2\\u0026policy_response=failing"},
			"",
		},
		{
			"vuln premium",
			fleet.TierPremium,
			`{"vulnerability":{"cve":"CVE-1234-5678"}}`,
			`"summary":"Vulnerability CVE-1234-5678 detected on 1 host(s)"`,
			[]string{
				"Affected hosts:",
				"https://fleetdm.com/hosts/1",
				"** /some/path/1",
				"** /some/path/2",
			},
			"Probability of exploit",
		},
		{
			"vuln with scores premium",
			fleet.TierPremium,
			`{"vulnerability":{"cve":"CVE-1234-5678","epss_probability":3.4,"cvss_score":50,"cisa_known_exploit":true}}`,
			`"summary":"Vulnerability CVE-1234-5678 detected on 1 host(s)"`,
			[]string{
				"Affected hosts:",
				"https://fleetdm.com/hosts/1",
				"** /some/path/1",
				"** /some/path/2",
			},
			"",
		},
		{
			"vuln with published date",
			fleet.TierPremium,
			`{"vulnerability":{"cve":"CVE-1234-5678","cve_published":"2012-04-23T18:25:43.511Z","epss_probability":3.4,"cvss_score":50,"cisa_known_exploit":true}}`,
			`"summary":"Vulnerability CVE-1234-5678 detected on 1 host(s)"`,
			[]string{
				"Affected hosts:",
				"https://fleetdm.com/hosts/1",
				"** /some/path/1",
				"** /some/path/2",
				"Published (reported by [NVD|https://nvd.nist.gov/]): 2012-04-23",
			},
			"",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			jira := &Jira{
				FleetURL:  "https://fleetdm.com",
				Datastore: ds,
				Log:       kitlog.NewNopLogger(),
				NewClientFunc: func(opts *externalsvc.JiraOptions) (JiraClient, error) {
					return client, nil
				},
			}

			expectedSummary = c.expectedSummary
			expectedDescription = c.expectedDescription
			expectedNotInDescription = c.expectedNotInDescription
			err = jira.Run(license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: c.licenseTier}), json.RawMessage(c.payload))
			require.NoError(t, err)
		})
	}
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
		meta := make(map[string]fleet.CVEMeta, len(vulns))
		for _, v := range vulns {
			meta[v.CVE] = fleet.CVEMeta{CVE: v.CVE}
		}

		err := QueueJiraVulnJobs(ctx, ds, logger, vulns, meta)
		require.NoError(t, err)
		require.True(t, ds.NewJobFuncInvoked)
		require.Equal(t, 1, count)
	})

	t.Run("success", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}
		theCVE := "CVE-1234-5678"
		meta := map[string]fleet.CVEMeta{
			theCVE: {CVE: theCVE},
		}
		err := QueueJiraVulnJobs(ctx, ds, logger, []fleet.SoftwareVulnerability{{CVE: theCVE}}, meta)
		require.NoError(t, err)
		require.True(t, ds.NewJobFuncInvoked)
	})

	t.Run("failure", func(t *testing.T) {
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return nil, io.EOF
		}
		theCVE := "CVE-1234-5678"
		meta := map[string]fleet.CVEMeta{
			theCVE: {CVE: theCVE},
		}
		err := QueueJiraVulnJobs(ctx, ds, logger, []fleet.SoftwareVulnerability{{CVE: theCVE}}, meta)
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
	opts   externalsvc.JiraOptions
	issues []jira.Issue
}

func (c *mockJiraClient) CreateJiraIssue(ctx context.Context, issue *jira.Issue) (*jira.Issue, error) {
	c.issues = append(c.issues, *issue)
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
	var clients []*mockJiraClient
	jiraJob := &Jira{
		FleetURL:  "http://example.com",
		Datastore: ds,
		Log:       kitlog.NewNopLogger(),
		NewClientFunc: func(opts *externalsvc.JiraOptions) (JiraClient, error) {
			// keep track of project keys received in calls to NewClientFunc
			projectKeys = append(projectKeys, opts.ProjectKey)
			client := &mockJiraClient{opts: *opts}
			clients = append(clients, client)
			return client, nil
		},
	}

	ctx := license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierFree})
	// run it globally - it is enabled and will not change
	err := jiraJob.Run(ctx, json.RawMessage(`{"failing_policy":{"policy_id": 1, "policy_name": "test-policy", "hosts": []}}`))
	require.NoError(t, err)

	// run it for team 123 a first time
	err = jiraJob.Run(ctx, json.RawMessage(`{"failing_policy":{"policy_id": 2, "policy_name": "test-policy-2", "team_id": 123, "hosts": []}}`))
	require.NoError(t, err)

	// run it globally again - it will reuse the cached client
	err = jiraJob.Run(ctx, json.RawMessage(`{"failing_policy":{"policy_id": 1, "policy_name": "test-policy", "hosts": [], "policy_critical": true}}`))
	require.NoError(t, err)

	// run it for team 123 a second time
	err = jiraJob.Run(ctx, json.RawMessage(`{"failing_policy":{"policy_id": 2, "policy_name": "test-policy-2", "team_id": 123, "hosts": []}}`))
	require.NoError(t, err)

	// run it for team 123 a third time, this time integration is disabled
	err = jiraJob.Run(ctx, json.RawMessage(`{"failing_policy":{"policy_id": 2, "policy_name": "test-policy-2", "team_id": 123, "hosts": []}}`))
	require.NoError(t, err)

	// it should've created 3 clients - the global one, and the first 2 calls with team 123
	require.Equal(t, []string{"0", "1", "2"}, projectKeys)
	require.Equal(t, 5, globalCount) // app config is requested every time
	require.Equal(t, 3, teamCount)
	require.Len(t, clients, 3)

	require.Len(t, clients[0].issues, 2)
	require.NotContains(t, clients[0].issues[0].Fields.Description, "Critical")
	require.Contains(t, clients[0].issues[1].Fields.Description, "Critical")

	require.Len(t, clients[1].issues, 1)
	require.NotContains(t, clients[1].issues[0].Fields.Description, "Critical")

	require.Len(t, clients[2].issues, 1)
	require.NotContains(t, clients[2].issues[0].Fields.Description, "Critical")
}
