package com.fleetdm.agent.osquery.tables

import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.io.BufferedReader
import java.io.InputStreamReader
import java.util.concurrent.TimeUnit

class RoutesTable : TablePlugin {
    override val name: String = "routes"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("destination"),
        ColumnDef("gateway"),
        ColumnDef("interface"),
        ColumnDef("raw"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        return runCatching {
            val process = ProcessBuilder("ip", "route", "show")
                .redirectErrorStream(true)
                .start()

            val rows = mutableListOf<Map<String, String>>()
            BufferedReader(InputStreamReader(process.inputStream)).useLines { lines ->
                lines.take(256).forEach { line ->
                    val destination = line.substringBefore(" ").trim()
                    val gateway = extractTokenAfter(line, "via")
                    val iface = extractTokenAfter(line, "dev")
                    rows.add(
                        mapOf(
                            "destination" to destination,
                            "gateway" to gateway,
                            "interface" to iface,
                            "raw" to line.take(512),
                        ),
                    )
                }
            }
            if (!process.waitFor(2, TimeUnit.SECONDS)) {
                process.destroyForcibly()
            }
            rows
        }.getOrElse { emptyList() }
    }

    private fun extractTokenAfter(line: String, token: String): String {
        val parts = line.split(Regex("\\s+"))
        val idx = parts.indexOf(token)
        return if (idx >= 0 && idx + 1 < parts.size) parts[idx + 1] else ""
    }
}
