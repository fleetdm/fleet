package com.fleetdm.agent.scep

import com.fleetdm.agent.GetCertificateTemplateResponse

/**
 * Interface for SCEP (Simple Certificate Enrollment Protocol) client operations.
 *
 * Implementations of this interface handle the complete SCEP enrollment process:
 * 1. Key pair generation
 * 2. Certificate Signing Request (CSR) creation
 * 3. Communication with SCEP server
 * 4. Certificate retrieval and validation
 */
interface ScepClient {
    /**
     * Performs SCEP enrollment to obtain a certificate from a SCEP server.
     *
     * @param config The SCEP enrollment configuration
     * @param scepUrl The SCEP server URL to enroll against
     * @return ScepResult containing the private key and certificate chain
     * @throws ScepException if enrollment fails
     */
    suspend fun enroll(config: GetCertificateTemplateResponse, scepUrl: String): ScepResult
}
