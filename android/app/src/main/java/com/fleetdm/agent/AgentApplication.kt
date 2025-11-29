package com.fleetdm.agent

import android.app.Application
import android.content.Context
import android.content.RestrictionsManager
import android.os.Build
import android.util.Log
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch

/**
 * Custom Application class for Fleet Agent.
 * Runs when the app process starts (triggered by broadcasts, not by user).
 */
class AgentApplication : Application() {
    companion object {
        private const val TAG = "fleet-app"
        private const val CONFIG_CHECK_WORK_NAME = "config_check_periodic"
    }

    private val applicationScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)

    override fun onCreate() {
        super.onCreate()
        Log.i(TAG, "Fleet agent process started")
        ApiClient.initialize(this)
        refreshEnrollmentCredentials()
        schedulePeriodicConfigCheck()
    }

    private fun refreshEnrollmentCredentials() {
        applicationScope.launch {
            try {
                val restrictionsManager = getSystemService(Context.RESTRICTIONS_SERVICE)
                    as? RestrictionsManager
                val appRestrictions = restrictionsManager?.applicationRestrictions ?: return@launch

                val enrollSecret = appRestrictions.getString("enrollSecret")
                val hostUUID = appRestrictions.getString("hostUUID")
                val serverURL = appRestrictions.getString("serverURL")

                if (enrollSecret != null && hostUUID != null && serverURL != null) {
                    Log.d(TAG, "Refreshing enrollment credentials from MDM config")
                    ApiClient.setEnrollmentCredentials(
                        enrollSecret = enrollSecret,
                        hardwareUUID = hostUUID,
                        baseUrl = serverURL,
                        computerName = "${Build.BRAND} ${Build.MODEL}",
                    )

                    // Trigger auto-enrollment if node key is missing
                    // This also fetches initial orbit config
                    val configResult = ApiClient.getOrbitConfig()
                    configResult.onSuccess {
                        Log.d(TAG, "Successfully enrolled and fetched initial orbit config")
                    }.onFailure { error ->
                        Log.w(TAG, "Auto-enrollment on startup failed: ${error.message}")
                    }
                } else {
                    Log.d(TAG, "MDM enrollment credentials not available")
                }
            } catch (e: Exception) {
                Log.e(TAG, "Error refreshing enrollment credentials", e)
            }
        }
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
