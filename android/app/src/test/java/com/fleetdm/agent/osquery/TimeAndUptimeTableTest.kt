package com.fleetdm.agent.osquery

import android.content.Context
import com.fleetdm.agent.osquery.core.TableRegistry
import java.util.TimeZone
import kotlinx.coroutines.runBlocking
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import org.robolectric.RobolectricTestRunner
import org.robolectric.RuntimeEnvironment

@RunWith(RobolectricTestRunner::class)
class TimeAndUptimeTableTest {

    @Test
    fun `time table returns one row with required columns and sane values`() = runBlocking {
        val context: Context = RuntimeEnvironment.getApplication()
        OsqueryTables.registerAll(context)

        val rows = OsqueryQueryEngine.execute("SELECT * FROM time;")
        assertEquals(1, rows.size)

        val row = rows[0]
        val required = listOf(
            "weekday",
            "year",
            "month",
            "day",
            "hour",
            "minutes",
            "seconds",
            "timezone",
            "local_timezone",
            "unix_time",
        )
        required.forEach { col -> assertTrue("Missing column $col", row.containsKey(col)) }

        val unixTime = row.getValue("unix_time").toLong()
        val now = System.currentTimeMillis() / 1000L
        assertTrue("unix_time drift too large", kotlin.math.abs(now - unixTime) <= 300L)

        assertTrue(row.getValue("local_timezone").isNotBlank())
        assertEquals(TimeZone.getDefault().id, row.getValue("local_timezone"))
    }

    @Test
    fun `uptime table returns one row with non-negative and consistent values`() = runBlocking {
        val context: Context = RuntimeEnvironment.getApplication()
        OsqueryTables.registerAll(context)

        val rows = OsqueryQueryEngine.execute("SELECT * FROM uptime;")
        assertEquals(1, rows.size)

        val row = rows[0]
        val required = listOf("days", "hours", "minutes", "seconds", "total_seconds")
        required.forEach { col -> assertTrue("Missing column $col", row.containsKey(col)) }

        val days = row.getValue("days").toLong()
        val hours = row.getValue("hours").toLong()
        val minutes = row.getValue("minutes").toLong()
        val seconds = row.getValue("seconds").toLong()
        val totalSeconds = row.getValue("total_seconds").toLong()

        assertTrue(days >= 0)
        assertTrue(hours >= 0)
        assertTrue(minutes >= 0)
        assertTrue(seconds >= 0)
        assertTrue(totalSeconds >= 0)

        val recomposed = days * 86400 + hours * 3600 + minutes * 60 + seconds
        assertTrue("recomposed uptime differs from total_seconds", kotlin.math.abs(recomposed - totalSeconds) <= 1L)
    }
}
