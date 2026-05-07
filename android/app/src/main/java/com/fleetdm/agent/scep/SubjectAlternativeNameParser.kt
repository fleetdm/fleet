package com.fleetdm.agent.scep

import org.bouncycastle.asn1.ASN1ObjectIdentifier
import org.bouncycastle.asn1.DERIA5String
import org.bouncycastle.asn1.DEROctetString
import org.bouncycastle.asn1.DERSequence
import org.bouncycastle.asn1.DERTaggedObject
import org.bouncycastle.asn1.DERUTF8String
import org.bouncycastle.asn1.x509.GeneralName
import org.bouncycastle.asn1.x509.GeneralNames
import java.net.InetAddress

/**
 * Parses the comma-separated KEY=value SAN string carried on the certificate template
 * (e.g. "DNS=example.com, UPN=user@corp.example.com") into a BouncyCastle GeneralNames
 * structure suitable for inclusion in a PKCS#10 CSR's subjectAltName extension.
 *
 * Supported KEYs (case-insensitive):
 *   DNS   -> dNSName (DERIA5String)
 *   EMAIL -> rfc822Name (DERIA5String)
 *   URI   -> uniformResourceIdentifier (DERIA5String)
 *   IP    -> iPAddress (DEROctetString, 4 bytes IPv4 or 16 bytes IPv6)
 *   UPN   -> otherName (OID 1.3.6.1.4.1.311.20.2.3, [0] EXPLICIT DERUTF8String)
 *           per Microsoft KB258605 / RFC 4556 §3.2.1.
 *
 * Returns null for null / empty / blank input so the caller can skip adding the extension.
 * Throws IllegalArgumentException for unknown KEYs, malformed tokens, empty values, or
 * unparseable IP literals.
 */
object SubjectAlternativeNameParser {

    private const val MICROSOFT_UPN_OID = "1.3.6.1.4.1.311.20.2.3"

    fun parse(sanString: String?): GeneralNames? {
        if (sanString.isNullOrBlank()) return null

        val names = sanString.split(',')
            .map { it.trim() }
            .filter { it.isNotEmpty() }
            .map { parseToken(it) }

        if (names.isEmpty()) return null

        return GeneralNames(names.toTypedArray())
    }

    private fun parseToken(token: String): GeneralName {
        val eqIndex = token.indexOf('=')
        if (eqIndex <= 0 || eqIndex == token.length - 1) {
            throw IllegalArgumentException("Malformed SAN token: \"$token\" (expected KEY=value)")
        }
        val key = token.substring(0, eqIndex).trim().uppercase()
        val value = token.substring(eqIndex + 1).trim()
        if (value.isEmpty()) {
            throw IllegalArgumentException("Malformed SAN token: \"$token\" (empty value)")
        }
        return when (key) {
            "DNS" -> GeneralName(GeneralName.dNSName, DERIA5String(value))
            "EMAIL" -> GeneralName(GeneralName.rfc822Name, DERIA5String(value))
            "URI" -> GeneralName(GeneralName.uniformResourceIdentifier, DERIA5String(value))
            "IP" -> GeneralName(GeneralName.iPAddress, encodeIp(value))
            "UPN" -> GeneralName(GeneralName.otherName, encodeUpn(value))
            else -> throw IllegalArgumentException("Unknown SAN KEY: \"$key\" (supported: DNS, EMAIL, URI, IP, UPN)")
        }
    }

    private fun encodeIp(value: String): DEROctetString {
        // We must reject hostnames here. InetAddress.getByName does DNS resolution for
        // anything that isn't an IP literal, which we never want for a SAN. So we parse
        // IPv4 strictly ourselves, and only fall back to InetAddress.getByName when the
        // value contains a colon (i.e. is shaped like IPv6 / IPv4-mapped IPv6); those
        // are never hostnames, so getByName is guaranteed not to issue a DNS lookup.
        val addr = parseIpLiteral(value)
            ?: throw IllegalArgumentException(
                "Unparseable IP address: \"$value\" (expected IPv4 dotted-quad or IPv6 colon-hex)",
            )
        return DEROctetString(addr.address)
    }

    private fun parseIpLiteral(value: String): InetAddress? {
        if (value.contains(':')) {
            return try {
                InetAddress.getByName(value)
            } catch (e: Exception) {
                null
            }
        }
        val parts = value.split('.')
        if (parts.size != 4) return null
        val bytes = ByteArray(4)
        for (i in 0..3) {
            val part = parts[i]
            if (part.isEmpty() || part.length > 3) return null
            val n = part.toIntOrNull() ?: return null
            if (n !in 0..255) return null
            if (part != n.toString()) return null
            bytes[i] = n.toByte()
        }
        return try {
            InetAddress.getByAddress(bytes)
        } catch (e: Exception) {
            null
        }
    }

    private fun encodeUpn(value: String): DERSequence = DERSequence(
        arrayOf(
            ASN1ObjectIdentifier(MICROSOFT_UPN_OID),
            DERTaggedObject(true, 0, DERUTF8String(value)),
        ),
    )
}
