package sync_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	depsync "github.com/fleetdm/fleet/v4/server/mdm/nanodep/sync"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	"github.com/stretchr/testify/require"
)

// TestSyncerCursorNotAdvancedOnCallbackError verifies that when the callback
// returns an error, the cursor is not advanced. Without this guard, a
// context-cancel mid-upsert would advance the cursor past unprocessed device
// events, dropping those devices permanently from Fleet.
func TestSyncerCursorNotAdvancedOnCallbackError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "test-token"}`))
		case "/server/devices":
			_ = json.NewEncoder(w).Encode(godep.DeviceResponse{
				Cursor:  "new-cursor",
				Devices: []godep.Device{{SerialNumber: "ABC123"}},
			})
		case "/devices/sync":
			_ = json.NewEncoder(w).Encode(godep.DeviceResponse{
				Cursor:  "new-cursor",
				Devices: []godep.Device{{SerialNumber: "ABC123"}},
			})
		}
	}))
	t.Cleanup(srv.Close)

	store := &nanodep_mock.Storage{}
	store.RetrieveConfigFunc = func(_ context.Context, _ string) (*client.Config, error) {
		return &client.Config{BaseURL: srv.URL}, nil
	}
	store.RetrieveAuthTokensFunc = func(_ context.Context, _ string) (*client.OAuth1Tokens, error) {
		return &client.OAuth1Tokens{}, nil
	}
	store.RetrieveCursorFunc = func(_ context.Context, _ string) (string, time.Time, error) {
		return "", time.Time{}, nil
	}
	cursorNotWritten := "cursor-not-written"
	storedCursor := cursorNotWritten
	store.StoreCursorFunc = func(_ context.Context, _ string, cursor string) error {
		storedCursor = cursor
		return nil
	}

	depClient := godep.NewClient(store, nil)
	syncer := depsync.NewSyncer(depClient, "test-dep", store,
		depsync.WithCallback(func(_ context.Context, _ bool, _ *godep.DeviceResponse) error {
			return errors.New("context canceled")
		}),
	)

	err := syncer.Run(t.Context())
	require.NoError(t, err)
	require.False(t, store.StoreCursorFuncInvoked)
	require.Equal(t, cursorNotWritten, storedCursor)
}

// TestSyncerCursorAdvancedOnCallbackSuccess verifies that when the callback
// succeeds, the cursor is advanced to the value returned by Apple.
func TestSyncerCursorAdvancedOnCallbackSuccess(t *testing.T) {
	const newCursor = "new-cursor"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "test-token"}`))
		case "/server/devices":
			_ = json.NewEncoder(w).Encode(godep.DeviceResponse{
				Cursor:  newCursor,
				Devices: []godep.Device{{SerialNumber: "ABC123"}},
			})
		case "/devices/sync":
			_ = json.NewEncoder(w).Encode(godep.DeviceResponse{
				Cursor:  newCursor,
				Devices: []godep.Device{{SerialNumber: "ABC123"}},
			})
		}
	}))
	t.Cleanup(srv.Close)

	var storedCursor string
	store := &nanodep_mock.Storage{}
	store.RetrieveConfigFunc = func(_ context.Context, _ string) (*client.Config, error) {
		return &client.Config{BaseURL: srv.URL}, nil
	}
	store.RetrieveAuthTokensFunc = func(_ context.Context, _ string) (*client.OAuth1Tokens, error) {
		return &client.OAuth1Tokens{}, nil
	}
	store.RetrieveCursorFunc = func(_ context.Context, _ string) (string, time.Time, error) {
		return "", time.Time{}, nil
	}
	store.StoreCursorFunc = func(_ context.Context, _ string, cursor string) error {
		storedCursor = cursor
		return nil
	}

	depClient := godep.NewClient(store, nil)
	syncer := depsync.NewSyncer(depClient, "test-dep", store,
		depsync.WithCallback(func(_ context.Context, _ bool, _ *godep.DeviceResponse) error {
			return nil
		}),
	)

	err := syncer.Run(t.Context())
	require.NoError(t, err)
	require.True(t, store.StoreCursorFuncInvoked)
	require.Equal(t, newCursor, storedCursor)
}
