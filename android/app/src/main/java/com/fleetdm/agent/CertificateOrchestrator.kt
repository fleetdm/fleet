package com.fleetdm.agent

import android.app.admin.DevicePolicyManager
import android.content.Context
import android.os.Bundle
import android.util.Log
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import com.fleetdm.agent.scep.ScepClient
import com.fleetdm.agent.scep.ScepClientImpl
import java.security.PrivateKey
import java.security.cert.Certificate
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

const val MAX_CERT_INSTALL_RETRIES = 3

/**
 * Orchestrates certificate enrollment operations by coordinating API calls,
 * SCEP enrollment, and certificate installation.
 *
 * This object provides a neutral orchestration layer that can be called from
 * multiple contexts (Service, Worker, direct calls) while maintaining separation
 * of concerns between Android framework code and business logic.
 *
 * ## Usage Examples
 *
 * Single certificate:
 * ```
 * val result = CertificateOrchestrator.enrollCertificate(
 *     context = applicationContext,
 *     certificateId = 123
 * )
 * ```
 *
 * Batch processing:
 * ```
 * val certificateIds = CertificateOrchestrator.getCertificateIDs(context)
 * val results = CertificateOrchestrator.enrollCertificates(
 *     context = applicationContext,
 *     certificateIds = certificateIds ?: emptyList()
 * )
 * ```
 */
object CertificateOrchestrator {
    private const val TAG = "CertificateOrchestrator"

    // DataStore key for storing installed certificates map as JSON
    private val INSTALLED_CERTIFICATES_KEY = stringPreferencesKey("installed_certificates")

    // JSON serializer instance
    private val json = Json {
        ignoreUnknownKeys = true
        encodeDefaults = true
        // Treat a missing field like a null field for optional types
        explicitNulls = false
    }

    // Mutex to protect concurrent access to certificate storage
    private val certificateStorageMutex = Mutex()

    fun installedCertsFlow(context: Context): Flow<CertStatusMap> = context.prefDataStore.data.map { preferences ->
        try {
            val jsonStr = preferences[INSTALLED_CERTIFICATES_KEY]
            Log.d("installedCertsFlow", "json: $jsonStr")
            json.decodeFromString(jsonStr!!)
        } catch (e: Exception) {
            Log.d("installedCertsFlow", e.toString())
            emptyMap()
        }
    }

    /**
     * Reads certificate IDs from Android Managed Configuration.
     *
     * @param context Android context
     * @return List of certificate IDs to enroll, or null if none configured
     */
    fun getCertificateIDs(context: Context): List<Int>? {
        val restrictionsManager = context.getSystemService(Context.RESTRICTIONS_SERVICE) as android.content.RestrictionsManager
        val appRestrictions = restrictionsManager.applicationRestrictions

        val certRequestList = appRestrictions.getParcelableArray("certificate_templates", Bundle::class.java)?.toList()
        return certRequestList?.map { bundle -> bundle.getInt("id") }
    }

    /**
     * Reads the installed certificates map from DataStore.
     *
     * @param context Android context
     * @return Map of certificate ID to alias, or empty map if none stored
     */
    internal suspend fun getCertificateInstallInfos(context: Context): CertStatusMap {
        certificateStorageMutex.withLock {
            return try {
                val prefs = context.prefDataStore.data.first()
                val jsonString = prefs[INSTALLED_CERTIFICATES_KEY]

                if (jsonString == null) {
                    Log.d(TAG, "No installed certificates found in DataStore")
                    return emptyMap()
                }

                val map = json.decodeFromString<CertStatusMap>(jsonString)
                Log.d(TAG, "Loaded ${map.size} installed certificate(s) from DataStore")
                map
            } catch (e: Exception) {
                Log.e(TAG, "Failed to read installed certificates from DataStore: ${e.message}", e)
                emptyMap()
            }
        }
    }

    internal suspend fun getCertificateInstallInfo(context: Context, certificateId: Int): CertificateInstallInfo? {
        val certs = getCertificateInstallInfos(context = context)
        return certs[certificateId]
    }

    internal suspend fun markCertificateInstalled(context: Context, certificateId: Int, alias: String) {
        val existingInfo = getCertificateInstallInfo(context = context, certificateId = certificateId)
            ?: CertificateInstallInfo(alias = alias, status = CertificateInstallStatus.INSTALLED, retries = 0)

        val newInfo = existingInfo.copy(alias = alias, status = CertificateInstallStatus.INSTALLED, retries = 0)
        storeCertificateInstallationInfo(context = context, certificateId = certificateId, certInstallInfo = newInfo)
    }

    internal suspend fun markCertificateFailure(context: Context, certificateId: Int, alias: String): CertificateInstallInfo {
        val existingInfo = getCertificateInstallInfo(context = context, certificateId = certificateId)
            ?: CertificateInstallInfo(alias = alias, status = CertificateInstallStatus.RETRY, retries = 0)

        if (existingInfo.status != CertificateInstallStatus.RETRY) {
            return existingInfo
        }

        var newInfo = existingInfo.copy(retries = existingInfo.retries + 1)

        if (newInfo.retries >= MAX_CERT_INSTALL_RETRIES) {
            newInfo = newInfo.copy(status = CertificateInstallStatus.FAILED)
        }

        storeCertificateInstallationInfo(context = context, certificateId = certificateId, newInfo)

        return newInfo
    }

    /**
     * Stores a certificate ID→alias mapping in DataStore after successful installation.
     * This performs a read-modify-write operation to update the map.
     *
     * @param context Android context
     * @param certificateId Certificate template ID
     * @param alias Certificate alias used during installation
     */
    internal suspend fun storeCertificateInstallationInfo(context: Context, certificateId: Int, certInstallInfo: CertificateInstallInfo) {
        certificateStorageMutex.withLock {
            try {
                context.prefDataStore.edit { preferences ->
                    // Read existing map
                    val existingJsonString = preferences[INSTALLED_CERTIFICATES_KEY]
                    val existingMap = if (existingJsonString != null) {
                        try {
                            json.decodeFromString<CertStatusMap>(existingJsonString)
                        } catch (e: Exception) {
                            Log.w(TAG, "Failed to parse existing certificates JSON, starting fresh: ${e.message}")
                            emptyMap()
                        }
                    } else {
                        emptyMap()
                    }

                    // Add new mapping
                    val updatedMap = existingMap.toMutableMap().apply {
                        put(certificateId, certInstallInfo)
                    }

                    // Serialize and store
                    val updatedJsonString = json.encodeToString(updatedMap)
                    preferences[INSTALLED_CERTIFICATES_KEY] = updatedJsonString

                    Log.d(TAG, "Stored certificate mapping: $certificateId → ${certInstallInfo.alias} (total: ${updatedMap.size})")
                }
            } catch (e: Exception) {
                Log.e(TAG, "Failed to store certificate installation: ${e.message}", e)
                // Non-fatal error - enrollment was successful, just tracking failed
            }
        }
    }

    /**
     * Removes a certificate installation record from DataStore.
     *
     * @param context Android context
     * @param certificateId Certificate template ID to remove
     */
    internal suspend fun removeCertificateInstallInfo(context: Context, certificateId: Int) {
        certificateStorageMutex.withLock {
            try {
                context.prefDataStore.edit { preferences ->
                    val existingJsonString = preferences[INSTALLED_CERTIFICATES_KEY]
                    val existingMap = if (existingJsonString != null) {
                        try {
                            json.decodeFromString<CertStatusMap>(existingJsonString)
                        } catch (e: Exception) {
                            Log.w(TAG, "Failed to parse existing certificates JSON: ${e.message}")
                            emptyMap()
                        }
                    } else {
                        emptyMap()
                    }

                    // Remove the entry
                    val updatedMap = existingMap.toMutableMap().apply {
                        remove(certificateId)
                    }

                    // Serialize and store
                    val updatedJsonString = json.encodeToString(updatedMap)
                    preferences[INSTALLED_CERTIFICATES_KEY] = updatedJsonString

                    Log.d(TAG, "Removed certificate mapping for ID $certificateId (remaining: ${updatedMap.size})")
                }
            } catch (e: Exception) {
                Log.e(TAG, "Failed to remove certificate installation info: ${e.message}", e)
                // Non-fatal error - cleanup was attempted
            }
        }
    }

    /**
     * Retrieves the certificate alias for a given certificate ID from DataStore.
     *
     * @param context Android context
     * @param certificateId Certificate template ID
     * @return Certificate alias if previously installed, null otherwise
     */
    internal suspend fun getCertificateAlias(context: Context, certificateId: Int): String? {
        val installedCerts = getCertificateInstallInfos(context)
        val status = installedCerts[certificateId]
        Log.d(TAG, "Certificate $certificateId alias lookup: ${status?.alias ?: "not found"}")
        return status?.alias
    }

    /**
     * Checks if a certificate is installed in the Android keystore.
     *
     * @param context Android context
     * @param alias Certificate alias
     * @return True if certificate exists in keystore
     */
    private fun isCertificateInstalled(context: Context, alias: String): Boolean = try {
        val dpm = context.getSystemService(Context.DEVICE_POLICY_SERVICE) as DevicePolicyManager
        val hasKeyPair = dpm.hasKeyPair(alias)
        Log.d(TAG, "Certificate '$alias' installation check: $hasKeyPair")
        hasKeyPair
    } catch (e: Exception) {
        Log.e(TAG, "Error checking if certificate '$alias' is installed: ${e.message}", e)
        false
    }

    /**
     * Removes a certificate keypair from the Android keystore.
     *
     * @param context Android context
     * @param alias Certificate alias to remove
     * @return True if removal was successful or certificate doesn't exist
     */
    private fun removeKeyPair(context: Context, alias: String): Boolean {
        return try {
            val dpm = context.getSystemService(Context.DEVICE_POLICY_SERVICE) as DevicePolicyManager

            // First check if keypair exists
            if (!dpm.hasKeyPair(alias)) {
                Log.i(TAG, "Certificate '$alias' doesn't exist in keystore, considering removal successful")
                return true
            }

            // Attempt to remove the keypair
            // admin component is null because we're using delegated certificate management
            val removed = dpm.removeKeyPair(null, alias)

            if (removed) {
                Log.i(TAG, "Successfully removed certificate keypair with alias: $alias")
            } else {
                Log.e(TAG, "Failed to remove certificate keypair '$alias'. Check MDM policy and delegation status.")
            }

            removed
        } catch (e: SecurityException) {
            Log.e(TAG, "Security exception removing certificate '$alias': ${e.message}", e)
            false
        } catch (e: Exception) {
            Log.e(TAG, "Error removing certificate '$alias': ${e.message}", e)
            false
        }
    }

    /**
     * Checks if a certificate ID has been successfully installed and still exists in keystore.
     * This is a fast check that doesn't require fetching the template from the API.
     *
     * @param context Android context
     * @param certificateId Certificate template ID
     * @return True if certificate is tracked in DataStore AND exists in keystore
     */
    internal suspend fun isCertificateIdInstalled(context: Context, certificateId: Int): Boolean {
        // Check DataStore for this certificate ID
        val storedAlias = getCertificateAlias(context, certificateId)
        if (storedAlias == null) {
            Log.d(TAG, "Certificate ID $certificateId not found in DataStore")
            return false
        }

        // Verify certificate still exists in keystore
        val existsInKeystore = isCertificateInstalled(context, storedAlias)
        if (!existsInKeystore) {
            Log.w(TAG, "Certificate ID $certificateId tracked in DataStore but missing from keystore - will re-enroll")
        }

        return existsInKeystore
    }

    /**
     * Cleans up certificates that were removed from managed configuration.
     *
     * This function:
     * 1. Identifies certificates in DataStore that are no longer in current config
     * 2. Removes the corresponding keypairs from the device using DevicePolicyManager
     * 3. Cleans up the DataStore tracking
     * 4. Reports removal status to the server
     *
     * @param context Android context for certificate operations
     * @param currentCertificateIds List of certificate IDs from current managed configuration
     * @return Map of certificate ID to cleanup result
     */
    suspend fun cleanupRemovedCertificates(context: Context, currentCertificateIds: List<Int>): Map<Int, CleanupResult> {
        Log.d(TAG, "Starting certificate cleanup. Current IDs: $currentCertificateIds")

        // Get all installed certificates from DataStore
        val installedCerts = getCertificateInstallInfos(context)
        Log.d(TAG, "Found ${installedCerts.size} certificate(s) in DataStore")

        // Identify certificates to remove (in DataStore but not in current config)
        val certificatesToRemove = installedCerts.keys.filter { it !in currentCertificateIds }

        if (certificatesToRemove.isEmpty()) {
            Log.d(TAG, "No certificates to remove")
            return emptyMap()
        }

        Log.i(TAG, "Removing ${certificatesToRemove.size} certificate(s): $certificatesToRemove")

        val results = mutableMapOf<Int, CleanupResult>()

        for (certificateId in certificatesToRemove) {
            val certInfo = installedCerts[certificateId]
            if (certInfo == null) {
                Log.w(TAG, "Certificate ID $certificateId not found in DataStore, skipping")
                continue
            }

            val alias = certInfo.alias
            Log.d(TAG, "Removing certificate ID $certificateId with alias '$alias' (status: ${certInfo.status})")

            // Attempt to remove the keypair
            val removed = removeKeyPair(context, alias)

            if (removed) {
                // Report successful removal to server
                ApiClient.updateCertificateStatus(
                    certificateId = certificateId,
                    status = UpdateCertificateStatusStatus.FAILED,
                    operationType = UpdateCertificateStatusOperation.REMOVE,
                ).onFailure { error ->
                    Log.e(TAG, "Failed to report certificate removal status for ID $certificateId: ${error.message}", error)
                }

                // Clean up DataStore
                removeCertificateInstallInfo(context, certificateId)

                results[certificateId] = CleanupResult.Success(alias)
                Log.i(TAG, "Successfully removed certificate ID $certificateId (alias: '$alias')")
            } else {
                // Report failure to server
                val errorDetail = "Failed to remove certificate keypair from device"
                ApiClient.updateCertificateStatus(
                    certificateId = certificateId,
                    status = UpdateCertificateStatusStatus.FAILED,
                    operationType = UpdateCertificateStatusOperation.REMOVE,
                    detail = errorDetail,
                ).onFailure { error ->
                    Log.e(TAG, "Failed to report certificate removal failure for ID $certificateId: ${error.message}", error)
                }

                results[certificateId] = CleanupResult.Failure(
                    reason = errorDetail,
                    exception = null,
                    shouldRetry = false, // Permission or configuration issue, don't retry
                )
                Log.e(TAG, "Failed to remove certificate ID $certificateId (alias: '$alias')")
            }
        }

        return results
    }

    /**
     * Enrolls a single certificate by fetching its template from the API,
     * performing SCEP enrollment, and installing it on the device.
     *
     * @param context Android context for certificate installation
     * @param certificateId ID of the certificate template to enroll
     * @param scepClient SCEP client implementation (defaults to ScepClientImpl)
     * @param certificateInstaller Certificate installer implementation (defaults to AndroidCertificateInstaller)
     * @return EnrollmentResult indicating success or failure with details
     */
    suspend fun enrollCertificate(
        context: Context,
        certificateId: Int,
        scepClient: ScepClient = ScepClientImpl(),
        certificateInstaller: CertificateEnrollmentHandler.CertificateInstaller? = null,
    ): CertificateEnrollmentHandler.EnrollmentResult {
        Log.d(TAG, "Starting certificate enrollment for certificate ID: $certificateId")

        // Check if certificate is already installed (BEFORE API call)
        if (isCertificateIdInstalled(context, certificateId)) {
            val alias = getCertificateAlias(context, certificateId)!!
            Log.i(TAG, "Certificate ID $certificateId (alias: '$alias') is already installed, skipping enrollment")
            return CertificateEnrollmentHandler.EnrollmentResult.Success(alias)
        }

        // Skip enrollment if already marked as permanently failed (max retries exceeded).
        // Returns Success to prevent retry loops - the failure has already been reported
        // to the Fleet server via updateCertificateStatus().
        val storedInfo = getCertificateInstallInfo(context = context, certificateId = certificateId)
        if (storedInfo?.status == CertificateInstallStatus.FAILED) {
            return CertificateEnrollmentHandler.EnrollmentResult.Success(storedInfo.alias)
        }

        // Fetch certificate template from API (only if not already installed)
        val templateResult = ApiClient.getCertificateTemplate(certificateId)
        val template = templateResult.getOrElse { error ->
            Log.e(TAG, "Failed to fetch certificate template for ID $certificateId: ${error.message}", error)
            return CertificateEnrollmentHandler.EnrollmentResult.Failure(
                reason = "Failed to fetch certificate template: ${error.message}",
                exception = error as? Exception,
            )
        }

        Log.d(TAG, "Successfully fetched certificate template: ${template.name}")

        if (template.status != "delivered") {
            // The certificate template hasn't failed on the device, but isn't ready to be processed yet.
            // Retry next time we fetch but don't mark as failed locally
            Log.i(TAG, "Certificate template ${template.name} does not have status \"delivered\": status \"${template.status}\"")
            return CertificateEnrollmentHandler.EnrollmentResult.Success(template.name)
        }

        // Step 3: Create certificate installer (use provided or create default)
        val installer = certificateInstaller ?: AndroidCertificateInstaller(context)

        // Step 4: Create enrollment handler
        val handler = CertificateEnrollmentHandler(
            scepClient = scepClient,
            certificateInstaller = installer,
        )

        // Step 5: Perform enrollment
        Log.d(TAG, "Starting SCEP enrollment for certificate: ${template.name}: $template")
        val result = handler.handleEnrollment(template)

        when (result) {
            is CertificateEnrollmentHandler.EnrollmentResult.Success -> {
                Log.i(TAG, "Certificate enrollment successful for ID $certificateId with alias: ${result.alias}")
                ApiClient.updateCertificateStatus(
                    certificateId = certificateId,
                    status = UpdateCertificateStatusStatus.VERIFIED,
                    operationType = UpdateCertificateStatusOperation.INSTALL,
                ).onFailure { error ->
                    Log.e(TAG, "Failed to update certificate status to verified for ID $certificateId: ${error.message}", error)
                }

                // Store certificate installation in DataStore
                markCertificateInstalled(context, certificateId = certificateId, alias = template.name)
            }
            is CertificateEnrollmentHandler.EnrollmentResult.Failure -> {
                val updatedInfo = markCertificateFailure(context = context, certificateId = certificateId, alias = template.name)
                if (!updatedInfo.shouldRetry()) {
                    Log.e(TAG, "Certificate enrollment failed for ID $certificateId: ${result.reason}", result.exception)
                    ApiClient.updateCertificateStatus(
                        certificateId = certificateId,
                        status = UpdateCertificateStatusStatus.FAILED,
                        operationType = UpdateCertificateStatusOperation.INSTALL,
                        detail = result.reason,
                    ).onFailure { error ->
                        Log.e(TAG, "Failed to update certificate status to failed for ID $certificateId: ${error.message}", error)
                    }
                }
            }
        }

        return result
    }

    /**
     * Enrolls multiple certificates in parallel.
     *
     * @param context Android context for certificate installation
     * @param certificateIds List of certificate template IDs to enroll
     * @param scepClient SCEP client implementation (defaults to ScepClientImpl)
     * @return Map of certificate ID to enrollment result
     */
    suspend fun enrollCertificates(
        context: Context,
        certificateIds: List<Int>,
        scepClient: ScepClient = ScepClientImpl(),
    ): Map<Int, CertificateEnrollmentHandler.EnrollmentResult> = coroutineScope {
        Log.d(TAG, "Starting batch certificate enrollment for ${certificateIds.size} certificates")

        certificateIds.associateWith { certificateId ->
            async {
                enrollCertificate(context, certificateId, scepClient)
            }
        }.mapValues { it.value.await() }
    }

    /**
     * Android-specific certificate installer using DevicePolicyManager.
     *
     * This implementation uses the delegated certificate installation API
     * which allows a non-DPC app to install certificates when properly
     * delegated by the Device Policy Controller.
     */
    class AndroidCertificateInstaller(private val context: Context) : CertificateEnrollmentHandler.CertificateInstaller {
        private val TAG = "AndroidCertInstaller"

        override fun installCertificate(alias: String, privateKey: PrivateKey, certificateChain: Array<Certificate>): Boolean {
            val dpm = context.getSystemService(Context.DEVICE_POLICY_SERVICE) as DevicePolicyManager

            // The admin component is null because the caller is a DELEGATED application,
            // not the Device Policy Controller itself. The DPM recognizes the delegation
            // via the calling package's UID and the granted CERT_INSTALL scope.
            val success = dpm.installKeyPair(
                null,
                privateKey,
                certificateChain,
                alias,
                true, // requestAccess: allows user confirmation if needed
            )

            if (success) {
                Log.i(TAG, "Certificate successfully installed with alias: $alias")
            } else {
                Log.e(TAG, "Certificate installation failed. Check MDM policy and delegation status.")
            }

            return success
        }
    }
}

typealias CertStatusMap = Map<Int, CertificateInstallInfo>

@Serializable
enum class CertificateInstallStatus {
    @SerialName("installed")
    INSTALLED,

    @SerialName("failed")
    FAILED,

    @SerialName("retry")
    RETRY,
}

@Serializable
data class CertificateInstallInfo(
    @SerialName("alias")
    val alias: String,
    @SerialName("status")
    val status: CertificateInstallStatus,
    @SerialName("retries")
    val retries: Int = 0,
) {
    fun shouldRetry(): Boolean = status == CertificateInstallStatus.RETRY && retries < (MAX_CERT_INSTALL_RETRIES)
}

/**
 * Result of certificate cleanup operation
 */
sealed class CleanupResult {
    data class Success(val alias: String) : CleanupResult()
    data class Failure(val reason: String, val exception: Exception?, val shouldRetry: Boolean) : CleanupResult()
    data class AlreadyRemoved(val alias: String) : CleanupResult()
}
