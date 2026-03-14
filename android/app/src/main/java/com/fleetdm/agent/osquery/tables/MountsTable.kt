package com.fleetdm.agent.osquery.tables

import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.io.File

class MountsTable : TablePlugin {
    override val name: String = "mounts"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("device"),
        ColumnDef("path"),
        ColumnDef("type"),
        ColumnDef("flags"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        return runCatching {
            val f = File("/proc/mounts")
            if (!f.exists()) return@runCatching emptyList<Map<String, String>>()
            f.readLines().mapNotNull { line ->
                val parts = line.split(Regex("\\s+"))
                if (parts.size < 4) return@mapNotNull null
                mapOf(
                    "device" to parts[0],
                    "path" to parts[1],
                    "type" to parts[2],
                    "flags" to parts[3],
                )
            }
        }.getOrElse { emptyList() }
    }
}
