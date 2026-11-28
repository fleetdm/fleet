package com.fleetdm.agent

import android.content.Context
import android.content.pm.PackageManager
import android.telephony.TelephonyManager

/**
 * Reads whether the device's cellular radio is registered on a roaming network.
 *
 * AMAPI doesn't expose roaming, so the agent reports it to Fleet itself. TelephonyManager.isNetworkRoaming
 * needs no runtime permission and reflects registration on the default subscription even while data goes
 * over Wi-Fi, which is what an admin asking "is this device roaming?" wants.
 */
object RoamingReader {
    /** Returns null on devices without telephony (Wi-Fi-only tablets), where roaming doesn't apply. */
    fun read(context: Context): Boolean? {
        if (!context.packageManager.hasSystemFeature(PackageManager.FEATURE_TELEPHONY)) {
            return null
        }
        val telephonyManager = context.getSystemService(TelephonyManager::class.java) ?: return null
        return telephonyManager.isNetworkRoaming
    }
}
