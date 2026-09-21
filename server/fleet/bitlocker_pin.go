package fleet

import (
	"time"
)

// BitLockerPINRequestTTL is how long a submitted PIN stays collectable by the agent. The agent checks in every 30 seconds,
// so the margin is for a host that sleeps or drops offline right after the end user submits. An uncollected
// submission's ciphertext is cleared by the hourly cleanups cron, so it can outlive the TTL by up to an hour.
const BitLockerPINRequestTTL = 15 * time.Minute

// BitLockerPINResultTimeout is how long Fleet waits for the agent to report on a PIN it collected. The agent retries its report
// on every config poll, so a real one lands within minutes. A PIN the agent applied without reporting still shows as set once
// osquery next reports the volume's protectors, so timing out gives up only on the report.
const BitLockerPINResultTimeout = time.Hour

// BitLockerPINClientErrorMaxLength matches the width of host_bitlocker_pin_requests.client_error.
const BitLockerPINClientErrorMaxLength = 255

// BitLockerPINRequestTimedOutError is recorded against a submission the agent never collected, or never reported on.
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
	Status BitLockerPINRequestStatus `json:"status" db:"status" csv:"-"`
	// Error is the agent's reason for a failure. Empty unless Status is failed.
	Error     string    `json:"error" db:"client_error" csv:"-"`
	CreatedAt time.Time `json:"-" db:"created_at" csv:"-"`
	// UpdatedAt is when the agent collected the PIN, while Status is delivered.
	UpdatedAt time.Time `json:"-" db:"updated_at" csv:"-"`
}

// Expired reports whether a request has waited too long: a pending one for the agent to collect it, or a delivered one for
// the agent to report back.
func (r *HostBitLockerPINRequest) Expired(now time.Time) bool {
	if r == nil {
		return false
	}
	switch r.Status {
	case BitLockerPINRequestPending:
		return now.Sub(r.CreatedAt) > BitLockerPINRequestTTL
	case BitLockerPINRequestDelivered:
		return now.Sub(r.UpdatedAt) > BitLockerPINResultTimeout
	default:
		return false
	}
}
