package externalsvc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	zendesk "github.com/nukosuke/go-zendesk/zendesk"
	"github.com/stretchr/testify/require"
)

func TestZendesk(t *testing.T) {
	var countCalls int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		countCalls++

		if r.URL.Path != "/api/v2/tickets.json" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch _, t, _ := r.BasicAuth(); t {
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
		_, err := w.Write([]byte(`{"ticket": {"id": 35436}}`))
		require.NoError(t, err)
	}))
	defer srv.Close()

	testCmt := &zendesk.TicketComment{Body: "test comment"}

	t.Run("failure", func(t *testing.T) {
		countCalls = 0

		client, err := NewZendeskTestClient(&ZendeskOptions{
			URL:      srv.URL,
			Email:    "fail",
			APIToken: "fail",
		})
		require.NoError(t, err)
		_, err = client.CreateZendeskTicket(context.Background(), &zendesk.Ticket{
			Subject:     "test",
			Description: "test",
			Comment:     testCmt,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "500: Internal Server Error")
		require.Equal(t, 6, countCalls)
	})

	t.Run("retry-after-small", func(t *testing.T) {
		countCalls = 0

		client, err := NewZendeskTestClient(&ZendeskOptions{
			URL:      srv.URL,
			Email:    "retrysmall",
			APIToken: "retrysmall",
		})
		require.NoError(t, err)

		start := time.Now()
		_, err = client.CreateZendeskTicket(context.Background(), &zendesk.Ticket{
			Subject:     "test",
			Description: "test",
			Comment:     testCmt,
		})
		require.NoError(t, err)
		require.Equal(t, 2, countCalls) // original + retry
		require.GreaterOrEqual(t, time.Since(start), time.Second)
	})

	t.Run("retry-after-too-big", func(t *testing.T) {
		countCalls = 0

		client, err := NewZendeskTestClient(&ZendeskOptions{
			URL:      srv.URL,
			Email:    "retrybig",
			APIToken: "retrybig",
		})
		require.NoError(t, err)

		_, err = client.CreateZendeskTicket(context.Background(), &zendesk.Ticket{
			Subject:     "test",
			Description: "test",
			Comment:     testCmt,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "429: Too Many Requests")
		require.Equal(t, 1, countCalls) // original only, no retry
	})

	t.Run("success", func(t *testing.T) {
		countCalls = 0
		client, err := NewZendeskTestClient(&ZendeskOptions{
			URL:      srv.URL,
			Email:    "ok",
			APIToken: "ok",
		})
		require.NoError(t, err)
		tkt, err := client.CreateZendeskTicket(context.Background(), &zendesk.Ticket{
			Subject:     "test",
			Description: "test",
			Comment:     testCmt,
		})
		require.NoError(t, err)
		require.NotZero(t, tkt.ID)
		require.Equal(t, 1, countCalls)
	})
}
