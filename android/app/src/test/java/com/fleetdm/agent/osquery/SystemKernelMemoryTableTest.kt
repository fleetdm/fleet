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
class SystemKernelMemoryTableTest {

    @Test
    fun `system_info returns one row with required keys`() = runBlocking {
        val context: Context = RuntimeEnvironment.getApplication()
        OsqueryTables.registerAll(context)

        val rows = OsqueryQueryEngine.execute("SELECT * FROM system_info;")
        assertEquals(1, rows.size)
        val row = rows.first()

        val required = listOf(
            "hostname",
            "computer_name",
            "uuid",
            "hardware_vendor",
            "hardware_model",
            "hardware_version",
            "cpu_brand",
            "physical_memory",
        )
        required.forEach { key -> assertTrue("Missing key $key", row.containsKey(key)) }
        assertTrue(row.getValue("uuid").isNotBlank())
        assertTrue(row.getValue("physical_memory").toLong() >= 0L)
    }

    @Test
    fun `kernel_info returns one row with kernel snapshot fields`() = runBlocking {
        val context: Context = RuntimeEnvironment.getApplication()
        OsqueryTables.registerAll(context)

        val rows = OsqueryQueryEngine.execute("SELECT * FROM kernel_info;")
        assertEquals(1, rows.size)
        val row = rows.first()

        val required = listOf("version", "release", "build", "platform")
        required.forEach { key -> assertTrue("Missing key $key", row.containsKey(key)) }
        assertEquals("android", row.getValue("platform"))
    }

    @Test
    fun `memory_info returns one row with parseable non-negative memory fields`() = runBlocking {
        val context: Context = RuntimeEnvironment.getApplication()
        OsqueryTables.registerAll(context)

        val rows = OsqueryQueryEngine.execute("SELECT * FROM memory_info;")
        assertEquals(1, rows.size)
        val row = rows.first()

        val total = row.getValue("total_bytes").toLong()
        val avail = row.getValue("available_bytes").toLong()
        val threshold = row.getValue("threshold_bytes").toLong()
        val low = row.getValue("low_memory")

        assertTrue(total >= 0L)
        assertTrue(avail >= 0L)
        assertTrue(threshold >= 0L)
        assertTrue(low == "0" || low == "1")
    }
}
