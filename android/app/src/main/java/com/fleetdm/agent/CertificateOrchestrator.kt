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
                    status = "verified",
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
                        status = "failed",
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
