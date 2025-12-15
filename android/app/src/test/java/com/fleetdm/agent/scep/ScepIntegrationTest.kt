package com.fleetdm.agent.scep

import com.fleetdm.agent.GetCertificateTemplateResponse
import com.fleetdm.agent.IntegrationTest
import com.fleetdm.agent.IntegrationTestRule
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import kotlinx.coroutines.test.runTest

/**
 * Integration tests for ScepClientImpl with a real SCEP server.
 *
 * These tests only run when explicitly enabled:
 * - Local: ./gradlew test -PrunIntegrationTests=true
 * - CI: ./gradlew test -PrunIntegrationTests=true
 *
 * Requirements:
 * 1. A test SCEP server is available
 * 2. Test credentials are configured
 * 3. Network connectivity is available
 *
 * Configure SCEP server:
 * ./gradlew test -PrunIntegrationTests=true \
 *   -Pscep.url=https://your-scep-server.com/scep \
 *   -Pscep.challenge=your-challenge
 */
class ScepIntegrationTest {

    @get:Rule
    val integrationTestRule = IntegrationTestRule()

    private lateinit var scepClient: ScepClientImpl
    private lateinit var testTemplate: GetCertificateTemplateResponse

    @Before
    fun setup() {
        scepClient = ScepClientImpl()

        // Use placeholder values for non-integration tests, real values provided by build config for integration tests
        val scepUrl = System.getProperty("scep.url") ?: "https://scep.example.com/scep"
        val challenge = System.getProperty("scep.challenge") ?: "test-challenge"

        // Generate unique subject DN to avoid duplicates on SCEP server
        val uniqueId = System.currentTimeMillis()
        testTemplate = createTemplate(
            url = scepUrl,
            challenge = challenge,
            name = "integration-test-cert-$uniqueId",
            subject = "CN=IntegrationTestDevice-$uniqueId,O=FleetDM,C=US",
        )
    }

    private fun createTemplate(
        url: String,
        challenge: String,
        name: String,
        subject: String,
        keyLength: Int = 2048,
    ): GetCertificateTemplateResponse = GetCertificateTemplateResponse(
        id = 1,
        name = name,
        certificateAuthorityId = 123,
        certificateAuthorityName = "Test CA",
        createdAt = "2024-01-01T00:00:00Z",
        subjectName = subject,
        certificateAuthorityType = "SCEP",
        status = "active",
        scepChallenge = challenge,
        fleetChallenge = "fleet-secret",
        keyLength = keyLength,
        signatureAlgorithm = "SHA256withRSA",
        url = url,
    )

    @IntegrationTest
    @Test
    fun `successful enrollment with real SCEP server`() = runTest {
        // This test requires a real SCEP server with auto-approval
        val result = scepClient.enroll(testTemplate)

        // Verify result structure
        assertNotNull("Private key should not be null", result.privateKey)
        assertTrue("Certificate chain should not be empty", result.certificateChain.isNotEmpty())

        // Verify private key
        assertEquals("Private key algorithm should be RSA", "RSA", result.privateKey.algorithm)
        assertNotNull("Private key encoded form should not be null", result.privateKey.encoded)

        // Verify certificate
        val leafCert = result.certificateChain[0] as java.security.cert.X509Certificate
        assertEquals("Certificate type should be X.509", "X.509", leafCert.type)
        assertNotNull("Certificate subject should not be null", leafCert.subjectX500Principal)

        println("Successfully enrolled certificate: ${leafCert.subjectX500Principal.name}")
    }

    @IntegrationTest
    @Test
    fun `enrollment with invalid challenge fails`() = runTest {
        val invalidTemplate = testTemplate.copy(scepChallenge = "invalid-challenge-that-should-fail")

        try {
            scepClient.enroll(invalidTemplate)
            fail("Expected ScepEnrollmentException for invalid challenge")
        } catch (e: ScepEnrollmentException) {
            // Expected - enrollment should fail with invalid challenge
            println("Correctly failed with: ${e.message}")
        }
    }

    @IntegrationTest
    @Test
    fun `enrollment with different key sizes`() = runTest {
        val keySizes = listOf(2048, 3072, 4096)

        keySizes.forEach { keySize ->
            val uniqueId = System.currentTimeMillis()
            val template = createTemplate(
                url = testTemplate.url ?: "https://scep.example.com/scep",
                challenge = testTemplate.scepChallenge ?: "test-challenge",
                name = "test-cert-$keySize-$uniqueId",
                subject = "CN=IntegrationTestDevice-$keySize-$uniqueId,O=FleetDM,C=US",
                keyLength = keySize,
            )

            val result = scepClient.enroll(template)

            assertNotNull(result.privateKey)
            println("Successfully enrolled with key size: $keySize")
        }
    }

    @IntegrationTest
    @Test
    fun `enrollment performance test`() = runTest {
        val startTime = System.currentTimeMillis()

        val result = scepClient.enroll(testTemplate)

        val duration = System.currentTimeMillis() - startTime

        assertNotNull(result)
        println("Enrollment completed in ${duration}ms")

        // Typical SCEP enrollment should complete within 30 seconds
        assertTrue("Enrollment should complete within 30 seconds", duration < 30000)
    }

    @Test
    fun `enrollment with unreachable server fails quickly`() = runTest {
        val unreachableTemplate = testTemplate.copy(
            url = "https://unreachable-scep-server.invalid/scep",
        )

        val startTime = System.currentTimeMillis()

        try {
            scepClient.enroll(unreachableTemplate)
            fail("Expected ScepNetworkException")
        } catch (e: ScepNetworkException) {
            val duration = System.currentTimeMillis() - startTime
            println("Failed as expected in ${duration}ms: ${e.message}")

            // Should fail within reasonable timeout
            assertTrue("Should fail within 30 seconds", duration < 30000)
        }
    }
}
