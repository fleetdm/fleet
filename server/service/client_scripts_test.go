package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestApplyScriptsBatchErrorNamesScript(t *testing.T) {
	t.Parallel()

	// 422 response as returned by POST /api/latest/fleet/scripts/batch when
	// one script in the batch fails validation.
	newClient := func(t *testing.T, status int, body map[string]any) *Client {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(body)
		}))
		t.Cleanup(srv.Close)
		client, err := NewClient(srv.URL, true, "", "")
		require.NoError(t, err)
		client.SetToken("test-token")
		return client
	}

	scripts := []fleet.ScriptPayload{
		{Name: "ok.sh", ScriptContents: []byte("echo ok")},
		{Name: "nonexistent.sh", ScriptContents: []byte("echo $FLEET_VAR_NONEXISTENT")},
	}
	invalidBody := map[string]any{
		"message": "Validation Failed",
		"errors":  []map[string]string{{"name": "scripts[1]", "reason": "Fleet variable $FLEET_VAR_NONEXISTENT is not supported in scripts."}},
	}

	t.Run("team scripts name the failing file", func(t *testing.T) {
		client := newClient(t, http.StatusUnprocessableEntity, invalidBody)
		_, err := client.ApplyTeamScripts("Workstations", scripts, fleet.ApplySpecOptions{})
		require.ErrorContains(t, err, `script "nonexistent.sh"`)
		require.ErrorContains(t, err, "Fleet variable $FLEET_VAR_NONEXISTENT is not supported in scripts.")
		// the original error stays in the chain for status-code checks
		var scErr *StatusCodeErr
		require.ErrorAs(t, err, &scErr)
		require.Equal(t, http.StatusUnprocessableEntity, scErr.Code)
	})

	t.Run("no-team scripts name the failing file", func(t *testing.T) {
		client := newClient(t, http.StatusUnprocessableEntity, invalidBody)
		_, err := client.ApplyNoTeamScripts(scripts, fleet.ApplySpecOptions{})
		require.ErrorContains(t, err, `script "nonexistent.sh"`)
	})

	t.Run("errors without a scripts index pass through unchanged", func(t *testing.T) {
		client := newClient(t, http.StatusUnprocessableEntity, map[string]any{
			"message": "Validation Failed",
			"errors":  []map[string]string{{"name": "script", "reason": "secret not found"}},
		})
		_, err := client.ApplyTeamScripts("Workstations", scripts, fleet.ApplySpecOptions{})
		require.Error(t, err)
		require.NotContains(t, err.Error(), `script "`)
		require.ErrorContains(t, err, "secret not found")
	})
}

func TestRewrapScriptBatchIndexErr(t *testing.T) {
	t.Parallel()

	scripts := []fleet.ScriptPayload{{Name: "a.sh"}, {Name: "b.sh"}}

	statusErr := func(name string) error {
		return &StatusCodeErr{Code: http.StatusUnprocessableEntity, Body: "Validation Failed: boom", Name: name}
	}

	require.NoError(t, rewrapScriptBatchIndexErr(nil, scripts))

	plain := errors.New("boom")
	require.Equal(t, plain, rewrapScriptBatchIndexErr(plain, scripts))

	cases := []struct {
		desc     string
		name     string
		wantWrap string // empty means unchanged
	}{
		{"valid index", "scripts[1]", `script "b.sh"`},
		{"first index", "scripts[0]", `script "a.sh"`},
		{"no name", "", ""},
		{"non-script field", "team_id", ""},
		{"index out of range", "scripts[2]", ""},
		{"negative index", "scripts[-1]", ""},
		{"not a number", "scripts[x]", ""},
		{"missing closing bracket", "scripts[1", ""},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			in := statusErr(c.name)
			out := rewrapScriptBatchIndexErr(in, scripts)
			if c.wantWrap == "" {
				require.Equal(t, in, out)
			} else {
				require.ErrorContains(t, out, c.wantWrap)
				require.ErrorIs(t, out, in)
			}
		})
	}
}
