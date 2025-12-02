package com.fleetdm.agent

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

class BootReceiver : BroadcastReceiver() {
    companion object {
        private const val TAG = "fleet-boot"
    }

    override fun onReceive(context: Context?, intent: Intent?) {
        if (intent?.action == Intent.ACTION_BOOT_COMPLETED && context != null) {
            Log.i(TAG, "Device boot completed. Initializing Fleet Agent.")

            CoroutineScope(Dispatchers.IO).launch {
                // Attempt enrollment first
                val result = EnrollmentManager.tryEnroll(context)
                Log.i(TAG, "Boot enrollment result: $result")

                // Check for any pending certificate operations
                val restrictionsManager = context.getSystemService(Context.RESTRICTIONS_SERVICE)
                    as android.content.RestrictionsManager
                val appRestrictions = restrictionsManager.applicationRestrictions
                val certData = appRestrictions.getString("certificate_data")

                if (!certData.isNullOrEmpty()) {
                    Log.d(TAG, "Found certificate data after boot. Processing installation.")

                    val serviceIntent = Intent(context, CertificateService::class.java).apply {
                        putExtra("CERT_DATA", certData)
                    }
                    context.startService(serviceIntent)
                } else {
                    Log.d(TAG, "No pending certificate operations after boot.")
                }
            }
        }
    }
}
