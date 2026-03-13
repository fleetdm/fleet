package com.fleetdm.agent.osquery.tables

import android.content.Context
import android.content.RestrictionsManager
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TableQueryContext
import com.fleetdm.agent.osquery.core.TablePlugin
import java.io.BufferedReader
import java.io.InputStreamReader
import java.util.concurrent.TimeUnit

class AndroidLogcatTable(private val context: Context) : TablePlugin {

    override val name: String = "android_logcat"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("timestamp"),
        ColumnDef("level"),
        ColumnDef("tag"),
        ColumnDef("message"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        if (!isLogcatTableEnabled()) return emptyList()

        val rows = mutableListOf<Map<String, String>>()
        val command =
            listOf(
                "logcat", "-d", "-v", "brief",
                "fleet-app:V",
                "fleet-ApiClient:V",
                "fleet-distributed:V",
                "fleet-CertificateEnrollmentWorker:V",
                "fleet-CertificateOrchestrator:V",
                "fleet-boot:V",
                "fleet-RoleNotificationReceiverService:V",
                "fleet-crash:V",
                "FleetOsquery:V",
                "*:S",
            ).toTypedArray()

        try {
            val process = Runtime.getRuntime().exec(command)

            BufferedReader(InputStreamReader(process.inputStream)).useLines { lines ->
                lines.take(200).forEach { line ->
                    // Example:
                    // D/TagName( 1234): message
                    val regex = Regex("""^([VDIWEF])\/([^ ]+).*?: (.*)$""")
                    val match = regex.find(line) ?: return@forEach

                    val (level, tag, message) = match.destructured

                    rows.add(
                        mapOf(
                            "timestamp" to System.currentTimeMillis().toString(),
                            "level" to level,
                            "tag" to tag,
                            "message" to redactSensitiveValues(message).take(500),
                        )
                    )
                }
            }

            if (!process.waitFor(2, TimeUnit.SECONDS)) {
                process.destroyForcibly()
            }
        } catch (_: Exception) {
            // osquery tables must never crash the agent
        }

        return rows
    }

    private fun isLogcatTableEnabled(): Boolean {
        val restrictionsManager = context.getSystemService(Context.RESTRICTIONS_SERVICE) as? RestrictionsManager
            ?: return false
        val appRestrictions = restrictionsManager.applicationRestrictions ?: return false
        val key = "enable_android_logcat_table"

        if (!appRestrictions.containsKey(key)) return false

        if (appRestrictions.getBoolean(key, false)) return true
        return appRestrictions.getString(key)?.equals("true", ignoreCase = true) == true
    }

    private fun redactSensitiveValues(input: String): String {
        val patterns =
            listOf(
                Regex("(?i)(authorization\\s*[:=]\\s*)(\\S+)"),
                Regex("(?i)(bearer\\s+)([A-Za-z0-9._-]+)"),
                Regex("(?i)(node[_ -]?key\\s*[:=]\\s*)(\\S+)"),
                Regex("(?i)(api[_ -]?key\\s*[:=]\\s*)(\\S+)"),
                Regex("(?i)(enroll_secret\\s*[:=]\\s*)(\\S+)"),
                Regex("(?i)(token\\s*[:=]\\s*)(\\S+)"),
                Regex("(?i)(password\\s*[:=]\\s*)(\\S+)"),
            )

        var redacted = input
        for (pattern in patterns) {
            redacted = pattern.replace(redacted, "$1<redacted>")
        }
        return redacted
    }
}
