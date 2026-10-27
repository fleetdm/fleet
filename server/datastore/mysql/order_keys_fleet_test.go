package mysql

import (
	"strings"
	"testing"

	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/stretchr/testify/assert"
)

// The order keys use the fleet names, not the old team names.
func TestOrderKeysUseFleetNames(t *testing.T) {
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
			for key := range allowlist {
				assert.False(t, strings.HasPrefix(key, "team_"), "old team order key %q", key)
			}
		})
	}
	assert.Contains(t, hostAllowedOrderKeys, "fleet_id")
	assert.Contains(t, hostAllowedOrderKeys, "fleet_name")
	assert.Contains(t, policyAllowedOrderKeys, "fleet_id")
	assert.Contains(t, queryAllowedOrderKeys, "fleet_id")
	assert.Contains(t, labelsAllowedOrderKeys, "fleet_id")
	assert.Contains(t, hostsInLabelAllowedOrderKeys, "fleet_name")
	assert.Contains(t, mdmConfigProfilesAllowedOrderKeys, "fleet_id")
}
