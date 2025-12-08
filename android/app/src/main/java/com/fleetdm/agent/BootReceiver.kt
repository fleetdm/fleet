package com.fleetdm.agent

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import androidx.work.Constraints
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager

class BootReceiver : BroadcastReceiver() {
    companion object {
        private const val TAG = "fleet-boot"
    }

    override fun onReceive(context: Context?, intent: Intent?) {
        if (intent?.action == Intent.ACTION_BOOT_COMPLETED) {
            Log.i(TAG, "Device boot completed. Triggering certificate enrollment.")

            context?.let {
                // Trigger immediate certificate enrollment on boot
                val workRequest = OneTimeWorkRequestBuilder<CertificateEnrollmentWorker>()
                    .setConstraints(
                        Constraints.Builder()
                            .setRequiredNetworkType(NetworkType.CONNECTED)
                            .build(),
                    )
                    .build()

                WorkManager.getInstance(it)
                    .enqueueUniqueWork(
                        "${CertificateEnrollmentWorker.WORK_NAME}_boot",
                        ExistingWorkPolicy.REPLACE, // Run fresh enrollment on boot
                        workRequest,
                    )

                Log.d(TAG, "Scheduled certificate enrollment after boot")
            }
        }
    }
}
