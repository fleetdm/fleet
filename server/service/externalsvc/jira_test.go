package externalsvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andygrunwald/go-jira"
	"github.com/stretchr/testify/require"
)

func TestJira(t *testing.T) {
	var countCalls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		countCalls++

		switch usr, _, _ := r.BasicAuth(); usr {
		case "fail":
			w.WriteHeader(http.StatusInternalServerError)
		case "ok":
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
        }
      `))
		}
	}))
	defer srv.Close()

	t.Run("failure", func(t *testing.T) {
		countCalls = 0

		client, err := NewJiraClient(&JiraOptions{
			BaseURL:           srv.URL,
			BasicAuthUsername: "fail",
			BasicAuthPassword: "fail",
		})
		require.NoError(t, err)
		_, err = client.CreateIssue(context.Background(), &jira.Issue{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "Status code: 500")
		require.Equal(t, 6, countCalls)
	})

	t.Run("success", func(t *testing.T) {
		countCalls = 0

		client, err := NewJiraClient(&JiraOptions{
			BaseURL:           srv.URL,
			BasicAuthUsername: "ok",
			BasicAuthPassword: "ok",
		})
		require.NoError(t, err)
		iss, err := client.CreateIssue(context.Background(), &jira.Issue{
			Fields: &jira.IssueFields{
				Summary:     "test",
				Description: "test",
			},
		})
		require.NoError(t, err)
		require.NotZero(t, iss.ID)
		require.Equal(t, 1, countCalls)
	})
}
