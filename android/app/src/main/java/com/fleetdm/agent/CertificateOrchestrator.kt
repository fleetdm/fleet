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
const val MAX_STATUS_REPORT_RETRIES = 10

/**
 * Orchestrates certificate enrollment operations by coordinating API calls,
 * SCEP enrollment, and certificate installation.
 *
 * This class provides a neutral orchestration layer that can be called from
 * multiple contexts (Service, Worker, direct calls) while maintaining separation
 * of concerns between Android framework code and business logic.
 *
 * @param apiClient Client for Fleet server API calls
 * @param scepClient Client for SCEP enrollment operations
 */
class CertificateOrchestrator(
    private val apiClient: CertificateApiClient = ApiClient,
    private val scepClient: ScepClient = ScepClientImpl(),
    private val deviceKeystoreManager: DeviceKeystoreManager? = null,
) {
    companion object {
        private const val TAG = "fleet-CertificateOrchestrator"
    }

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

    fun installedCertsFlow(context: Context): Flow<CertificateStateMap> = context.prefDataStore.data.map { preferences ->
        try {
            val jsonStr = preferences[INSTALLED_CERTIFICATES_KEY]
            json.decodeFromString(jsonStr!!)
        } catch (e: Exception) {
            Log.d("installedCertsFlow", e.toString())
            emptyMap()
        }
    }

    /**
     * Reads certificate templates from Android Managed Configuration.
     *
     * @param context Android context
     * @return List of certificate templates, or null if none configured
     */
    fun getHostCertificates(context: Context): List<HostCertificate>? {
        val restrictionsManager = context.getSystemService(Context.RESTRICTIONS_SERVICE) as android.content.RestrictionsManager
        val appRestrictions = restrictionsManager.applicationRestrictions

        val certRequestList = appRestrictions.getParcelableArray("certificate_templates", Bundle::class.java)?.toList()
        return certRequestList?.map { bundle ->
            HostCertificate(
                id = bundle.getInt("id"),
                status = bundle.getString("status", ""),
                operation = bundle.getString("operation", HostCertificate.OPERATION_INSTALL),
                version = bundle.getInt("version", 1),
            )
        }
    }

    /**
     * Reads the installed certificates map from DataStore.
     *
     * @param context Android context
     * @return Map of certificate ID to alias, or empty map if none stored
     */
    internal suspend fun getCertificateStates(context: Context): CertificateStateMap {
        certificateStorageMutex.withLock {
            return try {
                val prefs = context.prefDataStore.data.first()
                val jsonString = prefs[INSTALLED_CERTIFICATES_KEY]

                if (jsonString == null) {
                    Log.d(TAG, "No installed certificates found in DataStore")
                    return emptyMap()
                }

                json.decodeFromString<CertificateStateMap>(jsonString)
            } catch (e: Exception) {
                Log.e(TAG, "Failed to read installed certificates from DataStore: ${e.message}", e)
                emptyMap()
            }
        }
    }

    internal suspend fun getCertificateState(context: Context, certificateId: Int): CertificateState? {
        val certs = getCertificateStates(context = context)
        return certs[certificateId]
    }

    internal suspend fun markCertificateInstalled(context: Context, certificateId: Int, alias: String, version: Int) {
        val existingInfo = getCertificateState(context = context, certificateId = certificateId)
            ?: CertificateState(alias = alias, status = CertificateStatus.INSTALLED, retries = 0, version = version)

        val newInfo = existingInfo.copy(alias = alias, status = CertificateStatus.INSTALLED, retries = 0, version = version)
        storeCertificateState(context = context, certificateId = certificateId, certInstallInfo = newInfo)
    }

    internal suspend fun markCertificateFailure(context: Context, certificateId: Int, alias: String): CertificateState {
        val existingInfo = getCertificateState(context = context, certificateId = certificateId)
            ?: CertificateState(alias = alias, status = CertificateStatus.RETRY, retries = 0)

        if (existingInfo.status != CertificateStatus.RETRY) {
            return existingInfo
        }

        var newInfo = existingInfo.copy(retries = existingInfo.retries + 1)

        if (newInfo.retries >= MAX_CERT_INSTALL_RETRIES) {
            newInfo = newInfo.copy(status = CertificateStatus.FAILED)
        }

        storeCertificateState(context = context, certificateId = certificateId, newInfo)

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
    internal suspend fun storeCertificateState(context: Context, certificateId: Int, certInstallInfo: CertificateState) {
        certificateStorageMutex.withLock {
            try {
                context.prefDataStore.edit { preferences ->
                    // Read existing map
                    val existingJsonString = preferences[INSTALLED_CERTIFICATES_KEY]
                    val existingMap = if (existingJsonString != null) {
                        try {
                            json.decodeFromString<CertificateStateMap>(existingJsonString)
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
    internal suspend fun removeCertificateState(context: Context, certificateId: Int) {
        certificateStorageMutex.withLock {
            try {
                context.prefDataStore.edit { preferences ->
                    val existingJsonString = preferences[INSTALLED_CERTIFICATES_KEY]
                    val existingMap = if (existingJsonString != null) {
                        try {
                            json.decodeFromString<CertificateStateMap>(existingJsonString)
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
        val installedCerts = getCertificateStates(context)
        val status = installedCerts[certificateId]
        Log.d(TAG, "Certificate $certificateId alias lookup: ${status?.alias ?: "not found"}")
        return status?.alias
    }

    /**
     * Gets the device keystore manager, using injected instance or creating a default one.
     */
    private fun getDeviceKeystoreManager(context: Context): DeviceKeystoreManager =
        deviceKeystoreManager ?: AndroidDeviceKeystoreManager(context)

    /**
     * Checks if a certificate is installed in the Android keystore.
     *
     * @param context Android context
     * @param alias Certificate alias
     * @return True if certificate exists in keystore
     */
    private fun isCertificateInstalled(context: Context, alias: String): Boolean {
        val hasKeyPair = getDeviceKeystoreManager(context).hasKeyPair(alias)
        Log.d(TAG, "Certificate '$alias' installation check: $hasKeyPair")
        return hasKeyPair
    }

    /**
     * Removes a certificate keypair from the Android keystore.
     *
     * @param context Android context
     * @param alias Certificate alias to remove
     * @return True if removal was successful or certificate doesn't exist
     */
    private fun removeKeyPair(context: Context, alias: String): Boolean = getDeviceKeystoreManager(context).removeKeyPair(alias)

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
     * Cleans up certificates based on host certificate assignments.
     *
     * This function handles certificate removal in two cases:
     * 1. Certificates with operation="remove": Process removal and mark as REMOVED
     * 2. Orphaned certificates with status=REMOVED: Delete tracking entry (cleanup complete)
     *
     * Removal flow:
     * - operation="remove" + status=INSTALLED: remove keypair, notify server, mark REMOVED
     * - operation="remove" + status=REMOVED: skip (already done)
     * - operation="remove" + not in DataStore: notify server as verified, save as REMOVED
     * - Orphaned + status=REMOVED: delete from DataStore
     *
     * @param context Android context for certificate operations
     * @param hostCertificates List of host certificates from managed configuration
     * @return Map of certificate ID to cleanup result
     */
    suspend fun cleanupRemovedCertificates(context: Context, hostCertificates: List<HostCertificate>): Map<Int, CleanupResult> {
        Log.d(TAG, "Starting certificate cleanup. Host certificates: ${hostCertificates.map { "${it.id}:${it.operation}" }}")

        val certificateStates = getCertificateStates(context)
        Log.d(TAG, "Found ${certificateStates.size} certificate(s) in DataStore")

        val results = mutableMapOf<Int, CleanupResult>()

        // Step 1: Process certificates with operation="remove"
        val certificatesToRemove = hostCertificates.filter { it.shouldRemove() }
        Log.d(TAG, "Certificates marked for removal: ${certificatesToRemove.map { it.id }}")

        for (hostCert in certificatesToRemove) {
            val certId = hostCert.id
            val certState = certificateStates[certId]

            when {
                certState?.status == CertificateStatus.REMOVED -> {
                    // Already removed, skip
                    Log.d(TAG, "Certificate ID $certId already removed, skipping")
                    results[certId] = CleanupResult.AlreadyRemoved(certState.alias)
                }
                certState?.status == CertificateStatus.REMOVED_UNREPORTED -> {
                    // Already removed but not yet reported to server; skip removal, will be retried
                    Log.d(TAG, "Certificate ID $certId already removed (unreported), skipping")
                    results[certId] = CleanupResult.AlreadyRemoved(certState.alias)
                }
                certState != null -> {
                    // Certificate exists in DataStore, remove it
                    val result = removeCertificateFromDevice(context, certId, certState.alias, certState.version)
                    results[certId] = result
                }
                else -> {
                    // Not in DataStore (never installed or already cleaned up)
                    // Notify server and save as REMOVED to prevent re-notification
                    Log.d(TAG, "Certificate ID $certId not in DataStore, notifying server as verified")
                    val alias = "cert-$certId"
                    apiClient.updateCertificateStatus(
                        certificateId = certId,
                        status = UpdateCertificateStatusStatus.VERIFIED,
                        operationType = UpdateCertificateStatusOperation.REMOVE,
                    ).onFailure { error ->
                        Log.e(TAG, "Failed to report removal status for ID $certId: ${error.message}", error)
                    }
                    markCertificateRemoved(context, certId, alias)
                    results[certId] = CleanupResult.Success(alias)
                }
            }
        }

        // Step 2: Clean up orphaned certificates with REMOVED status
        val hostCertIds = hostCertificates.map { it.id }.toSet()
        val orphanedCerts = certificateStates.filter { it.key !in hostCertIds }
        Log.d(TAG, "Orphaned certificates: ${orphanedCerts.keys}")

        for ((certId, certState) in orphanedCerts) {
            if (certState.status == CertificateStatus.REMOVED || certState.status == CertificateStatus.REMOVED_UNREPORTED) {
                // Removal complete (or unreported) and host certificate gone, clean up tracking
                Log.d(TAG, "Cleaning up tracking for removed certificate ID $certId")
                removeCertificateState(context, certId)
                results[certId] = CleanupResult.AlreadyRemoved(certState.alias)
            } else {
                // Orphaned but not removed - this is unexpected, remove it
                Log.w(TAG, "Orphaned certificate ID $certId with status ${certState.status}, removing")
                val result = removeCertificateFromDevice(context, certId, certState.alias, certState.version)
                results[certId] = result
            }
        }

        return results
    }

    /**
     * Removes a certificate from the device and updates tracking.
     */
    private suspend fun removeCertificateFromDevice(context: Context, certificateId: Int, alias: String, version: Int): CleanupResult {
        Log.d(TAG, "Removing certificate ID $certificateId with alias '$alias'")

        val removed = removeKeyPair(context, alias)

        return if (removed) {
            // First, mark as unreported (persisted before network call)
            markCertificateUnreported(context, certificateId, alias, version = version, isInstall = false)

            // Attempt to report status
            val reportResult = apiClient.updateCertificateStatus(
                certificateId = certificateId,
                status = UpdateCertificateStatusStatus.VERIFIED,
                operationType = UpdateCertificateStatusOperation.REMOVE,
            )

            if (reportResult.isSuccess) {
                // Status reported successfully, mark as fully removed
                markCertificateRemoved(context, certificateId, alias)
            } else {
                // Status report failed; leave as REMOVED_UNREPORTED for retry later
                Log.w(
                    TAG,
                    "Removal status report failed for certificate $certificateId, will retry later: ${reportResult.exceptionOrNull()?.message}",
                )
            }

            Log.i(TAG, "Successfully removed certificate ID $certificateId (alias: '$alias')")
            CleanupResult.Success(alias)
        } else {
            val errorDetail = "Failed to remove certificate keypair from device"
            apiClient.updateCertificateStatus(
                certificateId = certificateId,
                status = UpdateCertificateStatusStatus.FAILED,
                operationType = UpdateCertificateStatusOperation.REMOVE,
                detail = errorDetail,
            ).onFailure { error ->
                Log.e(TAG, "Failed to report removal failure for ID $certificateId: ${error.message}", error)
            }

            Log.e(TAG, "Failed to remove certificate ID $certificateId (alias: '$alias')")
            CleanupResult.Failure(
                reason = errorDetail,
                exception = null,
                shouldRetry = false,
            )
        }
    }

    /**
     * Marks a certificate as removed in DataStore.
     */
    private suspend fun markCertificateRemoved(context: Context, certificateId: Int, alias: String) {
        val info = CertificateState(alias = alias, status = CertificateStatus.REMOVED)
        storeCertificateState(context, certificateId, info)
    }

    /**
     * Marks a certificate as unreported after successful install/remove.
     * We persist this state before attempting the network call so that we can retry later if needed.
     *
     * @param context Android context
     * @param certificateId Certificate template ID
     * @param alias Certificate alias
     * @param isInstall True for install operation, false for remove operation
     */
    internal suspend fun markCertificateUnreported(context: Context, certificateId: Int, alias: String, version: Int, isInstall: Boolean) {
        val status = if (isInstall) {
            CertificateStatus.INSTALLED_UNREPORTED
        } else {
            CertificateStatus.REMOVED_UNREPORTED
        }
        val info = CertificateState(alias = alias, status = status, statusReportRetries = 0, version = version)
        storeCertificateState(context, certificateId, info)
    }

    /**
     * Increments the status report retry count for a certificate.
     * If max retries reached, transitions to final status (INSTALLED or REMOVED).
     *
     * @param context Android context
     * @param certificateId Certificate template ID
     * @return The updated CertificateState, or null if not found
     */
    internal suspend fun incrementStatusReportRetries(context: Context, certificateId: Int): CertificateState? {
        val existingState = getCertificateState(context, certificateId) ?: return null

        val newRetries = existingState.statusReportRetries + 1
        val newStatus = when {
            newRetries >= MAX_STATUS_REPORT_RETRIES -> {
                // Max retries reached, transition to final status
                when (existingState.status) {
                    CertificateStatus.INSTALLED_UNREPORTED -> CertificateStatus.INSTALLED
                    CertificateStatus.REMOVED_UNREPORTED -> CertificateStatus.REMOVED
                    else -> existingState.status
                }
            }
            else -> existingState.status
        }

        val updatedState = existingState.copy(
            status = newStatus,
            statusReportRetries = newRetries,
        )
        storeCertificateState(context, certificateId, updatedState)

        if (newRetries >= MAX_STATUS_REPORT_RETRIES) {
            Log.w(TAG, "Certificate $certificateId reached max status report retries ($MAX_STATUS_REPORT_RETRIES), giving up")
        }

        return updatedState
    }

    /**
     * Retries unreported statuses for certificates that were installed/removed
     * but whose status wasn't successfully reported to the server.
     *
     * @param context Android context
     * @return Map of certificate ID to success (true) or failure (false)
     */
    suspend fun retryUnreportedStatuses(context: Context): Map<Int, Boolean> {
        val states = getCertificateStates(context)
        val results = mutableMapOf<Int, Boolean>()

        val unreportedStates = states.filter { (_, state) ->
            state.status == CertificateStatus.INSTALLED_UNREPORTED ||
                state.status == CertificateStatus.REMOVED_UNREPORTED
        }

        for ((certId, state) in unreportedStates) {
            val isInstall = state.status == CertificateStatus.INSTALLED_UNREPORTED
            val operationType = if (isInstall) UpdateCertificateStatusOperation.INSTALL else UpdateCertificateStatusOperation.REMOVE
            val operationName = if (isInstall) "install" else "removal"

            Log.d(TAG, "Retrying status report for $operationName certificate $certId (attempt ${state.statusReportRetries + 1})")

            val result = apiClient.updateCertificateStatus(
                certificateId = certId,
                status = UpdateCertificateStatusStatus.VERIFIED,
                operationType = operationType,
            )

            if (result.isSuccess) {
                if (isInstall) {
                    markCertificateInstalled(context, certId, state.alias, state.version)
                } else {
                    markCertificateRemoved(context, certId, state.alias)
                }
                Log.i(TAG, "Successfully reported $operationName status for certificate $certId")
                results[certId] = true
            } else {
                val updatedState = incrementStatusReportRetries(context, certId)
                Log.w(TAG, "Failed to report $operationName status for certificate $certId: ${result.exceptionOrNull()?.message}")
                results[certId] = false

                val finalStatus = if (isInstall) CertificateStatus.INSTALLED else CertificateStatus.REMOVED
                if (updatedState?.status == finalStatus) {
                    Log.w(TAG, "Gave up reporting $operationName status for certificate $certId after $MAX_STATUS_REPORT_RETRIES attempts")
                }
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
     * @param version Version number from managed config, used to detect when reinstallation is needed
     * @param certificateInstaller Certificate installer implementation (defaults to AndroidCertificateInstaller)
     * @return EnrollmentResult indicating success or failure with details
     */
    suspend fun enrollCertificate(
        context: Context,
        certificateId: Int,
        version: Int,
        certificateInstaller: CertificateEnrollmentHandler.CertificateInstaller? = null,
    ): CertificateEnrollmentHandler.EnrollmentResult {
        Log.d(TAG, "Starting certificate enrollment for certificate ID: $certificateId (version: $version)")

        // Check if certificate is already installed with matching version (BEFORE API call)
        val storedState = getCertificateState(context, certificateId)
        if (storedState != null) {
            val existsInKeystore = isCertificateInstalled(context, storedState.alias)
            if (existsInKeystore && storedState.version == version && storedState.status == CertificateStatus.INSTALLED) {
                Log.i(
                    TAG,
                    "Certificate ID $certificateId (alias: '${storedState.alias}', version: $version) is already installed, skipping enrollment",
                )
                return CertificateEnrollmentHandler.EnrollmentResult.Success(storedState.alias)
            }
            if (existsInKeystore && storedState.version != version) {
                Log.i(TAG, "Certificate ID $certificateId version changed (${storedState.version} -> $version), will reinstall")
            }
        }

        // Skip enrollment if already marked as permanently failed (max retries exceeded),
        // unless the version changed (server wants a fresh install).
        if (storedState?.status == CertificateStatus.FAILED && storedState.version == version) {
            return CertificateEnrollmentHandler.EnrollmentResult.Success(storedState.alias)
        }

        // Fetch certificate template from API (only if not already installed)
        val templateResult = apiClient.getCertificateTemplate(certificateId)
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

                // First, mark as unreported (persisted before network call)
                markCertificateUnreported(context, certificateId, template.name, version = version, isInstall = true)

                // Attempt to report status
                val reportResult = apiClient.updateCertificateStatus(
                    certificateId = certificateId,
                    status = UpdateCertificateStatusStatus.VERIFIED,
                    operationType = UpdateCertificateStatusOperation.INSTALL,
                )

                if (reportResult.isSuccess) {
                    // Status reported successfully, mark as fully installed
                    markCertificateInstalled(context, certificateId = certificateId, alias = template.name, version = version)
                } else {
                    // Status report failed - leave as INSTALLED_UNREPORTED for retry later
                    Log.w(
                        TAG,
                        "Status report failed for certificate $certificateId, will retry later: ${reportResult.exceptionOrNull()?.message}",
                    )
                }
            }
            is CertificateEnrollmentHandler.EnrollmentResult.Failure -> {
                val updatedInfo = markCertificateFailure(context = context, certificateId = certificateId, alias = template.name)
                if (!updatedInfo.shouldRetry()) {
                    Log.e(TAG, "Certificate enrollment failed for ID $certificateId: ${result.reason}", result.exception)
                    apiClient.updateCertificateStatus(
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
     * @param hostCertificates List of certificate templates to enroll
     * @return Map of certificate ID to enrollment result
     */
    suspend fun enrollCertificates(
        context: Context,
        hostCertificates: List<HostCertificate>,
    ): Map<Int, CertificateEnrollmentHandler.EnrollmentResult> = coroutineScope {
        Log.d(TAG, "Starting batch certificate enrollment for ${hostCertificates.size} certificates")

        hostCertificates.associate { cert ->
            cert.id to async {
                enrollCertificate(context, cert.id, cert.version)
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
        companion object {
            private const val TAG = "fleet-AndroidCertInstaller"
        }

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

typealias CertificateStateMap = Map<Int, CertificateState>

@Serializable
enum class CertificateStatus {
    @SerialName("installed")
    INSTALLED,

    @SerialName("installed_unreported")
    INSTALLED_UNREPORTED,

    @SerialName("failed")
    FAILED,

    @SerialName("retry")
    RETRY,

    @SerialName("removed")
    REMOVED,

    @SerialName("removed_unreported")
    REMOVED_UNREPORTED,
}

@Serializable
data class CertificateState(
    @SerialName("alias")
    val alias: String,
    @SerialName("status")
    val status: CertificateStatus,
    @SerialName("retries")
    val retries: Int = 0,
    @SerialName("status_report_retries")
    val statusReportRetries: Int = 0,
    @SerialName("version")
    val version: Int = 1,
) {
    fun shouldRetry(): Boolean = status == CertificateStatus.RETRY && retries < (MAX_CERT_INSTALL_RETRIES)
    fun shouldRetryStatusReport(): Boolean = statusReportRetries < MAX_STATUS_REPORT_RETRIES
}

/**
 * Result of certificate cleanup operation
 */
sealed class CleanupResult {
    data class Success(val alias: String) : CleanupResult()
    data class Failure(val reason: String, val exception: Exception?, val shouldRetry: Boolean) : CleanupResult()
    data class AlreadyRemoved(val alias: String) : CleanupResult()
}

/**
 * Represents a certificate template from managed configuration.
 *
 * @property id Certificate template ID
 * @property status Current status of the certificate template
 * @property operation Operation to perform: "install" or "remove"
 * @property version Version number, incremented by server to trigger reinstallation
 */
data class HostCertificate(val id: Int, val status: String, val operation: String, val version: Int) {
    companion object {
        const val OPERATION_INSTALL = "install"
        const val OPERATION_REMOVE = "remove"
    }

    fun shouldInstall(): Boolean = operation == OPERATION_INSTALL
    fun shouldRemove(): Boolean = operation == OPERATION_REMOVE
}
