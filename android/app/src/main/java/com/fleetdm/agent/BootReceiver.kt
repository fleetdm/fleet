package com.fleetdm.agent

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import androidx.work.BackoffPolicy
import androidx.work.Constraints
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkRequest
import java.util.concurrent.TimeUnit

class BootReceiver : BroadcastReceiver() {
    companion object {
        private const val TAG = "fleet-boot"
    }

    override fun onReceive(context: Context?, intent: Intent?) {
        if (intent?.action == Intent.ACTION_BOOT_COMPLETED) {
            context?.let {
                Log.i(TAG, "Device boot completed. Triggering certificate enrollment.")
                // Trigger immediate certificate enrollment on boot
                val workRequest = OneTimeWorkRequestBuilder<CertificateEnrollmentWorker>()
                    .setBackoffCriteria(
                        BackoffPolicy.EXPONENTIAL,
                        WorkRequest.MIN_BACKOFF_MILLIS,
                        TimeUnit.MILLISECONDS,
                    )
                    .setConstraints(
                        Constraints.Builder()
                            .setRequiredNetworkType(NetworkType.CONNECTED)
                            .build(),
                    )
                    .build()

                WorkManager.getInstance(it)
                    .enqueueUniqueWork(
                        CertificateEnrollmentWorker.WORK_NAME,
                        ExistingWorkPolicy.KEEP, // Use same name as periodic worker to prevent concurrent runs
                        workRequest,
                    )

                Log.d(TAG, "Scheduled certificate enrollment after boot")
            } ?: Log.w(TAG, "Device boot completed but context is null, cannot schedule enrollment")
        }
    }
}
