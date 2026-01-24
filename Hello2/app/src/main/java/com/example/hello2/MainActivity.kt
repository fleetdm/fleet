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
import com.example.hello2.core.TableQueryContext
import com.example.hello2.core.TableRegistry
import com.example.hello2.core.parseSelectSql
import com.example.hello2.ui.theme.Hello2Theme
import com.example.hello2.tables.DemoABTable
import com.example.hello2.tables.PhoneTimeTable
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject

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
                if (!resp.isSuccessful) throw RuntimeException("POST /result failed: HTTP ${resp.code}")
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

                    if (command.equals("sql", ignoreCase = true)) {
                        val parsed = parseSelectSql(message)
                        val allRows = TableRegistry.runTable(parsed.tableName, TableQueryContext())

                        val schemaCols = TableRegistry.getColumns(parsed.tableName).map { it.name }
                        val selectedCols = if (parsed.selectAll) schemaCols else parsed.selectedColumns

                        val schemaSet = schemaCols.toSet()
                        for (c in selectedCols) require(c in schemaSet) {
                            "Unknown column '$c' for table '${parsed.tableName}'"
                        }

                        val projected = if (parsed.selectAll) allRows else TableRegistry.projectRows(allRows, selectedCols)
                        val jsonRows = TableRegistry.rowsToJson(projected)

                        lastText += "\n\nSQL RESULT:\n$jsonRows"
                        Toast.makeText(this@MainActivity, "Ran SQL", Toast.LENGTH_SHORT).show()
                        Log.i("Hello2", "SQL '$message' => $jsonRows")

                        postResultToServer(number = number, sql = message, jsonRows = jsonRows)
                    }

                } catch (e: Exception) {
                    lastError = e.javaClass.simpleName + ": " + (e.message ?: "")
                }

                delay(2_000)
            }
        }

        Text(
            text = if (lastError != null) "Error: $lastError" else lastText,
            modifier = modifier
        )
    }
}
