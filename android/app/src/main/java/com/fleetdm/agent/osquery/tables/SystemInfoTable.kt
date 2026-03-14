package com.fleetdm.agent.osquery.tables

import android.app.ActivityManager
import android.content.Context
import android.os.Build
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.OsqueryIdentityStore
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext

class SystemInfoTable(private val context: Context) : TablePlugin {
    override val name: String = "system_info"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("hostname"),
        ColumnDef("computer_name"),
        ColumnDef("uuid"),
        ColumnDef("hardware_vendor"),
        ColumnDef("hardware_model"),
        ColumnDef("hardware_version"),
        ColumnDef("cpu_brand"),
        ColumnDef("physical_memory"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val uuid = OsqueryIdentityStore.getOrCreateUuid(context)
        val model = Build.MODEL.orEmpty()
        val manufacturer = Build.MANUFACTURER.orEmpty()
        val abi = Build.SUPPORTED_ABIS.firstOrNull().orEmpty()

        val memInfo = ActivityManager.MemoryInfo()
        val activityManager = context.getSystemService(Context.ACTIVITY_SERVICE) as? ActivityManager
        activityManager?.getMemoryInfo(memInfo)

        return listOf(
            mapOf(
                "hostname" to model,
                "computer_name" to model,
                "uuid" to uuid,
                "hardware_vendor" to manufacturer,
                "hardware_model" to model,
                "hardware_version" to Build.HARDWARE.orEmpty(),
                "cpu_brand" to abi,
                "physical_memory" to memInfo.totalMem.coerceAtLeast(0L).toString(),
            ),
        )
    }
}
