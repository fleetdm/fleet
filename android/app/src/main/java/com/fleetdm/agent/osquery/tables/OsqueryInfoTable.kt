package com.fleetdm.agent.osquery.tables

import android.content.Context
import android.provider.Settings
import com.fleetdm.agent.BuildConfig
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext
import java.util.UUID

class OsqueryInfoTable(private val context: Context) : TablePlugin {
    override val name: String = "osquery_info"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("uuid"),
        ColumnDef("instance_id"),
        ColumnDef("version"),
        ColumnDef("config_hash"),
        ColumnDef("extensions"),
        ColumnDef("build_platform"),
        ColumnDef("build_distro"),
        ColumnDef("platform_mask"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val uuid = stableDeviceId()
        val version = BuildConfig.VERSION_NAME

        return listOf(
            mapOf(
                "uuid" to uuid,
                "instance_id" to uuid,
                "version" to version,
                "config_hash" to "",
                "extensions" to "",
                "build_platform" to "android",
                "build_distro" to "android",
                "platform_mask" to "0",
            ),
        )
    }

    private fun stableDeviceId(): String {
        // Prefer ANDROID_ID. If it is unavailable, fall back to a random UUID per process.
        // ANDROID_ID is stable per device+user+signing key in most modern Android versions.
        val androidId = Settings.Secure.getString(context.contentResolver, Settings.Secure.ANDROID_ID)
        if (!androidId.isNullOrBlank()) return androidId

        return UUID.randomUUID().toString()
    }
}
