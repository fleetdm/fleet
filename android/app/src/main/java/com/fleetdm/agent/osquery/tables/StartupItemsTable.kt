package com.fleetdm.agent.osquery.tables

import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import com.fleetdm.agent.osquery.core.ColumnDef
import com.fleetdm.agent.osquery.core.TablePlugin
import com.fleetdm.agent.osquery.core.TableQueryContext

class StartupItemsTable(private val context: Context) : TablePlugin {
    override val name: String = "startup_items"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("package_name"),
        ColumnDef("component"),
        ColumnDef("type"),
        ColumnDef("enabled"),
        ColumnDef("exported"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val pm = context.packageManager
        val out = mutableListOf<Map<String, String>>()

        val bootIntent = Intent(Intent.ACTION_BOOT_COMPLETED)
        val bootResolvers = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            pm.queryBroadcastReceivers(bootIntent, PackageManager.ResolveInfoFlags.of(PackageManager.MATCH_ALL.toLong()))
        } else {
            @Suppress("DEPRECATION")
            pm.queryBroadcastReceivers(bootIntent, PackageManager.MATCH_ALL)
        }
        for (r in bootResolvers) {
            val ai = r.activityInfo ?: continue
            out.add(
                mapOf(
                    "package_name" to ai.packageName,
                    "component" to ComponentName(ai.packageName, ai.name).flattenToShortString(),
                    "type" to "boot_receiver",
                    "enabled" to bool01(ai.enabled),
                    "exported" to bool01(ai.exported),
                ),
            )
        }

        val launcherIntent = Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_LAUNCHER)
        val launchResolvers = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            pm.queryIntentActivities(launcherIntent, PackageManager.ResolveInfoFlags.of(PackageManager.MATCH_ALL.toLong()))
        } else {
            @Suppress("DEPRECATION")
            pm.queryIntentActivities(launcherIntent, PackageManager.MATCH_ALL)
        }
        for (r in launchResolvers) {
            val ai = r.activityInfo ?: continue
            out.add(
                mapOf(
                    "package_name" to ai.packageName,
                    "component" to ComponentName(ai.packageName, ai.name).flattenToShortString(),
                    "type" to "launcher_activity",
                    "enabled" to bool01(ai.enabled),
                    "exported" to bool01(ai.exported),
                ),
            )
        }

        return out.distinctBy { "${it["package_name"]}|${it["component"]}|${it["type"]}" }
    }

    private fun bool01(v: Boolean) = if (v) "1" else "0"
}
