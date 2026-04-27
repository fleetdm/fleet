package com.fleetdm.agent.osquery.tables

import android.os.SystemClock
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import kotlin.math.max

class UptimeTable : TablePlugin {
    override val name: String = "uptime"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("days"),
        ColumnDef("hours"),
        ColumnDef("minutes"),
        ColumnDef("seconds"),
        ColumnDef("total_seconds"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val totalSeconds = max(0L, SystemClock.elapsedRealtime() / 1000L)

        val days = totalSeconds / 86400
        val hours = (totalSeconds % 86400) / 3600
        val minutes = (totalSeconds % 3600) / 60
        val seconds = totalSeconds % 60

        return listOf(
            mapOf(
                "days" to days.toString(),
                "hours" to hours.toString(),
                "minutes" to minutes.toString(),
                "seconds" to seconds.toString(),
                "total_seconds" to totalSeconds.toString(),
            ),
        )
    }
}
