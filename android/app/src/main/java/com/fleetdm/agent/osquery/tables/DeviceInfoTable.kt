package com.fleetdm.agent.osquery.tables

import android.os.Build
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.util.Locale

class DeviceInfoTable : TablePlugin {
    override val name: String = "device_info"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("device"),
        ColumnDef("model"),
        ColumnDef("manufacturer"),
        ColumnDef("brand"),
        ColumnDef("product"),
        ColumnDef("hardware"),
        ColumnDef("board"),
        ColumnDef("fingerprint"),
        ColumnDef("bootloader"),
        ColumnDef("tags"),
        ColumnDef("type"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        fun s(v: String?): String = (v ?: "").trim()

        return listOf(
            mapOf(
                "device" to s(Build.DEVICE),
                "model" to s(Build.MODEL),
                "manufacturer" to s(Build.MANUFACTURER),
                "brand" to s(Build.BRAND),
                "product" to s(Build.PRODUCT),
                "hardware" to s(Build.HARDWARE),
                "board" to s(Build.BOARD),
                "fingerprint" to s(Build.FINGERPRINT),
                "bootloader" to s(Build.BOOTLOADER),
                "tags" to s(Build.TAGS).lowercase(Locale.ROOT),
                "type" to s(Build.TYPE).lowercase(Locale.ROOT),
            ),
        )
    }
}
