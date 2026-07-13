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
			return errors.New("callback error")
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

// TestSyncerCursorNotAdvancedOnCallbackErrorWithMoreToFollowSync verifies the
// same cursor-replay behaviour as TestSyncerCursorNotAdvancedOnCallbackErrorWithMoreToFollowFetch
// but for the sync phase (/devices/sync). MoreToFollow can occur on both
// fetch and sync, and the fix must hold for both.
func TestSyncerCursorNotAdvancedOnCallbackErrorWithMoreToFollowSync(t *testing.T) {
	const (
		fetchCursor = "fetch-cursor"
		syncCursor  = "sync-cursor"
	)

	syncCount := 0
	var syncCursorsReceivedByApple []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "test-token"}`))
		case "/server/devices":
			_ = json.NewEncoder(w).Encode(godep.DeviceResponse{
				Cursor:  fetchCursor,
				Devices: []godep.Device{{SerialNumber: "ABC123"}},
			})
		case "/devices/sync":
			var req struct {
				Cursor string `json:"cursor"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			syncCursorsReceivedByApple = append(syncCursorsReceivedByApple, req.Cursor)

			syncCount++
			moreToFollow := syncCount == 1 // only true on first sync call
			_ = json.NewEncoder(w).Encode(godep.DeviceResponse{
				Cursor:       syncCursor,
				MoreToFollow: moreToFollow,
				Devices:      []godep.Device{{SerialNumber: "ABC123"}},
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

	// fetch callback always succeeds; first sync callback errors, second succeeds.
	syncCallbackCount := 0
	depClient := godep.NewClient(store, nil)
	syncer := depsync.NewSyncer(depClient, "test-dep", store,
		depsync.WithCallback(func(_ context.Context, isFetch bool, _ *godep.DeviceResponse) error {
			if !isFetch {
				syncCallbackCount++
				if syncCallbackCount == 1 {
					return errors.New("callback error")
				}
			}
			return nil
		}),
	)

	err := syncer.Run(t.Context())
	require.NoError(t, err)

	// Apple should have received two sync requests.
	require.Len(t, syncCursorsReceivedByApple, 2)
	// Both requests must carry the same cursor because the first sync callback
	// errored and the cursor must not have advanced.
	require.Equal(t, syncCursorsReceivedByApple[0], syncCursorsReceivedByApple[1],
		"cursor should not advance after a sync callback error, so the same page is retried")

	// After the second sync callback succeeds the final cursor should be stored.
	require.Equal(t, syncCursor, storedCursor)
}

// TestSyncerCursorNotAdvancedOnCallbackErrorWithMoreToFollowFetch verifies that
// when MoreToFollow is true during the fetch phase and the callback errors, the
// cursor is not advanced, so the next request to Apple replays the same page
// rather than skipping it.
func TestSyncerCursorNotAdvancedOnCallbackErrorWithMoreToFollowFetch(t *testing.T) {
	const pageOneCursor = "page-one-cursor"

	fetchCount := 0
	var cursorsReceivedByApple []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "test-token"}`))
		case "/server/devices":
			var req struct {
				Cursor string `json:"cursor"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			cursorsReceivedByApple = append(cursorsReceivedByApple, req.Cursor)

			fetchCount++
			moreToFollow := fetchCount == 1 // only true on first call
			_ = json.NewEncoder(w).Encode(godep.DeviceResponse{
				Cursor:       pageOneCursor,
				MoreToFollow: moreToFollow,
				Devices:      []godep.Device{{SerialNumber: "ABC123"}},
			})
		case "/devices/sync":
			_ = json.NewEncoder(w).Encode(godep.DeviceResponse{
				Cursor:  pageOneCursor,
				Devices: []godep.Device{},
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

	callbackCount := 0
	depClient := godep.NewClient(store, nil)
	syncer := depsync.NewSyncer(depClient, "test-dep", store,
		depsync.WithCallback(func(_ context.Context, _ bool, _ *godep.DeviceResponse) error {
			callbackCount++
			if callbackCount == 1 {
				return errors.New("callback error")
			}
			return nil
		}),
	)

	err := syncer.Run(t.Context())
	require.NoError(t, err)

	// Apple should have received two fetch requests.
	require.Len(t, cursorsReceivedByApple, 2)
	// Both requests must carry the same cursor because the first callback
	// errored and the cursor must not have advanced.
	require.Equal(t, cursorsReceivedByApple[0], cursorsReceivedByApple[1],
		"cursor should not advance after a callback error, so the same page is retried")

	// After the second callback succeeds the cursor should be stored.
	require.True(t, store.StoreCursorFuncInvoked)
	require.Equal(t, pageOneCursor, storedCursor)
}

// TestSyncerCursorNotAdvancedOnAppleAPIError verifies that when Apple's API
// returns a non-cursor error (e.g. a 500), the cursor is not advanced and the
// syncer exits cleanly in run-once mode.
func TestSyncerCursorNotAdvancedOnAppleAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"auth_session_token": "test-token"}`))
		case "/server/devices":
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)

	cursorNotWritten := "cursor-not-written"
	storedCursor := cursorNotWritten
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
	// No callback — the Apple API error occurs before the callback is reached.
	syncer := depsync.NewSyncer(depClient, "test-dep", store)

	err := syncer.Run(t.Context())
	require.NoError(t, err)
	require.False(t, store.StoreCursorFuncInvoked)
	require.Equal(t, cursorNotWritten, storedCursor)
}

// TestSyncerCursorResetOnExpiredCursor verifies that when Apple returns an
// expired cursor error, the cursor is reset to empty and the syncer re-fetches
// the full device list from the beginning.
func TestSyncerCursorResetOnExpiredCursor(t *testing.T) {
	const freshCursor = "fresh-cursor"

	fetchCount := 0
	var cursorsReceivedByApple []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/session":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"auth_session_token": "test-token"}`))
		case "/server/devices":
			var req struct {
				Cursor string `json:"cursor"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			cursorsReceivedByApple = append(cursorsReceivedByApple, req.Cursor)

			fetchCount++
			if fetchCount == 1 {
				// Simulate Apple rejecting a stale cursor.
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`EXPIRED_CURSOR`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(godep.DeviceResponse{
				Cursor:  freshCursor,
				Devices: []godep.Device{{SerialNumber: "ABC123"}},
			})
		case "/devices/sync":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(godep.DeviceResponse{
				Cursor:  freshCursor,
				Devices: []godep.Device{},
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
		return "stale-cursor", time.Time{}, nil
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

	// First request sent the stale cursor, second sent empty after the reset.
	require.Len(t, cursorsReceivedByApple, 2)
	require.Equal(t, "stale-cursor", cursorsReceivedByApple[0])
	require.Empty(t, cursorsReceivedByApple[1], "cursor should be reset to empty after an expired cursor error")

	// After the re-fetch succeeds the fresh cursor should be stored.
	require.True(t, store.StoreCursorFuncInvoked)
	require.Equal(t, freshCursor, storedCursor)
}

// TestSyncerExitsOnStoreCursorError verifies that when storing the cursor
// fails, the syncer returns the error rather than silently continuing.
func TestSyncerExitsOnStoreCursorError(t *testing.T) {
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
		}
	}))
	t.Cleanup(srv.Close)

	storeCursorErr := errors.New("db connection lost")
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
	store.StoreCursorFunc = func(_ context.Context, _ string, _ string) error {
		return storeCursorErr
	}

	depClient := godep.NewClient(store, nil)
	syncer := depsync.NewSyncer(depClient, "test-dep", store,
		depsync.WithCallback(func(_ context.Context, _ bool, _ *godep.DeviceResponse) error {
			return nil
		}),
	)

	err := syncer.Run(t.Context())
	require.ErrorIs(t, err, storeCursorErr)
}
