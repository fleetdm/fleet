package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-kit/kit/transport"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/inmem"
	"github.com/fleetdm/fleet/v4/server/fleet"

	kitlog "github.com/go-kit/kit/log"

	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2/store/memstore"
)

func TestLogin(t *testing.T) {
	ds, users, server := setupAuthTest(t)
	var loginTests = []struct {
		email    string
		status   int
		password string
	}{
		{
			email:    "admin1@example.com",
			password: testUsers["admin1"].PlaintextPassword,
			status:   http.StatusOK,
		},
		{
			email:    "user1@example.com",
			password: testUsers["user1"].PlaintextPassword,
			status:   http.StatusOK,
		},
		{
			email:    "nosuchuser@example.com",
			password: "nosuchuser",
			status:   http.StatusUnauthorized,
		},
		{
			email:    "admin1@example.com",
			password: "badpassword",
			status:   http.StatusUnauthorized,
		},
	}

	for _, tt := range loginTests {
		// test sessions
		testUser := users[tt.email]

		params := loginRequest{
			Email:    tt.email,
			Password: tt.password,
		}
		j, err := json.Marshal(&params)
		assert.Nil(t, err)

		requestBody := &nopCloser{bytes.NewBuffer(j)}
		resp, err := http.Post(server.URL+"/api/v1/fleet/login", "application/json", requestBody)
		require.Nil(t, err)
		assert.Equal(t, tt.status, resp.StatusCode)

		var jsn = struct {
			User  *fleet.User         `json:"user"`
			Token string              `json:"token"`
			Err   []map[string]string `json:"errors,omitempty"`
		}{}
		err = json.NewDecoder(resp.Body).Decode(&jsn)
		require.Nil(t, err)

		if tt.status != http.StatusOK {
			assert.NotEqual(t, "", jsn.Err)
			continue // skip remaining tests
		}

		require.NotNil(t, jsn.User)
		assert.Equal(t, tt.email, jsn.User.Email)

		// ensure that a session was created for our test user and stored
		sessions, err := ds.ListSessionsForUser(testUser.ID)
		assert.Nil(t, err)
		assert.Len(t, sessions, 1)

		// ensure the session key is not blank
		assert.NotEqual(t, "", sessions[0].Key)

		// test logout
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/fleet/logout", nil)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jsn.Token))
		client := &http.Client{}
		resp, err = client.Do(req)
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, strconv.Itoa(tt.status))

		_, err = ioutil.ReadAll(resp.Body)
		assert.Nil(t, err)

		// ensure that our user's session was deleted from the store
		sessions, err = ds.ListSessionsForUser(testUser.ID)
		assert.Nil(t, err)
		assert.Len(t, sessions, 0)
	}
}

func setupAuthTest(t *testing.T) (*inmem.Datastore, map[string]fleet.User, *httptest.Server) {
	ds, _ := inmem.New(config.TestConfig())
	svc := newTestService(ds, nil, nil)
	users := createTestUsers(t, ds)
	logger := kitlog.NewLogfmtLogger(os.Stdout)

	opts := []kithttp.ServerOption{
		kithttp.ServerBefore(
			setRequestsContexts(svc),
		),
		kithttp.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
		),
	}
	r := mux.NewRouter()
	limitStore, _ := memstore.New(0)
	ke := MakeFleetServerEndpoints(svc, "", limitStore)
	kh := makeKitHandlers(ke, opts)
	attachFleetAPIRoutes(r, kh)
	r.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "index")
	}))

	server := httptest.NewServer(r)
	return ds, users, server
}

func TestNoHeaderErrorsDifferently(t *testing.T) {
	_, _, server := setupAuthTest(t)

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/fleet/users", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	assert.Equal(t, "Authorization header required", string(bodyBytes))

	req, _ = http.NewRequest("GET", server.URL+"/api/v1/fleet/users", nil)
	req.Header.Add("Authorization", "Bearer AAAA")
	resp, err = client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	assert.Equal(t, "Authentication required", string(bodyBytes))
}

// an io.ReadCloser for new request body
type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }
