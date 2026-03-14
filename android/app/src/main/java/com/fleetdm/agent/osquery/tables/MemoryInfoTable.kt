package com.fleetdm.agent.osquery.tables

import android.app.ActivityManager
import android.content.Context
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext

class MemoryInfoTable(private val context: Context) : TablePlugin {
    override val name: String = "memory_info"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("total_bytes"),
        ColumnDef("available_bytes"),
        ColumnDef("threshold_bytes"),
        ColumnDef("low_memory"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val memoryInfo = ActivityManager.MemoryInfo()
        val activityManager = context.getSystemService(Context.ACTIVITY_SERVICE) as? ActivityManager
        activityManager?.getMemoryInfo(memoryInfo)

        return listOf(
            mapOf(
                "total_bytes" to memoryInfo.totalMem.coerceAtLeast(0L).toString(),
                "available_bytes" to memoryInfo.availMem.coerceAtLeast(0L).toString(),
                "threshold_bytes" to memoryInfo.threshold.coerceAtLeast(0L).toString(),
                "low_memory" to if (memoryInfo.lowMemory) "1" else "0",
            ),
        )
    }
}
