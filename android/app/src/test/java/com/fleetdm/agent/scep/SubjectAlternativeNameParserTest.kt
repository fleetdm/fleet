package com.fleetdm.agent.scep

import org.bouncycastle.asn1.ASN1Encodable
import org.bouncycastle.asn1.ASN1ObjectIdentifier
import org.bouncycastle.asn1.ASN1Sequence
import org.bouncycastle.asn1.ASN1TaggedObject
import org.bouncycastle.asn1.DERIA5String
import org.bouncycastle.asn1.DEROctetString
import org.bouncycastle.asn1.DERUTF8String
import org.bouncycastle.asn1.x509.GeneralName
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Assert.fail
import org.junit.Test

class SubjectAlternativeNameParserTest {

    @Test
    fun `null input returns null`() {
        assertNull(SubjectAlternativeNameParser.parse(null))
    }

    @Test
    fun `empty input returns null`() {
        assertNull(SubjectAlternativeNameParser.parse(""))
    }

    @Test
    fun `whitespace-only input returns null`() {
        assertNull(SubjectAlternativeNameParser.parse("   \t\n  "))
    }

    @Test
    fun `single DNS entry`() {
        val names = SubjectAlternativeNameParser.parse("DNS=example.com")?.names
        assertNotNull(names)
        assertEquals(1, names!!.size)
        assertEquals(GeneralName.dNSName, names[0].tagNo)
        assertEquals("example.com", (names[0].name as DERIA5String).string)
    }

    @Test
    fun `single EMAIL entry`() {
        val names = SubjectAlternativeNameParser.parse("EMAIL=user@example.com")?.names
        assertNotNull(names)
        assertEquals(1, names!!.size)
        assertEquals(GeneralName.rfc822Name, names[0].tagNo)
        assertEquals("user@example.com", (names[0].name as DERIA5String).string)
    }

    @Test
    fun `single URI entry`() {
        val uri = "spiffe://example.org/workload"
        val names = SubjectAlternativeNameParser.parse("URI=$uri")?.names
        assertNotNull(names)
        assertEquals(1, names!!.size)
        assertEquals(GeneralName.uniformResourceIdentifier, names[0].tagNo)
        assertEquals(uri, (names[0].name as DERIA5String).string)
    }

    @Test
    fun `single IP entry IPv4`() {
        val names = SubjectAlternativeNameParser.parse("IP=192.168.1.100")?.names
        assertNotNull(names)
        assertEquals(1, names!!.size)
        assertEquals(GeneralName.iPAddress, names[0].tagNo)
        val octets = (names[0].name as DEROctetString).octets
        assertArrayEquals(byteArrayOf(192.toByte(), 168.toByte(), 1.toByte(), 100.toByte()), octets)
    }

    @Test
    fun `single IP entry IPv6`() {
        val names = SubjectAlternativeNameParser.parse("IP=2001:db8::1")?.names
        assertNotNull(names)
        assertEquals(1, names!!.size)
        assertEquals(GeneralName.iPAddress, names[0].tagNo)
        val octets = (names[0].name as DEROctetString).octets
        assertEquals(16, octets.size)
        // 2001:db8::1 -> 20 01 0d b8 00 00 ... 00 01
        assertEquals(0x20.toByte(), octets[0])
        assertEquals(0x01.toByte(), octets[1])
        assertEquals(0x0d.toByte(), octets[2])
        assertEquals(0xb8.toByte(), octets[3])
        assertEquals(0x01.toByte(), octets[15])
    }

    @Test
    fun `single UPN entry encodes as Microsoft otherName with UTF8 value`() {
        val upn = "marko@corp.example.com"
        val names = SubjectAlternativeNameParser.parse("UPN=$upn")?.names
        assertNotNull(names)
        assertEquals(1, names!!.size)
        assertEquals(GeneralName.otherName, names[0].tagNo)

        // OtherName ::= SEQUENCE { type-id OID, value [0] EXPLICIT ANY DEFINED BY type-id }
        val otherName = names[0].name as ASN1Sequence
        assertEquals(2, otherName.size())
        val oid = otherName.getObjectAt(0) as ASN1ObjectIdentifier
        assertEquals("1.3.6.1.4.1.311.20.2.3", oid.id)

        val tagged = otherName.getObjectAt(1) as ASN1TaggedObject
        assertEquals(0, tagged.tagNo)
        assertTrue("UPN value must be [0] EXPLICIT", tagged.isExplicit)

        val utf8 = tagged.baseObject as DERUTF8String
        assertEquals(upn, utf8.string)
    }

    @Test
    fun `mixed entries cover all five KEYs in document order`() {
        val san = "DNS=host.example.com, EMAIL=u@example.com, URI=spiffe://x/y, IP=10.0.0.1, " +
            "UPN=marko@corp.example.com"
        val names = SubjectAlternativeNameParser.parse(san)?.names
        assertNotNull(names)
        assertEquals(5, names!!.size)
        assertEquals(GeneralName.dNSName, names[0].tagNo)
        assertEquals(GeneralName.rfc822Name, names[1].tagNo)
        assertEquals(GeneralName.uniformResourceIdentifier, names[2].tagNo)
        assertEquals(GeneralName.iPAddress, names[3].tagNo)
        assertEquals(GeneralName.otherName, names[4].tagNo)
    }

    @Test
    fun `repeated keys produce repeated entries in document order`() {
        val san = "DNS=a.example.com, DNS=b.example.com, EMAIL=u1@x, EMAIL=u2@x, " +
            "UPN=u1@corp, UPN=u2@corp, IP=10.0.0.1, IP=10.0.0.2, " +
            "URI=spiffe://x/1, URI=spiffe://x/2"
        val names = SubjectAlternativeNameParser.parse(san)?.names
        assertNotNull(names)
        assertEquals(10, names!!.size)
        assertEquals("a.example.com", (names[0].name as DERIA5String).string)
        assertEquals("b.example.com", (names[1].name as DERIA5String).string)
        assertEquals("u1@x", (names[2].name as DERIA5String).string)
        assertEquals("u2@x", (names[3].name as DERIA5String).string)
        assertUpnValue(names[4].name, "u1@corp")
        assertUpnValue(names[5].name, "u2@corp")
        assertEquals(4, (names[6].name as DEROctetString).octets.size)
        assertEquals(4, (names[7].name as DEROctetString).octets.size)
        assertEquals("spiffe://x/1", (names[8].name as DERIA5String).string)
        assertEquals("spiffe://x/2", (names[9].name as DERIA5String).string)
    }

    @Test
    fun `KEY is case-insensitive`() {
        val names = SubjectAlternativeNameParser.parse("dns=example.com, Email=u@x, uPn=marko@corp")?.names
        assertNotNull(names)
        assertEquals(3, names!!.size)
        assertEquals(GeneralName.dNSName, names[0].tagNo)
        assertEquals(GeneralName.rfc822Name, names[1].tagNo)
        assertEquals(GeneralName.otherName, names[2].tagNo)
    }

    @Test
    fun `whitespace around tokens and around equals is tolerated`() {
        val names = SubjectAlternativeNameParser.parse("  DNS  =  example.com  ,   EMAIL =u@x  ")?.names
        assertNotNull(names)
        assertEquals(2, names!!.size)
        assertEquals("example.com", (names[0].name as DERIA5String).string)
        assertEquals("u@x", (names[1].name as DERIA5String).string)
    }

    @Test
    fun `trailing comma and empty tokens are skipped`() {
        val names = SubjectAlternativeNameParser.parse("DNS=example.com, , ,")?.names
        assertNotNull(names)
        assertEquals(1, names!!.size)
    }

    @Test
    fun `unknown KEY throws`() {
        try {
            SubjectAlternativeNameParser.parse("FOO=bar")
            fail("Expected IllegalArgumentException")
        } catch (e: IllegalArgumentException) {
            assertTrue(e.message!!.contains("FOO"))
        }
    }

    @Test
    fun `RFC822 KEY is rejected as unknown in v1`() {
        try {
            SubjectAlternativeNameParser.parse("RFC822=user@example.com")
            fail("Expected IllegalArgumentException")
        } catch (e: IllegalArgumentException) {
            assertTrue(e.message!!.contains("RFC822"))
        }
    }

    @Test
    fun `malformed token without equals throws`() {
        try {
            SubjectAlternativeNameParser.parse("DNS=ok, OOPS")
            fail("Expected IllegalArgumentException")
        } catch (e: IllegalArgumentException) {
            assertTrue(e.message!!.contains("OOPS"))
        }
    }

    @Test
    fun `token with empty value throws`() {
        try {
            SubjectAlternativeNameParser.parse("DNS=")
            fail("Expected IllegalArgumentException")
        } catch (e: IllegalArgumentException) {
            assertTrue(e.message!!.contains("DNS=") || e.message!!.contains("empty value"))
        }
    }

    @Test
    fun `unparseable IP throws`() {
        try {
            SubjectAlternativeNameParser.parse("IP=not.an.address")
            fail("Expected IllegalArgumentException")
        } catch (e: IllegalArgumentException) {
            assertTrue(e.message!!.contains("not.an.address"))
        }
    }

    @Test
    fun `IPv4 with out-of-range octet throws`() {
        try {
            SubjectAlternativeNameParser.parse("IP=999.0.0.1")
            fail("Expected IllegalArgumentException")
        } catch (e: IllegalArgumentException) {
            assertTrue(e.message!!.contains("999"))
        }
    }

    @Test
    fun `IPv4 with leading zeros throws`() {
        try {
            SubjectAlternativeNameParser.parse("IP=192.168.001.1")
            fail("Expected IllegalArgumentException")
        } catch (e: IllegalArgumentException) {
            assertTrue(e.message!!.contains("192.168.001.1"))
        }
    }

    @Test
    fun `hostname-shaped IP value does not trigger DNS resolution and is rejected`() {
        try {
            SubjectAlternativeNameParser.parse("IP=example.com")
            fail("Expected IllegalArgumentException")
        } catch (e: IllegalArgumentException) {
            assertTrue(e.message!!.contains("example.com"))
        }
    }

    private fun assertUpnValue(name: ASN1Encodable, expected: String) {
        val seq = name as ASN1Sequence
        val tagged = seq.getObjectAt(1) as ASN1TaggedObject
        val utf8 = tagged.baseObject as DERUTF8String
        assertEquals(expected, utf8.string)
    }
}
