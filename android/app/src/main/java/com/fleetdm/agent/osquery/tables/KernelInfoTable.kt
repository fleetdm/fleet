package com.fleetdm.agent.osquery.tables

import android.os.Build
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext

class KernelInfoTable : TablePlugin {
    override val name: String = "kernel_info"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("version"),
        ColumnDef("release"),
        ColumnDef("build"),
        ColumnDef("platform"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val osVersion = System.getProperty("os.version").orEmpty()

        return listOf(
            mapOf(
                "version" to osVersion,
                "release" to Build.VERSION.RELEASE.orEmpty(),
                "build" to Build.ID.orEmpty(),
                "platform" to "android",
            ),
        )
    }
}
