package com.fleetdm.agent.osquery.tables

import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.net.wifi.WifiInfo
import android.net.wifi.WifiManager
import android.os.Build
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext

class WifiNetworksTable(
    private val context: Context,
) : TablePlugin {

    override val name: String = "wifi_networks"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("ssid"),
        ColumnDef("bssid"),
        ColumnDef("ip_address"),
        ColumnDef("mac_address"),
        ColumnDef("rssi"),
        ColumnDef("link_speed_mbps"),
        ColumnDef("frequency_mhz"),
        ColumnDef("network_id"),
        ColumnDef("is_connected"),
        ColumnDef("transport"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val cm = context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
        val active = cm.activeNetwork ?: return emptyList()
        val caps = cm.getNetworkCapabilities(active) ?: return emptyList()

        val isWifi = caps.hasTransport(NetworkCapabilities.TRANSPORT_WIFI)
        val transport = when {
            isWifi -> "wifi"
            caps.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> "cellular"
            caps.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) -> "ethernet"
            else -> "other"
        }

        if (!isWifi) {
            // Keep it simple: if not on Wi-Fi, return a single row saying not connected.
            return listOf(
                mapOf(
                    "ssid" to "",
                    "bssid" to "",
                    "ip_address" to "",
                    "mac_address" to "",
                    "rssi" to "",
                    "link_speed_mbps" to "",
                    "frequency_mhz" to "",
                    "network_id" to "",
                    "is_connected" to "false",
                    "transport" to transport,
                ),
            )
        }

        val wm = context.applicationContext.getSystemService(Context.WIFI_SERVICE) as WifiManager
        val info: WifiInfo? = wm.connectionInfo

        val ssid = info?.ssid?.let { s ->
            // Android sometimes returns quoted SSID
            if (s.startsWith("\"") && s.endsWith("\"") && s.length >= 2) s.substring(1, s.length - 1) else s
        } ?: ""

        val bssid = info?.bssid ?: ""

        val ip = info?.ipAddress?.takeIf { it != 0 }?.let { intIpToString(it) } ?: ""

        // MAC is heavily restricted on modern Android; often returns 02:00:00:00:00:00
        val mac = info?.macAddress ?: ""

        val rssi = info?.rssi?.takeIf { it != -127 }?.toString() ?: ""
        val speed = info?.linkSpeed?.takeIf { it > 0 }?.toString() ?: ""
        val freq = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
            info?.frequency?.takeIf { it > 0 }?.toString() ?: ""
        } else {
            ""
        }

        val networkId = info?.networkId?.takeIf { it >= 0 }?.toString() ?: ""

        return listOf(
            mapOf(
                "ssid" to ssid,
                "bssid" to bssid,
                "ip_address" to ip,
                "mac_address" to mac,
                "rssi" to rssi,
                "link_speed_mbps" to speed,
                "frequency_mhz" to freq,
                "network_id" to networkId,
                "is_connected" to "true",
                "transport" to transport,
            ),
        )
    }

    private fun intIpToString(ip: Int): String {
        // WifiInfo.ipAddress is little-endian
        val b1 = ip and 0xff
        val b2 = (ip shr 8) and 0xff
        val b3 = (ip shr 16) and 0xff
        val b4 = (ip shr 24) and 0xff
        return "$b1.$b2.$b3.$b4"
    }
}
