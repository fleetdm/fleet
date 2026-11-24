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
import java.util.concurrent.ConcurrentHashMap

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
        private const val ON_CHANGE_WORK_NAME = "fleet_device_vitals_on_change"
        private const val TAG = "fleet-DeviceVitalsWorker"
        private const val MAX_RETRY_ATTEMPTS = 3

        /**
         * Reports as soon as a cellular data network's roaming state changes, rather than waiting for the next
         * periodic run. It only sees changes while cellular data is up, and only lives as long as the process
         * does; the periodic work is the backstop for both.
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
                    // Per network, so two SIMs with different roaming states don't re-trigger each other.
                    private val lastRoaming = ConcurrentHashMap<Network, Boolean>()

                    override fun onCapabilitiesChanged(network: Network, capabilities: NetworkCapabilities) {
                        val roaming = !capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_NOT_ROAMING)
                        if (lastRoaming.put(network, roaming) == roaming) return
                        enqueueReport(appContext)
                    }

                    override fun onLost(network: Network) {
                        lastRoaming.remove(network)
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
