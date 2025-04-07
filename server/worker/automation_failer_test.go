package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	kitlog "github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestJiraFailer(t *testing.T) {
	ds := new(mock.Store)
	ds.HostsByCVEFunc = func(ctx context.Context, cve string) ([]fleet.HostVulnerabilitySummary, error) {
		return []fleet.HostVulnerabilitySummary{{ID: 1, Hostname: "test"}}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{Integrations: fleet.Integrations{
			Jira: []*fleet.JiraIntegration{
				{EnableSoftwareVulnerabilities: true},
			},
		}}, nil
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(`
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

	// create the real client, that will never fail
	client, err := externalsvc.NewJiraClient(&externalsvc.JiraOptions{BaseURL: srv.URL})
	require.NoError(t, err)

	// create the failer, that will introduced forced errors
	failer := &TestAutomationFailer{
		FailCallCountModulo: 3,
		AlwaysFailCVEs:      []string{"CVE-2020-1234"},
		JiraClient:          client,
	}

	// create the Jira job with that failer-wrapped client
	jira := &Jira{
		FleetURL:  "http://example.com",
		Datastore: ds,
		Log:       kitlog.NewNopLogger(),
		NewClientFunc: func(opts *externalsvc.JiraOptions) (JiraClient, error) {
			return failer, nil
		},
	}

	var failedIndices []int
	cves := []string{"CVE-2018-1234", "CVE-2019-1234", "CVE-2020-1234", "CVE-2021-1234"}
	for i := 0; i < 10; i++ {
		cve := cves[i%len(cves)]
		err := jira.Run(license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierFree}), json.RawMessage(fmt.Sprintf(`{"vulnerability":{"cve":%q}}`, cve)))
		if err != nil {
			failedIndices = append(failedIndices, i)
		}
	}

	// want indices:
	// 2: always failing CVE
	// 5: modulo
	// 6: CVE
	// 8: modulo
	require.Equal(t, []int{2, 5, 6, 8}, failedIndices)
}

func TestZendeskFailer(t *testing.T) {
	ds := new(mock.Store)
	ds.HostsByCVEFunc = func(ctx context.Context, cve string) ([]fleet.HostVulnerabilitySummary, error) {
		return []fleet.HostVulnerabilitySummary{{ID: 1, Hostname: "test"}}, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{Integrations: fleet.Integrations{
			Zendesk: []*fleet.ZendeskIntegration{
				{EnableSoftwareVulnerabilities: true},
			},
		}}, nil
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(`{"ticket": {"id": 987}}`))
		require.NoError(t, err)
	}))
	defer srv.Close()

	// create the real client, that will never fail
	client, err := externalsvc.NewZendeskTestClient(&externalsvc.ZendeskOptions{URL: srv.URL})
	require.NoError(t, err)

	// create the failer, that will introduced forced errors
	failer := &TestAutomationFailer{
		FailCallCountModulo: 3,
		AlwaysFailCVEs:      []string{"CVE-2020-1234"},
		ZendeskClient:       client,
	}

	// create the Zendesk job with that failer-wrapped client
	zendesk := &Zendesk{
		FleetURL:  "http://example.com",
		Datastore: ds,
		Log:       kitlog.NewNopLogger(),
		NewClientFunc: func(opts *externalsvc.ZendeskOptions) (ZendeskClient, error) {
			return failer, nil
		},
	}

	var failedIndices []int
	cves := []string{"CVE-2018-1234", "CVE-2019-1234", "CVE-2020-1234", "CVE-2021-1234"}
	for i := 0; i < 10; i++ {
		cve := cves[i%len(cves)]
		err := zendesk.Run(license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierFree}), json.RawMessage(fmt.Sprintf(`{"vulnerability":{"cve":%q}}`, cve)))
		if err != nil {
			failedIndices = append(failedIndices, i)
		}
	}

	// want indices:
	// 2: always failing CVE
	// 5: modulo
	// 6: CVE
	// 8: modulo
	require.Equal(t, []int{2, 5, 6, 8}, failedIndices)
}
