package com.fleetdm.agent

import com.fleetdm.agent.scep.MockScepClient
import kotlinx.coroutines.test.runTest
import org.json.JSONObject
import org.junit.After
import org.junit.Assert.*
import org.junit.Before
import org.junit.Test
import java.security.PrivateKey
import java.security.cert.Certificate

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
            certificateInstaller = mockInstaller
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

        override fun installCertificate(
            alias: String,
            privateKey: PrivateKey,
            certificateChain: Array<Certificate>
        ): Boolean {
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
    fun `handler enrolls with valid CERT_DATA`() = runTest {
        val certData = createValidCertDataJson()

        val result = handler.handleEnrollment(certData.toString())

        // Verify SCEP client was called with correct config
        assertNotNull(mockScepClient.capturedConfig)
        assertEquals("https://scep.example.com/cgi-bin/pkiclient.exe", mockScepClient.capturedConfig?.url)
        assertEquals("secret123", mockScepClient.capturedConfig?.challenge)
        assertEquals("device-cert", mockScepClient.capturedConfig?.alias)
        assertEquals("CN=Device123,O=FleetDM", mockScepClient.capturedConfig?.subject)

        // Verify success
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Success)
    }

    @Test
    fun `handler installs certificate after successful enrollment`() = runTest {
        val certData = createValidCertDataJson()

        val result = handler.handleEnrollment(certData.toString())

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

        val certData = createValidCertDataJson()

        val result = handler.handleEnrollment(certData.toString())

        // Verify certificate installer was NOT called since enrollment failed
        assertFalse(mockInstaller.wasInstallCalled)

        // Verify failure result
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Failure)
    }

    @Test
    fun `handler handles network exception gracefully`() = runTest {
        mockScepClient.shouldThrowNetworkException = true

        val certData = createValidCertDataJson()

        val result = handler.handleEnrollment(certData.toString())

        // Verify certificate installer was NOT called
        assertFalse(mockInstaller.wasInstallCalled)

        // Verify failure result
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Failure)
    }

    @Test
    fun `handler handles installation failure`() = runTest {
        mockInstaller.shouldSucceed = false

        val certData = createValidCertDataJson()

        val result = handler.handleEnrollment(certData.toString())

        // Verify enrollment succeeded but installation failed
        assertTrue(mockInstaller.wasInstallCalled)
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Failure)
    }

    @Test
    fun `handler parses custom key length and signature algorithm`() = runTest {
        val certData = JSONObject().apply {
            put("scep_url", "https://scep.example.com/cgi-bin/pkiclient.exe")
            put("challenge", "secret123")
            put("alias", "device-cert")
            put("subject", "CN=Device123,O=FleetDM")
            put("key_length", 4096)
            put("signature_algorithm", "SHA512withRSA")
        }

        handler.handleEnrollment(certData.toString())

        // Verify config was parsed correctly
        assertEquals(4096, mockScepClient.capturedConfig?.keyLength)
        assertEquals("SHA512withRSA", mockScepClient.capturedConfig?.signatureAlgorithm)
    }

    @Test
    fun `handler uses default values when optional parameters missing`() = runTest {
        val certData = JSONObject().apply {
            put("scep_url", "https://scep.example.com/cgi-bin/pkiclient.exe")
            put("challenge", "secret123")
            put("alias", "device-cert")
            put("subject", "CN=Device123,O=FleetDM")
            // key_length and signature_algorithm not provided
        }

        handler.handleEnrollment(certData.toString())

        // Verify defaults were used
        assertEquals(2048, mockScepClient.capturedConfig?.keyLength)
        assertEquals("SHA256withRSA", mockScepClient.capturedConfig?.signatureAlgorithm)
    }

    @Test
    fun `handler rejects invalid JSON`() = runTest {
        val invalidJson = "not valid json"

        val result = handler.handleEnrollment(invalidJson)

        // Verify failure result
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Failure)
        val failure = result as CertificateEnrollmentHandler.EnrollmentResult.Failure
        assertTrue(failure.reason.contains("Invalid configuration"))
    }

    // Helper functions

    private fun createValidCertDataJson(): JSONObject {
        return JSONObject().apply {
            put("scep_url", "https://scep.example.com/cgi-bin/pkiclient.exe")
            put("challenge", "secret123")
            put("alias", "device-cert")
            put("subject", "CN=Device123,O=FleetDM")
        }
    }
}
