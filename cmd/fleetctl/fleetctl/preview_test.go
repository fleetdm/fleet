package fleetctl

import (
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/stretchr/testify/require"
)

func TestNameMatches(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"orbit", true},
		{"orbit.exe", true},
		{"ORBIT.EXE", true},
		{"Fleet", true},
		{"fleet", false},
		{"fleetctl", false},
		{"fleet-desktop", false},
		{"Fleet osquery", false},
		{"osqueryd", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, nameMatches(tc.name, "orbit", constant.AgentDisplayName))
		})
	}
}
