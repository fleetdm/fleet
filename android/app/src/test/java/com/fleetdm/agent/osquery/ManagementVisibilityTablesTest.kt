package com.fleetdm.agent.osquery

import android.content.Context
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment

@RunWith(RobolectricTestRunner::class)
class ManagementVisibilityTablesTest {

    @Test
    fun `mdm_status returns one row with boolean flags`() = runBlocking {
        val context: Context = RuntimeEnvironment.getApplication()
        OsqueryTables.registerAll(context)

        val rows = OsqueryQueryEngine.execute("SELECT * FROM mdm_status;")
        assertEquals(1, rows.size)
        val row = rows.first()

        listOf(
            "has_device_owner",
            "has_work_profile",
            "restrictions_present",
            "enroll_secret_present",
            "host_uuid_present",
            "server_url_present",
            "is_debug_build",
        ).forEach { key ->
            assertTrue(row[key] == "0" || row[key] == "1")
        }
    }

    @Test
    fun `app_signatures and startup_items execute safely`() = runBlocking {
        val context: Context = RuntimeEnvironment.getApplication()
        OsqueryTables.registerAll(context)

        val signatures = OsqueryQueryEngine.execute("SELECT * FROM app_signatures;")
        if (signatures.isNotEmpty()) {
            assertTrue(signatures.first().containsKey("package_name"))
            assertTrue(signatures.first().containsKey("sha256"))
        }

        val startup = OsqueryQueryEngine.execute("SELECT * FROM startup_items;")
        if (startup.isNotEmpty()) {
            assertTrue(startup.first().containsKey("package_name"))
            assertTrue(startup.first().containsKey("type"))
        }
    }
}
