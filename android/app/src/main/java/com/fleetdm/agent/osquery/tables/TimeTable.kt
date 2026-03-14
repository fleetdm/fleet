package com.fleetdm.agent.osquery.tables

import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.util.Calendar
import java.util.Locale
import java.util.TimeZone

class TimeTable : TablePlugin {
    override val name: String = "time"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("weekday"),
        ColumnDef("year"),
        ColumnDef("month"),
        ColumnDef("day"),
        ColumnDef("hour"),
        ColumnDef("minutes"),
        ColumnDef("seconds"),
        ColumnDef("timezone"),
        ColumnDef("local_timezone"),
        ColumnDef("unix_time"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val cal = Calendar.getInstance()
        val tz = TimeZone.getDefault()
        val nowMs = System.currentTimeMillis()

        val weekdayName = cal.getDisplayName(Calendar.DAY_OF_WEEK, Calendar.LONG, Locale.US)
            ?: cal.get(Calendar.DAY_OF_WEEK).toString()

        return listOf(
            mapOf(
                "weekday" to weekdayName,
                "year" to cal.get(Calendar.YEAR).toString(),
                "month" to (cal.get(Calendar.MONTH) + 1).toString(),
                "day" to cal.get(Calendar.DAY_OF_MONTH).toString(),
                "hour" to cal.get(Calendar.HOUR_OF_DAY).toString(),
                "minutes" to cal.get(Calendar.MINUTE).toString(),
                "seconds" to cal.get(Calendar.SECOND).toString(),
                "timezone" to tz.id,
                "local_timezone" to tz.id,
                "unix_time" to (nowMs / 1000L).toString(),
            ),
        )
    }
}
