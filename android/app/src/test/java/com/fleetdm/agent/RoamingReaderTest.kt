package com.fleetdm.agent

import android.content.Context
import android.content.pm.PackageManager
import android.telephony.TelephonyManager
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Before
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment
import org.robolectric.Shadows.shadowOf
import org.robolectric.annotation.Config

@RunWith(RobolectricTestRunner::class)
@Config(sdk = [33])
class RoamingReaderTest {

    private lateinit var context: Context

    @Before
    fun setup() {
        context = RuntimeEnvironment.getApplication()
    }

    private fun setTelephony(available: Boolean) {
        shadowOf(context.packageManager).setSystemFeature(PackageManager.FEATURE_TELEPHONY, available)
    }

    private fun setNetworkRoaming(roaming: Boolean) {
        shadowOf(context.getSystemService(TelephonyManager::class.java)).setIsNetworkRoaming(roaming)
    }

    @Test
    fun `reports roaming when the radio is on a roaming network`() {
        setTelephony(true)
        setNetworkRoaming(true)
        assertEquals(true, RoamingReader.read(context))
    }

    @Test
    fun `reports not roaming on the home network`() {
        setTelephony(true)
        setNetworkRoaming(false)
        assertEquals(false, RoamingReader.read(context))
    }

    @Test
    fun `returns null on devices without telephony`() {
        setTelephony(false)
        setNetworkRoaming(true)
        assertNull(RoamingReader.read(context))
    }
}
