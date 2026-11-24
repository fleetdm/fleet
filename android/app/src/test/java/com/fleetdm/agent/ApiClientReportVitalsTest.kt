package com.fleetdm.agent

import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import android.content.Context
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import org.json.JSONObject
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

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [33])
class ApiClientReportVitalsTest {

    private lateinit var context: Context
    private lateinit var mockWebServer: MockWebServer

    @Before
    fun setup() = runTest {
        KeystoreManager.enableTestMode()

        context = RuntimeEnvironment.getApplication()
        mockWebServer = MockWebServer()
        mockWebServer.start()

        ApiClient.initialize(context)
        ApiClient.useActiveNetworkBinding = false
        context.prefDataStore.edit { it.clear() }
        context.prefDataStore.edit {
            it[stringPreferencesKey("server_url")] = mockWebServer.url("/").toString().trimEnd('/')
            it[stringPreferencesKey("enroll_secret")] = "test-enroll-secret"
            it[stringPreferencesKey("hardware_uuid")] = "test-hardware-uuid"
            it[stringPreferencesKey("computer_name")] = "test-device"
        }
    }

    @After
    fun tearDown() {
        mockWebServer.shutdown()
        KeystoreManager.disableTestMode()
    }

    private fun enqueueEnrollment(nodeKey: String) {
        mockWebServer.enqueue(MockResponse().setResponseCode(200).setBody("""{"orbit_node_key": "$nodeKey"}"""))
    }

    @Test
    fun `reportVitals sends the vitals with the orbit node key`() = runTest {
        enqueueEnrollment("vitals-node-key")
        mockWebServer.enqueue(MockResponse().setResponseCode(200).setBody("{}"))

        val result = ApiClient.reportVitals(DeviceVitals(roaming = true))
        assertTrue("report should succeed: ${result.exceptionOrNull()}", result.isSuccess)

        assertEquals("/api/fleet/orbit/enroll", mockWebServer.takeRequest().path)
        val vitalsRequest = mockWebServer.takeRequest()
        assertEquals("PUT", vitalsRequest.method)
        assertEquals("/api/fleetd/vitals", vitalsRequest.path)
        assertEquals("Node key vitals-node-key", vitalsRequest.getHeader("Authorization"))
        assertEquals(true, JSONObject(vitalsRequest.body.readUtf8()).getBoolean("roaming"))
    }

    @Test
    fun `reportVitals sends false rather than omitting it`() = runTest {
        enqueueEnrollment("vitals-node-key")
        mockWebServer.enqueue(MockResponse().setResponseCode(200).setBody("{}"))

        assertTrue(ApiClient.reportVitals(DeviceVitals(roaming = false)).isSuccess)

        mockWebServer.takeRequest()
        val body = JSONObject(mockWebServer.takeRequest().body.readUtf8())
        assertTrue("roaming must be present so the server can tell false from unreported", body.has("roaming"))
        assertEquals(false, body.getBoolean("roaming"))
    }

    @Test
    fun `reportVitals leaves out vitals the device couldn't read`() = runTest {
        enqueueEnrollment("vitals-node-key")
        mockWebServer.enqueue(MockResponse().setResponseCode(200).setBody("{}"))

        ApiClient.reportVitals(DeviceVitals(roaming = null))

        mockWebServer.takeRequest()
        assertEquals("a missing vital must be omitted, not sent as null", "{}", mockWebServer.takeRequest().body.readUtf8())
    }

    @Test
    fun `reportVitals fails when the server rejects the report`() = runTest {
        enqueueEnrollment("vitals-node-key")
        mockWebServer.enqueue(MockResponse().setResponseCode(403).setBody("""{"message": "forbidden"}"""))

        assertTrue(ApiClient.reportVitals(DeviceVitals(roaming = true)).isFailure)
    }
}
