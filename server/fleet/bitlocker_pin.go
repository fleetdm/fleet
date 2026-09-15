package fleet

import (
	"time"
)

// BitLockerPINRequestTTL is how long a submitted PIN stays collectable by the agent. The agent checks in every 30 seconds,
// so the margin is for a host that sleeps or drops offline right after the end user submits. An uncollected
// submission's ciphertext is cleared by the hourly cleanups cron, so it can outlive the TTL by up to an hour.
const BitLockerPINRequestTTL = 15 * time.Minute

// BitLockerPINClientErrorMaxLength matches the width of host_bitlocker_pin_requests.client_error.
const BitLockerPINClientErrorMaxLength = 255

// BitLockerPINRequestTimedOutError is recorded against a submission the agent never collected.
const BitLockerPINRequestTimedOutError = "Fleet didn't hear back from this device. It may be offline."

// BitLockerPINRequestStatus is where an end user's PIN submission stands.
type BitLockerPINRequestStatus string

const (
	// BitLockerPINRequestPending means the PIN is stored and waiting for the agent to collect it.
	BitLockerPINRequestPending BitLockerPINRequestStatus = "pending"
	// BitLockerPINRequestDelivered means the agent has collected the PIN and the stored ciphertext has been cleared.
	BitLockerPINRequestDelivered BitLockerPINRequestStatus = "delivered"
	// BitLockerPINRequestSet means the agent applied the PIN. The row is kept so the waiting page has a positive signal.
	BitLockerPINRequestSet BitLockerPINRequestStatus = "set"
	// BitLockerPINRequestFailed means the agent could not apply the PIN.
	BitLockerPINRequestFailed BitLockerPINRequestStatus = "failed"
)

// HostBitLockerPINRequest is the state of a host's BitLocker PIN submission. It never carries the PIN.
type HostBitLockerPINRequest struct {
	Status BitLockerPINRequestStatus `json:"status" db:"status"`
	// Error is the agent's reason for a failure. Empty unless Status is failed.
	Error     string    `json:"error" db:"client_error"`
	CreatedAt time.Time `json:"-" db:"created_at"`
}

// Expired reports whether a pending request is too old to hand to the agent.
func (r *HostBitLockerPINRequest) Expired(now time.Time) bool {
	if r == nil || r.Status != BitLockerPINRequestPending {
		return false
	}
	return now.Sub(r.CreatedAt) > BitLockerPINRequestTTL
}
