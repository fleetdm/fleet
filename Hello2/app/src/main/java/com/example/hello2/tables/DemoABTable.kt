package com.example.hello2.tables

import com.example.hello2.core.ColumnDef
import com.example.hello2.core.TablePlugin
import com.example.hello2.core.TableQueryContext

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


