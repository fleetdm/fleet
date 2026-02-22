package com.fleetdm.agent.osquery.tables

import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.io.BufferedReader
import java.io.InputStreamReader

class SystemPropertiesTable : TablePlugin {
    override val name: String = "system_properties"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("key"),
        ColumnDef("value"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val rows = mutableListOf<Map<String, String>>()

        val proc = ProcessBuilder("getprop")
            .redirectErrorStream(true)
            .start()

        BufferedReader(InputStreamReader(proc.inputStream)).use { br ->
            while (true) {
                val line = br.readLine() ?: break
                // getprop output format: [ro.build.version.release]: [14]
                val parsed = parseGetpropLine(line) ?: continue
                rows.add(mapOf("key" to parsed.first, "value" to parsed.second))
            }
        }

        // Don’t hang forever if something weird happens
        runCatching { proc.waitFor() }

        return rows
    }

    private fun parseGetpropLine(line: String): Pair<String, String>? {
        val trimmed = line.trim()
        if (!trimmed.startsWith("[") || !trimmed.contains("]: [")) return null

        val mid = trimmed.indexOf("]: [")
        if (mid <= 1) return null

        val key = trimmed.substring(1, mid)
        val value = trimmed.substring(mid + 4, trimmed.length - 1) // after "]: [" and before trailing "]"
        return key to value
    }
}
