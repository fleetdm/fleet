package com.example.hello2.core

data class ParsedSql(val tableName: String, val selectedColumns: List<String>, val selectAll: Boolean)

fun parseSelectSql(sql: String): ParsedSql {
    val s = sql.trim().removeSuffix(";").trim()
    val re = Regex("""(?is)^\s*select\s+(.+?)\s+from\s+([a-zA-Z0-9_]+)\s*$""")
    val m = re.find(s) ?: throw IllegalArgumentException("Bad SQL. Expected: SELECT <cols> FROM <table>")

    val colsPart = m.groupValues[1].trim()
    val table = m.groupValues[2].trim()

    if (colsPart == "*") {
        return ParsedSql(tableName = table, selectedColumns = emptyList(), selectAll = true)
    }

    val cols = colsPart.split(",")
        .map { it.trim() }
        .filter { it.isNotEmpty() }

    if (cols.isEmpty()) throw IllegalArgumentException("No columns selected")

    return ParsedSql(tableName = table, selectedColumns = cols, selectAll = false)
}


