package com.fleetdm.agent

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.withContext
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlinx.serialization.encodeToString
import kotlinx.serialization.decodeFromString
import java.net.HttpURLConnection
import java.net.URL

private val Context.credentialStore: DataStore<Preferences> by preferencesDataStore(name = "api_credentials")

object ApiClient {
    val json = Json { ignoreUnknownKeys = true }

    private lateinit var dataStore: DataStore<Preferences>
    private val API_KEY = stringPreferencesKey("api_key")
    private val BASE_URL_KEY = stringPreferencesKey("base_url")

    fun initialize(context: Context) {
        if (!::dataStore.isInitialized) {
            dataStore = context.applicationContext.credentialStore
        }
    }

    private suspend fun setApiKey(key: String) {
        dataStore.edit { preferences ->
            preferences[API_KEY] = key
        }
    }

    private suspend fun setBaseUrl(url: String) {
        dataStore.edit { preferences ->
            preferences[BASE_URL_KEY] = url
        }
    }

    suspend fun getApiKey(): String? {
        return dataStore.data.first()[API_KEY]
    }

    suspend fun getBaseUrl(): String? {
        return dataStore.data.first()[BASE_URL_KEY]
    }

//    suspend fun makeRequest(
//        endpoint: String,
//        method: String = "GET",
//        body: String? = null
//    ): Result<String> = withContext(Dispatchers.IO) {
//        try {
//            val apiKey = getApiKey() ?: return@withContext Result.failure(
//                Exception("API key not configured")
//            )
//            val baseUrl = getBaseUrl() ?: return@withContext Result.failure(
//                Exception("Base URL not configured")
//            )
//
//            val url = URL("$baseUrl$endpoint")
//            val connection = url.openConnection() as HttpURLConnection
//
//            connection.apply {
//                requestMethod = method
//                setRequestProperty("Authorization", "Bearer $apiKey")
//                setRequestProperty("Content-Type", "application/json")
//                connectTimeout = 15000
//                readTimeout = 15000
//
//                if (body != null && method != "GET") {
//                    doOutput = true
//                    outputStream.use { it.write(body.toByteArray()) }
//                }
//            }
//
//            val responseCode = connection.responseCode
//            val response = if (responseCode in 200..299) {
//                connection.inputStream.bufferedReader().use { it.readText() }
//            } else {
//                connection.errorStream?.bufferedReader()?.use { it.readText() }
//                    ?: "HTTP $responseCode"
//            }
//
//            connection.disconnect()
//
//            if (responseCode in 200..299) {
//                Result.success(response)
//            } else {
//                Result.failure(Exception("HTTP $responseCode: $response"))
//            }
//
//        } catch (e: Exception) {
//            Result.failure(e)
//        }
//    }

    suspend inline fun <reified T> makeRequest(
        endpoint: String,
        method: String = "GET",
        body: Any? = null
    ): Result<T> = withContext(Dispatchers.IO) {
        try {
            val apiKey = getApiKey() ?: return@withContext Result.failure(
                Exception("API key not configured")
            )

            val baseUrl = getBaseUrl() ?: return@withContext Result.failure(
                Exception("Base URL not configured")
            )

            val url = URL("$baseUrl$endpoint")
            val connection = url.openConnection() as HttpURLConnection

            connection.apply {
                requestMethod = method
                setRequestProperty("Authorization", "Bearer $apiKey")
                setRequestProperty("Content-Type", "application/json")
                connectTimeout = 15000
                readTimeout = 15000

                if (body != null && method != "GET") {
                    doOutput = true
                    val bodyJson = json.encodeToString(body)
                    outputStream.use { it.write(bodyJson.toByteArray()) }
                }
            }

            val responseCode = connection.responseCode
            val response = if (responseCode in 200..299) {
                connection.inputStream.bufferedReader().use { it.readText() }
            } else {
                connection.errorStream?.bufferedReader()?.use { it.readText() }
                    ?: "HTTP $responseCode"
            }

            connection.disconnect()

            if (responseCode in 200..299) {
                val parsed = json.decodeFromString<T>(response)
                Result.success(parsed)
            } else {
                Result.failure(Exception("HTTP $responseCode: $response"))
            }

        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun enroll(baseUrl: String, enrollSecret: String, hardwareUUID: String, computerName: String) {
        setBaseUrl(baseUrl)
//
//        makeRequest(
//            endpoint = "/api/fleet/orbit/enroll",
//            method = "POST",
//            body = "{\"enroll_secret\": \"$enrollSecret\", \"hardware_uuid\": \"$hardwareUUID\", \"platform\": \"android\", \"computer_name\": \"$computerName\"}"
//        )
    }
}

@Serializable
data class EnrollRequest(
    @SerialName("enroll_secret")
    val enrollSecret: String,
    @SerialName("hardware_uuid")
    val hardwareUUID: String,
    @SerialName("platform")
    val platform: String = "android",
    @SerialName("computer_name")
    val computerName: String,
)