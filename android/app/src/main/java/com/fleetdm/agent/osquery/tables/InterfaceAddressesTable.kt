package com.fleetdm.agent.osquery.tables

import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.net.NetworkInterface

class InterfaceAddressesTable : TablePlugin {
    override val name: String = "interface_addresses"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("interface"),
        ColumnDef("address"),
        ColumnDef("family"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        return runCatching {
            val rows = mutableListOf<Map<String, String>>()
            val interfaces = NetworkInterface.getNetworkInterfaces() ?: return@runCatching emptyList<Map<String, String>>()
            while (interfaces.hasMoreElements()) {
                val ni = interfaces.nextElement()
                val addrs = ni.inetAddresses
                while (addrs.hasMoreElements()) {
                    val addr = addrs.nextElement()
                    val family = if (addr.hostAddress?.contains(":") == true) "ipv6" else "ipv4"
                    rows.add(
                        mapOf(
                            "interface" to (ni.name ?: ""),
                            "address" to (addr.hostAddress ?: ""),
                            "family" to family,
                        ),
                    )
                }
            }
            rows
        }.getOrElse { emptyList() }
    }
}
