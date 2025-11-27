package com.fleetdm.agent

import android.content.Context
import android.util.Log
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import java.net.HttpURLConnection
import java.net.URL
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.withContext
import kotlinx.serialization.KSerializer
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

private val Context.credentialStore: DataStore<Preferences> by preferencesDataStore(name = "api_credentials")

object ApiClient {
    private val json = Json { ignoreUnknownKeys = true }

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

    val apiKeyFlow: Flow<String?>
        get() = dataStore.data.map { preferences ->
            preferences[API_KEY]
        }

    val baseUrlFlow: Flow<String?>
        get() = dataStore.data.map { preferences ->
            preferences[BASE_URL_KEY]
        }

    suspend fun getApiKey(): String? = dataStore.data.first()[API_KEY]

    suspend fun getBaseUrl(): String? = dataStore.data.first()[BASE_URL_KEY]

    suspend fun <R, T> makeRequest(
        endpoint: String,
        method: String = "GET",
        body: R? = null,
        authenticated: Boolean = true,
        bodySerializer: KSerializer<R>,
        responseSerializer: KSerializer<T>,
    ): Result<T> = withContext(Dispatchers.IO) {
        var connection: HttpURLConnection? = null
        try {
            val apiKey = getApiKey()
            if (authenticated && apiKey == null) {
                return@withContext Result.failure(
                    Exception("API key not configured"),
                )
            }

            val baseUrl = getBaseUrl() ?: return@withContext Result.failure(
                Exception("Base URL not configured"),
            )

            // Validate base URL format and scheme
            try {
                val parsedUrl = URL(baseUrl)
                if (parsedUrl.protocol !in listOf("https", "http")) {
                    return@withContext Result.failure(
                        Exception("Base URL must use HTTP or HTTPS scheme"),
                    )
                }
            } catch (e: Exception) {
                return@withContext Result.failure(
                    Exception("Invalid base URL format: ${e.message}"),
                )
            }

            val url = URL("$baseUrl$endpoint")
            connection = url.openConnection() as HttpURLConnection

            connection.apply {
                requestMethod = method
                useCaches = false
                doInput = true
                if (authenticated) {
                    setRequestProperty("Authorization", "Bearer $apiKey")
                }
                setRequestProperty("Content-Type", "application/json")
                connectTimeout = 15000
                readTimeout = 15000

                if (body != null && method != "GET") {
                    doOutput = true
                    val bodyJson = json.encodeToString(value = body, serializer = bodySerializer)
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

            if (responseCode in 200..299) {
                val parsed = json.decodeFromString(string = response, deserializer = responseSerializer)
                Result.success(parsed)
            } else {
                Result.failure(Exception("HTTP $responseCode: $response"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        } finally {
            connection?.disconnect()
        }
    }

    suspend fun enroll(baseUrl: String, enrollSecret: String, hardwareUUID: String, computerName: String): Result<EnrollResponse> {
        setBaseUrl(baseUrl)
        val resp = makeRequest(
            endpoint = "/api/fleet/orbit/enroll",
            method = "POST",
            body = EnrollRequest(
                enrollSecret = enrollSecret,
                hardwareUUID = hardwareUUID,
                hardwareSerial = hardwareUUID,
                computerName = computerName,
            ),
            authenticated = false,
            bodySerializer = EnrollRequest.serializer(),
            responseSerializer = EnrollResponse.serializer(),
        )
        resp.onSuccess { value ->
            setApiKey(value.orbitNodeKey)
        }
        resp.onFailure { exception ->
            Log.d("ApiClient.enroll", "Enrollment failed: ${exception.message}")
        }

        return resp
    }
}

@Serializable
data class EnrollRequest(
    @SerialName("enroll_secret")
    val enrollSecret: String,
    @SerialName("hardware_uuid")
    val hardwareUUID: String,
    @SerialName("hardware_serial")
    val hardwareSerial: String,
    @SerialName("platform")
    val platform: String = "android",
    @SerialName("computer_name")
    val computerName: String,
)

@Serializable
data class EnrollResponse(
    @SerialName("orbit_node_key")
    val orbitNodeKey: String,
)
