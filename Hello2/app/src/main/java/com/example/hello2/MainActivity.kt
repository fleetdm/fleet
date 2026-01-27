package com.example.hello2

import android.os.Bundle
import android.util.Log
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
import com.example.hello2.tables.DemoABTable
import com.example.hello2.tables.InstalledAppsTable
import com.example.hello2.tables.PhoneTimeTable
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
import java.security.SecureRandom
import java.security.cert.X509Certificate
import javax.net.ssl.HostnameVerifier
import javax.net.ssl.SSLContext
import javax.net.ssl.SSLSession
import javax.net.ssl.TrustManager
import javax.net.ssl.X509TrustManager

class MainActivity : ComponentActivity() {

    /**
     * IMPORTANT:
     * - Emulator: https://10.0.2.2:8080
     * - Real phone: https://<YOUR_MAC_LAN_IP>:8080
     */
    private val fleetBaseUrl = "https://192.168.1.2:8080" // <-- set to your Fleet host reachable from this device

    // Already enrolled node_key — DO NOT enroll again
    private val nodeKey = "W0s/1eKzJwTWyaxy6C6p9p84jtYmuF24"

    // Dev only: accept self-signed TLS certs
    private val client: OkHttpClient = unsafeOkHttpClient()

    // If true: when we can't run a query, we still "answer" it with [] so Fleet stops sending it.
    private val clearUnknownQueries = true

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        // Register existing table plugins (keep table logic out of MainActivity)
        TableRegistry.register(PhoneTimeTable)
        TableRegistry.register(DemoABTable)
        TableRegistry.register(InstalledAppsTable(this))

        setContent {
            Hello2Theme {
                Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                    FleetPollExecuteWriteScreen(modifier = Modifier.padding(innerPadding))
                }
            }
        }
    }

    // ---------------------------
    // Fleet I/O
    // ---------------------------

    private suspend fun fleetDistributedRead(nodeKey: String): JSONObject = withContext(Dispatchers.IO) {
        val payload = JSONObject().apply {
            put("node_key", nodeKey)
            put("queries", JSONObject())
        }

        val req = Request.Builder()
            .url("$fleetBaseUrl/api/v1/osquery/distributed/read")
            .post(payload.toString().toRequestBody("application/json".toMediaType()))
            .build()

        client.newCall(req).execute().use { resp ->
            val body = resp.body?.string().orEmpty()
            if (!resp.isSuccessful) throw RuntimeException("Fleet distributed/read failed HTTP ${resp.code}: $body")
            JSONObject(body)
        }
    }

    private suspend fun fleetDistributedWrite(nodeKey: String, queryResults: Map<String, List<Map<String, String>>>): JSONObject =
        withContext(Dispatchers.IO) {

            val queriesObj = JSONObject()
            for ((queryName, rows) in queryResults) {
                val arr = JSONArray()
                for (row in rows) {
                    val o = JSONObject()
                    for ((k, v) in row) o.put(k, v)
                    arr.put(o)
                }
                queriesObj.put(queryName, arr)
            }

            val payload = JSONObject().apply {
                put("node_key", nodeKey)
                put("queries", queriesObj)
            }

            val req = Request.Builder()
                .url("$fleetBaseUrl/api/v1/osquery/distributed/write")
                .post(payload.toString().toRequestBody("application/json".toMediaType()))
                .build()

            client.newCall(req).execute().use { resp ->
                val body = resp.body?.string().orEmpty()
                if (!resp.isSuccessful) throw RuntimeException("Fleet distributed/write failed HTTP ${resp.code}: $body")
                if (body.isBlank()) JSONObject() else JSONObject(body)
            }
        }

    // ---------------------------
    // Execute using your core + tables
    // ---------------------------

    private suspend fun executeSqlViaTables(sql: String): List<Map<String, String>> {
        val parsed = parseSelectSql(sql) // supports: SELECT <cols|*> FROM <table>
        val fullRows = TableRegistry.runTable(parsed.tableName, TableQueryContext())

        if (parsed.selectAll) return fullRows

        // Validate requested columns exist and project
        val schemaCols = TableRegistry.getColumns(parsed.tableName).map { it.name }
        val schemaLowerToReal = schemaCols.associateBy { it.lowercase() }

        val selectedRealCols = parsed.selectedColumns.mapNotNull { sel ->
            schemaLowerToReal[sel.lowercase()]
        }

        if (selectedRealCols.size != parsed.selectedColumns.size) {
            val missing = parsed.selectedColumns.filter { schemaLowerToReal[it.lowercase()] == null }
            throw IllegalArgumentException("Unknown columns for table '${parsed.tableName}': ${missing.joinToString(", ")}")
        }

        return TableRegistry.projectRows(fullRows, selectedRealCols)
    }

    // ---------------------------
    // UI / Main loop
    // ---------------------------

    @Composable
    fun FleetPollExecuteWriteScreen(modifier: Modifier = Modifier) {
        var statusText by remember { mutableStateOf("Starting...") }
        var lastError by remember { mutableStateOf<String?>(null) }

        LaunchedEffect(Unit) {
            try {
                statusText = "Using existing node_key\nPolling + executing + writing to Fleet...\n\n$nodeKey"
                lastError = null

                while (true) {
                    val readResp = fleetDistributedRead(nodeKey)

                    val queriesObj = readResp.optJSONObject("queries") ?: JSONObject()
                    val discoveryObj = readResp.optJSONObject("discovery") ?: JSONObject()
                    val queryNames = queriesObj.keys().asSequence().toList().sorted()

                    val resultsToWrite = linkedMapOf<String, List<Map<String, String>>>()

                    val sb = StringBuilder()
                    sb.append("Fleet poll OK\n")
                    sb.append("queries=").append(queryNames.size).append("\n\n")

                    if (queryNames.isEmpty()) {
                        sb.append("No queries.\n")
                    } else {
                        for (qName in queryNames) {
                            val sql = queriesObj.optString(qName, "")
                            val discovery = discoveryObj.optString(qName, "")

                            sb.append("• ").append(qName).append("\n")
                            sb.append("  SQL: ").append(sql).append("\n")
                            if (discovery.isNotBlank()) sb.append("  discovery: ").append(discovery).append("\n")

                            try {
                                val rows = executeSqlViaTables(sql)
                                resultsToWrite[qName] = rows

                                sb.append("  => RUN ok, rows=").append(rows.size).append("\n")
                                rows.take(2).forEachIndexed { i, row ->
                                    sb.append("    [").append(i).append("] ").append(row).append("\n")
                                }
                                if (rows.size > 2) sb.append("    ...\n")
                            } catch (e: Exception) {
                                val msg = e.message ?: e.javaClass.simpleName
                                sb.append("  => ").append(if (clearUnknownQueries) "CLEAR" else "SKIP").append(": ").append(msg).append("\n")

                                if (clearUnknownQueries) {
                                    // Answer with [] to stop Fleet from repeatedly sending it.
                                    resultsToWrite[qName] = emptyList()
                                }
                            }

                            sb.append("\n")
                        }

                        if (resultsToWrite.isNotEmpty()) {
                            val writeResp = fleetDistributedWrite(nodeKey, resultsToWrite)
                            sb.append("WRITE => ").append(writeResp.toString()).append("\n")
                        } else {
                            sb.append("WRITE => nothing to send\n")
                        }
                    }

                    val accelerate = readResp.optInt("accelerate", -1)
                    if (accelerate >= 0) sb.append("accelerate=").append(accelerate).append("\n")

                    statusText = sb.toString()
                    Log.i("Hello2", "Fleet distributed/read => ${readResp.toString(2)}")

                    delay(5_000)
                }
            } catch (e: Exception) {
                lastError = "${e.javaClass.simpleName}: ${e.message ?: ""}"
                Log.e("Hello2", "Fleet loop error", e)
            }
        }

        Text(
            text = if (lastError != null) "Error: $lastError\n\n$statusText" else statusText,
            modifier = modifier
        )
    }

    // ---------------------------
    // Unsafe HTTPS (DEV ONLY)
    // ---------------------------

    private fun unsafeOkHttpClient(): OkHttpClient {
        val trustAllCerts = arrayOf<TrustManager>(
            object : X509TrustManager {
                override fun checkClientTrusted(chain: Array<X509Certificate>, authType: String) {}
                override fun checkServerTrusted(chain: Array<X509Certificate>, authType: String) {}
                override fun getAcceptedIssuers(): Array<X509Certificate> = arrayOf()
            }
        )

        val sslContext = SSLContext.getInstance("TLS")
        sslContext.init(null, trustAllCerts, SecureRandom())

        val trustManager = trustAllCerts[0] as X509TrustManager
        val hostnameVerifier = HostnameVerifier { _: String?, _: SSLSession? -> true }

        return OkHttpClient.Builder()
            .sslSocketFactory(sslContext.socketFactory, trustManager)
            .hostnameVerifier(hostnameVerifier)
            .build()
    }
}
