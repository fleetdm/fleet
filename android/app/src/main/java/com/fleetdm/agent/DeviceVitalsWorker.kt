package com.fleetdm.agent

import android.content.Context
import android.content.pm.PackageManager
import android.telephony.ServiceState
import android.telephony.TelephonyCallback
import android.telephony.TelephonyManager
import android.util.Log
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters

/**
 * Reports the vitals Fleet can't get from AMAPI ([DeviceVitals]) to PUT /api/fleetd/vitals.
 */
class DeviceVitalsWorker(context: Context, workerParams: WorkerParameters) : CoroutineWorker(context, workerParams) {

    override suspend fun doWork(): Result {
        val vitals = DeviceVitals.collect(applicationContext)
        if (vitals.isEmpty()) {
            Log.d(TAG, "no vitals to report on this device")
            return Result.success()
        }

        return ApiClient.reportVitals(vitals).fold(
            onSuccess = { Result.success() },
            onFailure = { error ->
                when {
                    // A Fleet server older than this agent has no vitals endpoint; retrying won't help.
                    error is ApiClient.NotFoundException -> Result.success()
                    runAttemptCount >= MAX_RETRY_ATTEMPTS - 1 -> {
                        Log.w(TAG, "vitals report failed after $MAX_RETRY_ATTEMPTS attempts, waiting for the next periodic run")
                        Result.success()
                    }
                    else -> Result.retry()
                }
            },
        )
    }

    companion object {
        const val WORK_NAME = "fleet_device_vitals"
        internal const val ON_CHANGE_WORK_NAME = "fleet_device_vitals_on_change"
        private const val TAG = "fleet-DeviceVitalsWorker"
        private const val MAX_RETRY_ATTEMPTS = 3

        /**
         * Reports as soon as the roaming state RoamingReader reads changes, rather than waiting for the next
         * periodic run. Watching the phone's service state rather than the cellular data network catches changes
         * while data goes over Wi-Fi too. It only lives as long as the process does; the periodic work is the
         * backstop.
         */
        fun registerRoamingChangeCallback(context: Context) {
            val appContext = context.applicationContext
            if (!appContext.packageManager.hasSystemFeature(PackageManager.FEATURE_TELEPHONY)) return
            val telephonyManager = appContext.getSystemService(TelephonyManager::class.java) ?: return
            telephonyManager.registerTelephonyCallback(
                appContext.mainExecutor,
                object : TelephonyCallback(), TelephonyCallback.ServiceStateListener {
                    private var lastRoaming: Boolean? = null

                    override fun onServiceStateChanged(serviceState: ServiceState) {
                        val roaming = serviceState.roaming
                        if (roaming == lastRoaming) return
                        lastRoaming = roaming
                        enqueueReport(appContext)
                    }
                },
            )
        }

        private fun enqueueReport(context: Context) {
            val workRequest = OneTimeWorkRequestBuilder<DeviceVitalsWorker>()
                .setConstraints(
                    Constraints.Builder()
                        .setRequiredNetworkType(NetworkType.CONNECTED)
                        .build(),
                ).build()
            WorkManager.getInstance(context)
                .enqueueUniqueWork(ON_CHANGE_WORK_NAME, ExistingWorkPolicy.REPLACE, workRequest)
        }
    }
}
