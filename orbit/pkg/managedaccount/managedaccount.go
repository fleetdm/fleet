// Package managedaccount creates and maintains the Fleet-managed local admin account on Windows hosts, and escrows its
// password to the Fleet server.
//
// The server asks for the account by setting the CreateWindowsManagedLocalAccount notification on the orbit config
// response, and stops asking once this host escrows a password for its current MDM enrollment. Every step is idempotent,
// so being asked again is always safe.
package managedaccount

import (
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// Escrower sends the managed local account password to the Fleet server.
type Escrower interface {
	SendManagedLocalAccountPassword(password, clientError string) error
}

// provisionFunc creates or updates the managed local admin account and hides it from the sign-in screen.
type provisionFunc func(username, password string) error

// Receiver reacts to the CreateWindowsManagedLocalAccount notification.
type Receiver struct {
	escrower Escrower

	// provision is indirected so tests can exercise the flow without touching Windows APIs. nil means
	// use the platform implementation.
	provision provisionFunc

	// retryFrequency is the minimum time between attempts after a failure. The success path needs no throttle because
	// the server stops sending the notification once a password is escrowed, so this only ever paces a host that fails.
	retryFrequency time.Duration

	// mu keeps a single provisioning attempt in flight. Held for the duration of the background
	// goroutine, so a notification that arrives again while work is running is dropped rather than
	// starting a second account reset. It also guards lastFailure.
	mu sync.Mutex

	// lastFailure is when the most recent attempt failed, zero after a success.
	lastFailure time.Time
}

// New returns a Receiver that escrows through the given Escrower, retrying at most once every
// retryFrequency after a failure.
func New(escrower Escrower, retryFrequency time.Duration) *Receiver {
	return &Receiver{escrower: escrower, retryFrequency: retryFrequency}
}

// Run implements fleet.OrbitConfigReceiver. It returns immediately; provisioning happens in the
// background so a slow Windows API call or HTTP request never gates the config receiver loop.
func (r *Receiver) Run(cfg *fleet.OrbitConfig) error {
	if cfg == nil || !cfg.Notifications.CreateWindowsManagedLocalAccount {
		return nil
	}
	r.attempt()
	return nil
}

// attempt starts provisioning in the background. The returned channel is closed once the attempt has
// finished and released the single-flight lock, or nil when another attempt was already running.
// Run discards it; it exists so callers that need to know an attempt is fully done, notably tests,
// observe a point where the lock is guaranteed free rather than one merely inside the work.
func (r *Receiver) attempt() <-chan struct{} {
	// TryLock rather than Lock: if an attempt is already running, drop this one instead of queueing a second account
	// reset behind it. The server keeps asking until an escrow succeeds, so nothing is lost by skipping.
	if !r.mu.TryLock() {
		log.Debug().Msg("managed local account: provisioning already in progress, skipping")
		return nil
	}
	// The server re-sends the notification on every config fetch, so without this a host that cannot
	// provision would redo the syscalls and re-post its error every 30 seconds, indefinitely.
	if !r.lastFailure.IsZero() && time.Since(r.lastFailure) <= r.retryFrequency {
		log.Debug().Msg("managed local account: last attempt failed too recently, skipping")
		r.mu.Unlock()
		return nil
	}
	done := make(chan struct{})
	go func() {
		// Deferred LIFO, so the mutex is released first, then the panic is contained, then completion is signaled.
		defer close(done)
		defer func() {
			// A panic in a goroutine takes down the whole process, and this one drives raw Windows syscalls.
			// Provisioning a local account must not be able to kill orbit/osquery. The next poll retries.
			if p := recover(); p != nil {
				log.Error().Interface("panic", p).Msg("managed local account: recovered from panic while provisioning")
			}
		}()
		defer r.mu.Unlock()

		// Assume failure, so an early return or a panic still paces the next attempt; cleared on success.
		// Both writes happen while the lock is held.
		r.lastFailure = time.Now()
		if r.createAndEscrow() {
			r.lastFailure = time.Time{} // clear time
		}
	}()
	return done
}

// createAndEscrow generates a password, provisions the account, and escrows the password. Any failure before the escrow
// returns without recording success, so the next config fetch retries the whole flow; the provisioning step resets the
// password of an existing account, which is what makes that retry safe.
// It reports whether the password was successfully escrowed.
func (r *Receiver) createAndEscrow() bool {
	password := fleet.GenerateManagedLocalAccountPassword(true)

	provision := r.provision
	if provision == nil {
		provision = provisionAccount
	}

	if err := provision(fleet.ManagedLocalAccountUsername, password); err != nil {
		log.Error().Err(err).Msg("managed local account: creating account")
		// Tell the server why, so it surfaces on the host instead of only in this log. The server
		// records the failure and keeps asking, so this is a report, not a terminal state.
		if escrowErr := r.escrower.SendManagedLocalAccountPassword("", err.Error()); escrowErr != nil {
			log.Error().Err(escrowErr).Msg("managed local account: reporting creation failure")
		}
		return false
	}

	if err := r.escrower.SendManagedLocalAccountPassword(password, ""); err != nil {
		// The account now exists with a password Fleet does not know. That is recovered by the next
		// notification: provisioning resets the password and escrows the new one.
		log.Error().Err(err).Msg("managed local account: escrowing password")
		return false
	}

	log.Info().Msg("managed local account: account created; password escrowed")
	return true
}
