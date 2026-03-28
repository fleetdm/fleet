package com.fleetdm.agent

import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.annotation.Config
import kotlinx.coroutines.test.runTest

/**
 * Tests that verify GET requests (getCertificateTemplate) are sent correctly compared to POST/PUT requests.
 *
 * Background: getCertificateTemplate is the only GET endpoint in ApiClient. All other endpoints use POST or PUT.
 * We observed that getCertificateTemplate fails with "Unable to resolve host" errors even when POST/PUT requests
 * to the same server succeed. This test verifies the request is properly formed and reaches the server.
 */
@RunWith(RobolectricTestRunner::class)
@Config(sdk = [33])
class ApiClientGetRequestTest {

    private lateinit var context: Context
    private lateinit var mockWebServer: MockWebServer

    private val serverUrlPref = stringPreferencesKey("server_url")
    private val enrollSecretPref = stringPreferencesKey("enroll_secret")
    private val hardwareUuidPref = stringPreferencesKey("hardware_uuid")
    private val computerNamePref = stringPreferencesKey("computer_name")

    @Before
    fun setup() = runTest {
        KeystoreManager.enableTestMode()
        context = RuntimeEnvironment.getApplication()
        mockWebServer = MockWebServer()
        mockWebServer.start()

        ApiClient.initialize(context)

        // Clear and configure DataStore
        context.prefDataStore.edit { it.clear() }
        val serverUrl = mockWebServer.url("/").toString().trimEnd('/')
        context.prefDataStore.edit {
            it[serverUrlPref] = serverUrl
            it[enrollSecretPref] = "test-enroll-secret"
            it[hardwareUuidPref] = "test-hardware-uuid"
            it[computerNamePref] = "test-device"
        }

        // Enroll to establish a node key
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setBody("""{"orbit_node_key": "test-node-key"}"""),
        )
        mockWebServer.enqueue(
            MockResponse()
                .setResponseCode(200)
                .setBody("""{"notifications": {}}"""),
        )
        ApiClient.getOrbitConfig()
        // Drain the setup requests
        mockWebServer.takeRequest() // enroll
        mockWebServer.takeRequest() // config
    }

    @After
    fun tearDown() {
        mockWebServer.shutdown()
        KeystoreManager.disableTestMode()
    }

    private val certificateTemplateResponseBody = """{
        "certificate": {
            "id": 42,
            "name": "test-cert",
            "certificate_authority_id": 1,
            "certificate_authority_name": "TestCA",
            "created_at": "2025-01-01T00:00:00Z",
            "subject_name": "CN=test",
            "certificate_authority_type": "custom_scep_proxy",
            "status": "delivered",
            "fleet_challenge": "test-challenge"
        }
    }"""

    @Test
    fun `getCertificateTemplate sends GET request and receives response`() = runTest {
        mockWebServer.enqueue(MockResponse().setResponseCode(200).setBody(certificateTemplateResponseBody))

        val result = ApiClient.getCertificateTemplate(42)

        assertTrue("Expected success but got: ${result.exceptionOrNull()}", result.isSuccess)
        assertEquals("test-cert", result.getOrThrow().template.name)

        val request = mockWebServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals("/api/fleetd/certificates/42", request.path)
        assertEquals("Node key test-node-key", request.getHeader("Authorization"))
    }

    @Test
    fun `getCertificateTemplate GET request should not send Content-Type header`() = runTest {
        // GET requests have no body, so Content-Type is meaningless and can confuse intermediaries.
        mockWebServer.enqueue(MockResponse().setResponseCode(200).setBody(certificateTemplateResponseBody))

        val result = ApiClient.getCertificateTemplate(42)
        assertTrue("Expected success but got: ${result.exceptionOrNull()}", result.isSuccess)

        val request = mockWebServer.takeRequest()
        assertEquals("GET", request.method)
        // A GET request should not have a Content-Type header since it has no body.
        // Setting Content-Type on a bodyless GET can cause issues with proxies and CDNs.
        assertNull(
            "GET request should not have Content-Type header, but found: ${request.getHeader("Content-Type")}",
            request.getHeader("Content-Type"),
        )
    }

    @Test
    fun `getCertificateTemplate GET request should not send a body`() = runTest {
        mockWebServer.enqueue(MockResponse().setResponseCode(200).setBody(certificateTemplateResponseBody))

        ApiClient.getCertificateTemplate(42)

        val request = mockWebServer.takeRequest()
        assertEquals("GET", request.method)
        assertEquals(
            "GET request should have an empty body",
            0,
            request.bodySize,
        )
    }

    @Test
    fun `POST request succeeds then GET request to same server also succeeds`() = runTest {
        // First: POST request (updateCertificateStatus uses PUT, getOrbitConfig uses POST)
        mockWebServer.enqueue(MockResponse().setResponseCode(200).setBody("""{"notifications": {}}"""))

        val postResult = ApiClient.getOrbitConfig()
        assertTrue("POST should succeed", postResult.isSuccess)

        val postRequest = mockWebServer.takeRequest()
        assertEquals("POST", postRequest.method)
        assertEquals("/api/fleet/orbit/config", postRequest.path)

        // Second: GET request to the same server
        mockWebServer.enqueue(MockResponse().setResponseCode(200).setBody(certificateTemplateResponseBody))

        val getResult = ApiClient.getCertificateTemplate(42)
        assertTrue("GET should also succeed against the same server, but got: ${getResult.exceptionOrNull()}", getResult.isSuccess)

        val getRequest = mockWebServer.takeRequest()
        assertEquals("GET", getRequest.method)
        assertEquals("/api/fleetd/certificates/42", getRequest.path)
    }
}
