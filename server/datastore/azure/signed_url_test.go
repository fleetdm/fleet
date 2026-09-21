package azure

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// TestSignSASURL verifies the SAS-signed URL download path added on top of
// Azure Blob Storage. It runs fully offline: GetSASURL computes the URL
// locally from the shared key credential without contacting the container.
func TestSignSASURL(t *testing.T) {
	baseCfg := func() config.AzureConfig {
		return config.AzureConfig{
			SoftwareInstallersAccountName: testAccountName,
			SoftwareInstallersAccountKey:  testAccountKey,
			SoftwareInstallersContainer:   "test-container",
		}
	}

	t.Run("signed url enabled returns SAS URL", func(t *testing.T) {
		cfg := baseCfg()
		cfg.SoftwareInstallersSignedURL = true
		store, err := NewSoftwareInstallerStore(cfg)
		require.NoError(t, err)

		signed, err := store.Sign(context.Background(), "abc123", 15*time.Minute)
		require.NoError(t, err)

		u, err := url.Parse(signed)
		require.NoError(t, err)
		require.Equal(t, "https", u.Scheme)
		require.Contains(t, u.Host, testAccountName)
		require.Contains(t, u.Path, "test-container")
		require.Contains(t, u.Path, "abc123")

		q := u.Query()
		require.NotEmpty(t, q.Get("sig"))
		require.Equal(t, "r", q.Get("sp")) // read-only permission
	})

	t.Run("signed url disabled returns ErrNotConfigured", func(t *testing.T) {
		store, err := NewSoftwareInstallerStore(baseCfg())
		require.NoError(t, err)

		_, err = store.Sign(context.Background(), "abc123", 15*time.Minute)
		require.ErrorIs(t, err, fleet.ErrNotConfigured)
	})
}
