package com.fleetdm.agent

import android.content.Context
import android.os.Build
import android.util.Log
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock

/**
 * Manages device enrollment with the Fleet server.
 * Uses a Mutex to ensure only one enrollment attempt runs at a time.
 */
object EnrollmentManager {
    private const val TAG = "fleet-enrollment"
    private val enrollMutex = Mutex()

    /**
     * Attempts to enroll the device if not already enrolled.
     * This method is thread-safe and will only allow one enrollment at a time.
     *
     * @param context Application context for accessing RestrictionsManager and ApiClient
     * @return EnrollmentResult indicating the outcome
     */
    suspend fun tryEnroll(context: Context): EnrollmentResult {
        return enrollMutex.withLock {
            Log.d(TAG, "Acquired enrollment lock, checking enrollment status")

            // Check if already enrolled
            ApiClient.initialize(context)
            val existingApiKey = ApiClient.getApiKey()
            if (existingApiKey != null) {
                Log.d(TAG, "Already enrolled, skipping enrollment")
                return@withLock EnrollmentResult.AlreadyEnrolled
            }

            // Read managed configuration
            val restrictionsManager = context.getSystemService(Context.RESTRICTIONS_SERVICE)
                as? android.content.RestrictionsManager
            val appRestrictions = restrictionsManager?.applicationRestrictions
            if (appRestrictions == null) {
                Log.w(TAG, "Unable to read managed configuration")
                return@withLock EnrollmentResult.MissingConfig("restrictions")
            }

            val enrollSecret = appRestrictions.getString("enrollSecret")
            val fleetBaseUrl = appRestrictions.getString("fleetBaseUrl")
            val enrollmentSpecificID = appRestrictions.getString("enrollmentSpecificID")

            // Validate required fields
            if (enrollSecret.isNullOrEmpty()) {
                Log.w(TAG, "Missing enrollSecret in managed configuration")
                return@withLock EnrollmentResult.MissingConfig("enrollSecret")
            }
            if (fleetBaseUrl.isNullOrEmpty()) {
                Log.w(TAG, "Missing fleetBaseUrl in managed configuration")
                return@withLock EnrollmentResult.MissingConfig("fleetBaseUrl")
            }
            if (enrollmentSpecificID.isNullOrEmpty()) {
                Log.w(TAG, "Missing enrollmentSpecificID in managed configuration")
                return@withLock EnrollmentResult.MissingConfig("enrollmentSpecificID")
            }

            Log.i(TAG, "Attempting enrollment to $fleetBaseUrl")

            // Attempt enrollment
            val result = ApiClient.enroll(
                baseUrl = fleetBaseUrl,
                enrollSecret = enrollSecret,
                hardwareUUID = enrollmentSpecificID,
                computerName = Build.MODEL,
            )

            result.fold(
                onSuccess = {
                    Log.i(TAG, "Enrollment successful")
                    EnrollmentResult.Success
                },
                onFailure = { exception ->
                    Log.e(TAG, "Enrollment failed: ${exception.message}")
                    EnrollmentResult.Error(exception.message ?: "Unknown error")
                },
            )
        }
    }
}

sealed class EnrollmentResult {
    data object Success : EnrollmentResult()
    data object AlreadyEnrolled : EnrollmentResult()
    data class MissingConfig(val field: String) : EnrollmentResult()
    data class Error(val message: String) : EnrollmentResult()
}
