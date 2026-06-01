package osquery_utils

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))
}

// appConfigWith returns an AppConfig with Cloud Identity settings populated.
func appConfigWith(domains []string, suffix string) *fleet.AppConfig {
	c := &fleet.AppConfig{}
	c.Integrations.GoogleCloudIdentity = &fleet.GoogleCloudIdentitySettings{
		WorkspaceDomains: domains,
		PartnerSuffix:    suffix,
		CustomerID:       "C0xxxxxxx",
	}
	return c
}

func TestDirectIngestGoogleEV_FiltersByDomain(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return appConfigWith([]string{"example.com"}, "fleet"), nil
	}

	var emails []string
	ds.UpsertHostGoogleCloudIdentityResolutionFunc = func(ctx context.Context, hostID uint, workspaceEmail, partnerSuffix string) error {
		emails = append(emails, workspaceEmail)
		assert.Equal(t, "fleet", partnerSuffix)
		return nil
	}

	host := &fleet.Host{ID: 1, Platform: "darwin"}
	rows := []map[string]string{
		// Corporate account — kept.
		{"username": "robbiet480", "gaia_id": "g1", "resource_id": "r1", "email": "robbie@example.com"},
		// Personal Gmail — filtered.
		{"username": "robbiet480", "gaia_id": "g2", "resource_id": "r2", "email": "personal@gmail.com"},
		// Other Workspace tenant — filtered.
		{"username": "robbiet480", "gaia_id": "g3", "resource_id": "r3", "email": "user@othercorp.com"},
		// Another corporate account — kept.
		{"username": "alice", "gaia_id": "g4", "resource_id": "r4", "email": "alice@example.com"},
	}

	require.NoError(t, directIngestGoogleEndpointVerificationDetails(context.Background(), nopLogger(), host, ds, rows))

	require.Len(t, emails, 2, "only corporate-domain rows survive filter")
	assert.Equal(t, "robbie@example.com", emails[0])
	assert.Equal(t, "alice@example.com", emails[1])
}

func TestDirectIngestGoogleEV_CaseInsensitiveDomainMatch(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return appConfigWith([]string{"Example.COM"}, "fleet"), nil
	}
	var upserts int
	ds.UpsertHostGoogleCloudIdentityResolutionFunc = func(ctx context.Context, hostID uint, workspaceEmail, partnerSuffix string) error {
		upserts++
		return nil
	}

	host := &fleet.Host{ID: 1, Platform: "darwin"}
	rows := []map[string]string{
		{"resource_id": "r1", "email": "USER@EXAMPLE.COM"},
		{"resource_id": "r2", "email": "user@example.com"},
		{"resource_id": "r3", "email": "user@EXAMPLE.com"},
	}

	require.NoError(t, directIngestGoogleEndpointVerificationDetails(context.Background(), nopLogger(), host, ds, rows))
	assert.Equal(t, 3, upserts)
}

func TestDirectIngestGoogleEV_MultipleDomains(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return appConfigWith([]string{"example.com", "sub.example.com"}, "fleet"), nil
	}
	var upserts int
	ds.UpsertHostGoogleCloudIdentityResolutionFunc = func(ctx context.Context, hostID uint, workspaceEmail, partnerSuffix string) error {
		upserts++
		return nil
	}

	host := &fleet.Host{ID: 1, Platform: "darwin"}
	rows := []map[string]string{
		{"resource_id": "r1", "email": "a@example.com"},
		{"resource_id": "r2", "email": "b@sub.example.com"},
		// Subdomain partial match must NOT be allowed — domain matching is
		// exact, not suffix-based.
		{"resource_id": "r3", "email": "c@otherexample.com"},
	}

	require.NoError(t, directIngestGoogleEndpointVerificationDetails(context.Background(), nopLogger(), host, ds, rows))
	assert.Equal(t, 2, upserts)
}

func TestDirectIngestGoogleEV_EmptyRowsNoOp(t *testing.T) {
	ds := new(mock.Store)
	// AppConfig should NOT be loaded on empty rows.
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		t.Fatal("AppConfig should not be loaded when rows are empty")
		return nil, nil
	}
	host := &fleet.Host{ID: 1, Platform: "darwin"}
	require.NoError(t, directIngestGoogleEndpointVerificationDetails(context.Background(), nopLogger(), host, ds, nil))
}

func TestDirectIngestGoogleEV_SkipsRowsMissingFields(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return appConfigWith([]string{"example.com"}, "fleet"), nil
	}
	var upserts int
	ds.UpsertHostGoogleCloudIdentityResolutionFunc = func(ctx context.Context, hostID uint, workspaceEmail, partnerSuffix string) error {
		upserts++
		return nil
	}

	host := &fleet.Host{ID: 1, Platform: "darwin"}
	rows := []map[string]string{
		{"resource_id": "r"},                            // missing email — skipped
		{"resource_id": "r1", "email": "u@example.com"}, // complete — kept
		{"email": "u2@example.com"},                     // missing resource_id but has email — kept (resource_id is no longer load-bearing)
	}
	require.NoError(t, directIngestGoogleEndpointVerificationDetails(context.Background(), nopLogger(), host, ds, rows))
	// 2 rows have valid emails; resource_id is informational only since the
	// resolution layer now uses serial+email rather than rawResourceId lookup.
	assert.Equal(t, 2, upserts)
}

func TestDirectIngestGoogleEV_NoDomainsConfiguredSkips(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		c := &fleet.AppConfig{}
		c.Integrations.GoogleCloudIdentity = &fleet.GoogleCloudIdentitySettings{
			PartnerSuffix: "fleet",
			// WorkspaceDomains empty — Fleet refuses to write anything
			// without the allowlist to filter against.
		}
		return c, nil
	}
	ds.UpsertHostGoogleCloudIdentityResolutionFunc = func(ctx context.Context, hostID uint, workspaceEmail, partnerSuffix string) error {
		t.Fatal("upsert should not run without configured domains")
		return nil
	}

	host := &fleet.Host{ID: 1, Platform: "darwin"}
	rows := []map[string]string{{"resource_id": "r", "email": "u@example.com"}}
	require.NoError(t, directIngestGoogleEndpointVerificationDetails(context.Background(), nopLogger(), host, ds, rows))
}

func TestDirectIngestGoogleEV_SettingsNilSkips(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		// Settings literally nil — initial install before admin configured anything.
		return &fleet.AppConfig{}, nil
	}
	ds.UpsertHostGoogleCloudIdentityResolutionFunc = func(ctx context.Context, hostID uint, workspaceEmail, partnerSuffix string) error {
		t.Fatal("upsert should not run without configured settings")
		return nil
	}
	host := &fleet.Host{ID: 1, Platform: "darwin"}
	rows := []map[string]string{{"resource_id": "r", "email": "u@example.com"}}
	require.NoError(t, directIngestGoogleEndpointVerificationDetails(context.Background(), nopLogger(), host, ds, rows))
}

func TestDirectIngestGoogleEV_DefaultSuffixApplied(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		// PartnerSuffix unset — should default to "fleet".
		c := &fleet.AppConfig{}
		c.Integrations.GoogleCloudIdentity = &fleet.GoogleCloudIdentitySettings{
			WorkspaceDomains: []string{"example.com"},
		}
		return c, nil
	}
	var gotSuffix string
	ds.UpsertHostGoogleCloudIdentityResolutionFunc = func(ctx context.Context, hostID uint, workspaceEmail, partnerSuffix string) error {
		gotSuffix = partnerSuffix
		return nil
	}

	host := &fleet.Host{ID: 1, Platform: "darwin"}
	rows := []map[string]string{{"resource_id": "r", "email": "u@example.com"}}
	require.NoError(t, directIngestGoogleEndpointVerificationDetails(context.Background(), nopLogger(), host, ds, rows))
	assert.Equal(t, "fleet", gotSuffix)
}

func TestDirectIngestGoogleEV_LowercaseEmailNormalization(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return appConfigWith([]string{"example.com"}, "fleet"), nil
	}
	var gotEmail string
	ds.UpsertHostGoogleCloudIdentityResolutionFunc = func(ctx context.Context, hostID uint, workspaceEmail, partnerSuffix string) error {
		gotEmail = workspaceEmail
		return nil
	}

	host := &fleet.Host{ID: 1, Platform: "darwin"}
	rows := []map[string]string{{"resource_id": "r", "email": "  ROBBIE@EXAMPLE.COM  "}}
	require.NoError(t, directIngestGoogleEndpointVerificationDetails(context.Background(), nopLogger(), host, ds, rows))
	assert.Equal(t, "robbie@example.com", gotEmail, "email is lowercased + trimmed before being persisted")
}
