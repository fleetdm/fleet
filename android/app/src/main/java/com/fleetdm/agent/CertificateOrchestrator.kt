package com.fleetdm.agent

import android.app.admin.DevicePolicyManager
import android.content.Context
import android.os.Bundle
import android.util.Log
import com.fleetdm.agent.scep.ScepClient
import com.fleetdm.agent.scep.ScepClientImpl
import java.security.PrivateKey
import java.security.cert.Certificate
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope

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

    /**
     * Reads certificate IDs from Android Managed Configuration.
     *
     * @param context Android context
     * @return List of certificate IDs to enroll, or null if none configured
     */
    fun getCertificateIDs(context: Context): List<Int>? {
        val restrictionsManager = context.getSystemService(Context.RESTRICTIONS_SERVICE) as android.content.RestrictionsManager
        val appRestrictions = restrictionsManager.applicationRestrictions

        val certRequestList = appRestrictions.getParcelableArray("certificates", Bundle::class.java)?.toList()
        return certRequestList?.map { bundle -> bundle.getInt("certificate_id") }
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

        // Step 1: Fetch certificate template from API
        val templateResult = ApiClient.getCertificateTemplate(certificateId)
        val template = templateResult.getOrElse { error ->
            Log.e(TAG, "Failed to fetch certificate template for ID $certificateId: ${error.message}", error)
            return CertificateEnrollmentHandler.EnrollmentResult.Failure(
                reason = "Failed to fetch certificate template: ${error.message}",
                exception = error as? Exception,
            )
        }

        Log.d(TAG, "Successfully fetched certificate template: ${template.name}")

        // Step 2: Create certificate installer (use provided or create default)
        val installer = certificateInstaller ?: AndroidCertificateInstaller(context)

        // Step 3: Create enrollment handler
        val handler = CertificateEnrollmentHandler(
            scepClient = scepClient,
            certificateInstaller = installer,
        )

        // Step 4: Perform enrollment
        Log.d(TAG, "Starting SCEP enrollment for certificate: ${template.name}")
        val result = handler.handleEnrollment(template)

        when (result) {
            is CertificateEnrollmentHandler.EnrollmentResult.Success -> {
                Log.i(TAG, "Certificate enrollment successful for ID $certificateId with alias: ${result.alias}")
            }
            is CertificateEnrollmentHandler.EnrollmentResult.Failure -> {
                Log.e(TAG, "Certificate enrollment failed for ID $certificateId: ${result.reason}", result.exception)
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
