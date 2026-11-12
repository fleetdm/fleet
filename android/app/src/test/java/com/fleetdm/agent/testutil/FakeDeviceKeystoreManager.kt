package com.fleetdm.agent.testutil

import com.fleetdm.agent.DeviceKeystoreManager

/**
 * Fake implementation of DeviceKeystoreManager for testing.
 * Provides configurable behavior for simulating keystore operations.
 */
class FakeDeviceKeystoreManager : DeviceKeystoreManager {
    val installedCerts = mutableSetOf<String>()
    var removeKeyPairShouldSucceed = true

    override fun hasKeyPair(alias: String): Boolean = alias in installedCerts

    override fun removeKeyPair(alias: String): Boolean {
        if (!removeKeyPairShouldSucceed) return false
        installedCerts.remove(alias)
        return true
    }

    fun installCert(alias: String) {
        installedCerts.add(alias)
    }

    fun reset() {
        installedCerts.clear()
        removeKeyPairShouldSucceed = true
    }
}
