package fleetctl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFleetNameForPackage(t *testing.T) {
	premium, ds := testing_utils.RunServerWithMockedDS(t, &service.TestServerOpts{
		License: &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)},
	})
	ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
		return []*fleet.Team{
			{ID: 1, Name: "Servers", Secrets: []*fleet.EnrollSecret{{Secret: "servers-secret"}}},
			{ID: 2, Name: "💻 Workstations", Secrets: []*fleet.EnrollSecret{{Secret: "old-secret"}, {Secret: "workstations-secret"}}},
		}, nil
	}
	// Tokens for an observer, whose view of enroll secrets Fleet masks, and for an expired session.
	observer := &fleet.User{ID: 9999, Name: "observer", Email: "observer@example.com", GlobalRole: new(fleet.RoleObserver)}
	adminSession, adminUser := ds.SessionByKeyFunc, ds.UserByIDFunc
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		switch key {
		case "expired-token":
			return nil, errors.New("invalid session")
		case "observer-token":
			return &fleet.Session{CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()}, AccessedAt: time.Now(), UserID: observer.ID, Key: key}, nil
		}
		return adminSession(ctx, key)
	}
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		if id == observer.ID {
			return observer, nil
		}
		return adminUser(ctx, id)
	}

	free, _ := testing_utils.RunServerWithMockedDS(t)

	// A server that never answers, as when --fleet-url points to the wrong host.
	stuck := make(chan struct{})
	hung := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-stuck }))
	t.Cleanup(func() { close(stuck); hung.Close() })

	type fleetctlContext struct{ name, address, token string }
	for _, tc := range []struct {
		name     string
		contexts []fleetctlContext
		fleetURL string
		secret   string
		want     string
		wantNote bool
	}{
		{
			name:     "fleet secret",
			contexts: []fleetctlContext{{"default", premium.URL, "admin-token"}},
			fleetURL: premium.URL, secret: "workstations-secret", want: "💻 Workstations",
		},
		{
			name:     "URL spelled differently from the context address",
			contexts: []fleetctlContext{{"default", premium.URL, "admin-token"}},
			fleetURL: strings.ToUpper(premium.URL) + "/", secret: "servers-secret", want: "Servers",
		},
		{
			name:     "global secret",
			contexts: []fleetctlContext{{"default", premium.URL, "admin-token"}},
			fleetURL: premium.URL, secret: "global-secret",
		},
		{
			name:     "role that can't read enroll secrets",
			contexts: []fleetctlContext{{"default", premium.URL, "observer-token"}},
			fleetURL: premium.URL, secret: "servers-secret",
		},
		{
			name:     "expired token prints a note",
			contexts: []fleetctlContext{{"default", premium.URL, "expired-token"}},
			fleetURL: premium.URL, secret: "servers-secret", wantNote: true,
		},
		{
			name:     "expired token in one context, valid in another",
			contexts: []fleetctlContext{{"default", premium.URL, "expired-token"}, {"other", premium.URL, "admin-token"}},
			fleetURL: premium.URL, secret: "servers-secret", want: "Servers",
		},
		{
			name:     "context for another server is not used",
			contexts: []fleetctlContext{{"default", free.URL, "admin-token"}, {"prod", premium.URL, "admin-token"}},
			fleetURL: premium.URL, secret: "servers-secret", want: "Servers",
		},
		{
			name:     "no context for the server",
			contexts: []fleetctlContext{{"default", premium.URL, "admin-token"}},
			fleetURL: "https://fleet.example.com", secret: "servers-secret",
		},
		{
			name:     "context without a token",
			contexts: []fleetctlContext{{"default", premium.URL, ""}},
			fleetURL: premium.URL, secret: "servers-secret",
		},
		{
			name:     "Fleet Free",
			contexts: []fleetctlContext{{"default", free.URL, "admin-token"}},
			fleetURL: free.URL, secret: "servers-secret",
		},
		{
			name:     "not a URL",
			contexts: []fleetctlContext{{"default", premium.URL, "admin-token"}},
			fleetURL: "dummy", secret: "dummy",
		},
		{
			name:     "server that never answers",
			contexts: []fleetctlContext{{"default", hung.URL, "admin-token"}},
			fleetURL: hung.URL, secret: "servers-secret",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var config strings.Builder
			config.WriteString("contexts:\n")
			for _, c := range tc.contexts {
				fmt.Fprintf(&config, "  %s:\n    address: %s\n    token: %q\n    tls-skip-verify: true\n", c.name, c.address, c.token)
			}
			configPath := filepath.Join(t.TempDir(), "config")
			require.NoError(t, os.WriteFile(configPath, []byte(config.String()), configFilePerms))

			oldTimeout := fleetNameLookupTimeout
			fleetNameLookupTimeout = 500 * time.Millisecond
			t.Cleanup(func() { fleetNameLookupTimeout = oldTimeout })

			var errOut bytes.Buffer
			start := time.Now()
			require.Equal(t, tc.want, fleetNameForPackage(configPath, tc.fleetURL, tc.secret, &errOut))
			assert.Less(t, time.Since(start), 5*time.Second)
			if tc.wantNote {
				assert.Contains(t, errOut.String(), "left the fleet name out of the file name")
				assert.Contains(t, errOut.String(), "fleetctl login")
			} else {
				assert.Empty(t, errOut.String())
			}
		})
	}

	t.Run("missing config file", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config")
		require.Empty(t, fleetNameForPackage(configPath, premium.URL, "servers-secret", &bytes.Buffer{}))
		require.NoFileExists(t, configPath)
	})
}

func TestNormalizeServerURL(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"https://fleet.example.com", "https://fleet.example.com"},
		{"https://Fleet.Example.com/", "https://fleet.example.com"},
		{"HTTPS://fleet.example.com:443", "https://fleet.example.com"},
		{"fleet.example.com", "https://fleet.example.com"},
		{"fleet.example.com:8080", "https://fleet.example.com:8080"},
		{"http://localhost:80/", "http://localhost"},
		{"https://example.com/fleet/", "https://example.com/fleet"},
		{"https://[::1]:8080", "https://[::1]:8080"},
		{"https://[::1]", "https://[::1]"},
		{"  https://fleet.example.com  ", "https://fleet.example.com"},
	} {
		got, ok := normalizeServerURL(tc.in)
		require.True(t, ok, tc.in)
		require.Equal(t, tc.want, got, tc.in)
	}
	for _, in := range []string{"", "   ", "https://", "://"} {
		_, ok := normalizeServerURL(in)
		require.False(t, ok, in)
	}

	require.Equal(t, "https://example.com/fleet", joinURLPrefix("https://example.com/", "/fleet/"))
	require.Equal(t, "https://example.com", joinURLPrefix("https://example.com", ""))
}
