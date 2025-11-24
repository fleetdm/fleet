package com.fleetdm.agent

import android.app.Application
import android.util.Log
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import java.util.concurrent.TimeUnit

/**
 * Custom Application class for Fleet Agent.
 * Runs when the app process starts (triggered by broadcasts, not by user).
 */
class FleetApplication : Application() {
    companion object {
        private const val TAG = "fleet-app"
        private const val CONFIG_CHECK_WORK_NAME = "config_check_periodic"
    }

    override fun onCreate() {
        super.onCreate()
        Log.i(TAG, "Fleet Agent process started")
        schedulePeriodicConfigCheck()
    }

    private fun schedulePeriodicConfigCheck() {
        val workRequest =
            PeriodicWorkRequestBuilder<ConfigCheckWorker>(
                15, // 15 is the minimum
                TimeUnit.MINUTES,
            ).build()

        WorkManager
            .getInstance(this)
            .enqueueUniquePeriodicWork(
                CONFIG_CHECK_WORK_NAME,
                ExistingPeriodicWorkPolicy.KEEP,
                workRequest,
            )

        Log.i(TAG, "Scheduled periodic config check every 15 minutes")
    }
}
