package mysql

import (
	"strings"
	"testing"

	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every team_* order key must also be accepted as fleet_*, sorting the same way.
func TestOrderKeysAcceptFleetNames(t *testing.T) {
	allowlists := map[string]common_mysql.OrderKeyAllowlist{
		"labels":          labelsAllowedOrderKeys,
		"hostsInLabel":    hostsInLabelAllowedOrderKeys,
		"hosts":           hostAllowedOrderKeys,
		"queries":         queryAllowedOrderKeys,
		"mdmProfiles":     mdmConfigProfilesAllowedOrderKeys,
		"teamPolicies":    policyAllowedOrderKeys,
		"batchScriptHost": batchScriptHostAllowedOrderKeys,
	}
	for name, allowlist := range allowlists {
		t.Run(name, func(t *testing.T) {
			for key, expr := range allowlist {
				if !strings.HasPrefix(key, "team_") {
					continue
				}
				fleetKey := "fleet_" + strings.TrimPrefix(key, "team_")
				got, ok := allowlist[fleetKey]
				require.True(t, ok, "missing %q alongside %q", fleetKey, key)
				assert.Equal(t, expr, got, fleetKey)
			}
		})
	}
}
