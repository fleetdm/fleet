package com.fleetdm.agent.osquery.tables

import android.app.ActivityManager
import android.content.Context
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext

class ProcessesTable(private val context: Context) : TablePlugin {
    override val name: String = "processes"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("pid"),
        ColumnDef("name"),
        ColumnDef("uid"),
        ColumnDef("package_name"),
        ColumnDef("importance"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val am = context.getSystemService(Context.ACTIVITY_SERVICE) as? ActivityManager
            ?: return emptyList()
        val procs = am.runningAppProcesses ?: return emptyList()

        return procs.map { p ->
            mapOf(
                "pid" to p.pid.toString(),
                "name" to (p.processName ?: ""),
                "uid" to p.uid.toString(),
                "package_name" to (p.pkgList?.firstOrNull() ?: ""),
                "importance" to p.importance.toString(),
            )
        }
    }
}
