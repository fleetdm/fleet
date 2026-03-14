package com.fleetdm.agent.osquery.tables


import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.util.Calendar

object PhoneTimeTable : TablePlugin {
    override val name: String = "phone_time"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("hour"),
        ColumnDef("minute"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val cal = Calendar.getInstance()
        val hour = cal.get(Calendar.HOUR_OF_DAY)
        val minute = cal.get(Calendar.MINUTE)

        return listOf(
            mapOf(
                "hour" to hour.toString(),
                "minute" to minute.toString(),
            )
        )
    }
}


