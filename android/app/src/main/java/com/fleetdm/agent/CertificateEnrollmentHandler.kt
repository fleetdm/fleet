package com.fleetdm.agent

import com.fleetdm.agent.scep.ScepClient
import com.fleetdm.agent.scep.ScepConfig
import com.fleetdm.agent.scep.ScepEnrollmentException
import com.fleetdm.agent.scep.ScepException
import com.fleetdm.agent.scep.ScepResult
import org.json.JSONObject
import java.security.PrivateKey
import java.security.cert.Certificate

/**
 * Handles certificate enrollment business logic without Android framework dependencies.
 * Can be easily tested without Robolectric.
 */
class CertificateEnrollmentHandler(private val scepClient: ScepClient, private val certificateInstaller: CertificateInstaller) {

    /**
     * Interface for certificate installation - allows different implementations
     * (production uses DevicePolicyManager, tests use mocks).
     */
    interface CertificateInstaller {
        fun installCertificate(alias: String, privateKey: PrivateKey, certificateChain: Array<Certificate>): Boolean
    }

    /**
     * Result of enrollment operation.
     */
    sealed class EnrollmentResult {
        data class Success(val alias: String) : EnrollmentResult()
        data class Failure(val reason: String, val exception: Exception? = null) : EnrollmentResult()
    }

    /**
     * Main enrollment flow: parse config, enroll via SCEP, install certificate.
     */
    suspend fun handleEnrollment(config: GetCertificateTemplateResponse): EnrollmentResult {
        return try {
            // Step 2: Perform SCEP enrollment
            val result = performEnrollment(config) ?: return EnrollmentResult.Failure(
                reason = "SCEP enrollment failed or returned null",
                exception = null,
            )

            // Step 3: Install certificate
            val installed = certificateInstaller.installCertificate(
                config.name,
                result.privateKey,
                result.certificateChain.toTypedArray(),
            )

            if (installed) {
                EnrollmentResult.Success(config.name)
            } else {
                EnrollmentResult.Failure("Certificate installation failed")
            }
        } catch (e: IllegalArgumentException) {
            EnrollmentResult.Failure("Invalid configuration: ${e.message}", e)
        } catch (e: Exception) {
            EnrollmentResult.Failure("Unexpected error: ${e.message}", e)
        }
    }

    /**
     * Performs SCEP enrollment, returning result or null on failure.
     */
    @Suppress("SwallowedException")
    suspend fun performEnrollment(config: GetCertificateTemplateResponse): ScepResult? = try {
        scepClient.enroll(config)
    } catch (e: ScepEnrollmentException) {
        // Enrollment failure is expected in some scenarios (pending approval, invalid challenge)
        null
    } catch (e: ScepException) {
        // SCEP protocol errors are expected in some scenarios
        null
    } catch (e: Exception) {
        // Unexpected errors are logged by the SCEP client
        null
    }
}
