package com.fleetdm.agent.osquery.tables

import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.security.MessageDigest
import java.security.cert.CertificateFactory
import java.security.cert.X509Certificate

class AppSignaturesTable(private val context: Context) : TablePlugin {
    override val name: String = "app_signatures"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("app_name"),
        ColumnDef("package_name"),
        ColumnDef("sha256"),
        ColumnDef("subject"),
        ColumnDef("issuer"),
        ColumnDef("version_name"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val pm = context.packageManager
        val packages = getInstalledPackages(pm)
        val out = mutableListOf<Map<String, String>>()

        for (pkg in packages) {
            val signers = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
                pkg.signingInfo?.apkContentsSigners?.toList().orEmpty()
            } else {
                @Suppress("DEPRECATION")
                pkg.signatures?.toList().orEmpty()
            }
            if (signers.isEmpty()) continue

            val signer = signers.first()
            val cert = parseX509(signer.toByteArray())
            val digest = sha256Hex(signer.toByteArray())
            out.add(
                mapOf(
                    "app_name" to (pkg.applicationInfo?.loadLabel(pm)?.toString() ?: pkg.packageName),
                    "package_name" to pkg.packageName,
                    "sha256" to digest,
                    "subject" to (cert?.subjectX500Principal?.name ?: ""),
                    "issuer" to (cert?.issuerX500Principal?.name ?: ""),
                    "version_name" to (pkg.versionName ?: ""),
                ),
            )
        }

        return out
    }

    private fun getInstalledPackages(pm: PackageManager) =
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            pm.getInstalledPackages(
                PackageManager.PackageInfoFlags.of(
                    (PackageManager.GET_SIGNING_CERTIFICATES or PackageManager.GET_META_DATA).toLong(),
                ),
            )
        } else {
            @Suppress("DEPRECATION")
            pm.getInstalledPackages(PackageManager.GET_SIGNATURES or PackageManager.GET_META_DATA)
        }

    private fun parseX509(bytes: ByteArray): X509Certificate? = runCatching {
        val cf = CertificateFactory.getInstance("X.509")
        cf.generateCertificate(bytes.inputStream()) as X509Certificate
    }.getOrNull()

    private fun sha256Hex(bytes: ByteArray): String {
        val md = MessageDigest.getInstance("SHA-256")
        return md.digest(bytes).joinToString("") { "%02x".format(it) }
    }
}
