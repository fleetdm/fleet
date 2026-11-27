package com.fleetdm.agent.testutil

import com.fleetdm.agent.DeviceKeystoreManager
import java.security.MessageDigest

/**
 * Fake implementation of DeviceKeystoreManager for testing.
 * Provides configurable behavior for simulating keystore operations.
 */
class FakeDeviceKeystoreManager : DeviceKeystoreManager {
    val installedCerts = mutableSetOf<String>()
    val installedCaCerts = mutableSetOf<String>()
    var removeKeyPairShouldSucceed = true
    var installCaCertShouldSucceed = true
    var uninstallCaCertShouldSucceed = true

    override fun hasKeyPair(alias: String): Boolean = alias in installedCerts

    override fun removeKeyPair(alias: String): Boolean {
        if (!removeKeyPairShouldSucceed) return false
        installedCerts.remove(alias)
        return true
    }

    private fun sha256Hex(bytes: ByteArray): String =
        MessageDigest.getInstance("SHA-256").digest(bytes).joinToString("") { "%02x".format(it) }

    override fun hasCaCert(certBytes: ByteArray): Boolean = sha256Hex(certBytes) in installedCaCerts

    override fun installCaCert(certBytes: ByteArray): Boolean {
        if (!installCaCertShouldSucceed) return false
        installedCaCerts.add(sha256Hex(certBytes))
        return true
    }

    override fun uninstallCaCert(certBytes: ByteArray): Boolean {
        if (!uninstallCaCertShouldSucceed) return false
        installedCaCerts.remove(sha256Hex(certBytes))
        return true
    }

    fun installCert(alias: String) {
        installedCerts.add(alias)
    }

    fun reset() {
        installedCerts.clear()
        installedCaCerts.clear()
        removeKeyPairShouldSucceed = true
        installCaCertShouldSucceed = true
        uninstallCaCertShouldSucceed = true
    }
}
