package fleet

import (
	"errors"
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

	// HostSecretPSSODeviceRegistrationToken is the host secret type for the Apple
	// Platform SSO device registration token. The token is not stored: it is a
	// Fleet-signed JWT minted on the fly for the requesting host at command
	// delivery time, so it never appears in the database or on /mdm/commands.
	HostSecretPSSODeviceRegistrationToken = "PSSO_DEVICE_REGISTRATION_TOKEN" // nolint:gosec // G101: this is a constant identifier, not a credential

	// HostSecretEnrollSecret is the host secret type for the per-device,
	// single-use enroll secret embedded in the fleetd configuration profile when
	// auth.use_one_time_enroll_secrets is enabled. The secret is minted for the
	// requesting host the first time the profile is delivered and re-delivered
	// unchanged until it is consumed by enrollment.
	HostSecretEnrollSecret = "ENROLL_SECRET" // nolint:gosec // G101: this is a constant identifier, not a credential
)

// HostSecretPlaceholder returns the placeholder string for a host secret type,
// e.g. "$FLEET_HOST_SECRET_ENROLL_SECRET".
func HostSecretPlaceholder(secretType string) string {
	return "$" + HostSecretPrefix + secretType
}

// ValidateNoHostSecretVariables rejects user-provided content that references
// a $FLEET_HOST_SECRET_* placeholder. Those are expanded to per-host secrets
// (recovery lock passwords, unlock tokens, enroll secrets) at delivery time and
// are only ever written by Fleet into the profiles and commands it manages
func ValidateNoHostSecretVariables(document string) error {
	vars := ContainsPrefixVars(document, HostSecretPrefix)
	if len(vars) == 0 {
		return nil
	}
	return &BadRequestError{Message: fmt.Sprintf("Variable %s is reserved for profiles managed by Fleet and can't be used.", HostSecretPlaceholder(vars[0]))}
}

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

// IsMissingSecretsError reports whether err is (or wraps) a MissingSecretsError,
// i.e. a reference to a secret variable that doesn't exist.
func IsMissingSecretsError(err error) bool {
	var valErr MissingSecretsError
	var ptrErr *MissingSecretsError
	return errors.As(err, &valErr) || errors.As(err, &ptrErr)
}
