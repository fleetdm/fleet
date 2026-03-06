package com.fleetdm.agent.osquery

import com.fleetdm.agent.osquery.core.TableQueryContext
import com.fleetdm.agent.osquery.core.TableRegistry
import com.fleetdm.agent.osquery.core.WhereCond
import com.fleetdm.agent.osquery.core.WhereOp
import com.fleetdm.agent.osquery.core.parseSelectSql

/**
 * Tiny, "good enough" osquery-ish executor:
 * - Supports SELECT * / SELECT col1,col2
 * - FROM <table>
 * - WHERE with AND + (= | LIKE)
 */
object OsqueryQueryEngine {

    suspend fun execute(sql: String): List<Map<String, String>> {
        val parsed = parseSelectSql(sql)

        val fullRows = TableRegistry.runTable(parsed.tableName, TableQueryContext())

        val filteredRows = if (parsed.where.isEmpty()) {
            fullRows
        } else {
            fullRows.filter { row -> matchesWhere(row, parsed.where) }
        }

        if (parsed.selectAll) return filteredRows

        val schemaCols = TableRegistry.getColumns(parsed.tableName).map { it.name }
        val schemaLowerToReal = schemaCols.associateBy { it.lowercase() }

        val selectedRealCols = parsed.selectedColumns.mapNotNull { sel ->
            schemaLowerToReal[sel.lowercase()]
        }

        if (selectedRealCols.size != parsed.selectedColumns.size) {
            val missing = parsed.selectedColumns.filter { schemaLowerToReal[it.lowercase()] == null }
            throw IllegalArgumentException(
                "Unknown columns for table '${parsed.tableName}': ${missing.joinToString(", ")}"
            )
        }

        return TableRegistry.projectRows(filteredRows, selectedRealCols)
    }

    private fun matchesWhere(row: Map<String, String>, where: List<WhereCond>): Boolean {
        for (cond in where) {
            val actual = row[cond.column]
                ?: row.entries.firstOrNull { it.key.equals(cond.column, ignoreCase = true) }?.value
                ?: return false

            when (cond.op) {
                WhereOp.EQ -> if (!actual.equals(cond.value, ignoreCase = true)) return false
                WhereOp.LIKE -> if (!likeMatch(actual, cond.value)) return false
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
