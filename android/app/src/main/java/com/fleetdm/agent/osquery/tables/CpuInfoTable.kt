package com.fleetdm.agent.osquery.tables

import android.os.Build
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.io.File

class CpuInfoTable : TablePlugin {
    override val name: String = "cpu_info"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("cores"),
        ColumnDef("arch"),
        ColumnDef("model"),
        ColumnDef("hardware"),
        ColumnDef("vendor"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val cores = Runtime.getRuntime().availableProcessors().coerceAtLeast(0)
        val arch = Build.SUPPORTED_ABIS.firstOrNull().orEmpty()
        val hardware = Build.HARDWARE.orEmpty()
        val model = readCpuInfoValue("model name")
            ?: readCpuInfoValue("Processor")
            ?: Build.MODEL.orEmpty()
        val vendor = readCpuInfoValue("vendor_id")
            ?: readCpuInfoValue("Hardware")
            ?: Build.MANUFACTURER.orEmpty()

        return listOf(
            mapOf(
                "cores" to cores.toString(),
                "arch" to arch,
                "model" to model,
                "hardware" to hardware,
                "vendor" to vendor,
            ),
        )
    }

    private fun readCpuInfoValue(key: String): String? {
        return runCatching {
            val f = File("/proc/cpuinfo")
            if (!f.exists()) return null
            f.useLines { lines ->
                lines.firstOrNull { it.startsWith("$key", ignoreCase = true) }
                    ?.substringAfter(":", "")
                    ?.trim()
                    ?.takeIf { it.isNotEmpty() }
            }
        }.getOrNull()
    }
}
