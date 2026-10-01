package sentinelone

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// darwinStatusFixture is `sentinelctl status` output from a healthy macOS host.
const darwinStatusFixture = `Agent
   Version:                                25.3.4.8365
   ID:                                     d7f94b33-7f99-4c43-9c6b-1c6e0d01aabb
   Install Date:                           6/1/26, 10:09:51 AM
   Missing Authorizations
   ES Framework:                           started
   Agent Operational State:                enabled
   Remote Profiler:                        not running
   Agent Network Monitoring:               started
   Network Extension:                      running
   Network Extension Content Filter:       active
   Ready:                                  yes
   Protection:                             enabled
   Infected:                               no
   Network Quarantine:                     no
   Compatible OS:                          compatible
Command
   Authentication:                         enabled
Daemons
   Services
      Agent Helper:                        ready
      Agent UI:                            ready
      Cleaner:                             ready
      Control Service:                     ready
      Framework:                           ready
      Guard:                               ready
      Helper Service:                      ready
      Lib Hooks Service:                   not ready
      Lib Logs Service:                    not ready
      Shell:                               ready
   Integrity
      sentineld:                           ok
      sentineld_guard:                     ok
      sentineld_helper:                    ok
      sentineld_shell:                     not running
Launchd
   agent-helper:                           valid
   agent-ui:                               valid
   sentinel-extensions:                    valid
   sentineld:                              valid
   sentineld-guard:                        valid
   sentineld-helper:                       valid
   sentineld-shell:                        valid
Management
   Server:                                 https://euce1-109.sentinelone.net
   Site Key:                               site-key-abc-123
   Last Seen:                              6/1/26, 1:14:28 PM
   Connected:                              yes
`

// windowsStatusFixture is `SentinelCtl.exe status` output from a Windows host
// running agent 25.1.4.434.
const windowsStatusFixture = `Disable State: Not disabled by the user
SentinelMonitor is loaded
Self-Protection status: On
Monitor Build id: 25.1.4.434+8d4abf01154f6752-Release.x64
SentinelNetworkMonitor is loaded
SentinelAgent is loaded
SentinelAgent is running as PPL
Mitigation policy: none
`

// mockSentinelctl replaces the sentinelctl runner for the duration of the test.
func mockSentinelctl(t *testing.T, fn func(args []string) ([]byte, error)) {
	t.Helper()
	original := runSentinelctl
	runSentinelctl = func(_ context.Context, args ...string) ([]byte, error) {
		return fn(args)
	}
	t.Cleanup(func() { runSentinelctl = original })
}

// mockStatusOutput replaces the sentinelctl runner with one that returns out
// for `sentinelctl status`.
func mockStatusOutput(t *testing.T, out string) {
	t.Helper()
	mockSentinelctl(t, func(args []string) ([]byte, error) {
		require.Equal(t, []string{"status"}, args)
		return []byte(out), nil
	})
}
