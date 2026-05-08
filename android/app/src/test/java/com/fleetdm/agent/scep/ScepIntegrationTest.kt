package com.fleetdm.agent.scep

import com.fleetdm.agent.GetCertificateTemplateResponse
import com.fleetdm.agent.IntegrationTest
import com.fleetdm.agent.IntegrationTestRule
import com.fleetdm.agent.testutil.TestCertificateTemplateFactory
import org.bouncycastle.asn1.ASN1OctetString
import org.bouncycastle.asn1.DERIA5String
import org.bouncycastle.asn1.DEROctetString
import org.bouncycastle.asn1.x509.Extension
import org.bouncycastle.asn1.x509.GeneralName
import org.bouncycastle.asn1.x509.GeneralNames
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
    private lateinit var testScepUrl: String

    @Before
    fun setup() {
        scepClient = ScepClientImpl()

        // Use placeholder values for non-integration tests, real values provided by build config for integration tests
        testScepUrl = System.getProperty("scep.url") ?: "https://scep.example.com/scep"
        val challenge = System.getProperty("scep.challenge") ?: "test-challenge"

        // Generate unique subject DN to avoid duplicates on SCEP server
        val uniqueId = System.currentTimeMillis()
        testTemplate = TestCertificateTemplateFactory.create(
            scepChallenge = challenge,
            name = "integration-test-cert-$uniqueId",
            subjectName = "CN=IntegrationTestDevice-$uniqueId,O=FleetDM,C=US",
        )
    }

    @IntegrationTest
    @Test
    fun `successful enrollment with real SCEP server`() = runTest {
        // This test requires a real SCEP server with auto-approval
        val result = scepClient.enroll(testTemplate, testScepUrl)

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
    }

    @IntegrationTest
    @Test
    fun `enrollment with invalid challenge fails`() = runTest {
        val invalidTemplate = testTemplate.copy(scepChallenge = "invalid-challenge-that-should-fail")

        try {
            scepClient.enroll(invalidTemplate, testScepUrl)
            fail("Expected ScepEnrollmentException for invalid challenge")
        } catch (e: ScepEnrollmentException) {
            // Expected - enrollment should fail with invalid challenge
            assertNotNull("Exception should have a message", e.message)
        }
    }

    @IntegrationTest
    @Test
    fun `enrollment with different key sizes`() = runTest {
        val keySizes = listOf(2048, 3072, 4096)

        keySizes.forEach { keySize ->
            val uniqueId = System.currentTimeMillis()
            val template = TestCertificateTemplateFactory.create(
                scepChallenge = testTemplate.scepChallenge ?: "test-challenge",
                name = "test-cert-$keySize-$uniqueId",
                subjectName = "CN=IntegrationTestDevice-$keySize-$uniqueId,O=FleetDM,C=US",
                keyLength = keySize,
            )

            val result = scepClient.enroll(template, testScepUrl)

            assertNotNull("Private key should not be null for key size $keySize", result.privateKey)
        }
    }

    @IntegrationTest
    @Test
    fun `enrollment performance test`() = runTest {
        val startTime = System.currentTimeMillis()

        val result = scepClient.enroll(testTemplate, testScepUrl)

        val duration = System.currentTimeMillis() - startTime

        assertNotNull(result)

        // Typical SCEP enrollment should complete within 30 seconds
        assertTrue("Enrollment should complete within 30 seconds (took ${duration}ms)", duration < 30000)
    }

    @IntegrationTest
    @Test
    fun `SAN entries on the certificate template appear on the issued certificate`() = runTest {
        // End-to-end check that the SAN extension we put on the CSR is accepted by the
        // SCEP CA and copied verbatim to the issued certificate. Two entries per type
        // exercise the repeated-key path and confirm the CA does not deduplicate.
        // Unique values per run avoid duplicate-cert collisions on CAs that enforce it.
        //
        // UPN (otherName) is intentionally not asserted here: the reference CA used in
        // CI (micromdm/scep) only copies DNS/Email/IP/URI from the CSR to the issued
        // cert via the typed fields on Go's x509.Certificate, and drops otherName.
        val uniqueId = System.currentTimeMillis()
        val expectedDns = listOf("a-$uniqueId.example.com", "b-$uniqueId.example.com")
        val expectedEmail = listOf("a-$uniqueId@example.com", "b-$uniqueId@example.com")
        val expectedUri = listOf(
            "spiffe://example.org/a-$uniqueId",
            "spiffe://example.org/b-$uniqueId",
        )
        val expectedIpStrings = listOf("10.0.0.1", "10.0.0.2")
        val expectedIpOctets = listOf(byteArrayOf(10, 0, 0, 1), byteArrayOf(10, 0, 0, 2))

        val sanString = listOf(
            "DNS=${expectedDns[0]}",
            "DNS=${expectedDns[1]}",
            "EMAIL=${expectedEmail[0]}",
            "EMAIL=${expectedEmail[1]}",
            "URI=${expectedUri[0]}",
            "URI=${expectedUri[1]}",
            "IP=${expectedIpStrings[0]}",
            "IP=${expectedIpStrings[1]}",
        ).joinToString(", ")

        val template = testTemplate.copy(
            name = "san-test-cert-$uniqueId",
            subjectName = "CN=SanIntegrationTest-$uniqueId,O=FleetDM,C=US",
            subjectAlternativeName = sanString,
        )

        val result = scepClient.enroll(template, testScepUrl)
        val leafCert = result.certificateChain[0] as java.security.cert.X509Certificate

        // Pull the SAN extension off the issued cert and parse via BouncyCastle. The
        // cert's getExtensionValue returns the OCTET STRING wrapper, not the SAN
        // contents directly, so we unwrap once before handing to GeneralNames.
        val sanBytes = leafCert.getExtensionValue(Extension.subjectAlternativeName.id)
        assertNotNull("Issued certificate has no SAN extension", sanBytes)
        val sanContents = ASN1OctetString.getInstance(sanBytes).octets
        val sanNames = GeneralNames.getInstance(sanContents).names

        fun ia5ValuesForTag(tag: Int): List<String> = sanNames
            .filter { it.tagNo == tag }
            .map { (it.name as DERIA5String).string }

        fun ipOctetsHex(): List<String> = sanNames
            .filter { it.tagNo == GeneralName.iPAddress }
            .map { (it.name as DEROctetString).octets.joinToString("") { b -> "%02x".format(b) } }

        // Order is not pinned: some CAs canonicalize SAN ordering during signing.
        // Sort each side and compare to verify both entries of each type round-trip.
        assertEquals(expectedDns.sorted(), ia5ValuesForTag(GeneralName.dNSName).sorted())
        assertEquals(expectedEmail.sorted(), ia5ValuesForTag(GeneralName.rfc822Name).sorted())
        assertEquals(
            expectedUri.sorted(),
            ia5ValuesForTag(GeneralName.uniformResourceIdentifier).sorted(),
        )
        assertEquals(
            expectedIpOctets.map { it.joinToString("") { b -> "%02x".format(b) } }.sorted(),
            ipOctetsHex().sorted(),
        )
    }

    @Test
    fun `enrollment with unreachable server fails quickly`() = runTest {
        val unreachableUrl = "https://unreachable-scep-server.invalid/scep"

        val startTime = System.currentTimeMillis()

        try {
            scepClient.enroll(testTemplate, unreachableUrl)
            fail("Expected ScepNetworkException")
        } catch (e: ScepNetworkException) {
            val duration = System.currentTimeMillis() - startTime

            // Should fail within reasonable timeout
            assertTrue("Should fail within 30 seconds (took ${duration}ms)", duration < 30000)
            assertNotNull("Exception should have a message", e.message)
        }
    }
}
