package com.fleetdm.agent.scep

import org.bouncycastle.asn1.ASN1ObjectIdentifier
import org.bouncycastle.asn1.ASN1Sequence
import org.bouncycastle.asn1.ASN1TaggedObject
import org.bouncycastle.asn1.DERIA5String
import org.bouncycastle.asn1.DEROctetString
import org.bouncycastle.asn1.DERUTF8String
import org.bouncycastle.asn1.x509.GeneralName
import org.bouncycastle.asn1.x509.GeneralNames
import org.junit.After
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test
import java.util.Locale

class SubjectAlternativeNameParserTest {

    private val originalLocale = Locale.getDefault()

    @After
    fun restoreLocale() {
        Locale.setDefault(originalLocale)
    }

    // -- helpers ---------------------------------------------------------

    private fun parseOrFail(input: String): GeneralNames {
        val result = SubjectAlternativeNameParser.parse(input)
        assertNotNull("Expected non-null GeneralNames for input: \"$input\"", result)
        return result!!
    }

    private fun parseSingle(input: String): GeneralName {
        val names = parseOrFail(input).names
        assertEquals("Expected exactly one entry for input: \"$input\"", 1, names.size)
        return names[0]
    }

    private fun assertRejects(input: String, expectedSubstring: String) {
        try {
            SubjectAlternativeNameParser.parse(input)
            fail("Expected IllegalArgumentException for input: \"$input\"")
        } catch (e: IllegalArgumentException) {
            val msg = e.message ?: ""
            assertTrue(
                "Expected message to contain \"$expectedSubstring\" for input \"$input\", got: $msg",
                msg.contains(expectedSubstring),
            )
        }
    }

    private fun upnUtf8Value(name: GeneralName): String {
        assertEquals(GeneralName.otherName, name.tagNo)
        val seq = name.name as ASN1Sequence
        val tagged = seq.getObjectAt(1) as ASN1TaggedObject
        // getExplicitBaseObject throws if the tag is implicit, so the multi-entry tests
        // that call this helper would catch a regression that emits implicit-tagged UPN
        // OtherName values without needing a separate isExplicit assertion per call.
        return (tagged.getExplicitBaseObject() as DERUTF8String).string
    }

    // -- null and blank inputs -------------------------------------------

    @Test
    fun `null and blank inputs return null`() {
        listOf(null, "", " ", "   \t\n  ").forEach { input ->
            assertNull(
                "Expected null for input \"$input\"",
                SubjectAlternativeNameParser.parse(input),
            )
        }
    }

    @Test
    fun `non-blank input that contains only empty tokens returns null`() {
        // Distinct from blank input: this string is non-blank, but every comma-separated
        // token is empty after trim, so no GeneralNames entries are produced.
        assertNull(SubjectAlternativeNameParser.parse(", , , "))
    }

    // -- per-KEY positive encoding ---------------------------------------

    @Test
    fun `each IA5String KEY encodes its value verbatim with the right tag`() {
        data class Case(val input: String, val expectedTag: Int, val expectedValue: String)
        listOf(
            Case("DNS=example.com", GeneralName.dNSName, "example.com"),
            Case("EMAIL=user@example.com", GeneralName.rfc822Name, "user@example.com"),
            Case(
                "URI=spiffe://example.org/workload",
                GeneralName.uniformResourceIdentifier,
                "spiffe://example.org/workload",
            ),
        ).forEach { case ->
            val name = parseSingle(case.input)
            assertEquals("Wrong tag for \"${case.input}\"", case.expectedTag, name.tagNo)
            assertEquals(
                "Wrong value for \"${case.input}\"",
                case.expectedValue,
                (name.name as DERIA5String).string,
            )
        }
    }

    @Test
    fun `URI value with percent-encoded comma passes through verbatim`() {
        val name = parseSingle("URI=https://example.com/a%2Cb?x=1")
        assertEquals(GeneralName.uniformResourceIdentifier, name.tagNo)
        assertEquals("https://example.com/a%2Cb?x=1", (name.name as DERIA5String).string)
    }

    @Test
    fun `IP addresses encode to canonical raw octets`() {
        // Two cases pin the BC integration: we route IP through BouncyCastle and BC
        // produces the RFC 5280 §4.2.1.6 raw octet form (4 bytes for IPv4, 16 for IPv6).
        // Exhaustive enumeration of IPv6 forms is BouncyCastle's responsibility, not ours.
        data class Case(val input: String, val expected: ByteArray)
        listOf(
            Case("192.168.1.100", byteArrayOf(192.toByte(), 168.toByte(), 1, 100)),
            Case(
                "2001:db8::1",
                byteArrayOf(
                    0x20, 0x01, 0x0d, 0xb8.toByte(),
                    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01,
                ),
            ),
        ).forEach { case ->
            val name = parseSingle("IP=${case.input}")
            assertEquals("Wrong tag for \"${case.input}\"", GeneralName.iPAddress, name.tagNo)
            assertArrayEquals(
                "Wrong octets for \"${case.input}\"",
                case.expected,
                (name.name as DEROctetString).octets,
            )
        }
    }

    @Test
    fun `UPN encodes as Microsoft otherName with EXPLICIT-tagged UTF8 value`() {
        val upn = "marko@corp.example.com"
        val name = parseSingle("UPN=$upn")
        assertEquals(GeneralName.otherName, name.tagNo)

        // OtherName ::= SEQUENCE { type-id OID, value [0] EXPLICIT ANY DEFINED BY type-id }
        val otherName = name.name as ASN1Sequence
        assertEquals(2, otherName.size())
        assertEquals(
            "1.3.6.1.4.1.311.20.2.3",
            (otherName.getObjectAt(0) as ASN1ObjectIdentifier).id,
        )
        val tagged = otherName.getObjectAt(1) as ASN1TaggedObject
        assertEquals(0, tagged.tagNo)
        assertTrue("UPN value must be [0] EXPLICIT", tagged.isExplicit)
        assertEquals(upn, (tagged.baseObject as DERUTF8String).string)
    }

    // -- multi-entry positive tests --------------------------------------

    @Test
    fun `mixed entries preserve type, value, and document order`() {
        val san = "DNS=host.example.com, EMAIL=u@example.com, URI=spiffe://x/y, " +
            "IP=10.0.0.1, UPN=marko@corp.example.com"
        val names = parseOrFail(san).names
        assertEquals(5, names.size)

        assertEquals(GeneralName.dNSName, names[0].tagNo)
        assertEquals("host.example.com", (names[0].name as DERIA5String).string)
        assertEquals(GeneralName.rfc822Name, names[1].tagNo)
        assertEquals("u@example.com", (names[1].name as DERIA5String).string)
        assertEquals(GeneralName.uniformResourceIdentifier, names[2].tagNo)
        assertEquals("spiffe://x/y", (names[2].name as DERIA5String).string)
        assertEquals(GeneralName.iPAddress, names[3].tagNo)
        assertArrayEquals(byteArrayOf(10, 0, 0, 1), (names[3].name as DEROctetString).octets)
        assertEquals("marko@corp.example.com", upnUtf8Value(names[4]))
    }

    @Test
    fun `repeated keys produce repeated entries in document order with distinct values`() {
        val san = "DNS=a.example.com, DNS=b.example.com, EMAIL=u1@x, EMAIL=u2@x, " +
            "UPN=u1@corp, UPN=u2@corp, IP=10.0.0.1, IP=10.0.0.2, " +
            "URI=spiffe://x/1, URI=spiffe://x/2"
        val names = parseOrFail(san).names
        assertEquals(10, names.size)

        assertEquals("a.example.com", (names[0].name as DERIA5String).string)
        assertEquals("b.example.com", (names[1].name as DERIA5String).string)
        assertEquals("u1@x", (names[2].name as DERIA5String).string)
        assertEquals("u2@x", (names[3].name as DERIA5String).string)
        assertEquals("u1@corp", upnUtf8Value(names[4]))
        assertEquals("u2@corp", upnUtf8Value(names[5]))
        assertArrayEquals(byteArrayOf(10, 0, 0, 1), (names[6].name as DEROctetString).octets)
        assertArrayEquals(byteArrayOf(10, 0, 0, 2), (names[7].name as DEROctetString).octets)
        assertEquals("spiffe://x/1", (names[8].name as DERIA5String).string)
        assertEquals("spiffe://x/2", (names[9].name as DERIA5String).string)
    }

    // -- behavioral / format tolerance -----------------------------------

    @Test
    fun `KEY matching is case-insensitive`() {
        val names = parseOrFail("dns=example.com, Email=u@x, uPn=marko@corp").names
        assertEquals(3, names.size)
        assertEquals(GeneralName.dNSName, names[0].tagNo)
        assertEquals(GeneralName.rfc822Name, names[1].tagNo)
        assertEquals(GeneralName.otherName, names[2].tagNo)
    }

    @Test
    fun `KEY matching is locale-insensitive (Turkish dotless-i regression)`() {
        // Turkish uppercase rules turn "i" into "İ" under Locale.getDefault();
        // the parser must use Locale.ROOT so "ip" still maps to "IP".
        Locale.setDefault(Locale.forLanguageTag("tr-TR"))
        val names = parseOrFail("ip=10.0.0.1, dns=host.example.com").names
        assertEquals(2, names.size)
        assertEquals(GeneralName.iPAddress, names[0].tagNo)
        assertEquals(GeneralName.dNSName, names[1].tagNo)
    }

    @Test
    fun `whitespace around tokens and around equals is tolerated`() {
        val names = parseOrFail("  DNS  =  example.com  ,   EMAIL =u@x  ").names
        assertEquals(2, names.size)
        assertEquals(GeneralName.dNSName, names[0].tagNo)
        assertEquals("example.com", (names[0].name as DERIA5String).string)
        assertEquals(GeneralName.rfc822Name, names[1].tagNo)
        assertEquals("u@x", (names[1].name as DERIA5String).string)
    }

    @Test
    fun `trailing and embedded empty tokens are skipped`() {
        val names = parseOrFail("DNS=example.com, , ,").names
        assertEquals(1, names.size)
        assertEquals("example.com", (names[0].name as DERIA5String).string)
    }

    // -- rejection cases (one table; one assertion shape) ----------------

    @Test
    fun `parser rejects malformed inputs`() {
        // Exhaustively testing what BouncyCastle rejects belongs in BC's own tests; we
        // keep one representative bad IPv4 and one bad IPv6 to confirm we route IP
        // through IPAddress.isValid and surface the right error message.
        listOf(
            // KEY allow-list violations.
            "FOO=bar" to "FOO",
            "RFC822=user@example.com" to "RFC822",
            // Token shape violations.
            "DNS=ok, OOPS" to "OOPS",
            "DNS=" to "DNS=",
            "=value" to "=value",
            // IP value violations: representative bad IPv4 and bad IPv6.
            "IP=999.0.0.1" to "999",
            "IP=fe80::1%eth0" to "fe80::1%eth0",
        ).forEach { (input, substring) -> assertRejects(input, substring) }
    }
}
