package com.fleetdm.agent.osquery

import android.content.Context
import android.util.Log
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import com.fleetdm.agent.BuildConfig
import java.util.concurrent.TimeUnit
import kotlin.random.Random


class OsqueryWorker(
    appContext: Context,
    params: WorkerParameters,
) : CoroutineWorker(appContext, params) {

    companion object {
        private val lock = Any()
        private var running = false
    }

    override suspend fun doWork(): Result {
        synchronized(lock) {
            if (running) {
                Log.i("FleetOsquery", "OsqueryWorker already running, skipping")
                return Result.success()
            }
            running = true
        }

        try {
            Log.i("FleetOsquery", "OsqueryWorker doWork start")

            OsqueryTables.registerAll(applicationContext)
            Log.i("FleetOsquery", "About to call FleetDistributedQueryRunner.runOnce()")
            FleetDistributedQueryRunner.runOnce()
            Log.i("FleetOsquery", "FleetDistributedQueryRunner.runOnce() finished")

            scheduleNext()
            return Result.success()
        } catch (e: IllegalArgumentException) {
            Log.e("FleetOsquery", "OsqueryWorker misconfigured: ${e.message}", e)
            return Result.failure()
        } catch (e: Exception) {
            Log.e("FleetOsquery", "OsqueryWorker error", e)
            scheduleNext()
            return Result.retry()
        } finally {
            synchronized(lock) { running = false }
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
