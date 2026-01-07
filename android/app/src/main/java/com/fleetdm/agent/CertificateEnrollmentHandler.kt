package com.fleetdm.agent

import com.fleetdm.agent.scep.ScepCertificateException
import com.fleetdm.agent.scep.ScepClient
import com.fleetdm.agent.scep.ScepCsrException
import com.fleetdm.agent.scep.ScepEnrollmentException
import com.fleetdm.agent.scep.ScepKeyGenerationException
import com.fleetdm.agent.scep.ScepNetworkException
import java.math.BigInteger
import java.security.PrivateKey
import java.security.cert.Certificate
import java.util.Date

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
        data class Success(
            val alias: String,
            val notAfter: Date,
            val notBefore: Date,
            val serialNumber: BigInteger,
        ) : EnrollmentResult()
        data class Failure(val reason: String, val exception: Exception? = null, val isRetryable: Boolean = false) : EnrollmentResult()
        data class PermanentlyFailed(val alias: String) : EnrollmentResult()
    }

    /**
     * Main enrollment flow: parse config, enroll via SCEP, install certificate.
     */
    suspend fun handleEnrollment(config: GetCertificateTemplateResponse, scepUrl: String): EnrollmentResult = try {
        // Perform SCEP enrollment
        val result = scepClient.enroll(config, scepUrl)

        // Install certificate
        val installed = certificateInstaller.installCertificate(
            config.name,
            result.privateKey,
            result.certificateChain.toTypedArray(),
        )

        if (installed) {
            EnrollmentResult.Success(
                alias = config.name,
                notAfter = result.notAfter,
                notBefore = result.notBefore,
                serialNumber = result.serialNumber,
            )
        } else {
            EnrollmentResult.Failure("Certificate installation failed")
        }
    } catch (e: ScepEnrollmentException) {
        // SCEP server rejected enrollment (e.g., PENDING status, invalid challenge)
        EnrollmentResult.Failure("SCEP enrollment failed: ${e.message}", e, isRetryable = false)
    } catch (e: ScepNetworkException) {
        // Network communication failure - likely transient, can retry
        EnrollmentResult.Failure("Network error during SCEP enrollment: ${e.message}", e, isRetryable = true)
    } catch (e: ScepCertificateException) {
        // Certificate validation or processing failed
        EnrollmentResult.Failure("Certificate validation failed: ${e.message}", e, isRetryable = false)
    } catch (e: ScepKeyGenerationException) {
        // Key generation failed - device cryptography issue
        EnrollmentResult.Failure("Failed to generate key pair: ${e.message}", e, isRetryable = false)
    } catch (e: ScepCsrException) {
        // CSR creation failed - likely configuration issue
        EnrollmentResult.Failure("Failed to create CSR: ${e.message}", e, isRetryable = false)
    } catch (e: IllegalArgumentException) {
        // Configuration validation failed
        EnrollmentResult.Failure("Invalid configuration: ${e.message}", e, isRetryable = false)
    } catch (e: Exception) {
        // Unexpected errors
        EnrollmentResult.Failure("Unexpected error during enrollment: ${e.message}", e, isRetryable = false)
    }
}
