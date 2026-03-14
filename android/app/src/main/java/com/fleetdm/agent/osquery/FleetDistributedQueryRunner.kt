package com.fleetdm.agent.osquery

import android.util.Log
import com.fleetdm.agent.ApiClient
import com.fleetdm.agent.osquery.core.TableQueryContext
import com.fleetdm.agent.osquery.core.TableRegistry
import com.fleetdm.agent.osquery.core.WhereCond
import com.fleetdm.agent.osquery.core.WhereOp
import com.fleetdm.agent.osquery.core.parseSelectSql

object FleetDistributedQueryRunner {

    private const val tag = "FleetOsquery"

    // If true, when we cannot run a query, we still answer it with [] so Fleet stops sending it.
    var clearUnknownQueries: Boolean = true

    suspend fun runOnce() {
        val startMs = System.currentTimeMillis()
        val readResp = ApiClient.distributedRead()
            .getOrElse { throw RuntimeException("distributed/read failed: ${it.message}") }
        val queryNames = readResp.queries.keys.sorted()

        val resultsToWrite = linkedMapOf<String, List<Map<String, String>>>()
        var handled = 0

        for (qName in queryNames) {
            val sql = readResp.queries[qName].orEmpty()
            if (sql.isBlank()) continue

            try {
                val rows = executeSqlViaTables(sql)
                resultsToWrite[qName] = rows
                handled++
            } catch (e: Exception) {
                val msg = e.message ?: e.javaClass.simpleName
                Log.w(tag, "Query failed: $qName err=$msg")

                if (clearUnknownQueries) {
                    resultsToWrite[qName] = emptyList()
                    handled++
                }
            }
        }

        if (resultsToWrite.isNotEmpty()) {
            ApiClient.distributedWrite(resultsToWrite)
                .getOrElse { throw RuntimeException("distributed/write failed: ${it.message}") }
        }

        val took = System.currentTimeMillis() - startMs
        Log.i(tag, "runOnce handled=$handled wrote=${resultsToWrite.size} tookMs=$took")
    }

    private suspend fun executeSqlViaTables(sql: String): List<Map<String, String>> {
        val parsed = parseSelectSql(sql)

        val fullRows = TableRegistry.runTable(parsed.tableName, TableQueryContext())

        val filteredRows =
            if (parsed.where.isEmpty()) fullRows
            else fullRows.filter { row -> matchesWhere(row, parsed.where) }

        if (parsed.selectAll) return filteredRows

        val schemaCols = TableRegistry.getColumns(parsed.tableName).map { it.name }
        val schemaLowerToReal = schemaCols.associateBy { it.lowercase() }

        val selectedRealCols = parsed.selectedColumns.mapNotNull { sel ->
            schemaLowerToReal[sel.lowercase()]
        }

        if (selectedRealCols.size != parsed.selectedColumns.size) {
            val missing = parsed.selectedColumns.filter {
                schemaLowerToReal[it.lowercase()] == null
            }
            throw IllegalArgumentException(
                "Unknown columns for table '${parsed.tableName}': ${missing.joinToString(", ")}"
            )
        }

        return TableRegistry.projectRows(filteredRows, selectedRealCols)
    }

    private fun matchesWhere(row: Map<String, String>, where: List<WhereCond>): Boolean {
        for (cond in where) {
            val actual =
                row[cond.column]
                    ?: row.entries.firstOrNull {
                        it.key.equals(cond.column, ignoreCase = true)
                    }?.value
                    ?: return false

            when (cond.op) {
                WhereOp.EQ ->
                    if (!actual.equals(cond.value, ignoreCase = true)) return false

                WhereOp.LIKE ->
                    if (!likeMatch(actual, cond.value)) return false
            }
        }
        return true
    }

    private fun likeMatch(actual: String, pattern: String): Boolean {
        val escaped = Regex.escape(pattern)
            .replace("%", ".*")
            .replace("_", ".")
        val re = Regex("^$escaped$", RegexOption.IGNORE_CASE)
        return re.matches(actual)
    }
}
