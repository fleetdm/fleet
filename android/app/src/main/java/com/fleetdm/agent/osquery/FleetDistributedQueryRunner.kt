package com.fleetdm.agent.osquery

import android.content.Context
import android.util.Log
import com.fleetdm.agent.osquery.core.TableQueryContext
import com.fleetdm.agent.osquery.core.TableRegistry
import com.fleetdm.agent.osquery.core.parseSelectSql
import com.fleetdm.agent.osquery.core.WhereCond
import com.fleetdm.agent.osquery.core.WhereOp
import com.fleetdm.agent.BuildConfig
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


object FleetDistributedQueryRunner {

    private const val tag = "FleetOsquery"

    // DEV ONLY defaults. We will wire these to Fleet settings later.

    val fleetBaseUrl: String = BuildConfig.FLEET_BASE_URL
    val nodeKey: String = BuildConfig.FLEET_NODE_KEY


    // If true, when we cannot run a query, we still answer it with [] so Fleet stops sending it.
    var clearUnknownQueries: Boolean = true

    // Dev only: accept self signed TLS certs
    private val client: OkHttpClient = unsafeOkHttpClient()

    suspend fun runForever(context: Context) {
        withContext(Dispatchers.Default) {
            Log.i(tag, "Starting distributed query loop")
            require(fleetBaseUrl.isNotBlank()) { "FLEET_BASE_URL is empty. Set it in android/config.properties" }
            require(nodeKey.isNotBlank()) { "FLEET_NODE_KEY is empty. Set it in android/config.properties" }

            while (true) {
                try {
                    val readResp = fleetDistributedRead(nodeKey)

                    val queriesObj = readResp.optJSONObject("queries") ?: JSONObject()
                    val queryNames = queriesObj.keys().asSequence().toList().sorted()

                    val resultsToWrite = linkedMapOf<String, List<Map<String, String>>>()

                    for (qName in queryNames) {
                        val sql = queriesObj.optString(qName, "")
                        if (sql.isBlank()) continue

                        try {
                            val rows = executeSqlViaTables(sql)
                            resultsToWrite[qName] = rows
                        } catch (e: Exception) {
                            val msg = e.message ?: e.javaClass.simpleName
                            Log.w(tag, "Query failed: $qName sql=$sql err=$msg")

                            if (clearUnknownQueries) {
                                resultsToWrite[qName] = emptyList()
                            }
                        }
                    }

                    if (resultsToWrite.isNotEmpty()) {
                        fleetDistributedWrite(nodeKey, resultsToWrite)
                    }
                } catch (e: Exception) {
                    Log.e(tag, "Loop error", e)
                }

                delay(5_000)
            }
        }
    }

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
            if (!resp.isSuccessful) throw RuntimeException("distributed/read HTTP ${resp.code}: $body")
            JSONObject(body)
        }
    }

    private suspend fun fleetDistributedWrite(
        nodeKey: String,
        queryResults: Map<String, List<Map<String, String>>>
    ): JSONObject = withContext(Dispatchers.IO) {

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
            if (!resp.isSuccessful) throw RuntimeException("distributed/write HTTP ${resp.code}: $body")
            if (body.isBlank()) JSONObject() else JSONObject(body)
        }
    }

    private suspend fun executeSqlViaTables(sql: String): List<Map<String, String>> {
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
            throw IllegalArgumentException("Unknown columns for table '${parsed.tableName}': ${missing.joinToString(", ")}")
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
