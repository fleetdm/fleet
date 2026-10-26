package main

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSHA = "09f249a06e7e3289930bf6d05f38fb562f748ebf"

func TestResolveCommit(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		want    githubCommit
		wantErr string
	}{
		{
			name: "ok, subject is first line",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/repos/apple/device-management/commits/release", r.URL.Path)
				_, _ = w.Write([]byte(`{"sha": "` + testSHA + `", "commit": {"message": "Release-v27.0\n\nMore detail."}, "files": [{"filename": "x"}]}`))
			},
			want: githubCommit{SHA: testSHA, Subject: "Release-v27.0"},
		},
		{
			name: "rate limited 403",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", "1792479600")
				w.WriteHeader(http.StatusForbidden)
			},
			wantErr: "GitHub rate limit hit (status 403, resets at unix time 1792479600)",
		},
		{
			name:    "rate limited 429",
			handler: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTooManyRequests) },
			wantErr: "GitHub rate limit hit (status 429",
		},
		{
			name:    "plain 403 is not a rate limit",
			handler: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) },
			wantErr: "unexpected status 403",
		},
		{
			name:    "server error",
			handler: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			wantErr: "unexpected status 500",
		},
		{
			name: "bad sha",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"sha": "nope", "commit": {"message": "x"}}`))
			},
			wantErr: `unexpected commit SHA "nope"`,
		},
		{
			name:    "invalid json",
			handler: func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`not json`)) },
			wantErr: "decode response",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(c.handler)
			defer srv.Close()

			got, err := resolveCommit(t.Context(), slog.New(slog.DiscardHandler), fleethttp.NewClient(), srv.URL, appleRepo, appleBranch)
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}
