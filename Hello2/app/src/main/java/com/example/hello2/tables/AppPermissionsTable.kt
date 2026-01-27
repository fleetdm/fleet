package com.example.hello2.tables

import android.content.Context
import android.content.pm.PackageInfo
import android.content.pm.PackageManager
import android.content.pm.PermissionInfo
import android.os.Build
import com.example.hello2.core.ColumnDef
import com.example.hello2.core.TablePlugin
import com.example.hello2.core.TableQueryContext

class AppPermissionsTable(private val context: Context) : TablePlugin {
    override val name: String = "app_permissions"

    override fun columns(): List<ColumnDef> = listOf(
        ColumnDef("package_name"),
        ColumnDef("app_name"),
        ColumnDef("permission"),
        ColumnDef("protection_level"), // normal | dangerous | signature | other
        ColumnDef("granted"),          // true | false
        ColumnDef("is_system"),        // true | false
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

        val rows = mutableListOf<Map<String, String>>()

        for (pi in packages) {
            val pkg = pi.packageName ?: continue
            val appInfo = pi.applicationInfo ?: continue

            val appLabel = try {
                pm.getApplicationLabel(appInfo).toString()
            } catch (_: Exception) {
                ""
            }

            val isSystem =
                (appInfo.flags and android.content.pm.ApplicationInfo.FLAG_SYSTEM) != 0

            // Need requestedPermissions -> fetch PackageInfo with GET_PERMISSIONS
            val piWithPerms: PackageInfo = try {
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                    pm.getPackageInfo(
                        pkg,
                        PackageManager.PackageInfoFlags.of(PackageManager.GET_PERMISSIONS.toLong())
                    )
                } else {
                    @Suppress("DEPRECATION")
                    pm.getPackageInfo(pkg, PackageManager.GET_PERMISSIONS)
                }
            } catch (_: Exception) {
                continue
            }

            val requested = piWithPerms.requestedPermissions ?: emptyArray()
            if (requested.isEmpty()) continue

            for (perm in requested) {
                if (perm.isNullOrBlank()) continue

                val granted =
                    pm.checkPermission(perm, pkg) == PackageManager.PERMISSION_GRANTED

                val protection = getProtectionLevelString(pm, perm)

                rows.add(
                    mapOf(
                        "package_name" to pkg,
                        "app_name" to appLabel,
                        "permission" to perm,
                        "protection_level" to protection,
                        "granted" to granted.toString(),
                        "is_system" to isSystem.toString(),
                    )
                )
            }
        }

        return rows
    }

    private fun getProtectionLevelString(pm: PackageManager, permission: String): String {
        val pi: PermissionInfo = try {
            pm.getPermissionInfo(permission, 0)
        } catch (_: Exception) {
            return "other"
        }

        val base = pi.protectionLevel and PermissionInfo.PROTECTION_MASK_BASE

        return when (base) {
            PermissionInfo.PROTECTION_NORMAL -> "normal"
            PermissionInfo.PROTECTION_DANGEROUS -> "dangerous"
            PermissionInfo.PROTECTION_SIGNATURE -> "signature"
            else -> "other"
        }
    }
}


