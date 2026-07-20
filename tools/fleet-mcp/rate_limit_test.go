package main

import (
	"net/http"
	"testing"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		xff        string
		want       string
	}{
		{
			name:       "public peer ignores XFF",
			remoteAddr: "203.0.113.5:51234",
			xff:        "10.0.0.1, 10.0.0.2",
			want:       "203.0.113.5",
		},
		{
			name:       "loopback peer no XFF returns peer",
			remoteAddr: "127.0.0.1:51234",
			xff:        "",
			want:       "127.0.0.1",
		},
		{
			name:       "loopback peer with valid XFF returns first",
			remoteAddr: "127.0.0.1:51234",
			xff:        "203.0.113.10, 10.0.0.1",
			want:       "203.0.113.10",
		},
		{
			name:       "private peer trims surrounding whitespace",
			remoteAddr: "10.0.0.5:443",
			xff:        " 203.0.113.10 ",
			want:       "203.0.113.10",
		},
		{
			name:       "loopback peer falls back when XFF token is not an IP",
			remoteAddr: "127.0.0.1:51234",
			xff:        "obfuscated",
			want:       "127.0.0.1",
		},
		{
			name:       "loopback peer falls back when XFF empty after trim",
			remoteAddr: "127.0.0.1:51234",
			xff:        "  ",
			want:       "127.0.0.1",
		},
		{
			name:       "loopback peer canonicalizes IPv6 XFF",
			remoteAddr: "127.0.0.1:51234",
			xff:        "2001:DB8::1",
			want:       "2001:db8::1",
		},
		{
			name:       "peer without port still resolves",
			remoteAddr: "203.0.113.5",
			xff:        "10.0.0.1",
			want:       "203.0.113.5",
		},
		{
			name:       "link-local peer trusted, accepts XFF",
			remoteAddr: "169.254.1.1:8080",
			xff:        "203.0.113.99",
			want:       "203.0.113.99",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &http.Request{
				RemoteAddr: tc.remoteAddr,
				Header:     http.Header{},
			}
			if tc.xff != "" {
				r.Header.Set("X-Forwarded-For", tc.xff)
			}
			if got := clientIP(r); got != tc.want {
				t.Errorf("clientIP() = %q, want %q", got, tc.want)
			}
		})
	}
}
