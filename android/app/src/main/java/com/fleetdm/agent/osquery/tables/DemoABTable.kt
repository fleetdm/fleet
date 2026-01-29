package com.fleetdm.agent.osquery.tables

import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext

object DemoABTable : TablePlugin {
    override val name: String = "demo_ab"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("A"),
        ColumnDef("B"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        return listOf(
            mapOf(
                "A" to "A",
                "B" to "B",
            )
        )
    }
}


