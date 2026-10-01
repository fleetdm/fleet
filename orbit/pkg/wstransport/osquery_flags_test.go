package wstransport

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOsqueryFlags(t *testing.T) {
	const orbitExt = "com.fleetdm.orbit.osquery_extension.v1"

	for _, tc := range []struct {
		name            string
		userFlags       map[string]string
		expectedRequire string
		expectedTimeout string
	}{
		{
			name:            "no user flags",
			userFlags:       nil,
			expectedRequire: orbitExt,
			expectedTimeout: "60",
		},
		{
			name:            "user required extensions are kept",
			userFlags:       map[string]string{"--extensions_require": "my_ext,other_ext"},
			expectedRequire: "my_ext,other_ext," + orbitExt,
			expectedTimeout: "60",
		},
		{
			name:            "orbit extension deduplicated, whitespace and empties trimmed",
			userFlags:       map[string]string{"--extensions_require": " my_ext , " + orbitExt + " ,,"},
			expectedRequire: "my_ext," + orbitExt,
			expectedTimeout: "60",
		},
		{
			name:            "quoted value from hand-edited flagfile",
			userFlags:       map[string]string{"--extensions_require": `"my_ext"`},
			expectedRequire: "my_ext," + orbitExt,
			expectedTimeout: "60",
		},
		{
			name:            "larger user timeout wins",
			userFlags:       map[string]string{"--extensions_timeout": "120"},
			expectedRequire: orbitExt,
			expectedTimeout: "120",
		},
		{
			name:            "smaller user timeout is raised to the default",
			userFlags:       map[string]string{"--extensions_timeout": "5"},
			expectedRequire: orbitExt,
			expectedTimeout: "60",
		},
		{
			name:            "non-numeric user timeout falls back to the default",
			userFlags:       map[string]string{"--extensions_timeout": "soon"},
			expectedRequire: orbitExt,
			expectedTimeout: "60",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, []string{
				"--distributed_plugin", DistributedPluginName,
				"--extensions_require", tc.expectedRequire,
				"--extensions_timeout", tc.expectedTimeout,
			}, OsqueryFlags(orbitExt, tc.userFlags))
		})
	}
}
