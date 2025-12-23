package com.fleetdm.agent

import android.app.Application
import android.content.Context
import android.content.RestrictionsManager
import android.os.Build
import android.util.Log
import androidx.work.Constraints
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.NetworkType
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
    /** Certificate orchestrator instance for the app */
    lateinit var certificateOrchestrator: CertificateOrchestrator
        private set

    companion object {
        private const val TAG = "fleet-app"

        /**
         * Gets the CertificateOrchestrator instance from the Application.
         * @param context Any context (will use applicationContext)
         * @return The shared CertificateOrchestrator instance
         */
        fun getCertificateOrchestrator(context: Context): CertificateOrchestrator {
            return (context.applicationContext as AgentApplication).certificateOrchestrator
        }
    }

    private val applicationScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)

    override fun onCreate() {
        super.onCreate()
        Log.i(TAG, "Fleet agent process started")

        // Initialize dependencies
        ApiClient.initialize(this)
        certificateOrchestrator = CertificateOrchestrator()

        refreshEnrollmentCredentials()
        schedulePeriodicCertificateEnrollment()
    }

    private fun refreshEnrollmentCredentials() {
        applicationScope.launch {
            try {
                val restrictionsManager = getSystemService(Context.RESTRICTIONS_SERVICE)
                    as? RestrictionsManager
                val appRestrictions = restrictionsManager?.applicationRestrictions ?: return@launch

                val enrollSecret = appRestrictions.getString("enroll_secret")
                val hostUUID = appRestrictions.getString("host_uuid")
                val serverURL = appRestrictions.getString("server_url")

                if (enrollSecret != null && hostUUID != null && serverURL != null) {
                    Log.d(TAG, "Refreshing enrollment credentials from MDM config")
                    ApiClient.setEnrollmentCredentials(
                        enrollSecret = enrollSecret,
                        hardwareUUID = hostUUID,
                        serverUrl = serverURL,
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

    private fun schedulePeriodicCertificateEnrollment() {
        val workRequest = PeriodicWorkRequestBuilder<CertificateEnrollmentWorker>(
            15, // 15 minutes is the minimum
            TimeUnit.MINUTES,
        ).setConstraints(
            Constraints.Builder()
                .setRequiredNetworkType(NetworkType.CONNECTED)
                .build(),
        ).build()

        WorkManager.getInstance(this)
            .enqueueUniquePeriodicWork(
                CertificateEnrollmentWorker.WORK_NAME,
                ExistingPeriodicWorkPolicy.KEEP,
                workRequest,
            )

        Log.i(TAG, "Scheduled periodic certificate enrollment every 15 minutes")
    }
}
