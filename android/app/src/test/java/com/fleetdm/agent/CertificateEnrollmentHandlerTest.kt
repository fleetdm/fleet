package com.fleetdm.agent

import com.fleetdm.agent.scep.MockScepClient
import com.fleetdm.agent.testutil.MockCertificateInstaller
import com.fleetdm.agent.testutil.TestCertificateTemplateFactory
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
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

    @Test
    fun `handler enrolls with valid certificate template`() = runTest {
        val template = TestCertificateTemplateFactory.create(
            name = "device-cert",
            scepChallenge = "secret123",
            subjectName = "CN=Device123,O=FleetDM",
        )

        val result = handler.handleEnrollment(template, TestCertificateTemplateFactory.DEFAULT_SCEP_URL)

        // Verify SCEP client was called with correct config and URL
        assertNotNull(mockScepClient.capturedConfig)
        assertEquals(TestCertificateTemplateFactory.DEFAULT_SCEP_URL, mockScepClient.capturedScepUrl)
        assertEquals("secret123", mockScepClient.capturedConfig?.scepChallenge)
        assertEquals("device-cert", mockScepClient.capturedConfig?.name)
        assertEquals("CN=Device123,O=FleetDM", mockScepClient.capturedConfig?.subjectName)

        // Verify success
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Success)
    }

    @Test
    fun `handler installs certificate after successful enrollment`() = runTest {
        val template = TestCertificateTemplateFactory.create(name = "device-cert")

        val result = handler.handleEnrollment(template, TestCertificateTemplateFactory.DEFAULT_SCEP_URL)

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

        val template = TestCertificateTemplateFactory.create()

        val result = handler.handleEnrollment(template, TestCertificateTemplateFactory.DEFAULT_SCEP_URL)

        // Verify certificate installer was NOT called since enrollment failed
        assertFalse(mockInstaller.wasInstallCalled)

        // Verify failure result - enrollment failures are not retryable
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Failure)
        assertFalse((result as CertificateEnrollmentHandler.EnrollmentResult.Failure).isRetryable)
    }

    @Test
    fun `handler handles network exception gracefully`() = runTest {
        mockScepClient.shouldThrowNetworkException = true

        val template = TestCertificateTemplateFactory.create()

        val result = handler.handleEnrollment(template, TestCertificateTemplateFactory.DEFAULT_SCEP_URL)

        // Verify certificate installer was NOT called
        assertFalse(mockInstaller.wasInstallCalled)

        // Verify failure result - network failures are retryable
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Failure)
        assertTrue((result as CertificateEnrollmentHandler.EnrollmentResult.Failure).isRetryable)
    }

    @Test
    fun `handler handles installation failure`() = runTest {
        mockInstaller.shouldSucceed = false

        val template = TestCertificateTemplateFactory.create()

        val result = handler.handleEnrollment(template, TestCertificateTemplateFactory.DEFAULT_SCEP_URL)

        // Verify enrollment succeeded but installation failed - installation failures are not retryable
        assertTrue(mockInstaller.wasInstallCalled)
        assertTrue(result is CertificateEnrollmentHandler.EnrollmentResult.Failure)
        assertFalse((result as CertificateEnrollmentHandler.EnrollmentResult.Failure).isRetryable)
    }

    @Test
    fun `handler uses custom key length and signature algorithm`() = runTest {
        val template = TestCertificateTemplateFactory.create(
            keyLength = 4096,
            signatureAlgorithm = "SHA512withRSA",
        )

        handler.handleEnrollment(template, TestCertificateTemplateFactory.DEFAULT_SCEP_URL)

        // Verify config was used correctly
        assertEquals(4096, mockScepClient.capturedConfig?.keyLength)
        assertEquals("SHA512withRSA", mockScepClient.capturedConfig?.signatureAlgorithm)
    }

    @Test
    fun `handler uses default values for optional parameters`() = runTest {
        val template = TestCertificateTemplateFactory.create()

        handler.handleEnrollment(template, TestCertificateTemplateFactory.DEFAULT_SCEP_URL)

        // Verify defaults were used
        assertEquals(2048, mockScepClient.capturedConfig?.keyLength)
        assertEquals("SHA256withRSA", mockScepClient.capturedConfig?.signatureAlgorithm)
    }
}
