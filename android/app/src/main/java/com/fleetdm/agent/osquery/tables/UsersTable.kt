package com.fleetdm.agent.osquery.tables

import android.os.Process
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext

class UsersTable : TablePlugin {
    override val name: String = "users"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("uid"),
        ColumnDef("gid"),
        ColumnDef("username"),
        ColumnDef("directory"),
        ColumnDef("shell"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val uid = Process.myUid()
        return listOf(
            mapOf(
                "uid" to uid.toString(),
                "gid" to uid.toString(),
                "username" to "android_app_uid_$uid",
                "directory" to "/data/user",
                "shell" to "",
            ),
        )
    }
}
