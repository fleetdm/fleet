package com.fleetdm.agent

import android.content.Context
import android.util.Log
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters

/**
 * WorkManager worker that periodically checks managed configurations.
 */
class ConfigCheckWorker(context: Context, params: WorkerParameters) : CoroutineWorker(context, params) {

    companion object {
        private const val TAG = "fleet-worker"
    }

    override suspend fun doWork(): Result {
        Log.i(TAG, "Periodic config check triggered")

        val configResult = ApiClient.getOrbitConfig()
        configResult.onSuccess { config ->
            Log.d(TAG, "Successfully fetched orbit config")
        }.onFailure { error ->
            Log.e(TAG, "Failed to fetch orbit config: ${error.message}", error)
        }

        return Result.success()
    }
}
