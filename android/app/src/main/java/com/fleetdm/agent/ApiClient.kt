package com.fleetdm.agent

import android.content.Context
import android.net.ConnectivityManager
import android.util.Log
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import java.math.BigInteger
import java.net.HttpURLConnection
import java.net.URL
import java.net.UnknownHostException
import java.util.Date
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlinx.coroutines.withContext
import kotlinx.serialization.EncodeDefault
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

    // Retry DNS resolution failures that occur when Android wakes from Doze mode.
    // The active network may be reported as connected before its DNS servers are fully operational.
    // Uses exponential backoff: 1s, 2s, 4s, 8s, 16s, 32s, 64s between attempts (127s total retry window, 8 attempts).
    private const val DNS_MAX_RETRIES = 7
    private const val DNS_INITIAL_RETRY_DELAY_MS = 1000L

    private val json = Json { ignoreUnknownKeys = true }

    private lateinit var dataStore: DataStore<Preferences>
    private lateinit var appContext: Context
    private val API_KEY = stringPreferencesKey("api_key")
    private val SERVER_URL_KEY = stringPreferencesKey("server_url")
    private val ENROLL_SECRET = stringPreferencesKey("enroll_secret")
    private val HARDWARE_UUID = stringPreferencesKey("hardware_uuid")
    private val COMPUTER_NAME = stringPreferencesKey("computer_name")

    private val enrollmentMutex = Mutex()

    fun initialize(context: Context) {
        Log.d(TAG, "initializing api client")
        appContext = context.applicationContext
        if (!::dataStore.isInitialized) {
            dataStore = appContext.prefDataStore
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

    val baseUrlFlow: Flow<String?>
        get() = dataStore.data.map { preferences ->
            preferences[SERVER_URL_KEY]
        }

    suspend fun getApiKey(): String? {
        val encrypted = dataStore.data.first()[API_KEY] ?: return null
        return try {
            KeystoreManager.decrypt(encrypted)
        } catch (e: Exception) {
            FleetLog.e(TAG, "Failed to decrypt API key", e)
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
        require(method != "GET" || body == null) { "GET requests must not include a body" }

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

        for (dnsAttempt in 0..DNS_MAX_RETRIES) {
            var connection: HttpURLConnection? = null
            try {
                if (dnsAttempt > 0) {
                    val delayMs = DNS_INITIAL_RETRY_DELAY_MS shl (dnsAttempt - 1)
                    Log.w(TAG, "retrying $method $endpoint after DNS failure (attempt ${dnsAttempt + 1}, delay ${delayMs}ms)")
                    delay(delayMs)
                }

                connection = openConnectionOnActiveNetwork(url)

                connection.apply {
                    requestMethod = method
                    useCaches = false
                    doInput = true
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

                    if (body != null) {
                        requireNotNull(bodySerializer) { "bodySerializer required when body is provided" }
                        setRequestProperty("Content-Type", "application/json")
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

                Log.d(TAG, "server response from $method $endpoint ($responseCode)")

                return@withContext if (responseCode in 200..299) {
                    val parsed = json.decodeFromString(string = response, deserializer = responseSerializer)
                    Result.success(parsed)
                } else if (responseCode == 401) {
                    Result.failure(UnauthorizedException(response))
                } else if (responseCode == 404) {
                    Result.failure(NotFoundException(response))
                } else {
                    Result.failure(Exception("HTTP $responseCode: $response"))
                }
            } catch (e: UnknownHostException) {
                Log.w(TAG, "DNS resolution failed for $method $endpoint: ${e.message}")
                if (dnsAttempt >= DNS_MAX_RETRIES) {
                    return@withContext Result.failure(e)
                }
            } catch (e: CancellationException) {
                throw e
            } catch (e: Exception) {
                return@withContext Result.failure(e)
            } finally {
                connection?.disconnect()
            }
        }

        // Unreachable: the loop always returns. Required by the compiler since it can't prove the range is non-empty.
        Result.failure(Exception("Exhausted DNS retries for $method $endpoint"))
    }

    /**
     * Exception thrown when the server returns HTTP 401 Unauthorized.
     * This typically indicates the node key has been invalidated (e.g., host was deleted).
     */
    class UnauthorizedException(message: String) : Exception("HTTP 401: $message")
    class NotFoundException(message: String) : Exception("HTTP 404: $message")

    /**
     * Opens an HTTP connection bound to the active network when available. This ensures DNS resolution uses
     * the active network's DNS servers, avoiding failures when Android reports connectivity before DNS is ready.
     * Falls back to a default connection if no active network is available.
     */
    internal fun openConnectionOnActiveNetwork(url: URL): HttpURLConnection {
        if (useActiveNetworkBinding) {
            val connectivityManager = appContext.getSystemService(ConnectivityManager::class.java)
            val activeNetwork = connectivityManager?.activeNetwork
            if (activeNetwork != null) {
                return activeNetwork.openConnection(url) as HttpURLConnection
            }
        }
        return url.openConnection() as HttpURLConnection
    }

    // Disabled in tests where Network.openConnection is not available (Robolectric)
    internal var useActiveNetworkBinding = true

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
        val credentials = getEnrollmentCredentials()
            ?: return@withReenrollOnUnauthorized Result.failure(Exception("enroll credentials not set"))

        makeRequest<Unit, GetCertificateTemplateResponseWrapper>(
            endpoint = "/api/fleetd/certificates/$certificateId",
            method = "GET",
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
                FleetLog.e(TAG, "failed to get certificate template $certificateId")
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
                    FleetLog.e(TAG, "failed to update certificate status $certificateId: ${response.error}")
                    Result.failure(Exception(response.error))
                } else {
                    Log.i(TAG, "successfully updated certificate status for $certificateId to $status")
                    Result.success(Unit)
                }
            },
            onFailure = { throwable ->
                if (throwable is NotFoundException) {
                    // Certificate template was deleted from the server -- nothing to report to
                    Log.i(TAG, "certificate template $certificateId no longer exists on server, nothing to report")
                    Result.success(Unit)
                } else {
                    FleetLog.e(TAG, "failed to update certificate status $certificateId: ${throwable.message}")
                    Result.failure(throwable)
                }
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
                    FleetLog.e(TAG, "Auto-enrollment failed: ${error.message}")
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

// @EncodeDefault is marked @ExperimentalSerializationApi, but it has shipped in kotlinx.serialization since 1.3 (2022)
// and is widely used and reliable in production. The opt-in only acknowledges that the API shape could change in a future version.
@OptIn(kotlinx.serialization.ExperimentalSerializationApi::class)
@Serializable
data class EnrollRequest(
    @SerialName("enroll_secret")
    val enrollSecret: String,
    @SerialName("hardware_uuid")
    val hardwareUUID: String,
    @SerialName("hardware_serial")
    val hardwareSerial: String,
    @EncodeDefault(EncodeDefault.Mode.ALWAYS)
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
