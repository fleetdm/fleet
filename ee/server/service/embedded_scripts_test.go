package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireScriptContains(t *testing.T, script, substr, name string) {
	t.Helper()
	require.True(t, strings.Contains(script, substr), "%s must contain %q", name, substr) //nolint:testifylint // require.Contains would dump the whole ~10KB script on failure
}

// The lock and wipe scripts are the two halves of keeping a user off a machine,
// so a login-lockout mechanism added to one belongs in the other. pam_nologin is
// the only one of them that reaches directory (SSSD/LDAP/Kerberos/AD) accounts.
func TestLinuxLockAndWipeDenyLoginsViaPAM(t *testing.T) {
	t.Parallel()
	for name, script := range map[string]string{
		"linux_lock.sh": string(linuxLockScript),
		"linux_wipe.sh": string(linuxWipeScript),
	} {
		requireScriptContains(t, script, "/etc/nologin", name)
		requireScriptContains(t, script, "/run/nologin", name)
	}
}

// Logins must be denied before sessions are torn down, or the user can race
// back in through the gap. Staging has to come first in turn: locking logins on
// a host the wipe can't start on leaves it unreachable and intact.
func TestLinuxWipeScriptStagesBeforeBlockingLogins(t *testing.T) {
	t.Parallel()
	script := string(linuxWipeScript)
	stage := strings.LastIndex(script, `cp "$0" "$WIPE_SCRIPT"`)
	block := strings.LastIndex(script, "\n    block_logins\n")
	logout := strings.LastIndex(script, "\n    logout_users\n")
	require.NotEqual(t, -1, stage, "linux_wipe.sh must stage the script out of fleetd's run directory")
	require.NotEqual(t, -1, block, "linux_wipe.sh must call block_logins")
	require.Less(t, stage, block, "staging must succeed before logins are blocked")
	require.Less(t, block, logout, "block_logins must run before logout_users")
}

// orbit.service is KillMode=control-group with Restart=always and CPUQuota=20%,
// so a nohup child dies with orbit and runs at a fifth of one CPU.
func TestLinuxWipeScriptDetachesOutsideOrbitCgroup(t *testing.T) {
	t.Parallel()
	script := string(linuxWipeScript)
	requireScriptContains(t, script, "systemd-run --unit=fleet-wipe", "linux_wipe.sh")
	// --collect is systemd 236+, and Fleet supports RHEL/CentOS 7 (systemd 219).
	require.NotContains(t, script, "--collect", "linux_wipe.sh must not use --collect")
	// Keying the fallback off systemd-run failing rather than existing: old
	// systemd and an unreachable bus must still reach it.
	requireScriptContains(t, script, "if ! systemd-run", "linux_wipe.sh")
	requireScriptContains(t, script, "/usr/bin/nohup sh", "linux_wipe.sh")
}
