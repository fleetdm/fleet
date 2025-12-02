package com.fleetdm.agent

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log

class BootReceiver : BroadcastReceiver() {
    companion object {
        private const val TAG = "fleet-boot"
    }

    override fun onReceive(context: Context?, intent: Intent?) {
        if (intent?.action == Intent.ACTION_BOOT_COMPLETED) {
            Log.i(TAG, "Device boot completed. Initializing Fleet Agent.")

            context?.let {
                // Check for any pending certificate operations from managed configuration
                val certificateIds = CertificateOrchestrator.getCertificateIDs(it)

                if (!certificateIds.isNullOrEmpty()) {
                    Log.d(TAG, "Found ${certificateIds.size} certificate(s) after boot. Processing first certificate.")

                    // Start the service to handle the first certificate
                    val serviceIntent = Intent(it, CertificateService::class.java).apply {
                        putExtra("CERTIFICATE_ID", certificateIds.first())
                    }
                    it.startService(serviceIntent)
                } else {
                    Log.d(TAG, "No pending certificate operations after boot.")
                }
            }
        }
    }
}
