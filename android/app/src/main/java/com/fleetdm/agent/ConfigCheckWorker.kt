package com.fleetdm.agent

import android.content.Context
import android.util.Log
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters

/**
 * WorkManager worker that periodically checks managed configurations
 * and attempts enrollment if not already enrolled.
 */
class ConfigCheckWorker(context: Context, params: WorkerParameters) : CoroutineWorker(context, params) {
    companion object {
        private const val TAG = "fleet-worker"
    }

    override suspend fun doWork(): Result {
        Log.i(TAG, "Periodic config check triggered")

        val enrollmentResult = EnrollmentManager.tryEnroll(applicationContext)
        Log.i(TAG, "Enrollment result: $enrollmentResult")

        return Result.success()
    }
}
