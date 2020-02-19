package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kolide/fleet/server/config"
	"github.com/kolide/fleet/server/datastore/inmem"
	"github.com/kolide/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	ds, _ := inmem.New(config.TestConfig())
	svc, _ := newTestService(ds, nil)
	users := createTestUsers(t, ds)
	logger := kitlog.NewLogfmtLogger(os.Stdout)

	opts := []kithttp.ServerOption{
		kithttp.ServerBefore(
			setRequestsContexts(svc, "CHANGEME"),
		),
		kithttp.ServerErrorLogger(logger),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
		),
	}
	r := mux.NewRouter()
	ke := MakeKolideServerEndpoints(svc, "CHANGEME", "")
	kh := makeKolideKitHandlers(ke, opts)
	attachKolideAPIRoutes(r, kh)
	r.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "index")
	}))

	server := httptest.NewServer(r)
	var loginTests = []struct {
		username string
		status   int
		password string
	}{
		{
			username: "admin1",
			password: testUsers["admin1"].PlaintextPassword,
			status:   http.StatusOK,
		},
		{
			username: "user1",
			password: testUsers["user1"].PlaintextPassword,
			status:   http.StatusOK,
		},
		{
			username: "nosuchuser",
			password: "nosuchuser",
			status:   http.StatusUnauthorized,
		},
		{
			username: "admin1",
			password: "badpassword",
			status:   http.StatusUnauthorized,
		},
	}

	for _, tt := range loginTests {
		var shouldBeAdmin bool
		if u, ok := testUsers[tt.username]; ok {
			shouldBeAdmin = u.IsAdmin
		}

		// test sessions
		testUser := users[tt.username]

		params := loginRequest{
			Username: tt.username,
			Password: tt.password,
		}
		j, err := json.Marshal(&params)
		assert.Nil(t, err)

		requestBody := &nopCloser{bytes.NewBuffer(j)}
		resp, err := http.Post(server.URL+"/api/v1/kolide/login", "application/json", requestBody)
		require.Nil(t, err)
		assert.Equal(t, tt.status, resp.StatusCode)

		var jsn = struct {
			User  *kolide.User        `json:"user"`
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
		assert.Equal(t, shouldBeAdmin, jsn.User.Admin)
		assert.Equal(t, tt.username, jsn.User.Username)

		// ensure that a session was created for our test user and stored
		sessions, err := ds.ListSessionsForUser(testUser.ID)
		assert.Nil(t, err)
		assert.Len(t, sessions, 1)

		// ensure the session key is not blank
		assert.NotEqual(t, "", sessions[0].Key)

		// test logout
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/kolide/logout", nil)
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

// an io.ReadCloser for new request body
type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

// helper to convert a bool pointer false
func falseIfNil(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
