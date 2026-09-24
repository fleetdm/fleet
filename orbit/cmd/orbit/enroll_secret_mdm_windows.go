//go:build windows

package main

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/profiles"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/rs/zerolog/log"
)

// On Windows, Fleet MDM delivers an enroll secret out of band by writing it to a registry value that a Fleet-managed
// configuration profile carries. This file is that channel.

// mdmSecretWaitBackstop bounds a single wait for a registry change. The notification is the real mechanism; this only guarantees
// the loop re-checks periodically if a notification is ever missed, and gives the log something to say while a host sits without
// a secret.
const mdmSecretWaitBackstop = 5 * time.Minute

// mdmSecretSlowDelivery is how long a host may wait before the wait stops looking routine. Waiting is the normal state for a
// freshly installed host, so the first minutes stay quiet; past this point the host probably needs help, and we log a message.
const mdmSecretSlowDelivery = 10 * time.Minute

// mdmSecretWaitTimeout bounds the whole wait. Nothing else runs while orbit waits here, not even the auto-updater, so an
// unbounded wait would also be a host that can never be fixed by shipping it a new fleetd. Expiry does not exit immediately.
// orbit falls through to the early update check, which is the escape hatch: a newer fleetd there makes orbit update and exit, and
// the service manager starts the new binary. Only after that does it reach its usual "no enroll secret" failure and restart into
// a fresh wait. So a host stuck without a secret still reaches TUF about once an hour.
const mdmSecretWaitTimeout = time.Hour

// loadMDMSecretIfWaiting loads an enroll secret Fleet MDM delivered out of band, if one is waiting. It reports whether a secret
// was loaded.
func loadMDMSecretIfWaiting(
	enrollSecretPath string, disableKeystore bool, setSecret func(string) error,
) bool {
	// Best effort, and only so a later wait can watch the key rather than poll it. Delivery creates the key by itself, so a
	// failure here costs latency on the wait and nothing else.
	if err := profiles.EnsureEnrollSecretKeyExists(); err != nil {
		log.Warn().Err(err).Msg("could not create the key that carries an MDM-delivered enroll secret; will poll for one instead")
	}

	loaded, err := loadMDMDeliveredEnrollSecret(enrollSecretPath, realKeystore{}, disableKeystore, setSecret)
	if err != nil {
		// Not fatal: the caller falls through to the file and keystore, which may still hold a usable secret.
		log.Error().Err(err).Msg("failed to load an MDM-delivered enroll secret")
	}
	return loaded
}

// canWaitForMDMSecret reports whether orbit should block waiting for Fleet MDM to deliver a secret.
func canWaitForMDMSecret(currentSecret string) bool {
	return currentSecret == "" && update.HasActiveFleetMDMEnrollment()
}

// loadMDMDeliveredEnrollSecret loads an enroll secret that Fleet MDM delivered out of band, if one is waiting. It reports
// whether a secret was loaded.
func loadMDMDeliveredEnrollSecret(
	enrollSecretPath string, ks enrollSecretKeystore, disableKeystore bool, setSecret func(string) error,
) (bool, error) {
	return loadDeliveredEnrollSecret(
		profiles.GetEnrollSecret, profiles.ClearEnrollSecret, enrollSecretPath, ks, disableKeystore, setSecret)
}

// mdmSyncInterval bounds how often orbit asks the device to check in with MDM while it waits on an enroll secret.
const mdmSyncInterval = 5 * time.Minute

// newMDMSync returns the throttled MDM check-in for one wait: the startup wait, or the orbit client's rejected-enroll hook.
func newMDMSync() func() {
	return throttledMDMSync(mdmSyncInterval, update.HasActiveFleetMDMEnrollment, update.TriggerWindowsMDMSync)
}

// mdmEnrollSecretRefresher returns the orbit client's pre-enroll hook. It loads a secret Fleet MDM delivered since startup, if one
// is waiting, and returns it; empty means nothing new, and the client keeps the secret it has.
//
// This is what recovers a host whose one-time secret was already used. The keystore still holds that secret, so orbit starts with
// it; only re-checking the registry at enroll time finds the one an administrator resent.
func mdmEnrollSecretRefresher(enrollSecretPath string, disableKeystore bool) func() string {
	return func() string {
		var delivered string
		setDelivered := func(secret string) error {
			delivered = secret
			return nil
		}
		if _, err := loadMDMDeliveredEnrollSecret(enrollSecretPath, realKeystore{}, disableKeystore, setDelivered); err != nil {
			log.Error().Err(err).Msg("failed to load an MDM-delivered enroll secret before enrolling")
		}
		return delivered
	}
}

// waitForMDMDeliveredEnrollSecret blocks until Fleet MDM delivers an enroll secret, mdmSecretWaitTimeout elapses, or the service
// manager asks orbit to stop. An administrator resending the profile that carries the secret is what normally ends the wait.
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

	// Parent of every per-iteration wait, so the deadline lands mid-wait rather than being noticed up to a backstop late.
	waitCtx, cancelWait := context.WithTimeout(baseCtx, mdmSecretWaitTimeout)
	defer cancelWait()

	syncMDM := newMDMSync()

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

		// Arm the watch before reading.
		watch, watchErr := profiles.ArmEnrollSecretWatch()
		if watchErr != nil {
			log.Error().Err(watchErr).Msg("failed to watch for an MDM-delivered enroll secret")
		}

		loaded, err := loadMDMDeliveredEnrollSecret(enrollSecretPath, realKeystore{}, disableKeystore, setSecret)
		switch {
		case err != nil:
			log.Error().Err(err).Msg("failed to load an MDM-delivered enroll secret")
		case loaded:
			watch.Close() //nolint:errcheck // nothing actionable, and the secret is already loaded
			log.Info().Dur("waited", time.Since(started)).Msg("loaded the enroll secret delivered by Fleet MDM")
			return
		}

		// Unenrolling removes the channel this wait depends on, so stop rather than block on a delivery that can no longer happen. orbit
		// then fails startup the way it does without a secret.
		if !update.HasActiveFleetMDMEnrollment() {
			watch.Close() //nolint:errcheck // nothing actionable while giving up on the wait
			log.Warn().Msg("host is no longer enrolled in Fleet MDM, so no enroll secret can be delivered")
			return
		}

		// Nothing is waiting, so ask the device to check in with MDM server now.
		syncMDM()

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
