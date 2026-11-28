package com.fleetdm.agent

import android.content.Context
import android.content.pm.PackageManager
import android.telephony.ServiceState
import android.telephony.TelephonyManager
import androidx.work.WorkManager
import androidx.work.testing.WorkManagerTestInitHelper
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config
import org.robolectric.shadows.ShadowLooper
import java.util.UUID

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [33])
class RoamingChangeCallbackTest {

    private lateinit var context: Context

    @Before
    fun setup() {
        context = RuntimeEnvironment.getApplication()
        WorkManagerTestInitHelper.initializeTestWorkManager(context)
        shadowOf(context.packageManager).setSystemFeature(PackageManager.FEATURE_TELEPHONY, true)
    }

    private fun setRoaming(roaming: Boolean) {
        val state = ServiceState().apply { this.roaming = roaming }
        shadowOf(context.getSystemService(TelephonyManager::class.java)).setServiceState(state)
        ShadowLooper.idleMainLooper()
    }

    // Each enqueue replaces the pending report with a new one, under a new ID.
    private fun pendingReportId(): UUID? = WorkManager.getInstance(context)
        .getWorkInfosForUniqueWork(DeviceVitalsWorker.ON_CHANGE_WORK_NAME).get()
        .lastOrNull { !it.state.isFinished }?.id

    @Test
    fun `reports each time roaming changes`() {
        DeviceVitalsWorker.registerRoamingChangeCallback(context)
        setRoaming(false)
        val home = pendingReportId()

        setRoaming(true)
        val roaming = pendingReportId()
        assertNotNull(roaming)
        assertNotEquals(home, roaming)

        setRoaming(false)
        assertNotEquals(roaming, pendingReportId())
    }

    @Test
    fun `does not report when the service state changes but roaming does not`() {
        DeviceVitalsWorker.registerRoamingChangeCallback(context)
        setRoaming(true)
        val before = pendingReportId()

        setRoaming(true)
        assertEquals(before, pendingReportId())
    }

    @Test
    fun `does nothing on devices without telephony`() {
        shadowOf(context.packageManager).setSystemFeature(PackageManager.FEATURE_TELEPHONY, false)
        DeviceVitalsWorker.registerRoamingChangeCallback(context)

        setRoaming(true)
        setRoaming(false)
        assertNull(pendingReportId())
    }
}
