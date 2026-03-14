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
class AdditionalParityTablesTest {

    @Test
    fun `users table returns one row with parseable uid`() = runBlocking {
        val context: Context = RuntimeEnvironment.getApplication()
        OsqueryTables.registerAll(context)

        val rows = OsqueryQueryEngine.execute("SELECT * FROM users;")
        assertEquals(1, rows.size)
        val row = rows.first()
        assertTrue(row.containsKey("uid"))
        assertTrue(row.getValue("uid").toLong() >= 0L)
        assertTrue(row.getValue("username").isNotBlank())
    }

    @Test
    fun `cpu_info table returns one row with non-negative cores`() = runBlocking {
        val context: Context = RuntimeEnvironment.getApplication()
        OsqueryTables.registerAll(context)

        val rows = OsqueryQueryEngine.execute("SELECT * FROM cpu_info;")
        assertEquals(1, rows.size)
        val row = rows.first()
        assertTrue(row.getValue("cores").toLong() >= 0L)
        assertTrue(row.containsKey("arch"))
        assertTrue(row.containsKey("model"))
    }

    @Test
    fun `processes and network style tables execute without crash`() = runBlocking {
        val context: Context = RuntimeEnvironment.getApplication()
        OsqueryTables.registerAll(context)

        val processes = OsqueryQueryEngine.execute("SELECT * FROM processes;")
        if (processes.isNotEmpty()) {
            assertTrue(processes.first().containsKey("pid"))
            assertTrue(processes.first().containsKey("name"))
        }

        val interfaceAddresses = OsqueryQueryEngine.execute("SELECT * FROM interface_addresses;")
        if (interfaceAddresses.isNotEmpty()) {
            assertTrue(interfaceAddresses.first().containsKey("interface"))
            assertTrue(interfaceAddresses.first().containsKey("address"))
        }

        val routes = OsqueryQueryEngine.execute("SELECT * FROM routes;")
        if (routes.isNotEmpty()) {
            assertTrue(routes.first().containsKey("destination"))
            assertTrue(routes.first().containsKey("raw"))
        }

        val mounts = OsqueryQueryEngine.execute("SELECT * FROM mounts;")
        if (mounts.isNotEmpty()) {
            assertTrue(mounts.first().containsKey("path"))
            assertTrue(mounts.first().containsKey("type"))
        }
    }
}
