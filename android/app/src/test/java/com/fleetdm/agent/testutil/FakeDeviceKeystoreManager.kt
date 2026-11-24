package com.fleetdm.agent.testutil

import com.fleetdm.agent.DeviceKeystoreManager

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

    override fun hasCaCert(certBytes: ByteArray): Boolean =
        java.security.MessageDigest.getInstance("SHA-256").digest(certBytes).joinToString("") { "%02x".format(it) } in installedCaCerts

    override fun installCaCert(certBytes: ByteArray): Boolean {
        if (!installCaCertShouldSucceed) return false
        val hash = java.security.MessageDigest.getInstance("SHA-256").digest(certBytes).joinToString("") { "%02x".format(it) }
        installedCaCerts.add(hash)
        return true
    }

    override fun uninstallCaCert(certBytes: ByteArray): Boolean {
        if (!uninstallCaCertShouldSucceed) return false
        val hash = java.security.MessageDigest.getInstance("SHA-256").digest(certBytes).joinToString("") { "%02x".format(it) }
        installedCaCerts.remove(hash)
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
