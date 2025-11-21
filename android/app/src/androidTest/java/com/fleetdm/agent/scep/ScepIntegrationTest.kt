package com.fleetdm.agent.scep

import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import kotlinx.coroutines.test.runTest
import org.junit.Assert.*
import org.junit.Before
import org.junit.Ignore
import org.junit.Test
import org.junit.runner.RunWith

/**
 * Integration tests for ScepClientImpl with a real SCEP server.
 *
 * These tests are IGNORED by default and should only be run when:
 * 1. A test SCEP server is available
 * 2. Test credentials are configured
 * 3. Network connectivity is available
 *
 * To run these tests:
 * ./gradlew connectedAndroidTest \
 *   -Pandroid.testInstrumentationRunnerArguments.scep.url=https://your-scep-server.com/scep \
 *   -Pandroid.testInstrumentationRunnerArguments.scep.challenge=your-challenge
 */
@RunWith(AndroidJUnit4::class)
class ScepIntegrationTest {

    private lateinit var scepClient: ScepClientImpl
    private lateinit var testConfig: ScepConfig

    @Before
    fun setup() {
        scepClient = ScepClientImpl()

        val arguments = InstrumentationRegistry.getArguments()
        val scepUrl = arguments.getString("scep.url") ?: "https://tim-fleet-2.ngrok.app/mdm/scep/proxy/foo,g-profile"
        val challenge = arguments.getString("scep.challenge") ?: "secret"

        testConfig = ScepConfig(
            url = scepUrl,
            challenge = challenge,
            alias = "integration-test-cert-${System.currentTimeMillis()}",
            subject = "CN=IntegrationTestDevice,O=FleetDM,C=US"
        )
    }

    @Test
    @Ignore("Requires real SCEP server - enable manually for integration testing")
    fun `successful enrollment with real SCEP server`() = runTest {
        // This test requires a real SCEP server with auto-approval
        val result = scepClient.enroll(testConfig)

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

    @Test
    @Ignore("Requires real SCEP server - enable manually for integration testing")
    fun `enrollment with invalid challenge fails`() = runTest {
        val invalidConfig = testConfig.copy(challenge = "invalid-challenge-that-should-fail")

        try {
            scepClient.enroll(invalidConfig)
            fail("Expected ScepEnrollmentException for invalid challenge")
        } catch (e: ScepEnrollmentException) {
            // Expected - enrollment should fail with invalid challenge
            println("Correctly failed with: ${e.message}")
        }
    }

    @Test
    @Ignore("Requires real SCEP server - enable manually for integration testing")
    fun `enrollment with different key sizes`() = runTest {
        val keySizes = listOf(2048, 3072, 4096)

        keySizes.forEach { keySize ->
            val config = testConfig.copy(
                keyLength = keySize,
                alias = "test-cert-${keySize}-${System.currentTimeMillis()}"
            )

            val result = scepClient.enroll(config)

            assertNotNull(result.privateKey)
            println("Successfully enrolled with key size: $keySize")
        }
    }

    @Test
    @Ignore("Requires real SCEP server - enable manually for integration testing")
    fun `enrollment performance test`() = runTest {
        val startTime = System.currentTimeMillis()

        val result = scepClient.enroll(testConfig)

        val duration = System.currentTimeMillis() - startTime

        assertNotNull(result)
        println("Enrollment completed in ${duration}ms")

        // Typical SCEP enrollment should complete within 30 seconds
        assertTrue("Enrollment should complete within 30 seconds", duration < 30000)
    }

    @Test
    fun `enrollment with unreachable server fails quickly`() = runTest {
        val unreachableConfig = testConfig.copy(
            url = "https://unreachable-scep-server.invalid/scep"
        )

        val startTime = System.currentTimeMillis()

        try {
            scepClient.enroll(unreachableConfig)
            fail("Expected ScepNetworkException")
        } catch (e: ScepNetworkException) {
            val duration = System.currentTimeMillis() - startTime
            println("Failed as expected in ${duration}ms: ${e.message}")

            // Should fail within reasonable timeout
            assertTrue("Should fail within 30 seconds", duration < 30000)
        }
    }
}
