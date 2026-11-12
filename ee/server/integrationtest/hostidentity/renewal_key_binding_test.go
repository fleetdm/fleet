//go:build !windows

// Windows is disabled because the TPM simulator requires CGO, which causes lint failures on Windows.

package hostidentity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/enrollment"
	"github.com/stretchr/testify/require"
)

func TestHostIdentityRenewalKeyBindingEnforced(t *testing.T) {
	s := SetUpSuiteWithConfig(t, "integrationtest.HostIdentityRenewalKeyBindingEnforced", false, func(cfg *config.FleetConfig) {
		cfg.Osquery.EnrollCooldown = 0
		cfg.Auth.RequireHostIdentityRenewalKeyBinding = true
	})
	defer mysqltest.TruncateTables(t, s.BaseSuite.DS, []string{
		"host_identity_scep_serials", "host_identity_scep_certificates",
	}...)

	cert, key := testGetCertWithCurve(t, s, elliptic.P256())
	newKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	_, err = enrollHostIdentityCSR(t, s, renewalCSR(t, cert, key, newKey, true))
	_, ok := errors.AsType[enrollment.RejectedError](err)
	require.True(t, ok, "legacy renewal proof should be rejected, got %v", err)

	renewed, err := enrollHostIdentityCSR(t, s, renewalCSR(t, cert, key, newKey, false))
	require.NoError(t, err)
	renewedKey, ok := renewed.PublicKey.(*ecdsa.PublicKey)
	require.True(t, ok)
	require.True(t, newKey.PublicKey.Equal(renewedKey))
}
