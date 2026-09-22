//go:build !windows

package main

// The registry channel Fleet Windows MDM uses to deliver an enroll secret has no counterpart here. macOS does have its own
// out-of-band delivery, a fleetd configuration profile carrying the secret and the Fleet URL, but orbit reads that directly in
// orbitAction under --use-system-configuration rather than through these functions.

// loadMDMSecretIfWaiting reports that nothing was waiting.
func loadMDMSecretIfWaiting(_ string, _ bool, _ func(string) error) (loaded bool, channelUsable bool) {
	return false, true
}

// canWaitForMDMSecret is always false.
func canWaitForMDMSecret(_ bool, _ string) bool { return false }

// waitForMDMDeliveredEnrollSecret never blocks, and canWaitForMDMSecret means it is never reached.
func waitForMDMDeliveredEnrollSecret(_ <-chan struct{}, _ string, _ bool, _ func(string) error) {}
