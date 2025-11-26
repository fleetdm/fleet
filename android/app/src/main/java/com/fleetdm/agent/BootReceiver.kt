package com.fleetdm.agent

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log

class BootReceiver : BroadcastReceiver() {
    private val TAG = "CertCompanionBoot"

    override fun onReceive(context: Context?, intent: Intent?) {
        if (intent?.action == Intent.ACTION_BOOT_COMPLETED) {
            Log.i(TAG, "Device boot completed. Initializing Fleet Agent.")

            context?.let {
                // Check for any pending certificate operations or managed configurations
                // that may need to be processed after boot
                val restrictionsManager = context.getSystemService(Context.RESTRICTIONS_SERVICE) as android.content.RestrictionsManager
                val appRestrictions = restrictionsManager.applicationRestrictions

                val certData = appRestrictions.getString("certificate_data")

                if (!certData.isNullOrEmpty()) {
                    Log.d(TAG, "Found certificate data after boot. Processing installation.")

                    // Start the service to handle the installation
                    val serviceIntent = Intent(it, CertificateService::class.java).apply {
                        putExtra("CERT_DATA", certData)
                    }
                    it.startService(serviceIntent)
                } else {
                    Log.d(TAG, "No pending certificate operations after boot.")
                }
            }
        }
    }
}
