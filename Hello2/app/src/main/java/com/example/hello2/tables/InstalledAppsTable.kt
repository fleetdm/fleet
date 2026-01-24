package com.example.hello2.tables

import android.content.Context
import android.content.pm.PackageInfo
import android.content.pm.PackageManager
import android.os.Build
import com.example.hello2.core.ColumnDef
import com.example.hello2.core.TablePlugin
import com.example.hello2.core.TableQueryContext

class InstalledAppsTable(private val context: Context) : TablePlugin {
    override val name: String = "installed_apps"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("package_name"),
        ColumnDef("app_name"),
        ColumnDef("version_name"),
        ColumnDef("version_code"),
        ColumnDef("first_install_time"),
        ColumnDef("last_update_time"),
        ColumnDef("is_system"),
    )

    override suspend fun generate(ctx: TableQueryContext): List<Map<String, String>> {
        val pm = context.packageManager

        val packages: List<PackageInfo> =
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                pm.getInstalledPackages(PackageManager.PackageInfoFlags.of(0))
            } else {
                @Suppress("DEPRECATION")
                pm.getInstalledPackages(0)
            }

        return packages.mapNotNull { pi ->
            val appInfo = pi.applicationInfo ?: return@mapNotNull null

            val appLabel = try {
                pm.getApplicationLabel(appInfo).toString()
            } catch (_: Exception) {
                ""
            }

            val isSystem =
                (appInfo.flags and android.content.pm.ApplicationInfo.FLAG_SYSTEM) != 0

            val versionCode =
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
                    pi.longVersionCode.toString()
                } else {
                    @Suppress("DEPRECATION")
                    pi.versionCode.toString()
                }

            mapOf(
                "package_name" to pi.packageName,
                "app_name" to appLabel,
                "version_name" to (pi.versionName ?: ""),
                "version_code" to versionCode,
                "first_install_time" to pi.firstInstallTime.toString(),
                "last_update_time" to pi.lastUpdateTime.toString(),
                "is_system" to isSystem.toString(),
            )
        }
    }
}
