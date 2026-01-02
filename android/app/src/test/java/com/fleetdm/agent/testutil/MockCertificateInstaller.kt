package com.fleetdm.agent.testutil

import com.fleetdm.agent.CertificateEnrollmentHandler
import java.security.PrivateKey
import java.security.cert.Certificate

/**
 * Mock certificate installer for testing.
 * Provides configurable behavior for different test scenarios.
 */
class MockCertificateInstaller : CertificateEnrollmentHandler.CertificateInstaller {
    var shouldSucceed = true
    var wasInstallCalled = false
    var capturedAlias: String? = null
    var capturedPrivateKey: PrivateKey? = null
    var capturedCertificateChain: Array<Certificate>? = null

    override fun installCertificate(
        alias: String,
        privateKey: PrivateKey,
        certificateChain: Array<Certificate>,
    ): Boolean {
        wasInstallCalled = true
        capturedAlias = alias
        capturedPrivateKey = privateKey
        capturedCertificateChain = certificateChain
        return shouldSucceed
    }

    fun reset() {
        shouldSucceed = true
        wasInstallCalled = false
        capturedAlias = null
        capturedPrivateKey = null
        capturedCertificateChain = null
    }
}
