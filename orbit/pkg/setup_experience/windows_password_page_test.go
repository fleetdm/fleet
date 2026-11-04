package setupexperience

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/stretchr/testify/require"
)

func TestPasswordPage(t *testing.T) {
	var created []string
	createErr := error(nil)
	p, err := newPasswordPage(10, func(pw string) error {
		if createErr != nil {
			return createErr
		}
		created = append(created, pw)
		return nil
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })

	client := fleethttp.NewClient()
	get := func(u string) (int, string) {
		resp, err := client.Get(u)
		require.NoError(t, err)
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return resp.StatusCode, string(b)
	}
	post := func(password, confirm string) string {
		resp, err := client.PostForm(p.URL(), url.Values{"password": {password}, "confirm": {confirm}})
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return string(b)
	}

	status, body := get(p.URL())
	require.Equal(t, http.StatusOK, status)
	require.Contains(t, body, "Choose your password")
	require.Contains(t, body, "At least 10 characters.")

	u, err := url.Parse(p.URL())
	require.NoError(t, err)
	status, _ = get("http://" + u.Host + "/?t=wrong")
	require.Equal(t, http.StatusNotFound, status)
	status, _ = get("http://" + u.Host + "/")
	require.Equal(t, http.StatusNotFound, status)

	// A different Host header (DNS rebinding) is rejected even with the token.
	req, err := http.NewRequest(http.MethodGet, p.URL(), nil)
	require.NoError(t, err)
	req.Host = "attacker.example:" + u.Port()
	resp, err := client.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	require.Contains(t, post("", ""), "Enter a password.")
	require.Contains(t, post("longenough1", "different11"), "The passwords don&#39;t match.")
	require.Contains(t, post("short", "short"), "Use at least 10 characters.")

	createErr = errPasswordPolicy
	require.Contains(t, post("longenough1", "longenough1"), "doesn&#39;t meet this computer&#39;s requirements")
	createErr = errors.New("NetUserAdd failed")
	require.Contains(t, post("longenough1", "longenough1"), "couldn&#39;t create your account")
	require.Empty(t, created)
	select {
	case <-p.Done():
		t.Fatal("done before the account was created")
	default:
	}

	createErr = nil
	body = post("longenough1", "longenough1")
	require.Contains(t, body, "Setting up your device")
	require.Equal(t, []string{"longenough1"}, created)
	<-p.Done()

	// Once created, the page doesn't create the account again.
	require.Contains(t, post("another-password", "another-password"), "Setting up your device")
	require.Len(t, created, 1)
	_, body = get(p.URL())
	require.True(t, strings.Contains(body, "Setting up your device"))
}
