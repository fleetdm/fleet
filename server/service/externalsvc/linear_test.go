package externalsvc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLinearGetTeam(t *testing.T) {
	var gotAuth string
	var gotBody struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))

		switch gotAuth {
		case "bad-key":
			w.WriteHeader(http.StatusUnauthorized)
			_, err := w.Write([]byte(`{"errors":[{"message":"Authentication required"}]}`))
			require.NoError(t, err)
		case "gql-error":
			_, err := w.Write([]byte(`{"errors":[{"message":"Entity not found"},{"message":"second"}]}`))
			require.NoError(t, err)
		case "no-team":
			_, err := w.Write([]byte(`{"data":{"team":null}}`))
			require.NoError(t, err)
		case "server-error":
			w.WriteHeader(http.StatusInternalServerError)
		case "bad-json":
			_, err := w.Write([]byte(`not json`))
			require.NoError(t, err)
		default:
			_, err := w.Write([]byte(`{"data":{"team":{"id":"team-123"}}}`))
			require.NoError(t, err)
		}
	}))
	defer srv.Close()

	newClient := func(key string) *Linear {
		return NewLinearClient(&LinearOptions{APIKey: key, TeamID: "team-123", BaseURL: srv.URL})
	}

	t.Run("success", func(t *testing.T) {
		team, err := newClient("lin_api_test").GetTeam(context.Background())
		require.NoError(t, err)
		require.Equal(t, "team-123", team.ID)
		require.Equal(t, "lin_api_test", gotAuth, "key must be sent without a Bearer prefix")
		require.Contains(t, gotBody.Query, "team(id: $id)")
		require.Equal(t, "team-123", gotBody.Variables["id"])
	})

	t.Run("unauthorized", func(t *testing.T) {
		_, err := newClient("bad-key").GetTeam(context.Background())
		require.ErrorContains(t, err, "status 401")
		require.ErrorContains(t, err, "Authentication required")
		require.NotContains(t, err.Error(), "bad-key")
	})

	t.Run("graphql error", func(t *testing.T) {
		_, err := newClient("gql-error").GetTeam(context.Background())
		require.ErrorContains(t, err, "Entity not found; second")
	})

	t.Run("team not found", func(t *testing.T) {
		_, err := newClient("no-team").GetTeam(context.Background())
		require.ErrorContains(t, err, "team not found")
	})

	t.Run("server error", func(t *testing.T) {
		_, err := newClient("server-error").GetTeam(context.Background())
		require.ErrorContains(t, err, "unexpected status 500")
	})

	t.Run("invalid json", func(t *testing.T) {
		_, err := newClient("bad-json").GetTeam(context.Background())
		require.ErrorContains(t, err, "decode response")
	})
}
