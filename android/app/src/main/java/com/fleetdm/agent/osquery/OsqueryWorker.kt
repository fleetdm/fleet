package com.fleetdm.agent.osquery

import android.content.Context
import android.util.Log
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import androidx.work.Constraints
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import java.util.concurrent.TimeUnit
import com.fleetdm.agent.BuildConfig

import kotlin.random.Random


class OsqueryWorker(
    appContext: Context,
    params: WorkerParameters,
) : CoroutineWorker(appContext, params) {

    override suspend fun doWork(): Result {
        Log.i("FleetOsquery", "OsqueryWorker doWork start")

        OsqueryTables.registerAll(applicationContext)

        return try {
            FleetDistributedQueryRunner.runOnce(applicationContext)

            scheduleNext()
            Result.success()
        } catch (e: IllegalArgumentException) {
            Log.e("FleetOsquery", "OsqueryWorker misconfigured: ${e.message}", e)
            // Permanent failure: do not reschedule
            return Result.failure()
        } catch (e: Exception) {
            Log.e("FleetOsquery", "OsqueryWorker error", e)

            scheduleNext()
            Result.retry()
        }
    }

    private fun scheduleNext() {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()

        val baseDelaySeconds =
            if (BuildConfig.DEBUG) 5L else 60L

        val jitterSeconds =
            if (BuildConfig.DEBUG) 0L else Random.nextLong(from = 0L, until = 15L)

        val delaySeconds = baseDelaySeconds + jitterSeconds

        val next = OneTimeWorkRequestBuilder<OsqueryWorker>()
            .setConstraints(constraints)
            .setInitialDelay(delaySeconds, TimeUnit.SECONDS)
            .build()

        WorkManager.getInstance(applicationContext).enqueueUniqueWork(
            "fleetOsqueryLoop",
            ExistingWorkPolicy.REPLACE,
            next,
        )
    }


}
