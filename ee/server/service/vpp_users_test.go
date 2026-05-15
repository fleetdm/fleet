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

	var createCalls int
	setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
		createCalls++
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/users/create", r.URL.Path)
		assert.Equal(t, "Bearer valid-token", r.Header.Get("Authorization"))

		var got struct {
			Users []struct {
				ClientUserId   string `json:"clientUserId"`
				ManagedAppleId string `json:"managedAppleId"`
			} `json:"users"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		assert.Len(t, got.Users, 1)
		assert.NotEmpty(t, got.Users[0].ClientUserId)
		assert.Equal(t, managedAppleID, got.Users[0].ManagedAppleId)

		_, _ = fmt.Fprintf(w, `{
			"eventId": "evt-1",
			"users": [{"userId":"apple-user-1","clientUserId":%q,"managedAppleId":%q,"status":"Registered"}]
		}`, got.Users[0].ClientUserId, managedAppleID)
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
	require.Equal(t, 1, createCalls)

	require.NotNil(t, insertedRow)
	require.Equal(t, tokenDB.ID, insertedRow.VPPTokenID)
	require.Equal(t, managedAppleID, insertedRow.ManagedAppleID)
	require.Equal(t, clientUserID, insertedRow.ClientUserID)
	require.Equal(t, fleet.VPPClientUserStatusRegistered, insertedRow.Status)
	require.NotNil(t, insertedRow.AppleUserID)
	require.Equal(t, "apple-user-1", *insertedRow.AppleUserID)
}

func TestEnsureVPPClientUser_ExistingRegisteredUser(t *testing.T) {
	const managedAppleID = "user@example.com"
	tokenDB := &fleet.VPPTokenDB{ID: 1, Token: "tok"}
	host := &fleet.Host{ID: 1}

	// Apple must NOT be called on cache hit.
	setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("Apple VPP create-users must not be called when a registered row exists; got %s %s", r.Method, r.URL.Path)
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

func TestEnsureVPPClientUser_PendingRetryReusesUUID(t *testing.T) {
	const (
		managedAppleID = "user@example.com"
		priorUUID      = "prior-uuid-1234"
	)
	tokenDB := &fleet.VPPTokenDB{ID: 1, Token: "tok"}
	host := &fleet.Host{ID: 1}

	setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
		var got struct {
			Users []struct {
				ClientUserId   string `json:"clientUserId"`
				ManagedAppleId string `json:"managedAppleId"`
			} `json:"users"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		assert.Equal(t, priorUUID, got.Users[0].ClientUserId, "retry must reuse the prior clientUserId")

		_, _ = fmt.Fprintf(w, `{
			"eventId": "evt",
			"users": [{"userId":"apple-2","clientUserId":%q,"managedAppleId":%q,"status":"Registered"}]
		}`, priorUUID, managedAppleID)
	})

	ds := new(mock.Store)
	ds.GetHostManagedAppleIDFunc = func(_ context.Context, _ uint) (string, error) {
		return managedAppleID, nil
	}
	ds.GetVPPClientUserFunc = func(_ context.Context, _ uint, _ string) (*fleet.VPPClientUser, error) {
		return &fleet.VPPClientUser{
			VPPTokenID:     tokenDB.ID,
			ManagedAppleID: managedAppleID,
			ClientUserID:   priorUUID,
			Status:         fleet.VPPClientUserStatusPending,
		}, nil
	}
	var lastInserted *fleet.VPPClientUser
	ds.InsertVPPClientUserFunc = func(_ context.Context, row *fleet.VPPClientUser) error {
		lastInserted = row
		return nil
	}

	svc := newTestServiceWithDS(ds)
	got, err := svc.ensureVPPClientUser(context.Background(), host, tokenDB)
	require.NoError(t, err)
	require.Equal(t, priorUUID, got)
	require.NotNil(t, lastInserted)
	require.Equal(t, fleet.VPPClientUserStatusRegistered, lastInserted.Status)
}

func TestEnsureVPPClientUser_PartialFailureKeepsPending(t *testing.T) {
	const managedAppleID = "user@example.com"
	tokenDB := &fleet.VPPTokenDB{ID: 1, Token: "tok"}
	host := &fleet.Host{ID: 1}

	setupFakeVPPServer(t, func(w http.ResponseWriter, r *http.Request) {
		var got struct {
			Users []struct {
				ClientUserId string `json:"clientUserId"`
			} `json:"users"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		_, _ = fmt.Fprintf(w, `{
			"eventId": "evt",
			"users": [{"clientUserId":%q,"managedAppleId":%q,"errorMessage":"Managed Apple ID not found","errorNumber":9637}]
		}`, got.Users[0].ClientUserId, managedAppleID)
	})

	ds := new(mock.Store)
	ds.GetHostManagedAppleIDFunc = func(_ context.Context, _ uint) (string, error) {
		return managedAppleID, nil
	}
	ds.GetVPPClientUserFunc = func(_ context.Context, _ uint, _ string) (*fleet.VPPClientUser, error) {
		return nil, &notFoundError{}
	}
	var inserted *fleet.VPPClientUser
	ds.InsertVPPClientUserFunc = func(_ context.Context, row *fleet.VPPClientUser) error {
		inserted = row
		return nil
	}

	svc := newTestServiceWithDS(ds)
	_, err := svc.ensureVPPClientUser(context.Background(), host, tokenDB)
	require.Error(t, err)
	require.Contains(t, err.Error(), "9637")
	require.NotNil(t, inserted)
	require.Equal(t, fleet.VPPClientUserStatusPending, inserted.Status, "partial failure must persist row as pending so retries can reuse the UUID")
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
