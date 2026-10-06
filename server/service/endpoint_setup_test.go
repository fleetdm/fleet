package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type setupStarterLibraryCall struct {
	serverURL string
	token     string
}

func newSetupTestServer(t *testing.T, ds fleet.Datastore) (*httptest.Server, func() []setupStarterLibraryCall) {
	svc, baseCtx := newTestService(t, ds, nil, nil)

	var mu sync.Mutex
	var calls []setupStarterLibraryCall
	applyStarterLibrary := func(ctx context.Context, serverURL, token string) error {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, setupStarterLibraryCall{serverURL: serverURL, token: token})
		return nil
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	h := WithSetup(svc, slog.New(slog.DiscardHandler), applyStarterLibrary, next)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(baseCtx))
	}))
	t.Cleanup(srv.Close)

	return srv, func() []setupStarterLibraryCall {
		mu.Lock()
		defer mu.Unlock()
		return append([]setupStarterLibraryCall(nil), calls...)
	}
}

func doSetupRequest(srvURL, email, orgName, serverURL string) (*http.Response, error) {
	body, err := json.Marshal(map[string]any{
		"admin": map[string]any{
			"name":     "Admin",
			"email":    email,
			"password": "p4ssw0rd.123456",
		},
		"org_info":   map[string]any{"org_name": orgName},
		"server_url": serverURL,
	})
	if err != nil {
		return nil, err
	}
	return http.Post(srvURL+"/api/v1/setup", "application/json", bytes.NewReader(body))
}

func postSetup(t *testing.T, srv *httptest.Server, email, orgName, serverURL string) (int, *fleet.User) {
	resp, err := doSetupRequest(srv.URL, email, orgName, serverURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	var out struct {
		Admin *fleet.User `json:"admin"`
	}
	if resp.StatusCode == http.StatusOK {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	}
	return resp.StatusCode, out.Admin
}

func TestSetupNotRearmedByStaleReplica(t *testing.T) {
	opts := &testing_utils.DatastoreTestOptions{DummyReplica: true}
	ds := mysqltest.CreateMySQLDSWithOptions(t, opts)
	ctx := ctxdb.RequirePrimary(t.Context(), true)

	srv, starterCalls := newSetupTestServer(t, ds)

	code, admin := postSetup(t, srv, "admin@example.com", "Legit Org", "https://fleet.example.com")
	require.Equal(t, http.StatusOK, code)
	require.NotNil(t, admin)

	secretsAfterSetup, err := ds.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, secretsAfterSetup, 1)

	// The replica has not caught up with the users insert, so reads from it
	// still see an empty users table.
	code, admin = postSetup(t, srv, "attacker@example.com", "Attacker Org", "https://attacker.example.com")
	assert.NotEqual(t, http.StatusOK, code)
	assert.Nil(t, admin)

	users, err := ds.ListUsers(ctx, fleet.UserListOptions{})
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "admin@example.com", users[0].Email)

	appCfg, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Legit Org", appCfg.OrgInfo.OrgName)
	assert.Equal(t, "https://fleet.example.com", appCfg.ServerSettings.ServerURL)

	secrets, err := ds.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, secretsAfterSetup[0].Secret, secrets[0].Secret)

	for _, c := range starterCalls() {
		assert.NotEqual(t, "https://attacker.example.com", c.serverURL)
	}
}

func TestSetupConcurrentRequestsCreateSingleAdmin(t *testing.T) {
	ds := mysqltest.CreateMySQLDS(t)
	ctx := t.Context()

	srv, _ := newSetupTestServer(t, ds)

	const n = 10
	var wg sync.WaitGroup
	codes := make([]int, n)
	errs := make([]error, n)
	for i := range n {
		wg.Go(func() {
			resp, err := doSetupRequest(srv.URL, fmt.Sprintf("admin%d@example.com", i), fmt.Sprintf("Org %d", i), "https://fleet.example.com")
			if err != nil {
				errs[i] = err
				return
			}
			resp.Body.Close()
			codes[i] = resp.StatusCode
		})
	}
	wg.Wait()
	for _, err := range errs {
		require.NoError(t, err)
	}

	var succeeded int
	for _, c := range codes {
		if c == http.StatusOK {
			succeeded++
			continue
		}
		// 404 means the request arrived after setup closed and never reached the setup router.
		assert.Contains(t, []int{http.StatusConflict, http.StatusNotFound}, c)
	}
	assert.Equal(t, 1, succeeded)

	users, err := ds.ListUsers(ctx, fleet.UserListOptions{})
	require.NoError(t, err)
	require.Len(t, users, 1)

	appCfg, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	var winner int
	_, err = fmt.Sscanf(users[0].Email, "admin%d@example.com", &winner)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("Org %d", winner), appCfg.OrgInfo.OrgName)

	secrets, err := ds.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, secrets, 1)
}

func TestSetupNotRearmedAfterUsersRemoved(t *testing.T) {
	ds := mysqltest.CreateMySQLDS(t)
	ctx := t.Context()

	srv, _ := newSetupTestServer(t, ds)

	code, _ := postSetup(t, srv, "admin@example.com", "Legit Org", "https://fleet.example.com")
	require.Equal(t, http.StatusOK, code)

	// Any request after setup completes latches the setup router closed.
	resp, err := http.Get(srv.URL + "/api/latest/fleet/version")
	require.NoError(t, err)
	resp.Body.Close()

	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM users`)
		return err
	})

	code, _ = postSetup(t, srv, "attacker@example.com", "Attacker Org", "https://attacker.example.com")
	assert.NotEqual(t, http.StatusOK, code)

	appCfg, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Legit Org", appCfg.OrgInfo.OrgName)
}

func TestSetupInvalidServerURLKeepsSetupOpen(t *testing.T) {
	ds := mysqltest.CreateMySQLDS(t)
	ctx := t.Context()

	srv, _ := newSetupTestServer(t, ds)

	code, admin := postSetup(t, srv, "admin@example.com", "Legit Org", "not-a-url")
	assert.Equal(t, http.StatusUnprocessableEntity, code)
	assert.Nil(t, admin)

	users, err := ds.ListUsers(ctx, fleet.UserListOptions{})
	require.NoError(t, err)
	require.Empty(t, users)

	code, admin = postSetup(t, srv, "admin@example.com", "Legit Org", "https://fleet.example.com")
	require.Equal(t, http.StatusOK, code)
	require.NotNil(t, admin)
}
