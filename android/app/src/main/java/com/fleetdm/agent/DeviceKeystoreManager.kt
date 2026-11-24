package com.fleetdm.agent

import android.app.admin.DevicePolicyManager
import android.content.Context
import android.util.Log

/**
 * Interface for device keystore operations (certificate management via DevicePolicyManager).
 * Abstracted to allow mocking in tests.
 */
interface DeviceKeystoreManager {
    /**
     * Check if a keypair with the given alias exists in the keystore.
     */
    fun hasKeyPair(alias: String): Boolean

    /**
     * Remove a keypair with the given alias from the keystore.
     * @return True if removal was successful or keypair doesn't exist
     */
    fun removeKeyPair(alias: String): Boolean

    /**
     * Check if a CA certificate is installed.
     */
    fun hasCaCert(certBytes: ByteArray): Boolean

    /**
     * Install a CA certificate.
     * @return True if installation was successful
     */
    fun installCaCert(certBytes: ByteArray): Boolean

    /**
     * Uninstall a CA certificate.
     * @return True if uninstallation was successful
     */
    fun uninstallCaCert(certBytes: ByteArray): Boolean
}

/**
 * Real implementation of DeviceKeystoreManager using DevicePolicyManager.
 */
class AndroidDeviceKeystoreManager(private val context: Context) : DeviceKeystoreManager {
    companion object {
        private const val TAG = "fleet-DeviceKeystoreManager"
    }

    private val dpm: DevicePolicyManager by lazy {
        context.getSystemService(DevicePolicyManager::class.java)
            ?: error("DevicePolicyManager not available")
    }

    override fun hasKeyPair(alias: String): Boolean = try {
        dpm.hasKeyPair(alias)
    } catch (e: Exception) {
        FleetLog.e(TAG, "Error checking if certificate '$alias' exists: ${e.message}", e)
        false
    }

    override fun removeKeyPair(alias: String): Boolean = try {
        if (!dpm.hasKeyPair(alias)) {
            Log.i(TAG, "Certificate '$alias' doesn't exist in keystore, considering removal successful")
            true
        } else {
            val removed = dpm.removeKeyPair(null, alias)
            if (removed) {
                Log.i(TAG, "Successfully removed certificate keypair with alias: $alias")
            } else {
                FleetLog.e(TAG, "Failed to remove certificate keypair '$alias'. Check MDM policy and delegation status.")
            }
            removed
        }
    } catch (e: SecurityException) {
        FleetLog.e(TAG, "Security exception removing certificate '$alias': ${e.message}", e)
        false
    } catch (e: Exception) {
        FleetLog.e(TAG, "Error removing certificate '$alias': ${e.message}", e)
        false
    }

    override fun hasCaCert(certBytes: ByteArray): Boolean = try {
        dpm.hasCaCertInstalled(null, certBytes)
    } catch (e: Exception) {
        FleetLog.e(TAG, "Error checking if CA certificate is installed: ${e.message}", e)
        false
    }

    override fun installCaCert(certBytes: ByteArray): Boolean = try {
        val installed = dpm.installCaCert(null, certBytes)
        if (installed) {
            Log.i(TAG, "Successfully installed CA certificate")
        } else {
            FleetLog.e(TAG, "Failed to install CA certificate. Check MDM policy and delegation status.")
        }
        installed
    } catch (e: SecurityException) {
        FleetLog.e(TAG, "Security exception installing CA certificate: ${e.message}", e)
        false
    } catch (e: Exception) {
        FleetLog.e(TAG, "Error installing CA certificate: ${e.message}", e)
        false
    }

    override fun uninstallCaCert(certBytes: ByteArray): Boolean = try {
        dpm.uninstallCaCert(null, certBytes)
        Log.i(TAG, "Successfully uninstalled CA certificate")
        true
    } catch (e: Exception) {
        FleetLog.e(TAG, "Error uninstalling CA certificate: ${e.message}", e)
        false
    }
}
