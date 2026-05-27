package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFakeVPPServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", server.URL, t)
	t.Cleanup(server.Close)
}

func newTestServiceWithDS(ds fleet.Datastore) *Service {
	return &Service{ds: ds}
}

func TestEnsureVPPClientUser_NewUser(t *testing.T) {
	const (
		hostID         = uint(7)
		managedAppleID = "user@example.com"
	)
	tokenDB := &fleet.VPPTokenDB{ID: 42, Token: "valid-token"}
	host := &fleet.Host{ID: hostID}

	var registerCalls int
	setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
		registerCalls++
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/registerVPPUserSrv", r.URL.Path)
		// v1 puts the token in the body, not the Authorization header.
		assert.Empty(t, r.Header.Get("Authorization"))

		var got struct {
			SToken            string `json:"sToken"`
			ClientUserIDStr   string `json:"clientUserIdStr"`
			ManagedAppleIDStr string `json:"managedAppleIDStr"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		assert.Equal(t, "valid-token", got.SToken)
		assert.NotEmpty(t, got.ClientUserIDStr)
		assert.Equal(t, managedAppleID, got.ManagedAppleIDStr)

		_, _ = fmt.Fprintf(w, `{
			"status": 0,
			"user": {
				"userId": 98765,
				"status": "Registered",
				"clientUserIdStr": %q,
				"managedAppleIDStr": %q
			}
		}`, got.ClientUserIDStr, managedAppleID)
	})

	ds := new(mock.Store)
	ds.GetHostManagedAppleIDFunc = func(_ context.Context, id uint) (string, error) {
		require.Equal(t, hostID, id)
		return managedAppleID, nil
	}
	ds.GetVPPClientUserFunc = func(_ context.Context, tokenID uint, mAppleID string) (*fleet.VPPClientUser, error) {
		require.Equal(t, tokenDB.ID, tokenID)
		require.Equal(t, managedAppleID, mAppleID)
		return nil, &notFoundError{}
	}
	var insertedRow *fleet.VPPClientUser
	ds.InsertVPPClientUserFunc = func(_ context.Context, row *fleet.VPPClientUser) error {
		insertedRow = row
		return nil
	}

	svc := newTestServiceWithDS(ds)
	clientUserID, err := svc.ensureVPPClientUser(context.Background(), host, tokenDB)
	require.NoError(t, err)
	require.NotEmpty(t, clientUserID)
	require.Equal(t, 1, registerCalls)

	require.NotNil(t, insertedRow)
	require.Equal(t, tokenDB.ID, insertedRow.VPPTokenID)
	require.Equal(t, managedAppleID, insertedRow.ManagedAppleID)
	require.Equal(t, clientUserID, insertedRow.ClientUserID)
	require.Equal(t, fleet.VPPClientUserStatusRegistered, insertedRow.Status)
	require.NotNil(t, insertedRow.AppleUserID)
	require.Equal(t, "98765", *insertedRow.AppleUserID)
}

func TestEnsureVPPClientUser_ExistingRegisteredUser(t *testing.T) {
	const managedAppleID = "user@example.com"
	tokenDB := &fleet.VPPTokenDB{ID: 1, Token: "tok"}
	host := &fleet.Host{ID: 1}

	// Apple must NOT be called on cache hit.
	setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("Apple VPP register-user must not be called when a registered row exists; got %s %s", r.Method, r.URL.Path)
	})

	ds := new(mock.Store)
	ds.GetHostManagedAppleIDFunc = func(_ context.Context, _ uint) (string, error) {
		return managedAppleID, nil
	}
	ds.GetVPPClientUserFunc = func(_ context.Context, _ uint, _ string) (*fleet.VPPClientUser, error) {
		return &fleet.VPPClientUser{
			VPPTokenID:     tokenDB.ID,
			ManagedAppleID: managedAppleID,
			ClientUserID:   "cached-uuid",
			Status:         fleet.VPPClientUserStatusRegistered,
		}, nil
	}

	svc := newTestServiceWithDS(ds)
	clientUserID, err := svc.ensureVPPClientUser(context.Background(), host, tokenDB)
	require.NoError(t, err)
	require.Equal(t, "cached-uuid", clientUserID)
	require.False(t, ds.InsertVPPClientUserFuncInvoked)
}

// A non-registered cache row (typically 'pending' from the legacy v2 async
// flow) must NOT be treated as a fresh registration target: Apple enforces
// uniqueness on (location, managedAppleId), so registering with a new UUID
// would collide. Instead, look the user up on Apple's side and resync.
func TestEnsureVPPClientUser_PendingRowAppleHasUser(t *testing.T) {
	const (
		managedAppleID = "user@example.com"
		appleClientID  = "apple-side-uuid"
	)
	tokenDB := &fleet.VPPTokenDB{ID: 1, Token: "tok"}
	host := &fleet.Host{ID: 1}

	setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/users" {
			assert.Equal(t, managedAppleID, r.URL.Query().Get("managedAppleId"))
			_, _ = fmt.Fprintf(w, `{"users":[{"clientUserId":%q,"status":"Registered"}]}`, appleClientID)
			return
		}
		t.Fatalf("unexpected Apple call %s %s — pending row + Apple-side user should resync without re-registering", r.Method, r.URL.Path)
	})

	ds := new(mock.Store)
	ds.GetHostManagedAppleIDFunc = func(_ context.Context, _ uint) (string, error) {
		return managedAppleID, nil
	}
	ds.GetVPPClientUserFunc = func(_ context.Context, _ uint, _ string) (*fleet.VPPClientUser, error) {
		return &fleet.VPPClientUser{
			VPPTokenID:     tokenDB.ID,
			ManagedAppleID: managedAppleID,
			ClientUserID:   "stale-pending-uuid",
			Status:         fleet.VPPClientUserStatusPending,
		}, nil
	}
	var insertedRow *fleet.VPPClientUser
	ds.InsertVPPClientUserFunc = func(_ context.Context, row *fleet.VPPClientUser) error {
		insertedRow = row
		return nil
	}

	svc := newTestServiceWithDS(ds)
	clientUserID, err := svc.ensureVPPClientUser(context.Background(), host, tokenDB)
	require.NoError(t, err)
	require.Equal(t, appleClientID, clientUserID)
	require.NotNil(t, insertedRow)
	require.Equal(t, appleClientID, insertedRow.ClientUserID)
	require.Equal(t, fleet.VPPClientUserStatusRegistered, insertedRow.Status)
}

// Pending row + Apple has no user (only retired entries, or fully cleared) —
// safe to mint a fresh registration via the v1 endpoint.
func TestEnsureVPPClientUser_PendingRowAppleHasNoUser(t *testing.T) {
	const managedAppleID = "user@example.com"
	tokenDB := &fleet.VPPTokenDB{ID: 1, Token: "tok"}
	host := &fleet.Host{ID: 1}

	var registerCalls int
	setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/users":
			_, _ = fmt.Fprint(w, `{"users":[]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/registerVPPUserSrv":
			registerCalls++
			var got struct {
				ClientUserIDStr   string `json:"clientUserIdStr"`
				ManagedAppleIDStr string `json:"managedAppleIDStr"`
			}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
			_, _ = fmt.Fprintf(w, `{"status":0,"user":{"userId":1234,"status":"Registered","clientUserIdStr":%q,"managedAppleIDStr":%q}}`,
				got.ClientUserIDStr, managedAppleID)
		default:
			t.Fatalf("unexpected Apple call %s %s", r.Method, r.URL.Path)
		}
	})

	ds := new(mock.Store)
	ds.GetHostManagedAppleIDFunc = func(_ context.Context, _ uint) (string, error) {
		return managedAppleID, nil
	}
	ds.GetVPPClientUserFunc = func(_ context.Context, _ uint, _ string) (*fleet.VPPClientUser, error) {
		return &fleet.VPPClientUser{
			VPPTokenID:     tokenDB.ID,
			ManagedAppleID: managedAppleID,
			ClientUserID:   "stale-pending-uuid",
			Status:         fleet.VPPClientUserStatusPending,
		}, nil
	}
	ds.InsertVPPClientUserFunc = func(_ context.Context, _ *fleet.VPPClientUser) error {
		return nil
	}

	svc := newTestServiceWithDS(ds)
	clientUserID, err := svc.ensureVPPClientUser(context.Background(), host, tokenDB)
	require.NoError(t, err)
	require.NotEmpty(t, clientUserID)
	require.NotEqual(t, "stale-pending-uuid", clientUserID)
	require.Equal(t, 1, registerCalls)
}

func TestEnsureVPPClientUser_AppleErrorSurfacesAndSkipsInsert(t *testing.T) {
	const managedAppleID = "missing@example.com"
	tokenDB := &fleet.VPPTokenDB{ID: 1, Token: "tok"}
	host := &fleet.Host{ID: 1}

	// v1 reports application-level errors synchronously — no Apple-side user
	// exists, so we should surface the error and skip the DB write entirely.
	setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{
			"status": -1,
			"errorNumber": 9637,
			"errorMessage": "Managed Apple ID not found"
		}`)
	})

	ds := new(mock.Store)
	ds.GetHostManagedAppleIDFunc = func(_ context.Context, _ uint) (string, error) {
		return managedAppleID, nil
	}
	ds.GetVPPClientUserFunc = func(_ context.Context, _ uint, _ string) (*fleet.VPPClientUser, error) {
		return nil, &notFoundError{}
	}
	ds.InsertVPPClientUserFunc = func(_ context.Context, _ *fleet.VPPClientUser) error {
		t.Fatal("InsertVPPClientUser must not be called when v1 register-user returns an error")
		return nil
	}

	svc := newTestServiceWithDS(ds)
	_, err := svc.ensureVPPClientUser(context.Background(), host, tokenDB)
	require.Error(t, err)
	require.Contains(t, err.Error(), "9637")
	require.False(t, ds.InsertVPPClientUserFuncInvoked)
}

func TestEnsureVPPClientUser_MissingManagedAppleID(t *testing.T) {
	tokenDB := &fleet.VPPTokenDB{ID: 1, Token: "tok"}
	host := &fleet.Host{ID: 1}

	// Apple must not be called.
	setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("Apple VPP create-users must not be called when Managed Apple ID is missing; got %s %s", r.Method, r.URL.Path)
	})

	ds := new(mock.Store)
	ds.GetHostManagedAppleIDFunc = func(_ context.Context, _ uint) (string, error) {
		return "", nil
	}

	svc := newTestServiceWithDS(ds)
	_, err := svc.ensureVPPClientUser(context.Background(), host, tokenDB)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Managed Apple ID")

	// User-facing message surface — important since this is shown to admins.
	var ume *fleet.UserMessageError
	require.ErrorAs(t, err, &ume)
	require.Equal(t, http.StatusUnprocessableEntity, ume.StatusCode())

	require.False(t, ds.GetVPPClientUserFuncInvoked)
	require.False(t, ds.InsertVPPClientUserFuncInvoked)
}
