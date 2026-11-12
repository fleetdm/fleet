package fleetctl

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/require"
)

// TestPrintAuthError verifies that the authentication-error message adapts to
// whether SSO is enabled on the server. The SSO-enabled case also guards the
// response contract relied on by the warning (the "settings" wrapper and the
// "sso_enabled" field): a server-side rename of either would drop the
// instructions and fail here.
func TestPrintAuthError(t *testing.T) {
	cfg := config.TestConfig()
	server, ds := testing_utils.RunServerWithMockedDS(t, &service.TestServerOpts{
		License:     &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)},
		FleetConfig: &cfg,
		// Bypass the app config cache so each sub-test sees its own SSO setting.
		NoCacheDatastore: true,
	})

	client, err := service.NewClient(server.URL, true, "", "")
	require.NoError(t, err)

	setSSO := func(enabled bool) {
		ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{SSOSettings: &fleet.SSOSettings{EnableSSO: enabled}}, nil
		}
	}

	t.Run("SSO enabled shows SSO instructions", func(t *testing.T) {
		setSSO(true)

		var buf bytes.Buffer
		printAuthError(&buf, client, "Token missing.")

		out := buf.String()
		require.Contains(t, out, "Token missing.")
		require.Contains(t, out, ssoAuthInstructions)
	})

	t.Run("SSO disabled shows default login message", func(t *testing.T) {
		setSSO(false)

		var buf bytes.Buffer
		printAuthError(&buf, client, "Token missing.")

		out := buf.String()
		require.Contains(t, out, "Token missing.")
		require.Contains(t, out, "Please log in with: fleetctl login")
		require.NotContains(t, out, ssoAuthInstructions)
	})
}
