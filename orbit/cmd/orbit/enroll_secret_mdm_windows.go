//go:build windows

package main

import (
	"context"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/profiles"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/rs/zerolog/log"
)

// On Windows, Fleet MDM delivers an enroll secret out of band by writing it to a registry value that a Fleet-managed
// configuration profile carries. This file is that channel.
//
// macOS has its own out-of-band delivery, a fleetd configuration profile read under --use-system-configuration in orbitAction, so
// what is Windows-specific here is the carrier rather than the idea. Three things follow from the carrier and keep the two paths
// from sharing code: the value is cleared after adoption, so its presence means "a secret is waiting"; waiting is event-driven
// through RegNotifyChangeKeyValue rather than a poll loop; and the wait has to run after the service manager is up, because
// Windows kills a service that blocks before reporting Running.

// mdmSecretWaitBackstop bounds a single wait for a registry change. The notification is the real mechanism; this only guarantees
// the loop re-checks periodically if a notification is ever missed, and gives the log something to say while a host sits without
// a secret.
const mdmSecretWaitBackstop = 5 * time.Minute

// mdmSecretSlowDelivery is how long a host may wait before the wait stops looking routine. Waiting is the normal state for a
// freshly installed host, so the first minutes stay quiet; past this point the host probably needs help, and the log has to say
// so rather than staying silent.
const mdmSecretSlowDelivery = 10 * time.Minute

// mdmSecretWaitTimeout bounds the whole wait. Nothing else runs while orbit waits here, not even the auto-updater, so an
// unbounded wait would also be a host that can never be fixed by shipping it a new fleetd. Expiry does not exit immediately.
// orbit falls through to the early update check, which is the escape hatch: a newer fleetd there makes orbit update and exit, and
// the service manager starts the new binary. Only after that does it reach its usual "no enroll secret" failure and restart into
// a fresh wait. So a host stuck without a secret still reaches TUF about once an hour. A restart re-enters the wait before that
// check rather than after it, so the first opportunity is an hour in, and a host packaged with --disable-updates gets none.
const mdmSecretWaitTimeout = time.Hour

// adoptMDMSecretIfWaiting adopts an enroll secret Fleet MDM delivered out of band, if one is waiting. It reports whether a secret
// was adopted, and whether the registry channel is usable at all.
//
// A secret delivered this way takes precedence over anything already stored, because a freshly delivered one is how an
// administrator recovers a host whose secret was spent or lost. Callers run it before the file and keystore so the newer value
// wins rather than being masked by them.
//
// Fails closed. If the key cannot be secured, the channel is reported unusable and nothing reads or waits on it, because doing so
// would invite Fleet to deliver a plaintext secret into a key a local non-admin can read.
func adoptMDMSecretIfWaiting(
	enrollSecretPath string, disableKeystore bool, setSecret func(string) error,
) (adopted bool, channelUsable bool) {
	// Create the key before reading it, so a secret delivered later lands somewhere only SYSTEM and Administrators can read rather
	// than in a key created implicitly with inherited permissions.
	if err := profiles.EnsureEnrollSecretKeyIsProtected(); err != nil {
		// This should not happen.
		log.Error().Err(err).Msg(
			"not using the registry channel for an MDM-delivered enroll secret: its key could not be secured")
		return false, false
	}

	adopted, err := adoptMDMDeliveredEnrollSecret(enrollSecretPath, realKeystore{}, disableKeystore, setSecret)
	if err != nil {
		// Not fatal: the caller falls through to the file and keystore, which may still hold a usable secret.
		log.Error().Err(err).Msg("failed to adopt an MDM-delivered enroll secret")
	}
	return adopted, true
}

// canWaitForMDMSecret reports whether orbit should block waiting for Fleet MDM to deliver a secret.
//
// Gated on an active Fleet MDM enrollment rather than on a packaging flag, because the question is whether a secret can still
// arrive, and only the host knows that. Without an enrollment there is no channel to wait on, so orbit keeps its usual behavior
// and fails fast with a clear error rather than blocking.
func canWaitForMDMSecret(channelUsable bool, currentSecret string) bool {
	return channelUsable && currentSecret == "" && update.HasActiveFleetMDMEnrollment()
}

// adoptMDMDeliveredEnrollSecret adopts an enroll secret that Fleet MDM delivered out of band, if one is waiting. It reports
// whether a secret was adopted.
//
// Every other copy of the secret is retired once it is in the keystore: the registry value, so that its presence keeps meaning "a
// secret is waiting", and the installer's secret file, which Fleet may have written with the same value so an older fleetd that
// cannot read the registry still enrolls. Leaving that file behind would strand the secret in plaintext on disk of a host that
// never reads it.
func adoptMDMDeliveredEnrollSecret(
	enrollSecretPath string, ks enrollSecretKeystore, disableKeystore bool, setSecret func(string) error,
) (bool, error) {
	return adoptDeliveredEnrollSecret(
		profiles.GetEnrollSecret, profiles.ClearEnrollSecret, enrollSecretPath, ks, disableKeystore, setSecret)
}

// waitForMDMDeliveredEnrollSecret blocks until Fleet MDM delivers an enroll secret, mdmSecretWaitTimeout elapses, or the service
// manager asks orbit to stop. An administrator resending the profile that carries the secret is what normally ends the wait.
//
// Windows signals when the key changes, so this waits on that notification rather than polling on an interval. The registration
// is good for one notification, so it is re-armed each time around. stop is closed when the service manager asks orbit to stop;
// the wait gives up then so a stop request is not held up by a secret that may never arrive.
func waitForMDMDeliveredEnrollSecret(
	stop <-chan struct{}, enrollSecretPath string, disableKeystore bool, setSecret func(string) error,
) {
	log.Info().Msg("no enroll secret yet, waiting for Fleet MDM to deliver one")

	// A stop request has to interrupt the in-flight Wait, not just be noticed between iterations.
	baseCtx, cancelBase := context.WithCancel(context.Background())
	defer cancelBase()
	go func() {
		select {
		case <-stop:
			cancelBase()
		case <-baseCtx.Done():
		}
	}()

	// Parent of every per-iteration wait, so the deadline lands mid-wait rather than being noticed up
	// to a backstop late.
	waitCtx, cancelWait := context.WithTimeout(baseCtx, mdmSecretWaitTimeout)
	defer cancelWait()

	// Guards the sync trigger below. deviceenroller can take minutes to return, so a loop that comes
	// around faster than that must not stack up invocations.
	var syncInFlight sync.Mutex

	for started := time.Now(); ; {
		select {
		case <-stop:
			log.Info().Msg("stop requested while waiting for an MDM-delivered enroll secret")
			return
		default:
		}

		if waitCtx.Err() != nil {
			log.Warn().Dur("waited", time.Since(started)).Msg(
				"gave up waiting for Fleet MDM to deliver an enroll secret; fleetd will exit and be restarted to try again")
			return
		}

		// Arm the watch before reading. Reading first leaves a gap in which a delivery is seen by
		// neither the read nor the not-yet-armed registration, which would strand an already delivered
		// secret until the backstop expired.
		watch, watchErr := profiles.ArmEnrollSecretWatch()
		if watchErr != nil {
			log.Error().Err(watchErr).Msg("failed to watch for an MDM-delivered enroll secret")
		}

		adopted, err := adoptMDMDeliveredEnrollSecret(enrollSecretPath, realKeystore{}, disableKeystore, setSecret)
		switch {
		case err != nil:
			log.Error().Err(err).Msg("failed to adopt an MDM-delivered enroll secret")
		case adopted:
			if watch != nil {
				watch.Close() //nolint:errcheck // nothing actionable, and the secret is already adopted
			}
			log.Info().Dur("waited", time.Since(started)).Msg("adopted an enroll secret delivered by Fleet MDM")
			return
		}

		// Unenrolling removes the channel this wait depends on, so stop rather than block on a delivery
		// that can no longer happen. orbit then fails startup the way it does without a secret.
		if !update.HasActiveFleetMDMEnrollment() {
			if watch != nil {
				watch.Close() //nolint:errcheck // nothing actionable while giving up on the wait
			}
			log.Warn().Msg("host is no longer enrolled in Fleet MDM, so no enroll secret can be delivered")
			return
		}

		// Nothing is waiting, so ask the device to check in now instead of at its next scheduled poll.
		// Once a host has run a sync-capable fleetd the server relaxes that poll to 8 hours, and the
		// usual wake cannot help here: it arrives over the orbit config endpoint, which needs the very
		// enroll secret this loop is waiting for. Best effort, and off the loop's goroutine so a stop
		// request is not held behind deviceenroller.
		if syncInFlight.TryLock() {
			go func() {
				defer syncInFlight.Unlock()
				if err := update.TriggerWindowsMDMSync(); err != nil {
					log.Debug().Err(err).Msg("failed to trigger an MDM sync while waiting for an enroll secret")
				}
			}()
		}

		if watch == nil {
			// Registration failed, so fall back to sleeping rather than spinning on a broken watch.
			select {
			case <-waitCtx.Done():
			case <-time.After(mdmSecretWaitBackstop):
			}
		} else {
			ctx, cancel := context.WithTimeout(waitCtx, mdmSecretWaitBackstop)
			if err := watch.Wait(ctx); err != nil {
				log.Error().Err(err).Msg("failed waiting for an MDM-delivered enroll secret")
			}
			cancel()
			watch.Close() //nolint:errcheck // the registration is spent; a fresh one is armed next pass
		}

		if waited := time.Since(started); waited > mdmSecretSlowDelivery {
			log.Warn().Dur("waited", waited).Msg(
				"still waiting for Fleet MDM to deliver an enroll secret; an administrator may need to resend the profile that carries it")
		}
	}
}
