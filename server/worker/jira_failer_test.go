package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	kitlog "github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

func TestTestJiraFailer(t *testing.T) {
	ds := new(mock.Store)
	ds.HostsByCVEFunc = func(ctx context.Context, cve string) ([]*fleet.HostShort, error) {
		return []*fleet.HostShort{{ID: 1, Hostname: "test"}}, nil
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	// create the real client, that will never fail
	client, err := externalsvc.NewJiraClient(&externalsvc.JiraOptions{BaseURL: srv.URL})
	require.NoError(t, err)

	// create the failer, that will introduced forced errors
	failer := &TestJiraFailer{
		FailCallCountModulo: 3,
		AlwaysFailCVEs:      []string{"CVE-2020-1234"},
		JiraClient:          client,
	}

	// create the Jira job with that failer-wrapped client
	jira := &Jira{
		FleetURL:   "http://example.com",
		Datastore:  ds,
		Log:        kitlog.NewNopLogger(),
		JiraClient: failer,
	}

	var failedIndices []int
	cves := []string{"CVE-2018-1234", "CVE-2019-1234", "CVE-2020-1234", "CVE-2021-1234"}
	for i := 0; i < 10; i++ {
		cve := cves[i%len(cves)]
		err := jira.Run(context.Background(), json.RawMessage(fmt.Sprintf(`{"cve":%q}`, cve)))
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
