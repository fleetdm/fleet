package com.fleetdm.agent

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.util.Log
import androidx.work.Constraints
import androidx.work.CoroutineWorker
import androidx.work.ExistingWorkPolicy
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkerParameters

/**
 * Reports the vitals Fleet can't get from AMAPI (currently roaming) to PUT /api/fleetd/vitals.
 */
class DeviceVitalsWorker(context: Context, workerParams: WorkerParameters) : CoroutineWorker(context, workerParams) {

    override suspend fun doWork(): Result {
        val roaming = RoamingReader.read(applicationContext)
        if (roaming == null) {
            Log.d(TAG, "device has no telephony, skipping vitals report")
            return Result.success()
        }

        return ApiClient.reportVitals(roaming = roaming).fold(
            onSuccess = { Result.success() },
            onFailure = { Result.retry() },
        )
    }

    companion object {
        const val WORK_NAME = "fleet_device_vitals"
        private const val ON_CHANGE_WORK_NAME = "fleet_device_vitals_on_change"
        private const val TAG = "fleet-DeviceVitalsWorker"

        /**
         * Reports as soon as the cellular network's roaming state changes, rather than waiting for the next
         * periodic run. Only lives as long as the process does; the periodic work is the backstop.
         */
        fun registerRoamingChangeCallback(context: Context) {
            val appContext = context.applicationContext
            val connectivityManager = appContext.getSystemService(ConnectivityManager::class.java) ?: return
            val request = NetworkRequest.Builder()
                .addTransportType(NetworkCapabilities.TRANSPORT_CELLULAR)
                .build()
            connectivityManager.registerNetworkCallback(
                request,
                object : ConnectivityManager.NetworkCallback() {
                    private var lastRoaming: Boolean? = null

                    override fun onCapabilitiesChanged(network: Network, capabilities: NetworkCapabilities) {
                        val roaming = !capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_NOT_ROAMING)
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
