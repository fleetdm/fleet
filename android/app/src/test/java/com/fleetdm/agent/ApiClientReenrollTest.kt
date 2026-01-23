package com.fleetdm.agent

import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.test.runTest

/**
 * Integration tests for ApiClient 401 re-enrollment logic using MockWebServer.
 * These tests verify the full flow: 401 -> clear key -> re-enroll -> retry with new key.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [33])
class ApiClientReenrollTest {

    private lateinit var context: Context
    private lateinit var mockWebServer: MockWebServer

    private val apiKeyPref = stringPreferencesKey("api_key")
    private val serverUrlPref = stringPreferencesKey("server_url")
    private val enrollSecretPref = stringPreferencesKey("enroll_secret")
    private val hardwareUuidPref = stringPreferencesKey("hardware_uuid")
    private val computerNamePref = stringPreferencesKey("computer_name")

    @Before
    fun setup() = runTest {
        context = RuntimeEnvironment.getApplication()
        mockWebServer = MockWebServer()
        mockWebServer.start()

        ApiClient.initialize(context)
        clearDataStore()

        // Set up enrollment credentials pointing to mock server
        val serverUrl = mockWebServer.url("/").toString().trimEnd('/')
        context.prefDataStore.edit {
            it[serverUrlPref] = serverUrl
            it[enrollSecretPref] = "test-enroll-secret"
            it[hardwareUuidPref] = "test-hardware-uuid"
            it[computerNamePref] = "test-device"
        }
    }

    @After
    fun tearDown() {
        mockWebServer.shutdown()
    }

    private suspend fun clearDataStore() {
        context.prefDataStore.edit { it.clear() }
    }

    private suspend fun setApiKey(key: String) {
        context.prefDataStore.edit {
            it[apiKeyPref] = KeystoreManager.encrypt(key)
        }
    }

    private suspend fun getStoredApiKey(): String? {
        val encrypted = context.prefDataStore.data.first()[apiKeyPref] ?: return null
        return KeystoreManager.decrypt(encrypted)
    }

    @Test
    fun `getOrbitConfig re-enrolls on 401 and retries with new key`() = runTest {
        // Set up old key
        setApiKey("old-node-key")

        // First request returns 401 (old key rejected)
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(401)
                .setBody("""{"error": "invalid node key"}"""),
        )

        // Re-enrollment request succeeds with new key
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setBody("""{"orbit_node_key": "new-node-key"}"""),
        )

        // Retry request succeeds
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setBody("""{"notifications": {}}"""),
        )

        // Act
        val result = ApiClient.getOrbitConfig()

        // Assert: Request succeeded
        assertTrue("Expected success but got: ${result.exceptionOrNull()}", result.isSuccess)

        // Assert: New key is stored
        assertEquals("new-node-key", getStoredApiKey())

        // Assert: Correct requests were made
        assertEquals(3, mockWebServer.requestCount)

        // First request: original config request with old key
        val firstRequest = mockWebServer.takeRequest()
        assertEquals("/api/fleet/orbit/config", firstRequest.path)
        assertTrue(firstRequest.body.readUtf8().contains("old-node-key"))

        // Second request: enrollment
        val enrollRequest = mockWebServer.takeRequest()
        assertEquals("/api/fleet/orbit/enroll", enrollRequest.path)
        assertTrue(enrollRequest.body.readUtf8().contains("test-enroll-secret"))

        // Third request: retry config request with new key
        val retryRequest = mockWebServer.takeRequest()
        assertEquals("/api/fleet/orbit/config", retryRequest.path)
        assertTrue(retryRequest.body.readUtf8().contains("new-node-key"))
    }

    @Test
    fun `getCertificateTemplate re-enrolls on 401 and retries with new key`() = runTest {
        // Set up old key
        setApiKey("old-node-key")

        // First request returns 401
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(401)
                .setBody("""{"error": "invalid node key"}"""),
        )

        // Re-enrollment succeeds
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setBody("""{"orbit_node_key": "new-node-key"}"""),
        )

        // Retry succeeds with certificate template
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setBody(
                    """{
                    "certificate": {
                        "id": 123,
                        "name": "test-cert",
                        "certificate_authority_id": 1,
                        "certificate_authority_name": "TestCA",
                        "created_at": "2025-01-01T00:00:00Z",
                        "subject_name": "CN=test",
                        "certificate_authority_type": "custom_scep_proxy",
                        "status": "delivered"
                    }
                }""",
                ),
        )

        // Act
        val result = ApiClient.getCertificateTemplate(123)

        // Assert
        assertTrue("Expected success but got: ${result.exceptionOrNull()}", result.isSuccess)
        assertEquals("new-node-key", getStoredApiKey())
        assertEquals(3, mockWebServer.requestCount)

        // Verify enrollment was called
        mockWebServer.takeRequest() // First request (401)
        val enrollRequest = mockWebServer.takeRequest()
        assertEquals("/api/fleet/orbit/enroll", enrollRequest.path)
    }

    @Test
    fun `does not re-enroll on non-401 errors`() = runTest {
        setApiKey("test-key")

        // Return 500 error
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(500)
                .setBody("""{"error": "server error"}"""),
        )

        // Act
        val result = ApiClient.getOrbitConfig()

        // Assert: Request failed
        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("500") == true)

        // Assert: Key unchanged, no enrollment
        assertEquals("test-key", getStoredApiKey())
        assertEquals(1, mockWebServer.requestCount)
    }

    @Test
    fun `re-enrollment failure propagates error`() = runTest {
        setApiKey("old-key")

        // First request returns 401
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(401)
                .setBody("""{"error": "invalid node key"}"""),
        )

        // Re-enrollment fails
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(500)
                .setBody("""{"error": "enrollment failed"}"""),
        )

        // Act
        val result = ApiClient.getOrbitConfig()

        // Assert: Request failed due to enrollment failure
        assertTrue(result.isFailure)

        // Assert: Only 2 requests (original + failed enrollment, no retry)
        assertEquals(2, mockWebServer.requestCount)
    }
}
