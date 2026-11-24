package com.fleetdm.agent

import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import android.content.Context
import android.content.pm.PackageManager
import android.telephony.TelephonyManager
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.work.ListenableWorker
import androidx.work.testing.TestListenableWorkerBuilder
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config
import kotlinx.coroutines.test.runTest

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [33])
class DeviceVitalsWorkerTest {

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
        // Enroll once up front so each test only queues the vitals response.
        mockWebServer.enqueue(MockResponse().setResponseCode(200).setBody("""{"orbit_node_key": "worker-node-key"}"""))
        mockWebServer.enqueue(MockResponse().setResponseCode(200).setBody("{}"))
        ApiClient.reportVitals(roaming = false)
        mockWebServer.takeRequest()
        mockWebServer.takeRequest()

        shadowOf(context.packageManager).setSystemFeature(PackageManager.FEATURE_TELEPHONY, true)
        shadowOf(context.getSystemService(TelephonyManager::class.java)).setIsNetworkRoaming(true)
    }

    @After
    fun tearDown() {
        mockWebServer.shutdown()
        KeystoreManager.disableTestMode()
    }

    private suspend fun runWorker(runAttemptCount: Int = 0): ListenableWorker.Result =
        TestListenableWorkerBuilder<DeviceVitalsWorker>(context)
            .setRunAttemptCount(runAttemptCount)
            .build()
            .doWork()

    @Test
    fun `reports roaming and succeeds`() = runTest {
        mockWebServer.enqueue(MockResponse().setResponseCode(200).setBody("{}"))

        assertEquals(ListenableWorker.Result.success(), runWorker())
        val request = mockWebServer.takeRequest()
        assertEquals("/api/fleetd/vitals", request.path)
        assertEquals("""{"roaming":true}""", request.body.readUtf8())
    }

    @Test
    fun `skips the report on devices without telephony`() = runTest {
        shadowOf(context.packageManager).setSystemFeature(PackageManager.FEATURE_TELEPHONY, false)

        assertEquals(ListenableWorker.Result.success(), runWorker())
        assertEquals("only the setup enrollment and report reached the server", 2, mockWebServer.requestCount)
    }

    @Test
    fun `does not retry against a server without the vitals endpoint`() = runTest {
        mockWebServer.enqueue(MockResponse().setResponseCode(404).setBody("""{"message": "not found"}"""))

        assertEquals(ListenableWorker.Result.success(), runWorker())
    }

    @Test
    fun `retries a failed report`() = runTest {
        mockWebServer.enqueue(MockResponse().setResponseCode(500).setBody("""{"message": "boom"}"""))

        assertEquals(ListenableWorker.Result.retry(), runWorker())
    }

    @Test
    fun `stops retrying after the last attempt`() = runTest {
        mockWebServer.enqueue(MockResponse().setResponseCode(422).setBody("""{"message": "invalid"}"""))

        assertEquals(ListenableWorker.Result.success(), runWorker(runAttemptCount = 2))
    }
}
