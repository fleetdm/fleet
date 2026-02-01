package com.fleetdm.agent.osquery.tables

import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.os.BatteryManager
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext

class BatteryTable(
    private val context: Context,
) : TablePlugin {

    override val name: String = "battery"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("percent_remaining"),
        ColumnDef("charging"),
        ColumnDef("plugged"),
        ColumnDef("health"),
        ColumnDef("status"),
        ColumnDef("technology"),
        ColumnDef("temperature_c"),
        ColumnDef("voltage_mv"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val intent = context.registerReceiver(
            null,
            IntentFilter(Intent.ACTION_BATTERY_CHANGED),
        ) ?: return emptyList()

        val level = intent.getIntExtra(BatteryManager.EXTRA_LEVEL, -1)
        val scale = intent.getIntExtra(BatteryManager.EXTRA_SCALE, -1)
        val percent = if (level >= 0 && scale > 0) {
            (level * 100 / scale).toString()
        } else {
            ""
        }

        val status = intent.getIntExtra(BatteryManager.EXTRA_STATUS, -1)
        val charging = (
            status == BatteryManager.BATTERY_STATUS_CHARGING ||
            status == BatteryManager.BATTERY_STATUS_FULL
        ).toString()

        val plugged = when (intent.getIntExtra(BatteryManager.EXTRA_PLUGGED, -1)) {
            BatteryManager.BATTERY_PLUGGED_AC -> "ac"
            BatteryManager.BATTERY_PLUGGED_USB -> "usb"
            BatteryManager.BATTERY_PLUGGED_WIRELESS -> "wireless"
            else -> "unplugged"
        }

        val health = when (intent.getIntExtra(BatteryManager.EXTRA_HEALTH, -1)) {
            BatteryManager.BATTERY_HEALTH_GOOD -> "good"
            BatteryManager.BATTERY_HEALTH_OVERHEAT -> "overheat"
            BatteryManager.BATTERY_HEALTH_DEAD -> "dead"
            BatteryManager.BATTERY_HEALTH_OVER_VOLTAGE -> "over_voltage"
            BatteryManager.BATTERY_HEALTH_UNSPECIFIED_FAILURE -> "failure"
            BatteryManager.BATTERY_HEALTH_COLD -> "cold"
            else -> "unknown"
        }

        val statusStr = when (status) {
            BatteryManager.BATTERY_STATUS_CHARGING -> "charging"
            BatteryManager.BATTERY_STATUS_DISCHARGING -> "discharging"
            BatteryManager.BATTERY_STATUS_FULL -> "full"
            BatteryManager.BATTERY_STATUS_NOT_CHARGING -> "not_charging"
            else -> "unknown"
        }

        val tempC = intent.getIntExtra(BatteryManager.EXTRA_TEMPERATURE, 0) / 10.0
        val voltage = intent.getIntExtra(BatteryManager.EXTRA_VOLTAGE, 0)

        return listOf(
            mapOf(
                "percent_remaining" to percent,
                "charging" to charging,
                "plugged" to plugged,
                "health" to health,
                "status" to statusStr,
                "technology" to (intent.getStringExtra(BatteryManager.EXTRA_TECHNOLOGY) ?: ""),
                "temperature_c" to tempC.toString(),
                "voltage_mv" to voltage.toString(),
            ),
        )
    }
}
