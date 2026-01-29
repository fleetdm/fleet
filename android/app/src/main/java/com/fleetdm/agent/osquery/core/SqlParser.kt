package com.fleetdm.agent.osquery.core

enum class WhereOp { EQ, LIKE }

data class WhereCond(
    val column: String,
    val op: WhereOp,
    val value: String
)

data class ParsedSql(
    val tableName: String,
    val selectedColumns: List<String>,
    val selectAll: Boolean,
    val where: List<WhereCond> = emptyList()
)

/**
 * Supports:
 *   SELECT * FROM table
 *   SELECT a,b FROM table
 *   SELECT ... FROM table WHERE col = 'x'
 *   SELECT ... FROM table WHERE col LIKE '%x%' AND other = "y"
 *
 * Notes:
 * - Only WHERE with AND is supported.
 * - Operators supported: = , LIKE
 * - Values can be: 'single quoted', "double quoted", or bare tokens without spaces.
 */
fun parseSelectSql(sql: String): ParsedSql {
    val s = sql.trim().removeSuffix(";").trim()

    // Capture:
    // 1) columns part
    // 2) table name
    // 3) optional where clause
    val re = Regex(
        pattern = """(?is)^\s*select\s+(.+?)\s+from\s+([a-zA-Z0-9_]+)(?:\s+where\s+(.+?))?\s*$"""
    )
    val m = re.find(s) ?: throw IllegalArgumentException("Bad SQL. Expected: SELECT <cols> FROM <table> [WHERE ...]")

    val colsPart = m.groupValues[1].trim()
    val table = m.groupValues[2].trim()
    val wherePart = m.groupValues.getOrNull(3)?.trim().orEmpty()

    val (selectAll, cols) = if (colsPart == "*") {
        true to emptyList()
    } else {
        val list = colsPart.split(",").map { it.trim() }.filter { it.isNotEmpty() }
        if (list.isEmpty()) throw IllegalArgumentException("No columns selected")
        false to list
    }

    val where = if (wherePart.isBlank()) emptyList() else parseWhere(wherePart)

    return ParsedSql(
        tableName = table,
        selectedColumns = cols,
        selectAll = selectAll,
        where = where
    )
}

private fun parseWhere(where: String): List<WhereCond> {
    val parts = splitByAnd(where)
    if (parts.isEmpty()) throw IllegalArgumentException("Bad WHERE clause")

    return parts.map { term ->
        val t = term.trim()
        val m = Regex("""(?is)^\s*([a-zA-Z0-9_]+)\s*(=|like)\s*(.+?)\s*$""").find(t)
            ?: throw IllegalArgumentException("Bad WHERE term: $term")

        val col = m.groupValues[1].trim()
        val opStr = m.groupValues[2].trim().lowercase()
        val rawVal = m.groupValues[3].trim()

        val value = unquote(rawVal)

        val op = when (opStr) {
            "=" -> WhereOp.EQ
            "like" -> WhereOp.LIKE
            else -> throw IllegalArgumentException("Unsupported WHERE operator: $opStr")
        }

        WhereCond(column = col, op = op, value = value)
    }
}

/**
 * Splits on AND, but respects quotes so this works:
 *   col = "a and b" AND x = 1
 */
private fun splitByAnd(s: String): List<String> {
    val out = mutableListOf<String>()
    val sb = StringBuilder()

    var inSingle = false
    var inDouble = false
    var i = 0

    fun flush() {
        val part = sb.toString().trim()
        if (part.isNotEmpty()) out.add(part)
        sb.setLength(0)
    }

    while (i < s.length) {
        val c = s[i]

        if (c == '\'' && !inDouble) {
            inSingle = !inSingle
            sb.append(c)
            i++
            continue
        }
        if (c == '"' && !inSingle) {
            inDouble = !inDouble
            sb.append(c)
            i++
            continue
        }

        // Check for AND when not inside quotes
        if (!inSingle && !inDouble) {
            // match case-insensitive " AND " with surrounding whitespace
            if (i + 3 <= s.length) {
                val remaining = s.substring(i)
                val m = Regex("""(?is)^\s+and\s+""").find(remaining)
                if (m != null && m.range.first == 0) {
                    flush()
                    i += m.value.length
                    continue
                }
            }
        }

        sb.append(c)
        i++
    }

    flush()
    return out
}

private fun unquote(v: String): String {
    if (v.length >= 2 && v.first() == '\'' && v.last() == '\'') {
        return v.substring(1, v.length - 1)
    }
    if (v.length >= 2 && v.first() == '"' && v.last() == '"') {
        return v.substring(1, v.length - 1)
    }
    return v
}
