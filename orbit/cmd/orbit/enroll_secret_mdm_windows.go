//go:build windows

package main

import (
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/profiles"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/rs/zerolog/log"
)

// On Windows, Fleet MDM delivers an enroll secret out of band by writing it to a registry value that a Fleet-managed
// configuration profile carries. This file is that channel.

// mdmSyncInterval bounds how often a rejected enroll attempt asks the device to check in with MDM.
const mdmSyncInterval = 5 * time.Minute

// loadMDMSecretIfWaiting loads an enroll secret Fleet MDM delivered out of band, if one is waiting. It reports whether a secret
// was loaded.
func loadMDMSecretIfWaiting(
	enrollSecretPath string, disableKeystore bool, setSecret func(string) error,
) bool {
	loaded, err := loadMDMDeliveredEnrollSecret(enrollSecretPath, realKeystore{}, disableKeystore, setSecret)
	if err != nil {
		// Not fatal: the caller falls through to the file and keystore, which may still hold a usable secret.
		log.Error().Err(err).Msg("failed to load an MDM-delivered enroll secret")
	}
	return loaded
}

// loadMDMDeliveredEnrollSecret loads an enroll secret that Fleet MDM delivered out of band, if one is waiting. It reports
// whether a secret was loaded.
func loadMDMDeliveredEnrollSecret(
	enrollSecretPath string, ks enrollSecretKeystore, disableKeystore bool, setSecret func(string) error,
) (bool, error) {
	return loadDeliveredEnrollSecret(
		profiles.GetEnrollSecret, profiles.ClearEnrollSecret, enrollSecretPath, ks, disableKeystore, setSecret)
}

// newMDMSync returns the orbit client's hook for a rejected enroll attempt: a throttled MDM check-in.
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
