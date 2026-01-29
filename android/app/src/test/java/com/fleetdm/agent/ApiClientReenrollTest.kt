package com.fleetdm.agent

import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
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

    private val serverUrlPref = stringPreferencesKey("server_url")
    private val enrollSecretPref = stringPreferencesKey("enroll_secret")
    private val hardwareUuidPref = stringPreferencesKey("hardware_uuid")
    private val computerNamePref = stringPreferencesKey("computer_name")

    @Before
    fun setup() = runTest {
        // Enable test mode for KeystoreManager to avoid Android Keystore dependency
        KeystoreManager.enableTestMode()

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
        KeystoreManager.disableTestMode()
    }

    private suspend fun clearDataStore() {
        context.prefDataStore.edit { it.clear() }
    }

    private fun enqueueEnrollmentSuccess(nodeKey: String) {
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setBody("""{"orbit_node_key": "$nodeKey"}"""),
        )
    }

    private fun enqueueConfigSuccess() {
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setBody("""{"notifications": {}}"""),
        )
    }

    private fun enqueue401() {
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(401)
                .setBody("""{"error": "invalid node key"}"""),
        )
    }

    @Test
    fun `getOrbitConfig re-enrolls on 401 and retries with new key`() = runTest {
        // First call: no key exists, so enrollment happens, then config succeeds
        enqueueEnrollmentSuccess("first-node-key")
        enqueueConfigSuccess()

        val firstResult = ApiClient.getOrbitConfig()
        assertTrue("First call should succeed", firstResult.isSuccess)
        assertEquals(2, mockWebServer.requestCount) // enroll + config

        // Verify first enrollment used the enroll secret
        val firstEnroll = mockWebServer.takeRequest()
        assertEquals("/api/fleet/orbit/enroll", firstEnroll.path)
        assertTrue(firstEnroll.body.readUtf8().contains("test-enroll-secret"))

        // Verify first config used first-node-key
        val firstConfig = mockWebServer.takeRequest()
        assertEquals("/api/fleet/orbit/config", firstConfig.path)
        assertTrue(firstConfig.body.readUtf8().contains("first-node-key"))

        // Second call: server returns 401 (simulating host deletion), triggering re-enrollment
        enqueue401()
        enqueueEnrollmentSuccess("second-node-key")
        enqueueConfigSuccess()

        val secondResult = ApiClient.getOrbitConfig()
        assertTrue("Second call should succeed after re-enrollment", secondResult.isSuccess)
        assertEquals(5, mockWebServer.requestCount) // +3: config(401) + enroll + config

        // Verify: config with old key returned 401
        val rejectedConfig = mockWebServer.takeRequest()
        assertEquals("/api/fleet/orbit/config", rejectedConfig.path)
        assertTrue(rejectedConfig.body.readUtf8().contains("first-node-key"))

        // Verify: re-enrollment happened
        val reEnroll = mockWebServer.takeRequest()
        assertEquals("/api/fleet/orbit/enroll", reEnroll.path)
        assertTrue(reEnroll.body.readUtf8().contains("test-enroll-secret"))

        // Verify: retry used new key
        val retryConfig = mockWebServer.takeRequest()
        assertEquals("/api/fleet/orbit/config", retryConfig.path)
        assertTrue(retryConfig.body.readUtf8().contains("second-node-key"))
    }

    @Test
    fun `getCertificateTemplate re-enrolls on 401 and retries with new key`() = runTest {
        // First: establish initial enrollment
        enqueueEnrollmentSuccess("first-node-key")
        enqueueConfigSuccess()
        ApiClient.getOrbitConfig()
        mockWebServer.takeRequest() // enroll
        mockWebServer.takeRequest() // config

        // Now test getCertificateTemplate with 401
        enqueue401()
        enqueueEnrollmentSuccess("second-node-key")
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

        val result = ApiClient.getCertificateTemplate(123)

        assertTrue("Expected success but got: ${result.exceptionOrNull()}", result.isSuccess)

        // Verify the flow: cert request (401) -> enroll -> cert request (success)
        val rejectedRequest = mockWebServer.takeRequest()
        assertEquals("/api/fleetd/certificates/123", rejectedRequest.path)
        // GET requests send node key in Authorization header, not body
        assertTrue(
            "Expected first-node-key in Authorization header",
            rejectedRequest.getHeader("Authorization")?.contains("first-node-key") == true,
        )

        val enrollRequest = mockWebServer.takeRequest()
        assertEquals("/api/fleet/orbit/enroll", enrollRequest.path)

        val retryRequest = mockWebServer.takeRequest()
        assertEquals("/api/fleetd/certificates/123", retryRequest.path)
        // Verify retry uses new key in Authorization header
        assertTrue(
            "Expected second-node-key in Authorization header",
            retryRequest.getHeader("Authorization")?.contains("second-node-key") == true,
        )
    }

    @Test
    fun `does not re-enroll on non-401 errors`() = runTest {
        // Establish initial enrollment
        enqueueEnrollmentSuccess("test-key")
        enqueueConfigSuccess()
        ApiClient.getOrbitConfig()
        val initialRequestCount = mockWebServer.requestCount

        // Clear recorded requests
        repeat(initialRequestCount) { mockWebServer.takeRequest() }

        // Return 500 error
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(500)
                .setBody("""{"error": "server error"}"""),
        )

        val result = ApiClient.getOrbitConfig()

        assertTrue(result.isFailure)
        assertTrue(result.exceptionOrNull()?.message?.contains("500") == true)

        // Only 1 request - no re-enrollment attempt
        assertEquals(1, mockWebServer.requestCount - initialRequestCount)
    }

    @Test
    fun `re-enrollment failure propagates error`() = runTest {
        // Establish initial enrollment
        enqueueEnrollmentSuccess("old-key")
        enqueueConfigSuccess()
        ApiClient.getOrbitConfig()
        val initialRequestCount = mockWebServer.requestCount

        // Config returns 401, then re-enrollment fails
        enqueue401()
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(500)
                .setBody("""{"error": "enrollment failed"}"""),
        )

        val result = ApiClient.getOrbitConfig()

        assertTrue(result.isFailure)

        // 2 requests: config(401) + failed enrollment (no retry after failed enrollment)
        assertEquals(2, mockWebServer.requestCount - initialRequestCount)
    }
}
