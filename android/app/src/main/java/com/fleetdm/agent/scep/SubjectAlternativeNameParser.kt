package com.fleetdm.agent.scep

import org.bouncycastle.asn1.ASN1ObjectIdentifier
import org.bouncycastle.asn1.DERIA5String
import org.bouncycastle.asn1.DERSequence
import org.bouncycastle.asn1.DERTaggedObject
import org.bouncycastle.asn1.DERUTF8String
import org.bouncycastle.asn1.x509.GeneralName
import org.bouncycastle.asn1.x509.GeneralNames
import org.bouncycastle.util.IPAddress
import java.util.Locale

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
 *
 * Reserved characters in values: a literal `,` would split the token, so values that need
 * one (most commonly URI paths/queries) must be percent-encoded per RFC 3986 — `%2C`. The
 * encoded form is preserved verbatim on the issued cert, which is the canonical wire form
 * per RFC 5280 §4.2.1.6. This mirrors OpenSSL's `openssl.cnf` SAN syntax.
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
        require(eqIndex > 0 && eqIndex != token.length - 1) {
            "Malformed SAN token: \"$token\" (expected KEY=value)"
        }
        // Locale.ROOT keeps the comparison locale-independent: a Turkish-locale device
        // would otherwise turn "ip" into "İP", which would not match the ASCII "IP" arm
        // and would fail enrollment.
        val key = token.substring(0, eqIndex).trim().uppercase(Locale.ROOT)
        val value = token.substring(eqIndex + 1).trim()
        require(value.isNotEmpty()) { "Malformed SAN token: \"$token\" (empty value)" }
        return when (key) {
            "DNS" -> GeneralName(GeneralName.dNSName, DERIA5String(value))
            "EMAIL" -> GeneralName(GeneralName.rfc822Name, DERIA5String(value))
            "URI" -> GeneralName(GeneralName.uniformResourceIdentifier, DERIA5String(value))
            "IP" -> {
                // BouncyCastle's IPAddress.isValid is literal-only (no DNS) and rejects
                // bracketed forms, zone IDs, and anything outside dotted-quad / colon-hex.
                // GeneralName(iPAddress, String) then encodes the literal to the raw 4-
                // or 16-byte octet string the SAN extension requires.
                require(IPAddress.isValid(value)) {
                    "Unparseable IP address: \"$value\" (expected IPv4 dotted-quad or IPv6 colon-hex)"
                }
                GeneralName(GeneralName.iPAddress, value)
            }
            "UPN" -> GeneralName(GeneralName.otherName, encodeUpn(value))
            else -> throw IllegalArgumentException("Unknown SAN KEY: \"$key\" (supported: DNS, EMAIL, URI, IP, UPN)")
        }
    }

    private fun encodeUpn(value: String): DERSequence = DERSequence(
        arrayOf(
            ASN1ObjectIdentifier(MICROSOFT_UPN_OID),
            DERTaggedObject(true, 0, DERUTF8String(value)),
        ),
    )
}
