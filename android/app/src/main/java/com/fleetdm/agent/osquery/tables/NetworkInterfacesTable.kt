package com.fleetdm.agent.osquery.tables

import android.content.Context
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.net.NetworkInterface
import java.util.Collections

class NetworkInterfacesTable(private val context: Context) : TablePlugin {
    override val name: String = "network_interfaces"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("name"),
        ColumnDef("is_up"),          // true | false
        ColumnDef("mac"),
        ColumnDef("mtu"),
        ColumnDef("is_loopback"),    // true | false
        ColumnDef("is_virtual"),     // true | false
        ColumnDef("addresses"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val rows = mutableListOf<Map<String, String>>()

        val en = try {
            NetworkInterface.getNetworkInterfaces()
        } catch (_: Exception) {
            null
        } ?: return rows

        for (ni in Collections.list(en)) {
            val ifaceName = ni.name ?: continue

            val mac = try {
                val hw = ni.hardwareAddress
                if (hw == null) "" else hw.joinToString(":") { b -> "%02x".format(b) }
            } catch (_: Exception) {
                ""
            }

            val addresses = try {
                Collections.list(ni.inetAddresses)
                    .mapNotNull { it.hostAddress }
                    .joinToString(",")
            } catch (_: Exception) {
                ""
            }

            rows.add(
                mapOf(
                    "name" to ifaceName,
                    "is_up" to ni.isUp.toString(),
                    "mac" to mac,
                    "mtu" to ni.mtu.toString(),
                    "is_loopback" to ni.isLoopback.toString(),
                    "is_virtual" to ni.isVirtual.toString(),
                    "addresses" to addresses,
                )
            )
        }

        return rows
    }
}
