package scim

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoogleWorkspaceExclusionMiddleware(t *testing.T) {
	newDS := func(gwConfigured bool, appConfigErr error) *mock.Store {
		ds := new(mock.Store)
		ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
			if appConfigErr != nil {
				return nil, appConfigErr
			}
			ac := &fleet.AppConfig{}
			if gwConfigured {
				ac.Integrations.GoogleWorkspace = []*fleet.GoogleWorkspaceIntegration{{Domain: "example.com"}}
			}
			return ac, nil
		}
		return ds
	}

	newReq := func() *http.Request {
		return httptest.NewRequest(http.MethodPost, "/Users", nil)
	}

	t.Run("blocks SCIM when google workspace configured", func(t *testing.T) {
		var nextCalled bool
		next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true })
		h := GoogleWorkspaceExclusionMiddleware(newDS(true, nil), slog.New(slog.DiscardHandler), next)

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, newReq())

		assert.False(t, nextCalled, "SCIM handler must not run when google workspace is configured")
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "Google Workspace")
	})

	t.Run("passes through when google workspace not configured", func(t *testing.T) {
		var nextCalled bool
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		})
		h := GoogleWorkspaceExclusionMiddleware(newDS(false, nil), slog.New(slog.DiscardHandler), next)

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, newReq())

		assert.True(t, nextCalled)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("fails open when app config cannot be read", func(t *testing.T) {
		var nextCalled bool
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		})
		h := GoogleWorkspaceExclusionMiddleware(newDS(false, assert.AnError), slog.New(slog.DiscardHandler), next)

		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, newReq())

		require.True(t, nextCalled, "must fall back to normal SCIM handling on config error")
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
