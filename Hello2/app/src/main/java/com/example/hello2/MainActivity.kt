package com.example.hello2

import android.os.Bundle
import android.util.Log
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.example.hello2.ui.theme.Hello2Theme
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONArray
import org.json.JSONObject
import java.util.Calendar

// -----------------------------
// 1) "osquery-like" table model
// -----------------------------
data class ColumnDef(val name: String, val type: String = "TEXT")

data class TableQueryContext(
    val constraints: Map<String, String> = emptyMap()
)

interface TablePlugin {
    val name: String
    fun columns(): List<ColumnDef>
    suspend fun generate(ctx: TableQueryContext): List<Map<String, String>>
}

object TableRegistry {
    private val tables = mutableMapOf<String, TablePlugin>()

    fun register(table: TablePlugin) {
        tables[table.name.lowercase()] = table
    }

    fun listTables(): List<String> = tables.keys.sorted()

    fun getTable(tableName: String): TablePlugin =
        tables[tableName.lowercase()] ?: error("unknown table: $tableName")

    fun getColumns(tableName: String): List<ColumnDef> =
        getTable(tableName).columns()

    suspend fun runTable(tableName: String, ctx: TableQueryContext): List<Map<String, String>> {
        val t = getTable(tableName)
        val schema = t.columns().map { it.name }.toSet()
        val rows = t.generate(ctx)

        // Strict: no unknown keys
        for (row in rows) {
            for (k in row.keys) {
                require(k in schema) { "table '$tableName' returned unknown column '$k'" }
            }
        }

        // Fill missing columns as ""
        return rows.map { row ->
            schema.associateWith { col -> row[col] ?: "" }
        }
    }

    fun projectRows(rows: List<Map<String, String>>, selectedCols: List<String>): List<Map<String, String>> {
        val selectedSet = selectedCols.toSet()
        return rows.map { row ->
            row.filterKeys { it in selectedSet }
        }
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

// ------------------------------------
// 2) Tables
// ------------------------------------

// Demo table: columns A and B (to match your SQL examples)
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

object PhoneTimeTable : TablePlugin {
    override val name: String = "phone_time"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("hour"),
        ColumnDef("minute"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val cal = Calendar.getInstance()
        val hour = cal.get(Calendar.HOUR_OF_DAY)
        val minute = cal.get(Calendar.MINUTE)

        return listOf(
            mapOf(
                "hour" to hour.toString(),
                "minute" to minute.toString(),
            )
        )
    }
}

// ------------------------------------
// 3) Tiny SQL parser: SELECT ... FROM ...
// Supports:
//   SELECT * FROM phone_time
//   SELECT hour FROM phone_time
//   SELECT A FROM demo_ab
//   SELECT A, B FROM demo_ab
// (No WHERE/JOIN yet)
// ------------------------------------
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

// -----------------------------
// 4) App: polls server, runs SQL, posts results back
// -----------------------------
class MainActivity : ComponentActivity() {

    private val client = OkHttpClient()

    private val baseURL = "http://192.168.1.57:8080"

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Register tables
        TableRegistry.register(DemoABTable)
        TableRegistry.register(PhoneTimeTable)

        enableEdgeToEdge()

        setContent {
            Hello2Theme {
                Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                    PollingScreen(
                        url = "$baseURL/next",
                        modifier = Modifier.padding(innerPadding)
                    )
                }
            }
        }
    }

    private suspend fun postResultToServer(number: Int, sql: String, jsonRows: String) {
        withContext(Dispatchers.IO) {
            val payload = JSONObject().apply {
                put("number", number)
                put("device", "android-phone")
                put("sql", sql)
                put("result_json", jsonRows)
            }

            val req = Request.Builder()
                .url("$baseURL/result")
                .post(payload.toString().toRequestBody("application/json".toMediaType()))
                .build()

            client.newCall(req).execute().use { resp ->
                if (!resp.isSuccessful) {
                    throw RuntimeException("POST /result failed: HTTP ${resp.code}")
                }
            }
        }
    }

    @Composable
    fun PollingScreen(url: String, modifier: Modifier = Modifier) {
        var lastText by remember { mutableStateOf("") }
        var lastError by remember { mutableStateOf<String?>(null) }

        LaunchedEffect(url) {
            while (true) {
                try {
                    val body = withContext(Dispatchers.IO) {
                        val req = Request.Builder().url(url).build()
                        client.newCall(req).execute().use { resp ->
                            if (!resp.isSuccessful) throw RuntimeException("HTTP ${resp.code}")
                            resp.body?.string().orEmpty()
                        }
                    }

                    val obj = JSONObject(body)
                    val command = obj.optString("command", "")
                    val message = obj.optString("message", "")
                    val number = obj.optInt("number", -1)

                    lastText = "number=$number\ncommand=$command\nmessage=$message"
                    lastError = null

                    // command == "sql": run SQL query and POST result back to server
                    if (command.equals("sql", ignoreCase = true)) {
                        val parsed = parseSelectSql(message)
                        val allRows = TableRegistry.runTable(parsed.tableName, TableQueryContext())

                        val schemaCols = TableRegistry.getColumns(parsed.tableName).map { it.name }
                        val selectedCols = if (parsed.selectAll) schemaCols else parsed.selectedColumns

                        // Validate selected columns exist (case-sensitive)
                        val schemaSet = schemaCols.toSet()
                        for (c in selectedCols) {
                            require(c in schemaSet) { "Unknown column '$c' for table '${parsed.tableName}'" }
                        }

                        val projected = if (parsed.selectAll) allRows else TableRegistry.projectRows(allRows, selectedCols)
                        val jsonRows = TableRegistry.rowsToJson(projected)

                        // Show result locally
                        lastText = lastText + "\n\nSQL RESULT:\n$jsonRows"
                        Toast.makeText(this@MainActivity, "Ran SQL", Toast.LENGTH_SHORT).show()
                        Log.i("Hello2", "SQL '$message' => $jsonRows")

                        // Post result back to server
                        postResultToServer(number = number, sql = message, jsonRows = jsonRows)
                    }

                    // command == "table": backwards compatibility (runs full table)
                    if (command.equals("table", ignoreCase = true)) {
                        val tableName = message.trim()
                        val rows = TableRegistry.runTable(tableName, TableQueryContext())
                        val jsonRows = TableRegistry.rowsToJson(rows)

                        lastText = lastText + "\n\nTABLE RESULT ($tableName):\n$jsonRows"
                        Toast.makeText(this@MainActivity, "Ran table: $tableName", Toast.LENGTH_SHORT).show()
                        Log.i("Hello2", "Table $tableName => $jsonRows")
                    }

                    // command == "text": display message (your server "Nothing to query")
                    if (command.equals("text", ignoreCase = true)) {
                        // We already show it in lastText; optional toast:
                        // Toast.makeText(this@MainActivity, message, Toast.LENGTH_SHORT).show()
                    }

                } catch (e: Exception) {
                    lastError = e.javaClass.simpleName + ": " + (e.message ?: "")
                }

                delay(3_000)
            }
        }

        Text(
            text = if (lastError != null) "Error: $lastError" else lastText,
            modifier = modifier
        )
    }
}
