package com.example.hello2.core

data class ColumnDef(val name: String, val type: String = "TEXT")

data class TableQueryContext(
    val constraints: Map<String, String> = emptyMap()
)

interface TablePlugin {
    val name: String
    fun columns(): List<ColumnDef>
    suspend fun generate(ctx: TableQueryContext): List<Map<String, String>>
}

