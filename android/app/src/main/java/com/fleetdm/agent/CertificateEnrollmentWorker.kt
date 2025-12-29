package com.fleetdm.agent

import android.content.Context
import android.util.Log
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters

/**
 * WorkManager worker that handles certificate enrollment operations in the background.
 *
 * This worker:
 * - Gets host certificates from managed configuration
 * - Calls CertificateOrchestrator to enroll all certificates in parallel
 * - Returns appropriate Result based on enrollment outcomes
 * - Supports automatic retry for transient failures
 */
class CertificateEnrollmentWorker(context: Context, workerParams: WorkerParameters) : CoroutineWorker(context, workerParams) {

    override suspend fun doWork(): Result {
        Log.d(TAG, "Starting certificate enrollment worker (attempt ${runAttemptCount + 1})")

        // Get orchestrator from Application
        val orchestrator = AgentApplication.getCertificateOrchestrator(applicationContext)

        // STEP 0: Retry any unreported statuses from previous runs
        val unreportedResults = orchestrator.retryUnreportedStatuses(applicationContext)
        unreportedResults.forEach { (certId, success) ->
            if (success) {
                Log.i(TAG, "Successfully reported unreported status for certificate $certId")
            } else {
                Log.w(TAG, "Failed to report unreported status for certificate $certId, will retry next run")
            }
        }

        val hostCertificates = orchestrator.getHostCertificates(applicationContext) ?: emptyList()

        // STEP 1: Cleanup certificates marked for removal and orphaned certificates
        val cleanupResults = orchestrator.cleanupRemovedCertificates(
            context = applicationContext,
            hostCertificates = hostCertificates,
        )

        // Log cleanup results
        cleanupResults.forEach { (certId, result) ->
            when (result) {
                is CleanupResult.Success ->
                    Log.i(TAG, "Cleaned up certificate $certId (alias: ${result.alias})")
                is CleanupResult.AlreadyRemoved ->
                    Log.d(TAG, "Certificate $certId already removed (alias: ${result.alias})")
                is CleanupResult.Failure ->
                    Log.e(TAG, "Failed to cleanup certificate $certId: ${result.reason}", result.exception)
            }
        }

        // STEP 2: Filter to only certificates marked for install
        val certificatesToInstall = hostCertificates.filter { it.shouldInstall() }

        // If no certificates to enroll, we're done
        if (certificatesToInstall.isEmpty()) {
            Log.d(TAG, "No certificates to enroll")
            return Result.success()
        }

        // STEP 3: Enroll new/updated certificates
        Log.i(TAG, "Enrolling ${certificatesToInstall.size} certificate(s): ${certificatesToInstall.map { it.id }}")

        val results = orchestrator.enrollCertificates(
            context = applicationContext,
            hostCertificates = certificatesToInstall,
        )

        // Analyze results to determine worker outcome
        var hasSuccess = false
        var hasTransientFailure = false
        var hasPermanentFailure = false

        results.forEach { (certificateId, result) ->
            when (result) {
                is CertificateEnrollmentHandler.EnrollmentResult.Success -> {
                    Log.i(TAG, "Certificate $certificateId enrolled successfully: ${result.alias}")
                    hasSuccess = true
                }
                is CertificateEnrollmentHandler.EnrollmentResult.Failure -> {
                    Log.e(TAG, "Certificate $certificateId enrollment failed: ${result.reason}", result.exception)
                    if (shouldRetry(result.reason)) {
                        hasTransientFailure = true
                    } else {
                        hasPermanentFailure = true
                    }
                }
            }
        }

        // Return result based on outcomes
        return when {
            hasTransientFailure -> {
                if (runAttemptCount >= MAX_RETRY_ATTEMPTS - 1) {
                    // Exhausted retries, return success to reset and let periodic schedule take over
                    Log.w(TAG, "Some certificates had transient failures, exhausted $MAX_RETRY_ATTEMPTS retries, will retry in 15 minutes")
                    Result.success()
                } else {
                    Log.w(
                        TAG,
                        "Some certificates had transient failures, will retry (attempt ${runAttemptCount + 1} of $MAX_RETRY_ATTEMPTS)",
                    )
                    Result.retry()
                }
            }
            hasPermanentFailure -> {
                if (hasSuccess) {
                    Log.w(TAG, "Some certificates succeeded, some failed permanently")
                }
                Result.failure()
            }
            else -> {
                Log.i(TAG, "All ${results.size} certificate(s) enrolled successfully")
                Result.success()
            }
        }
    }

    companion object {
        const val WORK_NAME = "certificate_enrollment"
        private const val TAG = "fleet-CertificateEnrollmentWorker"
        private const val MAX_RETRY_ATTEMPTS = 5

        private fun shouldRetry(reason: String): Boolean {
            // Retry on network/API failures, not on invalid config
            return reason.contains("network", ignoreCase = true) ||
                reason.contains("Failed to fetch", ignoreCase = true) ||
                reason.contains("timeout", ignoreCase = true)
        }
    }
}
