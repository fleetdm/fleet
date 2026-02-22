package com.fleetdm.agent

import android.content.Context
import android.util.Log
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import java.util.concurrent.TimeUnit

class DistributedCheckinWorker(
    appContext: Context,
    params: WorkerParameters,
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        try {
            Log.d(TAG, "Distributed check-in: starting")

            val result = ApiClient.distributedRead()
            result.fold(
                onSuccess = { resp ->
                    val count = resp.queries.size
                    Log.d(TAG, "Distributed check-in: received $count query(ies)")
                },
                onFailure = { err ->
                    Log.w(TAG, "Distributed check-in: failed: ${err.message}")
                },
            )

            return Result.success()
        } finally {
            // Debug-only fast polling (15s). Release scheduling handled elsewhere (15m).
            if (BuildConfig.DEBUG) {
                scheduleNextDebug(applicationContext)
                Log.d(TAG, "Distributed check-in: scheduled next run in 15 seconds")
            }
        }
    }

    companion object {
        private const val TAG = "fleet-distributed"

        /** Single logical chain for debug polling */
        private const val WORK_NAME_DEBUG = "fleet_distributed_checkin_debug"

        fun scheduleNextDebug(context: Context) {
            val request = OneTimeWorkRequestBuilder<DistributedCheckinWorker>()
                .setInitialDelay(15, TimeUnit.SECONDS)
                .setConstraints(
                    Constraints.Builder()
                        .setRequiredNetworkType(NetworkType.CONNECTED)
                        .build()
                )
                .addTag(WORK_NAME_DEBUG)
                .build()

            WorkManager.getInstance(context)
                .beginUniqueWork(
                    WORK_NAME_DEBUG,
                    ExistingWorkPolicy.APPEND, // <-- critical: do NOT cancel running work
                    request
                )
                .enqueue()
        }
    }
}
