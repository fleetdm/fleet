package com.fleetdm.agent

import com.fleetdm.agent.testutil.TestCertificateTemplateFactory
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * The SCEP challenge and the Fleet challenge are enrollment secrets, and anyone who can plug the
 * device into a computer can read logcat. These tests pin down that neither reaches a log line.
 */
class LogRedactionTest {

    private val proxyUrl = "https://fleet.example.com/mdm/scep/proxy/host-uuid-1234,g7,NDES,s3cr3t-challenge"

    @Test
    fun `certificate template toString hides both challenges`() {
        val template = TestCertificateTemplateFactory.create(
            scepChallenge = "scep-s3cr3t",
            fleetChallenge = "fleet-s3cr3t",
        )

        val logged = template.toString()

        assertFalse(logged.contains("scep-s3cr3t"))
        assertFalse(logged.contains("fleet-s3cr3t"))
        assertTrue(logged.contains("scepChallenge=$REDACTED"))
        assertTrue(logged.contains("fleetChallenge=$REDACTED"))
    }

    @Test
    fun `certificate template toString keeps troubleshooting fields`() {
        val template = TestCertificateTemplateFactory.create(id = 42, name = "wifi-cert", status = "delivered")

        val logged = template.toString()

        assertTrue(logged.contains("id=42"))
        assertTrue(logged.contains("name=wifi-cert"))
        assertTrue(logged.contains("status=delivered"))
        assertTrue(logged.contains("subjectName=CN=Test,O=FleetDM"))
    }

    @Test
    fun `certificate template toString distinguishes missing from set challenges`() {
        val template = GetCertificateTemplateResponse(
            id = 1,
            name = "test-cert",
            certificateAuthorityId = 1,
            certificateAuthorityName = "Test CA",
            createdAt = "2024-01-01T00:00:00Z",
            subjectName = "CN=Test",
            certificateAuthorityType = "SCEP",
            status = "delivered",
            scepChallenge = null,
            fleetChallenge = "",
        )

        val logged = template.toString()

        assertTrue(logged.contains("scepChallenge=null"))
        assertTrue(logged.contains("fleetChallenge=\"\""))
        assertFalse(logged.contains(REDACTED))
    }

    @Test
    fun `certificate template result toString hides the url challenge`() {
        val template = TestCertificateTemplateFactory.create(fleetChallenge = "fleet-s3cr3t")
        val result = CertificateTemplateResult(
            template = template,
            scepUrl = template.buildScepUrl(serverUrl = "https://fleet.example.com", hostUUID = "host-uuid-1234"),
        )

        val logged = result.toString()

        assertFalse(logged.contains("fleet-s3cr3t"))
        assertTrue(logged.contains("host-uuid-1234,g1,SCEP,$REDACTED"))
    }

    @Test
    fun `redactSecrets hides the challenge in a url built by buildScepUrl`() {
        val template = TestCertificateTemplateFactory.create(fleetChallenge = "fleet-s3cr3t")

        val url = template.buildScepUrl(serverUrl = "https://fleet.example.com", hostUUID = "host-uuid-1234")
        val redacted = url.redactSecrets()

        assertTrue(url.contains("fleet-s3cr3t"))
        assertFalse(redacted.contains("fleet-s3cr3t"))
        assertTrue(redacted.endsWith(REDACTED))
    }

    @Test
    fun `redactSecrets hides the challenge in a scep proxy url`() {
        assertEquals(
            "https://fleet.example.com/mdm/scep/proxy/host-uuid-1234,g7,NDES,$REDACTED",
            proxyUrl.redactSecrets(),
        )
    }

    @Test
    fun `redactSecrets keeps the query string appended by jscep`() {
        val withQuery = "$proxyUrl?operation=PKIOperation&message=abc123"

        assertEquals(
            "https://fleet.example.com/mdm/scep/proxy/host-uuid-1234,g7,NDES,$REDACTED?operation=PKIOperation&message=abc123",
            withQuery.redactSecrets(),
        )
    }

    @Test
    fun `redactSecrets hides a percent encoded challenge`() {
        val encoded = "https://fleet.example.com/mdm/scep/proxy/host-uuid-1234%2Cg7%2CNDES%2Cs3cr3t-challenge"

        assertEquals(
            "https://fleet.example.com/mdm/scep/proxy/host-uuid-1234%2Cg7%2CNDES%2C$REDACTED",
            encoded.redactSecrets(),
        )
    }

    @Test
    fun `redactSecrets keeps a closing delimiter after the url`() {
        val quoted = "{\"url\":\"$proxyUrl\"}"

        val redacted = quoted.redactSecrets()

        assertFalse(redacted.contains("s3cr3t-challenge"))
        assertEquals("{\"url\":\"https://fleet.example.com/mdm/scep/proxy/host-uuid-1234,g7,NDES,$REDACTED\"}", redacted)
    }

    @Test
    fun `redactSecrets hides the challenge inside a wrapped exception message`() {
        val cause = java.io.FileNotFoundException(proxyUrl)
        val wrapped = Exception("Error connecting to server", cause)

        val rendered = wrapped.stackTraceToString().redactSecrets()

        assertFalse(rendered.contains("s3cr3t-challenge"))
        assertTrue(rendered.contains("host-uuid-1234,g7,NDES,$REDACTED"))
    }

    @Test
    fun `redactSecrets leaves urls without a challenge alone`() {
        val noChallenge = "https://fleet.example.com/mdm/scep/proxy/host-uuid-1234,g7,NDES"
        val emptyChallenge = "https://fleet.example.com/mdm/scep/proxy/host-uuid-1234,g7,NDES,"

        assertEquals(noChallenge, noChallenge.redactSecrets())
        assertEquals(emptyChallenge, emptyChallenge.redactSecrets())
    }

    @Test
    fun `redactSecrets leaves unrelated text alone`() {
        val unrelated = "DNS resolution failed for GET /api/fleetd/certificates/7: fleet.example.com"
        val malformed = "Invalid SCEP URL: http://[invalid"

        assertEquals(unrelated, unrelated.redactSecrets())
        assertEquals(malformed, malformed.redactSecrets())
    }

    @Test
    fun `redactSecrets redacts every proxy url in the text`() {
        val twoUrls = "first $proxyUrl then https://fleet.example.com/mdm/scep/proxy/other-uuid,g8,DIGICERT,other-secret"

        val redacted = twoUrls.redactSecrets()

        assertFalse(redacted.contains("s3cr3t-challenge"))
        assertFalse(redacted.contains("other-secret"))
        assertTrue(redacted.contains("other-uuid,g8,DIGICERT,$REDACTED"))
    }

    @Test
    fun `redactSecrets hides challenges quoted from a certificate template body`() {
        val body = "{\"certificate\":{\"id\":7,\"name\":\"wifi-cert\"," +
            "\"scep_challenge\":\"scep-s3cr3t\",\"fleet_challenge\": \"fleet-s3cr3t\",\"status\":\"delivered\"}}"
        val decodingError = "Unexpected JSON token at offset 210: Expected quotation mark\nJSON input: $body"

        val redacted = decodingError.redactSecrets()

        assertFalse(redacted.contains("scep-s3cr3t"))
        assertFalse(redacted.contains("fleet-s3cr3t"))
        assertTrue(redacted.contains("\"scep_challenge\":\"$REDACTED\""))
        assertTrue(redacted.contains("\"fleet_challenge\":\"$REDACTED\""))
        assertTrue(redacted.contains("\"name\":\"wifi-cert\""))
    }

    @Test
    fun `redactSecrets hides a challenge cut off mid value`() {
        val truncated = "JSON input: {\"certificate\":{\"name\":\"wifi-cert\",\"fleet_challenge\":\"fleet-s3cr"

        val redacted = truncated.redactSecrets()

        assertFalse(redacted.contains("fleet-s3cr"))
        assertTrue(redacted.endsWith("\"fleet_challenge\":\"$REDACTED\""))
    }

    @Test
    fun `redactSecrets hides a challenge containing an escaped quote`() {
        // The JSON value is pass\"word-s3cr3t: a quote escaped inside the challenge itself.
        val body = """JSON input: {"certificate":{"scep_challenge":"pass\"word-s3cr3t","status":"delivered"}}"""

        val redacted = body.redactSecrets()

        assertFalse(redacted.contains("word-s3cr3t"))
        assertTrue(redacted.contains("\"scep_challenge\":\"$REDACTED\""))
        assertTrue(redacted.contains("\"status\":\"delivered\""))
    }

    @Test
    fun `redactSecrets hides a challenge cut off on an escape`() {
        val truncated = "JSON input: {\"certificate\":{\"fleet_challenge\":\"fleet-s3cr\\"

        val redacted = truncated.redactSecrets()

        assertFalse(redacted.contains("fleet-s3cr"))
        assertTrue(redacted.endsWith("\"fleet_challenge\":\"$REDACTED\""))
    }

    @Test
    fun `renderLogEntry redacts the message and the whole cause chain`() {
        val cause = java.io.FileNotFoundException(proxyUrl)
        val throwable = Exception("Error connecting to server", cause)

        val (message, stackTrace) = renderLogEntry("Certificate enrollment failed for $proxyUrl", throwable)

        assertFalse(message.contains("s3cr3t-challenge"))
        assertTrue(message.endsWith("host-uuid-1234,g7,NDES,$REDACTED"))
        assertFalse(stackTrace!!.contains("s3cr3t-challenge"))
        assertTrue(stackTrace.contains("Caused by"))
        assertTrue(stackTrace.contains("host-uuid-1234,g7,NDES,$REDACTED"))
    }

    @Test
    fun `renderLogEntry has no stack trace without a throwable`() {
        val (message, stackTrace) = renderLogEntry("plain message", null)

        assertEquals("plain message", message)
        assertNull(stackTrace)
    }
}
