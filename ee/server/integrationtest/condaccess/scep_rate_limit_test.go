package condaccess

import (
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSCEPRateLimit(t *testing.T) {
	// Set up suite with rate limiting configuration
	cooldown := 5 * time.Minute
	s := SetUpSuiteWithConfig(t, "integrationtest.ConditionalAccessSCEPRateLimit", func(cfg *config.FleetConfig) {
		cfg.Osquery.EnrollCooldown = cooldown
	})

	defer mysqltest.TruncateTables(t, s.BaseSuite.DS, []string{
		"conditional_access_scep_serials", "conditional_access_scep_certificates",
	}...)

	t.Run("RateLimitSameHost", func(t *testing.T) {
		ctx := t.Context()

		// Create a test host
		host, err := s.DS.NewHost(ctx, &fleet.Host{
			OsqueryHostID:   ptr.String("test-host-rate-limit"),
			NodeKey:         ptr.String("test-node-key-rate-limit"),
			UUID:            "test-uuid-rate-limit",
			Hostname:        "test-hostname-rate-limit",
			Platform:        "darwin",
			DetailUpdatedAt: time.Now(),
		})
		require.NoError(t, err)

		// First certificate request - should succeed
		challenge, err := s.DS.NewChallenge(ctx)
		require.NoError(t, err)
		cert1 := requestSCEPCertificate(t, s, host.UUID, challenge)
		require.NotNil(t, cert1, "First certificate request should succeed")
		assert.Equal(t, "urn:device:apple:uuid:"+host.UUID, cert1.URIs[0].String())

		// Second certificate request immediately after - should fail due to rate limit.
		// Rate limiting short-circuits before the challenge is consumed, so a fresh,
		// valid challenge keeps this assertion focused on the rate-limit behavior.
		secondChallenge, err := s.DS.NewChallenge(ctx)
		require.NoError(t, err)
		httpResp, pkiMsgResp, cert2 := requestSCEPCertificateWithChallenge(t, s, host.UUID, secondChallenge)
		require.Equal(t, http.StatusTooManyRequests, httpResp.StatusCode, "Should return HTTP 429 for rate limit")
		require.Nil(t, pkiMsgResp, "PKI message not parsed for rate limit errors")
		require.Nil(t, cert2, "Second certificate request should fail due to rate limit")

		// Different host should be able to get certificate
		differentHost, err := s.DS.NewHost(ctx, &fleet.Host{
			OsqueryHostID:   ptr.String("test-host-different"),
			NodeKey:         ptr.String("test-node-key-different"),
			UUID:            "test-uuid-different",
			Hostname:        "test-hostname-different",
			Platform:        "darwin",
			DetailUpdatedAt: time.Now(),
		})
		require.NoError(t, err)

		differentChallenge, err := s.DS.NewChallenge(ctx)
		require.NoError(t, err)
		certDifferent := requestSCEPCertificate(t, s, differentHost.UUID, differentChallenge)
		require.NotNil(t, certDifferent, "Different host should be able to get certificate")
		assert.Equal(t, "urn:device:apple:uuid:"+differentHost.UUID, certDifferent.URIs[0].String())
	})
}
