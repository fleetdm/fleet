package entratoken

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTenantID = "afb714de-f42c-41f2-802f-65b2911dd346"
	testClientID = "d6f479f6-9b7c-459d-baa4-9146b15e93bb"
	testHostUUID = "5080D39E-D146-58C3-BD4F-1ACAA9546C59"
)

func TestRequestValidate(t *testing.T) {
	valid := Request{TenantID: testTenantID, ClientID: testClientID, RedirectURI: DefaultRedirectURI, Nonce: testHostUUID + ".abc_-"}
	require.NoError(t, valid.Validate())

	for name, mutate := range map[string]func(*Request){
		"bad tenant":       func(r *Request) { r.TenantID = "contoso.com" },
		"bad client":       func(r *Request) { r.ClientID = "x" },
		"no redirect":      func(r *Request) { r.RedirectURI = "" },
		"empty nonce":      func(r *Request) { r.Nonce = "" },
		"nonce with quote": func(r *Request) { r.Nonce = "a'b" },
		"nonce with space": func(r *Request) { r.Nonce = "a b" },
	} {
		t.Run(name, func(t *testing.T) {
			r := valid
			mutate(&r)
			require.Error(t, r.Validate())
		})
	}
}

func TestExtractCode(t *testing.T) {
	const redirect = "msauth.com.fleetdm.orbit://auth"
	cases := []struct {
		name    string
		resp    authorizeResponse
		code    string
		errPart string
	}{
		{
			name: "redirect with code",
			resp: authorizeResponse{redirectURL: redirect + "?code=abc%2Fdef&session_state=x"},
			code: "abc/def",
		},
		{
			name:    "redirect with error",
			resp:    authorizeResponse{redirectURL: redirect + "?error=login_required&error_description=AADSTS50058%3a+no+user"},
			errPart: "login_required: AADSTS50058: no user",
		},
		{
			name: "code in body",
			resp: authorizeResponse{body: []byte(`<html><script>location.href="` + redirect + `/?code=0.AWMB-x_y&amp;state=1"</script>`)},
			code: "0.AWMB-x_y",
		},
		{
			name:    "code for another redirect URI is ignored",
			resp:    authorizeResponse{body: []byte(`https://evil.example/?code=stolen`)},
			errPart: "no code",
		},
		{
			name:    "error in body",
			resp:    authorizeResponse{body: []byte(`<html>AADSTS50011: The redirect URI ...</html>`)},
			errPart: "AADSTS50011",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code, err := extractCode(&c.resp, redirect)
			if c.errPart != "" {
				require.ErrorContains(t, err, c.errPart)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.code, code)
		})
	}
}

type fakeSender struct {
	mu     sync.Mutex
	tokens []string
	err    error
}

func (f *fakeSender) SendEntraDeviceToken(token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokens = append(f.tokens, token)
	return f.err
}

func (f *fakeSender) sent() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.tokens...)
}

func newTestRunner(sender tokenSender, getToken func(context.Context, Request) (string, error)) *Runner {
	r := NewRunner(sender, testHostUUID)
	r.getToken = getToken
	return r
}

func (r *Runner) waitIdle(t *testing.T) {
	require.Eventually(t, func() bool {
		r.mu.Lock()
		defer r.mu.Unlock()
		return !r.running
	}, 5*time.Second, 10*time.Millisecond)
}

func TestRunner(t *testing.T) {
	req := &fleet.OrbitEntraDeviceTokenRequest{TenantID: testTenantID, ClientID: testClientID, Nonce: testHostUUID + ".mac"}

	t.Run("no request", func(t *testing.T) {
		sender := &fakeSender{}
		r := newTestRunner(sender, func(context.Context, Request) (string, error) {
			t.Fatal("unexpected token request")
			return "", nil
		})
		require.NoError(t, r.Run(&fleet.OrbitConfig{}))
		r.waitIdle(t)
		assert.Empty(t, sender.sent())
	})

	t.Run("reports once per interval and on request change", func(t *testing.T) {
		sender := &fakeSender{}
		var got []Request
		r := newTestRunner(sender, func(_ context.Context, rr Request) (string, error) {
			got = append(got, rr)
			return "h.p.s", nil
		})
		require.NoError(t, r.Run(&fleet.OrbitConfig{EntraDeviceTokenRequest: req}))
		r.waitIdle(t)
		require.NoError(t, r.Run(&fleet.OrbitConfig{EntraDeviceTokenRequest: req}))
		r.waitIdle(t)
		assert.Equal(t, []string{"h.p.s"}, sender.sent())
		require.Len(t, got, 1)
		assert.Equal(t, Request{TenantID: testTenantID, ClientID: testClientID, RedirectURI: DefaultRedirectURI, Nonce: req.Nonce}, got[0])

		changed := *req
		changed.TenantID = "0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d"
		require.NoError(t, r.Run(&fleet.OrbitConfig{EntraDeviceTokenRequest: &changed}))
		r.waitIdle(t)
		assert.Len(t, sender.sent(), 2)

		// due again after the interval
		r.mu.Lock()
		r.nextRun = time.Now().Add(-time.Second)
		r.mu.Unlock()
		require.NoError(t, r.Run(&fleet.OrbitConfig{EntraDeviceTokenRequest: &changed}))
		r.waitIdle(t)
		assert.Len(t, sender.sent(), 3)
	})

	t.Run("refuses nonce for another host", func(t *testing.T) {
		sender := &fakeSender{}
		r := newTestRunner(sender, func(context.Context, Request) (string, error) {
			t.Fatal("unexpected token request")
			return "", nil
		})
		other := *req
		other.Nonce = "11111111-2222-3333-4444-555555555555.mac"
		require.NoError(t, r.Run(&fleet.OrbitConfig{EntraDeviceTokenRequest: &other}))
		r.waitIdle(t)
		assert.Empty(t, sender.sent())
	})

	t.Run("failure is retried with backoff without a new config", func(t *testing.T) {
		sender := &fakeSender{}
		var mu sync.Mutex
		var attempts []time.Time
		r := newTestRunner(sender, func(context.Context, Request) (string, error) {
			mu.Lock()
			defer mu.Unlock()
			attempts = append(attempts, time.Now())
			if len(attempts) < 4 {
				return "", errors.New("AADSTS50058")
			}
			return "h.p.s", nil
		})
		r.minRetry = 20 * time.Millisecond
		r.maxRetry = 50 * time.Millisecond
		require.NoError(t, r.Run(&fleet.OrbitConfig{EntraDeviceTokenRequest: req}))
		require.Eventually(t, func() bool { return len(sender.sent()) == 1 }, 5*time.Second, 5*time.Millisecond)
		r.waitIdle(t)

		mu.Lock()
		require.Len(t, attempts, 4)
		// 20ms, 40ms, then capped at 50ms.
		for i, want := range []time.Duration{20 * time.Millisecond, 40 * time.Millisecond, 50 * time.Millisecond} {
			assert.GreaterOrEqual(t, attempts[i+1].Sub(attempts[i]), want, "retry %d", i+1)
		}
		mu.Unlock()

		r.mu.Lock()
		defer r.mu.Unlock()
		assert.Zero(t, r.failures)
		assert.WithinDuration(t, time.Now().Add(refreshInterval), r.nextRun, 5*time.Second)
	})

	t.Run("retry stops when Fleet stops requesting the token", func(t *testing.T) {
		sender := &fakeSender{}
		var calls atomic.Int32
		r := newTestRunner(sender, func(context.Context, Request) (string, error) {
			calls.Add(1)
			return "", errors.New("AADSTS50058")
		})
		r.minRetry = 50 * time.Millisecond
		require.NoError(t, r.Run(&fleet.OrbitConfig{EntraDeviceTokenRequest: req}))
		r.waitIdle(t)
		require.NoError(t, r.Run(&fleet.OrbitConfig{}))
		time.Sleep(150 * time.Millisecond)
		assert.EqualValues(t, 1, calls.Load())
	})

	t.Run("request change during a report is reported next", func(t *testing.T) {
		sender := &fakeSender{}
		release := make(chan struct{})
		var mu sync.Mutex
		var tenants []string
		r := newTestRunner(sender, func(_ context.Context, rr Request) (string, error) {
			mu.Lock()
			tenants = append(tenants, rr.TenantID)
			first := len(tenants) == 1
			mu.Unlock()
			if first {
				<-release
			}
			return "h.p.s", nil
		})
		require.NoError(t, r.Run(&fleet.OrbitConfig{EntraDeviceTokenRequest: req}))
		changed := *req
		changed.TenantID = "0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d"
		require.NoError(t, r.Run(&fleet.OrbitConfig{EntraDeviceTokenRequest: &changed}))
		close(release)
		require.Eventually(t, func() bool { return len(sender.sent()) == 2 }, 5*time.Second, 5*time.Millisecond)
		r.waitIdle(t)
		mu.Lock()
		defer mu.Unlock()
		assert.Equal(t, []string{testTenantID, changed.TenantID}, tenants)
	})
}
