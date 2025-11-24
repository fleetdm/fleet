package com.fleetdm.agent

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log

class RestrictionsReceiver : BroadcastReceiver() {
    private val TAG = "CertCompanionRestrict"
    private val CERT_DATA_KEY = "certificate_data"

    override fun onReceive(context: Context?, intent: Intent?) {
        if (intent?.action == Intent.ACTION_APPLICATION_RESTRICTIONS_CHANGED) {
            Log.i(TAG, "Application restrictions changed. Checking for new certificate data.")

            // Ensure context is not null before proceeding
            context?.let {
                // 1. Fetch the Managed Configuration (Application Restrictions)
                val restrictionsManager = context.getSystemService(Context.RESTRICTIONS_SERVICE) as android.content.RestrictionsManager
                val appRestrictions = restrictionsManager.applicationRestrictions

                val certData = appRestrictions.getString(CERT_DATA_KEY)

                if (!certData.isNullOrEmpty()) {
                    Log.d(TAG, "New certificate data found in restrictions.")

                    // 2. Start the service to handle the installation asynchronously
                    val serviceIntent = Intent(it, CertificateService::class.java).apply {
                        putExtra("CERT_DATA", certData)
                    }
                    it.startService(serviceIntent)
                } else {
                    Log.d(TAG, "No relevant certificate data found for key '$CERT_DATA_KEY'.")
                }
            }
        }
    }
}
