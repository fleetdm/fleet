package worker

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

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
