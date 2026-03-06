package com.fleetdm.agent.osquery.tables

import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.io.BufferedReader
import java.io.InputStreamReader

class AndroidLogcatTable : TablePlugin {

    override val name: String = "android_logcat"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("timestamp"),
        ColumnDef("level"),
        ColumnDef("tag"),
        ColumnDef("message"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val rows = mutableListOf<Map<String, String>>()

        try {
            val process = Runtime.getRuntime().exec(
                arrayOf("logcat", "-d", "-v", "brief")
            )

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
                            "message" to message,
                        )
                    )
                }
            }
        } catch (_: Exception) {
            // osquery tables must never crash the agent
        }

        return rows
    }
}
