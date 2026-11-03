package mysql

import (
	"strings"
	"testing"

	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/stretchr/testify/assert"
)

// Every team_* order key is also accepted as fleet_*, sorting by the same expression. Remove this test together with the team_* keys.
func TestOrderKeysAcceptFleetNames(t *testing.T) {
	t.Parallel()

	for name, allowlist := range map[string]common_mysql.OrderKeyAllowlist{
		"hosts":        hostAllowedOrderKeys,
		"labels":       labelsAllowedOrderKeys,
		"hostsInLabel": hostsInLabelAllowedOrderKeys,
		"reports":      queryAllowedOrderKeys,
		"policies":     policyAllowedOrderKeys,
		"mdmProfiles":  mdmConfigProfilesAllowedOrderKeys,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			found := 0
			for key, expr := range allowlist {
				if suffix, ok := strings.CutPrefix(key, "team_"); ok {
					found++
					assert.Equal(t, expr, allowlist["fleet_"+suffix], "fleet_%s", suffix)
				}
			}
			assert.NotZero(t, found, "no team_ order keys found")
		})
	}
}
