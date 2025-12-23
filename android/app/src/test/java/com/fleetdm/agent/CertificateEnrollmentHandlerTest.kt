package com.fleetdm.agent

import com.fleetdm.agent.scep.MockScepClient
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.security.PrivateKey
import java.security.cert.Certificate
import kotlinx.coroutines.test.runTest

/**
 * Unit tests for CertificateEnrollmentHandler.
 *
 * Tests the business logic without Android framework dependencies.
 */
class CertificateEnrollmentHandlerTest {

    private lateinit var handler: CertificateEnrollmentHandler
    private lateinit var mockScepClient: MockScepClient
    private lateinit var mockInstaller: MockCertificateInstaller

    @Before
    fun setup() {
        mockScepClient = MockScepClient()
        mockInstaller = MockCertificateInstaller()
        handler = CertificateEnrollmentHandler(
            scepClient = mockScepClient,
            certificateInstaller = mockInstaller,
        )
    }

    @After
    fun tearDown() {
        mockScepClient.reset()
        mockInstaller.reset()
    }

    /**
     * Mock certificate installer for testing.
     */
    class MockCertificateInstaller : CertificateEnrollmentHandler.CertificateInstaller {
        var wasInstallCalled = false
        var capturedAlias: String? = null
        var capturedPrivateKey: PrivateKey? = null
        var capturedCertificateChain: Array<Certificate>? = null
        var shouldSucceed = true

        override fun installCertificate(alias: String, privateKey: PrivateKey, certificateChain: Array<Certificate>): Boolean {
            wasInstallCalled = true
            capturedAlias = alias
            capturedPrivateKey = privateKey
            capturedCertificateChain = certificateChain
            return shouldSucceed
        }

        fun reset() {
            wasInstallCalled = false
            capturedAlias = null
            capturedPrivateKey = null
            capturedCertificateChain = null
            shouldSucceed = true
        }
    }

    @Test
    fun `handler enrolls with valid certificate template`() = runTest {
        val template = createValidCertificateTemplate()

        val result = handler.handleEnrollment(template)

        // Verify SCEP client was called with correct config
        assertNotNull(mockScepClient.capturedConfig)
        assertEquals("https://scep.example.com/cgi-bin/pkiclient.exe", mockScepClient.capturedConfig?.url)
        assertEquals("secret123", mockScepClient.capturedConfig?.scepChallenge)
        assertEquals("device-cert", mockScepClient.capturedConfig?.name)
        assertEquals("CN=Device123,O=FleetDM", mockScepClient.capturedConfig?.subjectName)

        // Verify success
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Success)
    }

    @Test
    fun `handler installs certificate after successful enrollment`() = runTest {
        val template = createValidCertificateTemplate()

        val result = handler.handleEnrollment(template)

        // Verify certificate installer was called
        assertTrue(mockInstaller.wasInstallCalled)
        assertEquals("device-cert", mockInstaller.capturedAlias)
        assertNotNull(mockInstaller.capturedPrivateKey)
        assertNotNull(mockInstaller.capturedCertificateChain)

        // Verify success
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Success)
        assertEquals("device-cert", (result as CertificateEnrollmentHandler.EnrollmentResult.Success).alias)
    }

    @Test
    fun `handler handles enrollment failure gracefully`() = runTest {
        mockScepClient.shouldThrowEnrollmentException = true

        val template = createValidCertificateTemplate()

        val result = handler.handleEnrollment(template)

        // Verify certificate installer was NOT called since enrollment failed
        assertFalse(mockInstaller.wasInstallCalled)

        // Verify failure result
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Failure)
    }

    @Test
    fun `handler handles network exception gracefully`() = runTest {
        mockScepClient.shouldThrowNetworkException = true

        val template = createValidCertificateTemplate()

        val result = handler.handleEnrollment(template)

        // Verify certificate installer was NOT called
        assertFalse(mockInstaller.wasInstallCalled)

        // Verify failure result
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Failure)
    }

    @Test
    fun `handler handles installation failure`() = runTest {
        mockInstaller.shouldSucceed = false

        val template = createValidCertificateTemplate()

        val result = handler.handleEnrollment(template)

        // Verify enrollment succeeded but installation failed
        assertTrue(mockInstaller.wasInstallCalled)
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Failure)
    }

    @Test
    fun `handler uses custom key length and signature algorithm`() = runTest {
        val template = createValidCertificateTemplate(
            keyLength = 4096,
            signatureAlgorithm = "SHA512withRSA",
        )

        handler.handleEnrollment(template)

        // Verify config was used correctly
        assertEquals(4096, mockScepClient.capturedConfig?.keyLength)
        assertEquals("SHA512withRSA", mockScepClient.capturedConfig?.signatureAlgorithm)
    }

    @Test
    fun `handler uses default values for optional parameters`() = runTest {
        val template = createValidCertificateTemplate()

        handler.handleEnrollment(template)

        // Verify defaults were used
        assertEquals(2048, mockScepClient.capturedConfig?.keyLength)
        assertEquals("SHA256withRSA", mockScepClient.capturedConfig?.signatureAlgorithm)
    }

    // Helper functions

    private fun createValidCertificateTemplate(
        id: Int = 1,
        name: String = "device-cert",
        scepUrl: String = "https://scep.example.com/cgi-bin/pkiclient.exe",
        scepChallenge: String = "secret123",
        subjectName: String = "CN=Device123,O=FleetDM",
        keyLength: Int = 2048,
        signatureAlgorithm: String = "SHA256withRSA",
    ): GetCertificateTemplateResponse = GetCertificateTemplateResponse(
        id = id,
        name = name,
        certificateAuthorityId = 123,
        certificateAuthorityName = "Test CA",
        createdAt = "2024-01-01T00:00:00Z",
        subjectName = subjectName,
        certificateAuthorityType = "SCEP",
        status = "active",
        scepChallenge = scepChallenge,
        fleetChallenge = "fleet-secret",
        keyLength = keyLength,
        signatureAlgorithm = signatureAlgorithm,
        url = scepUrl,
    )
}
