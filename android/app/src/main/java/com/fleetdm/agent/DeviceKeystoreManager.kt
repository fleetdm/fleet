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
}

/**
 * Real implementation of DeviceKeystoreManager using DevicePolicyManager.
 */
class AndroidDeviceKeystoreManager(private val context: Context) : DeviceKeystoreManager {
    companion object {
        private const val TAG = "fleet-DeviceKeystoreManager"
    }

    private val dpm: DevicePolicyManager by lazy {
        context.getSystemService(Context.DEVICE_POLICY_SERVICE) as DevicePolicyManager
    }

    override fun hasKeyPair(alias: String): Boolean = try {
        dpm.hasKeyPair(alias)
    } catch (e: Exception) {
        Log.e(TAG, "Error checking if certificate '$alias' exists: ${e.message}", e)
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
                Log.e(TAG, "Failed to remove certificate keypair '$alias'. Check MDM policy and delegation status.")
            }
            removed
        }
    } catch (e: SecurityException) {
        Log.e(TAG, "Security exception removing certificate '$alias': ${e.message}", e)
        false
    } catch (e: Exception) {
        Log.e(TAG, "Error removing certificate '$alias': ${e.message}", e)
        false
    }
}
