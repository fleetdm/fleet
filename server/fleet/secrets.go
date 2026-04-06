package fleet

import (
	"fmt"
	"strings"
)

const ServerSecretPrefix = "FLEET_SECRET_"

// HostSecretPrefix is used for host-scoped secrets that are looked up by
// enrollment ID rather than by name. These are expanded at command delivery time.
//
// NOTE: This prefix is for Fleet-internal use only (e.g., injecting per-host
// recovery lock passwords into MDM commands). It is not user-configurable and
// should not be documented as a user-facing feature.
const HostSecretPrefix = "FLEET_HOST_SECRET_" //nolint:gosec // G101: this is a prefix constant, not a credential

// Host secret types
const (
	// HostSecretRecoveryLockPassword is the host secret type for macOS recovery lock passwords.
	// The password is stored encrypted in host_recovery_key_passwords and injected at delivery time.
	HostSecretRecoveryLockPassword = "RECOVERY_LOCK_PASSWORD"

	// HostSecretRecoveryLockPendingPassword is the host secret type for pending recovery lock passwords
	// during password rotation. The pending password is stored encrypted in host_recovery_key_passwords
	// (pending_encrypted_password column) and injected as the NewPassword during rotation.
	HostSecretRecoveryLockPendingPassword = "RECOVERY_LOCK_PENDING_PASSWORD"

	// HostSecretMDMUnlockToken is the host secret type for MDM unlock tokens.
	// The token is stored in the nano_devices table and injected at delivery time for ClearPasscode commands sent to Apple MDM-enrolled hosts.
	HostSecretMDMUnlockToken = "MDM_UNLOCK_TOKEN" // nolint:gosec // G101: this is a constant identifier, not a credential
)

type MissingSecretsError struct {
	MissingSecrets []string
}

func (e MissingSecretsError) Error() string {
	secretVars := make([]string, 0, len(e.MissingSecrets))
	for _, secret := range e.MissingSecrets {
		secretVars = append(secretVars, fmt.Sprintf("\"$%s%s\"", ServerSecretPrefix, secret))
	}
	plural := ""
	if len(secretVars) > 1 {
		plural = "s"
	}
	return fmt.Sprintf("Couldn't add. Secret variable%s %s missing from database", plural, strings.Join(secretVars, ", "))
}
