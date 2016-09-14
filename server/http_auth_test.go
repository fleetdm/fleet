package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/kolide/kolide-ose/config"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestLogin(t *testing.T) {
	ds, _ := datastore.New("inmem", "")
	svc, _ := NewService(ds, kitlog.NewNopLogger(), config.TestConfig())
	createTestUsers(t, ds)
	logger := kitlog.NewLogfmtLogger(os.Stdout)

	opts := []kithttp.ServerOption{
		kithttp.ServerBefore(
			setViewerContext(svc, ds, "foobar", logger),
		),
		kithttp.ServerErrorLogger(logger),
		kithttp.ServerAfter(
			kithttp.SetContentType("application/json; charset=utf-8"),
		),
	}
	r := mux.NewRouter()
	attachAPIRoutes(r, context.Background(), svc, opts)
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
			password: *testUsers["admin1"].Password,
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
		p, ok := testUsers[tt.username]
		if !ok {
			p = kolide.UserPayload{
				Username: stringPtr(tt.username),
				Password: stringPtr("foobar"),
				Email:    stringPtr("admin1@example.com"),
				Admin:    boolPtr(true),
			}
		}

		// test sessions
		testUser, err := ds.User(tt.username)
		if err != nil && err != datastore.ErrNotFound {
			t.Fatal(err)
		}

		params := loginRequest{
			Username: tt.username,
			Password: tt.password,
		}
		j, err := json.Marshal(&params)
		if err != nil {
			t.Fatal(err)
		}
		requestBody := &nopCloser{bytes.NewBuffer(j)}
		resp, err := http.Post(server.URL+"/api/v1/kolide/login", "application/json", requestBody)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := resp.StatusCode, tt.status; have != want {
			t.Errorf("have %d, want %d", have, want)
		}

		var jsn = struct {
			Token              string `json:"token"`
			ID                 uint   `json:"id"`
			Username           string `json:"username"`
			Email              string `json:"email"`
			Name               string `json:"name"`
			Admin              bool   `json:"admin"`
			Enabled            bool   `json:"enabled"`
			NeedsPasswordReset bool   `json:"needs_password_reset"`
			Err                string `json:"error,omitempty"`
		}{}
		if err := json.NewDecoder(resp.Body).Decode(&jsn); err != nil {
			t.Fatal(err)
		}

		if tt.status != http.StatusOK {
			if jsn.Err == "" {
				t.Errorf("expected json error, got empty result")
			}
			continue // skip remaining tests
		}

		if have, want := jsn.Admin, falseIfNil(p.Admin); have != want {
			t.Errorf("have %v, want %v", have, want)
		}

		// ensure that a session was created for our test user and stored
		sessions, err := ds.FindAllSessionsForUser(testUser.ID)
		assert.Nil(t, err)
		assert.Len(t, sessions, 1)

		// ensure the session key is not blank
		assert.NotEqual(t, "", sessions[0].Key)

		// test logout
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/kolide/logout", nil)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jsn.Token))
		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := resp.StatusCode, http.StatusOK; have != want {
			t.Errorf("have %d, want %d", have, want)
		}
		_, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		// ensure that our user's session was deleted from the store
		sessions, err = ds.FindAllSessionsForUser(testUser.ID)
		assert.Len(t, sessions, 0)
	}
}

func createTestUsers(t *testing.T, ds kolide.Datastore) {
	svc := svcWithNoValidation(ds, kitlog.NewNopLogger())
	ctx := context.Background()
	for _, tt := range testUsers {
		payload := kolide.UserPayload{
			Username: tt.Username,
			Password: tt.Password,
			Email:    tt.Email,
			Admin:    tt.Admin,
			AdminForcedPasswordReset: tt.AdminForcedPasswordReset,
		}
		_, err := svc.NewUser(ctx, payload)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func svcWithNoValidation(ds kolide.Datastore, logger kitlog.Logger) kolide.Service {
	var svc kolide.Service
	svc = service{
		ds:     ds,
		logger: logger,
		config: config.TestConfig(),
	}

	return svc
}

// an io.ReadCloser for new request body
type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }
