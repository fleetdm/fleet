package com.fleetdm.agent.osquery

import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.BatteryManager
import android.os.Build
import android.os.PowerManager
import android.util.Log
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters
import com.fleetdm.agent.AndroidOrbitConfig
import com.fleetdm.agent.ApiClient
import com.fleetdm.agent.BuildConfig
import java.util.concurrent.TimeUnit
import kotlin.random.Random


class OsqueryWorker(
    appContext: Context,
    params: WorkerParameters,
) : CoroutineWorker(appContext, params) {

    companion object {
        private const val TAG = "FleetOsquery"
        private const val WORK_NAME = "fleetOsqueryLoop"

        private val lock = Any()
        private var running = false

        fun scheduleNow(context: Context) {
            val request = OneTimeWorkRequestBuilder<OsqueryWorker>()
                .setConstraints(
                    Constraints.Builder()
                        .setRequiredNetworkType(NetworkType.CONNECTED)
                        .build()
                )
                .build()

            WorkManager.getInstance(context).enqueueUniqueWork(
                WORK_NAME,
                ExistingWorkPolicy.REPLACE,
                request,
            )
        }
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
            Log.i(TAG, "OsqueryWorker doWork start")

            val androidConfig = ApiClient.getOrbitConfig()
                .onFailure { Log.w(TAG, "Failed to fetch Orbit config, using local defaults: ${it.message}") }
                .getOrNull()
                ?.android
                ?: AndroidOrbitConfig()

            OsqueryTables.registerAll(applicationContext)
            Log.i(TAG, "About to call FleetDistributedQueryRunner.runOnce()")
            FleetDistributedQueryRunner.runOnce()
            Log.i(TAG, "FleetDistributedQueryRunner.runOnce() finished")

            scheduleNext(androidConfig)
            return Result.success()
        } catch (e: IllegalArgumentException) {
            Log.e(TAG, "OsqueryWorker misconfigured: ${e.message}", e)
            return Result.failure()
        } catch (e: Exception) {
            Log.e(TAG, "OsqueryWorker error", e)
            scheduleNext(AndroidOrbitConfig())
            return Result.retry()
        } finally {
            synchronized(lock) { running = false }
        }
    }

    private fun scheduleNext(config: AndroidOrbitConfig) {
        val constraints = Constraints.Builder()
            .setRequiredNetworkType(NetworkType.CONNECTED)
            .build()

        val baseDelaySeconds =
            if (BuildConfig.DEBUG) 5L else chooseDelaySeconds(config)

        val jitterSeconds =
            if (BuildConfig.DEBUG || baseDelaySeconds < 60L) 0L else Random.nextLong(from = 0L, until = 15L)

        val delaySeconds = baseDelaySeconds + jitterSeconds

        val next = OneTimeWorkRequestBuilder<OsqueryWorker>()
            .setConstraints(constraints)
            .setInitialDelay(delaySeconds, TimeUnit.SECONDS)
            .build()

        WorkManager.getInstance(applicationContext).enqueueUniqueWork(
            WORK_NAME,
            ExistingWorkPolicy.REPLACE,
            next,
        )
    }

    private fun chooseDelaySeconds(config: AndroidOrbitConfig): Long {
        val powerManager = applicationContext.getSystemService(Context.POWER_SERVICE) as? PowerManager
        val isBatterySaver = powerManager?.isPowerSaveMode == true
        val isDeviceIdle = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            powerManager?.isDeviceIdleMode == true
        } else {
            false
        }
        val isInteractive = powerManager?.isInteractive ?: true
        val isCharging = isCharging()

        return when {
            isBatterySaver -> config.batterySaverIntervalSeconds.toLong()
            isDeviceIdle -> config.idleIntervalSeconds.toLong()
            isCharging -> config.chargingIntervalSeconds.toLong()
            !isInteractive -> config.screenOffIntervalSeconds.toLong()
            else -> config.distributedReadIntervalSeconds.toLong()
        }
    }

    private fun isCharging(): Boolean {
        val batteryIntent = applicationContext.registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
        val status = batteryIntent?.getIntExtra(BatteryManager.EXTRA_STATUS, -1) ?: -1
        return status == BatteryManager.BATTERY_STATUS_CHARGING ||
            status == BatteryManager.BATTERY_STATUS_FULL
    }
}
