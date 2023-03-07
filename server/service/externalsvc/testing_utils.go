package externalsvc

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type MockOktaServer struct {
	Srv *httptest.Server

	Username     string
	UserPassword string
	ClientID     string

	mu           sync.Mutex
	clientSecret string
	customResp   func(w http.ResponseWriter)
}

func (m *MockOktaServer) ClientSecret() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.clientSecret
}

func (m *MockOktaServer) SetClientSecret(s string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clientSecret = s
}

func (m *MockOktaServer) CustomResp() func(w http.ResponseWriter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.customResp
}

func (m *MockOktaServer) SetCustomResp(f func(w http.ResponseWriter)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.customResp = f
}

func RunMockOktaServer(t *testing.T) *MockOktaServer {
	mock := &MockOktaServer{
		Username:     "test@example.com",
		UserPassword: "pwd1234",
		ClientID:     "ABC",
		clientSecret: "DEF",
	}

	mock.Srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientID, clientSecret, ok := r.BasicAuth()
		require.True(t, ok)

		err := r.ParseForm()
		require.NoError(t, err)

		require.Equal(t, "openid", r.FormValue("scope"))
		require.Equal(t, "password", r.FormValue("grant_type"))
		require.Equal(t, "POST", r.Method)

		if clientID != mock.ClientID || clientSecret != mock.ClientSecret() {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "invalid_client", "error_description": "The client secret supplied for a confidential client is invalid."}`))
			return
		}

		if r.FormValue("username") != mock.Username || r.FormValue("password") != mock.UserPassword {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "invalid_grant", "error_description": "The credentials provided were invalid."}`))
			return
		}

		if f := mock.CustomResp(); f != nil {
			f(w)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"token_type": "Bearer", "access_token": "asdf"}`))
		require.NoError(t, err)
	}))

	t.Cleanup(func() { mock.Srv.Close() })
	return mock
}
