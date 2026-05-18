package com.fleetdm.agent.scep

import com.fleetdm.agent.testutil.TestCertificateTemplateFactory
import org.bouncycastle.asn1.DERIA5String
import org.bouncycastle.asn1.pkcs.PKCSObjectIdentifiers
import org.bouncycastle.asn1.x500.X500Name
import org.bouncycastle.asn1.x509.Extension
import org.bouncycastle.asn1.x509.Extensions
import org.bouncycastle.asn1.x509.GeneralName
import org.bouncycastle.asn1.x509.GeneralNames
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Before
import org.junit.Test
import java.security.KeyPairGenerator
import kotlinx.coroutines.test.runTest

/**
 * Unit tests for ScepClientImpl.
 *
 * Note: These tests validate the structure and error handling.
 * Full integration testing with a real SCEP server should be done separately.
 */
class ScepClientImplTest {

    private lateinit var scepClient: ScepClientImpl

    @Before
    fun setup() {
        scepClient = ScepClientImpl()
    }

    @Test
    fun `enroll with malformed URL throws ScepNetworkException`() = runTest {
        val template = TestCertificateTemplateFactory.create()
        val malformedUrl = "http://[invalid"

        try {
            scepClient.enroll(template, malformedUrl)
            fail("Expected ScepNetworkException to be thrown")
        } catch (e: ScepNetworkException) {
            assertTrue(e.message?.contains("Invalid SCEP URL") == true)
        }
    }

    @Test
    fun `enroll with invalid subject throws ScepCsrException`() = runTest {
        val template = TestCertificateTemplateFactory.create(subjectName = "invalid-subject-format")

        try {
            scepClient.enroll(template, TestCertificateTemplateFactory.DEFAULT_SCEP_URL)
            fail("Expected ScepCsrException to be thrown")
        } catch (e: ScepCsrException) {
            assertTrue(e.message?.contains("Invalid X.500 subject name") == true)
        }
    }

    @Test
    fun `enroll with unreachable server throws ScepNetworkException`() = runTest {
        val template = TestCertificateTemplateFactory.create()
        val unreachableUrl = "https://invalid-scep-server-that-does-not-exist.example.com/scep"

        try {
            scepClient.enroll(template, unreachableUrl)
            fail("Expected ScepNetworkException to be thrown")
        } catch (e: ScepNetworkException) {
            assertTrue(e.message?.contains("Failed to communicate") == true)
        }
    }

    @Test
    fun `buildCsr omits SAN extension when SAN string is null or blank`() {
        listOf(null, "", "   ").forEach { input ->
            val csr = buildTestCsr(subjectAlternativeName = input)
            assertNull(
                "Expected no SAN extension for input \"$input\"",
                extractSanExtension(csr),
            )
        }
    }

    @Test
    fun `buildCsr emits a non-critical SAN extension when SAN string is present`() {
        // Per-KEY encoding correctness is covered by SubjectAlternativeNameParserTest.
        val csr = buildTestCsr(subjectAlternativeName = "DNS=example.com")
        val ext = extractSanExtension(csr)
        assertNotNull(ext)
        assertFalse("SAN extension must be non-critical", ext!!.isCritical)
        val names = GeneralNames.getInstance(ext.parsedValue).names
        assertEquals(1, names.size)
        assertEquals(GeneralName.dNSName, names[0].tagNo)
        assertEquals("example.com", (names[0].name as DERIA5String).string)
    }

    @Test
    fun `buildCsr wraps parser exceptions as ScepCsrException`() {
        try {
            buildTestCsr(subjectAlternativeName = "FOO=bar")
            fail("Expected ScepCsrException")
        } catch (e: ScepCsrException) {
            assertTrue(
                "Expected wrapper message to mention SAN; got: ${e.message}",
                e.message!!.contains("subject alternative name"),
            )
        }
    }

    private fun buildTestCsr(subjectAlternativeName: String?) = scepClient.buildCsr(
        entity = X500Name("CN=Test,O=FleetDM"),
        keyPair = KeyPairGenerator.getInstance("RSA").apply { initialize(2048) }.genKeyPair(),
        challenge = "test-challenge",
        signatureAlgorithm = "SHA256withRSA",
        subjectAlternativeName = subjectAlternativeName,
    )

    /**
     * Pulls the subjectAltName Extension out of a PKCS#10 CSR's extensionRequest attribute.
     * Returns null if there is no extensionRequest attribute or no SAN extension inside it.
     */
    private fun extractSanExtension(csr: org.bouncycastle.pkcs.PKCS10CertificationRequest): Extension? {
        val attributes = csr.getAttributes(PKCSObjectIdentifiers.pkcs_9_at_extensionRequest)
        if (attributes.isEmpty()) return null
        val extensions = Extensions.getInstance(attributes[0].attrValues.getObjectAt(0))
        return extensions.getExtension(Extension.subjectAlternativeName)
    }
}
