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
import org.json.JSONObject
import java.security.SecureRandom
import java.security.cert.X509Certificate
import javax.net.ssl.*

class MainActivity : ComponentActivity() {

    /**
     * IMPORTANT:
     * - Emulator: https://10.0.2.2:8080
     * - Real phone: https://<YOUR_MAC_LAN_IP>:8080
     * - https://localhost:8080 on Android points to the phone itself.
     */
    private val fleetBaseUrl = "https://192.168.1.2:8080"


    // Already enrolled node_key — DO NOT enroll again
    private val nodeKey = "W0s/1eKzJwTWyaxy6C6p9p84jtYmuF24"

    // Dev only: accept self-signed TLS certs
    private val client: OkHttpClient = unsafeOkHttpClient()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()

        // Register your table plugins (this is the right place)
        TableRegistry.register(PhoneTimeTable)
        TableRegistry.register(DemoABTable)
        TableRegistry.register(InstalledAppsTable(this))

        setContent {
            Hello2Theme {
                Scaffold(modifier = Modifier.fillMaxSize()) { innerPadding ->
                    FleetPollingAndInterpretScreen(modifier = Modifier.padding(innerPadding))
                }
            }
        }
    }

    // ---------------------------
    // Fleet distributed/read
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
            if (!resp.isSuccessful) {
                throw RuntimeException("Fleet distributed/read failed HTTP ${resp.code}: $body")
            }
            JSONObject(body)
        }
    }

    // ---------------------------
    // Execution helpers
    // ---------------------------

    /**
     * Fleet SQL -> your parser -> TableRegistry execution
     * Returns rows (already projected to selected columns if needed).
     */
    private suspend fun executeFleetSql(sql: String): List<Map<String, String>> {
        val parsed = parseSelectSql(sql) // throws if not SELECT <cols|*> FROM <table>
        val tableName = parsed.tableName

        // runTable returns full schema rows (with missing cols filled)
        val fullRows = TableRegistry.runTable(tableName, TableQueryContext())

        if (parsed.selectAll) return fullRows

        // Project requested columns, but match case-insensitively against the real schema.
        val schemaCols = TableRegistry.getColumns(tableName).map { it.name }
        val schemaMapLowerToReal = schemaCols.associateBy { it.lowercase() }

        val selectedRealCols = parsed.selectedColumns.mapNotNull { sel ->
            schemaMapLowerToReal[sel.lowercase()]
        }

        // If user asked for columns that don't exist, treat as error.
        if (selectedRealCols.size != parsed.selectedColumns.size) {
            val missing = parsed.selectedColumns.filter { schemaMapLowerToReal[it.lowercase()] == null }
            throw IllegalArgumentException("Unknown columns for table '$tableName': ${missing.joinToString(", ")}")
        }

        return TableRegistry.projectRows(fullRows, selectedRealCols)
    }

    // ---------------------------
    // UI / Poll Loop
    // ---------------------------
    @Composable
    fun FleetPollingAndInterpretScreen(modifier: Modifier = Modifier) {
        var statusText by remember { mutableStateOf("Starting...") }
        var lastError by remember { mutableStateOf<String?>(null) }

        LaunchedEffect(Unit) {
            try {
                statusText = "Using existing node_key\nPolling Fleet...\n\n$nodeKey"
                lastError = null

                while (true) {
                    val resp = fleetDistributedRead(nodeKey)

                    val queriesObj = resp.optJSONObject("queries") ?: JSONObject()
                    val discoveryObj = resp.optJSONObject("discovery") ?: JSONObject()
                    val names = queriesObj.keys().asSequence().toList().sorted()

                    val sb = StringBuilder()
                    sb.append("Fleet poll OK\n")
                    sb.append("queries=").append(names.size).append("\n\n")

                    if (names.isEmpty()) {
                        sb.append("No queries.\n")
                    } else {
                        for (name in names) {
                            val sql = queriesObj.optString(name, "")
                            val discovery = discoveryObj.optString(name, "")

                            sb.append("• ").append(name).append("\n")
                            sb.append("  SQL: ").append(sql).append("\n")
                            if (discovery.isNotBlank()) sb.append("  discovery: ").append(discovery).append("\n")

                            try {
                                val rows = executeFleetSql(sql)
                                sb.append("  => RUN ok\n")
                                sb.append("  rows=").append(rows.size).append("\n")

                                // preview first 2 rows
                                rows.take(2).forEachIndexed { i, row ->
                                    sb.append("    [").append(i).append("] ").append(row).append("\n")
                                }
                                if (rows.size > 2) sb.append("    ...\n")
                            } catch (e: Exception) {
                                // If table not registered / SQL not supported / bad cols etc.
                                sb.append("  => SKIP/ERR: ").append(e.message ?: e.javaClass.simpleName).append("\n")
                            }

                            sb.append("\n")
                        }
                    }

                    val accelerate = resp.optInt("accelerate", -1)
                    if (accelerate >= 0) sb.append("accelerate=").append(accelerate).append("\n")

                    statusText = sb.toString()
                    Log.i("Hello2", "Fleet distributed/read => ${resp.toString(2)}")

                    delay(5_000)
                }

            } catch (e: Exception) {
                lastError = "${e.javaClass.simpleName}: ${e.message ?: ""}"
                Log.e("Hello2", "Polling error", e)
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
