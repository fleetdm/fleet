package com.fleetdm.agent

import android.os.Build
import androidx.test.ext.junit.runners.AndroidJUnit4
import androidx.test.platform.app.InstrumentationRegistry
import androidx.work.ListenableWorker
import androidx.work.testing.TestListenableWorkerBuilder
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assume.assumeTrue
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import kotlinx.coroutines.runBlocking

/**
 * Enrolls the agent with a real Fleet server and reports vitals through the real telephony and network stack.
 *
 * Run by android/scripts/emulator-vitals-test.sh, which creates the Android host in Fleet and checks the
 * result through the Fleet API. Skipped unless these instrumentation arguments are set:
 * fleetUrl (as the device reaches it), enrollSecret, hostUuid, and optionally expectRoaming.
 *
 * The enrollment it stores is left in place, so the app stays enrolled when launched afterwards.
 */
@RunWith(AndroidJUnit4::class)
class DeviceVitalsEmulatorTest {

    private val args = InstrumentationRegistry.getArguments()
    private val context = InstrumentationRegistry.getInstrumentation().targetContext

    @Before
    fun setup() = runBlocking {
        val fleetUrl = args.getString("fleetUrl")
        val enrollSecret = args.getString("enrollSecret")
        val hostUuid = args.getString("hostUuid")
        assumeTrue(
            "needs a Fleet server; run android/scripts/emulator-vitals-test.sh",
            fleetUrl != null && enrollSecret != null && hostUuid != null,
        )

        // AMAPI delivers these through managed configuration in production.
        ApiClient.initialize(context)
        ApiClient.setEnrollmentCredentials(
            enrollSecret = enrollSecret!!,
            hardwareUUID = hostUuid!!,
            computerName = "${Build.BRAND} ${Build.MODEL}",
            serverUrl = fleetUrl!!,
        )
    }

    @Test
    fun reportsVitalsToFleet() = runBlocking {
        val vitals = DeviceVitals.collect(context)
        assertNotNull("the emulator has telephony, so roaming must be read", vitals.roaming)
        args.getString("expectRoaming")?.let { assertEquals(it.toBoolean(), vitals.roaming) }

        val worker = TestListenableWorkerBuilder<DeviceVitalsWorker>(context).build()
        assertEquals(ListenableWorker.Result.success(), worker.doWork())
        assertNotNull("the report should have enrolled the agent", ApiClient.getApiKey())
    }
}
