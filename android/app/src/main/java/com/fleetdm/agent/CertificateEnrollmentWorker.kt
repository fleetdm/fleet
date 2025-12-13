package com.fleetdm.agent

import android.content.Context
import android.util.Log
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters

/**
 * WorkManager worker that handles certificate enrollment operations in the background.
 *
 * This worker:
 * - Gets all certificate IDs from managed configuration
 * - Calls CertificateOrchestrator to enroll all certificates in parallel
 * - Returns appropriate Result based on enrollment outcomes
 * - Supports automatic retry for transient failures
 */
class CertificateEnrollmentWorker(context: Context, workerParams: WorkerParameters) : CoroutineWorker(context, workerParams) {

    override suspend fun doWork(): Result {
        val attemptCount = runAttemptCount
        Log.d(TAG, "Starting certificate enrollment worker (attempt $attemptCount)")

        // Limit retries to avoid infinite loops
        if (attemptCount >= MAX_RETRY_ATTEMPTS) {
            Log.e(TAG, "Maximum retry attempts ($MAX_RETRY_ATTEMPTS) reached, giving up")
            return Result.failure()
        }

        val certificateIds = CertificateOrchestrator.getCertificateIDs(applicationContext)

        // STEP 1: Cleanup removed certificates BEFORE enrolling new ones
        // This runs even if certificateIds is empty to clean up any orphaned certificates
        val currentIds = certificateIds ?: emptyList()
        val cleanupResults = CertificateOrchestrator.cleanupRemovedCertificates(
            context = applicationContext,
            currentCertificateIds = currentIds,
        )

        // Log cleanup results
        cleanupResults.forEach { (certId, result) ->
            when (result) {
                is CleanupResult.Success ->
                    Log.i(TAG, "Cleaned up certificate $certId (alias: ${result.alias})")
                is CleanupResult.AlreadyRemoved ->
                    Log.i(TAG, "Certificate $certId already removed (alias: ${result.alias})")
                is CleanupResult.Failure ->
                    Log.e(TAG, "Failed to cleanup certificate $certId: ${result.reason}", result.exception)
            }
        }

        // STEP 2: If no certificates to enroll, we're done
        if (certificateIds.isNullOrEmpty()) {
            Log.d(TAG, "No certificates to enroll")
            return Result.success()
        }

        // STEP 3: Enroll new/updated certificates
        Log.i(TAG, "Enrolling ${certificateIds.size} certificate(s)")

        val results = CertificateOrchestrator.enrollCertificates(
            context = applicationContext,
            certificateIds = certificateIds,
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
                Log.w(TAG, "Some certificates had transient failures, will retry (attempt $attemptCount of $MAX_RETRY_ATTEMPTS)")
                Result.retry()
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
        private const val TAG = "CertEnrollmentWorker"
        private const val MAX_RETRY_ATTEMPTS = 5

        private fun shouldRetry(reason: String): Boolean {
            // Retry on network/API failures, not on invalid config
            return reason.contains("network", ignoreCase = true) ||
                reason.contains("Failed to fetch", ignoreCase = true) ||
                reason.contains("timeout", ignoreCase = true)
        }
    }
}
