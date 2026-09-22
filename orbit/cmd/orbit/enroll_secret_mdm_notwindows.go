//go:build !windows

package main

// The registry channel Fleet Windows MDM uses to deliver an enroll secret has no counterpart here.
//
// macOS does have its own out-of-band delivery, a fleetd configuration profile carrying the secret and the Fleet URL, but orbit
// reads that directly in orbitAction under --use-system-configuration rather than through these functions. Everywhere else orbit
// resolves the secret from the packaged file and the keystore.

// adoptMDMSecretIfWaiting reports that nothing was waiting. The channel counts as usable so callers do not log or branch on a
// platform that simply routes its delivery elsewhere.
func adoptMDMSecretIfWaiting(_ string, _ bool, _ func(string) error) (adopted bool, channelUsable bool) {
	return false, true
}

// canWaitForMDMSecret is always false: there is no registry to watch. macOS waits on its configuration profile in orbitAction
// instead, so nothing is lost by answering no here.
func canWaitForMDMSecret(_ bool, _ string) bool { return false }

// waitForMDMDeliveredEnrollSecret never blocks, and canWaitForMDMSecret means it is never reached.
func waitForMDMDeliveredEnrollSecret(_ <-chan struct{}, _ string, _ bool, _ func(string) error) {}
