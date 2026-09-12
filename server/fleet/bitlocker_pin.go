package fleet

import (
	"time"
)

// BitLocker startup PINs are chosen by the end user on the My device page and applied by fleetd, which runs as SYSTEM
// and therefore needs no local admin rights from the person at the keyboard. Because the modal lives in a browser and
// nothing on the device lets that page reach fleetd, the PIN is relayed through the Fleet server: it is stored
// encrypted, handed to the agent exactly once on its next config poll, and cleared.

// BitLockerPINMinLength and BitLockerPINMaxLength bound a startup PIN. Windows enforces 6 to 20 characters for
// ProtectKeyWithTPMAndPIN unless SystemDrivesMinimumPINLength lowers the floor, and only digits unless the
// SystemDrivesEnhancedPIN policy is enabled. Fleet sets neither policy, so it validates against the stock range and
// lets Windows reject anything a host's own policy forbids.
const (
	BitLockerPINMinLength = 6
	BitLockerPINMaxLength = 20
)

// BitLockerPINRequestTTL is how long a submitted PIN stays collectable. The agent polls its config every 30 seconds,
// so this is generous, and it bounds how long the server holds the secret when an agent is offline.
const BitLockerPINRequestTTL = 5 * time.Minute

// BitLockerPINRequestStatus is where an end user's PIN submission stands.
type BitLockerPINRequestStatus string

const (
	// BitLockerPINRequestPending means the PIN is stored and waiting for the agent to collect it.
	BitLockerPINRequestPending BitLockerPINRequestStatus = "pending"
	// BitLockerPINRequestDelivered means the agent has collected the PIN and the stored ciphertext has been cleared.
	BitLockerPINRequestDelivered BitLockerPINRequestStatus = "delivered"
	// BitLockerPINRequestSet means the agent applied the PIN. The row is kept in this state rather than deleted so the
	// waiting page has a positive signal instead of having to infer success from a missing row.
	BitLockerPINRequestSet BitLockerPINRequestStatus = "set"
	// BitLockerPINRequestFailed means the agent could not apply the PIN; ClientError says why.
	BitLockerPINRequestFailed BitLockerPINRequestStatus = "failed"
)

// HostBitLockerPINRequest is the state of a host's BitLocker PIN submission, as exposed to the My device page. It
// never carries the PIN itself.
type HostBitLockerPINRequest struct {
	Status BitLockerPINRequestStatus `json:"status" db:"status"`
	// Error is the agent's reason for a failure, already sanitized and truncated. Empty unless Status is failed.
	Error string `json:"error" db:"client_error"`
	// CreatedAt is when the PIN was submitted, used to expire a request the agent never collected.
	CreatedAt time.Time `json:"-" db:"created_at"`
}

// Expired reports whether a still-pending request is too old to hand to the agent. Only a pending request can expire:
// once delivered, the outcome is the agent's to report.
func (r *HostBitLockerPINRequest) Expired(now time.Time) bool {
	if r == nil || r.Status != BitLockerPINRequestPending {
		return false
	}
	return now.Sub(r.CreatedAt) > BitLockerPINRequestTTL
}

// HostNeedsBitLockerPIN reports whether the end user is currently being asked to create a startup PIN.
//
// It reads the answer off the existing action_required derivation rather than re-deriving it, because that is what
// already accounts for a PIN being required, not yet set, and actually settable on this volume right now (Windows only
// offers PIN setup on a protected volume that can already unseal at boot). The submit endpoint, the orbit
// notification, and the Fleet Desktop toast all go through here so they cannot disagree about whether a PIN is wanted.
func HostNeedsBitLockerPIN(diskEncryption *HostMDMDiskEncryption) bool {
	return diskEncryption != nil &&
		diskEncryption.ActionRequired != nil &&
		*diskEncryption.ActionRequired == ActionRequiredCreatePIN
}

// ValidateBitLockerPIN checks a submitted PIN against the character set and length Windows accepts by default. It
// returns an InvalidArgumentError naming the "pin" field so the caller can return it directly.
func ValidateBitLockerPIN(pin string) error {
	// Count runes, not bytes, so a multi-byte character cannot masquerade as a long-enough PIN.
	length := 0
	digitsOnly := true
	for _, r := range pin {
		length++
		if r < '0' || r > '9' {
			digitsOnly = false
		}
	}

	switch {
	case !digitsOnly:
		// Fleet does not enable the enhanced-PIN policy, so Windows would reject anything but digits with
		// FVE_E_INVALID_PIN_CHARS. Saying so here avoids a pointless round trip to the device.
		return NewInvalidArgumentError("pin", "BitLocker PIN must contain only digits")
	case length < BitLockerPINMinLength || length > BitLockerPINMaxLength:
		return NewInvalidArgumentError("pin", "BitLocker PIN must be between 6 and 20 digits")
	}

	return nil
}
