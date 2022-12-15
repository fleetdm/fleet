package externalsvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
			return
		case "retrysmall":
			if countCalls == 1 {
				w.Header().Add("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
		case "retrybig":
			if countCalls == 1 {
				w.Header().Add("Retry-After", "12345")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
		}

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
        }
      `))
		require.NoError(t, err)
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
		_, err = client.CreateJiraIssue(context.Background(), &jira.Issue{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "Status code: 500")
		require.Equal(t, 6, countCalls)
	})

	t.Run("retry-after-small", func(t *testing.T) {
		countCalls = 0

		client, err := NewJiraClient(&JiraOptions{
			BaseURL:           srv.URL,
			BasicAuthUsername: "retrysmall",
			BasicAuthPassword: "retrysmall",
		})
		require.NoError(t, err)

		start := time.Now()
		_, err = client.CreateJiraIssue(context.Background(), &jira.Issue{})
		require.NoError(t, err)
		require.Equal(t, 2, countCalls) // original + retry
		require.GreaterOrEqual(t, time.Since(start), time.Second)
	})

	t.Run("retry-after-too-big", func(t *testing.T) {
		countCalls = 0

		client, err := NewJiraClient(&JiraOptions{
			BaseURL:           srv.URL,
			BasicAuthUsername: "retrybig",
			BasicAuthPassword: "retrybig",
		})
		require.NoError(t, err)

		_, err = client.CreateJiraIssue(context.Background(), &jira.Issue{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "Status code: 429")
		require.Equal(t, 1, countCalls) // original only, no retry
	})

	t.Run("success", func(t *testing.T) {
		countCalls = 0

		client, err := NewJiraClient(&JiraOptions{
			BaseURL:           srv.URL,
			BasicAuthUsername: "ok",
			BasicAuthPassword: "ok",
		})
		require.NoError(t, err)
		iss, err := client.CreateJiraIssue(context.Background(), &jira.Issue{
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
