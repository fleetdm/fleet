package kitserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestLogin(t *testing.T) {
	ds, _ := datastore.New("mock", "")
	svc, _ := NewService(testConfig(ds))
	createTestUsers(t, ds)

	r := http.NewServeMux()
	r.Handle("/api/logout", logout(svc, kitlog.NewNopLogger()))
	r.Handle("/api/login", login(svc, kitlog.NewNopLogger()))
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

		var loginRequest = struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}{
			Username: tt.username,
			Password: tt.password,
		}
		j, err := json.Marshal(&loginRequest)
		if err != nil {
			t.Fatal(err)
		}
		requestBody := &nopCloser{bytes.NewBuffer(j)}
		resp, err := http.Post(server.URL+"/api/login", "application/json", requestBody)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := resp.StatusCode, tt.status; have != want {
			t.Errorf("have %d, want %d", have, want)
		}

		var jsn = struct {
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

		// ensure that a non-empty cookie was in-fact set
		cookie := resp.Header.Get("Set-Cookie")
		assert.NotEmpty(t, cookie)

		// ensure that a session was created for our test user and stored
		sessions, err := ds.FindAllSessionsForUser(testUser.ID)
		assert.Nil(t, err)
		assert.Len(t, sessions, 1)

		// ensure the session key is not blank
		assert.NotEqual(t, "", sessions[0].Key)

		// test logout
		req, _ := http.NewRequest("GET", server.URL+"/api/logout", nil)
		req.Header.Set("Cookie", cookie)
		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := resp.StatusCode, http.StatusOK; have != want {
			t.Errorf("have %d, want %d", have, want)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := string(body), "index"; have != want {
			t.Errorf("have %q, want %q", have, want)
		}
		// ensure that our user's session was deleted from the store
		sessions, err = ds.FindAllSessionsForUser(testUser.ID)
		assert.Len(t, sessions, 0)
	}

	var unauthenticated = []struct {
		method   string
		endpoint string
		bodyType string
	}{
		{
			method:   "POST",
			endpoint: "/login",
			bodyType: "application/x-www-form-urlencoded",
		},
		// @groob TODO need to test logout with a cookie set
		// {
		// 	method:   "GET",
		// 	endpoint: "/logout",
		// },
	}

	for _, tt := range unauthenticated {
		req, _ := http.NewRequest(tt.method, server.URL+tt.endpoint, nil)
		req.Header.Set("Content-Type", tt.bodyType)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := resp.StatusCode, http.StatusOK; have != want {
			t.Errorf("have %d, want %d", have, want)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if have, want := string(body), "index"; have != want {
			t.Errorf("have %q, want %q", have, want)
		}
	}
}

func createTestUsers(t *testing.T, ds kolide.Datastore) {
	svc := svcWithNoValidation(testConfig(ds))
	ctx := context.Background()
	for _, tt := range testUsers {
		payload := kolide.UserPayload{
			Username:           tt.Username,
			Password:           tt.Password,
			Email:              tt.Email,
			Admin:              tt.Admin,
			NeedsPasswordReset: tt.NeedsPasswordReset,
		}
		_, err := svc.NewUser(ctx, payload)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func svcWithNoValidation(config ServiceConfig) kolide.Service {
	var svc kolide.Service
	svc = service{
		ds:          config.Datastore,
		logger:      config.Logger,
		saltKeySize: config.SaltKeySize,
		bcryptCost:  config.BcryptCost,
		jwtKey:      config.JWTKey,
		cookieName:  config.SessionCookieName,
	}

	return svc
}

func testConfig(ds kolide.Datastore) ServiceConfig {
	return ServiceConfig{
		Datastore:         ds,
		Logger:            kitlog.NewNopLogger(),
		BcryptCost:        12,
		SaltKeySize:       24,
		SessionCookieName: "KolideSession",
	}
}

// an io.ReadCloser for new request body
type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }
