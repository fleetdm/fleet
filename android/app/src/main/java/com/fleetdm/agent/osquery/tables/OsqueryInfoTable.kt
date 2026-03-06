package com.fleetdm.agent.osquery.tables

import android.content.Context
import com.fleetdm.agent.BuildConfig
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.OsqueryIdentityStore
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext

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
        val uuid = OsqueryIdentityStore.getOrCreateUuid(context)

        return listOf(
            mapOf(
                "uuid" to uuid,
                "instance_id" to uuid,
                "version" to BuildConfig.VERSION_NAME,
                "config_hash" to "",
                "extensions" to "",
                "build_platform" to "android",
                "build_distro" to "android",
                "platform_mask" to "0",
            ),
        )
    }
}
