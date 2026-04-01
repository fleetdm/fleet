package fleethttp

import (
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/oauth2"
)

func TestClient(t *testing.T) {
	cases := []struct {
		name         string
		opts         []ClientOpt
		defaultInner bool
		nilRedirect  bool
		timeout      time.Duration
	}{
		{"default", nil, true, true, 0},
		{"timeout", []ClientOpt{WithTimeout(time.Second)}, true, true, time.Second},
		{"nofollow", []ClientOpt{WithFollowRedir(false)}, true, false, 0},
		{"tlsconfig", []ClientOpt{WithTLSClientConfig(&tls.Config{})}, false, true, 0},
		{"combined", []ClientOpt{
			WithTLSClientConfig(&tls.Config{}),
			WithTimeout(time.Second),
			WithFollowRedir(false),
		}, false, false, time.Second},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cli := NewClient(c.opts...)
			require.IsType(t, &otelhttp.Transport{}, cli.Transport, "outer transport should be otelhttp")
			// Inspect the inner (base) transport wrapped by otelhttp via unsafe since the rt field is unexported.
			rtField := reflect.ValueOf(cli.Transport).Elem().FieldByName("rt")
			inner := *(*http.RoundTripper)(unsafe.Pointer(rtField.UnsafeAddr())) //nolint:gosec
			if c.defaultInner {
				assert.Equal(t, http.DefaultTransport, inner, "inner transport should be http.DefaultTransport")
			} else {
				assert.IsType(t, &http.Transport{}, inner, "inner transport should be a custom *http.Transport") //nolint:gocritic
			}
			if c.nilRedirect {
				assert.Nil(t, cli.CheckRedirect)
			} else {
				assert.NotNil(t, cli.CheckRedirect)
			}
			assert.Equal(t, c.timeout, cli.Timeout)
		})
	}
}

func TestTransport(t *testing.T) {
	defaultTLSConf := http.DefaultTransport.(*http.Transport).TLSClientConfig

	cases := []struct {
		name       string
		opts       []TransportOpt
		defaultTLS bool
	}{
		{"default", nil, true},
		{"tlsconf", []TransportOpt{WithTLSConfig(&tls.Config{})}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tr := NewTransport(c.opts...)
			if c.defaultTLS {
				assert.Equal(t, defaultTLSConf, tr.TLSClientConfig)
			} else {
				assert.NotEqual(t, defaultTLSConf, tr.TLSClientConfig)
			}
			assert.NotNil(t, tr.Proxy)
			assert.NotNil(t, tr.DialContext)
		})
	}
}

func TestNewGithubClientWithToken(t *testing.T) {
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)

	cases := []struct {
		name        string
		token       string
		opts        []ClientOpt
		timeout     time.Duration
		nilRedirect bool
		customTLS   bool
		cookieJar   http.CookieJar
	}{
		{"token only", "test-token", nil, 0, true, false, nil},
		{"token with timeout", "test-token", []ClientOpt{WithTimeout(5 * time.Second)}, 5 * time.Second, true, false, nil},
		{"token with nofollow", "test-token", []ClientOpt{WithFollowRedir(false)}, 0, false, false, nil},
		{"token with tls", "test-token", []ClientOpt{WithTLSClientConfig(&tls.Config{})}, 0, true, true, nil},
		{"token with cookie jar", "test-token", []ClientOpt{WithCookieJar(jar)}, 0, true, false, jar},
		{"token combined", "test-token", []ClientOpt{
			WithTimeout(3 * time.Second),
			WithFollowRedir(false),
			WithTLSClientConfig(&tls.Config{}),
		}, 3 * time.Second, false, true, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cli := NewGithubClientWithToken(c.token, c.opts...)

			require.IsType(t, &otelhttp.Transport{}, cli.Transport, "outer transport should be otelhttp")

			rtField := reflect.ValueOf(cli.Transport).Elem().FieldByName("rt")
			inner := *(*http.RoundTripper)(unsafe.Pointer(rtField.UnsafeAddr())) //nolint:gosec
			assert.IsType(t, &oauth2.Transport{}, inner, "inner transport should be oauth2.Transport")

			assert.Equal(t, c.timeout, cli.Timeout)

			if c.nilRedirect {
				assert.Nil(t, cli.CheckRedirect)
			} else {
				assert.NotNil(t, cli.CheckRedirect)
			}

			if c.customTLS {
				oauthTr := inner.(*oauth2.Transport)
				assert.IsType(t, &http.Transport{}, oauthTr.Base, "base transport should be custom *http.Transport for TLS")
			}

			if c.cookieJar != nil {
				assert.Equal(t, c.cookieJar, cli.Jar)
			} else {
				assert.Nil(t, cli.Jar)
			}
		})
	}
}

func TestHostnamesMatch(t *testing.T) {
	tests := []struct {
		name          string
		inputA        string
		inputB        string
		expectedMatch bool
		expectError   bool
	}{
		{
			name:          "ValidHostnamesMatch",
			inputA:        "https://www.example.com/path",
			inputB:        "http://www.example.com:80",
			expectedMatch: true,
			expectError:   false,
		},
		{
			name:          "ValidHostnamesDoNotMatch",
			inputA:        "https://www.example.com",
			inputB:        "https://sub.example.com",
			expectedMatch: false,
			expectError:   false,
		},
		{
			name:          "InvalidURLA",
			inputA:        "ht tp://foo.com",
			inputB:        "https://www.example.com",
			expectedMatch: false,
			expectError:   true,
		},
		{
			name:          "InvalidURLB",
			inputA:        "https://www.example.com",
			inputB:        "ht tp://foo.com",
			expectedMatch: false,
			expectError:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matched, err := HostnamesMatch(test.inputA, test.inputB)

			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedMatch, matched)

			}
		})
	}
}
