package com.example.hello2.core

import org.json.JSONArray
import org.json.JSONObject

object TableRegistry {
    private val tables = mutableMapOf<String, TablePlugin>()

    fun register(table: TablePlugin) {
        tables[table.name.lowercase()] = table
    }

    fun getTable(tableName: String): TablePlugin =
        tables[tableName.lowercase()] ?: error("unknown table: $tableName")

    fun getColumns(tableName: String): List<ColumnDef> =
        getTable(tableName).columns()

    suspend fun runTable(tableName: String, ctx: TableQueryContext): List<Map<String, String>> {
        val t = getTable(tableName)
        val schema = t.columns().map { it.name }.toSet()
        val rows = t.generate(ctx)

        // strict: table must not return unknown columns
        for (row in rows) {
            for (k in row.keys) {
                require(k in schema) { "table '$tableName' returned unknown column '$k'" }
            }
        }

        // fill missing cols as ""
        return rows.map { row ->
            schema.associateWith { col -> row[col] ?: "" }
        }
    }

    fun projectRows(rows: List<Map<String, String>>, selectedCols: List<String>): List<Map<String, String>> {
        val selectedSet = selectedCols.toSet()
        return rows.map { row -> row.filterKeys { it in selectedSet } }
    }

    fun rowsToJson(rows: List<Map<String, String>>): String {
        val arr = JSONArray()
        for (row in rows) {
            val obj = JSONObject()
            for ((k, v) in row) obj.put(k, v)
            arr.put(obj)
        }
        return arr.toString()
    }
}


