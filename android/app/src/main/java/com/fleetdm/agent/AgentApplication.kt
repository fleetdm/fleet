package com.fleetdm.agent

import android.app.Application
import android.content.Context
import android.content.RestrictionsManager
import android.os.Build
import android.util.Log
import androidx.work.BackoffPolicy
import androidx.work.Constraints
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.NetworkType
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import androidx.work.WorkRequest
import com.fleetdm.agent.device.DeviceIdManager
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import com.fleetdm.agent.device.Storage
import com.fleetdm.agent.osquery.core.TableRegistry

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

        fun getCertificateOrchestrator(context: Context): CertificateOrchestrator =
            (context.applicationContext as AgentApplication).certificateOrchestrator
    }

    private val applicationScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)

    override fun onCreate() {
        super.onCreate()
        Log.i(TAG, "Fleet agent process started")

        FleetLog.initialize(this)
        Storage.init(this)

        // Log device id (safe; not a secret)
        val deviceId = DeviceIdManager.getOrCreateDeviceId()
        Log.i(TAG, "DeviceId=$deviceId")

        val defaultHandler = Thread.getDefaultUncaughtExceptionHandler()
        Thread.setDefaultUncaughtExceptionHandler { thread, throwable ->
            FleetLog.e("fleet-crash", "Uncaught exception on thread ${thread.name}", throwable)
            if (defaultHandler != null) {
                defaultHandler.uncaughtException(thread, throwable)
            } else {
                android.os.Process.killProcess(android.os.Process.myPid())
            }
        }
        // Initialize dependencies
        ApiClient.initialize(this)

        // Register osquery table plugins
        com.fleetdm.agent.osquery.OsqueryTables.registerAll(this)

        // Register core osquery tables (including android_logcat)
        TableRegistry.ensureRegistered()

        if (BuildConfig.DEBUG) {
            DistributedCheckinWorker.scheduleNextDebug(this)
        }

        certificateOrchestrator = CertificateOrchestrator()

        refreshEnrollmentCredentials()
        schedulePeriodicCertificateEnrollment()
    }

    /**
     * Production path: MDM managed configuration (RestrictionsManager).
     * Debug-only fallback: BuildConfig.DEBUG_* values, ONLY if MDM values are missing.
     */
    private fun refreshEnrollmentCredentials() {
        applicationScope.launch {
            try {
                val restrictionsManager =
                    getSystemService(Context.RESTRICTIONS_SERVICE) as? RestrictionsManager
                val appRestrictions = restrictionsManager?.applicationRestrictions

                val mdmEnrollSecret = appRestrictions?.getString("enroll_secret")
                val mdmHostUUID = appRestrictions?.getString("host_uuid")
                val mdmServerURL = appRestrictions?.getString("server_url")

                val (enrollSecret, hostUUID, serverURL) = if (
                    !mdmEnrollSecret.isNullOrBlank() &&
                    !mdmHostUUID.isNullOrBlank() &&
                    !mdmServerURL.isNullOrBlank()
                ) {
                    Log.d(TAG, "Using MDM enrollment credentials (managed config)")
                    Triple(mdmEnrollSecret, mdmHostUUID, mdmServerURL)
                } else if (BuildConfig.DEBUG) {
                    val debugUrl = getOptionalBuildConfigString("DEBUG_FLEET_SERVER_URL")
                    val debugSecret = getOptionalBuildConfigString("DEBUG_FLEET_ENROLL_SECRET")

                    if (!debugUrl.isNullOrBlank() && !debugSecret.isNullOrBlank()) {
                        // Debug fallback host UUID: stable per app install (acceptable for dev)
                        val debugHostUUID = DeviceIdManager.getOrCreateDeviceId()

                        Log.w(TAG, "MDM config missing; using DEBUG enrollment credentials")
                        Triple(debugSecret, debugHostUUID, debugUrl)
                    } else {                        Log.d(TAG, "MDM config missing and DEBUG values not set")
                        return@launch
                    }
                } else {
                    Log.d(TAG, "MDM enrollment credentials not available")
                    return@launch
                }

                ApiClient.setEnrollmentCredentials(
                    enrollSecret = enrollSecret,
                    hardwareUUID = hostUUID,
                    serverUrl = serverURL,
                    computerName = "${Build.BRAND} ${Build.MODEL}",
                )

                // Only enroll if not already enrolled
                if (ApiClient.getApiKey() == null) {
                    val configResult = ApiClient.getOrbitConfig()
                    configResult.onSuccess {
                        Log.d(TAG, "Successfully enrolled host with Fleet server")
                    }.onFailure { error ->
                        Log.w(TAG, "Host enrollment failed: ${error.message}")
                    }
                }
            } catch (e: Exception) {
                FleetLog.e(TAG, "Error refreshing enrollment credentials", e)
            }
        }
    }


    private fun getOptionalBuildConfigString(fieldName: String): String? {
        return try {
            val clazz = Class.forName("${packageName}.BuildConfig")
            val field = clazz.getField(fieldName)
            (field.get(null) as? String)?.takeIf { it.isNotBlank() }
        } catch (_: Throwable) {
            null
        }
    }

    private fun schedulePeriodicCertificateEnrollment() {
        val workRequest = PeriodicWorkRequestBuilder<CertificateEnrollmentWorker>(
            15, // 15 minutes is the minimum
            TimeUnit.MINUTES,
        ).setBackoffCriteria(
            BackoffPolicy.EXPONENTIAL,
            WorkRequest.MIN_BACKOFF_MILLIS,
            TimeUnit.MILLISECONDS,
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
