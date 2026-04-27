package com.fleetdm.agent.osquery.tables

import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.security.KeyStore
import java.security.MessageDigest
import java.security.cert.X509Certificate
import java.util.Locale

class CertificatesTable : TablePlugin {
    override val name: String = "certificates"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("alias"),
        ColumnDef("subject"),
        ColumnDef("issuer"),
        ColumnDef("serial"),
        ColumnDef("not_before"),
        ColumnDef("not_after"),
        ColumnDef("sha256"),
        ColumnDef("store"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val rows = mutableListOf<Map<String, String>>()

        // This is the important part:
        // AndroidCAStore exposes system+user trusted CA certs.
        val ks = KeyStore.getInstance("AndroidCAStore")
        ks.load(null)

        val aliases = ks.aliases()
        while (aliases.hasMoreElements()) {
            val alias = aliases.nextElement()

            val cert = ks.getCertificate(alias)
            val x509 = cert as? X509Certificate ?: continue

            rows.add(
                mapOf(
                    "alias" to alias,
                    "subject" to (x509.subjectX500Principal?.name ?: ""),
                    "issuer" to (x509.issuerX500Principal?.name ?: ""),
                    "serial" to (x509.serialNumber?.toString() ?: ""),
                    "not_before" to (x509.notBefore?.toInstant()?.toString().orEmpty()),
                    "not_after" to (x509.notAfter?.toInstant()?.toString().orEmpty()),
                    "sha256" to sha256Hex(x509.encoded),
                    "store" to storeLabelFromAlias(alias),
                ),
            )
        }

        return rows
    }

    private fun sha256Hex(bytes: ByteArray): String {
        val md = MessageDigest.getInstance("SHA-256")
        val digest = md.digest(bytes)
        return digest.joinToString("") { "%02x".format(it) }
    }

    private fun storeLabelFromAlias(alias: String): String {
        // Convention in AndroidCAStore:
        // "system:" prefix = system store
        // "user:" prefix = user-installed
        val a = alias.lowercase(Locale.ROOT)
        return when {
            a.startsWith("system:") -> "system"
            a.startsWith("user:") -> "user"
            else -> "unknown"
        }
    }
}
