package fleethttp

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	cases := []struct {
		name         string
		opts         []ClientOpt
		nilTransport bool
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
			if c.nilTransport {
				assert.Nil(t, cli.Transport)
			} else {
				assert.NotNil(t, cli.Transport)
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
