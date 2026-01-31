package com.fleetdm.agent

import android.content.Context
import android.util.Log
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import java.math.BigInteger
import java.net.HttpURLConnection
import java.net.URL
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import kotlinx.serialization.KSerializer
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.Transient
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonElement

/**
 * Converts a java.util.Date to ISO8601 format string.
 * Format: "yyyy-MM-dd'T'HH:mm:ss'Z'" (UTC timezone)
 * Example: "2025-12-31T23:59:59Z"
 */
private fun Date.toISO8601String(): String = this.toInstant().toString() // Returns "2025-12-31T23:59:59Z"

val Context.prefDataStore: DataStore<Preferences> by preferencesDataStore(name = "pref_datastore")

/**
 * Result of fetching a certificate template, including the computed SCEP URL.
 */
data class CertificateTemplateResult(val template: GetCertificateTemplateResponse, val scepUrl: String)

/**
 * Interface for certificate-related API operations.
 * Used by CertificateOrchestrator for dependency injection and testability.
 */
interface CertificateApiClient {
    suspend fun getCertificateTemplate(certificateId: Int): Result<CertificateTemplateResult>
    suspend fun updateCertificateStatus(
        certificateId: Int,
        status: UpdateCertificateStatusStatus,
        operationType: UpdateCertificateStatusOperation,
        detail: String? = null,
        notAfter: Date? = null,
        notBefore: Date? = null,
        serialNumber: BigInteger? = null,
    ): Result<Unit>
}

object ApiClient : CertificateApiClient {
    private const val TAG = "fleet-ApiClient"
    private val json = Json { ignoreUnknownKeys = true }

    private lateinit var dataStore: DataStore<Preferences>
    private val API_KEY = stringPreferencesKey("api_key")
    private val SERVER_URL_KEY = stringPreferencesKey("server_url")
    private val ENROLL_SECRET = stringPreferencesKey("enroll_secret")
    private val HARDWARE_UUID = stringPreferencesKey("hardware_uuid")
    private val COMPUTER_NAME = stringPreferencesKey("computer_name")

    private val enrollmentMutex = Mutex()

    fun initialize(context: Context) {
        Log.d(TAG, "initializing api client")
        if (!::dataStore.isInitialized) {
            dataStore = context.applicationContext.prefDataStore
        }
    }

    private suspend fun setApiKey(key: String) {
        dataStore.edit { preferences ->
            preferences[API_KEY] = KeystoreManager.encrypt(key)
        }
    }

    private suspend fun clearApiKey() {
        dataStore.edit { preferences ->
            preferences.remove(API_KEY)
        }
    }
    suspend fun resetEnrollment() {
        clearApiKey()
    }

    val baseUrlFlow: Flow<String?>
        get() = dataStore.data.map { preferences ->
            preferences[SERVER_URL_KEY]
        }

    suspend fun getApiKey(): String? {
        val encrypted = dataStore.data.first()[API_KEY] ?: return null
        return try {
            KeystoreManager.decrypt(encrypted)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to decrypt API key", e)
            null
        }
    }

    suspend fun getBaseUrl(): String? = dataStore.data.first()[SERVER_URL_KEY]

    private suspend fun <R, T> makeRequest(
        endpoint: String,
        method: String = "GET",
        body: R? = null,
        bodySerializer: KSerializer<R>? = null,
        responseSerializer: KSerializer<T>,
        authorized: Boolean = true,
    ): Result<T> = withContext(Dispatchers.IO) {
        var connection: HttpURLConnection? = null
        try {
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




            // DEBUG only: allow self-signed / untrusted certs when FLEET_ALLOW_INSECURE_TLS=true
            if (BuildConfig.FLEET_ALLOW_INSECURE_TLS && connection is javax.net.ssl.HttpsURLConnection) {
                val trustAllCerts = arrayOf<javax.net.ssl.TrustManager>(
                    object : javax.net.ssl.X509TrustManager {
                        override fun checkClientTrusted(chain: Array<java.security.cert.X509Certificate>, authType: String) {}
                        override fun checkServerTrusted(chain: Array<java.security.cert.X509Certificate>, authType: String) {}
                        override fun getAcceptedIssuers(): Array<java.security.cert.X509Certificate> = emptyArray()
                    },
                )

                val sslContext = javax.net.ssl.SSLContext.getInstance("TLS")
                sslContext.init(null, trustAllCerts, java.security.SecureRandom())

                val https = connection as javax.net.ssl.HttpsURLConnection
                https.sslSocketFactory = sslContext.socketFactory
                https.hostnameVerifier = javax.net.ssl.HostnameVerifier { _, _ -> true }
            }



            connection.apply {
                requestMethod = method
                useCaches = false
                doInput = true
                setRequestProperty("Content-Type", "application/json")
                if (authorized) {
                    getNodeKeyOrEnroll().fold(
                        onFailure = { throwable -> return@withContext Result.failure(throwable) },
                        onSuccess = { nodeKey ->
                            setRequestProperty("Authorization", "Node key $nodeKey")
                        },
                    )
                }
                connectTimeout = 15000
                readTimeout = 15000

                if (body != null && method != "GET") {
                    requireNotNull(bodySerializer) { "bodySerializer required when body is provided" }
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

            Log.d(TAG, "server response from $method $endpoint ($responseCode): $response")

            if (responseCode in 200..299) {
                val parsed = json.decodeFromString(string = response, deserializer = responseSerializer)
                Result.success(parsed)
            } else if (responseCode == 401) {
                Result.failure(UnauthorizedException(response))
            } else {
                Result.failure(Exception("HTTP $responseCode: $response"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        } finally {
            connection?.disconnect()
        }
    }

    /**
     * Exception thrown when the server returns HTTP 401 Unauthorized.
     * This typically indicates the node key has been invalidated (e.g., host was deleted).
     */
    class UnauthorizedException(message: String) : Exception("HTTP 401: $message")

    /**
     * Executes a request block with automatic re-enrollment on 401 Unauthorized.
     * If the block returns a 401 failure, clears the stored node key and retries once.
     * On retry, the block is called fresh so it will get a new node key via enrollment.
     */
    private suspend fun <T> withReenrollOnUnauthorized(block: suspend () -> Result<T>): Result<T> {
        val result = block()
        if (result.isFailure && result.exceptionOrNull() is UnauthorizedException) {
            Log.d(TAG, "Received 401, clearing node key and retrying with re-enrollment")
            clearApiKey()
            return block()
        }
        return result
    }

    suspend fun enroll(): Result<EnrollResponse> {
        val credentials = getEnrollmentCredentials()
        credentials ?: return Result.failure(Exception("Credentials not set"))
        val resp = makeRequest(
            endpoint = "/api/fleet/orbit/enroll",
            method = "POST",
            body = EnrollRequest(
                enrollSecret = credentials.enrollSecret,
                hardwareUUID = credentials.hardwareUUID,
                hardwareSerial = credentials.hardwareUUID,
                computerName = credentials.computerName,
            ),
            bodySerializer = EnrollRequest.serializer(),
            responseSerializer = EnrollResponse.serializer(),
            authorized = false,
        )
        resp.onSuccess { value ->
            setApiKey(value.orbitNodeKey)
        }
        resp.onFailure { exception ->
            Log.d(TAG, "Enrollment failed: ${exception.message}")
        }

        return resp
    }

    suspend fun getOrbitConfig(): Result<OrbitConfig> = withReenrollOnUnauthorized {
        val nodeKeyResult = getNodeKeyOrEnroll()

        val orbitNodeKey = nodeKeyResult.getOrElse { error ->
            return@withReenrollOnUnauthorized Result.failure(error)
        }

        makeRequest(
            endpoint = "/api/fleet/orbit/config",
            method = "POST",
            body = GetConfigRequest(orbitNodeKey = orbitNodeKey),
            bodySerializer = GetConfigRequest.serializer(),
            responseSerializer = OrbitConfig.serializer(),
            authorized = false,
        )
    }

    suspend fun setEnrollmentCredentials(enrollSecret: String, hardwareUUID: String, computerName: String, serverUrl: String) {
        dataStore.edit { preferences ->
            preferences[ENROLL_SECRET] = enrollSecret
            preferences[HARDWARE_UUID] = hardwareUUID
            preferences[COMPUTER_NAME] = computerName
            preferences[SERVER_URL_KEY] = serverUrl
        }
    }

    override suspend fun getCertificateTemplate(certificateId: Int): Result<CertificateTemplateResult> = withReenrollOnUnauthorized {
        val nodeKeyResult = getNodeKeyOrEnroll()
        val orbitNodeKey = nodeKeyResult.getOrElse { error ->
            return@withReenrollOnUnauthorized Result.failure(error)
        }

        val credentials = getEnrollmentCredentials()
            ?: return@withReenrollOnUnauthorized Result.failure(Exception("enroll credentials not set"))

        makeRequest(
            endpoint = "/api/fleetd/certificates/$certificateId",
            method = "GET",
            body = GetCertificateTemplateRequest(orbitNodeKey = orbitNodeKey),
            bodySerializer = GetCertificateTemplateRequest.serializer(),
            responseSerializer = GetCertificateTemplateResponseWrapper.serializer(),
        ).fold(
            onSuccess = { wrapper ->
                val template = wrapper.certificate
                Log.i(TAG, "successfully retrieved certificate template ${template.id}: ${template.name}")
                val scepUrl = template.buildScepUrl(
                    serverUrl = credentials.baseUrl,
                    hostUUID = credentials.hardwareUUID,
                )
                Result.success(CertificateTemplateResult(template, scepUrl))
            },
            onFailure = { throwable ->
                Log.e(TAG, "failed to get certificate template $certificateId")
                Result.failure(throwable)
            },
        )
    }

    override suspend fun updateCertificateStatus(
        certificateId: Int,
        status: UpdateCertificateStatusStatus,
        operationType: UpdateCertificateStatusOperation,
        detail: String?,
        notAfter: Date?,
        notBefore: Date?,
        serialNumber: BigInteger?,
    ): Result<Unit> = withReenrollOnUnauthorized {
        makeRequest(
            endpoint = "/api/fleetd/certificates/$certificateId/status",
            method = "PUT",
            body = UpdateCertificateStatusRequest(
                status = status,
                operationType = operationType,
                detail = detail,
                notAfter = notAfter?.toISO8601String(),
                notBefore = notBefore?.toISO8601String(),
                serialNumber = serialNumber?.toString(),
            ),
            bodySerializer = UpdateCertificateStatusRequest.serializer(),
            responseSerializer = UpdateCertificateStatusResponse.serializer(),
        ).fold(
            onSuccess = { response ->
                if (response.error != null) {
                    Log.e(TAG, "failed to update certificate status $certificateId: ${response.error}")
                    Result.failure(Exception(response.error))
                } else {
                    Log.i(TAG, "successfully updated certificate status for $certificateId to $status")
                    Result.success(Unit)
                }
            },
            onFailure = { throwable ->
                Log.e(TAG, "failed to update certificate status $certificateId: ${throwable.message}")
                Result.failure(throwable)
            },
        )
    }

    private suspend fun getEnrollmentCredentials(): EnrollmentCredentials? {
        val prefs = dataStore.data.first()
        val enrollSecret = prefs[ENROLL_SECRET]
        val hardwareUUID = prefs[HARDWARE_UUID]
        val computerName = prefs[COMPUTER_NAME]
        val baseUrl = prefs[SERVER_URL_KEY]

        if (enrollSecret == null || hardwareUUID == null || computerName == null || baseUrl == null) {
            return null
        }

        return EnrollmentCredentials(
            baseUrl = baseUrl,
            enrollSecret = enrollSecret,
            hardwareUUID = hardwareUUID,
            computerName = computerName,
        )
    }

    private suspend fun getNodeKeyOrEnroll(): Result<String> {
        enrollmentMutex.withLock {
            // Check again inside lock in case another coroutine just enrolled
            val existingKey = getApiKey()
            if (existingKey != null) {
                return Result.success(existingKey)
            }

            // Node key is missing, attempt auto-enrollment
            Log.d(TAG, "Orbit node key missing, attempting auto-enrollment")

            // Re-enroll
            val enrollResult = enroll()

            return enrollResult.fold(
                onSuccess = { response ->
                    Log.d(TAG, "Auto-enrollment successful")
                    Result.success(response.orbitNodeKey)
                },
                onFailure = { error ->
                    Log.e(TAG, "Auto-enrollment failed: ${error.message}")
                    Result.failure(error)
                },
            )
        }
    }

    private data class EnrollmentCredentials(
        val baseUrl: String,
        val enrollSecret: String,
        val hardwareUUID: String,
        val computerName: String,
    )
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
    val platform: String = "darwin",
    @SerialName("computer_name")
    val computerName: String,
)

@Serializable
data class EnrollResponse(
    @SerialName("orbit_node_key")
    val orbitNodeKey: String,
)

@Serializable
private data class GetConfigRequest(
    @SerialName("orbit_node_key")
    val orbitNodeKey: String,
)

@Serializable
data class OrbitConfig(
    @SerialName("script_execution_timeout")
    val scriptExecutionTimeout: Int = 0,

    @SerialName("command_line_startup_flags")
    val commandLineStartupFlags: JsonElement? = null,

    @SerialName("extensions")
    val extensions: JsonElement? = null,

    @SerialName("nudge_config")
    val nudgeConfig: JsonElement? = null,

    @SerialName("notifications")
    val notifications: OrbitConfigNotifications = OrbitConfigNotifications(),

    @SerialName("update_channels")
    val updateChannels: OrbitUpdateChannels? = null,
)

@Serializable
data class OrbitConfigNotifications(
    @SerialName("pending_script_execution_ids")
    val pendingScriptExecutionIDs: List<String> = emptyList(),

    @SerialName("pending_software_installer_ids")
    val pendingSoftwareInstallerIDs: List<String> = emptyList(),

    @SerialName("renew_enrollment_profile")
    val renewEnrollmentProfile: Boolean = false,

    @SerialName("rotate_disk_encryption_key")
    val rotateDiskEncryptionKey: Boolean = false,

    @SerialName("needs_mdm_migration")
    val needsMDMMigration: Boolean = false,

    @SerialName("run_setup_experience")
    val runSetupExperience: Boolean = false,

    @SerialName("run_disk_encryption_escrow")
    val runDiskEncryptionEscrow: Boolean = false,

    @SerialName("needs_programmatic_windows_mdm_enrollment")
    val needsProgrammaticWindowsMDMEnrollment: Boolean = false,

    @SerialName("windows_mdm_discovery_endpoint")
    val windowsMDMDiscoveryEndpoint: String = "",

    @SerialName("needs_programmatic_windows_mdm_unenrollment")
    val needsProgrammaticWindowsMDMUnenrollment: Boolean = false,

    @SerialName("enforce_bitlocker_encryption")
    val enforceBitLockerEncryption: Boolean = false,
)

@Serializable
data class OrbitUpdateChannels(
    @SerialName("orbit")
    val orbit: String = "",

    @SerialName("osqueryd")
    val osqueryd: String = "",

    @SerialName("desktop")
    val desktop: String = "",
)

@Serializable
private data class GetCertificateTemplateRequest(
    @SerialName("orbit_node_key")
    val orbitNodeKey: String,
)

@Serializable
data class UpdateCertificateStatusRequest(
    @SerialName("status")
    val status: UpdateCertificateStatusStatus,
    @SerialName("operation_type")
    val operationType: UpdateCertificateStatusOperation,
    @SerialName("detail")
    val detail: String? = null,
    @SerialName("not_valid_after")
    val notAfter: String? = null,
    @SerialName("not_valid_before")
    val notBefore: String? = null,
    @SerialName("serial")
    val serialNumber: String? = null,
)

@Serializable
enum class UpdateCertificateStatusStatus {
    @SerialName("verified")
    VERIFIED,

    @SerialName("failed")
    FAILED,
}

@Serializable
enum class UpdateCertificateStatusOperation {
    @SerialName("install")
    INSTALL,

    @SerialName("remove")
    REMOVE,
}

@Serializable
private data class UpdateCertificateStatusResponse(
    @SerialName("error")
    val error: String? = null,
)

@Serializable
data class GetCertificateTemplateResponseWrapper(
    @SerialName("certificate")
    val certificate: GetCertificateTemplateResponse,
)

@Serializable
data class GetCertificateTemplateResponse(
    // CertificateTemplateResponseSummary
    @SerialName("id")
    val id: Int,

    @SerialName("name")
    val name: String,

    @SerialName("certificate_authority_id")
    val certificateAuthorityId: Int,

    @SerialName("certificate_authority_name")
    val certificateAuthorityName: String,

    @SerialName("created_at")
    val createdAt: String,

    // CertificateTemplateResponseFull
    @SerialName("subject_name")
    val subjectName: String,

    @SerialName("certificate_authority_type")
    val certificateAuthorityType: String,

    @SerialName("status")
    val status: String,

    @SerialName("scep_challenge")
    val scepChallenge: String? = null,

    @SerialName("fleet_challenge")
    val fleetChallenge: String? = null,

    @Transient
    val keyLength: Int = 2048,

    @Transient
    val signatureAlgorithm: String = "SHA256withRSA",
)

/**
 * Builds the SCEP proxy URL for this certificate template.
 */
fun GetCertificateTemplateResponse.buildScepUrl(serverUrl: String, hostUUID: String): String =
    "$serverUrl/mdm/scep/proxy/$hostUUID,g$id,$certificateAuthorityType,${fleetChallenge ?: ""}"
