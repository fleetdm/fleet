package com.fleetdm.agent

import android.content.Context
import android.util.Log
import androidx.work.Worker
import androidx.work.WorkerParameters

/**
 * WorkManager worker that periodically checks managed configurations.
 */
class ConfigCheckWorker(context: Context, params: WorkerParameters) : Worker(context, params) {
    companion object {
        private const val TAG = "fleet-worker"
    }

    override fun doWork(): Result {
        Log.i(TAG, "Periodic config check triggered")
        return Result.success()
    }
}
