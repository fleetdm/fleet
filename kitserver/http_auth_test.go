package kitserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/net/context"

	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
)

func TestLogin(t *testing.T) {
	ds, _ := datastore.New("mock", "")
	svc, _ := NewService(ds)
	createTestUsers(t, svc)

	r := http.NewServeMux()
	r.Handle("/logout", logout(ds, kitlog.NewNopLogger()))
	r.Handle("/login", login(ds, kitlog.NewNopLogger()))
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
		v := url.Values{}
		{
			v.Set("username", tt.username)
			v.Set("password", tt.password)
		}
		resp, err := http.PostForm(server.URL+"/login", v)
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
		{
			method:   "GET",
			endpoint: "/logout",
		},
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

func createTestUsers(t *testing.T, svc kolide.UserService) {
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
